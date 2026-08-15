// Package app owns MoleBar application-level state transitions.
package app

import (
	"context"
	"log"
	"time"

	"github.com/ayush-kumar912/molebar/internal/config"
	"github.com/ayush-kumar912/molebar/internal/molestatus"
	"github.com/ayush-kumar912/molebar/internal/presentation"
	"github.com/ayush-kumar912/molebar/internal/session"
)

// Config is the runtime configuration assembled by the composition root.
type Config struct {
	Interval     time.Duration
	BinPath      string
	DisplayMode  config.DisplayMode
	FetchTimeout time.Duration
	MaxGap       time.Duration
}

// Controller is the single owner of mutable application state.
// Methods are intended to be called from one event loop; there is no mutex.
type Controller struct {
	cfg     Config
	store   config.Store
	meter   *session.Meter
	now     func() time.Time
	mode    config.DisplayMode
	last    *molestatus.Status
	lastErr error
	updated time.Time
}

// New constructs a controller. now may be nil (time.Now is used).
// Startup does not write the display-mode preference.
func New(cfg Config, store config.Store, now func() time.Time) *Controller {
	if now == nil {
		now = time.Now
	}
	timeout := cfg.FetchTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	cfg.FetchTimeout = timeout
	return &Controller{
		cfg:   cfg,
		store: store,
		meter: session.New(cfg.MaxGap),
		now:   now,
		mode:  config.NormalizeDisplayMode(string(cfg.DisplayMode)),
	}
}

// OnResult applies a collection result: success updates status and the
// session meter; failure invalidates integration continuity and keeps
// last-good dropdown values.
func (c *Controller) OnResult(res molestatus.Result) {
	if res.Err != nil {
		c.meter.Invalidate()
		c.lastErr = res.Err
		log.Printf("molebar: refresh failed: %v", res.Err)
		return
	}
	if res.Status == nil {
		c.meter.Invalidate()
		c.lastErr = nil
		return
	}
	now := c.now()
	rx, tx := res.Status.TotalNetRates()
	c.meter.Observe(now, rx, tx)
	c.last = res.Status
	c.lastErr = nil
	c.updated = now
}

// Refresh performs a one-shot `mo status --json` fetch (manual "Refresh now").
func (c *Controller) Refresh(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	cctx, cancel := context.WithTimeout(ctx, c.cfg.FetchTimeout)
	defer cancel()
	st, err := molestatus.Fetch(cctx, c.cfg.BinPath)
	c.OnResult(molestatus.Result{Status: st, Err: err})
}

// ResetSession zeros totals and sampling state. The next sample primes only.
func (c *Controller) ResetSession() {
	c.meter.Reset()
}

// SetDisplayMode updates the runtime mode and persists it as a user preference.
func (c *Controller) SetDisplayMode(mode config.DisplayMode) {
	c.mode = config.NormalizeDisplayMode(string(mode))
	if c.store == nil {
		return
	}
	if err := c.store.Save(c.mode); err != nil {
		log.Printf("molebar: failed to save display mode: %v", err)
	}
}

// Mode is the current runtime display mode.
func (c *Controller) Mode() config.DisplayMode {
	return c.mode
}

// View renders the current state. It does not mutate anything.
func (c *Controller) View() presentation.ViewModel {
	rx, tx := c.meter.Totals()
	return presentation.Present(presentation.State{
		Mode:      c.mode,
		Status:    c.last,
		SessionRx: rx,
		SessionTx: tx,
		Updated:   c.updated,
		Err:       c.lastErr,
	})
}
