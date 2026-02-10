# cli-notes (Beta)

A terminal-based notes app. Write in Markdown, organize in folders, sync with Git.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-green)

---

## Why cli-notes?

Most note-taking tools lock you into proprietary formats or clunky GUIs.
**cli-notes** keeps it simple: your notes are plain `.md` files in a regular
directory. Open them in any editor, version them with Git, sync them however
you like — and when you want a fast, keyboard-driven experience, launch `notes`
in your terminal.

> **This project is under active development.** It works today, but new
> features are still landing. Please
> [open an issue](https://github.com/treykane/cli-notes/issues) if you have a
> feature request or find a bug.

---

## Quick Start

### Install

**With Go (recommended):**

```bash
go install github.com/treykane/cli-notes/cmd/notes@latest
```

**From source:**

```bash
git clone https://github.com/treykane/cli-notes.git
cd cli-notes
go build -o notes ./cmd/notes
```

### Run

```bash
notes
```

On first launch a short configurator asks where to store your notes. Your
choice is saved to `~/.cli-notes/config.json`.

### Optional Flags

| Flag              | Purpose                                                                 |
| ----------------- | ----------------------------------------------------------------------- |
| `--render-light`  | Render Markdown with a light theme (or set `CLI_NOTES_GLAMOUR_STYLE=light`) |
| `--configure`     | Re-run the configurator to change your notes directory                  |
| `--version`       | Print version and commit hash                                          |

---

## How It Works

```text
┌─ Tree ──────────┐┌─ Preview / Edit ──────────────────┐
│ 📁 work         ││                                    │
│   📄 standup.md ││  rendered Markdown or edit buffer   │
│ 📁 personal     ││                                    │
│   📄 ideas.md   ││                                    │
└─────────────────┘└────────────────────────────────────┘
```

1. **Browse** — navigate the folder tree on the left.
2. **Preview** — select any `.md` file to see rendered Markdown on the right.
3. **Edit** — press `e` to edit in-place with formatting helpers.
4. **Organize** — create (`n`/`f`), rename (`r`), move (`m`), or delete (`d`)
   notes and folders.

---

## Features

### Core

- Plain `.md` file storage — no lock-in
- Markdown preview with rendered output
- YAML frontmatter metadata (`title`, `date`, `category`, `tags`)
- Directory-based organization (folders as notebooks)
- Clipboard integration (copy/paste)
- Auto-saved edit drafts with recovery on next launch

### Navigation & Search

- **Search** (`Ctrl+P`) — filter notes by name, content, or `tag:<name>`; shows match counts
- **Recent files** (`Ctrl+O`) — quickly jump back to previously viewed notes
- **Heading outline** (`o`) — jump to any section in a long note
- **Wiki links** (`Shift+L`) — navigate `[[Note Name]]` references between notes
- **Split mode** (`z`) — view two notes side by side; toggle focus with `Tab`

### Editing

- Bold, italic, underline, strikethrough, link, and heading shortcuts
- Undo / redo (`Ctrl+Z` / `Ctrl+Y`) with smart history grouping
- Mouse text selection (left-click drag)
- Wiki-link autocomplete when typing `[[`
- Note templates from `~/.cli-notes/templates`

### Organization & Workflow

- **Workspaces** (`Ctrl+W`) — switch between multiple notes roots
- **Pinning** (`t`) — keep favorites at the top of their folder
- **Tree sorting** (`s`) — cycle through name / modified / size / created
- **Git integration** — commit (`c`), pull (`p`), and push (`P`) without leaving the app
- **Export** (`x`) — HTML or PDF (via Pandoc)

### Polish

- Three UI theme presets: Ocean/Citrus, Sunset, Neon Slate
- Configurable keybindings (inline or external keymap file)
- File watcher auto-refreshes on external edits
- Persistent scroll positions and cursor locations per note
- Adaptive footer with contextual key hints and note metrics
- Scrollable help panel for small terminals

---

## Keyboard Reference

### Browse Mode

| Key                              | Action                                    |
| -------------------------------- | ----------------------------------------- |
| `↑` / `↓` or `k` / `j`         | Move selection                            |
| `Enter` / `→` / `l`             | Expand or open                            |
| `←` / `h`                       | Collapse folder                           |
| `g` / `G`                       | Jump to top / bottom                      |
| `PgUp` / `PgDn`                 | Scroll preview one page                   |
| `Ctrl+U` / `Ctrl+D`             | Scroll preview half page                  |
| `Ctrl+P`                        | Search                                    |
| `Ctrl+O`                        | Recent files                              |
| `Ctrl+W`                        | Switch workspace                          |
| `o`                             | Heading outline                           |
| `x`                             | Export                                    |
| `Shift+L`                       | Wiki links                                |
| `z`                             | Toggle split mode                         |
| `Tab`                           | Toggle split focus                        |
| `n` / `f`                       | New note / new folder                     |
| `e`                             | Edit selected note                        |
| `r` / `m` / `d`                 | Rename / move / delete (with confirmation)|
| `s`                             | Cycle sort mode                           |
| `t`                             | Pin / unpin                               |
| `y` / `Y`                       | Copy content / copy path                  |
| `c` / `p` / `P` ¹              | Git commit / pull / push                  |
| `Shift+R` or `Ctrl+R`           | Refresh tree                              |
| `?`                             | Toggle help                               |
| `q` or `Ctrl+C`                 | Quit                                      |

> ¹ Git shortcuts only appear when `notes_dir` is inside a Git repository.

### Edit Mode

| Key                                        | Action                          |
| ------------------------------------------ | ------------------------------- |
| `Ctrl+S`                                   | Save                            |
| `Ctrl+Z` / `Ctrl+Y`                        | Undo / redo                     |
| `Shift+Arrow` / `Shift+Home` / `Shift+End` | Extend selection                |
| Left-click + drag                          | Mouse selection                 |
| `Alt+S`                                    | Set / clear selection anchor    |
| `Ctrl+B`                                   | Bold                            |
| `Alt+I`                                    | Italic                          |
| `Ctrl+U`                                   | Underline                       |
| `Alt+X`                                    | Strikethrough                   |
| `Ctrl+K`                                   | Insert link                     |
| `Ctrl+1` / `Ctrl+2` / `Ctrl+3`             | Toggle heading level            |
| `Ctrl+V`                                   | Paste                           |
| `Esc`                                      | Cancel                          |

### Popups (Search, Recent, Outline, Templates)

| Key                      | Action                |
| ------------------------ | --------------------- |
| `↑` / `↓` or `k` / `j`  | Move selection        |
| `Enter`                  | Confirm / jump        |
| `Esc`                    | Close                 |

In the **Search popup**, type to filter; use `tag:<name>` to filter by
frontmatter tags.

In the **Template picker** (shown when pressing `n` if templates exist in
`~/.cli-notes/templates`), choose a template before naming your note.

---

## Notes Storage

All notes live as plain `.md` files under your configured `notes_dir`. That
means you can:

- Edit them in any text editor
- Version them with Git
- Sync them with any cloud storage
- Use them with other Markdown tools

Notes saved by the app always end with exactly one trailing newline.

### State & Drafts

| Path                                        | Purpose                                      |
| ------------------------------------------- | --------------------------------------------- |
| `~/.cli-notes/config.json`                  | Global configuration                          |
| `<notes_dir>/.cli-notes/state.json`         | Recent files, pins, positions, open-frequency |
| `<notes_dir>/.cli-notes/.drafts/`           | Auto-saved edit drafts (recovered on launch)  |

### Configuration Options

Your `~/.cli-notes/config.json` supports:

| Key                           | Description                                                    |
| ----------------------------- | -------------------------------------------------------------- |
| `workspaces`                  | Named list of notes roots (`name` + `notes_dir`)               |
| `active_workspace`            | Currently active workspace name                                |
| `tree_sort_by_workspace`      | Sort mode per workspace (`name` / `modified` / `size` / `created`) |
| `keybindings`                 | Inline action-to-key overrides                                 |
| `keymap_file`                 | Path to external keymap JSON (default `~/.cli-notes/keymap.json`) |
| `theme_preset`                | `ocean_citrus`, `sunset`, or `neon_slate`                      |
| `file_watch_interval_seconds` | Filesystem poll interval in seconds (default `2`, range `1–300`) |

---

## Requirements

- **Go 1.21+**
- A terminal with ANSI color support

---

## Contributing & Development

- **Development docs:** `docs/DEVELOPMENT.md`
- **Contributing guide:** `.DEVNOTES/CONTRIBUTING.md`

---

## License

MIT
