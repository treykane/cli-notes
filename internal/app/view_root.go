package app

import "github.com/charmbracelet/lipgloss"

// View draws the full UI (left tree + right pane + status footer).
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	footerHeight := m.footerHeightForWidth(m.width)
	layout := m.calculateLayout()
	leftPane := m.renderTree(layout.LeftWidth, layout.ContentHeight)
	rightPane := m.renderRight(layout.RightWidth, layout.ContentHeight)
	row := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	if m.overlay != overlayNone {
		row = m.renderActiveOverlay(m.width, layout.ContentHeight)
	}
	row = padBlock(row, m.width, layout.ContentHeight)

	view := row + "\n" + m.renderStatus(m.width, footerHeight)
	return padBlock(view, m.width, m.height)
}
