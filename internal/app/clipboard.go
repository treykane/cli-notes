package app

import (
	"fmt"

	"github.com/atotto/clipboard"
)

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
