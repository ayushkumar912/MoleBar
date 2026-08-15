package platform

import "context"

// Clipboard copies text for "Copy System Summary".
type Clipboard interface {
	Copy(ctx context.Context, text string) error
}

// NopClipboard discards copies.
type NopClipboard struct{}

func (NopClipboard) Copy(context.Context, string) error { return nil }
