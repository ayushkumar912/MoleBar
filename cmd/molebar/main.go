// Command molebar is a macOS menu-bar widget that reads Mole
// (`mo status --watch` or `mo status --json`, https://github.com/tw93/Mole)
// and shows CPU, memory, swap, disk, battery, health, and related metrics.
//
// Requires the `mo` CLI to be installed and on $PATH:
//
//	brew install mole
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/getlantern/systray"

	"github.com/ayush-kumar912/molebar/internal/alerts"
	"github.com/ayush-kumar912/molebar/internal/app"
	"github.com/ayush-kumar912/molebar/internal/config"
	"github.com/ayush-kumar912/molebar/internal/molestatus"
	"github.com/ayush-kumar912/molebar/internal/platform"
	"github.com/ayush-kumar912/molebar/internal/presentation"
)

// version is stamped by the linker from the git tag when building a release.
var version = "dev"

func main() {
	store := config.NewFileStore("")
	cfg, err := parseRuntime(flag.CommandLine, os.Args[1:], store)
	if err != nil {
		fmt.Fprintf(os.Stderr, "molebar: %v\n", err)
		os.Exit(2)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	systray.Run(func() { onReady(ctx, cancel, cfg, store) }, cancel)
}

func parseRuntime(fs *flag.FlagSet, args []string, store config.Store) (app.Config, error) {
	interval := fs.Duration("interval", 5*time.Second, "refresh interval")
	binPath := fs.String("mo-bin", "", `path to the "mo" executable (default: resolve "mo" from $PATH)`)
	titleMode := fs.String("title", "", `what to show in the menu bar title: "sys" (CPU/RAM), "net" (↓/↑ rates), or "both"`)
	if err := fs.Parse(args); err != nil {
		return app.Config{}, err
	}
	if err := validateInterval(*interval); err != nil {
		return app.Config{}, err
	}
	osName, osVersion, arch := platformInfo()
	prefs := config.ResolvePreferences(store, *titleMode)
	return app.Config{
		Interval:    *interval,
		BinPath:     molestatus.ResolveBinary(*binPath),
		DisplayMode: prefs.DisplayMode(),
		Preferences: prefs,
		Version:     version,
		OSName:      osName,
		OSVersion:   osVersion,
		Arch:        arch,
	}, nil
}

func validateInterval(d time.Duration) error {
	if d <= 0 {
		return fmt.Errorf("-interval must be greater than 0 (got %v)", d)
	}
	return nil
}

func platformInfo() (osName, osVersion, arch string) {
	osName = runtime.GOOS
	arch = runtime.GOARCH
	if runtime.GOOS == "darwin" {
		cmd := exec.Command("sw_vers", "-productVersion")
		if out, err := cmd.Output(); err == nil {
			osVersion = strings.TrimSpace(string(out))
			return osName, osVersion, arch
		}
	}
	return osName, "", arch
}

func onReady(ctx context.Context, cancel context.CancelFunc, cfg app.Config, store config.Store) {
	ctrl := app.New(cfg, store, nil)
	login := platform.NewDarwinLoginItem("")
	syncLoginState(ctrl, login)

	detectCtx, detectCancel := context.WithTimeout(ctx, 3*time.Second)
	caps, err := molestatus.Detect(detectCtx, cfg.BinPath)
	detectCancel()
	if err != nil {
		log.Printf("molebar: capability detection: %v", err)
	}
	ctrl.SetCapabilities(caps)

	m := newMenu()
	platform.KeepMenuOpenOnToggles()
	platform.SetMenuClosedHandler(flushTray)
	apply(m, ctrl.View())

	opts := molestatus.Options{
		Bin:      cfg.BinPath,
		Interval: cfg.Interval,
	}
	if err == nil {
		opts.Caps = &caps
	}
	src := molestatus.NewSource(opts)
	updates := make(chan molestatus.Result, 4)
	go src.Run(ctx, func(r molestatus.Result) {
		select {
		case updates <- r:
		case <-ctx.Done():
		}
	})

	notifyCh := make(chan platform.Notification, 4)
	go runNotifier(ctx, platform.DarwinNotifier{}, notifyCh)
	clip := platform.PBCopy{}
	save := platform.DarwinSaveDialog{}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case r := <-updates:
				enqueueNotify(notifyCh, ctrl.OnResult(r))
				apply(m, ctrl.View())
			case <-m.refresh.ClickedCh:
				enqueueNotify(notifyCh, ctrl.Refresh(ctx))
				apply(m, ctrl.View())
			case <-m.sessionReset.ClickedCh:
				ctrl.ResetSession()
				apply(m, ctrl.View())
			case <-m.alertsToggle.ClickedCh:
				ctrl.SetAlertsEnabled(!ctrl.View().AlertsEnabled)
				apply(m, ctrl.View())
			case <-m.launchAtLogin.ClickedCh:
				toggleLogin(ctrl, login)
				apply(m, ctrl.View())
			case <-m.copySummary.ClickedCh:
				copySummary(ctx, clip, ctrl.View())
			case <-m.exportDiag.ClickedCh:
				exportDiagnostics(ctx, save, ctrl.View())
			case <-m.profiles[string(config.ProfileMinimal)].ClickedCh:
				ctrl.SetProfile(string(config.ProfileMinimal))
				apply(m, ctrl.View())
			case <-m.profiles[string(config.ProfileDeveloper)].ClickedCh:
				ctrl.SetProfile(string(config.ProfileDeveloper))
				apply(m, ctrl.View())
			case <-m.profiles[string(config.ProfileNetwork)].ClickedCh:
				ctrl.SetProfile(string(config.ProfileNetwork))
				apply(m, ctrl.View())
			case <-m.profiles[string(config.ProfileBattery)].ClickedCh:
				ctrl.SetProfile(string(config.ProfileBattery))
				apply(m, ctrl.View())
			case <-m.profiles[string(config.ProfileFull)].ClickedCh:
				ctrl.SetProfile(string(config.ProfileFull))
				apply(m, ctrl.View())
			case <-m.metrics[config.MetricCPU].ClickedCh:
				ctrl.ToggleMetric(config.MetricCPU)
				apply(m, ctrl.View())
			case <-m.metrics[config.MetricMemory].ClickedCh:
				ctrl.ToggleMetric(config.MetricMemory)
				apply(m, ctrl.View())
			case <-m.metrics[config.MetricRX].ClickedCh:
				ctrl.ToggleMetric(config.MetricRX)
				apply(m, ctrl.View())
			case <-m.metrics[config.MetricTX].ClickedCh:
				ctrl.ToggleMetric(config.MetricTX)
				apply(m, ctrl.View())
			case <-m.metrics[config.MetricHealth].ClickedCh:
				ctrl.ToggleMetric(config.MetricHealth)
				apply(m, ctrl.View())
			case <-m.metrics[config.MetricBattery].ClickedCh:
				ctrl.ToggleMetric(config.MetricBattery)
				apply(m, ctrl.View())
			case <-m.metrics[config.MetricTemperature].ClickedCh:
				ctrl.ToggleMetric(config.MetricTemperature)
				apply(m, ctrl.View())
			case <-m.metrics[config.MetricDisk].ClickedCh:
				ctrl.ToggleMetric(config.MetricDisk)
				apply(m, ctrl.View())
			case <-m.quit.ClickedCh:
				cancel()
				systray.Quit()
				return
			}
		}
	}()
}

func runNotifier(ctx context.Context, n platform.Notifier, ch <-chan platform.Notification) {
	for {
		select {
		case <-ctx.Done():
			return
		case note := <-ch:
			nctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			_ = n.Notify(nctx, note)
			cancel()
		}
	}
}

func enqueueNotify(ch chan<- platform.Notification, events []alerts.AlertEvent) {
	for _, ev := range events {
		if ev.State != alerts.StateFiring && ev.State != alerts.StateRecovered {
			continue
		}
		n := alerts.NotificationFromEvent(ev)
		select {
		case ch <- platform.Notification{Title: n.Title, Body: n.Body}:
		default:
		}
	}
}

func syncLoginState(ctrl *app.Controller, login platform.LoginItemManager) {
	on, err := login.Enabled()
	supported := !errors.Is(err, platform.ErrUnsupported)
	if err != nil {
		on = false
	}
	ctrl.SetLaunchAtLoginState(on, supported)
}

func toggleLogin(ctrl *app.Controller, login platform.LoginItemManager) {
	cur, err := login.Enabled()
	if errors.Is(err, platform.ErrUnsupported) {
		ctrl.SetLaunchAtLoginState(false, false)
		return
	}
	want := !cur
	if err := login.SetEnabled(want); err != nil {
		log.Printf("molebar: launch at login: %v", err)
	} else {
		ctrl.SetLaunchAtLoginPref(want)
	}
	on, err := login.Enabled()
	ctrl.SetLaunchAtLoginState(on && err == nil, !errors.Is(err, platform.ErrUnsupported))
}

func copySummary(ctx context.Context, clip platform.Clipboard, vm presentation.ViewModel) {
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := clip.Copy(cctx, vm.SystemSummary); err != nil {
		log.Printf("molebar: copy summary: %v", err)
	}
}

func exportDiagnostics(ctx context.Context, chooser platform.SavePathChooser, vm presentation.ViewModel) {
	path, err := chooser.Choose(ctx, "molebar-diagnostics.txt")
	if err != nil {
		if errors.Is(err, platform.ErrUnsupported) {
			path = platform.DefaultDiagnosticsPath()
		} else {
			log.Printf("molebar: export dialog: %v", err)
			return
		}
	}
	if err := platform.WriteDiagnostics(path, vm.Diagnostics); err != nil {
		log.Printf("molebar: export diagnostics: %v", err)
	}
}
