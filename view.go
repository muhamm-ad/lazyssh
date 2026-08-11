// Rendering: the tab bar and the active tab's panel, stacked exactly as in
// design/SSH_TUI.html. Sizes come from layout.go, colors from styles.go —
// this file only assembles them.

package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

func (a *appModel) View() tea.View {
	v := tea.NewView(a.render())
	v.AltScreen = true
	v.BackgroundColor = termBg
	return v
}

func (a *appModel) render() string {
	w := a.pageWidth()

	page := lipgloss.JoinVertical(lipgloss.Left, a.viewTabBar(w), a.viewPanel(w))
	if !a.confirmQuit {
		return page
	}
	return centerOver(page, a.viewQuitDialog(), w)
}

// centerOver draws modal on its own layer, centered over page. Composing on a
// canvas is what makes it a real modal: the dialog's cells replace whatever was
// underneath instead of pushing the layout around.
//
// Both layers go through a single Compositor on purpose — chaining two
// Canvas.Compose calls instead leaves only the last layer standing.
//
// The canvas is sized to the page, not the window, so putting the dialog up
// never changes how many rows the layout occupies.
func centerOver(page, modal string, w int) string {
	h := lipgloss.Height(page)
	top := lipgloss.NewLayer(modal).
		X(max(0, (w-lipgloss.Width(modal))/2)).
		Y(max(0, (h-lipgloss.Height(modal))/2)).
		Z(1)

	return lipgloss.NewCanvas(w, h).
		Compose(lipgloss.NewCompositor(lipgloss.NewLayer(page), top)).
		Render()
}

func (a *appModel) viewQuitDialog() string {
	lines := []string{dialogTitleStyle.Render("Quit lazyssh?")}
	if n := a.connectedCount(); n > 0 {
		body := "The SSH session will be closed."
		if n > 1 {
			body = fmt.Sprintf("%d SSH sessions will be closed.", n)
		}
		lines = append(lines, dialogBodyStyle.Render(body))
	}
	lines = append(lines, "", dialogKeyStyle.Render("y / enter  quit     n / esc  cancel"))

	return dialogStyle.Render(lipgloss.JoinVertical(lipgloss.Center, lines...))
}

func (a *appModel) connectedCount() int {
	n := 0
	for i := range a.tabs {
		if a.tabs[i].hasSSH && a.tabs[i].ssh.Connected() {
			n++
		}
	}
	return n
}

// viewTabBar renders one small rounded box per tab (dot + label + close
// glyph), plus a trailing "+" box — the browser-tab strip from the design.
func (a *appModel) viewTabBar(w int) string {
	parts := make([]string, 0, len(a.tabs)*2+2)
	for i := range a.tabs {
		if i > 0 {
			parts = append(parts, " ")
		}
		parts = append(parts, a.viewTab(&a.tabs[i]))
	}
	parts = append(parts, " ", addTabStyle.Render("+"))

	bar := lipgloss.JoinHorizontal(lipgloss.Bottom, parts...)
	return lineStyle.Width(w).Render(clipBlock(bar, w))
}

func (a *appModel) viewTab(t *tabState) string {
	style, dot := tabStyle, tabDotOffStyle
	if t.id == a.active {
		style = activeTabStyle
	}
	if t.connected {
		dot = tabDotOnStyle
	}

	label := clipLine(t.tabLabel(), maxTabLabelWidth)
	content := dot.Render("●") + " " + label + " " + tabCloseStyle.Render("×")
	return style.Render(content)
}

// viewPanel renders the active tab's content box: the form if it's not
// connected yet, the live terminal if it is. The box always fills the space
// below the tab bar, whichever screen is showing, so switching between them
// never resizes the window.
func (a *appModel) viewPanel(w int) string {
	t := a.curentTab()
	h := a.panelInnerHeight()

	content := a.viewForm(t)
	if t.connected {
		content = a.viewTerminal(t)
	}
	return panelStyle.Width(w).Height(h).Render(content)
}

func (a *appModel) viewForm(t *tabState) string {
	rows := []string{
		hintStyle.Render("tab / shift+tab move · enter connects"),
		"",
		a.fieldRow("address", t.host.View(), t.focus == fieldHost),
		a.fieldRow("port", t.port.View(), t.focus == fieldPort),
		a.fieldRow("user", t.user.View(), t.focus == fieldUser),
		a.methodRow(t),
		a.fieldRow(t.secretFieldLabel(), t.secret.View(), t.focus == fieldSecret),
		"",
		a.connectRow(t),
	}

	if status := a.viewStatus(t); status != "" {
		rows = append(rows, "", status)
	}

	rows = append(rows, "",
		hintStyle.Render("host keys: accept-new (silent) · ctrl+t opens a new tab"))

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (a *appModel) fieldRow(label, value string, focused bool) string {
	style := fieldStyle
	if focused {
		style = focusedFieldStyle
	}
	box := style.Width(a.fieldBoxWidth()).Render(clipLine(value, a.fieldInputWidth()))
	return lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render(label),
		lineStyle.Width(labelGap).Render(""),
		box,
	)
}

func (a *appModel) methodRow(t *tabState) string {
	style := fieldStyle
	if t.focus == fieldMethod {
		style = focusedFieldStyle
	}

	inner := a.fieldInputWidth()
	left, right := t.methodLabel(), "space to switch"
	content := left
	if pad := inner - lipgloss.Width(left) - lipgloss.Width(right); pad > 0 {
		content = left + strings.Repeat(" ", pad) + hintStyle.Render(right)
	} else {
		content = clipLine(left, inner)
	}

	box := style.Width(a.fieldBoxWidth()).Render(content)
	return lipgloss.JoinHorizontal(lipgloss.Top,
		labelStyle.Render("method"),
		lineStyle.Width(labelGap).Render(""),
		box,
	)
}

func (a *appModel) connectRow(t *tabState) string {
	style := connectStyle
	if t.focus == fieldConnect {
		style = focusedConnectStyle
	}
	indent := lineStyle.Width(labelColWidth + labelGap).Render("")
	return lipgloss.JoinHorizontal(lipgloss.Top, indent, style.Render("Connect"))
}

func (a *appModel) viewStatus(t *tabState) string {
	if t.status == "" {
		return ""
	}
	style := statusStyle
	if t.lastErr != nil {
		style = errStatusStyle
	}
	w := a.panelInnerWidth()
	return style.Width(w).Render(clipLine(t.status, w))
}

func (a *appModel) viewTerminal(t *tabState) string {
	cols, rows := a.termSize()

	left := lipgloss.JoinHorizontal(lipgloss.Top,
		statusStyle.Render(t.target()),
		"  ",
		hintStyle.Render(t.secretSummary()),
	)

	var badge string
	switch {
	case t.ssh.Connected():
		badge = connectedBadgeStyle.Render("connected")
	case t.lastErr != nil:
		badge = errStatusStyle.Render(t.status)
	default:
		badge = hintStyle.Render(t.status)
	}

	gap := max(1, cols-lipgloss.Width(left)-lipgloss.Width(badge))
	header := clipLine(left+strings.Repeat(" ", gap)+badge, cols)

	content := termContentStyle.
		Width(cols).
		Height(rows).
		Render(padBlock(a.termContent(t), cols, rows))

	footer := hintStyle.Render("ctrl+b — close the session and return to the form (values kept)")

	return lipgloss.JoinVertical(lipgloss.Left, header, "", content, "", footer)
}

func (a *appModel) termContent(t *tabState) string {
	if t.hasSSH {
		return t.ssh.Content()
	}
	return ""
}

// clipLine reduces s to its first line, truncated to cols cells. Truncation is
// ANSI-aware, so the styling a textinput emits survives it.
func clipLine(s string, cols int) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if lipgloss.Width(s) <= cols {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(cols).Render(s)
}

// clipBlock clamps every line of a possibly multi-line block to cols cells,
// unlike clipLine it never drops lines after the first — for the tab bar,
// whose boxes are three lines tall (top border, content, bottom border).
func clipBlock(s string, cols int) string {
	if lipgloss.Width(s) <= cols {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(cols).Render(s)
}

// padBlock ensures s fills exactly rows lines of width cols (lipgloss-aware
// width), so the terminal panel doesn't jump when content is short.
func padBlock(s string, cols, rows int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > rows {
		lines = lines[:rows]
	}
	for len(lines) < rows {
		lines = append(lines, "")
	}
	for i, line := range lines {
		if lipgloss.Width(line) > cols {
			lines[i] = lipgloss.NewStyle().MaxWidth(cols).Render(line)
		}
	}
	return strings.Join(lines, "\n")
}
