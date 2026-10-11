package status

import (
	"strings"
	"testing"
)

// step is one poll: the state the source reported and whether a push is expected.
type step struct {
	state  State
	notify bool
}

func TestNext(t *testing.T) {
	tests := []struct {
		name  string
		steps []step
	}{
		// R1: the first reading of a line is only a baseline.
		{"first reading is a baseline", []step{{Reduzida, false}}},
		{"first reading of a stopped line is a baseline", []step{{Paralisada, false}}},

		// R2: a change of state notifies.
		{"same state again", []step{{Normal, false}, {Normal, false}}},
		{"normal to stopped", []step{{Normal, false}, {Paralisada, true}}},
		{"stopped back to normal", []step{{Normal, false}, {Paralisada, true}, {Normal, true}}},
		{"between problems", []step{{Normal, false}, {Reduzida, true}, {Paralisada, true}, {Parcial, true}}},
		{"opens with a problem", []step{{Encerrada, false}, {Reduzida, true}}},
		{"flapping notifies every change", []step{{Normal, false}, {Paralisada, true}, {Normal, true}, {Paralisada, true}}},

		// R3: ordinary closing and opening are silent...
		{"closing", []step{{Normal, false}, {Encerrada, false}}},
		{"closing with a problem", []step{{Normal, false}, {Paralisada, true}, {Encerrada, false}}},
		{"ordinary opening", []step{{Normal, false}, {Encerrada, false}, {Normal, false}}},
		{"opening after a baseline at night", []step{{Encerrada, false}, {Normal, false}}},
		// ...except when the last push was about a problem: who was warned must hear it is over.
		{"opening after a warned problem", []step{{Normal, false}, {Paralisada, true}, {Encerrada, false}, {Normal, true}}},
		{"opening after a warned slowdown", []step{{Normal, false}, {Reduzida, true}, {Encerrada, false}, {Normal, true}}},
		{"next opening after the all clear is silent again", []step{
			{Normal, false}, {Paralisada, true}, {Encerrada, false}, {Normal, true}, {Encerrada, false}, {Normal, false},
		}},
		{"problem already cleared before closing", []step{
			{Normal, false}, {Paralisada, true}, {Normal, true}, {Encerrada, false}, {Normal, false},
		}},
		{"baseline problem was never warned", []step{{Paralisada, false}, {Encerrada, false}, {Normal, false}}},

		// Unknown means no information: compare with the last known state.
		{"unknown is silent", []step{{Normal, false}, {Desconhecido, false}}},
		{"unknown in between hides nothing", []step{{Normal, false}, {Desconhecido, false}, {Normal, false}}},
		{"change across unknown notifies once", []step{{Normal, false}, {Desconhecido, false}, {Paralisada, true}}},
		{"unknown first, then the baseline", []step{{Desconhecido, false}, {Paralisada, false}}},
		{"unknown does not break the all clear", []step{
			{Normal, false}, {Paralisada, true}, {Encerrada, false}, {Desconhecido, false}, {Normal, true},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m Memory
			var trail []string
			for i, s := range tt.steps {
				var notify bool
				m, notify = Next(m, s.state)
				trail = append(trail, string(s.state))
				if notify != s.notify {
					t.Fatalf("step %d (%s): notify = %v, want %v", i, strings.Join(trail, " → "), notify, s.notify)
				}
			}
		})
	}
}

func TestNextRemembersTheLastKnownState(t *testing.T) {
	m, _ := Next(Memory{}, Normal)
	m, _ = Next(m, Desconhecido)

	if m.State != Normal {
		t.Errorf("State = %q, want %q", m.State, Normal)
	}
}
