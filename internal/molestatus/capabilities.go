package molestatus

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Capabilities is what the installed Mole binary actually supports.
// It is derived from probing safe CLI help/version output, not a
// hardcoded version matrix.
type Capabilities struct {
	Version       string
	SupportsJSON  bool
	SupportsWatch bool
}

// Detect probes bin for version and status flags. It does not start
// a watch stream and does not interpret transient failures as
// "watch unsupported".
func Detect(ctx context.Context, bin string) (Capabilities, error) {
	if bin == "" {
		bin = ResolveBinary("")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if !binaryExists(bin) {
		return Capabilities{}, fmt.Errorf("%q: %w", bin, ErrNotFound)
	}

	var caps Capabilities
	if ver, err := probeVersion(ctx, bin); err == nil {
		caps.Version = ver
	} else if isNotFound(err) {
		return Capabilities{}, fmt.Errorf("%q: %w", bin, ErrNotFound)
	} else if ctx.Err() != nil {
		return Capabilities{}, wrapRunError(bin, err, "", ctx)
	}

	out, stderr, err := probeHelp(ctx, bin)
	if err != nil {
		if isNotFound(err) {
			return caps, fmt.Errorf("%q: %w", bin, ErrNotFound)
		}
		return caps, wrapRunError(bin, err, stderr, ctx)
	}
	text := strings.ToLower(out + "\n" + stderr)
	if strings.TrimSpace(text) == "" && caps.Version == "" {
		return caps, fmt.Errorf("%w", ErrMalformedCapabilities)
	}
	caps.SupportsJSON = strings.Contains(text, "--json")
	caps.SupportsWatch = strings.Contains(text, "--watch")
	return caps, nil
}

func binaryExists(bin string) bool {
	if bin == "" {
		return false
	}
	if _, err := exec.LookPath(bin); err == nil {
		return true
	}
	st, err := os.Stat(bin)
	return err == nil && !st.IsDir()
}

func probeVersion(ctx context.Context, bin string) (string, error) {
	for _, args := range [][]string{{"version"}, {"--version"}} {
		cmd := exec.CommandContext(ctx, bin, args...)
		out, stderr, err := runCommand(ctx, cmd)
		if err == nil {
			return firstLine(string(out)), nil
		}
		if isNotFound(err) || ctx.Err() != nil {
			return "", wrapRunError(bin, err, stderr, ctx)
		}
	}
	return "", fmt.Errorf("%w", ErrNonZero)
}

func probeHelp(ctx context.Context, bin string) (string, string, error) {
	cmd := exec.CommandContext(ctx, bin, "status", "--help")
	out, stderr, err := runCommand(ctx, cmd)
	if err == nil {
		return string(out), stderr, nil
	}
	if isNotFound(err) || ctx.Err() != nil {
		return string(out), stderr, err
	}
	cmd = exec.CommandContext(ctx, bin, "--help")
	out2, stderr2, err2 := runCommand(ctx, cmd)
	if err2 == nil {
		return string(out2), stderr2, nil
	}
	return string(out2), stderr2, err2
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}
