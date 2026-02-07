package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type noteHeading struct {
	Level int
	Title string
	Line  int
}

func (m *Model) openRecentPopup() {
	if m.searching {
		m.closeSearchPopup()
	}
	m.rebuildRecentEntries()
	m.showRecentPopup = true
	m.showOutlinePopup = false
	m.showHelp = false
	if len(m.recentEntries) == 0 {
		m.status = "No recent files yet"
		return
	}
	m.recentCursor = clamp(m.recentCursor, 0, len(m.recentEntries)-1)
	m.status = "Recent files: Enter to jump, Esc to close"
}

func (m *Model) closeRecentPopup() {
	m.showRecentPopup = false
}

func (m *Model) handleRecentPopupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.shouldIgnoreInput(msg) {
		return m, nil
	}
	if len(m.recentEntries) == 0 {
		if msg.String() == "esc" {
			m.closeRecentPopup()
			m.status = "Recent files closed"
		}
		return m, nil
	}

	switch msg.String() {
	case "esc":
		m.closeRecentPopup()
		m.status = "Recent files closed"
		return m, nil
	case "up", "k", "ctrl+p":
		m.recentCursor = clamp(m.recentCursor-1, 0, len(m.recentEntries)-1)
		return m, nil
	case "down", "j", "ctrl+n":
		m.recentCursor = clamp(m.recentCursor+1, 0, len(m.recentEntries)-1)
		return m, nil
	case "enter":
		return m.selectRecentEntry()
	default:
		return m, nil
	}
}

func (m *Model) selectRecentEntry() (tea.Model, tea.Cmd) {
	if len(m.recentEntries) == 0 {
		m.status = "No recent files"
		return m, nil
	}
	path := m.recentEntries[m.recentCursor]
	if _, err := os.Stat(path); err != nil {
		m.recentFiles = removePathFromList(m.recentFiles, path)
		m.rebuildRecentEntries()
		m.saveAppState()
		m.status = "Recent file no longer exists"
		return m, nil
	}

	m.closeRecentPopup()
	m.expandParentDirs(path)
	m.rebuildTreeKeep(path)
	m.status = "Jumped to recent: " + m.displayRelative(path)
	return m, m.setFocusedFile(path)
}

func (m *Model) openOutlinePopup() {
	if m.mode != modeBrowse || m.currentFile == "" {
		m.status = "Select a note first"
		return
	}
	if m.searching {
		m.closeSearchPopup()
	}
	headings := parseMarkdownHeadings(m.currentNoteContent)
	if len(headings) == 0 {
		m.status = "No markdown headings in current note"
		return
	}
	m.outlineHeadings = headings
	m.outlineCursor = clamp(m.outlineCursor, 0, len(headings)-1)
	m.showOutlinePopup = true
	m.showRecentPopup = false
	m.showHelp = false
	m.status = "Outline: Enter to jump, Esc to close"
}

func (m *Model) closeOutlinePopup() {
	m.showOutlinePopup = false
}

func (m *Model) handleOutlinePopupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.shouldIgnoreInput(msg) {
		return m, nil
	}
	if len(m.outlineHeadings) == 0 {
		if msg.String() == "esc" {
			m.closeOutlinePopup()
			m.status = "Outline closed"
		}
		return m, nil
	}

	switch msg.String() {
	case "esc":
		m.closeOutlinePopup()
		m.status = "Outline closed"
		return m, nil
	case "up", "k", "ctrl+p":
		m.outlineCursor = clamp(m.outlineCursor-1, 0, len(m.outlineHeadings)-1)
		return m, nil
	case "down", "j", "ctrl+n":
		m.outlineCursor = clamp(m.outlineCursor+1, 0, len(m.outlineHeadings)-1)
		return m, nil
	case "enter":
		m.jumpToOutlineHeading(m.outlineHeadings[m.outlineCursor])
		m.closeOutlinePopup()
		return m, nil
	default:
		return m, nil
	}
}

func (m *Model) jumpToOutlineHeading(heading noteHeading) {
	lines := strings.Split(m.viewport.View(), "\n")
	target := heading.Title
	index := -1
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), strings.ToLower(target)) {
			index = i
			break
		}
	}
	if index < 0 {
		index = max(0, heading.Line-1)
	}
	m.viewport.YOffset = max(0, index)
	m.rememberCurrentNotePosition()
	m.saveAppState()
	m.status = fmt.Sprintf("Jumped to heading: %s", heading.Title)
}

func parseMarkdownHeadings(content string) []noteHeading {
	lines := strings.Split(content, "\n")
	headings := make([]noteHeading, 0, 16)
	inFence := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if inFence || !strings.HasPrefix(trimmed, "#") {
			continue
		}

		level := 0
		for level < len(trimmed) && trimmed[level] == '#' {
			level++
		}
		if level == 0 || level > 6 {
			continue
		}
		if len(trimmed) <= level || trimmed[level] != ' ' {
			continue
		}
		title := strings.TrimSpace(trimmed[level:])
		if title == "" {
			continue
		}
		headings = append(headings, noteHeading{
			Level: level,
			Title: title,
			Line:  i + 1,
		})
	}
	return headings
}

func (m *Model) togglePinnedSelection() {
	item := m.selectedItem()
	if item == nil {
		m.status = "No item selected"
		return
	}
	if item.path == m.notesDir {
		m.status = "Cannot pin the root notes directory"
		return
	}
	if m.pinnedPaths == nil {
		m.pinnedPaths = map[string]bool{}
	}
	if m.pinnedPaths[item.path] {
		delete(m.pinnedPaths, item.path)
		m.status = "Unpinned: " + filepath.Base(item.path)
	} else {
		m.pinnedPaths[item.path] = true
		m.status = "Pinned: " + filepath.Base(item.path)
	}
	m.rebuildTreeKeep(item.path)
	m.saveAppState()
}
