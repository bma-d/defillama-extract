package tvl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/switchboard-xyz/defillama-extract/internal/api"
	"github.com/switchboard-xyz/defillama-extract/internal/models"
	"github.com/switchboard-xyz/defillama-extract/internal/storage"
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

// GenerateTVLOutput builds the root tvl-data.json document from merged
// protocols and their associated TVL responses. The protocols map is keyed by
// slug to satisfy AC3.
func GenerateTVLOutput(protocols []models.MergedProtocol, tvlData map[string]*api.ProtocolTVLResponse) *models.TVLOutput {
	result := &models.TVLOutput{
		Version:   "1.0.0",
		Metadata:  models.TVLOutputMetadata{},
		Protocols: make(map[string]models.TVLOutputProtocol),
	}

	for _, p := range protocols {
		if p.Slug == "" {
			continue
		}

		mapped := MapToOutputProtocol(p, tvlData[p.Slug])
		result.Protocols[p.Slug] = mapped
		result.Metadata.ProtocolCount++
		if p.Source == "custom" {
			result.Metadata.CustomProtocolCount++
		}
	}

	result.Metadata.LastUpdated = time.Now().UTC().Format(time.RFC3339)

	return result
}

// WriteTVLOutputs writes both indented and minified output files atomically.
// Context cancellation is honored between writes to prevent partial state.
func WriteTVLOutputs(ctx context.Context, outputDir string, output *models.TVLOutput) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if output == nil {
		return fmt.Errorf("output is nil")
	}

	fullPath := filepath.Join(outputDir, "tvl-data.json")
	minPath := filepath.Join(outputDir, "tvl-data.min.json")

	written := []string{}
	cleanup := func() {
		for i := len(written) - 1; i >= 0; i-- {
			_ = os.Remove(written[i])
		}
	}

	write := func(path string, data interface{}, indent bool) error {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := storage.WriteJSON(path, data, indent); err != nil {
			return err
		}

		written = append(written, path)

		if err := ctx.Err(); err != nil {
			cleanup()
			return err
		}

		return nil
	}

	if err := write(fullPath, output, true); err != nil {
		cleanup()
		return err
	}

	if err := write(minPath, output, false); err != nil {
		cleanup()
		return err
	}

	return nil
}
