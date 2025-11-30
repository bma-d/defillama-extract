# 4. Data Models & Structures

## 4.1 API Response Models

### 4.1.1 Oracle API Response

```go
// OracleAPIResponse represents the response from GET /oracles
type OracleAPIResponse struct {
    // Oracles maps oracle name to list of protocol names
    // Example: {"Switchboard": ["Kamino Lend", "marginfi Lending"]}
    Oracles map[string][]string `json:"oracles"`

    // Chart contains time-series TVS data
    // Structure: timestamp (string) -> oracle name -> chain name -> TVS value
    // Example: {"1732867200": {"Switchboard": {"Solana": 180000000}}}
    Chart map[string]map[string]map[string]float64 `json:"chart"`

    // ChainChart contains chain-specific TVS data (same structure as Chart)
    ChainChart map[string]map[string]map[string]float64 `json:"chainChart"`

    // OraclesTVS contains per-protocol TVS breakdown
    // Structure: oracle name -> protocol name -> chain name -> TVS value
    OraclesTVS map[string]map[string]map[string]float64 `json:"oraclesTVS"`

    // ChainsByOracle maps oracle name to list of supported chains
    // Example: {"Switchboard": ["Solana", "Sui", "Aptos"]}
    ChainsByOracle map[string][]string `json:"chainsByOracle"`
}
```

### 4.1.2 Protocol API Response

```go
// ProtocolAPIResponse represents the response from GET /lite/protocols2
type ProtocolAPIResponse struct {
    Protocols       []Protocol       `json:"protocols"`
    Chains          []string         `json:"chains"`
    ParentProtocols []ParentProtocol `json:"parentProtocols"`
}

// Protocol represents a single protocol's metadata
type Protocol struct {
    ID       string   `json:"id"`
    Name     string   `json:"name"`
    Slug     string   `json:"slug"`
    Chain    string   `json:"chain"`
    Chains   []string `json:"chains,omitempty"`
    Category string   `json:"category"`
    TVL      float64  `json:"tvl,omitempty"`
    Oracles  []string `json:"oracles,omitempty"`
    Oracle   string   `json:"oracle,omitempty"`  // Legacy single oracle field
    Symbol   string   `json:"symbol,omitempty"`
    URL      string   `json:"url,omitempty"`
}

// ParentProtocol represents a parent-child protocol relationship
type ParentProtocol struct {
    ID       string   `json:"id"`
    Name     string   `json:"name"`
    Chains   []string `json:"chains,omitempty"`
}
```

## 4.2 Internal Models

### 4.2.1 Aggregated Protocol

```go
// AggregatedProtocol represents a protocol with computed metrics
type AggregatedProtocol struct {
    Rank       int      `json:"rank"`
    Name       string   `json:"name"`
    Slug       string   `json:"slug"`
    Category   string   `json:"category"`
    TVL        float64  `json:"tvl"`
    Chains     []string `json:"chains"`
    URL        string   `json:"url"`

    // TVS represents this protocol's contribution to oracle TVS
    TVS        float64  `json:"tvs,omitempty"`

    // TVSByChain breaks down TVS by chain for this protocol
    TVSByChain map[string]float64 `json:"tvs_by_chain,omitempty"`
}
```

### 4.2.2 Chain Breakdown

```go
// ChainBreakdown contains TVS metrics for a single chain
type ChainBreakdown struct {
    Chain         string  `json:"chain"`
    TVS           float64 `json:"tvs"`
    ProtocolCount int     `json:"protocol_count"`
    Percentage    float64 `json:"percentage"`
}
```

### 4.2.3 Snapshot

```go
// Snapshot represents a point-in-time oracle data capture
type Snapshot struct {
    Timestamp     int64              `json:"timestamp"`
    Date          string             `json:"date"`           // ISO8601 date only (YYYY-MM-DD)
    DateTime      string             `json:"datetime"`       // Full ISO8601
    TVS           float64            `json:"tvs"`
    TVSByChain    map[string]float64 `json:"tvs_by_chain"`
    ProtocolCount int                `json:"protocol_count"`
    ChainCount    int                `json:"chain_count"`
}
```

### 4.2.4 Metrics

```go
// Metrics contains calculated derived metrics
type Metrics struct {
    CurrentTVS        float64  `json:"current_tvs"`
    TVS24hAgo         float64  `json:"tvs_24h_ago,omitempty"`
    TVS7dAgo          float64  `json:"tvs_7d_ago,omitempty"`
    TVS30dAgo         float64  `json:"tvs_30d_ago,omitempty"`
    Change24h         float64  `json:"change_24h"`
    Change7d          float64  `json:"change_7d"`
    Change30d         float64  `json:"change_30d"`
    ProtocolCount     int      `json:"protocol_count"`
    ProtocolGrowth7d  int      `json:"protocol_growth_7d"`
    ProtocolGrowth30d int      `json:"protocol_growth_30d"`
    ChainCount        int      `json:"chain_count"`
    Categories        []string `json:"categories"`
    LargestProtocol   string   `json:"largest_protocol"`
    LargestProtocolTVL float64 `json:"largest_protocol_tvl"`
}
```

## 4.3 Custom Error Types

```go
// internal/models/errors.go

package models

import (
    "errors"
    "fmt"
)

// Sentinel errors for common failure conditions
var (
    ErrNoNewData        = errors.New("no new data available")
    ErrOracleNotFound   = errors.New("oracle not found in response")
    ErrInvalidResponse  = errors.New("invalid API response")
    ErrStateCorrupted   = errors.New("state file corrupted")
)

// APIError represents an error from the DefiLlama API
type APIError struct {
    Endpoint   string
    StatusCode int
    Message    string
    Err        error
}

func (e *APIError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("API error for %s (status %d): %s: %v",
            e.Endpoint, e.StatusCode, e.Message, e.Err)
    }
    return fmt.Sprintf("API error for %s (status %d): %s",
        e.Endpoint, e.StatusCode, e.Message)
}

func (e *APIError) Unwrap() error {
    return e.Err
}

// IsRetryable returns true if the error should trigger a retry
func (e *APIError) IsRetryable() bool {
    switch e.StatusCode {
    case 429, 500, 502, 503, 504:
        return true
    default:
        return false
    }
}

// ValidationError represents a data validation failure
type ValidationError struct {
    Field   string
    Value   interface{}
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation error for %s=%v: %s", e.Field, e.Value, e.Message)
}

// AggregationError represents an error during data aggregation
type AggregationError struct {
    Operation string
    Err       error
}

func (e *AggregationError) Error() string {
    return fmt.Sprintf("aggregation error in %s: %v", e.Operation, e.Err)
}

func (e *AggregationError) Unwrap() error {
    return e.Err
}
```

## 4.4 Output Models

### 4.4.1 Full Output

```go
// FullOutput is the complete output structure written to JSON
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

// OracleInfo contains oracle identification
type OracleInfo struct {
    Name          string `json:"name"`
    Website       string `json:"website"`
    Documentation string `json:"documentation,omitempty"`
}

// OutputMetadata contains extraction metadata
type OutputMetadata struct {
    LastUpdated      string `json:"last_updated"`
    DataSource       string `json:"data_source"`
    UpdateFrequency  string `json:"update_frequency"`
    ExtractorVersion string `json:"extractor_version"`
}

// Summary contains high-level metrics
type Summary struct {
    TotalValueSecured float64  `json:"total_value_secured"`
    TotalProtocols    int      `json:"total_protocols"`
    ActiveChains      int      `json:"active_chains"`
    Categories        []string `json:"categories"`
}

// Breakdown contains categorical breakdowns
type Breakdown struct {
    ByChain    []ChainBreakdown    `json:"by_chain"`
    ByCategory []CategoryBreakdown `json:"by_category"`
}

// CategoryBreakdown contains TVS metrics for a category
type CategoryBreakdown struct {
    Category      string  `json:"category"`
    TVS           float64 `json:"tvs"`
    ProtocolCount int     `json:"protocol_count"`
    Percentage    float64 `json:"percentage"`
}
```

### 4.4.2 State Model

```go
// State represents the incremental update state
type State struct {
    OracleName        string  `json:"oracle_name"`
    LastUpdated       int64   `json:"last_updated"`
    LastUpdatedISO    string  `json:"last_updated_iso"`
    LastProtocolCount int     `json:"last_protocol_count"`
    LastTVS           float64 `json:"last_tvs"`
    SnapshotCount     int     `json:"snapshot_count"`
    OldestSnapshot    int64   `json:"oldest_snapshot"`
    NewestSnapshot    int64   `json:"newest_snapshot"`
}
```

---
