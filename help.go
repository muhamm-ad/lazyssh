package main

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
			{"? / ctrl+h", "show / hide this help"},
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
		},
	},
}
