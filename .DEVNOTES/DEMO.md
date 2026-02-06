# CLI Notes Demo

This document demonstrates the CLI Notes application features.

## Installation

See `DEVELOPMENT.md` for setup requirements and build commands.

## Running the Application

See `DEVELOPMENT.md` for run commands.

Tip: run `./notes --configure` any time to re-run setup and change `notes_dir`.

## Application Layout

```
┌─────────────────────────────────────────────────────────────┐
│                    Notes: /Users/you/notes                  │
├──────────────────┬──────────────────────────────────────────┤
│                  │                                          │
│ [-] Projects     │  # Welcome to CLI Notes!                 │
│     Ideas.md     │                                          │
│                  │  This is your personal notes manager     │
│                  │  in the terminal.                        │
│                  │                                          │
│ Directory Tree   │  Rendered Markdown View                  │
│                  │                                          │
└──────────────────┴──────────────────────────────────────────┘
```

## Features Demonstrated

### 1. Browse Notes
- Navigate through notes using arrow keys or `k`/`j`
- Directory tree shows folder structure
- Files and folders clearly distinguished

### 2. View Notes with Rendered Markdown
- Select a `.md` file to view
- Markdown is rendered with formatting:
  - Headers (#, ##, ###)
  - Bold (**text**)
  - Italic (*text*)
  - Lists (-, *)
  - Code blocks (```)

### 3. Create New Notes
- Press `n` to create a new note
- Enter note name (extension added automatically)
- Opens in the currently selected directory
- Template content provided

### 4. Edit Notes
- Press `e` to edit the selected note
- Save with Ctrl+S
- Set a selection anchor with `Alt+S` (move cursor to define range)
- Or hold `Shift` with arrow keys/home/end to extend selection
- Selected text is highlighted (light background with dark text)
- Formatting toggles on the active selection (or current word if no selection):
  - `Ctrl+B` for `**bold**`
  - `Alt+I` for `*italic*`
  - `Ctrl+U` for `<u>underline</u>`
  - `Alt+X` for `~~strikethrough~~`
- `Ctrl+K` inserts/wraps `[text](url)` links
- `Ctrl+1`/`Ctrl+2`/`Ctrl+3` toggle heading markers on the current line
- `Ctrl+V` pastes from clipboard in edit mode
- Cancel with Esc

### 5. Organize with Folders
- Press `f` to create a new folder
- Organize notes hierarchically
- Navigate folder structure easily

### 6. Delete Items
- Press `d` to delete selected note or folder
- Folders must be empty to delete
- Confirmation prompt appears (`y` to confirm, `n`/`Esc` to cancel)

### 7. Rename and Move Items
- Press `r` to rename the selected note/folder in-place
- Press `m` to move the selected note/folder to a destination directory
- Both actions reuse the inline text input and update tree/search state

### 8. Git Commit and Sync (when `notes_dir` is a Git repo)
- Press `c` to run `git add -A` + `git commit -m <message>`
- Press `p` to run `git pull --ff-only`
- Press `P` to run `git push`
- Footer displays branch + ahead/behind + dirty/clean status

### 9. Keyboard Shortcuts
| Key | Action |
|-----|--------|
| ↑/↓ or k/j (or Ctrl+N) | Move selection |
| Enter or → or l | Expand/collapse folder |
| ← or h | Collapse folder |
| g / G | Jump to top / bottom |
| Ctrl+P | Open search popup |
| n | New note |
| f | New folder |
| e | Edit note |
| r | Rename selected item |
| m | Move selected item |
| d | Delete selected item (with confirmation) |
| s | Cycle tree sort mode |
| y / Y | Copy note content / path to clipboard |
| Shift+R or Ctrl+R | Refresh |
| c* | Git add + commit |
| p* | Git pull --ff-only |
| P* | Git push |
| Shift+↑/↓/←/→ (edit mode) | Extend selection |
| Shift+Home/End (edit mode) | Extend selection to line boundary |
| Alt+S (edit mode) | Set/clear selection anchor |
| Ctrl+B (edit mode) | Toggle `**bold**` on selection/current word |
| Alt+I (edit mode) | Toggle `*italic*` on selection/current word |
| Ctrl+U (edit mode) | Toggle `<u>underline</u>` on selection/current word |
| Alt+X (edit mode) | Toggle `~~strikethrough~~` on selection/current word |
| Ctrl+K (edit mode) | Insert/wrap markdown link template |
| Ctrl+1/2/3 (edit mode) | Toggle `#`/`##`/`###` heading on current line |
| Ctrl+V (edit mode) | Paste clipboard text |
| ? | Toggle help |
| q or Ctrl+C | Quit |

\* Git shortcuts are shown only when `notes_dir` is a Git repository.

### 10. Search Notes
- Press `Ctrl+P` to open the search popup
- Type to filter folders by name and notes by name/content
- Use `↑/↓` or `j`/`k` to choose a match
- Press `Enter` to jump to the selected item
- Press `Esc` to close the popup

### 11. Draft Recovery
- Edit mode auto-saves drafts under `<notes_dir>/.cli-notes/.drafts/`
- On next launch, unresolved drafts are offered for recovery (`y`) or discard (`n`)

### 12. Note Templates
- Add template files under `~/.cli-notes/templates/` (or configured `templates_dir`)
- Press `n` to open a template picker before entering the new note name

## File Storage

Example storage layout (replace with your configured `notes_dir`; see `DEVELOPMENT.md` for details):

```
~/notes/
├── Welcome.md
├── Ideas.md
├── Projects/
│   └── CLI-Notes-Project.md
└── Personal/
    └── TODO.md
```

## Technical Implementation

- **Framework**: Bubble Tea + Bubbles
- **Rendering**: Glamour (Markdown to ANSI)
- **Storage**: Plain .md files on filesystem
- **Organization**: Directory-based (no database)
- **Language**: Go
