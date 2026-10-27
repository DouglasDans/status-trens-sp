package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/DouglasDans/status-trens-sp/backend/internal/board"
	"github.com/DouglasDans/status-trens-sp/backend/internal/catalog"
	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

// fakeBoards serves the board the poller would have saved.
type fakeBoards struct {
	board board.Board
	found bool
	err   error
	loads int
}

func (f *fakeBoards) LoadBoard(context.Context) (board.Board, bool, error) {
	f.loads++
	return f.board, f.found, f.err
}

// saved is a board saved at t0 by the given sources.
func saved(sources string, lines ...board.Line) *fakeBoards {
	return &fakeBoards{found: true, board: board.Board{UpdatedAt: t0, Sources: strings.Split(sources, ","), Lines: lines}}
}

type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time { return c.t }

var t0 = time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)

func newTestHandler(boards BoardLoader, clock *fakeClock, logs *bytes.Buffer) http.Handler {
	logger := slog.New(slog.NewTextHandler(logs, nil))
	return NewHandler(NewStatusService(boards, 30*time.Second, clock.Now, logger))
}

func get(t *testing.T, h http.Handler) (*httptest.ResponseRecorder, statusResponse) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/status", nil))
	var body statusResponse
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("invalid JSON: %v\n%s", err, rec.Body)
		}
	}
	return rec, body
}

// R16.1: the response is the saved board; no source is called.
func TestStatusMergesCatalogWithReadings(t *testing.T) {
	boards := saved("proximotrem",
		board.Line{Code: 9, State: status.Reduzida, Label: "Atividade Programada", Description: "Maiores intervalos.", SeenAt: t0})
	rec, body := get(t, newTestHandler(boards, &fakeClock{t0}, &bytes.Buffer{}))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("content type = %q", ct)
	}
	if body.Source != "proximotrem" || !body.UpdatedAt.Equal(t0) || body.Stale {
		t.Errorf("header fields = %+v", body)
	}
	if len(body.Lines) != 1 {
		t.Fatalf("got %d lines, want only the one the source reports", len(body.Lines))
	}
	line9 := body.Lines[0]
	entry, _ := catalog.Lookup(9)
	want := lineResponse{Code: 9, Name: "Esmeralda", ColorHex: entry.ColorHex, TextColorHex: entry.TextColorHex, Operator: "viamobilidade", OperatorName: "ViaMobilidade",
		Status: status.Reduzida, Label: "Atividade Programada", Description: "Maiores intervalos."}
	if line9 != want {
		t.Errorf("line 9 = %+v, want %+v", line9, want)
	}
}

func TestStatusIncludesLinesTheCatalogDoesNotKnow(t *testing.T) {
	boards := saved("proximotrem",
		board.Line{Code: 18, Name: "Bronze", ColorHex: "#a0522d", State: status.Normal, Label: "Operação Normal"},
		board.Line{Code: 19, State: status.Normal, Label: "Operação Normal"})
	_, body := get(t, newTestHandler(boards, &fakeClock{t0}, &bytes.Buffer{}))

	if len(body.Lines) != 2 {
		t.Fatalf("got %d lines, want the 2 the source reports", len(body.Lines))
	}
	last := body.Lines
	want := []lineResponse{
		{Code: 18, Name: "Bronze", ColorHex: "#a0522d", Status: status.Normal, Label: "Operação Normal"},
		{Code: 19, Name: "Linha 19", ColorHex: catalog.FallbackColor, Status: status.Normal, Label: "Operação Normal"},
	}
	for i := range want {
		if last[i] != want[i] {
			t.Errorf("new line = %+v, want %+v (sorted by code, source data, no operator)", last[i], want[i])
		}
	}
}

func TestStatusPrefersCatalogMetadataOverTheSource(t *testing.T) {
	boards := saved("proximotrem", board.Line{Code: 6, Name: "LARANJA raw", ColorHex: "#000000", State: status.Normal})
	_, body := get(t, newTestHandler(boards, &fakeClock{t0}, &bytes.Buffer{}))

	entry, _ := catalog.Lookup(6)
	for _, l := range body.Lines {
		if l.Code == 6 && (l.Name != entry.Name || l.ColorHex != entry.ColorHex) {
			t.Errorf("line 6 = %+v, want catalog name %q and color %q", l, entry.Name, entry.ColorHex)
		}
	}
}

func TestStatusHidesCatalogLinesTheSourceDoesNotReport(t *testing.T) {
	// The catalog may hold lines prepared before they open; users only see a
	// line once a source reports it.
	boards := saved("cptm",
		board.Line{Code: 7, Name: "Rubi", State: status.Normal, Label: "Operação Normal"},
		board.Line{Code: 12, Name: "Safira", State: status.Normal, Label: "Operação Normal"})
	_, body := get(t, newTestHandler(boards, &fakeClock{t0}, &bytes.Buffer{}))

	var codes []int
	for _, l := range body.Lines {
		codes = append(codes, l.Code)
	}
	if !slices.Equal(codes, []int{7, 12}) {
		t.Errorf("lines = %v, want only the reported ones, sorted: [7 12]", codes)
	}
}

func TestStatusCachesWithinTTL(t *testing.T) {
	boards := saved("proximotrem")
	clock := &fakeClock{t0}
	h := newTestHandler(boards, clock, &bytes.Buffer{})

	get(t, h)
	clock.t = t0.Add(29 * time.Second)
	get(t, h)
	if boards.loads != 1 {
		t.Fatalf("loads within TTL = %d, want 1", boards.loads)
	}

	clock.t = t0.Add(30 * time.Second)
	get(t, h)
	if boards.loads != 2 {
		t.Errorf("loads after TTL = %d, want 2", boards.loads)
	}
}

// R23.1
func TestStatusSourceListsTheSourcesThatAnswered(t *testing.T) {
	_, body := get(t, newTestHandler(saved("proximotrem,metro,cptm"), &fakeClock{t0}, &bytes.Buffer{}))

	if body.Source != "proximotrem,metro,cptm" {
		t.Errorf("source = %q, want proximotrem,metro,cptm", body.Source)
	}
}

// R17.1-R17.3, R17.5: stale once the board is more than 3 min old.
func TestStatusStaleness(t *testing.T) {
	tests := []struct {
		name  string
		at    time.Time
		stale bool
	}{
		{"R17.1: fresh below the threshold", t0.Add(2*time.Minute + 59*time.Second), false},
		{"R17.3: exactly at the threshold", t0.Add(StaleAfter), false},
		{"R17.2: stale above the threshold", t0.Add(3*time.Minute + time.Second), true},
		{"R17.5: a board from the future is not an error", t0.Add(-5 * time.Second), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, body := get(t, newTestHandler(saved("proximotrem"), &fakeClock{tt.at}, &bytes.Buffer{}))
			if rec.Code != http.StatusOK || body.Stale != tt.stale || !body.UpdatedAt.Equal(t0) {
				t.Errorf("status %d, stale = %v, updatedAt = %s; want stale %v and t0", rec.Code, body.Stale, body.UpdatedAt, tt.stale)
			}
		})
	}
}

// R17.4: the cache never freezes stale: false.
func TestStatusCachedResponseBecomesStale(t *testing.T) {
	clock := &fakeClock{t0.Add(2*time.Minute + 50*time.Second)}
	h := newTestHandler(saved("proximotrem"), clock, &bytes.Buffer{})
	if _, body := get(t, h); body.Stale {
		t.Fatal("want fresh first")
	}

	clock.t = t0.Add(3*time.Minute + 20*time.Second) // still within the 30 s cache
	if _, body := get(t, h); !body.Stale {
		t.Error("want stale: the board is now older than the threshold")
	}
}

// R16.3: a failed read serves the instance's last board, marked stale.
func TestStatusServesStaleDataWhenTheStoreFails(t *testing.T) {
	boards := saved("proximotrem", board.Line{Code: 1, State: status.Normal, Label: "Operação Normal"})
	clock := &fakeClock{t0}
	h := newTestHandler(boards, clock, &bytes.Buffer{})
	get(t, h)

	boards.err = errors.New("dynamodb unavailable")
	clock.t = t0.Add(time.Minute)
	rec, body := get(t, h)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 with stale data", rec.Code)
	}
	if !body.Stale || !body.UpdatedAt.Equal(t0) || body.Lines[0].Status != status.Normal {
		t.Errorf("got %+v, want stale data from t0", body)
	}
}

// R16.2, R16.3, R16.4: no board at all is a 503 with a JSON error.
func TestStatusFailsWhenThereIsNoData(t *testing.T) {
	tests := []struct {
		name   string
		boards *fakeBoards
	}{
		{"R16.2: nothing saved yet", &fakeBoards{}},
		{"R16.3: store down and nothing cached", &fakeBoards{err: errors.New("dynamodb unavailable")}},
		{"R16.4: corrupt board", &fakeBoards{err: errors.New("decoding board: unexpected end of JSON input")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, _ := get(t, newTestHandler(tt.boards, &fakeClock{t0}, &bytes.Buffer{}))

			if rec.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want 503", rec.Code)
			}
			var body map[string]string
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["error"] == "" {
				t.Errorf("body = %s, want JSON with an error message", rec.Body)
			}
		})
	}
}

func TestStatusRejectsOtherMethods(t *testing.T) {
	h := newTestHandler(&fakeBoards{}, &fakeClock{t0}, &bytes.Buffer{})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/status", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}
