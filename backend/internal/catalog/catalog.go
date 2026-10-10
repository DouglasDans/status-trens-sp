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
	Code         int
	Name         string
	ColorHex     string // line color as on the current network map
	TextColorHex string // color of the line number on top of ColorHex, as on station signs
	Operator     string // key of Operators
}

const (
	white = "#ffffff"
	black = "#231f20"
)

// Lines is sorted by code.
var Lines = []Line{
	{1, "Azul", "#0053a0", white, "metro"},
	{2, "Verde", "#008062", white, "metro"},
	{3, "Vermelha", "#ef4135", white, "metro"},
	{4, "Amarela", "#ffd401", black, "motiva"},
	{5, "Lilás", "#794e9f", white, "motiva"},
	{6, "Laranja", "#f47322", black, "linhauni"},
	{7, "Rubi", "#a01866", white, "tictrens"},
	{8, "Diamante", "#9f9f94", white, "viamobilidade"},
	{9, "Esmeralda", "#00a88e", white, "viamobilidade"},
	{10, "Turquesa", "#007c8f", white, "cptm"},
	{11, "Coral", "#f04e22", white, "cptm"},
	{12, "Safira", "#053f89", white, "cptm"},
	{13, "Jade", "#00ac5b", white, "trivia"},
	{15, "Prata", "#868e91", black, "metro"},
	{17, "Ouro", "#928058", black, "metro"},
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
