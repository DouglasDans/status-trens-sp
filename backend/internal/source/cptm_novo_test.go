package source

import (
	"context"
	"net/http"
	"testing"

	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

// R21.1
func TestCPTMNovoParsesRealResponse(t *testing.T) {
	srv := serve(t, "application/json; charset=utf-8", http.StatusOK, fixture(t, "cptm_novo.json"))

	snap, err := NewCPTMNovo(srv.URL, srv.Client()).Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snap.Source != "cptm-novo" {
		t.Errorf("source = %q, want cptm-novo", snap.Source)
	}
	if len(snap.Readings) != 4 {
		t.Fatalf("got %d readings, want 4", len(snap.Readings))
	}
	// The endpoint sends neither name nor color: the catalog fills them.
	got := byCode(snap.Readings)[13]
	want := Reading{Code: 13, State: status.Normal, Recognized: true, Label: "Operação Normal"}
	if got != want {
		t.Errorf("line 13 = %+v, want %+v", got, want)
	}
}

// R21.2: the real response carries a version parameter in its content type.
func TestCPTMNovoAcceptsVersionedJSONContentType(t *testing.T) {
	srv := serve(t, "application/json; charset=utf-8; x-api-version=1", http.StatusOK, fixture(t, "cptm_novo.json"))

	if _, err := NewCPTMNovo(srv.URL, srv.Client()).Fetch(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestCPTMNovoKeepsTheOperatorDescription(t *testing.T) {
	body := []byte(`[{"linhaId":11,"status":"Velocidade Reduzida","descricao":"Trens com maior tempo de parada em Guaianases."}]`)
	srv := serve(t, "application/json", http.StatusOK, body)

	snap, err := NewCPTMNovo(srv.URL, srv.Client()).Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got := snap.Readings[0]
	if got.State != status.Reduzida || got.Label != "Velocidade Reduzida" || got.Description != "Trens com maior tempo de parada em Guaianases." {
		t.Errorf("got %+v", got)
	}
}

func TestCPTMNovoErrors(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		statusCode  int
		body        []byte
	}{
		{"html instead of json", "text/html", http.StatusOK, []byte(`<html></html>`)},
		{"http error", "application/json", http.StatusInternalServerError, []byte(`[]`)},
		{"invalid json", "application/json", http.StatusOK, []byte(`[{`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := serve(t, tt.contentType, tt.statusCode, tt.body)
			if _, err := NewCPTMNovo(srv.URL, srv.Client()).Fetch(context.Background()); err == nil {
				t.Error("want error, got nil")
			}
		})
	}
}
