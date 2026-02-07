// sort.go implements sort-mode cycling and persistence for the tree view.
//
// The user presses `s` in browse mode to cycle through sort modes
// (name → modified → size → created → name). The chosen mode is persisted
// in config.json under "tree_sort_by_workspace" keyed by notes_dir, with
// "tree_sort" kept as compatibility fallback. After changing the sort mode
// the tree is rebuilt immediately to reflect the new ordering.
package app

import (
	"fmt"

	"github.com/treykane/cli-notes/internal/config"
)

// cycleSortMode advances to the next sort mode, rebuilds the tree to apply
// the new ordering, and persists the per-workspace preference to config.json.
// If the config save fails the sort mode is still applied in-memory for the
// current session.
func (m *Model) cycleSortMode() {
	m.sortMode = nextSortMode(m.sortMode)
	m.refreshTree()
	if err := m.persistWorkspaceSortMode(); err != nil {
		m.setStatusError("Sort mode changed but config save failed", err)
		return
	}
	m.status = fmt.Sprintf("Tree sort: %s", m.sortMode.Label())
}

func loadWorkspaceSortMode(cfg config.Config, notesDir string) sortMode {
	if notesDir != "" {
		if mode, ok := cfg.TreeSortByWorkspace[notesDir]; ok {
			return parseSortMode(mode)
		}
	}
	return parseSortMode(cfg.TreeSort)
}

// persistWorkspaceSortMode writes the current sort mode to the workspace map
// and updates the legacy tree_sort field as fallback compatibility.
func (m *Model) persistWorkspaceSortMode() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.TreeSort = m.sortMode.String()
	if cfg.TreeSortByWorkspace == nil {
		cfg.TreeSortByWorkspace = map[string]string{}
	}
	if m.notesDir != "" {
		cfg.TreeSortByWorkspace[m.notesDir] = m.sortMode.String()
	}
	if cfg.TemplatesDir == "" {
		cfg.TemplatesDir = m.templatesDir
	}
	return config.Save(cfg)
}
