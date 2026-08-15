# Configuration

## CLI Options

Defined in `cmd/molebar/main.go` — `parseRuntime()`. These are the only flags.

| Flag | Type | Default | Persisted? | Meaning |
| ---- | ---- | ------- | ---------- | ------- |
| `-interval` | `time.Duration` | `5s` | No | Refresh period for watch `--interval=` and the poll ticker. Must be `> 0`. |
| `-mo-bin` | `string` | `""` | No | Explicit `mo` path. Empty → `ResolveBinary("")`. |
| `-title` | `string` | `""` | No | Runtime layout override: `sys`/`system`, `net`/`network`, `both`. Empty → saved prefs / defaults. |

No other flags exist. There is no flag for profile, alerts, or config path.

## Precedence

```text
built-in DefaultPreferences()
    ↓
saved FileStore.Load()   (config.json, or in-memory migration of display_mode)
    ↓
CLI -title               (this process only; not written)
```

Details (`config.ResolvePreferences()`):

1. Start from `DefaultPreferences()`: version `1`, empty profile, layout from `DefaultDisplayMode` (`sys` → CPU + Memory), `AlertsEnabled: true`, `DefaultAlertPrefs()`, `LaunchAtLogin: false`.
2. If `store.Load()` returns `(prefs, true, nil)`, replace with `prefs.Normalize()`. Load errors and `ok=false` leave defaults (errors are not surfaced to the user).
3. If `-title` is non-whitespace:
   - Valid mode → `prefs.ApplyDisplayMode(mode)` (layout + matching profile).
   - Invalid mode → **return `DefaultPreferences()`**, discarding the loaded file for this process. `parseRuntime()` still succeeds.

`-interval` and `-mo-bin` are independent of this stack. They only come from the CLI (and binary discovery).

Menu changes (profile, tray metrics, alerts enabled, launch-at-login pref) call `Controller.persist()` and write the store. They do not write `-title` or `-interval`.

## Persistence

### Location

`internal/config/file_store.go`:

| Helper | Path |
| ------ | ---- |
| `DefaultConfigPath()` | `filepath.Join(os.UserConfigDir(), "molebar", "config.json")` |
| `DefaultDisplayModePath()` | `filepath.Join(os.UserConfigDir(), "molebar", "display_mode")` |

On current macOS, `os.UserConfigDir()` is `~/Library/Application Support`. If `UserConfigDir` fails, both helpers return `""`. `NewFileStore("")` uses `DefaultConfigPath()`. An empty path: `Load` → defaults/`ok=false`; `Save` → error `save config: empty path`.

### Permissions

`Save()` uses `os.MkdirAll(dir, 0o755)` then `os.CreateTemp` + write + `Sync` + `Rename`. The file mode is whatever `CreateTemp` applies (not set explicitly in this repo).

### JSON schema

`config.Preferences` (`preferences.go`), `CurrentVersion = 1`:

```json
{
  "version": 1,
  "profile": "developer",
  "layout": {
    "metrics": ["cpu", "memory", "rx", "tx"],
    "separator": " | "
  },
  "alerts_enabled": true,
  "alerts": [
    {"metric": "cpu", "operator": ">", "value": 90, "for": "30s"}
  ],
  "launch_at_login": false
}
```

`Normalize()`: version `<= 0` → `1`; layout normalized; if `profile` is a built-in id, layout is replaced by that preset; if profile is unknown/custom, profile is set from `MatchingProfile`; `alerts == nil` → `DefaultAlertPrefs()`.

### When saves happen

| Action | Saves? |
| ------ | ------ |
| Startup / `New` | No |
| `ResetSession` | No |
| `SetProfile` / `ToggleMetric` / `SetDisplayMode` / `SetAlertsEnabled` / `SetLaunchAtLoginPref` | Yes (`persist()`) |
| CLI `-title` | No |

Save failures are logged (`molebar: failed to save preferences: ...`).

### Absent / malformed / invalid

| Situation | `Load()` | Runtime effect |
| --------- | -------- | -------------- |
| File missing | `ok=false`, try legacy sibling | Defaults unless legacy is valid |
| Empty file | same as missing | Defaults or legacy |
| Non-`{` contents | parse as legacy `sys`/`net`/`both` | Valid mode → migrated prefs `ok=true`; invalid → defaults `ok=false` |
| JSON unmarshal error | defaults, `ok=false`, **err=nil** | Silent defaults |
| Unreadable file (not IsNotExist) | defaults, `ok=false`, error | `ResolvePreferences` ignores error → defaults |
| Unknown metrics in layout | dropped by `NormalizeLayout` | Empty list becomes sys (CPU+Memory) |
| Invalid stored profile | treated as custom / rematch | See `Normalize()` |

Legacy `display_mode` is **never deleted**, including after a successful `config.json` save.

## Display Mode

Legacy identifiers (`display_mode.go`):

| Identifier | Aliases | Layout (`LayoutFromDisplayMode`) |
| ---------- | ------- | -------------------------------- |
| `sys` | `system`, `""` when parsing | CPU, Memory |
| `net` | `network` | RX, TX |
| `both` | | CPU, Memory, RX, TX |

`DefaultDisplayMode` is `sys`.

The current menu does **not** have a Display submenu. Profiles and Tray Metrics are the UI. `TrayLayout.DisplayMode()` is a **lossy** reverse map: layouts that are not exactly sys/net/both report `sys`.

`-title` applies one of these three layouts for the process only.

## Profiles and tray metrics

Built-in profiles (`profile.go` — `BuiltInProfiles()`):

| Profile ID | Menu label | Metrics |
| ---------- | ---------- | ------- |
| `minimal` | Minimal | CPU |
| `developer` | Developer | CPU, Memory, RX, TX |
| `network` | Network | RX, TX |
| `battery` | Battery | Battery, Temperature, CPU |
| `full` | Full | Health, CPU, Memory, RX, TX |

`custom` is the match when the layout equals none of the above. Presenter receives the resolved `TrayLayout`, not hard-coded profile names.

Tray metric IDs (`metric.go` — `AllMetrics()` order): `cpu`, `memory`, `rx`, `tx`, `health`, `battery`, `temperature`, `disk`.

`ToggleMetric` does not remove the last remaining metric.

Optional values Mole omits are skipped in the title (`formatMetric` returns `ok=false`).

## Alerts (persisted)

`DefaultAlertPrefs()` / `alerts.DefaultRules()`:

| Metric | Operator | Value | For |
| ------ | -------- | ----- | --- |
| cpu | `>` | 90 | 30s |
| memory | `>` | 85 | 60s |
| disk | `>` | 90 | 5m |
| temperature | `>` | 90 | 30s |
| battery | `<` | 15 | 10s |
| rx | `>` | 50 | 20s |
| tx | `>` | 50 | 20s |

There is no menu to edit rules. Changing thresholds means editing `config.json` (or relying on defaults). The menu **Alerts** item only toggles `alerts_enabled`. Disabling calls `engine.Reset()`.

Engine cooldown is `5 * time.Minute` (`app.New`). A rule still needs two consecutive crossings and `Duration` before firing (`alerts.Engine`).

`process_cpu` is a valid `alerts.Metric` but is not in the default pref list.

## Reset Behavior

**Reset session totals** (`Controller.ResetSession()` → `session.Meter.Reset()`) clears:

- accumulated RX/TX bytes
- peak RX/TX
- session duration / start time
- sampling continuity (`hasPrev`)

It does **not** reset: preferences, profile, layout, alerts, history buffer, last Mole status, or capabilities.

The next successful sample primes the meter and adds zero bytes.

## Examples

```sh
# defaults: 5s refresh, mo from PATH/fallbacks, saved layout or sys
molebar

# poll/watch every 2s
molebar -interval=2s

# launchd-friendly explicit binary
molebar -mo-bin=/opt/homebrew/bin/mo

# this process only: network title (not written)
molebar -title=net

# invalid interval — process exits 2, no tray
molebar -interval=0
```

Unsupported (do not use): `-profile`, `-config`, `-alerts`, `-display`.
