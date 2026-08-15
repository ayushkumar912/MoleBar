//go:build !darwin

package platform

import "context"

// DarwinSaveDialog is unavailable off macOS.
type DarwinSaveDialog struct{}

func (DarwinSaveDialog) Choose(context.Context, string) (string, error) {
	return "", ErrUnsupported
}
