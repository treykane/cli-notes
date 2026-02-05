# Memory

## Project Overview
- CLI/TUI notes app built with Bubble Tea.
- Stores Markdown files in `~/notes` (created on first run).
- Entry point: `cmd/notes/main.go` (binary name `notes`).

## Conventions
- Notes live as plain `.md` files; folders are real directories.
- In-app help and README should stay in sync with keybindings.

## Decisions
- 2026-02-04: Added debounced + async Markdown preview rendering with caching (path + mtime + width) to keep the UI responsive while navigating notes.
- 2026-02-04: Documented full keybindings (browse, new note/folder, edit) in README and in-app welcome note.
- 2026-02-04: Refactored UI logic into `internal/app` to keep `cmd/notes/main.go` minimal and improve maintainability.
- 2026-02-04: Added an in-app help panel (toggle with `?`) and mode-specific status hints so keybindings are visible in the UI.
- 2026-02-05: Added `docs/DEVELOPMENT.md` with a developer guide (setup, layout, and rendering flow) and linked it from README.
- 2026-02-05: Added `.DEVNOTES/CONTRIBUTING.md` with code style and PR guidelines; linked it from README.
- 2026-02-05: Defaulted Glamour rendering to the `dark` style (configurable via `CLI_NOTES_GLAMOUR_STYLE` or `GLAMOUR_STYLE`) to avoid OSC background color queries leaking into note edits.
- 2026-02-05: Added `--render-light` CLI flag to force light markdown rendering (sets `CLI_NOTES_GLAMOUR_STYLE=light`).
- 2026-02-05: Filtered OSC background color response sequences from editor input to prevent stray `1;rgb:...` text on first edit.
- 2026-02-05: Extended OSC background response filtering to note name and folder name inputs.
- 2026-02-05: Hardened OSC background response detection to catch variants that include ESC or `]11;rgb:` prefixes.
- 2026-02-05: Added `CLI_NOTES_DEBUG_INPUT` to surface ignored input sequences in the status line.
- 2026-02-05: Relaxed OSC background response parsing to ignore variants with trailing characters.

## Useful Commands
- `go build -o notes ./cmd/notes`

## References
- 
