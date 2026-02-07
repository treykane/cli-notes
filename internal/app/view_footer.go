package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func (m *Model) renderStatus(width, rows int) string {
	statusRows, _ := m.buildStatusRows(width, rows)
	style := statusStyle
	if m.mode == modeEditNote {
		style = editStatus
	}
	for len(statusRows) < rows {
		statusRows = append(statusRows, "")
	}

	rendered := make([]string, 0, len(statusRows))
	for _, line := range statusRows {
		line = " " + truncate(line, max(0, width-1))
		rendered = append(rendered, style.Width(width).Render(line))
	}
	return strings.Join(rendered, "\n")
}

func (m *Model) buildStatusRows(width, rowLimit int) ([]string, bool) {
	if width <= 0 || rowLimit <= 0 {
		return nil, true
	}

	help := m.statusHelpSegments()
	context := m.statusContextSegments()
	status := m.statusMessageSegment()

	segments := make([]string, 0, len(help)+len(context)+2)
	if len(help) > 0 {
		segments = append(segments, "Keys: "+help[0])
		segments = append(segments, help[1:]...)
	}
	if len(context) > 0 {
		segments = append(segments, "Context: "+context[0])
		segments = append(segments, context[1:]...)
	}
	if status != "" {
		segments = append(segments, "Status: "+status)
	}

	rows := make([]string, 1, rowLimit)
	rowIndex := 0
	fit := true
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		segment := seg
		if lipgloss.Width(segment) > width {
			segment = truncateWithEllipsis(segment, width)
		}

		candidate := segment
		if rows[rowIndex] != "" {
			candidate = rows[rowIndex] + " | " + segment
		}
		if lipgloss.Width(candidate) <= width {
			rows[rowIndex] = candidate
			continue
		}
		if rowIndex+1 < rowLimit {
			rowIndex++
			rows = append(rows, segment)
			continue
		}

		fit = false
		if rows[rowIndex] == "" {
			rows[rowIndex] = truncateWithEllipsis(segment, width)
		} else {
			rows[rowIndex] = truncateWithEllipsis(rows[rowIndex]+" | "+segment, width)
		}
		break
	}
	return rows, fit
}

func truncateWithEllipsis(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	if width == 1 {
		return "…"
	}
	return ansi.Truncate(value, width-1, "") + "…"
}

func (m *Model) statusHelpSegments() []string {
	switch m.mode {
	case modeEditNote:
		return []string{
			"Ctrl+S save",
			"Ctrl+Z undo",
			"Ctrl+Y redo",
			"Shift+Arrows select",
			"Mouse drag select",
			"Alt+S anchor",
			"Ctrl+B bold",
			"Alt+I italic",
			"Ctrl+U underline",
			"Alt+X strike",
			"Ctrl+K link",
			"Ctrl+1..3 heading",
			"Ctrl+V paste",
			"Esc cancel",
		}
	case modeNewNote, modeNewFolder, modeRenameItem, modeMoveItem, modeGitCommit:
		return []string{"Enter/Ctrl+S save", "Esc cancel"}
	case modeTemplatePicker:
		return []string{"Template picker", "↑/↓ move", "Enter choose", "Esc cancel"}
	case modeDraftRecovery:
		return []string{"Draft recovery", "y recover", "n discard", "Esc skip all"}
	case modeConfirmDelete:
		return []string{"y confirm delete", "n/Esc cancel"}
	default:
		switch m.overlay {
		case overlaySearch:
			return []string{"Search popup", "type", "↑/↓ move", "Enter jump", "Esc cancel"}
		case overlayRecent:
			return []string{"Recent popup", "↑/↓ move", "Enter jump", "Esc cancel"}
		case overlayOutline:
			return []string{"Outline popup", "↑/↓ move", "Enter jump", "Esc cancel"}
		case overlayWorkspace:
			return []string{"Workspace popup", "↑/↓ move", "Enter switch", "Esc cancel"}
		case overlayExport:
			return []string{"Export popup", "↑/↓ move", "Enter export", "Esc cancel"}
		case overlayWikiLinks:
			return []string{"Wiki links popup", "↑/↓ move", "Enter jump", "Esc cancel"}
		case overlayWikiAutocomplete:
			return []string{"Wiki autocomplete", "↑/↓ move", "Tab/Enter insert", "Esc close"}
		}
		help := []string{
			"↑/↓ or k/j move",
			"Enter/→/l toggle",
			"←/h collapse",
			"g/G top-bottom",
			"PgUp/PgDn preview",
			"Ctrl+U/D half-preview",
			"Ctrl+P search",
			"n new",
			"f folder",
			"e edit",
			"r rename",
			"m move",
			"d delete",
			"Shift+R refresh",
			"s sort",
			"t pin",
			"Ctrl+O recents",
			"o outline",
			"Ctrl+W workspaces",
			"x export",
			"Shift+L wiki",
			"z split",
			"Tab split-focus",
			"y copy content",
			"Y copy path",
		}
		if m.git.isRepo {
			help = append(help, "c commit", "p pull", "P push")
		}
		help = append(help, "? help", "q quit", "notes --configure")
		return help
	}
}

func (m *Model) statusContextSegments() []string {
	parts := make([]string, 0, 2)
	if (m.mode == modeBrowse || m.mode == modeEditNote) && m.currentFile != "" {
		if metrics := m.noteMetricsSummary(); metrics != "" {
			parts = append(parts, metrics)
		}
	}
	if git := m.gitFooterSummary(); git != "" {
		parts = append(parts, git)
	}
	return parts
}

func (m *Model) statusMessageSegment() string {
	return strings.TrimSpace(m.status)
}

func (m *Model) renderHelp(width, height int) string {
	lines := []string{
		titleStyle.Render("Keyboard Shortcuts"),
		"",
		"Browse",
		"  ↑/↓, k/j, Ctrl+N          Move selection",
		"  Enter, →, l               Expand/collapse folder",
		"  ←, h                      Collapse folder",
		"  g / G                     Jump to top / bottom",
		"  PgUp / PgDn               Scroll preview up / down one page",
		"  Ctrl+U / Ctrl+D           Scroll preview up / down half page",
		"  Ctrl+P                    Open search popup",
		"  Ctrl+O                    Open recent-files popup",
		"  o                         Open heading outline popup",
		"  Ctrl+W                    Open workspace popup",
		"  x                         Export current note (HTML/PDF)",
		"  Shift+L                   Open wiki-links popup",
		"  z                         Toggle split mode",
		"  Tab                       Toggle split focus",
		"  n                         New note",
		"  f                         New folder",
		"  e                         Edit note",
		"  r                         Rename selected item",
		"  m                         Move selected item",
		"  d                         Delete (with confirmation)",
		"  Shift+R / Ctrl+R          Refresh",
		"  s                         Cycle tree sort mode",
		"  t                         Pin/unpin selected item",
		"  y / Y                     Copy note content / path",
		"  ?                         Toggle help",
		"  q or Ctrl+C               Quit",
	}
	if m.git.isRepo {
		lines = append(lines,
			"  c                         Git add+commit",
			"  p                         Git pull --ff-only",
			"  P                         Git push",
		)
	}
	lines = append(lines,
		"",
		"CLI",
		"  notes --configure         Re-run configurator",
		"",
		"Search Popup",
		"  Type                Filter folders by name, notes by name/content",
		"  ↑/↓, j/k            Move search selection",
		"  Enter               Jump to selected result",
		"  Esc                 Close search popup",
		"",
		"Recent Files Popup",
		"  ↑/↓, j/k            Move recent selection",
		"  Enter               Jump to selected recent note",
		"  Esc                 Close popup",
		"",
		"Heading Outline Popup",
		"  o                   Open heading outline for current note",
		"  ↑/↓, j/k            Move heading selection",
		"  Enter               Jump preview to heading",
		"  Esc                 Close popup",
		"",
		"New Note/Folder",
		"  Enter or Ctrl+S  Save",
		"  Esc              Cancel",
		"",
		"Rename/Move/Git Commit",
		"  Enter or Ctrl+S  Save",
		"  Esc              Cancel",
		"",
		"Template Picker",
		"  ↑/↓, j/k         Move template selection",
		"  Enter            Choose template",
		"  Esc              Cancel new-note flow",
		"",
		"Draft Recovery",
		"  y                Recover draft",
		"  n                Discard draft",
		"  Esc              Skip remaining drafts",
		"",
		"Delete Confirmation",
		"  y                Confirm delete",
		"  n or Esc         Cancel delete",
		"",
		"Edit Note",
		"  Ctrl+S         Save",
		"  Ctrl+Z         Undo",
		"  Ctrl+Y         Redo",
		"  Shift+Arrows   Extend selection",
		"  Mouse drag     Select text",
		"  Shift+Home/End Extend selection to line boundaries",
		"  Alt+S          Set/clear selection anchor",
		"  Ctrl+B         Toggle **bold** on selection/word",
		"  Alt+I          Toggle *italic* on selection/word",
		"  Ctrl+U         Toggle <u>underline</u> on selection/word",
		"  Alt+X          Toggle ~~strikethrough~~ on selection/word",
		"  Ctrl+K         Insert [text](url) link template",
		"  Ctrl+1..3      Toggle # / ## / ### heading on current line",
		"  Ctrl+V         Paste clipboard text",
		"  Esc            Cancel",
		"",
		"Press ? to return.",
	)

	visible := min(height, len(lines))
	out := make([]string, 0, visible)
	for i := 0; i < visible; i++ {
		out = append(out, truncate(lines[i], width))
	}
	return strings.Join(out, "\n")
}
