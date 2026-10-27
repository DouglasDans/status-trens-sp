package source

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

func newTestMetro(url string, client *http.Client, logs *bytes.Buffer) *Metro {
	return NewMetro(url, client, slog.New(slog.NewTextHandler(logs, nil)))
}

// R20.1
func TestMetroParsesRecordedPage(t *testing.T) {
	srv := serve(t, "text/html; charset=UTF-8", http.StatusOK, fixture(t, "metro.html"))

	snap, err := newTestMetro(srv.URL, srv.Client(), &bytes.Buffer{}).Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snap.Source != "metro" {
		t.Errorf("source = %q, want metro", snap.Source)
	}
	lines := byCode(snap.Readings)
	if len(snap.Readings) != 7 {
		t.Fatalf("got %d readings, want 7 (1, 2, 3, 4, 5, 15, 17)", len(snap.Readings))
	}
	tests := []struct {
		code int
		want Reading
	}{
		// R20.2: the tooltip description is the operator's text, kept as is.
		{1, Reading{Code: 1, Name: "Azul", ColorHex: "#171796", State: status.Normal, Recognized: true, Label: "Operação Normal", Description: "Situação Normal"}},
		// A description that repeats the label is dropped, as for every source.
		{4, Reading{Code: 4, Name: "Amarela", ColorHex: "#fcc540", State: status.Normal, Recognized: true, Label: "Operação Normal"}},
		{17, Reading{Code: 17, Name: "Ouro", ColorHex: "#d58405", State: status.Encerrada, Recognized: true, Label: "Operação Encerrada"}},
	}
	for _, tt := range tests {
		if got := lines[tt.code]; got != tt.want {
			t.Errorf("line %d = %+v, want %+v", tt.code, got, tt.want)
		}
	}
}

func TestMetroReadsAProblemFromTheTooltip(t *testing.T) {
	page := metroPage(metroLine("3", "Vermelha", "Velocidade Reduzida",
		"<div class='title'>Metrô</div><div class='description'>Trens com maior tempo de parada entre Sé e Brás.</div><div class='date'>(atualizado em 26/10/2026 08:01)</div>"))
	srv := serve(t, "text/html", http.StatusOK, []byte(page))

	snap, err := newTestMetro(srv.URL, srv.Client(), &bytes.Buffer{}).Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got := snap.Readings[0]
	if got.State != status.Reduzida || got.Description != "Trens com maior tempo de parada entre Sé e Brás." {
		t.Errorf("got %+v", got)
	}
}

// R5.1: a layout change fails the source instead of reporting no lines.
func TestMetroFailsWhenTheListIsMissing(t *testing.T) {
	srv := serve(t, "text/html", http.StatusOK, []byte(`<!DOCTYPE html><html><body><p>Página em manutenção</p></body></html>`))

	if _, err := newTestMetro(srv.URL, srv.Client(), &bytes.Buffer{}).Fetch(context.Background()); err == nil {
		t.Error("want error, got nil")
	}
}

// R5.3: one broken line is dropped and logged; the others still count.
func TestMetroSkipsALineWithInvalidNumber(t *testing.T) {
	page := metroPage(
		metroLine("A", "Azul", "Operação Normal", ""),
		metroLine("2", "Verde", "Operação Normal", ""),
	)
	srv := serve(t, "text/html", http.StatusOK, []byte(page))
	var logs bytes.Buffer

	snap, err := newTestMetro(srv.URL, srv.Client(), &logs).Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Readings) != 1 || snap.Readings[0].Code != 2 {
		t.Errorf("readings = %+v, want only line 2", snap.Readings)
	}
	if !strings.Contains(logs.String(), "skipping line") {
		t.Errorf("logs = %q, want the skipped line logged", logs.String())
	}
}

func TestMetroFailsWhenNoLineIsValid(t *testing.T) {
	srv := serve(t, "text/html", http.StatusOK, []byte(metroPage(metroLine("", "", "", ""))))

	if _, err := newTestMetro(srv.URL, srv.Client(), &bytes.Buffer{}).Fetch(context.Background()); err == nil {
		t.Error("want error, got nil")
	}
}

func TestMetroErrors(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		statusCode  int
	}{
		{"http error", "text/html", http.StatusServiceUnavailable},
		{"json instead of html", "application/json", http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := serve(t, tt.contentType, tt.statusCode, fixture(t, "metro.html"))
			if _, err := newTestMetro(srv.URL, srv.Client(), &bytes.Buffer{}).Fetch(context.Background()); err == nil {
				t.Error("want error, got nil")
			}
		})
	}
}

func metroPage(lines ...string) string {
	return `<!DOCTYPE html><html><body><ol class="direto-metro list-group">` + strings.Join(lines, "") + `</ol></body></html>`
}

func metroLine(number, name, label, tooltip string) string {
	return `<li class="linha list-group-item">` +
		`<div class="linha-numero" style="background-color: #ed2e38">` + number + `</div>` +
		`<div class="linha-nome">` + name + `</div>` +
		`<div class="linha-situacao" title="` + label + `">` + label + `</div>` +
		`<div class="linha-info" data-bs-title="` + tooltip + `"></div>` +
		`</li>`
}
