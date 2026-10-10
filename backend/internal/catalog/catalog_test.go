package catalog

import (
	"regexp"
	"testing"
)

// These tests check that the catalog data is consistent, not which lines or
// operators exist: both change over time and are edited as data.

func TestLinesAreSortedAndUnique(t *testing.T) {
	for i := 1; i < len(Lines); i++ {
		if Lines[i].Code <= Lines[i-1].Code {
			t.Errorf("Lines[%d].Code = %d after %d: must be sorted and unique", i, Lines[i].Code, Lines[i-1].Code)
		}
	}
}

func TestLineEntriesAreValid(t *testing.T) {
	hex := regexp.MustCompile(`^#[0-9a-f]{6}$`)
	for _, l := range Lines {
		if l.Name == "" {
			t.Errorf("line %d has no name", l.Code)
		}
		if !hex.MatchString(l.ColorHex) {
			t.Errorf("line %d color %q is not #rrggbb", l.Code, l.ColorHex)
		}
		if !hex.MatchString(l.TextColorHex) {
			t.Errorf("line %d text color %q is not #rrggbb", l.Code, l.TextColorHex)
		}
		if _, ok := Operators[l.Operator]; !ok {
			t.Errorf("line %d operator %q is missing from Operators", l.Code, l.Operator)
		}
	}
}

func TestOperatorsHaveNames(t *testing.T) {
	for id, name := range Operators {
		if id == "" || name == "" {
			t.Errorf("operator %q has an empty id or name", id)
		}
	}
}

func TestLookup(t *testing.T) {
	if l, ok := Lookup(9); !ok || l.Name != "Esmeralda" {
		t.Errorf("Lookup(9) = %+v, %v", l, ok)
	}
	if _, ok := Lookup(999); ok {
		t.Error("Lookup(999) found a line that does not exist")
	}
}
