// wiki.go implements wiki-style [[link]] support: parsing, resolution,
// navigation, and edit-mode autocomplete.
//
// Wiki links use the syntax [[Label]], where Label is matched against note
// titles (from YAML frontmatter) and filename stems (without extension).
// Links inside fenced code blocks (``` ... ```) are intentionally ignored
// to avoid false positives in code samples.
//
// Two UI surfaces consume wiki links:
//
//   - Browse-mode popup (Shift+L): lists all [[links]] in the current note,
//     shows whether each resolves to an existing note, and allows jumping
//     to the target with Enter.
//   - Edit-mode autocomplete: typing "[[" triggers a filterable popup of all
//     note titles/names. Selecting an entry inserts the label and closing "]]".
package app

import (
	"regexp"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// wikiLink represents a single parsed [[link]] from a note's content.
// Label is the raw text between the brackets, Target is the resolved absolute
// path (empty if unresolved), and Resolved indicates whether Target was found.
type wikiLink struct {
	Label    string
	Target   string
	Resolved bool
}

// wikiLinkPattern matches [[...]] tokens, capturing the inner label.
// It does not match nested brackets ([[a[b]c]]) by excluding [ and ] from
// the capture group.
var wikiLinkPattern = regexp.MustCompile(`\[\[([^\[\]]+)\]\]`)

// openWikiLinksPopup parses all [[links]] from the current note, resolves each
// against the search index (title match first, then filename stem), and opens
// a navigable popup listing the results. Unresolved links are shown but cannot
// be jumped to.
func (m *Model) openWikiLinksPopup() {
	if m.currentFile == "" {
		m.status = "Select a note first"
		return
	}
	links := parseWikiLinks(m.currentNoteContent)
	if len(links) == 0 {
		m.status = "No wiki links in current note"
		return
	}
	if m.searchIndex == nil {
		m.searchIndex = newSearchIndex(m.notesDir)
	}
	if err := m.searchIndex.ensureBuilt(); err != nil {
		m.status = "Wiki link index unavailable"
		return
	}
	wikiRows := make([]wikiLink, 0, len(links))
	for _, label := range links {
		path, ok := m.searchIndex.resolveWikiTarget(label)
		wikiRows = append(wikiRows, wikiLink{
			Label:    label,
			Target:   path,
			Resolved: ok,
		})
	}
	m.openOverlay(overlayWikiLinks)
	m.wikiLinks = wikiRows
	m.wikiLinkCursor = 0
	m.status = "Wiki links: Enter to open, Esc to close"
}

// handleWikiLinksPopupKey routes key presses while the wiki-links popup is
// visible. Supports up/down navigation, Enter to jump to a resolved link,
// and Esc to dismiss.
func (m *Model) handleWikiLinksPopupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.shouldIgnoreInput(msg) {
		return m, nil
	}
	next, selectPressed, closePressed, handled := handlePopupListNav(msg, m.wikiLinkCursor, len(m.wikiLinks))
	if !handled {
		return m, nil
	}
	if closePressed {
		m.closeOverlay()
		m.status = "Wiki links closed"
		return m, nil
	}
	if len(m.wikiLinks) == 0 {
		return m, nil
	}
	m.wikiLinkCursor = next
	if selectPressed {
		link := m.wikiLinks[m.wikiLinkCursor]
		if !link.Resolved || link.Target == "" {
			m.status = "Unresolved wiki link: " + link.Label
			return m, nil
		}
		m.closeOverlay()
		m.expandParentDirs(link.Target)
		m.rebuildTreeKeep(link.Target)
		m.status = "Opened wiki link: " + link.Label
		return m, m.setFocusedFile(link.Target)
	}
	return m, nil
}

// parseWikiLinks extracts unique wiki-link labels from markdown content.
//
// The parser is fence-aware: lines inside fenced code blocks (delimited by
// ```) are skipped so that [[...]] tokens in code samples are not treated
// as real links. Labels are deduplicated case-insensitively; only the first
// occurrence of each label is returned to keep the popup concise.
func parseWikiLinks(content string) []string {
	if strings.TrimSpace(content) == "" {
		return nil
	}
	lines := strings.Split(content, "\n")
	inFence := false
	out := make([]string, 0, 8)
	seen := map[string]bool{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		matches := wikiLinkPattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			label := strings.TrimSpace(match[1])
			if label == "" || seen[strings.ToLower(label)] {
				continue
			}
			seen[strings.ToLower(label)] = true
			out = append(out, label)
		}
	}
	return out
}

// renderWikiLinksPopup draws the wiki-links popup content showing each link's
// label, its resolved target path (or "(unresolved)"), and navigation hints.
func (m *Model) renderWikiLinksPopup(width, height int) string {
	innerWidth := max(0, width-popupStyle.GetHorizontalFrameSize())
	innerHeight := max(0, height-popupStyle.GetVerticalFrameSize())
	lines := []string{
		titleStyle.Render("Wiki Links"),
		"",
	}
	limit := max(0, innerHeight-len(lines)-1)
	for i := 0; i < min(limit, len(m.wikiLinks)); i++ {
		link := m.wikiLinks[i]
		label := "[[" + link.Label + "]]"
		if link.Resolved {
			label += " -> " + m.displayRelative(link.Target)
		} else {
			label += " -> (unresolved)"
		}
		line := truncate(label, innerWidth)
		if i == m.wikiLinkCursor {
			line = selectedStyle.Render(line)
		}
		lines = append(lines, line)
	}
	if len(m.wikiLinks) == 0 {
		lines = append(lines, mutedStyle.Render("No links"))
	}
	lines = append(lines, mutedStyle.Render("Enter: open  Esc: close"))
	content := padBlock(strings.Join(lines, "\n"), innerWidth, innerHeight)
	return popupStyle.Width(width).Height(height).Render(content)
}

// renderWikiAutocompletePopup draws the edit-mode autocomplete popup showing
// matching note titles/names filtered by the prefix typed after "[[".
func (m *Model) renderWikiAutocompletePopup(width, height int) string {
	innerWidth := max(0, width-popupStyle.GetHorizontalFrameSize())
	innerHeight := max(0, height-popupStyle.GetVerticalFrameSize())
	lines := []string{
		titleStyle.Render("Wiki Link Autocomplete"),
		"",
	}
	limit := max(0, innerHeight-len(lines)-1)
	for i := 0; i < min(limit, len(m.wikiAutocomplete)); i++ {
		target := m.wikiAutocomplete[i]
		label := target.Title
		if strings.TrimSpace(label) == "" {
			label = target.Name
		}
		line := truncate(label, innerWidth)
		if i == m.wikiAutocompleteCursor {
			line = selectedStyle.Render(line)
		}
		lines = append(lines, line)
	}
	lines = append(lines, mutedStyle.Render("Tab/Enter: insert  Esc: close"))
	content := padBlock(strings.Join(lines, "\n"), innerWidth, innerHeight)
	return popupStyle.Width(width).Height(height).Render(content)
}

// handleWikiAutocompleteKey intercepts key presses when the wiki autocomplete
// popup is visible in edit mode. It returns (model, cmd, handled) where handled
// indicates whether the key was consumed by the autocomplete popup.
//
// Supported keys:
//   - Esc: dismiss the popup without inserting anything.
//   - Up/Down: navigate the candidate list.
//   - Enter/Tab: accept the selected candidate, inserting its label and
//     closing brackets into the editor.
//   - Any other key: not handled — falls through to the normal editor handler.
func (m *Model) handleWikiAutocompleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if !m.isOverlay(overlayWikiAutocomplete) {
		return m, nil, false
	}
	switch msg.String() {
	case "esc":
		m.closeOverlay()
		return m, nil, true
	case "up", "k", "ctrl+p":
		if len(m.wikiAutocomplete) > 0 {
			m.wikiAutocompleteCursor = clamp(m.wikiAutocompleteCursor-1, 0, len(m.wikiAutocomplete)-1)
		}
		return m, nil, true
	case "down", "j", "ctrl+n":
		if len(m.wikiAutocomplete) > 0 {
			m.wikiAutocompleteCursor = clamp(m.wikiAutocompleteCursor+1, 0, len(m.wikiAutocomplete)-1)
		}
		return m, nil, true
	case "enter", "tab":
		if len(m.wikiAutocomplete) == 0 {
			m.closeOverlay()
			return m, nil, true
		}
		m.acceptWikiAutocomplete(m.wikiAutocomplete[m.wikiAutocompleteCursor])
		m.closeOverlay()
		return m, nil, true
	default:
		return m, nil, false
	}
}

// maybeTriggerWikiAutocomplete checks whether the editor cursor is positioned
// inside an open [[ token. If so, it builds a filtered list of note candidates
// matching the prefix typed so far and opens the autocomplete popup. If the
// cursor is not inside a [[ context, the popup is dismissed.
//
// This is called after every editor content change in edit mode so the popup
// tracks the user's typing in real time.
func (m *Model) maybeTriggerWikiAutocomplete() {
	prefix, ok := currentWikiPrefix(m.editor.Value(), m.currentEditorCursorOffset())
	if !ok {
		m.closeOverlay()
		return
	}
	if m.searchIndex == nil {
		m.searchIndex = newSearchIndex(m.notesDir)
	}
	if err := m.searchIndex.ensureBuilt(); err != nil {
		return
	}
	targets := m.searchIndex.noteTargets()
	filtered := rankWikiTargets(targets, prefix, m.noteOpenCounts)
	if len(filtered) == 0 {
		m.closeOverlay()
		return
	}
	m.openOverlay(overlayWikiAutocomplete)
	m.wikiAutocomplete = filtered
	m.wikiAutocompleteCursor = clamp(m.wikiAutocompleteCursor, 0, len(filtered)-1)
}

func rankWikiTargets(targets []noteTarget, prefix string, openCounts map[string]int) []noteTarget {
	prefixLower := strings.ToLower(strings.TrimSpace(prefix))
	type candidate struct {
		target noteTarget
		score  int
		opens  int
		title  string
		name   string
	}
	candidates := make([]candidate, 0, len(targets))
	for _, target := range targets {
		title := strings.ToLower(strings.TrimSpace(target.Title))
		name := strings.ToLower(strings.TrimSpace(target.Name))
		score := 0
		if prefixLower != "" {
			if strings.HasPrefix(title, prefixLower) {
				score += 300
			}
			if strings.HasPrefix(name, prefixLower) {
				score += 220
			}
			if strings.Contains(title, prefixLower) {
				score += 120
			}
			if strings.Contains(name, prefixLower) {
				score += 80
			}
			if score == 0 {
				continue
			}
		}
		opens := 0
		if openCounts != nil {
			opens = max(0, openCounts[target.Path])
		}
		score += min(100, opens)
		candidates = append(candidates, candidate{
			target: target,
			score:  score,
			opens:  opens,
			title:  title,
			name:   name,
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		if candidates[i].opens != candidates[j].opens {
			return candidates[i].opens > candidates[j].opens
		}
		if candidates[i].title != candidates[j].title {
			return candidates[i].title < candidates[j].title
		}
		if candidates[i].name != candidates[j].name {
			return candidates[i].name < candidates[j].name
		}
		return strings.ToLower(candidates[i].target.Path) < strings.ToLower(candidates[j].target.Path)
	})
	out := make([]noteTarget, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, c.target)
	}
	return out
}

// currentWikiPrefix scans backward from the cursor position in the editor
// value to find an open [[ token. If found, it returns the text between
// the [[ and the cursor (the prefix the user has typed so far) and true.
//
// Returns ("", false) when:
//   - No [[ is found before the cursor on the current line.
//   - A ]] closing token is encountered first (the link is already closed).
//   - A newline is reached (links do not span lines).
//   - The prefix contains unbalanced brackets.
func currentWikiPrefix(value string, cursor int) (string, bool) {
	runes := []rune(value)
	cursor = clamp(cursor, 0, len(runes))
	start := -1
	for i := cursor - 1; i >= 1; i-- {
		if runes[i-1] == '[' && runes[i] == '[' {
			start = i + 1
			break
		}
		if runes[i-1] == ']' && runes[i] == ']' {
			return "", false
		}
		if runes[i] == '\n' {
			break
		}
	}
	if start == -1 || start > cursor {
		return "", false
	}
	prefix := string(runes[start:cursor])
	if strings.Contains(prefix, "[") || strings.Contains(prefix, "]") {
		return "", false
	}
	return strings.TrimSpace(prefix), true
}

// acceptWikiAutocomplete inserts the selected note's label into the editor,
// replacing any partial prefix typed after [[. If closing brackets ]] are
// not already present after the cursor, they are appended automatically.
// The cursor is positioned immediately after the inserted label (and after
// the closing ]] if they were added).
func (m *Model) acceptWikiAutocomplete(target noteTarget) {
	label := strings.TrimSpace(target.Title)
	if label == "" {
		label = strings.TrimSpace(target.Name)
	}
	if label == "" {
		return
	}
	value := m.editor.Value()
	cursor := m.currentEditorCursorOffset()
	runes := []rune(value)
	cursor = clamp(cursor, 0, len(runes))
	start := -1
	for i := cursor - 1; i >= 1; i-- {
		if runes[i-1] == '[' && runes[i] == '[' {
			start = i + 1
			break
		}
	}
	if start == -1 {
		return
	}
	end := cursor
	needClosing := true
	if end+1 < len(runes) && runes[end] == ']' && runes[end+1] == ']' {
		needClosing = false
		end += 2
	}
	repl := []rune(label)
	if needClosing {
		repl = append(repl, []rune("]]")...)
	}
	updated := make([]rune, 0, len(runes)+len(repl))
	updated = append(updated, runes[:start]...)
	updated = append(updated, repl...)
	updated = append(updated, runes[end:]...)
	newCursor := start + len([]rune(label))
	if needClosing {
		newCursor += 2
	}
	m.setEditorValueAndCursorOffset(string(updated), newCursor)
}
