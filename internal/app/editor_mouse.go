package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.mode != modeEditNote {
		return m, nil
	}
	return m.handleEditMouse(msg)
}

func (m *Model) handleEditMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft {
			return m, nil
		}
		offset, ok := m.editorOffsetFromMouse(msg)
		if !ok {
			return m, nil
		}
		if m.isOverlay(overlayWikiAutocomplete) {
			m.closeOverlay()
		}
		m.setEditorValueAndCursorOffset(m.editor.Value(), offset)
		m.editorSelectionAnchor = offset
		m.editorSelectionActive = true
		m.editorMouseSelecting = true
		m.editorMouseSelectionOrigin = offset
		applyEditorSelectionVisual(&m.editor)
		m.updateEditorSelectionStatus()
	case tea.MouseActionMotion:
		if !m.editorMouseSelecting {
			return m, nil
		}
		offset, ok := m.editorOffsetFromMouse(msg)
		if !ok {
			return m, nil
		}
		m.setEditorValueAndCursorOffset(m.editor.Value(), offset)
		m.updateEditorSelectionStatus()
	case tea.MouseActionRelease:
		if !m.editorMouseSelecting {
			return m, nil
		}
		m.editorMouseSelecting = false
		m.editorMouseSelectionOrigin = noEditorSelectionAnchor
		if offset, ok := m.editorOffsetFromMouse(msg); ok {
			m.setEditorValueAndCursorOffset(m.editor.Value(), offset)
		}
		if _, _, ok := m.editorSelectionRange(); !ok {
			m.clearEditorSelection()
			m.status = "Selection cleared"
			return m, nil
		}
		m.updateEditorSelectionStatus()
	}
	return m, nil
}

func (m *Model) editPaneContentOrigin(layout LayoutDimensions) (x, y int) {
	x = layout.LeftWidth + editPane.GetBorderLeftSize() + editPane.GetPaddingLeft()
	y = editPane.GetBorderTopSize() + editPane.GetPaddingTop() + 1 // +1 for header line
	return x, y
}

func (m *Model) editorOffsetFromMouse(msg tea.MouseMsg) (int, bool) {
	layout := m.calculateLayout()
	contentOriginX, contentOriginY := m.editPaneContentOrigin(layout)
	paneWidth := layout.RightWidth
	if m.splitMode {
		paneWidth = paneWidth / 2
	}
	paneEndX := layout.LeftWidth + paneWidth
	if msg.X < contentOriginX || msg.X >= paneEndX {
		return 0, false
	}
	if msg.Y < contentOriginY || msg.Y >= contentOriginY+layout.ViewportHeight {
		return 0, false
	}

	gutterWidth := lipgloss.Width(m.editor.Prompt)
	if m.editor.ShowLineNumbers {
		gutterWidth += len(fmt.Sprintf("%3v ", max(1, m.editor.LineCount())))
	}
	col := msg.X - contentOriginX - gutterWidth
	if col < 0 {
		col = 0
	}
	row := msg.Y - contentOriginY

	return m.editorOffsetFromVisualPosition(row, col), true
}

func (m *Model) editorOffsetFromVisualPosition(row, col int) int {
	value := m.editor.Value()
	lines := splitEditorLines(value)
	width := max(1, m.editor.Width())
	row = max(0, row)
	col = max(0, col)

	offset := 0
	for i, line := range lines {
		lineLen := len(line)
		visualRows := visualRowsForLine(lineLen, width)
		if row < visualRows {
			lineCol := row*width + col
			lineCol = clamp(lineCol, 0, lineLen)
			return clamp(offset+lineCol, 0, len([]rune(value)))
		}

		row -= visualRows
		offset += lineLen
		if i < len(lines)-1 {
			offset++
		}
	}
	return clamp(offset, 0, len([]rune(value)))
}

func visualRowsForLine(lineLen, width int) int {
	if width <= 0 {
		width = 1
	}
	if lineLen <= 0 {
		return 1
	}
	return 1 + (lineLen / width)
}
