package aggregator

import "strings"

// ExtractProtocolTVS returns total TVS and per-chain TVS for a protocol.
// It excludes borrowed variants ("chain-borrowed", "borrowed") entirely since
// borrowed amounts are derived from deposits and should not be double-counted
// in TVS (Total Value Secured). Returns found=false when the protocol entry
// is absent or empty.
func ExtractProtocolTVS(oraclesTVS map[string]map[string]map[string]float64, oracleName, protocolSlug string) (float64, map[string]float64, bool) {
	if oraclesTVS == nil || oracleName == "" || protocolSlug == "" {
		return 0, nil, false
	}

	oracleData, ok := oraclesTVS[oracleName]
	if !ok {
		return 0, nil, false
	}

	chains, ok := oracleData[protocolSlug]
	if !ok || len(chains) == 0 {
		return 0, nil, false
	}

	byChain := make(map[string]float64)
	for chain, tvs := range chains {
		normalized, isBorrowed := normalizeChainKey(chain)
		if normalized == "" {
			continue
		}

		// Skip borrowed entries entirely - they are derived from deposits
		// and should not be counted in TVS to avoid double-counting.
		if isBorrowed {
			continue
		}

		byChain[normalized] += tvs
	}

	var total float64
	for _, tvs := range byChain {
		total += tvs
	}

	if len(byChain) == 0 {
		return 0, nil, false
	}

	return total, byChain, true
}

func normalizeChainKey(chain string) (string, bool) {
	c := strings.TrimSpace(chain)
	if c == "" {
		return "", false
	}

	lower := strings.ToLower(c)
	if lower == "borrowed" || strings.HasSuffix(lower, "-borrowed") {
		return "borrowed", true
	}

	return c, false
}
