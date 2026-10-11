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

	"github.com/DouglasDans/status-trens-sp/backend/internal/catalog"
	"github.com/DouglasDans/status-trens-sp/backend/internal/source"
	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

type fakeSource struct {
	snap  source.Snapshot
	err   error
	calls int
}

func (f *fakeSource) Fetch(context.Context) (source.Snapshot, error) {
	f.calls++
	return f.snap, f.err
}

type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time { return c.t }

var t0 = time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)

func newTestHandler(src source.StatusSource, clock *fakeClock, logs *bytes.Buffer) http.Handler {
	logger := slog.New(slog.NewTextHandler(logs, nil))
	return NewHandler(NewStatusService(src, 30*time.Second, clock.Now, logger))
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

func TestStatusMergesCatalogWithReadings(t *testing.T) {
	src := &fakeSource{snap: source.Snapshot{Source: "proximotrem", Readings: []source.Reading{
		{Code: 9, State: status.Reduzida, Recognized: true, Label: "Atividade Programada", Description: "Maiores intervalos."},
	}}}
	rec, body := get(t, newTestHandler(src, &fakeClock{t0}, &bytes.Buffer{}))

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
	src := &fakeSource{snap: source.Snapshot{Source: "proximotrem", Readings: []source.Reading{
		{Code: 18, Name: "Bronze", ColorHex: "#a0522d", State: status.Normal, Recognized: true, Label: "Operação Normal"},
		{Code: 19, Name: "", ColorHex: "", State: status.Normal, Recognized: true, Label: "Operação Normal"},
	}}}
	_, body := get(t, newTestHandler(src, &fakeClock{t0}, &bytes.Buffer{}))

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
	src := &fakeSource{snap: source.Snapshot{Source: "proximotrem", Readings: []source.Reading{
		{Code: 6, Name: "LARANJA raw", ColorHex: "#000000", State: status.Normal, Recognized: true},
	}}}
	_, body := get(t, newTestHandler(src, &fakeClock{t0}, &bytes.Buffer{}))

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
	src := &fakeSource{snap: source.Snapshot{Source: "cptm", Readings: []source.Reading{
		{Code: 12, Name: "Safira", State: status.Normal, Recognized: true, Label: "Operação Normal"},
		{Code: 7, Name: "Rubi", State: status.Normal, Recognized: true, Label: "Operação Normal"},
	}}}
	_, body := get(t, newTestHandler(src, &fakeClock{t0}, &bytes.Buffer{}))

	var codes []int
	for _, l := range body.Lines {
		codes = append(codes, l.Code)
	}
	if !slices.Equal(codes, []int{7, 12}) {
		t.Errorf("lines = %v, want only the reported ones, sorted: [7 12]", codes)
	}
}

func TestStatusCachesWithinTTL(t *testing.T) {
	src := &fakeSource{snap: source.Snapshot{Source: "proximotrem"}}
	clock := &fakeClock{t0}
	h := newTestHandler(src, clock, &bytes.Buffer{})

	get(t, h)
	clock.t = t0.Add(29 * time.Second)
	get(t, h)
	if src.calls != 1 {
		t.Fatalf("calls within TTL = %d, want 1", src.calls)
	}

	clock.t = t0.Add(30 * time.Second)
	get(t, h)
	if src.calls != 2 {
		t.Errorf("calls after TTL = %d, want 2", src.calls)
	}
}

func TestStatusServesStaleDataWhenSourcesFail(t *testing.T) {
	src := &fakeSource{snap: source.Snapshot{Source: "proximotrem", Readings: []source.Reading{
		{Code: 1, State: status.Normal, Recognized: true, Label: "Operação Normal"},
	}}}
	clock := &fakeClock{t0}
	h := newTestHandler(src, clock, &bytes.Buffer{})
	get(t, h)

	src.err = errors.New("all sources failed")
	clock.t = t0.Add(time.Minute)
	rec, body := get(t, h)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 with stale data", rec.Code)
	}
	if !body.Stale || !body.UpdatedAt.Equal(t0) || body.Lines[0].Status != status.Normal {
		t.Errorf("got %+v, want stale data from t0", body)
	}
}

func TestStatusFailsWhenSourcesFailAndThereIsNoData(t *testing.T) {
	src := &fakeSource{err: errors.New("all sources failed")}
	rec, _ := get(t, newTestHandler(src, &fakeClock{t0}, &bytes.Buffer{}))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["error"] == "" {
		t.Errorf("body = %s, want JSON with an error message", rec.Body)
	}
}

func TestStatusLogsUnrecognizedLabels(t *testing.T) {
	src := &fakeSource{snap: source.Snapshot{Source: "proximotrem", Readings: []source.Reading{
		{Code: 3, State: status.Desconhecido, Recognized: false, Label: "Rótulo Novo"},
		{Code: 1, State: status.Normal, Recognized: true, Label: "Operação Normal"},
	}}}
	var logs bytes.Buffer
	get(t, newTestHandler(src, &fakeClock{t0}, &logs))

	out := logs.String()
	if !strings.Contains(out, "Rótulo Novo") || !strings.Contains(out, "proximotrem") {
		t.Errorf("logs = %q, want the unrecognized label and its source", out)
	}
	if strings.Contains(out, "Operação Normal") {
		t.Errorf("logs = %q, recognized labels must not be logged", out)
	}
}

func TestStatusRejectsOtherMethods(t *testing.T) {
	h := newTestHandler(&fakeSource{}, &fakeClock{t0}, &bytes.Buffer{})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/status", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}
