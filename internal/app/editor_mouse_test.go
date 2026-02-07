package app

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func mouseCellForEditor(m *Model, row, col int) (int, int) {
	layout := m.calculateLayout()
	originX, originY := m.editPaneContentOrigin(layout)
	gutter := lipgloss.Width(m.editor.Prompt)
	if m.editor.ShowLineNumbers {
		gutter += len(fmt.Sprintf("%3v ", max(1, m.editor.LineCount())))
	}
	return originX + gutter + col, originY + row
}

func prepareEditMouseModel(value string) *Model {
	m := newFocusedEditModel(value)
	m.width = 120
	m.height = 40
	m.updateLayout()
	layout := m.calculateLayout()
	m.editor.SetWidth(layout.ViewportWidth)
	m.editor.SetHeight(layout.ViewportHeight)
	m.editor.Focus()
	return m
}

func TestMouseSelectionDragCreatesRange(t *testing.T) {
	m := prepareEditMouseModel("hello\nworld")
	pressX, pressY := mouseCellForEditor(m, 0, 1)
	dragX, dragY := mouseCellForEditor(m, 1, 3)

	_, _ = m.handleMouse(tea.MouseMsg{X: pressX, Y: pressY, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	_, _ = m.handleMouse(tea.MouseMsg{X: dragX, Y: dragY, Action: tea.MouseActionMotion, Button: tea.MouseButtonLeft})
	_, _ = m.handleMouse(tea.MouseMsg{X: dragX, Y: dragY, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})

	start, end, ok := m.editorSelectionRange()
	if !ok {
		t.Fatal("expected active selection after drag")
	}
	if start != 1 || end != 9 {
		t.Fatalf("expected selection [1,9), got [%d,%d)", start, end)
	}
}

func TestMouseSelectionReverseDragNormalizesRange(t *testing.T) {
	m := prepareEditMouseModel("hello\nworld")
	pressX, pressY := mouseCellForEditor(m, 1, 4)
	dragX, dragY := mouseCellForEditor(m, 0, 1)

	_, _ = m.handleMouse(tea.MouseMsg{X: pressX, Y: pressY, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	_, _ = m.handleMouse(tea.MouseMsg{X: dragX, Y: dragY, Action: tea.MouseActionMotion, Button: tea.MouseButtonLeft})
	_, _ = m.handleMouse(tea.MouseMsg{X: dragX, Y: dragY, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})

	start, end, ok := m.editorSelectionRange()
	if !ok {
		t.Fatal("expected active selection after reverse drag")
	}
	if start != 1 || end != 10 {
		t.Fatalf("expected selection [1,10), got [%d,%d)", start, end)
	}
}

func TestMouseSelectionReleaseAtAnchorClearsSelection(t *testing.T) {
	m := prepareEditMouseModel("hello\nworld")
	x, y := mouseCellForEditor(m, 0, 2)

	_, _ = m.handleMouse(tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	_, _ = m.handleMouse(tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})

	if m.editorSelectionActive {
		t.Fatal("expected selection to clear on zero-length drag")
	}
}

func TestMouseSelectionOutsideEditorNoOp(t *testing.T) {
	m := prepareEditMouseModel("hello\nworld")
	_, _ = m.handleMouse(tea.MouseMsg{X: 0, Y: 0, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if m.editorSelectionActive {
		t.Fatal("expected no selection change for outside click")
	}
}

func TestEditorOffsetFromVisualPositionExactWrapBoundary(t *testing.T) {
	m := newFocusedEditModel("abcd")
	m.editor.Prompt = ""
	m.editor.ShowLineNumbers = false
	m.editor.SetWidth(4)

	if got := m.editorOffsetFromVisualPosition(0, 2); got != 2 {
		t.Fatalf("expected row 0 col 2 => offset 2, got %d", got)
	}
	if got := m.editorOffsetFromVisualPosition(1, 0); got != 4 {
		t.Fatalf("expected boundary continuation row 1 col 0 => offset 4, got %d", got)
	}
}

func TestEditorOffsetFromVisualPositionWordWrap(t *testing.T) {
	m := newFocusedEditModel("hello world")
	m.editor.Prompt = ""
	m.editor.ShowLineNumbers = false
	m.editor.SetWidth(7)

	if got := m.editorOffsetFromVisualPosition(1, 0); got != 6 {
		t.Fatalf("expected wrapped row start offset 6, got %d", got)
	}
	if got := m.editorOffsetFromVisualPosition(1, 4); got != 10 {
		t.Fatalf("expected wrapped row col 4 => offset 10, got %d", got)
	}
	if got := m.editorOffsetFromVisualPosition(1, 99); got != 11 {
		t.Fatalf("expected wrapped row click past text to clamp to line end 11, got %d", got)
	}
}

func TestEditorOffsetFromVisualPositionWideRunes(t *testing.T) {
	m := newFocusedEditModel("ab界cd")
	m.editor.Prompt = ""
	m.editor.ShowLineNumbers = false
	m.editor.SetWidth(4)

	if got := m.editorOffsetFromVisualPosition(0, 2); got != 2 {
		t.Fatalf("expected wide-rune first column to map to rune offset 2, got %d", got)
	}
	if got := m.editorOffsetFromVisualPosition(0, 3); got != 3 {
		t.Fatalf("expected wide-rune second column to map after rune (offset 3), got %d", got)
	}
	if got := m.editorOffsetFromVisualPosition(1, 0); got != 3 {
		t.Fatalf("expected wrapped continuation start offset 3, got %d", got)
	}
}
