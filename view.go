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
	"github.com/charmbracelet/x/ansi"
)

func (a *AppModel) View() tea.View {
	v := tea.NewView(a.render())
	v.AltScreen = true
	v.BackgroundColor = termBg
	v.MouseMode = tea.MouseModeCellMotion
	v.Cursor = a.formCursor()
	if t := a.curentTab(); t.inSession() && !a.focusAdd && !t.hasTermSelection() {
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
	panel := a.renderAlertInPanel(a.viewAppPanel(a.width))
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

// renderAlertInPanel overlays BubbleUp on the content panel only (not the tab
// bar or help strip), bottom-centered with alertBottomPad rows kept clear
// above the panel's bottom edge.
func (a *AppModel) renderAlertInPanel(panel string) string {
	lines := strings.Split(panel, "\n")
	pad := alertBottomPad
	if pad <= 0 || len(lines) <= pad {
		return a.alert.Render(panel)
	}
	head := strings.Join(lines[:len(lines)-pad], "\n")
	tail := strings.Join(lines[len(lines)-pad:], "\n")
	return lipgloss.JoinVertical(lipgloss.Left, a.alert.Render(head), tail)
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
	left := renderHelpColumn(helpCategories[:3])
	right := renderHelpColumn(helpCategories[3:])
	gap := strings.Repeat(" ", 4)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, gap, right)

	return dialogStyle.
		Align(lipgloss.Left).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			dialogTitleStyle.Render("Help"),
			"",
			body,
			"",
			dialogKeyStyle.Render("esc / ctrl+h  close"),
		))
}

func renderHelpColumn(cats []helpCategory) string {
	rows := make([]string, 0, 32)
	for i, cat := range cats {
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
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// viewAppTabBar renders one small rounded box per tab (dot + label + close glyph),
// plus a trailing "+" box — the browser-tab strip from the design. When the
// strip is wider than the window the tabs scroll horizontally so the active
// tab stays fully in view; the "+" stays pinned on the right.
func (a *AppModel) viewAppTabBar(w int) string {
	addStyle := addTabStyle
	if a.focusAdd {
		addStyle = activeTabStyle
	}
	add := addStyle.Render("+")
	strip := a.measureTabStrip()
	tabsW := max(0, w-strip.addW)
	offset := max(0, min(a.tabBarScroll, max(0, strip.contentW-tabsW)))
	tabs := scrollBlock(strip.content, offset, tabsW)

	tabBar := lipgloss.JoinHorizontal(lipgloss.Bottom, tabs, " ", add)
	return lineStyle.Width(w).Render(tabBar)
}

type tabStrip struct {
	starts   []int
	widths   []int
	content  string
	contentW int
	addW     int // leading gap + "+" box
}

func (a *AppModel) measureTabStrip() tabStrip {
	parts := make([]string, 0, len(a.tabs)*2)
	starts := make([]int, len(a.tabs))
	widths := make([]int, len(a.tabs))
	x := 0
	for i := range a.tabs {
		if i > 0 {
			parts = append(parts, " ")
			x++
		}
		tab := a.viewTab(&a.tabs[i])
		starts[i] = x
		widths[i] = lipgloss.Width(tab)
		x += widths[i]
		parts = append(parts, tab)
	}
	content := lipgloss.JoinHorizontal(lipgloss.Bottom, parts...)
	addW := 1 + lipgloss.Width(addTabStyle.Render("+"))
	return tabStrip{
		starts:   starts,
		widths:   widths,
		content:  content,
		contentW: lipgloss.Width(content),
		addW:     addW,
	}
}

func (a *AppModel) clampTabBarScroll() {
	strip := a.measureTabStrip()
	tabsW := max(0, a.width-strip.addW)
	a.tabBarScroll = max(0, min(a.tabBarScroll, max(0, strip.contentW-tabsW)))
}

func (a *AppModel) ensureActiveTabVisible() {
	if a.focusAdd {
		a.clampTabBarScroll()
		return
	}
	strip := a.measureTabStrip()
	tabsW := max(0, a.width-strip.addW)
	idx := a.currentTabIndex()
	a.tabBarScroll = tabBarOffset(strip.starts[idx], strip.starts[idx]+strip.widths[idx], strip.contentW, tabsW)
}

// tabBarOffset is the leftmost column of the tab strip to show so [tabStart,
// tabEnd) sits inside a viewport of viewW. Prefers keeping the active tab's
// right edge visible when it would otherwise hang off the right.
func tabBarOffset(tabStart, tabEnd, contentW, viewW int) int {
	if contentW <= viewW || viewW <= 0 {
		return 0
	}
	maxOff := contentW - viewW
	off := 0
	if tabEnd > viewW {
		off = tabEnd - viewW
	}
	if tabStart < off {
		off = tabStart
	}
	return max(0, min(off, maxOff))
}

func (a *AppModel) viewTab(t *Tab) string {
	style, dot := tabStyle, tabDotOffStyle
	if t.ID == a.active && !a.focusAdd {
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
	return lipgloss.JoinVertical(lipgloss.Left, a.formRows(t)...)
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
		t := a.curentTab()
		value = highlightRange(echoedValue(*in), t.selStart, t.selEnd)
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
	if a.confirmQuit || a.showHelp || a.focusAdd {
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
	formW := a.formWidth()
	return lineStyle.Width(formW).Align(lipgloss.Center).Render(style.Render("Connect"))
}

func (a *AppModel) viewTerminal(t *Tab) string {
	cols, rows := a.termSize()
	content := ""
	if t.SSH != nil {
		content = t.SSH.Content()
	}
	content = padBlock(content, cols, rows)
	if t.hasTermSelection() {
		content = highlightTermSelection(content, t.termSel, cols)
	}
	return termContentStyle.
		Width(cols).Height(rows).
		Render(content)
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

// scrollBlock windows every line of a possibly multi-line block to cols cells
// starting at offset, ANSI-aware so tab borders and colors survive the cut.
func scrollBlock(s string, offset, cols int) string {
	if cols <= 0 {
		return ""
	}
	if offset < 0 {
		offset = 0
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		width := lipgloss.Width(line)
		off := offset
		if width <= cols {
			off = 0
		} else if off > width-cols {
			off = width - cols
		}
		lines[i] = ansi.Cut(line, off, off+cols)
	}
	return strings.Join(lines, "\n")
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
