package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func newFocusedEditModel(value string) *Model {
	m := &Model{
		mode:                  modeEditNote,
		editor:                textarea.New(),
		editorSelectionAnchor: noEditorSelectionAnchor,
		editorSelectionActive: false,
	}
	m.editor.SetValue(value)
	m.editor.Focus()
	m.editor.CursorEnd()
	return m
}

func TestHandleEditNoteKeyCtrlBWrapsCurrentWord(t *testing.T) {
	m := newFocusedEditModel("hello world")

	result, _ := m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyCtrlB})
	got := result.(*Model)

	if got.editor.Value() != "hello **world**" {
		t.Fatalf("expected value %q, got %q", "hello **world**", got.editor.Value())
	}
	if got.editorSelectionActive {
		t.Fatalf("expected selection to be cleared, got active anchor %d", got.editorSelectionAnchor)
	}
}

func TestHandleEditNoteKeyAltIWrapsCurrentWord(t *testing.T) {
	m := newFocusedEditModel("hello world")

	result, _ := m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}, Alt: true})
	got := result.(*Model)

	if got.editor.Value() != "hello *world*" {
		t.Fatalf("expected value %q, got %q", "hello *world*", got.editor.Value())
	}
}

func TestHandleEditNoteKeyCtrlUWrapsSelection(t *testing.T) {
	m := newFocusedEditModel("hello world")

	_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}, Alt: true})
	for i := 0; i < 5; i++ {
		_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyLeft})
	}

	result, _ := m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyCtrlU})
	got := result.(*Model)

	if got.editor.Value() != "hello <u>world</u>" {
		t.Fatalf("expected value %q, got %q", "hello <u>world</u>", got.editor.Value())
	}
	if got.editorSelectionActive {
		t.Fatalf("expected selection to be cleared, got active anchor %d", got.editorSelectionAnchor)
	}
}

func TestHandleEditNoteKeyShiftSelectThenBoldWrapsSelection(t *testing.T) {
	m := newFocusedEditModel("hello world")

	for i := 0; i < 5; i++ {
		_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyShiftLeft})
	}
	if !m.editorSelectionActive {
		t.Fatal("expected selection anchor to be active after shift selection")
	}

	result, _ := m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyCtrlB})
	got := result.(*Model)

	if got.editor.Value() != "hello **world**" {
		t.Fatalf("expected value %q, got %q", "hello **world**", got.editor.Value())
	}
}

func TestHandleEditNoteKeyCtrlBFallsBackToMarkerInsertion(t *testing.T) {
	m := newFocusedEditModel("hello ")

	result, _ := m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyCtrlB})
	got := result.(*Model)

	if got.editor.Value() != "hello ****" {
		t.Fatalf("expected value %q, got %q", "hello ****", got.editor.Value())
	}
}

func TestHandleEditNoteKeyCtrlBTogglesFormattedWordOff(t *testing.T) {
	m := newFocusedEditModel("hello **world**")
	m.setEditorValueAndCursorOffset("hello **world**", 10)

	result, _ := m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyCtrlB})
	got := result.(*Model)

	if got.editor.Value() != "hello world" {
		t.Fatalf("expected value %q, got %q", "hello world", got.editor.Value())
	}
}

func TestHandleEditNoteKeyCtrlBTogglesOnlyBoldInNestedFormatting(t *testing.T) {
	m := newFocusedEditModel("***word***")
	m.editorSelectionAnchor = 3
	m.editorSelectionActive = true
	m.setEditorValueAndCursorOffset("***word***", 7)

	result, _ := m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyCtrlB})
	got := result.(*Model)

	if got.editor.Value() != "*word*" {
		t.Fatalf("expected value %q, got %q", "*word*", got.editor.Value())
	}
}

func TestHandleEditNoteKeyTypingClearsSelectionAnchor(t *testing.T) {
	m := newFocusedEditModel("hello")

	_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}, Alt: true})
	if !m.editorSelectionActive {
		t.Fatal("expected selection anchor to be set")
	}

	_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'!'}})
	if m.editorSelectionActive {
		t.Fatalf("expected selection anchor cleared after edit, got active anchor %d", m.editorSelectionAnchor)
	}
}

func TestHandleConfirmDeleteKeyYDeletesPendingItem(t *testing.T) {
	root := t.TempDir()
	notePath := filepath.Join(root, "delete.md")
	if err := os.WriteFile(notePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write note: %v", err)
	}

	m := &Model{
		notesDir: root,
		mode:     modeConfirmDelete,
		pendingDelete: treeItem{
			path:  notePath,
			name:  "delete.md",
			isDir: false,
		},
		expanded: make(map[string]bool),
	}

	result, _ := m.handleConfirmDeleteKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	got := result.(*Model)

	if got.mode != modeBrowse {
		t.Fatalf("expected browse mode, got %v", got.mode)
	}
	if _, err := os.Stat(notePath); !os.IsNotExist(err) {
		t.Fatalf("expected file to be deleted, stat err: %v", err)
	}
}

func TestHandleConfirmDeleteKeyNDoesNotDeletePendingItem(t *testing.T) {
	root := t.TempDir()
	notePath := filepath.Join(root, "keep.md")
	if err := os.WriteFile(notePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write note: %v", err)
	}

	m := &Model{
		notesDir: root,
		mode:     modeConfirmDelete,
		pendingDelete: treeItem{
			path:  notePath,
			name:  "keep.md",
			isDir: false,
		},
		expanded: make(map[string]bool),
	}

	result, _ := m.handleConfirmDeleteKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	got := result.(*Model)

	if got.mode != modeBrowse {
		t.Fatalf("expected browse mode, got %v", got.mode)
	}
	if _, err := os.Stat(notePath); err != nil {
		t.Fatalf("expected file to remain, stat err: %v", err)
	}
}
