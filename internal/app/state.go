package app

import (
	"time"

	"github.com/ayush-kumar912/molebar/internal/alerts"
	"github.com/ayush-kumar912/molebar/internal/config"
	"github.com/ayush-kumar912/molebar/internal/history"
	"github.com/ayush-kumar912/molebar/internal/molestatus"
	"github.com/ayush-kumar912/molebar/internal/session"
)

// State is a read-only snapshot of application state for presentation
// and diagnostics. The Controller is the only writer.
type State struct {
	Status      *molestatus.Status
	Session     session.Snapshot
	History     history.Summary
	Alerts      []alerts.Alert
	Profile     string
	Layout      config.TrayLayout
	LastError   error
	LastUpdated time.Time
	Strategy    string
	Caps        molestatus.Capabilities
}
