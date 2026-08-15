package config

import "time"

// CurrentVersion is the persisted preferences schema version.
const CurrentVersion = 1

// AlertPref is the persisted form of a threshold rule.
type AlertPref struct {
	Metric   string  `json:"metric"`
	Operator string  `json:"operator"`
	Value    float64 `json:"value"`
	For      string  `json:"for"`
}

// Preferences is the versioned on-disk user configuration.
type Preferences struct {
	Version       int         `json:"version"`
	Profile       string      `json:"profile"`
	Layout        TrayLayout  `json:"layout"`
	AlertsEnabled bool        `json:"alerts_enabled"`
	Alerts        []AlertPref `json:"alerts"`
	LaunchAtLogin bool        `json:"launch_at_login"`
}

// DefaultPreferences is used when no file exists.
func DefaultPreferences() Preferences {
	return Preferences{
		Version:       CurrentVersion,
		Profile:       "",
		Layout:        LayoutFromDisplayMode(DefaultDisplayMode),
		AlertsEnabled: true,
		Alerts:        DefaultAlertPrefs(),
	}
}

// DefaultAlertPrefs are the shipped threshold rules.
func DefaultAlertPrefs() []AlertPref {
	return []AlertPref{
		{Metric: "cpu", Operator: ">", Value: 90, For: "30s"},
		{Metric: "memory", Operator: ">", Value: 85, For: "60s"},
		{Metric: "disk", Operator: ">", Value: 90, For: "5m"},
		{Metric: "temperature", Operator: ">", Value: 90, For: "30s"},
		{Metric: "battery", Operator: "<", Value: 15, For: "10s"},
		{Metric: "rx", Operator: ">", Value: 50, For: "20s"},
		{Metric: "tx", Operator: ">", Value: 50, For: "20s"},
	}
}

// Normalize fills defaults and drops invalid values without deleting
// unrecognized future fields the caller may have set on a copy.
func (p Preferences) Normalize() Preferences {
	if p.Version <= 0 {
		p.Version = CurrentVersion
	}
	p.Layout = NormalizeLayout(p.Layout)
	if id, err := ParseProfileID(p.Profile); err == nil && id != ProfileCustom && id != "" {
		if layout, ok := ResolveProfileLayout(string(id)); ok {
			p.Profile = string(id)
			p.Layout = layout
		}
	} else if p.Profile != "" {
		p.Profile = string(MatchingProfile(p.Layout))
		if p.Profile == string(ProfileCustom) {
			p.Profile = "custom"
		}
	}
	if p.Alerts == nil {
		p.Alerts = DefaultAlertPrefs()
	}
	return p
}

// ApplyDisplayMode sets the layout from a legacy mode and records a
// matching profile when one exists.
func (p *Preferences) ApplyDisplayMode(mode DisplayMode) {
	p.Layout = LayoutFromDisplayMode(mode)
	id := MatchingProfile(p.Layout)
	if id == ProfileCustom {
		p.Profile = ""
	} else {
		p.Profile = string(id)
	}
}

// ApplyProfile replaces the layout with a built-in preset.
func (p *Preferences) ApplyProfile(id string) bool {
	layout, ok := ResolveProfileLayout(id)
	if !ok {
		return false
	}
	p.Profile = string(ProfileID(id))
	p.Layout = layout
	return true
}

// ApplyMetricToggle flips a tray metric and updates the profile label.
func (p *Preferences) ApplyMetricToggle(m Metric) {
	p.Layout = p.Layout.ToggleMetric(m)
	p.Profile = string(MatchingProfile(p.Layout))
	if p.Profile == string(ProfileCustom) {
		p.Profile = "custom"
	}
}

// DisplayMode is the legacy approximation of the current layout.
func (p Preferences) DisplayMode() DisplayMode {
	return p.Layout.DisplayMode()
}

// ParseAlertDuration accepts a rule duration string such as "30s".
func ParseAlertDuration(v string) (time.Duration, error) {
	if v == "" {
		return 0, nil
	}
	return time.ParseDuration(v)
}
