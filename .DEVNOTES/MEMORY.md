# Memory

## Project Overview
- CLI/TUI notes app built with Bubble Tea.
- Stores Markdown files in a user-configured `notes_dir` from `~/.cli-notes/config.json`.
- Entry point: `cmd/notes/main.go` (binary name `notes`).

## Conventions
- Notes live as plain `.md` files; folders are real directories.
- In-app help and README should stay in sync with keybindings.

## Decisions
- 2026-02-07: Completed Core-5 TASKS sweep: (1) browse/footer/help legends now render from active action->key mappings so customized keymaps are reflected in UI hints; (2) Ctrl+P popup now displays search result counters (`N matches`, `M of N`); (3) help panel is now viewport-based and scrollable with dedicated navigation keys while suppressing background tree movement; (4) added `--version` flag output (`notes <version> (<commit>)`) with build metadata injected via CI ldflags; (5) watcher poll interval is now config-driven via `file_watch_interval_seconds` (default 2s, clamped 1..300). Added tests for remapped footer legends, help-scroll key routing, popup count rendering, config interval normalization, and `New()` interval wiring.
- 2026-02-07: Completed browse-key dispatch unification + cache/index hardening pass: (1) browse mode now routes all navigation/popup/expand-collapse/search-hint keys through action-based keybinding dispatch with multi-key defaults and override-replaces-default semantics; (2) search popup key handling now consolidates up/down movement aliases (`up/k/ctrl+p`, `down/j/ctrl+n`) in single match arms; (3) Glamour renderer cache now uses a bounded width-bucket LRU (cap=8) to avoid unbounded growth from frequent terminal resizes; (4) startup tree build path in `New()` now uses `buildTreeWithMetadataCache(..., cachedTagsForPath)` so production tree/tag reads use the metadata cache callback; (5) search index now maintains a sorted path slice alongside docs and removes descendants via binary-searched prefix ranges instead of full-map prefix scans. Added tests for keybinding alias/override behavior, renderer LRU bounds/recency, search popup ctrl bindings parity, and sorted path-index consistency; benchmark suite now includes descendant-removal coverage.
- 2026-02-07: Implemented precision + coverage + performance-tracking follow-up pass: (1) replaced editor mouse row/col mapping with display-width-aware soft-wrap projection (word-wrap + wide-rune aware) for more accurate selection offsets on wrapped lines; (2) added dedicated overlay lifecycle test suite (`overlay_test.go`) covering all overlay transitions, same-overlay idempotency, cleanup behavior, and guard tests that fail when overlay/cleanup coverage lists drift; (3) added mutation pipeline tests (`mutations_test.go`) validating `applyMutationEffects` ordering semantics (invalidate before path ops, remove then upsert), empty-path safety, side-effect execution, and `setCurrentFile` command return behavior; (4) added tree rebuild benchmarks (`BenchmarkTreeRebuild`) for medium/large datasets with cold-cache vs warm-cache sub-benchmarks and expanded CI benchmark tracking/comparison to include both search-index and tree-rebuild suites.
- 2026-02-08: Codebase optimization pass: removed custom `min`/`max` functions in favor of Go 1.21 builtins; removed redundant constant aliases (`renderDebounce`, `maxSearchFileBytes`); removed dead code in `normalizeKeyString`; removed unused `active` parameter from `applyEditorSelectionVisual`; removed `closeTransientPopups` wrapper (inlined `closeOverlay`); added `MaxUndoHistory` cap (1000 entries) on editor undo stack to prevent unbounded memory growth. All tests pass; no functional changes.
- 2026-02-08: Cleaned up TASKS.md: removed all completed items (40+ tasks across High/Medium/Lower priority, Follow-up, and Testing sections); retained 4 open follow-up tasks; added new task categories for code quality/optimization (5 items) and feature ideas (9 items).
- 2026-02-07: Added edit-mode interaction upgrades: left-click drag mouse selection mapped to shared anchor/range selection state, undo/redo (`Ctrl+Z`/`Ctrl+Y`) with typing-burst coalescing (750ms idle split) plus discrete-history boundaries for formatting/link/heading/paste/save, and multiline selection highlighting in the editor view.
- 2026-02-07: Refactored popup/search visibility to a single overlay state machine (`overlayMode`) replacing multiple popup booleans; centralized overlay open/close cleanup (including search and wiki-autocomplete teardown) and switched key/view footer routing to overlay-based dispatch.
- 2026-02-07: Consolidated repeated post-mutation side effects into `applyMutationEffects` (search-index upsert/remove/invalidate, tree rebuild, git refresh, state save, render-cache clear, current-file refresh) and applied it across refresh/watcher/CRUD flows.
- 2026-02-07: Added a tree frontmatter metadata cache keyed by path+mtime with invalidation/remap hooks for edit/delete/refresh/rename/move/workspace switch to reduce repeated tag parsing during tree rebuilds.
- 2026-02-07: Split large view responsibilities into focused files (`view_root.go`, `view_tree.go`, `view_right.go`, `view_footer.go`, `view_overlays.go`) to reduce `view.go` complexity and isolate rendering concerns.
- 2026-02-07: Added config-driven UI theme presets via `theme_preset` (`ocean_citrus`, `sunset`, `neon_slate`) with normalization/fallback to `ocean_citrus`; app startup now applies the selected preset to pane, footer, tree, and editor-adjacent styles.
- 2026-02-07: Added explicit browse-mode preview scroll keybindings with configurable actions (`preview.scroll.page_up`, `preview.scroll.page_down`, `preview.scroll.half_up`, `preview.scroll.half_down`) defaulting to `PgUp`/`PgDn`/`Ctrl+U`/`Ctrl+D`; scroll offsets now persist for both primary and split secondary panes when using keyboard scrolling.
- 2026-02-07: Added cross-platform CI matrix workflow (`.github/workflows/ci.yml`) running `go test ./...` on macOS, Linux, and Windows runners to catch platform-specific regressions early.
- 2026-02-07: Added cross-filesystem move fallback for note/folder moves: when `os.Rename` fails with cross-device (`EXDEV`) errors, the app now performs copy-then-delete with destination cleanup on partial failure.
- 2026-02-07: Updated created-time sorting semantics: macOS still uses true birth time, Linux now attempts `statx` `STATX_BTIME`, and all other platforms explicitly fall back to modification time (no ctime proxy).
- 2026-02-07: Added integration CRUD lifecycle tests (create/edit/save/delete) validating filesystem, tree state, and search index consistency, plus expanded formatting toggle edge-case tests (nested, overlap, boundaries, empty selection).
- 2026-02-07: Reworked UI styling around a semantic Ocean+Citrus ANSI palette (shared tokens for panes, badges, editor, selection, and footer) and replaced the single-line footer with an adaptive 2-3 row footer that packs grouped keys/context/status segments with overflow ellipsis.
- 2026-02-08: Added workspace-scoped tree sort persistence via config `tree_sort_by_workspace` (`notes_dir` key -> sort mode), with legacy `tree_sort` retained as fallback/default.
- 2026-02-08: Extended per-workspace state to persist pane-specific preview offsets (`primary_preview_offset` + `secondary_preview_offset`) while keeping legacy `preview_offset` migration compatibility.
- 2026-02-08: Upgraded edit-mode wiki autocomplete ordering to rank by exact-prefix match strength plus persisted per-note open-frequency counts in workspace state.
- 2026-02-08: Completed comprehensive inline documentation pass across all source files. Added godoc-style package comments, function/type/field comments, algorithm rationale, and edge-case notes. Fixed `gofmt` formatting across 14 files that had drifted. Documented the three platform-specific `file_time_*.go` files (darwin/unix/other) and added a file-level doc comment to `state.go`. All changes are documentation-only; no functional code was modified. Build, vet, and all tests pass cleanly.
- 2026-02-07: Added YAML frontmatter parsing for `title`/`date`/`category`/`tags`; tags now render in tree rows and `Ctrl+P` search supports `tag:<name>` filtering plus metadata-aware matches.
- 2026-02-07: Added named multi-workspace config (`workspaces` + `active_workspace`) with migration from legacy `notes_dir`; browse-mode workspace popup (`Ctrl+W`) switches roots and rebinds state/search/render caches.
- 2026-02-07: Added configurable keybindings via `config.json` `keybindings` map plus optional `keymap_file` (default `~/.cli-notes/keymap.json`), with conflict warnings and safe fallback to defaults.
- 2026-02-07: Added export popup (`x`) for current markdown note: HTML export writes alongside source note; PDF export uses Pandoc when available and otherwise shows install guidance.
- 2026-02-07: Added wiki-link tooling: `[[...]]` parsing (fence-aware), browse-mode wiki links popup (`Shift+L`) with title-first then filename fallback resolution, and edit-mode `[[` autocomplete.
- 2026-02-07: Added basic fenced-code visual highlighting in edit mode and split-pane mode (`z`) for side-by-side two-note viewing with focus toggle (`Tab`).
- 2026-02-06: Added poll-based filesystem watching for `notes_dir` external changes, with automatic tree rebuild, search-index invalidation, and markdown render-cache invalidation.
- 2026-02-06: Added persistent app state in `<notes_dir>/.cli-notes/state.json` for recent files, pinned paths, and per-note position memory (preview offset + edit cursor).
- 2026-02-06: Added browse-mode productivity popups: `Ctrl+O` recent files quick-jump and `o` heading outline jump-to-section.
- 2026-02-06: Added browse-mode pin toggle (`t`) and pin-aware tree sorting (pinned entries first within each folder across all sort modes).
- 2026-02-06: Added editor productivity shortcuts: `Alt+X` strikethrough toggle, `Ctrl+K` markdown link insertion/wrapping, and `Ctrl+1..3` heading toggles for the current line.
- 2026-02-06: Added footer note metrics (`W/C/L`) derived from raw note content in both preview and edit modes.
- 2026-02-06: Added browse-mode `s` sort cycling (name/modified/size/created) with persisted config in `tree_sort`.
- 2026-02-06: Added edit-mode draft auto-save to `<notes_dir>/.cli-notes/.drafts/` plus startup recovery prompt mode (`y` recover / `n` discard).
- 2026-02-06: Added clipboard actions: browse-mode `y` copies current note content, browse-mode `Y` copies current note path, and edit-mode `Ctrl+V` pastes clipboard text.
- 2026-02-06: Added note template picker flow for new notes using files from `templates_dir` (default `~/.cli-notes/templates`) with fallback to the default built-in template.
- 2026-02-06: Tree and search traversal now skip the internal managed directory `<notes_dir>/.cli-notes` so drafts are not shown or indexed.
- 2026-02-06: Added browse-mode rename (`r`) and move (`m`) workflows using the existing text input widget, including tree rebuild + search-index updates + `currentFile` path remapping after path changes.
- 2026-02-06: Added delete confirmation mode so `d` now prompts for `y/n` before deleting notes/folders, preventing immediate accidental removal.
- 2026-02-06: Added git-aware browse actions when `notes_dir` is in a git repo: `c` prompts for commit message then runs `git add -A && git commit -m`, `p` runs `git pull --ff-only`, `P` runs `git push`; footer now shows branch/sync/dirty status.
- 2026-02-06: Refined edit-mode selection visuals to highlight only selected text (light background, dark text) instead of the full cursor line.
- 2026-02-06: Improved edit-mode selection feedback: Shift-based selection handling now matches both shifted key types and key strings, and active selection updates status with selected character count.
- 2026-02-06: Formatting shortcuts in edit mode now toggle: if the exact selection/word is already wrapped by that formatter, the surrounding markers are removed; otherwise markers are added. This preserves nested formatting by only affecting the formatter immediately surrounding the target range.
- 2026-02-06: Added an app-level editor selection API (anchor + range) used by formatting shortcuts: `Alt+S` sets/clears the anchor, and `Ctrl+B`/`Alt+I`/`Ctrl+U` now wrap selection, or current word when no selection, with marker-insertion fallback.
- 2026-02-06: Added edit-mode formatting shortcuts: `Ctrl+B` inserts `** **`, `Alt+I` inserts `* *` (avoids `Ctrl+I`/Tab collision), and `Ctrl+U` inserts `<u></u>` with the cursor placed between markers.
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
- 2026-02-05: Extended `Ctrl+P` search to match Markdown note body content in addition to note/folder names (content search applies to `.md` files up to 1 MiB).
- 2026-02-05: Added a cached in-memory search index for `Ctrl+P` (name + markdown content), with incremental updates on note/folder create/edit/delete and explicit invalidation on manual refresh.
- 2026-02-05: Added more Vim-friendly navigation in browse mode (`h/l`, `j/k`, `g/G`, and `Ctrl+N`).
- 2026-02-05: Brought app color styling into the edit textarea (prompt, line numbers, cursor line, and muted placeholders).
- 2026-02-05: Expanded injected-input classification for debug mode to label ignored sequences (OSC/CSI/escape/control) via `CLI_NOTES_DEBUG_INPUT`.
- 2026-02-05: Normalized note writes to always end with exactly one trailing newline (applies to welcome note creation, new notes, and edits).
- 2026-02-05: Added mode-aware right-pane theming so preview and edit mode use distinct accents/badges and are visually easy to distinguish.
- 2026-02-05: Removed explicit "PREVIEW"/"EDIT MODE" labels from the right-pane header; the UI now uses path-only headers with mode-specific colors.
- 2026-02-05: Restyled the right-pane filename row as a solid-color status bar and refreshed tree row styling (DIR/MD tags plus colorized +/- folder markers).
- 2026-02-05: Updated tree palette to blue markdown tags/files and green folder tags/names, and made selection highlighting span the full tree row width.
- 2026-02-05: Updated selected tree rows to render unstyled row text before highlight so the selection color clearly covers both the full row and its text.
- 2026-02-05: Switched UI string truncation to ANSI-aware truncation to avoid clipping styled tree rows.
- 2026-02-05: Added first-run configurator and `--configure` flag; notes directory is now stored in `~/.cli-notes/config.json` and surfaced in in-app help.
- 2026-02-06: Increased bottom status/footer contrast (solid mode-aware background + bold text) and added left-padding so the line reads as a persistent info bar even in split panes.
- 2026-02-05: Clamped the main pane row to `height-1` before rendering footer status so border/padding growth cannot clip the bottom info line.
- 2026-02-06: Added unit tests in `internal/app` for tree building behavior (sorting, depth, and expansion) and markdown render cache behavior (cache hit vs async render path).
- 2026-02-06: Improved error handling/logging with a shared `internal/logging` package, contextual error wrapping in config/search/setup paths, and centralized app error logs for note operations, rendering, and search indexing.
- 2026-02-06: Added `internal/app/search_index_benchmark_test.go` with benchmark coverage for search-index cold builds and warm queries across small and large datasets to guard search performance regressions.
- 2026-02-06: Added CI benchmark tracking via `.github/workflows/search-index-benchmarks.yml`, including PR baseline-vs-current regression checks (20% threshold) and artifact uploads on PR/push/scheduled runs.

## Useful Commands
- `go build -o notes ./cmd/notes`

## References
- 
