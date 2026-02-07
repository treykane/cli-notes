package app

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/treykane/cli-notes/internal/config"
	"github.com/yuin/goldmark"
)

func (m *Model) openWorkspacePopup() {
	if len(m.workspaces) <= 1 {
		m.status = "No additional workspaces configured"
		return
	}
	m.closeTransientPopups()
	m.showWorkspacePopup = true
	m.workspaceCursor = 0
	for i, ws := range m.workspaces {
		if ws.Name == m.activeWorkspace {
			m.workspaceCursor = i
			break
		}
	}
	m.status = "Workspace: Enter to switch, Esc to close"
}

func (m *Model) handleWorkspacePopupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.shouldIgnoreInput(msg) {
		return m, nil
	}
	switch msg.String() {
	case "esc":
		m.showWorkspacePopup = false
		m.status = "Workspace picker closed"
		return m, nil
	case "up", "k", "ctrl+p":
		m.workspaceCursor = clamp(m.workspaceCursor-1, 0, len(m.workspaces)-1)
		return m, nil
	case "down", "j", "ctrl+n":
		m.workspaceCursor = clamp(m.workspaceCursor+1, 0, len(m.workspaces)-1)
		return m, nil
	case "enter":
		return m.selectWorkspaceEntry()
	default:
		return m, nil
	}
}

func (m *Model) selectWorkspaceEntry() (tea.Model, tea.Cmd) {
	if len(m.workspaces) == 0 {
		m.showWorkspacePopup = false
		return m, nil
	}
	ws := m.workspaces[m.workspaceCursor]
	if ws.Name == m.activeWorkspace && ws.NotesDir == m.notesDir {
		m.showWorkspacePopup = false
		m.status = "Workspace unchanged"
		return m, nil
	}

	m.rememberCurrentNotePosition()
	m.saveAppState()
	m.activeWorkspace = ws.Name
	m.notesDir = ws.NotesDir
	m.expanded = map[string]bool{m.notesDir: true}
	m.currentFile = ""
	m.secondaryFile = ""
	m.currentNoteContent = ""
	m.items = buildTree(m.notesDir, m.expanded, m.sortMode, nil)
	m.cursor = 0
	m.treeOffset = 0
	state, err := loadAppState(m.notesDir)
	if err != nil {
		appLog.Warn("load workspace app state", "path", appStatePath(m.notesDir), "error", err)
	}
	m.pinnedPaths = state.PinnedPaths
	m.recentFiles = state.RecentFiles
	m.notePositions = state.Positions
	m.rebuildTreeKeep(m.notesDir)
	m.rebuildRecentEntries()
	m.refreshGitStatus()
	m.searchIndex = newSearchIndex(m.notesDir)
	m.renderCache = map[string]renderCacheEntry{}
	m.fileWatchSnapshot = nil
	m.viewport.SetContent("Select a note to view")
	m.showWorkspacePopup = false
	m.status = "Switched workspace: " + ws.Name
	if err := m.persistActiveWorkspace(); err != nil {
		m.setStatusError("Switched workspace but failed to persist active workspace", err)
	}
	return m, nil
}

func (m *Model) persistActiveWorkspace() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.ActiveWorkspace = m.activeWorkspace
	cfg.Workspaces = m.workspaces
	cfg.NotesDir = m.notesDir
	if cfg.TemplatesDir == "" {
		cfg.TemplatesDir = m.templatesDir
	}
	return config.Save(cfg)
}

func (m *Model) openExportPopup() {
	if m.currentFile == "" {
		m.status = "Select a note first"
		return
	}
	if !hasSuffixCaseInsensitive(m.currentFile, ".md") {
		m.status = "Export supports markdown notes only"
		return
	}
	m.closeTransientPopups()
	m.showExportPopup = true
	m.exportCursor = 0
	m.status = "Export: choose HTML or PDF"
}

func (m *Model) handleExportPopupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.shouldIgnoreInput(msg) {
		return m, nil
	}
	switch msg.String() {
	case "esc":
		m.showExportPopup = false
		m.status = "Export cancelled"
		return m, nil
	case "up", "k", "ctrl+p":
		m.exportCursor = clamp(m.exportCursor-1, 0, 1)
		return m, nil
	case "down", "j", "ctrl+n":
		m.exportCursor = clamp(m.exportCursor+1, 0, 1)
		return m, nil
	case "enter":
		m.showExportPopup = false
		if m.exportCursor == 0 {
			return m, m.exportCurrentNoteHTML()
		}
		return m, m.exportCurrentNotePDF()
	default:
		return m, nil
	}
}

func (m *Model) exportCurrentNoteHTML() tea.Cmd {
	path := m.currentFile
	return func() tea.Msg {
		content, err := os.ReadFile(path)
		if err != nil {
			return statusMsg{Text: "Export failed: unable to read note"}
		}
		_, body := parseFrontmatterAndBody(string(content))
		var out bytes.Buffer
		if err := goldmark.Convert([]byte(body), &out); err != nil {
			return statusMsg{Text: "Export failed: unable to convert markdown to HTML"}
		}
		htmlPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".html"
		if err := os.WriteFile(htmlPath, out.Bytes(), FilePermission); err != nil {
			return statusMsg{Text: "Export failed: unable to write HTML file"}
		}
		return statusMsg{Text: "Exported HTML: " + m.displayRelative(htmlPath)}
	}
}

func (m *Model) exportCurrentNotePDF() tea.Cmd {
	path := m.currentFile
	return func() tea.Msg {
		if _, err := exec.LookPath("pandoc"); err != nil {
			return statusMsg{Text: "PDF export unavailable: install pandoc to enable PDF export"}
		}
		pdfPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".pdf"
		cmd := exec.Command("pandoc", "-f", "markdown", "-o", pdfPath, path)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			line := strings.TrimSpace(stderr.String())
			if line == "" {
				line = err.Error()
			}
			return statusMsg{Text: "PDF export failed: " + line}
		}
		return statusMsg{Text: "Exported PDF: " + m.displayRelative(pdfPath)}
	}
}

type statusMsg struct {
	Text string
}

func (m *Model) closeTransientPopups() {
	if m.searching {
		m.closeSearchPopup()
	}
	m.showRecentPopup = false
	m.showOutlinePopup = false
	m.showWorkspacePopup = false
	m.showExportPopup = false
	m.showWikiLinksPopup = false
	m.showWikiAutocomplete = false
}

func (m *Model) toggleSplitMode() {
	m.splitMode = !m.splitMode
	if !m.splitMode {
		m.splitFocusSecondary = false
		m.secondaryFile = ""
		m.status = "Split mode disabled"
		return
	}
	if m.secondaryFile == "" && m.currentFile != "" {
		m.secondaryFile = m.currentFile
	}
	m.status = "Split mode enabled"
}

func (m *Model) toggleSplitFocus() {
	if !m.splitMode {
		return
	}
	m.splitFocusSecondary = !m.splitFocusSecondary
	if m.splitFocusSecondary {
		m.status = "Split focus: secondary pane"
	} else {
		m.status = "Split focus: primary pane"
	}
}

func (m *Model) setFocusedFile(path string) tea.Cmd {
	if !m.splitMode || !m.splitFocusSecondary {
		return m.setCurrentFile(path)
	}
	m.secondaryFile = path
	m.trackRecentFile(path)
	m.status = "Opened in secondary pane: " + m.displayRelative(path)
	return nil
}
