package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildTreePinnedItemsSortFirstWithinDirectory(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "a.md"), "a\n")
	mustWriteFile(t, filepath.Join(root, "z.md"), "z\n")

	items := buildTree(root, map[string]bool{root: true}, sortModeName, map[string]bool{
		filepath.Join(root, "z.md"): true,
	})

	got := relPaths(root, items)
	if len(got) < 2 {
		t.Fatalf("expected at least 2 items, got %v", got)
	}
	if got[0] != "z.md" {
		t.Fatalf("expected pinned file first, got %v", got)
	}
}

func TestParseMarkdownHeadingsIgnoresFencedBlocks(t *testing.T) {
	content := "# Top\n\n```md\n# Not Heading\n```\n\n## Real\n"
	headings := parseMarkdownHeadings(content)
	if len(headings) != 2 {
		t.Fatalf("expected 2 headings, got %d", len(headings))
	}
	if headings[0].Title != "Top" || headings[1].Title != "Real" {
		t.Fatalf("unexpected headings: %+v", headings)
	}
}

func TestAppStateRoundTrip(t *testing.T) {
	root := t.TempDir()
	note := filepath.Join(root, "note.md")
	mustWriteFile(t, note, "hello\n")

	m := &Model{
		notesDir:       root,
		recentFiles:    []string{note},
		pinnedPaths:    map[string]bool{note: true},
		noteOpenCounts: map[string]int{note: 5},
		notePositions: map[string]notePosition{
			note: {PreviewOffset: 8, PrimaryPreviewOffset: 8, SecondaryPreviewOffset: 3, EditorCursor: 12},
		},
	}
	m.saveAppState()

	state, err := loadAppState(root)
	if err != nil {
		t.Fatalf("load app state: %v", err)
	}
	if len(state.RecentFiles) != 1 || state.RecentFiles[0] != note {
		t.Fatalf("unexpected recents: %+v", state.RecentFiles)
	}
	if !state.PinnedPaths[note] {
		t.Fatalf("expected pinned path %q", note)
	}
	if got := state.Positions[note]; got.PrimaryPreviewOffset != 8 || got.SecondaryPreviewOffset != 3 || got.EditorCursor != 12 {
		t.Fatalf("unexpected position: %+v", got)
	}
	if state.OpenCounts[note] != 5 {
		t.Fatalf("unexpected open count: %+v", state.OpenCounts)
	}
}

func TestLoadAppStateMigratesLegacyPreviewOffset(t *testing.T) {
	root := t.TempDir()
	note := filepath.Join(root, "legacy.md")
	mustWriteFile(t, note, "legacy\n")
	rel := "legacy.md"

	raw := persistedState{
		Positions: map[string]notePosition{
			rel: {PreviewOffset: 7, EditorCursor: 2},
		},
	}
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	path := appStatePath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir state dir: %v", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatalf("write state: %v", err)
	}

	state, err := loadAppState(root)
	if err != nil {
		t.Fatalf("load app state: %v", err)
	}
	got := state.Positions[note]
	if got.PrimaryPreviewOffset != 7 {
		t.Fatalf("expected migrated primary offset 7, got %+v", got)
	}
}

func TestRestorePaneOffsetKeepsPrimaryAndSecondaryIndependent(t *testing.T) {
	path := "/tmp/note.md"
	m := &Model{
		notePositions: map[string]notePosition{
			path: {PrimaryPreviewOffset: 11, SecondaryPreviewOffset: 29},
		},
	}
	if got := m.restorePaneOffset(path, false); got != 11 {
		t.Fatalf("expected primary offset 11, got %d", got)
	}
	if got := m.restorePaneOffset(path, true); got != 29 {
		t.Fatalf("expected secondary offset 29, got %d", got)
	}
}

func TestSetFocusedFilePreservesSecondaryPaneOffset(t *testing.T) {
	root := t.TempDir()
	oldPath := filepath.Join(root, "old.md")
	newPath := filepath.Join(root, "new.md")
	mustWriteFile(t, oldPath, "old\n")
	mustWriteFile(t, newPath, "new\n")

	m := &Model{
		notesDir:            root,
		splitMode:           true,
		splitFocusSecondary: true,
		secondaryFile:       oldPath,
		notePositions: map[string]notePosition{
			oldPath: {SecondaryPreviewOffset: 14},
		},
		noteOpenCounts: map[string]int{},
	}
	_ = m.setFocusedFile(newPath)
	if got := m.restorePaneOffset(oldPath, true); got != 14 {
		t.Fatalf("expected old secondary offset preserved, got %d", got)
	}
	if m.secondaryFile != newPath {
		t.Fatalf("expected secondary file switched to %q, got %q", newPath, m.secondaryFile)
	}
}

func TestScanFileWatchSnapshotSkipsManagedDir(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "visible.md"), "ok\n")
	mustWriteFile(t, filepath.Join(root, managedNotesDirName, "state.json"), "{}\n")

	snap, err := scanFileWatchSnapshot(root)
	if err != nil {
		t.Fatalf("scan snapshot: %v", err)
	}
	if _, ok := snap[filepath.Join(root, managedNotesDirName, "state.json")]; ok {
		t.Fatal("managed path should not be watched")
	}
	if _, ok := snap[filepath.Join(root, "visible.md")]; !ok {
		t.Fatal("expected visible.md in snapshot")
	}
}

func TestSelectRecentEntryDropsMissingFile(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "missing.md")

	m := &Model{notesDir: root, recentFiles: []string{missing}}
	m.rebuildRecentEntries()
	m.recentEntries = []string{missing}
	m.showRecentPopup = true
	_, _ = m.selectRecentEntry()

	if len(m.recentFiles) != 0 {
		t.Fatalf("expected missing recent file to be removed, got %v", m.recentFiles)
	}
	if m.showRecentPopup != true {
		// popup stays open so user can choose another item
		t.Fatalf("expected popup to stay open")
	}
	if _, err := os.Stat(appStatePath(root)); err != nil {
		t.Fatalf("expected app state file to be written: %v", err)
	}
}
