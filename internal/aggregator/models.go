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

// Snapshot represents a point-in-time TVS measurement for historical tracking.
type Snapshot struct {
	Timestamp     int64              `json:"timestamp"`
	Date          string             `json:"date"`
	TVS           float64            `json:"tvs"`
	TVSByChain    map[string]float64 `json:"tvs_by_chain"`
	ProtocolCount int                `json:"protocol_count"`
	ChainCount    int                `json:"chain_count"`
}

// ChangeMetrics captures TVS and protocol count changes over time windows.
// Pointer fields are used so nil conveys "no data available" for that period.
type ChangeMetrics struct {
	Change24h              *float64 `json:"change_24h,omitempty"`
	Change7d               *float64 `json:"change_7d,omitempty"`
	Change30d              *float64 `json:"change_30d,omitempty"`
	ProtocolCountChange7d  *int     `json:"protocol_count_change_7d,omitempty"`
	ProtocolCountChange30d *int     `json:"protocol_count_change_30d,omitempty"`
}

// AggregationResult contains the complete output of the aggregation pipeline.
type AggregationResult struct {
	TotalTVS          float64              `json:"total_tvs"`
	TotalProtocols    int                  `json:"total_protocols"`
	ActiveChains      []string             `json:"active_chains"`
	Categories        []string             `json:"categories"`
	ChainBreakdown    []ChainBreakdown     `json:"chain_breakdown"`
	CategoryBreakdown []CategoryBreakdown  `json:"category_breakdown"`
	Protocols         []AggregatedProtocol `json:"protocols"`
	LargestProtocol   *LargestProtocol     `json:"largest_protocol,omitempty"`
	ChangeMetrics     ChangeMetrics        `json:"change_metrics"`
	Timestamp         int64                `json:"timestamp"`
}
