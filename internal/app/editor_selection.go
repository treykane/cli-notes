package app

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
)

// noEditorSelectionAnchor is the sentinel value for editorSelectionAnchor,
// indicating that no selection anchor is currently set. When the anchor
// equals this value, text selection is inactive.
const noEditorSelectionAnchor = -1

// clearEditorSelection resets the editor selection state entirely.
//
// It sets the anchor back to the sentinel value, marks selection as inactive,
// and updates the editor's visual styling to reflect that no text is selected.
// This should be called after any operation that consumes or invalidates the
// selection (e.g. applying formatting, pasting, saving, or cancelling edit mode).
func (m *Model) clearEditorSelection() {
	m.editorSelectionAnchor = noEditorSelectionAnchor
	m.editorSelectionActive = false
	applyEditorSelectionVisual(&m.editor, false)
}

// hasEditorSelectionAnchor reports whether a selection anchor is currently
// active. When true, cursor movement will define a selection range between
// the anchor position and the current cursor position.
func (m *Model) hasEditorSelectionAnchor() bool {
	return m.editorSelectionActive
}

// currentEditorCursorOffset calculates the cursor's position as a rune offset
// from the beginning of the editor's text content.
//
// The editor widget tracks the cursor by (row, column) coordinates internally.
// This function converts those coordinates into a single linear offset that
// can be used for range-based operations like selection, formatting, and
// text replacement.
//
// The conversion works by:
//  1. Splitting the editor value into logical lines.
//  2. Summing the rune lengths of all lines before the current row (plus 1
//     for each newline separator).
//  3. Adding the character offset within the current line.
//
// The result is clamped to the valid range [0, total rune count] to prevent
// out-of-bounds access from edge cases during rapid editing.
func (m *Model) currentEditorCursorOffset() int {
	value := m.editor.Value()
	lines := splitEditorLines(value)
	row := clamp(m.editor.Line(), 0, max(0, len(lines)-1))
	col := clamp(m.editor.LineInfo().CharOffset, 0, len(lines[row]))

	offset := 0
	for i := 0; i < row; i++ {
		offset += len(lines[i]) + 1
	}
	return clamp(offset+col, 0, utf8.RuneCountInString(value))
}

// editorSelectionRange returns the start and end rune offsets of the current
// text selection, along with a boolean indicating whether a valid selection
// exists.
//
// The range is normalized so that start <= end regardless of whether the user
// selected text forwards (anchor before cursor) or backwards (anchor after
// cursor). If no anchor is set or the anchor equals the cursor (zero-length
// selection), ok is false.
//
// This function is the primary entry point for any code that needs to read
// the selected text range — formatting commands, copy operations, and visual
// highlighting all call this.
func (m *Model) editorSelectionRange() (start, end int, ok bool) {
	if !m.hasEditorSelectionAnchor() {
		return 0, 0, false
	}

	cursor := m.currentEditorCursorOffset()
	start = m.editorSelectionAnchor
	end = cursor
	if start > end {
		start, end = end, start
	}
	if start == end {
		return 0, 0, false
	}
	return start, end, true
}

// toggleEditorSelectionAnchor sets or clears the selection anchor at the
// current cursor position.
//
// This implements the Alt+S keybinding behavior:
//   - If no anchor is set: drops an anchor at the current cursor position and
//     enters selection mode. Subsequent cursor movement will extend the
//     selection between the anchor and the moving cursor.
//   - If an anchor is already set: clears the selection entirely, returning
//     to normal editing mode.
//
// The status bar is updated to reflect the current selection state.
func (m *Model) toggleEditorSelectionAnchor() {
	if m.hasEditorSelectionAnchor() {
		m.clearEditorSelection()
		m.status = "Selection cleared"
		return
	}
	m.editorSelectionAnchor = m.currentEditorCursorOffset()
	m.editorSelectionActive = true
	applyEditorSelectionVisual(&m.editor, true)
	m.updateEditorSelectionStatus()
}

// handleEditorShiftSelectionMove intercepts Shift+Arrow and Shift+Home/End
// key events to extend the editor selection while moving the cursor.
//
// When a shifted movement key is detected:
//  1. If no selection anchor exists yet, one is automatically placed at the
//     current cursor position (so Shift+Arrow starts selecting immediately).
//  2. The corresponding un-shifted movement key is forwarded to the editor
//     widget to actually move the cursor.
//  3. The selection status is updated to show the character count.
//
// Returns true if the key was handled as a selection movement, false if it
// should be processed by the normal key handler chain instead.
func (m *Model) handleEditorShiftSelectionMove(keyMsg tea.KeyMsg) bool {
	msg, ok := selectionMovementKeyMsg(keyMsg)
	if !ok {
		return false
	}

	// Auto-set the anchor on the first shift-movement if none exists yet.
	// This lets users start selecting with Shift+Arrow without pressing
	// Alt+S first, matching the behavior of most text editors.
	if !m.hasEditorSelectionAnchor() {
		m.editorSelectionAnchor = m.currentEditorCursorOffset()
		m.editorSelectionActive = true
		applyEditorSelectionVisual(&m.editor, true)
	}

	// Forward the un-shifted equivalent to the editor widget so the cursor
	// actually moves. The selection range will be recalculated from the
	// (now-moved) cursor position relative to the fixed anchor.
	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	_ = cmd
	m.updateEditorSelectionStatus()
	return true
}

// selectionMovementKeyMsg maps a shifted key event to its un-shifted
// equivalent for forwarding to the editor widget.
//
// It handles both the typed key constants (tea.KeyShiftLeft, etc.) and the
// string representations ("shift+left", etc.) because Bubble Tea may report
// shifted keys differently depending on the terminal.
//
// Returns the un-shifted key message and true if the input was a recognized
// selection movement key, or an empty message and false otherwise.
func selectionMovementKeyMsg(keyMsg tea.KeyMsg) (tea.KeyMsg, bool) {
	switch keyMsg.Type {
	case tea.KeyShiftLeft:
		return tea.KeyMsg{Type: tea.KeyLeft}, true
	case tea.KeyShiftRight:
		return tea.KeyMsg{Type: tea.KeyRight}, true
	case tea.KeyShiftUp:
		return tea.KeyMsg{Type: tea.KeyUp}, true
	case tea.KeyShiftDown:
		return tea.KeyMsg{Type: tea.KeyDown}, true
	case tea.KeyShiftHome:
		return tea.KeyMsg{Type: tea.KeyHome}, true
	case tea.KeyShiftEnd:
		return tea.KeyMsg{Type: tea.KeyEnd}, true
	}

	switch keyMsg.String() {
	case "shift+left":
		return tea.KeyMsg{Type: tea.KeyLeft}, true
	case "shift+right":
		return tea.KeyMsg{Type: tea.KeyRight}, true
	case "shift+up":
		return tea.KeyMsg{Type: tea.KeyUp}, true
	case "shift+down":
		return tea.KeyMsg{Type: tea.KeyDown}, true
	case "shift+home":
		return tea.KeyMsg{Type: tea.KeyHome}, true
	case "shift+end":
		return tea.KeyMsg{Type: tea.KeyEnd}, true
	default:
		return tea.KeyMsg{}, false
	}
}

// updateEditorSelectionStatus updates the status bar to reflect the current
// selection state. If a valid range is selected, it shows the character count.
// If only an anchor is set (cursor hasn't moved yet), it shows guidance on
// how to extend or clear the selection.
func (m *Model) updateEditorSelectionStatus() {
	if start, end, ok := m.editorSelectionRange(); ok {
		m.status = fmt.Sprintf("Selected %d chars (Alt+S to clear)", end-start)
		return
	}
	if m.hasEditorSelectionAnchor() {
		m.status = "Selection anchor set (move cursor to select, Alt+S to clear)"
	}
}

// applyEditorFormat applies or removes markdown formatting around the current
// selection or the word under the cursor.
//
// The function uses a three-tier fallback strategy:
//
//  1. Active selection: If text is currently selected (anchor + cursor define
//     a range), the formatting is toggled on that range. If the selected text
//     is already wrapped by the given open/close markers, they are removed
//     (un-formatting). Otherwise, the markers are added around the selection.
//
//  2. Word under cursor: If no selection is active, the function finds the
//     word boundaries around the cursor position. The same toggle logic
//     applies — remove markers if present, add them if not.
//
//  3. Insert markers: If the cursor is not on a word (e.g. on whitespace or
//     at the end of a line), empty markers are inserted with the cursor
//     positioned between them, ready for the user to type.
//
// The open and close parameters are the markdown syntax markers (e.g. "**"
// for bold, "*" for italic, "~~" for strikethrough). The label parameter
// is a human-readable name used in the status bar message.
//
// After formatting, any active selection is cleared.
func (m *Model) applyEditorFormat(open, close, label string) {
	if start, end, ok := m.editorSelectionRange(); ok {
		removed := m.toggleEditorFormatRange(start, end, open, close)
		m.clearEditorSelection()
		if removed {
			m.status = "Removed " + label + " formatting from selection"
		} else {
			m.status = "Applied " + label + " formatting to selection"
		}
		return
	}

	cursor := m.currentEditorCursorOffset()
	if start, end, ok := wordBoundsAtCursor(m.editor.Value(), cursor); ok {
		removed := m.toggleEditorFormatRange(start, end, open, close)
		m.clearEditorSelection()
		if removed {
			m.status = "Removed " + label + " formatting from word"
		} else {
			m.status = "Applied " + label + " formatting to word"
		}
		return
	}

	m.insertEditorWrapper(open, close)
	m.clearEditorSelection()
	m.status = "Inserted " + label + " markers"
}

// insertMarkdownLinkTemplate inserts a markdown link at the current cursor
// position or wraps the current selection/word in link syntax.
//
// Behavior depends on context:
//   - Active selection: The selected text becomes the link text, wrapped as
//     [selected text](url), with the cursor placed on "url" for easy typing.
//   - Word under cursor: The word is wrapped the same way.
//   - No word/selection: A full template "[text](url)" is inserted with the
//     cursor on "url".
//
// In all cases, the cursor is repositioned inside the URL placeholder so the
// user can immediately type the destination URL.
func (m *Model) insertMarkdownLinkTemplate() {
	if start, end, ok := m.editorSelectionRange(); ok {
		m.wrapEditorRange(start, end, "[", "](url)")
		m.clearEditorSelection()
		m.setEditorCursorInsideURLPlaceholder()
		m.status = "Wrapped selection in markdown link"
		return
	}

	cursor := m.currentEditorCursorOffset()
	if start, end, ok := wordBoundsAtCursor(m.editor.Value(), cursor); ok {
		m.wrapEditorRange(start, end, "[", "](url)")
		m.clearEditorSelection()
		m.setEditorCursorInsideURLPlaceholder()
		m.status = "Wrapped word in markdown link"
		return
	}

	m.editor.InsertString("[text](url)")
	m.setEditorCursorInsideURLPlaceholder()
	m.status = "Inserted markdown link template"
}

// setEditorCursorInsideURLPlaceholder moves the cursor left by the length of
// "url)" so it lands inside the URL placeholder of a freshly inserted
// markdown link. This allows the user to immediately start typing the URL
// without needing to navigate manually.
func (m *Model) setEditorCursorInsideURLPlaceholder() {
	const placeholder = "url)"
	for i := 0; i < len(placeholder); i++ {
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(tea.KeyMsg{Type: tea.KeyLeft})
		_ = cmd
	}
}

// toggleHeading toggles a markdown heading prefix (e.g. "# ", "## ", "### ")
// on the current editor line.
//
// The level parameter specifies the heading depth (1-6). The function:
//
//  1. Finds the line containing the cursor.
//  2. Checks if it already has a heading prefix at the specified level.
//  3. If it does: removes the heading prefix (toggles off).
//  4. If it doesn't: adds the heading prefix. If a different heading level
//     is already present, it is replaced with the requested level.
//
// Leading whitespace (indentation) is preserved in all cases. The cursor
// position is adjusted to account for the added or removed characters so
// it stays in the same logical position within the line content.
//
// Levels outside the range 1-6 are silently ignored.
func (m *Model) toggleHeading(level int) {
	if level < 1 || level > 6 {
		return
	}
	value := m.editor.Value()
	runes := []rune(value)
	cursor := m.currentEditorCursorOffset()
	start, end := lineBoundsAtOffset(runes, cursor)
	line := string(runes[start:end])

	// Measure leading whitespace so we can preserve indentation.
	indentLen := 0
	for _, r := range []rune(line) {
		if r == ' ' || r == '\t' {
			indentLen++
			continue
		}
		break
	}
	rest := []rune(line)[indentLen:]
	existingLen := existingHeadingPrefixLen(rest)
	marker := strings.Repeat("#", level) + " "
	markerRunes := []rune(marker)

	var updatedLine []rune
	if existingLen > 0 && string(rest[:existingLen]) == marker {
		// Same heading level is already present — remove it (toggle off).
		updatedLine = append(updatedLine, []rune(line)[:indentLen]...)
		updatedLine = append(updatedLine, rest[existingLen:]...)
		m.status = fmt.Sprintf("Removed H%d heading", level)
	} else {
		// No heading or a different heading level — add/replace with the
		// requested level.
		updatedLine = append(updatedLine, []rune(line)[:indentLen]...)
		updatedLine = append(updatedLine, markerRunes...)
		if existingLen > 0 {
			updatedLine = append(updatedLine, rest[existingLen:]...)
		} else {
			updatedLine = append(updatedLine, rest...)
		}
		m.status = fmt.Sprintf("Applied H%d heading", level)
	}

	// Reconstruct the full editor content with the updated line.
	updated := make([]rune, 0, len(runes)-len([]rune(line))+len(updatedLine))
	updated = append(updated, runes[:start]...)
	updated = append(updated, updatedLine...)
	updated = append(updated, runes[end:]...)

	// Adjust cursor position by the number of characters added/removed
	// so the cursor stays at the same logical position within the line.
	delta := len(updatedLine) - len([]rune(line))
	if cursor > start {
		cursor += delta
	}
	m.setEditorValueAndCursorOffset(string(updated), cursor)
	m.clearEditorSelection()
}

// toggleEditorFormatRange checks whether the text at [start, end) is already
// wrapped by the given open/close markers. If so, it removes them (unwraps);
// otherwise it adds them (wraps).
//
// The check looks at the characters immediately before start and after end
// in the full editor text. This means:
//   - "**hello**" with start/end covering "hello" → removes the ** markers.
//   - "hello" with start/end covering "hello" → adds ** markers around it.
//
// Returns true if markers were removed (unwrapped), false if they were added.
//
// This toggle behavior allows the same keybinding to both apply and remove
// formatting, matching the behavior users expect from rich text editors.
func (m *Model) toggleEditorFormatRange(start, end int, open, close string) bool {
	value := m.editor.Value()
	runes := []rune(value)
	start = clamp(start, 0, len(runes))
	end = clamp(end, 0, len(runes))
	if start > end {
		start, end = end, start
	}

	openRunes := []rune(open)
	closeRunes := []rune(close)
	openLen := len(openRunes)
	closeLen := len(closeRunes)

	// Check if the exact open/close markers exist immediately surrounding
	// the selected range.
	openStart := start - openLen
	closeEnd := end + closeLen

	if openStart >= 0 &&
		closeEnd <= len(runes) &&
		runesEqual(runes[openStart:start], openRunes) &&
		runesEqual(runes[end:closeEnd], closeRunes) {
		// Markers found — remove them by reconstructing the text without
		// the surrounding marker characters.
		updated := make([]rune, 0, len(runes)-openLen-closeLen)
		updated = append(updated, runes[:openStart]...)
		updated = append(updated, runes[start:end]...)
		updated = append(updated, runes[closeEnd:]...)
		m.setEditorValueAndCursorOffset(string(updated), end-openLen)
		return true
	}

	// No existing markers — wrap the range with open/close.
	m.wrapEditorRange(start, end, open, close)
	return false
}

// wrapEditorRange inserts open and close strings around the specified rune
// range [start, end) in the editor content.
//
// After wrapping, the cursor is positioned immediately after the closing
// marker. The start and end offsets are clamped and normalized (start <= end)
// for safety.
//
// This is the low-level wrapping primitive used by toggleEditorFormatRange,
// insertMarkdownLinkTemplate, and similar operations.
func (m *Model) wrapEditorRange(start, end int, open, close string) {
	value := m.editor.Value()
	runes := []rune(value)
	start = clamp(start, 0, len(runes))
	end = clamp(end, 0, len(runes))
	if start > end {
		start, end = end, start
	}

	openRunes := []rune(open)
	closeRunes := []rune(close)

	updated := make([]rune, 0, len(runes)+len(openRunes)+len(closeRunes))
	updated = append(updated, runes[:start]...)
	updated = append(updated, openRunes...)
	updated = append(updated, runes[start:end]...)
	updated = append(updated, closeRunes...)
	updated = append(updated, runes[end:]...)

	cursor := end + len(openRunes) + len(closeRunes)
	m.setEditorValueAndCursorOffset(string(updated), cursor)
}

// splitEditorLines splits the editor's text value into logical lines,
// where each line is represented as a slice of runes.
//
// This function is used by currentEditorCursorOffset to convert (row, col)
// coordinates into a linear rune offset. It preserves empty lines (a
// trailing newline produces an empty final element).
func splitEditorLines(value string) [][]rune {
	lines := make([][]rune, 1)
	for _, r := range []rune(value) {
		if r == '\n' {
			lines = append(lines, nil)
			continue
		}
		last := len(lines) - 1
		lines[last] = append(lines[last], r)
	}
	return lines
}

// setEditorValueAndCursorOffset replaces the entire editor content and
// positions the cursor at the specified rune offset.
//
// Since the Bubble Tea textarea widget does not expose a direct
// "set cursor offset" API, this function uses a workaround:
//  1. Set the editor value (which places the cursor at the end).
//  2. Re-focus the editor.
//  3. Send (total - desired offset) left-arrow key events to walk the cursor
//     back to the target position.
//
// The cursor offset is clamped to [0, total rune count] for safety.
//
// This approach is O(n) in the distance from the end, but is acceptable
// because it only runs on explicit user actions (formatting, heading toggle,
// etc.) rather than on every keystroke.
func (m *Model) setEditorValueAndCursorOffset(value string, cursorOffset int) {
	total := utf8.RuneCountInString(value)
	cursorOffset = clamp(cursorOffset, 0, total)

	m.editor.SetValue(value)
	m.editor.Focus()

	movesLeft := total - cursorOffset
	for i := 0; i < movesLeft; i++ {
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(tea.KeyMsg{Type: tea.KeyLeft})
		_ = cmd
	}
}

// wordBoundsAtCursor finds the start and end rune offsets of the word
// surrounding the given cursor position.
//
// A "word" is defined as a contiguous sequence of letters, digits, or
// underscores (see isWordRune). The function handles two cases:
//   - Cursor is directly on a word character → expand in both directions.
//   - Cursor is immediately after a word character (e.g. at the end of a
//     word) → expand from the preceding character.
//
// Returns (start, end, true) if a word was found, or (0, 0, false) if the
// cursor is not on or adjacent to any word characters.
//
// This is used by formatting commands to determine the "target word" when
// no explicit selection is active.
func wordBoundsAtCursor(value string, cursor int) (start, end int, ok bool) {
	runes := []rune(value)
	if len(runes) == 0 {
		return 0, 0, false
	}

	cursor = clamp(cursor, 0, len(runes))
	idx := cursor
	if idx < len(runes) && isWordRune(runes[idx]) {
		// Cursor is directly on a word rune — use this as the starting point.
	} else if idx > 0 && isWordRune(runes[idx-1]) {
		// Cursor is immediately after a word rune (e.g. end of word).
		idx--
	} else {
		// Cursor is not on or adjacent to any word — no target word found.
		return 0, 0, false
	}

	// Expand leftward to find the start of the word.
	start = idx
	for start > 0 && isWordRune(runes[start-1]) {
		start--
	}

	// Expand rightward to find the end of the word.
	end = idx + 1
	for end < len(runes) && isWordRune(runes[end]) {
		end++
	}
	return start, end, start < end
}

// lineBoundsAtOffset returns the start and end rune offsets of the line
// containing the given offset position.
//
// The "line" is defined as the span of runes between newline characters
// (or the start/end of the text). The returned range [start, end) does NOT
// include the newline characters themselves.
//
// This is used by toggleHeading to isolate the current line for heading
// prefix manipulation.
func lineBoundsAtOffset(runes []rune, offset int) (start, end int) {
	offset = clamp(offset, 0, len(runes))

	// Scan backward to find the start of the line (character after the
	// preceding newline, or position 0).
	start = offset
	for start > 0 && runes[start-1] != '\n' {
		start--
	}

	// Scan forward to find the end of the line (character before the next
	// newline, or the end of the text).
	end = offset
	for end < len(runes) && runes[end] != '\n' {
		end++
	}
	return start, end
}

// existingHeadingPrefixLen returns the length (in runes) of an existing
// markdown heading prefix at the start of the given line content.
//
// A valid heading prefix is 1-6 '#' characters followed by a space.
// For example:
//   - "# Title"   → returns 2 (the "# " prefix)
//   - "### Title" → returns 4 (the "### " prefix)
//   - "No heading" → returns 0
//   - "#NoSpace"   → returns 0 (no space after #)
//
// The returned length includes the trailing space, so it can be used
// directly to slice off the heading prefix from the line.
func existingHeadingPrefixLen(line []rune) int {
	if len(line) == 0 {
		return 0
	}
	i := 0
	for i < len(line) && i < 6 && line[i] == '#' {
		i++
	}
	if i == 0 || i >= len(line) || line[i] != ' ' {
		return 0
	}
	return i + 1
}

// isWordRune reports whether the given rune is considered part of a "word"
// for the purposes of word-boundary detection.
//
// Word characters are: Unicode letters, Unicode digits, and underscores.
// This matches the common definition used by text editors for double-click
// word selection and Ctrl+Arrow word movement.
func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// runesEqual reports whether two rune slices are identical in length and
// content. It is used by toggleEditorFormatRange to check for existing
// formatting markers adjacent to the selection boundaries.
func runesEqual(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
