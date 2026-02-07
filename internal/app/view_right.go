package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	rw "github.com/mattn/go-runewidth"
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
			m.helpViewport.Width = innerWidth
			m.helpViewport.Height = contentHeight
			m.helpViewport.SetContent(m.helpContent())
			content = m.helpViewport.View()
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

	spans := m.editorSelectionRowSpans(start, end)
	if len(spans) == 0 {
		return view
	}

	lines := strings.Split(view, "\n")
	contentStart := m.editorContentStartColumn()
	for _, span := range spans {
		if span.row < 0 || span.row >= len(lines) {
			continue
		}
		lines[span.row] = highlightEditorRowSpan(lines[span.row], contentStart, span.startCol, span.endCol)
	}
	return strings.Join(lines, "\n")
}

type editorRowSelectionSpan struct {
	row      int
	startCol int
	endCol   int
}

func (m *Model) editorSelectionRowSpans(start, end int) []editorRowSelectionSpan {
	value := m.editor.Value()
	runes := []rune(value)
	start = clamp(start, 0, len(runes))
	end = clamp(end, 0, len(runes))
	if start > end {
		start, end = end, start
	}
	if start >= end {
		return nil
	}

	lines := splitEditorLines(value)
	wrapWidth := max(1, m.editor.Width())
	spans := make([]editorRowSelectionSpan, 0, len(lines))

	globalOffset := 0
	rowIndex := 0
	for i, line := range lines {
		wrapped := wrapEditorLineWithSources(line, wrapWidth)
		for _, row := range wrapped {
			if startCol, endCol, ok := selectionColumnsForWrappedRow(row, globalOffset, start, end); ok {
				spans = append(spans, editorRowSelectionSpan{
					row:      rowIndex,
					startCol: startCol,
					endCol:   endCol,
				})
			}
			rowIndex++
		}

		globalOffset += len(line)
		if i < len(lines)-1 {
			globalOffset++
		}
	}

	return spans
}

func selectionColumnsForWrappedRow(row []wrappedEditorCell, lineOffset, selectionStart, selectionEnd int) (startCol, endCol int, ok bool) {
	col := 0
	startCol = -1
	endCol = -1
	for _, cell := range row {
		cellWidth := max(1, cell.display)
		if cell.source >= 0 {
			offset := lineOffset + cell.source
			if offset >= selectionStart && offset < selectionEnd {
				if startCol < 0 {
					startCol = col
				}
				endCol = col + cellWidth
			}
		}
		col += cellWidth
	}

	if startCol < 0 || endCol <= startCol {
		return 0, 0, false
	}
	return startCol, endCol, true
}

func (m *Model) editorContentStartColumn() int {
	gutter := lipgloss.Width(m.editor.Prompt)
	if m.editor.ShowLineNumbers {
		gutter += len(fmt.Sprintf("%3v ", max(1, m.editor.LineCount())))
	}
	return max(0, gutter)
}

func highlightEditorRowSpan(line string, contentStart, startCol, endCol int) string {
	if endCol <= startCol {
		return line
	}

	raw := ansi.Strip(line)
	if raw == "" {
		return line
	}

	gutter, content := splitByDisplayColumn(raw, contentStart)
	startCol = clamp(startCol, 0, rw.StringWidth(content))
	endCol = clamp(endCol, startCol, rw.StringWidth(content))
	if endCol <= startCol {
		return line
	}

	before, selected, after := splitByDisplayRange(content, startCol, endCol)
	if selected == "" {
		return line
	}

	switch renderedLineStyleKind(line, raw) {
	case renderedLineStyleCode:
		return editorCodeLine.Render(gutter+before) + selectionText.Render(selected) + editorCodeLine.Render(after)
	case renderedLineStyleFence:
		return editorFenceLine.Render(gutter+before) + selectionText.Render(selected) + editorFenceLine.Render(after)
	default:
		return gutter + before + selectionText.Render(selected) + after
	}
}

type renderedLineStyle int

const (
	renderedLineStyleNone renderedLineStyle = iota
	renderedLineStyleCode
	renderedLineStyleFence
)

func renderedLineStyleKind(line, raw string) renderedLineStyle {
	switch {
	case line == editorCodeLine.Render(raw):
		return renderedLineStyleCode
	case line == editorFenceLine.Render(raw):
		return renderedLineStyleFence
	default:
		return renderedLineStyleNone
	}
}

func splitByDisplayRange(s string, startCol, endCol int) (before, middle, after string) {
	before, rest := splitByDisplayColumn(s, startCol)
	middle, after = splitByDisplayColumn(rest, max(0, endCol-startCol))
	return before, middle, after
}

func splitByDisplayColumn(s string, col int) (left, right string) {
	if col <= 0 {
		return "", s
	}

	runes := []rune(s)
	width := 0
	for i, r := range runes {
		w := rw.RuneWidth(r)
		if w <= 0 {
			w = 1
		}
		if width+w > col {
			return string(runes[:i]), string(runes[i:])
		}
		width += w
		if width == col {
			return string(runes[:i+1]), string(runes[i+1:])
		}
	}
	return s, ""
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
