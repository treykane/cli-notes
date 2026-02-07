package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type notePosition struct {
	PreviewOffset int `json:"preview_offset,omitempty"`
	EditorCursor  int `json:"editor_cursor,omitempty"`
}

type persistedState struct {
	RecentFiles []string                `json:"recent_files,omitempty"`
	PinnedPaths []string                `json:"pinned_paths,omitempty"`
	Positions   map[string]notePosition `json:"positions,omitempty"`
}

type appPersistentState struct {
	RecentFiles []string
	PinnedPaths map[string]bool
	Positions   map[string]notePosition
}

func appStatePath(notesDir string) string {
	return filepath.Join(notesDir, managedNotesDirName, "state.json")
}

func loadAppState(notesDir string) (appPersistentState, error) {
	state := appPersistentState{
		PinnedPaths: map[string]bool{},
		Positions:   map[string]notePosition{},
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
		if pos.EditorCursor < 0 {
			pos.EditorCursor = 0
		}
		state.Positions[abs] = pos
	}

	state.RecentFiles = dedupePaths(state.RecentFiles)
	trimRecentFiles(&state.RecentFiles)
	return state, nil
}

func (m *Model) saveAppState() {
	if m.notesDir == "" {
		return
	}
	state := persistedState{
		RecentFiles: make([]string, 0, len(m.recentFiles)),
		PinnedPaths: make([]string, 0, len(m.pinnedPaths)),
		Positions:   make(map[string]notePosition, len(m.notePositions)),
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
		if pos.PreviewOffset <= 0 && pos.EditorCursor <= 0 {
			continue
		}
		rel, ok := absToStatePath(m.notesDir, path)
		if !ok {
			continue
		}
		state.Positions[rel] = notePosition{
			PreviewOffset: max(0, pos.PreviewOffset),
			EditorCursor:  max(0, pos.EditorCursor),
		}
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

func (m *Model) rememberCurrentNotePosition() {
	if m.currentFile == "" {
		return
	}
	m.rememberNotePosition(m.currentFile)
}

func (m *Model) rememberNotePosition(path string) {
	if path == "" {
		return
	}
	if m.notePositions == nil {
		m.notePositions = map[string]notePosition{}
	}
	pos := m.notePositions[path]
	pos.PreviewOffset = max(0, m.viewport.YOffset)
	if m.mode == modeEditNote && path == m.currentFile {
		pos.EditorCursor = max(0, m.currentEditorCursorOffset())
	}
	m.notePositions[path] = pos
}

func (m *Model) restorePreviewOffset(path string) {
	if path == "" {
		return
	}
	pos, ok := m.notePositions[path]
	if !ok {
		m.viewport.YOffset = 0
		return
	}
	m.viewport.YOffset = max(0, pos.PreviewOffset)
}

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

func (m *Model) trackRecentFile(path string) {
	if path == "" || !hasSuffixCaseInsensitive(path, ".md") {
		return
	}
	m.recentFiles = append([]string{path}, removePathFromList(m.recentFiles, path)...)
	trimRecentFiles(&m.recentFiles)
	m.rebuildRecentEntries()
	m.saveAppState()
}

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

func trimRecentFiles(paths *[]string) {
	if len(*paths) > MaxRecentFiles {
		*paths = (*paths)[:MaxRecentFiles]
	}
}

func (m *Model) clearStateForPath(path string) {
	if path == "" {
		return
	}
	delete(m.pinnedPaths, path)
	delete(m.notePositions, path)
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
	m.recentFiles = removePathsWithPrefix(m.recentFiles, prefix)
	m.rebuildRecentEntries()
	m.saveAppState()
}

func (m *Model) remapStatePaths(oldPath, newPath string) {
	if oldPath == "" || newPath == "" || oldPath == newPath {
		return
	}
	m.remapPinnedPaths(oldPath, newPath)
	m.remapPositionPaths(oldPath, newPath)
	m.remapRecentPaths(oldPath, newPath)
	m.rebuildRecentEntries()
	m.saveAppState()
}

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

func hasPathPrefix(path, prefix string) bool {
	return len(prefix) > 0 && len(path) >= len(prefix) && path[:len(prefix)] == prefix
}

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
