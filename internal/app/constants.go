package app

import "time"

// Layout constants define the default dimensions and spacing for the UI
const (
	// DefaultTreeWidth is the minimum width allocated to the tree view
	DefaultTreeWidth = 40

	// TreeWidthDivider determines tree width as terminal_width / this value
	// when terminal is wide enough
	TreeWidthDivider = 3

	// SearchPopupPadding is the horizontal padding inside the search popup
	SearchPopupPadding = 8

	// SearchPopupHeight is the fixed height of the search popup
	SearchPopupHeight = 10
	// RecentPopupHeight is the fixed height of the recent-files popup.
	RecentPopupHeight = 12
	// OutlinePopupHeight is the fixed height of the heading outline popup.
	OutlinePopupHeight = 14
	// WorkspacePopupHeight is the fixed height of workspace chooser popup.
	WorkspacePopupHeight = 12
	// ExportPopupHeight is the fixed height of export chooser popup.
	ExportPopupHeight = 8
	// WikiLinksPopupHeight is the fixed height of wiki links popup.
	WikiLinksPopupHeight = 14
	// WikiAutocompletePopupHeight is popup height for edit autocomplete.
	WikiAutocompletePopupHeight = 10

	// FooterMinRows is the default number of rows reserved for the bottom
	// status/help area. The app targets two rows on typical terminal widths.
	FooterMinRows = 2
	// FooterMaxRows is the expanded footer height used when content does not
	// fit within FooterMinRows.
	FooterMaxRows = 3
)

// Input limits define maximum sizes for user input
const (
	// InputCharLimit is the maximum number of characters allowed in text inputs
	InputCharLimit = 120
)

// Rendering constants control render timing and optimization
const (
	// RenderDebounce is the delay before triggering a render after window resize
	RenderDebounce = 500 * time.Millisecond

	// RenderWidthBucket is the granularity for width-based render caching
	// Widths are rounded to nearest multiple of this value
	RenderWidthBucket = 20
)

// File system permissions
const (
	// DirPermission is the permission mode for newly created directories
	DirPermission = 0o755

	// FilePermission is the permission mode for newly created files
	FilePermission = 0o644
)

// Search constants
const (
	// MaxSearchFileBytes is the maximum file size (in bytes) that will be
	// searched. Files larger than this are skipped.
	MaxSearchFileBytes = 1024 * 1024 // 1 MB
	// MaxRecentFiles is the maximum number of recent files retained in state.
	MaxRecentFiles = 20
)

// Watcher constants
const (
	// FileWatchInterval is the poll interval for external filesystem changes.
	FileWatchInterval = 2 * time.Second
)
