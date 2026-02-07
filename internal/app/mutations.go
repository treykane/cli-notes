package app

import tea "github.com/charmbracelet/bubbletea"

type mutationEffects struct {
	upsertPaths      []string
	removePaths      []string
	invalidateSearch bool
	refreshTree      bool
	rebuildKeepPath  string
	refreshGit       bool
	saveState        bool
	clearRenderCache bool
	setCurrentFile   string
}

// applyMutationEffects centralizes post-filesystem-mutation side effects to keep update flows consistent.
func (m *Model) applyMutationEffects(opts mutationEffects) tea.Cmd {
	if opts.saveState {
		m.saveAppState()
	}

	if m.searchIndex != nil {
		if opts.invalidateSearch {
			m.searchIndex.invalidate()
		}
		for _, path := range opts.removePaths {
			if path != "" {
				m.searchIndex.removePath(path)
			}
		}
		for _, path := range opts.upsertPaths {
			if path != "" {
				m.searchIndex.upsertPath(path)
			}
		}
	}

	if opts.clearRenderCache {
		m.renderCache = map[string]renderCacheEntry{}
	}
	if opts.refreshGit {
		m.refreshGitStatus()
	}
	if opts.refreshTree {
		m.refreshTree()
	}
	if opts.rebuildKeepPath != "" {
		m.rebuildTreeKeep(opts.rebuildKeepPath)
	}
	if opts.setCurrentFile != "" {
		return m.setCurrentFile(opts.setCurrentFile)
	}
	return nil
}
