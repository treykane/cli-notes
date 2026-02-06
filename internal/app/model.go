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
	// Filesystem state
	notesDir    string
	items       []treeItem
	expanded    map[string]bool
	cursor      int
	treeOffset  int
	currentFile string
	searchIdx   *searchIndex

	// UI widgets
	viewport   viewport.Model
	input      textinput.Model
	search     textinput.Model
	editor     textarea.Model
	mode       mode
	status     string
	showHelp   bool
	debugInput bool
	searching  bool
	searchRows []treeItem
	searchPos  int

	// Layout sizing
	width      int
	height     int
	leftHeight int
	newParent  string

	// Rendering indicator
	spinner   spinner.Model
	rendering bool

	// Debounced render bookkeeping
	renderSeq     int
	pendingPath   string
	pendingWidth  int
	renderCache   map[string]renderCacheEntry
	renderingPath string
	renderingSeq  int
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
	input.CharLimit = 120

	search := textinput.New()
	search.Prompt = ""
	search.Placeholder = "Type to search notes"
	search.CharLimit = 120

	editor := textarea.New()
	editor.Placeholder = "Your note content here..."
	editor.CharLimit = 0
	applyEditorTheme(&editor)

	spin := spinner.New()
	spin.Spinner = spinner.Line

	return &Model{
		notesDir:    notesDir,
		items:       items,
		expanded:    expanded,
		searchIdx:   newSearchIndex(notesDir),
		viewport:    vp,
		input:       input,
		search:      search,
		editor:      editor,
		mode:        modeBrowse,
		status:      "Ready",
		spinner:     spin,
		leftHeight:  0,
		renderCache: map[string]renderCacheEntry{},
		debugInput:  os.Getenv("CLI_NOTES_DEBUG_INPUT") != "",
	}, nil
}

// Init starts the spinner so we can show async rendering progress.
func (m *Model) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update is the Bubble Tea update loop: handle events and emit commands.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
		if m.rendering {
			m.viewport.SetContent(m.spinner.View() + " Rendering...")
		}
		return m, tea.Batch(cmds...)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.leftHeight = max(0, m.height-2)
		m.updateLayout()
		cmd := m.refreshViewport()
		m.adjustTreeOffset()
		return m, cmd
	case renderRequestMsg:
		if msg.seq != m.renderSeq || msg.path != m.pendingPath || msg.width != m.pendingWidth {
			return m, nil
		}
		return m, renderMarkdownCmd(msg.path, msg.width, msg.seq)
	case renderResultMsg:
		if msg.err != nil {
			if msg.seq == m.renderSeq && msg.path == m.currentFile {
				m.viewport.SetContent("Error reading note")
				m.status = "Error reading note"
				m.rendering = false
				m.renderingPath = ""
				m.renderingSeq = 0
			}
			return m, nil
		}
		if entry, ok := m.renderCache[msg.path]; !ok || !entry.mtime.After(msg.mtime) {
			m.renderCache[msg.path] = renderCacheEntry{
				mtime:   msg.mtime,
				width:   msg.width,
				content: msg.content,
			}
		}
		if msg.seq != m.renderSeq || msg.path != m.currentFile {
			return m, nil
		}
		if msg.width == renderWidthBucket(m.viewport.Width) {
			m.viewport.SetContent(msg.content)
			m.rendering = false
			m.renderingPath = ""
			m.renderingSeq = 0
		}
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case modeEditNote:
			if m.shouldIgnoreInput(msg) {
				return m, nil
			}
			switch msg.String() {
			case "ctrl+s":
				return m.saveEdit()
			case "esc":
				m.mode = modeBrowse
				m.status = "Edit cancelled"
				return m, nil
			default:
				var cmd tea.Cmd
				m.editor, cmd = m.editor.Update(msg)
				return m, cmd
			}
		case modeNewNote:
			if m.shouldIgnoreInput(msg) {
				return m, nil
			}
			switch msg.String() {
			case "ctrl+s", "enter":
				return m.saveNewNote()
			case "esc":
				m.mode = modeBrowse
				m.status = "New note cancelled"
				return m, nil
			default:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
		case modeNewFolder:
			if m.shouldIgnoreInput(msg) {
				return m, nil
			}
			switch msg.String() {
			case "ctrl+s", "enter":
				return m.saveNewFolder()
			case "esc":
				m.mode = modeBrowse
				m.status = "New folder cancelled"
				return m, nil
			default:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
		default:
			return m.handleKey(msg)
		}
	}

	return m, nil
}

// handleKey routes key presses in browse mode.
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch m.mode {
	case modeBrowse:
		if m.searching {
			if m.shouldIgnoreInput(msg) {
				return m, nil
			}
			switch key {
			case "esc":
				m.closeSearchPopup()
				m.status = "Search cancelled"
				return m, nil
			case "up", "k":
				if len(m.searchRows) > 0 {
					m.searchPos = clamp(m.searchPos-1, 0, len(m.searchRows)-1)
				}
				return m, nil
			case "down", "j":
				if len(m.searchRows) > 0 {
					m.searchPos = clamp(m.searchPos+1, 0, len(m.searchRows)-1)
				}
				return m, nil
			case "ctrl+n":
				if len(m.searchRows) > 0 {
					m.searchPos = clamp(m.searchPos+1, 0, len(m.searchRows)-1)
				}
				return m, nil
			case "ctrl+p":
				if len(m.searchRows) > 0 {
					m.searchPos = clamp(m.searchPos-1, 0, len(m.searchRows)-1)
				}
				return m, nil
			case "enter":
				return m.selectSearchResult()
			}

			before := m.search.Value()
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			if before != m.search.Value() {
				m.updateSearchRows()
			}
			return m, cmd
		}

		switch key {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
			if m.showHelp {
				m.status = ""
			}
			return m, nil
		case "up", "k":
			m.moveCursor(-1)
			cmd := m.maybeShowSelectedFile()
			return m, cmd
		case "down", "j", "ctrl+n":
			m.moveCursor(1)
			cmd := m.maybeShowSelectedFile()
			return m, cmd
		case "g":
			if len(m.items) > 0 {
				m.cursor = 0
				m.adjustTreeOffset()
			}
			cmd := m.maybeShowSelectedFile()
			return m, cmd
		case "G":
			if len(m.items) > 0 {
				m.cursor = len(m.items) - 1
				m.adjustTreeOffset()
			}
			cmd := m.maybeShowSelectedFile()
			return m, cmd
		case "enter", "right", "l":
			m.toggleExpand(true)
			return m, nil
		case "left", "h":
			m.toggleExpand(false)
			return m, nil
		case "/":
			m.status = "Use Ctrl+P for search popup"
			return m, nil
		case "ctrl+p":
			m.openSearchPopup()
			return m, nil
		case "n":
			m.startNewNote()
			return m, nil
		case "f":
			m.startNewFolder()
			return m, nil
		case "e":
			return m.startEditNote()
		case "d":
			m.deleteSelected()
			return m, nil
		case "r":
			m.refreshTree()
			if m.searchIdx != nil {
				m.searchIdx.invalidate()
			}
			m.status = "Refreshed"
			return m, nil
		}
	}

	return m, nil
}

func (m *Model) openSearchPopup() {
	m.searching = true
	m.search.SetValue("")
	m.search.Focus()
	m.searchRows = nil
	m.searchPos = 0
	m.showHelp = false
	if m.searchIdx != nil {
		_ = m.searchIdx.ensureBuilt()
	}
	m.status = "Search popup: type to filter, Enter to jump, Esc to cancel"
}

func (m *Model) closeSearchPopup() {
	m.searching = false
	m.search.Blur()
	m.search.SetValue("")
	m.searchRows = nil
	m.searchPos = 0
}

func (m *Model) updateSearchRows() {
	query := strings.TrimSpace(m.search.Value())
	if m.searchIdx == nil {
		m.searchIdx = newSearchIndex(m.notesDir)
	}
	if err := m.searchIdx.ensureBuilt(); err != nil {
		m.searchRows = nil
		m.searchPos = 0
		m.status = "Search index error"
		return
	}
	m.searchRows = m.searchIdx.search(query)
	if len(m.searchRows) == 0 {
		m.searchPos = 0
		m.status = fmt.Sprintf("Search \"%s\" (0 matches)", query)
		return
	}
	m.searchPos = clamp(m.searchPos, 0, len(m.searchRows)-1)
	m.status = fmt.Sprintf("Search \"%s\" (%d matches)", query, len(m.searchRows))
}

func (m *Model) selectSearchResult() (tea.Model, tea.Cmd) {
	if len(m.searchRows) == 0 {
		m.status = "No search matches"
		return m, nil
	}

	item := m.searchRows[m.searchPos]
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
