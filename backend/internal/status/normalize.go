package status

import (
	"strings"
	"unicode"
)

// rules are ordered from most to least severe: when a label matches keywords
// of several states, the most severe one wins. Showing green while there is a
// problem is the worst mistake the product can make.
//
// Keywords are already lowercase and accent-free. Each word of a keyword must
// be the start of the corresponding word in the label, so "normal" matches
// "normalmente" but not "anormal".
var rules = []struct {
	state    State
	keywords []string
}{
	{Paralisada, []string{"paralisad", "paralisac", "interromp", "suspens", "sem circulac"}},
	{Parcial, []string{"parcial"}},
	{Reduzida, []string{"reduzid", "maior tempo", "maiores intervalos", "lentid", "diferenciad", "restric", "atividade programada"}},
	{Encerrada, []string{"encerrad", "fora de operac", "fechad", "nao opera"}},
	{Normal, []string{"normal", "transitori", "assistid"}},
}

// Normalize maps a free-text status label from a source to a canonical State.
// ok is false when no rule matched (including an empty label); the caller is
// expected to log the raw label so new labels can be added to the rules.
func Normalize(label string) (state State, ok bool) {
	words := tokenize(label)
	for _, rule := range rules {
		for _, keyword := range rule.keywords {
			if containsPhrase(words, strings.Fields(keyword)) {
				return rule.state, true
			}
		}
	}
	return Desconhecido, false
}

// tokenize lowercases the label, strips accents and splits it into words made
// of letters and digits.
func tokenize(label string) []string {
	var b strings.Builder
	for _, r := range strings.ToLower(label) {
		if unicode.Is(unicode.Mn, r) {
			continue // combining mark from a decomposed accent (NFD)
		}
		if base, found := unaccented[r]; found {
			r = base
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.Fields(b.String())
}

// containsPhrase reports whether phrase appears in words as consecutive words,
// each phrase word being a prefix of the corresponding label word.
func containsPhrase(words, phrase []string) bool {
	for start := 0; start+len(phrase) <= len(words); start++ {
		matched := true
		for i, p := range phrase {
			if !strings.HasPrefix(words[start+i], p) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

// unaccented covers the precomposed accented letters used in Portuguese.
var unaccented = map[rune]rune{
	'á': 'a', 'à': 'a', 'â': 'a', 'ã': 'a', 'ä': 'a',
	'é': 'e', 'è': 'e', 'ê': 'e', 'ë': 'e',
	'í': 'i', 'ì': 'i', 'î': 'i', 'ï': 'i',
	'ó': 'o', 'ò': 'o', 'ô': 'o', 'õ': 'o', 'ö': 'o',
	'ú': 'u', 'ù': 'u', 'û': 'u', 'ü': 'u',
	'ç': 'c',
}
