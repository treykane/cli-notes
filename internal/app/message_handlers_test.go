package app

import (
	"os"
	"path/filepath"
	"strings"
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

func TestHandleEditNoteKeyAltXWrapsCurrentWord(t *testing.T) {
	m := newFocusedEditModel("hello world")

	result, _ := m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}, Alt: true})
	got := result.(*Model)

	if got.editor.Value() != "hello ~~world~~" {
		t.Fatalf("expected value %q, got %q", "hello ~~world~~", got.editor.Value())
	}
}

func TestHandleEditNoteKeyCtrlKWrapsCurrentWordWithMarkdownLink(t *testing.T) {
	m := newFocusedEditModel("hello world")

	result, _ := m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyCtrlK})
	got := result.(*Model)

	if got.editor.Value() != "hello [world](url)" {
		t.Fatalf("expected value %q, got %q", "hello [world](url)", got.editor.Value())
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

func TestHandleEditNoteKeyUnshiftedArrowClearsSelectionAnchor(t *testing.T) {
	m := newFocusedEditModel("hello world")

	for i := 0; i < 5; i++ {
		_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyShiftLeft})
	}
	if !m.editorSelectionActive {
		t.Fatal("expected selection active after shift selection")
	}

	_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyLeft})
	if m.editorSelectionActive {
		t.Fatal("expected unshifted cursor movement to clear selection")
	}
	if m.editorSelectionAnchor != noEditorSelectionAnchor {
		t.Fatalf("expected anchor cleared, got %d", m.editorSelectionAnchor)
	}
	if got := m.status; got != "Selection cleared" {
		t.Fatalf("expected status %q, got %q", "Selection cleared", got)
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

func TestToggleHeadingAppliesAndRemovesHeading(t *testing.T) {
	m := newFocusedEditModel("hello world")
	m.setEditorValueAndCursorOffset("hello world", 5)

	m.toggleHeading(2)
	if got := m.editor.Value(); got != "## hello world" {
		t.Fatalf("expected heading applied, got %q", got)
	}

	m.toggleHeading(2)
	if got := m.editor.Value(); got != "hello world" {
		t.Fatalf("expected heading removed, got %q", got)
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

func TestFormattingRoundTripForBoldItalicUnderline(t *testing.T) {
	cases := []struct {
		name    string
		initial string
		key     tea.KeyMsg
	}{
		{name: "bold", initial: "hello world", key: tea.KeyMsg{Type: tea.KeyCtrlB}},
		{name: "italic", initial: "hello world", key: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}, Alt: true}},
		{name: "underline", initial: "hello world", key: tea.KeyMsg{Type: tea.KeyCtrlU}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newFocusedEditModel(tc.initial)
			before := m.editor.Value()
			_, _ = m.handleEditNoteKey(tc.key)
			first := m.editor.Value()
			switch tc.name {
			case "bold":
				m.setEditorValueAndCursorOffset(first, strings.Index(first, "world")+2)
			case "italic":
				m.setEditorValueAndCursorOffset(first, strings.Index(first, "world")+2)
			case "underline":
				m.setEditorValueAndCursorOffset(first, strings.Index(first, "world")+2)
			}
			_, _ = m.handleEditNoteKey(tc.key)
			if got := m.editor.Value(); got != before {
				t.Fatalf("expected round-trip to restore original. want=%q got=%q", before, got)
			}
		})
	}
}

func TestHandleInputModeKeyEscCancelsToBrowse(t *testing.T) {
	m := &Model{
		mode: modeNewNote,
	}
	gotModel, _ := m.handleInputModeKey(tea.KeyMsg{Type: tea.KeyEsc}, func() (tea.Model, tea.Cmd) {
		t.Fatal("save should not be called on esc")
		return m, nil
	}, "New note cancelled")
	got := gotModel.(*Model)
	if got.mode != modeBrowse {
		t.Fatalf("expected modeBrowse, got %v", got.mode)
	}
	if got.status != "New note cancelled" {
		t.Fatalf("expected cancel status, got %q", got.status)
	}
}

func TestHandleInputModeKeyEnterCallsSave(t *testing.T) {
	m := &Model{mode: modeNewFolder}
	called := false
	gotModel, _ := m.handleInputModeKey(tea.KeyMsg{Type: tea.KeyEnter}, func() (tea.Model, tea.Cmd) {
		called = true
		return m, nil
	}, "ignored")
	_ = gotModel.(*Model)
	if !called {
		t.Fatal("expected save callback to be called")
	}
}

func TestFormattingNestedToggleItalicAndUnderline(t *testing.T) {
	t.Run("italic inside bold", func(t *testing.T) {
		m := newFocusedEditModel("***word***")
		m.editorSelectionAnchor = 3
		m.editorSelectionActive = true
		m.setEditorValueAndCursorOffset("***word***", 7)
		_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}, Alt: true})
		if got := m.editor.Value(); got != "**word**" {
			t.Fatalf("expected nested italic removed only, got %q", got)
		}
	})

	t.Run("underline around bold", func(t *testing.T) {
		m := newFocusedEditModel("<u>**word**</u>")
		start := strings.Index(m.editor.Value(), "**word**")
		if start < 0 {
			t.Fatal("missing bold content")
		}
		m.editorSelectionAnchor = start
		m.editorSelectionActive = true
		m.setEditorValueAndCursorOffset(m.editor.Value(), start+len("**word**"))
		_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyCtrlU})
		if got := m.editor.Value(); got != "**word**" {
			t.Fatalf("expected only underline removed, got %q", got)
		}
	})
}

func TestFormattingPartialOverlapWrapsInsteadOfUnwrap(t *testing.T) {
	cases := []struct {
		name   string
		key    tea.KeyMsg
		open   string
		close  string
		expect string
	}{
		{
			name:   "bold partial overlap",
			key:    tea.KeyMsg{Type: tea.KeyCtrlB},
			open:   "**",
			close:  "**",
			expect: "****he**llo**",
		},
		{
			name:   "italic partial overlap",
			key:    tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}, Alt: true},
			open:   "*",
			close:  "*",
			expect: "**he*llo*",
		},
		{
			name:   "underline partial overlap",
			key:    tea.KeyMsg{Type: tea.KeyCtrlU},
			open:   "<u>",
			close:  "</u>",
			expect: "<u><u>he</u>llo</u>",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := tc.open + "hello" + tc.close
			m := newFocusedEditModel(source)
			contentStart := len([]rune(tc.open))
			m.editorSelectionAnchor = contentStart
			m.editorSelectionActive = true
			m.setEditorValueAndCursorOffset(source, contentStart+2)
			_, _ = m.handleEditNoteKey(tc.key)
			if got := m.editor.Value(); got != tc.expect {
				t.Fatalf("expected partial overlap to wrap, got %q", got)
			}
		})
	}
}

func TestFormattingCursorBoundaryTargetsWord(t *testing.T) {
	t.Run("cursor before word", func(t *testing.T) {
		m := newFocusedEditModel("hello world")
		m.setEditorValueAndCursorOffset("hello world", 6)
		_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyCtrlB})
		if got := m.editor.Value(); got != "hello **world**" {
			t.Fatalf("expected boundary wrap, got %q", got)
		}
	})

	t.Run("cursor at end of word", func(t *testing.T) {
		m := newFocusedEditModel("hello world")
		m.setEditorValueAndCursorOffset("hello world", 11)
		_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}, Alt: true})
		if got := m.editor.Value(); got != "hello *world*" {
			t.Fatalf("expected boundary wrap at word end, got %q", got)
		}
	})
}

func TestFormattingEmptySelectionFallsBack(t *testing.T) {
	t.Run("empty selection on word toggles word", func(t *testing.T) {
		m := newFocusedEditModel("hello world")
		m.editorSelectionAnchor = 6
		m.editorSelectionActive = true
		m.setEditorValueAndCursorOffset("hello world", 6)
		_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyCtrlU})
		if got := m.editor.Value(); got != "hello <u>world</u>" {
			t.Fatalf("expected word fallback, got %q", got)
		}
	})

	t.Run("empty selection on whitespace inserts markers", func(t *testing.T) {
		m := newFocusedEditModel("hello ")
		m.editorSelectionAnchor = 6
		m.editorSelectionActive = true
		m.setEditorValueAndCursorOffset("hello ", 6)
		_, _ = m.handleEditNoteKey(tea.KeyMsg{Type: tea.KeyCtrlB})
		if got := m.editor.Value(); got != "hello ****" {
			t.Fatalf("expected marker insertion fallback, got %q", got)
		}
	})
}
