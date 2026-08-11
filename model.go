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

const (
	fieldHost = iota
	fieldPort
	fieldUser
	fieldMethod
	fieldSecret
	fieldConnect
	fieldCount
)

type tabState struct {
	id int

	host, port, user, secret textinput.Model
	usePassword              bool // method toggle: true = password, false = private key
	focus                    int  // one of the fieldXxx constants above
	connected                bool // true => this tab shows the terminal screen

	ssh     bubblessh.Model
	hasSSH  bool // true once connect has built a Model at least once
	lastErr error
	status  string
}

type appModel struct {
	tabs   []tabState
	active int // id of the active tab — stable across closes/reorders
	nextID int

	confirmQuit bool // the quit dialog is up and owns the keyboard

	width, height int
}

func newTabState(id int) tabState {
	host := textinput.New()
	host.Prompt = ""
	host.Placeholder = "host"
	host.CharLimit = 64
	styleInput(&host)

	port := textinput.New()
	port.Prompt = ""
	port.Placeholder = "22"
	port.SetValue("22")
	port.CharLimit = 5
	styleInput(&port)

	user := textinput.New()
	user.Prompt = ""
	user.Placeholder = "user"
	user.CharLimit = 32
	styleInput(&user)

	secret := textinput.New()
	secret.Prompt = ""
	secret.Placeholder = "~/.ssh/id_ed25519"
	secret.CharLimit = 128
	styleInput(&secret)

	return tabState{
		id:     id,
		host:   host,
		port:   port,
		user:   user,
		secret: secret,
		focus:  fieldHost,
	}
}

func newApp() *appModel {
	t := newTabState(1)
	t.usePassword = true
	t.applyMethod()
	t.host.SetValue("bandit.labs.overthewire.org")
	t.port.SetValue("2220")
	t.user.SetValue("bandit0")
	t.secret.SetValue("bandit0")
	t.host.Focus()

	a := &appModel{
		tabs:   []tabState{t},
		active: t.id,
		nextID: 2,
		width:  80,
		height: 24,
	}
	a.syncInputWidths()
	return a
}

func (a *appModel) Init() tea.Cmd {
	return textinput.Blink
}

func (a *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.syncInputWidths()
		return a, a.resizeAll()

	case tea.KeyPressMsg:
		if cmd, handled := a.handleChromeKey(msg); handled {
			return a, cmd
		}
		if a.curentTab().connected {
			return a, a.updateTerminal(msg)
		}
		return a, a.updateForm(msg)

	case tea.PasteMsg, tea.MouseMsg:
		if a.curentTab().connected {
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
		if !a.curentTab().connected {
			cmd = tea.Batch(cmd, a.updateForm(msg))
		}
		return a, cmd
	}
}

func (a *appModel) handleChromeKey(key tea.KeyPressMsg) (tea.Cmd, bool) {
	switch {
	case a.confirmQuit:
		return a.updateQuitDialog(key), true
	case key.String() == "ctrl+t":
		return a.addTab(), true
	case key.String() == "ctrl+w":
		return a.closeTab(a.active), true
	case key.String() == "ctrl+right":
		return a.nextTab(), true
	case key.String() == "ctrl+left":
		return a.prevTab(), true
	// case key.String() == "ctrl+c" && !a.sessionOwnsKeys():
	case key.String() == "ctrl+q":
		a.confirmQuit = true
		a.blurAll()
		return nil, true
	}
	return nil, false
}

// func (a *appModel) sessionOwnsKeys() bool {
// 	t := a.cur()
// 	return t.connected && t.ssh.Connected()
// }

func (a *appModel) currentTabIndex() int {
	for i := range a.tabs {
		if a.tabs[i].id == a.active {
			return i
		}
	}
	return 0
}

func (a *appModel) curentTab() *tabState {
	return &a.tabs[a.currentTabIndex()]
}

func (a *appModel) updateQuitDialog(key tea.KeyPressMsg) tea.Cmd {
	switch key.String() {
	case "y", "Y", "enter", "ctrl+c":
		for i := range a.tabs {
			if a.tabs[i].hasSSH {
				_ = a.tabs[i].ssh.Close()
			}
		}
		return tea.Quit
	case "n", "N", "esc":
		a.confirmQuit = false
		return a.focusCurrent()
	}
	return nil
}

func (a *appModel) updateTerminal(msg tea.Msg) tea.Cmd {
	t := a.curentTab()
	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "ctrl+b" {
		_ = t.ssh.Close()
		t.lastErr = t.ssh.Err()
		t.connected = false
		t.status = ""
		if t.lastErr != nil {
			t.status = t.lastErr.Error()
		}
		return a.focusCurrent()
	}

	m, cmd := t.ssh.Update(msg)
	t.ssh = m.(bubblessh.Model)

	wasConnected := t.connected
	t.syncFromSSH()
	if wasConnected && !t.connected {
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
func (a *appModel) pumpSSH(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd
	for i := range a.tabs {
		t := &a.tabs[i]
		if !t.hasSSH {
			continue
		}
		m, cmd := t.ssh.Update(msg)
		t.ssh = m.(bubblessh.Model)
		t.syncFromSSH()
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

// syncFromSSH mirrors the ssh sub-model's state into the tab's own status
// fields, so the tab bar dot and the form's status line stay correct even for
// a session that's currently running in the background.
func (t *tabState) syncFromSSH() {
	switch {
	case t.ssh.Connected():
		t.connected = true
		t.status = fmt.Sprintf("connected — %s@%s", t.user.Value(), t.host.Value())
		t.lastErr = nil
	case t.connected:
		// Still dialing, or the connection just ended.
		content := t.ssh.Content()
		if strings.HasPrefix(content, "connecting") {
			t.status = content
			return
		}
		t.connected = false
		if t.ssh.Err() != nil {
			t.lastErr = t.ssh.Err()
			t.status = t.lastErr.Error()
		} else {
			t.lastErr = nil
			t.status = ""
		}
	}
}

func (a *appModel) updateForm(msg tea.Msg) tea.Cmd {
	t := a.curentTab()
	if key, ok := msg.(tea.KeyPressMsg); ok {
		// Bubble Tea v2 / ultraviolet: space is "space", not " " (String()
		// deliberately skips the literal space and falls back to Keystroke).
		switch key.String() {
		case "tab", "right":
			return a.moveFocus(1)
		case "shift+tab", "left":
			return a.moveFocus(-1)
		case "enter":
			return a.connect()
		case "space", " ":
			switch t.focus {
			case fieldMethod:
				t.usePassword = !t.usePassword
				t.applyMethod()
				return nil
			case fieldConnect:
				return a.connect()
			}
		}
	}

	var cmd tea.Cmd
	switch t.focus {
	case fieldHost:
		t.host, cmd = t.host.Update(msg)
	case fieldPort:
		t.port, cmd = t.port.Update(msg)
	case fieldUser:
		t.user, cmd = t.user.Update(msg)
	case fieldSecret:
		t.secret, cmd = t.secret.Update(msg)
	}
	return cmd
}

func (a *appModel) moveFocus(delta int) tea.Cmd {
	a.blurAll()
	t := a.curentTab()
	t.focus = (t.focus + delta + fieldCount) % fieldCount
	return a.focusCurrent()
}

func (a *appModel) blurAll() {
	t := a.curentTab()
	t.host.Blur()
	t.port.Blur()
	t.user.Blur()
	t.secret.Blur()
}

func (a *appModel) focusCurrent() tea.Cmd {
	t := a.curentTab()
	switch t.focus {
	case fieldHost:
		return t.host.Focus()
	case fieldPort:
		return t.port.Focus()
	case fieldUser:
		return t.user.Focus()
	case fieldSecret:
		return t.secret.Focus()
	}
	return nil
}

func (t *tabState) methodLabel() string {
	if t.usePassword {
		return "password"
	}
	return "publickey"
}

// secretFieldLabel is the row label to the left of the secret field — it
// swaps with the method, same as the design.
func (t *tabState) secretFieldLabel() string {
	if t.usePassword {
		return "password"
	}
	return "key path"
}

// secretSummary is the one-line auth summary shown on the terminal screen.
func (t *tabState) secretSummary() string {
	if t.usePassword {
		return "password auth"
	}
	return "key: " + t.secret.Value()
}

// target is the "user@host:port" line shown on the terminal screen.
func (t *tabState) target() string {
	host := t.host.Value()
	if host == "" {
		host = "host"
	}
	user := t.user.Value()
	if user == "" {
		user = "user"
	}
	return user + "@" + host + ":" + t.port.Value()
}

// tabLabel is what's shown in the tab strip — matches the design's
// `tab.host ? tab.user + "@" + tab.host : "nouvel onglet"`.
func (t *tabState) tabLabel() string {
	if t.host.Value() == "" {
		return "new tab"
	}
	if t.user.Value() == "" {
		return t.host.Value()
	}
	return t.user.Value() + "@" + t.host.Value()
}

// applyMethod swaps the secret field between a key path and a password: the
// echo mode changes, and a value that clearly belonged to the old method is
// dropped rather than silently sent as the wrong kind of credential.
func (t *tabState) applyMethod() {
	if t.usePassword {
		t.secret.Placeholder = "password"
		t.secret.EchoMode = textinput.EchoPassword
		if strings.HasPrefix(t.secret.Value(), "~/") || t.secret.Value() == "~/.ssh/id_ed25519" {
			t.secret.SetValue("")
		}
	} else {
		t.secret.Placeholder = "~/.ssh/id_ed25519"
		t.secret.EchoMode = textinput.EchoNormal
		if t.secret.Value() == "" {
			t.secret.SetValue("~/.ssh/id_ed25519")
		}
	}
}

// connect builds a fresh bubblessh.Model from the active tab's form values.
// Host keys are trusted on first use against the default ~/.ssh/known_hosts,
// the trade-off WithAcceptNewHostKeys documents: bubblessh can't show the
// interactive "are you sure?" prompt a real ssh client would, since Bubble Tea
// already owns the terminal.
func (a *appModel) connect() tea.Cmd {
	t := a.curentTab()
	if t.hasSSH {
		_ = t.ssh.Close()
	}

	port, err := strconv.Atoi(strings.TrimSpace(t.port.Value()))
	if err != nil || port < 1 || port > 65535 {
		t.lastErr = fmt.Errorf("invalid port %q", t.port.Value())
		t.status = t.lastErr.Error()
		return nil
	}

	opts := []bubblessh.Option{
		bubblessh.WithUser(t.user.Value()),
		bubblessh.WithPort(port),
		bubblessh.WithSize(a.termSize()),
	}
	if t.usePassword {
		opts = append(opts, bubblessh.WithPassword(t.secret.Value()))
	} else {
		opts = append(opts, bubblessh.WithPrivateKeyFile(expandHome(t.secret.Value()), ""))
	}
	if home, err := os.UserHomeDir(); err == nil {
		opts = append(opts, bubblessh.WithAcceptNewHostKeys(filepath.Join(home, ".ssh", "known_hosts")))
	}

	t.ssh = bubblessh.New(t.host.Value(), opts...)
	t.hasSSH = true
	t.connected = true
	t.lastErr = nil
	t.status = "connecting…"
	a.blurAll()
	return t.ssh.Init()
}

// resizeAll keeps every connected tab's PTY in sync with the window, not just
// the active one, so a backgrounded session is already correctly sized by the
// time you switch to it.
func (a *appModel) resizeAll() tea.Cmd {
	var cmds []tea.Cmd
	for i := range a.tabs {
		if a.tabs[i].hasSSH {
			var cmd tea.Cmd
			a.tabs[i].ssh, cmd = a.tabs[i].ssh.SetSize(a.termSize())
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

// addTab opens a fresh, blank tab and makes it active.
func (a *appModel) addTab() tea.Cmd {
	t := newTabState(a.nextID)
	a.nextID++
	a.tabs = append(a.tabs, t)
	a.active = t.id
	a.syncInputWidths()
	return a.focusCurrent()
}

// closeTab closes id's SSH session (if any) and removes it. Closing the last
// remaining tab resets to one fresh blank tab rather than leaving the app
// with none — matches the design's `list.length ? list : [newTab()]`.
func (a *appModel) closeTab(id int) tea.Cmd {
	idx := -1
	for i := range a.tabs {
		if a.tabs[i].id == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil
	}
	if a.tabs[idx].hasSSH {
		_ = a.tabs[idx].ssh.Close()
	}
	wasActive := a.tabs[idx].id == a.active

	a.tabs = append(a.tabs[:idx], a.tabs[idx+1:]...)

	if len(a.tabs) == 0 {
		t := newTabState(a.nextID)
		a.nextID++
		a.tabs = []tabState{t}
		a.active = t.id
		return a.focusCurrent()
	}

	if !wasActive {
		return nil
	}
	next := min(idx, len(a.tabs)-1)
	a.active = a.tabs[next].id
	if a.curentTab().connected {
		return nil
	}
	return a.focusCurrent()
}

func (a *appModel) nextTab() tea.Cmd { return a.switchTab(1) }
func (a *appModel) prevTab() tea.Cmd { return a.switchTab(-1) }

func (a *appModel) switchTab(delta int) tea.Cmd {
	if len(a.tabs) < 2 {
		return nil
	}
	idx := (a.currentTabIndex() + delta + len(a.tabs)) % len(a.tabs)
	a.active = a.tabs[idx].id
	if a.curentTab().connected {
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
