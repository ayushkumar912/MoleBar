package config

import "strings"

// ProfileID is a named layout preset.
type ProfileID string

const (
	ProfileMinimal   ProfileID = "minimal"
	ProfileDeveloper ProfileID = "developer"
	ProfileNetwork   ProfileID = "network"
	ProfileBattery   ProfileID = "battery"
	ProfileFull      ProfileID = "full"
	ProfileCustom    ProfileID = "custom"
)

// Profile is a named preset. Presenters receive the resolved layout, not this type.
type Profile struct {
	ID     ProfileID
	Label  string
	Layout TrayLayout
}

// BuiltInProfiles returns the shipped presets in menu order.
func BuiltInProfiles() []Profile {
	return []Profile{
		{ID: ProfileMinimal, Label: "Minimal", Layout: TrayLayout{Metrics: []Metric{MetricCPU}}},
		{ID: ProfileDeveloper, Label: "Developer", Layout: TrayLayout{Metrics: []Metric{MetricCPU, MetricMemory, MetricRX, MetricTX}}},
		{ID: ProfileNetwork, Label: "Network", Layout: TrayLayout{Metrics: []Metric{MetricRX, MetricTX}}},
		{ID: ProfileBattery, Label: "Battery", Layout: TrayLayout{Metrics: []Metric{MetricBattery, MetricTemperature, MetricCPU}}},
		{ID: ProfileFull, Label: "Full", Layout: TrayLayout{Metrics: []Metric{MetricHealth, MetricCPU, MetricMemory, MetricRX, MetricTX}}},
	}
}

// LookupProfile returns a built-in profile by id.
func LookupProfile(id string) (Profile, bool) {
	want := ProfileID(strings.ToLower(strings.TrimSpace(id)))
	for _, p := range BuiltInProfiles() {
		if p.ID == want {
			return p, true
		}
	}
	return Profile{}, false
}

// ResolveProfileLayout returns the layout for a built-in profile.
func ResolveProfileLayout(id string) (TrayLayout, bool) {
	p, ok := LookupProfile(id)
	if !ok {
		return TrayLayout{}, false
	}
	return NormalizeLayout(p.Layout), true
}

// MatchingProfile returns the built-in profile whose layout equals l.
func MatchingProfile(l TrayLayout) ProfileID {
	for _, p := range BuiltInProfiles() {
		if NormalizeLayout(p.Layout).Equal(l) {
			return p.ID
		}
	}
	return ProfileCustom
}

// ParseProfileID accepts a user- or file-supplied profile name.
func ParseProfileID(v string) (ProfileID, error) {
	id := ProfileID(strings.ToLower(strings.TrimSpace(v)))
	if id == "" || id == ProfileCustom {
		return ProfileCustom, nil
	}
	if _, ok := LookupProfile(string(id)); !ok {
		return ProfileCustom, invalidValueError("profile", v)
	}
	return id, nil
}
