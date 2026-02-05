package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// truncate fits a string to the given terminal width.
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	return ansi.Truncate(s, width, "")
}

// padBlock normalizes content to a fixed width and height so old UI text is cleared.
func padBlock(content string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	lines := strings.Split(content, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}

	for i, line := range lines {
		line = truncate(line, width)
		visible := lipgloss.Width(line)
		if visible < width {
			line += strings.Repeat(" ", width-visible)
		}
		lines[i] = line
	}

	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	return strings.Join(lines, "\n")
}

// clamp bounds a value between minVal and maxVal.
func clamp(value, minVal, maxVal int) int {
	if value < minVal {
		return minVal
	}
	if value > maxVal {
		return maxVal
	}
	return value
}

// min returns the smaller of two ints.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the larger of two ints.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// renderWidthBucket buckets widths so the cache is more reusable.
func renderWidthBucket(width int) int {
	if width <= 0 {
		return 80
	}
	if width < 20 {
		return width
	}
	return (width / 20) * 20
}

func stringsHasSuffixFold(value, suffix string) bool {
	return strings.HasSuffix(strings.ToLower(value), strings.ToLower(suffix))
}
