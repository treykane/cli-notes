package app

import (
	"os"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

// renderDebounce prevents excessive markdown re-rendering during fast navigation.
const renderDebounce = RenderDebounce

// renderCacheEntry stores rendered markdown and the inputs that created it.
type renderCacheEntry struct {
	mtime   time.Time
	width   int
	content string
	raw     string
}

// renderRequestMsg triggers the debounced renderer.
type renderRequestMsg struct {
	path  string
	width int
	seq   int
}

// renderResultMsg carries the render output back to Update.
type renderResultMsg struct {
	path    string
	width   int
	seq     int
	content string
	raw     string
	mtime   time.Time
	err     error
}

var (
	// Cache per-width Glamour renderers; keyed by terminal width bucket.
	rendererCacheMu sync.Mutex
	rendererCache   = map[int]*glamour.TermRenderer{}
)

// maybeShowSelectedFile shows the file in the right pane if it is markdown.
func (m *Model) maybeShowSelectedFile() tea.Cmd {
	item := m.selectedItem()
	if item == nil || item.isDir {
		return nil
	}
	if hasSuffixCaseInsensitive(item.path, ".md") {
		return m.setCurrentFile(item.path)
	}
	return nil
}

// setCurrentFile tracks the file and triggers a render.
func (m *Model) setCurrentFile(path string) tea.Cmd {
	m.currentFile = path
	if content, err := os.ReadFile(path); err == nil {
		m.currentNoteContent = string(content)
	}
	return m.requestRender(path)
}

// refreshViewport rerenders the active file, if any.
func (m *Model) refreshViewport() tea.Cmd {
	if m.currentFile != "" {
		return m.requestRender(m.currentFile)
	}
	return nil
}

// requestRender initiates a debounced render with caching.
func (m *Model) requestRender(path string) tea.Cmd {
	if path == "" {
		return nil
	}
	width := roundWidthToNearestBucket(m.viewport.Width)
	if info, err := os.Stat(path); err == nil {
		if entry, ok := m.renderCache[path]; ok && entry.width == width && entry.mtime.Equal(info.ModTime()) {
			m.viewport.SetContent(entry.content)
			m.currentNoteContent = entry.raw
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
	return tea.Tick(renderDebounce, func(time.Time) tea.Msg {
		return renderRequestMsg{path: path, width: width, seq: seq}
	})
}

// renderMarkdownCmd performs the file read + markdown render off the UI thread.
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

// renderMarkdown converts markdown text to ANSI output for the viewport.
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

// getRenderer returns a cached Glamour renderer for the given width.
func getRenderer(width int) (*glamour.TermRenderer, error) {
	if width <= 0 {
		width = 80
	}
	rendererCacheMu.Lock()
	defer rendererCacheMu.Unlock()
	if renderer, ok := rendererCache[width]; ok {
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
	return renderer, nil
}

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
