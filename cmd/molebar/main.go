// Command molebar is a macOS menu-bar widget that reads Mole
// (`mo status --watch` or `mo status --json`, https://github.com/tw93/Mole)
// and shows CPU, memory, swap, disk, battery, and health-score in the menu bar.
//
// Requires the `mo` CLI to be installed and on $PATH:
//
//	brew install mole
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/getlantern/systray"

	"github.com/ayush-kumar912/molebar/internal/app"
	"github.com/ayush-kumar912/molebar/internal/config"
	"github.com/ayush-kumar912/molebar/internal/molestatus"
	"github.com/ayush-kumar912/molebar/internal/presentation"
)

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
	titleMode := fs.String("title", "", `what to show in the menu bar title: "sys" (CPU/MEM), "net" (↓/↑ rates), or "both"`)
	if err := fs.Parse(args); err != nil {
		return app.Config{}, err
	}
	if err := validateInterval(*interval); err != nil {
		return app.Config{}, err
	}
	return app.Config{
		Interval:    *interval,
		BinPath:     molestatus.ResolveBinary(*binPath),
		DisplayMode: config.ResolveDisplayMode(store, *titleMode),
	}, nil
}

func validateInterval(d time.Duration) error {
	if d <= 0 {
		return fmt.Errorf("-interval must be greater than 0 (got %v)", d)
	}
	return nil
}

type menu struct {
	cpu, mem, swap, disk, batt                   *systray.MenuItem
	down, up, session, sessionReset              *systray.MenuItem
	display, displaySys, displayNet, displayBoth *systray.MenuItem
	health, updated                              *systray.MenuItem
	refresh, quit                                *systray.MenuItem
}

func newMenu() *menu {
	m := &menu{}
	m.cpu = systray.AddMenuItem("CPU: —", "")
	m.mem = systray.AddMenuItem("Memory: —", "")
	m.swap = systray.AddMenuItem("Swap: —", "")
	m.disk = systray.AddMenuItem("Disk: —", "")
	m.batt = systray.AddMenuItem("Battery: —", "")
	systray.AddSeparator()
	m.down = systray.AddMenuItem("↓ —", "Current download rate")
	m.up = systray.AddMenuItem("↑ —", "Current upload rate")
	m.session = systray.AddMenuItem("Session: —", "Estimated data transferred since molebar launched")
	m.sessionReset = systray.AddMenuItem("Reset session totals", "Zero out the session download/upload counters")
	systray.AddSeparator()
	m.display = systray.AddMenuItem("Display: System", "Choose what to show in the menu bar")
	m.displaySys = m.display.AddSubMenuItem("System", "Show CPU and memory")
	m.displayNet = m.display.AddSubMenuItem("Network", "Show download/upload speed")
	m.displayBoth = m.display.AddSubMenuItem("Both", "Show CPU, memory, and network")
	systray.AddSeparator()
	m.health = systray.AddMenuItem("Health: —", "")
	m.updated = systray.AddMenuItem("Updated: —", "")
	m.updated.Disable()
	systray.AddSeparator()
	m.refresh = systray.AddMenuItem("Refresh now", "Fetch mo status immediately")
	m.quit = systray.AddMenuItem("Quit", "Quit molebar")
	return m
}

func apply(m *menu, vm presentation.ViewModel) {
	systray.SetTitle(vm.Title)
	systray.SetTooltip(vm.Tooltip)
	m.cpu.SetTitle(vm.CPU)
	m.mem.SetTitle(vm.Memory)
	m.swap.SetTitle(vm.Swap)
	m.disk.SetTitle(vm.Disk)
	m.batt.SetTitle(vm.Battery)
	m.down.SetTitle(vm.Down)
	m.up.SetTitle(vm.Up)
	m.session.SetTitle(vm.Session)
	m.display.SetTitle(vm.DisplayLabel)
	m.health.SetTitle(vm.Health)
	m.updated.SetTitle(vm.Updated)
	setChecked(m.displaySys, vm.ModeSys)
	setChecked(m.displayNet, vm.ModeNet)
	setChecked(m.displayBoth, vm.ModeBoth)
}

func setChecked(item *systray.MenuItem, on bool) {
	if on {
		item.Check()
	} else {
		item.Uncheck()
	}
}

func onReady(ctx context.Context, cancel context.CancelFunc, cfg app.Config, store config.Store) {
	ctrl := app.New(cfg, store, nil)
	m := newMenu()
	apply(m, ctrl.View())

	src := molestatus.NewSource(molestatus.Options{
		Bin:      cfg.BinPath,
		Interval: cfg.Interval,
	})
	updates := make(chan molestatus.Result, 4)
	go src.Run(ctx, func(r molestatus.Result) {
		select {
		case updates <- r:
		case <-ctx.Done():
		}
	})

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case r := <-updates:
				ctrl.OnResult(r)
				apply(m, ctrl.View())
			case <-m.refresh.ClickedCh:
				ctrl.Refresh(ctx)
				apply(m, ctrl.View())
			case <-m.sessionReset.ClickedCh:
				ctrl.ResetSession()
				apply(m, ctrl.View())
			case <-m.displaySys.ClickedCh:
				ctrl.SetDisplayMode(config.DisplayModeSys)
				apply(m, ctrl.View())
			case <-m.displayNet.ClickedCh:
				ctrl.SetDisplayMode(config.DisplayModeNet)
				apply(m, ctrl.View())
			case <-m.displayBoth.ClickedCh:
				ctrl.SetDisplayMode(config.DisplayModeBoth)
				apply(m, ctrl.View())
			case <-m.quit.ClickedCh:
				cancel()
				systray.Quit()
				return
			}
		}
	}()
}
