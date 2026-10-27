package board

import (
	"testing"
	"time"

	"github.com/DouglasDans/status-trens-sp/backend/internal/source"
	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

var t0 = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)

func TestBuildTakesTheMergedReadings(t *testing.T) {
	merged := []source.Reading{
		{Code: 3, Name: "Vermelha", ColorHex: "#ed2e38", State: status.Paralisada, Label: "Paralisada", Description: "Falha de energia."},
	}

	got := Build(Board{}, merged, []string{"proximotrem", "cptm"}, t0)

	want := Board{UpdatedAt: t0, Sources: []string{"proximotrem", "cptm"}, Lines: []Line{
		{Code: 3, Name: "Vermelha", ColorHex: "#ed2e38", State: status.Paralisada, Label: "Paralisada", Description: "Falha de energia.", SeenAt: t0},
	}}
	if !equal(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestBuildKeepsAnUnseenLine(t *testing.T) {
	line6 := Line{Code: 6, Name: "Laranja", State: status.Paralisada, Label: "Paralisada", Description: "Falha.", SeenAt: t0}
	prev := Board{UpdatedAt: t0, Lines: []Line{line6}}
	merged := []source.Reading{{Code: 3, State: status.Normal, Label: "Operação Normal"}}

	tests := []struct {
		name  string
		after time.Duration
		want  Line
	}{
		// R25.1
		{"R25.1: last reading kept within 10 min", 9 * time.Minute, line6},
		{"exactly 10 min still kept", KeepUnseen, line6},
		// R25.2: unknown, the operator text dropped, the last sighting kept.
		{"R25.2: unknown after 10 min", KeepUnseen + time.Second, Line{Code: 6, Name: "Laranja", State: status.Desconhecido, SeenAt: t0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Build(prev, merged, []string{"cptm"}, t0.Add(tt.after))
			if len(got.Lines) != 2 || got.Lines[0].Code != 3 || got.Lines[1] != tt.want {
				t.Errorf("lines = %+v, want line 3 then %+v", got.Lines, tt.want)
			}
		})
	}
}

func TestBuildSortsTheLines(t *testing.T) {
	prev := Board{Lines: []Line{{Code: 15, State: status.Normal, SeenAt: t0}, {Code: 2, State: status.Normal, SeenAt: t0}}}
	merged := []source.Reading{{Code: 9, State: status.Normal}}

	got := Build(prev, merged, nil, t0)

	var codes []int
	for _, l := range got.Lines {
		codes = append(codes, l.Code)
	}
	if len(codes) != 3 || codes[0] != 2 || codes[1] != 9 || codes[2] != 15 {
		t.Errorf("codes = %v, want [2 9 15]", codes)
	}
}

func equal(a, b Board) bool {
	if !a.UpdatedAt.Equal(b.UpdatedAt) || len(a.Sources) != len(b.Sources) || len(a.Lines) != len(b.Lines) {
		return false
	}
	for i := range a.Sources {
		if a.Sources[i] != b.Sources[i] {
			return false
		}
	}
	for i := range a.Lines {
		if a.Lines[i] != b.Lines[i] {
			return false
		}
	}
	return true
}
