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
	Status *Status
	Err    error
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

// NewSource prefers `mo status --watch --interval=...` and falls back to
// one-shot `mo status --json` polling when watch is missing or gives up.
func NewSource(opts Options) Source {
	return &adaptiveSource{
		watch: NewWatchSource(opts),
		poll:  NewPollingSource(opts),
	}
}

type adaptiveSource struct {
	watch *WatchSource
	poll  *PollingSource
}

func (s *adaptiveSource) Run(ctx context.Context, emit func(Result)) {
	if ctx == nil {
		ctx = context.Background()
	}
	err := s.watch.run(ctx, emit)
	if ctx.Err() != nil {
		return
	}
	if err != nil && !errors.Is(err, ErrWatchUnsupported) {
		log.Printf("molebar: falling back to polling: %v", err)
	}
	s.poll.Run(ctx, emit)
}
