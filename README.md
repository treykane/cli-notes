# cli-notes - BETA VERSION

A beautiful TUI (Text User Interface) notes application that lives in your CLI. Manage your notes with Markdown formatting, organized in a directory structure.

## Features

- Markdown support with rendered preview
- Mode-aware colors with distinct preview vs edit accents
- Colorful tree rows that visually separate folders and notes
- Directory organization (folders instead of notebooks)
- Search popup (`Ctrl+P`) for filtering notes and folders by name
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

On first run, the app runs a configurator and asks where to store your notes. This path is saved in `~/.cli-notes/config.json` as `notes_dir`.

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
| `n` | Create a new note |
| `f` | Create a new folder |
| `e` | Edit the selected note |
| `d` | Delete the selected note/folder |
| `r` | Refresh the directory tree |
| `?` | Toggle help |
| `q` or `Ctrl+C` | Quit the application |

### New Note/Folder

| Key | Action |
|-----|--------|
| `Enter` or `Ctrl+S` | Save |
| `Esc` | Cancel |

### Edit Note

| Key | Action |
|-----|--------|
| `Ctrl+S` | Save |
| `Esc` | Cancel |

### Search Popup

| Key | Action |
|-----|--------|
| Type while popup is open | Filter notes/folders by name |
| `↑`/`↓` or `j`/`k` | Move search selection |
| `Enter` | Jump to selected result |
| `Esc` | Close popup |

## How It Works

1. Browse notes in the directory tree on the left
2. Select a `.md` file to view it with rendered Markdown formatting
3. Press `n` to create a new note in the current directory
4. Press `e` to edit the selected note
5. Press `f` to create folders and organize your notes
6. Press `d` to delete notes or empty folders

## Notes Storage

All notes are stored as plain Markdown files in your configured `notes_dir` (set in `~/.cli-notes/config.json`). You can:
- Edit them with any text editor
- Version control them with Git
- Sync them with cloud storage
- Use them with other Markdown tools
- Expect notes saved by the app to end with exactly one trailing newline

## Requirements

- Go 1.21 or higher
- Terminal with ANSI color support for best experience

## Developer Documentation

See `docs/DEVELOPMENT.md` for local setup, project layout, and runtime flow details.

## Contributing

See `.DEVNOTES/CONTRIBUTING.md` for code style and PR guidelines.

## License

MIT
