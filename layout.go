// Geometry: how the window is divided between the two frames, and how the five
// form boxes share a row. Every number here is in terminal cells.
//
// The rule for the field row is "content first, then fill": each box asks for
// the width its text needs, and whatever is left over (or missing) is spread
// across the boxes so the row always spans the panel exactly.

package main

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	lipgloss "charm.land/lipgloss/v2"
)

const (
	fieldChrome = 4 // rounded border (2) + horizontal padding (2)
	fieldGap    = 1 // one blank column between boxes

	// What panelStyle eats out of the space it's given: a border on each side,
	// plus Padding(1, 2).
	panelChromeW = 2 + 2*2
	panelChromeH = 2 + 2*1
)

// growWeight is how eagerly each box claims leftover row space. The three text
// inputs share it, weighted like the design's grid (1.35 / .85 / 1.25); the
// method and Connect chips hold whatever their label needs and no more.
var growWeight = [fieldCount]int{
	fieldHost:   135,
	fieldUser:   85,
	fieldMethod: 0,
	fieldSecret: 125,
}

// minContent is the narrowest each box may get before it stops shrinking, so a
// tiny window degrades into scrollable inputs rather than unreadable slivers.
var minContent = [fieldCount]int{
	fieldHost:    6,
	fieldUser:    4,
	fieldMethod:  8,
	fieldSecret:  6,
	fieldConnect: 7,
}

// pageWidth is the window width the layout is drawn against, floored so a very
// narrow terminal still produces a sane frame.
func (a appModel) pageWidth() int {
	return max(40, a.width)
}

// formInnerWidth is the usable width inside the form panel. The panel spans the
// whole window, so this has to agree with the width handed to panelStyle —
// disagree by even one column and the last field wraps onto its own line.
func (a appModel) formInnerWidth() int {
	return max(20, a.pageWidth()-panelChromeW)
}

// formFrameHeight is the vertical budget for the screenForm block: its label
// line, plus a panel holding the field row and the status bar. Everything left
// over goes to screenTerminal.
func (a appModel) formFrameHeight() int {
	const h = 1 + panelChromeH + 3 + 1 + 1 // label, chrome, field row, blank, status
	if a.height < h+6 {
		return max(4, a.height/3)
	}
	return h
}

// termSize is the inner size of the screenTerminal content area (inside the
// panel border and padding, below its label line).
func (a appModel) termSize() (cols, rows int) {
	cols = max(10, a.pageWidth()-panelChromeW)
	// What's left after the form frame, the blank row between frames, and this
	// frame's own label line and panel chrome.
	rows = max(3, a.height-a.formFrameHeight()-1-1-panelChromeH)
	return cols, rows
}

// contentWidths is what each box needs to show its text in full — the natural
// size it would take if the row weren't stretched to the panel.
func (a appModel) contentWidths() [fieldCount]int {
	// +1 leaves room for the caret sitting past the last character.
	input := func(m textinput.Model) int {
		shown := m.Value()
		if m.EchoMode == textinput.EchoPassword {
			shown = strings.Repeat("•", len([]rune(m.Value())))
		}
		return max(lipgloss.Width(shown), lipgloss.Width(m.Placeholder)) + 1
	}

	var w [fieldCount]int
	w[fieldHost] = input(a.host)
	w[fieldUser] = input(a.user)
	w[fieldMethod] = lipgloss.Width(a.methodLabel())
	w[fieldSecret] = input(a.secret)
	w[fieldConnect] = lipgloss.Width("Connect")
	return w
}

// fieldWidths returns each box's total width (content + border + padding).
// Boxes are sized to their content first, then the row is stretched — or
// squeezed — so it always spans exactly the panel's inner width.
func (a appModel) fieldWidths() [fieldCount]int {
	innerW := a.formInnerWidth()
	content := a.contentWidths()

	var boxes [fieldCount]int
	total := fieldGap * (fieldCount - 1)
	for i := range boxes {
		boxes[i] = max(content[i], minContent[i]) + fieldChrome
		total += boxes[i]
	}

	if slack := innerW - total; slack > 0 {
		distribute(&boxes, slack)
	} else if slack < 0 {
		reclaim(&boxes, -slack)
	}
	return boxes
}

// distribute hands extra columns to the text inputs, proportionally to
// growWeight. Integer division leaves a remainder; the host field takes it,
// being the widest column in the design.
func distribute(boxes *[fieldCount]int, extra int) {
	var totalWeight int
	for _, wt := range growWeight {
		totalWeight += wt
	}
	if totalWeight == 0 {
		return
	}

	given := 0
	for i, wt := range growWeight {
		if wt == 0 {
			continue
		}
		share := extra * wt / totalWeight
		boxes[i] += share
		given += share
	}
	boxes[fieldHost] += extra - given
}

// reclaim takes columns back when the window is too narrow for every box at
// its natural size: the growable inputs give first, down to their floor, then
// the fixed chips start truncating, and past that every box shrinks to the
// bare border. Overflowing would tear the panel's border, so something has to give.
func reclaim(boxes *[fieldCount]int, deficit int) {
	passes := []struct {
		fields []int
		floor  func(i int) int
	}{
		{[]int{fieldHost, fieldSecret, fieldUser}, func(i int) int { return minContent[i] + fieldChrome }},
		{[]int{fieldMethod, fieldConnect}, func(i int) int { return minContent[i] + fieldChrome }},
		// fieldChrome is the hard floor: a box narrower than its own border and
		// padding can't be drawn, and lipgloss would silently render it wider,
		// overflowing the row again.
		{[]int{fieldHost, fieldSecret, fieldUser, fieldMethod, fieldConnect}, func(int) int { return fieldChrome }},
	}

	for _, pass := range passes {
		for deficit > 0 {
			trimmed := 0
			for _, i := range pass.fields {
				if deficit == 0 || boxes[i] <= pass.floor(i) {
					continue
				}
				boxes[i]--
				deficit--
				trimmed++
			}
			if trimmed == 0 {
				break
			}
		}
	}
}

// syncInputWidths tells each textinput how many columns it actually got, so a
// value longer than its box scrolls inside it instead of wrapping.
func (a *appModel) syncInputWidths() {
	boxes := a.fieldWidths()
	a.host.SetWidth(max(4, boxes[fieldHost]-fieldChrome))
	a.user.SetWidth(max(4, boxes[fieldUser]-fieldChrome))
	a.secret.SetWidth(max(4, boxes[fieldSecret]-fieldChrome))
}
