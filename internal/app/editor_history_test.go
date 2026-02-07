package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func TestEditUndoRedoDiscreteFormatting(t *testing.T) {
	m := newFocusedEditModel("hello world")

	_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyCtrlB})
	if got := m.editor.Value(); got != "hello **world**" {
		t.Fatalf("expected formatted value, got %q", got)
	}

	_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyCtrlZ})
	if got := m.editor.Value(); got != "hello world" {
		t.Fatalf("expected undo to restore original value, got %q", got)
	}

	_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyCtrlY})
	if got := m.editor.Value(); got != "hello **world**" {
		t.Fatalf("expected redo to reapply format, got %q", got)
	}
}

func TestTypingBurstCoalescesIntoSingleUndoStep(t *testing.T) {
	m := newFocusedEditModel("x")

	_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})

	if got := len(m.editorUndo); got != 1 {
		t.Fatalf("expected one undo snapshot for typing burst, got %d", got)
	}

	_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyCtrlZ})
	if got := m.editor.Value(); got != "x" {
		t.Fatalf("expected undo to remove burst edits, got %q", got)
	}
}

func TestTypingBurstSplitsAfterIdleWindow(t *testing.T) {
	m := newFocusedEditModel("x")

	_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m.typingBurstLastInputAt = time.Now().Add(-typingBurstIdleWindow - time.Millisecond)
	_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})

	if got := len(m.editorUndo); got != 2 {
		t.Fatalf("expected two undo snapshots after idle split, got %d", got)
	}

	_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyCtrlZ})
	if got := m.editor.Value(); got != "xa" {
		t.Fatalf("expected first undo to remove only latest burst, got %q", got)
	}
}

func TestRedoClearsAfterFreshEdit(t *testing.T) {
	m := newFocusedEditModel("x")

	_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyCtrlZ})
	if got := len(m.editorRedo); got == 0 {
		t.Fatal("expected redo stack to contain one snapshot after undo")
	}

	_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	if got := len(m.editorRedo); got != 0 {
		t.Fatalf("expected redo stack cleared after fresh edit, got %d entries", got)
	}
}

func TestSaveResetsTypingBurstAndHistoryForNextEditSession(t *testing.T) {
	root := t.TempDir()
	notePath := filepath.Join(root, "note.md")
	if err := os.WriteFile(notePath, []byte("x\n"), 0o644); err != nil {
		t.Fatalf("write note: %v", err)
	}

	m := &Model{
		notesDir:      root,
		currentFile:   notePath,
		mode:          modeBrowse,
		expanded:      map[string]bool{root: true},
		notePositions: map[string]notePosition{},
		editor:        textarea.New(),
	}

	model, _ := m.startEditNote()
	m = model.(*Model)
	_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyCtrlS})
	if m.mode != modeBrowse {
		t.Fatalf("expected browse mode after save, got %v", m.mode)
	}

	model, _ = m.startEditNote()
	m = model.(*Model)
	_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	if got := len(m.editorUndo); got != 1 {
		t.Fatalf("expected fresh edit session with one typing snapshot, got %d", got)
	}
}
