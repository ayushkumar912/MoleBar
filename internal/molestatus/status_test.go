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
	score, msg, ok := s.Health()
	if !ok || score != 92 || msg != "Excellent" {
		t.Fatalf("health = %d %q ok=%v", score, msg, ok)
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

func TestHealthMissingIsUnavailable(t *testing.T) {
	s, err := Parse([]byte(`{"cpu":{"usage":1}}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, ok := s.Health(); ok {
		t.Fatal("missing health_score should be unavailable")
	}
}

func TestCPUTemperature(t *testing.T) {
	s, err := Parse([]byte(`{"thermal":{"cpu_temp":58.5}}`))
	if err != nil {
		t.Fatal(err)
	}
	c, ok := s.CPUTemperature()
	if !ok || c != 58.5 {
		t.Fatalf("temp = %v ok=%v", c, ok)
	}
	s2, err := Parse([]byte(`{"thermal":{"cpu_temp":0}}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s2.CPUTemperature(); ok {
		t.Fatal("zero temp should be unavailable")
	}
}

func TestPrimaryBatteryInfoOptionalFields(t *testing.T) {
	s, err := Parse([]byte(baseJSON))
	if err != nil {
		t.Fatal(err)
	}
	info, ok := s.PrimaryBatteryInfo()
	if !ok || info.Health == nil || *info.Health != "Good" || info.CycleCount == nil || *info.CycleCount != 100 {
		t.Fatalf("info = %+v ok=%v", info, ok)
	}
	if info.Charging == nil || !*info.Charging {
		t.Fatalf("AC should be charging=%v", info.Charging)
	}
}

func TestTopCPUProcessesDeterministic(t *testing.T) {
	s, err := Parse([]byte(`{
		"top_processes": [
			{"pid": 3, "name": "chrome", "cpu": 10, "memory_bytes": 1},
			{"pid": 1, "name": "cursor", "cpu": 40, "memory_bytes": 2},
			{"pid": 2, "name": "aaa", "cpu": 40, "memory_bytes": 3}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	got := s.TopCPUProcesses(2)
	if len(got) != 2 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].Name != "aaa" || got[0].PID != 2 || got[1].Name != "cursor" {
		t.Fatalf("order = %#v", got)
	}
	max, ok := s.MaxProcessCPU()
	if !ok || max != 40 {
		t.Fatalf("max = %v ok=%v", max, ok)
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
