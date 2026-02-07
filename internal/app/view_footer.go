package app

import (
	"fmt"
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
		if m.showHelp {
			return []string{
				"Help panel",
				"↑/↓ or j/k scroll",
				"PgUp/PgDn page",
				"Home/End top-bottom",
				"? close",
			}
		}
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
			fmt.Sprintf("%s up", m.primaryActionKey(actionCursorUp, "↑")),
			fmt.Sprintf("%s down", m.primaryActionKey(actionCursorDown, "↓")),
			fmt.Sprintf("%s toggle", m.primaryActionKey(actionExpandToggle, "Enter")),
			fmt.Sprintf("%s collapse", m.primaryActionKey(actionCollapse, "←")),
			fmt.Sprintf("%s top", m.primaryActionKey(actionJumpTop, "G")),
			fmt.Sprintf("%s bottom", m.primaryActionKey(actionJumpBottom, "Shift+G")),
			fmt.Sprintf("%s page-up", m.primaryActionKey(actionPreviewScrollPageUp, "PgUp")),
			fmt.Sprintf("%s page-down", m.primaryActionKey(actionPreviewScrollPageDown, "PgDn")),
			fmt.Sprintf("%s search", m.primaryActionKey(actionSearch, "Ctrl+P")),
			fmt.Sprintf("%s recents", m.primaryActionKey(actionRecent, "Ctrl+O")),
			fmt.Sprintf("%s workspace", m.primaryActionKey(actionWorkspace, "Ctrl+W")),
			fmt.Sprintf("%s edit", m.primaryActionKey(actionEditNote, "E")),
			fmt.Sprintf("%s new", m.primaryActionKey(actionNewNote, "N")),
			fmt.Sprintf("%s folder", m.primaryActionKey(actionNewFolder, "F")),
			fmt.Sprintf("%s rename", m.primaryActionKey(actionRename, "R")),
			fmt.Sprintf("%s move", m.primaryActionKey(actionMove, "M")),
			fmt.Sprintf("%s delete", m.primaryActionKey(actionDelete, "D")),
			fmt.Sprintf("%s refresh", m.primaryActionKey(actionRefresh, "Shift+R")),
			fmt.Sprintf("%s help", m.primaryActionKey(actionHelp, "?")),
		}
		if m.git.isRepo {
			help = append(
				help,
				fmt.Sprintf("%s commit", m.primaryActionKey(actionGitCommit, "C")),
				fmt.Sprintf("%s pull", m.primaryActionKey(actionGitPull, "P")),
				fmt.Sprintf("%s push", m.primaryActionKey(actionGitPush, "Shift+P")),
			)
		}
		help = append(help, fmt.Sprintf("%s quit", m.primaryActionKey(actionQuit, "Q")), "notes --configure")
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

func (m *Model) helpContent() string {
	lines := []string{
		titleStyle.Render("Keyboard Shortcuts"),
		"",
		"Browse",
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionCursorUp, "↑, K"), "Move selection up"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionCursorDown, "↓, J, Ctrl+N"), "Move selection down"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionExpandToggle, "Enter, →, L"), "Expand/collapse folder"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionCollapse, "←, H"), "Collapse folder"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionJumpTop, "G"), "Jump to top"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionJumpBottom, "Shift+G"), "Jump to bottom"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionPreviewScrollPageUp, "PgUp"), "Scroll preview up one page"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionPreviewScrollPageDown, "PgDn"), "Scroll preview down one page"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionPreviewScrollHalfUp, "Ctrl+U"), "Scroll preview up half page"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionPreviewScrollHalfDown, "Ctrl+D"), "Scroll preview down half page"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionSearch, "Ctrl+P"), "Open search popup"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionRecent, "Ctrl+O"), "Open recent-files popup"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionOutline, "O"), "Open heading outline popup"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionWorkspace, "Ctrl+W"), "Open workspace popup"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionExport, "X"), "Export current note (HTML/PDF)"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionWikiLinks, "Shift+L"), "Open wiki-links popup"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionSplitToggle, "Z"), "Toggle split mode"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionSplitFocus, "Tab"), "Toggle split focus"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionNewNote, "N"), "New note"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionNewFolder, "F"), "New folder"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionEditNote, "E"), "Edit note"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionRename, "R"), "Rename selected item"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionMove, "M"), "Move selected item"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionDelete, "D"), "Delete (with confirmation)"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionRefresh, "Ctrl+R, Shift+R"), "Refresh"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionSort, "S"), "Cycle tree sort mode"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionPin, "T"), "Pin/unpin selected item"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionCopyContent, "Y"), "Copy note content"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionCopyPath, "Shift+Y"), "Copy note path"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionHelp, "?"), "Toggle help"),
		fmt.Sprintf("  %-24s %s", m.allActionKeys(actionQuit, "Q, Ctrl+C"), "Quit"),
	}
	if m.git.isRepo {
		lines = append(lines,
			fmt.Sprintf("  %-24s %s", m.allActionKeys(actionGitCommit, "C"), "Git add+commit"),
			fmt.Sprintf("  %-24s %s", m.allActionKeys(actionGitPull, "P"), "Git pull --ff-only"),
			fmt.Sprintf("  %-24s %s", m.allActionKeys(actionGitPush, "Shift+P"), "Git push"),
		)
	}
	lines = append(lines,
		"",
		"CLI",
		"  notes --configure         Re-run configurator",
		"  notes --version           Print build version and commit",
		"",
		"Search Popup",
		"  Type                      Filter folders by name, notes by name/content",
		"  ↑/↓, j/k, Ctrl+P/N        Move search selection",
		"  Enter                     Jump to selected result",
		"  Esc                       Close search popup",
		"",
		"Recent Files Popup",
		"  ↑/↓, j/k                  Move recent selection",
		"  Enter                     Jump to selected recent note",
		"  Esc                       Close popup",
		"",
		"Heading Outline Popup",
		"  o                         Open heading outline for current note",
		"  ↑/↓, j/k                  Move heading selection",
		"  Enter                     Jump preview to heading",
		"  Esc                       Close popup",
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
		"Help Panel Navigation",
		"  ↑/↓, j/k      Scroll line",
		"  PgUp / PgDn   Scroll page",
		"  Home / g      Jump to top",
		"  End / G       Jump to bottom",
		"  ?             Return to app",
	)
	return strings.Join(lines, "\n")
}
