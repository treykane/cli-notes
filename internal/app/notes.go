package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// welcomeNote is written to ~/notes/Welcome.md on first run.
const welcomeNote = "# Welcome to CLI Notes!\n\n" +
	"This is your personal notes manager in the terminal.\n\n" +
	"## Features\n\n" +
	"- Create and edit notes in Markdown\n" +
	"- Organize notes in folders\n" +
	"- View rendered Markdown formatting\n" +
	"- Keyboard-driven interface\n\n" +
	"## Keyboard Shortcuts\n\n" +
	"- Up/Down or k/j: Move selection\n" +
	"- Enter/Right/l: Expand or collapse folder\n" +
	"- Left/h: Collapse folder\n" +
	"- g / G: Jump to top / bottom\n" +
	"- Ctrl+P: Open search popup\n" +
	"- n: Create a new note\n" +
	"- f: Create a new folder\n" +
	"- e: Edit the selected note\n" +
	"- d: Delete the selected note\n" +
	"- r: Refresh the directory tree\n" +
	"- ?: Toggle help\n" +
	"- Enter or Ctrl+S: Save (when naming new note/folder)\n" +
	"- Ctrl+S: Save (when editing)\n" +
	"- Esc: Cancel (when naming or editing)\n" +
	"- q or Ctrl+C: Quit the application\n\n" +
	"## Getting Started\n\n" +
	"1. Press n to create a new note\n" +
	"2. Select a note and press e to edit it\n" +
	"3. Press f to create folders and organize your notes\n\n" +
	"Happy note-taking!\n"

func ensureNotesDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	notesDir := filepath.Join(home, "notes")
	if err := os.MkdirAll(notesDir, 0o755); err != nil {
		return "", err
	}

	if isDirEmpty(notesDir) {
		welcomePath := filepath.Join(notesDir, "Welcome.md")
		_ = os.WriteFile(welcomePath, []byte(welcomeNote), 0o644)
	}

	return notesDir, nil
}

// selectedItem returns the currently highlighted tree item, if any.
func (m *Model) selectedItem() *treeItem {
	if len(m.items) == 0 || m.cursor < 0 || m.cursor >= len(m.items) {
		return nil
	}
	return &m.items[m.cursor]
}

// selectedPath returns the selected item's path or the root notes dir.
func (m *Model) selectedPath() string {
	item := m.selectedItem()
	if item == nil {
		return m.notesDir
	}
	return item.path
}

// selectedParentDir returns a directory suitable for creating new items.
func (m *Model) selectedParentDir() string {
	path := m.selectedPath()
	info, err := os.Stat(path)
	if err != nil {
		return m.notesDir
	}
	if info.IsDir() {
		return path
	}
	return filepath.Dir(path)
}

// startNewNote switches to new-note mode and configures the input.
func (m *Model) startNewNote() {
	m.mode = modeNewNote
	m.showHelp = false
	m.newParent = m.selectedParentDir()
	m.input.Reset()
	m.input.Placeholder = "Note name (without .md extension)"
	m.input.Focus()
	m.status = ""
}

// startNewFolder switches to new-folder mode and configures the input.
func (m *Model) startNewFolder() {
	m.mode = modeNewFolder
	m.showHelp = false
	m.newParent = m.selectedParentDir()
	m.input.Reset()
	m.input.Placeholder = "Folder name"
	m.input.Focus()
	m.status = ""
}

// startEditNote loads the current file and opens the editor.
func (m *Model) startEditNote() (tea.Model, tea.Cmd) {
	if m.currentFile == "" {
		m.status = "No note selected"
		return m, nil
	}

	content, err := os.ReadFile(m.currentFile)
	if err != nil {
		m.status = "Error reading note"
		return m, nil
	}

	m.mode = modeEditNote
	m.showHelp = false
	m.editor.SetValue(string(content))
	m.editor.CursorEnd()
	m.editor.Focus()
	m.status = "Editing " + filepath.Base(m.currentFile)
	return m, nil
}

// saveNewNote writes a new markdown file and refreshes the tree.
func (m *Model) saveNewNote() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.input.Value())
	if name == "" {
		m.status = "Note name is required"
		return m, nil
	}
	if !strings.HasSuffix(strings.ToLower(name), ".md") {
		name += ".md"
	}

	path := filepath.Join(m.newParent, name)
	content := fmt.Sprintf("# %s\n\nYour note content here...\n", strings.TrimSuffix(name, ".md"))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		m.status = "Error creating note"
		return m, nil
	}

	m.mode = modeBrowse
	m.status = "Created note: " + name
	m.expanded[m.newParent] = true
	m.refreshTree()
	cmd := m.setCurrentFile(path)
	return m, cmd
}

// saveNewFolder creates a directory and refreshes the tree.
func (m *Model) saveNewFolder() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.input.Value())
	if name == "" {
		m.status = "Folder name is required"
		return m, nil
	}

	path := filepath.Join(m.newParent, name)
	if err := os.MkdirAll(path, 0o755); err != nil {
		m.status = "Error creating folder"
		return m, nil
	}

	m.mode = modeBrowse
	m.status = "Created folder: " + name
	m.expanded[m.newParent] = true
	m.refreshTree()
	return m, nil
}

// saveEdit writes the editor contents to the current file.
func (m *Model) saveEdit() (tea.Model, tea.Cmd) {
	if m.currentFile == "" {
		m.status = "No note selected"
		return m, nil
	}
	if err := os.WriteFile(m.currentFile, []byte(m.editor.Value()), 0o644); err != nil {
		m.status = "Error saving note"
		return m, nil
	}

	m.mode = modeBrowse
	m.status = "Saved: " + filepath.Base(m.currentFile)
	cmd := m.setCurrentFile(m.currentFile)
	return m, cmd
}

// deleteSelected removes the selected file or empty folder.
func (m *Model) deleteSelected() {
	item := m.selectedItem()
	if item == nil {
		m.status = "No item selected"
		return
	}

	if item.path == m.notesDir {
		m.status = "Cannot delete the root notes directory"
		return
	}

	if item.isDir {
		if !isDirEmpty(item.path) {
			m.status = "Folder is not empty. Delete contents first."
			return
		}
		if err := os.Remove(item.path); err != nil {
			m.status = "Error deleting folder"
			return
		}
		m.status = "Deleted folder: " + item.name
	} else {
		if err := os.Remove(item.path); err != nil {
			m.status = "Error deleting file"
			return
		}
		m.status = "Deleted: " + item.name
	}

	if item.path == m.currentFile {
		m.currentFile = ""
		m.viewport.SetContent("Select a note to view")
	}
	m.refreshTree()
}

// displayRelative shows paths relative to the notes root for UI display.
func (m *Model) displayRelative(path string) string {
	rel, err := filepath.Rel(m.notesDir, path)
	if err != nil || rel == "." {
		return "/"
	}
	return rel
}

// isDirEmpty reports whether a directory has no entries.
func isDirEmpty(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	return len(entries) == 0
}
