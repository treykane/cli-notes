// workspace_export.go implements workspace switching, note export (HTML/PDF),
// split-pane mode, and transient popup lifecycle management.
//
// # Workspaces
//
// Users can configure multiple named workspaces in config.json, each pointing
// to a different notes directory. The workspace popup (Ctrl+W) lists all
// configured workspaces and allows switching between them. Switching a
// workspace tears down the current tree, search index, render cache, and
// file-watch snapshot, then reinitializes everything from the new workspace's
// notes directory and per-workspace state file.
//
// # Export
//
// The export popup (x key) offers two formats:
//
//   - HTML: Uses Goldmark to convert the current note's markdown body (with
//     frontmatter stripped) to HTML and writes it alongside the source file.
//   - PDF: Shells out to Pandoc (if installed). If Pandoc is not available,
//     the user is shown an install guidance message.
//
// Both export operations run as async Bubble Tea Cmds to keep the UI
// responsive during file I/O.
//
// # Split Pane
//
// Split mode (z key) divides the right pane into two side-by-side panels,
// each displaying a different note. The Tab key toggles focus between the
// primary and secondary panes, and file selection from the tree opens in
// whichever pane currently has focus.
//
// # Transient Popups
//
// closeTransientPopups is the central function for dismissing all overlay
// popups. It is called whenever a new popup is opened to enforce the
// one-popup-at-a-time invariant.
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

// openWorkspacePopup shows the workspace chooser popup (Ctrl+W). If only one
// workspace is configured, a status message is shown instead. The popup
// pre-selects the currently active workspace so the user can see which one
// is in use.
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

// handleWorkspacePopupKey routes key presses while the workspace popup is
// visible. Up/Down navigate the list, Enter switches to the selected
// workspace, and Esc dismisses the popup.
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

// selectWorkspaceEntry switches to the workspace at the current popup cursor.
//
// Switching a workspace is a heavy operation that resets most app state:
//  1. Persists the current note position and app state for the old workspace.
//  2. Updates notesDir, activeWorkspace, and clears the current/secondary file.
//  3. Rebuilds the tree from the new workspace's notes directory.
//  4. Loads the new workspace's per-workspace state (pins, recents, positions).
//  5. Creates a fresh search index and clears the render cache.
//  6. Resets the file-watch snapshot so the watcher re-baselines.
//  7. Persists the active workspace choice to config.json.
//
// If the selected workspace is already active, the popup is simply closed.
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

// persistActiveWorkspace writes the current active workspace name and
// workspace list back to ~/.cli-notes/config.json so the choice survives
// across app restarts.
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

// openExportPopup shows the export format chooser popup (x key). Only
// markdown files can be exported; non-markdown files show a status message
// instead. The popup offers HTML and PDF as export targets.
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

// handleExportPopupKey routes key presses while the export popup is visible.
// Enter triggers the selected export format; Esc cancels.
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

// exportCurrentNoteHTML returns an async Cmd that converts the current note
// to HTML using Goldmark and writes it alongside the source file (same name,
// .html extension). Frontmatter is stripped before conversion so the output
// contains only rendered markdown content.
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

// exportCurrentNotePDF returns an async Cmd that converts the current note
// to PDF by shelling out to Pandoc. If Pandoc is not installed (not found in
// PATH), a user-friendly status message with install guidance is returned
// instead of attempting the conversion.
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

// statusMsg is a Bubble Tea message that updates the footer status bar.
// It is used by async Cmds (export, git operations) to communicate results
// back to the Update loop without needing direct access to the Model.
type statusMsg struct {
	Text string
}

// closeTransientPopups dismisses all overlay popups (search, recent files,
// outline, workspace, export, wiki links, and wiki autocomplete). This is
// called before opening a new popup to enforce the invariant that at most
// one popup overlay is visible at any time.
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

// toggleSplitMode enables or disables the horizontal split-pane view.
//
// When enabling split mode, the secondary pane is initialized with the
// currently viewed file (so the user sees the same note in both panes as
// a starting point). When disabling, the secondary pane state is cleared
// and focus returns to the primary pane.
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

// toggleSplitFocus switches keyboard focus between the primary and secondary
// panes in split mode. This determines which pane receives newly opened files
// when the user selects notes from the tree or recent-files popup.
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

// setFocusedFile opens a file in the currently focused pane. In normal mode
// (or when the primary pane has focus in split mode), this delegates to
// setCurrentFile which handles tracking, rendering, and position restoration.
// When the secondary pane has focus in split mode, the file is opened there
// instead without affecting the primary pane's state.
func (m *Model) setFocusedFile(path string) tea.Cmd {
	if !m.splitMode || !m.splitFocusSecondary {
		return m.setCurrentFile(path)
	}
	m.secondaryFile = path
	m.trackRecentFile(path)
	m.status = "Opened in secondary pane: " + m.displayRelative(path)
	return nil
}
