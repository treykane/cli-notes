// popups.go implements browse-mode popup overlays for recent files, heading
// outline, and pin/unpin toggling.
//
// Each popup follows a consistent interaction pattern:
//
//   - A keybinding opens the popup, populating its data and resetting the cursor.
//   - Up/Down (or j/k) navigate the list; Enter selects; Esc closes.
//   - Only one popup is visible at a time — opening one closes others via
//     closeTransientPopups (defined in workspace_export.go).
//
// The heading outline popup parses markdown headings from the current note's
// raw content and renders them with indentation matching their heading level.
// Selecting a heading scrolls the preview viewport to that section.
//
// The recent files popup filters the persisted recent-files list to only show
// entries that still exist on disk and are within the current workspace root.
//
// Pin toggling does not use a popup — it simply toggles the pinned flag for the
// selected tree item and rebuilds the tree so pinned items float to the top.
package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// noteHeading represents a single parsed markdown heading for the outline popup.
// Level is the heading depth (1–6, corresponding to # through ######), Title
// is the heading text (without the leading # markers), and Line is the 1-based
// line number in the raw note content for jump-to-section targeting.
type noteHeading struct {
	Level int
	Title string
	Line  int
}

// openRecentPopup shows the recent-files popup (Ctrl+O). It rebuilds the
// visible entries list from the persisted recent files, filtering out entries
// that no longer exist on disk. The search popup is closed if open, since
// only one overlay is shown at a time.
func (m *Model) openRecentPopup() {
	m.closeOverlay()
	m.rebuildRecentEntries()
	m.openOverlay(overlayRecent)
	m.showHelp = false
	if len(m.recentEntries) == 0 {
		m.status = "No recent files yet"
		return
	}
	m.recentCursor = clamp(m.recentCursor, 0, len(m.recentEntries)-1)
	m.status = "Recent files: Enter to jump, Esc to close"
}

// closeRecentPopup hides the recent-files popup without selecting an entry.
func (m *Model) closeRecentPopup() {
	if m.isOverlay(overlayRecent) {
		m.closeOverlay()
	}
}

// handleRecentPopupKey routes key presses while the recent-files popup is visible.
// Navigation uses j/k or arrow keys; Enter jumps to the selected file; Esc closes.
func (m *Model) handleRecentPopupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.shouldIgnoreInput(msg) {
		return m, nil
	}
	next, selectPressed, closePressed, handled := handlePopupListNav(msg, m.recentCursor, len(m.recentEntries))
	if !handled {
		return m, nil
	}
	if closePressed {
		m.closeRecentPopup()
		m.status = "Recent files closed"
		return m, nil
	}
	if len(m.recentEntries) == 0 {
		return m, nil
	}
	m.recentCursor = next
	if selectPressed {
		return m.selectRecentEntry()
	}
	return m, nil
}

// selectRecentEntry opens the file at the current recent-files cursor position.
// If the file no longer exists on disk it is silently removed from the list and
// the user is notified. Otherwise the popup is closed, the tree is expanded to
// reveal the file, and it is loaded into the viewport.
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

// openOutlinePopup shows the heading outline popup (o key in browse mode).
// It parses all markdown headings (# through ######) from the current note's
// raw content, skipping headings inside fenced code blocks. If no headings are
// found, a status message is shown instead of opening an empty popup.
func (m *Model) openOutlinePopup() {
	if m.mode != modeBrowse || m.currentFile == "" {
		m.status = "Select a note first"
		return
	}
	m.closeOverlay()
	headings := parseMarkdownHeadings(m.currentNoteContent)
	if len(headings) == 0 {
		m.status = "No markdown headings in current note"
		return
	}
	m.outlineHeadings = headings
	m.outlineCursor = clamp(m.outlineCursor, 0, len(headings)-1)
	m.openOverlay(overlayOutline)
	m.showHelp = false
	m.status = "Outline: Enter to jump, Esc to close"
}

// closeOutlinePopup hides the heading outline popup without jumping.
func (m *Model) closeOutlinePopup() {
	if m.isOverlay(overlayOutline) {
		m.closeOverlay()
	}
}

// handleOutlinePopupKey routes key presses while the outline popup is visible.
// Enter jumps the preview viewport to the selected heading; Esc closes.
func (m *Model) handleOutlinePopupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.shouldIgnoreInput(msg) {
		return m, nil
	}
	next, selectPressed, closePressed, handled := handlePopupListNav(msg, m.outlineCursor, len(m.outlineHeadings))
	if !handled {
		return m, nil
	}
	if closePressed {
		m.closeOutlinePopup()
		m.status = "Outline closed"
		return m, nil
	}
	if len(m.outlineHeadings) == 0 {
		return m, nil
	}
	m.outlineCursor = next
	if selectPressed {
		m.jumpToOutlineHeading(m.outlineHeadings[m.outlineCursor])
		m.closeOutlinePopup()
	}
	return m, nil
}

// jumpToOutlineHeading scrolls the preview viewport so the selected heading is
// at the top of the visible area. It first searches the rendered view for the
// heading text (since Glamour rendering may shift line numbers) and falls back
// to the raw source line number if no rendered match is found. The viewport
// offset is saved to per-note position memory so the scroll position persists.
func (m *Model) jumpToOutlineHeading(heading noteHeading) {
	path := m.currentFile
	secondary := false
	rendered := m.viewport.View()
	if m.splitMode && m.splitFocusSecondary && m.secondaryFile != "" {
		path = m.secondaryFile
		secondary = true
		if renderedSecondary, ok := m.renderedForPath(path, m.viewport.Width); ok {
			rendered = renderedSecondary
		}
	}
	lines := strings.Split(rendered, "\n")
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
	m.setPaneOffset(path, secondary, max(0, index))
	if !secondary {
		m.viewport.YOffset = max(0, index)
	}
	m.saveAppState()
	m.status = fmt.Sprintf("Jumped to heading: %s", heading.Title)
}

// parseMarkdownHeadings extracts all ATX-style markdown headings from content.
//
// Parsing rules:
//   - Lines starting with one or more '#' characters followed by a space are
//     recognized as headings (levels 1–6).
//   - Headings inside fenced code blocks (``` delimited) are ignored.
//   - Leading/trailing whitespace is trimmed from the heading title.
//   - Empty titles (e.g. "# " with nothing after) are skipped.
//
// Returns headings in document order with 1-based line numbers.
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

// togglePinnedSelection pins or unpins the currently selected tree item.
// Pinned items are sorted to the top of their directory level across all sort
// modes. The pin state is persisted in per-workspace state.json. The root
// notes directory cannot be pinned.
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
