// lazyssh is a full-screen multi-tab SSH client.
//
// Each tab has a connection form and, once connected, a live terminal.
// Background tabs keep running. Press ctrl+h for the in-app key reference.

package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

func main() {
	p := tea.NewProgram(newApp())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "lazyssh:", err)
		os.Exit(1)
	}
}
