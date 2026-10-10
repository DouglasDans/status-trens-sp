// Package catalog holds the line metadata that sources do not send reliably:
// display name, official color and operator. It is data, edited by hand.
//
// A line the sources report but the catalog lacks still shows up, using the
// source's name and color and no operator. Add an entry here only to fix its
// presentation. Adding an operator means adding it to Operators (and,
// optionally, its logo in the app).
package catalog

// Operators maps a stable operator id to its display name. The app uses the
// id to pick a logo and falls back to the name when it has none.
var Operators = map[string]string{
	"metro":         "Metrô",
	"cptm":          "CPTM",
	"motiva":        "Motiva",
	"viamobilidade": "ViaMobilidade",
	"viaquatro":     "ViaQuatro",
	"linhauni":      "Linha Uni",
	"tictrens":      "TIC Trens",
	"trivia":        "Trivia Trens",
}

// Line is the display metadata of one line.
type Line struct {
	Code     int
	Name     string
	ColorHex string
	Operator string // key of Operators
}

// Lines is sorted by code.
var Lines = []Line{
	{1, "Azul", "#00378c", "metro"},
	{2, "Verde", "#186d55", "metro"},
	{3, "Vermelha", "#f51200", "metro"},
	{4, "Amarela", "#efba00", "motiva"},
	{5, "Lilás", "#9271b1", "motiva"},
	{6, "Laranja", "#f27c00", "linhauni"},
	{7, "Rubi", "#c80857", "tictrens"},
	{8, "Diamante", "#949488", "viamobilidade"},
	{9, "Esmeralda", "#219896", "viamobilidade"},
	{10, "Turquesa", "#1c8aab", "cptm"},
	{11, "Coral", "#f46c55", "trivia"},
	{12, "Safira", "#1f2086", "trivia"},
	{13, "Jade", "#29b352", "trivia"},
	{15, "Prata", "#848d90", "metro"},
	{17, "Ouro", "#d58405", "metro"},
}

// Lookup returns the catalog entry for a line code.
func Lookup(code int) (Line, bool) {
	for _, l := range Lines {
		if l.Code == code {
			return l, true
		}
	}
	return Line{}, false
}
