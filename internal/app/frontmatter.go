package app

import (
	"strings"
)

// NoteMetadata holds structured metadata extracted from the YAML frontmatter
// block at the top of a markdown note file.
//
// Frontmatter is an optional section delimited by "---" lines at the very
// beginning of a file. It allows users to attach structured data to their
// notes that can be used for search filtering, display labels, and
// organization.
//
// Example frontmatter:
//
//	---
//	title: "My Note Title"
//	date: 2025-02-07
//	category: work
//	tags: [go, cli, notes]
//	---
//
// If a note has no frontmatter block, all fields will be zero-valued.
type NoteMetadata struct {
	// Title is the human-readable display name for the note. If set, it
	// takes priority over the filename in search results and wiki link
	// resolution.
	Title string

	// Date is the note's creation or publication date as a raw string.
	// No date parsing is performed; it is stored exactly as written in
	// the frontmatter (e.g. "2025-02-07", "Feb 7, 2025").
	Date string

	// Category is an optional organizational label (e.g. "work", "personal").
	// It is matched during search queries alongside title and content.
	Category string

	// Tags is a list of lowercase, deduplicated tag strings attached to the
	// note. Tags are used for:
	//   - Display in tree view rows (as compact badge labels).
	//   - Filtering in the Ctrl+P search popup via "tag:<name>" syntax.
	//   - Metadata-aware search matching.
	Tags []string
}

// parseFrontmatterAndBody splits a markdown file's content into its YAML
// frontmatter metadata and the remaining body text.
//
// The function looks for a frontmatter block at the very beginning of the
// content: a line containing exactly "---" followed by YAML key-value pairs,
// terminated by another "---" line.
//
// Returns:
//   - meta: Parsed NoteMetadata from the frontmatter block (zero-valued if
//     no valid frontmatter is found).
//   - body: The remaining content after the closing "---" delimiter. If no
//     frontmatter is found, the entire original content is returned as body.
//
// The function handles BOM-prefixed files (strips the leading \ufeff) and
// supports both Unix (\n) and Windows (\r\n) line endings in the delimiter
// detection.
func parseFrontmatterAndBody(content string) (NoteMetadata, string) {
	const delim = "---"

	// Strip optional Unicode BOM (byte order mark) that some editors prepend.
	trimmed := strings.TrimPrefix(content, "\ufeff")

	// The frontmatter block must start at the very first line of the file.
	if !strings.HasPrefix(trimmed, delim+"\n") && !strings.HasPrefix(trimmed, delim+"\r\n") {
		return NoteMetadata{}, content
	}

	lines := strings.Split(trimmed, "\n")
	if len(lines) == 0 {
		return NoteMetadata{}, content
	}

	// Find the closing "---" delimiter. We start at line 1 because line 0
	// is the opening delimiter.
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == delim {
			end = i
			break
		}
	}
	if end <= 0 {
		// No closing delimiter found — treat the entire content as body
		// with no frontmatter.
		return NoteMetadata{}, content
	}

	// Extract the YAML text between the two "---" delimiters and parse it.
	yamlText := strings.Join(lines[1:end], "\n")
	meta := parseSimpleFrontmatter(yamlText)

	// Everything after the closing delimiter is the note body.
	body := strings.Join(lines[end+1:], "\n")
	return meta, body
}

// parseSimpleFrontmatter performs a lightweight, dependency-free parse of
// YAML-like frontmatter text.
//
// This is intentionally not a full YAML parser. It supports only the subset
// of YAML syntax commonly used in markdown frontmatter:
//
//   - Simple key: value pairs (one per line).
//   - Inline arrays: tags: [go, cli, notes]
//   - Comma-separated values: tags: go, cli, notes
//   - Bullet-list arrays (indented "- item" lines following a key with no
//     inline value).
//   - Quoted values (single or double quotes are stripped).
//   - Comment lines (starting with #) and blank lines are skipped.
//
// Recognized keys (case-insensitive): title, date, category, tags.
// Unrecognized keys are silently ignored.
func parseSimpleFrontmatter(yamlText string) NoteMetadata {
	meta := NoteMetadata{}
	lines := strings.Split(yamlText, "\n")
	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		i++

		// Skip blank lines and YAML comments.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Each line should be a "key: value" pair.
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])

		switch key {
		case "title":
			meta.Title = trimQuoted(value)
		case "date":
			meta.Date = trimQuoted(value)
		case "category":
			meta.Category = trimQuoted(value)
		case "tags":
			// Tags support three syntax variants:
			//
			// 1. Inline JSON-style array:  tags: [go, cli, notes]
			// 2. Comma-separated inline:   tags: go, cli, notes
			// 3. YAML bullet list:
			//      tags:
			//        - go
			//        - cli
			if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
				// Variant 1: Inline bracketed array — strip brackets and
				// split on commas.
				value = strings.TrimPrefix(strings.TrimSuffix(value, "]"), "[")
				meta.Tags = normalizeTagList(strings.Split(value, ","))
				continue
			}
			if value != "" {
				// Variant 2: Comma-separated values on the same line as
				// the key.
				meta.Tags = normalizeTagList(strings.Split(value, ","))
				continue
			}
			// Variant 3: No inline value — look for indented bullet items
			// on subsequent lines. Each bullet line starts with "-".
			bullets := make([]string, 0, 4)
			for i < len(lines) {
				next := strings.TrimSpace(lines[i])
				if strings.HasPrefix(next, "-") {
					bullets = append(bullets, strings.TrimSpace(strings.TrimPrefix(next, "-")))
					i++
					continue
				}
				break
			}
			meta.Tags = normalizeTagList(bullets)
		}
	}
	return meta
}

// trimQuoted removes surrounding single or double quotes from a value string,
// plus any leading/trailing whitespace.
//
// Examples:
//
//	trimQuoted(`"My Title"`) → "My Title"
//	trimQuoted(`'hello'`)    → "hello"
//	trimQuoted(`plain`)      → "plain"
func trimQuoted(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	return value
}

// normalizeTagList processes a raw slice of tag strings into a clean,
// deduplicated, lowercase list suitable for storage and matching.
//
// Processing steps for each tag:
//  1. Trim surrounding whitespace.
//  2. Convert to lowercase for case-insensitive matching.
//  3. Skip empty strings and duplicates.
//
// Returns nil (not an empty slice) if no valid tags remain after filtering,
// which keeps JSON serialization clean (omitted rather than "tags": []).
func normalizeTagList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		tag := strings.ToLower(strings.TrimSpace(value))
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// compactTagLabel formats a slice of tags into a short display string for
// use in tree view row badges.
//
// If the number of tags exceeds maxTags, only the first maxTags are shown
// followed by a ",+" suffix to indicate truncation.
//
// Examples:
//
//	compactTagLabel(["go", "cli"], 3)        → "go,cli"
//	compactTagLabel(["go", "cli", "tui"], 2) → "go,cli,+"
//	compactTagLabel([], 3)                   → ""
//	compactTagLabel(["go"], 0)               → ""
func compactTagLabel(tags []string, maxTags int) string {
	if len(tags) == 0 || maxTags <= 0 {
		return ""
	}
	if len(tags) <= maxTags {
		return strings.Join(tags, ",")
	}
	return strings.Join(tags[:maxTags], ",") + ",+"
}

// searchQuery represents a parsed Ctrl+P search input, split into separate
// text search terms and tag filter terms.
//
// The search popup supports a special "tag:<name>" prefix syntax for
// filtering results by tag. All other words are treated as text search
// terms that match against note names, titles, categories, and content.
//
// Example query: "meeting notes tag:work tag:important"
//
//	textTerms: ["meeting", "notes"]
//	tagTerms:  ["work", "important"]
type searchQuery struct {
	// textTerms contains lowercase words that are matched against note
	// filenames, frontmatter titles/categories, and body content.
	textTerms []string

	// tagTerms contains lowercase tag names (without the "tag:" prefix)
	// that must all be present in a note's frontmatter tags for the note
	// to match the query.
	tagTerms []string
}

// parseSearchQuery splits a raw search input string into text terms and
// tag filter terms.
//
// The input is lowercased and split on whitespace. Tokens that start with
// "tag:" are extracted as tag filter terms (with the prefix stripped); all
// other tokens become text search terms.
//
// Both term lists are pre-allocated with reasonable initial capacities to
// minimize allocations during interactive search.
func parseSearchQuery(query string) searchQuery {
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	parsed := searchQuery{
		textTerms: make([]string, 0, len(fields)),
		tagTerms:  make([]string, 0, 4),
	}
	for _, token := range fields {
		if strings.HasPrefix(token, "tag:") {
			tag := strings.TrimSpace(strings.TrimPrefix(token, "tag:"))
			if tag != "" {
				parsed.tagTerms = append(parsed.tagTerms, tag)
			}
			continue
		}
		parsed.textTerms = append(parsed.textTerms, token)
	}
	return parsed
}
