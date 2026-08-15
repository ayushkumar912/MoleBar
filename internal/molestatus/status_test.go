package molestatus

import (
	"errors"
	"testing"
)

const baseJSON = `{
  "collected_at": "2026-01-01T00:00:00Z",
  "health_score": 92,
  "health_score_msg": "Excellent",
  "cpu": {"usage": 12.2, "load1": 2.94, "per_core": [10, 20]},
  "memory": {"used_percent": 67.0, "total": 100, "used": 67, "available": 33, "swap_used": 25, "swap_total": 100},
  "disks": [{"mount": "/", "used_percent": 39.5, "total": 1000, "used": 395, "purgeable": 0}],
  "batteries": [{"percent": 80, "status": "AC", "health": "Good", "cycle_count": 100}],
  "network": [{"name": "en0", "rx_rate_mbs": 1.2, "tx_rate_mbs": 0.3, "ip": "10.0.0.2"}],
  "procs": 400
}`

func TestParseSchema(t *testing.T) {
	s, err := Parse([]byte(baseJSON))
	if err != nil {
		t.Fatal(err)
	}
	if s.HealthScore != 92 || s.HealthMsg != "Excellent" {
		t.Fatalf("health = %d %q", s.HealthScore, s.HealthMsg)
	}
	if s.CPU.Usage != 12.2 || s.CPU.Load1 != 2.94 {
		t.Fatalf("cpu = %+v", s.CPU)
	}
	if s.Memory.UsedPercent != 67 {
		t.Fatalf("mem = %v", s.Memory.UsedPercent)
	}
	if s.SwapPercent() != 25 {
		t.Fatalf("swap = %v", s.SwapPercent())
	}
}

func TestFractionalBatteryPercentage(t *testing.T) {
	s, err := Parse([]byte(`{
		"batteries": [{"percent": 87.6, "status": "Battery"}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	pct, status, ok := s.PrimaryBattery()
	if !ok || status != "Battery" {
		t.Fatalf("ok=%v status=%q", ok, status)
	}
	if pct != 87.6 {
		t.Fatalf("percent = %v, want 87.6 (must not truncate)", pct)
	}
}

func TestPrimaryDiskNotFirstEntry(t *testing.T) {
	s, err := Parse([]byte(`{
		"disks": [
			{"mount": "/Volumes/Data", "used_percent": 10},
			{"mount": "/", "used_percent": 42.5}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := s.PrimaryDiskPercent(); got != 42.5 {
		t.Fatalf("PrimaryDiskPercent = %v, want 42.5", got)
	}
}

func TestPrimaryDiskMissingRoot(t *testing.T) {
	s, err := Parse([]byte(`{
		"disks": [{"mount": "/Volumes/Data", "used_percent": 10}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := s.PrimaryDiskPercent(); got != -1 {
		t.Fatalf("missing root should be -1, got %v", got)
	}
}

func TestMultipleNetworkEntries(t *testing.T) {
	s, err := Parse([]byte(`{
		"network": [
			{"name": "en0", "rx_rate_mbs": 1.0, "tx_rate_mbs": 0.2},
			{"name": "utun0", "rx_rate_mbs": 0.5, "tx_rate_mbs": 0.1}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	rx, tx := s.TotalNetRates()
	if rx != 1.5 || tx < 0.299 || tx > 0.301 {
		t.Fatalf("rates = %v/%v", rx, tx)
	}
}

func TestEmptyOptionalArrays(t *testing.T) {
	s, err := Parse([]byte(`{"disks": [], "batteries": [], "network": []}`))
	if err != nil {
		t.Fatal(err)
	}
	if s.PrimaryDiskPercent() != -1 {
		t.Fatal("empty disks")
	}
	if _, _, ok := s.PrimaryBattery(); ok {
		t.Fatal("empty batteries")
	}
	rx, tx := s.TotalNetRates()
	if rx != 0 || tx != 0 {
		t.Fatalf("empty network rates %v/%v", rx, tx)
	}
}

func TestMalformedJSON(t *testing.T) {
	_, err := Parse([]byte(`{"health_score":`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrMalformedJSON) {
		t.Fatalf("err = %v, want ErrMalformedJSON", err)
	}
}
