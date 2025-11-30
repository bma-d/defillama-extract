package aggregator

import (
	"strconv"

	"github.com/switchboard-xyz/defillama-extract/internal/api"
)

// ExtractProtocolData enriches filtered protocols with TVS data and returns the latest timestamp.
func ExtractProtocolData(protocols []api.Protocol, oracleResp *api.OracleAPIResponse, oracleName string) ([]AggregatedProtocol, int64) {
	timestamp := ExtractLatestTimestamp(oracleResp)
	if len(protocols) == 0 {
		return []AggregatedProtocol{}, timestamp
	}

	chainTVS := resolveChainTVS(oracleResp, oracleName, timestamp)

	result := make([]AggregatedProtocol, 0, len(protocols))
	for _, p := range protocols {
		agg := AggregatedProtocol{
			Name:       p.Name,
			Slug:       p.Slug,
			Category:   p.Category,
			URL:        p.URL,
			TVL:        p.TVL,
			Chains:     copyChains(p.Chains),
			TVSByChain: make(map[string]float64),
		}

		for _, chain := range p.Chains {
			if chainTVS == nil {
				continue
			}

			if tvs, ok := chainTVS[chain]; ok {
				agg.TVSByChain[chain] = tvs
				agg.TVS += tvs
			}
		}

		result = append(result, agg)
	}

	return result, timestamp
}

// ExtractLatestTimestamp returns the latest Unix timestamp found in the oracle chart data.
// Returns 0 when chart data is absent or cannot be parsed.
func ExtractLatestTimestamp(oracleResp *api.OracleAPIResponse) int64 {
	if oracleResp == nil || len(oracleResp.Chart) == 0 {
		return 0
	}

	var maxTimestamp int64
	for tsStr := range oracleResp.Chart {
		ts, err := strconv.ParseInt(tsStr, 10, 64)
		if err != nil {
			continue
		}
		if ts > maxTimestamp {
			maxTimestamp = ts
		}
	}

	return maxTimestamp
}

func resolveChainTVS(oracleResp *api.OracleAPIResponse, oracleName string, timestamp int64) map[string]float64 {
	if oracleResp == nil || oracleResp.OraclesTVS == nil {
		return nil
	}

	oracleData, ok := oracleResp.OraclesTVS[oracleName]
	if !ok {
		return nil
	}

	timestampStr := strconv.FormatInt(timestamp, 10)
	return oracleData[timestampStr]
}

func copyChains(chains []string) []string {
	if len(chains) == 0 {
		return nil
	}

	copyOfChains := make([]string, len(chains))
	copy(copyOfChains, chains)
	return copyOfChains
}
