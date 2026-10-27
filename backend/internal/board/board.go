// Package board holds the merged status of every line: the poller writes it
// each cycle and GET /status serves it, so the API never calls a source.
package board

import (
	"slices"
	"time"

	"github.com/DouglasDans/status-trens-sp/backend/internal/source"
	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

// Board is what the poller saw in its last successful cycle.
type Board struct {
	UpdatedAt time.Time `json:"updatedAt"`
	Sources   []string  `json:"sources"` // the sources that answered, in priority order
	Lines     []Line    `json:"lines"`   // sorted by code
}

// Line is one line as the sources reported it. Name and ColorHex are the
// sources' (the catalog is applied when serving); SeenAt is the last time any
// source reported the line.
type Line struct {
	Code        int          `json:"code"`
	Name        string       `json:"name"`
	ColorHex    string       `json:"colorHex"`
	State       status.State `json:"state"`
	Label       string       `json:"label"`
	Description string       `json:"description"`
	SeenAt      time.Time    `json:"seenAt"`
}

// KeepUnseen is how long a line no source reports (its only sources failed)
// keeps its last reading. After that it shows as unknown until a source
// reports it again; the notification memory is untouched meanwhile, so the
// return only notifies if the state changed.
const KeepUnseen = 10 * time.Minute

// Build makes the board of a cycle: the merged readings, seen now, plus the
// lines of the previous board that no source reported this time.
func Build(prev Board, merged []source.Reading, sources []string, now time.Time) Board {
	b := Board{UpdatedAt: now, Sources: sources}
	seen := map[int]bool{}
	for _, r := range merged {
		seen[r.Code] = true
		b.Lines = append(b.Lines, Line{
			Code: r.Code, Name: r.Name, ColorHex: r.ColorHex,
			State: r.State, Label: r.Label, Description: r.Description,
			SeenAt: now,
		})
	}
	for _, l := range prev.Lines {
		if seen[l.Code] {
			continue
		}
		if now.Sub(l.SeenAt) > KeepUnseen {
			l.State, l.Label, l.Description = status.Desconhecido, "", ""
		}
		b.Lines = append(b.Lines, l)
	}
	slices.SortFunc(b.Lines, func(x, y Line) int { return x.Code - y.Code })
	return b
}
