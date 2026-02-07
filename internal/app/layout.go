// layout.go centralizes all terminal layout calculations for the two-pane UI.
//
// The UI is laid out as a horizontal split: a fixed-width tree pane on the left
// and a flexible content pane on the right. One row at the bottom is reserved
// for the footer status bar (hence contentHeight = height - 1).
//
// The right pane's usable area depends on the active mode because preview and
// edit modes use different Lipgloss border styles with different frame sizes.
// An additional row is subtracted for the right-pane header bar that shows the
// current file path.
//
// All dimension calculations are gathered into a single LayoutDimensions struct
// so they can be computed once per resize and reused by View, updateLayout, and
// handleWindowResize without redundant arithmetic.
package app

// LayoutDimensions holds all calculated layout dimensions for the UI.
//
// These values are derived from the current terminal width/height and the
// active mode's border style. They are recalculated on every window resize
// and consumed by the View function and its helpers.
type LayoutDimensions struct {
	LeftWidth      int // width allocated to the tree pane (including border/padding)
	RightWidth     int // width allocated to the right pane (remainder after tree)
	ContentHeight  int // total height available for pane content (terminal height minus footer)
	ViewportWidth  int // usable width inside the right pane (after border/padding)
	ViewportHeight int // usable height inside the right pane (after border/padding and header)
}

// calculateLayout computes all UI dimensions based on terminal size and mode.
//
// The tree pane width is the smaller of DefaultTreeWidth and terminal_width / TreeWidthDivider,
// so narrow terminals still get a usable tree. The right pane fills the remaining space.
// The viewport dimensions account for the active pane style's border and padding
// (which differ between preview and edit modes) and subtract one row for the
// right-pane header bar.
func (m *Model) calculateLayout() LayoutDimensions {
	leftWidth := min(DefaultTreeWidth, m.width/TreeWidthDivider)
	rightWidth := max(0, m.width-leftWidth)
	contentHeight := max(0, m.height-1)

	rightPaneStyle := previewPane
	if m.mode == modeEditNote {
		rightPaneStyle = editPane
	}

	viewportWidth := max(0, rightWidth-rightPaneStyle.GetHorizontalFrameSize())
	viewportHeight := max(0, contentHeight-rightPaneStyle.GetVerticalFrameSize()-1)

	return LayoutDimensions{
		LeftWidth:      leftWidth,
		RightWidth:     rightWidth,
		ContentHeight:  contentHeight,
		ViewportWidth:  viewportWidth,
		ViewportHeight: viewportHeight,
	}
}

// applyLayout updates the viewport widget dimensions to match the calculated
// layout. This is called after every window resize so the viewport knows how
// many columns and rows of rendered markdown it can display.
func (m *Model) applyLayout(layout LayoutDimensions) {
	m.viewport.Width = layout.ViewportWidth
	m.viewport.Height = layout.ViewportHeight
}
