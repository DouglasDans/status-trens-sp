// Package poll runs one polling cycle: read the sources, apply the
// notification rules to each line and push the changes that matter.
package poll

import (
	"context"
	"fmt"
	"log/slog"

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

// Notifier delivers a push to the favorites of a line.
type Notifier interface {
	Send(ctx context.Context, m notify.Message) error
}

type Poller struct {
	Source   source.StatusSource
	Store    Store
	Notifier Notifier
	Logger   *slog.Logger
}

// Run executes one cycle. A line whose push fails keeps its old memory, so the
// same change is detected and pushed again on the next cycle.
func (p *Poller) Run(ctx context.Context) error {
	snap, err := p.Source.Fetch(ctx)
	if err != nil {
		return fmt.Errorf("fetching status: %w", err)
	}
	memory, err := p.Store.Load(ctx)
	if err != nil {
		return fmt.Errorf("loading memory: %w", err)
	}

	for _, r := range snap.Readings {
		prev := memory[r.Code]
		next, push := status.Next(prev, r.State)
		if next == prev {
			continue
		}
		if push {
			d := catalog.Present(r.Code, r.Name, r.ColorHex)
			msg := notify.Compose(r.Code, d.Name, d.ColorHex, r.Label, r.Description)
			if err := p.Notifier.Send(ctx, msg); err != nil {
				p.Logger.Error("push failed, retrying next poll", "line", r.Code, "error", err)
				continue
			}
			p.Logger.Info("pushed", "line", r.Code, "from", prev.State, "to", next.State)
		}
		if err := p.Store.Save(ctx, r.Code, next); err != nil {
			return fmt.Errorf("saving line %d: %w", r.Code, err)
		}
	}
	return nil
}
