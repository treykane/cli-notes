package app

import "strings"

func highlightFencedCodeInEditorView(view string) string {
	if strings.TrimSpace(view) == "" {
		return view
	}
	lines := strings.Split(view, "\n")
	inFence := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "```") {
			lines[i] = editorFenceLine.Render(line)
			inFence = !inFence
			continue
		}
		if inFence {
			lines[i] = editorCodeLine.Render(line)
		}
	}
	return strings.Join(lines, "\n")
}
