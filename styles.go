// Colors and styles, transcribed from design/SSH_TUI.html. Keeping them in one
// place means the design can be re-checked against this file alone.
//
// Unlike the single-shared-background version this used to be, the new design
// paints real backgrounds on its panels and boxes (tabs, the content panel,
// input fields, the connected badge) — each is its own layer, not just a
// border on the page background.

package main

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

var (
	termBg = lipgloss.Color("#141414")

	textFg   = lipgloss.Color("#e6e6e6")
	mutedFg  = lipgloss.Color("#8a8a8a")
	dimFg    = lipgloss.Color("#5c5c5c")
	statusFg = lipgloss.Color("#6b6b6b")
	errFg    = lipgloss.Color("#ff8a80")

	borderFg      = lipgloss.Color("#333333")
	accentFg      = lipgloss.Color("#4AFF75")
	connectFg     = lipgloss.Color("#4AFF75")
	connectBorder = lipgloss.Color("#2A9439")

	tabActiveFg   = lipgloss.Color("#e6e6e6")
	tabInactiveFg = lipgloss.Color("#7a7a7a")
	tabDotOn      = lipgloss.Color("#4AFF75")
	tabDotOff     = lipgloss.Color("#4a4a4a")
	tabCloseFg    = lipgloss.Color("#6b6b6b")
)

var (
	lineStyle = lipgloss.NewStyle()

	labelStyle = lipgloss.NewStyle().
			Foreground(mutedFg).
			Width(labelColWidth).
			Align(lipgloss.Right)

	hintStyle = lipgloss.NewStyle().Foreground(dimFg)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderFg).
			Padding(1, 2)

	// Height(1) keeps a box exactly one text line tall. Don't reach for
	// MaxHeight here: it clips the rendered box itself, border included.
	fieldStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderFg).
			Foreground(textFg).
			Padding(0, 1).
			Height(1)

	connectStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(connectBorder).
			Foreground(connectFg).
			Padding(0, 1).
			Height(1)

	focusedFieldStyle   = fieldStyle.BorderForeground(accentFg)
	focusedConnectStyle = connectStyle.BorderForeground(accentFg)

	selectedTextStyle = lipgloss.NewStyle().Foreground(termBg).Background(accentFg)

	termContentStyle = lipgloss.NewStyle().Foreground(dimFg)

	dialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentFg).
			Padding(1, 4).
			Align(lipgloss.Center)
	dialogTitleStyle = lipgloss.NewStyle().Foreground(textFg).Bold(true)
	dialogBodyStyle  = lipgloss.NewStyle().Foreground(statusFg)
	dialogKeyStyle   = lipgloss.NewStyle().Foreground(dimFg)
	dialogErrStyle   = lipgloss.NewStyle().Foreground(errFg).Bold(true)

	dialogBtnStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderFg).
			Foreground(textFg).
			Padding(0, 2).
			Height(1)
	dialogQuitBtnStyle = dialogBtnStyle.Foreground(errFg)

	helpCategoryStyle = lipgloss.NewStyle().Foreground(accentFg).Bold(true)
	helpKeyStyle      = lipgloss.NewStyle().Foreground(textFg).Width(22)
	helpDescStyle     = lipgloss.NewStyle().Foreground(dimFg)

	tabStyle = lipgloss.NewStyle().
			Foreground(tabInactiveFg).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderFg).
			Padding(0, 1).
			Height(1)
	addTabStyle    = tabStyle.Foreground(mutedFg)
	activeTabStyle = tabStyle.Foreground(tabActiveFg).BorderForeground(accentFg)
	tabDotOnStyle  = lipgloss.NewStyle().Foreground(tabDotOn)
	tabDotOffStyle = lipgloss.NewStyle().Foreground(tabDotOff)
	tabCloseStyle  = lipgloss.NewStyle().Foreground(tabCloseFg)
)

func styleInput(m *textinput.Model) {
	s := textinput.DefaultDarkStyles()
	s.Focused.Text = lipgloss.NewStyle().Foreground(textFg)
	s.Focused.Placeholder = lipgloss.NewStyle().Foreground(dimFg)
	s.Blurred.Text = lipgloss.NewStyle().Foreground(textFg)
	s.Blurred.Placeholder = lipgloss.NewStyle().Foreground(dimFg)
	s.Cursor.Color = accentFg
	s.Cursor.Shape = tea.CursorBar
	s.Cursor.Blink = true
	m.SetStyles(s)
	m.SetVirtualCursor(false)
	// ctrl+a is select-all in the form, not "go to start of line".
	m.KeyMap.LineStart = key.NewBinding(key.WithKeys("home"))
}
