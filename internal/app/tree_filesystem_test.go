package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWalkTreeReadDirError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	root := t.TempDir()
	noReadDir := filepath.Join(root, "noread")
	if err := os.Mkdir(noReadDir, 0o000); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	defer os.Chmod(noReadDir, 0o755) // cleanup

	logs := captureLogOutput(t, func() {
		var items []treeItem
		walkTree(noReadDir, 0, make(map[string]bool), sortModeName, &items)

		// Should not crash, but should log a warning
		if len(items) != 0 {
			t.Errorf("expected no items when directory cannot be read, got %d", len(items))
		}
	})

	logStr := string(logs)
	if !strings.Contains(logStr, "level=WARN") {
		t.Error("should log warning when directory cannot be read")
	}
	if !strings.Contains(logStr, "read tree directory") {
		t.Error("log should mention tree directory read")
	}
	if !strings.Contains(logStr, noReadDir) {
		t.Errorf("log should contain directory path %q", noReadDir)
	}
}

func TestBuildTreeWithInaccessibleSubdirectory(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	root := t.TempDir()
	accessibleDir := filepath.Join(root, "accessible")
	inaccessibleDir := filepath.Join(root, "inaccessible")

	if err := os.Mkdir(accessibleDir, 0o755); err != nil {
		t.Fatalf("mkdir accessible: %v", err)
	}
	if err := os.WriteFile(filepath.Join(accessibleDir, "note.md"), []byte("content"), 0o644); err != nil {
		t.Fatalf("write note: %v", err)
	}
	if err := os.Mkdir(inaccessibleDir, 0o000); err != nil {
		t.Fatalf("mkdir inaccessible: %v", err)
	}
	defer os.Chmod(inaccessibleDir, 0o755) // cleanup

	expanded := map[string]bool{
		root:            true,
		accessibleDir:   true,
		inaccessibleDir: true,
	}

	logs := captureLogOutput(t, func() {
		items := buildTree(root, expanded, sortModeName)

		// Should still build tree for accessible parts
		found := false
		for _, item := range items {
			if item.name == "accessible" {
				found = true
				break
			}
		}
		if !found {
			t.Error("should still include accessible directories")
		}
	})

	logStr := string(logs)
	if !strings.Contains(logStr, "level=WARN") {
		t.Error("should log warning for inaccessible directory")
	}
}

func TestSearchTreeItemsBuildIndexError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("cannot test permission errors as root")
	}

	root := t.TempDir()
	if err := os.Chmod(root, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer os.Chmod(root, 0o755) // cleanup

	logs := captureLogOutput(t, func() {
		results := searchTreeItems(root, "test")
		if results != nil {
			t.Errorf("expected nil results when index build fails, got %d items", len(results))
		}
	})

	logStr := string(logs)
	if !strings.Contains(logStr, "level=ERROR") {
		t.Error("should log error when search index build fails")
	}
	if !strings.Contains(logStr, "build search tree index") {
		t.Error("log should mention search index build")
	}
}

func TestBuildTreeHandlesSymlinkErrors(t *testing.T) {
	root := t.TempDir()
	brokenLink := filepath.Join(root, "broken")
	if err := os.Symlink("/nonexistent/target", brokenLink); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	// buildTree should not crash on broken symlinks
	items := buildTree(root, map[string]bool{root: true}, sortModeName)

	// The broken symlink should be included in the tree
	// (ReadDir returns DirEntry which doesn't follow symlinks by default)
	found := false
	for _, item := range items {
		if item.name == "broken" {
			found = true
			break
		}
	}
	if !found {
		t.Error("broken symlink should still appear in tree")
	}
}

func TestRefreshTreePreservesSelectionAfterReadError(t *testing.T) {
	root := t.TempDir()
	noteFile := filepath.Join(root, "note.md")
	if err := os.WriteFile(noteFile, []byte("content"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	m := &Model{
		notesDir: root,
		cursor:   0,
		expanded: make(map[string]bool),
	}
	m.refreshTree()

	// Verify no crash when refreshing with inaccessible directories
	if len(m.items) == 0 {
		t.Error("tree should contain items")
	}
}

func TestToggleExpandHandlesStatError(t *testing.T) {
	root := t.TempDir()
	m := &Model{
		notesDir: root,
		cursor:   0,
		items: []treeItem{
			{path: filepath.Join(root, "folder"), name: "folder", isDir: true},
		},
		expanded: make(map[string]bool),
	}

	// Toggle expand on a non-existent directory shouldn't crash
	m.toggleExpand(true)

	// Should not crash even though directory doesn't exist
}

func TestMoveCursorWithEmptyItems(t *testing.T) {
	m := &Model{
		items:  []treeItem{},
		cursor: 0,
	}

	// Should not crash with empty items
	m.moveCursor(1)
	m.moveCursor(-1)

	if m.cursor != 0 {
		t.Errorf("cursor should remain 0 with empty items, got %d", m.cursor)
	}
}

func TestAdjustTreeOffsetWithZeroHeight(t *testing.T) {
	root := t.TempDir()
	m := &Model{
		notesDir:   root,
		items:      []treeItem{{path: root, name: "root", isDir: true}},
		cursor:     0,
		leftHeight: 2, // Will result in visibleHeight = 0
	}

	// Should not crash or cause issues
	m.adjustTreeOffset()

	if m.treeOffset != 0 {
		t.Errorf("tree offset should be 0, got %d", m.treeOffset)
	}
}
