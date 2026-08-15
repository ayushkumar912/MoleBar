# MoleBar Development Guide

MoleBar is a macOS menu-bar widget. It invokes the Mole CLI (`mo`), parses status JSON, and renders metrics in the system tray via `github.com/getlantern/systray`.

## Prerequisites

Documented from `go.mod`, `Makefile`, `packaging/Info.plist`, and source — not invented minima.

| Requirement | Source | Notes |
| ----------- | ------ | ----- |
| Go 1.21 | `go.mod` (`go 1.21`); CI `go-version: "1.21"` | Language/module version used by the repo and CI. |
| macOS | `Makefile` (`GOOS=darwin`); `packaging/Info.plist` `LSMinimumSystemVersion` `11.0` | The application is a Darwin menu-bar app. Unit tests for non-UI packages can compile on other OSes via `//go:build` stubs. |
| CGO | `Makefile` (`CGO_ENABLED=1`); `internal/platform/stayopen_darwin.go` | Required for systray’s macOS backend and the AppKit stay-open menu code. |
| Xcode Command Line Tools | `Makefile` (`xcrun --sdk macosx`); CGO / AppKit | Needed to compile CGO. The repo does not pin an Xcode version. |
| Mole CLI (`mo`) | `cmd/molebar/main.go`; `internal/molestatus/exec.go` — `ResolveBinary()` | Runtime dependency. Not vendored. Must be discoverable (see [Mole Integration](MOLE_INTEGRATION.md)). |
| Homebrew | README / `ResolveBinary()` fallbacks | Not required to build MoleBar. Relevant only as a common way to install Mole and as fallback paths `/opt/homebrew/bin/mo` and `/usr/local/bin/mo`. |

Direct dependency: `github.com/getlantern/systray v1.2.2` (`go.mod`).

## Clone / Setup

```sh
git clone https://github.com/ayushkumar912/MoleBar.git
cd MoleBar   # directory name follows the clone URL, not go.mod
```

The Go module path is `github.com/ayush-kumar912/molebar` (`go.mod`). That hyphenated path does not match the README clone URL.

No `go generate` step exists. From the repo root:

```sh
go mod download
```

Install Mole separately so `mo` is on `$PATH` (or pass `-mo-bin`). The README’s current Homebrew command is `brew install mole`.

## Build

Commands that exist in `Makefile` / work from the module root:

```sh
go build -o build/molebar ./cmd/molebar
make build              # alias for build-native
make build-native       # CGO_ENABLED=1 GOOS=darwin → build/molebar
make build-universal    # arm64 + amd64 via lipo → build/molebar
make app                # native .app at build/MoleBar.app
make app UNIVERSAL=1    # universal .app (fails if lipo cannot target both)
make run                # build-native, then ./build/molebar
```

`make all` is `app`.

Release-oriented targets (`dist`, `sign`, `clean`) are documented in [Build and Release](BUILD_AND_RELEASE.md).

`Makefile` always passes `-ldflags="-s -w -X main.version=$(VERSION)"`. `VERSION` defaults to `git describe --tags --always --dirty` with a leading `v` stripped, or `dev` if that is empty.

## Run

```sh
make run
# or
go run ./cmd/molebar
# or, after make build:
./build/molebar
```

`systray.Run` starts the menu-bar UI. There is no headless/dev flag.

Flags are defined in `cmd/molebar/main.go` — `parseRuntime()`:

| Flag | Type | Default | Meaning |
| ---- | ---- | ------- | ------- |
| `-interval` | `time.Duration` | `5s` | Refresh period. Must be `> 0`. Used as `mo status --watch --interval=...` and as the polling ticker. |
| `-mo-bin` | `string` | `""` | Explicit path to `mo`. Empty means `molestatus.ResolveBinary("")` (`$PATH`, then Homebrew fallbacks, then the name `"mo"`). |
| `-title` | `string` | `""` | Runtime-only tray-title override: `sys` / `system`, `net` / `network`, or `both`. Empty uses the saved preference, then the built-in default layout. **Not written to disk.** |

Invalid `-interval` (`<= 0`): `validateInterval()` returns an error; `main()` prints `molebar: ...` to stderr and exits `2`. There is no panic.

Invalid `-title` (for example `bogus`): `parseRuntime()` does **not** fail. `config.ResolvePreferences()` returns `DefaultPreferences()`, discarding any loaded saved preference for that process.

`-interval` and `-mo-bin` are not persisted.

## Test

```sh
go test ./...
go vet ./...
go test -race ./...
```

Makefile equivalents: `make test`, `make vet`, `make race`, `make check` (gofmt `-l` without rewrite, `go mod verify`, test, vet), `make fmt` (`gofmt -l -w .`).

Platform notes:

- `cmd/molebar` imports systray; compiling that package needs CGO on macOS.
- `internal/platform` Darwin files use AppKit / `osascript` / `pbcopy`. `//go:build !darwin` stubs exist for other OSes.
- `internal/molestatus` fake-`mo` helpers skip on Windows (`source_test.go`).
- `go test ./...` and `go vet ./...` passed during this documentation pass on macOS.
- `go test -race ./...` did **not** finish here: after `internal/molestatus` it did not complete `internal/platform` within several minutes. Do not assume race is clean locally. CI still runs `go test -race ./...` on `macos-latest` (`.github/workflows/ci.yml`).

See [Testing](TESTING.md).

## Formatting

The repo expects `gofmt`. `make check` and CI fail if `gofmt -l .` prints any path. `make fmt` rewrites files. There is no `golangci-lint` config.

## Common Development Loop

```text
change code
→ make fmt          # or gofmt -l -w .
→ go test ./...
→ go vet ./...
→ make build        # or make app
→ launch ./build/molebar or build/MoleBar.app
→ confirm menu-bar title, dropdown, Profile / Tray Metrics, Quit
```

Manual tray validation is required for systray, stay-open checkboxes, login item, notifications, clipboard, and the save dialog. Those are not covered by unit tests starting the real tray.

## Debugging

| Symptom | What the code does | Where to look |
| ------- | ------------------ | ------------- |
| Mole binary missing | `ResolveBinary()` may still return `"mo"`; `Fetch` / `Detect` wrap `ErrNotFound`. Tray title becomes `mo: err`; last-good dropdown lines stay. Logs: `molebar: refresh failed: ...` | `internal/molestatus/exec.go` — `ResolveBinary()`, `Fetch()`; `cmd/molebar/main.go` — `onReady()` |
| Malformed Mole output | `Parse()` returns `ErrMalformedJSON`. Watch decode errors are **not** wrapped as `ErrMalformedJSON`. Failed interval: meter `Invalidate()`, title `mo: err`. | `internal/molestatus/status.go` — `Parse()`; `watch_source.go` — `stream()` |
| Not in the menu bar | `systray.Run` must reach `onReady()`. Bundle is `LSUIElement` (no Dock icon). Wrong binary (non-`.app`) still can show a status item if systray starts. | `cmd/molebar/main.go`; `packaging/Info.plist` |
| Incorrect / stale config | Preferences live in `~/Library/Application Support/molebar/config.json` (via `os.UserConfigDir()`). Legacy `display_mode` is read in memory and **left on disk**. Malformed JSON → defaults, `ok=false`, no error. CLI `-title` is not saved. | `internal/config/file_store.go` |
| Stale tray title while menu is open | `apply()` skips `systray.SetTitle` while `platform.MenuIsTracking()`; `flushTray()` runs when tracking ends. | `cmd/molebar/menu.go` — `apply()`, `flushTray()` |
| Build / CGO / systray failure | Need `CGO_ENABLED=1`, a C compiler, and macOS SDK. Stay-open code links AppKit. | `Makefile`; `internal/platform/stayopen_darwin.go` |
| `.app` problems | `make app` copies `build/molebar` → `Contents/MacOS/molebar`, stamps `Info.plist`, copies `packaging/MoleBar.icns`. Missing icns fails the `cp`. Unsigned unless `CODESIGN_IDENTITY` is set. | `Makefile` — `app`, `sign`; [Troubleshooting](TROUBLESHOOTING.md) |

## Repository Conventions

Visible in the tree; there is no separate contribution policy file beyond [Contributing](CONTRIBUTING.md).

- Commands live in `cmd/molebar`. Domain logic is under `internal/`.
- Composition root is `cmd/molebar/main.go` — `onReady()`. Mutable app state is owned by `app.Controller` (no mutex; one event loop).
- Mole is invoked as a subprocess, never linked as a library.
- Darwin-only behavior uses `//go:build darwin` plus `!darwin` stubs.
- Tests are table-driven or fake-`mo` shell scripts; they do not start systray.
- `gofmt` is enforced. Version is the git tag, stamped at link time.
