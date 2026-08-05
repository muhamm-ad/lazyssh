# lazyssh

A small, full-screen SSH client: a connection form on top, the live terminal below. Fill in a server, hit connect, and you're in — no config file, no flags to remember.

Built on [`bubble-ssh`](https://github.com/muhamm-ad/bubble-ssh), which does the actual SSH-session-as-a-Bubble-Tea-component work; `lazyssh` is the form and screen wrapped around it.

## Install

```bash
go install github.com/muhamm-ad/lazyssh@latest
```

or clone and run directly:

```bash
git clone https://github.com/muhamm-ad/lazyssh.git
cd lazyssh
go run .
```

## Usage

```bash
lazyssh
```

That's it — no flags. Everything happens in the form:

| Key | Does |
| --- | --- |
| `Tab` / `→` | next field |
| `Shift+Tab` / `←` | previous field |
| `Space` | toggle auth method, on the method field |
| `Enter` | connect, from anywhere in the form |
| `Ctrl+B` | disconnect, back to the form (still filled in) |
| `Ctrl+C` | quit |

Fill in address (`host:port`), user, pick password or key, type the secret, hit Connect (or just Enter). Once connected, every key you press goes straight to the remote shell — colors, `vim`, `htop`, all of it — exactly like a normal `ssh` session.

Host keys are trusted on first connection and remembered in your normal `~/.ssh/known_hosts` from then on (same as `ssh -o StrictHostKeyChecking=accept-new`) — see [bubble-ssh's docs](https://github.com/muhamm-ad/bubble-ssh) if you want to know exactly what that trade-off means.

## What this is (and isn't) today

v0 is intentionally small: one connection at a time, nothing saved between runs, password or key file only. That's not the ceiling — see [TODO.md](./TODO.md) for what's planned, roughly in priority order, with the reasoning behind each one. Saved connection profiles are the top of that list.

## License

MIT, see [LICENSE](./LICENSE).
