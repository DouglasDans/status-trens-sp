package source

import (
	"context"
	"fmt"
	"net/http"

	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

// CPTMNovoURL is the endpoint the CPTM website itself loads its line status
// from.
const CPTMNovoURL = "https://api.cptm.sp.gov.br/AppCPTM/v1/Linhas/ObterStatus"

// CPTMNovo reports only the lines CPTM shows on its website, with neither
// name nor color: the catalog fills them.
type CPTMNovo struct {
	url    string
	client *http.Client
}

func NewCPTMNovo(url string, client *http.Client) *CPTMNovo {
	return &CPTMNovo{url: url, client: client}
}

func (c *CPTMNovo) Name() string { return "cptm-novo" }

type cptmNovoLine struct {
	LinhaID   int    `json:"linhaId"`
	Status    string `json:"status"`
	Descricao string `json:"descricao"`
}

func (c *CPTMNovo) Fetch(ctx context.Context) (Snapshot, error) {
	var lines []cptmNovoLine
	if err := getJSON(ctx, c.client, c.url, &lines); err != nil {
		return Snapshot{}, fmt.Errorf("%s: %w", c.Name(), err)
	}

	readings := make([]Reading, 0, len(lines))
	for _, l := range lines {
		state, ok := status.Normalize(l.Status)
		readings = append(readings, Reading{
			Code:        l.LinhaID,
			State:       state,
			Recognized:  ok,
			Label:       l.Status,
			Description: cleanDescription(l.Status, l.Descricao),
		})
	}
	return Snapshot{Source: c.Name(), Readings: readings}, nil
}
