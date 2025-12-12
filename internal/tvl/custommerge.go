package tvl

import (
	"log/slog"
	"time"

	"github.com/switchboard-xyz/defillama-extract/internal/api"
	"github.com/switchboard-xyz/defillama-extract/internal/models"
)

// mergeStats summarizes custom-data merge results for logging.
type mergeStats struct {
	ProtocolsWithCustomData int
	EntriesMerged           int
	CustomOnlyProtocols     int
}

// mergeCustomTVLData merges custom history into API TVL data and returns a new map.
func mergeCustomTVLData(tvlData map[string]*api.ProtocolTVLResponse, custom map[string][]models.TVLHistoryItem, logger *slog.Logger) (map[string]*api.ProtocolTVLResponse, mergeStats) {
	if tvlData == nil {
		tvlData = make(map[string]*api.ProtocolTVLResponse)
	}

	result := make(map[string]*api.ProtocolTVLResponse, len(tvlData))
	for slug, resp := range tvlData {
		result[slug] = resp
	}

	stats := mergeStats{}

	for slug, history := range custom {
		if len(history) == 0 {
			continue
		}

		apiHistory := tvlHistoryFromAPI(result[slug])
		mergedHistory := MergeTVLHistory(apiHistory, history)

		mergedPoints := make([]api.TVLDataPoint, 0, len(mergedHistory))
		for _, item := range mergedHistory {
			ts := item.Timestamp
			if ts == 0 && item.Date != "" {
				if t, err := time.Parse("2006-01-02", item.Date); err == nil {
					ts = t.Unix()
				}
			}
			mergedPoints = append(mergedPoints, api.TVLDataPoint{
				Date:              ts,
				TotalLiquidityUSD: item.TVL,
			})
		}

		existingResp, exists := result[slug]
		if !exists {
			stats.CustomOnlyProtocols++
		}

		name := slug
		currentChains := map[string]float64{}
		if existingResp != nil {
			if existingResp.Name != "" {
				name = existingResp.Name
			}
			if existingResp.CurrentChainTvls != nil {
				currentChains = existingResp.CurrentChainTvls
			}
		}

		result[slug] = &api.ProtocolTVLResponse{
			Name:             name,
			TVL:              mergedPoints,
			CurrentChainTvls: currentChains,
		}

		stats.ProtocolsWithCustomData++
		stats.EntriesMerged += len(history)
	}

	// Re-attach API entries without custom data untouched
	for slug, resp := range tvlData {
		if _, ok := custom[slug]; ok {
			continue
		}
		result[slug] = resp
	}

	if logger != nil && stats.ProtocolsWithCustomData == 0 {
		logger.Info("custom_data_merge_noop", "reason", "no custom data")
	}

	return result, stats
}

func tvlHistoryFromAPI(resp *api.ProtocolTVLResponse) []models.TVLHistoryItem {
	if resp == nil || len(resp.TVL) == 0 {
		return nil
	}
	history := make([]models.TVLHistoryItem, 0, len(resp.TVL))
	for _, point := range resp.TVL {
		history = append(history, models.TVLHistoryItem{
			Date:      time.Unix(point.Date, 0).UTC().Format("2006-01-02"),
			Timestamp: point.Date,
			TVL:       point.TotalLiquidityUSD,
		})
	}
	return history
}
