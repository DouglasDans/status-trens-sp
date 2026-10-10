// Package httpapi exposes the HTTP API consumed by the app.
package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
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

// fallbackColor is used for a line neither the catalog nor the source colors.
const fallbackColor = "#8e8e8a"

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

// build lists the lines the source reports, sorted by code. The catalog only
// enriches them (it may hold lines prepared before they open, which stay
// hidden until a source reports them); the source fills what it lacks.
func (s *StatusService) build(snap source.Snapshot) *statusResponse {
	lines := make([]lineResponse, 0, len(snap.Readings))
	for _, r := range snap.Readings {
		if !r.Recognized {
			s.logger.Warn("unrecognized status label", "source", snap.Source, "line", r.Code, "label", r.Label)
		}
		lines = append(lines, s.line(r))
	}
	slices.SortFunc(lines, func(a, b lineResponse) int { return a.Code - b.Code })
	return &statusResponse{UpdatedAt: s.now().UTC(), Source: snap.Source, Lines: lines}
}

func (s *StatusService) line(r source.Reading) lineResponse {
	entry, known := catalog.Lookup(r.Code)
	if !known {
		s.logger.Info("line missing from catalog", "line", r.Code, "name", r.Name)
	}
	return lineResponse{
		Code:         r.Code,
		Name:         firstNonEmpty(entry.Name, r.Name, fmt.Sprintf("Linha %d", r.Code)),
		ColorHex:     firstNonEmpty(entry.ColorHex, r.ColorHex, fallbackColor),
		TextColorHex: entry.TextColorHex,
		Operator:     entry.Operator,
		OperatorName: catalog.Operators[entry.Operator],
		Status:       r.State,
		Label:        r.Label,
		Description:  r.Description,
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
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
