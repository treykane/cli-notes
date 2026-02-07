// Package main is the entry point for the cli-notes application.
//
// It handles CLI flag parsing, first-run configuration, and launching
// the Bubble Tea TUI program. On first run (or when --configure is passed),
// the user is prompted to choose a notes directory before the UI starts.
//
// Flags:
//
//	--render-light  Force light-theme markdown rendering (sets CLI_NOTES_GLAMOUR_STYLE=light).
//	--configure     Re-run the interactive configurator to change the notes directory.
//
// Environment:
//
//	CLI_NOTES_LOG_LEVEL   Controls log verbosity (debug, info, warn, error). Default: info.
//	CLI_NOTES_GLAMOUR_STYLE  Overrides the Glamour markdown rendering style (dark, light, notty, auto).
//	CLI_NOTES_DEBUG_INPUT    When set, surfaces ignored terminal escape sequences in the status bar.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/treykane/cli-notes/internal/app"
	"github.com/treykane/cli-notes/internal/config"
	"github.com/treykane/cli-notes/internal/logging"
)

// log is the structured logger for the main package, tagged with component="main".
var log = logging.New("main")

// main parses flags, ensures configuration exists, and starts the TUI.
//
// Startup sequence:
//  1. Parse CLI flags (--render-light, --configure).
//  2. Check whether a config file exists at ~/.cli-notes/config.json.
//  3. If missing or --configure was passed, run the interactive configurator.
//  4. Initialize the app Model (loads config, builds tree, sets up search index).
//  5. Launch Bubble Tea in alt-screen mode.
func main() {
	renderLight := flag.Bool("render-light", false, "render markdown using a light theme")
	configure := flag.Bool("configure", false, "run configurator to choose the notes directory")
	flag.Parse()

	if *renderLight {
		_ = os.Setenv("CLI_NOTES_GLAMOUR_STYLE", "light")
	}

	configured, err := config.Exists()
	if err != nil {
		log.Error("check config", "error", err)
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if *configure || !configured {
		if err := runConfigurator(os.Stdin, os.Stdout); err != nil {
			log.Error("run configurator", "error", err)
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	}

	m, err := app.New()
	if err != nil {
		if errors.Is(err, config.ErrNotConfigured) {
			log.Warn("app not configured")
			fmt.Fprintln(os.Stderr, "error: app is not configured; run notes --configure")
		} else {
			log.Error("initialize app", "error", err)
			fmt.Fprintln(os.Stderr, "error:", err)
		}
		os.Exit(1)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Error("run bubbletea program", "error", err)
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// runConfigurator prompts the user to choose a notes directory and persists
// the result to ~/.cli-notes/config.json.
//
// It reads from in and writes prompts to out, making it testable with mock
// readers/writers. The user can accept the default directory (~/notes) by
// pressing Enter, or type a custom path. The path is normalized (~ expanded,
// made absolute) and validated before saving. If the directory doesn't exist,
// it is created automatically.
//
// The configurator loops until a valid directory is provided or EOF is reached.
func runConfigurator(in io.Reader, out io.Writer) error {
	defaultDir, err := config.DefaultNotesDir()
	if err != nil {
		return fmt.Errorf("resolve default notes directory: %w", err)
	}

	reader := bufio.NewReader(in)
	fmt.Fprintln(out, "CLI Notes Configurator")
	fmt.Fprintln(out, "Set the directory where your markdown notes will be stored.")

	for {
		fmt.Fprintf(out, "Notes directory [%s]: ", defaultDir)
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("read notes directory input: %w", err)
		}

		value := strings.TrimSpace(line)
		if value == "" {
			value = defaultDir
		}

		notesDir, normErr := config.NormalizeNotesDir(value)
		if normErr != nil {
			fmt.Fprintf(out, "Invalid directory: %v\n", normErr)
			if errors.Is(err, io.EOF) {
				return normErr
			}
			continue
		}

		if mkErr := os.MkdirAll(notesDir, 0o755); mkErr != nil {
			fmt.Fprintf(out, "Unable to create directory: %v\n", mkErr)
			if errors.Is(err, io.EOF) {
				return mkErr
			}
			continue
		}

		if saveErr := config.Save(config.Config{NotesDir: notesDir}); saveErr != nil {
			return fmt.Errorf("save config: %w", saveErr)
		}

		fmt.Fprintf(out, "Saved configuration: notes_dir=%s\n", notesDir)
		return nil
	}
}
