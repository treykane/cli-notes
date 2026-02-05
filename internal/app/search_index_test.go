package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchIndexBuildAndSearch(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "Alpha.md"), "project zeus\n")
	mustWriteFile(t, filepath.Join(root, "Docs", "Guide.md"), "no match\n")
	mustWriteFile(t, filepath.Join(root, "Docs", "notes.txt"), "project zeus\n")

	idx := newSearchIndex(root)
	if err := idx.ensureBuilt(); err != nil {
		t.Fatalf("build index: %v", err)
	}

	got := relPathSet(root, idx.search("zeus"))
	expectContains(t, got, "Alpha.md")
	expectNotContains(t, got, filepath.Join("Docs", "notes.txt"))
}

func TestSearchIndexUpsertPathUpdatesChangedContent(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Work.md")
	mustWriteFile(t, path, "alpha only\n")

	idx := newSearchIndex(root)
	if err := idx.ensureBuilt(); err != nil {
		t.Fatalf("build index: %v", err)
	}
	if len(idx.search("beta")) != 0 {
		t.Fatal("expected no beta match before update")
	}

	mustWriteFile(t, path, "alpha and beta\n")
	idx.upsertPath(path)

	got := relPathSet(root, idx.search("beta"))
	expectContains(t, got, "Work.md")
}

func TestSearchIndexRemovePathRemovesMatches(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "DeleteMe.md")
	mustWriteFile(t, path, "needle\n")

	idx := newSearchIndex(root)
	if err := idx.ensureBuilt(); err != nil {
		t.Fatalf("build index: %v", err)
	}
	if len(idx.search("needle")) != 1 {
		t.Fatal("expected one match before delete")
	}

	if err := os.Remove(path); err != nil {
		t.Fatalf("remove file: %v", err)
	}
	idx.removePath(path)
	if len(idx.search("needle")) != 0 {
		t.Fatal("expected no matches after delete")
	}
}
