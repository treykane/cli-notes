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
	item          treeItem
	nameLower     string
	contentLower  string
	titleLower    string
	categoryLower string
	tagsLower     []string
	metadata      NoteMetadata
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
		if shouldSkipManagedPath(entry.Name()) {
			continue
		}
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
	parsed := parseSearchQuery(query)
	if len(parsed.textTerms) == 0 && len(parsed.tagTerms) == 0 {
		return nil
	}

	results := make([]treeItem, 0, 32)
	for _, doc := range i.docs {
		if !docMatchesTags(doc, parsed.tagTerms) {
			continue
		}
		if !docMatchesText(doc, parsed.textTerms) {
			continue
		}
		if len(parsed.textTerms) == 0 && len(parsed.tagTerms) > 0 && doc.item.isDir {
			continue
		}
		if len(parsed.textTerms) == 0 && len(parsed.tagTerms) > 0 && len(doc.tagsLower) == 0 {
			continue
		}
		if len(parsed.textTerms) > 0 || len(parsed.tagTerms) > 0 {
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
	if shouldSkipManagedPath(filepath.Base(path)) {
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
		content, metadata := readMarkdownContentAndMetadata(path)
		doc.contentLower = strings.ToLower(content)
		doc.metadata = metadata
		doc.titleLower = strings.ToLower(metadata.Title)
		doc.categoryLower = strings.ToLower(metadata.Category)
		doc.tagsLower = metadata.Tags
		doc.item.tags = metadata.Tags
	}
	i.docs[path] = doc
}

func readMarkdownContentAndMetadata(path string) (string, NoteMetadata) {
	if !strings.EqualFold(filepath.Ext(path), ".md") {
		return "", NoteMetadata{}
	}

	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Size() > maxSearchFileBytes {
		return "", NoteMetadata{}
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", NoteMetadata{}
	}
	meta, body := parseFrontmatterAndBody(string(content))
	return body, meta
}

func readLowerMarkdownContent(path string) string {
	content, _ := readMarkdownContentAndMetadata(path)
	return strings.ToLower(content)
}

func docMatchesText(doc searchDoc, terms []string) bool {
	if len(terms) == 0 {
		return true
	}
	for _, term := range terms {
		if term == "" {
			continue
		}
		if strings.Contains(doc.nameLower, term) {
			continue
		}
		if strings.Contains(doc.titleLower, term) {
			continue
		}
		if strings.Contains(doc.categoryLower, term) {
			continue
		}
		if !doc.item.isDir && strings.Contains(doc.contentLower, term) {
			continue
		}
		return false
	}
	return true
}

func docMatchesTags(doc searchDoc, tags []string) bool {
	if len(tags) == 0 {
		return true
	}
	if len(doc.tagsLower) == 0 {
		return false
	}
	have := map[string]bool{}
	for _, tag := range doc.tagsLower {
		have[tag] = true
	}
	for _, tag := range tags {
		if !have[tag] {
			return false
		}
	}
	return true
}

type noteTarget struct {
	Path  string
	Title string
	Name  string
}

func (i *searchIndex) noteTargets() []noteTarget {
	out := make([]noteTarget, 0, len(i.docs))
	for _, doc := range i.docs {
		if doc.item.isDir || !hasSuffixCaseInsensitive(doc.item.path, ".md") {
			continue
		}
		out = append(out, noteTarget{
			Path:  doc.item.path,
			Title: doc.metadata.Title,
			Name:  strings.TrimSuffix(doc.item.name, filepath.Ext(doc.item.name)),
		})
	}
	sort.Slice(out, func(a, b int) bool {
		return strings.ToLower(out[a].Path) < strings.ToLower(out[b].Path)
	})
	return out
}

func (i *searchIndex) resolveWikiTarget(label string) (string, bool) {
	label = strings.TrimSpace(strings.ToLower(label))
	if label == "" {
		return "", false
	}
	// 1) title match
	for _, doc := range i.docs {
		if doc.item.isDir {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(doc.metadata.Title), label) {
			return doc.item.path, true
		}
	}
	// 2) filename stem fallback
	for _, doc := range i.docs {
		if doc.item.isDir {
			continue
		}
		stem := strings.TrimSuffix(doc.item.name, filepath.Ext(doc.item.name))
		if strings.EqualFold(strings.TrimSpace(stem), label) {
			return doc.item.path, true
		}
	}
	return "", false
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
