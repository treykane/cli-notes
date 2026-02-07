package app

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// noteMetrics holds computed statistics about a note's text content.
//
// These metrics are displayed in the footer status bar (in both preview and
// edit modes) to give the user a quick overview of the note's size. The
// metrics are recomputed on every View call from the current note content,
// so they always reflect the latest state — including unsaved edits in
// progress.
type noteMetrics struct {
	// words is the number of whitespace-separated tokens in the content,
	// as determined by strings.Fields (which splits on any Unicode
	// whitespace). This provides a reasonable word count for prose-heavy
	// markdown notes.
	words int

	// chars is the number of Unicode code points (runes) in the content,
	// NOT the number of bytes. This gives a more intuitive character count
	// for content that may include non-ASCII characters (emoji, accented
	// letters, CJK characters, etc.).
	chars int

	// lines is the number of visual lines in the content. A trailing
	// newline does NOT add an extra line (i.e. "hello\n" is 1 line, not 2),
	// matching the behavior most text editors display in their status bars.
	lines int
}

// currentNoteTextForMetrics returns the raw text content that should be used
// for computing note metrics.
//
// The source depends on the current mode:
//   - In edit mode (modeEditNote): returns the live editor buffer, so the
//     displayed metrics update in real time as the user types.
//   - In all other modes: returns the last-loaded file content stored in
//     currentNoteContent, which reflects the on-disk state.
//
// This same function is also used by copyCurrentNoteContentToClipboard to
// determine what content to copy, ensuring consistency between the displayed
// metrics and the copied text.
func (m *Model) currentNoteTextForMetrics() string {
	if m.mode == modeEditNote {
		return m.editor.Value()
	}
	return m.currentNoteContent
}

// computeNoteMetrics calculates word count, character count, and line count
// for the given content string.
//
// The function handles edge cases:
//   - Empty content returns all-zero metrics.
//   - Content without a trailing newline still counts the last line (e.g.
//     "hello" → 1 line, "hello\nworld" → 2 lines).
//   - Content with a trailing newline does not count an extra empty line
//     (e.g. "hello\n" → 1 line), matching typical editor status bar behavior.
//
// Word counting uses strings.Fields, which splits on any Unicode whitespace
// and automatically handles multiple consecutive spaces, tabs, and newlines.
//
// Character counting uses utf8.RuneCountInString rather than len() to count
// Unicode code points instead of bytes, giving a more intuitive count for
// international text.
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

// noteMetricsSummary produces a compact summary string of the current note's
// metrics for display in the footer status bar.
//
// The output format is: "W:<words> C:<chars> L:<lines>"
//
// For example, a note with 150 words, 823 characters, and 42 lines would
// produce: "W:150 C:823 L:42"
//
// Returns an empty string if the note content is empty or whitespace-only,
// which causes the footer rendering to omit the metrics section entirely
// (avoiding a distracting "W:0 C:0 L:0" display when no note is loaded).
func (m *Model) noteMetricsSummary() string {
	content := m.currentNoteTextForMetrics()
	if strings.TrimSpace(content) == "" {
		return ""
	}
	metrics := computeNoteMetrics(content)
	return fmt.Sprintf("W:%d C:%d L:%d", metrics.words, metrics.chars, metrics.lines)
}
