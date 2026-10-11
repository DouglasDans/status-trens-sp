package status

// Memory is what is kept about a line between polls.
type Memory struct {
	State        State // last known state; empty before the first reading
	LastNotified State // state of the last push sent; empty when none was sent
}

// Next applies a new reading to a line's memory and reports whether the
// favorites of that line must be notified.
//
//   - The first known state is a baseline and notifies nothing.
//   - Unknown carries no information: the memory stays as it was.
//   - Closing never notifies, and neither does an ordinary opening
//     (closed → normal), unless the last push was about a problem: who was
//     warned must hear that it is over.
//   - Any other change of state notifies. Label or description changes
//     within the same state do not.
//
// A line missing from a snapshot, or a poll where every source failed, is
// simply not passed here.
func Next(m Memory, s State) (Memory, bool) {
	switch {
	case s == Desconhecido || s == m.State:
		return m, false
	case m.State == "":
		return Memory{State: s}, false
	case s == Encerrada:
		return Memory{State: s, LastNotified: m.LastNotified}, false
	case m.State == Encerrada && s == Normal && !isProblem(m.LastNotified):
		return Memory{State: s, LastNotified: m.LastNotified}, false
	}
	return Memory{State: s, LastNotified: s}, true
}

func isProblem(s State) bool {
	return s == Reduzida || s == Parcial || s == Paralisada
}
