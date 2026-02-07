package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// handleBrowseKey routes key presses in browse mode (not searching).
func (m *Model) handleBrowseKey(key string) (tea.Model, tea.Cmd) {
	if m.showHelp {
		return m.handleHelpKey(key)
	}

	action := m.actionForKey(key)
	switch action {
	case actionSearchHint:
		m.status = "Use Ctrl+P for search popup"
		return m, nil
	case actionCursorUp:
		return m.handleCursorUp()
	case actionCursorDown:
		return m.handleCursorDown()
	case actionJumpTop:
		return m.handleJumpTop()
	case actionJumpBottom:
		return m.handleJumpBottom()
	case actionExpandToggle:
		m.toggleExpand(true)
		return m, nil
	case actionCollapse:
		m.toggleExpand(false)
		return m, nil
	case actionQuit:
		return m, tea.Quit
	case actionHelp:
		return m.toggleHelp()
	case actionSearch:
		m.openSearchPopup()
		return m, nil
	case actionRecent:
		m.openRecentPopup()
		return m, nil
	case actionOutline:
		m.openOutlinePopup()
		return m, nil
	case actionWorkspace:
		m.openWorkspacePopup()
		return m, nil
	case actionNewNote:
		m.startNewNote()
		return m, nil
	case actionNewFolder:
		m.startNewFolder()
		return m, nil
	case actionEditNote:
		return m.startEditNote()
	case actionSort:
		m.cycleSortMode()
		return m, nil
	case actionPreviewScrollPageUp:
		return m.scrollActivePreviewBy(-m.previewPageStep())
	case actionPreviewScrollPageDown:
		return m.scrollActivePreviewBy(m.previewPageStep())
	case actionPreviewScrollHalfUp:
		return m.scrollActivePreviewBy(-m.previewHalfPageStep())
	case actionPreviewScrollHalfDown:
		return m.scrollActivePreviewBy(m.previewHalfPageStep())
	case actionPin:
		m.togglePinnedSelection()
		return m, nil
	case actionDelete:
		m.deleteSelected()
		return m, nil
	case actionCopyContent:
		m.copyCurrentNoteContentToClipboard()
		return m, nil
	case actionCopyPath:
		m.copyCurrentNotePathToClipboard()
		return m, nil
	case actionRename:
		m.startRenameSelected()
		return m, nil
	case actionRefresh:
		return m.handleRefresh()
	case actionMove:
		m.startMoveSelected()
		return m, nil
	case actionGitCommit:
		return m.handleGitCommitStart()
	case actionGitPull:
		return m.handleGitPull()
	case actionGitPush:
		return m.handleGitPush()
	case actionExport:
		m.openExportPopup()
		return m, nil
	case actionWikiLinks:
		m.openWikiLinksPopup()
		return m, nil
	case actionSplitToggle:
		m.toggleSplitMode()
		return m, nil
	case actionSplitFocus:
		m.toggleSplitFocus()
		return m, nil
	}
	return m, nil
}

func (m *Model) handleHelpKey(key string) (tea.Model, tea.Cmd) {
	if m.actionForKey(key) == actionHelp || normalizeKeyString(key) == "?" {
		m.showHelp = false
		m.status = "Help closed"
		return m, nil
	}

	switch normalizeKeyString(key) {
	case "up", "k":
		m.scrollHelpBy(-1)
	case "down", "j":
		m.scrollHelpBy(1)
	case "pgup":
		m.scrollHelpBy(-max(1, m.helpViewport.Height))
	case "pgdown":
		m.scrollHelpBy(max(1, m.helpViewport.Height))
	case "home", "g":
		m.helpViewport.YOffset = 0
	case "end", "shift+g":
		m.helpViewport.YOffset = m.maxHelpViewportOffset()
	}
	return m, nil
}

func (m *Model) scrollHelpBy(delta int) {
	maxOffset := m.maxHelpViewportOffset()
	m.helpViewport.YOffset = clamp(m.helpViewport.YOffset+delta, 0, maxOffset)
}

func (m *Model) maxHelpViewportOffset() int {
	total := m.helpViewport.TotalLineCount()
	return max(0, total-m.helpViewport.Height)
}

// handleSearchKey routes key presses while the search popup is active.
func (m *Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if m.shouldIgnoreInput(msg) {
		return m, nil
	}

	switch key {
	case "esc":
		m.closeSearchPopup()
		m.status = "Search cancelled"
		return m, nil
	case "up", "k", "ctrl+p":
		return m.moveSearchCursor(-1)
	case "down", "j", "ctrl+n":
		return m.moveSearchCursor(1)
	case "enter":
		return m.selectSearchResult()
	}

	// Handle text input for search query
	before := m.search.Value()
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	if before != m.search.Value() {
		m.updateSearchRows()
	}
	return m, cmd
}

// handleCursorUp moves the cursor up and updates the displayed file.
func (m *Model) handleCursorUp() (tea.Model, tea.Cmd) {
	m.moveCursor(-1)
	cmd := m.maybeShowSelectedFile()
	return m, cmd
}

// handleCursorDown moves the cursor down and updates the displayed file.
func (m *Model) handleCursorDown() (tea.Model, tea.Cmd) {
	m.moveCursor(1)
	cmd := m.maybeShowSelectedFile()
	return m, cmd
}

// handleJumpTop jumps to the first item in the tree.
func (m *Model) handleJumpTop() (tea.Model, tea.Cmd) {
	if len(m.items) > 0 {
		m.cursor = 0
		m.adjustTreeOffset()
	}
	cmd := m.maybeShowSelectedFile()
	return m, cmd
}

// handleJumpBottom jumps to the last item in the tree.
func (m *Model) handleJumpBottom() (tea.Model, tea.Cmd) {
	if len(m.items) > 0 {
		m.cursor = len(m.items) - 1
		m.adjustTreeOffset()
	}
	cmd := m.maybeShowSelectedFile()
	return m, cmd
}

// handleRefresh rebuilds the tree and search index.
func (m *Model) handleRefresh() (tea.Model, tea.Cmd) {
	m.rememberCurrentNotePosition()
	cmd := m.applyMutationEffects(mutationEffects{
		saveState:        true,
		invalidateSearch: true,
		clearRenderCache: true,
		refreshTree:      true,
		refreshGit:       true,
	})
	m.invalidateTreeMetadataCache()
	m.status = "Refreshed"
	return m, cmd
}

// toggleHelp shows or hides the help screen.
func (m *Model) toggleHelp() (tea.Model, tea.Cmd) {
	m.showHelp = !m.showHelp
	if m.showHelp {
		m.helpViewport.YOffset = 0
		m.helpViewport.SetContent(m.helpContent())
		m.status = ""
	}
	return m, nil
}

// moveSearchCursor moves the search result cursor by the given delta.
func (m *Model) moveSearchCursor(delta int) (tea.Model, tea.Cmd) {
	if len(m.searchResults) > 0 {
		m.searchResultCursor = clamp(m.searchResultCursor+delta, 0, len(m.searchResults)-1)
	}
	return m, nil
}

func (m *Model) previewPageStep() int {
	return max(1, m.viewport.Height)
}

func (m *Model) previewHalfPageStep() int {
	return max(1, m.viewport.Height/2)
}

func (m *Model) activePreviewTarget() (path string, secondary bool) {
	if m.splitMode && m.splitFocusSecondary {
		return m.secondaryFile, true
	}
	return m.currentFile, false
}

func (m *Model) previewLineCount(path string, secondary bool) int {
	if path == "" {
		return 0
	}
	if !secondary {
		if total := m.viewport.TotalLineCount(); total > 0 {
			return total
		}
	}
	rendered, ok := m.renderedForPath(path, m.viewport.Width)
	if !ok {
		return 0
	}
	return len(strings.Split(rendered, "\n"))
}

func (m *Model) scrollActivePreviewBy(delta int) (tea.Model, tea.Cmd) {
	if delta == 0 {
		return m, nil
	}
	path, secondary := m.activePreviewTarget()
	if path == "" {
		return m, nil
	}
	lineCount := m.previewLineCount(path, secondary)
	if lineCount <= 0 {
		return m, nil
	}

	maxOffset := lineCount - 1
	currentOffset := m.restorePaneOffset(path, secondary)
	if !secondary {
		currentOffset = max(0, m.viewport.YOffset)
	}
	nextOffset := clamp(currentOffset+delta, 0, maxOffset)
	if nextOffset == currentOffset {
		return m, nil
	}

	m.setPaneOffset(path, secondary, nextOffset)
	if !secondary {
		m.viewport.YOffset = nextOffset
	}
	m.saveAppState()
	return m, nil
}
