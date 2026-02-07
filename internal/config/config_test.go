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
	if loaded.ThemePreset != ThemePresetOceanCitrus {
		t.Fatalf("expected default theme preset %q, got %q", ThemePresetOceanCitrus, loaded.ThemePreset)
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

func TestSaveAndLoadWithWorkspaceSortMap(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	notesA := filepath.Join(home, "notes-a")
	notesB := filepath.Join(home, "notes-b")

	cfg := Config{
		NotesDir: notesA,
		TreeSort: "name",
		Workspaces: []WorkspaceConfig{
			{Name: "A", NotesDir: notesA},
			{Name: "B", NotesDir: notesB},
		},
		ActiveWorkspace: "A",
		TreeSortByWorkspace: map[string]string{
			notesA: "modified",
			notesB: "size",
		},
	}
	if err := Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if loaded.TreeSortByWorkspace[notesA] != "modified" {
		t.Fatalf("expected workspace sort %q, got %q", "modified", loaded.TreeSortByWorkspace[notesA])
	}
	if loaded.TreeSortByWorkspace[notesB] != "size" {
		t.Fatalf("expected workspace sort %q, got %q", "size", loaded.TreeSortByWorkspace[notesB])
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

func TestLoadKeepsTreeSortFallbackWhenWorkspaceMapMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	notes := filepath.Join(home, "notes")
	cfg := Config{
		NotesDir: notes,
		TreeSort: "modified",
		Workspaces: []WorkspaceConfig{
			{Name: "default", NotesDir: notes},
		},
		ActiveWorkspace: "default",
	}
	if err := Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if loaded.TreeSort != "modified" {
		t.Fatalf("expected tree_sort fallback to remain modified, got %q", loaded.TreeSort)
	}
	if len(loaded.TreeSortByWorkspace) != 0 {
		t.Fatalf("expected empty workspace sort map, got %v", loaded.TreeSortByWorkspace)
	}
}

func TestLoadIgnoresInvalidWorkspaceSortEntries(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("config path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data := `{
  "notes_dir": "~/notes",
  "tree_sort": "created",
  "tree_sort_by_workspace": {
    "~/notes": "modified",
    "": "size",
    "~/notes-2": "bad-value"
  }
}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	notes := filepath.Join(home, "notes")
	if cfg.TreeSortByWorkspace[notes] != "modified" {
		t.Fatalf("expected valid workspace sort for %q", notes)
	}
	if len(cfg.TreeSortByWorkspace) != 1 {
		t.Fatalf("expected only valid workspace sort entries, got %v", cfg.TreeSortByWorkspace)
	}
	if cfg.TreeSort != "created" {
		t.Fatalf("expected fallback tree_sort to stay %q, got %q", "created", cfg.TreeSort)
	}
}

func TestLoadNormalizesThemePreset(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("config path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data := `{
  "notes_dir": "~/notes",
  "theme_preset": "Neon-Slate"
}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.ThemePreset != ThemePresetNeonSlate {
		t.Fatalf("expected normalized theme %q, got %q", ThemePresetNeonSlate, cfg.ThemePreset)
	}
}

func TestLoadFallsBackToDefaultThemePresetOnInvalidValue(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("config path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data := `{
  "notes_dir": "~/notes",
  "theme_preset": "bogus-theme"
}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.ThemePreset != ThemePresetOceanCitrus {
		t.Fatalf("expected fallback theme %q, got %q", ThemePresetOceanCitrus, cfg.ThemePreset)
	}
}
