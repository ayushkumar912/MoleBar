//go:build darwin

package platform

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// DarwinSaveDialog shows a choose-file-name panel via osascript stdin.
type DarwinSaveDialog struct{}

func (DarwinSaveDialog) Choose(ctx context.Context, defaultName string) (string, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
	}
	if defaultName == "" {
		defaultName = "molebar-diagnostics.txt"
	}
	script := fmt.Sprintf(
		`POSIX path of (choose file name with prompt "Export Diagnostics" default name %s)`,
		quoteAppleScript(defaultName),
	)
	cmd := exec.CommandContext(ctx, "osascript")
	cmd.Stdin = strings.NewReader(script)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("save dialog: %w", err)
	}
	path := strings.TrimSpace(string(out))
	if path == "" {
		return "", fmt.Errorf("save dialog: canceled")
	}
	return path, nil
}
