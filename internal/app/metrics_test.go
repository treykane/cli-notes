package app

import "testing"

func TestComputeNoteMetrics(t *testing.T) {
	metrics := computeNoteMetrics("one two\nthree\n")
	if metrics.words != 3 {
		t.Fatalf("expected 3 words, got %d", metrics.words)
	}
	if metrics.chars != 14 {
		t.Fatalf("expected 14 chars, got %d", metrics.chars)
	}
	if metrics.lines != 2 {
		t.Fatalf("expected 2 lines, got %d", metrics.lines)
	}
}

func TestNoteMetricsSummaryEmpty(t *testing.T) {
	m := &Model{}
	if got := m.noteMetricsSummary(); got != "" {
		t.Fatalf("expected empty summary, got %q", got)
	}
}
