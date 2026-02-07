package app

import "testing"

func TestRankWikiTargetsPrioritizesPrefixThenFrequency(t *testing.T) {
	targets := []noteTarget{
		{Path: "/a.md", Title: "Project Plan", Name: "project-plan"},
		{Path: "/b.md", Title: "Alpha Notes", Name: "my-project"},
		{Path: "/c.md", Title: "Prototype Ideas", Name: "proto-ideas"},
	}
	openCounts := map[string]int{
		"/a.md": 2,
		"/b.md": 80,
		"/c.md": 6,
	}

	ranked := rankWikiTargets(targets, "pro", openCounts)
	if len(ranked) != 3 {
		t.Fatalf("expected 3 ranked targets, got %d", len(ranked))
	}
	if ranked[0].Path != "/c.md" {
		t.Fatalf("expected /c.md first (prefix + higher opens), got %s", ranked[0].Path)
	}
	if ranked[1].Path != "/a.md" {
		t.Fatalf("expected /a.md second (prefix), got %s", ranked[1].Path)
	}
	if ranked[2].Path != "/b.md" {
		t.Fatalf("expected /b.md last (substring only), got %s", ranked[2].Path)
	}
}

func TestRankWikiTargetsEmptyPrefixUsesFrequency(t *testing.T) {
	targets := []noteTarget{
		{Path: "/a.md", Title: "Alpha", Name: "alpha"},
		{Path: "/b.md", Title: "Beta", Name: "beta"},
	}
	openCounts := map[string]int{
		"/a.md": 1,
		"/b.md": 9,
	}
	ranked := rankWikiTargets(targets, "", openCounts)
	if len(ranked) != 2 {
		t.Fatalf("expected 2 ranked targets, got %d", len(ranked))
	}
	if ranked[0].Path != "/b.md" {
		t.Fatalf("expected /b.md first by open count, got %s", ranked[0].Path)
	}
}
