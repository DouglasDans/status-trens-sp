package source

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// Named is a source that knows its own name, so its failure can be logged
// even when it returns no snapshot.
type Named interface {
	StatusSource
	Name() string
}

type result struct {
	snap Snapshot
	err  error
}

// Collect fetches every source at once and returns the snapshots of the ones
// that answered, in the order given (the priority Merge uses). Each source
// gets its own timeout, so the cycle takes as long as the slowest source up
// to that limit, never the sum; a source that ignores its context is
// abandoned when the limit passes. Every failure and every label no rule
// recognizes is logged.
func Collect(ctx context.Context, sources []Named, timeout time.Duration, logger *slog.Logger) []Snapshot {
	results := make([]chan result, len(sources))
	for i, s := range sources {
		results[i] = make(chan result, 1) // buffered: an abandoned fetch never blocks
		go func() {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			snap, err := s.Fetch(ctx)
			results[i] <- result{snap, err}
		}()
	}

	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	var snaps []Snapshot
	for i, s := range sources {
		var r result
		select {
		case r = <-results[i]:
		case <-deadline.C:
			r = result{err: context.DeadlineExceeded}
		}
		if r.err != nil {
			logger.Warn("source failed", "source", s.Name(), "error", r.err)
			continue
		}
		for _, reading := range r.snap.Readings {
			if !reading.Recognized {
				logger.Warn("unrecognized status label", "source", s.Name(), "line", reading.Code, "label", reading.Label)
			}
		}
		snaps = append(snaps, r.snap)
	}
	return snaps
}

// Timeout is how long each source gets per cycle. The sources usually answer
// in well under a second; the legacy CPTM endpoint sometimes takes over 20 s,
// and waiting for it only delays the others' news.
const Timeout = 4 * time.Second

// All is every source, in priority order: when sources agree on the state,
// the earlier one's text is shown.
func All(client *http.Client, logger *slog.Logger) []Named {
	return []Named{
		NewProximoTrem(ProximoTremURL, client),
		NewMetro(MetroURL, client, logger),
		NewCPTMNovo(CPTMNovoURL, client),
		NewCPTM(CPTMURL, client),
	}
}
