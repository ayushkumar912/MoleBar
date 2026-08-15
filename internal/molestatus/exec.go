package molestatus

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Sentinel errors so callers can distinguish failure classes without
// scraping implementation text into the tray.
var (
	ErrNotFound         = errors.New("mo executable not found")
	ErrWatchUnsupported = errors.New("mo status --watch is not supported")
	ErrMalformedJSON    = errors.New("malformed mole status JSON")
	ErrCanceled         = errors.New("mole command canceled")
	ErrTimeout          = errors.New("mole command timed out")
	ErrNonZero          = errors.New("mole command exited non-zero")
)

const defaultFetchTimeout = 5 * time.Second

// homebrewFallbacks cover GUI/launchd launches that do not inherit a login PATH.
var homebrewFallbacks = []string{
	"/opt/homebrew/bin/mo",
	"/usr/local/bin/mo",
}

// ResolveBinary returns the mo executable to invoke.
// An explicit path wins. Otherwise $PATH, then Homebrew prefixes.
// The lookup is done once by the caller and the result reused.
func ResolveBinary(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if p, err := exec.LookPath("mo"); err == nil {
		return p
	}
	for _, p := range homebrewFallbacks {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "mo"
}

// Fetch runs `mo status --json` once and parses the snapshot.
func Fetch(ctx context.Context, bin string) (*Status, error) {
	if bin == "" {
		bin = ResolveBinary("")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultFetchTimeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, bin, "status", "--json")
	out, stderr, err := runCommand(ctx, cmd)
	if err != nil {
		return nil, wrapRunError(bin, err, stderr, ctx)
	}
	s, err := Parse(out)
	if err != nil {
		return nil, fmt.Errorf("parse %q output: %w", bin, err)
	}
	return s, nil
}

func runCommand(ctx context.Context, cmd *exec.Cmd) ([]byte, string, error) {
	configureProcessGroup(cmd)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, stderr.String(), err
	}
	waitErr := make(chan error, 1)
	go func() {
		waitErr <- cmd.Wait()
	}()
	stopKiller := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			killProcessGroup(cmd)
		case <-stopKiller:
		}
	}()
	err := <-waitErr
	close(stopKiller)
	return stdout.Bytes(), stderr.String(), err
}

func wrapRunError(bin string, err error, stderr string, ctx context.Context) error {
	if ctx != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("%q: %w", bin, ErrTimeout)
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return fmt.Errorf("%q: %w", bin, ErrCanceled)
		}
	}
	if isNotFound(err) {
		return fmt.Errorf("%q: %w", bin, ErrNotFound)
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		msg := strings.TrimSpace(stderr)
		if msg == "" {
			msg = strings.TrimSpace(string(ee.Stderr))
		}
		if msg != "" {
			return fmt.Errorf("%q: %w: exit %d: %s", bin, ErrNonZero, ee.ExitCode(), msg)
		}
		return fmt.Errorf("%q: %w: exit %d", bin, ErrNonZero, ee.ExitCode())
	}
	return fmt.Errorf("run %q: %w", bin, err)
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
		return true
	}
	var ee *exec.Error
	if errors.As(err, &ee) && (errors.Is(ee.Err, exec.ErrNotFound) || errors.Is(ee.Err, os.ErrNotExist)) {
		return true
	}
	return false
}

func isWatchUnsupported(err error, stderr string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrWatchUnsupported) {
		return true
	}
	combined := strings.ToLower(err.Error() + "\n" + stderr)
	if !strings.Contains(combined, "watch") {
		return false
	}
	if strings.Contains(combined, "flag provided but not defined") ||
		strings.Contains(combined, "unknown flag") ||
		strings.Contains(combined, "unknown option") {
		return true
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) && ee.ExitCode() == 2 {
		return true
	}
	return false
}
