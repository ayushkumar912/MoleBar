# MoleBar

MoleBar is a tiny macOS menu bar app for [Mole](https://github.com/tw93/Mole).
It shows your current CPU, memory, swap, disk, battery, and system health in one
small place, without adding a background daemon or Electron runtime.

```
CPU 12% MEM 67%   ← menu bar title, always visible
├── CPU: 12.2%  (load1 2.94)
├── Memory: 67.0%
├── Swap: 24.8%
├── Disk: 39.5%
├── Battery: 80% (AC)
├── ──────────────
├── Health: 100 (Excellent)
├── Updated: 14:32:07
├── ──────────────
├── Refresh now
└── Quit
```

## What it does

MoleBar runs a small timer and calls:

```sh
mo status --json
```

Then it reads the JSON output and updates the menu bar and dropdown menu with the
important values. It is intentionally lightweight: the app does not embed Mole as
an imported library, it just wraps the CLI output in a native macOS menu bar UI.

## Requirements

- macOS 11+
- [Mole](https://github.com/tw93/Mole) installed and `mo` available on your `$PATH`:

  ```sh
  brew install tw93/tap/mole
  ```

- Go 1.21+ to build from source
- Xcode Command Line Tools (`xcode-select --install`) because the tray library uses
  macOS cgo support

## Install

Build the app bundle:

```sh
git clone https://github.com/<you>/molebar.git
cd molebar
make app
```

This creates `build/MoleBar.app`.

Then drag `build/MoleBar.app` into `/Applications` and launch it from Spotlight.
The app is marked as a menu-bar-only app, so it does not appear in the Dock and
it does not show up in Cmd+Tab.

To run it directly from the terminal while developing:

```sh
make run
```

## Configuration

Both flags are optional:

```sh
molebar -interval=5s -mo-bin=/opt/homebrew/bin/mo
```

- `-interval` — how often to refresh the values (default: `5s`)
- `-mo-bin` — path to the `mo` binary if it is not on `$PATH`

This is useful when the app is started by a login item or `launchd`, because those
processes often have a limited `PATH` and may not find Homebrew-installed tools.

## Start at login

In System Settings:

System Settings → General → Login Items & Extensions → add `MoleBar.app`

## How it works

The app starts with the main Go entry point in `cmd/molebar/main.go`. It creates
menu items with the [`getlantern/systray`](https://github.com/getlantern/systray)
library and updates them on a timer.

The status values are fetched in `internal/molestatus/status.go`, which runs:

```go
exec.Command("mo", "status", "--json")
```

and parses the JSON into typed Go structs. If a refresh fails, the app logs the
error and keeps the previous values instead of crashing the tray.

This makes the menu bar app resilient during short Mole hiccups or a temporary
CLI delay.

## What is "CPU load1"?

`load1` is the 1-minute load average. It is different from CPU percentage.

- `CPU: 25%` tells you how busy the CPU is right now.
- `load1: 1.2` tells you how much work was waiting to run on average over the
  last minute.

A higher load value usually means the machine is busier, but you should compare it
to the number of CPU cores. On a 4-core Mac:

- `load1 = 1.0` is usually light load
- `load1 = 4.0` is roughly full saturation
- `load1 > 8.0` means the system is heavily overloaded

So `load1` is a system-level signal: it helps you see whether the machine is
under pressure even when the CPU percentage alone looks modest.

## Releasing

Tag a release and push it. The GitHub Actions workflow builds on macOS and uploads
a zipped `.app` file to the GitHub Release:

```sh
git tag v0.1.0
git push origin v0.1.0
```

## License

MIT — see [LICENSE](LICENSE).
