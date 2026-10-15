package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

var seedable = []status.State{status.Normal, status.Reduzida, status.Parcial, status.Paralisada, status.Encerrada}

// parseSeed reads a -seed value such as "3=paralisada": the state a line is
// assumed to have had before the first poll, so a real reading that differs
// is detected as a change and pushed.
func parseSeed(s string) (int, status.State, error) {
	codeText, stateText, found := strings.Cut(s, "=")
	if !found {
		return 0, "", fmt.Errorf("seed %q: want <line>=<state>", s)
	}
	code, err := strconv.Atoi(codeText)
	if err != nil || code <= 0 {
		return 0, "", fmt.Errorf("seed %q: line must be a positive number", s)
	}
	state := status.State(stateText)
	if !slices.Contains(seedable, state) {
		return 0, "", fmt.Errorf("seed %q: state must be one of %v", s, seedable)
	}
	return code, state, nil
}
