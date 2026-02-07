package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
)

func TestRenderSearchPopupShowsMatchCountAndPosition(t *testing.T) {
	m := &Model{
		search: textinput.New(),
		searchResults: []treeItem{
			{path: "/tmp/notes/a.md", name: "a.md"},
			{path: "/tmp/notes/b.md", name: "b.md"},
		},
		searchResultCursor: 1,
		notesDir:           "/tmp/notes",
	}
	m.search.SetValue("a")

	out := m.renderSearchPopup(60, 12)
	if !strings.Contains(out, "2 matches") {
		t.Fatalf("expected total match count in popup, got %q", out)
	}
	if !strings.Contains(out, "2 of 2") {
		t.Fatalf("expected selection position in popup, got %q", out)
	}
}

func TestRenderSearchPopupShowsZeroMatchCountForQuery(t *testing.T) {
	m := &Model{
		search:   textinput.New(),
		notesDir: "/tmp/notes",
	}
	m.search.SetValue("missing")

	out := m.renderSearchPopup(60, 10)
	if !strings.Contains(out, "0 matches") {
		t.Fatalf("expected zero-match indicator in popup, got %q", out)
	}
}
