package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type fileWatchTickMsg struct{}

type fileWatchEntry struct {
	Path    string
	ModNano int64
	Size    int64
	IsDir   bool
}

type fileWatchSnapshot map[string]fileWatchEntry

func (m *Model) scheduleFileWatchTick() tea.Cmd {
	return tea.Tick(FileWatchInterval, func(time.Time) tea.Msg {
		return fileWatchTickMsg{}
	})
}

func (m *Model) handleFileWatchTick(_ fileWatchTickMsg) (tea.Model, tea.Cmd) {
	snapshot, err := scanFileWatchSnapshot(m.notesDir)
	if err != nil {
		appLog.Warn("scan filesystem watcher", "root", m.notesDir, "error", err)
		return m, m.scheduleFileWatchTick()
	}

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

func (m *Model) handleExternalFilesystemChange() tea.Cmd {
	m.rememberCurrentNotePosition()
	m.saveAppState()
	m.refreshTree()
	if m.searchIndex != nil {
		m.searchIndex.invalidate()
	}
	m.renderCache = map[string]renderCacheEntry{}
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
