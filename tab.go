package main

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"

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
	Connected                bool // true => this tab shows the terminal screen

	SSH     bubblessh.Model
	HasSSH  bool // true once connect has built a Model at least once
	LastErr error
	Status  string
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

// syncFromSSH mirrors the ssh sub-model's state into the tab's own status
// fields, so the tab bar dot and the form's status line stay correct even for
// a session that's currently running in the background.
func (t *Tab) syncFromSSH() {
	switch {
	case t.SSH.Connected():
		t.Connected = true
		t.Status = fmt.Sprintf("connected — %s@%s", t.User.Value(), t.Host.Value())
		t.LastErr = nil
	case t.Connected:
		// Still dialing, or the connection just ended.
		content := t.SSH.Content()
		if strings.HasPrefix(content, "connecting") {
			t.Status = content
			return
		}
		t.Connected = false
		if t.SSH.Err() != nil {
			t.LastErr = t.SSH.Err()
			t.Status = t.LastErr.Error()
		} else {
			t.LastErr = nil
			t.Status = ""
		}
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

// secretSummary is the one-line auth summary shown on the terminal screen.
func (t *Tab) secretSummary() string {
	if t.UsePassword {
		return "password auth"
	}
	return "key: " + t.Secret.Value()
}

// target is the "user@host:port" line shown on the terminal screen.
func (t *Tab) target() string {
	host := t.Host.Value()
	if host == "" {
		host = "host"
	}
	user := t.User.Value()
	if user == "" {
		user = "user"
	}
	return user + "@" + host + ":" + t.Port.Value()
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
