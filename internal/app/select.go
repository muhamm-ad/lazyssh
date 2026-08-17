package app

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/x/ansi"
)

func runeIndexAt(s string, cells int) int {
	if cells <= 0 {
		return 0
	}
	i, w := 0, 0
	for _, r := range s {
		cw := lipgloss.Width(string(r))
		if w+cw > cells {
			return i
		}
		w += cw
		i++
		if w >= cells {
			return i
		}
	}
	return i
}

func highlightRange(s string, start, end int) string {
	runes := []rune(s)
	lo, hi := start, end
	if lo > hi {
		lo, hi = hi, lo
	}
	lo = max(0, min(lo, len(runes)))
	hi = max(0, min(hi, len(runes)))
	if lo == hi {
		return s
	}
	return string(runes[:lo]) + selectedTextStyle.Render(string(runes[lo:hi])) + string(runes[hi:])
}

func (a *AppModel) fieldInnerOrigin() (x, y int) {
	t := a.curentTab()
	form := a.viewForm(t)
	formW, formH := lipgloss.Width(form), lipgloss.Height(form)
	innerW := a.panelInnerWidth()
	innerH := a.panelInnerHeight()
	formX := panelStyle.GetBorderLeftSize() + panelStyle.GetPaddingLeft() + max(0, (innerW-formW)/2)
	formY := tabBarHeight + panelStyle.GetBorderTopSize() + panelStyle.GetPaddingTop() + max(0, (innerH-formH)/2)
	x = formX + labelColWidth + labelGap + fieldStyle.GetBorderLeftSize() + fieldStyle.GetPaddingLeft()
	y = formY
	return x, y
}

func (a *AppModel) termOrigin() (x, y int) {
	x = panelStyle.GetBorderLeftSize() + panelStyle.GetPaddingLeft()
	y = tabBarHeight + panelStyle.GetBorderTopSize() + panelStyle.GetPaddingTop()
	return x, y
}

func (a *AppModel) screenToTermCell(x, y int) (cx, cy int, ok bool) {
	ox, oy := a.termOrigin()
	cols, rows := a.termSize()
	cx, cy = x-ox, y-oy
	if cx < 0 || cy < 0 || cx >= cols || cy >= rows {
		return 0, 0, false
	}
	return cx, cy, true
}

func (t *Tab) startTermSelect(x, y int) {
	t.termSel = termSelection{on: true, ax: x, ay: y, bx: x, by: y}
}

func (t *Tab) updateTermSelect(x, y int) {
	if !t.termSel.on {
		t.startTermSelect(x, y)
		return
	}
	t.termSel.bx, t.termSel.by = x, y
}

func (t *Tab) finishTermSelect() {
	if t.termSel.ax == t.termSel.bx && t.termSel.ay == t.termSel.by {
		t.termSel.on = false
	}
}

func (t *Tab) clearTermSelect() {
	t.termSel = termSelection{}
}

func (t *Tab) hasTermSelection() bool {
	return t.termSel.on && (t.termSel.ax != t.termSel.bx || t.termSel.ay != t.termSel.by)
}

func orderedCells(ax, ay, bx, by int) (x0, y0, x1, y1 int) {
	if ay > by || (ay == by && ax > bx) {
		return bx, by, ax, ay
	}
	return ax, ay, bx, by
}

func highlightTermSelection(s string, sel termSelection, cols int) string {
	if !sel.on {
		return s
	}
	x0, y0, x1, y1 := orderedCells(sel.ax, sel.ay, sel.bx, sel.by)
	lines := strings.Split(s, "\n")
	for y, line := range lines {
		if y < y0 || y > y1 {
			continue
		}
		width := max(cols, lipgloss.Width(line))
		if lipgloss.Width(line) < width {
			line += strings.Repeat(" ", width-lipgloss.Width(line))
		}
		start, end := 0, width
		if y == y0 {
			start = x0
		}
		if y == y1 {
			end = x1 + 1
		}
		start = max(0, min(start, width))
		end = max(start, min(end, width))
		lines[y] = ansi.Cut(line, 0, start) +
			selectedTextStyle.Render(ansi.Cut(line, start, end)) +
			ansi.Cut(line, end, width)
	}
	return strings.Join(lines, "\n")
}

func termSelectionPlain(s string, sel termSelection, cols int) string {
	if !sel.on {
		return ""
	}
	x0, y0, x1, y1 := orderedCells(sel.ax, sel.ay, sel.bx, sel.by)
	lines := strings.Split(s, "\n")
	var out []string
	for y, line := range lines {
		if y < y0 || y > y1 {
			continue
		}
		plain := ansi.Strip(line)
		if lipgloss.Width(plain) < cols {
			plain += strings.Repeat(" ", cols-lipgloss.Width(plain))
		}
		start, end := 0, lipgloss.Width(plain)
		if y == y0 {
			start = x0
		}
		if y == y1 {
			end = x1 + 1
		}
		start = max(0, min(start, lipgloss.Width(plain)))
		end = max(start, min(end, lipgloss.Width(plain)))
		out = append(out, strings.TrimRight(ansi.Cut(plain, start, end), " "))
	}
	return strings.Join(out, "\n")
}

func (a *AppModel) copyTermSelection() bool {
	t := a.curentTab()
	if !t.inSession() || !t.hasTermSelection() || t.SSH == nil {
		return false
	}
	cols, rows := a.termSize()
	text := termSelectionPlain(padBlock(t.SSH.Content(), cols, rows), t.termSel, cols)
	if text == "" {
		return false
	}
	_ = clipboard.WriteAll(text)
	return true
}

func (a *AppModel) beginFieldSelect(in *textinput.Model, screenX int) {
	t := a.curentTab()
	innerX, _ := a.fieldInnerOrigin()
	idx := runeIndexAt(echoedValue(*in), screenX-innerX)
	in.SetCursor(idx)
	t.selStart, t.selEnd = idx, idx
	t.TextSelected = false
	a.fieldDragging = true
}

func (a *AppModel) dragFieldSelect(screenX int) {
	t := a.curentTab()
	in := t.focusedInput()
	if in == nil {
		return
	}
	innerX, _ := a.fieldInnerOrigin()
	idx := runeIndexAt(echoedValue(*in), screenX-innerX)
	in.SetCursor(idx)
	t.selEnd = idx
	t.TextSelected = t.selStart != t.selEnd
}

func (a *AppModel) finishFieldSelect() {
	t := a.curentTab()
	a.fieldDragging = false
	if t.selStart == t.selEnd {
		t.TextSelected = false
	}
}
