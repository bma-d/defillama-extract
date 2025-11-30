package storage

import (
	"time"

	"github.com/switchboard-xyz/defillama-extract/internal/aggregator"
)

// CreateSnapshot builds a Snapshot from an AggregationResult for historical tracking.
// It maps aggregation output fields to the snapshot structure, ensuring TVSByChain
// is always initialized for safe JSON marshaling.
func CreateSnapshot(result *aggregator.AggregationResult) aggregator.Snapshot {
	if result == nil {
		return aggregator.Snapshot{
			TVSByChain: make(map[string]float64),
		}
	}

	tvsByChain := make(map[string]float64)
	for _, cb := range result.ChainBreakdown {
		tvsByChain[cb.Chain] = cb.TVS
	}

	chainCount := len(result.ActiveChains)

	return aggregator.Snapshot{
		Timestamp:     result.Timestamp,
		Date:          time.Unix(result.Timestamp, 0).UTC().Format("2006-01-02"),
		TVS:           result.TotalTVS,
		TVSByChain:    tvsByChain,
		ProtocolCount: result.TotalProtocols,
		ChainCount:    chainCount,
	}
}
