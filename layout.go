package main

const (
	tabBarHeight     = 3 // one content row + top/bottom border
	helpBarHeight    = 1
	maxTabLabelWidth = 35

	fieldChrome   = 4 // rounded border (2) + horizontal padding (2)
	labelColWidth = 10
	labelGap      = 1
)

func (a *AppModel) panelInnerWidth() int {
	panelChromeW := 2 + 2*2
	return max(100, a.width-panelChromeW)
}

func (a *AppModel) panelInnerHeight() int {
	return max(20, a.height-tabBarHeight-helpBarHeight)
}

func (a *AppModel) termSize() (cols, rows int) {
	return a.panelInnerWidth(), a.panelInnerHeight()
}

func (a *AppModel) fieldBoxWidth() int {
	return max(HostCharLimit, PortCharLimit, UserCharLimit, SecretCharLimit) + fieldChrome + 1
}

func (a *AppModel) fieldInnerWidth() int {
	return max(1, a.fieldBoxWidth()-fieldStyle.GetHorizontalFrameSize())
}

// syncInputWidths tells every tab's textinputs how many columns they actually
// got, so a value longer than its box scrolls inside it instead of wrapping.
// All tabs share the same geometry (only the active one is ever drawn, but
// any of them can become active without another resize event happening
// first), so every tab is kept in sync, not just the active one.
func (a *AppModel) syncInputWidths() {
	w := a.fieldInnerWidth()
	for i := range a.tabs {
		t := &a.tabs[i]
		t.Host.SetWidth(w)
		t.Port.SetWidth(w)
		t.User.SetWidth(w)
		t.Secret.SetWidth(w)
	}
}
