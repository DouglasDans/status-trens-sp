// Package status holds the domain rules for São Paulo line status.
// It is pure: no I/O, no logging, no clock.
package status

// State is the canonical operational state of a line. Its string value is
// part of the GET /status JSON contract.
type State string

const (
	Normal       State = "normal"
	Reduzida     State = "reduzida"
	Parcial      State = "parcial"
	Paralisada   State = "paralisada"
	Encerrada    State = "encerrada"
	Desconhecido State = "desconhecido"
)
