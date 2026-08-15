<p align="center">
  <img src="cmd/molebar/molebar-icon.png" alt="MoleBar icon" width="128" />
</p>

# MoleBar

A macOS menu-bar widget for [Mole](https://github.com/tw93/Mole) — shows live
CPU, memory, swap, disk, battery, and health score. No Electron, no background
daemon beyond the binary itself, no telemetry.

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

- **MoleBar** requires macOS 11+ (see `LSMinimumSystemVersion` in the app bundle).
- **[Mole](https://github.com/tw93/Mole)** must be installed separately so `mo` is
  on `$PATH`. The current Homebrew install path is:
  ```sh
  brew install mole
  ```
  Mole's own OS support is defined by that project; MoleBar does not add extra
  compatibility claims for Mole.
- Go 1.21+ (only needed to build; the released `.app` has no runtime Go dependency)
- Xcode Command Line Tools (`xcode-select --install`) — required to build
  `systray`'s cgo-based macOS backend

## Build

```sh
git clone https://github.com/ayushkumar912/MoleBar.git
cd molebar
make app        # builds build/MoleBar.app
```

`make check` verifies formatting (without rewriting files), modules, tests, and `go vet`.
`make test` / `make vet` run those steps individually. `make clean` removes `build/` and `dist/`.
`make app` builds a native-architecture bundle; `make app UNIVERSAL=1` (and `make dist`) builds a universal `arm64` + `x86_64` binary when the local macOS SDK can target both.

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

- `-interval` — refresh period (default `5s`). Must be greater than zero;
  invalid values exit with an error instead of starting the tray.
- `-mo-bin` — explicit path to the `mo` binary. Set this if you launch
  MoleBar via `launchd`/login item and it can't find `mo` on `$PATH`
  (launchd-started processes get a minimal `PATH` that often excludes
  Homebrew's `bin` directories)
- `-title` — runtime-only override of the always-visible menu bar text:
  - `sys` (default) — `CPU 12% MEM 67%`
  - `net` — `↓1.2 MB/s ↑340 KB/s`, Bandwidth+-style
  - `both` — both, space permitting

  A saved Display-menu preference is the default. An explicit `-title` wins
  for that process only and is **not** written to disk. Choosing System /
  Network / Both from the menu persists the new preference for later launches.

### Bandwidth monitoring

Live down/up rate and a "Session" line (estimated data transferred since
MoleBar launched) are always in the dropdown regardless of `-title`.

- **Rates are the sum of the network records Mole supplies.** Mole decides
  which interfaces appear in its JSON. MoleBar does not assume it receives
  every physical or VPN interface.
- **Session totals are estimated, not exact.** Mole exposes an instantaneous
  rate (MB/s), not a cumulative byte counter, so MoleBar integrates
  rate × elapsed time between valid samples. The first successful sample
  (and the first sample after a failure, a long gap, or a reset) only primes
  the meter — it does not add bytes. A failed refresh breaks that continuity
  so an outage is not attributed to a later rate. Click **Reset session totals**
  to zero the counters and clear sampling state.

## How it works

MoleBar prefers Mole's streaming command
`mo status --watch --interval=...` (newline-delimited JSON from one process).
If the installed Mole build does not support watch mode, it falls back to
polling `mo status --json`. Failed refreshes are logged and skipped rather
than crashing the tray; last-good dropdown values stay visible.

It does not link Mole as a library — Mole ships as a CLI.

## App icon

`packaging/MoleBar.icns` is bundled automatically by `make app` (via
`CFBundleIconFile` in `Info.plist`) — it's what shows in Finder, Launchpad,
and Get Info. It is **not** the menu-bar glyph itself; the menu bar currently
shows text only (`CPU 12% MEM 67%`). To also show a small icon next to that
text, wire in a monochrome template image via `systray.SetIcon()`.

## Auto-start at login

System Settings → General → Login Items & Extensions → add `MoleBar.app`.

## Releasing

The release version is the git tag (for example `v0.1.2`). That tag is the
source of truth: `make app` / `make dist` stamp `CFBundleVersion` and
`CFBundleShortVersionString` from `git describe`, or from `VERSION=` when
the GitHub Actions release job runs on a `v*` tag.

```sh
git tag v0.1.2
git push origin v0.1.2
```

`make dist` writes `dist/MoleBar-<version>.app.zip` (via `ditto`) and a
source tarball from `git archive` (no `.git`, no working-tree junk).

### Signing and notarization

Local `make app` does not require Apple credentials. To sign a local bundle:

```sh
make app CODESIGN_IDENTITY="Developer ID Application: Your Name (TEAMID)"
```

GitHub Releases can sign when `CODESIGN_IDENTITY` is provided as a repository
secret. Notarization (`notarytool` / a stored keychain profile) is optional and
is not run unless you add those secrets yourself. Do not commit certificates,
passwords, or profiles.

Typical secrets if you enable notarization later:

- `CODESIGN_IDENTITY` — Developer ID Application identity name
- An Apple API key or app-specific password for `notarytool` (store as Actions
  secrets; never in the repo)

## License

Apache License — see [LICENSE](LICENSE).
