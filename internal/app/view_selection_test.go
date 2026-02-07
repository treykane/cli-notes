package app

import (
	"strings"
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

func TestEditorViewWithSelectionHighlightSupportsMultilineSelection(t *testing.T) {
	m := &Model{
		editor:                textarea.New(),
		editorSelectionAnchor: noEditorSelectionAnchor,
		editorSelectionActive: false,
	}
	m.editor.SetValue("hello\nworld")
	m.editor.Focus()
	m.editorSelectionAnchor = 1
	m.editorSelectionActive = true
	m.setEditorValueAndCursorOffset("hello\nworld", 9)
	if start, end, ok := m.editorSelectionRange(); !ok {
		t.Fatalf("expected selection range, got none (anchor=%d active=%t cursor=%d)", m.editorSelectionAnchor, m.editorSelectionActive, m.currentEditorCursorOffset())
	} else if start >= end {
		t.Fatalf("invalid selection range [%d,%d)", start, end)
	}

	out := m.editorViewWithSelectionHighlight("hello\nworld")
	if want := selectionText.Render("ello"); !strings.Contains(out, want) {
		t.Fatalf("expected first line highlighted, got %q", out)
	}
	if want := selectionText.Render("wor"); !strings.Contains(out, want) {
		t.Fatalf("expected second line highlighted, got %q", out)
	}
}

func TestEditorViewWithSelectionHighlightMultilinePartialSegments(t *testing.T) {
	m := &Model{
		editor:                textarea.New(),
		editorSelectionAnchor: noEditorSelectionAnchor,
		editorSelectionActive: false,
	}
	m.editor.SetValue("alpha\nbravo")
	m.editor.Focus()
	m.editorSelectionAnchor = 2
	m.editorSelectionActive = true
	m.setEditorValueAndCursorOffset("alpha\nbravo", 8)

	out := m.editorViewWithSelectionHighlight("alpha\nbravo")
	if want := selectionText.Render("pha"); !strings.Contains(out, want) {
		t.Fatalf("expected first partial segment highlighted, got %q", out)
	}
	if want := selectionText.Render("br"); !strings.Contains(out, want) {
		t.Fatalf("expected second partial segment highlighted, got %q", out)
	}
}
