package platform

import "errors"

// ErrUnsupported is returned when a platform service is not available.
var ErrUnsupported = errors.New("unsupported on this platform")

// LoginItemManager is the OS boundary for Launch at Login.
type LoginItemManager interface {
	Enabled() (bool, error)
	SetEnabled(bool) error
}

// AppPath is the path registered as a login item. Empty uses the
// current executable / enclosing .app bundle.
type AppPath string
