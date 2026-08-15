# Testing

## Test Layout

Every `*_test.go` in the repository:

| File | What it covers |
| ---- | -------------- |
| `cmd/molebar/main_test.go` | `validateInterval`; `parseRuntime` rejects `interval=0`; CLI `-title` does not persist; saved preference used when CLI title empty |
| `internal/app/controller_test.go` | First sample adds 0; failure breaks continuity; reset; mode/profile/metric persist; strategy-only result does not invalidate |
| `internal/session/meter_test.go` | Prime/integrate/reset/invalidate/gaps/negative clamp/peaks/averages/duration |
| `internal/config/display_mode_test.go` | Parse/normalize/labels including aliases and invalid |
| `internal/config/layout_test.go` | Normalize, mode migration, toggle last-metric, profile resolution, matching, apply toggle |
| `internal/config/profile_test.go` | Built-in labels; `ParseProfileID` |
| `internal/config/file_store_test.go` | Missing/legacy/sibling migrate/invalid/round-trip/legacy not deleted/CLI override not persisted |
| `internal/molestatus/status_test.go` | Parse schema, fractional battery, disk `/` vs order, multi-net, empties, health/temp, processes, malformed JSON |
| `internal/molestatus/source_test.go` | `Fetch` stderr/nonzero/malformed/timeout/cancel/not-found; polling emit; watch NDJSON; watch-unsupported fallback; watch shutdown |
| `internal/molestatus/capabilities_test.go` | Detect watch yes/no, missing binary, malformed help, transient failure, timeout |
| `internal/presentation/presenter_test.go` | Titles per mode/profile/metric; missing optionals; session/battery/processes/alerts; error keeps last-good menu text |
| `internal/alerts/engine_test.go` | Inactive, spike, sustained fire, two-sample rule, no repeat fire, recover, cooldown, invalid rule, missing metric |
| `internal/history/history_test.go` | `CapacityFor` clamps; add/summarize; wrap without growth |
| `internal/history/ring_test.go` | Empty, order, capacity, wrap, cap-1, no growth, zero→1 |
| `internal/diagnostics/report_test.go` | Render fields, determinism, no IP/env/path leak; summary omits missing health |
| `internal/platform/stayopen_test.go` | `KeepMenuOpenOnToggles()` does not panic; `MenuIsTracking()` false before tray |
| `internal/platform/loginitem_test.go` | Empty path → `ErrUnsupported`; AppleScript quoting |
| `internal/platform/loginitem_darwin_test.go` | Darwin-only mocked osascript enable/disable |

There are no tests that start `systray.Run` or click real menu items.

## Running Tests

Verified from the repo root on macOS during this documentation pass:

```sh
go test ./...          # passed
go vet ./...           # passed
```

Makefile:

```sh
make test
make vet
make race              # go test -race ./...
make check             # gofmt -l, go mod verify, test, vet
```

`internal/molestatus` fake-`mo` scripts skip on Windows.

Compiling `cmd/molebar` tests pulls in systray (CGO). Darwin `internal/platform` tests compile AppKit stay-open code.

## Unit Test Boundaries

Testable without starting the macOS tray:

- Flag parsing and interval validation
- Preference load/save/normalize/CLI override
- Mole JSON parse and domain helpers
- Fake-`mo` Fetch / Detect / watch / poll (POSIX)
- Session meter (injected timestamps)
- Controller `OnResult` / persist (memory `Store`)
- Presenter string formatting
- Alert engine (injected timestamps)
- History ring
- Diagnostics render
- Login-item logic with a fake runner (Darwin test)

## macOS-specific Validation

Still requires a running app / `.app` bundle:

- Status item appearing (`LSUIElement` — no Dock icon)
- Title/tooltip while the menu is open vs closed (`MenuIsTracking` / `flushTray`)
- Stay-open checkboxes (AppKit swizzle + hardcoded titles)
- Real `osascript` notifications, System Events login item, save dialog
- `pbcopy`
- Launch from login / launchd `PATH` vs `-mo-bin`
- Universal vs native binary (`lipo`)
- Codesign (only if `CODESIGN_IDENTITY` is set)

## Race Testing

```sh
go test -race ./...
```

CI runs this on `macos-latest` (`.github/workflows/ci.yml`).

This documentation pass: `go test -race ./...` completed `cmd/molebar` through `internal/molestatus`, then did **not** finish `internal/platform` within several minutes and was stopped. **Do not claim a local race-clean run.**

A plausible hang: `stayopen_darwin.go` — `keepMenuOpenOnToggles()` uses `dispatch_sync` to the main queue when not on the main thread. `TestKeepMenuOpenOnTogglesDoesNotPanic` calls that from a test goroutine. Without a running AppKit loop, `dispatch_sync` can block. Regular `go test` for that package was cached-success here; race may change scheduling.

`Controller` and `alerts.Engine` are documented as not concurrency-safe; tests call them from one goroutine.

## Test Gaps

### Critical

- **Watch JSON error category.** `WatchSource.stream()` does not wrap decode failures as `ErrMalformedJSON`. No test asserts `ErrorCategory` for a bad NDJSON line.
- **Invalid `-title`.** `ResolvePreferences` returns `DefaultPreferences()` and discards a valid store. No `parseRuntime` / `ResolvePreferences` test for `bogus`.
- **`OnResult` + alerts.** Controller tests do not assert `Evaluate` / firing / cooldown / `SetAlertsEnabled`.
- **Adaptive source `ErrNotFound` then poll.** Detect-missing still starts `PollingSource`; untested at the `NewSource` level.
- **Session `MaxGap` through Controller.** Meter tests cover gap; Controller does not pass a custom `MaxGap` in tests (zero → default 60s).

### Important

- **`FileStore` malformed JSON** (`ok=false`, `err=nil`) — no dedicated test (legacy invalid is covered).
- **Capability inconclusive → try watch** branch in `adaptiveSource.Run`.
- **Watch give-up after 8 restarts** (`watchMaxRestarts`).
- **Manual `Refresh` vs active watch** (extra `Fetch` process).
- **`persist()` save error** (logged only).
- **Login toggle failure** in `toggleLogin` (composition root, untested).
- **Diagnostics export / clipboard** at the `main` layer.
- **`rulesFromPrefs` invalid `for` duration** skipped; empty list → `DefaultRules()`.

### Nice to Have

- Presenter `FormatRate` / `FormatBytes` edge cases (NaN battery is coded; limited tests).
- Menu `apply` / `flushTray` / `lastTrayMu` (needs systray or a seam).
- Stay-open title-set drift vs `menu.go` labels.
- `platformInfo` `sw_vers` failure (empty `OSVersion`).
- Non-Darwin stub behavior in CI (CI is macOS only).
