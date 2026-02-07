package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
)

func TestRequestRenderUsesCachedEntryWhenMtimeAndWidthMatch(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "note.md")
	mustWriteFile(t, path, "# cached\n")

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}

	vp := viewport.New(81, 5) // width bucket is 80
	m := &Model{
		viewport:    vp,
		spinner:     spinner.New(),
		renderSeq:   9,
		renderCache: map[string]renderCacheEntry{},
	}
	m.renderCache[path] = renderCacheEntry{
		mtime:   info.ModTime(),
		width:   80,
		content: "cached-render-output",
	}

	cmd := m.requestRender(path)
	if cmd != nil {
		t.Fatal("expected no render command on cache hit")
	}
	if !strings.Contains(m.viewport.View(), "cached-render-output") {
		t.Fatalf("expected cached content in viewport, got %q", m.viewport.View())
	}
	if m.rendering {
		t.Fatal("expected rendering to be false on cache hit")
	}
	if m.renderSeq != 9 {
		t.Fatalf("expected renderSeq to stay 9, got %d", m.renderSeq)
	}
}

func TestRequestRenderStartsAsyncRenderWhenCacheMissing(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "note.md")
	mustWriteFile(t, path, "# cache miss\n")

	vp := viewport.New(81, 5) // width bucket is 80
	m := &Model{
		viewport:    vp,
		spinner:     spinner.New(),
		renderCache: map[string]renderCacheEntry{},
	}

	cmd := m.requestRender(path)
	if cmd == nil {
		t.Fatal("expected render command on cache miss")
	}
	if !m.rendering {
		t.Fatal("expected rendering to be true on cache miss")
	}
	if m.pendingPath != path {
		t.Fatalf("expected pendingPath %q, got %q", path, m.pendingPath)
	}
	if m.pendingWidth != 80 {
		t.Fatalf("expected pendingWidth 80, got %d", m.pendingWidth)
	}
	if m.renderingPath != path {
		t.Fatalf("expected renderingPath %q, got %q", path, m.renderingPath)
	}
	if m.renderSeq != 1 || m.renderingSeq != 1 {
		t.Fatalf("expected render sequence to be 1/1, got %d/%d", m.renderSeq, m.renderingSeq)
	}
	if !strings.Contains(m.viewport.View(), "Rendering...") {
		t.Fatalf("expected rendering indicator in viewport, got %q", m.viewport.View())
	}
}

func TestGetRendererCacheIsBoundedByCap(t *testing.T) {
	resetRendererCacheForTests()
	t.Cleanup(resetRendererCacheForTests)

	oldCap := maxRendererCacheEntries
	maxRendererCacheEntries = 8
	t.Cleanup(func() { maxRendererCacheEntries = oldCap })

	for width := 20; width <= 240; width += 20 {
		if _, err := getRenderer(width); err != nil {
			t.Fatalf("getRenderer(%d): %v", width, err)
		}
	}

	if got := len(rendererCache); got != maxRendererCacheEntries {
		t.Fatalf("expected renderer cache size %d, got %d", maxRendererCacheEntries, got)
	}
	if _, ok := rendererCache[20]; ok {
		t.Fatal("expected oldest width to be evicted")
	}
	if _, ok := rendererCache[240]; !ok {
		t.Fatal("expected newest width to remain cached")
	}
}

func TestGetRendererLRURefreshesRecentlyUsedWidth(t *testing.T) {
	resetRendererCacheForTests()
	t.Cleanup(resetRendererCacheForTests)

	oldCap := maxRendererCacheEntries
	maxRendererCacheEntries = 3
	t.Cleanup(func() { maxRendererCacheEntries = oldCap })

	for _, width := range []int{10, 20, 30} {
		if _, err := getRenderer(width); err != nil {
			t.Fatalf("seed getRenderer(%d): %v", width, err)
		}
	}
	if _, err := getRenderer(10); err != nil {
		t.Fatalf("refresh getRenderer(10): %v", err)
	}
	if _, err := getRenderer(40); err != nil {
		t.Fatalf("insert getRenderer(40): %v", err)
	}

	if _, ok := rendererCache[20]; ok {
		t.Fatal("expected width 20 to be evicted as least recently used")
	}
	if _, ok := rendererCache[10]; !ok {
		t.Fatal("expected width 10 to remain after recent access")
	}
}
