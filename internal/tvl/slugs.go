package tvl

import (
	"regexp"
	"sort"
	"strings"

	"github.com/switchboard-xyz/defillama-extract/internal/api"
)

// slugify converts a protocol name to a DefiLlama-compatible slug.
// Rules: lowercase, spaces to hyphens, remove special chars (except dots/hyphens).
func slugify(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	// lowercase
	slug := strings.ToLower(name)
	// replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	// remove characters that aren't alphanumeric, hyphens, or dots
	re := regexp.MustCompile(`[^a-z0-9\-.]`)
	slug = re.ReplaceAllString(slug, "")
	// collapse multiple hyphens
	re = regexp.MustCompile(`-+`)
	slug = re.ReplaceAllString(slug, "-")
	// trim leading/trailing hyphens
	slug = strings.Trim(slug, "-")

	return slug
}

// GetAutoDetectedSlugs extracts protocol slugs for the configured oracle from
// the protocols list (from /lite/protocols2 endpoint). Returns a sorted,
// de-duplicated slice. Nil inputs yield an empty slice to keep the pipeline
// resilient (AC6).
//
// A protocol is included if its Oracles array contains the oracle name OR if
// its single Oracle field matches. This captures ALL protocols using the oracle,
// not just those listed in the /oracles endpoint.
//
// Since /lite/protocols2 doesn't include slug field, slugs are derived from
// protocol names using DefiLlama's slugification rules.
func GetAutoDetectedSlugs(protocols []api.Protocol, oracleName string) []string {
	if len(protocols) == 0 {
		return []string{}
	}

	key := strings.TrimSpace(oracleName)
	if key == "" {
		return []string{}
	}

	seen := make(map[string]struct{})
	result := make([]string, 0)

	for _, p := range protocols {
		if !protocolUsesOracle(p, key) {
			continue
		}

		// Use slug if available, otherwise derive from name
		slug := strings.TrimSpace(p.Slug)
		if slug == "" {
			slug = slugify(p.Name)
		}
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

// protocolUsesOracle checks if a protocol uses the given oracle.
// Matches against both the Oracles array and the single Oracle field.
func protocolUsesOracle(p api.Protocol, oracleName string) bool {
	for _, o := range p.Oracles {
		if strings.EqualFold(o, oracleName) {
			return true
		}
	}
	if strings.EqualFold(p.Oracle, oracleName) {
		return true
	}
	return false
}
