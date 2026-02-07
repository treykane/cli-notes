package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReturnsErrNotConfiguredWhenMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, err := Load()
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected ErrNotConfigured, got %v", err)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := Config{NotesDir: "~/my-notes"}
	if err := Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	exists, err := Exists()
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if !exists {
		t.Fatal("expected config file to exist")
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	expected := filepath.Join(home, "my-notes")
	if loaded.NotesDir != expected {
		t.Fatalf("expected notes dir %q, got %q", expected, loaded.NotesDir)
	}
	if loaded.TreeSort != "name" {
		t.Fatalf("expected default tree sort %q, got %q", "name", loaded.TreeSort)
	}
	expectedTemplates := filepath.Join(home, ".cli-notes", "templates")
	if loaded.TemplatesDir != expectedTemplates {
		t.Fatalf("expected templates dir %q, got %q", expectedTemplates, loaded.TemplatesDir)
	}

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("config path: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("stat config path: %v", err)
	}
}

func TestSaveAndLoadWithCustomSortAndTemplatesDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := Config{
		NotesDir:     "~/my-notes",
		TreeSort:     "size",
		TemplatesDir: "~/my-templates",
	}
	if err := Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if loaded.TreeSort != "size" {
		t.Fatalf("expected tree sort %q, got %q", "size", loaded.TreeSort)
	}
	expectedTemplates := filepath.Join(home, "my-templates")
	if loaded.TemplatesDir != expectedTemplates {
		t.Fatalf("expected templates dir %q, got %q", expectedTemplates, loaded.TemplatesDir)
	}
}

func TestNormalizeNotesDirRejectsEmpty(t *testing.T) {
	if _, err := NormalizeNotesDir("   "); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestLoadMigratesLegacyNotesDirToDefaultWorkspace(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := Save(Config{NotesDir: "~/legacy-notes"}); err != nil {
		t.Fatalf("save config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if len(cfg.Workspaces) != 1 {
		t.Fatalf("expected one workspace, got %d", len(cfg.Workspaces))
	}
	if cfg.ActiveWorkspace == "" {
		t.Fatal("expected active workspace")
	}
	if cfg.Workspaces[0].NotesDir != cfg.NotesDir {
		t.Fatalf("workspace notes_dir %q should equal active notes_dir %q", cfg.Workspaces[0].NotesDir, cfg.NotesDir)
	}
}
