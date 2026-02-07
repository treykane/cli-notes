package app

import (
	"time"
)

const typingBurstIdleWindow = 750 * time.Millisecond

// editorSnapshot captures the editor's text and cursor in rune-offset form.
type editorSnapshot struct {
	value        string
	cursorOffset int
}

func (m *Model) captureEditorSnapshot() editorSnapshot {
	return editorSnapshot{
		value:        m.editor.Value(),
		cursorOffset: m.currentEditorCursorOffset(),
	}
}

func (m *Model) restoreEditorSnapshot(snapshot editorSnapshot) {
	m.setEditorValueAndCursorOffset(snapshot.value, snapshot.cursorOffset)
	m.clearEditorSelection()
}

func (m *Model) resetEditHistory() {
	m.editorUndo = nil
	m.editorRedo = nil
	m.typingBurstActive = false
	m.typingBurstLastInputAt = time.Time{}
}

func (m *Model) pushUndo(snapshot editorSnapshot) {
	m.editorUndo = append(m.editorUndo, snapshot)
	// Any forward mutation invalidates the redo chain.
	m.editorRedo = nil
}

func (m *Model) finalizeTypingBurstBoundary() {
	m.typingBurstActive = false
	m.typingBurstLastInputAt = time.Time{}
}

func (m *Model) recordDiscreteEditMutation(before, after editorSnapshot) {
	if before.value == after.value && before.cursorOffset == after.cursorOffset {
		return
	}
	m.finalizeTypingBurstBoundary()
	m.pushUndo(before)
}

func (m *Model) recordTypingMutation(before, after editorSnapshot, now time.Time) {
	if before.value == after.value && before.cursorOffset == after.cursorOffset {
		return
	}
	if !m.typingBurstActive || now.Sub(m.typingBurstLastInputAt) > typingBurstIdleWindow {
		m.pushUndo(before)
	}
	m.typingBurstActive = true
	m.typingBurstLastInputAt = now
}

func (m *Model) undoEditorChange() {
	m.finalizeTypingBurstBoundary()
	if len(m.editorUndo) == 0 {
		m.status = "Nothing to undo"
		return
	}
	current := m.captureEditorSnapshot()
	last := m.editorUndo[len(m.editorUndo)-1]
	m.editorUndo = m.editorUndo[:len(m.editorUndo)-1]
	m.editorRedo = append(m.editorRedo, current)
	m.restoreEditorSnapshot(last)
	m.status = "Undid edit"
}

func (m *Model) redoEditorChange() {
	m.finalizeTypingBurstBoundary()
	if len(m.editorRedo) == 0 {
		m.status = "Nothing to redo"
		return
	}
	current := m.captureEditorSnapshot()
	next := m.editorRedo[len(m.editorRedo)-1]
	m.editorRedo = m.editorRedo[:len(m.editorRedo)-1]
	m.editorUndo = append(m.editorUndo, current)
	m.restoreEditorSnapshot(next)
	m.status = "Redid edit"
}
