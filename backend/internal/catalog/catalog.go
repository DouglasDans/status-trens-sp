// Package catalog is the static list of lines with their display metadata.
// Sources only report status: their names and colors are inconsistent, and
// operators change with concessions, so this is the single place to update.
package catalog

// Line is the display metadata of one line. Operator is a stable id the app
// maps to a logo.
type Line struct {
	Code     int
	Name     string
	ColorHex string
	Operator string
}

// Lines is ordered by code. Colors come from the Próximo Trem API; operators
// as of 2026-10 (see Notion, Design e UI).
var Lines = []Line{
	{1, "Azul", "#00378c", "metro"},
	{2, "Verde", "#186d55", "metro"},
	{3, "Vermelha", "#f51200", "metro"},
	{4, "Amarela", "#efba00", "viaquatro"},
	{5, "Lilás", "#9271b1", "viamobilidade"},
	{6, "Laranja", "#f27c00", "linhauni"},
	{7, "Rubi", "#c80857", "tictrens"},
	{8, "Diamante", "#949488", "viamobilidade"},
	{9, "Esmeralda", "#219896", "viamobilidade"},
	{10, "Turquesa", "#1c8aab", "cptm"},
	{11, "Coral", "#f46c55", "cptm"},  // temporarily back from Trivia Trens since 2026-07-24
	{12, "Safira", "#1f2086", "cptm"}, // idem
	{13, "Jade", "#29b352", "cptm"},   // idem
	{15, "Prata", "#848d90", "metro"},
	{17, "Ouro", "#d58405", "metro"},
}
