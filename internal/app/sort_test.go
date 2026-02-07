package app

import (
	"path/filepath"
	"testing"

	"github.com/treykane/cli-notes/internal/config"
)

func TestLoadWorkspaceSortModePrefersWorkspaceMap(t *testing.T) {
	cfg := config.Config{
		TreeSort: "name",
		TreeSortByWorkspace: map[string]string{
			"/notes/a": "modified",
		},
	}
	if got := loadWorkspaceSortMode(cfg, "/notes/a"); got != sortModeModified {
		t.Fatalf("expected modified, got %s", got)
	}
	if got := loadWorkspaceSortMode(cfg, "/notes/b"); got != sortModeName {
		t.Fatalf("expected fallback name, got %s", got)
	}
}

func TestSelectWorkspaceEntryRestoresWorkspaceSortMode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	notesA := filepath.Join(home, "notes-a")
	notesB := filepath.Join(home, "notes-b")
	mustWriteFile(t, filepath.Join(notesA, "a.md"), "a\n")
	mustWriteFile(t, filepath.Join(notesB, "b.md"), "b\n")

	cfg := config.Config{
		NotesDir: notesA,
		TreeSort: "name",
		Workspaces: []config.WorkspaceConfig{
			{Name: "A", NotesDir: notesA},
			{Name: "B", NotesDir: notesB},
		},
		ActiveWorkspace: "A",
		TreeSortByWorkspace: map[string]string{
			notesA: "name",
			notesB: "size",
		},
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}

	m := &Model{
		notesDir:        notesA,
		sortMode:        sortModeName,
		workspaces:      cfg.Workspaces,
		activeWorkspace: "A",
		workspaceCursor: 1,
		pinnedPaths:     map[string]bool{},
		notePositions:   map[string]notePosition{},
		noteOpenCounts:  map[string]int{},
		renderCache:     map[string]renderCacheEntry{},
		expanded:        map[string]bool{notesA: true},
	}
	model, _ := m.selectWorkspaceEntry()
	got := model.(*Model)
	if got.sortMode != sortModeSize {
		t.Fatalf("expected sort mode size for workspace B, got %s", got.sortMode)
	}
}
