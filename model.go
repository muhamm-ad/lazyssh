// State and key handling: what the app knows, and how each key changes it.
// Nothing here decides how anything looks — see view.go for that.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	bubblessh "github.com/muhamm-ad/bubble-ssh"
)

type AppModel struct {
	tabs   []Tab
	active int
	nextID int

	confirmQuit bool // the quit dialog is up and owns the keyboard
	showHelp    bool // the help modal is up and owns the keyboard

	width, height int
}

func newApp() *AppModel {
	t := NewTab(1)
	t.Host.SetValue("bandit.labs.overthewire.org")
	t.Port.SetValue("2220")
	t.User.SetValue("bandit0")
	t.SetPassword("bandit0")
	t.Host.Focus()

	a := &AppModel{
		tabs:   []Tab{t},
		active: t.ID,
		nextID: 2,
		width:  80,
		height: 24,
	}
	a.syncInputWidths()
	return a
}

func (a *AppModel) Init() tea.Cmd {
	return textinput.Blink
}

func (a *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.syncInputWidths()
		return a, a.resizeAll()

	case tea.KeyPressMsg:
		if cmd, handled := a.handleChromeKey(msg); handled {
			return a, cmd
		}
		if a.curentTab().Connected {
			return a, a.updateTerminal(msg)
		}
		return a, a.updateForm(msg)

	case tea.PasteMsg, tea.MouseMsg:
		if a.curentTab().Connected {
			return a, a.updateTerminal(msg)
		}
		return a, a.updateForm(msg)

	default:
		// Async messages that belong to a specific tab's SSH session (or to
		// nobody) — connect/output/close/error from bubblessh, and
		// textinput's blink tick. pumpSSH fans it to every tab's session;
		// each bubblessh.Model silently ignores a message carrying another
		// instance's id, which is what keeps a backgrounded session's own
		// read loop armed instead of stalling while another tab is on
		// screen. The active tab's form (if that's what it's showing) still
		// needs the same message for its focused input's blinking cursor.
		cmd := a.pumpSSH(msg)
		if !a.curentTab().Connected {
			cmd = tea.Batch(cmd, a.updateForm(msg))
		}
		return a, cmd
	}
}


// resizeAll keeps every connected tab's PTY in sync with the window, not just
// the active one, so a backgrounded session is already correctly sized by the
// time you switch to it.
func (a *AppModel) resizeAll() tea.Cmd {
	var cmds []tea.Cmd
	for i := range a.tabs {
		t := &a.tabs[i]
		if t.HasSSH {
			var cmd tea.Cmd
			t.SSH, cmd = t.SSH.SetSize(a.termSize())
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func (a *AppModel) handleChromeKey(key tea.KeyPressMsg) (tea.Cmd, bool) {
	switch {
	case a.confirmQuit:
		return a.updateQuitDialog(key), true
	case a.showHelp:
		return a.updateHelpModal(key), true
	case key.String() == "ctrl+h":
		return a.openHelp(), true
	case key.String() == "ctrl+t":
		return a.addTab(), true
	case key.String() == "ctrl+w":
		return a.closeTab(a.active), true
	case key.String() == "ctrl+right":
		return a.nextTab(), true
	case key.String() == "ctrl+left":
		return a.prevTab(), true
	case key.String() == "ctrl+q":
		a.confirmQuit = true
		a.blurAll()
		return nil, true
	}
	return nil, false
}

func (a *AppModel) openHelp() tea.Cmd {
	a.showHelp = true
	a.blurAll()
	return nil
}

func (a *AppModel) updateHelpModal(key tea.KeyPressMsg) tea.Cmd {
	switch key.String() {
	case "esc", "ctrl+h":
		a.showHelp = false
		if a.curentTab().Connected {
			return nil
		}
		return a.focusCurrent()
	}
	return nil
}

func (a *AppModel) currentTabIndex() int {
	for i := range a.tabs {
		if a.tabs[i].ID == a.active {
			return i
		}
	}
	return 0
}

func (a *AppModel) curentTab() *Tab {
	return &a.tabs[a.currentTabIndex()]
}

func (a *AppModel) updateQuitDialog(key tea.KeyPressMsg) tea.Cmd {
	switch key.String() {
	case "y", "Y", "enter":
		for i := range a.tabs {
			if a.tabs[i].HasSSH {
				_ = a.tabs[i].SSH.Close()
			}
		}
		return tea.Quit
	case "n", "N", "esc":
		a.confirmQuit = false
		return a.focusCurrent()
	}
	return nil
}

func (a *AppModel) updateTerminal(msg tea.Msg) tea.Cmd {
	t := a.curentTab()
	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "ctrl+b" {
		_ = t.SSH.Close()
		t.LastErr = t.SSH.Err()
		t.Connected = false
		t.Status = ""
		if t.LastErr != nil {
			t.Status = t.LastErr.Error()
		}
		return a.focusCurrent()
	}

	m, cmd := t.SSH.Update(msg)
	t.SSH = m.(bubblessh.Model)

	wasConnected := t.Connected
	t.syncFromSSH()
	if wasConnected && !t.Connected {
		// The session just dropped or errored out — give the form back.
		return tea.Batch(cmd, a.focusCurrent())
	}
	return cmd
}

// pumpSSH feeds msg to every tab that has an SSH model, regardless of which
// tab is active, and syncs each tab's status from the result. This is what
// keeps a backgrounded session's own read loop armed — bubblessh's
// waitForActivity command has to be re-issued from inside Update every time
// it fires, and that has to happen whether or not that tab is on screen.
func (a *AppModel) pumpSSH(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	for i := range a.tabs {
		t := &a.tabs[i]
		if !t.HasSSH {
			continue
		}
		m, cmd := t.SSH.Update(msg)
		t.SSH = m.(bubblessh.Model)
		t.syncFromSSH()
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func (a *AppModel) updateForm(msg tea.Msg) tea.Cmd {
	t := a.curentTab()
	if key, ok := msg.(tea.KeyPressMsg); ok {
		// Bubble Tea v2 / ultraviolet: space is "space", not " " (String()
		// deliberately skips the literal space and falls back to Keystroke).
		switch key.String() {
		case "tab":
			return a.moveFocus(1)
		case "shift+tab":
			return a.moveFocus(-1)
		case "enter":
			return a.connect()
		case "space", " ":
			switch t.Focus {
			case fieldMethod:
				t.UsePassword = !t.UsePassword
				t.ApplyMethod()
				return nil
			case fieldConnect:
				return a.connect()
			}
		}
	}

	var cmd tea.Cmd
	switch t.Focus {
	case fieldHost:
		t.Host, cmd = t.Host.Update(msg)
	case fieldPort:
		t.Port, cmd = t.Port.Update(msg)
	case fieldUser:
		t.User, cmd = t.User.Update(msg)
	case fieldSecret:
		t.Secret, cmd = t.Secret.Update(msg)
	}
	return cmd
}

func (a *AppModel) moveFocus(delta int) tea.Cmd {
	a.blurAll()
	t := a.curentTab()
	t.Focus = (t.Focus + delta + fieldCount) % fieldCount
	return a.focusCurrent()
}

func (a *AppModel) blurAll() {
	t := a.curentTab()
	t.Host.Blur()
	t.Port.Blur()
	t.User.Blur()
	t.Secret.Blur()
}

func (a *AppModel) focusCurrent() tea.Cmd {
	t := a.curentTab()
	switch t.Focus {
	case fieldHost:
		return t.Host.Focus()
	case fieldPort:
		return t.Port.Focus()
	case fieldUser:
		return t.User.Focus()
	case fieldSecret:
		return t.Secret.Focus()
	}
	return nil
}

// connect builds a fresh bubblessh.Model from the active tab's form values.
// Host keys are trusted on first use against the default ~/.ssh/known_hosts,
// the trade-off WithAcceptNewHostKeys documents: bubblessh can't show the
// interactive "are you sure?" prompt a real ssh client would, since Bubble Tea
// already owns the terminal.
func (a *AppModel) connect() tea.Cmd {
	t := a.curentTab()
	if t.HasSSH {
		_ = t.SSH.Close()
	}

	port, err := strconv.Atoi(strings.TrimSpace(t.Port.Value()))
	if err != nil || port < 1 || port > 65535 {
		t.LastErr = fmt.Errorf("invalid port %q", t.Port.Value())
		t.Status = t.LastErr.Error()
		return nil
	}

	opts := []bubblessh.Option{
		bubblessh.WithUser(t.User.Value()),
		bubblessh.WithPort(port),
		bubblessh.WithSize(a.termSize()),
	}
	if t.UsePassword {
		opts = append(opts, bubblessh.WithPassword(t.Secret.Value()))
	} else {
		opts = append(opts, bubblessh.WithPrivateKeyFile(expandHome(t.Secret.Value()), ""))
	}
	if home, err := os.UserHomeDir(); err == nil {
		opts = append(opts, bubblessh.WithAcceptNewHostKeys(filepath.Join(home, ".ssh", "known_hosts")))
	}

	t.SSH = bubblessh.New(t.Host.Value(), opts...)
	t.HasSSH = true
	t.Connected = true
	t.LastErr = nil
	t.Status = "connecting…"
	a.blurAll()
	return t.SSH.Init()
}

// addTab opens a fresh, blank tab and makes it active.
func (a *AppModel) addTab() tea.Cmd {
	t := NewTab(a.nextID)
	a.nextID++
	a.tabs = append(a.tabs, t)
	a.active = t.ID
	a.syncInputWidths()
	return a.focusCurrent()
}

// closeTab closes id's SSH session (if any) and removes it. Closing the last
// remaining tab resets to one fresh blank tab rather than leaving the app
// with none — matches the design's `list.length ? list : [newTab()]`.
func (a *AppModel) closeTab(id int) tea.Cmd {
	idx := -1
	for i := range a.tabs {
		if a.tabs[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil
	}
	if a.tabs[idx].HasSSH {
		_ = a.tabs[idx].SSH.Close()
	}
	wasActive := a.tabs[idx].ID == a.active

	a.tabs = append(a.tabs[:idx], a.tabs[idx+1:]...)

	if len(a.tabs) == 0 {
		t := NewTab(a.nextID)
		a.nextID++
		a.tabs = []Tab{t}
		a.active = t.ID
		return a.focusCurrent()
	}

	if !wasActive {
		return nil
	}
	next := min(idx, len(a.tabs)-1)
	a.active = a.tabs[next].ID
	if a.curentTab().Connected {
		return nil
	}
	return a.focusCurrent()
}

func (a *AppModel) nextTab() tea.Cmd { return a.switchTab(1) }
func (a *AppModel) prevTab() tea.Cmd { return a.switchTab(-1) }

func (a *AppModel) switchTab(delta int) tea.Cmd {
	if len(a.tabs) < 2 {
		return nil
	}
	idx := (a.currentTabIndex() + delta + len(a.tabs)) % len(a.tabs)
	a.active = a.tabs[idx].ID
	if a.curentTab().Connected {
		return nil
	}
	return a.focusCurrent()
}

// expandHome replaces a leading "~" with the user's home directory — Go doesn't
// do this automatically, unlike a shell expanding an unquoted ~ typed on a
// command line. A value typed into a text field never passes through a shell.
func expandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	return filepath.Join(home, path[2:])
}
