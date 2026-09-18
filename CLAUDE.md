# NetworkProfile — notes for Claude

Windows 11 desktop utility that switches between saved network configurations
(IP, mask, gateway, DNS) across several Ethernet adapters and one Wi-Fi adapter.
Go + Wails v2. Single-user tool.

## Commands

```bash
go test ./...                  # unit and application tests, no side effects
cd frontend && npm test        # Vitest, pure helpers
go vet ./... && gofmt -l .     # CI fails on either
wails build                    # -> build/bin/NetworkProfile.exe
go run tools/icongen/main.go   # redraws build/windows/icon.ico
```

Integration tests touch real adapters and never run on their own:

```bash
go test -tags integration ./internal/network -run TestRealAdapters -v
```

## Architecture, and why

```
main.go            interface, elevated helper (--helper), or a command
tray.go            notification area icon and the pinned-profile menu
console_windows.go attaches stdout when a command is run from a terminal
internal/gui       the surface bound to the frontend, one method per action
internal/cli       --list and --profile, tested without touching a machine
internal/profile   model, validation, YAML store
internal/apply     turns a profile into adapter writes
internal/ipc       named-pipe protocol, client, server, helper spawning
internal/network   Manager interface, Windows implementation, Fake
frontend/src       vanilla JS, no framework
tools/icongen      draws build/windows/icon.ico and build/appicon.png
```

**Every exported method on `gui.App` is bound to the frontend**, so anything
that cannot cross that boundary — a callback, for instance — is a constructor
argument instead. That is why `New` takes the change hook rather than offering
a setter.

The notification area menu follows the stored profiles through that hook; it is
never polled. Its slots are created once and then renamed or hidden, because
Windows offers no way to remove a menu item once added.

**Anything two surfaces need lives in Go, never in the frontend.** The window is
one way in; the notification area menu and the command line are two others, and
neither passes through the window at all. Whether a profile is active
(`apply.Active`) and which was last applied (`App.LastApplied`) were each
written in JavaScript first, and each was wrong the moment a profile was applied
from the menu. If a fact is needed by more than the window, it belongs to the
application, not to its view.

**An inference must never remove the only way to act.** Whether a profile is
already in effect is deduced from adapter state, and that deduction can be
wrong. It may tick a menu entry or highlight a row; it must not replace or
disable the apply control. A Wi-Fi profile once could not be applied at all
because it was wrongly believed active and its button had been swapped for the
word "Actif" — nothing ran, so nothing was logged, and the failure looked like
it came from netsh.

**And a view must never block what it shows.** The refresh hook runs on its own
goroutine, and `tray` never calls into the menu while holding its state lock:
an earlier version held that lock across the whole menu construction, so saving
a profile waited on the notification area and silently never completed.

**Privileges are split, and this is load-bearing.** The target user is not a
local administrator, so elevation switches to a *different Windows account*.
Anything keyed on the current account breaks under elevation: `%AppData%` moves,
and WebView2 cannot create its cache. So the interface runs unelevated and only
writes cross into a helper over a named pipe. Do not "simplify" this by adding
`requireAdministrator` back to the manifest — that was tried, and it put the
profiles in the administrator's directory and broke WebView2 outright.

The helper is spawned on the **first write**, never at startup: opening the
window must never cost a credentials prompt. `ipc.Manager` caches the connection
so one prompt covers the whole session, and distinguishes a dead pipe (start a
new helper) from a refusal by netsh (keep the one we have).

**Never parse netsh output.** It is translated into the language of the running
Windows; a French install prints `DHCP activé : Non`. Adapter state comes from
the IP Helper API (`GetAdaptersAddresses`), known Wi-Fi networks from the WLAN
profile XML under `ProgramData\Microsoft\Wlansvc`, and netsh is used only to
*write*, where nothing but the exit code is read.

**Reads need no privileges.** Listing adapters and Wi-Fi profiles works fine
unelevated — this is measured, not assumed, and it is what makes the split
worthwhile. Do not route reads through the helper.

A profile holds a **list of targets**, not one interface, so a single profile can
put several adapters back on DHCP at once.

## Conventions

- **English for everything public**: code, comments, tests, commit messages,
  README, this file, CI step names, release notes, the repository description.
- **French only for what the operator reads inside the application**: interface
  labels, validation messages, command-line output. `profile.FieldError`
  messages are UI copy and are French on purpose.
- In doubt, ask who reads the text. A contributor or a visitor to the
  repository reads English; a person in front of the window reads French.
- Frontend is vanilla JS with no framework and no bundled dependencies beyond
  Vite and the embedded fonts. Fonts are vendored, never fetched: this tool is
  used to repair broken network connections.
- Structs crossing into the frontend carry `json` tags so the UI sees lowercase
  field names.

## Verifying the interface without elevation

`wails dev` needs an elevated terminal (the dev binary inherits the manifest).
To inspect the UI from an ordinary session, run the frontend alone and fake the
Go bridge:

```bash
cd frontend && npm run dev          # http://localhost:5173
```

Then in the page, define `window.go.gui.App` with stub methods and re-import the
module with a changed query string so it runs again:

```js
window.go = { gui: { App: { Interfaces: async () => ([...]), /* ... */ } } }
await import('/src/main.js?stub=1')
```

The module re-executes and renders against the stubs. This is how the screens
were checked without administrator rights.

## Deliberately not here

These were considered and closed. Do not offer to build them again unless the
maintainer asks.

- **An audit log of applied changes.** It was in the first architecture sketch
  and dropped: the application log already records every change and every netsh
  command, which covers what the log was for.
- **Integration tests that drive real adapters.** `internal/network` has a
  `TestRealAdapters` read-only check behind the `integration` tag, and that is
  as far as it goes. Exercising the write path for real needs a disposable
  Windows VM, which is not available. The consequence is worth stating plainly:
  the test suite proves the right netsh command is produced, never that it has
  the intended effect. Every bug found after the first release was found by
  using the tool, not by the tests.
- **Code signing.** The SignPath Foundation declined — their programme favours
  projects with an established community. Releases are unsigned and say so.
  SmartScreen only warns about downloaded files, so a locally built binary
  raises nothing.

## Traps already hit

- `winio.DialPipe`'s timeout only covers a pipe that exists but is busy. A pipe
  that does not exist yet fails immediately, so waiting for the helper to appear
  is done in `dialWhenPublished`, not by the library.
- A failed action must not replace the whole interface with its message. It gets
  a dismissible banner; the full-screen treatment is only for being unable to
  read anything at startup.
- Vite wipes `frontend/dist` on build, and `main.go` embeds that directory — a
  placeholder is restored by a plugin in `vite.config.js` so a fresh clone can
  still compile before any frontend build.
- Wails warns that `network.Interface` is a reserved word. The bindings and
  `models.ts` are generated correctly regardless; the warning is cosmetic.
- The console keeps its default code page, so UTF-8 output arrives mangled
  unless `SetConsoleOutputCP(CP_UTF8)` is called — profile names carry em
  dashes and accents, and came out as `Ethernet ÔÇö Atelier`.
- A GUI-subsystem binary is never waited on by the shell. `--list` prints
  underneath a prompt that has already returned, which looks like a hang and is
  not. Say so wherever the command line is documented.
