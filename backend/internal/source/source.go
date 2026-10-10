// Package source fetches line status from third-party endpoints and
// normalizes it into domain readings.
package source

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"regexp"
	"strings"
	"unicode"

	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

// Reading is the status of one line as reported by a source. Name and
// ColorHex come from the source so a line the catalog does not know yet still
// shows up; the catalog overrides them when it has an entry.
type Reading struct {
	Code        int
	Name        string
	ColorHex    string // empty when the source sends no color
	State       status.State
	Recognized  bool   // false when the source label matched no normalization rule
	Label       string // raw label from the source, kept for logging and display
	Description string
}

// Snapshot is the set of readings returned by one source in one fetch.
type Snapshot struct {
	Source   string
	Readings []Reading
}

// StatusSource is anything that can report the current status of the lines.
type StatusSource interface {
	Fetch(ctx context.Context) (Snapshot, error)
}

// Fallback tries each source in order and returns the first successful
// snapshot.
type Fallback []StatusSource

func (f Fallback) Fetch(ctx context.Context) (Snapshot, error) {
	var errs []error
	for _, s := range f {
		snap, err := s.Fetch(ctx)
		if err == nil {
			return snap, nil
		}
		errs = append(errs, err)
	}
	return Snapshot{}, fmt.Errorf("all sources failed: %w", errors.Join(errs...))
}

// cleanDescription drops descriptions that only repeat the label
// (e.g. label and description both "Operação Normal").
func cleanDescription(label, description string) string {
	description = strings.TrimSpace(description)
	if strings.EqualFold(description, strings.TrimSpace(label)) {
		return ""
	}
	return description
}

var linePrefix = regexp.MustCompile(`(?i)^linha\s*\d+\s*-?\s*`)

// lineName cleans source line names: "Linha 4-Amarela" and "LARANJA" become
// "Amarela" and "Laranja".
func lineName(raw string) string {
	name := linePrefix.ReplaceAllString(strings.TrimSpace(raw), "")
	if name != strings.ToUpper(name) {
		return name
	}
	words := strings.Fields(strings.ToLower(name))
	for i, w := range words {
		r := []rune(w)
		r[0] = unicode.ToUpper(r[0])
		words[i] = string(r)
	}
	return strings.Join(words, " ")
}

func getJSON(ctx context.Context, client *http.Client, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}
	// Some sources negotiate content and may answer XML; never guess.
	if mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type")); err != nil || mediaType != "application/json" {
		return fmt.Errorf("unexpected content type %q", resp.Header.Get("Content-Type"))
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}
