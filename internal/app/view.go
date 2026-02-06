package app

import (
	"fmt"
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
		content = m.editor.View()
	case modeNewNote, modeNewFolder:
		m.input.Width = innerWidth
		prompt := "New note name"
		if m.mode == modeNewFolder {
			prompt = "New folder name"
		}
		location := "Location: " + m.displayRelative(m.newParent)
		helper := "Ctrl+S or Enter to save. Esc to cancel."
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

// renderStatus renders the footer help line and any status message.
func (m *Model) renderStatus(width int) string {
	help := m.statusHelp()
	line := help
	if m.status != "" {
		line = help + " | " + m.status
	}
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
		return "Ctrl+S save  Ctrl+B bold  Alt+I italic  Ctrl+U underline  Esc cancel"
	case modeNewNote, modeNewFolder:
		return "Enter/Ctrl+S save  Esc cancel"
	default:
		if m.searching {
			return "Search popup: type  ↑/↓ move  Enter jump  Esc cancel"
		}
		return "↑/↓ or k/j move  Enter/→/l toggle  ←/h collapse  g/G top/bottom  Ctrl+P search  n new  f folder  e edit  d delete  r refresh  ? help  q quit  (reconfigure: notes --configure)"
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
		"  n                         New note",
		"  f                         New folder",
		"  e                         Edit note",
		"  d                         Delete",
		"  r                         Refresh",
		"  ?                         Toggle help",
		"  q or Ctrl+C               Quit",
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
		"New Note/Folder",
		"  Enter or Ctrl+S  Save",
		"  Esc              Cancel",
		"",
		"Edit Note",
		"  Ctrl+S         Save",
		"  Ctrl+B         Insert **bold** markers",
		"  Alt+I          Insert *italic* markers",
		"  Ctrl+U         Insert <u>underline</u> markers",
		"  Esc            Cancel",
		"",
		"Press ? to return.",
	}

	visible := min(height, len(lines))
	out := make([]string, 0, visible)
	for i := 0; i < visible; i++ {
		out = append(out, truncate(lines[i], width))
	}
	return strings.Join(out, "\n")
}

func (m *Model) renderSearchPopupOverlay(width, height int) string {
	popupWidth := min(70, max(44, width-SearchPopupPadding))
	popupHeight := min(16, max(SearchPopupHeight, height-4))
	popup := m.renderSearchPopup(popupWidth, popupHeight)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, popup)
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
		return fmt.Sprintf("%s%s %s %s", indent, marker, treeDirTag.Render("DIR"), treeDirName.Render(item.name))
	}
	return fmt.Sprintf("%s    %s %s", indent, treeFileTag.Render("MD"), treeFileName.Render(item.name))
}

func (m *Model) formatTreeItemSelected(item treeItem) string {
	indent := strings.Repeat("  ", item.depth)
	if item.isDir {
		expanded := m.expanded[item.path]
		marker := "[+]"
		if expanded || strings.TrimSpace(m.search.Value()) != "" {
			marker = "[-]"
		}
		return fmt.Sprintf("%s%s DIR %s", indent, marker, item.name)
	}
	return fmt.Sprintf("%s    MD %s", indent, item.name)
}

// updateLayout recomputes viewport sizing after a window resize.
func (m *Model) updateLayout() {
	layout := m.calculateLayout()
	m.applyLayout(layout)
}
