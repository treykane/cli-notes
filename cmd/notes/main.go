package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/treykane/cli-notes/internal/app"
)

func main() {
	renderLight := flag.Bool("render-light", false, "render markdown using a light theme")
	flag.Parse()

	if *renderLight {
		_ = os.Setenv("CLI_NOTES_GLAMOUR_STYLE", "light")
	}

	m, err := app.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
