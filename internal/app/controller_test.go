package app

import (
	"testing"
	"time"

	"github.com/ayush-kumar912/molebar/internal/config"
	"github.com/ayush-kumar912/molebar/internal/molestatus"
)

type memStore struct {
	mode    config.DisplayMode
	ok      bool
	saves   int
	last    config.DisplayMode
	saveErr error
}

func (m *memStore) Load() (config.DisplayMode, bool, error) {
	return m.mode, m.ok, nil
}

func (m *memStore) Save(mode config.DisplayMode) error {
	m.saves++
	m.last = mode
	m.mode = mode
	m.ok = true
	return m.saveErr
}

func statusWithRates(rx, tx float64) *molestatus.Status {
	s := &molestatus.Status{}
	s.CPU.Usage = 10
	s.Memory.UsedPercent = 20
	s.Network = []molestatus.Network{{RxRateMBs: rx, TxRateMBs: tx}}
	return s
}

func TestControllerFirstSampleDoesNotAdd(t *testing.T) {
	clk := t0()
	c := New(Config{DisplayMode: config.DisplayModeSys}, &memStore{}, func() time.Time { return clk })
	c.OnResult(molestatus.Result{Status: statusWithRates(1, 1)})
	vm := c.View()
	if vm.Session != "Session: ↓0 B ↑0 B" {
		t.Fatalf("session = %q", vm.Session)
	}
}

func TestControllerFailureBreaksContinuity(t *testing.T) {
	clk := t0()
	now := func() time.Time { return clk }
	c := New(Config{DisplayMode: config.DisplayModeSys}, &memStore{}, now)
	c.OnResult(molestatus.Result{Status: statusWithRates(1, 1)})
	clk = clk.Add(time.Second)
	c.OnResult(molestatus.Result{Status: statusWithRates(1, 1)})
	clk = clk.Add(time.Second)
	c.OnResult(molestatus.Result{Err: errTest})
	if c.View().Title != "mo: err" {
		t.Fatalf("title = %q", c.View().Title)
	}
	clk = clk.Add(30 * time.Second)
	c.OnResult(molestatus.Result{Status: statusWithRates(8, 8)})
	if c.View().Session != "Session: ↓1.0 MB ↑1.0 MB" {
		t.Fatalf("session after failure re-prime = %q", c.View().Session)
	}
}

func TestControllerResetAndModePersist(t *testing.T) {
	store := &memStore{mode: config.DisplayModeSys, ok: true}
	clk := t0()
	c := New(Config{DisplayMode: config.DisplayModeSys}, store, func() time.Time { return clk })
	c.OnResult(molestatus.Result{Status: statusWithRates(1, 1)})
	clk = clk.Add(time.Second)
	c.OnResult(molestatus.Result{Status: statusWithRates(1, 1)})
	c.ResetSession()
	if c.View().Session != "Session: ↓0 B ↑0 B" {
		t.Fatalf("after reset %q", c.View().Session)
	}
	if store.saves != 0 {
		t.Fatalf("startup/reset must not persist, saves=%d", store.saves)
	}
	c.SetDisplayMode(config.DisplayModeNet)
	if store.saves != 1 || store.last != config.DisplayModeNet {
		t.Fatalf("menu change should persist: saves=%d last=%q", store.saves, store.last)
	}
	if c.Mode() != config.DisplayModeNet {
		t.Fatalf("mode = %q", c.Mode())
	}
}

var errTest = errString("sample failed")

type errString string

func (e errString) Error() string { return string(e) }

func t0() time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}
