// Command lazyssh is a demonstrative full-screen SSH client that mirrors the
// design/SSH_TUI.html layout: a row of tabs across the top, each an
// independent SSH session, and a single panel below showing whichever
// screen the active tab is on — its connection form, or its live terminal.
// A tab keeps running in the background while another tab is on screen.
//
// Commands (as on the design, plus the tab shortcuts a mouse-driven mockup
// doesn't need):
//
//	tab / ←→          move between form fields (address, port, user, method, key/password)
//	space             toggle auth method (when method is focused)
//	enter             connect
//	ctrl+b            close the active tab's session (form stays filled, ready to reconnect)
//	ctrl+t            open a new tab
//	ctrl+w            close the active tab
//	ctrl+→ / ctrl+←   switch tabs
//	ctrl+q            quit, after confirming in a dialog
//
// The tab shortcuts and ctrl+b work even while a session owns the keyboard,
// same as ctrl+q's confirm gate — everything else routes straight to the
// remote shell once connected, ctrl+q included, so it interrupts the remote
// command instead of this program.
//
//	go run ./lazyssh
//
// The example is split by concern, in the order it's easiest to read:
//
//	model.go   state and key handling (the Elm-architecture half)
//	layout.go  how many columns and rows each piece of chrome gets
//	styles.go  colors and lipgloss styles, transcribed from the design
//	view.go    turning all of the above into a frame of text
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

func main() {
	p := tea.NewProgram(newApp())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "bubblessh:", err)
		os.Exit(1)
	}
}
