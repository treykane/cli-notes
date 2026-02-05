package app

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
)

var (
	paneStyle     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	popupStyle    = lipgloss.NewStyle().Border(lipgloss.ThickBorder()).Padding(0, 1)
	previewPane   = paneStyle.Copy().BorderForeground(lipgloss.Color("62"))
	editPane      = paneStyle.Copy().BorderForeground(lipgloss.Color("204"))
	selectedStyle = lipgloss.NewStyle().Reverse(true)
	titleStyle    = lipgloss.NewStyle().Bold(true)
	statusStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	editStatus    = lipgloss.NewStyle().Foreground(lipgloss.Color("211"))
	mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

func applyEditorTheme(editor *textarea.Model) {
	focused, blurred := textarea.DefaultStyles()

	base := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	cursorLine := lipgloss.NewStyle().Background(lipgloss.Color("53")).Foreground(lipgloss.Color("252"))
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
