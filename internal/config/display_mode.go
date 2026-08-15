// Package config holds runtime settings and persisted user preferences.
package config

import (
	"fmt"
	"strings"
)

// DisplayMode is the menu-bar title presentation. It is a value object:
// parsing/normalization live here so the rest of the program does not
// scatter raw "sys"/"net"/"both" strings.
type DisplayMode string

const (
	DisplayModeSys  DisplayMode = "sys"
	DisplayModeNet  DisplayMode = "net"
	DisplayModeBoth DisplayMode = "both"
)

// DefaultDisplayMode is used when no CLI override and no saved preference exist.
const DefaultDisplayMode = DisplayModeSys

// ParseDisplayMode accepts a user- or file-supplied value.
// Empty input is the default (sys). Unknown values return the default and an error
// so callers can treat them as invalid rather than silently persisting junk.
func ParseDisplayMode(v string) (DisplayMode, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "sys", "system":
		return DisplayModeSys, nil
	case "net", "network":
		return DisplayModeNet, nil
	case "both":
		return DisplayModeBoth, nil
	default:
		return DefaultDisplayMode, fmt.Errorf("invalid display mode %q", v)
	}
}

// NormalizeDisplayMode maps any input to a valid mode. Invalid values become sys.
func NormalizeDisplayMode(v string) DisplayMode {
	mode, _ := ParseDisplayMode(v)
	return mode
}

func (m DisplayMode) String() string {
	switch m {
	case DisplayModeNet:
		return string(DisplayModeNet)
	case DisplayModeBoth:
		return string(DisplayModeBoth)
	default:
		return string(DisplayModeSys)
	}
}

// Label is the user-visible name used in the Display menu.
func (m DisplayMode) Label() string {
	switch NormalizeDisplayMode(string(m)) {
	case DisplayModeNet:
		return "Network"
	case DisplayModeBoth:
		return "Both"
	default:
		return "System"
	}
}
