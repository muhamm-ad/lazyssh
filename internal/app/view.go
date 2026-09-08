package app

// Rendering: the tab bar and the active tab's panel, stacked exactly as in
// design/SSH_TUI.html. Sizes come from layout.go, colors from styles.go —
// this file only assembles them.

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
	if a.tooSmall() || a.confirmQuit || a.showHelp {
		return v
	}
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
	case a.tooSmall():
		return centerOver(page, a.viewMinSizeDialog(), a.width, a.height)
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

// blurContent dims the page behind a modal: strip colors and paint it faint so
// the dialog reads as the only interactive layer.
func blurContent(s string) string {
	return lipgloss.NewStyle().
		Foreground(dimFg).
		Faint(true).
		Render(ansi.Strip(s))
}

func centerOver(page, modal string, w int, h int) string {
	x, y := modalOrigin(modal, w, h)
	topLayer := lipgloss.NewLayer(modal).
		X(x).
		Y(y).
		Z(1)

	bottomLayer := lipgloss.NewLayer(blurContent(page))

	return lipgloss.NewCanvas(max(1, w), max(1, h)).
		Compose(lipgloss.NewCompositor(bottomLayer, topLayer)).Render()
}

func modalOrigin(modal string, w, h int) (x, y int) {
	return max(0, (w-lipgloss.Width(modal))/2), max(0, (h-lipgloss.Height(modal))/2)
}

func (a *AppModel) pointInModal(x, y int, modal string) bool {
	mx, my := modalOrigin(modal, a.width, a.height)
	mw, mh := lipgloss.Width(modal), lipgloss.Height(modal)
	return x >= mx && x < mx+mw && y >= my && y < my+mh
}

func (a *AppModel) viewMinSizeDialog() string {
	required := fmt.Sprintf("%d × %d", minAppWidth, minAppHeight)
	wStyle, hStyle := dialogTitleStyle, dialogTitleStyle
	if a.width < minAppWidth {
		wStyle = dialogErrStyle
	}
	if a.height < minAppHeight {
		hStyle = dialogErrStyle
	}
	current := lipgloss.JoinHorizontal(lipgloss.Center,
		wStyle.Render(fmt.Sprintf("%d", a.width)),
		dialogTitleStyle.Render(" × "),
		hStyle.Render(fmt.Sprintf("%d", a.height)),
	)
	lines := []string{
		dialogTitleStyle.Render("Window too small"),
		"",
		dialogBodyStyle.Render("Minimum required size:"),
		dialogTitleStyle.Render(required),
		"",
		dialogBodyStyle.Render("Current size:"),
		current,
		"",
		dialogKeyStyle.Render("Resize the terminal to continue"),
	}
	return dialogStyle.Render(lipgloss.JoinVertical(lipgloss.Center, lines...))
}

const (
	quitBtnQuit   = 0
	quitBtnCancel = 1
	quitBtnGap    = 2
)

func (a *AppModel) quitDialogLines() []string {
	lines := []string{dialogTitleStyle.Render("Quit lazyssh?")}
	if n := a.connectedCount(); n > 0 {
		body := "The SSH session will be closed."
		if n > 1 {
			body = fmt.Sprintf("%d SSH sessions will be closed.", n)
		}
		lines = append(lines, dialogBodyStyle.Render(body))
	}
	lines = append(lines, "", a.quitButtonsRow())
	return lines
}

func (a *AppModel) viewQuitDialog() string {
	return dialogStyle.Render(lipgloss.JoinVertical(lipgloss.Center, a.quitDialogLines()...))
}

func (a *AppModel) quitButtonsRow() string {
	quit := a.quitButton("Quit", a.quitFocus == quitBtnQuit, true)
	cancel := a.quitButton("Cancel", a.quitFocus == quitBtnCancel, false)
	return lipgloss.JoinHorizontal(lipgloss.Center, quit, strings.Repeat(" ", quitBtnGap), cancel)
}

func (a *AppModel) quitButton(label string, focused, danger bool) string {
	style := dialogBtnStyle
	if danger {
		style = dialogQuitBtnStyle
	}
	if focused {
		style = style.BorderForeground(accentFg)
	}
	return style.Render(label)
}

// hitQuitDialogButton returns quitBtnQuit, quitBtnCancel, or -1 if the click
// landed on the dialog but not on a button.
func (a *AppModel) hitQuitDialogButton(x, y int) int {
	modal := a.viewQuitDialog()
	dx, dy := modalOrigin(modal, a.width, a.height)
	ix := dx + dialogStyle.GetBorderLeftSize() + dialogStyle.GetPaddingLeft()
	iy := dy + dialogStyle.GetBorderTopSize() + dialogStyle.GetPaddingTop()

	lines := a.quitDialogLines()
	innerW := 0
	for _, line := range lines {
		if w := lipgloss.Width(line); w > innerW {
			innerW = w
		}
	}

	rowY := iy
	for _, line := range lines[:len(lines)-1] {
		rowY += lipgloss.Height(line)
	}

	btnRow := a.quitButtonsRow()
	btnH := lipgloss.Height(btnRow)
	btnW := lipgloss.Width(btnRow)
	btnX := ix + max(0, (innerW-btnW)/2)
	if y < rowY || y >= rowY+btnH || x < btnX || x >= btnX+btnW {
		return -1
	}

	quitW := lipgloss.Width(a.quitButton("Quit", a.quitFocus == quitBtnQuit, true))
	if x < btnX+quitW {
		return quitBtnQuit
	}
	cancelX := btnX + quitW + quitBtnGap
	if x >= cancelX && x < btnX+btnW {
		return quitBtnCancel
	}
	return -1
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
	stacked := a.helpShouldStack()
	body := a.helpBody(stacked)
	bodyLines := strings.Split(body, "\n")
	bodyH := a.helpBodyHeight()
	scrollable := len(bodyLines) > bodyH
	if scrollable {
		a.clampHelpScroll(len(bodyLines), bodyH)
		end := min(a.helpScroll+bodyH, len(bodyLines))
		body = strings.Join(bodyLines[a.helpScroll:end], "\n")
	} else {
		a.helpScroll = 0
	}

	footer := "esc / ctrl+h  close"
	if scrollable {
		footer = "↑↓ / wheel  scroll     esc / ctrl+h  close"
	}

	modal := dialogStyle.
		Align(lipgloss.Left).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			dialogTitleStyle.Render("Help"),
			"",
			body,
			"",
			dialogKeyStyle.Render(footer),
		))
	if stacked {
		// Keep the stacked page inside the window/panel width.
		maxW := max(20, a.width-2)
		if lipgloss.Width(modal) > maxW {
			modal = lipgloss.NewStyle().MaxWidth(maxW).Render(modal)
		}
	}
	return modal
}

func (a *AppModel) helpShouldStack() bool {
	twoCol := a.helpBody(false)
	frame := dialogStyle.GetHorizontalFrameSize()
	return lipgloss.Width(twoCol)+frame > a.width
}

func (a *AppModel) helpBody(stacked bool) string {
	if stacked {
		return renderHelpColumn(helpCategories)
	}
	left := renderHelpColumn(helpCategories[:3])
	right := renderHelpColumn(helpCategories[3:])
	return lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", 4), right)
}

// helpBodyHeight is how many content rows fit under the title/footer chrome
// inside the window.
func (a *AppModel) helpBodyHeight() int {
	chrome := dialogStyle.GetVerticalFrameSize() +
		lipgloss.Height(dialogTitleStyle.Render("Help")) +
		lipgloss.Height(dialogKeyStyle.Render("esc / ctrl+h  close")) +
		2 // blank lines around the body
	return max(3, a.height-chrome-2)
}

func (a *AppModel) helpMaxScroll() int {
	lines := lipgloss.Height(a.helpBody(a.helpShouldStack()))
	return max(0, lines-a.helpBodyHeight())
}

func (a *AppModel) clampHelpScroll(totalLines, viewH int) {
	a.helpScroll = max(0, min(a.helpScroll, max(0, totalLines-viewH)))
}

func (a *AppModel) scrollHelp(delta int) {
	a.helpScroll += delta
	a.clampHelpScroll(lipgloss.Height(a.helpBody(a.helpShouldStack())), a.helpBodyHeight())
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
	// rightInfo := fmt.Sprintf("Window size: %dx%d", a.width, a.height)
	padding := 1
	// gap := max(1, w-lipgloss.Width(leftInfo)-lipgloss.Width(rightInfo)-padding*2)
	// line := leftInfo + strings.Repeat(" ", gap) + rightInfo
	line := leftInfo
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
