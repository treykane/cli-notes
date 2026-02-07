package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyMutationEffectsInvalidatesBeforePathOps(t *testing.T) {
	root := t.TempDir()
	existing := filepath.Join(root, "existing.md")
	newPath := filepath.Join(root, "new.md")
	mustWriteFile(t, existing, "# existing\n")

	idx := newSearchIndex(root)
	if err := idx.ensureBuilt(); err != nil {
		t.Fatalf("build search index: %v", err)
	}
	if _, ok := idx.docs[existing]; !ok {
		t.Fatalf("expected existing path in index before mutation")
	}
	mustWriteFile(t, newPath, "# new\n")

	m := &Model{searchIndex: idx}
	_ = m.applyMutationEffects(mutationEffects{
		invalidateSearch: true,
		removePaths:      []string{existing},
		upsertPaths:      []string{newPath},
	})

	if idx.ready {
		t.Fatal("expected index invalidated")
	}
	if _, ok := idx.docs[existing]; !ok {
		t.Fatal("expected existing doc to remain when path ops run after invalidation")
	}
	if _, ok := idx.docs[newPath]; ok {
		t.Fatal("did not expect upsert while invalidated")
	}
}

func TestApplyMutationEffectsRemoveThenUpsertAndIgnoreEmptyPaths(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "note.md")
	mustWriteFile(t, path, "# one\n")

	idx := newSearchIndex(root)
	if err := idx.ensureBuilt(); err != nil {
		t.Fatalf("build search index: %v", err)
	}

	mustWriteFile(t, path, "# two\n")
	m := &Model{searchIndex: idx}
	_ = m.applyMutationEffects(mutationEffects{
		removePaths: []string{"", path},
		upsertPaths: []string{"", path},
	})

	doc, ok := idx.docs[path]
	if !ok {
		t.Fatal("expected path to be present after remove-then-upsert")
	}
	if doc.contentLower != "# two\n" {
		t.Fatalf("expected re-indexed content, got %q", doc.contentLower)
	}
}

func TestApplyMutationEffectsSideEffectsAndSetCurrentFileCmd(t *testing.T) {
	root := t.TempDir()
	note := filepath.Join(root, "note.md")
	mustWriteFile(t, note, "# note\n")

	idx := newSearchIndex(root)
	if err := idx.ensureBuilt(); err != nil {
		t.Fatalf("build search index: %v", err)
	}

	m := &Model{
		notesDir: root,
		expanded: map[string]bool{root: true},
		items: []treeItem{
			{path: note, name: "note.md", isDir: false},
		},
		sortMode: sortModeName,
		renderCache: map[string]renderCacheEntry{
			note: {content: "cached"},
		},
		searchIndex: idx,
		git: gitRepoStatus{
			isRepo: true,
			branch: "main",
			dirty:  true,
		},
	}

	cmd := m.applyMutationEffects(mutationEffects{
		saveState:        true,
		clearRenderCache: true,
		refreshGit:       true,
		refreshTree:      true,
		rebuildKeepPath:  note,
		setCurrentFile:   note,
	})

	if cmd == nil {
		t.Fatal("expected non-nil command when setCurrentFile is provided")
	}
	if got := len(m.renderCache); got != 0 {
		t.Fatalf("expected render cache cleared, got %d entries", got)
	}
	if m.git.isRepo {
		t.Fatalf("expected refreshGitStatus to reset non-repo git state")
	}
	if len(m.items) == 0 {
		t.Fatal("expected refreshTree/rebuildKeepPath to populate items")
	}
	if got := m.selectedPath(); got != note {
		t.Fatalf("expected rebuildKeepPath to keep %q selected, got %q", note, got)
	}
	if _, err := os.Stat(appStatePath(root)); err != nil {
		t.Fatalf("expected app state file to be written: %v", err)
	}
}

func TestApplyMutationEffectsNoSetCurrentFileReturnsNilCmd(t *testing.T) {
	m := &Model{}
	if cmd := m.applyMutationEffects(mutationEffects{}); cmd != nil {
		t.Fatal("expected nil command when setCurrentFile is empty")
	}
}
