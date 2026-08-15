package molestatus

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestDetectWatchSupported(t *testing.T) {
	bin := writeFakeMo(t, `
if [ "$1" = "version" ] || [ "$1" = "--version" ]; then
  echo "mo 1.2.3"
  exit 0
fi
if [ "$1" = "status" ] && [ "$2" = "--help" ]; then
  echo "usage: mo status --json --watch --interval"
  exit 0
fi
exit 1
`)
	caps, err := Detect(context.Background(), bin)
	if err != nil {
		t.Fatal(err)
	}
	if caps.Version != "mo 1.2.3" {
		t.Fatalf("version = %q", caps.Version)
	}
	if !caps.SupportsJSON || !caps.SupportsWatch {
		t.Fatalf("caps = %+v", caps)
	}
}

func TestDetectWatchUnsupported(t *testing.T) {
	bin := writeFakeMo(t, `
echo "usage: mo status --json"
`)
	caps, err := Detect(context.Background(), bin)
	if err != nil {
		t.Fatal(err)
	}
	if !caps.SupportsJSON {
		t.Fatal("expected JSON support")
	}
	if caps.SupportsWatch {
		t.Fatal("watch should be unsupported")
	}
}

func TestDetectExecutableMissing(t *testing.T) {
	_, err := Detect(context.Background(), filepath.Join(t.TempDir(), "missing-mo"))
	if err == nil || !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestDetectMalformedCapabilityOutput(t *testing.T) {
	bin := writeFakeMo(t, `exit 0`)
	_, err := Detect(context.Background(), bin)
	if err == nil || !errors.Is(err, ErrMalformedCapabilities) {
		t.Fatalf("err = %v", err)
	}
}

func TestDetectTransientCommandFailure(t *testing.T) {
	bin := writeFakeMo(t, `
if [ "$1" = "version" ] || [ "$1" = "--version" ]; then
  echo "mo 0.1"
  exit 0
fi
echo "busy" >&2
exit 1
`)
	_, err := Detect(context.Background(), bin)
	if err == nil || !errors.Is(err, ErrNonZero) {
		t.Fatalf("err = %v", err)
	}
	if ErrorCategory(err) != "command_failure" {
		t.Fatalf("category = %q", ErrorCategory(err))
	}
}

func TestDetectTimeout(t *testing.T) {
	bin := writeFakeMo(t, `sleep 5`)
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	_, err := Detect(ctx, bin)
	if err == nil || !errors.Is(err, ErrTimeout) {
		t.Fatalf("err = %v", err)
	}
}
