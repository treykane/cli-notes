# cli-notes

A beautiful TUI (Text User Interface) notes application that lives in your CLI. Manage your notes with Markdown formatting, organized in a directory structure.

## Features

- 📝 **Markdown Support** - Write notes in Markdown with live rendered preview
- 📁 **Directory Organization** - Organize notes in folders instead of notebooks
- 🎨 **Beautiful TUI** - Modern terminal interface powered by Textual
- ⌨️  **Keyboard-Driven** - Fast navigation and editing with keyboard shortcuts
- 💾 **Plain Text Storage** - Notes stored as `.md` files on your filesystem

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/treykane/cli-notes.git
cd cli-notes

# Install the package
pip install -e .
```

### Using pip (once published)

```bash
pip install cli-notes
```

## Usage

Simply run the `notes` command to start the application:

```bash
notes
```

Your notes will be stored in `~/notes` by default.

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `n` | Create a new note |
| `f` | Create a new folder |
| `e` | Edit the selected note |
| `d` | Delete the selected note/folder |
| `r` | Refresh the directory tree |
| `q` | Quit the application |
| `Ctrl+S` | Save (when editing) |
| `Esc` | Cancel (when editing/creating) |

## How It Works

1. **Browse Notes** - Use arrow keys to navigate through your notes in the directory tree on the left
2. **View Notes** - Select a `.md` file to view it with rendered Markdown formatting on the right
3. **Create Notes** - Press `n` to create a new note in the current directory
4. **Edit Notes** - Press `e` to edit the selected note with syntax highlighting
5. **Organize** - Press `f` to create folders and organize your notes
6. **Delete** - Press `d` to delete notes or empty folders

## Notes Storage

All notes are stored as plain Markdown files in `~/notes`. You can:
- Edit them with any text editor
- Version control them with Git
- Sync them with cloud storage
- Use them with other Markdown tools

## Requirements

- Python 3.8 or higher
- Terminal with Unicode support for best experience

## License

MIT
