// sort.go implements sort-mode cycling and persistence for the tree view.
//
// The user presses `s` in browse mode to cycle through sort modes
// (name → modified → size → created → name). The chosen mode is persisted
// in config.json under "tree_sort" so it survives restarts. After changing
// the sort mode the tree is rebuilt immediately to reflect the new ordering.
package app

import (
	"fmt"

	"github.com/treykane/cli-notes/internal/config"
)

// cycleSortMode advances to the next sort mode, rebuilds the tree to apply
// the new ordering, and persists the preference to config.json. If the config
// save fails the sort mode is still applied in-memory for the current session.
func (m *Model) cycleSortMode() {
	m.sortMode = nextSortMode(m.sortMode)
	m.refreshTree()
	if err := m.persistSortMode(); err != nil {
		m.setStatusError("Sort mode changed but config save failed", err)
		return
	}
	m.status = fmt.Sprintf("Tree sort: %s", m.sortMode.Label())
}

// persistSortMode writes the current sort mode to the config file so it is
// restored on next launch. The config is loaded, updated, and saved to avoid
// overwriting other settings that may have changed externally.
func (m *Model) persistSortMode() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	cfg.TreeSort = m.sortMode.String()
	if cfg.TemplatesDir == "" {
		cfg.TemplatesDir = m.templatesDir
	}
	return config.Save(cfg)
}
