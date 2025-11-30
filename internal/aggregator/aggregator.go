package aggregator

import (
	"context"
	"sort"

	"github.com/switchboard-xyz/defillama-extract/internal/api"
)

// Aggregator orchestrates the aggregation pipeline for a specific oracle.
type Aggregator struct {
	oracleName string
}

// NewAggregator creates an Aggregator configured for the provided oracle name.
func NewAggregator(oracleName string) *Aggregator {
	return &Aggregator{oracleName: oracleName}
}

// Aggregate processes raw API data through the full pipeline and returns an AggregationResult.
// Nil or empty inputs return a zero-valued result without panicking.
func (a *Aggregator) Aggregate(ctx context.Context, oracleResp *api.OracleAPIResponse, protocols []api.Protocol, history []Snapshot) *AggregationResult {
	// Context reserved for future cancellation/timeout support.
	_ = ctx

	filtered := FilterByOracle(protocols, a.oracleName)
	aggregated, timestamp := ExtractProtocolData(filtered, oracleResp, a.oracleName)

	chainBreakdown := CalculateChainBreakdown(aggregated)
	categoryBreakdown := CalculateCategoryBreakdown(aggregated)
	ranked := RankProtocols(aggregated)
	largest := GetLargestProtocol(aggregated)

	totalTVS := calculateTotalTVS(aggregated)
	changeMetrics := CalculateChangeMetrics(totalTVS, len(aggregated), history)
	activeChains := extractActiveChains(chainBreakdown)
	categories := extractUniqueCategories(aggregated)

	return &AggregationResult{
		TotalTVS:          totalTVS,
		TotalProtocols:    len(aggregated),
		ActiveChains:      activeChains,
		Categories:        categories,
		ChainBreakdown:    chainBreakdown,
		CategoryBreakdown: categoryBreakdown,
		Protocols:         ranked,
		LargestProtocol:   largest,
		ChangeMetrics:     changeMetrics,
		Timestamp:         timestamp,
	}
}

func calculateTotalTVS(protocols []AggregatedProtocol) float64 {
	var total float64
	for _, p := range protocols {
		total += p.TVS
	}
	return total
}

func extractActiveChains(breakdown []ChainBreakdown) []string {
	if len(breakdown) == 0 {
		return []string{}
	}

	chains := make([]string, 0, len(breakdown))
	for _, item := range breakdown {
		chains = append(chains, item.Chain)
	}

	sort.Strings(chains)
	return chains
}

func extractUniqueCategories(protocols []AggregatedProtocol) []string {
	if len(protocols) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(protocols))
	for _, p := range protocols {
		category := p.Category
		if category == "" {
			category = "Uncategorized"
		}
		seen[category] = struct{}{}
	}

	categories := make([]string, 0, len(seen))
	for cat := range seen {
		categories = append(categories, cat)
	}

	sort.Strings(categories)
	return categories
}
