package app

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// handleSpinnerTick updates the spinner animation state.
func (m *Model) handleSpinnerTick(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	if m.rendering {
		m.viewport.SetContent(m.spinner.View() + " Rendering...")
	}
	return m, cmd
}

// handleWindowResize updates layout dimensions after terminal resize.
func (m *Model) handleWindowResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.leftHeight = max(0, m.height-2)
	m.updateLayout()
	cmd := m.refreshViewport()
	m.adjustTreeOffset()
	return m, cmd
}

// handleRenderRequest validates and dispatches a render command.
func (m *Model) handleRenderRequest(msg renderRequestMsg) (tea.Model, tea.Cmd) {
	if msg.seq != m.renderSeq || msg.path != m.pendingPath || msg.width != m.pendingWidth {
		return m, nil
	}
	return m, renderMarkdownCmd(msg.path, msg.width, msg.seq)
}

// handleRenderResult processes the completed markdown render.
//
// This function implements a sophisticated debouncing and caching strategy:
//
// 1. Error Handling: If the render failed, show error only if it's still current.
//
//  2. Cache Update: Store the render result keyed by path, width bucket, and mtime.
//     This allows instant display when re-selecting a file or resizing to a cached width.
//
//  3. Sequence Validation: Only display if this render is still current (seq and path match).
//     This prevents stale renders from appearing after the user has moved to a different file.
//
//  4. Width Validation: Only display if the width still matches (prevents wrong-width renders).
//     The user may have resized the terminal while the render was in flight.
//
// The debouncing prevents excessive work during rapid navigation (e.g., holding down j/k).
// Each navigation increments renderSeq, which invalidates in-flight renders.
func (m *Model) handleRenderResult(msg renderResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		appLog.Error("render markdown", "path", msg.path, "seq", msg.seq, "error", msg.err)
		if msg.seq == m.renderSeq && msg.path == m.currentFile {
			m.viewport.SetContent("Error reading note")
			m.status = "Error reading note"
			m.clearRenderingState()
		}
		return m, nil
	}

	// Update cache if this is newer than what we have
	if entry, ok := m.renderCache[msg.path]; !ok || !entry.mtime.After(msg.mtime) {
		m.renderCache[msg.path] = renderCacheEntry{
			mtime:   msg.mtime,
			width:   msg.width,
			content: msg.content,
		}
	}

	// Only display if this render is still current
	if msg.seq != m.renderSeq || msg.path != m.currentFile {
		return m, nil
	}

	// Only update viewport if the width still matches
	if msg.width == roundWidthToNearestBucket(m.viewport.Width) {
		m.viewport.SetContent(msg.content)
		m.clearRenderingState()
	}
	return m, nil
}

// handleEditNoteKey processes keypresses while editing a note.
func (m *Model) handleEditNoteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.shouldIgnoreInput(msg) {
		return m, nil
	}
	key := msg.String()
	if m.handleEditorShiftSelectionMove(msg) {
		return m, nil
	}
	switch key {
	case "ctrl+s":
		return m.saveEdit()
	case "alt+s":
		m.toggleEditorSelectionAnchor()
		return m, nil
	case "ctrl+b":
		m.applyEditorFormat("**", "**", "bold")
		return m, nil
	case "alt+i":
		m.applyEditorFormat("*", "*", "italic")
		return m, nil
	case "ctrl+u":
		m.applyEditorFormat("<u>", "</u>", "underline")
		return m, nil
	case "esc":
		m.mode = modeBrowse
		m.clearEditorSelection()
		m.status = "Edit cancelled"
		return m, nil
	default:
		before := m.editor.Value()
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		if before != m.editor.Value() {
			m.clearEditorSelection()
		} else if m.hasEditorSelectionAnchor() {
			m.updateEditorSelectionStatus()
		}
		return m, cmd
	}
}

// insertEditorWrapper inserts open+close markers and positions the cursor between them.
func (m *Model) insertEditorWrapper(open, close string) {
	m.editor.InsertString(open + close)
	m.editor.Focus()
	for i := 0; i < len([]rune(close)); i++ {
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(tea.KeyMsg{Type: tea.KeyLeft})
		_ = cmd
	}
}

// handleNewNoteKey processes keypresses while creating a new note.
func (m *Model) handleNewNoteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
}

// handleNewFolderKey processes keypresses while creating a new folder.
func (m *Model) handleNewFolderKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
}

// clearRenderingState resets rendering flags after completion or error.
func (m *Model) clearRenderingState() {
	m.rendering = false
	m.renderingPath = ""
	m.renderingSeq = 0
}
