package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func lineBlock(count int) string {
	lines := make([]string, 0, count)
	for i := 1; i <= count; i++ {
		lines = append(lines, "line")
	}
	return strings.Join(lines, "\n")
}

func TestHandleBrowseKeyPreviewScrollPageClampsAndPersistsPrimary(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "note.md")
	mustWriteFile(t, path, "x\n")

	vp := viewport.New(80, 10)
	vp.SetContent(lineBlock(100))
	vp.YOffset = 95

	m := &Model{
		notesDir:      root,
		currentFile:   path,
		viewport:      vp,
		notePositions: map[string]notePosition{},
		keyToAction: map[string]string{
			"pgup":   actionPreviewScrollPageUp,
			"pgdown": actionPreviewScrollPageDown,
		},
	}

	_, _ = m.handleBrowseKey("pgdown")
	if got := m.viewport.YOffset; got != 99 {
		t.Fatalf("expected pgdown clamp to 99, got %d", got)
	}
	if got := m.notePositions[path].PrimaryPreviewOffset; got != 99 {
		t.Fatalf("expected stored primary offset 99, got %d", got)
	}

	m.viewport.YOffset = 2
	m.setPaneOffset(path, false, 2)
	_, _ = m.handleBrowseKey("pgup")
	if got := m.viewport.YOffset; got != 0 {
		t.Fatalf("expected pgup clamp to 0, got %d", got)
	}
	if got := m.notePositions[path].PrimaryPreviewOffset; got != 0 {
		t.Fatalf("expected stored primary offset 0, got %d", got)
	}

	if _, err := os.Stat(appStatePath(root)); err != nil {
		t.Fatalf("expected app state to be saved, got err: %v", err)
	}
}

func TestHandleBrowseKeyPreviewScrollHalfPageUsesViewportHeight(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "note.md")
	mustWriteFile(t, path, "x\n")

	vp := viewport.New(80, 9)
	vp.SetContent(lineBlock(40))
	vp.YOffset = 8

	m := &Model{
		notesDir:      root,
		currentFile:   path,
		viewport:      vp,
		notePositions: map[string]notePosition{},
		keyToAction: map[string]string{
			"ctrl+u": actionPreviewScrollHalfUp,
			"ctrl+d": actionPreviewScrollHalfDown,
		},
	}

	_, _ = m.handleBrowseKey("ctrl+u")
	if got := m.viewport.YOffset; got != 4 {
		t.Fatalf("expected ctrl+u to move half-page up to 4, got %d", got)
	}

	_, _ = m.handleBrowseKey("ctrl+d")
	if got := m.viewport.YOffset; got != 8 {
		t.Fatalf("expected ctrl+d to move half-page down to 8, got %d", got)
	}
}

func TestHandleBrowseKeyPreviewScrollNoSelectionNoOp(t *testing.T) {
	vp := viewport.New(80, 10)
	vp.SetContent(lineBlock(20))
	vp.YOffset = 5

	m := &Model{
		viewport: vp,
		keyToAction: map[string]string{
			"pgdown": actionPreviewScrollPageDown,
		},
	}

	_, _ = m.handleBrowseKey("pgdown")
	if got := m.viewport.YOffset; got != 5 {
		t.Fatalf("expected no-op without selected note, got offset %d", got)
	}
	if len(m.notePositions) != 0 {
		t.Fatalf("expected no position memory writes, got %+v", m.notePositions)
	}
}

func TestHandleBrowseKeyPreviewScrollSplitPrimaryFocusOnlyTouchesPrimary(t *testing.T) {
	root := t.TempDir()
	primary := filepath.Join(root, "primary.md")
	secondary := filepath.Join(root, "secondary.md")
	mustWriteFile(t, primary, "p\n")
	mustWriteFile(t, secondary, "s\n")

	vp := viewport.New(80, 10)
	vp.SetContent(lineBlock(80))
	vp.YOffset = 20

	m := &Model{
		notesDir:            root,
		splitMode:           true,
		splitFocusSecondary: false,
		currentFile:         primary,
		secondaryFile:       secondary,
		viewport:            vp,
		notePositions: map[string]notePosition{
			secondary: {SecondaryPreviewOffset: 7},
		},
		keyToAction: map[string]string{
			"pgdown": actionPreviewScrollPageDown,
		},
	}

	_, _ = m.handleBrowseKey("pgdown")
	if got := m.viewport.YOffset; got != 30 {
		t.Fatalf("expected primary viewport offset 30, got %d", got)
	}
	if got := m.notePositions[primary].PrimaryPreviewOffset; got != 30 {
		t.Fatalf("expected primary persisted offset 30, got %d", got)
	}
	if got := m.notePositions[secondary].SecondaryPreviewOffset; got != 7 {
		t.Fatalf("expected secondary offset unchanged at 7, got %d", got)
	}
}

func TestHandleBrowseKeyPreviewScrollSplitSecondaryFocusOnlyTouchesSecondary(t *testing.T) {
	root := t.TempDir()
	primary := filepath.Join(root, "primary.md")
	secondary := filepath.Join(root, "secondary.md")
	mustWriteFile(t, primary, "p\n")
	mustWriteFile(t, secondary, "s\n")

	vp := viewport.New(80, 10)
	vp.SetContent(lineBlock(80))
	vp.YOffset = 13

	info, err := os.Stat(secondary)
	if err != nil {
		t.Fatalf("stat secondary note: %v", err)
	}
	bucket := roundWidthToNearestBucket(vp.Width)

	m := &Model{
		notesDir:            root,
		splitMode:           true,
		splitFocusSecondary: true,
		currentFile:         primary,
		secondaryFile:       secondary,
		viewport:            vp,
		renderCache: map[string]renderCacheEntry{
			secondary: {
				mtime:   info.ModTime(),
				width:   bucket,
				content: lineBlock(50),
				raw:     lineBlock(50),
			},
		},
		notePositions: map[string]notePosition{
			primary:   {PrimaryPreviewOffset: 13, PreviewOffset: 13},
			secondary: {SecondaryPreviewOffset: 5},
		},
		keyToAction: map[string]string{
			"ctrl+d": actionPreviewScrollHalfDown,
		},
	}

	_, _ = m.handleBrowseKey("ctrl+d")
	if got := m.viewport.YOffset; got != 13 {
		t.Fatalf("expected primary viewport offset unchanged at 13, got %d", got)
	}
	if got := m.notePositions[secondary].SecondaryPreviewOffset; got != 10 {
		t.Fatalf("expected secondary offset moved to 10, got %d", got)
	}
	if got := m.notePositions[primary].PrimaryPreviewOffset; got != 13 {
		t.Fatalf("expected primary persisted offset unchanged at 13, got %d", got)
	}
}

func TestHandleRefreshClearsRenderAndMetadataCaches(t *testing.T) {
	root := t.TempDir()
	note := filepath.Join(root, "note.md")
	mustWriteFile(t, note, "x\n")

	m := &Model{
		notesDir: root,
		expanded: map[string]bool{root: true},
		items: []treeItem{
			{path: note, name: "note.md"},
		},
		renderCache: map[string]renderCacheEntry{
			note: {content: "cached"},
		},
		treeMetadataCache: map[string]treeMetadataCacheEntry{
			note: {tags: []string{"go"}},
		},
	}

	_, _ = m.handleRefresh()
	if len(m.renderCache) != 0 {
		t.Fatalf("expected render cache cleared, got %d entries", len(m.renderCache))
	}
	if len(m.treeMetadataCache) != 0 {
		t.Fatalf("expected metadata cache cleared, got %d entries", len(m.treeMetadataCache))
	}
	if m.status != "Refreshed" {
		t.Fatalf("expected refreshed status, got %q", m.status)
	}
}

func TestHandleBrowseKeyUsesActionDispatchForSearch(t *testing.T) {
	m := &Model{
		search: textinput.New(),
		keyToAction: map[string]string{
			"alt+s": actionSearch,
		},
	}

	_, _ = m.handleBrowseKey("ctrl+p")
	if m.isOverlay(overlaySearch) {
		t.Fatal("expected ctrl+p to do nothing when not bound to search action")
	}

	_, _ = m.handleBrowseKey("alt+s")
	if !m.isOverlay(overlaySearch) {
		t.Fatal("expected custom bound key to open search overlay")
	}
}

func TestHandleSearchKeyCtrlBindingsMatchArrowBehavior(t *testing.T) {
	m := &Model{
		searchResults: []treeItem{
			{name: "a"},
			{name: "b"},
			{name: "c"},
		},
		searchResultCursor: 1,
	}

	_, _ = m.handleSearchKey(tea.KeyMsg{Type: tea.KeyCtrlP})
	if got := m.searchResultCursor; got != 0 {
		t.Fatalf("expected ctrl+p to move cursor up to 0, got %d", got)
	}

	_, _ = m.handleSearchKey(tea.KeyMsg{Type: tea.KeyCtrlN})
	if got := m.searchResultCursor; got != 1 {
		t.Fatalf("expected ctrl+n to move cursor down to 1, got %d", got)
	}
}
