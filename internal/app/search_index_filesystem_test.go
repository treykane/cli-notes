package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchIndexWalkReadDirError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	root := t.TempDir()
	noReadDir := filepath.Join(root, "noread")
	if err := os.Mkdir(noReadDir, 0o000); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	defer os.Chmod(noReadDir, 0o755) // cleanup

	idx := newSearchIndex(noReadDir)
	err := idx.build()
	if err == nil {
		t.Fatal("expected error when building index from unreadable directory")
	}

	if !strings.Contains(err.Error(), "read search dir") {
		t.Errorf("error should mention search dir read, got: %v", err)
	}

	if idx.ready {
		t.Error("index should not be marked ready after build error")
	}
}

func TestSearchIndexWalkSubdirectoryError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	root := t.TempDir()
	subdir := filepath.Join(root, "subdir")
	if err := os.Mkdir(subdir, 0o000); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	defer os.Chmod(subdir, 0o755) // cleanup

	idx := newSearchIndex(root)
	err := idx.build()
	if err == nil {
		t.Fatal("expected error when walking into unreadable subdirectory")
	}

	if !strings.Contains(err.Error(), "read search dir") {
		t.Errorf("error should mention search dir read, got: %v", err)
	}
}

func TestSearchIndexReadLowerMarkdownContentStatError(t *testing.T) {
	// Non-existent file should return empty string, not crash
	content := readLowerMarkdownContent("/nonexistent/file.md")
	if content != "" {
		t.Errorf("expected empty string for non-existent file, got: %q", content)
	}
}

func TestSearchIndexReadLowerMarkdownContentReadError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	root := t.TempDir()
	noReadFile := filepath.Join(root, "noread.md")
	if err := os.WriteFile(noReadFile, []byte("secret"), 0o000); err != nil {
		t.Fatalf("write file: %v", err)
	}
	defer os.Chmod(noReadFile, 0o644) // cleanup

	content := readLowerMarkdownContent(noReadFile)
	if content != "" {
		t.Errorf("expected empty string when file cannot be read, got: %q", content)
	}
}

func TestSearchIndexReadLowerMarkdownContentLargeFile(t *testing.T) {
	root := t.TempDir()
	largeFile := filepath.Join(root, "large.md")

	// Create a file larger than MaxSearchFileBytes
	largeContent := make([]byte, MaxSearchFileBytes+1)
	for i := range largeContent {
		largeContent[i] = 'a'
	}
	if err := os.WriteFile(largeFile, largeContent, 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	content := readLowerMarkdownContent(largeFile)
	if content != "" {
		t.Error("expected empty string for file exceeding max size")
	}
}

func TestSearchIndexReadLowerMarkdownContentNonMarkdown(t *testing.T) {
	root := t.TempDir()
	txtFile := filepath.Join(root, "file.txt")
	if err := os.WriteFile(txtFile, []byte("content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	content := readLowerMarkdownContent(txtFile)
	if content != "" {
		t.Error("expected empty string for non-markdown file")
	}
}

func TestSearchIndexReadLowerMarkdownContentDirectory(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "folder.md")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	content := readLowerMarkdownContent(dir)
	if content != "" {
		t.Error("expected empty string for directory even with .md extension")
	}
}

func TestSearchIndexUpsertPathStatError(t *testing.T) {
	root := t.TempDir()
	idx := newSearchIndex(root)
	idx.ready = true

	nonExistentPath := filepath.Join(root, "nonexistent.md")
	idx.docs[nonExistentPath] = searchDoc{
		item: treeItem{path: nonExistentPath, name: "nonexistent.md"},
	}

	// Should remove path when stat fails
	idx.upsertPath(nonExistentPath)

	if _, exists := idx.docs[nonExistentPath]; exists {
		t.Error("path should be removed when stat fails")
	}
}

func TestSearchIndexUpsertPathOutsideRoot(t *testing.T) {
	root := t.TempDir()
	idx := newSearchIndex(root)
	idx.ready = true

	outsidePath := "/tmp/outside.md"
	initialCount := len(idx.docs)

	// Should not upsert path outside root
	idx.upsertPath(outsidePath)

	if len(idx.docs) != initialCount {
		t.Error("should not upsert path outside root")
	}
}

func TestSearchIndexUpsertPathWhenNotReady(t *testing.T) {
	root := t.TempDir()
	idx := newSearchIndex(root)
	idx.ready = false

	testPath := filepath.Join(root, "test.md")
	if err := os.WriteFile(testPath, []byte("content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Should not update when not ready
	idx.upsertPath(testPath)

	if len(idx.docs) != 0 {
		t.Error("should not update docs when index is not ready")
	}
}

func TestSearchIndexUpsertDirectoryWalkError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	root := t.TempDir()
	dir := filepath.Join(root, "folder")
	accessible := filepath.Join(dir, "accessible.md")
	subdir := filepath.Join(dir, "noread")

	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(accessible, []byte("content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.Mkdir(subdir, 0o000); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}
	defer os.Chmod(subdir, 0o755) // cleanup

	idx := newSearchIndex(root)
	idx.ready = true

	logs := captureLogOutput(t, func() {
		// Upsert the parent directory which will try to walk the inaccessible subdir
		idx.upsertPath(dir)
	})

	logStr := string(logs)
	if !strings.Contains(logStr, "level=WARN") {
		t.Error("should log warning when updating descendants fails")
	}
	if !strings.Contains(logStr, "update search descendants") {
		t.Error("log should mention updating descendants")
	}
}

func TestSearchIndexRemovePathWhenNotReady(t *testing.T) {
	root := t.TempDir()
	idx := newSearchIndex(root)
	idx.ready = false
	idx.docs["test"] = searchDoc{}

	// Should not remove when not ready
	idx.removePath("test")

	if len(idx.docs) != 1 {
		t.Error("should not remove docs when index is not ready")
	}
}

func TestSearchIndexRemoveDescendants(t *testing.T) {
	root := t.TempDir()
	idx := newSearchIndex(root)

	parentPath := filepath.Join(root, "parent")
	childPath := filepath.Join(parentPath, "child.md")
	grandchildPath := filepath.Join(parentPath, "subfolder", "grandchild.md")
	unrelatedPath := filepath.Join(root, "unrelated.md")

	idx.docs[parentPath] = searchDoc{item: treeItem{path: parentPath}}
	idx.docs[childPath] = searchDoc{item: treeItem{path: childPath}}
	idx.docs[grandchildPath] = searchDoc{item: treeItem{path: grandchildPath}}
	idx.docs[unrelatedPath] = searchDoc{item: treeItem{path: unrelatedPath}}

	idx.removeDescendants(parentPath)

	if _, exists := idx.docs[childPath]; exists {
		t.Error("child path should be removed")
	}
	if _, exists := idx.docs[grandchildPath]; exists {
		t.Error("grandchild path should be removed")
	}
	if _, exists := idx.docs[unrelatedPath]; !exists {
		t.Error("unrelated path should not be removed")
	}
	if _, exists := idx.docs[parentPath]; !exists {
		t.Error("parent path itself should not be removed by removeDescendants")
	}
}

func TestSearchIndexEnsureBuiltSkipsWhenReady(t *testing.T) {
	root := t.TempDir()
	idx := newSearchIndex(root)
	idx.ready = true

	// Mark a file that would cause error if walked
	idx.root = "/nonexistent/path"

	// Should not rebuild if already ready
	err := idx.ensureBuilt()
	if err != nil {
		t.Errorf("ensureBuilt should not rebuild when already ready, got error: %v", err)
	}
}

func TestSearchIndexInvalidate(t *testing.T) {
	root := t.TempDir()
	idx := newSearchIndex(root)
	idx.ready = true

	idx.invalidate()

	if idx.ready {
		t.Error("index should not be ready after invalidation")
	}
}

func TestSearchIndexSearchEmptyQuery(t *testing.T) {
	root := t.TempDir()
	idx := newSearchIndex(root)
	idx.docs["test"] = searchDoc{
		item:      treeItem{path: "test", name: "test"},
		nameLower: "test",
	}

	results := idx.search("")
	if results != nil {
		t.Error("search with empty query should return nil")
	}

	results = idx.search("   ")
	if results != nil {
		t.Error("search with whitespace-only query should return nil")
	}
}

func TestDepthFromRootEdgeCases(t *testing.T) {
	root := t.TempDir()

	// Same path as root
	depth := depthFromRoot(root, root)
	if depth != 0 {
		t.Errorf("depth from root to itself should be 0, got %d", depth)
	}

	// Direct child
	child := filepath.Join(root, "child")
	depth = depthFromRoot(root, child)
	if depth != 0 {
		t.Errorf("depth from root to direct child should be 0, got %d", depth)
	}

	// Nested child
	nested := filepath.Join(root, "a", "b", "c")
	depth = depthFromRoot(root, nested)
	if depth != 2 {
		t.Errorf("depth from root to a/b/c should be 2, got %d", depth)
	}
}

func TestIsWithinRootEdgeCases(t *testing.T) {
	root := t.TempDir()

	// Same path
	if !isWithinRoot(root, root) {
		t.Error("root should be within itself")
	}

	// Direct child
	child := filepath.Join(root, "child")
	if !isWithinRoot(root, child) {
		t.Error("direct child should be within root")
	}

	// Parent directory
	parent := filepath.Dir(root)
	if isWithinRoot(root, parent) {
		t.Error("parent directory should not be within root")
	}

	// Sibling directory
	sibling := filepath.Join(filepath.Dir(root), "sibling")
	if isWithinRoot(root, sibling) {
		t.Error("sibling directory should not be within root")
	}

	// Path with .. that escapes
	escaped := filepath.Join(root, "..", "escaped")
	if isWithinRoot(root, escaped) {
		t.Error("escaped path should not be within root")
	}
}
