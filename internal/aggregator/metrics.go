package aggregator

import "sort"

// CalculateChainBreakdown aggregates TVS metrics per chain and returns them sorted by TVS descending.
func CalculateChainBreakdown(protocols []AggregatedProtocol) []ChainBreakdown {
	if len(protocols) == 0 {
		return []ChainBreakdown{}
	}

	chainData := make(map[string]struct {
		tvs           float64
		protocolCount int
	})

	var totalTVS float64
	for _, p := range protocols {
		for chain, tvs := range p.TVSByChain {
			if tvs == 0 {
				continue
			}

			data := chainData[chain]
			data.tvs += tvs
			data.protocolCount++
			chainData[chain] = data
			totalTVS += tvs
		}
	}

	if len(chainData) == 0 {
		return []ChainBreakdown{}
	}

	result := make([]ChainBreakdown, 0, len(chainData))
	for chain, data := range chainData {
		percentage := 0.0
		if totalTVS > 0 {
			percentage = (data.tvs / totalTVS) * 100
		}

		result = append(result, ChainBreakdown{
			Chain:         chain,
			TVS:           data.tvs,
			Percentage:    percentage,
			ProtocolCount: data.protocolCount,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].TVS > result[j].TVS
	})

	return result
}

// CalculateCategoryBreakdown aggregates TVS metrics per category and returns them sorted by TVS descending.
func CalculateCategoryBreakdown(protocols []AggregatedProtocol) []CategoryBreakdown {
	if len(protocols) == 0 {
		return []CategoryBreakdown{}
	}

	categoryData := make(map[string]struct {
		tvs           float64
		protocolCount int
	})

	var totalTVS float64
	for _, p := range protocols {
		category := p.Category
		if category == "" {
			category = "Uncategorized"
		}

		data := categoryData[category]
		data.tvs += p.TVS
		data.protocolCount++
		categoryData[category] = data
		totalTVS += p.TVS
	}

	if len(categoryData) == 0 {
		return []CategoryBreakdown{}
	}

	result := make([]CategoryBreakdown, 0, len(categoryData))
	for category, data := range categoryData {
		percentage := 0.0
		if totalTVS > 0 {
			percentage = (data.tvs / totalTVS) * 100
		}

		result = append(result, CategoryBreakdown{
			Category:      category,
			TVS:           data.tvs,
			Percentage:    percentage,
			ProtocolCount: data.protocolCount,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].TVS > result[j].TVS
	})

	return result
}

// RankProtocols sorts protocols by TVL descending, breaking ties alphabetically by Name, and assigns Rank starting at 1.
// Returns a new slice without mutating the input.
func RankProtocols(protocols []AggregatedProtocol) []AggregatedProtocol {
	if len(protocols) == 0 {
		return []AggregatedProtocol{}
	}

	ranked := make([]AggregatedProtocol, len(protocols))
	copy(ranked, protocols)

	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].TVL != ranked[j].TVL {
			return ranked[i].TVL > ranked[j].TVL
		}
		return ranked[i].Name < ranked[j].Name
	})

	for i := range ranked {
		ranked[i].Rank = i + 1
	}

	return ranked
}

// GetLargestProtocol returns the protocol with the highest TVL as a LargestProtocol pointer. Nil when input is empty.
func GetLargestProtocol(protocols []AggregatedProtocol) *LargestProtocol {
	if len(protocols) == 0 {
		return nil
	}

	best := protocols[0]
	for i := 1; i < len(protocols); i++ {
		candidate := protocols[i]
		if candidate.TVL > best.TVL {
			best = candidate
			continue
		}
		if candidate.TVL == best.TVL && candidate.Name < best.Name {
			best = candidate
		}
	}

	return &LargestProtocol{
		Name: best.Name,
		Slug: best.Slug,
		TVL:  best.TVL,
		TVS:  best.TVS,
	}
}
