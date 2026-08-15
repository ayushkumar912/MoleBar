//go:build !darwin

package platform

import "context"

// PBCopy is unavailable off macOS.
type PBCopy struct{}

func (PBCopy) Copy(context.Context, string) error { return ErrUnsupported }
