package app

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	bubbleup "go.dalton.dog/bubbleup/v2"
)

const alertBottomPad = 2 // empty panel rows kept clear below the alert

func newAlertModel(maxWidth int) bubbleup.AlertModel {
	if maxWidth < 20 {
		maxWidth = 20
	}
	return bubbleup.NewAlertModel(maxWidth, false, 5*time.Second).
		WithMinWidth(1). // dynamic width: box hugs the full message
		WithUnicodePrefix().
		WithPosition(bubbleup.BottomCenterPosition).
		WithAllowEscToClose()
}

// syncAlertWidth rebuilds the alert model so its max width tracks the window.
// BubbleUp has no SetWidth — recreating is the only way to grow with resize.
func (a *AppModel) syncAlertWidth() {
	a.alert = newAlertModel(max(20, a.width-4))
}

func (a *AppModel) pipeAlert(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	out, alertCmd := a.alert.Update(msg)
	a.alert = out.(bubbleup.AlertModel)
	return a, tea.Batch(cmd, alertCmd)
}

func (a *AppModel) notify(key, message string) tea.Cmd {
	// Collapse whitespace/newlines but never truncate — the alert box grows
	// to the message (up to the window width).
	message = strings.Join(strings.Fields(message), " ")
	if message == "" {
		return nil
	}
	return a.alert.NewAlertCmd(key, message)
}

func (a *AppModel) notifyInfo(message string) tea.Cmd {
	return a.notify(bubbleup.InfoKey, message)
}

func (a *AppModel) notifyWarn(message string) tea.Cmd {
	return a.notify(bubbleup.WarnKey, message)
}

func (a *AppModel) notifyError(message string) tea.Cmd {
	return a.notify(bubbleup.ErrorKey, message)
}

// alertFromTabs fires a BubbleUp notification when a tab's status text changes
// (connecting, connected, or a friendly error). Deduped per tab so ticks
// don't spam the same message.
func (a *AppModel) alertFromTabs() tea.Cmd {
	var cmds []tea.Cmd
	for i := range a.tabs {
		if cmd := a.alertFromTab(&a.tabs[i]); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func (a *AppModel) alertFromTab(t *Tab) tea.Cmd {
	live := t.inSession()
	wasLive := t.wasLive

	s := t.statusText()
	switch {
	case live && s != "" && s != t.lastNotified:
		t.lastNotified = s
		t.wasLive = true
		key := bubbleup.InfoKey
		if t.statusIsError() {
			key = bubbleup.ErrorKey
		}
		return a.notify(key, s)

	case !live && wasLive:
		t.wasLive = false
		if t.statusIsError() {
			msg := s
			if msg == "" {
				msg = "connection failed"
			}
			t.lastNotified = msg
			return a.notifyError(msg)
		}
		t.lastNotified = "session closed"
		return a.notifyWarn(fmt.Sprintf("%s — session closed", t.GetTabLabel()))

	case !live && s != "" && s != t.lastNotified:
		// Form-level / setup errors (invalid port, key not found before dial).
		t.lastNotified = s
		if t.statusIsError() {
			return a.notifyError(s)
		}
		return a.notifyInfo(s)
	}

	t.wasLive = live
	if s == "" && !live {
		t.lastNotified = ""
	}
	return nil
}
