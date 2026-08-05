// Command lazyssh is a demonstrative full-screen SSH client that mirrors the
// drafts/ssh-tui.html layout: both frames stay on screen at once — the
// connection form on top and the live terminal below. There is no page or
// window switch.
//
// Commands (as on the design):
//
//	tab / ←→   move between form fields
//	space      toggle auth method (when method is focused)
//	enter      connect
//	ctrl+b     close the session (form stays filled, ready to reconnect)
//	ctrl+c     quit, after confirming in a dialog
//
// Once connected, every other key routes straight to the remote shell —
// ctrl+c included, so it interrupts the remote command instead of this
// program. Press ctrl+b first to take the keyboard back, then ctrl+c.
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
