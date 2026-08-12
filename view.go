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

func (a *AppModel) View() tea.View {
	v := tea.NewView(a.render())
	v.AltScreen = true
	v.BackgroundColor = termBg
	return v
}

func (a *AppModel) render() string {
	tabBar := a.viewTabBar(a.width)
	panel := a.viewPanel(a.width)
	help := a.viewHelpBar(a.width)
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

func (a *AppModel) connectedCount() int {
	n := 0
	for i := range a.tabs {
		if a.tabs[i].HasSSH && a.tabs[i].SSH.Connected() {
			n++
		}
	}
	return n
}

// viewTabBar renders one small rounded box per tab (dot + label + close glyph),
// plus a trailing "+" box — the browser-tab strip from the design.
func (a *AppModel) viewTabBar(w int) string {
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
	if t.Connected {
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

// viewPanel renders the active tab's content box: the form if it's not
// connected yet, the live terminal if it is. The box always fills the space
// below the tab bar, whichever screen is showing, so switching between them
// never resizes the window.
func (a *AppModel) viewPanel(w int) string {
	t := a.curentTab()
	h := a.panelInnerHeight()

	content := a.viewForm(t)
	if t.Connected {
		content = a.viewTerminal(t)
	}
	return panelStyle.Width(w).Height(h).Align(lipgloss.Center, lipgloss.Center).Render(content)
}

func (a *AppModel) viewForm(t *Tab) string {
	rows := []string{
		// hintStyle.Render("tab / shift+tab move · enter connects"),
		// "",
		a.fieldRow("address", t.Host.View(), t.Focus == fieldHost),
		a.fieldRow("port", t.Port.View(), t.Focus == fieldPort),
		a.fieldRow("user", t.User.View(), t.Focus == fieldUser),
		a.methodRow(t),
		a.fieldRow(t.secretFieldLabel(), t.Secret.View(), t.Focus == fieldSecret),
		"",
		a.connectRow(t),
	}

	if status := a.viewStatus(t); status != "" {
		rows = append(rows, "", status)
	}

	// rows = append(rows, "", hintStyle.Render("host keys: accept-new (silent) · ctrl+t opens a new tab"))

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (a *AppModel) fieldRow(label, value string, focused bool) string {
	style := fieldStyle
	if focused {
		style = focusedFieldStyle
	}
	box := style.Width(a.fieldBoxWidth()).Render(clipLine(value, a.fieldBoxWidth()))
	return lipgloss.JoinHorizontal(lipgloss.Center,
		labelStyle.Render(label),
		lineStyle.Width(labelGap).Render(""),
		box,
	)
}

func (a *AppModel) methodRow(t *Tab) string {
	style := fieldStyle
	if t.Focus == fieldMethod {
		style = focusedFieldStyle
	}

	inner := a.fieldBoxWidth()
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
	if t.Status == "" {
		return ""
	}
	style := statusStyle
	if t.LastErr != nil {
		style = errStyle
	}
	w := a.panelInnerWidth()
	return style.Width(w).Render(clipLine(t.Status, w))
}

func (a *AppModel) viewTerminal(t *Tab) string {
	cols, rows := a.termSize()

	left := lipgloss.JoinHorizontal(lipgloss.Top,
		statusStyle.Render(t.target()),
		"  ",
		hintStyle.Render(t.secretSummary()),
	)

	var badge string
	switch {
	case t.SSH.Connected():
		badge = connectedBadgeStyle.Render("connected")
	case t.LastErr != nil:
		badge = errStyle.Render(t.Status)
	default:
		badge = hintStyle.Render(t.Status)
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

func (a *AppModel) termContent(t *Tab) string {
	if t.HasSSH {
		return t.SSH.Content()
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
// unlike clipLine it never drops lines after the first
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

func (a *AppModel) viewHelpBar(w int) string {
	leftInfo := "ctrl+h — show help"
	rightInfo := fmt.Sprintf("Window size: %dx%d", a.width, a.height)
	padding := 1
	gap := max(1, w-lipgloss.Width(leftInfo)-lipgloss.Width(rightInfo)-padding*2)
	line := leftInfo + strings.Repeat(" ", gap) + rightInfo
	return hintStyle.Width(w).Padding(0, padding).Render(clipLine(line, w))
}
