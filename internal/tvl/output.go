package tvl

import (
	"time"

	"github.com/switchboard-xyz/defillama-extract/internal/api"
	"github.com/switchboard-xyz/defillama-extract/internal/models"
)

// MapToOutputProtocol converts a merged protocol plus its TVL data into the
// output contract used by tvl-data.json. IntegrationDate is passed through
// unchanged (nil for auto or missing custom dates) and the full TVL history is
// preserved without filtering.
func MapToOutputProtocol(protocol models.MergedProtocol, tvl *api.ProtocolTVLResponse) models.TVLOutputProtocol {
	history := make([]models.TVLHistoryItem, 0)
	currentTVL := 0.0

	if tvl != nil {
		history = make([]models.TVLHistoryItem, 0, len(tvl.TVL))
		for _, point := range tvl.TVL {
			history = append(history, models.TVLHistoryItem{
				Date:      time.Unix(point.Date, 0).UTC().Format("2006-01-02"),
				Timestamp: point.Date,
				TVL:       point.TotalLiquidityUSD,
			})
		}

		if len(tvl.TVL) > 0 {
			currentTVL = tvl.TVL[len(tvl.TVL)-1].TotalLiquidityUSD
		}

		if protocol.Name == "" {
			protocol.Name = tvl.Name
		}
	}

	return models.TVLOutputProtocol{
		Name:            protocol.Name,
		Slug:            protocol.Slug,
		Source:          protocol.Source,
		IsOngoing:       protocol.IsOngoing,
		SimpleTVSRatio:  protocol.SimpleTVSRatio,
		IntegrationDate: protocol.IntegrationDate,
		DocsProof:       protocol.DocsProof,
		GitHubProof:     protocol.GitHubProof,
		CurrentTVL:      currentTVL,
		TVLHistory:      history,
	}
}
