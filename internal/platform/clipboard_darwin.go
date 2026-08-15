//go:build darwin

package platform

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// PBCopy writes text to the macOS pasteboard via stdin.
type PBCopy struct{}

func (PBCopy) Copy(ctx context.Context, text string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, "pbcopy")
	cmd.Stdin = strings.NewReader(text)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pbcopy: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
