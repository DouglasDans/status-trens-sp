package catalog

import "fmt"

// FallbackColor is used for a line neither the catalog nor the source colors.
const FallbackColor = "#8e8e8a"

// Display is how a line is presented to users, in the app and in pushes.
type Display struct {
	Name         string
	ColorHex     string
	TextColorHex string // empty when unknown: the app picks by contrast
	Operator     string // id, empty when unknown
	OperatorName string
	Known        bool // the catalog has an entry for the line
}

// Present resolves a line's presentation: the catalog wins, the source fills
// the gaps, and a line nobody describes still gets a name and a neutral color.
func Present(code int, sourceName, sourceColorHex string) Display {
	l, known := Lookup(code)
	return Display{
		Name:         firstNonEmpty(l.Name, sourceName, fmt.Sprintf("Linha %d", code)),
		ColorHex:     firstNonEmpty(l.ColorHex, sourceColorHex, FallbackColor),
		TextColorHex: l.TextColorHex,
		Operator:     l.Operator,
		OperatorName: Operators[l.Operator],
		Known:        known,
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
