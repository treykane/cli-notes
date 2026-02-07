// render.go implements debounced, cached markdown rendering for the preview pane.
//
// Rendering markdown through Glamour is relatively expensive, so this module
// applies two optimizations to keep the UI responsive:
//
// # Debouncing
//
// When the user navigates the tree (e.g. holding down j/k), each cursor move
// would trigger a new render. Instead, requestRender increments a sequence
// number and schedules a render after a 500 ms delay. If another navigation
// happens before the timer fires, the sequence number changes and the stale
// request is discarded. Only the final render (with the latest sequence) is
// actually executed.
//
// # Caching
//
// Completed renders are cached in a map keyed by file path. Each cache entry
// records the file's modification time and the terminal width bucket used for
// rendering. A cache hit occurs when the path, mtime, and width all match,
// allowing instant display without re-reading the file or invoking Glamour.
//
// Width bucketing (via roundWidthToNearestBucket) rounds the terminal width
// to the nearest multiple of RenderWidthBucket (20 columns). This means small
// width changes (e.g. dragging a window edge) reuse cached renders rather than
// invalidating the cache on every pixel.
//
// # Glamour Renderers
//
// Glamour TermRenderer instances are themselves cached per width bucket in a
// global map (rendererCache) protected by a mutex. Creating a renderer is
// moderately expensive, so reusing them across renders avoids repeated setup.
// The rendering style is determined by the CLI_NOTES_GLAMOUR_STYLE or
// GLAMOUR_STYLE environment variable, defaulting to "dark".
package app

import (
	"container/list"
	"os"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

// renderCacheEntry stores a completed render alongside the inputs that produced
// it. The mtime and width fields act as cache keys: if the file's modification
// time or the terminal width bucket has changed, the cached content is stale
// and a new render is needed.
type renderCacheEntry struct {
	mtime   time.Time // file modification time at render time
	width   int       // terminal width bucket used for word wrapping
	content string    // ANSI-formatted rendered output (ready for viewport)
	raw     string    // original raw markdown content (used for metrics, clipboard)
}

// renderRequestMsg is emitted by the debounce timer to trigger the actual
// render. The seq field is compared to the model's current renderSeq to
// discard stale requests that were superseded by newer navigation.
type renderRequestMsg struct {
	path  string // absolute path to the file to render
	width int    // width bucket to render at
	seq   int    // sequence number for staleness detection
}

// renderResultMsg carries the completed render output (or error) back from
// the async render Cmd to the Update loop. The seq and path fields are checked
// against the model's current state to discard results that are no longer
// relevant (e.g. the user navigated away while the render was in flight).
type renderResultMsg struct {
	path    string    // file that was rendered
	width   int       // width bucket used
	seq     int       // sequence number for staleness detection
	content string    // ANSI-formatted rendered output
	raw     string    // raw markdown source
	mtime   time.Time // file modification time (for cache key)
	err     error     // non-nil if the render failed
}

var (
	// maxRendererCacheEntries bounds the number of width-specific Glamour
	// renderers retained in memory.
	maxRendererCacheEntries = 8

	// rendererCacheMu protects concurrent access to the renderer cache.
	// Renders can happen on background goroutines via renderMarkdownCmd,
	// so the cache must be thread-safe.
	rendererCacheMu sync.Mutex

	// rendererCache maps terminal width buckets to reusable Glamour
	// TermRenderer instances. Creating a renderer involves parsing style
	// JSON and allocating internal buffers, so caching them avoids
	// repeated setup costs when the terminal width hasn't changed.
	rendererCache = map[int]*glamour.TermRenderer{}

	// rendererCacheOrder tracks width buckets in LRU order (front = least recent,
	// back = most recent).
	rendererCacheOrder = list.New()

	// rendererCacheNodes stores the LRU-list node for each cached width bucket.
	rendererCacheNodes = map[int]*list.Element{}
)

// maybeShowSelectedFile triggers a render of the currently selected tree item
// if it is a markdown file. Called after cursor movement so the preview pane
// tracks the tree selection. Non-markdown files and directories are ignored.
func (m *Model) maybeShowSelectedFile() tea.Cmd {
	item := m.selectedItem()
	if item == nil || item.isDir {
		return nil
	}
	if hasSuffixCaseInsensitive(item.path, ".md") {
		return m.setFocusedFile(item.path)
	}
	return nil
}

// setCurrentFile sets the given file as the active note displayed in the
// primary viewport. It saves the position of the previously viewed note,
// records this file in the recent-files list, reads the raw content for
// metrics/clipboard, and initiates a debounced render.
func (m *Model) setCurrentFile(path string) tea.Cmd {
	if m.currentFile != "" && m.currentFile != path {
		m.rememberCurrentNotePosition()
		m.saveAppState()
	}
	m.currentFile = path
	m.trackFileOpen(path)
	m.trackRecentFile(path)
	if content, err := os.ReadFile(path); err == nil {
		m.currentNoteContent = string(content)
	}
	return m.requestRender(path)
}

// refreshViewport re-renders the currently displayed file, if any. This is
// called after terminal resizes to re-wrap content at the new width.
func (m *Model) refreshViewport() tea.Cmd {
	if m.currentFile != "" {
		return m.requestRender(m.currentFile)
	}
	return nil
}

// requestRender initiates a debounced render for the given file path.
//
// Fast path (cache hit): If the render cache contains an entry for this path
// with a matching mtime and width bucket, the cached content is displayed
// immediately and no Cmd is returned.
//
// Slow path (cache miss): A spinner is shown, the renderSeq is incremented
// (invalidating any in-flight render), and a debounce timer is started. After
// RenderDebounce (500 ms), a renderRequestMsg is emitted which — if its
// sequence number still matches — triggers the actual async render via
// renderMarkdownCmd.
func (m *Model) requestRender(path string) tea.Cmd {
	if path == "" {
		return nil
	}
	width := roundWidthToNearestBucket(m.viewport.Width)
	if info, err := os.Stat(path); err == nil {
		if entry, ok := m.renderCache[path]; ok && entry.width == width && entry.mtime.Equal(info.ModTime()) {
			m.viewport.SetContent(entry.content)
			m.currentNoteContent = entry.raw
			m.restorePreviewOffset(path)
			m.rendering = false
			m.renderingPath = ""
			m.renderingSeq = 0
			return nil
		}
	}
	m.rendering = true
	m.viewport.SetContent(m.spinner.View() + " Rendering...")
	m.renderSeq++
	seq := m.renderSeq
	m.pendingPath = path
	m.pendingWidth = width
	m.renderingPath = path
	m.renderingSeq = seq
	return tea.Tick(RenderDebounce, func(time.Time) tea.Msg {
		return renderRequestMsg{path: path, width: width, seq: seq}
	})
}

// renderMarkdownCmd returns a Bubble Tea Cmd that reads and renders a markdown
// file on a background goroutine. This keeps the UI thread free to process
// spinner ticks and other input while the (potentially slow) Glamour render
// runs. The result is sent back to Update as a renderResultMsg.
func renderMarkdownCmd(path string, width int, seq int) tea.Cmd {
	return func() tea.Msg {
		info, err := os.Stat(path)
		if err != nil {
			return renderResultMsg{path: path, width: width, seq: seq, err: err}
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return renderResultMsg{path: path, width: width, seq: seq, err: err}
		}
		rendered := renderMarkdown(string(content), width)
		return renderResultMsg{
			path:    path,
			width:   width,
			seq:     seq,
			content: rendered,
			raw:     string(content),
			mtime:   info.ModTime(),
		}
	}
}

// renderMarkdown converts raw markdown text to ANSI-formatted output suitable
// for display in the Bubble Tea viewport. It uses a cached Glamour renderer
// for the given width. If renderer creation or rendering fails, the raw
// markdown is returned as-is so the user still sees content (just unformatted).
func renderMarkdown(content string, width int) string {
	if width <= 0 {
		width = 80
	}
	renderer, err := getRenderer(width)
	if err != nil {
		appLog.Error("create markdown renderer", "width", width, "error", err)
		return content
	}
	out, err := renderer.Render(content)
	if err != nil {
		appLog.Error("render markdown content", "width", width, "error", err)
		return content
	}
	return out
}

// getRenderer returns a cached Glamour TermRenderer for the given width,
// creating one if it doesn't exist. The renderer is configured with word
// wrapping at the specified width and the user's chosen Glamour style.
// Access is serialized via rendererCacheMu since renders may run concurrently
// on background goroutines.
func getRenderer(width int) (*glamour.TermRenderer, error) {
	if width <= 0 {
		width = 80
	}
	rendererCacheMu.Lock()
	defer rendererCacheMu.Unlock()
	if renderer, ok := rendererCache[width]; ok {
		if node, ok := rendererCacheNodes[width]; ok {
			rendererCacheOrder.MoveToBack(node)
		}
		return renderer, nil
	}
	renderer, err := glamour.NewTermRenderer(
		glamourStyleOption(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil, err
	}
	rendererCache[width] = renderer
	rendererCacheNodes[width] = rendererCacheOrder.PushBack(width)
	evictOldestRendererIfNeeded()
	return renderer, nil
}

func evictOldestRendererIfNeeded() {
	for len(rendererCache) > maxRendererCacheEntries && rendererCacheOrder.Len() > 0 {
		oldest := rendererCacheOrder.Front()
		width, _ := oldest.Value.(int)
		rendererCacheOrder.Remove(oldest)
		delete(rendererCache, width)
		delete(rendererCacheNodes, width)
	}
}

func resetRendererCacheForTests() {
	rendererCacheMu.Lock()
	defer rendererCacheMu.Unlock()
	rendererCache = map[int]*glamour.TermRenderer{}
	rendererCacheOrder = list.New()
	rendererCacheNodes = map[int]*list.Element{}
}

// glamourStyleOption resolves the Glamour rendering style from environment
// variables. The lookup order is:
//
//  1. CLI_NOTES_GLAMOUR_STYLE (app-specific override)
//  2. GLAMOUR_STYLE (Glamour's own environment variable)
//  3. "dark" (hardcoded default — avoids OSC background queries that can
//     leak escape sequences into the editor)
//
// The special value "auto" delegates to Glamour's auto-detection, which
// queries the terminal's background color. All other values are passed
// through as standard style names (dark, light, notty).
func glamourStyleOption() glamour.TermRendererOption {
	style := strings.ToLower(strings.TrimSpace(os.Getenv("CLI_NOTES_GLAMOUR_STYLE")))
	if style == "" {
		style = strings.ToLower(strings.TrimSpace(os.Getenv("GLAMOUR_STYLE")))
	}
	if style == "" {
		style = "dark"
	}
	if style == "auto" {
		return glamour.WithAutoStyle()
	}
	switch style {
	case "dark", "light", "notty":
		return glamour.WithStandardStyle(style)
	default:
		return glamour.WithStandardStyle("dark")
	}
}
