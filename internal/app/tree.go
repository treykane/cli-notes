package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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
	m.items = buildTree(m.notesDir, m.expanded)
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
func buildTree(root string, expanded map[string]bool) []treeItem {
	items := []treeItem{}
	walkTree(root, 0, expanded, &items)
	return items
}

// walkTree recursively appends directory contents in sorted order.
func walkTree(dir string, depth int, expanded map[string]bool, items *[]treeItem) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		item := treeItem{
			path:  path,
			name:  entry.Name(),
			depth: depth,
			isDir: entry.IsDir(),
		}
		*items = append(*items, item)
		if entry.IsDir() && expanded[path] {
			walkTree(path, depth+1, expanded, items)
		}
	}
}

func searchTreeItems(root, query string) []treeItem {
	if strings.TrimSpace(query) == "" {
		return nil
	}
	idx := newSearchIndex(root)
	if err := idx.ensureBuilt(); err != nil {
		return nil
	}
	return idx.search(query)
}
