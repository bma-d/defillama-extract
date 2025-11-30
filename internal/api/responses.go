package api

// OracleAPIResponse represents the payload returned by GET /oracles.
type OracleAPIResponse struct {
	Oracles        map[string][]string                      `json:"oracles"`
	Chart          map[string]map[string]map[string]float64 `json:"chart"`
	OraclesTVS     map[string]map[string]map[string]float64 `json:"oraclesTVS"`
	ChainsByOracle map[string][]string                      `json:"chainsByOracle"`
}
