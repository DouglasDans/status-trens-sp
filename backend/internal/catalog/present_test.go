package catalog

import "testing"

func TestPresentPrefersTheCatalog(t *testing.T) {
	l := Lines[0]

	got := Present(l.Code, "NOME DA FONTE", "#000000")

	want := Display{Name: l.Name, ColorHex: l.ColorHex, TextColorHex: l.TextColorHex, Operator: l.Operator, OperatorName: Operators[l.Operator], Known: true}
	if got != want {
		t.Errorf("Present() = %+v, want %+v", got, want)
	}
}

func TestPresentFallsBackToTheSourceThenToDefaults(t *testing.T) {
	const unknown = 9999
	tests := []struct {
		name, srcName, srcColor string
		want                    Display
	}{
		{"source fills the gaps", "Bronze", "#b08d57", Display{Name: "Bronze", ColorHex: "#b08d57"}},
		{"nothing known", "", "", Display{Name: "Linha 9999", ColorHex: FallbackColor}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Present(unknown, tt.srcName, tt.srcColor); got != tt.want {
				t.Errorf("Present() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
