package tvl

import (
	"fmt"
	"sort"

	"github.com/switchboard-xyz/defillama-extract/internal/models"
)

// MergeProtocolLists combines auto-detected protocol slugs with custom protocol
// definitions, deduplicating by slug. Custom protocols always take precedence
// over auto-detected entries when a slug exists in both inputs.
func MergeProtocolLists(autoSlugs []string, custom []models.CustomProtocol) []models.MergedProtocol {
	merged := make(map[string]models.MergedProtocol, len(autoSlugs)+len(custom))

	for _, slug := range autoSlugs {
		docs := autoDocsProof(slug)
		merged[slug] = models.MergedProtocol{
			Slug:            slug,
			Source:          "auto",
			IsOngoing:       true,
			SimpleTVSRatio:  1.0,
			IntegrationDate: nil,
			DocsProof:       &docs,
		}
	}

	for _, cp := range custom {
		merged[cp.Slug] = models.MergedProtocol{
			Slug:            cp.Slug,
			Source:          "custom",
			IsOngoing:       cp.IsOngoing,
			SimpleTVSRatio:  cp.SimpleTVSRatio,
			IntegrationDate: cp.Date,
			DocsProof:       cp.DocsProof,
			GitHubProof:     cp.GitHubProof,
		}
	}

	result := make([]models.MergedProtocol, 0, len(merged))
	for _, protocol := range merged {
		result = append(result, protocol)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Slug < result[j].Slug
	})

	return result
}

func autoDocsProof(slug string) string {
	return fmt.Sprintf("https://defillama.com/protocol/%s", slug)
}
