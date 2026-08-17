package app

import (
	"io"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"

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

type Tab struct {
	ID int

	Host, Port, User, Secret textinput.Model
	UsePassword              bool // method toggle: true = password, false = private key
	Focus                    int  // one of the fieldXxx constants above

	SSH *bubblessh.Model

	Status       string
	TextSelected bool
	selStart     int
	selEnd       int
	lastNotified string // last BubbleUp message fired for this tab
	wasLive      bool   // previous inSession() — detects disconnect edges

	termSel termSelection
	undo    []fieldSnapshot
}

type termSelection struct {
	on     bool
	ax, ay int
	bx, by int
}

type fieldSnapshot struct {
	Host, Port, User, Secret             string
	HostPos, PortPos, UserPos, SecretPos int
	Focus                                int
	TextSelected                         bool
	selStart, selEnd                     int
}

const (
	HostCharLimit   = 64
	PortCharLimit   = 5
	UserCharLimit   = 32
	SecretCharLimit = 64
)

func NewTab(id int) Tab {
	host := textinput.New()
	host.Prompt = ""
	host.Placeholder = "host"
	host.CharLimit = HostCharLimit
	styleInput(&host)

	port := textinput.New()
	port.Prompt = ""
	port.Placeholder = "22"
	port.SetValue("22")
	port.CharLimit = PortCharLimit
	styleInput(&port)

	user := textinput.New()
	user.Prompt = ""
	user.Placeholder = "user"
	user.CharLimit = UserCharLimit
	styleInput(&user)

	secret := textinput.New()
	secret.Prompt = ""
	secret.CharLimit = SecretCharLimit
	styleInput(&secret)

	tab := Tab{
		ID:          id,
		Host:        host,
		Port:        port,
		User:        user,
		UsePassword: false,
		Secret:      secret,
		Focus:       fieldHost,
	}
	tab.ApplyMethod()
	return tab
}

// inSession is true while this tab should show the terminal screen: either
// still dialing or already up. Error/closed/no-model fall back to the form.
func (t *Tab) inSession() bool {
	if t.SSH == nil {
		return false
	}
	s := t.SSH.State()
	return s == bubblessh.StateConnecting || s == bubblessh.StateConnected
}

// statusText is the one-line message under the form (and the terminal badge
// while connecting). Prefers a form-level Status when the session is not live.
func (t *Tab) statusText() string {
	if t.SSH != nil {
		// switch t.SSH.State() {
		// case bubblessh.StateConnecting:
		// 	return "connecting…"
		// case bubblessh.StateConnected:
		// 	return fmt.Sprintf("connected — %s@%s", t.User.Value(), t.Host.Value())
		// }
		if t.Status != "" {
			return friendlyError(t.Status)
		}
		if err := t.SSH.Err(); err != nil && err != io.EOF {
			return friendlyError(err.Error())
		}
		return ""
	}
	if t.Status != "" && !t.inSession() {
		return friendlyError(t.Status)
	}
	return t.Status
}

// friendlyError turns a raw SSH / network error into a short line the form
// can show. The original text is often a wrapped stdlib dump; this keeps the
// cause and drops the rest.
func friendlyError(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return ""
	}
	low := strings.ToLower(s)
	switch {
	case strings.Contains(low, "invalid port"):
		return s
	case strings.Contains(low, "no such host"), strings.Contains(low, "server misbehaving"):
		return "host not found — check the address"
	case strings.Contains(low, "connection refused"):
		return "connection refused — check host and port"
	case strings.Contains(low, "i/o timeout"), strings.Contains(low, "deadline exceeded"), strings.Contains(low, "timed out"):
		return "connection timed out"
	case strings.Contains(low, "network is unreachable"), strings.Contains(low, "no route to host"):
		return "network unreachable"
	case strings.Contains(low, "connection reset"):
		return "connection reset by the remote host"
	case strings.Contains(low, "knownhosts"), strings.Contains(low, "key mismatch"), strings.Contains(low, "remote host identification"):
		return "host key changed — check known_hosts"
	case strings.Contains(low, "unable to authenticate"), strings.Contains(low, "no supported methods remain"), strings.Contains(low, "permission denied"):
		return "authentication failed — check user, key, or password"
	case strings.Contains(low, "passphrase"):
		return "this key needs a passphrase"
	case strings.Contains(low, "no such file"), strings.Contains(low, "cannot find the file"):
		return "key file not found"
	case strings.Contains(low, "unable to parse"), strings.Contains(low, "parse private key"), strings.Contains(low, "ssh: this private key"):
		return "couldn't read the private key"
	case strings.Contains(low, "handshake"):
		return "SSH handshake failed"
	default:
		s = strings.TrimPrefix(s, "ssh: ")
		s = strings.TrimPrefix(s, "ssh error: ")
		return s
	}
}

func (t *Tab) statusIsError() bool {
	if t.Status != "" && !t.inSession() {
		return true
	}
	if t.SSH == nil {
		return false
	}
	err := t.SSH.Err()
	return (err != nil && err != io.EOF) || t.SSH.State() == bubblessh.StateError
}

func (t *Tab) updateSSH(msg tea.Msg) tea.Cmd {
	if t.SSH == nil {
		return nil
	}
	m, cmd := t.SSH.Update(msg)
	updated := m.(bubblessh.Model)
	t.SSH = &updated
	return cmd
}

func (t *Tab) setSSHSize(cols, rows int) tea.Cmd {
	if t.SSH == nil {
		return nil
	}
	updated, cmd := t.SSH.SetSize(cols, rows)
	t.SSH = &updated
	return cmd
}

func (t *Tab) focusedInput() *textinput.Model {
	switch t.Focus {
	case fieldHost:
		return &t.Host
	case fieldPort:
		return &t.Port
	case fieldUser:
		return &t.User
	case fieldSecret:
		return &t.Secret
	default:
		return nil
	}
}

func (t *Tab) secretLocked() bool {
	return t.UsePassword && t.Focus == fieldSecret
}

func (t *Tab) selectAllFocused() {
	in := t.focusedInput()
	if in == nil {
		return
	}
	t.selStart = 0
	t.selEnd = len([]rune(in.Value()))
	t.TextSelected = t.selStart != t.selEnd
}

func (t *Tab) selectedRange() (lo, hi int, ok bool) {
	if !t.TextSelected {
		return 0, 0, false
	}
	lo, hi = t.selStart, t.selEnd
	if lo > hi {
		lo, hi = hi, lo
	}
	return lo, hi, lo != hi
}

func (t *Tab) selectedText() string {
	if t.secretLocked() {
		return ""
	}
	in := t.focusedInput()
	if in == nil {
		return ""
	}
	runes := []rune(in.Value())
	lo, hi, ok := t.selectedRange()
	if !ok {
		return in.Value()
	}
	lo = max(0, min(lo, len(runes)))
	hi = max(0, min(hi, len(runes)))
	return string(runes[lo:hi])
}

func (t *Tab) copyFocused() {
	if t.secretLocked() {
		return
	}
	_ = clipboard.WriteAll(t.selectedText())
}

func (t *Tab) cutFocused() {
	if t.secretLocked() {
		return
	}
	in := t.focusedInput()
	if in == nil {
		return
	}
	t.pushUndo()
	_ = clipboard.WriteAll(t.selectedText())
	t.deleteSelected()
}

func (t *Tab) clearSelection() {
	t.TextSelected = false
	t.selStart, t.selEnd = 0, 0
}

func (t *Tab) deleteSelected() {
	in := t.focusedInput()
	lo, hi, ok := t.selectedRange()
	if in == nil || !ok {
		t.clearSelection()
		return
	}
	runes := []rune(in.Value())
	lo = max(0, min(lo, len(runes)))
	hi = max(0, min(hi, len(runes)))
	in.SetValue(string(append(append([]rune{}, runes[:lo]...), runes[hi:]...)))
	in.SetCursor(lo)
	t.clearSelection()
}

func (t *Tab) replaceSelectedIfNeeded() {
	if !t.TextSelected {
		return
	}
	t.deleteSelected()
}

func (t *Tab) capture() fieldSnapshot {
	return fieldSnapshot{
		Host:         t.Host.Value(),
		Port:         t.Port.Value(),
		User:         t.User.Value(),
		Secret:       t.Secret.Value(),
		HostPos:      t.Host.Position(),
		PortPos:      t.Port.Position(),
		UserPos:      t.User.Position(),
		SecretPos:    t.Secret.Position(),
		Focus:        t.Focus,
		TextSelected: t.TextSelected,
		selStart:     t.selStart,
		selEnd:       t.selEnd,
	}
}

func (t *Tab) pushUndo() {
	t.undo = append(t.undo, t.capture())
	const maxUndo = 64
	if len(t.undo) > maxUndo {
		t.undo = t.undo[len(t.undo)-maxUndo:]
	}
}

func (t *Tab) dropUndoIfUnchanged() {
	if len(t.undo) == 0 {
		return
	}
	last := t.undo[len(t.undo)-1]
	if t.Host.Value() == last.Host &&
		t.Port.Value() == last.Port &&
		t.User.Value() == last.User &&
		t.Secret.Value() == last.Secret {
		t.undo = t.undo[:len(t.undo)-1]
	}
}

func (t *Tab) undoLast() {
	if len(t.undo) == 0 {
		return
	}
	s := t.undo[len(t.undo)-1]
	t.undo = t.undo[:len(t.undo)-1]
	t.Host.SetValue(s.Host)
	t.Port.SetValue(s.Port)
	t.User.SetValue(s.User)
	t.Secret.SetValue(s.Secret)
	t.Host.SetCursor(s.HostPos)
	t.Port.SetCursor(s.PortPos)
	t.User.SetCursor(s.UserPos)
	t.Secret.SetCursor(s.SecretPos)
	t.Focus = s.Focus
	t.TextSelected = s.TextSelected
	t.selStart = s.selStart
	t.selEnd = s.selEnd
}

func (t *Tab) closeSSH() {
	if t.SSH == nil {
		return
	}
	_ = t.SSH.Close()
	t.SSH = nil
}

func (t *Tab) SetPassword(usePassword string) {
	t.UsePassword = true
	t.Secret.Placeholder = "password"
	t.Secret.EchoMode = textinput.EchoPassword
	t.Secret.SetValue(usePassword)
}

func (t *Tab) ApplyMethod() {
	if t.UsePassword {
		t.Secret.Placeholder = "password"
		t.Secret.EchoMode = textinput.EchoPassword
		t.Secret.SetValue("")
	} else {
		t.Secret.Placeholder = "~/.ssh/id_ed25519"
		t.Secret.EchoMode = textinput.EchoNormal
		t.Secret.SetValue("~/.ssh/id_ed25519")
	}
}

func (t *Tab) methodLabel() string {
	label := "publickey"
	if t.UsePassword {
		label = "password"
	}
	return label
}

// secretFieldLabel is the row label to the left of the secret field — it
// swaps with the method, same as the design.
func (t *Tab) secretFieldLabel() string {
	if t.UsePassword {
		return "password"
	}
	return "key path"
}

// GetTabLabel is what's shown in the tab strip — matches the design's
// `tab.host ? tab.user + "@" + tab.host : "nouvel onglet"`.
func (t *Tab) GetTabLabel() string {
	if t.Host.Value() == "" {
		return "new tab"
	}
	if t.User.Value() == "" {
		return t.Host.Value()
	}
	return t.User.Value() + "@" + t.Host.Value()
}
