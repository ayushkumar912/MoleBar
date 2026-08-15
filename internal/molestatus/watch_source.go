package molestatus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"time"
)

const (
	watchMaxRestarts = 8
	watchMaxBackoff  = 30 * time.Second
)

// WatchSource runs `mo status --watch --interval=<d>` and decodes NDJSON
// from a single long-lived process.
type WatchSource struct {
	opts Options
}

// NewWatchSource constructs a watch-mode source. Prefer NewSource unless
// a test needs this strategy in isolation.
func NewWatchSource(opts Options) *WatchSource {
	return &WatchSource{opts: opts}
}

func (s *WatchSource) Run(ctx context.Context, emit func(Result)) {
	_ = s.run(ctx, emit)
}

func (s *WatchSource) run(ctx context.Context, emit func(Result)) error {
	if ctx == nil {
		ctx = context.Background()
	}
	backoff := time.Second
	failures := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		gotSample, err := s.stream(ctx, emit)
		if gotSample {
			failures = 0
			backoff = time.Second
		}
		if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return ctx.Err()
		}
		if isWatchUnsupported(err, "") {
			return ErrWatchUnsupported
		}
		if errors.Is(err, ErrNotFound) {
			return err
		}
		failures++
		if failures >= watchMaxRestarts {
			return fmt.Errorf("watch: giving up after %d restarts: %w", failures, err)
		}
		log.Printf("molebar: watch stream ended (%v); restarting in %s", err, backoff)
		if emit != nil {
			emit(Result{Err: fmt.Errorf("watch stream ended: %w", err)})
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		if backoff < watchMaxBackoff {
			backoff *= 2
			if backoff > watchMaxBackoff {
				backoff = watchMaxBackoff
			}
		}
	}
}

func (s *WatchSource) stream(ctx context.Context, emit func(Result)) (bool, error) {
	bin := s.opts.bin()
	interval := s.opts.interval()
	cmd := exec.CommandContext(ctx, bin, "status", "--watch", "--interval="+interval.String())
	configureProcessGroup(cmd)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return false, fmt.Errorf("watch stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return false, wrapRunError(bin, err, stderr.String(), ctx)
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

	dec := json.NewDecoder(stdout)
	var decodeErr error
	gotSample := false
	for {
		var st Status
		if err := dec.Decode(&st); err != nil {
			decodeErr = err
			break
		}
		gotSample = true
		if emit != nil {
			emit(Result{Status: &st})
		}
	}

	killProcessGroup(cmd)
	close(stopKiller)
	procErr := <-waitErr
	if ctx.Err() != nil {
		return gotSample, ctx.Err()
	}
	if isWatchUnsupported(procErr, stderr.String()) {
		return gotSample, ErrWatchUnsupported
	}
	if procErr != nil {
		return gotSample, wrapRunError(bin, procErr, stderr.String(), ctx)
	}
	if decodeErr != nil && !errors.Is(decodeErr, io.EOF) {
		if !gotSample && isWatchUnsupported(decodeErr, stderr.String()) {
			return false, ErrWatchUnsupported
		}
		return gotSample, fmt.Errorf("parse watch output from %q: %w", bin, decodeErr)
	}
	return gotSample, fmt.Errorf("watch stream from %q ended", bin)
}
