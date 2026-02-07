package app

import (
	"fmt"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	rw "github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
)

func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.mode != modeEditNote {
		return m, nil
	}
	return m.handleEditMouse(msg)
}

func (m *Model) handleEditMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft {
			return m, nil
		}
		offset, ok := m.editorOffsetFromMouse(msg)
		if !ok {
			return m, nil
		}
		if m.isOverlay(overlayWikiAutocomplete) {
			m.closeOverlay()
		}
		m.setEditorValueAndCursorOffset(m.editor.Value(), offset)
		m.editorSelectionAnchor = offset
		m.editorSelectionActive = true
		m.editorMouseSelecting = true
		m.editorMouseSelectionOrigin = offset
		applyEditorSelectionVisual(&m.editor)
		m.updateEditorSelectionStatus()
	case tea.MouseActionMotion:
		if !m.editorMouseSelecting {
			return m, nil
		}
		offset, ok := m.editorOffsetFromMouse(msg)
		if !ok {
			return m, nil
		}
		m.setEditorValueAndCursorOffset(m.editor.Value(), offset)
		m.updateEditorSelectionStatus()
	case tea.MouseActionRelease:
		if !m.editorMouseSelecting {
			return m, nil
		}
		m.editorMouseSelecting = false
		m.editorMouseSelectionOrigin = noEditorSelectionAnchor
		if offset, ok := m.editorOffsetFromMouse(msg); ok {
			m.setEditorValueAndCursorOffset(m.editor.Value(), offset)
		}
		if _, _, ok := m.editorSelectionRange(); !ok {
			m.clearEditorSelection()
			m.status = "Selection cleared"
			return m, nil
		}
		m.updateEditorSelectionStatus()
	}
	return m, nil
}

func (m *Model) editPaneContentOrigin(layout LayoutDimensions) (x, y int) {
	x = layout.LeftWidth + editPane.GetBorderLeftSize() + editPane.GetPaddingLeft()
	y = editPane.GetBorderTopSize() + editPane.GetPaddingTop() + 1 // +1 for header line
	return x, y
}

func (m *Model) editorOffsetFromMouse(msg tea.MouseMsg) (int, bool) {
	layout := m.calculateLayout()
	contentOriginX, contentOriginY := m.editPaneContentOrigin(layout)
	paneWidth := layout.RightWidth
	if m.splitMode {
		paneWidth = paneWidth / 2
	}
	paneEndX := layout.LeftWidth + paneWidth
	if msg.X < contentOriginX || msg.X >= paneEndX {
		return 0, false
	}
	if msg.Y < contentOriginY || msg.Y >= contentOriginY+layout.ViewportHeight {
		return 0, false
	}

	gutterWidth := lipgloss.Width(m.editor.Prompt)
	if m.editor.ShowLineNumbers {
		gutterWidth += len(fmt.Sprintf("%3v ", max(1, m.editor.LineCount())))
	}
	col := msg.X - contentOriginX - gutterWidth
	if col < 0 {
		col = 0
	}
	row := msg.Y - contentOriginY

	return m.editorOffsetFromVisualPosition(row, col), true
}

func (m *Model) editorOffsetFromVisualPosition(row, col int) int {
	value := m.editor.Value()
	lines := splitEditorLines(value)
	wrapWidth := max(1, m.editor.Width())
	row = max(0, row)
	col = max(0, col)

	offset := 0
	for i, line := range lines {
		wrapped := wrapEditorLineWithSources(line, wrapWidth)
		if row < len(wrapped) {
			return clamp(offset+lineOffsetForVisualPosition(wrapped[row], col, len(line)), 0, len([]rune(value)))
		}

		row -= len(wrapped)
		offset += len(line)
		if i < len(lines)-1 {
			offset++
		}
	}
	return clamp(offset, 0, len([]rune(value)))
}

type wrappedEditorCell struct {
	r       rune
	source  int
	synth   bool
	display int
}

func wrapEditorLineWithSources(line []rune, width int) [][]wrappedEditorCell {
	if width <= 0 {
		width = 1
	}

	lines := [][]wrappedEditorCell{{}}
	word := []wrappedEditorCell{}
	row := 0
	spaces := 0
	sourceSpaces := []int{}

	for idx, r := range line {
		if unicode.IsSpace(r) {
			spaces++
			sourceSpaces = append(sourceSpaces, idx)
		} else {
			word = append(word, newWrappedEditorCell(r, idx, false))
		}

		if spaces > 0 {
			if wrappedRowDisplayWidth(lines[row])+wrappedCellsDisplayWidth(word)+spaces > width {
				row++
				lines = append(lines, []wrappedEditorCell{})
				lines[row] = append(lines[row], word...)
				lines[row] = append(lines[row], wrappedEditorSpaces(sourceSpaces, spaces, false)...)
			} else {
				lines[row] = append(lines[row], word...)
				lines[row] = append(lines[row], wrappedEditorSpaces(sourceSpaces, spaces, false)...)
			}
			spaces = 0
			sourceSpaces = nil
			word = nil
		} else if len(word) > 0 {
			lastCharLen := rw.RuneWidth(word[len(word)-1].r)
			if lastCharLen <= 0 {
				lastCharLen = 1
			}
			if wrappedCellsDisplayWidth(word)+lastCharLen > width {
				if len(lines[row]) > 0 {
					row++
					lines = append(lines, []wrappedEditorCell{})
				}
				lines[row] = append(lines[row], word...)
				word = nil
			}
		}
	}

	if wrappedRowDisplayWidth(lines[row])+wrappedCellsDisplayWidth(word)+spaces >= width {
		lines = append(lines, []wrappedEditorCell{})
		nextRow := row + 1
		lines[nextRow] = append(lines[nextRow], word...)
		spaces++
		lines[nextRow] = append(lines[nextRow], wrappedEditorSpaces(sourceSpaces, spaces, true)...)
		return lines
	}

	lines[row] = append(lines[row], word...)
	spaces++
	lines[row] = append(lines[row], wrappedEditorSpaces(sourceSpaces, spaces, true)...)
	return lines
}

func lineOffsetForVisualPosition(row []wrappedEditorCell, col int, lineLen int) int {
	col = max(0, col)

	lastSource := -1
	displayCol := 0
	for _, cell := range row {
		if cell.source >= 0 {
			lastSource = max(lastSource, cell.source)
		}
		cellWidth := max(1, cell.display)
		if col < displayCol+cellWidth {
			if cell.synth || cell.source < 0 {
				if lastSource < 0 {
					return lineLen
				}
				return lastSource + 1
			}
			if col > displayCol {
				return cell.source + 1
			}
			return cell.source
		}
		displayCol += cellWidth
	}

	if lastSource < 0 {
		return lineLen
	}
	return lastSource + 1
}

func wrappedRowDisplayWidth(cells []wrappedEditorCell) int {
	return wrappedCellsDisplayWidth(cells)
}

func wrappedCellsDisplayWidth(cells []wrappedEditorCell) int {
	width := 0
	for _, cell := range cells {
		width += max(1, cell.display)
	}
	return width
}

func wrappedEditorSpaces(sourceSpaces []int, count int, appendSynthetic bool) []wrappedEditorCell {
	spaces := make([]wrappedEditorCell, 0, count+1)
	for i := 0; i < count; i++ {
		source := -1
		if i < len(sourceSpaces) {
			source = sourceSpaces[i]
		}
		spaces = append(spaces, newWrappedEditorCell(' ', source, false))
	}
	if appendSynthetic {
		spaces = append(spaces, newWrappedEditorCell(' ', -1, true))
	}
	return spaces
}

func newWrappedEditorCell(r rune, source int, synthetic bool) wrappedEditorCell {
	display := rw.RuneWidth(r)
	if display <= 0 {
		display = uniseg.StringWidth(string(r))
	}
	if display <= 0 {
		display = 1
	}
	return wrappedEditorCell{
		r:       r,
		source:  source,
		synth:   synthetic,
		display: display,
	}
}
