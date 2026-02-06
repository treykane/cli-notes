package app

import (
	"os"
	"path/filepath"
	"slices"
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

func TestBuildTreeSortsAndRespectsExpandedDirectories(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "beta", "B.md"), "b\n")
	mustWriteFile(t, filepath.Join(root, "Alpha", "A.md"), "a\n")
	mustWriteFile(t, filepath.Join(root, "z.md"), "z\n")
	mustWriteFile(t, filepath.Join(root, "a.md"), "a\n")

	items := buildTree(root, map[string]bool{
		root:                         true,
		filepath.Join(root, "Alpha"): true,
	}, sortModeName)

	want := []string{
		"Alpha",
		filepath.Join("Alpha", "A.md"),
		"beta",
		"a.md",
		"z.md",
	}
	got := relPaths(root, items)
	if !slices.Equal(got, want) {
		t.Fatalf("unexpected tree order.\nwant: %v\ngot:  %v", want, got)
	}

	for _, item := range items {
		rel, err := filepath.Rel(root, item.path)
		if err != nil {
			t.Fatalf("rel path: %v", err)
		}
		rel = filepath.Clean(rel)
		switch rel {
		case "Alpha", "beta", "a.md", "z.md":
			if item.depth != 0 {
				t.Fatalf("expected depth 0 for %q, got %d", rel, item.depth)
			}
		case filepath.Join("Alpha", "A.md"):
			if item.depth != 1 {
				t.Fatalf("expected depth 1 for %q, got %d", rel, item.depth)
			}
		}
	}
}

func TestBuildTreeCollapsedDirectoryExcludesChildren(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "Docs", "Guide.md"), "guide\n")

	items := buildTree(root, map[string]bool{root: true}, sortModeName)
	got := relPathSet(root, items)

	expectContains(t, got, "Docs")
	expectNotContains(t, got, filepath.Join("Docs", "Guide.md"))
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

func relPaths(root string, items []treeItem) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		rel, err := filepath.Rel(root, item.path)
		if err != nil {
			continue
		}
		out = append(out, filepath.Clean(rel))
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
