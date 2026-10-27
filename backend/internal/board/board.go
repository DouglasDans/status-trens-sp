// Package board holds the merged status of every line: the poller writes it
// each cycle and GET /status serves it, so the API never calls a source.
package board

import (
	"time"

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
