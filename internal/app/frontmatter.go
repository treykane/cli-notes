package app

import (
	"strings"
)

type NoteMetadata struct {
	Title    string
	Date     string
	Category string
	Tags     []string
}

func parseFrontmatterAndBody(content string) (NoteMetadata, string) {
	const delim = "---"
	trimmed := strings.TrimPrefix(content, "\ufeff")
	if !strings.HasPrefix(trimmed, delim+"\n") && !strings.HasPrefix(trimmed, delim+"\r\n") {
		return NoteMetadata{}, content
	}

	lines := strings.Split(trimmed, "\n")
	if len(lines) == 0 {
		return NoteMetadata{}, content
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == delim {
			end = i
			break
		}
	}
	if end <= 0 {
		return NoteMetadata{}, content
	}

	yamlText := strings.Join(lines[1:end], "\n")
	meta := parseSimpleFrontmatter(yamlText)
	body := strings.Join(lines[end+1:], "\n")
	return meta, body
}

func parseSimpleFrontmatter(yamlText string) NoteMetadata {
	meta := NoteMetadata{}
	lines := strings.Split(yamlText, "\n")
	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		i++
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
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
			if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
				value = strings.TrimPrefix(strings.TrimSuffix(value, "]"), "[")
				meta.Tags = normalizeTagList(strings.Split(value, ","))
				continue
			}
			if value != "" {
				meta.Tags = normalizeTagList(strings.Split(value, ","))
				continue
			}
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

func trimQuoted(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	return value
}

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

func compactTagLabel(tags []string, maxTags int) string {
	if len(tags) == 0 || maxTags <= 0 {
		return ""
	}
	if len(tags) <= maxTags {
		return strings.Join(tags, ",")
	}
	return strings.Join(tags[:maxTags], ",") + ",+"
}

type searchQuery struct {
	textTerms []string
	tagTerms  []string
}

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
