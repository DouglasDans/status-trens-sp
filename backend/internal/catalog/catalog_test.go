package catalog

import (
	"regexp"
	"testing"
)

func TestCatalogHasTheFifteenLines(t *testing.T) {
	want := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 15, 17}
	if len(Lines) != len(want) {
		t.Fatalf("got %d lines, want %d", len(Lines), len(want))
	}
	for i, code := range want {
		if Lines[i].Code != code {
			t.Errorf("Lines[%d].Code = %d, want %d (catalog must be ordered by code)", i, Lines[i].Code, code)
		}
	}
}

func TestCatalogEntriesAreComplete(t *testing.T) {
	hex := regexp.MustCompile(`^#[0-9a-f]{6}$`)
	operators := map[string]bool{
		"metro": true, "viaquatro": true, "viamobilidade": true,
		"linhauni": true, "tictrens": true, "cptm": true, "trivia": true,
	}
	for _, l := range Lines {
		if l.Name == "" {
			t.Errorf("line %d has no name", l.Code)
		}
		if !hex.MatchString(l.ColorHex) {
			t.Errorf("line %d color %q is not #rrggbb", l.Code, l.ColorHex)
		}
		if !operators[l.Operator] {
			t.Errorf("line %d operator %q is not a known operator id", l.Code, l.Operator)
		}
	}
}
