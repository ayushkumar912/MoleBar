package molestatus

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func writeFakeMo(t *testing.T, script string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake mo helper is a POSIX shell script")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "mo")
	body := "#!/bin/sh\n" + script + "\n"
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

const okJSON = `{"health_score":7,"health_score_msg":"ok","cpu":{"usage":1},"memory":{"used_percent":2},"network":[]}`

func TestFetchSuccessAndStderrOnFailure(t *testing.T) {
	bin := writeFakeMo(t, `
if [ "$1" = "status" ] && [ "$2" = "--json" ]; then
  echo "mole exploded" >&2
  exit 3
fi
exit 1
`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := Fetch(ctx, bin)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrNonZero) {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(err.Error(), "mole exploded") {
		t.Fatalf("stderr not propagated: %v", err)
	}
}

func TestFetchMalformedJSON(t *testing.T) {
	bin := writeFakeMo(t, `echo 'not-json'`)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := Fetch(ctx, bin)
	if err == nil || !errors.Is(err, ErrMalformedJSON) {
		t.Fatalf("err = %v", err)
	}
}

func TestFetchTimeoutAndCancel(t *testing.T) {
	bin := writeFakeMo(t, `sleep 5`)
	t.Run("timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
		defer cancel()
		_, err := Fetch(ctx, bin)
		if err == nil || !errors.Is(err, ErrTimeout) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(40 * time.Millisecond)
			cancel()
		}()
		_, err := Fetch(ctx, bin)
		if err == nil || !errors.Is(err, ErrCanceled) {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestFetchNotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := Fetch(ctx, filepath.Join(t.TempDir(), "missing-mo"))
	if err == nil || !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestPollingSourceEmitsAndStops(t *testing.T) {
	bin := writeFakeMo(t, `echo '`+okJSON+`'`)
	src := NewPollingSource(Options{Bin: bin, Interval: 50 * time.Millisecond, Timeout: time.Second})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	got := make(chan Result, 4)
	go src.Run(ctx, func(r Result) { got <- r })
	select {
	case r := <-got:
		if r.Err != nil || r.Status == nil || r.Status.HealthScore != 7 {
			t.Fatalf("first = %+v err=%v", r.Status, r.Err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for poll")
	}
	cancel()
	time.Sleep(30 * time.Millisecond)
}

func TestWatchSourceNDJSON(t *testing.T) {
	bin := writeFakeMo(t, `
echo '`+okJSON+`'
echo '`+okJSON+`'
`)
	src := NewWatchSource(Options{Bin: bin, Interval: time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var n atomic.Int32
	done := make(chan struct{})
	go func() {
		src.Run(ctx, func(r Result) {
			if r.Err == nil && r.Status != nil {
				if n.Add(1) >= 2 {
					cancel()
				}
			}
		})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("watch did not exit, samples=%d", n.Load())
	}
	if n.Load() < 2 {
		t.Fatalf("got %d samples", n.Load())
	}
}

func TestWatchUnsupportedFallsBackToPolling(t *testing.T) {
	bin := writeFakeMo(t, `
for a in "$@"; do
  if [ "$a" = "--watch" ]; then
    echo "flag provided but not defined: -watch" >&2
    exit 2
  fi
done
echo '`+okJSON+`'
`)
	src := NewSource(Options{Bin: bin, Interval: time.Hour, Timeout: time.Second})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	got := make(chan Result, 2)
	go src.Run(ctx, func(r Result) { got <- r })
	select {
	case r := <-got:
		if r.Err != nil {
			t.Fatalf("fallback should succeed: %v", r.Err)
		}
		if r.Status == nil || r.Status.HealthScore != 7 {
			t.Fatalf("status = %+v", r.Status)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no sample from fallback")
	}
	cancel()
}

func TestWatchShutdownCancelsProcess(t *testing.T) {
	bin := writeFakeMo(t, `
echo '`+okJSON+`'
sleep 30
`)
	src := NewWatchSource(Options{Bin: bin, Interval: time.Second})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		src.Run(ctx, func(Result) {})
		close(done)
	}()
	time.Sleep(80 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("watch did not shut down")
	}
}
