# Build and Release

## Local Binary Build

From the module root:

```sh
make build
# equivalent:
make build-native
```

`Makefile` — `build-native`:

```text
CGO_ENABLED=1 GOOS=darwin MACOSX_DEPLOYMENT_TARGET=11.0
go build -buildvcs=false -trimpath \
  -ldflags="-s -w -X main.version=$(VERSION)" \
  -o build/molebar ./cmd/molebar
```

`VERSION` ?= `git describe --tags --always --dirty` with leading `v` stripped, else `dev`.

```sh
go build -o build/molebar ./cmd/molebar
```

also works but does not apply the Makefile ldflags / `CGO_ENABLED=1` / deployment target unless the environment already has them.

`make build-universal` builds `molebar-arm64` and `molebar-amd64`, then `lipo -create` into `build/molebar`. The amd64 build sets `CC="clang -arch x86_64"` and optional `-isysroot $(SDKROOT)` (`xcrun --sdk macosx --show-sdk-path`). It prints `lipo -info` and `file`.

## macOS App Bundle

`make app` (default `make all`):

1. `build-universal` if `UNIVERSAL=1`, else `build-native`
2. `mkdir` `build/MoleBar.app/Contents/{MacOS,Resources}`
3. Copy `build/molebar` → `Contents/MacOS/molebar`
4. `sed "s|__VERSION__|$(VERSION)|g"` `packaging/Info.plist` → `Contents/Info.plist`
5. Copy `packaging/MoleBar.icns` → `Contents/Resources/MoleBar.icns`
6. `make sign`

Resulting tree:

```text
MoleBar.app/
└── Contents/
    ├── Info.plist
    ├── MacOS/
    │   └── molebar
    └── Resources/
        └── MoleBar.icns
```

`make run` builds the raw binary only and executes `./build/molebar` (not the `.app`).

## `packaging/Info.plist`

| Key | Value |
| --- | ----- |
| `CFBundleName` / `CFBundleDisplayName` | `MoleBar` |
| `CFBundleIdentifier` | `com.ayushkumar912.molebar` |
| `CFBundleExecutable` | `molebar` |
| `CFBundlePackageType` | `APPL` |
| `CFBundleIconFile` | `MoleBar.icns` |
| `CFBundleVersion` | `__VERSION__` (replaced at `make app`) |
| `CFBundleShortVersionString` | `__VERSION__` (same) |
| `LSUIElement` | `true` (no Dock / Cmd+Tab) |
| `LSMinimumSystemVersion` | `11.0` |
| `NSHighResolutionCapable` | `true` |

## Makefile

| Target | Behavior |
| ------ | -------- |
| `all` | `app` |
| `build` | `build-native` |
| `build-native` | Darwin CGO binary → `build/molebar` |
| `build-universal` | lipo arm64+amd64 → `build/molebar` |
| `app` | Bundle + sign; `UNIVERSAL=1` requires universal |
| `sign` | If `CODESIGN_IDENTITY` set: `codesign --force --options runtime --sign ... --timestamp` and verify; else print skip message |
| `run` | `build` then `./build/molebar` |
| `test` | `go test ./...` |
| `vet` | `go vet ./...` |
| `race` | `go test -race ./...` |
| `check` | `gofmt -l` must be empty; `go mod verify`; test; vet |
| `fmt` | `gofmt -l -w .` |
| `dist` | `app UNIVERSAL=1`; `ditto` zip; `git archive` tarball |
| `clean` | `rm -rf build dist` |

Variables: `APP_NAME=MoleBar`, `BIN_NAME=molebar`, `MACOSX_DEPLOYMENT_TARGET=11.0`, `SDKROOT`, `CODESIGN_IDENTITY` (empty by default), `VERSION`.

## CI

`.github/workflows/`:

| Workflow file | Workflow `name` | Trigger | Purpose | Important steps |
| ------------- | --------------- | ------- | ------- | --------------- |
| `ci.yml` | `ci` | `push` to `main`; `pull_request` | Verify | checkout; setup-go **1.21**; `gofmt -l`; `go mod verify`; `go test ./...`; `go vet ./...`; `go test -race ./...` |
| `build.yml` | `release` | `push` tags `v*` | Build and attach GitHub Release | checkout `fetch-depth: 0`; setup-go **1.21**; `VERSION=${GITHUB_REF_NAME#v}`; `go test ./...`; try `make app UNIVERSAL=1 VERSION=...` else `make app VERSION=...`; print `file` / `lipo` / PlistBuddy versions; optional `make sign` if `secrets.CODESIGN_IDENTITY` non-empty; `ditto` zip + `git archive` tarball; `upload-artifact` name `MoleBar`; `softprops/action-gh-release` with those two files, `generate_release_notes: true` |

`.github/dependabot.yml`: weekly `gomod` and `github-actions`.

CI does **not** run `make check` as a single target (it inlines the same fmt/mod/test/vet steps; release job skips fmt/vet/race).

## Release

Authoritative version is the **git tag** (example `v0.1.2` → `0.1.2`).

```sh
git tag v0.1.2
git push origin v0.1.2
```

That tag push runs `release`. Locally:

```sh
make dist                 # requires universal success
# or
make app VERSION=0.1.2
```

`make dist` writes:

- `dist/MoleBar-<VERSION>.app.zip` (`ditto -c -k --keepParent --norsrc --noextattr --noqtn`)
- `dist/molebar-<VERSION>.tar.gz` (`git archive` of `HEAD`, prefix `molebar-<VERSION>/`)

## Versioning

| File | Field | Current value/source |
| ---- | ----- | -------------------- |
| `cmd/molebar/main.go` | `var version` | Default `"dev"`; overridden by `-X main.version=$(VERSION)` |
| `Makefile` | `VERSION` | `git describe --tags --always --dirty` minus `v`, else `dev` |
| `packaging/Info.plist` | `CFBundleVersion` / `CFBundleShortVersionString` | Placeholder `__VERSION__` until `make app` |
| Built `Info.plist` | same keys | Stamped `VERSION` |
| `.github/workflows/build.yml` | `VERSION` env | Tag name with leading `v` stripped |
| `.github/workflows/ci.yml` | (none) | Does not stamp or release |
| `go.mod` | module / `go` | `github.com/ayush-kumar912/molebar`, `go 1.21` — not the app version |
| `internal/config/preferences.go` | `CurrentVersion` | `1` — **preference schema**, not the app version |

**Drift:** a dirty or untagged tree produces `git describe` values such as `0.1.2-4-g<hash>` (this inspection: tag `v0.1.2`, describe `v0.1.2-4-g10c6165` → `0.1.2-4-g10c6165`). Runtime `version` and plist match only if both were built with the same `VERSION`. A `go build` without Makefile ldflags stays `"dev"` while a previously stamped `.app` may differ.

## Signing / Notarization

**Signing exists** as an optional Makefile/`release` step. It runs only when `CODESIGN_IDENTITY` is non-empty:

```sh
make app CODESIGN_IDENTITY="Developer ID Application: Your Name (TEAMID)"
```

Flags: `--force --options runtime --sign ... --timestamp`, then `codesign --verify --verbose`.

**Notarization is not implemented.** There is no `notarytool` target, workflow step, or required secret. README states the same.

Local `make app` without the identity prints that codesign is skipped. The bundle is then unsigned.

## Release Artifacts

GitHub Actions `release` job produces:

1. `dist/MoleBar-${VERSION}.app.zip`
2. `dist/molebar-${VERSION}.tar.gz`

Uploaded as artifact `MoleBar` and attached to the GitHub Release for that tag.

The zip is whatever architecture `make app` produced (universal if that succeeded, else native). `make dist` locally does **not** fall back to native.

## Release Checklist

Based on current repo behavior:

- [ ] `go test ./...` (or `make test`)
- [ ] `go vet ./...` (CI `ci` job; not the release job)
- [ ] `gofmt -l .` clean (`make check` / `ci`)
- [ ] `go test -race ./...` (`ci` job; see [Testing](TESTING.md) for local hang risk)
- [ ] `make app` (and `UNIVERSAL=1` if you claim a universal binary; confirm with `lipo -info`)
- [ ] Plist `CFBundleShortVersionString` / `CFBundleVersion` match the intended tag (`PlistBuddy` as in `build.yml`)
- [ ] Linker version: binary built with `-X main.version=...` matching the tag
- [ ] Inspect `build/MoleBar.app` (`file`, `lipo`, icon present)
- [ ] Tag `v*` and push to trigger `release`, or run `make dist` locally
- [ ] If signing: set `CODESIGN_IDENTITY` (repo secret for CI)
- [ ] Confirm Release assets: `.app.zip` and source `.tar.gz`
