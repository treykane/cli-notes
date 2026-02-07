package app

import (
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type wikiLink struct {
	Label    string
	Target   string
	Resolved bool
}

var wikiLinkPattern = regexp.MustCompile(`\[\[([^\[\]]+)\]\]`)

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
	m.closeTransientPopups()
	m.showWikiLinksPopup = true
	m.wikiLinks = wikiRows
	m.wikiLinkCursor = 0
	m.status = "Wiki links: Enter to open, Esc to close"
}

func (m *Model) handleWikiLinksPopupKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.shouldIgnoreInput(msg) {
		return m, nil
	}
	switch msg.String() {
	case "esc":
		m.showWikiLinksPopup = false
		m.status = "Wiki links closed"
		return m, nil
	case "up", "k", "ctrl+p":
		m.wikiLinkCursor = clamp(m.wikiLinkCursor-1, 0, len(m.wikiLinks)-1)
		return m, nil
	case "down", "j", "ctrl+n":
		m.wikiLinkCursor = clamp(m.wikiLinkCursor+1, 0, len(m.wikiLinks)-1)
		return m, nil
	case "enter":
		if len(m.wikiLinks) == 0 {
			return m, nil
		}
		link := m.wikiLinks[m.wikiLinkCursor]
		if !link.Resolved || link.Target == "" {
			m.status = "Unresolved wiki link: " + link.Label
			return m, nil
		}
		m.showWikiLinksPopup = false
		m.expandParentDirs(link.Target)
		m.rebuildTreeKeep(link.Target)
		m.status = "Opened wiki link: " + link.Label
		return m, m.setFocusedFile(link.Target)
	default:
		return m, nil
	}
}

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

func (m *Model) handleWikiAutocompleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	if !m.showWikiAutocomplete {
		return m, nil, false
	}
	switch msg.String() {
	case "esc":
		m.showWikiAutocomplete = false
		m.wikiAutocomplete = nil
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
			m.showWikiAutocomplete = false
			return m, nil, true
		}
		m.acceptWikiAutocomplete(m.wikiAutocomplete[m.wikiAutocompleteCursor])
		m.showWikiAutocomplete = false
		m.wikiAutocomplete = nil
		return m, nil, true
	default:
		return m, nil, false
	}
}

func (m *Model) maybeTriggerWikiAutocomplete() {
	prefix, ok := currentWikiPrefix(m.editor.Value(), m.currentEditorCursorOffset())
	if !ok {
		m.showWikiAutocomplete = false
		m.wikiAutocomplete = nil
		m.wikiAutocompleteCursor = 0
		return
	}
	if m.searchIndex == nil {
		m.searchIndex = newSearchIndex(m.notesDir)
	}
	if err := m.searchIndex.ensureBuilt(); err != nil {
		return
	}
	targets := m.searchIndex.noteTargets()
	filtered := make([]noteTarget, 0, len(targets))
	prefixLower := strings.ToLower(strings.TrimSpace(prefix))
	for _, target := range targets {
		title := strings.ToLower(strings.TrimSpace(target.Title))
		name := strings.ToLower(strings.TrimSpace(target.Name))
		if prefixLower == "" || strings.HasPrefix(title, prefixLower) || strings.HasPrefix(name, prefixLower) {
			filtered = append(filtered, target)
		}
	}
	if len(filtered) == 0 {
		m.showWikiAutocomplete = false
		m.wikiAutocomplete = nil
		m.wikiAutocompleteCursor = 0
		return
	}
	m.showWikiAutocomplete = true
	m.wikiAutocomplete = filtered
	m.wikiAutocompleteCursor = clamp(m.wikiAutocompleteCursor, 0, len(filtered)-1)
}

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
