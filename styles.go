// Colors and styles, transcribed from drafts/ssh-tui.html. Keeping them in one
// place means the design can be re-checked against this file alone.
//
// Only one background is painted: the terminal's own, set once on the tea.View.
// Every element below is transparent and carries color in its text and borders
// alone, so the whole UI sits flat on that single background.

package main

import (
	"charm.land/bubbles/v2/textinput"
	lipgloss "charm.land/lipgloss/v2"
)

// Palette, straight from the design's CSS.
var (
	// termBg is the one background in the program — see View in view.go.
	termBg = lipgloss.Color("#141414")

	textFg   = lipgloss.Color("#e6e6e6")
	mutedFg  = lipgloss.Color("#8a8a8a")
	dimFg    = lipgloss.Color("#5c5c5c")
	statusFg = lipgloss.Color("#6b6b6b")
	errFg    = lipgloss.Color("#ff8a80")
	termFg   = lipgloss.Color("#3fb950")

	borderFg      = lipgloss.Color("#333333")
	accentFg      = lipgloss.Color("#4a9eff")
	connectFg     = lipgloss.Color("#4AFF75")
	connectBorder = lipgloss.Color("#2A9439")
)

var (
	// lineStyle carries no color at all: it only pads a line out to the full
	// page width, so joined blocks line up instead of leaving ragged rows.
	lineStyle = lipgloss.NewStyle()

	labelStyle = lipgloss.NewStyle().
			Foreground(mutedFg).
			Bold(true)

	hintStyle = lipgloss.NewStyle().
			Foreground(dimFg)

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

	focusedFieldStyle = fieldStyle.
				BorderForeground(accentFg)

	connectStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(connectBorder).
			Foreground(connectFg).
			Padding(0, 1).
			Height(1)

	focusedConnectStyle = connectStyle.
				BorderForeground(accentFg)

	statusStyle = lipgloss.NewStyle().
			Foreground(statusFg).
			Align(lipgloss.Center)

	errStatusStyle = statusStyle.
			Foreground(errFg)

	okStatusStyle = statusStyle.
			Foreground(termFg)

	termContentStyle = lipgloss.NewStyle().
				Foreground(textFg)

	termIdleStyle = lipgloss.NewStyle().
			Foreground(termFg)

	// The dialog is drawn on its own layer over the rest of the UI. Its cells
	// are opaque even without a background — spaces overwrite what's beneath.
	dialogStyle = lipgloss.NewStyle().
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
)

// styleInput keeps a textinput transparent like the box around it, and only
// recolors its text, placeholder and cursor.
func styleInput(m *textinput.Model) {
	s := textinput.DefaultDarkStyles()
	s.Focused.Text = lipgloss.NewStyle().Foreground(textFg)
	s.Focused.Placeholder = lipgloss.NewStyle().Foreground(dimFg)
	s.Blurred.Text = lipgloss.NewStyle().Foreground(textFg)
	s.Blurred.Placeholder = lipgloss.NewStyle().Foreground(dimFg)
	s.Cursor.Color = accentFg
	m.SetStyles(s)
}
