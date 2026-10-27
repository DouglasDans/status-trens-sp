package source

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

type result struct {
	snap Snapshot
	err  error
}

// Collect fetches every source at once and returns the snapshots of the ones
// that answered, in the order given (the priority Merge uses). Every source
// gets the same deadline, timeout from now, so the cycle takes as long as the
// slowest source up to that limit, never the sum; any source that ignores its
// context is abandoned when the deadline passes. A source answering with no
// line counts as failed. Every failure and every label no rule recognizes is
// logged.
func Collect(ctx context.Context, sources []StatusSource, timeout time.Duration, logger *slog.Logger) []Snapshot {
	ctx, cancel := context.WithTimeout(ctx, timeout) // one deadline shared by every source
	defer cancel()
	results := make([]chan result, len(sources))
	for i, s := range sources {
		results[i] = make(chan result, 1) // buffered: an abandoned fetch never blocks
		go func() {
			snap, err := s.Fetch(ctx)
			results[i] <- result{snap, err}
		}()
	}

	var snaps []Snapshot
	for i, s := range sources {
		r := wait(ctx, results[i])
		if r.err == nil && len(r.snap.Readings) == 0 {
			r.err = errors.New("answered with no line")
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

// wait returns the source's result, or a timeout once the shared deadline has
// passed. Every source is waited on against the same deadline, so after it
// each remaining source is abandoned at once; an answer already there is
// always taken first.
func wait(ctx context.Context, ch <-chan result) result {
	select {
	case r := <-ch:
		return r
	default:
	}
	select {
	case r := <-ch:
		return r
	case <-ctx.Done():
		return result{err: ctx.Err()}
	}
}

// Timeout is how long each source gets per cycle. The sources usually answer
// in well under a second; the legacy CPTM endpoint sometimes takes over 20 s,
// and waiting for it only delays the others' news.
const Timeout = 4 * time.Second

// All is every source, in priority order: when sources agree on the state,
// the earlier one's text is shown.
func All(client *http.Client, logger *slog.Logger) []StatusSource {
	return []StatusSource{
		NewProximoTrem(ProximoTremURL, client),
		NewMetro(MetroURL, client, logger),
		NewCPTMNovo(CPTMNovoURL, client),
		NewCPTM(CPTMURL, client),
	}
}
