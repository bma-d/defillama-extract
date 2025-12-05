package aggregator

import (
	"log/slog"
	"strconv"
	"strings"

	"github.com/switchboard-xyz/defillama-extract/internal/api"
)

// ExtractProtocolData enriches filtered protocols with TVS data and returns the latest timestamp
// along with counts of protocols that have/are missing TVS.
func ExtractProtocolData(protocols []api.Protocol, oracleResp *api.OracleAPIResponse, oracleName string) ([]AggregatedProtocol, int64, int, int) {
	timestamp := ExtractLatestTimestamp(oracleResp)
	if len(protocols) == 0 {
		return []AggregatedProtocol{}, timestamp, 0, 0
	}

	logger := slog.Default()
	result := make([]AggregatedProtocol, 0, len(protocols))
	withTVS := 0
	withoutTVS := 0

	if oracleResp == nil {
		for _, p := range protocols {
			result = append(result, AggregatedProtocol{
				Name:       p.Name,
				Slug:       p.Slug,
				Category:   p.Category,
				URL:        p.URL,
				TVL:        p.TVL,
				Chains:     copyChains(p.Chains),
				TVSByChain: make(map[string]float64),
			})
		}
		return result, timestamp, 0, len(protocols)
	}

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

		slugKey := strings.TrimSpace(p.Slug)
		nameKey := strings.TrimSpace(p.Name)
		protocolKey := slugKey
		if protocolKey == "" {
			protocolKey = nameKey
		}

		total, byChain, found := ExtractProtocolTVS(oracleResp.OraclesTVS, oracleName, protocolKey)
		if !found && nameKey != "" && nameKey != protocolKey {
			total, byChain, found = ExtractProtocolTVS(oracleResp.OraclesTVS, oracleName, nameKey)
			if !found {
				protocolKey = nameKey
			}
		}

		if found {
			agg.TVS = total
			agg.TVSByChain = byChain
			withTVS++
		} else {
			withoutTVS++
			logger.Warn("protocol_tvs_unavailable",
				"protocol", protocolKey,
				"reason", "not found in oraclesTVS",
			)
		}

		result = append(result, agg)
	}

	return result, timestamp, withTVS, withoutTVS
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

func resolveProtocolChainTVS(oracleResp *api.OracleAPIResponse, oracleName, protocolName string, timestamp int64) map[string]float64 {
	if oracleResp == nil || oracleResp.OraclesTVS == nil {
		return nil
	}

	oracleData, ok := oracleResp.OraclesTVS[oracleName]
	if !ok {
		return nil
	}

	if data, ok := oracleData[protocolName]; ok {
		return data
	}

	if timestamp <= 0 {
		return nil
	}

	tsKey := strconv.FormatInt(timestamp, 10)
	return oracleData[tsKey]
}

func copyChains(chains []string) []string {
	if len(chains) == 0 {
		return nil
	}

	copyOfChains := make([]string, len(chains))
	copy(copyOfChains, chains)
	return copyOfChains
}
