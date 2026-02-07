package app

import (
	"fmt"
	"strings"
)

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
