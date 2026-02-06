# Development Guide

This guide covers local development, project layout, and the core runtime flow.

## Quickstart

Requirements:
- Go 1.21+
- A terminal that supports ANSI colors

Build and run:

```bash
go build -o notes ./cmd/notes
./notes
```

Run without building a binary:

```bash
go run ./cmd/notes
```

Optional logging:
- Set `CLI_NOTES_LOG_LEVEL` to `debug`, `info`, `warn`, or `error` to control runtime log verbosity (default: `info`).

Notes storage:
- On first run (or with `--configure`), a configurator prompts for the notes directory and saves it in `~/.cli-notes/config.json` as `notes_dir`.
- Notes are stored as Markdown files in the configured `notes_dir`.
- The configured directory is created on startup and seeded with `Welcome.md` if empty.

## Project Layout

- `cmd/notes/main.go`: Program entry point. Runs first-time configuration and starts the Bubble Tea app.
- `internal/config/config.go`: Config file pathing, load/save, and notes directory normalization.
- `internal/app/model.go`: Core Bubble Tea model and update loop; handles modes and input routing.
- `internal/app/view.go`: UI layout and rendering (tree pane, right pane, status line).
- `internal/app/tree.go`: Filesystem tree building and selection movement logic.
- `internal/app/search_index.go`: Cached/incremental search index used by the `Ctrl+P` popup.
- `internal/app/render.go`: Debounced markdown rendering and render cache.
- `internal/app/notes.go`: Notes workspace seeding and file operations (create/edit/delete).
- `internal/app/styles.go`: Lip Gloss styles for panes, headers, and status line.
- `internal/app/util.go`: Rendering helpers and small utilities.

## Runtime Flow

1. `main()` ensures config exists (runs configurator when needed), then starts the app.
2. `New()` loads config and ensures the configured notes directory exists.
3. `Update()` handles key input, window resize, and render results.
4. Opening `Ctrl+P` search uses a cached content index; normal create/edit/delete operations update that index incrementally.
5. Selecting a Markdown file triggers a debounced render pipeline.
6. The right pane shows either rendered Markdown, edit mode, or help text.

## Rendering Pipeline

Markdown rendering is intentionally asynchronous and debounced to keep navigation snappy.

- `requestRender()` starts a debounce timer (`renderDebounce`).
- `renderMarkdownCmd()` runs file IO + Glamour rendering off the UI thread.
- `renderCache` stores rendered output keyed by file path + mtime + width bucket.
- A width bucket (`renderWidthBucket`) improves cache reuse across slight terminal resizes.

## Testing

Current entry point:

```bash
go test ./...
```

Search index benchmark suite:

```bash
go test ./internal/app -run '^$' -bench '^BenchmarkSearchIndex$' -benchmem
```

## Troubleshooting

- If the UI looks misaligned, ensure your terminal supports ANSI colors and has enough width.
- If rendered content appears stale, press `r` to refresh the tree and re-render.
- If `Welcome.md` is missing, ensure the configured `notes_dir` exists and is empty, then re-run the app.
