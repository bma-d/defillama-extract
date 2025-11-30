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
