package platform

import "context"

// Notification is a user-visible alert payload.
type Notification struct {
	Title    string
	Subtitle string
	Body     string
}

// Notifier delivers notifications. Implementations must not invoke a shell.
type Notifier interface {
	Notify(ctx context.Context, n Notification) error
}

// NopNotifier discards notifications.
type NopNotifier struct{}

func (NopNotifier) Notify(context.Context, Notification) error { return nil }
