package app

import (
	"fmt"

	"github.com/treykane/cli-notes/internal/config"
)

func (m *Model) cycleSortMode() {
	m.sortMode = nextSortMode(m.sortMode)
	m.refreshTree()
	if err := m.persistSortMode(); err != nil {
		m.setStatusError("Sort mode changed but config save failed", err)
		return
	}
	m.status = fmt.Sprintf("Tree sort: %s", m.sortMode.Label())
}

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
