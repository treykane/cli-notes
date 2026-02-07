// util.go provides small, general-purpose helper functions used throughout the
// app package. These include ANSI-aware string truncation, fixed-size block
// padding for the TUI layout, numeric clamping, render-width bucketing, and
// filesystem helpers for managed-path detection and file creation-time
// resolution.
//
// None of the helpers in this file hold or mutate Model state — they are pure
// functions (or thin wrappers around standard library calls) that can be called
// from any context without side effects.
package app

import (
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// managedNotesDirName is the name of the hidden directory inside each workspace
// root that the app uses for internal bookkeeping (state.json, drafts, etc.).
// This directory is excluded from tree rendering, search indexing, and
// filesystem watching so its contents never appear as user-visible notes.
const managedNotesDirName = ".cli-notes"

// truncate fits a string to the given terminal width, accounting for ANSI
// escape sequences that take up zero visible columns. If the string already
// fits, it is returned unchanged. This is used extensively in the View layer
// to prevent long file paths or headings from overflowing pane boundaries.
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	return ansi.Truncate(s, width, "")
}

// padBlock normalizes content to exactly the specified width and height by
// truncating long lines, right-padding short lines with spaces, and appending
// blank lines at the bottom. This ensures that every frame fully overwrites
// the previous one in the terminal — without padding, leftover characters from
// a taller or wider previous frame would remain visible as visual artifacts.
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

// roundWidthToNearestBucket quantizes terminal widths to multiples of
// RenderWidthBucket (20 columns). This reduces the number of distinct cache
// entries the render cache needs to maintain — a 1-column resize does not
// invalidate the cached render, which avoids expensive re-renders during
// incremental window resizing. Widths below one bucket are left as-is to
// avoid rounding very narrow terminals down to zero.
func roundWidthToNearestBucket(width int) int {
	if width <= 0 {
		return 80
	}
	if width < RenderWidthBucket {
		return width
	}
	return (width / RenderWidthBucket) * RenderWidthBucket
}

// hasSuffixCaseInsensitive checks whether value ends with suffix, ignoring
// case differences. Used primarily to identify markdown files (".md") regardless
// of whether the user named them with uppercase or mixed-case extensions.
func hasSuffixCaseInsensitive(value, suffix string) bool {
	return strings.HasSuffix(strings.ToLower(value), strings.ToLower(suffix))
}

// shouldSkipManagedPath reports whether the given directory entry name is the
// internal managed directory (.cli-notes). This is checked during tree walks,
// search indexing, and filesystem watching to exclude app-internal files from
// user-visible listings.
func shouldSkipManagedPath(name string) bool {
	return strings.EqualFold(name, managedNotesDirName)
}

// resolveCreatedAt returns the best available creation timestamp for a file.
// On macOS, the true birth time (Birthtimespec) is used. On other Unix systems,
// the metadata-change time (Ctim) is used as a proxy (see file_time_*.go).
// If the platform does not support creation time at all, the modification time
// is returned as a fallback.
func resolveCreatedAt(info os.FileInfo) time.Time {
	if t, ok := fileCreationTime(info); ok {
		return t
	}
	return info.ModTime()
}
