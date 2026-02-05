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
- 2026-02-05: Replaced inline search with a `Ctrl+P` popup that filters notes/folders by name and jumps directly to the selected result.
- 2026-02-05: Added more Vim-friendly navigation in browse mode (`h/l`, `j/k`, `g/G`, and `Ctrl+N`).
- 2026-02-05: Brought app color styling into the edit textarea (prompt, line numbers, cursor line, and muted placeholders).
- 2026-02-05: Expanded injected-input classification for debug mode to label ignored sequences (OSC/CSI/escape/control) via `CLI_NOTES_DEBUG_INPUT`.
- 2026-02-05: Normalized note writes to always end with exactly one trailing newline (applies to welcome note creation, new notes, and edits).
- 2026-02-05: Added mode-aware right-pane theming so preview and edit mode use distinct accents/badges and are visually easy to distinguish.
- 2026-02-05: Removed explicit "PREVIEW"/"EDIT MODE" labels from the right-pane header; the UI now uses path-only headers with mode-specific colors.
- 2026-02-05: Restyled the right-pane filename row as a solid-color status bar and refreshed tree row styling (DIR/MD tags plus colorized +/- folder markers).
- 2026-02-05: Updated tree palette to blue markdown tags/files and green folder tags/names, and made selection highlighting span the full tree row width.
- 2026-02-05: Updated selected tree rows to render unstyled row text before highlight so the selection color clearly covers both the full row and its text.

## Useful Commands
- `go build -o notes ./cmd/notes`

## References
- 
