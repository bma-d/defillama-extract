package storage

import (
	"encoding/json"
	"log/slog"
	"os"
	"sort"
	"time"

	"github.com/switchboard-xyz/defillama-extract/internal/aggregator"
)

// outputHistoryExtract is used for partial parsing of the output file to
// extract only the historical snapshots slice.
type outputHistoryExtract struct {
	Historical []aggregator.Snapshot `json:"historical"`
}

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

// LoadFromOutput extracts the historical snapshots from an existing output
// file. It returns snapshots sorted by timestamp ascending. Missing or
// corrupted files degrade gracefully by returning an empty slice and logging
// the condition instead of failing the caller.
func LoadFromOutput(outputPath string, logger *slog.Logger) ([]aggregator.Snapshot, error) {
	if logger == nil {
		logger = slog.Default()
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("no existing history found, starting fresh", "path", outputPath)
			return []aggregator.Snapshot{}, nil
		}

		logger.Warn("failed to read history file, starting fresh", "path", outputPath, "error", err)
		return []aggregator.Snapshot{}, nil
	}

	var extract outputHistoryExtract
	if err := json.Unmarshal(data, &extract); err != nil {
		logger.Warn("failed to load history, starting fresh", "path", outputPath, "error", err)
		return []aggregator.Snapshot{}, nil
	}

	if len(extract.Historical) == 0 {
		return []aggregator.Snapshot{}, nil
	}

	sort.Slice(extract.Historical, func(i, j int) bool {
		return extract.Historical[i].Timestamp < extract.Historical[j].Timestamp
	})

	logger.Debug("history loaded",
		"path", outputPath,
		"snapshot_count", len(extract.Historical),
		"oldest", extract.Historical[0].Timestamp,
		"newest", extract.Historical[len(extract.Historical)-1].Timestamp,
	)

	return extract.Historical, nil
}
