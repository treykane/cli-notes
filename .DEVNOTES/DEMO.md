# CLI Notes Demo

This document demonstrates the CLI Notes application features.

## Installation

See `DEVELOPMENT.md` for setup requirements and build commands.

## Running the Application

See `DEVELOPMENT.md` for run commands.

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
- Cancel with Esc

### 5. Organize with Folders
- Press `f` to create a new folder
- Organize notes hierarchically
- Navigate folder structure easily

### 6. Delete Items
- Press `d` to delete selected note or folder
- Folders must be empty to delete
- Confirmation via status message

### 7. Keyboard Shortcuts
| Key | Action |
|-----|--------|
| ↑/↓ or k/j | Move selection |
| Enter or → | Expand/collapse folder |
| ← | Collapse folder |
| n | New note |
| f | New folder |
| e | Edit note |
| d | Delete |
| r | Refresh |
| ? | Toggle help |
| q or Ctrl+C | Quit |

## File Storage

Example storage layout (notes are Markdown files; see `DEVELOPMENT.md` for details):

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
