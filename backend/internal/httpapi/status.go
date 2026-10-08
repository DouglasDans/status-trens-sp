// Package httpapi exposes the HTTP API consumed by the app.
package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/DouglasDans/status-trens-sp/backend/internal/catalog"
	"github.com/DouglasDans/status-trens-sp/backend/internal/source"
	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

type statusResponse struct {
	UpdatedAt time.Time      `json:"updatedAt"`
	Source    string         `json:"source"`
	Stale     bool           `json:"stale"` // true when sources failed and this is the last known data
	Lines     []lineResponse `json:"lines"`
}

type lineResponse struct {
	Code        int          `json:"code"`
	Name        string       `json:"name"`
	ColorHex    string       `json:"colorHex"`
	Operator    string       `json:"operator"`
	Status      status.State `json:"status"`
	Label       string       `json:"label"`
	Description string       `json:"description"`
}

// StatusService fetches from the source at most once per ttl, so many app
// clients never translate into many requests to a third-party endpoint.
// When the source fails it keeps serving the last known data, marked stale.
type StatusService struct {
	source source.StatusSource
	ttl    time.Duration
	now    func() time.Time
	logger *slog.Logger

	mu   sync.Mutex
	last *statusResponse
}

func NewStatusService(src source.StatusSource, ttl time.Duration, now func() time.Time, logger *slog.Logger) *StatusService {
	return &StatusService{source: src, ttl: ttl, now: now, logger: logger}
}

func (s *StatusService) current(ctx context.Context) (*statusResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.last != nil && s.now().Sub(s.last.UpdatedAt) < s.ttl {
		return s.last, nil
	}

	snap, err := s.source.Fetch(ctx)
	if err != nil {
		if s.last == nil {
			return nil, err
		}
		s.logger.Warn("serving stale status", "error", err, "updatedAt", s.last.UpdatedAt)
		stale := *s.last
		stale.Stale = true
		return &stale, nil
	}

	s.last = s.build(snap)
	return s.last, nil
}

func (s *StatusService) build(snap source.Snapshot) *statusResponse {
	readings := make(map[int]source.Reading, len(snap.Readings))
	for _, r := range snap.Readings {
		if !r.Recognized {
			s.logger.Warn("unrecognized status label", "source", snap.Source, "line", r.Code, "label", r.Label)
		}
		readings[r.Code] = r
	}

	lines := make([]lineResponse, 0, len(catalog.Lines))
	for _, l := range catalog.Lines {
		r, found := readings[l.Code]
		if !found {
			r = source.Reading{State: status.Desconhecido}
		}
		lines = append(lines, lineResponse{
			Code:        l.Code,
			Name:        l.Name,
			ColorHex:    l.ColorHex,
			Operator:    l.Operator,
			Status:      r.State,
			Label:       r.Label,
			Description: r.Description,
		})
	}
	return &statusResponse{UpdatedAt: s.now().UTC(), Source: snap.Source, Lines: lines}
}

// NewHandler routes the API.
func NewHandler(svc *StatusService) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /status", func(w http.ResponseWriter, r *http.Request) {
		resp, err := svc.current(r.Context())
		if err != nil {
			svc.logger.Error("fetching status", "error", err)
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
