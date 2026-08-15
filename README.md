<p align="center">
  <img src="cmd/molebar/molebar-icon.png" alt="MoleBar icon" width="128" />
</p>

# MoleBar

A macOS menu-bar widget for [Mole](https://github.com/tw93/Mole) — live CPU,
memory, disk, battery, temperature, health score, network rates, and top
processes. No Electron, no background daemon beyond the binary itself, no
telemetry.

```
CPU 12% | RAM 67%
├── CPU / Memory / Swap / Disk / Temperature / Battery / Health
├── ↓ / ↑ rates, session totals, peak rates
├── Top Processes (read-only)
├── Profile (Minimal, Developer, Network, Battery, Full)
├── Tray Metrics (compose the menu-bar title)
├── Alerts
├── Settings → Launch MoleBar at Login
├── Reset session totals
├── Refresh now
├── Copy System Summary
├── Export Diagnostics...
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
`make test` / `make vet` / `make race` run those steps individually. `make clean` removes `build/` and `dist/`.
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
  - `sys` (default) — `CPU 12% | RAM 67%`
  - `net` — `↓1.2 MB/s | ↑340 KB/s`
  - `both` — CPU, memory, and network

  Saved Profile / Tray Metrics preferences are the default. An explicit
  `-title` wins for that process only and is **not** written to disk.
  Changing Profile or Tray Metrics from the menu persists the new layout.
  A previous `display_mode` file (`sys` / `net` / `both`) is migrated in
  memory and left on disk.

### Profiles and tray metrics

Profiles are layout presets. The presenter receives the resolved layout; it
does not hard-code profile names.

| Profile    | Menu-bar metrics                          |
|------------|-------------------------------------------|
| Minimal    | CPU                                       |
| Developer  | CPU, Memory, RX, TX                       |
| Network    | RX, TX                                    |
| Battery    | Battery, Temperature, CPU                 |
| Full       | Health, CPU, Memory, RX, TX               |

Tray Metrics lets you compose any subset (CPU, Memory, RX, TX, Health,
Battery, Temperature, Disk). Those checkboxes live on the main menu and stay open after each toggle
so you can select or deselect several without the menu disappearing. Optional values that Mole does not supply are
omitted from the title rather than faked.

### Health score

Mole's `health_score` is shown in the menu and can be selected as a tray
metric (`92` or `92 | CPU 31%`). MoleBar does not recompute the score. If
Mole omits it, the title and Health line degrade to unavailable.

### Historical monitoring

MoleBar keeps a bounded in-memory history (about 10 minutes, sized from the
refresh interval) of CPU, memory, network rates, and optional temperature /
health. There is no on-disk database.

### Alerts

Sustained threshold alerts (not single-sample spikes) can notify on CPU,
memory, disk, temperature, battery, and network rates. Delivery uses the
macOS notification path; rule evaluation is separate from notification.
Alerts can be toggled from the menu. Cooldown prevents notification spam.

### Session statistics

Live down/up rate and session totals are always in the dropdown.

- **Rates are the sum of the network records Mole supplies.** Mole decides
  which interfaces appear in its JSON. MoleBar does not assume it receives
  every physical or VPN interface.
- **Session totals are estimated, not exact.** Mole exposes an instantaneous
  rate (MB/s), not a cumulative byte counter, so MoleBar integrates
  rate × elapsed time between valid samples. The first successful sample
  (and the first sample after a failure, a long gap, or a reset) only primes
  the meter — it does not add bytes. A failed refresh breaks that continuity
  so an outage is not attributed to a later rate. Peak and average rates and
  session duration are tracked the same way. Click **Reset session totals**
  to zero the counters and clear sampling state.

### Launch at Login

**Settings → Launch MoleBar at Login** registers the app as a login item
through supported macOS APIs (System Events). It does not edit arbitrary
user plists, require sudo, or install a privileged daemon. On unsupported
environments the item is disabled.

You can still add `MoleBar.app` from System Settings → General → Login
Items & Extensions.

### Diagnostics

**Copy System Summary** copies a short CPU/memory/disk/battery/health
snapshot. **Export Diagnostics...** writes a text report with MoleBar/Mole
versions, OS/arch, collector strategy, capability detection, profile,
refresh interval, latest metrics, session stats, and the last error
category. Reports do not include environment variables, tokens, IP
addresses, command lines, or home-directory contents.

## How it works

At startup MoleBar probes `mo` help/version to detect JSON and watch
support. It prefers Mole's streaming command
`mo status --watch --interval=...` (newline-delimited JSON from one process)
when watch is available. If the installed Mole build does not support watch
mode, or watch gives up after bounded restarts, it falls back to polling
`mo status --json`. Transient watch failures are retried; they do not
permanently switch to polling. Failed refreshes are logged and skipped
rather than crashing the tray; last-good dropdown values stay visible.

It does not link Mole as a library — Mole ships as a CLI.

Top processes, temperature, battery health, and cycle count are shown only
when Mole's JSON includes them. Process rows are read-only: MoleBar does
not send signals or offer kill actions.

## App icon

`packaging/MoleBar.icns` is bundled automatically by `make app` (via
`CFBundleIconFile` in `Info.plist`) — it's what shows in Finder, Launchpad,
and Get Info. It is **not** the menu-bar glyph itself; the menu bar currently
shows text only. To also show a small icon next to that
text, wire in a monochrome template image via `systray.SetIcon()`.

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
secret. The Makefile and workflows do not run notarization (`notarytool`).
Do not commit certificates, passwords, or profiles.

## Development

Developer documentation (authoritative for current code; this README is the user-facing entry point):

- [Development Guide](docs/DEVELOPMENT.md)
- [Architecture](docs/ARCHITECTURE.md)
- [Mole Integration](docs/MOLE_INTEGRATION.md)
- [Configuration](docs/CONFIGURATION.md)
- [Testing](docs/TESTING.md)
- [Build and Release](docs/BUILD_AND_RELEASE.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [Contributing](docs/CONTRIBUTING.md)
- [Repository Map](docs/REPOSITORY_MAP.md)

## License

Apache License — see [LICENSE](LICENSE).
