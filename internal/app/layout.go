package app

// LayoutDimensions holds all calculated layout dimensions for the UI.
type LayoutDimensions struct {
	LeftWidth     int
	RightWidth    int
	ContentHeight int
	ViewportWidth int
	ViewportHeight int
}

// calculateLayout computes all UI dimensions based on terminal size and mode.
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

// applyLayout updates the viewport dimensions based on calculated layout.
func (m *Model) applyLayout(layout LayoutDimensions) {
	m.viewport.Width = layout.ViewportWidth
	m.viewport.Height = layout.ViewportHeight
}
