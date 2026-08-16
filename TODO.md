# TODO

## P1 — next up

- [ ] **Saved connection profiles.** The form re-filling itself with the last values already helps, but retyping a fresh server each time is still the main friction. Persist a small list of `{name, addr, user, method, key_path}` profiles to `~/.config/lazyssh/profiles.{json,yaml}` (no secrets stored — passwords still prompt live, only the non-secret fields are remembered). Add a profile picker as a third screen, reachable before the form: arrow keys to pick, enter to load it into the form pre-filled, `n` for a blank new one.

- [ ] **Passphrase-protected keys.** `WithPrivateKeyFile` already supports a passphrase — the form just never asks for one. Add a 6th field, only shown
when method is "key", using the same masked `textinput.EchoPassword` as the password field.

- [ ] **`ssh-agent` as a method option.** `bubblessh.WithAgent()` exists and works today; the form's method toggle just doesn't offer it. Cycle password → key → agent instead of a binary toggle, hide the secret field entirely when agent is selected (nothing to type).

## P2 — worth doing, less urgent

- [ ] **Polish Status and Notification.** Use https://github.com/DaltonSW/BubbleUp to show status and notifications.

- [ ] **Host key policy as a real choice, not a silent default.** Right now every connection silently uses `WithAcceptNewHostKeys` against the default `~/.ssh/known_hosts`. Surface this as an explicit, visible setting — strict / accept-new / insecure — at least in a config file if not in the form itself, so the trade-off is a decision the user made, not one made for them.

- [ ] **`~/.ssh/config` awareness.** Parse `Host` blocks to pre-fill the form (and maybe the profile picker) from aliases the user already has — `ssh myserver` should be enough of a hint to offer the same shortcut here.

## P3 — someday, only if there's real demand

- [ ] **Session logging.** Write a session's `Content()` output to a file for later review. Needs a decision on rotation/redaction (a logged session can
easily contain a password typed at a remote prompt) before this is safe to ship casually.

- [ ] **Theming.** The lipgloss colors in `styles.go` are hardcoded. Pulling them into a small `Theme` struct with a couple of presets is easy; deciding it's worth the surface area to maintain is the actual open question.

- [ ] **Distribution polish.** `-version`/`-h` flags, a Homebrew tap, a Scoop manifest for Windows. Only worth it once there's something worth distributing beyond `go install`.
