# cli-notes - BETA

A TUI (Text User Interface) notes application that lives in your CLI. Manage your notes with Markdown formatting, organized in a directory structure. This tool was born out of my desire to have a simple notes repository that I could sync via git, and quickly and easily manage notes in any environment.

This application is a work in progress, it currently works, but I'm working on further developing the useful features that make it a notes app and not just a text editor with a file tree. Please open issues with feature requests.

## Features

- Markdown support with rendered preview
- YAML frontmatter metadata (`title`, `date`, `category`, `tags`)
- Tag-aware tree rows and `Ctrl+P` filtering with `tag:<name>`
- Mode-aware colors with distinct preview vs edit accents
- Colorful tree rows that visually separate folders and notes
- Directory organization (folders instead of notebooks)
- Search popup (`Ctrl+P`) for filtering folders by name and notes by name/content
- Workspace popup (`Ctrl+W`) for switching named notes roots
- Recent-files popup (`Ctrl+O`) for quick jumps to previously viewed notes
- Heading outline popup (`o`) for jump-to-section in long notes
- Wiki links popup (`Shift+L`) for navigating `[[Note Name]]` references
- Edit-mode wiki autocomplete when typing `[[`
- Export popup (`x`) for HTML and PDF (Pandoc-backed) export
- Split mode (`z`) for side-by-side two-note viewing with focus toggle (`Tab`)
- Pinning (`t`) keeps favorite notes/folders sorted to the top of their folder
- Poll-based file watcher auto-refreshes tree/search/render state on external edits
- Persistent per-note positions restore preview and edit locations when revisiting files
- Cached search index keeps `Ctrl+P` responsive on larger note collections
- Edit-mode markdown helpers: bold/italic/underline/strikethrough, links, and heading toggles
- Status-bar note metrics (words/chars/lines) in preview and edit modes
- Tree sorting modes (name/modified/size/created) with persisted preference
- Auto-saved edit drafts with startup recovery prompts
- Clipboard integration for copy/paste workflows
- Optional note templates from `~/.cli-notes/templates`
- Keyboard-driven workflow
- Plain text storage as `.md` files on your filesystem

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/treykane/cli-notes.git
cd cli-notes

go build -o notes ./cmd/notes
```

### Using Go

```bash
go install github.com/treykane/cli-notes/cmd/notes@latest
```

## Usage

Run the `notes` command to start the application:

```bash
./notes
```

On first run, the app runs a configurator and asks where to store your notes. The default workspace is saved in `~/.cli-notes/config.json`.

### Optional Flags

```bash
./notes --render-light
./notes --configure
```

Use `--render-light` to render markdown with a light theme (default is dark). This is equivalent to setting `CLI_NOTES_GLAMOUR_STYLE=light`.

Use `--configure` to re-run the configurator and change the notes directory.

## Keyboard Shortcuts

### Browse Mode

| Key | Action |
|-----|--------|
| `↑`/`↓` or `k`/`j` or `Ctrl+N` | Move selection |
| `Enter` or `→` or `l` | Expand/collapse folder |
| `←` or `h` | Collapse folder |
| `g` / `G` | Jump to top / bottom |
| `Ctrl+P` | Open search popup |
| `Ctrl+O` | Open recent files popup |
| `Ctrl+W` | Open workspace popup |
| `o` | Open heading outline popup |
| `x` | Open export popup |
| `Shift+L` | Open wiki links popup |
| `z` | Toggle split mode |
| `Tab` | Toggle split focus |
| `n` | Create a new note |
| `f` | Create a new folder |
| `e` | Edit the selected note |
| `r` | Rename the selected note/folder |
| `m` | Move the selected note/folder |
| `d` | Delete the selected note/folder (with confirmation) |
| `s` | Cycle tree sort mode (`name` → `modified` → `size` → `created`) |
| `t` | Pin/unpin selected note/folder |
| `y` | Copy current note content to clipboard |
| `Y` | Copy current note path to clipboard |
| `Shift+R` or `Ctrl+R` | Refresh the directory tree |
| `c`* | Git add + commit (prompts for message) |
| `p`* | Git pull (`--ff-only`) |
| `P`* | Git push |
| `?` | Toggle help |
| `q` or `Ctrl+C` | Quit the application |

\* Git shortcuts only appear when your configured `notes_dir` is inside a Git repository.

### New Note/Folder

| Key | Action |
|-----|--------|
| `Enter` or `Ctrl+S` | Save |
| `Esc` | Cancel |

### Edit Note

| Key | Action |
|-----|--------|
| `Ctrl+S` | Save |
| `Shift+↑` / `Shift+↓` / `Shift+←` / `Shift+→` | Extend selection |
| `Shift+Home` / `Shift+End` | Extend selection to line boundary |
| `Alt+S` | Set/clear selection anchor |
| `Ctrl+B` | Toggle `**bold**` on selection/current word |
| `Alt+I` | Toggle `*italic*` on selection/current word |
| `Ctrl+U` | Toggle `<u>underline</u>` on selection/current word |
| `Alt+X` | Toggle `~~strikethrough~~` on selection/current word |
| `Ctrl+K` | Insert/wrap markdown link as `[text](url)` |
| `Ctrl+1` / `Ctrl+2` / `Ctrl+3` | Toggle `#` / `##` / `###` on current line |
| `Ctrl+V` | Paste from system clipboard |
| `Esc` | Cancel |

Edit mode highlights selected text with a light background and dark text.

### Template Picker

When templates exist in `~/.cli-notes/templates` (or your configured `templates_dir`), pressing `n` opens a template picker before note naming.

| Key | Action |
|-----|--------|
| `↑`/`↓` or `j`/`k` | Move template selection |
| `Enter` | Choose template and continue |
| `Esc` | Cancel new-note flow |

### Draft Recovery

Edit-mode drafts are auto-saved every few seconds in `<notes_dir>/.cli-notes/.drafts/`. On launch, unresolved drafts can be recovered or discarded.

### Search Popup

| Key | Action |
|-----|--------|
| Type while popup is open | Filter folders by name and notes by name/content |
| `↑`/`↓` or `j`/`k` | Move search selection |
| `Enter` | Jump to selected result |
| `Esc` | Close popup |

Search also supports `tag:<name>` tokens (for example `tag:go`) to filter by frontmatter tags.

### Recent Files Popup

| Key | Action |
|-----|--------|
| `↑`/`↓` or `j`/`k` | Move selection |
| `Enter` | Jump to selected recent note |
| `Esc` | Close popup |

### Heading Outline Popup

| Key | Action |
|-----|--------|
| `o` (browse mode) | Open outline for current note |
| `↑`/`↓` or `j`/`k` | Move heading selection |
| `Enter` | Jump preview to selected heading |
| `Esc` | Close popup |

### Persistent UI State

- Recent files, pinned paths, and note positions are stored in `<notes_dir>/.cli-notes/state.json`.
- Returning to a note restores its preview scroll offset and last edit cursor location.

## How It Works

1. Browse notes in the directory tree on the left
2. Select a `.md` file to view it with rendered Markdown formatting
3. Press `n` to create a new note in the current directory
4. Press `e` to edit the selected note
5. Press `f` to create folders and organize your notes
6. Press `r` to rename, `m` to move, and `d` to delete notes or empty folders (with confirmation)

## Notes Storage

All notes are stored as plain Markdown files in your configured `notes_dir` (set in `~/.cli-notes/config.json`). You can:
- Edit them with any text editor
- Version control them with Git
- Sync them with cloud storage
- Use them with other Markdown tools
- Expect notes saved by the app to end with exactly one trailing newline

### Config Additions

`~/.cli-notes/config.json` now supports:
- `workspaces`: named list of notes roots (`name` + `notes_dir`)
- `active_workspace`: active workspace name
- `keybindings`: inline action-to-key overrides
- `keymap_file`: optional external keymap JSON path (default `~/.cli-notes/keymap.json`)

## Requirements

- Go 1.21 or higher
- Terminal with ANSI color support for best experience

## Developer Documentation

See `docs/DEVELOPMENT.md` for local setup, project layout, and runtime flow details.

## Contributing

See `.DEVNOTES/CONTRIBUTING.md` for code style and PR guidelines.

## License

MIT
