package app

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
			content = m.renderPreviewWithOffset(path, rendered, secondary)
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
	if selected == "" {
		return view
	}
	segments := selectionSegments(selected)
	if len(segments) == 0 {
		return view
	}
	return highlightSelectionSegmentsInView(view, segments)
}

func selectionSegments(selected string) []string {
	parts := strings.Split(selected, "\n")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		segments = append(segments, part)
	}
	return segments
}

func highlightSelectionSegmentsInView(view string, segments []string) string {
	searchFrom := 0
	for _, segment := range segments {
		if segment == "" {
			continue
		}
		idx := strings.Index(view[searchFrom:], segment)
		if idx < 0 {
			continue
		}
		absolute := searchFrom + idx
		highlighted := selectionText.Render(segment)
		view = view[:absolute] + highlighted + view[absolute+len(segment):]
		searchFrom = absolute + len(highlighted)
	}
	return view
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
