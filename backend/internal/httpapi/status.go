// Package httpapi exposes the HTTP API consumed by the app.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/DouglasDans/status-trens-sp/backend/internal/board"
	"github.com/DouglasDans/status-trens-sp/backend/internal/catalog"
	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

type statusResponse struct {
	UpdatedAt time.Time      `json:"updatedAt"`
	Source    string         `json:"source"` // the sources that answered, comma separated, in priority order
	Stale     bool           `json:"stale"`  // true when this is old data: the poller or the store is behind
	Lines     []lineResponse `json:"lines"`
}

type lineResponse struct {
	Code         int          `json:"code"`
	Name         string       `json:"name"`
	ColorHex     string       `json:"colorHex"`
	TextColorHex string       `json:"textColorHex"` // empty when unknown: the app picks by contrast
	Operator     string       `json:"operator"`     // id, empty when unknown
	OperatorName string       `json:"operatorName"` // empty when unknown
	Status       status.State `json:"status"`
	Label        string       `json:"label"`
	Description  string       `json:"description"`
}

// StaleAfter is how old the board may be before it is served as stale: the
// poller runs every minute, so this means more than two missed cycles.
const StaleAfter = 3 * time.Minute

// BoardLoader reads the board the poller saved.
type BoardLoader interface {
	LoadBoard(ctx context.Context) (board.Board, bool, error)
}

// StatusService serves the board the poller saved, reading it at most once
// per ttl. It never calls a source. When the store fails it keeps serving the
// last board it read, marked stale.
type StatusService struct {
	boards BoardLoader
	ttl    time.Duration
	now    func() time.Time
	logger *slog.Logger

	mu       sync.Mutex
	last     *board.Board
	lastRead time.Time
}

func NewStatusService(boards BoardLoader, ttl time.Duration, now func() time.Time, logger *slog.Logger) *StatusService {
	return &StatusService{boards: boards, ttl: ttl, now: now, logger: logger}
}

var errNoBoard = errors.New("no board saved yet")

func (s *StatusService) current(ctx context.Context) (*statusResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	if s.last != nil && now.Sub(s.lastRead) < s.ttl {
		return s.build(*s.last, now, false), nil
	}

	b, found, err := s.boards.LoadBoard(ctx)
	if err == nil && !found {
		err = errNoBoard
	}
	if err != nil {
		if s.last == nil {
			return nil, err
		}
		s.logger.Warn("serving stale status", "error", err, "updatedAt", s.last.UpdatedAt)
		return s.build(*s.last, now, true), nil
	}

	for _, l := range b.Lines {
		if _, known := catalog.Lookup(l.Code); !known {
			s.logger.Info("line missing from catalog", "line", l.Code, "name", l.Name)
		}
	}
	s.last, s.lastRead = &b, now
	return s.build(b, now, false), nil
}

// build presents the board's lines, sorted by code. The catalog only enriches
// them (it may hold lines prepared before they open, which stay hidden until
// a source reports them); the sources fill what it lacks. Staleness is worked
// out on every response, so the cache never freezes it.
func (s *StatusService) build(b board.Board, now time.Time, storeFailed bool) *statusResponse {
	lines := make([]lineResponse, 0, len(b.Lines))
	for _, l := range b.Lines {
		lines = append(lines, s.line(l))
	}
	slices.SortFunc(lines, func(a, b lineResponse) int { return a.Code - b.Code })
	return &statusResponse{
		UpdatedAt: b.UpdatedAt.UTC(),
		Source:    strings.Join(b.Sources, ","),
		Stale:     storeFailed || now.Sub(b.UpdatedAt) > StaleAfter,
		Lines:     lines,
	}
}

func (s *StatusService) line(l board.Line) lineResponse {
	d := catalog.Present(l.Code, l.Name, l.ColorHex)
	return lineResponse{
		Code:         l.Code,
		Name:         d.Name,
		ColorHex:     d.ColorHex,
		TextColorHex: d.TextColorHex,
		Operator:     d.Operator,
		OperatorName: d.OperatorName,
		Status:       l.State,
		Label:        l.Label,
		Description:  l.Description,
	}
}

// NewHandler routes the API.
func NewHandler(svc *StatusService) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /status", func(w http.ResponseWriter, r *http.Request) {
		resp, err := svc.current(r.Context())
		if err != nil {
			svc.logger.Error("reading status", "error", err)
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "status temporarily unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})
	return mux
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
