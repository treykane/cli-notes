package app

import (
	"fmt"

	"github.com/atotto/clipboard"
)

// copyCurrentNoteContentToClipboard copies the raw text content of the
// currently displayed note to the system clipboard.
//
// In browse/preview mode the last-loaded file content is used; in edit mode
// the live editor buffer is used instead (via currentNoteTextForMetrics).
//
// The status bar is updated with a success message showing the character
// count, or an error message if the clipboard write fails or no content
// is available.
func (m *Model) copyCurrentNoteContentToClipboard() {
	content := m.currentNoteTextForMetrics()
	if content == "" {
		m.status = "No note content to copy"
		return
	}
	if err := clipboard.WriteAll(content); err != nil {
		m.setStatusError("Clipboard copy failed", err)
		return
	}
	m.status = fmt.Sprintf("Copied note content (%d chars)", len([]rune(content)))
}

// copyCurrentNotePathToClipboard copies the absolute filesystem path of the
// currently selected note to the system clipboard.
//
// This is useful for referencing the note file in external tools (e.g.
// opening it in another editor, passing it to a script, etc.).
//
// The status bar is updated with a confirmation or an error if the clipboard
// write fails or no note is currently selected.
func (m *Model) copyCurrentNotePathToClipboard() {
	if m.currentFile == "" {
		m.status = "No note selected"
		return
	}
	if err := clipboard.WriteAll(m.currentFile); err != nil {
		m.setStatusError("Clipboard copy failed", err)
		return
	}
	m.status = "Copied note path"
}

// pasteFromClipboardIntoEditor reads text from the system clipboard and
// inserts it at the current cursor position in the editor textarea.
//
// This function is only active in edit mode (modeEditNote). It clears any
// active editor selection after pasting so the cursor moves to the end of
// the inserted text. If the clipboard is empty or unreadable, the status
// bar is updated with an appropriate message.
func (m *Model) pasteFromClipboardIntoEditor() {
	if m.mode != modeEditNote {
		return
	}
	value, err := clipboard.ReadAll()
	if err != nil {
		m.setStatusError("Clipboard paste failed", err)
		return
	}
	if value == "" {
		m.status = "Clipboard is empty"
		return
	}
	m.editor.InsertString(value)
	m.clearEditorSelection()
	m.status = "Pasted from clipboard"
}
