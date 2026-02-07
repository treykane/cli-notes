package app

import tea "github.com/charmbracelet/bubbletea"

// handleBrowseKey routes key presses in browse mode (not searching).
func (m *Model) handleBrowseKey(key string) (tea.Model, tea.Cmd) {
	action := m.actionForKey(key)
	switch key {
	case "ctrl+c":
		return m, tea.Quit
	case "?":
		return m.toggleHelp()
	case "up", "k":
		return m.handleCursorUp()
	case "down", "j", "ctrl+n":
		return m.handleCursorDown()
	case "g":
		return m.handleJumpTop()
	case "G":
		return m.handleJumpBottom()
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
	case "ctrl+o":
		m.openRecentPopup()
		return m, nil
	case "o":
		m.openOutlinePopup()
		return m, nil
	}
	switch action {
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
	if key == "R" || key == "shift+r" {
		return m.handleRefresh()
	}
	return m, nil
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
	case "up", "k":
		return m.moveSearchCursor(-1)
	case "down", "j":
		return m.moveSearchCursor(1)
	case "ctrl+n":
		return m.moveSearchCursor(1)
	case "ctrl+p":
		return m.moveSearchCursor(-1)
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
	m.saveAppState()
	m.refreshTree()
	if m.searchIndex != nil {
		m.searchIndex.invalidate()
	}
	m.renderCache = map[string]renderCacheEntry{}
	m.refreshGitStatus()
	m.status = "Refreshed"
	return m, nil
}

// toggleHelp shows or hides the help screen.
func (m *Model) toggleHelp() (tea.Model, tea.Cmd) {
	m.showHelp = !m.showHelp
	if m.showHelp {
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
