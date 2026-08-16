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
	bubbleup "go.dalton.dog/bubbleup/v2"

	bubblessh "github.com/muhamm-ad/bubble-ssh"
)

type AppModel struct {
	tabs   []Tab
	active int
	nextID int

	confirmQuit bool // the quit dialog is up and owns the keyboard
	showHelp    bool // the help modal is up and owns the keyboard
	focusAdd    bool // the tab-bar "+" is selected (ctrl+←/→), enter adds a tab

	alert bubbleup.AlertModel

	tabBarScroll  int // horizontal offset of the tab strip, in cells
	fieldDragging bool
	termDragging  bool
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
		alert:  newAlertModel(80),
		width:  80,
		height: 24,
	}
	a.syncInputWidths()
	return a
}

func (a *AppModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, a.alert.Init())
}

func (a *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.syncInputWidths()
		a.clampTabBarScroll()
		a.syncAlertWidth()
		return a.pipeAlert(msg, a.resizeAll())

	case tea.KeyPressMsg:
		if cmd, handled := a.handleChromeKey(msg); handled {
			return a.pipeAlert(msg, cmd)
		}
		if a.curentTab().inSession() {
			return a.pipeAlert(msg, a.updateTerminal(msg))
		}
		return a.pipeAlert(msg, a.updateForm(msg))

	case tea.MouseMsg:
		return a.pipeAlert(msg, a.handleMouse(msg))

	case tea.PasteMsg:
		if a.focusAdd {
			return a.pipeAlert(msg, nil)
		}
		if a.curentTab().inSession() {
			return a.pipeAlert(msg, a.updateTerminal(msg))
		}
		return a.pipeAlert(msg, a.updateForm(msg))

	default:
		wasLive := a.curentTab().inSession()
		cmd := a.pumpSSH(msg)
		if !a.curentTab().inSession() {
			if wasLive {
				cmd = tea.Batch(cmd, a.focusCurrent())
			}
			cmd = tea.Batch(cmd, a.updateForm(msg))
		}
		cmd = tea.Batch(cmd, a.alertFromTabs())
		return a.pipeAlert(msg, cmd)
	}
}

// resizeAll keeps every connected tab's PTY in sync with the window, not just
// the active one, so a backgrounded session is already correctly sized by the
// time you switch to it.
func (a *AppModel) resizeAll() tea.Cmd {
	var cmds []tea.Cmd
	for i := range a.tabs {
		if cmd := a.tabs[i].setSSHSize(a.termSize()); cmd != nil {
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
	case key.String() == "ctrl+c":
		if a.copyTermSelection() {
			return a.notifyInfo("copied"), true
		}
	case key.String() == "ctrl+w":
		if a.focusAdd {
			return nil, true
		}
		return a.closeTab(a.active), true
	case key.String() == "ctrl+right":
		return a.nextTab(), true
	case key.String() == "ctrl+left":
		return a.prevTab(), true
	case key.String() == "ctrl+q":
		a.confirmQuit = true
		a.blurAll()
		return nil, true
	case a.focusAdd && key.String() == "enter":
		return a.addTab(), true
	case a.focusAdd:
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
		if a.curentTab().inSession() {
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
			a.tabs[i].closeSSH()
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
	wasLive := t.inSession()
	cmd := t.updateSSH(msg)
	cmd = tea.Batch(cmd, a.alertFromTabs())
	if wasLive && !t.inSession() {
		// The remote shell exited (or the session dropped) — give the form back.
		return tea.Batch(cmd, a.focusCurrent())
	}
	return cmd
}

// pumpSSH feeds msg to every tab that has an SSH model, regardless of which
// tab is active. This is what keeps a backgrounded session's own read loop
// armed — bubblessh's waitForActivity command has to be re-issued from
// inside Update every time it fires, and that has to happen whether or not
// that tab is on screen.
func (a *AppModel) pumpSSH(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	for i := range a.tabs {
		if cmd := a.tabs[i].updateSSH(msg); cmd != nil {
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
			t.clearSelection()
			return a.moveFocus(1)
		case "shift+tab":
			t.clearSelection()
			return a.moveFocus(-1)
		case "enter":
			t.clearSelection()
			return a.connect()
		case "ctrl+a":
			t.selectAllFocused()
			return nil
		case "ctrl+c":
			t.copyFocused()
			if !t.secretLocked() && t.focusedInput() != nil {
				return a.notifyInfo("copied")
			}
			return nil
		case "ctrl+x":
			t.cutFocused()
			if !t.secretLocked() {
				return a.notifyInfo("cut")
			}
			return nil
		case "ctrl+z":
			t.undoLast()
			a.blurAll()
			return a.focusCurrent()
		}

		if t.focusedInput() != nil {
			t.pushUndo()
		}

		switch key.String() {
		case "ctrl+v":
			t.replaceSelectedIfNeeded()
		case "left", "right", "home", "end":
			t.clearSelection()
		case "backspace", "delete", "ctrl+h", "ctrl+d":
			if t.TextSelected {
				t.replaceSelectedIfNeeded()
				t.dropUndoIfUnchanged()
				return nil
			}
		case "space", " ":
			switch t.Focus {
			case fieldMethod:
				t.dropUndoIfUnchanged()
				t.UsePassword = !t.UsePassword
				t.ApplyMethod()
				return nil
			case fieldConnect:
				t.dropUndoIfUnchanged()
				return a.connect()
			}
			t.replaceSelectedIfNeeded()
		default:
			if t.TextSelected && t.focusedInput() != nil && key.Text != "" {
				t.replaceSelectedIfNeeded()
			}
		}
	} else if t.focusedInput() != nil {
		t.pushUndo()
	}
	if _, ok := msg.(tea.PasteMsg); ok {
		t.replaceSelectedIfNeeded()
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
	t.dropUndoIfUnchanged()
	return cmd
}

func (a *AppModel) moveFocus(delta int) tea.Cmd {
	a.blurAll()
	t := a.curentTab()
	t.clearSelection()
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

	port, err := strconv.Atoi(strings.TrimSpace(t.Port.Value()))
	if err != nil || port < 1 || port > 65535 {
		msg := friendlyError(fmt.Sprintf("invalid port %q", t.Port.Value()))
		t.Status = msg
		t.lastNotified = msg
		return a.notifyError(msg)
	}

	t.closeSSH()

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

	m := bubblessh.New(t.Host.Value(), opts...)
	t.SSH = &m
	t.Status = ""
	t.lastNotified = ""
	t.wasLive = false
	a.blurAll()
	return tea.Batch(t.SSH.Init(), a.alertFromTab(t))
}

// addTab opens a fresh, blank tab and makes it active.
func (a *AppModel) addTab() tea.Cmd {
	t := NewTab(a.nextID)
	a.nextID++
	a.tabs = append(a.tabs, t)
	a.active = t.ID
	a.focusAdd = false
	a.syncInputWidths()
	a.ensureActiveTabVisible()
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
	a.tabs[idx].closeSSH()
	wasActive := a.tabs[idx].ID == a.active

	a.tabs = append(a.tabs[:idx], a.tabs[idx+1:]...)

	if len(a.tabs) == 0 {
		t := NewTab(a.nextID)
		a.nextID++
		a.tabs = []Tab{t}
		a.active = t.ID
		a.focusAdd = false
		a.ensureActiveTabVisible()
		return a.focusCurrent()
	}

	if !wasActive {
		a.clampTabBarScroll()
		return nil
	}
	next := min(idx, len(a.tabs)-1)
	a.active = a.tabs[next].ID
	a.ensureActiveTabVisible()
	if a.curentTab().inSession() {
		return nil
	}
	return a.focusCurrent()
}

func (a *AppModel) nextTab() tea.Cmd { return a.switchTab(1) }
func (a *AppModel) prevTab() tea.Cmd { return a.switchTab(-1) }

func (a *AppModel) switchTab(delta int) tea.Cmd {
	n := len(a.tabs)
	// One slot per tab, plus the trailing "+" in the tab bar.
	pos := n
	if !a.focusAdd {
		pos = a.currentTabIndex()
	}
	pos = (pos + delta + n + 1) % (n + 1)
	if pos == n {
		a.focusAdd = true
		a.blurAll()
		return nil
	}
	a.focusAdd = false
	a.active = a.tabs[pos].ID
	a.ensureActiveTabVisible()
	if a.curentTab().inSession() {
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
