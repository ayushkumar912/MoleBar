package config

import (
	"fmt"
	"strings"
)

const defaultSeparator = " | "

// TrayLayout is the resolved menu-bar title composition.
type TrayLayout struct {
	Metrics   []Metric `json:"metrics"`
	Separator string   `json:"separator"`
}

// NormalizeLayout drops unknown/duplicate metrics and supplies defaults.
// An empty metric list becomes the default System layout so the tray
// never renders nothing.
func NormalizeLayout(l TrayLayout) TrayLayout {
	if strings.TrimSpace(l.Separator) == "" {
		l.Separator = defaultSeparator
	}
	seen := make(map[Metric]bool, len(l.Metrics))
	out := make([]Metric, 0, len(l.Metrics))
	for _, raw := range l.Metrics {
		m, err := ParseMetric(string(raw))
		if err != nil || seen[m] {
			continue
		}
		seen[m] = true
		out = append(out, m)
	}
	if len(out) == 0 {
		out = LayoutFromDisplayMode(DefaultDisplayMode).Metrics
	}
	l.Metrics = out
	return l
}

// Contains reports whether m is in the layout.
func (l TrayLayout) Contains(m Metric) bool {
	for _, x := range l.Metrics {
		if x == m {
			return true
		}
	}
	return false
}

// Equal reports whether the metric sequences match (separator ignored).
func (l TrayLayout) Equal(other TrayLayout) bool {
	a := NormalizeLayout(l)
	b := NormalizeLayout(other)
	if len(a.Metrics) != len(b.Metrics) {
		return false
	}
	for i := range a.Metrics {
		if a.Metrics[i] != b.Metrics[i] {
			return false
		}
	}
	return true
}

// ToggleMetric adds m or removes it. At least one metric always remains.
func (l TrayLayout) ToggleMetric(m Metric) TrayLayout {
	l = NormalizeLayout(l)
	if !ValidMetric(m) {
		return l
	}
	if l.Contains(m) {
		if len(l.Metrics) == 1 {
			return l
		}
		next := make([]Metric, 0, len(l.Metrics)-1)
		for _, x := range l.Metrics {
			if x != m {
				next = append(next, x)
			}
		}
		l.Metrics = next
		return l
	}
	l.Metrics = append(l.Metrics, m)
	return l
}

// LayoutFromDisplayMode maps a legacy sys/net/both preference to a layout.
func LayoutFromDisplayMode(mode DisplayMode) TrayLayout {
	switch NormalizeDisplayMode(string(mode)) {
	case DisplayModeNet:
		return TrayLayout{Metrics: []Metric{MetricRX, MetricTX}, Separator: defaultSeparator}
	case DisplayModeBoth:
		return TrayLayout{
			Metrics:   []Metric{MetricCPU, MetricMemory, MetricRX, MetricTX},
			Separator: defaultSeparator,
		}
	default:
		return TrayLayout{Metrics: []Metric{MetricCPU, MetricMemory}, Separator: defaultSeparator}
	}
}

// DisplayMode approximates a legacy mode from a layout for CLI/tests.
func (l TrayLayout) DisplayMode() DisplayMode {
	n := NormalizeLayout(l)
	if n.Equal(LayoutFromDisplayMode(DisplayModeNet)) {
		return DisplayModeNet
	}
	if n.Equal(LayoutFromDisplayMode(DisplayModeBoth)) {
		return DisplayModeBoth
	}
	if n.Equal(LayoutFromDisplayMode(DisplayModeSys)) {
		return DisplayModeSys
	}
	return DisplayModeSys
}

func invalidValueError(kind, v string) error {
	return fmt.Errorf("invalid %s %q", kind, v)
}
