// notes.go implements note and folder CRUD operations (create, read, update, delete)
// as well as rename and move workflows.
//
// All filesystem mutations funnel through this file. Each operation follows a
// consistent pattern:
//
//  1. Validate inputs (non-empty name, path within notes root, no collisions).
//  2. Perform the filesystem operation (WriteFile, MkdirAll, Rename, Remove).
//  3. Update in-memory caches: rebuild the tree, upsert/remove from the search
//     index, refresh git status, and remap persisted state (pins, positions,
//     recent files) when paths change.
//  4. Return a Bubble Tea Cmd to re-render the viewport if a file was affected.
//
// Note content is normalized before writing so that every file ends with
// exactly one trailing newline (see normalizeNoteContent).
package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// welcomeNote is the markdown content seeded into a new notes directory on
// first run. It serves as both a quick-start guide and a smoke test that the
// directory was created successfully.
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
	"- Ctrl+O: Open recent files popup\n" +
	"- Ctrl+W: Open workspace popup\n" +
	"- o: Open heading outline popup\n" +
	"- x: Open export popup\n" +
	"- Shift+L: Open wiki links popup\n" +
	"- n: Create a new note\n" +
	"- f: Create a new folder\n" +
	"- e: Edit the selected note\n" +
	"- r: Rename the selected item\n" +
	"- m: Move the selected item\n" +
	"- d: Delete the selected note/folder (with confirmation)\n" +
	"- Shift+R or Ctrl+R: Refresh the directory tree\n" +
	"- z: Toggle split mode (two notes)\n" +
	"- Tab: Toggle split focus\n" +
	"- ?: Toggle help\n" +
	"- Enter or Ctrl+S: Save (when naming new note/folder)\n" +
	"- Ctrl+S: Save (when editing)\n" +
	"- Shift+Arrows or Shift+Home/End: Extend selection (when editing)\n" +
	"- Alt+S: Set/clear selection anchor (when editing)\n" +
	"- Ctrl+B / Alt+I / Ctrl+U / Alt+X: Toggle bold/italic/underline/strikethrough on selection/word (when editing)\n" +
	"- Ctrl+K: Insert [text](url) link template (when editing)\n" +
	"- Ctrl+1/2/3: Toggle heading level on current line (when editing)\n" +
	"- Ctrl+V: Paste from clipboard (when editing)\n" +
	"- Type [[ in edit mode for wiki note-name autocomplete\n" +
	"- y / Y: Copy current note content / path to clipboard\n" +
	"- s: Cycle tree sort mode (name/modified/size/created)\n" +
	"- t: Pin/unpin selected item\n" +
	"- Esc: Cancel (when naming or editing)\n" +
	"- q or Ctrl+C: Quit the application\n\n" +
	"## Getting Started\n\n" +
	"1. Press n to create a new note\n" +
	"2. Select a note and press e to edit it\n" +
	"3. Press f to create folders and organize your notes\n\n" +
	"Happy note-taking!\n"

// ensureNotesDir creates the notes directory if it does not exist and seeds
// it with a Welcome.md note when the directory is empty. This is called
// during app initialization to guarantee the filesystem is ready.
func ensureNotesDir(notesDir string) error {
	if err := os.MkdirAll(notesDir, DirPermission); err != nil {
		return fmt.Errorf("create notes directory %q: %w", notesDir, err)
	}

	if isDirEmpty(notesDir) {
		welcomePath := filepath.Join(notesDir, "Welcome.md")
		if err := os.WriteFile(welcomePath, []byte(normalizeNoteContent(welcomeNote)), FilePermission); err != nil {
			return fmt.Errorf("seed welcome note %q: %w", welcomePath, err)
		}
	}

	return nil
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

// configureInputForMode prepares the input widget for new note/folder creation.
func (m *Model) configureInputForMode(mode mode, placeholder string) {
	m.mode = mode
	m.showHelp = false
	m.newParent = m.selectedParentDir()
	m.input.Reset()
	m.input.Placeholder = placeholder
	m.input.Focus()
	m.status = ""
}

// startNewNote switches to new-note mode and configures the input.
func (m *Model) startNewNote() {
	m.selectedTemplate = nil
	m.templates = m.loadTemplates()
	m.templateCursor = 0
	if len(m.templates) > 0 {
		m.mode = modeTemplatePicker
		m.status = "Choose a template for the new note"
		return
	}
	m.configureInputForMode(modeNewNote, "Note name (without .md extension)")
}

// startNewFolder switches to new-folder mode and configures the input.
func (m *Model) startNewFolder() {
	m.configureInputForMode(modeNewFolder, "Folder name")
}

// startRenameSelected switches to rename mode with the current item name prefilled.
func (m *Model) startRenameSelected() {
	item := m.selectedItem()
	if item == nil {
		m.status = "No item selected"
		return
	}
	if item.path == m.notesDir {
		m.status = "Cannot rename the root notes directory"
		return
	}
	if !isWithinRoot(m.notesDir, item.path) {
		m.status = "Cannot rename item outside notes directory"
		return
	}

	m.mode = modeRenameItem
	m.showHelp = false
	m.actionPath = item.path
	m.input.Reset()
	m.input.Placeholder = "New name"
	m.input.SetValue(item.name)
	m.input.CursorEnd()
	m.input.Focus()
	m.status = "Rename: Enter or Ctrl+S to save, Esc to cancel"
}

// startMoveSelected switches to move mode with the current parent directory prefilled.
// startMoveSelected switches to move mode with the current parent directory
// prefilled in the input widget. The user can edit the destination path
// relative to the notes root.
func (m *Model) startMoveSelected() {
	item := m.selectedItem()
	if item == nil {
		m.status = "No item selected"
		return
	}
	if item.path == m.notesDir {
		m.status = "Cannot move the root notes directory"
		return
	}
	if !isWithinRoot(m.notesDir, item.path) {
		m.status = "Cannot move item outside notes directory"
		return
	}

	m.mode = modeMoveItem
	m.showHelp = false
	m.actionPath = item.path
	m.input.Reset()
	m.input.Placeholder = "Destination folder (relative to notes root)"
	m.input.SetValue(m.displayRelative(filepath.Dir(item.path)))
	m.input.CursorEnd()
	m.input.Focus()
	m.status = "Move: Enter or Ctrl+S to save, Esc to cancel"
}

// startEditNote loads the current file and opens the editor.
func (m *Model) startEditNote() (tea.Model, tea.Cmd) {
	if m.currentFile == "" {
		m.status = "No note selected"
		return m, nil
	}

	content, err := os.ReadFile(m.currentFile)
	if err != nil {
		m.setStatusError("Error reading note", err, "path", m.currentFile)
		return m, nil
	}

	m.mode = modeEditNote
	m.showHelp = false
	m.clearEditorSelection()
	m.editor.SetValue(string(content))
	m.currentNoteContent = string(content)
	m.restoreEditorCursor(m.currentFile)
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
	if !isWithinRoot(m.notesDir, path) {
		m.status = "Invalid note name"
		return m, nil
	}
	content := m.defaultNewNoteContent(name)
	if m.selectedTemplate != nil {
		content = m.selectedTemplate.content
	}
	if err := os.WriteFile(path, []byte(normalizeNoteContent(content)), FilePermission); err != nil {
		m.setStatusError("Error creating note", err, "path", path)
		return m, nil
	}

	m.mode = modeBrowse
	m.status = "Created note: " + name
	m.expanded[m.newParent] = true
	m.selectedTemplate = nil
	m.refreshTree()
	if m.searchIndex != nil {
		m.searchIndex.upsertPath(path)
	}
	m.refreshGitStatus()
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
	if !isWithinRoot(m.notesDir, path) {
		m.status = "Invalid folder name"
		return m, nil
	}
	if err := os.MkdirAll(path, DirPermission); err != nil {
		m.setStatusError("Error creating folder", err, "path", path)
		return m, nil
	}

	m.mode = modeBrowse
	m.status = "Created folder: " + name
	m.expanded[m.newParent] = true
	m.refreshTree()
	if m.searchIndex != nil {
		m.searchIndex.upsertPath(path)
	}
	m.refreshGitStatus()
	return m, nil
}

// saveRenameItem validates the new name, performs the filesystem rename, and
// updates all in-memory state (expanded paths, pinned paths, recent files,
// note positions, search index, and git status) to reflect the new path.
func (m *Model) saveRenameItem() (tea.Model, tea.Cmd) {
	oldPath := m.actionPath
	name := strings.TrimSpace(m.input.Value())
	if name == "" {
		m.status = "Name is required"
		return m, nil
	}
	if filepath.Base(name) != name {
		m.status = "Name cannot include path separators"
		return m, nil
	}
	if !isWithinRoot(m.notesDir, oldPath) {
		m.status = "Invalid rename target"
		m.mode = modeBrowse
		return m, nil
	}

	parent := filepath.Dir(oldPath)
	newPath := filepath.Join(parent, name)
	if oldPath == newPath {
		m.mode = modeBrowse
		m.status = "Name unchanged"
		return m, nil
	}
	if !isWithinRoot(m.notesDir, newPath) {
		m.status = "Invalid target name"
		return m, nil
	}
	if _, err := os.Stat(newPath); err == nil {
		m.status = "Target already exists"
		return m, nil
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		m.setStatusError("Error renaming item", err, "from", oldPath, "to", newPath)
		return m, nil
	}

	m.mode = modeBrowse
	m.remapExpandedPaths(oldPath, newPath)
	m.remapStatePaths(oldPath, newPath)
	m.currentFile = replacePathPrefix(m.currentFile, oldPath, newPath)
	if m.searchIndex != nil {
		m.searchIndex.removePath(oldPath)
		m.searchIndex.upsertPath(newPath)
	}
	m.refreshGitStatus()
	m.refreshTree()
	m.rebuildTreeKeep(newPath)
	m.status = "Renamed to: " + name
	if m.currentFile != "" {
		return m, m.setCurrentFile(m.currentFile)
	}
	return m, nil
}

// saveMoveItem validates the destination folder, performs the filesystem move
// via os.Rename, and updates all in-memory state to reflect the new location.
// Moving a folder into itself (or a descendant) is detected and rejected.
//
// Note: os.Rename may fail across filesystem boundaries (EXDEV). A copy-then-
// delete fallback is tracked as a future improvement in TASKS.md.
func (m *Model) saveMoveItem() (tea.Model, tea.Cmd) {
	oldPath := m.actionPath
	if !isWithinRoot(m.notesDir, oldPath) {
		m.status = "Invalid move target"
		m.mode = modeBrowse
		return m, nil
	}

	destDir, err := m.resolveMoveDestination(m.input.Value())
	if err != nil {
		m.status = err.Error()
		return m, nil
	}

	newPath := filepath.Join(destDir, filepath.Base(oldPath))
	info, statErr := os.Stat(oldPath)
	if statErr != nil {
		m.setStatusError("Error reading source item", statErr, "path", oldPath)
		m.mode = modeBrowse
		return m, nil
	}
	if info.IsDir() {
		prefix := oldPath + string(os.PathSeparator)
		if newPath == oldPath || strings.HasPrefix(newPath, prefix) {
			m.status = "Cannot move a folder into itself"
			return m, nil
		}
	}
	if newPath == oldPath {
		m.mode = modeBrowse
		m.status = "Item already in that folder"
		return m, nil
	}
	if _, err := os.Stat(newPath); err == nil {
		m.status = "Destination already exists"
		return m, nil
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		m.setStatusError("Error moving item", err, "from", oldPath, "to", newPath)
		return m, nil
	}

	m.mode = modeBrowse
	m.expanded[destDir] = true
	m.remapExpandedPaths(oldPath, newPath)
	m.remapStatePaths(oldPath, newPath)
	m.currentFile = replacePathPrefix(m.currentFile, oldPath, newPath)
	if m.searchIndex != nil {
		m.searchIndex.removePath(oldPath)
		m.searchIndex.upsertPath(newPath)
	}
	m.refreshGitStatus()
	m.refreshTree()
	m.rebuildTreeKeep(newPath)
	m.status = "Moved to: " + m.displayRelative(destDir)
	if m.currentFile != "" {
		return m, m.setCurrentFile(m.currentFile)
	}
	return m, nil
}

// saveEdit writes the editor contents to the current file.
func (m *Model) saveEdit() (tea.Model, tea.Cmd) {
	if m.currentFile == "" {
		m.status = "No note selected"
		return m, nil
	}
	content := normalizeNoteContent(m.editor.Value())
	if err := os.WriteFile(m.currentFile, []byte(content), FilePermission); err != nil {
		m.setStatusError("Error saving note", err, "path", m.currentFile)
		return m, nil
	}

	m.mode = modeBrowse
	m.rememberNotePosition(m.currentFile)
	m.saveAppState()
	m.clearEditorSelection()
	m.currentNoteContent = content
	m.clearDraftForPath(m.currentFile)
	m.status = "Saved: " + filepath.Base(m.currentFile)
	if m.searchIndex != nil {
		m.searchIndex.upsertPath(m.currentFile)
	}
	m.refreshGitStatus()
	cmd := m.setCurrentFile(m.currentFile)
	return m, cmd
}

// normalizeNoteContent ensures notes always end with exactly one newline.
func normalizeNoteContent(content string) string {
	return strings.TrimRight(content, "\r\n") + "\n"
}

func (m *Model) defaultNewNoteContent(name string) string {
	return fmt.Sprintf("# %s\n\nYour note content here...\n", strings.TrimSuffix(name, ".md"))
}

// validateDeleteTarget checks if the item can be deleted and returns an error message if not.
func (m *Model) validateDeleteTarget(item *treeItem) string {
	if item == nil {
		return "No item selected"
	}
	if item.path == m.notesDir {
		return "Cannot delete the root notes directory"
	}
	if !isWithinRoot(m.notesDir, item.path) {
		return "Cannot delete item outside notes directory"
	}
	if item.isDir && !isDirEmpty(item.path) {
		return "Folder is not empty. Delete contents first."
	}
	return ""
}

// performDelete executes the deletion and updates state.
func (m *Model) performDelete(item *treeItem) {
	if err := os.Remove(item.path); err != nil {
		itemType := "file"
		if item.isDir {
			itemType = "folder"
		}
		m.setStatusError("Error deleting "+itemType, err, "path", item.path)
		return
	}

	// Update status message
	if item.isDir {
		m.status = "Deleted folder: " + item.name
	} else {
		m.status = "Deleted: " + item.name
	}

	// Clear viewport if we deleted the current file
	if item.path == m.currentFile {
		m.clearStateForPath(item.path)
		m.currentFile = ""
		m.viewport.SetContent("Select a note to view")
	} else {
		m.clearStateForPath(item.path)
	}

	// Update search index and refresh tree
	if m.searchIndex != nil {
		m.searchIndex.removePath(item.path)
	}
	m.refreshGitStatus()
	m.refreshTree()
}

// deleteSelected removes the selected file or empty folder.
func (m *Model) deleteSelected() {
	item := m.selectedItem()

	if errMsg := m.validateDeleteTarget(item); errMsg != "" {
		m.status = errMsg
		return
	}
	m.pendingDelete = *item
	m.mode = modeConfirmDelete
	targetType := "note"
	if item.isDir {
		targetType = "folder"
	}
	m.status = fmt.Sprintf("Delete %s \"%s\"? (y/N)", targetType, item.name)
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

// resolveMoveDestination parses user input into an absolute directory path.
//
// Input handling:
//   - Leading "/" is treated as relative to the notes root (not filesystem root).
//   - Absolute paths are accepted as-is.
//   - Bare relative paths are resolved relative to the notes root.
//
// The resolved path must be an existing directory within the notes root.
func (m *Model) resolveMoveDestination(value string) (string, error) {
	destValue := strings.TrimSpace(value)
	if destValue == "" {
		return "", fmt.Errorf("Destination folder is required")
	}

	var destDir string
	switch {
	case strings.HasPrefix(destValue, "/"):
		destDir = filepath.Join(m.notesDir, strings.TrimPrefix(filepath.Clean(destValue), "/"))
	case filepath.IsAbs(destValue):
		destDir = filepath.Clean(destValue)
	default:
		destDir = filepath.Join(m.notesDir, destValue)
	}
	destDir = filepath.Clean(destDir)

	if !isWithinRoot(m.notesDir, destDir) {
		return "", fmt.Errorf("Destination must be inside notes directory")
	}
	info, err := os.Stat(destDir)
	if err != nil {
		return "", fmt.Errorf("Destination folder not found")
	}
	if !info.IsDir() {
		return "", fmt.Errorf("Destination must be a folder")
	}
	return destDir, nil
}

// remapExpandedPaths updates the expanded-directories map after a rename or
// move so that the tree remembers which folders were open under the new path.
func (m *Model) remapExpandedPaths(oldPath, newPath string) {
	if oldPath == "" || newPath == "" || oldPath == newPath {
		return
	}
	remapped := make(map[string]bool, len(m.expanded))
	for path, expanded := range m.expanded {
		remapped[replacePathPrefix(path, oldPath, newPath)] = expanded
	}
	m.expanded = remapped
}

// replacePathPrefix swaps oldPrefix for newPrefix at the start of path.
// If path equals oldPrefix exactly, newPrefix is returned. If path is a
// descendant (starts with oldPrefix + separator), only the prefix portion
// is replaced. Otherwise, path is returned unchanged.
func replacePathPrefix(path, oldPrefix, newPrefix string) string {
	if path == "" || oldPrefix == "" || newPrefix == "" {
		return path
	}
	if path == oldPrefix {
		return newPrefix
	}
	withSep := oldPrefix + string(os.PathSeparator)
	if !strings.HasPrefix(path, withSep) {
		return path
	}
	return newPrefix + path[len(oldPrefix):]
}
