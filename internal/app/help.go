package app

// Help entries shown in the Help modal, grouped by category. Keys match what
// handleChromeKey / updateForm / updateTerminal actually listen for.

type helpEntry struct {
	keys string
	desc string
}

type helpCategory struct {
	title   string
	entries []helpEntry
}

var helpCategories = []helpCategory{
	{
		title: "General",
		entries: []helpEntry{
			{"ctrl+h", "show / hide this help"},
			{"ctrl+q", "quit (asks for confirmation)"},
		},
	},
	{
		title: "Tabs",
		entries: []helpEntry{
			{"ctrl+t", "open a new tab"},
			{"ctrl+w", "close the active tab"},
			{"ctrl+← / ctrl+→", "switch tabs (includes +)"},
			{"enter (on +)", "open a new tab"},
		},
	},
	{
		title: "Form",
		entries: []helpEntry{
			{"tab / shift+tab", "move between fields"},
			{"space", "toggle auth method (on method)"},
			{"enter", "connect"},
			{"ctrl+a", "select all in the focused field"},
			{"ctrl+c", "copy (not password)"},
			{"ctrl+x", "cut (not password)"},
			{"ctrl+v", "paste"},
			{"ctrl+z", "undo"},
		},
	},
	{
		title: "Session",
		entries: []helpEntry{
			{"exit", "leave the remote shell and return to the form"},
			{"ctrl+c", "copy selection"},
		},
	},
	{
		title: "Quit",
		entries: []helpEntry{
			{"tab / enter", "switch / select quit buttons"},
			{"esc", "dismiss an alert or dialog"},
		},
	},
	{
		title: "Mouse",
		entries: []helpEntry{
			{"↑↓ / wheel", "scroll help (narrow)"},
			{"click (tab)", "select / close / add a tab"},
			{"wheel (tab bar)", "scroll tabs horizontally"},
			{"click (form)", "focus a field or connect"},
			{"drag (field)", "select text in a field"},
			{"wheel (terminal)", "scroll terminal history"},
			{"drag (terminal)", "select terminal text"},
			{"click (outside)", "close help or quit dialog"},
			{"click (quit btn)", "quit or cancel"},
		},
	},
}
