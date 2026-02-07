package app

import tea "github.com/charmbracelet/bubbletea"

var overlayCleanupByMode = map[overlayMode]func(*Model){
	overlaySearch: func(m *Model) {
		m.search.Blur()
		m.search.SetValue("")
		m.searchResults = nil
		m.searchResultCursor = 0
	},
	overlayWikiAutocomplete: func(m *Model) {
		m.wikiAutocomplete = nil
		m.wikiAutocompleteCursor = 0
	},
}

func cleanupOverlayModes() []overlayMode {
	modes := make([]overlayMode, 0, len(overlayCleanupByMode))
	for mode := range overlayCleanupByMode {
		modes = append(modes, mode)
	}
	return modes
}

// openOverlay activates one overlay and ensures any previous overlay state is cleaned up.
func (m *Model) openOverlay(mode overlayMode) {
	if m.overlay == mode {
		return
	}
	m.closeOverlay()
	m.overlay = mode
}

// closeOverlay dismisses the active overlay and resets overlay-specific state.
func (m *Model) closeOverlay() {
	if cleanup, ok := overlayCleanupByMode[m.overlay]; ok {
		cleanup(m)
	}
	m.overlay = overlayNone
}

func (m *Model) isOverlay(mode overlayMode) bool {
	return m.overlay == mode
}

// handlePopupListNav handles the shared up/down/select/close key patterns used by list popups.
// It returns (nextCursor, selectPressed, closePressed, handled).
func handlePopupListNav(msg tea.KeyMsg, cursor, count int) (int, bool, bool, bool) {
	key := msg.String()
	switch key {
	case "esc":
		return cursor, false, true, true
	case "up", "k", "ctrl+p":
		if count <= 0 {
			return 0, false, false, true
		}
		return clamp(cursor-1, 0, count-1), false, false, true
	case "down", "j", "ctrl+n":
		if count <= 0 {
			return 0, false, false, true
		}
		return clamp(cursor+1, 0, count-1), false, false, true
	case "enter":
		return cursor, true, false, true
	default:
		return cursor, false, false, false
	}
}
