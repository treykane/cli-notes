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
| Ctrl+O | Open recent files popup |
| Ctrl+W | Open workspace popup |
| o | Open heading outline popup |
| x | Open export popup |
| Shift+L | Open wiki links popup |
| n | New note |
| f | New folder |
| e | Edit note |
| r | Rename selected item |
| m | Move selected item |
| d | Delete selected item (with confirmation) |
| s | Cycle tree sort mode |
| t | Pin/unpin selected item |
| y / Y | Copy note content / path to clipboard |
| Shift+R or Ctrl+R | Refresh |
| z | Toggle split mode |
| Tab | Toggle split focus |
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

### 11. Recent Files
- Press `Ctrl+O` to open the recent-files popup
- Select with `↑/↓` (or `j/k`) and press `Enter` to jump
- Recent list is persisted in `<notes_dir>/.cli-notes/state.json`

### 12. Heading Outline
- With a note open in preview, press `o` to open heading outline
- Select a heading and press `Enter` to jump the preview to that section

### 13. Pinning Favorites
- Press `t` on any note/folder to toggle pinning
- Pinned entries sort to the top of their folder regardless of active sort mode
- Pin state persists in `<notes_dir>/.cli-notes/state.json`

### 14. Draft Recovery
- Edit mode auto-saves drafts under `<notes_dir>/.cli-notes/.drafts/`
- On next launch, unresolved drafts are offered for recovery (`y`) or discard (`n`)

### 15. Note Templates
- Add template files under `~/.cli-notes/templates/` (or configured `templates_dir`)
- Press `n` to open a template picker before entering the new note name

### 16. Frontmatter Metadata + Tag Search
- Add frontmatter fields (`title`, `date`, `category`, `tags`) between `---` delimiters
- Tree rows show compact `TAGS:` badges for notes with frontmatter tags
- In `Ctrl+P`, use `tag:<name>` tokens (for example `tag:go`) to filter matches

### 17. Workspace Switching
- Configure multiple named workspaces in config (`workspaces` + `active_workspace`)
- Press `Ctrl+W` to open workspace picker and switch roots quickly
- Workspace switching rebinds tree/search/render/app state to the selected workspace

### 18. Wiki Links + Autocomplete
- Add `[[Note Name]]` links inside notes
- Press `Shift+L` in browse mode to open a wiki-links popup for the current note
- In edit mode, typing `[[` opens autocomplete sourced from workspace note names/titles

### 19. Export
- Press `x` to export current note
- HTML export writes `<note>.html` in the same directory
- PDF export writes `<note>.pdf` using Pandoc when available

### 20. Split Mode
- Press `z` to toggle side-by-side split mode for two notes
- Press `Tab` to switch which pane receives open/jump actions

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
