package source

import (
	"context"
	"fmt"
	"net/http"

	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

// CPTMURL is the production URL of the CPTM mobile app endpoint.
const CPTMURL = "http://apps.cptm.sp.gov.br:8080/AppMobileService/api/LinhasMetropolitanas"

// CPTM is the legacy CPTM mobile app endpoint: 11 lines, Metrô and
// concessions included (no 6, 13, 15, 17).
// It uses content negotiation and answers XML to browsers, so the client
// always asks for JSON and rejects anything else.
type CPTM struct {
	url    string
	client *http.Client
}

func NewCPTM(url string, client *http.Client) *CPTM {
	return &CPTM{url: url, client: client}
}

func (c *CPTM) Name() string { return "cptm" }

type cptmLine struct {
	LinhaId   int
	Nome      string
	Status    string
	Descricao string
}

func (c *CPTM) Fetch(ctx context.Context) (Snapshot, error) {
	var lines []cptmLine
	if err := getJSON(ctx, c.client, c.url, &lines); err != nil {
		return Snapshot{}, fmt.Errorf("%s: %w", c.Name(), err)
	}

	readings := make([]Reading, 0, len(lines))
	for _, l := range lines {
		state, ok := status.Normalize(l.Status)
		readings = append(readings, Reading{
			Code:        l.LinhaId,
			Name:        lineName(l.Nome),
			State:       state,
			Recognized:  ok,
			Label:       l.Status,
			Description: cleanDescription(l.Status, l.Descricao),
		})
	}
	return Snapshot{Source: c.Name(), Readings: readings}, nil
}
