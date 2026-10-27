// Package poll runs one polling cycle: read every source, merge them line by
// line, save the board GET /status serves, apply the notification rules to
// each line and push the changes that matter.
package poll

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/DouglasDans/status-trens-sp/backend/internal/board"
	"github.com/DouglasDans/status-trens-sp/backend/internal/catalog"
	"github.com/DouglasDans/status-trens-sp/backend/internal/notify"
	"github.com/DouglasDans/status-trens-sp/backend/internal/source"
	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

// Store keeps each line's memory between polls.
type Store interface {
	Load(ctx context.Context) (map[int]status.Memory, error)
	Save(ctx context.Context, code int, m status.Memory) error
}

// BoardStore keeps the merged board between polls and for GET /status.
type BoardStore interface {
	LoadBoard(ctx context.Context) (board.Board, bool, error)
	SaveBoard(ctx context.Context, b board.Board) error
}

// Notifier delivers a push to the favorites of a line.
type Notifier interface {
	Send(ctx context.Context, m notify.Message) error
}

type Poller struct {
	Sources  []source.Named // in priority order
	Timeout  time.Duration  // per source
	Store    Store
	Boards   BoardStore
	Notifier Notifier
	Logger   *slog.Logger
	Now      func() time.Time
}

// Run executes one cycle. The board is saved even when the line memory
// cannot be read, and the pushes go out even when the board cannot be saved:
// the app's data and the notifications fail independently. A line whose push
// fails keeps its old memory, so the same change is detected and pushed again
// on the next cycle.
func (p *Poller) Run(ctx context.Context) error {
	snaps := source.Collect(ctx, p.Sources, p.Timeout, p.Logger)
	if len(snaps) == 0 {
		return errors.New("fetching status: all sources failed")
	}
	merged := source.Merge(snaps)

	var errs []error
	if err := p.saveBoard(ctx, merged, snaps); err != nil {
		errs = append(errs, err)
	}

	memory, err := p.Store.Load(ctx)
	if err != nil {
		return errors.Join(append(errs, fmt.Errorf("loading memory: %w", err))...)
	}
	for _, r := range merged {
		if err := p.apply(ctx, memory[r.Code], r); err != nil {
			return errors.Join(append(errs, err)...)
		}
	}
	return errors.Join(errs...)
}

// saveBoard builds this cycle's board on top of the last one, so a line no
// source reported this time keeps its last reading for a while.
func (p *Poller) saveBoard(ctx context.Context, merged []source.Reading, snaps []source.Snapshot) error {
	prev, _, err := p.Boards.LoadBoard(ctx)
	if err != nil {
		p.Logger.Error("loading board, unseen lines are dropped this cycle", "error", err)
	}
	sources := make([]string, len(snaps))
	for i, s := range snaps {
		sources[i] = s.Source
	}
	if err := p.Boards.SaveBoard(ctx, board.Build(prev, merged, sources, p.Now())); err != nil {
		return fmt.Errorf("saving board: %w", err)
	}
	return nil
}

// apply runs the notification rules on one merged line.
func (p *Poller) apply(ctx context.Context, prev status.Memory, r source.Reading) error {
	next, push := status.Next(prev, r.State)
	if next == prev {
		return nil
	}
	if push {
		d := catalog.Present(r.Code, r.Name, r.ColorHex)
		msg := notify.Compose(r.Code, d.Name, d.ColorHex, r.Label, r.Description)
		if err := p.Notifier.Send(ctx, msg); err != nil {
			p.Logger.Error("push failed, retrying next poll", "line", r.Code, "error", err)
			return nil
		}
		p.Logger.Info("pushed", "line", r.Code, "from", prev.State, "to", next.State, "source", r.Source)
	}
	if err := p.Store.Save(ctx, r.Code, next); err != nil {
		return fmt.Errorf("saving line %d: %w", r.Code, err)
	}
	return nil
}
