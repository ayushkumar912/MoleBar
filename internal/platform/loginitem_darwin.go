//go:build darwin

package platform

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DarwinLoginItem uses System Events via osascript stdin. It does not
// edit plist files, require sudo, or write launch daemons.
type DarwinLoginItem struct {
	Path   string
	runner func(ctx context.Context, stdin string) (string, error)
}

// NewDarwinLoginItem manages login-item state for path (or the current app).
func NewDarwinLoginItem(path string) *DarwinLoginItem {
	if path == "" {
		path = currentAppPath()
	}
	return &DarwinLoginItem{Path: path, runner: runOSAscript}
}

func (m *DarwinLoginItem) Enabled() (bool, error) {
	if m == nil || m.Path == "" {
		return false, ErrUnsupported
	}
	script := fmt.Sprintf(`tell application "System Events" to get the path of every login item`)
	out, err := m.run(script)
	if err != nil {
		return false, err
	}
	want := strings.TrimRight(m.Path, "/")
	for _, item := range strings.Split(out, ", ") {
		if strings.TrimRight(strings.TrimSpace(item), "/") == want {
			return true, nil
		}
	}
	return false, nil
}

func (m *DarwinLoginItem) SetEnabled(on bool) error {
	if m == nil || m.Path == "" {
		return ErrUnsupported
	}
	enabled, err := m.Enabled()
	if err != nil {
		return err
	}
	if enabled == on {
		return nil
	}
	if on {
		script := fmt.Sprintf(
			`tell application "System Events" to make login item at end with properties {path:%s, hidden:false}`,
			quoteAppleScript(m.Path),
		)
		_, err = m.run(script)
		return err
	}
	script := fmt.Sprintf(
		`tell application "System Events" to delete (every login item whose path is %s)`,
		quoteAppleScript(m.Path),
	)
	_, err = m.run(script)
	return err
}

func (m *DarwinLoginItem) run(script string) (string, error) {
	runner := m.runner
	if runner == nil {
		runner = runOSAscript
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return runner(ctx, script)
}

func runOSAscript(ctx context.Context, stdin string) (string, error) {
	cmd := exec.CommandContext(ctx, "osascript")
	cmd.Stdin = strings.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return "", fmt.Errorf("osascript: %w: %s", err, msg)
		}
		return "", fmt.Errorf("osascript: %w", err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func currentAppPath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return ""
	}
	const marker = ".app/Contents/MacOS/"
	if i := strings.Index(exe, marker); i >= 0 {
		return exe[:i+len(".app")]
	}
	return exe
}
