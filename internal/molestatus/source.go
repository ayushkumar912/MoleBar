package molestatus

import (
	"context"
	"errors"
	"log"
	"time"
)

// Result is one collection attempt. Status is nil when Err is set, except
// that a source may still attach a partial snapshot; callers should treat
// a non-nil Err as a failed interval.
type Result struct {
	Status   *Status
	Err      error
	Strategy string
}

// Source streams Mole status snapshots until ctx is cancelled.
// The rest of the application does not care whether watch or polling is active.
type Source interface {
	Run(ctx context.Context, emit func(Result))
}

// Options configure how Mole is invoked.
type Options struct {
	Bin      string
	Interval time.Duration
	Timeout  time.Duration
	// Caps is an optional pre-detected capability set. When set, Run
	// does not probe Mole again.
	Caps *Capabilities
}

func (o Options) bin() string {
	if o.Bin != "" {
		return o.Bin
	}
	return ResolveBinary("")
}

func (o Options) interval() time.Duration {
	if o.Interval > 0 {
		return o.Interval
	}
	return 5 * time.Second
}

func (o Options) timeout() time.Duration {
	if o.Timeout > 0 {
		return o.Timeout
	}
	return defaultFetchTimeout
}

// NewSource prefers `mo status --watch --interval=...` when capability
// detection (or a live probe) says watch is available, and falls back to
// one-shot `mo status --json` polling when watch is unsupported or the
// watch process gives up after bounded restarts.
func NewSource(opts Options) Source {
	return &adaptiveSource{
		opts:  opts,
		watch: NewWatchSource(opts),
		poll:  NewPollingSource(opts),
	}
}

type adaptiveSource struct {
	opts     Options
	watch    *WatchSource
	poll     *PollingSource
	caps     Capabilities
	strategy string
}

// Strategy is "watch" or "poll" after Run has chosen a collector.
func (s *adaptiveSource) Strategy() string {
	if s == nil || s.strategy == "" {
		return "unknown"
	}
	return s.strategy
}

// Caps is the last capability-detection result.
func (s *adaptiveSource) Caps() Capabilities {
	if s == nil {
		return Capabilities{}
	}
	return s.caps
}

func (s *adaptiveSource) Run(ctx context.Context, emit func(Result)) {
	if ctx == nil {
		ctx = context.Background()
	}
	var (
		caps Capabilities
		err  error
	)
	if s.opts.Caps != nil {
		caps = *s.opts.Caps
		s.caps = caps
	} else {
		caps, err = Detect(ctx, s.opts.bin())
		if ctx.Err() != nil {
			return
		}
		if err == nil {
			s.caps = caps
		} else if !errors.Is(err, ErrNotFound) {
			log.Printf("molebar: capability probe failed: %v", err)
		} else {
			s.caps = caps
			if emit != nil {
				emit(Result{Err: err})
			}
		}
	}

	tryWatch := false
	switch {
	case err == nil && caps.SupportsWatch:
		tryWatch = true
	case err != nil && !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrWatchUnsupported):
		// Inconclusive probe (timeout, crash, transient): try watch once
		// rather than permanently assuming polling.
		tryWatch = true
	}

	wrap := func(r Result) {
		if r.Strategy == "" {
			r.Strategy = s.strategy
		}
		if emit != nil {
			emit(r)
		}
	}

	if tryWatch {
		s.strategy = "watch"
		werr := s.watch.run(ctx, wrap)
		if ctx.Err() != nil {
			return
		}
		if werr == nil {
			return
		}
		if !errors.Is(werr, ErrWatchUnsupported) {
			log.Printf("molebar: falling back to polling: %v", werr)
		}
	}
	s.strategy = "poll"
	s.poll.Run(ctx, wrap)
}
