# lazyssh README demo — recorded with VHS (https://github.com/charmbracelet/vhs)
#
#   vhs lazyssh.tape
#
# Width/Height are pixels, not cells. The TUI needs at least 90×32 cells
# (see minAppWidth / minAppHeight). Wait+Screen is required because the
# last line is always the status bar, never the form or the SSH prompt.
#
# Needs network: the tape connects to OverTheWire Bandit (the form's
# default host) so the GIF shows a live session, not just the chrome.

Output docs/lazyssh.gif

Require go

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 780
Set Padding 12
Set TypingSpeed 30ms
Set PlaybackSpeed 1

Set WindowBar Colorful
Set WindowBarSize 12
Set BorderRadius 8
Set Framerate 24
Set CursorBlink false
Set WaitTimeout 60s

Hide
Type "go build -o lazyssh ."
Enter
Wait
Type "./lazyssh"
Enter
Wait+Screen /address/
Show

Sleep 1200ms

# Walk the connection form down to Connect.
Tab@500ms 5
Sleep 800ms

# Connect to the pre-filled Bandit host.
Enter
Wait+Screen /bandit0@bandit/
Sleep 1s

Type "ls -la"
Sleep 300ms
Enter
Sleep 1500ms

# Open a second tab while the session keeps running in the background.
Ctrl+T
Sleep 1s
Type "bandit.labs.overthewire.org"
Sleep 500ms
Tab
Type "20"
Sleep 500ms
Tab
Type "bandit1"
Sleep 500ms
Tab
Space
Sleep 500ms
Tab
Type "6y2kwnwK6grgvwvpvLaa2T1cpFEKOhNR"
Enter
Sleep 1s
Wait+Screen /bandit1@bandit/
Sleep 1s
Type "ls -la"
Enter
Sleep 2s

# Back to the live session.
Ctrl+Left
Sleep 2s
Ctrl+l
Sleep 1s
Ctrl+T
Sleep 2s
Ctrl+Left
Sleep 1s
Ctrl+Left
Sleep 1s

# Key reference, then linger on the session so the loop lands there.
Ctrl+Q
Sleep 2s
Enter
