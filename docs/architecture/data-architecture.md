# Data Architecture

> **Spec Reference:** [4-data-models-structures.md](../docs-from-user/seed-doc/4-data-models-structures.md)

## API Response Models

```go
// OracleAPIResponse from GET /oracles
type OracleAPIResponse struct {
    Oracles        map[string][]string                       `json:"oracles"`
    Chart          map[string]map[string]map[string]float64  `json:"chart"`
    OraclesTVS     map[string]map[string]map[string]float64  `json:"oraclesTVS"`
    ChainsByOracle map[string][]string                       `json:"chainsByOracle"`
}

// Protocol from GET /lite/protocols2
type Protocol struct {
    ID       string   `json:"id"`
    Name     string   `json:"name"`
    Slug     string   `json:"slug"`
    Category string   `json:"category"`
    TVL      float64  `json:"tvl,omitempty"`
    Chains   []string `json:"chains,omitempty"`
    Oracles  []string `json:"oracles,omitempty"`
    Oracle   string   `json:"oracle,omitempty"`  // Legacy field
    URL      string   `json:"url,omitempty"`
}
```

## Output Models

```go
// FullOutput written to JSON files
type FullOutput struct {
    Version    string               `json:"version"`
    Oracle     OracleInfo           `json:"oracle"`
    Metadata   OutputMetadata       `json:"metadata"`
    Summary    Summary              `json:"summary"`
    Metrics    Metrics              `json:"metrics"`
    Breakdown  Breakdown            `json:"breakdown"`
    Protocols  []AggregatedProtocol `json:"protocols"`
    Historical []Snapshot           `json:"historical"`
}

// State for incremental updates
type State struct {
    OracleName        string  `json:"oracle_name"`
    LastUpdated       int64   `json:"last_updated"`
    LastUpdatedISO    string  `json:"last_updated_iso"`
    LastProtocolCount int     `json:"last_protocol_count"`
    LastTVS           float64 `json:"last_tvs"`
}
```
