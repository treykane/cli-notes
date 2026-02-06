package app

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type noteMetrics struct {
	words int
	chars int
	lines int
}

func (m *Model) currentNoteTextForMetrics() string {
	if m.mode == modeEditNote {
		return m.editor.Value()
	}
	return m.currentNoteContent
}

func computeNoteMetrics(content string) noteMetrics {
	if content == "" {
		return noteMetrics{}
	}
	lines := strings.Count(content, "\n")
	if !strings.HasSuffix(content, "\n") {
		lines++
	}
	return noteMetrics{
		words: len(strings.Fields(content)),
		chars: utf8.RuneCountInString(content),
		lines: lines,
	}
}

func (m *Model) noteMetricsSummary() string {
	content := m.currentNoteTextForMetrics()
	if strings.TrimSpace(content) == "" {
		return ""
	}
	metrics := computeNoteMetrics(content)
	return fmt.Sprintf("W:%d C:%d L:%d", metrics.words, metrics.chars, metrics.lines)
}
