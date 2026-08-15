package config

import "strings"

// Metric is a tray-title metric identifier.
type Metric string

const (
	MetricCPU         Metric = "cpu"
	MetricMemory      Metric = "memory"
	MetricRX          Metric = "rx"
	MetricTX          Metric = "tx"
	MetricBattery     Metric = "battery"
	MetricTemperature Metric = "temperature"
	MetricHealth      Metric = "health"
	MetricDisk        Metric = "disk"
)

// AllMetrics is the supported tray-metric set in menu order.
func AllMetrics() []Metric {
	return []Metric{
		MetricCPU,
		MetricMemory,
		MetricRX,
		MetricTX,
		MetricHealth,
		MetricBattery,
		MetricTemperature,
		MetricDisk,
	}
}

// ParseMetric accepts a user- or file-supplied metric name.
func ParseMetric(v string) (Metric, error) {
	m := Metric(strings.ToLower(strings.TrimSpace(v)))
	if !ValidMetric(m) {
		return "", errInvalidMetric(v)
	}
	return m, nil
}

// ValidMetric reports whether m is a known tray metric.
func ValidMetric(m Metric) bool {
	switch m {
	case MetricCPU, MetricMemory, MetricRX, MetricTX,
		MetricBattery, MetricTemperature, MetricHealth, MetricDisk:
		return true
	default:
		return false
	}
}

func errInvalidMetric(v string) error {
	return invalidValueError("metric", v)
}

// Label is the short name used in the Tray Metrics menu.
func (m Metric) Label() string {
	switch m {
	case MetricCPU:
		return "CPU"
	case MetricMemory:
		return "Memory"
	case MetricRX:
		return "RX"
	case MetricTX:
		return "TX"
	case MetricBattery:
		return "Battery"
	case MetricTemperature:
		return "Temperature"
	case MetricHealth:
		return "Health"
	case MetricDisk:
		return "Disk"
	default:
		return string(m)
	}
}
