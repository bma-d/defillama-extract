package aggregator

import "github.com/switchboard-xyz/defillama-extract/internal/api"

// FilterByOracle returns protocols that use the specified oracle name.
// It matches case-sensitively against the Oracles slice and the legacy Oracle field.
func FilterByOracle(protocols []api.Protocol, oracleName string) []api.Protocol {
	if oracleName == "" || len(protocols) == 0 {
		return []api.Protocol{}
	}

	filtered := make([]api.Protocol, 0, len(protocols))
	for _, protocol := range protocols {
		if containsOracle(protocol.Oracles, oracleName) || protocol.Oracle == oracleName {
			filtered = append(filtered, protocol)
		}
	}

	return filtered
}

func containsOracle(oracles []string, target string) bool {
	for _, oracle := range oracles {
		if oracle == target {
			return true
		}
	}

	return false
}
