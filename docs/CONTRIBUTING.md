# Contributing

Internal notes for changing this repository. There is no CLA, issue template, or review SLA in-tree.

## Before Editing

- Read the package you are changing and its `*_test.go`.
- Search call sites (`Controller`, `Present`, `Fetch`, `ResolveBinary`, menu `ClickedCh`).
- Read [Mole Integration](MOLE_INTEGRATION.md) before touching JSON fields, `mo` flags, or capability detection.
- Read [Architecture](ARCHITECTURE.md) before changing ownership of state or adding goroutines.

## Scope Discipline

- Do not mix unrelated refactors with a behavioral change.
- Preserve external behavior (CLI flags, config path/schema, tray strings, Mole commands) unless the change is intentional.
- Keep Darwin-only code behind `//go:build darwin` with a `!darwin` stub when a symbol is referenced from portable code.
- Do not add dependencies without a concrete need. The only direct module requirement today is `github.com/getlantern/systray`.

## Coding Expectations

Derived from the current tree:

- Run `gofmt` (`make fmt` / `make check`).
- Return wrapped sentinel errors (`fmt.Errorf("%w", Err…)`) so `ErrorCategory()` and `errors.Is` keep working.
- Pass `context.Context` into subprocesses (`CommandContext`, timeouts in `Fetch` / `Detect` / platform helpers).
- Tests should be deterministic: inject clocks (`Controller.now`, `Meter.Observe(at)`, `Engine.Evaluate(at)`); do not sleep to express time except in process-timeout tests.
- Bound long-running state (`history` capacity, watch restart cap, channel buffers).
- Do not leak Mole subprocesses: keep `configureProcessGroup` / `killProcessGroup` on cancel.
- Do not call `Controller` methods from a second goroutine; there is no mutex.
- Presenter stays a pure function of `presentation.State`.
- Do not persist CLI `-title`. Do not delete `display_mode`.
- Diagnostics must not grow env/IP/command-line/home-path output.

## Testing Expectations

- Add or extend tests in the same package when changing parse, meter, config, alerts, or presenter behavior.
- Prefer fake `mo` scripts (`molestatus/source_test.go`) over a real Mole install.
- `go test ./...` and `go vet ./...` should pass. `make check` is what CI’s fmt/mod/test/vet amount to.
- Do not assume `go test -race ./...` is clean locally; see [Testing](TESTING.md).
- Tray/AppKit/osascript paths still need a manual macOS run.

## Documentation Expectations

If you change a behavioral contract, update the matching developer doc in the same change:

| Contract | Doc |
| -------- | --- |
| Flags, interval, run/build loop | `docs/DEVELOPMENT.md` |
| Packages, data flow, concurrency | `docs/ARCHITECTURE.md` |
| `mo` commands, JSON fields, timeouts | `docs/MOLE_INTEGRATION.md` |
| Config path, precedence, profiles | `docs/CONFIGURATION.md` |
| Tests / coverage | `docs/TESTING.md` |
| Make/CI/release/version | `docs/BUILD_AND_RELEASE.md` |
| Failure symptoms | `docs/TROUBLESHOOTING.md` |

Do not document unimplemented work as if it shipped.

## Pull Request Checklist

- [ ] Behavior change is intentional and covered by tests where the logic is unit-testable
- [ ] `gofmt` clean; `go test ./...`; `go vet ./...`
- [ ] No new Mole subcommands or host collectors unless that is the point of the PR
- [ ] Config compatibility: legacy `display_mode` still readable; `-title` still not saved
- [ ] Darwin stay-open titles updated in `stayopen_darwin.m` if menu labels change
- [ ] Developer docs updated for contract changes
- [ ] Version/plist/ldflags left consistent if the PR is a release
