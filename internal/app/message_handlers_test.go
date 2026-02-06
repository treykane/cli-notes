package app

import (
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func TestHandleEditNoteKeyCtrlBInsertsBoldMarkers(t *testing.T) {
	m := &Model{mode: modeEditNote, editor: textarea.New()}
	m.editor.SetValue("hello")
	m.editor.CursorEnd()

	result, _ := m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyCtrlB})
	got := result.(*Model)

	if got.editor.Value() != "hello****" {
		t.Fatalf("expected value %q, got %q", "hello****", got.editor.Value())
	}
	if got.editor.LineInfo().CharOffset != 7 {
		t.Fatalf("expected cursor at 7, got %d", got.editor.LineInfo().CharOffset)
	}
}

func TestHandleEditNoteKeyAltIInsertsItalicMarkers(t *testing.T) {
	m := &Model{mode: modeEditNote, editor: textarea.New()}
	m.editor.SetValue("hello")
	m.editor.CursorEnd()

	result, _ := m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}, Alt: true})
	got := result.(*Model)

	if got.editor.Value() != "hello**" {
		t.Fatalf("expected value %q, got %q", "hello**", got.editor.Value())
	}
	if got.editor.LineInfo().CharOffset != 6 {
		t.Fatalf("expected cursor at 6, got %d", got.editor.LineInfo().CharOffset)
	}
}

func TestHandleEditNoteKeyCtrlUInsertsUnderlineMarkers(t *testing.T) {
	m := &Model{mode: modeEditNote, editor: textarea.New()}
	m.editor.SetValue("hello")
	m.editor.CursorEnd()

	result, _ := m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyCtrlU})
	got := result.(*Model)

	if got.editor.Value() != "hello<u></u>" {
		t.Fatalf("expected value %q, got %q", "hello<u></u>", got.editor.Value())
	}
	if got.editor.LineInfo().CharOffset != 8 {
		t.Fatalf("expected cursor at 8, got %d", got.editor.LineInfo().CharOffset)
	}
}
