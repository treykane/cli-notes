// search_index.go implements the in-memory full-text search index for Ctrl+P.
//
// The index is built lazily on first use and incrementally updated when notes
// are created, edited, renamed, moved, or deleted. It supports two query types:
//
//   - Free-text terms: matched case-insensitively against filename, frontmatter
//     title, frontmatter category, and note body content.
//   - Tag filters: queries containing "tag:<name>" restrict results to notes
//     whose YAML frontmatter includes the specified tag(s).
//
// Files larger than MaxSearchFileBytes (1 MiB) are excluded from content
// indexing to avoid excessive memory use, but their filenames are still
// searchable. The managed `.cli-notes` directory is always skipped.
//
// The index stores pre-lowercased copies of all searchable fields so that
// query matching is a simple strings.Contains call with no per-query
// allocation for case folding.
package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// searchDoc holds the indexed data for a single file or directory.
//
// All text fields are stored in lowercase so query matching is case-insensitive
// without needing to lowercase on every comparison. The original treeItem is
// embedded so search results can be returned directly as tree items.
type searchDoc struct {
	item          treeItem     // the tree row this document represents
	nameLower     string       // lowercased filename (always populated)
	contentLower  string       // lowercased markdown body (files only, empty for dirs)
	titleLower    string       // lowercased frontmatter title (files only)
	categoryLower string       // lowercased frontmatter category (files only)
	tagsLower     []string     // lowercased frontmatter tags (files only)
	metadata      NoteMetadata // parsed frontmatter metadata (files only)
}

// searchIndex is the in-memory search index for the notes directory.
//
// It is keyed by absolute file path. The index is considered "ready" after a
// successful full build; incremental updates (upsert/remove) only operate when
// the index is already built. If the index is invalidated (e.g. by the file
// watcher detecting external changes), the next ensureBuilt call triggers a
// complete rebuild.
type searchIndex struct {
	root        string               // absolute path to the notes directory root
	docs        map[string]searchDoc // path -> indexed document
	sortedPaths []string             // lexicographically sorted paths for prefix range operations
	ready       bool                 // true after a successful build; false after invalidate()
}

// newSearchIndex creates an unbuilt search index rooted at the given directory.
// The index must be populated by calling ensureBuilt before any queries.
func newSearchIndex(root string) *searchIndex {
	return &searchIndex{
		root: root,
		docs: map[string]searchDoc{},
	}
}

// invalidate marks the index as stale, forcing a full rebuild on the next
// ensureBuilt call. This is used when the file watcher detects external
// changes or when the user explicitly refreshes (Shift+R).
func (i *searchIndex) invalidate() {
	i.ready = false
}

// ensureBuilt lazily builds the index if it has not been built yet or was
// invalidated. Returns nil if the index is already ready.
func (i *searchIndex) ensureBuilt() error {
	if i.ready {
		return nil
	}
	return i.build()
}

// build performs a full index rebuild by walking the entire notes directory
// tree. Any previously indexed documents are discarded. On success the index
// is marked ready; on failure it remains in an unready state so the next
// ensureBuilt call will retry.
func (i *searchIndex) build() error {
	i.docs = map[string]searchDoc{}
	i.sortedPaths = nil
	if err := i.walk(i.root, 0); err != nil {
		i.ready = false
		return err
	}
	i.ready = true
	return nil
}

// walk recursively traverses dir, indexing each entry. Directories and files
// in the managed `.cli-notes` path are skipped. Entries are sorted
// (directories first, then case-insensitive alphabetical) to produce
// deterministic index ordering.
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

// search returns all indexed items matching the query string.
//
// Query parsing (via parseSearchQuery):
//   - Tokens prefixed with "tag:" are treated as tag filters.
//   - All other tokens are free-text search terms.
//
// Matching algorithm:
//  1. Parse the query into tag terms and text terms.
//  2. For each indexed document, check tag match first (all specified tags
//     must be present in the document's frontmatter tags).
//  3. Then check text match: every text term must appear in at least one of
//     the document's searchable fields (filename, title, category, or body
//     content). Directory entries are only matched against their name.
//  4. Tag-only queries (no text terms) exclude directories and documents
//     without tags, since tag filtering only applies to markdown files.
//  5. Results are sorted: directories first, then alphabetically by path.
//
// Returns nil if the query is empty or has no terms after parsing.
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

// upsertPath incrementally updates the index for a single path. If the path
// no longer exists on disk, it is removed from the index. For directories,
// all previously indexed descendants are removed and the subtree is re-walked
// to pick up any new or changed files within it.
//
// This is a no-op if the index has not been built yet (i.ready == false),
// because there is nothing to update incrementally — the next ensureBuilt
// call will do a full rebuild anyway.
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

// removePath removes a single path and all its descendants from the index.
// This is a no-op if the index is not built.
func (i *searchIndex) removePath(path string) {
	if !i.ready {
		return
	}
	i.deleteDoc(path)
	i.removeDescendants(path)
}

// removeDescendants deletes all indexed entries whose path is a child of the
// given directory path. Used when a directory is removed or when upsertPath
// needs to re-walk a directory's contents.
func (i *searchIndex) removeDescendants(path string) {
	i.ensurePathIndex()
	prefix := path + string(os.PathSeparator)
	start := sort.SearchStrings(i.sortedPaths, prefix)
	end := start
	for end < len(i.sortedPaths) && strings.HasPrefix(i.sortedPaths[end], prefix) {
		delete(i.docs, i.sortedPaths[end])
		end++
	}
	if end > start {
		i.sortedPaths = append(i.sortedPaths[:start], i.sortedPaths[end:]...)
	}
}

// indexPath creates a searchDoc for the given filesystem entry and stores it
// in the index. For markdown files, the file is read and its frontmatter
// parsed to populate title, category, tags, and body content fields. Files
// larger than MaxSearchFileBytes or non-markdown files get a name-only entry.
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
	i.upsertDoc(path, doc)
}

func (i *searchIndex) ensurePathIndex() {
	if len(i.sortedPaths) == len(i.docs) {
		return
	}
	i.sortedPaths = i.sortedPaths[:0]
	for path := range i.docs {
		i.sortedPaths = append(i.sortedPaths, path)
	}
	sort.Strings(i.sortedPaths)
}

func (i *searchIndex) upsertDoc(path string, doc searchDoc) {
	if _, exists := i.docs[path]; exists {
		i.docs[path] = doc
		return
	}
	i.ensurePathIndex()
	i.docs[path] = doc
	pos := sort.SearchStrings(i.sortedPaths, path)
	i.sortedPaths = append(i.sortedPaths, "")
	copy(i.sortedPaths[pos+1:], i.sortedPaths[pos:])
	i.sortedPaths[pos] = path
}

func (i *searchIndex) deleteDoc(path string) {
	if _, exists := i.docs[path]; !exists {
		return
	}
	i.ensurePathIndex()
	delete(i.docs, path)
	pos := sort.SearchStrings(i.sortedPaths, path)
	if pos < len(i.sortedPaths) && i.sortedPaths[pos] == path {
		i.sortedPaths = append(i.sortedPaths[:pos], i.sortedPaths[pos+1:]...)
	}
}

// readMarkdownContentAndMetadata reads a markdown file, parses its YAML
// frontmatter, and returns the body content (without frontmatter) and the
// extracted metadata. Returns empty values for non-markdown files, directories,
// files larger than MaxSearchFileBytes, or on read errors.
func readMarkdownContentAndMetadata(path string) (string, NoteMetadata) {
	if !strings.EqualFold(filepath.Ext(path), ".md") {
		return "", NoteMetadata{}
	}

	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Size() > MaxSearchFileBytes {
		return "", NoteMetadata{}
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", NoteMetadata{}
	}
	meta, body := parseFrontmatterAndBody(string(content))
	return body, meta
}

// readLowerMarkdownContent is a convenience wrapper that returns the lowercased
// body content of a markdown file, suitable for case-insensitive searching.
func readLowerMarkdownContent(path string) string {
	content, _ := readMarkdownContentAndMetadata(path)
	return strings.ToLower(content)
}

// docMatchesText returns true if every text term appears in at least one of the
// document's searchable fields: filename, title, category, or body content.
// Directory entries are only matched against their name (contentLower is empty).
// An empty terms slice matches everything (vacuous truth).
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

// docMatchesTags returns true if the document's frontmatter tags include all
// of the specified tag terms. An empty tags slice matches everything. A
// document with no tags will not match any non-empty tag query.
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

// noteTarget represents a candidate for wiki-link autocomplete. It carries
// the note's path, frontmatter title (if any), and filename stem so the
// autocomplete popup can display the most useful label.
type noteTarget struct {
	Path  string // absolute filesystem path to the note
	Title string // frontmatter title (may be empty)
	Name  string // filename without extension
}

// noteTargets returns all indexed markdown files as autocomplete candidates,
// sorted alphabetically by path. This is used by the wiki-link autocomplete
// popup to provide a filterable list of all notes in the workspace.
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

// resolveWikiTarget attempts to find a note matching the given wiki-link label.
//
// Resolution strategy (first match wins):
//  1. Exact frontmatter title match (case-insensitive, whitespace-trimmed).
//  2. Filename stem match (case-insensitive) — the filename without its
//     extension is compared to the label.
//
// Returns the absolute path and true if a match is found, or ("", false)
// if no note matches. This two-pass approach means a note with a title
// "My Note" can be linked as [[My Note]] even if its filename is different.
func (i *searchIndex) resolveWikiTarget(label string) (string, bool) {
	label = strings.TrimSpace(strings.ToLower(label))
	if label == "" {
		return "", false
	}
	// Pass 1: match against frontmatter title.
	for _, doc := range i.docs {
		if doc.item.isDir {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(doc.metadata.Title), label) {
			return doc.item.path, true
		}
	}
	// Pass 2: match against filename stem (name without extension).
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

// depthFromRoot calculates how many directory levels path is below root.
// This is used to set the correct indentation depth for search result items
// so they render properly in the tree view. The depth is zero-indexed
// relative to the root's immediate children.
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

// isWithinRoot reports whether path is equal to or contained within root.
// It uses filepath.Rel to check that the relative path does not escape via
// ".." components. This is a safety check used throughout the app to prevent
// operations on paths outside the configured notes directory.
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
