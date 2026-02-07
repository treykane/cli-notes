# Tasks

## Recently Completed

- [x] **Comprehensive inline documentation**: Added godoc-style package comments, function/type/field documentation, algorithm rationale, and edge-case notes across all source files. Documented platform-specific `file_time_*.go` files and added file-level doc to `state.go`. Fixed `gofmt` formatting across 14 files. All changes documentation-only; build, vet, and tests pass cleanly.

## In Progress

- [ ] Add mouse-driven editor selection using Bubble Tea mouse events and map drag gestures to the shared selection API.
- [ ] Add Undo/Redo support in edit mode (keyboard shortcuts + predictable history boundaries for format toggles, typing bursts, and save).
- [ ] Add multiline visual selection highlighting in edit mode (currently text-only highlight is limited to single-line selections).

## High Priority — Core Note Management

- [x] **Rename notes and folders**: Add an `r` keybinding (or similar) in browse mode to rename the selected item in-place. Reuse the existing text input widget, pre-populate with the current name, and update the tree + search index on save.
- [x] **Move notes between folders**: Add a keybinding (e.g. `m`) to move the selected note or folder into a different directory. Could use a folder-picker popup or a path input. Update tree, search index, and `currentFile` references after the move.
- [x] **Delete confirmation prompt**: Currently `d` deletes immediately with no confirmation. Add a yes/no confirmation step (inline status prompt or popup) before removing files or folders to prevent accidental data loss.
- [x] **Git integration — commit & sync**: The app was built for git-synced notes. Add keybindings or a command palette action for `git add + commit` (with auto-generated or user-supplied message) and `git pull / push`. Show sync status in the footer. Detect if `notes_dir` is a git repo and surface relevant UI only when applicable.

## Medium Priority — Editor & Productivity

- [x] **Strikethrough formatting toggle**: Add `~~strikethrough~~` toggle shortcut (e.g. `Alt+D` or `Alt+X`) following the same pattern as bold/italic/underline in `editor_selection.go`.
- [x] **Markdown link insertion**: Add a shortcut (e.g. `Ctrl+K`) to insert a `[text](url)` link template, placing the cursor between the parentheses. When text is selected, wrap it as the link text.
- [x] **Heading insertion shortcuts**: Add shortcuts (e.g. `Ctrl+1` through `Ctrl+3`) to insert or toggle `#`, `##`, `###` heading markers on the current line.
- [x] **Word and character count in status bar**: Display word count, character count, and line count for the current note in the footer/status bar during both preview and edit modes.
- [x] **Note sorting options**: Allow sorting the tree view by name (current default), last modified date, file size, or creation date. Add a keybinding to cycle through sort modes and persist the preference in config.
- [x] **Auto-save / crash recovery**: Periodically save a draft of the editor buffer to a temp file (e.g. `.cli-notes/.drafts/`) during edit mode. On next launch, detect unsaved drafts and offer recovery. Clear draft on successful save or cancel.
- [x] **Clipboard integration**: Add a keybinding to copy the current note's content or file path to the system clipboard using `atotto/clipboard` (already a dependency). Could also support paste-from-clipboard in edit mode.
- [x] **Note templates**: Allow users to define custom note templates in `~/.cli-notes/templates/` (or via config). When creating a new note, offer a template picker if templates exist. Fall back to the current default template.

## Medium Priority — Navigation & UX

- [x] **File watcher / auto-refresh**: Watch the `notes_dir` for external filesystem changes (e.g. from another editor or `git pull`) and automatically rebuild the tree + invalidate the search index and render cache. Use `fsnotify` or poll-based detection.
- [x] **Recent files list**: Track recently viewed/edited notes (up to N items) and surface them in a quick-access popup or section. Persist the list across sessions in config or a separate state file.
- [x] **Markdown heading outline / jump-to-section**: In preview mode, parse headings from the current note and display an outline or offer a popup to jump to a specific section. Useful for long documents.
- [x] **Pinning / favorites**: Allow pinning notes or folders so they appear at the top of the tree regardless of sort order. Store pin state in a dotfile (e.g. `.pinned`) in the notes directory or in the app config.
- [x] **Scroll position memory**: Remember the viewport scroll position and cursor location for previously viewed notes so returning to a note restores the reading position.
- [ ] **Preview-pane scroll keybindings**: Add explicit preview scrolling shortcuts (e.g. PgUp/PgDn/Ctrl+U/Ctrl+D) so scroll-position memory has full keyboard-only ergonomics without relying on jumps or mouse wheel behavior.

## Lower Priority — Advanced Features

- [x] **YAML frontmatter support**: Parse YAML frontmatter (`---` delimited) from notes to extract metadata like tags, title, date, and category. Display tags in the tree view and make them searchable via `Ctrl+P`.
- [x] **Tag-based filtering**: When frontmatter tags are supported, add a tag filter mode to the tree view or search popup that shows only notes matching selected tags.
- [x] **Wiki-style `[[links]]` between notes**: Detect `[[Note Name]]` patterns in note content and render them as navigable links. In preview mode, clicking or pressing Enter on a link jumps to the referenced note. In edit mode, offer auto-complete for note names.
- [x] **Export to HTML / PDF**: Add an export command that converts the current note to HTML (using the existing Glamour renderer or Pandoc if available) and writes it to a file or opens it in a browser. PDF export via Pandoc would be a stretch goal.
- [x] **Multiple workspaces / quick directory switching**: Allow users to define multiple `notes_dir` paths in config and switch between them with a keybinding or command. Each workspace maintains its own tree, search index, and render cache.
- [x] **Configurable keybindings**: Allow users to customize keybindings via `~/.cli-notes/config.json` or a separate keymap file. Map action names to key combinations with sensible defaults matching current behavior.
- [x] **Syntax highlighting in fenced code blocks**: Enhance the editor to apply basic syntax highlighting within fenced code blocks during edit mode. The preview pane already renders these via Glamour, but the editor shows them as plain text.
- [x] **Horizontal split / multi-pane editing**: Allow viewing two notes side-by-side or splitting the right pane into preview + source. Useful for referencing one note while editing another.

## Follow-up Improvements

- [x] **Workspace-aware sort persistence**: Persist tree sort mode per workspace instead of globally so switching workspaces restores each workspace’s previous sort preference.
- [x] **Secondary-pane independent scroll memory**: In split mode, persist and restore independent viewport offsets for primary and secondary panes so the second note keeps context while navigating.
- [x] **Advanced wiki autocomplete ranking**: Rank suggestions by recent-file frequency and exact-prefix score to improve insertion speed in large note collections.

## Testing & Quality

- [ ] **Integration tests for note CRUD operations**: Add end-to-end tests that exercise the full create → edit → save → delete lifecycle using a temp `notes_dir`, verifying file contents, tree state, and search index consistency.
- [ ] **Editor formatting round-trip tests**: Add tests for bold/italic/underline toggle behavior covering edge cases: nested formatting, partial overlap, cursor-at-boundary, and empty selection.
- [ ] **Cross-platform CI**: Add CI matrix testing on Linux and Windows (WSL) in addition to the existing macOS development environment to catch platform-specific path handling or terminal issues.
- [ ] **Cross-filesystem move fallback**: Moving with `os.Rename` fails across filesystems (`EXDEV`); add a copy-then-delete fallback for notes/folders when source and destination are on different devices.
- [ ] **True creation-time sort on Linux**: Current `created` sorting uses stat `ctime` on non-macOS Unix, which is metadata-change time rather than true file birth time. Add a platform-aware birth-time implementation (or explicit label/fallback behavior) to avoid misleading ordering.
