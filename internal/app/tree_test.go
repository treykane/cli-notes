package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchTreeItemsMatchesNamesAndMarkdownContent(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "Welcome.md"), "# Welcome\n\nGeneral notes.\n")
	mustWriteFile(t, filepath.Join(root, "Rocket-Plan.md"), "# Launch Plan\n\nTBD.\n")
	mustWriteFile(t, filepath.Join(root, "Projects", "Ideas.md"), "# Ideas\n\nBuild a rocket using Go.\n")
	mustWriteFile(t, filepath.Join(root, "Projects", "scratch.txt"), "contains rocket but should not match content search")

	results := searchTreeItems(root, "rocket")
	got := relPathSet(root, results)

	expectContains(t, got, "Rocket-Plan.md")
	expectContains(t, got, filepath.Join("Projects", "Ideas.md"))
	expectNotContains(t, got, filepath.Join("Projects", "scratch.txt"))
}

func TestSearchTreeItemsMatchesFoldersByName(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "Projects", "Ideas.md"), "# Ideas\n\nmisc\n")

	results := searchTreeItems(root, "project")
	got := relPathSet(root, results)

	expectContains(t, got, "Projects")
}

func TestSearchTreeItemsEmptyQueryReturnsNil(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "Welcome.md"), "hi\n")

	results := searchTreeItems(root, "   ")
	if results != nil {
		t.Fatalf("expected nil for empty query, got %v entries", len(results))
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func relPathSet(root string, items []treeItem) map[string]bool {
	out := make(map[string]bool, len(items))
	for _, item := range items {
		rel, err := filepath.Rel(root, item.path)
		if err != nil {
			continue
		}
		out[filepath.Clean(rel)] = true
	}
	return out
}

func expectContains(t *testing.T, paths map[string]bool, rel string) {
	t.Helper()
	if !paths[filepath.Clean(rel)] {
		t.Fatalf("expected %q in search results; got %v", rel, paths)
	}
}

func expectNotContains(t *testing.T, paths map[string]bool, rel string) {
	t.Helper()
	if paths[filepath.Clean(rel)] {
		t.Fatalf("did not expect %q in search results; got %v", rel, paths)
	}
}
