package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

type mode int

const (
	modeBrowse mode = iota
	modeNewNote
	modeNewFolder
	modeEditNote
)

type treeItem struct {
	path  string
	name  string
	depth int
	isDir bool
}

type model struct {
	notesDir    string
	items       []treeItem
	expanded    map[string]bool
	cursor      int
	treeOffset  int
	currentFile string

	viewport viewport.Model
	input    textinput.Model
	editor   textarea.Model
	mode     mode
	status   string

	width      int
	height     int
	leftHeight int
	newParent  string
}

var (
	paneStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	selectedStyle = lipgloss.NewStyle().Reverse(true)
	titleStyle    = lipgloss.NewStyle().Bold(true)
	statusStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

const welcomeNote = "# Welcome to CLI Notes!\n\n" +
	"This is your personal notes manager in the terminal.\n\n" +
	"## Features\n\n" +
	"- Create and edit notes in Markdown\n" +
	"- Organize notes in folders\n" +
	"- View rendered Markdown formatting\n" +
	"- Keyboard-driven interface\n\n" +
	"## Keyboard Shortcuts\n\n" +
	"- n: Create a new note\n" +
	"- f: Create a new folder\n" +
	"- e: Edit the selected note\n" +
	"- d: Delete the selected note\n" +
	"- r: Refresh the directory tree\n" +
	"- q: Quit the application\n\n" +
	"## Getting Started\n\n" +
	"1. Press n to create a new note\n" +
	"2. Select a note and press e to edit it\n" +
	"3. Press f to create folders and organize your notes\n\n" +
	"Happy note-taking!\n"

func main() {
	m, err := initialModel()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func initialModel() (*model, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	notesDir := filepath.Join(home, "notes")
	if err := os.MkdirAll(notesDir, 0o755); err != nil {
		return nil, err
	}

	if isDirEmpty(notesDir) {
		welcomePath := filepath.Join(notesDir, "Welcome.md")
		_ = os.WriteFile(welcomePath, []byte(welcomeNote), 0o644)
	}

	expanded := map[string]bool{notesDir: true}
	items := buildTree(notesDir, expanded)

	vp := viewport.New(0, 0)
	vp.SetContent("Select a note to view")

	input := textinput.New()
	input.Placeholder = "Name"
	input.CharLimit = 120

	editor := textarea.New()
	editor.Placeholder = "Your note content here..."
	editor.CharLimit = 0

	return &model{
		notesDir:  notesDir,
		items:     items,
		expanded:  expanded,
		viewport:  vp,
		input:     input,
		editor:    editor,
		mode:      modeBrowse,
		status:    "Ready",
		leftHeight: 0,
	}, nil
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.leftHeight = max(0, m.height-2)
		m.updateLayout()
		m.refreshViewport()
		m.adjustTreeOffset()
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	if m.mode == modeEditNote {
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		return m, cmd
	}

	if m.mode == modeNewNote || m.mode == modeNewFolder {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch m.mode {
	case modeEditNote:
		switch key {
		case "ctrl+s":
			return m.saveEdit()
		case "esc":
			m.mode = modeBrowse
			m.status = "Edit cancelled"
			return m, nil
		}
	case modeNewNote:
		switch key {
		case "ctrl+s", "enter":
			return m.saveNewNote()
		case "esc":
			m.mode = modeBrowse
			m.status = "New note cancelled"
			return m, nil
		}
	case modeNewFolder:
		switch key {
		case "ctrl+s", "enter":
			return m.saveNewFolder()
		case "esc":
			m.mode = modeBrowse
			m.status = "New folder cancelled"
			return m, nil
		}
	case modeBrowse:
		switch key {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			m.moveCursor(-1)
			m.maybeShowSelectedFile()
			return m, nil
		case "down", "j":
			m.moveCursor(1)
			m.maybeShowSelectedFile()
			return m, nil
		case "enter", "right":
			m.toggleExpand(true)
			return m, nil
		case "left":
			m.toggleExpand(false)
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
			m.status = "Refreshed"
			return m, nil
		}
	}

	return m, nil
}

func (m *model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	leftWidth := min(40, m.width/3)
	rightWidth := max(0, m.width-leftWidth-1)
	contentHeight := max(0, m.height-2)

	leftPane := m.renderTree(leftWidth, contentHeight)
	rightPane := m.renderRight(rightWidth, contentHeight)
	row := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	return row + "\n" + m.renderStatus(m.width)
}

func (m *model) renderTree(width, height int) string {
	innerWidth := max(0, width-2)
	innerHeight := max(0, height-2)

	header := titleStyle.Render("Notes: " + m.notesDir)
	lines := []string{truncate(header, innerWidth)}

	visibleHeight := max(0, innerHeight-1)
	start := min(m.treeOffset, max(0, len(m.items)-1))
	end := min(len(m.items), start+visibleHeight)

	for i := start; i < end; i++ {
		item := m.items[i]
		line := m.formatTreeItem(item)
		if i == m.cursor {
			line = selectedStyle.Render(line)
		}
		lines = append(lines, truncate(line, innerWidth))
	}

	content := strings.Join(lines, "\n")
	return paneStyle.Width(width).Height(height).Render(content)
}

func (m *model) renderRight(width, height int) string {
	innerWidth := max(0, width-2)
	innerHeight := max(0, height-2)

	var content string
	switch m.mode {
	case modeEditNote:
		m.editor.SetWidth(innerWidth)
		m.editor.SetHeight(innerHeight)
		content = m.editor.View()
	case modeNewNote, modeNewFolder:
		m.input.Width = innerWidth
		prompt := "New note name"
		if m.mode == modeNewFolder {
			prompt = "New folder name"
		}
		location := "Location: " + m.displayRelative(m.newParent)
		helper := "Ctrl+S or Enter to save. Esc to cancel."
		content = strings.Join([]string{
			titleStyle.Render(prompt),
			location,
			"",
			m.input.View(),
			"",
			helper,
		}, "\n")
	default:
		m.viewport.Width = innerWidth
		m.viewport.Height = innerHeight
		content = m.viewport.View()
	}

	return paneStyle.Width(width).Height(height).Render(content)
}

func (m *model) renderStatus(width int) string {
	help := "n new  f folder  e edit  d delete  r refresh  q quit"
	line := help
	if m.status != "" {
		line = help + " | " + m.status
	}
	return statusStyle.Width(width).Render(truncate(line, width))
}

func (m *model) formatTreeItem(item treeItem) string {
	indent := strings.Repeat("  ", item.depth)
	if item.isDir {
		expanded := m.expanded[item.path]
		marker := "[+]"
		if expanded {
			marker = "[-]"
		}
		return fmt.Sprintf("%s%s %s", indent, marker, item.name)
	}
	return fmt.Sprintf("%s    %s", indent, item.name)
}

func (m *model) moveCursor(delta int) {
	if len(m.items) == 0 {
		return
	}

	m.cursor = clamp(m.cursor+delta, 0, len(m.items)-1)
	m.adjustTreeOffset()
}

func (m *model) adjustTreeOffset() {
	visibleHeight := max(0, m.leftHeight-2-1)
	if visibleHeight == 0 {
		m.treeOffset = 0
		return
	}

	if m.cursor < m.treeOffset {
		m.treeOffset = m.cursor
	}
	if m.cursor >= m.treeOffset+visibleHeight {
		m.treeOffset = m.cursor - visibleHeight + 1
	}
}

func (m *model) toggleExpand(expandIfDir bool) {
	item := m.selectedItem()
	if item == nil || !item.isDir {
		return
	}

	if expandIfDir {
		m.expanded[item.path] = !m.expanded[item.path]
	} else {
		if item.path == m.notesDir {
			return
		}
		m.expanded[item.path] = false
	}

	m.rebuildTreeKeep(item.path)
}

func (m *model) selectedItem() *treeItem {
	if len(m.items) == 0 || m.cursor < 0 || m.cursor >= len(m.items) {
		return nil
	}
	return &m.items[m.cursor]
}

func (m *model) selectedPath() string {
	item := m.selectedItem()
	if item == nil {
		return m.notesDir
	}
	return item.path
}

func (m *model) selectedParentDir() string {
	path := m.selectedPath()
	info, err := os.Stat(path)
	if err != nil {
		return m.notesDir
	}
	if info.IsDir() {
		return path
	}
	return filepath.Dir(path)
}

func (m *model) startNewNote() {
	m.mode = modeNewNote
	m.newParent = m.selectedParentDir()
	m.input.Reset()
	m.input.Placeholder = "Note name (without .md extension)"
	m.input.Focus()
	m.status = ""
}

func (m *model) startNewFolder() {
	m.mode = modeNewFolder
	m.newParent = m.selectedParentDir()
	m.input.Reset()
	m.input.Placeholder = "Folder name"
	m.input.Focus()
	m.status = ""
}

func (m *model) startEditNote() (tea.Model, tea.Cmd) {
	if m.currentFile == "" {
		m.status = "No note selected"
		return m, nil
	}

	content, err := os.ReadFile(m.currentFile)
	if err != nil {
		m.status = "Error reading note"
		return m, nil
	}

	m.mode = modeEditNote
	m.editor.SetValue(string(content))
	m.editor.CursorEnd()
	m.editor.Focus()
	m.status = "Editing " + filepath.Base(m.currentFile)
	return m, nil
}

func (m *model) saveNewNote() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.input.Value())
	if name == "" {
		m.status = "Note name is required"
		return m, nil
	}
	if !strings.HasSuffix(strings.ToLower(name), ".md") {
		name += ".md"
	}

	path := filepath.Join(m.newParent, name)
	content := "# " + strings.TrimSuffix(name, ".md") + "\n\nYour note content here...\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		m.status = "Error creating note"
		return m, nil
	}

	m.mode = modeBrowse
	m.status = "Created note: " + name
	m.expanded[m.newParent] = true
	m.refreshTree()
	m.setCurrentFile(path)
	return m, nil
}

func (m *model) saveNewFolder() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.input.Value())
	if name == "" {
		m.status = "Folder name is required"
		return m, nil
	}

	path := filepath.Join(m.newParent, name)
	if err := os.MkdirAll(path, 0o755); err != nil {
		m.status = "Error creating folder"
		return m, nil
	}

	m.mode = modeBrowse
	m.status = "Created folder: " + name
	m.expanded[m.newParent] = true
	m.refreshTree()
	return m, nil
}

func (m *model) saveEdit() (tea.Model, tea.Cmd) {
	if m.currentFile == "" {
		m.status = "No note selected"
		return m, nil
	}
	if err := os.WriteFile(m.currentFile, []byte(m.editor.Value()), 0o644); err != nil {
		m.status = "Error saving note"
		return m, nil
	}

	m.mode = modeBrowse
	m.status = "Saved: " + filepath.Base(m.currentFile)
	m.setCurrentFile(m.currentFile)
	return m, nil
}

func (m *model) deleteSelected() {
	item := m.selectedItem()
	if item == nil {
		m.status = "No item selected"
		return
	}

	if item.path == m.notesDir {
		m.status = "Cannot delete the root notes directory"
		return
	}

	if item.isDir {
		if !isDirEmpty(item.path) {
			m.status = "Folder is not empty. Delete contents first."
			return
		}
		if err := os.Remove(item.path); err != nil {
			m.status = "Error deleting folder"
			return
		}
		m.status = "Deleted folder: " + item.name
	} else {
		if err := os.Remove(item.path); err != nil {
			m.status = "Error deleting file"
			return
		}
		m.status = "Deleted: " + item.name
	}

	if item.path == m.currentFile {
		m.currentFile = ""
		m.viewport.SetContent("Select a note to view")
	}
	m.refreshTree()
}

func (m *model) refreshTree() {
	selected := m.selectedPath()
	m.rebuildTreeKeep(selected)
	m.adjustTreeOffset()
}

func (m *model) rebuildTreeKeep(path string) {
	m.items = buildTree(m.notesDir, m.expanded)
	m.cursor = 0
	for i, item := range m.items {
		if item.path == path {
			m.cursor = i
			break
		}
	}
	m.adjustTreeOffset()
}

func (m *model) maybeShowSelectedFile() {
	item := m.selectedItem()
	if item == nil || item.isDir {
		return
	}
	if strings.HasSuffix(strings.ToLower(item.path), ".md") {
		m.setCurrentFile(item.path)
	}
}

func (m *model) setCurrentFile(path string) {
	m.currentFile = path
	content, err := os.ReadFile(path)
	if err != nil {
		m.viewport.SetContent("Error reading note")
		m.status = "Error reading note"
		return
	}

	m.viewport.SetContent(renderMarkdown(string(content), m.viewport.Width))
}

func (m *model) refreshViewport() {
	if m.currentFile != "" {
		content, err := os.ReadFile(m.currentFile)
		if err == nil {
			m.viewport.SetContent(renderMarkdown(string(content), m.viewport.Width))
		}
	}
}

func (m *model) updateLayout() {
	leftWidth := min(40, m.width/3)
	rightWidth := max(0, m.width-leftWidth-1)
	contentHeight := max(0, m.height-2)
	m.viewport.Width = max(0, rightWidth-2)
	m.viewport.Height = max(0, contentHeight-2)
}

func (m *model) displayRelative(path string) string {
	rel, err := filepath.Rel(m.notesDir, path)
	if err != nil || rel == "." {
		return "/"
	}
	return rel
}

func buildTree(root string, expanded map[string]bool) []treeItem {
	items := []treeItem{{
		path:  root,
		name:  "/",
		depth: 0,
		isDir: true,
	}}

	if expanded[root] {
		walkTree(root, 1, expanded, &items)
	}

	return items
}

func walkTree(dir string, depth int, expanded map[string]bool, items *[]treeItem) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		item := treeItem{
			path:  path,
			name:  entry.Name(),
			depth: depth,
			isDir: entry.IsDir(),
		}
		*items = append(*items, item)
		if entry.IsDir() && expanded[path] {
			walkTree(path, depth+1, expanded, items)
		}
	}
}

func renderMarkdown(content string, width int) string {
	if width <= 0 {
		width = 80
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return content
	}
	out, err := renderer.Render(content)
	if err != nil {
		return content
	}
	return out
}

func isDirEmpty(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	return len(entries) == 0
}

func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	return runewidth.Truncate(s, width, "")
}

func clamp(value, minVal, maxVal int) int {
	if value < minVal {
		return minVal
	}
	if value > maxVal {
		return maxVal
	}
	return value
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
