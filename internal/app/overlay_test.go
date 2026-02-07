package app

import (
	"slices"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
)

func allConcreteOverlayModesForTest() []overlayMode {
	return []overlayMode{
		overlaySearch,
		overlayRecent,
		overlayOutline,
		overlayWorkspace,
		overlayExport,
		overlayWikiLinks,
		overlayWikiAutocomplete,
	}
}

func TestOverlayModeCoverageGuard(t *testing.T) {
	modes := allConcreteOverlayModesForTest()
	if want := int(overlayWikiAutocomplete); len(modes) != want {
		t.Fatalf("overlay coverage list out of date: got %d overlays, expected %d", len(modes), want)
	}
}

func TestOverlayCleanupCoverageGuard(t *testing.T) {
	assertedCleanup := []overlayMode{
		overlaySearch,
		overlayWikiAutocomplete,
	}

	got := cleanupOverlayModes()
	slices.Sort(got)
	slices.Sort(assertedCleanup)
	if !slices.Equal(got, assertedCleanup) {
		t.Fatalf("cleanup assertions out of sync.\nmap=%v\ntest=%v", got, assertedCleanup)
	}
}

func TestOpenOverlayTransitionsAndCleanup(t *testing.T) {
	modes := append([]overlayMode{overlayNone}, allConcreteOverlayModesForTest()...)

	for _, from := range modes {
		for _, to := range allConcreteOverlayModesForTest() {
			name := from.String() + "_to_" + to.String()
			t.Run(name, func(t *testing.T) {
				m := dirtyOverlayModel(from)
				searchBefore := m.search.Value()
				wikiBefore := len(m.wikiAutocomplete)

				m.openOverlay(to)

				if m.overlay != to {
					t.Fatalf("expected overlay %v, got %v", to, m.overlay)
				}

				if from == to {
					if from == overlaySearch && m.search.Value() != searchBefore {
						t.Fatalf("same-mode open should not clear search state")
					}
					if from == overlayWikiAutocomplete && len(m.wikiAutocomplete) != wikiBefore {
						t.Fatalf("same-mode open should not clear wiki autocomplete state")
					}
					return
				}

				switch from {
				case overlaySearch:
					assertSearchCleanup(t, m)
				case overlayWikiAutocomplete:
					assertWikiAutocompleteCleanup(t, m)
				}
			})
		}
	}
}

func TestCloseOverlayCleanupByMode(t *testing.T) {
	for _, from := range allConcreteOverlayModesForTest() {
		t.Run(from.String(), func(t *testing.T) {
			m := dirtyOverlayModel(from)
			m.closeOverlay()

			if m.overlay != overlayNone {
				t.Fatalf("expected overlayNone after close, got %v", m.overlay)
			}

			switch from {
			case overlaySearch:
				assertSearchCleanup(t, m)
			case overlayWikiAutocomplete:
				assertWikiAutocompleteCleanup(t, m)
			default:
				if got := m.search.Value(); got != "search-term" {
					t.Fatalf("expected unrelated search state unchanged, got %q", got)
				}
				if got := len(m.wikiAutocomplete); got != 1 {
					t.Fatalf("expected unrelated wiki autocomplete state unchanged, got %d entries", got)
				}
			}
		})
	}
}

func dirtyOverlayModel(mode overlayMode) *Model {
	search := textinput.New()
	search.SetValue("search-term")
	search.Focus()
	return &Model{
		overlay: mode,
		search:  search,
		searchResults: []treeItem{
			{name: "match"},
		},
		searchResultCursor: 3,
		wikiAutocomplete: []noteTarget{
			{Name: "note"},
		},
		wikiAutocompleteCursor: 2,
	}
}

func assertSearchCleanup(t *testing.T, m *Model) {
	t.Helper()
	if got := m.search.Value(); got != "" {
		t.Fatalf("expected search value reset, got %q", got)
	}
	if got := len(m.searchResults); got != 0 {
		t.Fatalf("expected search results cleared, got %d", got)
	}
	if m.searchResultCursor != 0 {
		t.Fatalf("expected search cursor reset, got %d", m.searchResultCursor)
	}
	if m.search.Focused() {
		t.Fatal("expected search input blurred")
	}
}

func assertWikiAutocompleteCleanup(t *testing.T, m *Model) {
	t.Helper()
	if got := len(m.wikiAutocomplete); got != 0 {
		t.Fatalf("expected wiki autocomplete cleared, got %d", got)
	}
	if m.wikiAutocompleteCursor != 0 {
		t.Fatalf("expected wiki autocomplete cursor reset, got %d", m.wikiAutocompleteCursor)
	}
}

func (m overlayMode) String() string {
	switch m {
	case overlayNone:
		return "none"
	case overlaySearch:
		return "search"
	case overlayRecent:
		return "recent"
	case overlayOutline:
		return "outline"
	case overlayWorkspace:
		return "workspace"
	case overlayExport:
		return "export"
	case overlayWikiLinks:
		return "wiki_links"
	case overlayWikiAutocomplete:
		return "wiki_autocomplete"
	default:
		return "unknown"
	}
}
