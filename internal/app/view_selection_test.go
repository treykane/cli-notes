package app

import (
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
)

func newSelectionRenderModel(value string) *Model {
	m := &Model{
		editor:                textarea.New(),
		editorSelectionAnchor: noEditorSelectionAnchor,
		editorSelectionActive: false,
	}
	m.editor.Prompt = ""
	m.editor.ShowLineNumbers = false
	m.editor.SetWidth(20)
	m.editor.SetHeight(6)
	m.editor.SetValue(value)
	m.editor.Focus()
	return m
}

func TestEditorSelectionRowSpansRepeatedTextUsesOffsets(t *testing.T) {
	m := newSelectionRenderModel("alpha alpha alpha")
	spans := m.editorSelectionRowSpans(6, 11)

	if len(spans) != 1 {
		t.Fatalf("expected 1 row span, got %d", len(spans))
	}
	if spans[0].row != 0 || spans[0].startCol != 6 || spans[0].endCol != 11 {
		t.Fatalf("expected span row=0 col=[6,11), got row=%d col=[%d,%d)", spans[0].row, spans[0].startCol, spans[0].endCol)
	}
}

func TestEditorSelectionRowSpansSupportsMultilineSelection(t *testing.T) {
	m := newSelectionRenderModel("hello\nworld")
	spans := m.editorSelectionRowSpans(1, 9)

	if len(spans) != 2 {
		t.Fatalf("expected 2 row spans, got %d", len(spans))
	}
	if spans[0].row != 0 || spans[0].startCol != 1 || spans[0].endCol != 5 {
		t.Fatalf("expected first span row=0 col=[1,5), got row=%d col=[%d,%d)", spans[0].row, spans[0].startCol, spans[0].endCol)
	}
	if spans[1].row != 1 || spans[1].startCol != 0 || spans[1].endCol != 3 {
		t.Fatalf("expected second span row=1 col=[0,3), got row=%d col=[%d,%d)", spans[1].row, spans[1].startCol, spans[1].endCol)
	}
}

func TestEditorSelectionRowSpansReverseSelection(t *testing.T) {
	m := newSelectionRenderModel("hello\nworld")
	spans := m.editorSelectionRowSpans(10, 1)

	if len(spans) != 2 {
		t.Fatalf("expected 2 row spans, got %d", len(spans))
	}
	if spans[0].row != 0 || spans[0].startCol != 1 || spans[0].endCol != 5 {
		t.Fatalf("expected first span row=0 col=[1,5), got row=%d col=[%d,%d)", spans[0].row, spans[0].startCol, spans[0].endCol)
	}
	if spans[1].row != 1 || spans[1].startCol != 0 || spans[1].endCol != 4 {
		t.Fatalf("expected second span row=1 col=[0,4), got row=%d col=[%d,%d)", spans[1].row, spans[1].startCol, spans[1].endCol)
	}
}

func TestEditorSelectionRowSpansWrapAware(t *testing.T) {
	m := newSelectionRenderModel("hello world")
	m.editor.SetWidth(7)
	spans := m.editorSelectionRowSpans(4, 10)

	if len(spans) != 2 {
		t.Fatalf("expected 2 row spans for wrapped selection, got %d", len(spans))
	}
	if spans[0].row != 0 || spans[0].startCol != 4 || spans[0].endCol != 6 {
		t.Fatalf("expected wrapped first span row=0 col=[4,6), got row=%d col=[%d,%d)", spans[0].row, spans[0].startCol, spans[0].endCol)
	}
	if spans[1].row != 1 || spans[1].startCol != 0 || spans[1].endCol != 4 {
		t.Fatalf("expected wrapped second span row=1 col=[0,4), got row=%d col=[%d,%d)", spans[1].row, spans[1].startCol, spans[1].endCol)
	}
}

func TestEditorSelectionRowSpansZeroLengthSelection(t *testing.T) {
	m := newSelectionRenderModel("hello world")
	spans := m.editorSelectionRowSpans(5, 5)
	if len(spans) != 0 {
		t.Fatalf("expected no spans for zero-length selection, got %d", len(spans))
	}
}
