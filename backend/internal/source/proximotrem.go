package source

import (
	"context"
	"fmt"
	"net/http"

	"github.com/DouglasDans/status-trens-sp/backend/internal/status"
)

// ProximoTremURL is the production base URL of the Próximo Trem API.
const ProximoTremURL = "https://fde-proximotrem-prd-hhbqdtdzdyhtgxhr.a01.azurefd.net/api/v1"

// proximoTremCodes maps the source's StatusCode enum. Unknown codes fall back
// to normalizing the StatusLabel text.
var proximoTremCodes = map[string]status.State{
	"OperacaoNormal":      status.Normal,
	"OperacaoEncerrada":   status.Encerrada,
	"AtividadeProgramada": status.Reduzida,
}

// ProximoTrem has no auth and covers every line; it comes first in the
// merge priority.
type ProximoTrem struct {
	baseURL string
	client  *http.Client
}

func NewProximoTrem(baseURL string, client *http.Client) *ProximoTrem {
	return &ProximoTrem{baseURL: baseURL, client: client}
}

func (p *ProximoTrem) Name() string { return "proximotrem" }

type proximoTremResponse struct {
	Status  bool
	Message string
	Data    []struct {
		Code        int
		Line        string
		ColorHex    string
		StatusCode  string
		StatusLabel string
		Description *string
	}
}

func (p *ProximoTrem) Fetch(ctx context.Context) (Snapshot, error) {
	var body proximoTremResponse
	if err := getJSON(ctx, p.client, p.baseURL+"/lines", &body); err != nil {
		return Snapshot{}, fmt.Errorf("%s: %w", p.Name(), err)
	}
	if !body.Status {
		return Snapshot{}, fmt.Errorf("%s: response status false: %q", p.Name(), body.Message)
	}

	readings := make([]Reading, 0, len(body.Data))
	for _, l := range body.Data {
		state, ok := proximoTremCodes[l.StatusCode]
		if !ok {
			state, ok = status.Normalize(l.StatusLabel)
		}
		var description string
		if l.Description != nil {
			description = *l.Description
		}
		readings = append(readings, Reading{
			Code:        l.Code,
			Name:        lineName(l.Line),
			ColorHex:    l.ColorHex,
			State:       state,
			Recognized:  ok,
			Label:       l.StatusLabel,
			Description: cleanDescription(l.StatusLabel, description),
		})
	}
	return Snapshot{Source: p.Name(), Readings: readings}, nil
}
