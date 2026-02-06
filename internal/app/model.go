// Package app implements the terminal UI for cli-notes using the Bubble Tea framework.
//
// # Architecture Overview
//
// This package follows the Elm Architecture (Model-Update-View) pattern via Bubble Tea:
//
//   - Model: Holds all application state (see Model struct)
//   - Update: Processes messages and updates state (see Update function)
//   - View: Renders the current state to a string (see View function)
//
// # File Organization
//
// The app package is organized into focused modules:
//
//   - model.go: Core Model struct and Update loop
//   - view.go: UI rendering (View function and helpers)
//   - key_handlers.go: Keyboard event routing
//   - message_handlers.go: Message type handlers (resize, render, etc.)
//   - notes.go: Note and folder CRUD operations
//   - tree.go: Directory tree building and navigation
//   - render.go: Markdown rendering with caching
//   - search_index.go: Full-text search index
//   - layout.go: Layout dimension calculations
//   - constants.go: All magic numbers and configuration
//   - styles.go: Lipgloss styling
//   - util.go: Helper functions
//   - logging.go: Structured logging
//
// # Key Concepts
//
// Tree Navigation: The left pane shows a tree of folders and markdown files.
// Users navigate with vim-style keys (j/k, h/l) or arrows.
//
// Modes: The app has four modes that determine which widget is active:
//   - modeBrowse: Default mode, navigate tree and view notes
//   - modeEditNote: Textarea widget is active for editing
//   - modeNewNote: Input widget is active for naming a new note
//   - modeNewFolder: Input widget is active for naming a new folder
//
// Rendering: Markdown rendering is debounced and cached to prevent lag.
// When a file is selected, we wait 500ms before rendering to avoid
// excessive work during rapid navigation. Renders are cached by file
// path, modification time, and terminal width bucket.
//
// Search: Ctrl+P opens a popup that searches both filenames and content.
// The search index is built on-demand and kept in sync with file changes.
package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/treykane/cli-notes/internal/config"
)

// mode controls the UI state and which input widget is active.
type mode int

const (
	modeBrowse mode = iota
	modeNewNote
	modeNewFolder
	modeEditNote
)

// treeItem represents a single row in the left-hand tree pane.
type treeItem struct {
	path  string
	name  string
	depth int
	isDir bool
}

// Model holds the Bubble Tea state for the entire UI.
type Model struct {
	// File System State
	// The root directory containing all notes
	notesDir string
	// All visible tree items (files and folders) in the current view
	items []treeItem
	// Tracks which folders are expanded (path -> true if expanded)
	expanded map[string]bool
	// The file currently displayed in the viewport
	currentFile string
	// Full-text search index for quick lookup
	searchIndex *searchIndex

	// Tree Navigation
	// Index of the currently selected item in items slice
	cursor int
	// Scroll offset for the tree view
	treeOffset int
	// Height available for tree content (cached for scroll calculations)
	leftHeight int

	// Search State
	// Whether the search popup is currently visible
	searching bool
	// Items matching the current search query
	searchResults []treeItem
	// Index of the selected result in searchResults slice
	searchResultCursor int

	// UI Widgets
	// Markdown viewport for displaying notes
	viewport viewport.Model
	// Text input for new note/folder names
	input textinput.Model
	// Text input for search queries
	search textinput.Model
	// Textarea for editing note content
	editor textarea.Model
	// Loading spinner for async operations
	spinner spinner.Model

	// UI State
	// Current mode (browse, edit, new note, new folder)
	mode mode
	// Message shown in the status bar
	status string
	// Whether the help screen is displayed
	showHelp bool
	// Debug mode for input sequence logging
	debugInput bool

	// Layout Dimensions
	// Terminal width and height
	width  int
	height int

	// Mode-specific State
	// Parent directory for new note/folder creation
	newParent string
	// Anchor offset (in runes) for editor range selection
	editorSelectionAnchor int
	// Whether the editor selection anchor is currently active
	editorSelectionActive bool

	// Rendering State
	// Whether a markdown render is in progress
	rendering bool
	// Sequence number for the current render request (prevents stale renders)
	renderSeq int
	// Path that is pending render
	pendingPath string
	// Width for which we're rendering (bucketed for caching)
	pendingWidth int
	// Cache of rendered markdown (path -> renderCacheEntry)
	renderCache map[string]renderCacheEntry
	// Path currently being rendered (for error handling)
	renderingPath string
	// Sequence number of the in-flight render
	renderingSeq int
}

// New prepares the initial UI model and ensures the configured notes directory exists.
func New() (*Model, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	notesDir := cfg.NotesDir
	if err := ensureNotesDir(notesDir); err != nil {
		return nil, err
	}

	expanded := map[string]bool{notesDir: true}
	items := buildTree(notesDir, expanded)

	vp := viewport.New(0, 0)
	vp.SetContent("Select a note to view")

	input := textinput.New()
	input.Placeholder = "Name"
	input.CharLimit = InputCharLimit

	search := textinput.New()
	search.Prompt = ""
	search.Placeholder = "Type to search notes"
	search.CharLimit = InputCharLimit

	editor := textarea.New()
	editor.Placeholder = "Your note content here..."
	editor.CharLimit = 0
	applyEditorTheme(&editor)

	spin := spinner.New()
	spin.Spinner = spinner.Line

	return &Model{
		notesDir:              notesDir,
		items:                 items,
		expanded:              expanded,
		searchIndex:           newSearchIndex(notesDir),
		viewport:              vp,
		input:                 input,
		search:                search,
		editor:                editor,
		mode:                  modeBrowse,
		status:                "Ready",
		spinner:               spin,
		leftHeight:            0,
		renderCache:           map[string]renderCacheEntry{},
		editorSelectionAnchor: noEditorSelectionAnchor,
		editorSelectionActive: false,
		debugInput:            os.Getenv("CLI_NOTES_DEBUG_INPUT") != "",
	}, nil
}

// Init starts the spinner so we can show async rendering progress.
func (m *Model) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update is the Bubble Tea update loop: handle events and emit commands.
//
// This is the heart of the application. It receives messages from Bubble Tea
// (key presses, window resizes, async results) and routes them to specialized
// handlers based on message type and current mode.
//
// Message flow:
//   - spinner.TickMsg: Updates spinner animation (runs continuously)
//   - tea.WindowSizeMsg: Recalculates layout when terminal is resized
//   - renderRequestMsg: Dispatches markdown rendering after debounce delay
//   - renderResultMsg: Receives completed render and updates viewport
//   - tea.KeyMsg: Routes to mode-specific key handler
//
// The function is kept small by delegating to handlers in message_handlers.go
// and key_handlers.go.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		return m.handleSpinnerTick(msg)
	case tea.WindowSizeMsg:
		return m.handleWindowResize(msg)
	case renderRequestMsg:
		return m.handleRenderRequest(msg)
	case renderResultMsg:
		return m.handleRenderResult(msg)
	case tea.KeyMsg:
		switch m.mode {
		case modeEditNote:
			return m.handleEditNoteKey(msg)
		case modeNewNote:
			return m.handleNewNoteKey(msg)
		case modeNewFolder:
			return m.handleNewFolderKey(msg)
		default:
			return m.handleKey(msg)
		}
	}
	return m, nil
}

// handleKey routes key presses in browse mode.
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.searching {
		return m.handleSearchKey(msg)
	}
	return m.handleBrowseKey(msg.String())
}

func (m *Model) openSearchPopup() {
	m.searching = true
	m.search.SetValue("")
	m.search.Focus()
	m.searchResults = nil
	m.searchResultCursor = 0
	m.showHelp = false
	if m.searchIndex != nil {
		if err := m.searchIndex.ensureBuilt(); err != nil {
			appLog.Error("build search index", "root", m.notesDir, "error", err)
		}
	}
	m.status = "Search popup: type to filter, Enter to jump, Esc to cancel"
}

func (m *Model) closeSearchPopup() {
	m.searching = false
	m.search.Blur()
	m.search.SetValue("")
	m.searchResults = nil
	m.searchResultCursor = 0
}

func (m *Model) updateSearchRows() {
	query := strings.TrimSpace(m.search.Value())
	if m.searchIndex == nil {
		m.searchIndex = newSearchIndex(m.notesDir)
	}
	if err := m.searchIndex.ensureBuilt(); err != nil {
		m.searchResults = nil
		m.searchResultCursor = 0
		m.status = "Search index error"
		appLog.Error("build search index", "root", m.notesDir, "error", err)
		return
	}
	m.searchResults = m.searchIndex.search(query)
	if len(m.searchResults) == 0 {
		m.searchResultCursor = 0
		m.status = fmt.Sprintf("Search \"%s\" (0 matches)", query)
		return
	}
	m.searchResultCursor = clamp(m.searchResultCursor, 0, len(m.searchResults)-1)
	m.status = fmt.Sprintf("Search \"%s\" (%d matches)", query, len(m.searchResults))
}

func (m *Model) selectSearchResult() (tea.Model, tea.Cmd) {
	if len(m.searchResults) == 0 {
		m.status = "No search matches"
		return m, nil
	}

	item := m.searchResults[m.searchResultCursor]
	m.closeSearchPopup()
	m.expandParentDirs(item.path)
	if item.isDir {
		m.expanded[item.path] = true
	}
	m.rebuildTreeKeep(item.path)
	m.status = "Jumped to " + m.displayRelative(item.path)
	if item.isDir {
		return m, nil
	}
	return m, m.setCurrentFile(item.path)
}

func (m *Model) expandParentDirs(path string) {
	dir := path
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		dir = filepath.Dir(path)
	}

	for {
		if dir == "" || dir == "." {
			break
		}
		if !strings.HasPrefix(dir, m.notesDir) {
			break
		}
		m.expanded[dir] = true
		if dir == m.notesDir {
			break
		}
		next := filepath.Dir(dir)
		if next == dir {
			break
		}
		dir = next
	}
}

func isOSCBackgroundResponse(msg tea.KeyMsg) bool {
	if msg.Type != tea.KeyRunes {
		return false
	}
	sequence := msg.String()
	if sequence == "" {
		return false
	}
	sequence = trimOSCSequenceSuffix(sequence)
	if !strings.Contains(sequence, "rgb:") {
		return false
	}
	if !strings.Contains(sequence, "\x1b") &&
		!strings.Contains(sequence, "11;rgb:") &&
		!strings.Contains(sequence, "1;rgb:") &&
		!strings.Contains(sequence, "]11;rgb:") &&
		!strings.Contains(sequence, "]1;rgb:") {
		return false
	}
	return hasRGBTriple(sequence)
}

func (m *Model) shouldIgnoreInput(msg tea.KeyMsg) bool {
	classification, shouldIgnore := classifyInjectedInput(msg)
	if shouldIgnore {
		if m.debugInput {
			m.status = fmt.Sprintf("Ignored %s: %q", classification, msg.String())
		}
		return true
	}
	return false
}

func classifyInjectedInput(msg tea.KeyMsg) (string, bool) {
	if msg.Type != tea.KeyRunes {
		return "", false
	}
	sequence := msg.String()
	if sequence == "" {
		return "", false
	}
	if isOSCBackgroundResponse(msg) {
		return "osc_background_response", true
	}
	if strings.Contains(sequence, "\x1b[") || strings.Contains(sequence, "\x9b") {
		return "csi_escape_sequence", true
	}
	if strings.Contains(sequence, "\x1b]") || strings.Contains(sequence, "\x9d") {
		return "osc_escape_sequence", true
	}
	if strings.Contains(sequence, "\x1b") {
		return "escape_sequence", true
	}
	if containsControlRunes(sequence) {
		return "control_runes", true
	}
	return "", false
}

func trimOSCSequenceSuffix(sequence string) string {
	if strings.HasSuffix(sequence, "\x1b\\") {
		return strings.TrimSuffix(sequence, "\x1b\\")
	}
	if strings.HasSuffix(sequence, "\a") {
		return strings.TrimSuffix(sequence, "\a")
	}
	if strings.HasSuffix(sequence, "\\") {
		return strings.TrimSuffix(sequence, "\\")
	}
	if strings.HasSuffix(sequence, "\x1b") {
		return strings.TrimSuffix(sequence, "\x1b")
	}
	return sequence
}

func containsControlRunes(sequence string) bool {
	for _, r := range sequence {
		switch {
		case r == '\n' || r == '\t':
			continue
		case r < 32 || r == 127:
			return true
		}
	}
	return false
}

func isHex(value string) bool {
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'f':
		case r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}

func hasRGBTriple(sequence string) bool {
	index := strings.Index(sequence, "rgb:")
	if index == -1 {
		return false
	}
	tail := sequence[index+len("rgb:"):]
	for i := 0; i < 3; i++ {
		component, rest, ok := readHexComponent(tail)
		if !ok {
			return false
		}
		if len(component) < 4 || !isHex(component[:4]) {
			return false
		}
		if i < 2 {
			if rest == "" || rest[0] != '/' {
				return false
			}
			tail = rest[1:]
		} else {
			tail = rest
		}
	}
	return true
}

func readHexComponent(sequence string) (string, string, bool) {
	if sequence == "" {
		return "", "", false
	}
	var b strings.Builder
	for _, r := range sequence {
		if r == '/' {
			break
		}
		if !isHex(string(r)) {
			break
		}
		b.WriteRune(r)
	}
	component := b.String()
	if component == "" {
		return "", "", false
	}
	return component, sequence[len(component):], true
}
