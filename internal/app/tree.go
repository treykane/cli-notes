package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type sortMode string

const (
	sortModeName     sortMode = "name"
	sortModeModified sortMode = "modified"
	sortModeSize     sortMode = "size"
	sortModeCreated  sortMode = "created"
)

func parseSortMode(value string) sortMode {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(sortModeModified):
		return sortModeModified
	case string(sortModeSize):
		return sortModeSize
	case string(sortModeCreated):
		return sortModeCreated
	default:
		return sortModeName
	}
}

func (s sortMode) String() string {
	return string(s)
}

func (s sortMode) Label() string {
	switch s {
	case sortModeModified:
		return "modified"
	case sortModeSize:
		return "size"
	case sortModeCreated:
		return "created"
	default:
		return "name"
	}
}

func nextSortMode(current sortMode) sortMode {
	switch current {
	case sortModeName:
		return sortModeModified
	case sortModeModified:
		return sortModeSize
	case sortModeSize:
		return sortModeCreated
	default:
		return sortModeName
	}
}

// moveCursor changes the selection and keeps it within bounds.
func (m *Model) moveCursor(delta int) {
	if len(m.items) == 0 {
		return
	}

	m.cursor = clamp(m.cursor+delta, 0, len(m.items)-1)
	m.adjustTreeOffset()
}

// adjustTreeOffset scrolls the tree so the cursor remains visible.
func (m *Model) adjustTreeOffset() {
	visibleHeight := max(0, m.leftHeight-2-1)
	if visibleHeight == 0 {
		m.treeOffset = 0
		return
	}

	if m.cursor < m.treeOffset {
		m.treeOffset = m.cursor
	}
	if m.cursor >= m.treeOffset+visibleHeight {
		m.treeOffset = m.cursor - visibleHeight + 1
	}
}

// toggleExpand expands or collapses a directory row.
func (m *Model) toggleExpand(expandIfDir bool) {
	item := m.selectedItem()
	if item == nil || !item.isDir {
		return
	}

	if expandIfDir {
		m.expanded[item.path] = !m.expanded[item.path]
	} else {
		if item.path == m.notesDir {
			return
		}
		m.expanded[item.path] = false
	}

	m.rebuildTreeKeep(item.path)
}

// refreshTree rebuilds the tree while preserving selection.
func (m *Model) refreshTree() {
	selected := m.selectedPath()
	m.rebuildTreeKeep(selected)
	m.adjustTreeOffset()
}

// rebuildTreeKeep rebuilds the tree and keeps the cursor near the given path.
func (m *Model) rebuildTreeKeep(path string) {
	m.items = buildTree(m.notesDir, m.expanded, m.sortMode, m.pinnedPaths)
	if len(m.items) == 0 {
		m.cursor = 0
		m.treeOffset = 0
		return
	}
	m.cursor = 0
	for i, item := range m.items {
		if item.path == path {
			m.cursor = i
			break
		}
	}
	m.adjustTreeOffset()
}

// buildTree builds a flat list of items for rendering the tree view.
//
// The tree is built by recursively walking the directory structure, respecting
// the expanded map to determine which folders to traverse. The result is a flat
// slice of treeItems that can be rendered with proper indentation.
//
// Algorithm:
//  1. Start at root with depth 0
//  2. For each directory entry:
//     - Add it to the items list
//     - If it's a directory AND expanded, recursively walk its children
//  3. Sort each level: directories first, then alphabetically within each group
//
// This produces a depth-first traversal that matches typical file browser UIs.
func buildTree(root string, expanded map[string]bool, mode sortMode, pinned map[string]bool) []treeItem {
	items := []treeItem{}
	walkTree(root, 0, expanded, mode, pinned, &items)
	return items
}

// walkTree recursively appends directory contents in sorted order.
//
// Each directory is sorted with folders first, then alphabetically (case-insensitive).
// Only expanded folders have their children added to the tree.
func walkTree(dir string, depth int, expanded map[string]bool, mode sortMode, pinned map[string]bool, items *[]treeItem) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		appLog.Warn("read tree directory", "path", dir, "error", err)
		return
	}

	type sortableEntry struct {
		entry   os.DirEntry
		path    string
		info    os.FileInfo
		modTime time.Time
		size    int64
		created time.Time
	}

	sortable := make([]sortableEntry, 0, len(entries))
	for _, entry := range entries {
		if shouldSkipManagedPath(entry.Name()) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		info, statErr := entry.Info()
		if statErr != nil {
			appLog.Warn("stat tree entry", "path", path, "error", statErr)
			continue
		}
		created := resolveCreatedAt(info)
		sortable = append(sortable, sortableEntry{
			entry:   entry,
			path:    path,
			info:    info,
			modTime: info.ModTime(),
			size:    info.Size(),
			created: created,
		})
	}

	sort.Slice(sortable, func(i, j int) bool {
		left := sortable[i]
		right := sortable[j]
		leftPinned := pinned[left.path]
		rightPinned := pinned[right.path]
		if leftPinned != rightPinned {
			return leftPinned
		}
		if left.entry.IsDir() != right.entry.IsDir() {
			return left.entry.IsDir()
		}

		switch mode {
		case sortModeModified:
			if !left.modTime.Equal(right.modTime) {
				return left.modTime.After(right.modTime)
			}
		case sortModeSize:
			if left.size != right.size {
				return left.size > right.size
			}
		case sortModeCreated:
			if !left.created.Equal(right.created) {
				return left.created.After(right.created)
			}
		}

		return strings.ToLower(left.entry.Name()) < strings.ToLower(right.entry.Name())
	})

	for _, entry := range sortable {
		path := entry.path
		item := treeItem{
			path:   path,
			name:   entry.entry.Name(),
			depth:  depth,
			isDir:  entry.entry.IsDir(),
			pinned: pinned[path],
		}
		*items = append(*items, item)
		if entry.entry.IsDir() && expanded[path] {
			walkTree(path, depth+1, expanded, mode, pinned, items)
		}
	}
}

func searchTreeItems(root, query string) []treeItem {
	if strings.TrimSpace(query) == "" {
		return nil
	}
	idx := newSearchIndex(root)
	if err := idx.ensureBuilt(); err != nil {
		appLog.Error("build search tree index", "root", root, "error", err)
		return nil
	}
	return idx.search(query)
}
