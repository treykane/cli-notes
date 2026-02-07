package app

import (
	"testing"

	"github.com/treykane/cli-notes/internal/config"
)

func TestActionForKeySupportsDefaultBrowseAliases(t *testing.T) {
	m := &Model{}
	m.loadKeybindings(config.Config{})

	cases := map[string]string{
		"up":      actionCursorUp,
		"k":       actionCursorUp,
		"down":    actionCursorDown,
		"ctrl+n":  actionCursorDown,
		"enter":   actionExpandToggle,
		"right":   actionExpandToggle,
		"l":       actionExpandToggle,
		"left":    actionCollapse,
		"h":       actionCollapse,
		"g":       actionJumpTop,
		"G":       actionJumpBottom,
		"shift+r": actionRefresh,
		"ctrl+c":  actionQuit,
	}
	for key, want := range cases {
		if got := m.actionForKey(key); got != want {
			t.Fatalf("actionForKey(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestLoadKeybindingsOverrideReplacesDefaultAliases(t *testing.T) {
	m := &Model{}
	m.loadKeybindings(config.Config{
		Keybindings: map[string]string{
			actionCursorDown: "alt+j",
		},
	})

	if got := m.actionForKey("alt+j"); got != actionCursorDown {
		t.Fatalf("expected override key to map to cursor down, got %q", got)
	}
	if got := m.actionForKey("down"); got != "" {
		t.Fatalf("expected default alias 'down' to be replaced, got %q", got)
	}
	if got := m.actionForKey("j"); got != "" {
		t.Fatalf("expected default alias 'j' to be replaced, got %q", got)
	}
	if got := m.actionForKey("ctrl+n"); got != "" {
		t.Fatalf("expected default alias 'ctrl+n' to be replaced, got %q", got)
	}
}
