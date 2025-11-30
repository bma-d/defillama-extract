package api

// OracleAPIResponse represents the payload returned by GET /oracles.
type OracleAPIResponse struct {
	Oracles        map[string][]string                      `json:"oracles"`
	Chart          map[string]map[string]map[string]float64 `json:"chart"`
	OraclesTVS     map[string]map[string]map[string]float64 `json:"oraclesTVS"`
	ChainsByOracle map[string][]string                      `json:"chainsByOracle"`
}

// Protocol represents a protocol returned by GET /lite/protocols2.
type Protocol struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Slug     string   `json:"slug"`
	Category string   `json:"category"`
	TVL      float64  `json:"tvl,omitempty"`
	Chains   []string `json:"chains,omitempty"`
	Oracles  []string `json:"oracles,omitempty"`
	Oracle   string   `json:"oracle,omitempty"`
	URL      string   `json:"url,omitempty"`
}
