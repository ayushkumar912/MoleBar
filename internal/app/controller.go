package app

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ayush-kumar912/molebar/internal/alerts"
	"github.com/ayush-kumar912/molebar/internal/config"
	"github.com/ayush-kumar912/molebar/internal/diagnostics"
	"github.com/ayush-kumar912/molebar/internal/history"
	"github.com/ayush-kumar912/molebar/internal/molestatus"
	"github.com/ayush-kumar912/molebar/internal/presentation"
	"github.com/ayush-kumar912/molebar/internal/session"
)

// Config is the runtime configuration assembled by the composition root.
type Config struct {
	Interval     time.Duration
	BinPath      string
	DisplayMode  config.DisplayMode
	Preferences  config.Preferences
	FetchTimeout time.Duration
	MaxGap       time.Duration
	Version      string
	OSName       string
	OSVersion    string
	Arch         string
}

// Controller is the single owner of mutable application state.
// Methods are intended to be called from one event loop; there is no mutex.
type Controller struct {
	cfg             Config
	store           config.Store
	meter           *session.Meter
	hist            *history.History
	engine          *alerts.Engine
	now             func() time.Time
	prefs           config.Preferences
	last            *molestatus.Status
	lastErr         error
	updated         time.Time
	strategy        string
	caps            molestatus.Capabilities
	launchAtLogin   bool
	launchSupported bool
}

// New constructs a controller. now may be nil (time.Now is used).
// Startup does not write the preference file.
func New(cfg Config, store config.Store, now func() time.Time) *Controller {
	if now == nil {
		now = time.Now
	}
	timeout := cfg.FetchTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	cfg.FetchTimeout = timeout
	prefs := cfg.Preferences
	if prefs.Version == 0 && len(prefs.Layout.Metrics) == 0 {
		prefs = config.DefaultPreferences()
		if cfg.DisplayMode != "" {
			prefs.ApplyDisplayMode(config.NormalizeDisplayMode(string(cfg.DisplayMode)))
		}
	} else {
		prefs = prefs.Normalize()
	}
	return &Controller{
		cfg:    cfg,
		store:  store,
		meter:  session.New(cfg.MaxGap),
		hist:   history.New(history.CapacityFor(cfg.Interval, history.DefaultWindow)),
		engine: alerts.NewEngine(rulesFromPrefs(prefs), 5*time.Minute),
		now:    now,
		prefs:  prefs,
	}
}

// OnResult applies a collection result: success updates status, history,
// session meter, and alerts; failure invalidates integration continuity
// and keeps last-good dropdown values.
func (c *Controller) OnResult(res molestatus.Result) []alerts.AlertEvent {
	if res.Strategy != "" {
		c.strategy = res.Strategy
	}
	if res.Err != nil {
		c.meter.Invalidate()
		c.lastErr = res.Err
		log.Printf("molebar: refresh failed: %v", res.Err)
		return nil
	}
	if res.Status == nil {
		if res.Strategy != "" {
			return nil
		}
		c.meter.Invalidate()
		c.lastErr = nil
		return nil
	}
	now := c.now()
	rx, tx := res.Status.TotalNetRates()
	c.meter.Observe(now, rx, tx)
	c.hist.Add(now, historyPoint(res.Status))
	c.last = res.Status
	c.lastErr = nil
	c.updated = now
	if !c.prefs.AlertsEnabled {
		return nil
	}
	return c.engine.Evaluate(now, metricValues(res.Status))
}

// Refresh performs a one-shot `mo status --json` fetch (manual "Refresh now").
func (c *Controller) Refresh(ctx context.Context) []alerts.AlertEvent {
	if ctx == nil {
		ctx = context.Background()
	}
	cctx, cancel := context.WithTimeout(ctx, c.cfg.FetchTimeout)
	defer cancel()
	st, err := molestatus.Fetch(cctx, c.cfg.BinPath)
	return c.OnResult(molestatus.Result{Status: st, Err: err})
}

// ResetSession zeros totals and sampling state. The next sample primes only.
func (c *Controller) ResetSession() {
	c.meter.Reset()
}

// SetDisplayMode updates the runtime layout from a legacy mode and persists it.
func (c *Controller) SetDisplayMode(mode config.DisplayMode) {
	c.prefs.ApplyDisplayMode(mode)
	c.persist()
}

// SetProfile applies a built-in profile and persists it.
func (c *Controller) SetProfile(id string) {
	if !c.prefs.ApplyProfile(id) {
		return
	}
	c.persist()
}

// ToggleMetric flips a tray metric and persists the custom layout.
func (c *Controller) ToggleMetric(m config.Metric) {
	c.prefs.ApplyMetricToggle(m)
	c.persist()
}

// SetAlertsEnabled toggles the rule engine and persists the preference.
func (c *Controller) SetAlertsEnabled(on bool) {
	c.prefs.AlertsEnabled = on
	if !on {
		c.engine.Reset()
	}
	c.persist()
}

// SetLaunchAtLoginPref stores the user preference; the OS adapter is
// invoked by the composition root.
func (c *Controller) SetLaunchAtLoginPref(on bool) {
	c.prefs.LaunchAtLogin = on
	c.launchAtLogin = on
	c.persist()
}

// SetLaunchAtLoginState reflects the current OS login-item state in the menu.
func (c *Controller) SetLaunchAtLoginState(on, supported bool) {
	c.launchAtLogin = on
	c.launchSupported = supported
}

// SetCapabilities records Mole capability detection.
func (c *Controller) SetCapabilities(caps molestatus.Capabilities) {
	c.caps = caps
	if c.strategy == "" {
		if caps.SupportsWatch {
			c.strategy = "watch"
		} else {
			c.strategy = "poll"
		}
	}
}

// Mode is the current runtime display mode approximation.
func (c *Controller) Mode() config.DisplayMode {
	return c.prefs.DisplayMode()
}

// Snapshot is the current domain state.
func (c *Controller) Snapshot() State {
	var firing []alerts.Alert
	if c.prefs.AlertsEnabled {
		firing = c.engine.Firing()
	}
	return State{
		Status:      c.last,
		Session:     c.meter.Snapshot(),
		History:     c.hist.Summarize(),
		Alerts:      firing,
		Profile:     c.prefs.Profile,
		Layout:      c.prefs.Layout,
		LastError:   c.lastErr,
		LastUpdated: c.updated,
		Strategy:    c.strategy,
		Caps:        c.caps,
	}
}

// View renders the current state. It does not mutate anything.
func (c *Controller) View() presentation.ViewModel {
	st := c.Snapshot()
	return presentation.Present(presentation.State{
		Layout:                 st.Layout,
		Mode:                   c.Mode(),
		Profile:                st.Profile,
		Status:                 st.Status,
		Session:                st.Session,
		Alerts:                 st.Alerts,
		AlertsEnabled:          c.prefs.AlertsEnabled,
		LaunchAtLogin:          c.launchAtLogin,
		LaunchAtLoginSupported: c.launchSupported,
		Updated:                st.LastUpdated,
		Err:                    st.LastError,
		Diag:                   c.diagInput(st),
	})
}

func (c *Controller) persist() {
	if c.store == nil {
		return
	}
	if err := c.store.Save(c.prefs); err != nil {
		log.Printf("molebar: failed to save preferences: %v", err)
	}
}

func (c *Controller) diagInput(st State) diagnostics.Input {
	moleVer := c.caps.Version
	return diagnostics.Input{
		MoleBarVersion: c.cfg.Version,
		MoleVersion:    moleVer,
		OSName:         c.cfg.OSName,
		OSVersion:      c.cfg.OSVersion,
		Arch:           c.cfg.Arch,
		Strategy:       st.Strategy,
		Profile:        st.Profile,
		Interval:       c.cfg.Interval,
		Capabilities:   c.caps,
		Status:         st.Status,
		Session:        st.Session,
		History:        st.History,
		LastError:      molestatus.ErrorCategory(st.LastError),
		GeneratedAt:    c.updated,
	}
}

func rulesFromPrefs(prefs config.Preferences) []alerts.Rule {
	if len(prefs.Alerts) == 0 {
		return alerts.DefaultRules()
	}
	out := make([]alerts.Rule, 0, len(prefs.Alerts))
	for i, p := range prefs.Alerts {
		d, err := config.ParseAlertDuration(p.For)
		if err != nil {
			continue
		}
		out = append(out, alerts.Rule{
			ID:       fmt.Sprintf("%s-%d", p.Metric, i),
			Metric:   alerts.Metric(p.Metric),
			Operator: alerts.Operator(p.Operator),
			Value:    p.Value,
			Duration: d,
		})
	}
	if len(out) == 0 {
		return alerts.DefaultRules()
	}
	return out
}

func metricValues(s *molestatus.Status) map[alerts.Metric]float64 {
	if s == nil {
		return nil
	}
	rx, tx := s.TotalNetRates()
	m := map[alerts.Metric]float64{
		alerts.MetricCPU:    s.CPU.Usage,
		alerts.MetricMemory: s.Memory.UsedPercent,
		alerts.MetricRX:     rx,
		alerts.MetricTX:     tx,
	}
	if pct := s.PrimaryDiskPercent(); pct >= 0 {
		m[alerts.MetricDisk] = pct
	}
	if temp, ok := s.CPUTemperature(); ok {
		m[alerts.MetricTemperature] = temp
	}
	if pct, _, ok := s.PrimaryBattery(); ok {
		m[alerts.MetricBattery] = pct
	}
	if cpu, ok := s.MaxProcessCPU(); ok {
		m[alerts.MetricProcessCPU] = cpu
	}
	return m
}

func historyPoint(s *molestatus.Status) history.Point {
	rx, tx := s.TotalNetRates()
	p := history.Point{CPU: s.CPU.Usage, Memory: s.Memory.UsedPercent, RX: rx, TX: tx}
	if temp, ok := s.CPUTemperature(); ok {
		t := temp
		p.Temperature = &t
	}
	if score, _, ok := s.Health(); ok {
		h := float64(score)
		p.Health = &h
	}
	return p
}
