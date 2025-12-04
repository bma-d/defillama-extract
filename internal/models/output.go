package models

import "github.com/switchboard-xyz/defillama-extract/internal/aggregator"

// OracleInfo identifies the oracle producing the output files.
type OracleInfo struct {
	Name          string `json:"name"`
	Website       string `json:"website"`
	Documentation string `json:"documentation"`
}

// OutputMetadata captures provenance details for generated outputs.
type OutputMetadata struct {
	LastUpdated      string `json:"last_updated"`
	DataSource       string `json:"data_source"`
	UpdateFrequency  string `json:"update_frequency"`
	ExtractorVersion string `json:"extractor_version"`
}

// Summary aggregates high-level metrics for dashboards.
type Summary struct {
	TotalValueSecured float64  `json:"total_value_secured"`
	TotalProtocols    int      `json:"total_protocols"`
	ActiveChains      []string `json:"active_chains"`
	Categories        []string `json:"categories"`
}

// Metrics surfaces change and growth indicators for dashboards.
type Metrics struct {
	CurrentTVS             float64  `json:"current_tvs"`
	Change24h              *float64 `json:"change_24h,omitempty"`
	Change7d               *float64 `json:"change_7d,omitempty"`
	Change30d              *float64 `json:"change_30d,omitempty"`
	ProtocolCountChange7d  *int     `json:"protocol_count_change_7d,omitempty"`
	ProtocolCountChange30d *int     `json:"protocol_count_change_30d,omitempty"`
}

// Breakdown provides per-chain and per-category details.
type Breakdown struct {
	ByChain    []aggregator.ChainBreakdown    `json:"by_chain"`
	ByCategory []aggregator.CategoryBreakdown `json:"by_category"`
}

// FullOutput is the complete output including historical snapshots.
type FullOutput struct {
	Version      string                          `json:"version"`
	Oracle       OracleInfo                      `json:"oracle"`
	Metadata     OutputMetadata                  `json:"metadata"`
	Summary      Summary                         `json:"summary"`
	Metrics      Metrics                         `json:"metrics"`
	Breakdown    Breakdown                       `json:"breakdown"`
	Protocols    []aggregator.AggregatedProtocol `json:"protocols"`
	ChartHistory []aggregator.ChartDataPoint     `json:"chart_history"`
	Historical   []aggregator.Snapshot           `json:"historical"`
}

// SummaryOutput is the compact snapshot-only output.
type SummaryOutput struct {
	Version      string                          `json:"version"`
	Oracle       OracleInfo                      `json:"oracle"`
	Metadata     OutputMetadata                  `json:"metadata"`
	Summary      Summary                         `json:"summary"`
	Metrics      Metrics                         `json:"metrics"`
	Breakdown    Breakdown                       `json:"breakdown"`
	TopProtocols []aggregator.AggregatedProtocol `json:"top_protocols"`
}
