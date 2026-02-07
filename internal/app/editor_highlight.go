package app

import "strings"

// highlightFencedCodeInEditorView applies visual styling to fenced code blocks
// within the editor's rendered view string.
//
// It scans the editor output line by line, toggling an "in fence" state each
// time it encounters a line containing triple backticks (```). Lines that
// serve as fence delimiters are rendered with the editorFenceLine style
// (a warm accent color), while lines inside a fenced block are rendered
// with the editorCodeLine style (a cool monospace-friendly color).
//
// Lines outside of any fenced block are left untouched, preserving the
// editor's default styling for normal prose.
//
// This function operates on the final rendered view string (after the
// textarea widget has produced its output), so it does not interfere with
// the editor's internal state or cursor positioning. It is called during
// the View phase only.
//
// If the input view is empty or whitespace-only, it is returned as-is.
func highlightFencedCodeInEditorView(view string) string {
	if strings.TrimSpace(view) == "" {
		return view
	}
	lines := strings.Split(view, "\n")
	inFence := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Toggle fence state when we encounter a triple-backtick delimiter.
		// The delimiter line itself is styled as a fence marker regardless
		// of whether it opens or closes the block.
		if strings.Contains(trimmed, "```") {
			lines[i] = editorFenceLine.Render(line)
			inFence = !inFence
			continue
		}
		// Lines inside a fenced code block get code-style highlighting
		// to visually distinguish them from surrounding prose.
		if inFence {
			lines[i] = editorCodeLine.Render(line)
		}
	}
	return strings.Join(lines, "\n")
}
