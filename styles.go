// Colors and styles, transcribed from design/SSH_TUI.html. Keeping them in one
// place means the design can be re-checked against this file alone.
//
// Unlike the single-shared-background version this used to be, the new design
// paints real backgrounds on its panels and boxes (tabs, the content panel,
// input fields, the connected badge) — each is its own layer, not just a
// border on the page background.

package main

import (
	"charm.land/bubbles/v2/textinput"
	lipgloss "charm.land/lipgloss/v2"
)

// Palette, straight from the design's CSS.
var (
	// termBg is the page background, set once on the tea.View.
	termBg = lipgloss.Color("#141414")

	panelBg = lipgloss.Color("#0e0e0e")
	fieldBg = lipgloss.Color("#161616")

	textFg   = lipgloss.Color("#e6e6e6")
	mutedFg  = lipgloss.Color("#8a8a8a")
	dimFg    = lipgloss.Color("#5c5c5c")
	statusFg = lipgloss.Color("#6b6b6b")
	errFg    = lipgloss.Color("#ff8a80")
	termFg   = lipgloss.Color("#4AFF75")

	borderFg      = lipgloss.Color("#333333")
	accentFg      = lipgloss.Color("#4AFF75")
	connectFg     = lipgloss.Color("#4AFF75")
	connectBorder = lipgloss.Color("#2A9439")

	tabActiveBg   = lipgloss.Color("#0e0e0e")
	tabActiveFg   = lipgloss.Color("#e6e6e6")
	tabInactiveBg = lipgloss.Color("#191919")
	tabInactiveFg = lipgloss.Color("#7a7a7a")
	tabDotOn      = lipgloss.Color("#4AFF75")
	tabDotOff     = lipgloss.Color("#4a4a4a")
	tabCloseFg    = lipgloss.Color("#6b6b6b")

	badgeBg = lipgloss.Color("#12291a")
)

var (
	// lineStyle carries no color at all: it only pads a line out to the full
	// page width, so joined blocks line up instead of leaving ragged rows.
	lineStyle = lipgloss.NewStyle()

	// labelStyle draws a form row's label, right-aligned in the fixed column
	// that sits before every input box (labelColWidth, in layout.go).
	labelStyle = lipgloss.NewStyle().
			Foreground(mutedFg).
			Width(labelColWidth).
			Align(lipgloss.Right)

	hintStyle = lipgloss.NewStyle().
			Foreground(dimFg)

	panelStyle = lipgloss.NewStyle().
			Background(panelBg).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderFg).
			Padding(1, 2)

	// Height(1) keeps a box exactly one text line tall. Don't reach for
	// MaxHeight here: it clips the rendered box itself, border included.
	fieldStyle = lipgloss.NewStyle().
			Background(fieldBg).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderFg).
			Foreground(textFg).
			Padding(0, 1).
			Height(1)

	focusedFieldStyle = fieldStyle.
				BorderForeground(accentFg)

	connectStyle = lipgloss.NewStyle().
			Background(fieldBg).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(connectBorder).
			Foreground(connectFg).
			Padding(0, 1).
			Height(1)

	focusedConnectStyle = connectStyle.
				BorderForeground(accentFg)

	statusStyle = lipgloss.NewStyle().
			Foreground(statusFg)

	errStatusStyle = statusStyle.
			Foreground(errFg)

	termContentStyle = lipgloss.NewStyle().
				Foreground(textFg)

	// The dialog is drawn on its own layer over the rest of the UI. Its cells
	// are opaque even without a background — spaces overwrite what's beneath.
	dialogStyle = lipgloss.NewStyle().
			Background(panelBg).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentFg).
			Padding(1, 4).
			Align(lipgloss.Center)

	dialogTitleStyle = lipgloss.NewStyle().
				Foreground(textFg).
				Bold(true)

	dialogBodyStyle = lipgloss.NewStyle().
			Foreground(statusFg)

	dialogKeyStyle = lipgloss.NewStyle().
			Foreground(dimFg)

	// Tab bar styles. Height(1) plus no vertical padding keeps every tab
	// exactly as tall as the "+" button beside it.
	tabStyle = lipgloss.NewStyle().
			Background(tabInactiveBg).
			Foreground(tabInactiveFg).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderFg).
			Padding(0, 1).
			Height(1)

	activeTabStyle = tabStyle.
			Background(tabActiveBg).
			Foreground(tabActiveFg).
			BorderForeground(accentFg)

	addTabStyle = lipgloss.NewStyle().
			Background(tabInactiveBg).
			Foreground(mutedFg).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderFg).
			Padding(0, 1).
			Height(1)

	tabDotOnStyle  = lipgloss.NewStyle().Foreground(tabDotOn)
	tabDotOffStyle = lipgloss.NewStyle().Foreground(tabDotOff)
	tabCloseStyle  = lipgloss.NewStyle().Foreground(tabCloseFg)

	connectedBadgeStyle = lipgloss.NewStyle().
				Background(badgeBg).
				Foreground(termFg).
				Padding(0, 1)
)

// styleInput keeps a textinput's box background/border to its caller, and only
// recolors its text, placeholder and cursor.
func styleInput(m *textinput.Model) {
	s := textinput.DefaultDarkStyles()
	s.Focused.Text = lipgloss.NewStyle().Foreground(textFg).Background(fieldBg)
	s.Focused.Placeholder = lipgloss.NewStyle().Foreground(dimFg).Background(fieldBg)
	s.Blurred.Text = lipgloss.NewStyle().Foreground(textFg).Background(fieldBg)
	s.Blurred.Placeholder = lipgloss.NewStyle().Foreground(dimFg).Background(fieldBg)
	s.Cursor.Color = accentFg
	m.SetStyles(s)
}
