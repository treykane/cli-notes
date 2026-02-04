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

	leftWidth := min(40, m.width/3)
	rightWidth := max(0, m.width-leftWidth-1)
	contentHeight := max(0, m.height-2)

	leftPane := m.renderTree(leftWidth, contentHeight)
	rightPane := m.renderRight(rightWidth, contentHeight)
	row := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	return row + "\n" + m.renderStatus(m.width)
}

// renderTree draws the left-hand directory tree pane.
func (m *Model) renderTree(width, height int) string {
	innerWidth := max(0, width-2)
	innerHeight := max(0, height-2)

	header := titleStyle.Render("Notes: " + m.notesDir)
	lines := []string{truncate(header, innerWidth)}

	visibleHeight := max(0, innerHeight-1)
	start := min(m.treeOffset, max(0, len(m.items)-1))
	end := min(len(m.items), start+visibleHeight)

	for i := start; i < end; i++ {
		item := m.items[i]
		line := m.formatTreeItem(item)
		if i == m.cursor {
			line = selectedStyle.Render(line)
		}
		lines = append(lines, truncate(line, innerWidth))
	}

	content := strings.Join(lines, "\n")
	return paneStyle.Width(width).Height(height).Render(content)
}

// renderRight draws the right-hand pane (editor, input, or markdown viewport).
func (m *Model) renderRight(width, height int) string {
	innerWidth := max(0, width-2)
	innerHeight := max(0, height-2)

	var content string
	switch m.mode {
	case modeEditNote:
		m.editor.SetWidth(innerWidth)
		m.editor.SetHeight(innerHeight)
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
		m.viewport.Width = innerWidth
		m.viewport.Height = innerHeight
		content = m.viewport.View()
	}

	return paneStyle.Width(width).Height(height).Render(content)
}

// renderStatus renders the footer help line and any status message.
func (m *Model) renderStatus(width int) string {
	help := "n new  f folder  e edit  d delete  r refresh  q quit"
	line := help
	if m.status != "" {
		line = help + " | " + m.status
	}
	return statusStyle.Width(width).Render(truncate(line, width))
}

// formatTreeItem formats a directory or file row with indentation and markers.
func (m *Model) formatTreeItem(item treeItem) string {
	indent := strings.Repeat("  ", item.depth)
	if item.isDir {
		expanded := m.expanded[item.path]
		marker := "[+]"
		if expanded {
			marker = "[-]"
		}
		return fmt.Sprintf("%s%s %s", indent, marker, item.name)
	}
	return fmt.Sprintf("%s    %s", indent, item.name)
}

// updateLayout recomputes viewport sizing after a window resize.
func (m *Model) updateLayout() {
	leftWidth := min(40, m.width/3)
	rightWidth := max(0, m.width-leftWidth-1)
	contentHeight := max(0, m.height-2)
	m.viewport.Width = max(0, rightWidth-2)
	m.viewport.Height = max(0, contentHeight-2)
}
