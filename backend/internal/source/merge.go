package source

import (
	"cmp"
	"slices"

	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

// severity orders the states a source can vote for. Between sources an open
// line beats a closed one (closing is not a problem); any problem beats both.
// desconhecido is missing on purpose: it carries no information and does not
// vote.
var severity = map[status.State]int{
	status.Encerrada:  1,
	status.Normal:     2,
	status.Reduzida:   3,
	status.Parcial:    4,
	status.Paralisada: 5,
}

// Merge joins the snapshots of the sources that answered, given in priority
// order, into one reading per line, sorted by code. A source that failed, or
// does not cover a line, simply has no reading for it and does not vote.
//
// The most severe state wins, so a line is only normal when every source that
// reported it says so. Label and description always come from the winning
// reading, never mixed across sources; on a tie the reading with a
// description wins, then the source with higher priority. Name and color come
// from the first source that has them, whoever decided the state.
func Merge(snaps []Snapshot) []Reading {
	byLine := map[int][]Reading{}
	var codes []int
	for _, snap := range snaps {
		for _, r := range snap.Readings {
			if _, seen := byLine[r.Code]; !seen {
				codes = append(codes, r.Code)
			}
			r.Source = snap.Source
			byLine[r.Code] = append(byLine[r.Code], r)
		}
	}
	slices.Sort(codes)

	merged := make([]Reading, 0, len(codes))
	for _, code := range codes {
		merged = append(merged, mergeLine(byLine[code]))
	}
	return merged
}

func mergeLine(readings []Reading) Reading {
	winner := readings[0] // all unknown: the highest-priority text
	for _, r := range readings {
		if beats(r, winner) {
			winner = r
		}
	}
	winner.Name, winner.ColorHex = "", ""
	for _, r := range readings {
		winner.Name = cmp.Or(winner.Name, r.Name)
		winner.ColorHex = cmp.Or(winner.ColorHex, r.ColorHex)
	}
	return winner
}

// beats reports whether r should replace the current winner. readings come in
// priority order, so keeping the first of equals respects priority.
func beats(r, winner Reading) bool {
	rank, votes := severity[r.State]
	if !votes {
		return false
	}
	current, ok := severity[winner.State]
	if !ok || rank > current {
		return true
	}
	return rank == current && winner.Description == "" && r.Description != ""
}
