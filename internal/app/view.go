// view.go implements popup and auxiliary rendering helpers for the terminal UI.
package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderSearchPopupOverlay sizes and centers the search popup within the
// available terminal area.
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

// renderWikiAutocompletePopupOverlay sizes and bottom-aligns the wiki autocomplete popup.
func (m *Model) renderWikiAutocompletePopupOverlay(width, height int) string {
	popupWidth := min(70, max(42, width-SearchPopupPadding))
	popupHeight := min(16, max(WikiAutocompletePopupHeight, height-4))
	popup := m.renderWikiAutocompletePopup(popupWidth, popupHeight)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Bottom, popup)
}

// renderSearchPopup draws the interior content of the Ctrl+P search popup.
func (m *Model) renderSearchPopup(width, height int) string {
	innerWidth := max(0, width-popupStyle.GetHorizontalFrameSize())
	innerHeight := max(0, height-popupStyle.GetVerticalFrameSize())
	m.search.Width = innerWidth

	lines := []string{
		titleStyle.Render("Search Notes (" + m.primaryActionKey(actionSearch, "Ctrl+P") + ")"),
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

	query := strings.TrimSpace(m.search.Value())
	if len(m.searchResults) == 0 {
		lines = append(lines, mutedStyle.Render("No matches yet"))
	}
	if query != "" {
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("%d matches", len(m.searchResults))))
		if len(m.searchResults) > 0 {
			lines = append(lines, mutedStyle.Render(fmt.Sprintf("%d of %d", m.searchResultCursor+1, len(m.searchResults))))
		}
	}
	lines = append(lines, mutedStyle.Render("Enter: jump  Esc: close"))

	content := padBlock(strings.Join(lines, "\n"), innerWidth, innerHeight)
	return popupStyle.Width(width).Height(height).Render(content)
}

// renderRecentPopup draws the interior content of the Ctrl+O recent-files popup.
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

// renderWorkspacePopup draws the workspace chooser popup.
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

// renderTemplatePicker draws the template selection list shown during the new-note flow.
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

// renderDraftRecovery draws the startup draft recovery prompt.
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

// updateLayout recomputes viewport sizing after a window resize.
func (m *Model) updateLayout() {
	layout := m.calculateLayout()
	m.applyLayout(layout)
}
