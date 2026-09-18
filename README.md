# NetworkProfile

Switch between saved network configurations on Windows 11.

Save one configuration per site, per machine or per piece of equipment, then
move between them in a click — without reopening the Windows network panels,
and without retyping an address from memory.

The tool comes from a concrete need: configuring industrial equipment that each
lives on its own address range (`192.168.0.x` for a PLC, `10.10.128.x` on a
customer site, `172.16.x.x` on a job site), several times a day, on a machine
you are not an administrator of.

The interface is in French.

## What it does

- **Profiles covering several adapters.** A profile holds a list of target
  adapters, not a single one. The "Défaut" profile therefore puts Ethernet
  *and* Wi-Fi back on DHCP in one go.
- **Fixed address or DHCP**, with address, mask, gateway and several DNS
  servers.
- **Wi-Fi**: switch to a network Windows already knows, or change the address
  while staying on the current one. Keys are never handled: Windows holds them.
- **Quick change**: apply a one-off address without creating a profile.
- **Virtual adapters hidden** by default — Hyper-V, WSL and VMware do not
  clutter the picker, but can be shown.
- **The active profile is marked**: the row whose configuration matches what the
  adapters actually report is highlighted.
- **Stays in the notification area**, so credentials are asked for once.
- **Pinned profiles** appear directly in the icon's menu, so changing site takes
  two clicks without opening the window.

## The privilege model

This is the part of the project that is not obvious, and it is worth
understanding before touching the code.

Reconfiguring a network adapter requires administrator rights. But on a company
machine the operator is **not** an administrator: elevating a program there
means switching to a **different account**. A fully elevated application
therefore loses its own user profile — `%AppData%` then points at the
administrator account, and the network profiles disappear between launches.

So the application is cut in two:

```
NetworkProfile.exe              your account, no elevation
  ├─ profiles       %AppData%\NetworkProfile\profiles.yaml
  ├─ adapter state  Windows API, directly — no privilege needed
  └─ writes ────────┐
                    │  named pipe, random name, restricted to your account
NetworkProfile.exe --helper     elevated, started at the first change
  └─ netsh
```

What follows from it:

- Opening the application and looking at the network state asks for **nothing**.
- Administrator credentials are asked for **at the first change**, then not
  again for as long as the application stays open — hence the notification area
  icon, which keeps it alive after the window is closed.
- The elevated helper watches the window that started it and **exits with it**:
  no privileged process is left behind.
- Its pipe carries a random name and a protected access control list naming only
  your account, SYSTEM and the administrators: no other user of the machine can
  talk to it.

## From the command line

So that a site can have its own desktop shortcut, without going through the
window:

```bat
NetworkProfile.exe --profile "Ethernet — Atelier"
NetworkProfile.exe --list
```

The name accepts either the profile identifier or its displayed name, ignoring
case; an ambiguous name is refused rather than resolved arbitrarily. The exit
code is 0 on success, so it can be chained in a script.

Two consequences of the executable being a GUI application. First, the shell
**does not wait for it**: the prompt comes back before the output does, so the
listing appears underneath a prompt that is already accepting input. It looks
like a hang and is not — press Enter for a clean prompt, or redirect the output
(`NetworkProfile.exe --list > profiles.txt`) and read the file. Second, each run
is a separate process, so applying a profile from a shortcut asks for
administrator rights every time, where the open window asks only once.

## Profile format

`%AppData%\NetworkProfile\profiles.yaml`, readable and editable by hand:

```yaml
version: 1
profiles:
  - id: ethernet-automate-s7
    name: Ethernet — Automate S7-1500
    targets:
      - interface: Ethernet
        mode: static
        address: 192.168.0.241
        mask: 255.255.255.0

  - id: wifi-site-client-b
    name: Wi-Fi — Site client B
    pinned: true
    targets:
      - interface: Wi-Fi
        ssid: SiteB-Corp        # a network Windows already knows
        mode: static
        address: 10.10.128.20
        mask: 255.255.255.0
        gateway: 10.10.128.1
        dns: [10.10.128.1, 9.9.9.9]

  - id: defaut-dhcp
    name: Défaut — DHCP
    targets:
      - interface: Ethernet
        mode: dhcp
      - interface: Wi-Fi
        mode: dhcp
```

A gateway outside the subnet of its address is refused: Windows accepts it and
then routes unpredictably, which costs an hour of diagnosis.

## Building

Requires [Go](https://go.dev/dl/), [Node.js](https://nodejs.org/) and the
[Wails CLI](https://wails.io/docs/gettingstarted/installation).

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails build
```

The binary lands in `build/bin/NetworkProfile.exe`.

To iterate on the interface, `wails dev` must run in an **administrator
terminal**: the development binary inherits the manifest and otherwise refuses
to start.

The icon is drawn programmatically — to change it, edit the coordinates in
`tools/icongen/main.go` and run:

```bash
go run tools/icongen/main.go
```

## Tests

```bash
go test ./...
cd frontend && npm test
```

The Go tests drive a fake network layer: none of them alters the machine's
configuration. The ones that touch real adapters sit behind a build tag and
never run on their own:

```bash
go test -tags integration ./internal/network -run TestRealAdapters -v
```

## Released binaries are not signed

There is no code signing certificate behind this project, so a binary
downloaded from the releases page carries no publisher identity. Windows
attaches a mark to anything downloaded from the internet, and SmartScreen will
warn about it: *Windows protected your PC*, publisher unknown.

What you can do about it:

- **Check the file** against `checksums.txt` published alongside it:
  `Get-FileHash NetworkProfile.exe -Algorithm SHA256`.
- **Or build it yourself** from this repository — see *Building* above. A binary
  you compiled locally carries no such mark and raises no warning at all.

To run the downloaded file anyway: *More info* → *Run anyway*, or clear the mark
with `Unblock-File .\NetworkProfile.exe`.

A free certificate was applied for through the
[SignPath Foundation](https://signpath.org) and declined: their programme
favours projects with an established community, which this one does not have.

## Privacy

This program will not transfer any information to other networked systems
unless specifically requested by the user or the person installing or operating
it. No telemetry, no update check, no third-party service. Full statement:
[PRIVACY.md](PRIVACY.md).

## Licence

[MIT](LICENSE).
