# Mole CLI Integration

## Responsibility Boundary

| Side | Provides | Does not do |
| ---- | -------- | ----------- |
| **Mole (`mo`)** | Host metrics as JSON via `status` (one-shot or watch). Chooses which network interfaces appear. | MoleBar does not assume Mole returns every physical or VPN interface. |
| **MoleBar** | Binary discovery, capability probe, watch-or-poll collection, JSON decode of the fields it models, session integration, alerts, tray UI. | Does not link Mole. Does not recompute `health_score`. Does not kill processes. Does not run other Mole subcommands (clean, uninstall, …). |

## Binary Discovery

`internal/molestatus/exec.go` — `ResolveBinary(explicit)`:

1. If `explicit != ""` (CLI `-mo-bin`), return it unchanged (not `LookPath`’d).
2. Else `exec.LookPath("mo")`.
3. Else `os.Stat` these Homebrew fallbacks, first hit wins:
   - `/opt/homebrew/bin/mo`
   - `/usr/local/bin/mo`
4. Else return `"mo"` (unresolved name; later exec fails as `ErrNotFound`).

`parseRuntime()` always stores `ResolveBinary(*binPath)` in `app.Config.BinPath`. Lookup is done once at startup.

`Detect()` / `Fetch()` also call `ResolveBinary("")` if they receive an empty bin.

## Commands Executed

Every Mole invocation in this repository:

### Capability: version

```text
Command:     <bin> version
             then, if that fails (and not not-found / ctx err): <bin> --version
Arguments:   version   OR   --version
Caller:      internal/molestatus/capabilities.go — probeVersion() via Detect()
Purpose:     Populate Capabilities.Version (first line of stdout)
Expected:    any text; first line kept
Failure:     version probe failure is ignored unless not-found or ctx err;
             both failing with a normal exit error → ErrNonZero
```

### Capability: help

```text
Command:     <bin> status --help
             then, if that fails (and not not-found / ctx err): <bin> --help
Arguments:   status --help   OR   --help
Caller:      capabilities.go — probeHelp() via Detect()
Purpose:     Set SupportsJSON / SupportsWatch if the lowercased text contains
             "--json" / "--watch"
Expected:    help text on stdout and/or stderr
Failure:     not-found / ctx → wrapped; empty text and empty version →
             ErrMalformedCapabilities; other help failures → wrapRunError
```

`Detect()` is called from `cmd/molebar/main.go` — `onReady()` with a **3s** timeout, and from `adaptiveSource.Run()` if `opts.Caps` is nil.

### Status watch

```text
Command:     <bin> status --watch --interval=<duration>
Arguments:   status, --watch, --interval=<d>   (d from Options.interval(), default 5s)
Caller:      internal/molestatus/watch_source.go — WatchSource.stream()
Purpose:     Long-lived NDJSON status stream
Expected:    successive JSON objects on stdout
Failure:     unsupported-flag heuristics → ErrWatchUnsupported;
             process error → wrapRunError;
             decode error (not EOF) → returned as parse error (not ErrMalformedJSON);
             clean EOF → "watch stream from <bin> ended";
             up to 8 restarts with backoff, then give up
```

### Status poll / fetch / Refresh now

```text
Command:     <bin> status --json
Arguments:   status, --json
Caller:      exec.go — Fetch()
             polling_source.go — PollingSource.emitOnce()
             app/controller.go — Controller.Refresh()
             cmd/molebar/main.go — menu "Refresh now"
Purpose:     One snapshot
Expected:    a single JSON object on stdout
Failure:     timeout / cancel / not-found / non-zero (stderr attached) / Parse error
```

No other `mo` subcommands are executed.

Non-Mole commands (for completeness; not Mole):

| Command | Caller | Purpose |
| ------- | ------ | ------- |
| `sw_vers -productVersion` | `cmd/molebar/main.go` — `platformInfo()` | Diagnostics OS version |
| `osascript` (stdin script) | `platform` notification, login item, save dialog | macOS UI services |
| `pbcopy` | `platform/clipboard_darwin.go` | Copy summary |

## JSON Contract

Only fields MoleBar unmarshals and then reads. Types are Go fields on `molestatus.Status` and nested structs (`model.go`).

| JSON path | Local field/type | Used for | Required? |
| --------- | ---------------- | -------- | --------- |
| `health_score` | `HealthScore *int` | Tray/menu health, tooltip, history, diagnostics | No; nil → unavailable |
| `health_score_msg` | `HealthMsg string` | Health line / tooltip / diagnostics | No |
| `cpu.usage` | `CPU.Usage float64` | Title, menu, alerts, history | No (zero if omitted) |
| `cpu.load1` | `CPU.Load1 float64` | Menu CPU line | No |
| `cpu.per_core` | `CPU.Cores []float64` | Parsed only | No |
| `memory.used_percent` | `Memory.UsedPercent float64` | Title, menu, alerts, history | No |
| `memory.swap_used` / `swap_total` | `int64` | `SwapPercent()` | No; total `0` → 0% |
| `memory.total` / `used` / `available` | `int64` | Parsed only | No |
| `disks[].mount` | `Disk.Mount string` | Select `/` | No |
| `disks[].used_percent` | `Disk.UsedPercent float64` | Disk title/menu/alerts | Only if a `/` row exists |
| `disks[].total` / `used` / `purgeable` | `int64` | Parsed only | No |
| `batteries[0].percent` | `Battery.Percent float64` | Battery title/menu/alerts | No if array empty |
| `batteries[0].status` | `Battery.Status string` | Menu text; charging inference | No |
| `batteries[0].health` | `string` | `BatteryInfo.Health` / diagnostics | No; empty → omitted |
| `batteries[0].cycle_count` | `int` | diagnostics if != 0 | No |
| `batteries[0].capacity` | `int` | `BatteryInfo` if != 0 | No |
| `batteries[0].time_left` | `string` | Parsed only | No |
| `network[].rx_rate_mbs` | `Network.RxRateMBs float64` | Summed rates, session, alerts, history | No (empty → 0) |
| `network[].tx_rate_mbs` | `Network.TxRateMBs float64` | Same | No |
| `network[].name` | `string` | Parsed only | No |
| `network[].ip` | `string` | Parsed; **never displayed or exported** | No |
| `thermal.cpu_temp` | `Thermal.CPUTemp float64` | Title/menu/alerts/history if `> 0` | No |
| `thermal.gpu_temp` / `battery_temp` / `fan_speed` | various | Parsed only | No |
| `top_processes[].pid` | `int` | Sort / fallback label | No |
| `top_processes[].name` | `string` | Process rows / diagnostics | No |
| `top_processes[].cpu` | `float64` | Sort, display, `MaxProcessCPU` | No |
| `top_processes[].memory_bytes` | `uint64` | Stored on `ProcessStat` | No |
| `top_processes[].memory` | `float64` | Parsed only | No |
| `collected_at` | `string` | Parsed only | No |
| `procs` | `int` | Parsed only | No |

`Parse()` (`status.go`) wraps `json.Unmarshal` errors as `ErrMalformedJSON`. Extra keys are ignored.

## Network Semantics

`Status.TotalNetRates()` sums every `network` record’s `rx_rate_mbs` and `tx_rate_mbs`.

Mole decides membership of that array. MoleBar does not filter, dedupe, or query the OS for interfaces. An empty or missing array yields `0, 0` (shown as `0 KB/s`).

Units and session integration: [Architecture — Session Network Accounting](ARCHITECTURE.md#session-network-accounting).

## Disk Selection

`Status.PrimaryDiskPercent()` (`status.go`):

- Walk `disks` and return `used_percent` of the entry whose `mount == "/"`.
- If none, return `-1` (UI: `Disk: n/a`; metric omitted from title; disk alert not given a value).

This is **not** “first array element.” Tests cover a `/Volumes/...` row listed first.

## Battery Semantics

`PrimaryBattery()` / `PrimaryBatteryInfo()` use **`batteries[0]`** when the array is non-empty.

- `percent` is `float64`. Presentation does not truncate: integers as `%.0f%%`, otherwise `%.1f%%` (`formatBatteryPercent()`).
- Charging is inferred from `status` (`chargingFromStatus()`):  
  true: `ac`, `charging`, `charged`, `finishing charge`  
  false: `battery`, `discharging`, `running on battery`  
  other / empty: `Charging` pointer left nil.
- Optional `health` / `cycle_count` / `capacity` use zero/empty as absent.

Desktop machines with an empty `batteries` array: unavailable (`Battery: n/a`).

## Timeout and Failure Handling

| Path | Timeout | On failure |
| ---- | ------- | ---------- |
| `Detect` from `onReady` | 3s | Logged; `opts.Caps` omitted so `NewSource` may probe again |
| `Fetch` without deadline | `defaultFetchTimeout` 5s | `ErrTimeout` |
| Polling `emitOnce` | `Options.Timeout` or 5s | `Result.Err` set |
| `Controller.Refresh` | `cfg.FetchTimeout` (defaulted to 5s in `New`) | same as `OnResult` failure |
| Watch stream | none per sample; process tied to app ctx | restart / fallback |

`wrapRunError()` maps ctx deadline → `ErrTimeout`, cancel → `ErrCanceled`, missing binary → `ErrNotFound`, `ExitError` → `ErrNonZero` (stderr included).

Unix: process group kill on timeout/cancel (`proc_unix.go`).

Failed collection: `Controller.OnResult` invalidates the session meter, stores `lastErr`, logs, keeps `last` status. Title `mo: err`.

## Compatibility Assumptions

MoleBar currently assumes:

- A `mo` executable exists that accepts `status --json` and, optionally, `status --watch --interval=...`.
- Watch vs JSON support can be inferred from help text containing `--watch` / `--json` (not a version matrix).
- Status output is a JSON object (watch: one object per decode).
- Field names and shapes listed above.
- Network rates are in MB/s under `rx_rate_mbs` / `tx_rate_mbs`.
- Temperatures are Celsius; `cpu_temp <= 0` means “not supplied.”
- `health_score` omitted means unavailable; `0` is valid.
- Unknown flags for `--watch` produce recognizable stderr (`flag provided but not defined`, `unknown flag`/`option`) and/or exit `2`.
- `mo version` or `mo --version` may work; both are tried.

These are assumptions encoded in this repo, not a guarantee of every upstream Mole release.

## Failure Matrix

| Failure | Current behavior | User impact | Debugging action |
| ------- | ---------------- | ----------- | ---------------- |
| `mo` missing | `Detect`/`Fetch` → `ErrNotFound`. Adaptive source emits the error, then polls (also fails). Title `mo: err`. | Persistent error title; dropdown placeholders or last-good. | Confirm `which mo`; pass `-mo-bin`; check launchd `PATH` vs Homebrew prefixes. `ResolveBinary()` / `exec.go`. |
| Mole exits non-zero | `ErrNonZero`; stderr in error text | Same as other refresh failures | Run `mo status --json` in a terminal; read MoleBar logs. |
| Timeout | `ErrTimeout` after 5s (fetch/poll) or 3s (startup detect) | Error title for that interval; meter invalidated | Check a hung `mo`; inspect `ErrorCategory` in an exported report (`timeout`). |
| Malformed JSON (fetch/poll) | `Parse` → `ErrMalformedJSON` | Error title; last-good dropdown | Capture `mo status --json`; compare to `model.go`. |
| Malformed NDJSON (watch) | Decoder error, **not** wrapped as `ErrMalformedJSON`; stream ends; restart/fallback | Transient `mo: err`; possible switch to polling after 8 restarts | Logs `watch stream ended`; diagnostics category may be `unknown`. |
| Missing expected field | Zero / nil / empty; helpers treat as unavailable | Metric omitted from title; menu `n/a` where coded | Expected: health, disk `/`, battery, temp. |
| Watch unsupported | `ErrWatchUnsupported`; fallback to polling | Polling CPU cost; functionally same snapshots | `mo status --help` should lack `--watch`. |
| Watch crashes repeatedly | 8 restarts, then polling for this process | Same | Logs `falling back to polling`. Restart MoleBar to retry watch. |
| Empty capability output | `ErrMalformedCapabilities` | Probe error logged; source may still try watch | Check `mo status --help` / `mo --help`. |
