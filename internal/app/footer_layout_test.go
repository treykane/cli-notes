package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestFooterHeightForWidthPrefersTwoRowsWhenFit(t *testing.T) {
	m := &Model{
		mode: modeBrowse,
	}

	if got := m.footerHeightForWidth(240); got != FooterMinRows {
		t.Fatalf("expected %d footer rows at wide width, got %d", FooterMinRows, got)
	}
}

func TestFooterHeightForWidthExpandsToThreeRowsWhenNeeded(t *testing.T) {
	m := &Model{
		mode:   modeBrowse,
		status: "Auto-refreshed after external filesystem changes and search index rebuild",
	}

	if got := m.footerHeightForWidth(72); got != FooterMaxRows {
		t.Fatalf("expected %d footer rows at narrow width, got %d", FooterMaxRows, got)
	}
}

func TestBuildStatusRowsTruncatesWithEllipsisWhenOverCapacity(t *testing.T) {
	m := &Model{
		mode:   modeBrowse,
		status: strings.Repeat("status ", 30),
	}

	rows, fit := m.buildStatusRows(28, FooterMaxRows)
	if fit {
		t.Fatal("expected rows to overflow and require truncation")
	}
	if len(rows) != FooterMaxRows {
		t.Fatalf("expected %d rows, got %d", FooterMaxRows, len(rows))
	}
	if !strings.Contains(rows[len(rows)-1], "…") {
		t.Fatalf("expected ellipsis in final row, got %q", rows[len(rows)-1])
	}
}

func TestStatusHelpSegmentsByMode(t *testing.T) {
	t.Run("browse", func(t *testing.T) {
		m := &Model{mode: modeBrowse}
		joined := strings.Join(m.statusHelpSegments(), " | ")
		if !strings.Contains(joined, "Ctrl+P search") {
			t.Fatalf("expected browse help to include search, got %q", joined)
		}
		if !strings.Contains(joined, "PgUp/PgDn preview") {
			t.Fatalf("expected browse help to include preview page scroll, got %q", joined)
		}
		if !strings.Contains(joined, "q quit") {
			t.Fatalf("expected browse help to include quit, got %q", joined)
		}
	})

	t.Run("edit", func(t *testing.T) {
		m := &Model{mode: modeEditNote}
		joined := strings.Join(m.statusHelpSegments(), " | ")
		if !strings.Contains(joined, "Ctrl+S save") {
			t.Fatalf("expected edit help to include save, got %q", joined)
		}
		if !strings.Contains(joined, "Ctrl+V paste") {
			t.Fatalf("expected edit help to include paste, got %q", joined)
		}
	})

	t.Run("popup", func(t *testing.T) {
		m := &Model{mode: modeBrowse, searching: true}
		joined := strings.Join(m.statusHelpSegments(), " | ")
		if !strings.Contains(joined, "Search popup") {
			t.Fatalf("expected popup help to include popup context, got %q", joined)
		}
	})
}

func TestCalculateLayoutReservesFooterRowsAndStaysNonNegative(t *testing.T) {
	m := &Model{
		mode:   modeBrowse,
		width:  70,
		height: 2,
	}
	layout := m.calculateLayout()
	if layout.ContentHeight < 0 {
		t.Fatalf("expected non-negative content height, got %d", layout.ContentHeight)
	}

	m.width = 240
	m.height = 24
	layout = m.calculateLayout()
	expected := 24 - FooterMinRows
	if layout.ContentHeight != expected {
		t.Fatalf("expected content height %d, got %d", expected, layout.ContentHeight)
	}
}

func TestViewPadsToTerminalSizeWithAdaptiveFooter(t *testing.T) {
	m := &Model{
		notesDir: "/tmp/notes",
		mode:     modeBrowse,
		width:    90,
		height:   20,
		status:   "Ready",
	}

	out := m.View()
	lines := strings.Split(out, "\n")
	if len(lines) != m.height {
		t.Fatalf("expected %d lines, got %d", m.height, len(lines))
	}
	for i, line := range lines {
		if w := lipgloss.Width(line); w != m.width {
			t.Fatalf("line %d width mismatch: expected %d, got %d", i+1, m.width, w)
		}
	}
}
