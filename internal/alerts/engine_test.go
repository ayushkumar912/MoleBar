package alerts

import (
	"testing"
	"time"
)

func t0() time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}

func cpuRule(d time.Duration) Rule {
	return Rule{ID: "cpu", Metric: MetricCPU, Operator: OpGT, Value: 90, Duration: d}
}

func TestBelowThresholdStaysInactive(t *testing.T) {
	e := NewEngine([]Rule{cpuRule(30 * time.Second)}, 0)
	ev := e.Evaluate(t0(), map[Metric]float64{MetricCPU: 10})
	if len(ev) != 0 {
		t.Fatalf("events = %#v", ev)
	}
	if len(e.Firing()) != 0 {
		t.Fatal("firing")
	}
}

func TestThresholdCrossedBrieflyDoesNotFire(t *testing.T) {
	e := NewEngine([]Rule{cpuRule(30 * time.Second)}, 0)
	if ev := e.Evaluate(t0(), map[Metric]float64{MetricCPU: 95}); len(ev) != 0 {
		t.Fatalf("first spike fired: %#v", ev)
	}
	if ev := e.Evaluate(t0().Add(time.Second), map[Metric]float64{MetricCPU: 10}); len(ev) != 0 {
		t.Fatalf("recovery from pending fired: %#v", ev)
	}
	if ev := e.Evaluate(t0().Add(40*time.Second), map[Metric]float64{MetricCPU: 10}); len(ev) != 0 {
		t.Fatalf("still inactive: %#v", ev)
	}
}

func TestSustainedThresholdFires(t *testing.T) {
	e := NewEngine([]Rule{cpuRule(30 * time.Second)}, 0)
	e.Evaluate(t0(), map[Metric]float64{MetricCPU: 95})
	ev := e.Evaluate(t0().Add(30*time.Second), map[Metric]float64{MetricCPU: 96})
	if len(ev) != 1 || ev[0].State != StateFiring {
		t.Fatalf("events = %#v", ev)
	}
	if len(e.Firing()) != 1 {
		t.Fatal("expected firing")
	}
}

func TestZeroDurationStillNeedsTwoSamples(t *testing.T) {
	e := NewEngine([]Rule{cpuRule(0)}, 0)
	if ev := e.Evaluate(t0(), map[Metric]float64{MetricCPU: 95}); len(ev) != 0 {
		t.Fatalf("single sample fired: %#v", ev)
	}
	ev := e.Evaluate(t0().Add(time.Second), map[Metric]float64{MetricCPU: 95})
	if len(ev) != 1 || ev[0].State != StateFiring {
		t.Fatalf("second sample = %#v", ev)
	}
}

func TestStaysFiringWithoutRepeatEvent(t *testing.T) {
	e := NewEngine([]Rule{cpuRule(0)}, 0)
	e.Evaluate(t0(), map[Metric]float64{MetricCPU: 95})
	e.Evaluate(t0().Add(time.Second), map[Metric]float64{MetricCPU: 95})
	ev := e.Evaluate(t0().Add(2*time.Second), map[Metric]float64{MetricCPU: 99})
	if len(ev) != 0 {
		t.Fatalf("repeat fire: %#v", ev)
	}
}

func TestRecoveryAndSecondFiring(t *testing.T) {
	e := NewEngine([]Rule{cpuRule(0)}, 0)
	e.Evaluate(t0(), map[Metric]float64{MetricCPU: 95})
	e.Evaluate(t0().Add(time.Second), map[Metric]float64{MetricCPU: 95})
	ev := e.Evaluate(t0().Add(2*time.Second), map[Metric]float64{MetricCPU: 10})
	if len(ev) != 1 || ev[0].State != StateRecovered {
		t.Fatalf("recover = %#v", ev)
	}
	e.Evaluate(t0().Add(3*time.Second), map[Metric]float64{MetricCPU: 95})
	ev = e.Evaluate(t0().Add(4*time.Second), map[Metric]float64{MetricCPU: 95})
	if len(ev) != 1 || ev[0].State != StateFiring {
		t.Fatalf("second fire = %#v", ev)
	}
}

func TestCooldownSuppressesRefire(t *testing.T) {
	e := NewEngine([]Rule{cpuRule(0)}, time.Minute)
	e.Evaluate(t0(), map[Metric]float64{MetricCPU: 95})
	e.Evaluate(t0().Add(time.Second), map[Metric]float64{MetricCPU: 95})
	e.Evaluate(t0().Add(2*time.Second), map[Metric]float64{MetricCPU: 10})
	e.Evaluate(t0().Add(3*time.Second), map[Metric]float64{MetricCPU: 95})
	ev := e.Evaluate(t0().Add(4*time.Second), map[Metric]float64{MetricCPU: 95})
	if len(ev) != 0 {
		t.Fatalf("fired during cooldown: %#v", ev)
	}
	e.Evaluate(t0().Add(2*time.Minute), map[Metric]float64{MetricCPU: 10})
	e.Evaluate(t0().Add(2*time.Minute+time.Second), map[Metric]float64{MetricCPU: 95})
	ev = e.Evaluate(t0().Add(2*time.Minute+2*time.Second), map[Metric]float64{MetricCPU: 95})
	if len(ev) != 1 || ev[0].State != StateFiring {
		t.Fatalf("after cooldown = %#v", ev)
	}
}

func TestInvalidRuleIgnored(t *testing.T) {
	e := NewEngine([]Rule{{ID: "bad", Metric: "nope", Operator: OpGT, Value: 1}}, 0)
	ev := e.Evaluate(t0(), map[Metric]float64{"nope": 100})
	if len(ev) != 0 {
		t.Fatalf("invalid rule produced %#v", ev)
	}
}

func TestMissingMetricCannotFire(t *testing.T) {
	e := NewEngine([]Rule{cpuRule(0)}, 0)
	ev := e.Evaluate(t0(), map[Metric]float64{MetricMemory: 99})
	if len(ev) != 0 {
		t.Fatalf("missing metric fired: %#v", ev)
	}
	e.Evaluate(t0(), map[Metric]float64{MetricCPU: 95})
	e.Evaluate(t0().Add(time.Second), map[Metric]float64{MetricCPU: 95})
	ev = e.Evaluate(t0().Add(2*time.Second), map[Metric]float64{})
	if len(ev) != 1 || ev[0].State != StateRecovered {
		t.Fatalf("missing after fire = %#v", ev)
	}
}
