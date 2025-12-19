package tvl

import (
	"fmt"
	"sort"

	"github.com/switchboard-xyz/defillama-extract/internal/api"
	"github.com/switchboard-xyz/defillama-extract/internal/models"
)

// MergeProtocolLists combines auto-detected protocol slugs with custom protocol
// definitions, deduplicating by slug. Custom protocols always take precedence
// over auto-detected entries when a slug exists in both inputs.
func MergeProtocolLists(autoSlugs []string, custom []models.CustomProtocol, autoMeta map[string]api.Protocol) []models.MergedProtocol {
	merged := make(map[string]models.MergedProtocol, len(autoSlugs)+len(custom))

	for _, slug := range autoSlugs {
		docs := autoDocsProof(slug)
		meta := autoMeta[slug]
		merged[slug] = models.MergedProtocol{
			Slug:            slug,
			Source:          "auto",
			IsOngoing:       false,
			URL:             meta.URL,
			SimpleTVSRatio:  0.0, // Default to 0; only custom-protocols.json can set non-zero values
			IntegrationDate: nil,
			DocsProof:       &docs,
			IsDefillama:     true, // auto-detected from /oracles endpoint
			Category:        meta.Category,
			Chains:          meta.Chains,
		}
	}

	for _, cp := range custom {
		isDefillama := false
		if cp.IsDefillama != nil {
			isDefillama = *cp.IsDefillama
		}
		merged[cp.Slug] = models.MergedProtocol{
			Slug:            cp.Slug,
			Source:          "custom",
			IsOngoing:       cp.IsOngoing,
			URL:             cp.URL,
			SimpleTVSRatio:  cp.SimpleTVSRatio,
			IntegrationDate: cp.Date,
			DocsProof:       cp.DocsProof,
			GitHubProof:     cp.GitHubProof,
			IsDefillama:     isDefillama,
			Category:        cp.Category,
			Chains:          cp.Chains,
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
