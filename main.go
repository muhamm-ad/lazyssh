// Command lazyssh is a small full-screen SSH client: a connection form on
// top, the live terminal below. Tab/←→ move between fields, space toggles
// the auth method, enter connects. Once connected, every key routes
// straight to the remote shell — ctrl+b closes the session and gives the
// form back, still filled in, ready to connect somewhere else.
//
// Built on github.com/muhamm-ad/bubble-ssh — see TODO.md for what's
// planned beyond this first, deliberately simple version.
//
//	go run .
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"

	bubblessh "github.com/muhamm-ad/bubble-ssh"
)

type screen int

const (
	screenForm screen = iota
	screenTerminal
)

// Form field indices, in tab order.
const (
	fieldAddr = iota
	fieldUser
	fieldMethod
	fieldSecret
	fieldConnect
	fieldCount
)

type appModel struct {
	scr screen

	addr, user, secret textinput.Model
	usePassword        bool // method toggle: true = password, false = private key file
	focus              int  // one of the fieldXxx constants above

	ssh     bubblessh.Model
	lastErr error

	width, height int
}

func newApp() appModel {
	addr := textinput.New()
	addr.Prompt = ""
	addr.Placeholder = "host:port"
	addr.SetValue("example.com:22")
	addr.Focus()

	user := textinput.New()
	user.Prompt = ""
	user.Placeholder = "user"
	user.SetValue(os.Getenv("USER"))

	secret := textinput.New()
	secret.Prompt = ""
	secret.Placeholder = "password"
	secret.EchoMode = textinput.EchoPassword

	return appModel{
		addr:        addr,
		user:        user,
		secret:      secret,
		usePassword: true,
		focus:       fieldAddr,
		width:       80,
		height:      24,
	}
}

func (a appModel) Init() tea.Cmd {
	return textinput.Blink
}

func (a appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "ctrl+c" {
		if a.scr == screenTerminal {
			_ = a.ssh.Close()
		}
		return a, tea.Quit
	}

	if size, ok := msg.(tea.WindowSizeMsg); ok {
		a.width, a.height = size.Width, size.Height
		if a.scr == screenTerminal {
			var cmd tea.Cmd
			a.ssh, cmd = a.ssh.SetSize(a.termSize())
			return a, cmd
		}
		return a, nil
	}

	if a.scr == screenTerminal {
		return a.updateTerminal(msg)
	}
	return a.updateForm(msg)
}

// termSize is the size the ssh pane gets: the full window, minus the one
// line reserved for the hint text above it.
func (a appModel) termSize() (cols, rows int) {
	h := a.height - 1
	if h < 1 {
		h = 1
	}
	return a.width, h
}

func (a appModel) updateTerminal(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok && key.String() == "ctrl+b" {
		_ = a.ssh.Close()
		a.lastErr = a.ssh.Err()
		a.scr = screenForm
		return a, nil
	}

	m, cmd := a.ssh.Update(msg)
	a.ssh = m.(bubblessh.Model)

	if !a.ssh.Connected() && a.ssh.Err() != nil {
		// Stay on screenTerminal — Content() already renders this error,
		// and the user presses ctrl+b when ready to go back and retry.
		a.lastErr = a.ssh.Err()
	}

	return a, cmd
}

func (a appModel) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "tab", "right":
			return a.moveFocus(1)
		case "shift+tab", "left":
			return a.moveFocus(-1)
		case "enter":
			return a.connect()
		case " ":
			switch a.focus {
			case fieldMethod:
				a.usePassword = !a.usePassword
				a.applyMethod()
				return a, nil
			case fieldConnect:
				return a.connect()
			}
			// any other field: fall through, a space is just typed text
		}
	}

	// Everything else — regular keys, and non-key messages like the
	// cursor blink tick — goes to whichever field currently has focus.
	var cmd tea.Cmd
	switch a.focus {
	case fieldAddr:
		a.addr, cmd = a.addr.Update(msg)
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
	a.addr.Blur()
	a.user.Blur()
	a.secret.Blur()
}

func (a *appModel) focusCurrent() tea.Cmd {
	switch a.focus {
	case fieldAddr:
		return a.addr.Focus()
	case fieldUser:
		return a.user.Focus()
	case fieldSecret:
		return a.secret.Focus()
	}
	return nil // fieldMethod and fieldConnect aren't text inputs
}

func (a *appModel) applyMethod() {
	if a.usePassword {
		a.secret.Placeholder = "password"
		a.secret.EchoMode = textinput.EchoPassword
	} else {
		a.secret.Placeholder = "~/.ssh/id_ed25519"
		a.secret.EchoMode = textinput.EchoNormal
	}
}

// connect builds a fresh bubblessh.Model from the current form values and
// switches to the terminal screen. Host keys are trusted-on-first-use
// against the default ~/.ssh/known_hosts automatically (the same trade-off
// bubblessh.WithAcceptNewHostKeys documents: it can't show the interactive
// "are you sure?" prompt a real ssh client would, since Bubble Tea already
// owns the terminal) — see TODO.md for exposing this as a real form field.
func (a appModel) connect() (tea.Model, tea.Cmd) {
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

	a.ssh = bubblessh.New(a.addr.Value(), opts...)
	a.scr = screenTerminal
	a.lastErr = nil
	return a, a.ssh.Init()
}

// expandHome replaces a leading "~" with the user's home directory — Go
// doesn't do this automatically, unlike a shell expanding an unquoted ~
// typed directly on a command line. A value typed into a text field never
// passes through a shell at all.
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

func (a appModel) View() tea.View {
	if a.scr == screenTerminal {
		return tea.NewView(a.viewTerminal())
	}
	return tea.NewView(a.viewForm())
}

var (
	hintStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	fieldStyle        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).BorderForeground(lipgloss.Color("240"))
	focusedFieldStyle = fieldStyle.BorderForeground(lipgloss.Color("39"))
	connectStyle      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).BorderForeground(lipgloss.Color("42")).Foreground(lipgloss.Color("42"))
	statusStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	termPanelStyle    = lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(lipgloss.Color("252"))
)

func (a appModel) viewForm() string {
	hint := hintStyle.Render("LAZYSSH  tab / ←→ move · space toggles method · enter connects")

	box := func(focused bool) lipgloss.Style {
		if focused {
			return focusedFieldStyle
		}
		return fieldStyle
	}

	addrBox := box(a.focus == fieldAddr).Width(18).Render(a.addr.View())
	userBox := box(a.focus == fieldUser).Width(10).Render(a.user.View())

	methodLabel := "method: password"
	if !a.usePassword {
		methodLabel = "method: key"
	}
	methodBox := box(a.focus == fieldMethod).Width(17).Render(methodLabel)

	secretBox := box(a.focus == fieldSecret).Width(22).Render(a.secret.View())
	connectBox := connectStyle.Render("Connect")

	row := lipgloss.JoinHorizontal(lipgloss.Top, addrBox, userBox, methodBox, secretBox, connectBox)

	status := "not connected — fill the form and press connect"
	st := statusStyle
	if a.lastErr != nil {
		status = a.lastErr.Error()
		st = errStyle
	}

	return strings.Join([]string{hint, row, st.Render(status)}, "\n")
}

func (a appModel) viewTerminal() string {
	hint := hintStyle.Render("LAZYSSH  every key is routed to the session · ctrl+b closes it")
	w, h := a.termSize()
	panel := termPanelStyle.Width(w).Height(h).Render(a.ssh.Content())
	return hint + "\n" + panel
}

func main() {
	p := tea.NewProgram(newApp())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "lazyssh:", err)
		os.Exit(1)
	}
}
