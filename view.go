// Rendering: both frames, always both on screen, stacked exactly as in
// drafts/ssh-tui.html. Sizes come from layout.go, colors from styles.go — this
// file only assembles them.

package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

const (
	formHint = "tab / ← → move · space toggles method · enter connects · ctrl+c quits"
	termHint = "every key goes to the session, ctrl+c included · ctrl+b closes it"
)

func (a appModel) View() tea.View {
	v := tea.NewView(a.render())
	v.AltScreen = true
	v.BackgroundColor = termBg
	return v
}

func (a appModel) render() string {
	w := a.pageWidth()
	// Box widths follow the field contents, which change on every keystroke —
	// re-sync here (on the render copy) so the inputs never disagree with the
	// boxes they're drawn in.
	a.syncInputWidths()

	form := a.viewFormFrame(w)
	term := a.viewTermFrame(w)
	gap := lineStyle.Width(w).Render(" ")

	page := lipgloss.JoinVertical(lipgloss.Left, form, gap, term)
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

func (a appModel) viewQuitDialog() string {
	lines := []string{dialogTitleStyle.Render("Quit lazyssh?")}
	if a.hasSSH && a.ssh.Connected() {
		lines = append(lines, dialogBodyStyle.Render("The SSH session will be closed."))
	}
	lines = append(lines, "", dialogKeyStyle.Render("y / enter  quit     n / esc  cancel"))

	return dialogStyle.Render(lipgloss.JoinVertical(lipgloss.Center, lines...))
}

func (a appModel) viewFormFrame(w int) string {
	header := frameHeader("Navigation", formHint, w)

	boxes := a.fieldWidths()

	// Width on the style is the total box width (border and padding included);
	// clipLine keeps whatever goes inside on a single line, so an overlong
	// value can never wrap and push the row to two lines.
	box := func(i int, blurred, focused lipgloss.Style, content string) string {
		style := blurred
		// While the session owns the keyboard, no field is really focused.
		if a.focus == i && !a.connected {
			style = focused
		}
		return style.Width(boxes[i]).Render(clipLine(content, max(1, boxes[i]-fieldChrome)))
	}

	centered := func(s lipgloss.Style) lipgloss.Style { return s.Align(lipgloss.Center) }

	row := lipgloss.JoinHorizontal(lipgloss.Top,
		box(fieldHost, fieldStyle, focusedFieldStyle, a.host.View()), " ",
		box(fieldUser, fieldStyle, focusedFieldStyle, a.user.View()), " ",
		box(fieldMethod, centered(fieldStyle), centered(focusedFieldStyle), a.methodLabel()), " ",
		box(fieldSecret, fieldStyle, focusedFieldStyle, a.secret.View()), " ",
		box(fieldConnect, centered(connectStyle), centered(focusedConnectStyle), "Connect"),
	)

	body := row
	if a.status != "" {
		body = lipgloss.JoinVertical(lipgloss.Left, row, "", a.viewStatusBar())
	}

	// return lipgloss.JoinVertical(lipgloss.Left, header, panelStyle.Width(w).Render(body))
	return lipgloss.JoinVertical(lipgloss.Left, header, body)
}

func (a appModel) viewStatusBar() string {
	style, status := statusStyle, a.status

	switch {
	case a.connected && a.ssh.Connected():
		status = fmt.Sprintf("connected — %s@%s", a.user.Value(), a.host.Value())
		style = okStatusStyle
	case a.lastErr != nil && !a.connected:
		style = errStatusStyle
	}
	return style.Width(a.formInnerWidth()).Render(status)
}

func (a appModel) viewTermFrame(w int) string {
	header := frameHeader("Terminal", termHint, w)

	cols, rows := a.termSize()
	content := a.termContent()

	// Pad / clip the content block to the allocated size so the panel keeps a
	// stable footprint whether connected or not.
	content = termContentStyle.
		Width(cols).
		Height(rows).
		Render(padBlock(content, cols, rows))

	return lipgloss.JoinVertical(lipgloss.Left, header, panelStyle.Width(w).Render(content))
}

func (a appModel) termContent() string {
	if a.hasSSH {
		return a.ssh.Content()
	}
	return ""
}

// frameHeader renders a frame's label and key hint on exactly one line, spanning
// the page. The hint is the first thing to go on a narrow terminal — letting it
// wrap would silently steal a row from the layout below.
func frameHeader(label, hint string, w int) string {
	line := lipgloss.JoinHorizontal(lipgloss.Bottom,
		labelStyle.Render(label),
		"  ",
		hintStyle.Render(hint),
	)
	return lineStyle.Width(w).Render(clipLine(line, w))
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
