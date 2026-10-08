package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

func serve(t *testing.T, contentType string, statusCode int, body []byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(statusCode)
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func byCode(readings []Reading) map[int]Reading {
	m := make(map[int]Reading, len(readings))
	for _, r := range readings {
		m[r.Code] = r
	}
	return m
}

func TestProximoTremParsesRealResponse(t *testing.T) {
	srv := serve(t, "application/json", http.StatusOK, fixture(t, "proximotrem.json"))

	snap, err := NewProximoTrem(srv.URL, srv.Client()).Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snap.Source != "proximotrem" {
		t.Errorf("source = %q, want proximotrem", snap.Source)
	}
	if len(snap.Readings) != 15 {
		t.Fatalf("got %d readings, want 15", len(snap.Readings))
	}

	lines := byCode(snap.Readings)
	tests := []struct {
		code        int
		state       status.State
		description string
	}{
		{1, status.Normal, ""}, // description equal to the label is dropped
		{5, status.Normal, ""}, // null description
		{6, status.Encerrada, ""},
		{7, status.Normal, "Linha opera normalmente em toda sua extensão."},
		{17, status.Encerrada, ""},
	}
	for _, tt := range tests {
		got := lines[tt.code]
		if got.State != tt.state || !got.Recognized || got.Description != tt.description {
			t.Errorf("line %d = %+v, want state %q, recognized, description %q", tt.code, got, tt.state, tt.description)
		}
	}
}

func TestProximoTremStatusCodes(t *testing.T) {
	tests := []struct {
		name           string
		code, label    string
		wantState      status.State
		wantRecognized bool
	}{
		{"atividade programada", "AtividadeProgramada", "Atividade Programada", status.Reduzida, true},
		{"unknown code falls back to label", "VelocidadeReduzidaNova", "Velocidade Reduzida", status.Reduzida, true},
		{"unknown code and label", "Xyz", "Algo novo", status.Desconhecido, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := []byte(`{"Status":true,"Data":[{"Code":9,"StatusCode":"` + tt.code + `","StatusLabel":"` + tt.label + `","Description":null}]}`)
			srv := serve(t, "application/json", http.StatusOK, body)

			snap, err := NewProximoTrem(srv.URL, srv.Client()).Fetch(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			got := snap.Readings[0]
			if got.State != tt.wantState || got.Recognized != tt.wantRecognized || got.Label != tt.label {
				t.Errorf("got %+v, want state %q recognized %v label %q", got, tt.wantState, tt.wantRecognized, tt.label)
			}
		})
	}
}

func TestProximoTremErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{"http error", http.StatusInternalServerError, `{}`},
		{"envelope status false", http.StatusOK, `{"Status":false,"Message":"falha","Data":[]}`},
		{"invalid json", http.StatusOK, `{not json`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := serve(t, "application/json", tt.statusCode, []byte(tt.body))
			if _, err := NewProximoTrem(srv.URL, srv.Client()).Fetch(context.Background()); err == nil {
				t.Error("want error, got nil")
			}
		})
	}
}
