//go:build darwin

package platform

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// DarwinNotifier shows a macOS notification via osascript stdin.
type DarwinNotifier struct{}

func (DarwinNotifier) Notify(ctx context.Context, n Notification) error {
	if ctx == nil {
		ctx = context.Background()
	}
	script := fmt.Sprintf("display notification %s with title %s", quoteAppleScript(n.Body), quoteAppleScript(n.Title))
	if n.Subtitle != "" {
		script += " subtitle " + quoteAppleScript(n.Subtitle)
	}
	cmd := exec.CommandContext(ctx, "osascript")
	cmd.Stdin = strings.NewReader(script)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("notification: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
