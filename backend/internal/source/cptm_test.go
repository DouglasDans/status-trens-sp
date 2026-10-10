package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

func TestCPTMParsesRealResponse(t *testing.T) {
	srv := serve(t, "application/json; charset=utf-8", http.StatusOK, fixture(t, "cptm.json"))

	snap, err := NewCPTM(srv.URL, srv.Client()).Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snap.Source != "cptm" {
		t.Errorf("source = %q, want cptm", snap.Source)
	}
	if len(snap.Readings) != 11 {
		t.Fatalf("got %d readings, want 11", len(snap.Readings))
	}
	got := byCode(snap.Readings)[7]
	// CPTM sends no color (Cor is null).
	want := Reading{Code: 7, Name: "Rubi", State: status.Normal, Recognized: true, Label: "Operação Normal", Description: "Linha opera normalmente em toda sua extensão."}
	if got != want {
		t.Errorf("line 7 = %+v, want %+v", got, want)
	}
}

func TestCPTMSendsAcceptJSON(t *testing.T) {
	var accept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	if _, err := NewCPTM(srv.URL, srv.Client()).Fetch(context.Background()); err != nil {
		t.Fatal(err)
	}
	if accept != "application/json" {
		t.Errorf("Accept = %q, want application/json", accept)
	}
}

func TestCPTMNormalizesFreeText(t *testing.T) {
	body := []byte(`[{"LinhaId":9,"Status":"Atividade Programada","Descricao":"Maiores intervalos entre Osasco e Varginha."}]`)
	srv := serve(t, "application/json", http.StatusOK, body)

	snap, err := NewCPTM(srv.URL, srv.Client()).Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := snap.Readings[0]; got.State != status.Reduzida || !got.Recognized {
		t.Errorf("got %+v, want reduzida recognized", got)
	}
}

func TestCPTMErrors(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		statusCode  int
		body        []byte
	}{
		{"xml instead of json", "application/xml; charset=utf-8", http.StatusOK, fixture(t, "cptm.xml")},
		{"http error", "application/json", http.StatusBadGateway, []byte(`[]`)},
		{"invalid json", "application/json", http.StatusOK, []byte(`[{`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := serve(t, tt.contentType, tt.statusCode, tt.body)
			if _, err := NewCPTM(srv.URL, srv.Client()).Fetch(context.Background()); err == nil {
				t.Error("want error, got nil")
			}
		})
	}
}
