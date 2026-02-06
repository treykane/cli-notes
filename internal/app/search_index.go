package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxSearchFileBytes int64 = MaxSearchFileBytes

type searchDoc struct {
	item         treeItem
	nameLower    string
	contentLower string
}

type searchIndex struct {
	root  string
	docs  map[string]searchDoc
	ready bool
}

func newSearchIndex(root string) *searchIndex {
	return &searchIndex{
		root: root,
		docs: map[string]searchDoc{},
	}
}

func (i *searchIndex) invalidate() {
	i.ready = false
}

func (i *searchIndex) ensureBuilt() error {
	if i.ready {
		return nil
	}
	return i.build()
}

func (i *searchIndex) build() error {
	i.docs = map[string]searchDoc{}
	if err := i.walk(i.root, 0); err != nil {
		i.ready = false
		return err
	}
	i.ready = true
	return nil
}

func (i *searchIndex) walk(dir string, depth int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read search dir %q: %w", dir, err)
	}
	sort.Slice(entries, func(a, b int) bool {
		if entries[a].IsDir() != entries[b].IsDir() {
			return entries[a].IsDir()
		}
		return strings.ToLower(entries[a].Name()) < strings.ToLower(entries[b].Name())
	})

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		i.indexPath(path, entry.Name(), depth, entry.IsDir())
		if entry.IsDir() {
			if err := i.walk(path, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

// search returns all items matching the query string.
//
// Search algorithm:
//  1. Convert query to lowercase for case-insensitive matching
//  2. For each indexed document:
//     - Match against filename (always)
//     - Match against content (only for files, not directories)
//  3. Sort results: directories first, then alphabetically
//
// This provides a simple but effective full-text search across all notes.
// Files larger than MaxSearchFileBytes are excluded from content search
// (but their names are still searchable).
func (i *searchIndex) search(query string) []treeItem {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return nil
	}

	results := make([]treeItem, 0, 32)
	for _, doc := range i.docs {
		if strings.Contains(doc.nameLower, query) || (!doc.item.isDir && strings.Contains(doc.contentLower, query)) {
			results = append(results, doc.item)
		}
	}

	sort.Slice(results, func(a, b int) bool {
		if results[a].isDir != results[b].isDir {
			return results[a].isDir
		}
		return strings.ToLower(results[a].path) < strings.ToLower(results[b].path)
	})

	return results
}

func (i *searchIndex) upsertPath(path string) {
	if !i.ready {
		return
	}
	if !isWithinRoot(i.root, path) {
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		i.removePath(path)
		return
	}

	depth := depthFromRoot(i.root, path)
	name := filepath.Base(path)
	i.indexPath(path, name, depth, info.IsDir())
	if !info.IsDir() {
		return
	}

	i.removeDescendants(path)
	if err := i.walk(path, depth+1); err != nil {
		appLog.Warn("update search descendants", "path", path, "error", err)
	}
}

func (i *searchIndex) removePath(path string) {
	if !i.ready {
		return
	}
	delete(i.docs, path)
	i.removeDescendants(path)
}

func (i *searchIndex) removeDescendants(path string) {
	prefix := path + string(os.PathSeparator)
	for p := range i.docs {
		if strings.HasPrefix(p, prefix) {
			delete(i.docs, p)
		}
	}
}

func (i *searchIndex) indexPath(path, name string, depth int, isDir bool) {
	doc := searchDoc{
		item: treeItem{
			path:  path,
			name:  name,
			depth: depth,
			isDir: isDir,
		},
		nameLower: strings.ToLower(name),
	}
	if !isDir {
		doc.contentLower = readLowerMarkdownContent(path)
	}
	i.docs[path] = doc
}

func readLowerMarkdownContent(path string) string {
	if !strings.EqualFold(filepath.Ext(path), ".md") {
		return ""
	}

	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Size() > maxSearchFileBytes {
		return ""
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.ToLower(string(content))
}

func depthFromRoot(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}

	depth := 0
	for _, part := range strings.Split(filepath.Clean(rel), string(os.PathSeparator)) {
		if part != "" && part != "." {
			depth++
		}
	}
	if depth == 0 {
		return 0
	}
	return depth - 1
}

func isWithinRoot(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
