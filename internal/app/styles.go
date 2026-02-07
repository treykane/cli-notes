package app

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
)

var (
	paneStyle      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	popupStyle     = lipgloss.NewStyle().Border(lipgloss.ThickBorder()).Padding(0, 1)
	previewPane    = paneStyle.Copy().BorderForeground(lipgloss.Color("62"))
	editPane       = paneStyle.Copy().BorderForeground(lipgloss.Color("204"))
	selectedStyle  = lipgloss.NewStyle().Reverse(true)
	titleStyle     = lipgloss.NewStyle().Bold(true)
	statusStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Background(lipgloss.Color("24"))
	editStatus     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Background(lipgloss.Color("89"))
	mutedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	previewHeader  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Background(lipgloss.Color("24"))
	editHeader     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Background(lipgloss.Color("89"))
	treeDirName    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("114"))
	treeFileName   = lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
	treeDirTag     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Background(lipgloss.Color("29"))
	treeFileTag    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Background(lipgloss.Color("25"))
	treePinTag     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("16")).Background(lipgloss.Color("220"))
	treeOpenMark   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("121"))
	treeClosedMark = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	selectionText  = lipgloss.NewStyle().Background(lipgloss.Color("255")).Foreground(lipgloss.Color("16"))
)

func applyEditorTheme(editor *textarea.Model) {
	focused, blurred := textarea.DefaultStyles()

	base := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	cursorLine := lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("252"))
	lineNumber := lipgloss.NewStyle().Foreground(lipgloss.Color("218"))
	prompt := lipgloss.NewStyle().Foreground(lipgloss.Color("204"))

	focused.Base = base
	focused.Text = base
	focused.CursorLine = cursorLine
	focused.CursorLineNumber = lineNumber.Bold(true)
	focused.LineNumber = lineNumber
	focused.Prompt = prompt
	focused.Placeholder = mutedStyle

	blurred.Base = base
	blurred.Text = mutedStyle
	blurred.CursorLine = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
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

func applyEditorSelectionVisual(editor *textarea.Model, active bool) {
	// Keep cursor-line visuals stable; selection highlighting is applied to selected text only.
	_ = active
	editor.FocusedStyle.CursorLine = lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("252"))
	editor.FocusedStyle.CursorLineNumber = lipgloss.NewStyle().
		Foreground(lipgloss.Color("218")).
		Bold(true)
}
