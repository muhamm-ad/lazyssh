// lazyssh is a lightweight TUI that simplifies SSH connections.
//
// Each tab has a connection form and, once connected, a live terminal background tabs keep running. Press ctrl+h for the in-app key reference.

package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	tea "charm.land/bubbletea/v2"
)

func main() {
	showVersion := false
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.BoolVar(&showVersion, "v", false, "print version and exit")
	flag.Parse()

	if showVersion {
		fmt.Println("lazyssh", resolveVersion())
		return
	}

	p := tea.NewProgram(newApp())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "lazyssh:", err)
		os.Exit(1)
	}
}

func resolveVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	return "dev" + devSuffix(info.Settings)
}

// devSuffix formats the VCS revision Go stamps into local builds, e.g.
// " (a1b2c3d)" or " (a1b2c3d, dirty)". Returns "" if no revision is known.
func devSuffix(settings []debug.BuildSetting) string {
	var revision string
	dirty := false
	for _, s := range settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if revision == "" {
		return ""
	}
	if len(revision) > 7 {
		revision = revision[:7]
	}
	if dirty {
		return fmt.Sprintf(" (%s, dirty)", revision)
	}
	return fmt.Sprintf(" (%s)", revision)
}
