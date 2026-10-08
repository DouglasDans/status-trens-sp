package status

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		name   string
		label  string
		want   State
		wantOK bool
	}{
		// Real labels observed on 2026-10-04.
		{"real: normal", "Operação Normal", Normal, true},
		{"real: encerrada", "Operação Encerrada", Encerrada, true},
		{"real: atividade programada", "Atividade Programada", Reduzida, true},

		// Case, accents and whitespace.
		{"uppercase", "OPERAÇÃO NORMAL", Normal, true},
		{"no accents", "operacao encerrada", Encerrada, true},
		{"surrounding whitespace", "  Operação Normal \n", Normal, true},
		{"inner whitespace", "Operação   Normal", Normal, true},
		{"non-breaking space", "Operação Normal", Normal, true},
		{"decomposed accents (NFD)", "Operação Normal", Normal, true},
		{"trailing punctuation", "Operação Normal.", Normal, true},

		// Keywords per state.
		{"paralisada", "Paralisada", Paralisada, true},
		{"paralisacao", "Paralisação", Paralisada, true},
		{"interrompida", "Circulação interrompida", Paralisada, true},
		{"suspensa", "Operação suspensa", Paralisada, true},
		{"sem circulacao", "Sem circulação", Paralisada, true},
		{"parcial", "Operação Parcial", Parcial, true},
		{"velocidade reduzida", "Velocidade Reduzida", Reduzida, true},
		{"maior tempo", "Maior tempo de viagem", Reduzida, true},
		{"maiores intervalos", "Trens circulando com maiores intervalos", Reduzida, true},
		{"lentidao", "Lentidão", Reduzida, true},
		{"diferenciada", "Operação Diferenciada", Reduzida, true},
		{"restricao", "Operação com restrições", Reduzida, true},
		{"fora de operacao", "Fora de operação", Encerrada, true},
		{"fechada", "Linha fechada", Encerrada, true},
		{"nao opera", "Não opera", Encerrada, true},

		// Keywords match at the start of a word, not anywhere inside it.
		{"anormal is not normal", "Anormal", Desconhecido, false},
		{"subnormal is not normal", "Subnormal", Desconhecido, false},

		// Several keywords: the most severe state wins.
		{"paralisacao parcial", "Paralisação parcial", Paralisada, true},
		{"reduzida e parcial", "Velocidade reduzida e operação parcial", Parcial, true},
		{"parcial e reduzida, any order", "Operação parcial e velocidade reduzida", Parcial, true},
		{"normal com restricao", "Operação normal com restrição", Reduzida, true},
		{"encerrada por paralisacao", "Operação encerrada por paralisação", Paralisada, true},

		// Accepted risk: negation is not understood (see Notion, Fontes de Dados).
		{"negation is not handled", "Operação normal, sem paralisações", Paralisada, true},

		// Unmapped or empty.
		{"unrelated text", "Foo bar", Desconhecido, false},
		{"empty", "", Desconhecido, false},
		{"whitespace only", " \t\n ", Desconhecido, false},
		{"invalid utf-8", "\xff\xfe", Desconhecido, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := Normalize(tt.label)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("Normalize(%q) = (%q, %v), want (%q, %v)", tt.label, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestStateValuesAreStable(t *testing.T) {
	// These strings are the GET /status JSON contract.
	want := map[State]string{
		Normal:       "normal",
		Reduzida:     "reduzida",
		Parcial:      "parcial",
		Paralisada:   "paralisada",
		Encerrada:    "encerrada",
		Desconhecido: "desconhecido",
	}
	for state, value := range want {
		if string(state) != value {
			t.Errorf("state %q, want %q", state, value)
		}
	}
}
