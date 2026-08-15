# MoleBar Architecture

## High-Level Overview

MoleBar does not collect host metrics. Mole (the `mo` CLI) is the metrics source. MoleBar discovers `mo`, prefers a watch stream, falls back to polling, accounts session network totals, evaluates threshold alerts, and paints a macOS menu-bar item.

```text
macOS
  │
  ▼
Mole CLI  (mo status --watch | mo status --json)
  │
  ▼
MoleBar   (parse → Controller → presentation)
  │
  ▼
macOS Menu Bar  (getlantern/systray + AppKit stay-open helpers)
```

Destructive Mole operations (kill, clean, etc.) are not invoked.

## Runtime Data Flow

```text
parseRuntime() + FileStore.Load()
   │
   ▼
systray.Run → onReady()
   │
   ├── Detect(mo help/version)     3s timeout
   ├── newMenu() + KeepMenuOpenOnToggles()
   ├── NewSource() → goroutine Run()
   │         │
   │         ├── watch: mo status --watch --interval=<d>  (NDJSON)
   │         └── else:  poll mo status --json per ticker
   │
   ▼
updates chan (buffer 4)
   │
   ▼
event-loop goroutine
   ├── Controller.OnResult() / Refresh() / menu actions
   │       ├── session.Meter
   │       ├── history.History
   │       └── alerts.Engine
   ├── Controller.View() → presentation.Present()
   └── apply(menu, ViewModel) → systray titles / checks
           │
           └── firing/recovered events → notifyCh → DarwinNotifier (osascript)
```

Manual **Refresh now** always calls `molestatus.Fetch()` (`mo status --json`), even if watch is running.

## Component Map

| Component | File/Package | Responsibility |
| --------- | ------------ | -------------- |
| Composition root | `cmd/molebar/main.go` | Flags, `systray.Run`, `onReady()`, event loop, login/clipboard/export/notify wiring |
| Tray menu | `cmd/molebar/menu.go` | Menu construction, `apply()` / `flushTray()` |
| Controller | `internal/app/controller.go` | Single owner of mutable state; persist preferences |
| Domain snapshot | `internal/app/state.go` | Read-only `State` for presentation/diagnostics |
| Preferences API | `internal/config/config.go` | `Store`, `ResolvePreferences()`, `ResolveDisplayMode()` |
| Display mode | `internal/config/display_mode.go` | Legacy `sys` / `net` / `both` |
| Tray metrics | `internal/config/metric.go` | Metric identifiers and labels |
| Layout | `internal/config/layout.go` | `TrayLayout`, normalize/toggle, mode↔layout mapping |
| Profiles | `internal/config/profile.go` | Built-in presets |
| Preference schema | `internal/config/preferences.go` | `Preferences`, defaults, alert prefs, normalize |
| Persistence | `internal/config/file_store.go` | `config.json` + legacy `display_mode` |
| Mole JSON model | `internal/molestatus/model.go` | Partial `Status` struct |
| Mole helpers | `internal/molestatus/status.go` | `Parse()`, disk/battery/net/process helpers |
| Exec / errors | `internal/molestatus/exec.go` | `ResolveBinary()`, `Fetch()`, process group, sentinels |
| Capabilities | `internal/molestatus/capabilities.go` | `Detect()` via version/help |
| Adaptive source | `internal/molestatus/source.go` | Watch-then-poll `NewSource()` |
| Watch collector | `internal/molestatus/watch_source.go` | Long-lived `mo status --watch` |
| Poll collector | `internal/molestatus/polling_source.go` | Ticker + `Fetch()` |
| Process group (unix) | `internal/molestatus/proc_unix.go` | `Setpgid` / `SIGKILL` process group |
| Process group (other) | `internal/molestatus/proc_other.go` | `Process.Kill()` only |
| Session meter | `internal/session/meter.go` | Rate × time byte estimates |
| History | `internal/history/history.go`, `ring.go` | Bounded in-memory series |
| Alerts | `internal/alerts/rule.go`, `engine.go`, `notification.go` | Sustained-threshold engine; payload formatting only |
| Presenter | `internal/presentation/presenter.go`, `model.go` | `ViewModel` strings |
| Diagnostics | `internal/diagnostics/report.go` | Text report + clipboard summary |
| Login item | `internal/platform/loginitem*.go` | System Events via osascript |
| Notifications | `internal/platform/notification*.go` | `display notification` via osascript |
| Clipboard | `internal/platform/clipboard*.go` | `pbcopy` |
| Save dialog | `internal/platform/savedialog*.go` | osascript choose-file-name |
| Export write | `internal/platform/export.go` | Atomic write; fallback path |
| Stay-open menu | `internal/platform/stayopen*.go`, `stayopen_darwin.m` | AppKit checkbox views + `NSMenu` swizzle |
| AppleScript quoting | `internal/platform/applescript.go` | `quoteAppleScript()` |
| Bundle metadata | `packaging/Info.plist` | Identifier, `LSUIElement`, version placeholders |
| App icon | `packaging/MoleBar.icns` | Finder/Launchpad icon (not the menu-bar glyph) |
| README icon | `cmd/molebar/molebar-icon.png` | README image only |

## Application Startup

```text
cmd/molebar/main.go
main()
→ config.NewFileStore("")
→ parseRuntime(flag.CommandLine, os.Args[1:], store)
     → validateInterval()
     → platformInfo()                    // runtime.GOOS/GOARCH; darwin: sw_vers -productVersion
     → molestatus.ResolveBinary(-mo-bin)
     → config.ResolvePreferences(store, -title)
→ context.WithCancel
→ systray.Run(onReady, cancel)

onReady(ctx, cancel, cfg, store)
→ app.New(cfg, store, nil)
→ platform.NewDarwinLoginItem("")
→ syncLoginState()
→ molestatus.Detect(3s timeout)
→ ctrl.SetCapabilities(caps)
→ newMenu()
→ platform.KeepMenuOpenOnToggles()
→ platform.SetMenuClosedHandler(flushTray)
→ apply(m, ctrl.View())
→ molestatus.NewSource(opts)
→ go src.Run(ctx, emit→updates)
→ go runNotifier(...)
→ go event-loop select { updates | menu clicks | quit }
```

Startup does not write the preference file (`app.New` / `Controller` comments and tests).

## Mole Status Collection

Implemented in `internal/molestatus`. The rest of the app uses `Source` / `Result` and does not choose watch vs poll.

### Adaptive source

`source.go` — `NewSource()`, `adaptiveSource.Run()`:

1. Use `opts.Caps` if set; otherwise `Detect()`.
2. Try watch when `SupportsWatch` is true, **or** when the probe failed with something other than `ErrNotFound` / `ErrWatchUnsupported` (inconclusive probe).
3. If watch returns `nil`, stay on watch.
4. If watch returns an error (including giving up after restarts) and ctx is still live, log and run `PollingSource.Run()`.

`ErrNotFound` during detect emits one `Result{Err}` and then still starts polling (which will keep failing).

### Watch

`watch_source.go` — `WatchSource.stream()`:

- Command: `exec.CommandContext(ctx, bin, "status", "--watch", "--interval="+interval.String())`
- Default interval if unset: `5s` (`Options.interval()`)
- stdout: `json.Decoder` NDJSON into `Status`
- stderr captured for unsupported-flag heuristics
- Process group configured; cancelled ctx → `killProcessGroup()`
- Restarts: up to `watchMaxRestarts` (8) with backoff `1s` … `watchMaxBackoff` (`30s`)
- After 8 failures: error, adaptive source falls back to polling for the rest of the process
- Unsupported watch: `ErrWatchUnsupported` (stderr/exit heuristics in `isWatchUnsupported()`)

There is no per-sample timeout on the watch stream.

### Polling

`polling_source.go` — `PollingSource.Run()`:

- Immediate `emitOnce`, then `time.NewTicker(interval)`
- Each tick: `Fetch()` with `Options.timeout()` (default `5s`)

### One-shot fetch

`exec.go` — `Fetch()`:

- `mo status --json`
- If ctx has no deadline, wraps with `defaultFetchTimeout` (`5s`)
- stdout → `Parse()`; stderr included in `ErrNonZero` messages

### Refresh behavior

Background collection is continuous (watch or poll). Failed intervals log and skip; last-good `Controller.last` remains. Title shows `mo: err` when `ViewModel.Err != nil` (`presentation.Present()`).

## Data Model

`internal/molestatus/model.go` — `Status` is a **partial** mirror. Unknown JSON keys are ignored.

Fields MoleBar actually uses:

| Area | Fields consumed | Notes |
| ---- | --------------- | ----- |
| Health | `health_score` (`*int`), `health_score_msg` | Missing key → unavailable; present `0` is a real score (`Health()`) |
| CPU | `cpu.usage`, `cpu.load1` | `per_core` is parsed, unused in UI |
| Memory | `used_percent`, `swap_used`, `swap_total` | `total`/`used`/`available` parsed, unused in UI |
| Disks | `mount`, `used_percent` | Root only: `mount == "/"` (`PrimaryDiskPercent()`). Not array order. Missing `/` → `-1` / `n/a` |
| Batteries | first element: `percent`, `status`, optional `health`, `cycle_count`, `capacity` | `percent` is `float64`. `time_left` parsed, unused |
| Network | all records: `rx_rate_mbs`, `tx_rate_mbs` | `name`/`ip` parsed; IP never shown |
| Thermal | `cpu_temp` | `<= 0` → unavailable. GPU/battery/fan parsed, unused |
| Processes | `top_processes` `pid`, `name`, `cpu`, `memory_bytes` | Sorted by CPU, then name, then PID. Command lines are not modeled |
| Other | `collected_at`, `procs` | Parsed, unused (`Updated` uses Controller clock) |

## Session Network Accounting

`internal/session/meter.go` — `Meter`. Wired in `app.Controller.OnResult()`.

### Source of RX/TX rates

`Status.TotalNetRates()` sums `rx_rate_mbs` / `tx_rate_mbs` across the `network` array Mole supplied. MoleBar does not enumerate OS interfaces.

### Units

- Input: Mole’s `*_rate_mbs` treated as MB/s.
- Accumulation: `rate * (1<<20) * elapsed.Seconds()` (`bytesPerMB`).
- Display (`presentation.FormatRate()`): if rate `< 1`, `rate*1024` as `KB/s`; else one-decimal `MB/s`.
- Byte display (`FormatBytes()`): binary steps (KB/MB/GB via `1<<10` / `1<<20` / `1<<30`).

### Timing

Callers pass explicit timestamps. Controller uses `time.Now` (injectable in tests).

Integration is a right Riemann sum: **newly observed rate × elapsed since last accepted sample**.

| Event | Effect |
| ----- | ------ |
| First valid sample | Primes (`hasPrev=true`); adds **zero** bytes. Peaks record this rate. Duration starts at 0 until a later sample. |
| Sample after `Reset()` or `Invalidate()` | Same as first sample (prime only). |
| `elapsed <= 0` | Ignored (no add). |
| `elapsed >= MaxGap` | Re-prime; add zero. `DefaultMaxGap` is `60s`. `Controller` passes `cfg.MaxGap`; unset → `New(0)` → `DefaultMaxGap`. |
| Failed refresh (`Result.Err != nil`) | `meter.Invalidate()` — totals/peaks/duration kept; continuity broken. |
| Strategy-only result (`Status==nil`, `Err==nil`, `Strategy` set) | Does **not** invalidate (`controller_test.go`). |
| `Reset()` (menu **Reset session totals**) | Zeros RX/TX, peaks, duration, start time; next sample primes only. |

Averages: `(bytes / durationSeconds) / (1<<20)` when duration `> 0`.

### Known Accounting Limitations

- Totals are **estimates**, not Mole cumulative counters. Mole exposes instantaneous MB/s only.
- `1<<20` bytes per “MB” may not match a decimal-MB interpretation of Mole’s field name.
- Gaps just under 60s are fully integrated at the arriving rate (sleep/wake shorter than `MaxGap` can over-attribute).
- Watch jitter vs `-interval` changes the rectangle width.
- Negative rates are clamped so totals never go negative; they can still reduce a total toward zero.
- Peak/average/duration are computed on the meter but **average and duration are not shown in the tray menu** (they appear in diagnostics). Menu shows session combined line, Session RX/TX, Peak RX/TX.

## Configuration

See [Configuration](CONFIGURATION.md) for the full contract.

Summary:

- Path: `os.UserConfigDir()` + `molebar/config.json` (`FileStore.DefaultConfigPath()`). On macOS this is `~/Library/Application Support/molebar/config.json`.
- Legacy sibling `display_mode` (`sys`/`net`/`both`) is migrated **in memory** and never deleted.
- Format: versioned JSON (`Preferences`, `CurrentVersion = 1`).
- CLI `-title` overrides layout for the process and is not saved.
- Profile / Tray Metrics / Alerts / Launch-at-login preference changes call `Controller.persist()` → `Store.Save()`.

## Presentation / Systray

`cmd/molebar/menu.go` + `internal/presentation`.

### Tray title

`Present()` → `formatTitle(layout, status)` joins selected metrics with `layout.Separator` (default `" | "`). Optional metrics Mole omitted are skipped, not faked. No metrics renderable → `"mo …"`. Any `Err` → title `"mo: err"` (dropdown can still show last-good lines).

Examples from tests: `CPU 12% | RAM 67%`; health-only `100`; battery `80%` or `87.6%`.

Tooltip: `Health <score> (<msg>)` when health is present, else `"Mole system status"`.

While the menu is tracking, title/tooltip updates are deferred (`MenuIsTracking()` / `flushTray()`).

### Menu construction

`newMenu()` builds, in order: CPU, Memory, Swap, Disk, Temperature, Battery, Health; ↓/↑; Session / Session RX/TX / Peak RX/TX / Reset; Top Processes (5 disabled rows); Profile submenu (built-in profiles); Tray Metrics header (disabled) + metric checkboxes; Alerts toggle + 3 alert rows; Settings → Launch at Login; Updated (disabled); Refresh now; Copy System Summary; Export Diagnostics...; Quit.

`AvgRX`, `AvgTX`, `SessionDuration`, and `DisplayLabel` exist on `ViewModel` but are **not** applied to menu items.

### Events

Handled on the event-loop goroutine in `onReady()`:

| Menu | Controller / side effect |
| ---- | ------------------------ |
| Refresh now | `ctrl.Refresh(ctx)` + notify + `apply` |
| Reset session totals | `ctrl.ResetSession()` + `apply` |
| Alerts | `SetAlertsEnabled(!current)` + `apply` |
| Launch at Login | `toggleLogin()` + `apply` |
| Copy System Summary | `clip.Copy(vm.SystemSummary)` |
| Export Diagnostics... | save dialog or `DefaultDiagnosticsPath()`, then `WriteDiagnostics` |
| Profile items | `SetProfile(id)` + `apply` |
| Tray metric items | `ToggleMetric(m)` + `apply` |
| Quit | `cancel()`; `systray.Quit()` |

Stay-open titles in `stayopen_darwin.m` must match these English labels or checkboxes will dismiss the menu.

## Concurrency Model

```text
main goroutine
  systray.Run  (AppKit / systray main)

source goroutine
  WatchSource / PollingSource
    ├── cmd.Wait goroutine
    └── killer goroutine (ctx.Done → killProcessGroup)

notifier goroutine
  DarwinNotifier.Notify

event-loop goroutine
  select on ctx, updates, ClickedCh
  exclusive caller of Controller methods

AppKit thread
  stay-open swizzle / menu tracking
  molebarMenuTrackingEnded → flushTray()
```

Shared mutable state:

| State | Owner | Protection |
| ----- | ----- | ---------- |
| `Controller` fields | event loop | **No mutex.** Comment: not safe for concurrent use. |
| `alerts.Engine` | Controller | Same single-thread assumption. |
| `lastTray` ViewModel | `apply` / `flushTray` | `lastTrayMu` |
| `menuClosedHandler` | stay-open CGO | `menuClosedMu` |
| `updates` / `notifyCh` | channels, buffer 4 | Notify enqueue uses `select`/`default` (drops if full) |

Do not treat the process as generally thread-safe. Safety is “one event loop owns the Controller.”

## Process Lifecycle

Mole child processes:

- `CommandContext` + `configureProcessGroup()` (`Setpgid` on unix).
- On ctx cancel or watch decode end: `killProcessGroup()` (`SIGKILL` to `-pid`, then `Process.Kill()`).
- Polling: each `Fetch` is a short-lived process with its own timeout ctx.

Application shutdown:

- Quit menu: `cancel()` then `systray.Quit()`.
- `systray.Run` onExit is `cancel` (same cancel func).
- There is **no** dedicated `signal.Notify` for SIGINT/SIGTERM in `main.go`. Terminal interrupt behavior is whatever systray / the OS delivers to `onExit`.
- `defer cancel()` in `main()` runs after `systray.Run` returns.

## Error Handling

Sentinel errors (`exec.go`): `ErrNotFound`, `ErrWatchUnsupported`, `ErrMalformedJSON`, `ErrCanceled`, `ErrTimeout`, `ErrNonZero`, `ErrMalformedCapabilities`.

`ErrorCategory()` maps these to stable diagnostic labels (`executable_missing`, `timeout`, …).

User-visible:

- Failed refresh: log `molebar: refresh failed: ...`; title `mo: err`; dropdown last-good values kept.
- Capability probe failure: log `molebar: capability detection: ...`; collection still starts.
- Preference save failure: log `molebar: failed to save preferences: ...`.
- Login / copy / export / notify failures: log; notify errors ignored (`_ = n.Notify`).
- CLI parse / interval: stderr + exit `2`.

## Platform Dependencies

- **systray** CGO macOS backend.
- **AppKit** (`stayopen_darwin.m`): checkbox views, `NSMenu` `cancelTracking` swizzle, tracking notifications.
- **osascript**: notifications, login items (System Events), save dialog.
- **pbcopy**: clipboard.
- **sw_vers**: OS version for diagnostics.
- Bundle: `LSUIElement` true — no Dock / Cmd+Tab entry.
- `MACOSX_DEPLOYMENT_TARGET` / `LSMinimumSystemVersion`: `11.0`.
- Non-Darwin builds: platform operations return `ErrUnsupported`; stay-open is a no-op.

## Architectural Constraints

Preserve these unless intentionally changing the product:

- Mole is the metrics source. The tray must not grow its own host collectors.
- Do not link Mole as a library; keep the CLI subprocess boundary.
- Do not send signals or offer kill actions on process rows (read-only).
- Do not persist CLI `-title`.
- Do not delete the legacy `display_mode` file.
- Diagnostics must not include env vars, tokens, IPs, command lines, or home-directory contents (`diagnostics.Render` / tests).
- Controller remains the single writer of app state; presenter is a pure function.
- Destructive Mole commands stay out of scope.

## Known Limitations

These are **current specified behavior**, not open design questions and not planned work. Change them only when the user asks to change the contract, then update `docs/` in the same change.

- **Stay-open menu is fragile.** `stayopen_darwin.m` swizzles `NSMenu` `cancelTracking` / `cancelTrackingWithoutAnimation` process-wide and matches items by hardcoded English titles. Label changes break stay-open. It also walks the systray delegate’s ivars to find the status menu.
- **Controller has no mutex.** Correctness depends on `onReady`’s single event loop. A second caller would race.
- **Watch decode errors are not `ErrMalformedJSON`.** `watch_source.go` returns the raw `json.Decoder` error; diagnostics category becomes `unknown`. Same contract: `docs/MOLE_INTEGRATION.md`, `docs/DEVELOPMENT.md`.
- **Invalid `-title` drops saved prefs.** `ResolvePreferences()` returns `DefaultPreferences()` on parse error instead of keeping the store or failing CLI parse. Same contract: `docs/CONFIGURATION.md`, `docs/DEVELOPMENT.md`.
- **`TrayLayout.DisplayMode()` maps unknown/custom layouts to `sys`.** Legacy mode is a lossy approximation.
- **History is unused in the tray.** Samples exist only for diagnostics maxima.
- **ViewModel fields unused by `menu.go`:** `AvgRX`, `AvgTX`, `SessionDuration`, `DisplayLabel`, `ModeSys`/`ModeNet`/`ModeBoth`.
- **Alert rules have no menu editor.** Only enable/disable is exposed; thresholds live in `config.json` / defaults.
- **`process_cpu` is a valid alert metric** (`alerts.MetricProcessCPU`) but is not in `DefaultRules()` / `DefaultAlertPrefs()`.
- **Notification channel drops** when 4 events are already queued.
- **Login preference vs OS state** can diverge if `SetEnabled` fails (logged; menu re-reads OS).
- **CI `make dist` vs local `make dist`:** CI falls back to native if universal fails; `Makefile` `dist` requires `UNIVERSAL=1` with no fallback.
- **Notarization is not implemented.** Makefile and workflows have no `notarytool` step. README and `docs/BUILD_AND_RELEASE.md` state the same.
- **Module path vs clone URL** differ (`ayush-kumar912/molebar` vs `ayushkumar912/MoleBar`).
