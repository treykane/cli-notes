package app

import (
	"testing"
	"time"

	"github.com/treykane/cli-notes/internal/config"
)

func TestNewUsesConfiguredFileWatchInterval(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := config.Save(config.Config{
		NotesDir:                 t.TempDir(),
		FileWatchIntervalSeconds: 10,
	}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	m, err := New()
	if err != nil {
		t.Fatalf("new model: %v", err)
	}

	if got := m.fileWatchInterval; got != 10*time.Second {
		t.Fatalf("expected file watch interval 10s, got %s", got)
	}
}
