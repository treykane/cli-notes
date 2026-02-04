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

## Useful Commands
- `go build -o notes ./cmd/notes`

## References
- 
