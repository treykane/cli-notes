// watcher.go implements poll-based filesystem monitoring for external changes.
//
// Because the application cannot rely on OS-level filesystem event APIs across
// all platforms (and because the notes directory may live on a network mount),
// we use a simple polling strategy:
//
//  1. Every configured poll interval (default: 2 s), walk the notes directory and capture a
//     snapshot of every file/directory: path, modification time (nanoseconds),
//     size, and whether it is a directory.
//  2. Compare the new snapshot to the previous one. If anything differs
//     (file added, removed, modified, or resized), trigger a full refresh:
//     rebuild the tree, invalidate the search index, and clear the render
//     cache.
//  3. If the user is currently editing a note, skip re-rendering that note
//     to avoid clobbering unsaved changes.
//
// The internal `.cli-notes` managed directory is excluded from the snapshot so
// draft files, state, and other metadata do not trigger spurious refreshes.
package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// fileWatchTickMsg is the Bubble Tea message emitted by the periodic poll timer.
// Receiving this message triggers a new filesystem snapshot comparison.
type fileWatchTickMsg struct{}

// fileWatchEntry records the observable attributes of a single filesystem entry
// that we track for change detection. We intentionally use UnixNano for the
// modification time rather than time.Time to make equality comparison trivial
// and avoid timezone/location issues.
type fileWatchEntry struct {
	Path    string
	ModNano int64
	Size    int64
	IsDir   bool
}

// fileWatchSnapshot is a map from absolute path to its observed attributes.
// Two snapshots are compared entry-by-entry to detect external changes.
type fileWatchSnapshot map[string]fileWatchEntry

// scheduleFileWatchTick queues the next poll after the configured interval.
// The returned Cmd emits a fileWatchTickMsg when the timer fires.
func (m *Model) scheduleFileWatchTick() tea.Cmd {
	return tea.Tick(m.effectiveFileWatchInterval(), func(time.Time) tea.Msg {
		return fileWatchTickMsg{}
	})
}

func (m *Model) effectiveFileWatchInterval() time.Duration {
	if m.fileWatchInterval <= 0 {
		return DefaultFileWatchInterval
	}
	return m.fileWatchInterval
}

// handleFileWatchTick runs on every poll tick. It captures a fresh filesystem
// snapshot and compares it to the last known state. On the very first tick the
// snapshot is simply stored as the baseline. On subsequent ticks a diff is
// performed; if any entry changed the application refreshes its caches.
//
// Regardless of outcome, the next poll tick is always scheduled so monitoring
// continues for the lifetime of the application.
func (m *Model) handleFileWatchTick(_ fileWatchTickMsg) (tea.Model, tea.Cmd) {
	snapshot, err := scanFileWatchSnapshot(m.notesDir)
	if err != nil {
		appLog.Warn("scan filesystem watcher", "root", m.notesDir, "error", err)
		return m, m.scheduleFileWatchTick()
	}

	// First tick — establish baseline snapshot without triggering a refresh.
	if len(m.fileWatchSnapshot) == 0 {
		m.fileWatchSnapshot = snapshot
		return m, m.scheduleFileWatchTick()
	}

	if !fileWatchSnapshotsEqual(m.fileWatchSnapshot, snapshot) {
		m.fileWatchSnapshot = snapshot
		cmd := m.handleExternalFilesystemChange()
		return m, tea.Batch(cmd, m.scheduleFileWatchTick())
	}
	return m, m.scheduleFileWatchTick()
}

// scanFileWatchSnapshot walks the entire notes directory tree and returns a
// snapshot mapping every visible path to its current attributes.
func scanFileWatchSnapshot(root string) (fileWatchSnapshot, error) {
	snapshot := make(fileWatchSnapshot)
	entries, err := walkFileWatchEntries(root)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		snapshot[entry.Path] = entry
	}
	return snapshot, nil
}

// walkFileWatchEntries recursively walks root and collects a fileWatchEntry for
// every file and directory, excluding the managed `.cli-notes` subtree. The
// returned slice is sorted by path to make downstream comparison deterministic.
func walkFileWatchEntries(root string) ([]fileWatchEntry, error) {
	entries := make([]fileWatchEntry, 0, 128)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		if shouldSkipManagedPath(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		entries = append(entries, fileWatchEntry{
			Path:    path,
			ModNano: info.ModTime().UnixNano(),
			Size:    info.Size(),
			IsDir:   d.IsDir(),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk notes dir %q: %w", root, err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})
	return entries, nil
}

// fileWatchSnapshotsEqual returns true when two snapshots contain exactly the
// same set of paths with identical modification times, sizes, and directory
// flags. Any difference — an added, removed, or modified entry — returns false.
func fileWatchSnapshotsEqual(left, right fileWatchSnapshot) bool {
	if len(left) != len(right) {
		return false
	}
	for path, leftEntry := range left {
		rightEntry, ok := right[path]
		if !ok {
			return false
		}
		if leftEntry.ModNano != rightEntry.ModNano || leftEntry.Size != rightEntry.Size || leftEntry.IsDir != rightEntry.IsDir {
			return false
		}
	}
	return true
}

// handleExternalFilesystemChange is called when the watcher detects that the
// on-disk state has diverged from the last snapshot. It performs a full refresh:
//
//  1. Persists the current note position and app state (so nothing is lost).
//  2. Rebuilds the tree from the filesystem.
//  3. Invalidates the search index (forces a lazy rebuild on next query).
//  4. Clears the render cache (forces re-render on next view).
//  5. Re-renders the currently viewed file — unless the user is in edit mode,
//     in which case we leave the editor buffer untouched to avoid clobbering
//     unsaved changes.
//  6. If the currently viewed file was deleted externally, clears the viewport.
func (m *Model) handleExternalFilesystemChange() tea.Cmd {
	m.rememberCurrentNotePosition()
	_ = m.applyMutationEffects(mutationEffects{
		saveState:        true,
		refreshTree:      true,
		invalidateSearch: true,
		clearRenderCache: true,
	})
	m.invalidateTreeMetadataCache()
	m.rebuildRecentEntries()

	if m.currentFile != "" {
		if _, err := os.Stat(m.currentFile); err == nil {
			if m.mode != modeEditNote {
				m.status = "Auto-refreshed (external filesystem changes detected)"
				return m.setCurrentFile(m.currentFile)
			}
		} else {
			m.currentFile = ""
			m.currentNoteContent = ""
			m.viewport.SetContent("Select a note to view")
		}
	}
	m.status = "Auto-refreshed (external filesystem changes detected)"
	return nil
}
