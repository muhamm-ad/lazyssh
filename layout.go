// Geometry: how the window is divided between the tab bar and the content
// panel, and how a form row's label column and input box share the panel's
// width. Every number here is in terminal cells.
//
// Unlike the old two-stacked-frames layout, the panel is a single box that
// always fills whatever's left below the tab bar — the form's content sits
// top-aligned inside it, and the terminal fills it completely. That keeps the
// panel's footprint stable across the form/terminal swap instead of the whole
// window resizing every time a tab connects or disconnects.

package main

const (
	fieldChrome   = 4 // rounded border (2) + horizontal padding (2)
	labelColWidth = 10
	labelGap      = 1

	// What panelStyle eats out of the space it's given: a border on each side,
	// plus Padding(1, 2).
	panelChromeW = 2 + 2*2
	panelChromeH = 2 + 2*1

	tabBarHeight = 3 // one content row + top/bottom border

	minPanelInner   = 10 // floor so the form's fixed rows always have room
	maxTabLabelWidth = 24
)

// pageWidth is the window width the layout is drawn against, floored so a very
// narrow terminal still produces a sane frame.
func (a *appModel) pageWidth() int {
	return max(40, a.width)
}

// panelInnerWidth is the usable width inside the content panel. The panel
// spans the whole window, so this has to agree with the width handed to
// panelStyle — disagree by even one column and a row wraps onto its own line.
func (a *appModel) panelInnerWidth() int {
	return max(20, a.pageWidth()-panelChromeW)
}

// panelInnerHeight is the content panel's height: everything below the tab
// bar, floored so the form's fixed set of rows always fits.
func (a *appModel) panelInnerHeight() int {
	return max(minPanelInner, a.height-tabBarHeight-panelChromeH)
}

// fieldBoxWidth is the total width (border and padding included) of every
// field box in the form — host, port, user, method and secret all share it,
// same as the design, where every input spans the same right edge.
func (a *appModel) fieldBoxWidth() int {
	return max(fieldChrome+1, a.panelInnerWidth()-labelColWidth-labelGap)
}

// fieldInputWidth is what's left for the textinput itself once its box's
// border and padding are subtracted.
func (a *appModel) fieldInputWidth() int {
	return max(1, a.fieldBoxWidth()-fieldChrome)
}

// termSize is the inner size of the live terminal content area: the panel's
// content box, minus the header row (target + connected badge), the footer
// hint row, and a blank line around each.
func (a *appModel) termSize() (cols, rows int) {
	cols = a.panelInnerWidth()
	rows = max(3, a.panelInnerHeight()-4)
	return cols, rows
}

// syncInputWidths tells every tab's textinputs how many columns they actually
// got, so a value longer than its box scrolls inside it instead of wrapping.
// All tabs share the same geometry (only the active one is ever drawn, but
// any of them can become active without another resize event happening
// first), so every tab is kept in sync, not just the active one.
func (a *appModel) syncInputWidths() {
	w := a.fieldInputWidth()
	for i := range a.tabs {
		t := &a.tabs[i]
		t.host.SetWidth(w)
		t.port.SetWidth(w)
		t.user.SetWidth(w)
		t.secret.SetWidth(w)
	}
}
