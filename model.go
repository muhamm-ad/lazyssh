// State and key handling: what the app knows, and how each key changes it.
// Nothing here decides how anything looks — see view.go for that.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	bubblessh "github.com/muhamm-ad/bubble-ssh"
)

// Form field indices, in tab order — matches the design's FIELDS array.
const (
	fieldHost = iota
	fieldUser
	fieldMethod
	fieldSecret
	fieldConnect
	fieldCount
)

// const idleStatus = "not connected — fill the form and press connect"

type appModel struct {
	host, user, secret textinput.Model
	usePassword        bool // method toggle: true = password, false = private key
	focus              int  // one of the fieldXxx constants above
	connected          bool // keys go to the SSH pane when true
	confirmQuit        bool // the quit dialog is up and owns the keyboard

	ssh     bubblessh.Model
	hasSSH  bool // true once Connect has built a Model at least once
	lastErr error
	status  string

	width, height int
}

func newApp() appModel {
	host := textinput.New()
	host.Prompt = ""
	host.Placeholder = "host:port"
	host.SetValue("bandit.labs.overthewire.org:2220")
	host.Focus()
	host.CharLimit = 64
	styleInput(&host)

	user := textinput.New()
	user.Prompt = ""
	user.Placeholder = "user"
	// if u := os.Getenv("USER"); u != "" {
	// 	user.SetValue(u)
	// } else {
	user.SetValue("bandit0")
	// }
	user.CharLimit = 32
	styleInput(&user)

	secret := textinput.New()
	secret.Prompt = ""
	secret.SetValue("bandit0")
	secret.EchoMode = textinput.EchoPassword
	secret.CharLimit = 128
	styleInput(&secret)

	return appModel{
		host:        host,
		user:        user,
		secret:      secret,
		usePassword: true, // design default: method: key
		focus:       fieldHost,
		// status:      idleStatus,
		status: "",
		width:  80,
		height: 24,
	}
}

func (a appModel) Init() tea.Cmd {
	return textinput.Blink
}

func (a appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		a.width, a.height = size.Width, size.Height
		a.syncInputWidths()
		if a.hasSSH {
			var cmd tea.Cmd
			a.ssh, cmd = a.ssh.SetSize(a.termSize())
			return a, cmd
		}
		return a, nil
	}

	// Only keys are intercepted below. Everything else — remote output above
	// all — has to keep flowing even with the dialog up: bubblessh reads the
	// session by returning a fresh "wait for output" command from each
	// outputMsg, so swallowing one would stall the session for good.
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case a.confirmQuit:
			return a.updateQuitDialog(key)

		case key.String() == "ctrl+c" && !a.sessionOwnsKeys():
			// Nothing is listening on the other end, so ctrl+c means "quit
			// this program" — ask first, it's a keystroke away from the
			// interrupt someone may have meant for a remote command.
			a.confirmQuit = true
			a.blurAll()
			return a, nil
		}
	}

	if a.connected {
		return a.updateTerminal(msg)
	}
	return a.updateForm(msg)
}

// sessionOwnsKeys reports whether keystrokes currently reach the remote shell,
// which is what makes ctrl+c the remote's interrupt rather than ours. While
// still dialing, the terminal frame has the keyboard but nothing is listening
// yet — so ctrl+c stays local and can get you out of a hanging connection.
func (a appModel) sessionOwnsKeys() bool {
	return a.connected && a.ssh.Connected()
}

// updateQuitDialog handles the modal's keys. It's the only thing reading the
// keyboard while it's up, so an unrecognized key does nothing at all rather
// than leaking through to the form or the session.
func (a appModel) updateQuitDialog(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "y", "Y", "enter", "ctrl+c":
		if a.hasSSH {
			_ = a.ssh.Close()
		}
		return a, tea.Quit
	case "n", "N", "esc":
		a.confirmQuit = false
		return a, a.focusCurrent()
	}
	return a, nil
}

func (a appModel) updateTerminal(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "ctrl+b" {
		_ = a.ssh.Close()
		a.lastErr = a.ssh.Err()
		a.connected = false
		// a.status = idleStatus
		a.status = ""
		if a.lastErr != nil {
			a.status = a.lastErr.Error()
		}
		return a, a.focusCurrent()
	}

	m, cmd := a.ssh.Update(msg)
	a.ssh = m.(bubblessh.Model)

	if a.ssh.Connected() {
		a.status = fmt.Sprintf("connected — %s@%s", a.user.Value(), a.host.Value())
		a.lastErr = nil
		return a, cmd
	}

	// Still dialing — stay in terminal key mode until we connect or fail.
	content := a.ssh.Content()
	if strings.HasPrefix(content, "connecting") {
		a.status = content
		return a, cmd
	}

	// Error or remote close — give the form back, keep both frames visible.
	a.connected = false
	if a.ssh.Err() != nil {
		a.lastErr = a.ssh.Err()
		a.status = a.lastErr.Error()
	} else {
		a.lastErr = nil
		// a.status = idleStatus
		a.status = ""
	}
	return a, a.focusCurrent()
}

func (a appModel) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			switch a.focus {
			case fieldMethod:
				a.usePassword = !a.usePassword
				a.applyMethod()
				return a, nil
			case fieldConnect:
				return a.connect()
			}
		}
	}

	var cmd tea.Cmd
	switch a.focus {
	case fieldHost:
		a.host, cmd = a.host.Update(msg)
	case fieldUser:
		a.user, cmd = a.user.Update(msg)
	case fieldSecret:
		a.secret, cmd = a.secret.Update(msg)
	}
	return a, cmd
}

func (a appModel) moveFocus(delta int) (tea.Model, tea.Cmd) {
	a.blurAll()
	a.focus = (a.focus + delta + fieldCount) % fieldCount
	cmd := a.focusCurrent()
	return a, cmd
}

func (a *appModel) blurAll() {
	a.host.Blur()
	a.user.Blur()
	a.secret.Blur()
}

func (a *appModel) focusCurrent() tea.Cmd {
	switch a.focus {
	case fieldHost:
		return a.host.Focus()
	case fieldUser:
		return a.user.Focus()
	case fieldSecret:
		return a.secret.Focus()
	}
	return nil
}

func (a appModel) methodLabel() string {
	if a.usePassword {
		return "method: password"
	}
	return "method: key"
}

// applyMethod swaps the secret field between a key path and a password: the
// echo mode changes, and a value that clearly belonged to the old method is
// dropped rather than silently sent as the wrong kind of credential.
func (a *appModel) applyMethod() {
	if a.usePassword {
		a.secret.Placeholder = "password"
		a.secret.EchoMode = textinput.EchoPassword
		if strings.HasPrefix(a.secret.Value(), "~/") || a.secret.Value() == "~/.ssh/id_ed25519" {
			a.secret.SetValue("")
		}
	} else {
		a.secret.Placeholder = "~/.ssh/id_ed25519"
		a.secret.EchoMode = textinput.EchoNormal
		if a.secret.Value() == "" {
			a.secret.SetValue("~/.ssh/id_ed25519")
		}
	}
}

// connect builds a fresh bubblessh.Model from the current form values. Host
// keys are trusted on first use against the default ~/.ssh/known_hosts, the
// trade-off WithAcceptNewHostKeys documents: bubblessh can't show the
// interactive "are you sure?" prompt a real ssh client would, since Bubble Tea
// already owns the terminal.
func (a appModel) connect() (tea.Model, tea.Cmd) {
	if a.hasSSH {
		_ = a.ssh.Close()
	}

	opts := []bubblessh.Option{
		bubblessh.WithUser(a.user.Value()),
		bubblessh.WithSize(a.termSize()),
	}
	if a.usePassword {
		opts = append(opts, bubblessh.WithPassword(a.secret.Value()))
	} else {
		opts = append(opts, bubblessh.WithPrivateKeyFile(expandHome(a.secret.Value()), ""))
	}
	if home, err := os.UserHomeDir(); err == nil {
		opts = append(opts, bubblessh.WithAcceptNewHostKeys(filepath.Join(home, ".ssh", "known_hosts")))
	}

	a.ssh = bubblessh.New(a.host.Value(), opts...)
	a.hasSSH = true
	a.connected = true
	a.lastErr = nil
	a.status = "connecting…"
	a.blurAll()
	return a, a.ssh.Init()
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
