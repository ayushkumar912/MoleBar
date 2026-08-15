# MoleBar Agent Instructions

Operating contract for AI coding agents in this checkout. Read this file before editing. Link out to `docs/` instead of copying them.

## Mandatory Reading

Before any edit:

1. The user request (complete).
2. This file.
3. The `docs/` files that cover the affected contract (table below).
4. The implementation and `*_test.go` in the affected package(s).
5. Call sites of symbols you will change.

| If the task touches | Read first |
| ------------------- | ---------- |
| Packages, data flow, ownership, goroutines | `docs/ARCHITECTURE.md` |
| `mo` commands, JSON, discovery, timeouts | `docs/MOLE_INTEGRATION.md` |
| Flags, prefs, profiles, alerts persistence | `docs/CONFIGURATION.md` |
| Tests, race, coverage gaps | `docs/TESTING.md` |
| Make / CI / version / `.app` | `docs/BUILD_AND_RELEASE.md` |
| Setup, flags, local loop | `docs/DEVELOPMENT.md` |
| Change discipline | `docs/CONTRIBUTING.md` |
| Symptom → files | `docs/TROUBLESHOOTING.md` |
| Tree orientation | `docs/REPOSITORY_MAP.md` |

Do not start by editing the first matching file. Name exact paths in plans, conflicts, and reports (`internal/config/file_store.go`, not “the config code”).

## Source of Truth

`docs/` is the primary source for **required / intended repository behavior** and project rules.

Source and tests are authoritative for **what the current implementation actually does**.

| Question | Authority |
| -------- | --------- |
| What should this repo do? What must stay true? | `docs/` (especially Architecture, Mole Integration, Configuration) |
| What does this checkout do today? | `.go` sources and `*_test.go` |
| How to build / release? | `Makefile`, `.github/workflows/`, `docs/BUILD_AND_RELEASE.md` |
| CLI flag defaults | `cmd/molebar/main.go` — `parseRuntime()` (not README) |

If docs and code disagree in a way that changes what you would implement: **stop** (see Resolving Ambiguity). Do not silently pick one.

`README.md` is user-facing. Do not treat it as the contract when `docs/` or source differ.

## Resolving Ambiguity

If documentation, source, tests, or the user task conflict **and** that conflict changes the implementation:

```text
Ambiguity detected

Relevant files:
- `docs/...`
- `internal/...`
- `..._test.go`

Conflict:
<precise description>

Option A:
<behavior>

Option B:
<behavior>

Impact:
<what changes depending on the choice>

Need decision:
<single specific question>
```

Do not guess, pick the easiest option, invent a third behavior, or silently rewrite docs to match code (or code to match docs).

If `docs/` already states the intended rule clearly, follow `docs/` and do not ask.

`docs/ARCHITECTURE.md` — **Known Limitations** are the current contract (invalid `-title` → `DefaultPreferences()`; watch NDJSON errors are not `ErrMalformedJSON`; custom layouts report `DisplayMode()` `sys`). Do not treat those bullets as unresolved ambiguity. Change them only when the user asks to change the contract.

## Required Workflow

1. **Read** — request, this file, relevant `docs/`, implementation, tests.
2. **Discover** — repository-wide search for definitions, call sites, interfaces, tests, config keys, doc references, and (if relevant) Make/CI/plist. Map the full surface before editing.
3. **Define scope** — files that must change, may change, and must not change. Do not expand because adjacent code “could be cleaner.”
4. **Implement** — smallest change that satisfies the request, `docs/` contract, architecture rules, and tests.
5. **Validate** — run the commands in Testing and Validation. Never claim a command passed if it was not run.
6. **Review diff** — `git diff`; drop unrelated edits, formatting churn, and debug leftovers; confirm docs still match if a contract changed.
7. **Report** — use Completion Report.

## Scope Discipline

Must not:

- Opportunistic refactors, unrelated renames, or extra `gofmt` of files you did not need
- Dependency or Go-version upgrades unless requested (`go.mod` is `go 1.21`; only direct require is `github.com/getlantern/systray`)
- Tray UX redesign, new features, or architecture swaps “because a better pattern exists”
- Abstractions, interfaces, or event buses not required by the task
- Silent user-visible changes (flags, tray strings, Mole commands, config path/schema)
- Build/release/CI changes unless the task requires them

Unrelated defect: leave it; mention it in Remaining Issues if relevant; fix only if it blocks the requested change.

If the working tree already has edits: inspect overlapping files first; do not reset, revert, or overwrite them; distinguish pre-existing vs agent changes in the report.

## Repository Architecture

Details: `docs/ARCHITECTURE.md`. Preserve:

- **Composition root:** `cmd/molebar/main.go` — `main()`, `parseRuntime()`, `onReady()`. Wire dependencies; do not dump domain logic here.
- **Single writer:** `internal/app/controller.go` — `Controller` owns status, `session.Meter`, `history.History`, `alerts.Engine`, prefs. No mutex. Call only from the `onReady` event loop.
- **Mole is the metrics source.** Do not add host collectors in the tray. Do not link Mole. Do not run destructive Mole commands. Process rows stay read-only (no kill/signals).
- **Mole I/O** stays in `internal/molestatus` (`ResolveBinary`, `Detect`, `Fetch`, `NewSource`, `Parse`). UI callbacks must not exec `mo` or unmarshal JSON.
- **Session accounting** stays in `internal/session`. Presentation must not integrate rates.
- **Presenter** (`internal/presentation.Present`) is a pure function of `presentation.State`.
- **Darwin I/O** stays in `internal/platform` with `//go:build darwin` and `!darwin` stubs (`ErrUnsupported`).
- **Bounded state:** history capacity, `watchMaxRestarts` (8), channel buffers (updates/notify = 4). Do not grow unbounded queues or retry loops.
- **Shutdown:** cancel the root `context`; Mole children use `configureProcessGroup` / `killProcessGroup`. Do not leak processes or goroutines.
- **Diagnostics:** `internal/diagnostics` must not emit env vars, tokens, IPs, command lines, or home paths.

Stay-open titles in `internal/platform/stayopen_darwin.m` must stay in sync with `cmd/molebar/menu.go` labels.

## Mole Integration Rules

Contract: `docs/MOLE_INTEGRATION.md`. Implementation: `internal/molestatus/`.

Before changing Mole-related code, verify against that doc and the current sources:

- Commands actually invoked: `version` / `--version`; `status --help` / `--help`; `status --watch --interval=<d>`; `status --json`
- `ResolveBinary()` order (explicit `-mo-bin` → `LookPath` → Homebrew paths → `"mo"`)
- JSON fields MoleBar **consumes** (`model.go`) — do not invent upstream schema
- Units: `rx_rate_mbs` / `tx_rate_mbs` treated as MB/s; session uses `1<<20` bytes per MB (`internal/session/meter.go`)
- Optional fields (health, `/` disk, battery, `cpu_temp > 0`)
- Timeouts: detect 3s in `onReady`; fetch/poll/refresh default 5s; watch has no per-sample timeout
- stderr on `ErrNonZero`; cancel/timeout sentinels in `exec.go`
- Compatibility is help-text probing, not a version matrix

Never guess Mole’s schema from memory. If the task needs undocumented upstream behavior, inspect this repo’s integration first. Do not add `mo` subcommands unless that is the task.

Use `exec.CommandContext(bin, args...)` with separate arguments. Never `sh -c` for Mole.

## Configuration Rules

Contract: `docs/CONFIGURATION.md`. Implementation: `internal/config/`, `cmd/molebar/main.go` — `parseRuntime()`.

- Only flags: `-interval` (default `5s`, must be `> 0`, exit `2` if not), `-mo-bin`, `-title`. None of these are persisted.
- Precedence: `DefaultPreferences()` → `FileStore.Load()` (`config.json` or in-memory `display_mode`) → CLI `-title` (process only).
- Path: `os.UserConfigDir()` + `molebar/config.json`. Legacy sibling `display_mode` is readable and **must not be deleted**.
- Do not persist `-title`. Menu profile / tray metrics / alerts / launch-at-login pref go through `Controller.persist()`.
- Preserve schema compatibility (`Preferences` version `1`). Do not invent a new discard path for old/invalid files; follow documented Load behavior (including silent defaults on malformed JSON).
- Tests must use isolated temp paths (`t.TempDir()`, `NewFileStore(path)`). Never read or write the developer’s real `~/Library/Application Support/molebar/`.

## Go Engineering Rules

Match the current tree (`docs/CONTRIBUTING.md`):

- `gofmt`; wrap errors with `%w` so `molestatus.ErrorCategory` / `errors.Is` keep working
- `context.Context` on subprocess and platform I/O
- Concrete types by default; small interfaces only at substitution points that already exist (`config.Store`, `molestatus.Source`, `platform.Notifier` / `Clipboard` / `LoginItemManager` / `SavePathChooser`)
- Inject clocks and deps via constructors/parameters (`app.New(..., now)`, fake `mo` scripts). No DI framework, service locator, or generic event bus
- Deterministic tests: explicit timestamps (`Meter.Observe(at)`, `Engine.Evaluate(at)`). Do not sleep to fake time except process-timeout tests
- Do not add mutexes to `Controller` / `alerts.Engine` unless the task is to change that ownership model
- Do not add a goroutine per feature; extend the existing event loop / source / notifier

## Concurrency and Process Lifecycle

- Every goroutine needs an owner and a termination path (root `ctx` or `systray.Quit`)
- No unbounded channels or unbounded retries. Watch backoff is already capped (`watch_source.go`)
- New ticker: duration `> 0`; `Stop()` on shutdown (see `polling_source.go`)
- New retry: bounded + backoff; distinguish permanent (`ErrNotFound`, `ErrWatchUnsupported`) vs transient
- No concurrent `Controller` mutation
- `lastTray` is the UI snapshot guarded by `lastTrayMu` in `cmd/molebar/menu.go` — do not conflate it with domain state

## Security Rules

- No `sh -c` / shell interpolation for Mole or platform tools (`osascript` and `pbcopy` already take stdin scripts, not `sh -c`)
- Do not commit secrets, keychains, or `CODESIGN_IDENTITY` values
- Do not expand diagnostics to env, tokens, IPs, command lines, or home contents
- No new arbitrary command execution; no `sudo` / privileged daemons unless explicitly required
- Keep GitHub Actions permissions least-privilege (`contents: read` on `ci.yml`; `contents: write` only on the release job)

## Testing and Validation

Authoritative commands: `docs/TESTING.md`, `Makefile`, `.github/workflows/ci.yml`.

Prefer:

```sh
make check    # gofmt -l, go mod verify, go test ./..., go vet ./...
```

Also valid: `go test ./...`, `go vet ./...`, `make test` / `make vet`.

`go test -race ./...` (`make race`) is what CI runs. Locally it has hung in `internal/platform` (AppKit `dispatch_sync`). Run it when the change is concurrency-sensitive; if it does not finish, report `NOT RUN` or `INCOMPLETE` with the reason. Do not claim it passed.

`make app` when the bundle, plist, or packaging is in scope.

Rules:

- Never delete/disable tests to get green; do not weaken assertions unless behavior is intentionally changed
- Add a regression test for a bug you fix
- Prefer fake `mo` (`internal/molestatus/source_test.go` — `writeFakeMo`) over a real Mole install
- Unit tests do not start `systray.Run`. Tray, stay-open, login item, notifications, clipboard, and save dialog need a manual macOS run — say so; do not pretend coverage exists
- Distinguish test failure from missing CGO / SDK / not-Darwin

## macOS / Platform Rules

This is a **macOS menu-bar** app (`GOOS=darwin`, CGO, `LSMinimumSystemVersion` `11.0`). The repo does not claim Linux/Windows product support. `!darwin` stubs exist so some packages compile elsewhere.

- Systray + `stayopen_darwin.go` / `.m` require CGO and Xcode CLT
- Bundle: `make app` → `build/MoleBar.app` (`packaging/Info.plist`, `LSUIElement`, `packaging/MoleBar.icns`). Version is `__VERSION__` stamped at bundle time; linker `main.version` defaults to `"dev"`
- Signing only if `CODESIGN_IDENTITY` is set. **Notarization is not implemented**
- `make dist` requires universal (`UNIVERSAL=1`); CI release may fall back to native — do not document them as the same
- Changing a stay-open menu label requires updating the title set in `stayopen_darwin.m`

## Documentation Maintenance

Update `docs/` in the same change when a **documented contract** changes. Mapping: `docs/CONTRIBUTING.md` (Documentation Expectations).

Do not update docs for internal-only details that do not change those contracts. Do not describe unimplemented work as shipped.

## Working Tree Safety

Inspect `git status` / `git diff` before overlapping edits. Preserve unrelated user changes. Do not `git reset --hard`, force-push, or skip hooks unless the user explicitly asks.

## Never Assume

Always inspect this checkout. Do not assume:

- A file exists because another branch or zip had it
- Upstream Mole JSON/flags/units from memory
- A test or race run from an earlier session
- Identical macOS / AppKit / System Events behavior across OS versions
- CLI defaults or version strings from README or a git tag alone (`main.version`, plist, `git describe` can differ)
- Config semantics from the filename (`display_mode` is legacy; `config.json` is current)
- A prior chat’s refactor is already applied

## Completion Report

```markdown
## Changed Files
- `path` — what changed

## Validation
`command` — PASS | FAIL | NOT RUN: <reason>

## Behavior
User-visible or architectural change (or “none”).

## Documentation
Updated docs, or `No documented contract changed.`

## Remaining Issues
Task-relevant only.
```

Do not write “all good”, “production ready”, or “fully tested” unless the commands you ran support it.
