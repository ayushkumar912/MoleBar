package main

import (
	"sync"

	"github.com/getlantern/systray"

	"github.com/ayush-kumar912/molebar/internal/config"
	"github.com/ayush-kumar912/molebar/internal/platform"
	"github.com/ayush-kumar912/molebar/internal/presentation"
)

var (
	lastTrayMu sync.Mutex
	lastTray   presentation.ViewModel
)

const (
	maxProcessRows = 5
	maxAlertRows   = 3
)

type menu struct {
	cpu, mem, swap, disk, temp, batt, health *systray.MenuItem
	down, up                                 *systray.MenuItem
	sessionRX, sessionTX, peakRX, peakTX     *systray.MenuItem
	session, sessionReset                    *systray.MenuItem
	procs                                    []*systray.MenuItem
	profiles                                 map[string]*systray.MenuItem
	metrics                                  map[config.Metric]*systray.MenuItem
	alertsToggle                             *systray.MenuItem
	alertRows                                []*systray.MenuItem
	launchAtLogin                            *systray.MenuItem
	updated                                  *systray.MenuItem
	refresh, copySummary, exportDiag, quit   *systray.MenuItem
}

func newMenu() *menu {
	m := &menu{
		profiles: make(map[string]*systray.MenuItem),
		metrics:  make(map[config.Metric]*systray.MenuItem),
	}
	m.cpu = systray.AddMenuItem("CPU: —", "")
	m.mem = systray.AddMenuItem("Memory: —", "")
	m.swap = systray.AddMenuItem("Swap: —", "")
	m.disk = systray.AddMenuItem("Disk: —", "")
	m.temp = systray.AddMenuItem("Temperature: —", "")
	m.batt = systray.AddMenuItem("Battery: —", "")
	m.health = systray.AddMenuItem("Health: —", "")
	systray.AddSeparator()
	m.down = systray.AddMenuItem("↓ —", "Current download rate")
	m.up = systray.AddMenuItem("↑ —", "Current upload rate")
	m.session = systray.AddMenuItem("Session: —", "Estimated data transferred since molebar launched")
	m.sessionRX = systray.AddMenuItem("Session RX: —", "")
	m.sessionTX = systray.AddMenuItem("Session TX: —", "")
	m.peakRX = systray.AddMenuItem("Peak RX: —", "")
	m.peakTX = systray.AddMenuItem("Peak TX: —", "")
	m.sessionReset = systray.AddMenuItem("Reset session totals", "Zero out the session download/upload counters")
	systray.AddSeparator()
	procs := systray.AddMenuItem("Top Processes", "Highest CPU processes reported by Mole")
	m.procs = make([]*systray.MenuItem, maxProcessRows)
	for i := 0; i < maxProcessRows; i++ {
		m.procs[i] = procs.AddSubMenuItem("—", "")
		m.procs[i].Disable()
	}
	systray.AddSeparator()
	profileMenu := systray.AddMenuItem("Profile", "Apply a tray-metric preset")
	for _, p := range config.BuiltInProfiles() {
		m.profiles[string(p.ID)] = profileMenu.AddSubMenuItem(p.Label, "Use the "+p.Label+" layout")
	}
	systray.AddSeparator()
	metricsHeader := systray.AddMenuItem("Tray Metrics", "Toggle several without closing the menu")
	metricsHeader.Disable()
	for _, metric := range config.AllMetrics() {
		m.metrics[metric] = systray.AddMenuItem(metric.Label(), "Toggle "+metric.Label())
	}
	m.alertsToggle = systray.AddMenuItem("Alerts", "Enable threshold alerts")
	m.alertRows = make([]*systray.MenuItem, maxAlertRows)
	for i := 0; i < maxAlertRows; i++ {
		m.alertRows[i] = systray.AddMenuItem("—", "")
		m.alertRows[i].Disable()
	}
	systray.AddSeparator()
	settings := systray.AddMenuItem("Settings", "")
	m.launchAtLogin = settings.AddSubMenuItem("Launch MoleBar at Login", "Open MoleBar when you log in")
	m.updated = systray.AddMenuItem("Updated: —", "")
	m.updated.Disable()
	systray.AddSeparator()
	m.refresh = systray.AddMenuItem("Refresh now", "Fetch mo status immediately")
	m.copySummary = systray.AddMenuItem("Copy System Summary", "Copy a short status summary")
	m.exportDiag = systray.AddMenuItem("Export Diagnostics...", "Write a diagnostics report")
	m.quit = systray.AddMenuItem("Quit", "Quit molebar")
	return m
}

func apply(m *menu, vm presentation.ViewModel) {
	lastTrayMu.Lock()
	lastTray = vm
	lastTrayMu.Unlock()
	applyItems(m, vm)
	if !platform.MenuIsTracking() {
		applyTray(vm)
	}
}

func flushTray() {
	lastTrayMu.Lock()
	vm := lastTray
	lastTrayMu.Unlock()
	applyTray(vm)
}

func applyTray(vm presentation.ViewModel) {
	systray.SetTitle(vm.Title)
	systray.SetTooltip(vm.Tooltip)
}

func applyItems(m *menu, vm presentation.ViewModel) {
	m.cpu.SetTitle(vm.CPU)
	m.mem.SetTitle(vm.Memory)
	m.swap.SetTitle(vm.Swap)
	m.disk.SetTitle(vm.Disk)
	m.temp.SetTitle(vm.Temperature)
	m.batt.SetTitle(vm.Battery)
	m.health.SetTitle(vm.Health)
	m.down.SetTitle(vm.Down)
	m.up.SetTitle(vm.Up)
	m.session.SetTitle(vm.Session)
	m.sessionRX.SetTitle(vm.SessionRX)
	m.sessionTX.SetTitle(vm.SessionTX)
	m.peakRX.SetTitle(vm.PeakRX)
	m.peakTX.SetTitle(vm.PeakTX)
	m.updated.SetTitle(vm.Updated)
	setChecked(m.alertsToggle, vm.AlertsEnabled)
	if vm.LaunchAtLoginSupported {
		m.launchAtLogin.Enable()
	} else {
		m.launchAtLogin.Disable()
	}
	setChecked(m.launchAtLogin, vm.LaunchAtLogin)
	for id, item := range m.profiles {
		checked := false
		for _, row := range vm.ProfileRows {
			if row.ID == id {
				checked = row.Checked
				break
			}
		}
		setChecked(item, checked)
	}
	for metric, item := range m.metrics {
		checked := false
		for _, row := range vm.MetricRows {
			if row.ID == string(metric) {
				checked = row.Checked
				break
			}
		}
		setChecked(item, checked)
	}
	for i, item := range m.procs {
		if i < len(vm.ProcessRows) {
			item.SetTitle(vm.ProcessRows[i].Title)
		} else {
			item.SetTitle("—")
		}
	}
	for i, item := range m.alertRows {
		if i < len(vm.AlertRows) {
			item.SetTitle(vm.AlertRows[i].Title)
		} else if i == 0 && len(vm.AlertRows) == 0 {
			item.SetTitle("No active alerts")
		} else {
			item.SetTitle("—")
		}
	}
	platform.SyncStayOpenMenu()
}

func setChecked(item *systray.MenuItem, on bool) {
	if item == nil {
		return
	}
	if on {
		item.Check()
	} else {
		item.Uncheck()
	}
}
