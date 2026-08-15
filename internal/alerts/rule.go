package alerts

import (
	"fmt"
	"math"
	"time"
)

// Metric is a value the rule engine can observe.
type Metric string

const (
	MetricCPU         Metric = "cpu"
	MetricMemory      Metric = "memory"
	MetricDisk        Metric = "disk"
	MetricTemperature Metric = "temperature"
	MetricBattery     Metric = "battery"
	MetricRX          Metric = "rx"
	MetricTX          Metric = "tx"
	MetricProcessCPU  Metric = "process_cpu"
)

// Operator compares a sample to the rule threshold.
type Operator string

const (
	OpGT Operator = ">"
	OpLT Operator = "<"
	OpGE Operator = ">="
	OpLE Operator = "<="
)

// State is the lifecycle of one rule.
type State string

const (
	StateInactive  State = "inactive"
	StatePending   State = "pending"
	StateFiring    State = "firing"
	StateRecovered State = "recovered"
)

// Rule is a sustained-threshold predicate. Duration is the time the
// condition must hold before firing. A zero duration still requires two
// consecutive crossings so a single transient sample cannot fire.
type Rule struct {
	ID       string
	Metric   Metric
	Operator Operator
	Value    float64
	Duration time.Duration
}

// Validate reports whether the rule can be evaluated.
func (r Rule) Validate() error {
	if !validMetric(r.Metric) {
		return fmt.Errorf("invalid metric %q", r.Metric)
	}
	if !validOperator(r.Operator) {
		return fmt.Errorf("invalid operator %q", r.Operator)
	}
	if math.IsNaN(r.Value) || math.IsInf(r.Value, 0) {
		return fmt.Errorf("invalid value %v", r.Value)
	}
	if r.Duration < 0 {
		return fmt.Errorf("invalid duration %s", r.Duration)
	}
	return nil
}

func (r Rule) id() string {
	if r.ID != "" {
		return r.ID
	}
	return fmt.Sprintf("%s%s%g/%s", r.Metric, r.Operator, r.Value, r.Duration)
}

func (r Rule) holds(sample float64) bool {
	switch r.Operator {
	case OpGT:
		return sample > r.Value
	case OpLT:
		return sample < r.Value
	case OpGE:
		return sample >= r.Value
	case OpLE:
		return sample <= r.Value
	default:
		return false
	}
}

func validMetric(m Metric) bool {
	switch m {
	case MetricCPU, MetricMemory, MetricDisk, MetricTemperature,
		MetricBattery, MetricRX, MetricTX, MetricProcessCPU:
		return true
	default:
		return false
	}
}

func validOperator(op Operator) bool {
	switch op {
	case OpGT, OpLT, OpGE, OpLE:
		return true
	default:
		return false
	}
}

// DefaultRules are the shipped thresholds.
func DefaultRules() []Rule {
	return []Rule{
		{ID: "cpu", Metric: MetricCPU, Operator: OpGT, Value: 90, Duration: 30 * time.Second},
		{ID: "memory", Metric: MetricMemory, Operator: OpGT, Value: 85, Duration: 60 * time.Second},
		{ID: "disk", Metric: MetricDisk, Operator: OpGT, Value: 90, Duration: 5 * time.Minute},
		{ID: "temperature", Metric: MetricTemperature, Operator: OpGT, Value: 90, Duration: 30 * time.Second},
		{ID: "battery", Metric: MetricBattery, Operator: OpLT, Value: 15, Duration: 10 * time.Second},
		{ID: "rx", Metric: MetricRX, Operator: OpGT, Value: 50, Duration: 20 * time.Second},
		{ID: "tx", Metric: MetricTX, Operator: OpGT, Value: 50, Duration: 20 * time.Second},
	}
}
