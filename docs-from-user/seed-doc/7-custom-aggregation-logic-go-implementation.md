# 7. Custom Aggregation Logic (Go Implementation)

This section provides complete Go implementations for all aggregation algorithms, addressing the evaluation feedback about pseudocode.

## 7.1 Metric Calculations

```go
// internal/aggregator/metrics.go

package aggregator

import (
    "math"
    "sort"
    "time"

    "github.com/yourorg/switchboard-extractor/internal/models"
)

const (
    // Time offsets for historical comparison
    Hours24 = 24 * 60 * 60
    Days7   = 7 * 24 * 60 * 60
    Days30  = 30 * 24 * 60 * 60

    // Tolerance for finding historical snapshots (2 hours)
    SnapshotTolerance = 2 * 60 * 60
)

// CalculatePercentageChange computes the percentage change between two values
// Formula: ((new - old) / old) * 100
// Returns 0 if old is 0 to avoid division by zero
func CalculatePercentageChange(old, new float64) float64 {
    if old == 0 {
        return 0
    }
    change := ((new - old) / old) * 100
    // Round to 2 decimal places
    return math.Round(change*100) / 100
}

// FindSnapshotAtTime finds the snapshot closest to the target time
// Returns nil if no snapshot found within tolerance
func FindSnapshotAtTime(snapshots []models.Snapshot, targetTime int64, tolerance int64) *models.Snapshot {
    var closest *models.Snapshot
    minDiff := int64(math.MaxInt64)

    for i := range snapshots {
        diff := abs(snapshots[i].Timestamp - targetTime)
        if diff <= tolerance && diff < minDiff {
            minDiff = diff
            closest = &snapshots[i]
        }
    }

    return closest
}

// abs returns the absolute value of an int64
func abs(x int64) int64 {
    if x < 0 {
        return -x
    }
    return x
}

// CalculateMetrics computes all derived metrics from current snapshot and history
func CalculateMetrics(
    current models.Snapshot,
    history []models.Snapshot,
    protocols []models.AggregatedProtocol,
) models.Metrics {
    now := time.Now().Unix()

    // Find historical snapshots
    snapshot24h := FindSnapshotAtTime(history, now-Hours24, SnapshotTolerance)
    snapshot7d := FindSnapshotAtTime(history, now-Days7, SnapshotTolerance)
    snapshot30d := FindSnapshotAtTime(history, now-Days30, SnapshotTolerance)

    metrics := models.Metrics{
        CurrentTVS:    current.TVS,
        ProtocolCount: current.ProtocolCount,
        ChainCount:    current.ChainCount,
    }

    // Calculate TVS changes
    if snapshot24h != nil {
        metrics.TVS24hAgo = snapshot24h.TVS
        metrics.Change24h = CalculatePercentageChange(snapshot24h.TVS, current.TVS)
    }

    if snapshot7d != nil {
        metrics.TVS7dAgo = snapshot7d.TVS
        metrics.Change7d = CalculatePercentageChange(snapshot7d.TVS, current.TVS)
        metrics.ProtocolGrowth7d = current.ProtocolCount - snapshot7d.ProtocolCount
    }

    if snapshot30d != nil {
        metrics.TVS30dAgo = snapshot30d.TVS
        metrics.Change30d = CalculatePercentageChange(snapshot30d.TVS, current.TVS)
        metrics.ProtocolGrowth30d = current.ProtocolCount - snapshot30d.ProtocolCount
    }

    // Extract unique categories
    categorySet := make(map[string]bool)
    for _, p := range protocols {
        if p.Category != "" {
            categorySet[p.Category] = true
        }
    }
    categories := make([]string, 0, len(categorySet))
    for cat := range categorySet {
        categories = append(categories, cat)
    }
    sort.Strings(categories)
    metrics.Categories = categories

    // Find largest protocol
    if len(protocols) > 0 {
        largest := protocols[0] // Already sorted by TVL descending
        metrics.LargestProtocol = largest.Name
        metrics.LargestProtocolTVL = largest.TVL
    }

    return metrics
}

// GetLatestTimestamp extracts the most recent timestamp from chart data
func GetLatestTimestamp(chart map[string]map[string]map[string]float64) (int64, error) {
    if len(chart) == 0 {
        return 0, models.ErrInvalidResponse
    }

    var maxTimestamp int64
    for tsStr := range chart {
        var ts int64
        if _, err := fmt.Sscanf(tsStr, "%d", &ts); err != nil {
            continue
        }
        if ts > maxTimestamp {
            maxTimestamp = ts
        }
    }

    if maxTimestamp == 0 {
        return 0, models.ErrInvalidResponse
    }

    return maxTimestamp, nil
}

// CreateSnapshot creates a new snapshot from current TVS data
func CreateSnapshot(
    timestamp int64,
    tvsByChain map[string]float64,
    protocolCount int,
) models.Snapshot {
    t := time.Unix(timestamp, 0).UTC()

    var totalTVS float64
    for _, tvs := range tvsByChain {
        totalTVS += tvs
    }

    return models.Snapshot{
        Timestamp:     timestamp,
        Date:          t.Format("2006-01-02"),
        DateTime:      t.Format(time.RFC3339),
        TVS:           totalTVS,
        TVSByChain:    tvsByChain,
        ProtocolCount: protocolCount,
        ChainCount:    len(tvsByChain),
    }
}
```

## 7.2 Protocol Filtering

```go
// internal/aggregator/filter.go

package aggregator

import (
    "slices"

    "github.com/yourorg/switchboard-extractor/internal/models"
)

// FilterProtocolsByOracle filters protocols using the specified oracle
//
// Matching rules (in order of priority):
// 1. Check if protocol.Oracles array contains oracleName
// 2. Fallback: Check if protocol.Oracle == oracleName
//
// Oracle names are case-sensitive.
func FilterProtocolsByOracle(protocols []models.Protocol, oracleName string) []models.Protocol {
    result := make([]models.Protocol, 0)

    for _, p := range protocols {
        if protocolUsesOracle(p, oracleName) {
            result = append(result, p)
        }
    }

    return result
}

// protocolUsesOracle checks if a protocol uses the specified oracle
func protocolUsesOracle(p models.Protocol, oracleName string) bool {
    // Check oracles array (preferred)
    if len(p.Oracles) > 0 {
        return slices.Contains(p.Oracles, oracleName)
    }

    // Fallback to legacy oracle field
    return p.Oracle == oracleName
}

// FilterMultiOracleProtocols returns protocols that use multiple oracles
// including the specified one (useful for understanding shared usage)
func FilterMultiOracleProtocols(protocols []models.Protocol, oracleName string) []models.Protocol {
    result := make([]models.Protocol, 0)

    for _, p := range protocols {
        if len(p.Oracles) > 1 && slices.Contains(p.Oracles, oracleName) {
            result = append(result, p)
        }
    }

    return result
}
```

## 7.3 Breakdown Calculations

```go
// internal/aggregator/breakdown.go

package aggregator

import (
    "math"
    "sort"

    "github.com/yourorg/switchboard-extractor/internal/models"
)

// CalculateChainBreakdown aggregates TVS data by chain
func CalculateChainBreakdown(oracleTVS map[string]map[string]float64) []models.ChainBreakdown {
    chainTotals := make(map[string]float64)
    chainProtocolCounts := make(map[string]int)

    // Aggregate TVS and protocol counts by chain
    for _, chainData := range oracleTVS {
        for chain, tvs := range chainData {
            chainTotals[chain] += tvs
            chainProtocolCounts[chain]++
        }
    }

    // Calculate total TVS
    var totalTVS float64
    for _, tvs := range chainTotals {
        totalTVS += tvs
    }

    // Build result slice
    result := make([]models.ChainBreakdown, 0, len(chainTotals))
    for chain, tvs := range chainTotals {
        percentage := 0.0
        if totalTVS > 0 {
            percentage = math.Round((tvs/totalTVS)*10000) / 100 // Round to 2 decimal places
        }

        result = append(result, models.ChainBreakdown{
            Chain:         chain,
            TVS:           tvs,
            ProtocolCount: chainProtocolCounts[chain],
            Percentage:    percentage,
        })
    }

    // Sort by TVS descending
    sort.Slice(result, func(i, j int) bool {
        return result[i].TVS > result[j].TVS
    })

    return result
}

// CalculateCategoryBreakdown aggregates TVS data by protocol category
func CalculateCategoryBreakdown(protocols []models.AggregatedProtocol) []models.CategoryBreakdown {
    categoryTotals := make(map[string]float64)
    categoryProtocolCounts := make(map[string]int)

    // Aggregate TVL and protocol counts by category
    for _, p := range protocols {
        if p.Category == "" {
            continue
        }
        categoryTotals[p.Category] += p.TVL
        categoryProtocolCounts[p.Category]++
    }

    // Calculate total TVL
    var totalTVL float64
    for _, tvl := range categoryTotals {
        totalTVL += tvl
    }

    // Build result slice
    result := make([]models.CategoryBreakdown, 0, len(categoryTotals))
    for category, tvl := range categoryTotals {
        percentage := 0.0
        if totalTVL > 0 {
            percentage = math.Round((tvl/totalTVL)*10000) / 100
        }

        result = append(result, models.CategoryBreakdown{
            Category:      category,
            TVS:           tvl, // Using TVL as proxy for TVS
            ProtocolCount: categoryProtocolCounts[category],
            Percentage:    percentage,
        })
    }

    // Sort by TVS descending
    sort.Slice(result, func(i, j int) bool {
        return result[i].TVS > result[j].TVS
    })

    return result
}
```

---
