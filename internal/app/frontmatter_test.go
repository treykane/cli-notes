package app

import "testing"

func TestParseFrontmatterAndBody(t *testing.T) {
	content := "---\n" +
		"title: Project Plan\n" +
		"date: 2026-02-07\n" +
		"category: work\n" +
		"tags:\n" +
		"  - Go\n" +
		"  - CLI\n" +
		"---\n" +
		"# Body\nhello\n"

	meta, body := parseFrontmatterAndBody(content)
	if meta.Title != "Project Plan" {
		t.Fatalf("expected title, got %q", meta.Title)
	}
	if meta.Category != "work" {
		t.Fatalf("expected category, got %q", meta.Category)
	}
	if len(meta.Tags) != 2 || meta.Tags[0] != "go" || meta.Tags[1] != "cli" {
		t.Fatalf("unexpected tags: %#v", meta.Tags)
	}
	if body == content {
		t.Fatal("expected body to exclude frontmatter")
	}
}

func TestParseSearchQueryTagTokens(t *testing.T) {
	q := parseSearchQuery("rocket tag:go tag:cli")
	if len(q.textTerms) != 1 || q.textTerms[0] != "rocket" {
		t.Fatalf("unexpected text terms: %#v", q.textTerms)
	}
	if len(q.tagTerms) != 2 || q.tagTerms[0] != "go" || q.tagTerms[1] != "cli" {
		t.Fatalf("unexpected tag terms: %#v", q.tagTerms)
	}
}
