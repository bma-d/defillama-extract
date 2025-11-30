package aggregator

// AggregatedProtocol represents a protocol with enriched TVS data for output.
type AggregatedProtocol struct {
	Name       string             `json:"name"`
	Slug       string             `json:"slug"`
	Category   string             `json:"category"`
	URL        string             `json:"url"`
	TVL        float64            `json:"tvl"`
	Chains     []string           `json:"chains"`
	TVS        float64            `json:"tvs"`
	TVSByChain map[string]float64 `json:"tvs_by_chain"`
	Rank       int                `json:"rank"`
}

// LargestProtocol represents the top protocol by TVL.
type LargestProtocol struct {
	Name string  `json:"name"`
	Slug string  `json:"slug"`
	TVL  float64 `json:"tvl"`
	TVS  float64 `json:"tvs"`
}

// ChainBreakdown represents TVS metrics for a single blockchain.
type ChainBreakdown struct {
	Chain         string  `json:"chain"`
	TVS           float64 `json:"tvs"`
	Percentage    float64 `json:"percentage"`
	ProtocolCount int     `json:"protocol_count"`
}

// CategoryBreakdown represents TVS metrics for a protocol category.
type CategoryBreakdown struct {
	Category      string  `json:"category"`
	TVS           float64 `json:"tvs"`
	Percentage    float64 `json:"percentage"`
	ProtocolCount int     `json:"protocol_count"`
}
