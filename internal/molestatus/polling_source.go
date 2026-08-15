package molestatus

import (
	"context"
	"time"
)

// PollingSource runs `mo status --json` once per interval.
// Used when watch mode is unavailable or has given up.
type PollingSource struct {
	opts Options
}

// NewPollingSource constructs a one-shot polling source.
func NewPollingSource(opts Options) *PollingSource {
	return &PollingSource{opts: opts}
}

func (s *PollingSource) Run(ctx context.Context, emit func(Result)) {
	if ctx == nil {
		ctx = context.Background()
	}
	s.emitOnce(ctx, emit)

	ticker := time.NewTicker(s.opts.interval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.emitOnce(ctx, emit)
		}
	}
}

func (s *PollingSource) emitOnce(ctx context.Context, emit func(Result)) {
	cctx, cancel := context.WithTimeout(ctx, s.opts.timeout())
	defer cancel()
	st, err := Fetch(cctx, s.opts.bin())
	if emit != nil {
		emit(Result{Status: st, Err: err})
	}
}
