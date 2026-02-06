package app

import (
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
)

func TestEditorViewWithSelectionHighlightHighlightsSelectedTextOnly(t *testing.T) {
	m := &Model{
		editor:                textarea.New(),
		editorSelectionAnchor: noEditorSelectionAnchor,
		editorSelectionActive: false,
	}
	m.editor.SetValue("hello world")
	m.editor.Focus()
	m.editorSelectionAnchor = 6
	m.editorSelectionActive = true
	m.setEditorValueAndCursorOffset("hello world", 11)

	out := m.editorViewWithSelectionHighlight("hello world")
	highlighted := selectionText.Render("world")
	expected := "hello " + highlighted

	if out != expected {
		t.Fatalf("expected %q, got %q", expected, out)
	}
}

func TestEditorViewWithSelectionHighlightSkipsMultilineSelection(t *testing.T) {
	m := &Model{
		editor:                textarea.New(),
		editorSelectionAnchor: noEditorSelectionAnchor,
		editorSelectionActive: false,
	}
	m.editor.SetValue("hello\nworld")
	m.editor.Focus()
	m.editorSelectionAnchor = 0
	m.editorSelectionActive = true
	m.setEditorValueAndCursorOffset("hello\nworld", 11)

	in := "hello\nworld"
	out := m.editorViewWithSelectionHighlight(in)
	if out != in {
		t.Fatalf("expected unchanged output for multiline selection, got %q", out)
	}
}
