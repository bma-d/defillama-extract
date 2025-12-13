package tvl

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
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
		// Dedupe by date: DefiLlama returns daily snapshots plus a real-time value,
		// which can produce two entries for the same date. Keep the latest per date.
		dateMap := make(map[string]models.TVLHistoryItem)
		for _, point := range tvl.TVL {
			dateStr := time.Unix(point.Date, 0).UTC().Format("2006-01-02")
			existing, exists := dateMap[dateStr]
			if !exists || point.Date > existing.Timestamp {
				dateMap[dateStr] = models.TVLHistoryItem{
					Date:      dateStr,
					Timestamp: point.Date,
					TVL:       point.TotalLiquidityUSD,
				}
			}
		}

		// Build sorted history from deduped map
		history = make([]models.TVLHistoryItem, 0, len(dateMap))
		for _, item := range dateMap {
			history = append(history, item)
		}
		// Sort by timestamp ascending
		sort.Slice(history, func(i, j int) bool {
			return history[i].Timestamp < history[j].Timestamp
		})

		if len(history) > 0 {
			currentTVL = history[len(history)-1].TVL
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
		URL:             protocol.URL,
		SimpleTVSRatio:  protocol.SimpleTVSRatio,
		IntegrationDate: protocol.IntegrationDate,
		DocsProof:       protocol.DocsProof,
		GitHubProof:     protocol.GitHubProof,
		IsDefillama:     protocol.IsDefillama,
		CurrentTVL:      currentTVL,
		TVLHistory:      history,
	}
}

// MapToCustomOutputProtocol mirrors MapToOutputProtocol but includes category
// and chains provided by custom-data files.
func MapToCustomOutputProtocol(protocol models.MergedProtocol, tvl *api.ProtocolTVLResponse, attrs CustomDataAttributes) models.CustomDataOutputEntry {
	history := make([]models.TVLHistoryItem, 0)
	currentTVL := 0.0

	if tvl != nil {
		dateMap := make(map[string]models.TVLHistoryItem)
		for _, point := range tvl.TVL {
			dateStr := time.Unix(point.Date, 0).UTC().Format("2006-01-02")
			existing, exists := dateMap[dateStr]
			if !exists || point.Date > existing.Timestamp {
				dateMap[dateStr] = models.TVLHistoryItem{
					Date:      dateStr,
					Timestamp: point.Date,
					TVL:       point.TotalLiquidityUSD,
				}
			}
		}

		history = make([]models.TVLHistoryItem, 0, len(dateMap))
		for _, item := range dateMap {
			history = append(history, item)
		}
		sort.Slice(history, func(i, j int) bool {
			return history[i].Timestamp < history[j].Timestamp
		})

		if len(history) > 0 {
			currentTVL = history[len(history)-1].TVL
		}

		if protocol.Name == "" && tvl.Name != "" {
			protocol.Name = tvl.Name
		}
	}

	category := protocol.Category
	if category == "" {
		category = attrs.Category
	}
	chains := protocol.Chains
	if len(chains) == 0 {
		chains = attrs.Chains
	}
	url := protocol.URL
	if url == "" {
		url = attrs.URL
	}

	return models.CustomDataOutputEntry{
		Name:            protocol.Name,
		Slug:            protocol.Slug,
		Source:          protocol.Source,
		IsOngoing:       protocol.IsOngoing,
		URL:             url,
		SimpleTVSRatio:  protocol.SimpleTVSRatio,
		IntegrationDate: protocol.IntegrationDate,
		DocsProof:       protocol.DocsProof,
		GitHubProof:     protocol.GitHubProof,
		IsDefillama:     protocol.IsDefillama,
		Category:        category,
		Chains:          chains,
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

// GenerateCustomDataOutput builds custom-data.json for protocols supplied via
// custom-data files. Category/chains metadata from the custom-data file is
// preserved when present.
func GenerateCustomDataOutput(protocols []models.MergedProtocol, tvlData map[string]*api.ProtocolTVLResponse, attrs map[string]CustomDataAttributes) *models.CustomDataOutput {
	result := &models.CustomDataOutput{
		Version:   "1.0.0",
		Metadata:  models.CustomDataOutputMetadata{},
		Protocols: make(map[string]models.CustomDataOutputEntry),
	}

	for _, p := range protocols {
		if p.Slug == "" {
			continue
		}
		entry := MapToCustomOutputProtocol(p, tvlData[p.Slug], attrs[p.Slug])
		result.Protocols[p.Slug] = entry
		result.Metadata.ProtocolCount++
	}

	result.Metadata.LastUpdated = time.Now().UTC().Format(time.RFC3339)
	return result
}

// WriteTVLOutputs writes the TVL output file atomically.
// Context cancellation is honored to prevent partial state.
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

	if err := storage.WriteJSON(fullPath, output, true); err != nil {
		return err
	}

	return nil
}

// WriteCustomDataOutputs writes the custom-data.json file atomically.
func WriteCustomDataOutputs(ctx context.Context, outputDir string, output *models.CustomDataOutput) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if output == nil {
		return fmt.Errorf("custom data output is nil")
	}

	fullPath := filepath.Join(outputDir, "custom-data.json")
	if err := storage.WriteJSON(fullPath, output, true); err != nil {
		return err
	}
	return nil
}
