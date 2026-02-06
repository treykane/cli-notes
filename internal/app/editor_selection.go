package app

import (
	"fmt"
	"unicode"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

const noEditorSelectionAnchor = -1

func (m *Model) clearEditorSelection() {
	m.editorSelectionAnchor = noEditorSelectionAnchor
	m.editorSelectionActive = false
	applyEditorSelectionVisual(&m.editor, false)
}

func (m *Model) hasEditorSelectionAnchor() bool {
	return m.editorSelectionActive
}

func (m *Model) currentEditorCursorOffset() int {
	value := m.editor.Value()
	lines := splitEditorLines(value)
	row := clamp(m.editor.Line(), 0, max(0, len(lines)-1))
	col := clamp(m.editor.LineInfo().CharOffset, 0, len(lines[row]))

	offset := 0
	for i := 0; i < row; i++ {
		offset += len(lines[i]) + 1
	}
	return clamp(offset+col, 0, utf8.RuneCountInString(value))
}

func (m *Model) editorSelectionRange() (start, end int, ok bool) {
	if !m.hasEditorSelectionAnchor() {
		return 0, 0, false
	}

	cursor := m.currentEditorCursorOffset()
	start = m.editorSelectionAnchor
	end = cursor
	if start > end {
		start, end = end, start
	}
	if start == end {
		return 0, 0, false
	}
	return start, end, true
}

func (m *Model) toggleEditorSelectionAnchor() {
	if m.hasEditorSelectionAnchor() {
		m.clearEditorSelection()
		m.status = "Selection cleared"
		return
	}
	m.editorSelectionAnchor = m.currentEditorCursorOffset()
	m.editorSelectionActive = true
	applyEditorSelectionVisual(&m.editor, true)
	m.updateEditorSelectionStatus()
}

func (m *Model) handleEditorShiftSelectionMove(keyMsg tea.KeyMsg) bool {
	msg, ok := selectionMovementKeyMsg(keyMsg)
	if !ok {
		return false
	}

	if !m.hasEditorSelectionAnchor() {
		m.editorSelectionAnchor = m.currentEditorCursorOffset()
		m.editorSelectionActive = true
		applyEditorSelectionVisual(&m.editor, true)
	}

	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	_ = cmd
	m.updateEditorSelectionStatus()
	return true
}

func selectionMovementKeyMsg(keyMsg tea.KeyMsg) (tea.KeyMsg, bool) {
	switch keyMsg.Type {
	case tea.KeyShiftLeft:
		return tea.KeyMsg{Type: tea.KeyLeft}, true
	case tea.KeyShiftRight:
		return tea.KeyMsg{Type: tea.KeyRight}, true
	case tea.KeyShiftUp:
		return tea.KeyMsg{Type: tea.KeyUp}, true
	case tea.KeyShiftDown:
		return tea.KeyMsg{Type: tea.KeyDown}, true
	case tea.KeyShiftHome:
		return tea.KeyMsg{Type: tea.KeyHome}, true
	case tea.KeyShiftEnd:
		return tea.KeyMsg{Type: tea.KeyEnd}, true
	}

	switch keyMsg.String() {
	case "shift+left":
		return tea.KeyMsg{Type: tea.KeyLeft}, true
	case "shift+right":
		return tea.KeyMsg{Type: tea.KeyRight}, true
	case "shift+up":
		return tea.KeyMsg{Type: tea.KeyUp}, true
	case "shift+down":
		return tea.KeyMsg{Type: tea.KeyDown}, true
	case "shift+home":
		return tea.KeyMsg{Type: tea.KeyHome}, true
	case "shift+end":
		return tea.KeyMsg{Type: tea.KeyEnd}, true
	default:
		return tea.KeyMsg{}, false
	}
}

func (m *Model) updateEditorSelectionStatus() {
	if start, end, ok := m.editorSelectionRange(); ok {
		m.status = fmt.Sprintf("Selected %d chars (Alt+S to clear)", end-start)
		return
	}
	if m.hasEditorSelectionAnchor() {
		m.status = "Selection anchor set (move cursor to select, Alt+S to clear)"
	}
}

func (m *Model) applyEditorFormat(open, close, label string) {
	if start, end, ok := m.editorSelectionRange(); ok {
		removed := m.toggleEditorFormatRange(start, end, open, close)
		m.clearEditorSelection()
		if removed {
			m.status = "Removed " + label + " formatting from selection"
		} else {
			m.status = "Applied " + label + " formatting to selection"
		}
		return
	}

	cursor := m.currentEditorCursorOffset()
	if start, end, ok := wordBoundsAtCursor(m.editor.Value(), cursor); ok {
		removed := m.toggleEditorFormatRange(start, end, open, close)
		m.clearEditorSelection()
		if removed {
			m.status = "Removed " + label + " formatting from word"
		} else {
			m.status = "Applied " + label + " formatting to word"
		}
		return
	}

	m.insertEditorWrapper(open, close)
	m.clearEditorSelection()
	m.status = "Inserted " + label + " markers"
}

// toggleEditorFormatRange unwraps when exact wrappers surround the range, else wraps.
func (m *Model) toggleEditorFormatRange(start, end int, open, close string) bool {
	value := m.editor.Value()
	runes := []rune(value)
	start = clamp(start, 0, len(runes))
	end = clamp(end, 0, len(runes))
	if start > end {
		start, end = end, start
	}

	openRunes := []rune(open)
	closeRunes := []rune(close)
	openLen := len(openRunes)
	closeLen := len(closeRunes)

	openStart := start - openLen
	closeEnd := end + closeLen

	if openStart >= 0 &&
		closeEnd <= len(runes) &&
		runesEqual(runes[openStart:start], openRunes) &&
		runesEqual(runes[end:closeEnd], closeRunes) {
		updated := make([]rune, 0, len(runes)-openLen-closeLen)
		updated = append(updated, runes[:openStart]...)
		updated = append(updated, runes[start:end]...)
		updated = append(updated, runes[closeEnd:]...)
		m.setEditorValueAndCursorOffset(string(updated), end-openLen)
		return true
	}

	m.wrapEditorRange(start, end, open, close)
	return false
}

func (m *Model) wrapEditorRange(start, end int, open, close string) {
	value := m.editor.Value()
	runes := []rune(value)
	start = clamp(start, 0, len(runes))
	end = clamp(end, 0, len(runes))
	if start > end {
		start, end = end, start
	}

	openRunes := []rune(open)
	closeRunes := []rune(close)

	updated := make([]rune, 0, len(runes)+len(openRunes)+len(closeRunes))
	updated = append(updated, runes[:start]...)
	updated = append(updated, openRunes...)
	updated = append(updated, runes[start:end]...)
	updated = append(updated, closeRunes...)
	updated = append(updated, runes[end:]...)

	cursor := end + len(openRunes) + len(closeRunes)
	m.setEditorValueAndCursorOffset(string(updated), cursor)
}

func splitEditorLines(value string) [][]rune {
	lines := make([][]rune, 1)
	for _, r := range []rune(value) {
		if r == '\n' {
			lines = append(lines, nil)
			continue
		}
		last := len(lines) - 1
		lines[last] = append(lines[last], r)
	}
	return lines
}

func (m *Model) setEditorValueAndCursorOffset(value string, cursorOffset int) {
	total := utf8.RuneCountInString(value)
	cursorOffset = clamp(cursorOffset, 0, total)

	m.editor.SetValue(value)
	m.editor.Focus()

	movesLeft := total - cursorOffset
	for i := 0; i < movesLeft; i++ {
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(tea.KeyMsg{Type: tea.KeyLeft})
		_ = cmd
	}
}

func wordBoundsAtCursor(value string, cursor int) (start, end int, ok bool) {
	runes := []rune(value)
	if len(runes) == 0 {
		return 0, 0, false
	}

	cursor = clamp(cursor, 0, len(runes))
	idx := cursor
	if idx < len(runes) && isWordRune(runes[idx]) {
		// cursor is directly on a word rune
	} else if idx > 0 && isWordRune(runes[idx-1]) {
		idx--
	} else {
		return 0, 0, false
	}

	start = idx
	for start > 0 && isWordRune(runes[start-1]) {
		start--
	}

	end = idx + 1
	for end < len(runes) && isWordRune(runes[end]) {
		end++
	}
	return start, end, start < end
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func runesEqual(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
