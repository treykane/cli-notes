package app

var overlayRenderers = map[overlayMode]func(*Model, int, int) string{
	overlaySearch:           (*Model).renderSearchPopupOverlay,
	overlayRecent:           (*Model).renderRecentPopupOverlay,
	overlayOutline:          (*Model).renderOutlinePopupOverlay,
	overlayWorkspace:        (*Model).renderWorkspacePopupOverlay,
	overlayExport:           (*Model).renderExportPopupOverlay,
	overlayWikiLinks:        (*Model).renderWikiLinksPopupOverlay,
	overlayWikiAutocomplete: (*Model).renderWikiAutocompletePopupOverlay,
}

func (m *Model) renderActiveOverlay(width, height int) string {
	if render, ok := overlayRenderers[m.overlay]; ok {
		return render(m, width, height)
	}
	return ""
}
