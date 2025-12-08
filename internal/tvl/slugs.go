package tvl

import (
	"sort"
	"strings"

	"github.com/switchboard-xyz/defillama-extract/internal/api"
)

// GetAutoDetectedSlugs extracts protocol slugs for the configured oracle from
// the oracle API response. Returns a sorted, de-duplicated slice. Nil inputs
// yield an empty slice to keep the pipeline resilient (AC6).
//
// Protocol names are preserved as-is from the oracle response (including spaces).
// The HTTP client automatically URL-encodes slugs when making API requests.
func GetAutoDetectedSlugs(oracleResp *api.OracleAPIResponse, oracleName string) []string {
	if oracleResp == nil {
		return []string{}
	}

	key := strings.TrimSpace(oracleName)
	if key == "" {
		return []string{}
	}

	raw := oracleResp.Oracles[key]
	if len(raw) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(raw))
	result := make([]string, 0, len(raw))
	for _, slug := range raw {
		slug = strings.TrimSpace(slug)
		if slug == "" {
			continue
		}
		if _, ok := seen[slug]; ok {
			continue
		}
		seen[slug] = struct{}{}
		result = append(result, slug)
	}

	sort.Strings(result)
	return result
}
