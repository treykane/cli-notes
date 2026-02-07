// view.go implements the View function and all rendering helpers for the
// terminal UI.
//
// The UI is composed of three visual regions drawn top-to-bottom:
//
//  1. Main row: a horizontal join of the left tree pane and the right content
//     pane. When a popup is active, the main row is replaced entirely by the
//     popup overlay centered on screen.
//  2. Footer status bar: an adaptive 2-3 row block at the bottom showing
//     grouped interaction hints, context telemetry (W/C/L + git), and status.
//
// The View function is called by Bubble Tea on every frame. It must return
// a complete string representation of the screen — there is no incremental
// redraw. To avoid visual artifacts from previous frames, all regions are
// padded to exact terminal dimensions via padBlock.
//
// # Popup Overlays
//
// Popups (search, recent files, outline, workspace, export, wiki links, wiki
// autocomplete) replace the main row content entirely. They are rendered as
// fixed-size boxes centered (or bottom-aligned for autocomplete) within the
// available space using lipgloss.Place.
//
// # Split Pane
//
// When split mode is active, the right pane is divided into two side-by-side
// sub-panes via renderRightSplit, each independently showing a rendered note
// or the editor.
package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// View draws the full UI (left tree + right pane + status line).
//
// This is the top-level Bubble Tea View function. It calculates layout
// dimensions, renders each pane, overlays any active popup, and assembles
// the final output string. The result is padded to exactly fill the terminal
// so that every frame fully overwrites the previous one.
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	footerHeight := m.footerHeightForWidth(m.width)
	layout := m.calculateLayout()
	leftPane := m.renderTree(layout.LeftWidth, layout.ContentHeight)
	rightPane := m.renderRight(layout.RightWidth, layout.ContentHeight)
	row := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	if m.searching {
		row = m.renderSearchPopupOverlay(m.width, layout.ContentHeight)
	} else if m.showRecentPopup {
		row = m.renderRecentPopupOverlay(m.width, layout.ContentHeight)
	} else if m.showOutlinePopup {
		row = m.renderOutlinePopupOverlay(m.width, layout.ContentHeight)
	} else if m.showWorkspacePopup {
		row = m.renderWorkspacePopupOverlay(m.width, layout.ContentHeight)
	} else if m.showExportPopup {
		row = m.renderExportPopupOverlay(m.width, layout.ContentHeight)
	} else if m.showWikiLinksPopup {
		row = m.renderWikiLinksPopupOverlay(m.width, layout.ContentHeight)
	} else if m.showWikiAutocomplete {
		row = m.renderWikiAutocompletePopupOverlay(m.width, layout.ContentHeight)
	}
	// Clamp the pane row so the dynamic footer area remains reserved.
	row = padBlock(row, m.width, layout.ContentHeight)

	view := row + "\n" + m.renderStatus(m.width, footerHeight)
	return padBlock(view, m.width, m.height)
}

// renderTree draws the left-hand directory tree pane.
//
// The tree pane shows a scrollable list of treeItem rows within a bordered
// Lipgloss box. A header line shows the notes directory path. Items are
// rendered with indentation, type badges (DIR/MD), pin markers, and tag
// labels. The currently selected row is rendered with reversed colors
// spanning the full pane width.
//
// Scrolling is handled by slicing the items array from treeOffset to
// treeOffset + visibleHeight, so only the visible window of items is
// rendered each frame.
func (m *Model) renderTree(width, height int) string {
	innerWidth := max(0, width-paneStyle.GetHorizontalFrameSize())
	innerHeight := max(0, height-paneStyle.GetVerticalFrameSize())

	header := titleStyle.Render("Notes: " + m.notesDir)
	lines := []string{truncate(header, innerWidth)}

	visibleHeight := max(0, innerHeight-len(lines))
	start := min(m.treeOffset, max(0, len(m.items)-1))
	end := min(len(m.items), start+visibleHeight)

	for i := start; i < end; i++ {
		item := m.items[i]
		line := m.formatTreeItem(item)
		if i == m.cursor {
			line = m.formatTreeItemSelected(item)
			line = truncate(line, innerWidth)
			line = selectedStyle.Width(innerWidth).Render(line)
			lines = append(lines, line)
			continue
		}
		line = truncate(line, innerWidth)
		lines = append(lines, line)
	}
	if len(m.items) == 0 {
		lines = append(lines, truncate(mutedStyle.Render("(no matches)"), innerWidth))
	}

	content := padBlock(strings.Join(lines, "\n"), innerWidth, innerHeight)
	return paneStyle.Width(width).Height(height).Render(content)
}

// renderRight draws the right-hand pane (editor, input, or markdown viewport).
//
// The right pane's content depends on the current mode:
//   - modeBrowse: rendered markdown in the viewport (or help screen if toggled)
//   - modeEditNote: the textarea editor with syntax highlighting
//   - modeTemplatePicker: template selection list
//   - modeDraftRecovery: draft recovery prompt
//   - modeNewNote/modeNewFolder/modeRenameItem/modeMoveItem/modeGitCommit:
//     text input with contextual prompt and location info
//
// In split mode, rendering is delegated to renderRightSplit which divides
// the available width between two independent sub-panes.
func (m *Model) renderRight(width, height int) string {
	if m.splitMode {
		return m.renderRightSplit(width, height)
	}
	rightPaneStyle := previewPane
	headerStyle := previewHeader
	if m.mode == modeEditNote {
		rightPaneStyle = editPane
		headerStyle = editHeader
	}

	innerWidth := max(0, width-rightPaneStyle.GetHorizontalFrameSize())
	innerHeight := max(0, height-rightPaneStyle.GetVerticalFrameSize())
	contentHeight := max(0, innerHeight-1)

	var content string
	switch m.mode {
	case modeEditNote:
		m.editor.SetWidth(innerWidth)
		m.editor.SetHeight(contentHeight)
		content = m.editorViewWithSelectionHighlight(m.editor.View())
	case modeTemplatePicker:
		content = m.renderTemplatePicker(innerWidth, contentHeight)
	case modeDraftRecovery:
		content = m.renderDraftRecovery(innerWidth, contentHeight)
	case modeNewNote, modeNewFolder, modeRenameItem, modeMoveItem, modeGitCommit:
		m.input.Width = innerWidth
		prompt, location, helper := m.inputModeMeta()
		content = strings.Join([]string{
			titleStyle.Render(prompt),
			location,
			"",
			m.input.View(),
			"",
			helper,
		}, "\n")
	default:
		if m.showHelp {
			content = m.renderHelp(innerWidth, contentHeight)
		} else {
			m.viewport.Width = innerWidth
			m.viewport.Height = contentHeight
			content = m.viewport.View()
		}
	}

	header := m.renderRightHeader(innerWidth, headerStyle)
	body := padBlock(content, innerWidth, contentHeight)
	return rightPaneStyle.Width(width).Height(height).Render(header + "\n" + body)
}

// renderRightSplit divides the right pane into two equal-width sub-panes
// for side-by-side note viewing. The primary pane shows the current file
// (or editor in edit mode); the secondary pane shows a second file in
// read-only preview. Each sub-pane has its own header and focus indicator.
func (m *Model) renderRightSplit(width, height int) string {
	leftWidth := width / 2
	rightWidth := width - leftWidth
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.renderSingleRightPane(leftWidth, height, m.currentFile, false, !m.splitFocusSecondary),
		m.renderSingleRightPane(rightWidth, height, m.secondaryFile, true, m.splitFocusSecondary),
	)
}

// renderSingleRightPane renders one half of the split view. It shows either
// the editor (for the primary pane in edit mode) or a rendered markdown preview.
// The header indicates pane number ([1]/[2]), focus state (▶), and file path.
func (m *Model) renderSingleRightPane(width, height int, path string, secondary bool, focused bool) string {
	rightPaneStyle := previewPane
	headerStyle := previewHeader
	if m.mode == modeEditNote && !secondary {
		rightPaneStyle = editPane
		headerStyle = editHeader
	}

	innerWidth := max(0, width-rightPaneStyle.GetHorizontalFrameSize())
	innerHeight := max(0, height-rightPaneStyle.GetVerticalFrameSize())
	contentHeight := max(0, innerHeight-1)

	headerLabel := "No note selected"
	if path != "" {
		headerLabel = m.displayRelative(path)
	}
	if secondary {
		headerLabel = "[2] " + headerLabel
	} else {
		headerLabel = "[1] " + headerLabel
	}
	if focused {
		headerLabel = "▶ " + headerLabel
	}

	content := "Select a note to view"
	if path != "" {
		if m.mode == modeEditNote && !secondary && path == m.currentFile {
			m.editor.SetWidth(innerWidth)
			m.editor.SetHeight(contentHeight)
			content = m.editorViewWithSelectionHighlight(m.editor.View())
		} else if rendered, ok := m.renderedForPath(path, innerWidth); ok {
			content = m.renderPreviewWithOffset(path, rendered, secondary)
		}
	}

	header := headerStyle.Width(innerWidth).Render(" " + truncate(headerLabel, max(0, innerWidth-1)))
	body := padBlock(content, innerWidth, contentHeight)
	return rightPaneStyle.Width(width).Height(height).Render(header + "\n" + body)
}

// renderedForPath returns the cached rendered markdown for a file, or renders
// it synchronously if no cache entry exists. This is used by the secondary
// split pane which cannot use the async debounced render pipeline (that
// pipeline is tied to the primary pane's currentFile). Returns the rendered
// content and true on success, or ("", false) if the file cannot be read.
func (m *Model) renderedForPath(path string, width int) (string, bool) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return "", false
	}
	bucket := roundWidthToNearestBucket(width)
	if entry, ok := m.renderCache[path]; ok && entry.width == bucket && entry.mtime.Equal(info.ModTime()) {
		return entry.content, true
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	_, body := parseFrontmatterAndBody(string(content))
	rendered := renderMarkdown(body, bucket)
	m.renderCache[path] = renderCacheEntry{
		mtime:   info.ModTime(),
		width:   bucket,
		content: rendered,
		raw:     string(content),
	}
	return rendered, true
}

func (m *Model) renderPreviewWithOffset(path, rendered string, secondary bool) string {
	offset := m.restorePaneOffset(path, secondary)
	if offset <= 0 {
		return rendered
	}
	lines := strings.Split(rendered, "\n")
	if len(lines) == 0 {
		return rendered
	}
	offset = clamp(offset, 0, len(lines)-1)
	return strings.Join(lines[offset:], "\n")
}

// editorViewWithSelectionHighlight post-processes the editor's rendered view
// to apply two visual enhancements:
//
//  1. Fenced code block highlighting (via highlightFencedCodeInEditorView)
//  2. Selection highlighting: if a text selection is active, the selected
//     text span is wrapped in selectionText style (white-on-black).
//
// Selection highlighting is limited to single-line selections. Multi-line
// visual selection highlighting is tracked as a future improvement in TASKS.md.
func (m *Model) editorViewWithSelectionHighlight(view string) string {
	view = highlightFencedCodeInEditorView(view)
	start, end, ok := m.editorSelectionRange()
	if !ok {
		return view
	}

	runes := []rune(m.editor.Value())
	start = clamp(start, 0, len(runes))
	end = clamp(end, 0, len(runes))
	if start >= end {
		return view
	}

	selected := string(runes[start:end])
	if selected == "" || strings.Contains(selected, "\n") {
		return view
	}

	idx := strings.Index(view, selected)
	if idx < 0 {
		return view
	}

	return view[:idx] + selectionText.Render(selected) + view[idx+len(selected):]
}

// renderStatus renders the adaptive multi-row footer status bar.
//
// The footer reserves 2 or 3 rows depending on available width and content
// density. Rows are packed greedily from grouped segments in this order:
//   - Keys: interaction hints for the current mode
//   - Context: note metrics and git summary
//   - Status: the latest operation/error message
//
// If content still overflows the available rows, the final row is truncated
// with an ellipsis to make clipping explicit.
func (m *Model) renderStatus(width, rows int) string {
	statusRows, _ := m.buildStatusRows(width, rows)
	style := statusStyle
	if m.mode == modeEditNote {
		style = editStatus
	}
	for len(statusRows) < rows {
		statusRows = append(statusRows, "")
	}

	rendered := make([]string, 0, len(statusRows))
	for _, line := range statusRows {
		line = " " + truncate(line, max(0, width-1))
		rendered = append(rendered, style.Width(width).Render(line))
	}
	return strings.Join(rendered, "\n")
}

// buildStatusRows packs footer segments into up to rowLimit rows. It returns
// the rendered row strings and whether all segments fit without dropping any.
func (m *Model) buildStatusRows(width, rowLimit int) ([]string, bool) {
	if width <= 0 || rowLimit <= 0 {
		return nil, true
	}

	help := m.statusHelpSegments()
	context := m.statusContextSegments()
	status := m.statusMessageSegment()

	segments := make([]string, 0, len(help)+len(context)+2)
	if len(help) > 0 {
		segments = append(segments, "Keys: "+help[0])
		segments = append(segments, help[1:]...)
	}
	if len(context) > 0 {
		segments = append(segments, "Context: "+context[0])
		segments = append(segments, context[1:]...)
	}
	if status != "" {
		segments = append(segments, "Status: "+status)
	}

	rows := make([]string, 1, rowLimit)
	rowIndex := 0
	fit := true

	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		segment := seg
		if lipgloss.Width(segment) > width {
			segment = truncateWithEllipsis(segment, width)
		}

		candidate := segment
		if rows[rowIndex] != "" {
			candidate = rows[rowIndex] + " | " + segment
		}
		if lipgloss.Width(candidate) <= width {
			rows[rowIndex] = candidate
			continue
		}

		if rowIndex+1 < rowLimit {
			rowIndex++
			rows = append(rows, segment)
			continue
		}

		fit = false
		if rows[rowIndex] == "" {
			rows[rowIndex] = truncateWithEllipsis(segment, width)
		} else {
			rows[rowIndex] = truncateWithEllipsis(rows[rowIndex]+" | "+segment, width)
		}
		break
	}

	return rows, fit
}

func truncateWithEllipsis(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	if width == 1 {
		return "…"
	}
	return ansi.Truncate(value, width-1, "") + "…"
}

// statusHelpSegments returns mode-specific interaction hints grouped as short
// segments for multi-row footer packing.
func (m *Model) statusHelpSegments() []string {
	switch m.mode {
	case modeEditNote:
		return []string{
			"Ctrl+S save",
			"Shift+Arrows select",
			"Alt+S anchor",
			"Ctrl+B bold",
			"Alt+I italic",
			"Ctrl+U underline",
			"Alt+X strike",
			"Ctrl+K link",
			"Ctrl+1..3 heading",
			"Ctrl+V paste",
			"Esc cancel",
		}
	case modeNewNote, modeNewFolder, modeRenameItem, modeMoveItem, modeGitCommit:
		return []string{"Enter/Ctrl+S save", "Esc cancel"}
	case modeTemplatePicker:
		return []string{"Template picker", "↑/↓ move", "Enter choose", "Esc cancel"}
	case modeDraftRecovery:
		return []string{"Draft recovery", "y recover", "n discard", "Esc skip all"}
	case modeConfirmDelete:
		return []string{"y confirm delete", "n/Esc cancel"}
	default:
		if m.searching {
			return []string{"Search popup", "type", "↑/↓ move", "Enter jump", "Esc cancel"}
		}
		if m.showRecentPopup {
			return []string{"Recent popup", "↑/↓ move", "Enter jump", "Esc cancel"}
		}
		if m.showOutlinePopup {
			return []string{"Outline popup", "↑/↓ move", "Enter jump", "Esc cancel"}
		}
		if m.showWorkspacePopup {
			return []string{"Workspace popup", "↑/↓ move", "Enter switch", "Esc cancel"}
		}
		if m.showExportPopup {
			return []string{"Export popup", "↑/↓ move", "Enter export", "Esc cancel"}
		}
		if m.showWikiLinksPopup {
			return []string{"Wiki links popup", "↑/↓ move", "Enter jump", "Esc cancel"}
		}
		if m.showWikiAutocomplete {
			return []string{"Wiki autocomplete", "↑/↓ move", "Tab/Enter insert", "Esc close"}
		}
		help := []string{
			"↑/↓ or k/j move",
			"Enter/→/l toggle",
			"←/h collapse",
			"g/G top-bottom",
			"Ctrl+P search",
			"n new",
			"f folder",
			"e edit",
			"r rename",
			"m move",
			"d delete",
			"Shift+R refresh",
			"s sort",
			"t pin",
			"Ctrl+O recents",
			"o outline",
			"Ctrl+W workspaces",
			"x export",
			"Shift+L wiki",
			"z split",
			"Tab split-focus",
			"y copy content",
			"Y copy path",
		}
		if m.git.isRepo {
			help = append(help, "c commit", "p pull", "P push")
		}
		help = append(help, "? help", "q quit", "notes --configure")
		return help
	}
}

// statusContextSegments returns footer context telemetry segments.
func (m *Model) statusContextSegments() []string {
	parts := make([]string, 0, 2)
	if (m.mode == modeBrowse || m.mode == modeEditNote) && m.currentFile != "" {
		if metrics := m.noteMetricsSummary(); metrics != "" {
			parts = append(parts, metrics)
		}
	}
	if git := m.gitFooterSummary(); git != "" {
		parts = append(parts, git)
	}
	return parts
}

// statusMessageSegment returns the latest status message to show in the footer.
func (m *Model) statusMessageSegment() string {
	return strings.TrimSpace(m.status)
}

// renderHelp renders the full-screen help overlay listing all keyboard
// shortcuts organized by mode. The help text is a static multi-line string
// truncated to the available width and height. Git-specific shortcuts are
// only shown when the notes directory is inside a git repository.
func (m *Model) renderHelp(width, height int) string {
	lines := []string{
		titleStyle.Render("Keyboard Shortcuts"),
		"",
		"Browse",
		"  ↑/↓, k/j, Ctrl+N          Move selection",
		"  Enter, →, l               Expand/collapse folder",
		"  ←, h                      Collapse folder",
		"  g / G                     Jump to top / bottom",
		"  Ctrl+P                    Open search popup",
		"  Ctrl+O                    Open recent-files popup",
		"  o                         Open heading outline popup",
		"  Ctrl+W                    Open workspace popup",
		"  x                         Export current note (HTML/PDF)",
		"  Shift+L                   Open wiki-links popup",
		"  z                         Toggle split mode",
		"  Tab                       Toggle split focus",
		"  n                         New note",
		"  f                         New folder",
		"  e                         Edit note",
		"  r                         Rename selected item",
		"  m                         Move selected item",
		"  d                         Delete (with confirmation)",
		"  Shift+R / Ctrl+R          Refresh",
		"  s                         Cycle tree sort mode",
		"  t                         Pin/unpin selected item",
		"  y / Y                     Copy note content / path",
		"  ?                         Toggle help",
		"  q or Ctrl+C               Quit",
	}
	if m.git.isRepo {
		lines = append(lines,
			"  c                         Git add+commit",
			"  p                         Git pull --ff-only",
			"  P                         Git push",
		)
	}
	lines = append(lines,
		"",
		"CLI",
		"  notes --configure         Re-run configurator",
		"",
		"Search Popup",
		"  Type                Filter folders by name, notes by name/content",
		"  ↑/↓, j/k            Move search selection",
		"  Enter               Jump to selected result",
		"  Esc                 Close search popup",
		"",
		"Recent Files Popup",
		"  ↑/↓, j/k            Move recent selection",
		"  Enter               Jump to selected recent note",
		"  Esc                 Close popup",
		"",
		"Heading Outline Popup",
		"  o                   Open heading outline for current note",
		"  ↑/↓, j/k            Move heading selection",
		"  Enter               Jump preview to heading",
		"  Esc                 Close popup",
		"",
		"New Note/Folder",
		"  Enter or Ctrl+S  Save",
		"  Esc              Cancel",
		"",
		"Rename/Move/Git Commit",
		"  Enter or Ctrl+S  Save",
		"  Esc              Cancel",
		"",
		"Template Picker",
		"  ↑/↓, j/k         Move template selection",
		"  Enter            Choose template",
		"  Esc              Cancel new-note flow",
		"",
		"Draft Recovery",
		"  y                Recover draft",
		"  n                Discard draft",
		"  Esc              Skip remaining drafts",
		"",
		"Delete Confirmation",
		"  y                Confirm delete",
		"  n or Esc         Cancel delete",
		"",
		"Edit Note",
		"  Ctrl+S         Save",
		"  Shift+Arrows   Extend selection",
		"  Shift+Home/End Extend selection to line boundaries",
		"  Alt+S          Set/clear selection anchor",
		"  Ctrl+B         Toggle **bold** on selection/word",
		"  Alt+I          Toggle *italic* on selection/word",
		"  Ctrl+U         Toggle <u>underline</u> on selection/word",
		"  Alt+X          Toggle ~~strikethrough~~ on selection/word",
		"  Ctrl+K         Insert [text](url) link template",
		"  Ctrl+1..3      Toggle # / ## / ### heading on current line",
		"  Ctrl+V         Paste clipboard text",
		"  Esc            Cancel",
		"",
		"Press ? to return.",
	)

	visible := min(height, len(lines))
	out := make([]string, 0, visible)
	for i := 0; i < visible; i++ {
		out = append(out, truncate(lines[i], width))
	}
	return strings.Join(out, "\n")
}

// inputModeMeta returns three strings for the text-input modes (new note,
// new folder, rename, move, git commit): a title prompt, a location/context
// line, and a helper hint line. These are displayed above the text input
// widget to orient the user on what they're being asked to enter.
func (m *Model) inputModeMeta() (string, string, string) {
	switch m.mode {
	case modeNewFolder:
		return "New folder name", "Location: " + m.displayRelative(m.newParent), "Ctrl+S or Enter to save. Esc to cancel."
	case modeRenameItem:
		return "Rename selected item", "Current path: " + m.displayRelative(m.actionPath), "Ctrl+S or Enter to save. Esc to cancel."
	case modeMoveItem:
		return "Move selected item", "Current path: " + m.displayRelative(m.actionPath), "Enter destination folder path. Esc to cancel."
	case modeGitCommit:
		return "Git commit message", "Repository: " + m.notesDir, "Ctrl+S or Enter to commit. Esc to cancel."
	default:
		return "New note name", "Location: " + m.displayRelative(m.newParent), "Ctrl+S or Enter to save. Esc to cancel."
	}
}

// renderSearchPopupOverlay sizes and centers the search popup within the
// available terminal area. The popup width is clamped between 44 and 70
// columns; height between SearchPopupHeight and 16 rows, leaving some
// margin around the edges.
func (m *Model) renderSearchPopupOverlay(width, height int) string {
	popupWidth := min(70, max(44, width-SearchPopupPadding))
	popupHeight := min(16, max(SearchPopupHeight, height-4))
	popup := m.renderSearchPopup(popupWidth, popupHeight)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, popup)
}

// renderRecentPopupOverlay sizes and centers the recent-files popup.
func (m *Model) renderRecentPopupOverlay(width, height int) string {
	popupWidth := min(70, max(44, width-SearchPopupPadding))
	popupHeight := min(18, max(RecentPopupHeight, height-4))
	popup := m.renderRecentPopup(popupWidth, popupHeight)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, popup)
}

// renderOutlinePopupOverlay sizes and centers the heading outline popup.
func (m *Model) renderOutlinePopupOverlay(width, height int) string {
	popupWidth := min(80, max(50, width-SearchPopupPadding))
	popupHeight := min(20, max(OutlinePopupHeight, height-4))
	popup := m.renderOutlinePopup(popupWidth, popupHeight)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, popup)
}

// renderWorkspacePopupOverlay sizes and centers the workspace chooser popup.
func (m *Model) renderWorkspacePopupOverlay(width, height int) string {
	popupWidth := min(80, max(48, width-SearchPopupPadding))
	popupHeight := min(20, max(WorkspacePopupHeight, height-4))
	popup := m.renderWorkspacePopup(popupWidth, popupHeight)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, popup)
}

// renderExportPopupOverlay sizes and centers the export format popup.
func (m *Model) renderExportPopupOverlay(width, height int) string {
	popupWidth := min(52, max(40, width-SearchPopupPadding))
	popupHeight := min(12, max(ExportPopupHeight, height-4))
	popup := m.renderExportPopup(popupWidth, popupHeight)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, popup)
}

// renderWikiLinksPopupOverlay sizes and centers the wiki-links popup.
func (m *Model) renderWikiLinksPopupOverlay(width, height int) string {
	popupWidth := min(90, max(52, width-SearchPopupPadding))
	popupHeight := min(20, max(WikiLinksPopupHeight, height-4))
	popup := m.renderWikiLinksPopup(popupWidth, popupHeight)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, popup)
}

// renderWikiAutocompletePopupOverlay sizes and bottom-aligns the wiki
// autocomplete popup. It is placed at the bottom of the screen (rather than
// centered) so it appears near the editor cursor where the user is typing.
func (m *Model) renderWikiAutocompletePopupOverlay(width, height int) string {
	popupWidth := min(70, max(42, width-SearchPopupPadding))
	popupHeight := min(16, max(WikiAutocompletePopupHeight, height-4))
	popup := m.renderWikiAutocompletePopup(popupWidth, popupHeight)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Bottom, popup)
}

// renderSearchPopup draws the interior content of the Ctrl+P search popup:
// a title, the search text input, and a scrollable list of matching results
// with the selected entry highlighted. Results show relative paths with a
// trailing "/" for directories.
func (m *Model) renderSearchPopup(width, height int) string {
	innerWidth := max(0, width-popupStyle.GetHorizontalFrameSize())
	innerHeight := max(0, height-popupStyle.GetVerticalFrameSize())
	m.search.Width = innerWidth

	lines := []string{
		titleStyle.Render("Search Notes (Ctrl+P)"),
		m.search.View(),
		"",
	}

	limit := max(0, innerHeight-len(lines)-1)
	for i := 0; i < min(limit, len(m.searchResults)); i++ {
		item := m.searchResults[i]
		label := m.displayRelative(item.path)
		if item.isDir {
			label += "/"
		}
		line := truncate(label, innerWidth)
		if i == m.searchResultCursor {
			line = selectedStyle.Render(line)
		}
		lines = append(lines, line)
	}

	if len(m.searchResults) == 0 {
		lines = append(lines, mutedStyle.Render("No matches yet"))
	}
	lines = append(lines, mutedStyle.Render("Enter: jump  Esc: close"))

	content := padBlock(strings.Join(lines, "\n"), innerWidth, innerHeight)
	return popupStyle.Width(width).Height(height).Render(content)
}

// renderRecentPopup draws the interior content of the Ctrl+O recent-files
// popup: a title and a list of recently viewed notes with relative paths.
func (m *Model) renderRecentPopup(width, height int) string {
	innerWidth := max(0, width-popupStyle.GetHorizontalFrameSize())
	innerHeight := max(0, height-popupStyle.GetVerticalFrameSize())
	lines := []string{
		titleStyle.Render("Recent Files (Ctrl+O)"),
		"",
	}
	limit := max(0, innerHeight-len(lines)-1)
	for i := 0; i < min(limit, len(m.recentEntries)); i++ {
		label := truncate(m.displayRelative(m.recentEntries[i]), innerWidth)
		if i == m.recentCursor {
			label = selectedStyle.Render(label)
		}
		lines = append(lines, label)
	}
	if len(m.recentEntries) == 0 {
		lines = append(lines, mutedStyle.Render("No recent files"))
	}
	lines = append(lines, mutedStyle.Render("Enter: jump  Esc: close"))
	content := padBlock(strings.Join(lines, "\n"), innerWidth, innerHeight)
	return popupStyle.Width(width).Height(height).Render(content)
}

// renderOutlinePopup draws the heading outline popup for the current note.
// Headings are indented by level (two spaces per level) to show the document
// structure at a glance. Selecting a heading and pressing Enter scrolls the
// preview to that section.
func (m *Model) renderOutlinePopup(width, height int) string {
	innerWidth := max(0, width-popupStyle.GetHorizontalFrameSize())
	innerHeight := max(0, height-popupStyle.GetVerticalFrameSize())
	lines := []string{
		titleStyle.Render("Heading Outline (o)"),
		"",
	}
	limit := max(0, innerHeight-len(lines)-1)
	for i := 0; i < min(limit, len(m.outlineHeadings)); i++ {
		heading := m.outlineHeadings[i]
		indent := strings.Repeat("  ", max(0, heading.Level-1))
		label := truncate(fmt.Sprintf("%s%s", indent, heading.Title), innerWidth)
		if i == m.outlineCursor {
			label = selectedStyle.Render(label)
		}
		lines = append(lines, label)
	}
	if len(m.outlineHeadings) == 0 {
		lines = append(lines, mutedStyle.Render("No headings"))
	}
	lines = append(lines, mutedStyle.Render("Enter: jump  Esc: close"))
	content := padBlock(strings.Join(lines, "\n"), innerWidth, innerHeight)
	return popupStyle.Width(width).Height(height).Render(content)
}

// renderWorkspacePopup draws the workspace chooser popup listing all
// configured workspaces. The active workspace is prefixed with "* " so the
// user can see which one is currently in use.
func (m *Model) renderWorkspacePopup(width, height int) string {
	innerWidth := max(0, width-popupStyle.GetHorizontalFrameSize())
	innerHeight := max(0, height-popupStyle.GetVerticalFrameSize())
	lines := []string{
		titleStyle.Render("Switch Workspace"),
		"",
	}
	limit := max(0, innerHeight-len(lines)-1)
	for i := 0; i < min(limit, len(m.workspaces)); i++ {
		ws := m.workspaces[i]
		label := ws.Name + "  (" + ws.NotesDir + ")"
		if ws.Name == m.activeWorkspace {
			label = "* " + label
		}
		label = truncate(label, innerWidth)
		if i == m.workspaceCursor {
			label = selectedStyle.Render(label)
		}
		lines = append(lines, label)
	}
	lines = append(lines, mutedStyle.Render("Enter: switch  Esc: close"))
	content := padBlock(strings.Join(lines, "\n"), innerWidth, innerHeight)
	return popupStyle.Width(width).Height(height).Render(content)
}

// renderExportPopup draws the export format chooser with HTML and PDF options.
func (m *Model) renderExportPopup(width, height int) string {
	innerWidth := max(0, width-popupStyle.GetHorizontalFrameSize())
	innerHeight := max(0, height-popupStyle.GetVerticalFrameSize())
	options := []string{"HTML", "PDF (pandoc)"}
	lines := []string{
		titleStyle.Render("Export Note"),
		"",
	}
	for i, opt := range options {
		line := truncate(opt, innerWidth)
		if i == m.exportCursor {
			line = selectedStyle.Render(line)
		}
		lines = append(lines, line)
	}
	lines = append(lines, "")
	lines = append(lines, mutedStyle.Render("Enter: export  Esc: cancel"))
	content := padBlock(strings.Join(lines, "\n"), innerWidth, innerHeight)
	return popupStyle.Width(width).Height(height).Render(content)
}

// renderTemplatePicker draws the template selection list shown during the
// new-note flow when templates are available. The first entry is always
// "Default (no template)"; subsequent entries are files from the templates
// directory.
func (m *Model) renderTemplatePicker(width, height int) string {
	lines := []string{
		titleStyle.Render("Choose Note Template"),
		"",
	}
	for i, tpl := range m.templates {
		line := tpl.name
		if i == m.templateCursor {
			line = selectedStyle.Render(line)
		}
		lines = append(lines, truncate(line, width))
	}
	lines = append(lines, "")
	lines = append(lines, mutedStyle.Render("Enter: choose template  Esc: cancel"))

	visible := min(height, len(lines))
	return strings.Join(lines[:visible], "\n")
}

// renderDraftRecovery draws the startup draft recovery prompt. If a draft
// is pending, the user is shown the source note path and offered three
// choices: recover (y), discard (n), or skip remaining drafts (Esc).
func (m *Model) renderDraftRecovery(width, height int) string {
	lines := []string{
		titleStyle.Render("Unsaved Draft Recovery"),
		"",
	}
	if m.activeDraft == nil {
		lines = append(lines, "No draft selected.")
	} else {
		lines = append(lines, "Recover draft for:")
		lines = append(lines, truncate(m.displayRelative(m.activeDraft.SourcePath), width))
		lines = append(lines, "")
		lines = append(lines, mutedStyle.Render("y: recover and overwrite note"))
		lines = append(lines, mutedStyle.Render("n: discard this draft"))
		lines = append(lines, mutedStyle.Render("Esc: skip remaining drafts"))
	}

	visible := min(height, len(lines))
	return strings.Join(lines[:visible], "\n")
}

// rightHeaderPath returns the display text for the right-pane header bar:
// the relative path of the current note, or "No note selected" as a fallback.
func (m *Model) rightHeaderPath() string {
	path := "No note selected"
	if m.currentFile != "" {
		path = m.displayRelative(m.currentFile)
	}
	return path
}

// renderRightHeader renders the solid-color header bar at the top of the right
// pane showing the current file path. The style parameter determines the
// background color (blue for preview, magenta for edit).
func (m *Model) renderRightHeader(width int, style lipgloss.Style) string {
	line := " " + truncate(m.rightHeaderPath(), max(0, width-1))
	return style.Width(width).Render(line)
}

// formatTreeItem formats a non-selected directory or file row with styled
// indentation, type markers ([+]/[-] for dirs), badges (DIR/MD/PIN), and
// optional tag labels. Colors are applied via Lipgloss styles so the row
// is visually distinct from the selected row.
func (m *Model) formatTreeItem(item treeItem) string {
	indent := strings.Repeat("  ", item.depth)
	if item.isDir {
		expanded := m.expanded[item.path]
		marker := treeClosedMark.Render("[+]")
		if expanded || strings.TrimSpace(m.search.Value()) != "" {
			marker = treeOpenMark.Render("[-]")
		}
		pin := ""
		if item.pinned {
			pin = " " + treePinTag.Render("PIN")
		}
		return fmt.Sprintf("%s%s %s %s%s", indent, marker, treeDirTag.Render("DIR"), treeDirName.Render(item.name), pin)
	}
	pin := ""
	if item.pinned {
		pin = " " + treePinTag.Render("PIN")
	}
	tagBadge := ""
	if label := compactTagLabel(item.tags, 2); label != "" {
		tagBadge = " " + treeTagBadge.Render("TAGS:"+label)
	}
	return fmt.Sprintf("%s    %s %s%s%s", indent, treeFileTag.Render("MD"), treeFileName.Render(item.name), pin, tagBadge)
}

// formatTreeItemSelected formats the currently selected tree row using plain
// unstyled text (no Lipgloss colors). The selected row is then wrapped in
// selectedStyle (reversed colors) by renderTree, so applying colors here
// would create illegible double-styling. The markers, badges, and layout
// match formatTreeItem but without color codes.
func (m *Model) formatTreeItemSelected(item treeItem) string {
	indent := strings.Repeat("  ", item.depth)
	if item.isDir {
		expanded := m.expanded[item.path]
		marker := "[+]"
		if expanded || strings.TrimSpace(m.search.Value()) != "" {
			marker = "[-]"
		}
		pin := ""
		if item.pinned {
			pin = " PIN"
		}
		return fmt.Sprintf("%s%s DIR %s%s", indent, marker, item.name, pin)
	}
	pin := ""
	if item.pinned {
		pin = " PIN"
	}
	tagBadge := ""
	if label := compactTagLabel(item.tags, 2); label != "" {
		tagBadge = " TAGS:" + label
	}
	return fmt.Sprintf("%s    MD %s%s%s", indent, item.name, pin, tagBadge)
}

// updateLayout recomputes viewport sizing after a window resize. This is a
// convenience wrapper that calculates the new layout dimensions and applies
// them to the viewport widget in a single call.
func (m *Model) updateLayout() {
	layout := m.calculateLayout()
	m.applyLayout(layout)
}
