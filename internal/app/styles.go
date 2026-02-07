// styles.go defines the Lipgloss styles used throughout the terminal UI.
//
// The UI uses ANSI 256-color palettes so it renders correctly in virtually all
// modern terminal emulators without requiring true-color support. The palette
// is selected from config via theme_preset (ocean_citrus, sunset, neon_slate).
// Preview and edit modes are distinguished by separate accent tokens.
//
// Tree rows use green for directories and blue for markdown files, with
// distinct badge styles for DIR/MD/PIN/TAGS labels. The selected row is
// rendered with reversed colors for high contrast.
//
// The editor textarea has its own theme (see applyEditorTheme) with a dark
// background cursor line, pink line numbers, and a muted placeholder. Fenced
// code blocks in the editor are styled via editorCodeLine and editorFenceLine
// (applied by editor_highlight.go).
package app

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
	"github.com/treykane/cli-notes/internal/config"
)

var (
	// Semantic palette tokens reused across panes, badges, editor, and footer.
	// Values are set by applyThemePreset during app startup.
	surface     lipgloss.Color
	surfaceAlt  lipgloss.Color
	textPrimary lipgloss.Color
	textMuted   lipgloss.Color

	accentBrowse  lipgloss.Color
	accentEdit    lipgloss.Color
	accentWarn    lipgloss.Color
	accentSuccess lipgloss.Color

	badgeDir  lipgloss.Color
	badgeFile lipgloss.Color
	badgePin  lipgloss.Color
	badgeTags lipgloss.Color

	selectionBg lipgloss.Color
	selectionFg lipgloss.Color

	// paneStyle is the base style for left and right panes: rounded border
	// with horizontal padding. previewPane and editPane derive from this
	// with mode-specific border colors.
	paneStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)

	// popupStyle is the base style for centered overlay popups (search,
	// recent files, outline, workspace, export, wiki links). It uses a
	// thicker border to visually separate the popup from the background.
	popupStyle = lipgloss.NewStyle().Border(lipgloss.ThickBorder()).Padding(0, 1)

	// previewPane styles the right pane border in browse/preview mode.
	previewPane = paneStyle.Copy().BorderForeground(accentBrowse)

	// editPane styles the right pane border in edit mode.
	editPane = paneStyle.Copy().BorderForeground(accentEdit)

	// selectedStyle highlights the currently selected tree row or popup entry
	// by reversing foreground and background colors.
	selectedStyle = lipgloss.NewStyle().Reverse(true)

	// titleStyle renders section headings (popup titles, tree header) in bold.
	titleStyle = lipgloss.NewStyle().Bold(true)

	// statusStyle renders the footer status bar in preview/browse mode:
	// primary text on an ocean background for high contrast.
	statusStyle = lipgloss.NewStyle().Bold(true).Foreground(textPrimary).Background(accentBrowse)

	// editStatus renders the footer status bar in edit mode with the edit accent.
	editStatus = lipgloss.NewStyle().Bold(true).Foreground(textPrimary).Background(accentEdit)

	// mutedStyle renders de-emphasized text (hints, placeholders, empty-state
	// messages) in a mid-gray that recedes visually.
	mutedStyle = lipgloss.NewStyle().Foreground(textMuted)

	// previewHeader is the solid-color bar at the top of the right pane in
	// preview mode, showing the current note's relative path.
	previewHeader = lipgloss.NewStyle().Bold(true).Foreground(textPrimary).Background(accentBrowse)

	// editHeader is the solid-color bar at the top of the right pane in
	// edit mode, using the edit-mode accent color for visual distinction.
	editHeader = lipgloss.NewStyle().Bold(true).Foreground(textPrimary).Background(accentEdit)

	// treeDirName styles directory names in the tree view (bold green).
	treeDirName = lipgloss.NewStyle().Bold(true).Foreground(accentSuccess)

	// treeFileName styles markdown file names in the tree view (light blue).
	treeFileName = lipgloss.NewStyle().Foreground(accentBrowse)

	// treeDirTag is the badge style for the "DIR" label on directory rows
	// (white text on dark green background).
	treeDirTag = lipgloss.NewStyle().Bold(true).Foreground(textPrimary).Background(badgeDir)

	// treeFileTag is the badge style for the "MD" label on markdown file rows
	// (white text on dark blue background).
	treeFileTag = lipgloss.NewStyle().Bold(true).Foreground(textPrimary).Background(badgeFile)

	// treePinTag is the badge style for the "PIN" label on pinned items
	// (black text on yellow background for maximum visibility).
	treePinTag = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("16")).Background(badgePin)

	// treeTagBadge styles the compact "TAGS:..." label shown next to markdown
	// files that have frontmatter tags (light text on muted purple background).
	treeTagBadge = lipgloss.NewStyle().Foreground(textPrimary).Background(badgeTags)

	// treeOpenMark styles the "[-]" marker for expanded directories (green).
	treeOpenMark = lipgloss.NewStyle().Bold(true).Foreground(accentSuccess)

	// treeClosedMark styles the "[+]" marker for collapsed directories (orange).
	treeClosedMark = lipgloss.NewStyle().Bold(true).Foreground(accentWarn)

	// selectionText styles editor text that is currently selected (white
	// background with black text for a clear highlight).
	selectionText = lipgloss.NewStyle().Background(selectionBg).Foreground(selectionFg)

	// editorCodeLine styles lines inside fenced code blocks in the editor
	// (light blue to differentiate code from prose).
	editorCodeLine = lipgloss.NewStyle()

	// editorFenceLine styles the ``` fence delimiters themselves in the
	// editor (gold/amber for easy identification of code block boundaries).
	editorFenceLine = lipgloss.NewStyle()
)

type themePalette struct {
	surface     string
	surfaceAlt  string
	textPrimary string
	textMuted   string

	accentBrowse  string
	accentEdit    string
	accentWarn    string
	accentSuccess string

	badgeDir  string
	badgeFile string
	badgePin  string
	badgeTags string

	selectionBg  string
	selectionFg  string
	editorCodeFg string
}

func init() {
	applyThemePreset(config.ThemePresetOceanCitrus)
}

func paletteForPreset(preset string) themePalette {
	switch config.NormalizeThemePreset(preset) {
	case config.ThemePresetSunset:
		return themePalette{
			surface:       "236",
			surfaceAlt:    "238",
			textPrimary:   "230",
			textMuted:     "180",
			accentBrowse:  "209",
			accentEdit:    "175",
			accentWarn:    "220",
			accentSuccess: "150",
			badgeDir:      "94",
			badgeFile:     "130",
			badgePin:      "220",
			badgeTags:     "131",
			selectionBg:   "224",
			selectionFg:   "52",
			editorCodeFg:  "216",
		}
	case config.ThemePresetNeonSlate:
		return themePalette{
			surface:       "234",
			surfaceAlt:    "236",
			textPrimary:   "255",
			textMuted:     "249",
			accentBrowse:  "51",
			accentEdit:    "141",
			accentWarn:    "227",
			accentSuccess: "118",
			badgeDir:      "22",
			badgeFile:     "24",
			badgePin:      "227",
			badgeTags:     "60",
			selectionBg:   "195",
			selectionFg:   "16",
			editorCodeFg:  "87",
		}
	default:
		return themePalette{
			surface:       "236",
			surfaceAlt:    "238",
			textPrimary:   "255",
			textMuted:     "250",
			accentBrowse:  "39",
			accentEdit:    "44",
			accentWarn:    "214",
			accentSuccess: "114",
			badgeDir:      "29",
			badgeFile:     "25",
			badgePin:      "214",
			badgeTags:     "37",
			selectionBg:   "230",
			selectionFg:   "17",
			editorCodeFg:  "117",
		}
	}
}

// applyThemePreset rebuilds global style tokens for the selected theme.
func applyThemePreset(preset string) {
	p := paletteForPreset(preset)

	surface = lipgloss.Color(p.surface)
	surfaceAlt = lipgloss.Color(p.surfaceAlt)
	textPrimary = lipgloss.Color(p.textPrimary)
	textMuted = lipgloss.Color(p.textMuted)
	accentBrowse = lipgloss.Color(p.accentBrowse)
	accentEdit = lipgloss.Color(p.accentEdit)
	accentWarn = lipgloss.Color(p.accentWarn)
	accentSuccess = lipgloss.Color(p.accentSuccess)
	badgeDir = lipgloss.Color(p.badgeDir)
	badgeFile = lipgloss.Color(p.badgeFile)
	badgePin = lipgloss.Color(p.badgePin)
	badgeTags = lipgloss.Color(p.badgeTags)
	selectionBg = lipgloss.Color(p.selectionBg)
	selectionFg = lipgloss.Color(p.selectionFg)

	previewPane = paneStyle.Copy().BorderForeground(accentBrowse)
	editPane = paneStyle.Copy().BorderForeground(accentEdit)
	statusStyle = lipgloss.NewStyle().Bold(true).Foreground(textPrimary).Background(accentBrowse)
	editStatus = lipgloss.NewStyle().Bold(true).Foreground(textPrimary).Background(accentEdit)
	mutedStyle = lipgloss.NewStyle().Foreground(textMuted)
	previewHeader = lipgloss.NewStyle().Bold(true).Foreground(textPrimary).Background(accentBrowse)
	editHeader = lipgloss.NewStyle().Bold(true).Foreground(textPrimary).Background(accentEdit)
	treeDirName = lipgloss.NewStyle().Bold(true).Foreground(accentSuccess)
	treeFileName = lipgloss.NewStyle().Foreground(accentBrowse)
	treeDirTag = lipgloss.NewStyle().Bold(true).Foreground(textPrimary).Background(badgeDir)
	treeFileTag = lipgloss.NewStyle().Bold(true).Foreground(textPrimary).Background(badgeFile)
	treePinTag = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("16")).Background(badgePin)
	treeTagBadge = lipgloss.NewStyle().Foreground(textPrimary).Background(badgeTags)
	treeOpenMark = lipgloss.NewStyle().Bold(true).Foreground(accentSuccess)
	treeClosedMark = lipgloss.NewStyle().Bold(true).Foreground(accentWarn)
	selectionText = lipgloss.NewStyle().Background(selectionBg).Foreground(selectionFg)
	editorCodeLine = lipgloss.NewStyle().Foreground(lipgloss.Color(p.editorCodeFg))
	editorFenceLine = lipgloss.NewStyle().Foreground(accentWarn)
}

// applyEditorTheme configures the textarea widget's visual appearance to match
// the app's dark theme. It sets distinct styles for focused and blurred states:
//
//   - Focused: light gray text, dark-gray cursor-line highlight, pink line
//     numbers and prompt, and a muted placeholder.
//   - Blurred: dimmed text so the editor visually recedes when not active.
//
// The prompt character "│ " provides a subtle vertical gutter between line
// numbers and content. Line numbers are always shown to help with navigation.
func applyEditorTheme(editor *textarea.Model) {
	focused, blurred := textarea.DefaultStyles()

	base := lipgloss.NewStyle().Foreground(textPrimary)
	cursorLine := lipgloss.NewStyle().Background(surface).Foreground(textPrimary)
	lineNumber := lipgloss.NewStyle().Foreground(accentBrowse)
	prompt := lipgloss.NewStyle().Foreground(accentEdit)

	focused.Base = base
	focused.Text = base
	focused.CursorLine = cursorLine
	focused.CursorLineNumber = lineNumber.Bold(true)
	focused.LineNumber = lineNumber
	focused.Prompt = prompt
	focused.Placeholder = mutedStyle

	blurred.Base = base
	blurred.Text = mutedStyle
	blurred.CursorLine = lipgloss.NewStyle().Foreground(surfaceAlt)
	blurred.CursorLineNumber = lineNumber
	blurred.LineNumber = lineNumber
	blurred.Prompt = prompt
	blurred.Placeholder = mutedStyle

	editor.FocusedStyle = focused
	editor.BlurredStyle = blurred
	editor.Prompt = "│ "
	editor.EndOfBufferCharacter = ' '
	editor.ShowLineNumbers = true
}

// applyEditorSelectionVisual ensures cursor-line styling remains stable
// regardless of whether an editor selection is active. Selection highlighting
// is handled separately by editorViewWithSelectionHighlight (in view.go) which
// wraps selected text spans in selectionText style, so this function simply
// re-applies the default cursor-line and line-number styles to prevent any
// visual drift when the selection anchor is toggled on or off.
func applyEditorSelectionVisual(editor *textarea.Model) {
	// Keep cursor-line visuals stable; selection highlighting is applied to selected text only.
	editor.FocusedStyle.CursorLine = lipgloss.NewStyle().
		Background(surface).
		Foreground(textPrimary)
	editor.FocusedStyle.CursorLineNumber = lipgloss.NewStyle().
		Foreground(accentBrowse).
		Bold(true)
}
