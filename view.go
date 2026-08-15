// Rendering: the tab bar and the active tab's panel, stacked exactly as in
// design/SSH_TUI.html. Sizes come from layout.go, colors from styles.go —
// this file only assembles them.

package main

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

func (a *AppModel) View() tea.View {
	v := tea.NewView(a.render())
	v.AltScreen = true
	v.BackgroundColor = termBg
	v.Cursor = a.formCursor()
	if t := a.curentTab(); t.inSession() {
		v.Cursor = a.terminalCursor()
	}
	return v
}

// terminalCursor places a hardware cursor at the live PTY's own cursor
// position once a session is connected. Cursor() reports content-local
// coordinates (0,0 at the top-left of the PTY content), so it's offset here
// by everywhere that content sits on screen: past the tab bar, past the
// panel's own border and padding, and past any header rows viewTerminal adds.
func (a *AppModel) terminalCursor() *tea.Cursor {
	t := a.curentTab()
	if t.SSH == nil {
		return nil
	}
	c := t.SSH.Cursor()
	if c == nil {
		return nil
	}
	dx := panelStyle.GetBorderLeftSize() + panelStyle.GetPaddingLeft()
	dy := tabBarHeight + panelStyle.GetBorderTopSize() + panelStyle.GetPaddingTop()
	c.Position.X += dx
	c.Position.Y += dy
	return c
}

func (a *AppModel) render() string {
	tabBar := a.viewAppTabBar(a.width)
	panel := a.viewAppPanel(a.width)
	help := a.viewAppStatusBar(a.width)
	page := lipgloss.JoinVertical(lipgloss.Left, tabBar, panel, help)
	switch {
	case a.confirmQuit:
		return centerOver(page, a.viewQuitDialog(), a.width, a.height)
	case a.showHelp:
		return centerOver(page, a.viewHelpModal(), a.width, a.height)
	default:
		return page
	}
}

func centerOver(page, modal string, w int, h int) string {
	topLayer := lipgloss.NewLayer(modal).
		X(max(0, (w-lipgloss.Width(modal))/2)).
		Y(max(0, (h-lipgloss.Height(modal))/2)).
		Z(1)

	bottomLayer := lipgloss.NewLayer(page)

	return lipgloss.NewCanvas(w, h).
		Compose(lipgloss.NewCompositor(bottomLayer, topLayer)).Render()
}

func (a *AppModel) viewQuitDialog() string {
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

func (a *AppModel) connectedCount() int {
	n := 0
	for i := range a.tabs {
		if a.tabs[i].SSH != nil && a.tabs[i].SSH.Connected() {
			n++
		}
	}
	return n
}

func (a *AppModel) viewHelpModal() string {
	rows := []string{dialogTitleStyle.Render("Help"), ""}
	for i, cat := range helpCategories {
		if i > 0 {
			rows = append(rows, "")
		}
		rows = append(rows, helpCategoryStyle.Render(cat.title))
		for _, e := range cat.entries {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
				helpKeyStyle.Render(e.keys),
				helpDescStyle.Render(e.desc),
			))
		}
	}
	rows = append(rows, "", dialogKeyStyle.Render("esc / ctrl+h  close"))

	return dialogStyle.
		Align(lipgloss.Left).
		Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}

// viewAppTabBar renders one small rounded box per tab (dot + label + close glyph),
// plus a trailing "+" box — the browser-tab strip from the design.
func (a *AppModel) viewAppTabBar(w int) string {
	parts := make([]string, 0, len(a.tabs)*2+2)
	for i := range a.tabs {
		if i > 0 {
			parts = append(parts, " ")
		}
		parts = append(parts, a.viewTab(&a.tabs[i]))
	}
	parts = append(parts, " ", addTabStyle.Render("+"))

	tabBar := lipgloss.JoinHorizontal(lipgloss.Bottom, parts...)
	return lineStyle.Width(w).Render(clipBlock(tabBar, w))
}

func (a *AppModel) viewTab(t *Tab) string {
	style, dot := tabStyle, tabDotOffStyle
	if t.ID == a.active {
		style = activeTabStyle
	}
	if t.inSession() {
		dot = tabDotOnStyle
	}

	label := clipLine(t.GetTabLabel(), maxTabLabelWidth)
	content := lipgloss.JoinHorizontal(lipgloss.Center,
		dot.Render("●"),
		" ",
		label,
		" ",
		tabCloseStyle.Render("×"),
	)
	return style.Render(content)
}

// viewAppPanel renders the active tab's content box: the form if it's not
// connected yet, the live terminal if it is. The box always fills the space
// below the tab bar, whichever screen is showing, so switching between them
// never resizes the window.
func (a *AppModel) viewAppPanel(w int) string {
	t := a.curentTab()
	h := a.panelBoxHeight()

	content := a.viewForm(t)
	if t.inSession() {
		content = a.viewTerminal(t)
	}
	return panelStyle.Width(w).Height(h).Align(lipgloss.Center, lipgloss.Center).Render(content)
}

func (a *AppModel) viewForm(t *Tab) string {
	rows := a.formRows(t)
	if status := a.viewStatus(t); status != "" {
		rows = append(rows, "", status)
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (a *AppModel) formRows(t *Tab) []string {
	return []string{
		a.fieldRow("address", &t.Host, t.Focus == fieldHost, t.TextSelected && t.Focus == fieldHost),
		a.fieldRow("port", &t.Port, t.Focus == fieldPort, t.TextSelected && t.Focus == fieldPort),
		a.fieldRow("user", &t.User, t.Focus == fieldUser, t.TextSelected && t.Focus == fieldUser),
		a.methodRow(t),
		a.fieldRow(t.secretFieldLabel(), &t.Secret, t.Focus == fieldSecret, t.TextSelected && t.Focus == fieldSecret),
		"",
		a.connectRow(t),
	}
}

func (a *AppModel) fieldRow(label string, in *textinput.Model, focused, selected bool) string {
	style := fieldStyle
	if focused {
		style = focusedFieldStyle
	}
	inner := a.fieldInnerWidth()
	value := in.View()
	if selected && in.Value() != "" {
		value = selectedTextStyle.Render(echoedValue(*in))
	}
	box := style.Width(a.fieldBoxWidth()).Render(clipLine(value, inner))
	return lipgloss.JoinHorizontal(lipgloss.Center,
		labelStyle.Render(label),
		lineStyle.Width(labelGap).Render(""),
		box,
	)
}

func echoedValue(in textinput.Model) string {
	v := in.Value()
	if in.EchoMode == textinput.EchoPassword {
		ch := in.EchoCharacter
		if ch == 0 {
			ch = '*'
		}
		return strings.Repeat(string(ch), len([]rune(v)))
	}
	return v
}

// formCursor places a real bar cursor inside the focused form field.
func (a *AppModel) formCursor() *tea.Cursor {
	if a.confirmQuit || a.showHelp {
		return nil
	}
	t := a.curentTab()
	if t.inSession() || t.TextSelected {
		return nil
	}
	in := t.focusedInput()
	if in == nil {
		return nil
	}
	c := in.Cursor()
	if c == nil {
		return nil
	}

	form := a.viewForm(t)
	formW, formH := lipgloss.Width(form), lipgloss.Height(form)
	innerW := a.panelInnerWidth()
	innerH := a.panelInnerHeight()
	formX := panelStyle.GetBorderLeftSize() + panelStyle.GetPaddingLeft() + max(0, (innerW-formW)/2)
	formY := tabBarHeight + panelStyle.GetBorderTopSize() + panelStyle.GetPaddingTop() + max(0, (innerH-formH)/2)

	y := formY
	for i, row := range a.formRows(t) {
		if i == t.Focus {
			c.Position.X += formX + labelColWidth + labelGap + fieldStyle.GetBorderLeftSize() + fieldStyle.GetPaddingLeft()
			c.Position.Y += y + fieldStyle.GetBorderTopSize()
			return c
		}
		y += lipgloss.Height(row)
	}
	return nil
}

func (a *AppModel) methodRow(t *Tab) string {
	style := fieldStyle
	if t.Focus == fieldMethod {
		style = focusedFieldStyle
	}

	inner := a.fieldInnerWidth()
	left := t.methodLabel()
	hint := hintStyle.Render("(space to switch)")
	content := left + " " + hint
	if lipgloss.Width(content) > inner {
		content = clipLine(left, inner)
	}

	box := style.Width(a.fieldBoxWidth()).Render(content)
	return lipgloss.JoinHorizontal(lipgloss.Center,
		labelStyle.Render("method"),
		lineStyle.Width(labelGap).Render(""),
		box,
	)
}

func (a *AppModel) connectRow(t *Tab) string {
	style := connectStyle
	if t.Focus == fieldConnect {
		style = focusedConnectStyle
	}
	formW := labelColWidth + labelGap + a.fieldBoxWidth()
	return lineStyle.Width(formW).Align(lipgloss.Center).Render(style.Render("Connect"))
}

func (a *AppModel) viewStatus(t *Tab) string {
	s := t.statusText()
	if s == "" {
		return ""
	}
	style := statusStyle
	if t.statusIsError() {
		style = errStyle
	}
	w := a.panelInnerWidth()
	return style.Width(w).Render(clipLine(s, w))
}

func (a *AppModel) viewTerminal(t *Tab) string {
	cols, rows := a.termSize()
	content := ""
	if t.SSH != nil {
		content = t.SSH.Content()
	}
	return termContentStyle.
		Width(cols).Height(rows).
		Render(padBlock(content, cols, rows))
}

// viewAppStatusBar renders the status bar at the bottom of the app.
func (a *AppModel) viewAppStatusBar(w int) string {
	leftInfo := "ctrl+h — show help"
	rightInfo := fmt.Sprintf("Window size: %dx%d", a.width, a.height)
	padding := 1
	gap := max(1, w-lipgloss.Width(leftInfo)-lipgloss.Width(rightInfo)-padding*2)
	line := leftInfo + strings.Repeat(" ", gap) + rightInfo
	return hintStyle.Width(w).Padding(0, padding).Render(clipLine(line, w))
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
// unlike clipLine it never drops lines after the first
func clipBlock(s string, cols int) string {
	if lipgloss.Width(s) <= cols {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(cols).Render(s)
}

// padBlock ensures s fills exactly rows lines of width cols, so the
// terminal panel doesn't jump when content is short.
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
