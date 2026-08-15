//go:build !darwin

package platform

import "context"

// DarwinNotifier is a no-op on non-macOS builds.
type DarwinNotifier struct{}

func (DarwinNotifier) Notify(context.Context, Notification) error { return ErrUnsupported }
