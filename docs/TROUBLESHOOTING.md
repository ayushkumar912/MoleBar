# Troubleshooting

## `mo` executable not found

```text
Symptom
  Tray title "mo: err". Logs: molebar: refresh failed: ... ErrNotFound
  ("mo executable not found"). Diagnostics last_error_category: executable_missing.

Likely cause
  mo is not on the process PATH. GUI / login-item / launchd launches often
  get a minimal PATH that omits Homebrew.

How to verify
  which mo
  ls /opt/homebrew/bin/mo /usr/local/bin/mo
  Run the same binary MoleBar uses: molebar -mo-bin=$(which mo)

Resolution
  Install Mole; pass -mo-bin=/opt/homebrew/bin/mo (or /usr/local/bin/mo);
  or put mo on the PATH of the launching context.
  ResolveBinary() already tries those two Homebrew paths after LookPath.

Relevant files
  internal/molestatus/exec.go — ResolveBinary(), Fetch()
  cmd/molebar/main.go — parseRuntime()
```

## `mo status --json` fails

```text
Symptom
  Title "mo: err" on an interval. Log includes exit code and stderr
  (ErrNonZero) or a timeout/cancel/parse error.

Likely cause
  Mole exited non-zero, hung past 5s, or printed non-JSON.

How to verify
  mo status --json
  mo status --help
  Compare stdout to fields in internal/molestatus/model.go.

Resolution
  Fix the Mole install / permissions. MoleBar does not retry a single
  Fetch beyond the next poll tick or watch restart. Last-good dropdown
  lines remain.

Relevant files
  internal/molestatus/exec.go — Fetch(), wrapRunError()
  internal/app/controller.go — OnResult(), Refresh()
```

## MoleBar displays stale values

```text
Symptom
  Dropdown still shows old CPU/memory/etc. after a failure, or the
  menu-bar title does not change while the menu is open.

Likely cause
  (1) Failed refresh keeps Controller.last and only sets lastErr
      (title becomes "mo: err").
  (2) apply() skips systray.SetTitle while platform.MenuIsTracking();
      flushTray() runs when tracking ends.
  (3) Watch/poll interval is -interval (default 5s).

How to verify
  Quit the menu and look at the title. Check logs for refresh failed.
  Use Refresh now (always Fetch --json). Export diagnostics for
  last_error_category and strategy.

Resolution
  Restore mo; wait for a successful sample. Do not expect the dropdown
  to clear on error. Close the menu to see a deferred title.

Relevant files
  cmd/molebar/menu.go — apply(), flushTray()
  internal/app/controller.go — OnResult()
  internal/presentation/presenter.go — Present() error title
```

## Session totals appear incorrect

```text
Symptom
  Session RX/TX do not match Activity Monitor or “feel” low/high.

Likely cause
  Totals are rate × elapsed estimates, not Mole byte counters.
  First sample (and first after failure / Reset / gap ≥ 60s) adds zero.
  Rates are the sum of whatever interfaces Mole listed.
  Integration uses 1<<20 bytes per MB/s.

How to verify
  Reset session totals, wait two successful samples, compare to
  displayed rates × time. Check diagnostics [session] and [status] rx_mbs.

Resolution
  This is current design, not a hidden exact counter. After an outage
  the next sample primes only (failure calls Meter.Invalidate()).

Relevant files
  internal/session/meter.go
  internal/molestatus/status.go — TotalNetRates()
  docs/ARCHITECTURE.md — Session Network Accounting
```

## Invalid refresh interval

```text
Symptom
  Process exits immediately; no menu-bar item.

Likely cause
  -interval <= 0 (including 0 and negative durations).

How to verify
  molebar -interval=0
  stderr: molebar: -interval must be greater than 0 (got 0s)
  exit status 2

Resolution
  Pass a positive duration (default 5s). There is no panic in
  validateInterval() or main(); failure is a returned error and os.Exit(2).

Relevant files
  cmd/molebar/main.go — validateInterval(), parseRuntime(), main()
  cmd/molebar/main_test.go
```

## Config changes are not behaving as expected

```text
Symptom
  Profile / title / alerts revert, or -title seems ignored after restart.

Likely cause
  -title is runtime-only and is not written. Invalid -title replaces
  the loaded file with DefaultPreferences() for that process.
  Malformed config.json loads as defaults with ok=false and err=nil.
  Legacy display_mode is left on disk and can confuse inspection.
  Custom layouts report DisplayMode() as sys.

How to verify
  cat ~/Library/Application\ Support/molebar/config.json
  ls ~/Library/Application\ Support/molebar/display_mode
  Restart without -title.

Resolution
  Change Profile or Tray Metrics in the menu (those persist).
  Do not rely on -title across restarts. Fix or remove a broken
  config.json. Do not expect the legacy file to be deleted.

Relevant files
  internal/config/file_store.go
  internal/config/config.go — ResolvePreferences()
  internal/app/controller.go — persist()
```

## App builds but does not appear in the menu bar

```text
Symptom
  Process running; nothing in the menu bar. No Dock icon.

Likely cause
  LSUIElement is true — no Dock / Cmd+Tab by design.
  systray.Run never reached onReady (flag error exits first).
  Status item title is "mo …" or a short string and is easy to miss.
  Another MoleBar instance may already own the item (not guarded in code).

How to verify
  Launch from Terminal: make run — errors go to stderr/logs.
  Activity Monitor for molebar. Confirm you are looking at the menu bar,
  not the Dock.

Resolution
  Run from a terminal to see logs. Confirm parseRuntime succeeded.
  Look for a text-only status item (no tray icon is set; README notes
  systray.SetIcon is not wired).

Relevant files
  cmd/molebar/main.go — main(), onReady()
  packaging/Info.plist — LSUIElement
  cmd/molebar/menu.go — applyTray()
```

## `go test` fails because systray/CGO dependency cannot build

```text
Symptom
  cmd/molebar or platform tests fail to compile: cgo, AppKit, systray
  Objective-C, missing SDK.

Likely cause
  CGO disabled; no C compiler / Xcode CLT; building Darwin files off macOS
  without using the !darwin stubs correctly; MACOSX_DEPLOYMENT_TARGET /
  SDK issues on universal builds.

How to verify
  go env CGO_ENABLED CC GOOS
  xcode-select -p
  go test ./internal/session ./internal/config   # no systray

Resolution
  On macOS: install Xcode Command Line Tools; do not set CGO_ENABLED=0
  when testing ./cmd/molebar. Off macOS, packages with //go:build darwin
  are excluded; cmd/molebar still imports systray and may not compile.
  Makefile sets CGO_ENABLED=1 GOOS=darwin for app builds.

Relevant files
  go.mod — github.com/getlantern/systray
  internal/platform/stayopen_darwin.go
  Makefile — build-native
```

## `.app` bundle does not launch

```text
Symptom
  MoleBar.app does nothing, or macOS Gatekeeper blocks it.

Likely cause
  Unsigned local build (CODESIGN_IDENTITY unset — expected).
  make app failed partway (missing packaging/MoleBar.icns would fail cp).
  Info.plist executable name mismatch (must be molebar).
  Quarantine on a downloaded zip.

How to verify
  open -a build/MoleBar.app
  ./build/MoleBar.app/Contents/MacOS/molebar
  /usr/libexec/PlistBuddy -c 'Print :CFBundleExecutable' \
    build/MoleBar.app/Contents/Info.plist
  codesign -dv build/MoleBar.app

Resolution
  Run the inner binary from Terminal to see logs. Rebuild with make app.
  For Gatekeeper, sign with CODESIGN_IDENTITY or remove quarantine on
  a bundle you built yourself. Notarization is not provided by this repo.

Relevant files
  Makefile — app, sign
  packaging/Info.plist
```

## Release version does not match app metadata

```text
Symptom
  About/diagnostics molebar_version is "dev" or a git-describe string
  while the tag is v0.1.2, or plist does not match the GitHub Release.

Likely cause
  Binary built without Makefile ldflags (version stays "dev").
  Untagged / dirty tree: VERSION from git describe (e.g. 0.1.2-4-g…).
  Plist only stamped by make app, not by go build.
  CI sets VERSION from the tag; a local make app without VERSION= may differ.

How to verify
  git describe --tags --always --dirty
  /usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' \
    build/MoleBar.app/Contents/Info.plist
  Export Diagnostics and read molebar_version.

Resolution
  make app VERSION=<tag-without-v> or tag a clean tree before dist.
  Confirm the release workflow printed matching PlistBuddy values.

Relevant files
  Makefile — VERSION, GOFLAGS
  cmd/molebar/main.go — var version
  .github/workflows/build.yml
```
