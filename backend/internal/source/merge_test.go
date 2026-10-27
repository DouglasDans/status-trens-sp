package source

import (
	"testing"

	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

// vote is one source's reading of line 3, the line most cases use.
func vote(src string, s status.State, label, desc string) Snapshot {
	return Snapshot{Source: src, Readings: []Reading{{Code: 3, State: s, Recognized: s != status.Desconhecido, Label: label, Description: desc}}}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name      string
		snaps     []Snapshot // in priority order
		wantState status.State
		wantLabel string
		wantDesc  string
		wantFrom  string
	}{
		{"R6.1: one slowdown beats normals", []Snapshot{
			vote("proximotrem", status.Normal, "Operação Normal", ""),
			vote("cptm", status.Reduzida, "Atividade Programada", "Maiores intervalos."),
			vote("metro", status.Normal, "Operação Normal", ""),
		}, status.Reduzida, "Atividade Programada", "Maiores intervalos.", "cptm"},
		{"R6.2: stopped beats partial", []Snapshot{
			vote("proximotrem", status.Parcial, "Operação Parcial", ""),
			vote("cptm", status.Paralisada, "Paralisada", ""),
		}, status.Paralisada, "Paralisada", "", "cptm"},
		{"R6.3: partial beats slowdown", []Snapshot{
			vote("proximotrem", status.Reduzida, "Velocidade Reduzida", ""),
			vote("cptm", status.Parcial, "Operação Parcial", ""),
		}, status.Parcial, "Operação Parcial", "", "cptm"},
		{"R6.5: worst wins within a single source", []Snapshot{
			{Source: "proximotrem", Readings: []Reading{
				{Code: 3, State: status.Normal, Recognized: true, Label: "Operação Normal"},
				{Code: 3, State: status.Parcial, Recognized: true, Label: "Operação Parcial"},
			}},
		}, status.Parcial, "Operação Parcial", "", "proximotrem"},
		{"R7.1: normal only when every voter says normal", []Snapshot{
			vote("proximotrem", status.Normal, "Operação Normal", ""),
			vote("cptm", status.Normal, "Operação Normal", ""),
		}, status.Normal, "Operação Normal", "", "proximotrem"},
		{"R7.2: stays stopped while one source still reports it", []Snapshot{
			vote("proximotrem", status.Normal, "Operação Normal", ""),
			vote("cptm", status.Paralisada, "Paralisada", "Falha de energia entre Luz e Brás."),
		}, status.Paralisada, "Paralisada", "Falha de energia entre Luz e Brás.", "cptm"},
		{"R8.1/R8.2: a source without the line does not vote", []Snapshot{
			vote("proximotrem", status.Reduzida, "Velocidade Reduzida", ""),
			{Source: "cptm", Readings: []Reading{{Code: 9, State: status.Normal, Recognized: true, Label: "Operação Normal"}}},
		}, status.Reduzida, "Velocidade Reduzida", "", "proximotrem"},
		{"R9.1: normal beats closed", []Snapshot{
			vote("proximotrem", status.Encerrada, "Operação Encerrada", ""),
			vote("metro", status.Normal, "Operação Normal", "Situação Normal"),
		}, status.Normal, "Operação Normal", "Situação Normal", "metro"},
		{"R9.2: a problem beats closed and normal", []Snapshot{
			vote("proximotrem", status.Encerrada, "Operação Encerrada", ""),
			vote("metro", status.Normal, "Operação Normal", ""),
			vote("cptm", status.Reduzida, "Velocidade Reduzida", ""),
		}, status.Reduzida, "Velocidade Reduzida", "", "cptm"},
		{"R9.3: closed when every voter says closed", []Snapshot{
			vote("proximotrem", status.Encerrada, "Operação Encerrada", ""),
			vote("metro", status.Encerrada, "Operação Encerrada", ""),
		}, status.Encerrada, "Operação Encerrada", "", "proximotrem"},
		{"R10.1: unknown does not vote against normal", []Snapshot{
			vote("proximotrem", status.Normal, "Operação Normal", ""),
			vote("cptm", status.Desconhecido, "Operação Especial", "Texto novo."),
		}, status.Normal, "Operação Normal", "", "proximotrem"},
		{"R10.2: unknown when every vote is unknown", []Snapshot{
			vote("proximotrem", status.Desconhecido, "Operação Especial", ""),
			vote("cptm", status.Desconhecido, "Situação Atípica", ""),
		}, status.Desconhecido, "Operação Especial", "", "proximotrem"},
		{"R11.2: tie goes to the source with a description", []Snapshot{
			vote("proximotrem", status.Reduzida, "Atividade Programada", ""),
			vote("cptm", status.Reduzida, "Velocidade Reduzida", "Maiores intervalos entre Luz e Brás."),
		}, status.Reduzida, "Velocidade Reduzida", "Maiores intervalos entre Luz e Brás.", "cptm"},
		{"R11.2: then to the source with higher priority", []Snapshot{
			vote("proximotrem", status.Reduzida, "Atividade Programada", "Obras."),
			vote("cptm", status.Reduzida, "Velocidade Reduzida", "Maiores intervalos."),
		}, status.Reduzida, "Atividade Programada", "Obras.", "proximotrem"},
		{"R11.3: tie between normal sources", []Snapshot{
			vote("proximotrem", status.Normal, "Operação Normal", ""),
			vote("metro", status.Normal, "Operação Normal", "Situação Normal"),
		}, status.Normal, "Operação Normal", "Situação Normal", "metro"},
		{"R11.4: never mixes a losing source's description", []Snapshot{
			vote("proximotrem", status.Normal, "Operação Normal", "Linha opera normalmente."),
			vote("cptm", status.Paralisada, "Paralisada", ""),
		}, status.Paralisada, "Paralisada", "", "cptm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, ok := byCode(Merge(tt.snaps))[3]
			if !ok {
				t.Fatal("line 3 missing from the merge")
			}
			if r.State != tt.wantState || r.Label != tt.wantLabel || r.Description != tt.wantDesc || r.Source != tt.wantFrom {
				t.Errorf("got %s %q %q from %q, want %s %q %q from %q",
					r.State, r.Label, r.Description, r.Source, tt.wantState, tt.wantLabel, tt.wantDesc, tt.wantFrom)
			}
		})
	}
}

// R6.4: the state never depends on the order the sources answered in.
func TestMergeStateDoesNotDependOnOrder(t *testing.T) {
	a := vote("proximotrem", status.Normal, "Operação Normal", "")
	b := vote("cptm", status.Reduzida, "Velocidade Reduzida", "")
	c := vote("metro", status.Encerrada, "Operação Encerrada", "")
	for _, order := range [][]Snapshot{{a, b, c}, {a, c, b}, {b, a, c}, {b, c, a}, {c, a, b}, {c, b, a}} {
		if got := Merge(order)[0].State; got != status.Reduzida {
			t.Errorf("order %s,%s,%s: got %s, want reduzida", order[0].Source, order[1].Source, order[2].Source, got)
		}
	}
}

// R8.4: the lines are the union of the sources that answered, sorted.
func TestMergeIsTheUnionOfLines(t *testing.T) {
	got := Merge([]Snapshot{
		{Source: "metro", Readings: []Reading{{Code: 17, State: status.Encerrada}, {Code: 15, State: status.Normal}}},
		{Source: "cptm-novo", Readings: []Reading{{Code: 13, State: status.Normal}}},
	})
	var codes []int
	for _, r := range got {
		codes = append(codes, r.Code)
	}
	if len(codes) != 3 || codes[0] != 13 || codes[1] != 15 || codes[2] != 17 {
		t.Errorf("codes = %v, want [13 15 17]", codes)
	}
}

// R19.2: a line's name and color come from the first source that has them,
// whichever source decided the state.
func TestMergeNameAndColorOfALineOutsideTheCatalog(t *testing.T) {
	got := Merge([]Snapshot{
		{Source: "proximotrem", Readings: []Reading{{Code: 18, Name: "Bronze", State: status.Normal, Label: "Operação Normal"}}},
		{Source: "metro", Readings: []Reading{{Code: 18, Name: "Bronze Metrô", ColorHex: "#a0522d", State: status.Paralisada, Label: "Paralisada"}}},
	})[0]
	if got.Name != "Bronze" || got.ColorHex != "#a0522d" || got.State != status.Paralisada {
		t.Errorf("got %+v, want Bronze #a0522d paralisada", got)
	}
}

func TestMergeOfNothingIsEmpty(t *testing.T) {
	if got := Merge(nil); len(got) != 0 {
		t.Errorf("got %+v, want no lines", got)
	}
}
