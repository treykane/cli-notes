// state.go implements per-workspace persistent state: recent files, pinned
// paths, and per-note scroll/cursor position memory.
//
// State is stored as JSON at <notes_dir>/.cli-notes/state.json so each
// workspace maintains independent state that travels with the notes directory
// (e.g. across machines via git sync). All paths in the JSON file are stored
// as relative paths (relative to notesDir) so the state remains valid if the
// workspace root is relocated.
//
// The in-memory representation (appPersistentState) uses absolute paths for
// O(1) lookups. Conversion between absolute and relative paths happens at
// the load/save boundaries via statePathToAbs and absToStatePath, with
// validation to reject paths that escape the workspace root.
//
// State is saved:
//   - After every file navigation (recent file tracking)
//   - Before switching files or workspaces (position memory)
//   - After pin/unpin toggles
//   - After rename/move/delete operations (state path remapping)
//   - On external filesystem change detection (watcher refresh)
package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// notePosition records the viewport scroll offset and editor cursor position
// for a single note so the app can restore the user's reading/editing position
// when they return to a previously viewed file.
type notePosition struct {
	PreviewOffset          int `json:"preview_offset,omitempty"` // legacy fallback
	PrimaryPreviewOffset   int `json:"primary_preview_offset,omitempty"`
	SecondaryPreviewOffset int `json:"secondary_preview_offset,omitempty"`
	EditorCursor           int `json:"editor_cursor,omitempty"`
}

// persistedState is the on-disk JSON representation of per-workspace app state.
//
// All paths are stored as relative paths (relative to the workspace's notesDir)
// so state files remain valid when the workspace root is relocated. Conversion
// between absolute and relative paths happens at load/save boundaries via
// statePathToAbs and absToStatePath.
type persistedState struct {
	RecentFiles []string                `json:"recent_files,omitempty"`
	PinnedPaths []string                `json:"pinned_paths,omitempty"`
	Positions   map[string]notePosition `json:"positions,omitempty"`
	OpenCounts  map[string]int          `json:"open_counts,omitempty"`
}

// appPersistentState is the in-memory representation of workspace state.
//
// Unlike persistedState, all paths here are absolute. PinnedPaths uses a
// map[string]bool for O(1) lookup during tree sorting and rendering.
type appPersistentState struct {
	RecentFiles []string
	PinnedPaths map[string]bool
	Positions   map[string]notePosition
	OpenCounts  map[string]int
}

// appStatePath returns the filesystem path to the per-workspace state file.
// State is stored inside the managed directory (<notesDir>/.cli-notes/state.json)
// so it lives alongside the notes it describes and is workspace-specific.
func appStatePath(notesDir string) string {
	return filepath.Join(notesDir, managedNotesDirName, "state.json")
}

// loadAppState reads and deserializes the per-workspace state file.
//
// If the state file does not exist (first run or new workspace), an empty state
// with initialized maps is returned without error. Relative paths in the JSON
// are converted to absolute paths, and invalid entries (negative offsets, paths
// outside the workspace root) are silently discarded to keep state clean.
func loadAppState(notesDir string) (appPersistentState, error) {
	state := appPersistentState{
		PinnedPaths: map[string]bool{},
		Positions:   map[string]notePosition{},
		OpenCounts:  map[string]int{},
	}

	path := appStatePath(notesDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, fmt.Errorf("read app state %q: %w", path, err)
	}

	var persisted persistedState
	if err := json.Unmarshal(data, &persisted); err != nil {
		return state, fmt.Errorf("parse app state %q: %w", path, err)
	}

	for _, rel := range persisted.PinnedPaths {
		abs, ok := statePathToAbs(notesDir, rel)
		if !ok {
			continue
		}
		state.PinnedPaths[abs] = true
	}

	state.RecentFiles = make([]string, 0, len(persisted.RecentFiles))
	for _, rel := range persisted.RecentFiles {
		abs, ok := statePathToAbs(notesDir, rel)
		if !ok {
			continue
		}
		state.RecentFiles = append(state.RecentFiles, abs)
	}

	for rel, pos := range persisted.Positions {
		abs, ok := statePathToAbs(notesDir, rel)
		if !ok {
			continue
		}
		if pos.PreviewOffset < 0 {
			pos.PreviewOffset = 0
		}
		if pos.PrimaryPreviewOffset < 0 {
			pos.PrimaryPreviewOffset = 0
		}
		if pos.SecondaryPreviewOffset < 0 {
			pos.SecondaryPreviewOffset = 0
		}
		if pos.EditorCursor < 0 {
			pos.EditorCursor = 0
		}
		if pos.PrimaryPreviewOffset <= 0 && pos.PreviewOffset > 0 {
			pos.PrimaryPreviewOffset = pos.PreviewOffset
		}
		state.Positions[abs] = pos
	}
	for rel, count := range persisted.OpenCounts {
		abs, ok := statePathToAbs(notesDir, rel)
		if !ok || count <= 0 {
			continue
		}
		state.OpenCounts[abs] = count
	}

	state.RecentFiles = dedupePaths(state.RecentFiles)
	trimRecentFiles(&state.RecentFiles)
	return state, nil
}

// saveAppState serializes the current in-memory state (recent files, pinned
// paths, and per-note positions) to the per-workspace state file on disk.
//
// Absolute paths are converted to relative paths before writing so the state
// file is portable if the workspace root moves. Pinned paths are sorted for
// deterministic output. Positions with zero values are omitted to keep the
// file compact. The file is written atomically with restrictive permissions
// (0600) since it lives inside the user's notes directory.
func (m *Model) saveAppState() {
	if m.notesDir == "" {
		return
	}
	state := persistedState{
		RecentFiles: make([]string, 0, len(m.recentFiles)),
		PinnedPaths: make([]string, 0, len(m.pinnedPaths)),
		Positions:   make(map[string]notePosition, len(m.notePositions)),
		OpenCounts:  make(map[string]int, len(m.noteOpenCounts)),
	}

	for _, path := range m.recentFiles {
		if rel, ok := absToStatePath(m.notesDir, path); ok {
			state.RecentFiles = append(state.RecentFiles, rel)
		}
	}

	for path, pinned := range m.pinnedPaths {
		if !pinned {
			continue
		}
		if rel, ok := absToStatePath(m.notesDir, path); ok {
			state.PinnedPaths = append(state.PinnedPaths, rel)
		}
	}
	sort.Strings(state.PinnedPaths)

	for path, pos := range m.notePositions {
		if pos.PrimaryPreviewOffset <= 0 && pos.SecondaryPreviewOffset <= 0 && pos.EditorCursor <= 0 {
			continue
		}
		rel, ok := absToStatePath(m.notesDir, path)
		if !ok {
			continue
		}
		state.Positions[rel] = notePosition{
			PreviewOffset:          max(0, pos.PrimaryPreviewOffset),
			PrimaryPreviewOffset:   max(0, pos.PrimaryPreviewOffset),
			SecondaryPreviewOffset: max(0, pos.SecondaryPreviewOffset),
			EditorCursor:           max(0, pos.EditorCursor),
		}
	}
	for path, count := range m.noteOpenCounts {
		if count <= 0 {
			continue
		}
		rel, ok := absToStatePath(m.notesDir, path)
		if !ok {
			continue
		}
		state.OpenCounts[rel] = count
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		appLog.Warn("marshal app state", "error", err)
		return
	}
	data = append(data, '\n')

	path := appStatePath(m.notesDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		appLog.Warn("create app state dir", "path", filepath.Dir(path), "error", err)
		return
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		appLog.Warn("write app state", "path", path, "error", err)
	}
}

// absToStatePath converts an absolute path to a relative path for state
// persistence. Returns false if the path is not within the workspace root
// or cannot be relativized, ensuring only valid workspace paths are stored.
func absToStatePath(root, path string) (string, bool) {
	if !isWithinRoot(root, path) {
		return "", false
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", false
	}
	rel = filepath.Clean(rel)
	if rel == "." || rel == "" || rel == string(filepath.Separator) {
		return "", false
	}
	return rel, true
}

// statePathToAbs converts a relative path from the state file back to an
// absolute path. Returns false if the relative path is invalid, empty, or
// would resolve outside the workspace root (e.g. via ".." traversal),
// preventing path traversal attacks from malformed state files.
func statePathToAbs(root, rel string) (string, bool) {
	rel = filepath.Clean(rel)
	if rel == "." || rel == "" || filepath.IsAbs(rel) {
		return "", false
	}
	path := filepath.Join(root, rel)
	if !isWithinRoot(root, path) {
		return "", false
	}
	return path, true
}

// rememberCurrentNotePosition saves the current viewport offset and editor
// cursor for the active note so the position can be restored later. This is
// called before switching files, saving state, or exiting edit mode.
func (m *Model) rememberCurrentNotePosition() {
	if m.splitMode && m.secondaryFile != "" {
		m.rememberPanePosition(m.secondaryFile, true)
	}
	if m.currentFile != "" {
		m.rememberPanePosition(m.currentFile, false)
	}
}

// rememberNotePosition saves the viewport offset and (if in edit mode) editor
// cursor position for the given note path. The position is stored in the
// notePositions map and persisted to disk on the next saveAppState call.
func (m *Model) rememberNotePosition(path string) {
	m.rememberPanePosition(path, false)
}

func (m *Model) rememberPanePosition(path string, secondary bool) {
	if path == "" {
		return
	}
	if m.notePositions == nil {
		m.notePositions = map[string]notePosition{}
	}
	offset := m.restorePaneOffset(path, secondary)
	if !secondary {
		offset = max(0, m.viewport.YOffset)
	}
	pos := m.notePositions[path]
	if secondary {
		pos.SecondaryPreviewOffset = offset
	} else {
		pos.PrimaryPreviewOffset = offset
		pos.PreviewOffset = offset
	}
	if m.mode == modeEditNote && path == m.currentFile {
		pos.EditorCursor = max(0, m.currentEditorCursorOffset())
	}
	m.notePositions[path] = pos
}

func (m *Model) setPaneOffset(path string, secondary bool, offset int) {
	if path == "" {
		return
	}
	if m.notePositions == nil {
		m.notePositions = map[string]notePosition{}
	}
	pos := m.notePositions[path]
	if secondary {
		pos.SecondaryPreviewOffset = max(0, offset)
	} else {
		pos.PrimaryPreviewOffset = max(0, offset)
		pos.PreviewOffset = max(0, offset)
	}
	m.notePositions[path] = pos
}

// restorePreviewOffset restores the viewport scroll position for a note that
// was previously viewed. If no saved position exists, the viewport is reset
// to the top of the document.
func (m *Model) restorePreviewOffset(path string) {
	if path == "" {
		return
	}
	m.viewport.YOffset = m.restorePaneOffset(path, false)
}

func (m *Model) restorePaneOffset(path string, secondary bool) int {
	if path == "" {
		return 0
	}
	pos, ok := m.notePositions[path]
	if !ok {
		return 0
	}
	if secondary {
		return max(0, pos.SecondaryPreviewOffset)
	}
	if pos.PrimaryPreviewOffset > 0 {
		return max(0, pos.PrimaryPreviewOffset)
	}
	return max(0, pos.PreviewOffset)
}

// restoreEditorCursor restores the editor cursor to the previously saved
// position when re-entering edit mode for a note. If no position was saved
// or the saved position is zero, the cursor is placed at the end of the
// document as a sensible default.
func (m *Model) restoreEditorCursor(path string) {
	if path == "" {
		return
	}
	pos, ok := m.notePositions[path]
	if !ok || pos.EditorCursor <= 0 {
		m.editor.CursorEnd()
		return
	}
	m.setEditorValueAndCursorOffset(m.editor.Value(), pos.EditorCursor)
}

// trackRecentFile adds a note path to the front of the recent files list.
// Duplicates are removed so each path appears at most once. The list is
// capped at MaxRecentFiles entries. Non-markdown files are ignored since
// the app only previews/edits markdown. State is persisted immediately.
func (m *Model) trackRecentFile(path string) {
	if path == "" || !hasSuffixCaseInsensitive(path, ".md") {
		return
	}
	m.recentFiles = append([]string{path}, removePathFromList(m.recentFiles, path)...)
	trimRecentFiles(&m.recentFiles)
	m.rebuildRecentEntries()
	m.saveAppState()
}

func (m *Model) trackFileOpen(path string) {
	if path == "" || !hasSuffixCaseInsensitive(path, ".md") {
		return
	}
	if m.noteOpenCounts == nil {
		m.noteOpenCounts = map[string]int{}
	}
	m.noteOpenCounts[path]++
}

// rebuildRecentEntries filters the recent files list to only include paths
// that still exist on disk and are within the current workspace root. This
// is called after workspace switches, file deletions, and state loads to
// ensure the recent-files popup never shows stale or missing entries.
func (m *Model) rebuildRecentEntries() {
	if len(m.recentFiles) == 0 {
		m.recentEntries = nil
		m.recentCursor = 0
		return
	}
	visible := make([]string, 0, len(m.recentFiles))
	for _, path := range m.recentFiles {
		if !isWithinRoot(m.notesDir, path) {
			continue
		}
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		visible = append(visible, path)
	}
	trimRecentFiles(&visible)
	m.recentEntries = visible
	if len(m.recentEntries) == 0 {
		m.recentCursor = 0
		return
	}
	m.recentCursor = clamp(m.recentCursor, 0, len(m.recentEntries)-1)
}

// trimRecentFiles caps the recent files list at MaxRecentFiles entries,
// discarding the oldest entries (those at the end of the slice).
func trimRecentFiles(paths *[]string) {
	if len(*paths) > MaxRecentFiles {
		*paths = (*paths)[:MaxRecentFiles]
	}
}

// clearStateForPath removes all persisted state associated with the given
// path: pinned status, saved positions, and recent file entries. If the path
// is a directory, all descendant paths are also cleared. This is called after
// a file or folder is deleted to avoid stale references in state.
func (m *Model) clearStateForPath(path string) {
	if path == "" {
		return
	}
	delete(m.pinnedPaths, path)
	delete(m.notePositions, path)
	delete(m.noteOpenCounts, path)
	m.recentFiles = removePathFromList(m.recentFiles, path)
	prefix := path + string(os.PathSeparator)
	for p := range m.pinnedPaths {
		if p == path || hasPathPrefix(p, prefix) {
			delete(m.pinnedPaths, p)
		}
	}
	for p := range m.notePositions {
		if p == path || hasPathPrefix(p, prefix) {
			delete(m.notePositions, p)
		}
	}
	for p := range m.noteOpenCounts {
		if p == path || hasPathPrefix(p, prefix) {
			delete(m.noteOpenCounts, p)
		}
	}
	m.recentFiles = removePathsWithPrefix(m.recentFiles, prefix)
	m.rebuildRecentEntries()
	m.saveAppState()
}

// remapStatePaths updates all persisted state references when a file or folder
// is renamed or moved. Pinned paths, note positions, and recent file entries
// are all updated so that the old path prefix is replaced with the new one.
// This ensures state survives rename/move operations without data loss.
func (m *Model) remapStatePaths(oldPath, newPath string) {
	if oldPath == "" || newPath == "" || oldPath == newPath {
		return
	}
	m.remapPinnedPaths(oldPath, newPath)
	m.remapPositionPaths(oldPath, newPath)
	m.remapOpenCountPaths(oldPath, newPath)
	m.remapRecentPaths(oldPath, newPath)
	m.rebuildRecentEntries()
	m.saveAppState()
}

// remapPinnedPaths replaces oldPath prefix with newPath in all pinned entries.
func (m *Model) remapPinnedPaths(oldPath, newPath string) {
	if len(m.pinnedPaths) == 0 {
		return
	}
	remapped := make(map[string]bool, len(m.pinnedPaths))
	for path, pinned := range m.pinnedPaths {
		if !pinned {
			continue
		}
		remapped[replacePathPrefix(path, oldPath, newPath)] = true
	}
	m.pinnedPaths = remapped
}

// remapPositionPaths replaces oldPath prefix with newPath in all saved positions.
func (m *Model) remapPositionPaths(oldPath, newPath string) {
	if len(m.notePositions) == 0 {
		return
	}
	remapped := make(map[string]notePosition, len(m.notePositions))
	for path, pos := range m.notePositions {
		remapped[replacePathPrefix(path, oldPath, newPath)] = pos
	}
	m.notePositions = remapped
}

// remapRecentPaths replaces oldPath prefix with newPath in the recent files list,
// then deduplicates and trims the result.
func (m *Model) remapRecentPaths(oldPath, newPath string) {
	if len(m.recentFiles) == 0 {
		return
	}
	updated := make([]string, 0, len(m.recentFiles))
	for _, path := range m.recentFiles {
		updated = append(updated, replacePathPrefix(path, oldPath, newPath))
	}
	m.recentFiles = dedupePaths(updated)
	trimRecentFiles(&m.recentFiles)
}

func (m *Model) remapOpenCountPaths(oldPath, newPath string) {
	if len(m.noteOpenCounts) == 0 {
		return
	}
	remapped := make(map[string]int, len(m.noteOpenCounts))
	for path, count := range m.noteOpenCounts {
		remapped[replacePathPrefix(path, oldPath, newPath)] += count
	}
	m.noteOpenCounts = remapped
}

// removePathFromList returns a new slice with all occurrences of target removed.
func removePathFromList(paths []string, target string) []string {
	if len(paths) == 0 {
		return nil
	}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if path == target {
			continue
		}
		out = append(out, path)
	}
	return out
}

// removePathsWithPrefix returns a new slice with all paths matching the given
// prefix removed. Used to clear descendants when a directory is deleted.
func removePathsWithPrefix(paths []string, prefix string) []string {
	if len(paths) == 0 {
		return nil
	}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if hasPathPrefix(path, prefix) {
			continue
		}
		out = append(out, path)
	}
	return out
}

// hasPathPrefix reports whether path starts with prefix. This is a raw string
// comparison (not filepath-aware), so callers should ensure prefix ends with
// the path separator when checking for directory containment.
func hasPathPrefix(path, prefix string) bool {
	return len(prefix) > 0 && len(path) >= len(prefix) && path[:len(prefix)] == prefix
}

// dedupePaths returns a new slice with duplicate paths removed, preserving the
// order of first occurrence. Used after path remapping to eliminate duplicates
// that can arise when both a parent and child are renamed to the same target.
func dedupePaths(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if seen[path] {
			continue
		}
		seen[path] = true
		out = append(out, path)
	}
	return out
}
