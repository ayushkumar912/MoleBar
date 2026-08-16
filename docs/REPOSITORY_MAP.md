# Repository Map

Annotated tree of tracked sources. Omits `.git`, `build/`, `dist/`, and other generated output.

```text
.
├── cmd/molebar/
│   ├── main.go              # flags, systray.Run, onReady event loop
│   ├── main_test.go         # interval + parseRuntime persistence
│   ├── menu.go              # systray items, apply / flushTray
│   └── molebar-icon.png     # README image only (not the status-item icon)
│
├── internal/
│   ├── app/
│   │   ├── controller.go    # owns status, meter, history, alerts, prefs
│   │   ├── controller_test.go
│   │   └── state.go         # read-only snapshot
│   ├── alerts/
│   │   ├── engine.go        # sustained-threshold state machine
│   │   ├── engine_test.go
│   │   ├── rule.go          # Rule / DefaultRules
│   │   └── notification.go  # NotificationFromEvent (no I/O)
│   ├── config/
│   │   ├── config.go        # Store, ResolvePreferences
│   │   ├── preferences.go   # JSON schema, defaults
│   │   ├── file_store.go    # config.json + legacy display_mode
│   │   ├── display_mode.go  # sys / net / both
│   │   ├── layout.go        # TrayLayout
│   │   ├── metric.go        # tray metric IDs
│   │   ├── profile.go       # built-in presets
│   │   └── *_test.go
│   ├── diagnostics/
│   │   ├── report.go        # text report + clipboard summary
│   │   └── report_test.go
│   ├── history/
│   │   ├── history.go       # bounded in-memory series
│   │   ├── ring.go          # circular buffer
│   │   └── *_test.go
│   ├── molestatus/
│   │   ├── model.go         # Status JSON subset
│   │   ├── status.go        # Parse + domain helpers
│   │   ├── exec.go          # ResolveBinary, Fetch, errors
│   │   ├── capabilities.go  # Detect help/version
│   │   ├── source.go        # watch-then-poll Source
│   │   ├── watch_source.go
│   │   ├── polling_source.go
│   │   ├── proc_unix.go     # process-group kill
│   │   ├── proc_other.go
│   │   └── *_test.go
│   ├── platform/
│   │   ├── stayopen.go / stayopen_darwin.go / stayopen_darwin.m
│   │   ├── loginitem*.go    # System Events login item
│   │   ├── notification*.go # osascript notification
│   │   ├── clipboard*.go    # pbcopy
│   │   ├── savedialog*.go   # export path picker
│   │   ├── export.go        # atomic diagnostics write
│   │   ├── applescript.go
│   │   └── *_test.go
│   ├── presentation/
│   │   ├── model.go         # ViewModel
│   │   ├── presenter.go     # Present()
│   │   └── presenter_test.go
│   └── session/
│       ├── meter.go         # RX/TX integration
│       └── meter_test.go
│
├── packaging/
│   ├── Info.plist           # bundle id, LSUIElement, __VERSION__
│   └── MoleBar.icns         # Finder / Launchpad icon
│
├── .github/
│   ├── workflows/ci.yml     # fmt, mod verify, test, vet, race
│   ├── workflows/build.yml  # tag v* → .app.zip + tarball release
│   └── dependabot.yml
│
├── Formula/
│   └── molebar.rb           # Homebrew tap formula (source tarball + make app)
├── scripts/
│   └── update-homebrew-formula.sh  # refresh formula url/sha256 after a release
│
├── docs/                    # this developer documentation set
├── Makefile
├── go.mod / go.sum
├── README.md
├── LICENSE                  # Apache 2.0
└── .gitignore               # /build /dist *.app *.zip ...
```

See [Architecture](ARCHITECTURE.md) for runtime relationships.
