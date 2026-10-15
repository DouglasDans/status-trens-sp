package main

import (
	"testing"

	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

func TestParseSeed(t *testing.T) {
	code, state, err := parseSeed("3=paralisada")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 3 || state != status.Paralisada {
		t.Errorf("got %d=%q, want 3=%q", code, state, status.Paralisada)
	}
}

func TestParseSeedRejectsInvalidInput(t *testing.T) {
	for _, in := range []string{
		"",
		"3",
		"=paralisada",
		"tres=paralisada",
		"0=paralisada",
		"-1=paralisada",
		"3=",
		"3=quebrada",
		"3=desconhecido", // carries no information: Next would ignore the reading
	} {
		if _, _, err := parseSeed(in); err == nil {
			t.Errorf("parseSeed(%q): expected an error", in)
		}
	}
}
