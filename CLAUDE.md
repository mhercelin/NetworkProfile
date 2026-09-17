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
main.go            interface, or the elevated helper when given --helper
internal/gui       the surface bound to the frontend, one method per action
internal/profile   model, validation, YAML store
internal/apply     turns a profile into adapter writes
internal/ipc       named-pipe protocol, client, server, helper spawning
internal/network   Manager interface, Windows implementation, Fake
frontend/src       vanilla JS, no framework
```

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

- Code, comments, tests, commit messages: **English**.
- Everything the user reads — interface text and validation messages: **French**.
  `profile.FieldError` messages are UI copy and are French on purpose.
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
