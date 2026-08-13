<p align="center">
  <img src="cmd/molebar/molebar-icon.png" alt="MoleBar icon" width="128" />
</p>

# MoleBar

A macOS menu-bar widget for [Mole](https://github.com/tw93/Mole) — shows live
CPU, memory, swap, disk, battery, and health score by polling `mo status --json`
on an interval. No Electron, no background daemon beyond the binary itself,
no telemetry.

```
CPU 12% MEM 67%   ← menu bar title (default; switch to net rates or both — see Configuration)
├── CPU: 12.2%  (load1 2.94)
├── Memory: 67.0%
├── Swap: 24.8%
├── Disk: 39.5%
├── Battery: 80% (AC)
├── ──────────────
├── ↓ 1.2 MB/s
├── ↑ 340 KB/s
├── Session: ↓842.1 MB ↑112.4 MB
├── Reset session totals
├── Display: System
│   ├── System
│   ├── Network
│   └── Both
├── ──────────────
├── Health: 100 (Excellent)
├── Updated: 14:32:07
├── ──────────────
├── Refresh now
└── Quit
```

## Requirements

- macOS 11+
- [Mole](https://github.com/tw93/Mole) installed and `mo` on `$PATH`:
  ```sh
  brew install tw93/tap/mole
  ```
- Go 1.21+ (only needed to build; the released `.app` has no runtime Go dependency)
- Xcode Command Line Tools (`xcode-select --install`) — required to build
  `systray`'s cgo-based macOS backend

## Build

```sh
git clone https://github.com/<you>/molebar.git
cd molebar
make app        # builds build/MoleBar.app
```

Drag `build/MoleBar.app` into `/Applications`, then launch it from Spotlight.
It registers as a menu-bar-only app (`LSUIElement`, see `packaging/Info.plist`)
— no Dock icon, no Cmd+Tab entry.

To run it directly from the terminal while developing instead:

```sh
make run
```

## Configuration

Three flags, all optional:

```sh
molebar -interval=5s -mo-bin=/opt/homebrew/bin/mo -title=net
```

- `-interval` — refresh period (default `5s`)
- `-mo-bin` — explicit path to the `mo` binary. Set this if you launch
  MoleBar via `launchd`/login item and it can't find `mo` on `$PATH`
  (launchd-started processes get a minimal `PATH` that often excludes
  Homebrew's `bin` directories)
- `-title` — what the always-visible menu bar text shows:
  - `sys` (default) — `CPU 12% MEM 67%`
  - `net` — `↓1.2 MB/s ↑340 KB/s`, Bandwidth+-style
  - `both` — both, space permitting

The app also includes a menu-bar Display selector so users can switch between
System, Network, and Both modes directly from the tray menu. The chosen mode is
saved to the user's config directory and restored on the next launch.

### Bandwidth monitoring

Live down/up rate and a "Session" line (estimated data transferred since
MoleBar launched) are always in the dropdown regardless of `-title`. Two
things worth knowing:

- **Rates are summed across every interface** Mole reports (Wi-Fi,
  Ethernet, VPN tunnel, etc). If you're on a VPN, this can roughly
  double-count traffic, since the tunnel and the physical interface both
  carry the same bytes. See `TotalNetRates` in `internal/molestatus/status.go`
  if you want to filter to one named interface instead.
- **Session totals are estimated, not exact.** Mole's JSON exposes an
  instantaneous rate (MB/s), not a cumulative byte counter, so MoleBar
  integrates rate × elapsed time on every refresh tick. This is a good
  approximation at typical refresh intervals but isn't a true kernel-level
  byte count — expect it to drift somewhat from Activity Monitor's Network
  tab over long sessions. Click "Reset session totals" to zero it out.

## App icon

`packaging/MoleBar.icns` is bundled automatically by `make app` (via
`CFBundleIconFile` in `Info.plist`) — it's what shows in Finder, Launchpad,
and Get Info. It is **not** the menu-bar glyph itself; the menu bar currently
shows text only (`CPU 12% MEM 67%`). To also show a small icon next to that
text, wire in a monochrome template image via `systray.SetIcon()`.

## Auto-start at login

System Settings → General → Login Items & Extensions → add `MoleBar.app`.

## How it works

`molebar` shells out to `mo status --json` on a timer (see
`internal/molestatus/status.go`) and renders the fields it cares about into
a [`getlantern/systray`](https://github.com/getlantern/systray) menu-bar item.
It does not link Mole as a library — Mole ships as a CLI, so this just wraps
its JSON output. Failed refreshes (e.g. Mole mid-update) are logged and
skipped rather than crashing the tray; the last-good values stay displayed.

## Releasing

This repo is currently at version `0.1.1` and includes the display-mode
selection feature. Tag a release and push — `.github/workflows/build.yml`
builds on `macos-latest` and attaches a zipped `.app` to the GitHub Release:

```sh
git tag v0.1.1
git push origin v0.1.1
```

If you have a fork or remote set up, you can push the current branch to GitHub
with:

```sh
git add .
git commit -m "Add display mode selector"
git push origin main
```

If you want to publish a tagged release, use the tag command above.

## License

Apache License — see [LICENSE](LICENSE).
