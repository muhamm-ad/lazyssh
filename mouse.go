package main

import (
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

const tabBarScrollStep = 8

func (a *AppModel) handleMouse(msg tea.MouseMsg) tea.Cmd {
	if a.confirmQuit || a.showHelp {
		return nil
	}

	m := msg.Mouse()
	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		if m.Y < tabBarHeight {
			a.scrollTabBar(msg)
			return nil
		}
		if a.inTerminalPanel(m.Y) {
			return a.updateTerminal(msg)
		}
		return nil
	case tea.MouseClickMsg:
		if m.Button != tea.MouseLeft {
			return nil
		}
		if m.Y < tabBarHeight {
			a.curentTab().clearTermSelect()
			return a.clickTabBar(m.X)
		}
		if a.curentTab().inSession() {
			return a.clickTerminal(m.X, m.Y)
		}
		return a.clickForm(m.X, m.Y)
	case tea.MouseMotionMsg:
		if a.fieldDragging {
			a.dragFieldSelect(m.X)
			return nil
		}
		if a.termDragging {
			if cx, cy, ok := a.screenToTermCell(m.X, m.Y); ok {
				a.curentTab().updateTermSelect(cx, cy)
			}
			return nil
		}
		return nil
	case tea.MouseReleaseMsg:
		if a.fieldDragging {
			a.finishFieldSelect()
		}
		if a.termDragging {
			a.termDragging = false
			a.curentTab().finishTermSelect()
		}
		return nil
	default:
		return nil
	}
}

func (a *AppModel) inTerminalPanel(y int) bool {
	return y >= tabBarHeight && y < a.height-helpBarHeight
}

func (a *AppModel) scrollTabBar(msg tea.MouseWheelMsg) {
	switch msg.Button {
	case tea.MouseWheelUp, tea.MouseWheelLeft:
		a.tabBarScroll -= tabBarScrollStep
	case tea.MouseWheelDown, tea.MouseWheelRight:
		a.tabBarScroll += tabBarScrollStep
	}
	a.clampTabBarScroll()
}

func (a *AppModel) clickTabBar(x int) tea.Cmd {
	strip := a.measureTabStrip()
	if x >= a.width-strip.addW {
		return a.addTab()
	}

	tabsW := max(0, a.width-strip.addW)
	offset := max(0, min(a.tabBarScroll, max(0, strip.contentW-tabsW)))
	if x < 0 || x >= tabsW {
		return nil
	}
	cx := x + offset
	closeW := tabStyle.GetBorderRightSize() + tabStyle.GetPaddingRight() + 1
	for i := range a.tabs {
		start := strip.starts[i]
		end := start + strip.widths[i]
		if cx < start || cx >= end {
			continue
		}
		if cx >= end-closeW {
			return a.closeTab(a.tabs[i].ID)
		}
		return a.selectTabIndex(i)
	}
	return nil
}

func (a *AppModel) selectTabIndex(idx int) tea.Cmd {
	if idx < 0 || idx >= len(a.tabs) {
		return nil
	}
	a.focusAdd = false
	a.active = a.tabs[idx].ID
	a.ensureActiveTabVisible()
	if a.curentTab().inSession() {
		return nil
	}
	return a.focusCurrent()
}

func (a *AppModel) clickTerminal(x, y int) tea.Cmd {
	t := a.curentTab()
	cx, cy, ok := a.screenToTermCell(x, y)
	if !ok {
		t.clearTermSelect()
		return nil
	}
	t.startTermSelect(cx, cy)
	a.termDragging = true
	a.fieldDragging = false
	return nil
}

func (a *AppModel) clickForm(x, y int) tea.Cmd {
	t := a.curentTab()
	form := a.viewForm(t)
	formW, formH := lipgloss.Width(form), lipgloss.Height(form)
	innerW := a.panelInnerWidth()
	innerH := a.panelInnerHeight()
	formX := panelStyle.GetBorderLeftSize() + panelStyle.GetPaddingLeft() + max(0, (innerW-formW)/2)
	formY := tabBarHeight + panelStyle.GetBorderTopSize() + panelStyle.GetPaddingTop() + max(0, (innerH-formH)/2)
	if x < formX || x >= formX+formW || y < formY || y >= formY+formH {
		t.clearSelection()
		return nil
	}

	fields := []int{fieldHost, fieldPort, fieldUser, fieldMethod, fieldSecret, -1, fieldConnect}
	rowY := formY
	for i, row := range a.formRows(t) {
		h := lipgloss.Height(row)
		if y >= rowY && y < rowY+h {
			f := fields[i]
			if f < 0 {
				return nil
			}
			a.blurAll()
			t.clearSelection()
			t.Focus = f
			if f == fieldConnect {
				return a.connect()
			}
			cmd := a.focusCurrent()
			if in := t.focusedInput(); in != nil {
				a.beginFieldSelect(in, x)
			}
			return cmd
		}
		rowY += h
	}
	return nil
}
