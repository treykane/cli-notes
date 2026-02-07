package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View draws the full UI (left tree + right pane + status line).
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

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
	// Clamp the pane row so the last terminal line is always reserved for footer status.
	row = padBlock(row, m.width, layout.ContentHeight)

	view := row + "\n" + m.renderStatus(m.width)
	return padBlock(view, m.width, m.height)
}

// renderTree draws the left-hand directory tree pane.
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

func (m *Model) renderRightSplit(width, height int) string {
	leftWidth := width / 2
	rightWidth := width - leftWidth
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.renderSingleRightPane(leftWidth, height, m.currentFile, false, !m.splitFocusSecondary),
		m.renderSingleRightPane(rightWidth, height, m.secondaryFile, true, m.splitFocusSecondary),
	)
}

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
			content = rendered
		}
	}

	header := headerStyle.Width(innerWidth).Render(" " + truncate(headerLabel, max(0, innerWidth-1)))
	body := padBlock(content, innerWidth, contentHeight)
	return rightPaneStyle.Width(width).Height(height).Render(header + "\n" + body)
}

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

// renderStatus renders the footer help line and any status message.
func (m *Model) renderStatus(width int) string {
	help := m.statusHelp()
	parts := []string{help}
	if (m.mode == modeBrowse || m.mode == modeEditNote) && m.currentFile != "" {
		if metrics := m.noteMetricsSummary(); metrics != "" {
			parts = append(parts, metrics)
		}
	}
	if git := m.gitFooterSummary(); git != "" {
		parts = append(parts, git)
	}
	if m.status != "" {
		parts = append(parts, m.status)
	}
	line := strings.Join(parts, " | ")
	line = " " + truncate(line, max(0, width-1))
	style := statusStyle
	if m.mode == modeEditNote {
		style = editStatus
	}
	return style.Width(width).Render(line)
}

func (m *Model) statusHelp() string {
	switch m.mode {
	case modeEditNote:
		return "Ctrl+S save  Shift+Arrows select  Alt+S anchor  Ctrl+B bold  Alt+I italic  Ctrl+U underline  Alt+X strike  Ctrl+K link  Ctrl+1..3 heading  Ctrl+V paste  Esc cancel"
	case modeNewNote, modeNewFolder, modeRenameItem, modeMoveItem, modeGitCommit:
		return "Enter/Ctrl+S save  Esc cancel"
	case modeTemplatePicker:
		return "Template picker: ↑/↓ move  Enter choose  Esc cancel"
	case modeDraftRecovery:
		return "Draft recovery: y recover  n discard  Esc skip all"
	case modeConfirmDelete:
		return "y confirm delete  n/Esc cancel"
	default:
		if m.searching {
			return "Search popup: type  ↑/↓ move  Enter jump  Esc cancel"
		}
		if m.showRecentPopup {
			return "Recent popup: ↑/↓ move  Enter jump  Esc cancel"
		}
		if m.showOutlinePopup {
			return "Outline popup: ↑/↓ move  Enter jump  Esc cancel"
		}
		if m.showWorkspacePopup {
			return "Workspace popup: ↑/↓ move  Enter switch  Esc cancel"
		}
		if m.showExportPopup {
			return "Export popup: ↑/↓ move  Enter export  Esc cancel"
		}
		if m.showWikiLinksPopup {
			return "Wiki links popup: ↑/↓ move  Enter jump  Esc cancel"
		}
		if m.showWikiAutocomplete {
			return "Wiki autocomplete: ↑/↓ move  Tab/Enter insert  Esc close"
		}
		help := "↑/↓ or k/j move  Enter/→/l toggle  ←/h collapse  g/G top/bottom  Ctrl+P search  n new  f folder  e edit  r rename  m move  d delete  Shift+R refresh"
		help += "  s sort  t pin"
		help += "  Ctrl+O recents  o outline  Ctrl+W workspaces"
		help += "  x export  Shift+L wiki links  z split  Tab split-focus"
		help += "  y copy content  Y copy path"
		if m.git.isRepo {
			help += "  c commit  p pull  P push"
		}
		help += "  ? help  q quit  (reconfigure: notes --configure)"
		return help
	}
}

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

func (m *Model) renderSearchPopupOverlay(width, height int) string {
	popupWidth := min(70, max(44, width-SearchPopupPadding))
	popupHeight := min(16, max(SearchPopupHeight, height-4))
	popup := m.renderSearchPopup(popupWidth, popupHeight)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, popup)
}

func (m *Model) renderRecentPopupOverlay(width, height int) string {
	popupWidth := min(70, max(44, width-SearchPopupPadding))
	popupHeight := min(18, max(RecentPopupHeight, height-4))
	popup := m.renderRecentPopup(popupWidth, popupHeight)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, popup)
}

func (m *Model) renderOutlinePopupOverlay(width, height int) string {
	popupWidth := min(80, max(50, width-SearchPopupPadding))
	popupHeight := min(20, max(OutlinePopupHeight, height-4))
	popup := m.renderOutlinePopup(popupWidth, popupHeight)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, popup)
}

func (m *Model) renderWorkspacePopupOverlay(width, height int) string {
	popupWidth := min(80, max(48, width-SearchPopupPadding))
	popupHeight := min(20, max(WorkspacePopupHeight, height-4))
	popup := m.renderWorkspacePopup(popupWidth, popupHeight)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, popup)
}

func (m *Model) renderExportPopupOverlay(width, height int) string {
	popupWidth := min(52, max(40, width-SearchPopupPadding))
	popupHeight := min(12, max(ExportPopupHeight, height-4))
	popup := m.renderExportPopup(popupWidth, popupHeight)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, popup)
}

func (m *Model) renderWikiLinksPopupOverlay(width, height int) string {
	popupWidth := min(90, max(52, width-SearchPopupPadding))
	popupHeight := min(20, max(WikiLinksPopupHeight, height-4))
	popup := m.renderWikiLinksPopup(popupWidth, popupHeight)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, popup)
}

func (m *Model) renderWikiAutocompletePopupOverlay(width, height int) string {
	popupWidth := min(70, max(42, width-SearchPopupPadding))
	popupHeight := min(16, max(WikiAutocompletePopupHeight, height-4))
	popup := m.renderWikiAutocompletePopup(popupWidth, popupHeight)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Bottom, popup)
}

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

func (m *Model) rightHeaderPath() string {
	path := "No note selected"
	if m.currentFile != "" {
		path = m.displayRelative(m.currentFile)
	}
	return path
}

func (m *Model) renderRightHeader(width int, style lipgloss.Style) string {
	line := " " + truncate(m.rightHeaderPath(), max(0, width-1))
	return style.Width(width).Render(line)
}

// formatTreeItem formats a directory or file row with indentation and markers.
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

// updateLayout recomputes viewport sizing after a window resize.
func (m *Model) updateLayout() {
	layout := m.calculateLayout()
	m.applyLayout(layout)
}
