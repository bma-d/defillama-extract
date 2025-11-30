# Switchboard Oracle Data Extraction System - Implementation Specification (Revised v1)

## Document Purpose

This document provides comprehensive specifications for building a Go-based data extraction system that retrieves Switchboard oracle metrics from DefiLlama's public APIs. The system implements **Custom Aggregation** (Approach 2) with **Incremental Updates** (Method 3) as its data strategy.

**Target Audience:** LLM or developer implementing the extraction system
**Implementation Language:** Go (Golang) 1.21+
**Primary Data Source:** DefiLlama Public API
**Document Version:** 1.1.0 (Revised)

### Revision Notes (v1.1.0)
- Added complete Go implementations for all aggregation algorithms (Section 7)
- Enhanced Go-specific patterns: error wrapping, context propagation, structured logging (Section 15)
- Added comprehensive test implementations with table-driven tests and mocking examples (Section 11)
- Added operational concerns: graceful shutdown, monitoring, API versioning (Section 16)
- Expanded dependency injection and main.go implementation examples (Section 17)

---

## Table of Contents

1. [System Overview](#1-system-overview)
2. [Architecture Design](#2-architecture-design)
3. [Data Sources & API Specifications](#3-data-sources--api-specifications)
4. [Data Models & Structures](#4-data-models--structures)
5. [Core Components](#5-core-components)
6. [Incremental Update Strategy](#6-incremental-update-strategy)
7. [Custom Aggregation Logic (Go Implementation)](#7-custom-aggregation-logic-go-implementation)
8. [Storage & Caching](#8-storage--caching)
9. [Error Handling & Resilience](#9-error-handling--resilience)
10. [Configuration & Environment](#10-configuration--environment)
11. [Testing Strategy (Complete Implementation)](#11-testing-strategy-complete-implementation)
12. [Deployment Considerations](#12-deployment-considerations)
13. [API Response Examples](#13-api-response-examples)
14. [Implementation Checklist](#14-implementation-checklist)
15. [Go-Specific Patterns & Idioms](#15-go-specific-patterns--idioms)
16. [Operational Concerns](#16-operational-concerns)
17. [Complete Main.go Implementation](#17-complete-maingo-implementation)

---

## 1. System Overview

### 1.1 Objective

Build a production-ready Go service that:
1. Fetches Switchboard oracle data from DefiLlama APIs
2. Implements custom aggregation logic for enhanced metrics
3. Uses incremental updates to minimize API calls and bandwidth
4. Outputs structured JSON files for consumption by other services/websites
5. Provides real-time metrics and historical data tracking

### 1.2 Key Features

| Feature | Description |
|---------|-------------|
| Custom Aggregation | Calculate derived metrics (7d/30d changes, growth rates, rankings) |
| Incremental Updates | Only fetch/process new data since last successful update |
| Multi-Source Correlation | Combine oracle data with protocol metadata |
| Historical Tracking | Maintain 90-day rolling history of snapshots |
| Fault Tolerance | Retry logic, graceful degradation, state recovery |
| JSON Output | Multiple output formats (full, compact, summary) |
| Graceful Shutdown | Proper signal handling and cleanup |
| Monitoring | Prometheus metrics and health endpoints |

### 1.3 Data Flow Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         DefiLlama APIs                               │
│  ┌─────────────────────┐  ┌─────────────────────────────────────┐  │
│  │ GET /oracles        │  │ GET /lite/protocols2?b=2            │  │
│  │ (Oracle TVS data)   │  │ (Protocol metadata)                 │  │
│  └──────────┬──────────┘  └──────────────────┬──────────────────┘  │
└─────────────┼────────────────────────────────┼──────────────────────┘
              │                                │
              ▼                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Go Extraction Service                          │
│                                                                     │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────────────┐   │
│  │   Fetcher    │──▶│  Aggregator  │──▶│  Incremental Store   │   │
│  │  (HTTP Client)│   │  (Business   │   │  (State Management)  │   │
│  │              │   │   Logic)     │   │                      │   │
│  └──────────────┘   └──────────────┘   └──────────┬───────────┘   │
│                                                    │               │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │                     JSON File Writer                          │ │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────────────────┐  │ │
│  │  │ Full JSON  │  │ Min JSON   │  │ Summary JSON           │  │ │
│  │  └────────────┘  └────────────┘  └────────────────────────┘  │ │
│  └──────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        Output Directory                             │
│  ./data/                                                            │
│  ├── switchboard-oracle-data.json      (Full data with history)    │
│  ├── switchboard-oracle-data.min.json  (Minified version)          │
│  ├── switchboard-summary.json          (Current snapshot only)     │
│  └── state.json                        (Incremental update state)  │
└─────────────────────────────────────────────────────────────────────┘
```

### 1.4 Update Cycle

```
┌─────────────────────────────────────────────────────────────────┐
│                    15-Minute Update Cycle                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. Load State (state.json)                                     │
│     └─▶ Get last_updated timestamp                              │
│                                                                 │
│  2. Fetch API Data (parallel)                                   │
│     ├─▶ GET /oracles                                            │
│     └─▶ GET /lite/protocols2?b=2                                │
│                                                                 │
│  3. Check for New Data                                          │
│     └─▶ Compare latest chart timestamp vs last_updated          │
│         └─▶ If no new data: EXIT (skip processing)              │
│                                                                 │
│  4. Filter & Aggregate                                          │
│     ├─▶ Filter protocols using Switchboard                      │
│     ├─▶ Calculate TVS metrics                                   │
│     ├─▶ Compute derived metrics (changes, growth)               │
│     └─▶ Generate rankings                                       │
│                                                                 │
│  5. Merge with History                                          │
│     ├─▶ Load existing snapshots                                 │
│     ├─▶ Append new snapshot                                     │
│     └─▶ Prune snapshots older than 90 days                      │
│                                                                 │
│  6. Write Output Files                                          │
│     ├─▶ Write full JSON                                         │
│     ├─▶ Write minified JSON                                     │
│     ├─▶ Write summary JSON                                      │
│     └─▶ Update state.json                                       │
│                                                                 │
│  7. Log Metrics                                                 │
│     └─▶ Protocol count, TVS, duration, etc.                     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. Architecture Design

### 2.1 Package Structure

```
switchboard-oracle-extractor/
├── cmd/
│   └── extractor/
│       └── main.go              # Application entry point
├── internal/
│   ├── api/
│   │   ├── client.go            # HTTP client with retry logic
│   │   ├── client_test.go       # Client tests
│   │   ├── endpoints.go         # API endpoint definitions
│   │   └── responses.go         # API response type definitions
│   ├── aggregator/
│   │   ├── aggregator.go        # Core aggregation logic
│   │   ├── aggregator_test.go   # Aggregator tests
│   │   ├── metrics.go           # Metric calculation functions
│   │   ├── metrics_test.go      # Metrics tests
│   │   └── filter.go            # Protocol filtering logic
│   ├── storage/
│   │   ├── state.go             # Incremental state management
│   │   ├── state_test.go        # State tests
│   │   ├── writer.go            # JSON file writer
│   │   └── history.go           # Historical data management
│   ├── models/
│   │   ├── oracle.go            # Oracle-related data structures
│   │   ├── protocol.go          # Protocol data structures
│   │   ├── snapshot.go          # Snapshot data structures
│   │   ├── output.go            # Output format structures
│   │   └── errors.go            # Custom error types
│   ├── config/
│   │   └── config.go            # Configuration management
│   └── monitoring/
│       ├── metrics.go           # Prometheus metrics
│       └── health.go            # Health check handlers
├── pkg/
│   └── utils/
│       ├── time.go              # Time utilities
│       ├── math.go              # Math utilities
│       └── slice.go             # Slice utilities
├── configs/
│   └── config.yaml              # Default configuration
├── testdata/                    # Test fixtures
│   ├── oracle_response.json
│   └── protocol_response.json
├── data/                        # Output directory (gitignored)
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml
└── README.md
```

### 2.2 Component Responsibilities

| Component | Responsibility |
|-----------|---------------|
| `cmd/extractor/main.go` | CLI setup, dependency injection, scheduler initialization, signal handling |
| `internal/api/client.go` | HTTP client with retries, rate limiting, timeout handling |
| `internal/api/endpoints.go` | API URL constants and request builders |
| `internal/api/responses.go` | Structs matching DefiLlama API response shapes |
| `internal/aggregator/aggregator.go` | Main orchestration of data processing pipeline |
| `internal/aggregator/metrics.go` | Derived metric calculations (changes, growth rates) |
| `internal/aggregator/filter.go` | Protocol filtering by oracle name |
| `internal/storage/state.go` | Read/write incremental update state |
| `internal/storage/writer.go` | JSON serialization and file writing |
| `internal/storage/history.go` | Snapshot history management, pruning |
| `internal/models/*` | Data structure definitions |
| `internal/models/errors.go` | Custom error types for domain errors |
| `internal/config/config.go` | Configuration loading and validation |
| `internal/monitoring/metrics.go` | Prometheus metrics registration |
| `internal/monitoring/health.go` | Health check endpoint handlers |

### 2.3 Dependency Graph

```
main.go
    │
    ├──▶ config.Config
    │
    ├──▶ api.Client
    │       └──▶ net/http.Client
    │
    ├──▶ storage.StateManager
    │       └──▶ models.State
    │
    ├──▶ aggregator.Aggregator
    │       ├──▶ api.Client
    │       ├──▶ aggregator.MetricsCalculator
    │       └──▶ aggregator.ProtocolFilter
    │
    ├──▶ storage.Writer
    │       └──▶ models.Output
    │
    └──▶ monitoring.Server
            ├──▶ prometheus.Registry
            └──▶ health.Checker
```

---

## 3. Data Sources & API Specifications

### 3.1 Primary API Endpoints

#### 3.1.1 Oracle Data Endpoint

```
Endpoint: GET https://api.llama.fi/oracles
Method: GET
Authentication: None (public API)
Rate Limit: No official limit (recommend 15+ minute intervals)
Cache-Control: public, max-age=600 (10 minutes)
```

**Request Headers:**
```
User-Agent: SwitchboardOracleExtractor/1.0 (Go)
Accept: application/json
```

**Response Content-Type:** `application/json`

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `oracles` | `map[string][]string` | Maps oracle name to list of protocol names |
| `chart` | `map[string]map[string]map[string]float64` | `timestamp -> oracle -> chain -> TVS` |
| `chainChart` | `map[string]map[string]map[string]float64` | Same structure as `chart` (chain-specific) |
| `oraclesTVS` | `map[string]map[string]map[string]float64` | `oracle -> protocol -> chain -> TVS` |
| `chainsByOracle` | `map[string][]string` | Maps oracle name to list of chain names |

#### 3.1.2 Protocol Metadata Endpoint

```
Endpoint: GET https://api.llama.fi/lite/protocols2?b=2
Method: GET
Authentication: None (public API)
Rate Limit: No official limit
```

**Request Headers:**
```
User-Agent: SwitchboardOracleExtractor/1.0 (Go)
Accept: application/json
```

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `protocols` | `[]Protocol` | Array of protocol objects |
| `chains` | `[]string` | List of all chain names |
| `parentProtocols` | `[]ParentProtocol` | Parent protocol relationships |

**Protocol Object Fields:**

| Field | Type | Description | Nullable |
|-------|------|-------------|----------|
| `id` | `string` | Unique protocol identifier | No |
| `name` | `string` | Display name | No |
| `slug` | `string` | URL-friendly identifier | No |
| `chain` | `string` | Primary chain | No |
| `chains` | `[]string` | All supported chains | Yes |
| `category` | `string` | Protocol category (Lending, CDP, etc.) | No |
| `tvl` | `float64` | Current total value locked | Yes |
| `oracles` | `[]string` | List of oracle names used | Yes |
| `oracle` | `string` | Single oracle name (legacy) | Yes |
| `symbol` | `string` | Token symbol | Yes |
| `url` | `string` | Protocol website URL | Yes |

### 3.2 API Response Timing

```
DefiLlama Update Schedule:
┌────────────────────────────────────────────────────────────────┐
│ :00 - Protocol adapters start running                          │
│ :20 - Most adapters complete                                   │
│ :21 - Backend cache invalidation                               │
│ :22 - New oracle data generated (writeOracles cron)            │
│ :22 - Data available via API                                   │
│ :32 - API cache expires (10 min TTL)                           │
└────────────────────────────────────────────────────────────────┘

Recommended polling: Every 15 minutes, starting at :07, :22, :37, :52
This aligns with DefiLlama's update cycle while avoiding peak times.
```

### 3.3 Switchboard-Specific Data

**Oracle Name (exact match required):** `"Switchboard"`

**Expected Chains:**
- Solana (primary)
- Sui
- Aptos
- Arbitrum
- Ethereum

**Expected Protocol Count:** ~21 protocols

**Protocol Categories:**
- Lending
- CDP (Collateralized Debt Position)
- Liquid Staking
- Dexes
- Derivatives

---

## 4. Data Models & Structures

### 4.1 API Response Models

#### 4.1.1 Oracle API Response

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

#### 4.1.2 Protocol API Response

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

### 4.2 Internal Models

#### 4.2.1 Aggregated Protocol

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

#### 4.2.2 Chain Breakdown

```go
// ChainBreakdown contains TVS metrics for a single chain
type ChainBreakdown struct {
    Chain         string  `json:"chain"`
    TVS           float64 `json:"tvs"`
    ProtocolCount int     `json:"protocol_count"`
    Percentage    float64 `json:"percentage"`
}
```

#### 4.2.3 Snapshot

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

#### 4.2.4 Metrics

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

### 4.3 Custom Error Types

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

### 4.4 Output Models

#### 4.4.1 Full Output

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

#### 4.4.2 State Model

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

## 5. Core Components

### 5.1 HTTP Client Component

#### 5.1.1 Complete Implementation

```go
// internal/api/client.go

package api

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "log/slog"
    "math/rand"
    "net/http"
    "sync"
    "time"

    "github.com/yourorg/switchboard-extractor/internal/models"
)

// ClientConfig holds HTTP client configuration
type ClientConfig struct {
    OracleURL      string
    ProtocolsURL   string
    Timeout        time.Duration
    MaxRetries     int
    RetryBaseDelay time.Duration
    RetryMaxDelay  time.Duration
    UserAgent      string
}

// DefaultClientConfig returns sensible defaults
func DefaultClientConfig() ClientConfig {
    return ClientConfig{
        OracleURL:      "https://api.llama.fi/oracles",
        ProtocolsURL:   "https://api.llama.fi/lite/protocols2?b=2",
        Timeout:        30 * time.Second,
        MaxRetries:     3,
        RetryBaseDelay: 1 * time.Second,
        RetryMaxDelay:  30 * time.Second,
        UserAgent:      "SwitchboardOracleExtractor/1.0 (Go)",
    }
}

// Client implements the APIClient interface
type Client struct {
    httpClient *http.Client
    config     ClientConfig
    logger     *slog.Logger
}

// NewClient creates a new API client
func NewClient(config ClientConfig, logger *slog.Logger) *Client {
    return &Client{
        httpClient: &http.Client{
            Timeout: config.Timeout,
        },
        config: config,
        logger: logger,
    }
}

// FetchOracles retrieves oracle data from /oracles endpoint
func (c *Client) FetchOracles(ctx context.Context) (*models.OracleAPIResponse, error) {
    var response models.OracleAPIResponse
    if err := c.fetchWithRetry(ctx, c.config.OracleURL, &response); err != nil {
        return nil, fmt.Errorf("fetching oracles: %w", err)
    }
    return &response, nil
}

// FetchProtocols retrieves protocol metadata from /lite/protocols2
func (c *Client) FetchProtocols(ctx context.Context) (*models.ProtocolAPIResponse, error) {
    var response models.ProtocolAPIResponse
    if err := c.fetchWithRetry(ctx, c.config.ProtocolsURL, &response); err != nil {
        return nil, fmt.Errorf("fetching protocols: %w", err)
    }
    return &response, nil
}

// FetchBoth fetches both endpoints in parallel
func (c *Client) FetchBoth(ctx context.Context) (*models.OracleAPIResponse, *models.ProtocolAPIResponse, error) {
    var (
        oracleResp   *models.OracleAPIResponse
        protocolResp *models.ProtocolAPIResponse
        oracleErr    error
        protocolErr  error
        wg           sync.WaitGroup
    )

    wg.Add(2)

    go func() {
        defer wg.Done()
        oracleResp, oracleErr = c.FetchOracles(ctx)
    }()

    go func() {
        defer wg.Done()
        protocolResp, protocolErr = c.FetchProtocols(ctx)
    }()

    wg.Wait()

    // Return first error encountered
    if oracleErr != nil {
        return nil, nil, fmt.Errorf("oracle fetch failed: %w", oracleErr)
    }
    if protocolErr != nil {
        return nil, nil, fmt.Errorf("protocol fetch failed: %w", protocolErr)
    }

    return oracleResp, protocolResp, nil
}

// fetchWithRetry performs HTTP GET with retry logic
func (c *Client) fetchWithRetry(ctx context.Context, url string, target interface{}) error {
    var lastErr error

    for attempt := 0; attempt < c.config.MaxRetries; attempt++ {
        if attempt > 0 {
            delay := c.calculateBackoff(attempt)
            c.logger.Warn("retrying request",
                slog.String("url", url),
                slog.Int("attempt", attempt+1),
                slog.Duration("delay", delay),
            )

            select {
            case <-ctx.Done():
                return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
            case <-time.After(delay):
            }
        }

        err := c.doRequest(ctx, url, target)
        if err == nil {
            return nil
        }

        lastErr = err

        // Check if error is retryable
        var apiErr *models.APIError
        if errors.As(err, &apiErr) && !apiErr.IsRetryable() {
            return err // Don't retry non-retryable errors
        }

        c.logger.Error("request failed",
            slog.String("url", url),
            slog.Int("attempt", attempt+1),
            slog.String("error", err.Error()),
        )
    }

    return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// doRequest performs a single HTTP GET request
func (c *Client) doRequest(ctx context.Context, url string, target interface{}) error {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return fmt.Errorf("creating request: %w", err)
    }

    req.Header.Set("User-Agent", c.config.UserAgent)
    req.Header.Set("Accept", "application/json")

    start := time.Now()
    resp, err := c.httpClient.Do(req)
    duration := time.Since(start)

    if err != nil {
        return &models.APIError{
            Endpoint:   url,
            StatusCode: 0,
            Message:    "request failed",
            Err:        err,
        }
    }
    defer resp.Body.Close()

    c.logger.Debug("request completed",
        slog.String("url", url),
        slog.Int("status", resp.StatusCode),
        slog.Duration("duration", duration),
    )

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
        return &models.APIError{
            Endpoint:   url,
            StatusCode: resp.StatusCode,
            Message:    string(body),
        }
    }

    if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
        return fmt.Errorf("decoding response: %w", err)
    }

    return nil
}

// calculateBackoff returns the delay for exponential backoff with jitter
func (c *Client) calculateBackoff(attempt int) time.Duration {
    // Exponential backoff: baseDelay * 2^attempt
    delay := c.config.RetryBaseDelay * time.Duration(1<<attempt)

    // Add jitter (±10%)
    jitter := time.Duration(float64(delay) * (rand.Float64()*0.2 - 0.1))
    delay += jitter

    // Cap at max delay
    if delay > c.config.RetryMaxDelay {
        delay = c.config.RetryMaxDelay
    }

    return delay
}
```

### 5.2 Aggregator Component Interface

```go
// internal/aggregator/aggregator.go

package aggregator

import (
    "context"
    "fmt"
    "log/slog"
    "sort"
    "time"

    "github.com/yourorg/switchboard-extractor/internal/api"
    "github.com/yourorg/switchboard-extractor/internal/models"
)

// Aggregator orchestrates the data processing pipeline
type Aggregator struct {
    client     *api.Client
    oracleName string
    logger     *slog.Logger
}

// NewAggregator creates a new aggregator instance
func NewAggregator(client *api.Client, oracleName string, logger *slog.Logger) *Aggregator {
    return &Aggregator{
        client:     client,
        oracleName: oracleName,
        logger:     logger,
    }
}

// Result contains the aggregated data
type Result struct {
    Protocols       []models.AggregatedProtocol
    ChainBreakdown  []models.ChainBreakdown
    CategoryBreakdown []models.CategoryBreakdown
    Metrics         models.Metrics
    Snapshot        models.Snapshot
    LatestTimestamp int64
}

// Process fetches and aggregates all oracle data
func (a *Aggregator) Process(ctx context.Context, history []models.Snapshot) (*Result, error) {
    // Fetch both APIs in parallel
    oracleResp, protocolResp, err := a.client.FetchBoth(ctx)
    if err != nil {
        return nil, fmt.Errorf("fetching API data: %w", err)
    }

    // Extract latest timestamp
    latestTimestamp, err := GetLatestTimestamp(oracleResp.Chart)
    if err != nil {
        return nil, fmt.Errorf("extracting timestamp: %w", err)
    }

    // Validate oracle exists
    if _, exists := oracleResp.Oracles[a.oracleName]; !exists {
        return nil, fmt.Errorf("%w: %s", models.ErrOracleNotFound, a.oracleName)
    }

    // Filter protocols
    protocols := FilterProtocolsByOracle(protocolResp.Protocols, a.oracleName)
    a.logger.Info("filtered protocols",
        slog.Int("count", len(protocols)),
        slog.String("oracle", a.oracleName),
    )

    // Get TVS data for this oracle
    oracleTVS := oracleResp.OraclesTVS[a.oracleName]

    // Aggregate protocols with TVS
    aggregated := a.aggregateProtocols(protocols, oracleTVS)

    // Calculate breakdowns
    chainBreakdown := CalculateChainBreakdown(oracleTVS)
    categoryBreakdown := CalculateCategoryBreakdown(aggregated)

    // Calculate current snapshot
    currentTVS := oracleResp.Chart[fmt.Sprintf("%d", latestTimestamp)][a.oracleName]
    snapshot := CreateSnapshot(latestTimestamp, currentTVS, len(aggregated))

    // Calculate metrics with historical comparison
    metrics := CalculateMetrics(snapshot, history, aggregated)

    return &Result{
        Protocols:         aggregated,
        ChainBreakdown:    chainBreakdown,
        CategoryBreakdown: categoryBreakdown,
        Metrics:           metrics,
        Snapshot:          snapshot,
        LatestTimestamp:   latestTimestamp,
    }, nil
}

// aggregateProtocols merges protocol metadata with TVS data
func (a *Aggregator) aggregateProtocols(
    protocols []models.Protocol,
    oracleTVS map[string]map[string]float64,
) []models.AggregatedProtocol {
    result := make([]models.AggregatedProtocol, 0, len(protocols))

    for _, p := range protocols {
        tvsByChain := oracleTVS[p.Name]
        totalTVS := sumMapValues(tvsByChain)

        chains := p.Chains
        if len(chains) == 0 && p.Chain != "" {
            chains = []string{p.Chain}
        }

        result = append(result, models.AggregatedProtocol{
            Name:       p.Name,
            Slug:       p.Slug,
            Category:   p.Category,
            TVL:        p.TVL,
            Chains:     chains,
            URL:        fmt.Sprintf("https://defillama.com/protocol/%s", p.Slug),
            TVS:        totalTVS,
            TVSByChain: tvsByChain,
        })
    }

    // Sort by TVL descending and assign ranks
    sort.Slice(result, func(i, j int) bool {
        return result[i].TVL > result[j].TVL
    })

    for i := range result {
        result[i].Rank = i + 1
    }

    return result
}

func sumMapValues(m map[string]float64) float64 {
    var sum float64
    for _, v := range m {
        sum += v
    }
    return sum
}
```

---

## 6. Incremental Update Strategy

### 6.1 Overview

The incremental update strategy minimizes redundant processing by:
1. Tracking the timestamp of the last processed data
2. Comparing with the latest available timestamp
3. Only processing when new data is detected
4. Maintaining a rolling window of historical snapshots

### 6.2 State Manager Implementation

```go
// internal/storage/state.go

package storage

import (
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "time"

    "github.com/yourorg/switchboard-extractor/internal/models"
)

// StateManager handles incremental update state
type StateManager struct {
    filePath string
}

// NewStateManager creates a new state manager
func NewStateManager(outputDir string) *StateManager {
    return &StateManager{
        filePath: filepath.Join(outputDir, "state.json"),
    }
}

// Load reads state from disk, returns empty state if not found
func (sm *StateManager) Load() (*models.State, error) {
    data, err := os.ReadFile(sm.filePath)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            return &models.State{}, nil
        }
        return nil, fmt.Errorf("reading state file: %w", err)
    }

    var state models.State
    if err := json.Unmarshal(data, &state); err != nil {
        return nil, fmt.Errorf("%w: %v", models.ErrStateCorrupted, err)
    }

    return &state, nil
}

// Save writes state to disk atomically
func (sm *StateManager) Save(state *models.State) error {
    state.LastUpdatedISO = time.Unix(state.LastUpdated, 0).UTC().Format(time.RFC3339)

    data, err := json.MarshalIndent(state, "", "  ")
    if err != nil {
        return fmt.Errorf("marshaling state: %w", err)
    }

    return WriteAtomic(sm.filePath, data, 0644)
}

// ShouldUpdate determines if new data should be processed
func (sm *StateManager) ShouldUpdate(state *models.State, latestTimestamp int64) bool {
    return latestTimestamp > state.LastUpdated
}

// UpdateState creates a new state from extraction result
func (sm *StateManager) UpdateState(
    state *models.State,
    oracleName string,
    timestamp int64,
    protocolCount int,
    tvs float64,
    snapshots []models.Snapshot,
) *models.State {
    var oldest, newest int64
    if len(snapshots) > 0 {
        oldest = snapshots[len(snapshots)-1].Timestamp
        newest = snapshots[0].Timestamp
    }

    return &models.State{
        OracleName:        oracleName,
        LastUpdated:       timestamp,
        LastProtocolCount: protocolCount,
        LastTVS:           tvs,
        SnapshotCount:     len(snapshots),
        OldestSnapshot:    oldest,
        NewestSnapshot:    newest,
    }
}
```

### 6.3 History Manager Implementation

```go
// internal/storage/history.go

package storage

import (
    "encoding/json"
    "os"
    "sort"
    "time"

    "github.com/yourorg/switchboard-extractor/internal/models"
)

const (
    DefaultRetentionDays = 90
    SecondsPerDay        = 24 * 60 * 60
)

// HistoryManager handles historical snapshot data
type HistoryManager struct {
    retentionDays int
}

// NewHistoryManager creates a new history manager
func NewHistoryManager(retentionDays int) *HistoryManager {
    if retentionDays <= 0 {
        retentionDays = DefaultRetentionDays
    }
    return &HistoryManager{retentionDays: retentionDays}
}

// LoadFromOutput reads historical snapshots from the full output file
func (hm *HistoryManager) LoadFromOutput(outputPath string) ([]models.Snapshot, error) {
    data, err := os.ReadFile(outputPath)
    if err != nil {
        if os.IsNotExist(err) {
            return []models.Snapshot{}, nil
        }
        return nil, err
    }

    var output models.FullOutput
    if err := json.Unmarshal(data, &output); err != nil {
        return nil, err
    }

    return output.Historical, nil
}

// Append adds a new snapshot, maintaining sort order (newest first)
func (hm *HistoryManager) Append(snapshots []models.Snapshot, newSnapshot models.Snapshot) []models.Snapshot {
    // Check for duplicate
    for i, s := range snapshots {
        if s.Timestamp == newSnapshot.Timestamp {
            // Replace existing with newer data
            snapshots[i] = newSnapshot
            return snapshots
        }
    }

    // Append and re-sort
    snapshots = append(snapshots, newSnapshot)
    sort.Slice(snapshots, func(i, j int) bool {
        return snapshots[i].Timestamp > snapshots[j].Timestamp
    })

    return snapshots
}

// Prune removes snapshots older than retentionDays
func (hm *HistoryManager) Prune(snapshots []models.Snapshot) []models.Snapshot {
    cutoffTime := time.Now().Unix() - int64(hm.retentionDays*SecondsPerDay)

    result := make([]models.Snapshot, 0, len(snapshots))
    for _, s := range snapshots {
        if s.Timestamp >= cutoffTime {
            result = append(result, s)
        }
    }

    return result
}

// Deduplicate removes duplicate timestamps, keeping the one with newer extraction time
func (hm *HistoryManager) Deduplicate(snapshots []models.Snapshot) []models.Snapshot {
    seen := make(map[int64]bool)
    result := make([]models.Snapshot, 0, len(snapshots))

    for _, s := range snapshots {
        if !seen[s.Timestamp] {
            seen[s.Timestamp] = true
            result = append(result, s)
        }
    }

    return result
}
```

---

## 7. Custom Aggregation Logic (Go Implementation)

This section provides complete Go implementations for all aggregation algorithms, addressing the evaluation feedback about pseudocode.

### 7.1 Metric Calculations

```go
// internal/aggregator/metrics.go

package aggregator

import (
    "math"
    "sort"
    "time"

    "github.com/yourorg/switchboard-extractor/internal/models"
)

const (
    // Time offsets for historical comparison
    Hours24 = 24 * 60 * 60
    Days7   = 7 * 24 * 60 * 60
    Days30  = 30 * 24 * 60 * 60

    // Tolerance for finding historical snapshots (2 hours)
    SnapshotTolerance = 2 * 60 * 60
)

// CalculatePercentageChange computes the percentage change between two values
// Formula: ((new - old) / old) * 100
// Returns 0 if old is 0 to avoid division by zero
func CalculatePercentageChange(old, new float64) float64 {
    if old == 0 {
        return 0
    }
    change := ((new - old) / old) * 100
    // Round to 2 decimal places
    return math.Round(change*100) / 100
}

// FindSnapshotAtTime finds the snapshot closest to the target time
// Returns nil if no snapshot found within tolerance
func FindSnapshotAtTime(snapshots []models.Snapshot, targetTime int64, tolerance int64) *models.Snapshot {
    var closest *models.Snapshot
    minDiff := int64(math.MaxInt64)

    for i := range snapshots {
        diff := abs(snapshots[i].Timestamp - targetTime)
        if diff <= tolerance && diff < minDiff {
            minDiff = diff
            closest = &snapshots[i]
        }
    }

    return closest
}

// abs returns the absolute value of an int64
func abs(x int64) int64 {
    if x < 0 {
        return -x
    }
    return x
}

// CalculateMetrics computes all derived metrics from current snapshot and history
func CalculateMetrics(
    current models.Snapshot,
    history []models.Snapshot,
    protocols []models.AggregatedProtocol,
) models.Metrics {
    now := time.Now().Unix()

    // Find historical snapshots
    snapshot24h := FindSnapshotAtTime(history, now-Hours24, SnapshotTolerance)
    snapshot7d := FindSnapshotAtTime(history, now-Days7, SnapshotTolerance)
    snapshot30d := FindSnapshotAtTime(history, now-Days30, SnapshotTolerance)

    metrics := models.Metrics{
        CurrentTVS:    current.TVS,
        ProtocolCount: current.ProtocolCount,
        ChainCount:    current.ChainCount,
    }

    // Calculate TVS changes
    if snapshot24h != nil {
        metrics.TVS24hAgo = snapshot24h.TVS
        metrics.Change24h = CalculatePercentageChange(snapshot24h.TVS, current.TVS)
    }

    if snapshot7d != nil {
        metrics.TVS7dAgo = snapshot7d.TVS
        metrics.Change7d = CalculatePercentageChange(snapshot7d.TVS, current.TVS)
        metrics.ProtocolGrowth7d = current.ProtocolCount - snapshot7d.ProtocolCount
    }

    if snapshot30d != nil {
        metrics.TVS30dAgo = snapshot30d.TVS
        metrics.Change30d = CalculatePercentageChange(snapshot30d.TVS, current.TVS)
        metrics.ProtocolGrowth30d = current.ProtocolCount - snapshot30d.ProtocolCount
    }

    // Extract unique categories
    categorySet := make(map[string]bool)
    for _, p := range protocols {
        if p.Category != "" {
            categorySet[p.Category] = true
        }
    }
    categories := make([]string, 0, len(categorySet))
    for cat := range categorySet {
        categories = append(categories, cat)
    }
    sort.Strings(categories)
    metrics.Categories = categories

    // Find largest protocol
    if len(protocols) > 0 {
        largest := protocols[0] // Already sorted by TVL descending
        metrics.LargestProtocol = largest.Name
        metrics.LargestProtocolTVL = largest.TVL
    }

    return metrics
}

// GetLatestTimestamp extracts the most recent timestamp from chart data
func GetLatestTimestamp(chart map[string]map[string]map[string]float64) (int64, error) {
    if len(chart) == 0 {
        return 0, models.ErrInvalidResponse
    }

    var maxTimestamp int64
    for tsStr := range chart {
        var ts int64
        if _, err := fmt.Sscanf(tsStr, "%d", &ts); err != nil {
            continue
        }
        if ts > maxTimestamp {
            maxTimestamp = ts
        }
    }

    if maxTimestamp == 0 {
        return 0, models.ErrInvalidResponse
    }

    return maxTimestamp, nil
}

// CreateSnapshot creates a new snapshot from current TVS data
func CreateSnapshot(
    timestamp int64,
    tvsByChain map[string]float64,
    protocolCount int,
) models.Snapshot {
    t := time.Unix(timestamp, 0).UTC()

    var totalTVS float64
    for _, tvs := range tvsByChain {
        totalTVS += tvs
    }

    return models.Snapshot{
        Timestamp:     timestamp,
        Date:          t.Format("2006-01-02"),
        DateTime:      t.Format(time.RFC3339),
        TVS:           totalTVS,
        TVSByChain:    tvsByChain,
        ProtocolCount: protocolCount,
        ChainCount:    len(tvsByChain),
    }
}
```

### 7.2 Protocol Filtering

```go
// internal/aggregator/filter.go

package aggregator

import (
    "slices"

    "github.com/yourorg/switchboard-extractor/internal/models"
)

// FilterProtocolsByOracle filters protocols using the specified oracle
//
// Matching rules (in order of priority):
// 1. Check if protocol.Oracles array contains oracleName
// 2. Fallback: Check if protocol.Oracle == oracleName
//
// Oracle names are case-sensitive.
func FilterProtocolsByOracle(protocols []models.Protocol, oracleName string) []models.Protocol {
    result := make([]models.Protocol, 0)

    for _, p := range protocols {
        if protocolUsesOracle(p, oracleName) {
            result = append(result, p)
        }
    }

    return result
}

// protocolUsesOracle checks if a protocol uses the specified oracle
func protocolUsesOracle(p models.Protocol, oracleName string) bool {
    // Check oracles array (preferred)
    if len(p.Oracles) > 0 {
        return slices.Contains(p.Oracles, oracleName)
    }

    // Fallback to legacy oracle field
    return p.Oracle == oracleName
}

// FilterMultiOracleProtocols returns protocols that use multiple oracles
// including the specified one (useful for understanding shared usage)
func FilterMultiOracleProtocols(protocols []models.Protocol, oracleName string) []models.Protocol {
    result := make([]models.Protocol, 0)

    for _, p := range protocols {
        if len(p.Oracles) > 1 && slices.Contains(p.Oracles, oracleName) {
            result = append(result, p)
        }
    }

    return result
}
```

### 7.3 Breakdown Calculations

```go
// internal/aggregator/breakdown.go

package aggregator

import (
    "math"
    "sort"

    "github.com/yourorg/switchboard-extractor/internal/models"
)

// CalculateChainBreakdown aggregates TVS data by chain
func CalculateChainBreakdown(oracleTVS map[string]map[string]float64) []models.ChainBreakdown {
    chainTotals := make(map[string]float64)
    chainProtocolCounts := make(map[string]int)

    // Aggregate TVS and protocol counts by chain
    for _, chainData := range oracleTVS {
        for chain, tvs := range chainData {
            chainTotals[chain] += tvs
            chainProtocolCounts[chain]++
        }
    }

    // Calculate total TVS
    var totalTVS float64
    for _, tvs := range chainTotals {
        totalTVS += tvs
    }

    // Build result slice
    result := make([]models.ChainBreakdown, 0, len(chainTotals))
    for chain, tvs := range chainTotals {
        percentage := 0.0
        if totalTVS > 0 {
            percentage = math.Round((tvs/totalTVS)*10000) / 100 // Round to 2 decimal places
        }

        result = append(result, models.ChainBreakdown{
            Chain:         chain,
            TVS:           tvs,
            ProtocolCount: chainProtocolCounts[chain],
            Percentage:    percentage,
        })
    }

    // Sort by TVS descending
    sort.Slice(result, func(i, j int) bool {
        return result[i].TVS > result[j].TVS
    })

    return result
}

// CalculateCategoryBreakdown aggregates TVS data by protocol category
func CalculateCategoryBreakdown(protocols []models.AggregatedProtocol) []models.CategoryBreakdown {
    categoryTotals := make(map[string]float64)
    categoryProtocolCounts := make(map[string]int)

    // Aggregate TVL and protocol counts by category
    for _, p := range protocols {
        if p.Category == "" {
            continue
        }
        categoryTotals[p.Category] += p.TVL
        categoryProtocolCounts[p.Category]++
    }

    // Calculate total TVL
    var totalTVL float64
    for _, tvl := range categoryTotals {
        totalTVL += tvl
    }

    // Build result slice
    result := make([]models.CategoryBreakdown, 0, len(categoryTotals))
    for category, tvl := range categoryTotals {
        percentage := 0.0
        if totalTVL > 0 {
            percentage = math.Round((tvl/totalTVL)*10000) / 100
        }

        result = append(result, models.CategoryBreakdown{
            Category:      category,
            TVS:           tvl, // Using TVL as proxy for TVS
            ProtocolCount: categoryProtocolCounts[category],
            Percentage:    percentage,
        })
    }

    // Sort by TVS descending
    sort.Slice(result, func(i, j int) bool {
        return result[i].TVS > result[j].TVS
    })

    return result
}
```

---

## 8. Storage & Caching

### 8.1 Atomic File Writer

```go
// internal/storage/writer.go

package storage

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"

    "github.com/yourorg/switchboard-extractor/internal/models"
)

// Writer handles JSON file output
type Writer struct {
    outputDir string
}

// NewWriter creates a new file writer
func NewWriter(outputDir string) (*Writer, error) {
    if err := os.MkdirAll(outputDir, 0755); err != nil {
        return nil, fmt.Errorf("creating output directory: %w", err)
    }
    return &Writer{outputDir: outputDir}, nil
}

// WriteAll writes all output files atomically
func (w *Writer) WriteAll(output *models.FullOutput) error {
    // Full output (indented)
    fullPath := filepath.Join(w.outputDir, "switchboard-oracle-data.json")
    if err := w.writeJSON(fullPath, output, true); err != nil {
        return fmt.Errorf("writing full output: %w", err)
    }

    // Minified output
    minPath := filepath.Join(w.outputDir, "switchboard-oracle-data.min.json")
    if err := w.writeJSON(minPath, output, false); err != nil {
        return fmt.Errorf("writing minified output: %w", err)
    }

    // Summary output (current snapshot only)
    summary := models.SummaryOutput{
        Version:  output.Version,
        Oracle:   output.Oracle,
        Metadata: output.Metadata,
        Summary:  output.Summary,
        Metrics:  output.Metrics,
    }
    summaryPath := filepath.Join(w.outputDir, "switchboard-summary.json")
    if err := w.writeJSON(summaryPath, summary, true); err != nil {
        return fmt.Errorf("writing summary: %w", err)
    }

    return nil
}

// writeJSON writes a JSON file with optional indentation
func (w *Writer) writeJSON(path string, data interface{}, indent bool) error {
    var jsonData []byte
    var err error

    if indent {
        jsonData, err = json.MarshalIndent(data, "", "  ")
    } else {
        jsonData, err = json.Marshal(data)
    }

    if err != nil {
        return fmt.Errorf("marshaling JSON: %w", err)
    }

    return WriteAtomic(path, jsonData, 0644)
}

// WriteAtomic writes data to a file atomically using temp file + rename
func WriteAtomic(path string, data []byte, perm os.FileMode) error {
    dir := filepath.Dir(path)

    // Create temp file in same directory for atomic rename
    tmpFile, err := os.CreateTemp(dir, ".tmp-*")
    if err != nil {
        return fmt.Errorf("creating temp file: %w", err)
    }
    tmpPath := tmpFile.Name()

    // Clean up on any error
    defer func() {
        if tmpPath != "" {
            os.Remove(tmpPath)
        }
    }()

    // Write data
    if _, err := tmpFile.Write(data); err != nil {
        tmpFile.Close()
        return fmt.Errorf("writing data: %w", err)
    }

    // Sync to disk
    if err := tmpFile.Sync(); err != nil {
        tmpFile.Close()
        return fmt.Errorf("syncing file: %w", err)
    }

    // Close file
    if err := tmpFile.Close(); err != nil {
        return fmt.Errorf("closing file: %w", err)
    }

    // Set permissions
    if err := os.Chmod(tmpPath, perm); err != nil {
        return fmt.Errorf("setting permissions: %w", err)
    }

    // Atomic rename
    if err := os.Rename(tmpPath, path); err != nil {
        return fmt.Errorf("renaming file: %w", err)
    }

    tmpPath = "" // Prevent deferred cleanup
    return nil
}
```

---

## 9. Error Handling & Resilience

### 9.1 Error Wrapping Patterns

```go
// Example of proper error wrapping throughout the codebase

// In API client
func (c *Client) FetchOracles(ctx context.Context) (*models.OracleAPIResponse, error) {
    var response models.OracleAPIResponse
    if err := c.fetchWithRetry(ctx, c.config.OracleURL, &response); err != nil {
        // Wrap with context about what operation failed
        return nil, fmt.Errorf("fetching oracles from %s: %w", c.config.OracleURL, err)
    }
    return &response, nil
}

// In aggregator
func (a *Aggregator) Process(ctx context.Context, history []models.Snapshot) (*Result, error) {
    oracleResp, protocolResp, err := a.client.FetchBoth(ctx)
    if err != nil {
        // Wrap to add aggregator context
        return nil, fmt.Errorf("aggregator.Process: %w", err)
    }
    // ...
}

// In main extraction loop
func (e *Extractor) Run(ctx context.Context) error {
    result, err := e.aggregator.Process(ctx, history)
    if err != nil {
        // Check for specific error types
        if errors.Is(err, models.ErrNoNewData) {
            e.logger.Info("no new data, skipping extraction")
            return nil
        }
        if errors.Is(err, models.ErrOracleNotFound) {
            e.logger.Error("oracle not found", slog.String("oracle", e.oracleName))
            return err // Don't retry this
        }
        // Wrap for context
        return fmt.Errorf("extraction failed: %w", err)
    }
    // ...
}

// Checking error types
func handleError(err error) {
    var apiErr *models.APIError
    if errors.As(err, &apiErr) {
        if apiErr.IsRetryable() {
            // Schedule retry
        } else {
            // Log and alert
        }
    }

    var validationErr *models.ValidationError
    if errors.As(err, &validationErr) {
        // Log validation failure with field details
    }
}
```

### 9.2 Retry Configuration

```go
type RetryConfig struct {
    MaxAttempts   int           // Maximum retry attempts (default: 3)
    BaseDelay     time.Duration // Initial delay (default: 1s)
    MaxDelay      time.Duration // Maximum delay (default: 30s)
    Multiplier    float64       // Backoff multiplier (default: 2.0)
    RetryableHTTP []int         // HTTP codes to retry (default: 429, 500, 502, 503, 504)
}
```

### 9.3 Graceful Degradation Table

| Failure | Degradation Strategy |
|---------|---------------------|
| Oracle API fails | Use cached data, skip update |
| Protocol API fails | Use cached data, skip update |
| Both APIs fail | Keep existing files, retry next cycle |
| State file corrupt | Delete and restart |
| Output write fails | Keep previous output, log error |

---

## 10. Configuration & Environment

### 10.1 Configuration File (YAML)

```yaml
# config.yaml

# Oracle settings
oracle:
  name: "Switchboard"
  website: "https://switchboard.xyz"
  documentation: "https://docs.switchboard.xyz"

# API settings
api:
  oracle_url: "https://api.llama.fi/oracles"
  protocols_url: "https://api.llama.fi/lite/protocols2?b=2"
  timeout: 30s
  max_retries: 3
  retry_base_delay: 1s
  retry_max_delay: 30s
  user_agent: "SwitchboardOracleExtractor/1.0"

# Output settings
output:
  directory: "./data"
  full_file: "switchboard-oracle-data.json"
  min_file: "switchboard-oracle-data.min.json"
  summary_file: "switchboard-summary.json"
  state_file: "state.json"

# History settings
history:
  retention_days: 90
  max_snapshots: 2160

# Scheduler settings (for daemon mode)
scheduler:
  enabled: true
  interval: 15m
  start_immediately: true

# Logging settings
logging:
  level: "info"
  format: "json"
  output: "stdout"

# Monitoring settings
monitoring:
  enabled: true
  port: 9090
  path: "/metrics"
```

### 10.2 Configuration Loading

```go
// internal/config/config.go

package config

import (
    "fmt"
    "os"
    "time"

    "gopkg.in/yaml.v3"
)

type Config struct {
    Oracle     OracleConfig     `yaml:"oracle"`
    API        APIConfig        `yaml:"api"`
    Output     OutputConfig     `yaml:"output"`
    History    HistoryConfig    `yaml:"history"`
    Scheduler  SchedulerConfig  `yaml:"scheduler"`
    Logging    LoggingConfig    `yaml:"logging"`
    Monitoring MonitoringConfig `yaml:"monitoring"`
}

type OracleConfig struct {
    Name          string `yaml:"name"`
    Website       string `yaml:"website"`
    Documentation string `yaml:"documentation"`
}

type APIConfig struct {
    OracleURL      string        `yaml:"oracle_url"`
    ProtocolsURL   string        `yaml:"protocols_url"`
    Timeout        time.Duration `yaml:"timeout"`
    MaxRetries     int           `yaml:"max_retries"`
    RetryBaseDelay time.Duration `yaml:"retry_base_delay"`
    RetryMaxDelay  time.Duration `yaml:"retry_max_delay"`
    UserAgent      string        `yaml:"user_agent"`
}

type OutputConfig struct {
    Directory   string `yaml:"directory"`
    FullFile    string `yaml:"full_file"`
    MinFile     string `yaml:"min_file"`
    SummaryFile string `yaml:"summary_file"`
    StateFile   string `yaml:"state_file"`
}

type HistoryConfig struct {
    RetentionDays int `yaml:"retention_days"`
    MaxSnapshots  int `yaml:"max_snapshots"`
}

type SchedulerConfig struct {
    Enabled          bool          `yaml:"enabled"`
    Interval         time.Duration `yaml:"interval"`
    StartImmediately bool          `yaml:"start_immediately"`
}

type LoggingConfig struct {
    Level  string `yaml:"level"`
    Format string `yaml:"format"`
    Output string `yaml:"output"`
}

type MonitoringConfig struct {
    Enabled bool   `yaml:"enabled"`
    Port    int    `yaml:"port"`
    Path    string `yaml:"path"`
}

// Load reads configuration from file and applies environment overrides
func Load(path string) (*Config, error) {
    cfg := DefaultConfig()

    // Load from file if exists
    if path != "" {
        data, err := os.ReadFile(path)
        if err != nil && !os.IsNotExist(err) {
            return nil, fmt.Errorf("reading config file: %w", err)
        }
        if err == nil {
            if err := yaml.Unmarshal(data, cfg); err != nil {
                return nil, fmt.Errorf("parsing config file: %w", err)
            }
        }
    }

    // Apply environment variable overrides
    applyEnvOverrides(cfg)

    return cfg, nil
}

// DefaultConfig returns sensible defaults
func DefaultConfig() *Config {
    return &Config{
        Oracle: OracleConfig{
            Name:    "Switchboard",
            Website: "https://switchboard.xyz",
        },
        API: APIConfig{
            OracleURL:      "https://api.llama.fi/oracles",
            ProtocolsURL:   "https://api.llama.fi/lite/protocols2?b=2",
            Timeout:        30 * time.Second,
            MaxRetries:     3,
            RetryBaseDelay: 1 * time.Second,
            RetryMaxDelay:  30 * time.Second,
            UserAgent:      "SwitchboardOracleExtractor/1.0",
        },
        Output: OutputConfig{
            Directory:   "./data",
            FullFile:    "switchboard-oracle-data.json",
            MinFile:     "switchboard-oracle-data.min.json",
            SummaryFile: "switchboard-summary.json",
            StateFile:   "state.json",
        },
        History: HistoryConfig{
            RetentionDays: 90,
            MaxSnapshots:  2160,
        },
        Scheduler: SchedulerConfig{
            Enabled:          true,
            Interval:         15 * time.Minute,
            StartImmediately: true,
        },
        Logging: LoggingConfig{
            Level:  "info",
            Format: "json",
            Output: "stdout",
        },
        Monitoring: MonitoringConfig{
            Enabled: true,
            Port:    9090,
            Path:    "/metrics",
        },
    }
}

func applyEnvOverrides(cfg *Config) {
    if v := os.Getenv("ORACLE_NAME"); v != "" {
        cfg.Oracle.Name = v
    }
    if v := os.Getenv("OUTPUT_DIR"); v != "" {
        cfg.Output.Directory = v
    }
    if v := os.Getenv("LOG_LEVEL"); v != "" {
        cfg.Logging.Level = v
    }
    // Add more overrides as needed
}
```

---

## 11. Testing Strategy (Complete Implementation)

### 11.1 Table-Driven Tests for Metrics

```go
// internal/aggregator/metrics_test.go

package aggregator

import (
    "testing"

    "github.com/yourorg/switchboard-extractor/internal/models"
)

func TestCalculatePercentageChange(t *testing.T) {
    tests := []struct {
        name     string
        old      float64
        new      float64
        expected float64
    }{
        {
            name:     "positive change",
            old:      100,
            new:      110,
            expected: 10.0,
        },
        {
            name:     "negative change",
            old:      100,
            new:      90,
            expected: -10.0,
        },
        {
            name:     "no change",
            old:      100,
            new:      100,
            expected: 0.0,
        },
        {
            name:     "zero old value",
            old:      0,
            new:      100,
            expected: 0.0, // Avoid division by zero
        },
        {
            name:     "decimal result",
            old:      100,
            new:      106.52,
            expected: 6.52,
        },
        {
            name:     "large change",
            old:      1000000,
            new:      2500000,
            expected: 150.0,
        },
        {
            name:     "small values",
            old:      0.001,
            new:      0.002,
            expected: 100.0,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := CalculatePercentageChange(tt.old, tt.new)
            if result != tt.expected {
                t.Errorf("CalculatePercentageChange(%f, %f) = %f, want %f",
                    tt.old, tt.new, result, tt.expected)
            }
        })
    }
}

func TestFindSnapshotAtTime(t *testing.T) {
    snapshots := []models.Snapshot{
        {Timestamp: 1732867200, TVS: 100000000}, // Most recent
        {Timestamp: 1732780800, TVS: 95000000},  // 1 day ago
        {Timestamp: 1732694400, TVS: 90000000},  // 2 days ago
        {Timestamp: 1732262400, TVS: 85000000},  // 7 days ago
    }

    tests := []struct {
        name        string
        targetTime  int64
        tolerance   int64
        expectFound bool
        expectedTVS float64
    }{
        {
            name:        "exact match",
            targetTime:  1732867200,
            tolerance:   3600,
            expectFound: true,
            expectedTVS: 100000000,
        },
        {
            name:        "within tolerance",
            targetTime:  1732867200 + 1800, // 30 min off
            tolerance:   3600,
            expectFound: true,
            expectedTVS: 100000000,
        },
        {
            name:        "outside tolerance",
            targetTime:  1732867200 + 7200, // 2 hours off
            tolerance:   3600,
            expectFound: false,
        },
        {
            name:        "no snapshots match",
            targetTime:  1700000000,
            tolerance:   3600,
            expectFound: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := FindSnapshotAtTime(snapshots, tt.targetTime, tt.tolerance)

            if tt.expectFound {
                if result == nil {
                    t.Errorf("expected to find snapshot, got nil")
                } else if result.TVS != tt.expectedTVS {
                    t.Errorf("expected TVS %f, got %f", tt.expectedTVS, result.TVS)
                }
            } else {
                if result != nil {
                    t.Errorf("expected nil, got snapshot with timestamp %d", result.Timestamp)
                }
            }
        })
    }
}

func TestCalculateChainBreakdown(t *testing.T) {
    oracleTVS := map[string]map[string]float64{
        "Kamino Lend": {"Solana": 145000000},
        "marginfi":    {"Solana": 25000000},
        "Scallop":     {"Sui": 35000000},
        "MovePosition": {"Aptos": 20000000},
    }

    result := CalculateChainBreakdown(oracleTVS)

    // Should be sorted by TVS descending
    if len(result) != 3 {
        t.Errorf("expected 3 chains, got %d", len(result))
    }

    // First should be Solana (highest TVS)
    if result[0].Chain != "Solana" {
        t.Errorf("expected Solana first, got %s", result[0].Chain)
    }
    if result[0].TVS != 170000000 {
        t.Errorf("expected Solana TVS 170000000, got %f", result[0].TVS)
    }
    if result[0].ProtocolCount != 2 {
        t.Errorf("expected Solana protocol count 2, got %d", result[0].ProtocolCount)
    }

    // Verify percentages sum to ~100
    var totalPct float64
    for _, cb := range result {
        totalPct += cb.Percentage
    }
    if totalPct < 99.9 || totalPct > 100.1 {
        t.Errorf("percentages should sum to ~100, got %f", totalPct)
    }
}
```

### 11.2 Mock API Client for Testing

```go
// internal/api/mock_client.go

package api

import (
    "context"
    "encoding/json"
    "os"

    "github.com/yourorg/switchboard-extractor/internal/models"
)

// MockClient implements APIClient for testing
type MockClient struct {
    OracleResponse   *models.OracleAPIResponse
    ProtocolResponse *models.ProtocolAPIResponse
    OracleError      error
    ProtocolError    error
    FetchCount       int
}

// NewMockClient creates a mock client from test fixture files
func NewMockClient(oracleFixture, protocolFixture string) (*MockClient, error) {
    mock := &MockClient{}

    if oracleFixture != "" {
        data, err := os.ReadFile(oracleFixture)
        if err != nil {
            return nil, err
        }
        var resp models.OracleAPIResponse
        if err := json.Unmarshal(data, &resp); err != nil {
            return nil, err
        }
        mock.OracleResponse = &resp
    }

    if protocolFixture != "" {
        data, err := os.ReadFile(protocolFixture)
        if err != nil {
            return nil, err
        }
        var resp models.ProtocolAPIResponse
        if err := json.Unmarshal(data, &resp); err != nil {
            return nil, err
        }
        mock.ProtocolResponse = &resp
    }

    return mock, nil
}

func (m *MockClient) FetchOracles(ctx context.Context) (*models.OracleAPIResponse, error) {
    m.FetchCount++
    if m.OracleError != nil {
        return nil, m.OracleError
    }
    return m.OracleResponse, nil
}

func (m *MockClient) FetchProtocols(ctx context.Context) (*models.ProtocolAPIResponse, error) {
    m.FetchCount++
    if m.ProtocolError != nil {
        return nil, m.ProtocolError
    }
    return m.ProtocolResponse, nil
}

func (m *MockClient) FetchBoth(ctx context.Context) (*models.OracleAPIResponse, *models.ProtocolAPIResponse, error) {
    m.FetchCount += 2
    if m.OracleError != nil {
        return nil, nil, m.OracleError
    }
    if m.ProtocolError != nil {
        return nil, nil, m.ProtocolError
    }
    return m.OracleResponse, m.ProtocolResponse, nil
}
```

### 11.3 Integration Test Example

```go
// internal/aggregator/aggregator_test.go

package aggregator

import (
    "context"
    "log/slog"
    "os"
    "testing"

    "github.com/yourorg/switchboard-extractor/internal/api"
    "github.com/yourorg/switchboard-extractor/internal/models"
)

func TestAggregator_Process(t *testing.T) {
    // Create mock client with test fixtures
    mockClient, err := api.NewMockClient(
        "../../testdata/oracle_response.json",
        "../../testdata/protocol_response.json",
    )
    if err != nil {
        t.Fatalf("failed to create mock client: %v", err)
    }

    logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
    agg := NewAggregator(mockClient, "Switchboard", logger)

    history := []models.Snapshot{
        {Timestamp: 1732780800, TVS: 95000000, ProtocolCount: 20},
        {Timestamp: 1732262400, TVS: 85000000, ProtocolCount: 18},
    }

    ctx := context.Background()
    result, err := agg.Process(ctx, history)
    if err != nil {
        t.Fatalf("Process failed: %v", err)
    }

    // Verify results
    if len(result.Protocols) == 0 {
        t.Error("expected protocols, got none")
    }

    if len(result.ChainBreakdown) == 0 {
        t.Error("expected chain breakdown, got none")
    }

    if result.Metrics.CurrentTVS == 0 {
        t.Error("expected non-zero current TVS")
    }

    // Verify protocols are ranked
    for i, p := range result.Protocols {
        if p.Rank != i+1 {
            t.Errorf("protocol %s has rank %d, expected %d", p.Name, p.Rank, i+1)
        }
    }
}

func TestAggregator_Process_OracleNotFound(t *testing.T) {
    mockClient := &api.MockClient{
        OracleResponse: &models.OracleAPIResponse{
            Oracles: map[string][]string{
                "Chainlink": {"Protocol A"},
            },
        },
        ProtocolResponse: &models.ProtocolAPIResponse{},
    }

    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    agg := NewAggregator(mockClient, "Switchboard", logger)

    _, err := agg.Process(context.Background(), nil)
    if err == nil {
        t.Error("expected error for missing oracle")
    }
    if !errors.Is(err, models.ErrOracleNotFound) {
        t.Errorf("expected ErrOracleNotFound, got %v", err)
    }
}
```

### 11.4 Test Fixtures

Create `testdata/oracle_response.json`:
```json
{
  "oracles": {
    "Switchboard": ["Kamino Lend", "marginfi Lending", "Scallop Lend"]
  },
  "chart": {
    "1732867200": {
      "Switchboard": {
        "Solana": 170000000,
        "Sui": 35000000
      }
    }
  },
  "oraclesTVS": {
    "Switchboard": {
      "Kamino Lend": {"Solana": 145000000},
      "marginfi Lending": {"Solana": 25000000},
      "Scallop Lend": {"Sui": 35000000}
    }
  },
  "chainsByOracle": {
    "Switchboard": ["Solana", "Sui", "Aptos"]
  }
}
```

Create `testdata/protocol_response.json`:
```json
{
  "protocols": [
    {
      "id": "1",
      "name": "Kamino Lend",
      "slug": "kamino-lend",
      "chain": "Solana",
      "category": "Lending",
      "tvl": 145000000,
      "oracles": ["Switchboard"]
    },
    {
      "id": "2",
      "name": "marginfi Lending",
      "slug": "marginfi-lending",
      "chain": "Solana",
      "category": "Lending",
      "tvl": 25000000,
      "oracles": ["Switchboard", "Pyth"]
    },
    {
      "id": "3",
      "name": "Scallop Lend",
      "slug": "scallop-lend",
      "chain": "Sui",
      "category": "Lending",
      "tvl": 35000000,
      "oracles": ["Switchboard"]
    }
  ],
  "chains": ["Solana", "Sui"]
}
```

### 11.5 Benchmark Tests

```go
// internal/aggregator/metrics_benchmark_test.go

package aggregator

import (
    "testing"

    "github.com/yourorg/switchboard-extractor/internal/models"
)

func BenchmarkCalculateChainBreakdown(b *testing.B) {
    // Create large dataset
    oracleTVS := make(map[string]map[string]float64)
    for i := 0; i < 100; i++ {
        oracleTVS[fmt.Sprintf("Protocol%d", i)] = map[string]float64{
            "Solana": float64(i * 1000000),
            "Sui":    float64(i * 500000),
        }
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        CalculateChainBreakdown(oracleTVS)
    }
}

func BenchmarkFindSnapshotAtTime(b *testing.B) {
    // Create 90 days of snapshots
    snapshots := make([]models.Snapshot, 2160)
    now := time.Now().Unix()
    for i := 0; i < 2160; i++ {
        snapshots[i] = models.Snapshot{
            Timestamp: now - int64(i*3600),
            TVS:       float64(100000000 + i*10000),
        }
    }

    targetTime := now - 7*24*3600 // 7 days ago

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        FindSnapshotAtTime(snapshots, targetTime, 7200)
    }
}
```

---

## 12. Deployment Considerations

### 12.1 Docker Configuration

```dockerfile
# Dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o extractor ./cmd/extractor

# Runtime image
FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/extractor .
COPY configs/config.yaml ./config.yaml

# Create non-root user
RUN adduser -D -g '' extractor
USER extractor

VOLUME /app/data

EXPOSE 9090

ENTRYPOINT ["./extractor"]
CMD ["--config", "config.yaml"]
```

### 12.2 Docker Compose

```yaml
# docker-compose.yml
version: '3.8'

services:
  extractor:
    build: .
    container_name: switchboard-extractor
    restart: unless-stopped
    volumes:
      - ./data:/app/data
      - ./config.yaml:/app/config.yaml:ro
    environment:
      - LOG_LEVEL=info
      - TZ=UTC
    ports:
      - "9090:9090"
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:9090/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
```

---

## 13. API Response Examples

(Same as original document - section retained for completeness)

---

## 14. Implementation Checklist

### 14.1 Phase 1: Foundation
- [ ] Set up Go module and directory structure
- [ ] Implement configuration loading (YAML + env vars)
- [ ] Create all data models including custom error types
- [ ] Implement HTTP client with retry logic
- [ ] Write unit tests for HTTP client

### 14.2 Phase 2: API Integration
- [ ] Implement oracle API fetcher
- [ ] Implement protocol API fetcher
- [ ] Add parallel fetching capability
- [ ] Handle all API error cases
- [ ] Write integration tests with mock server

### 14.3 Phase 3: Aggregation Logic
- [ ] Implement protocol filtering by oracle
- [ ] Implement TVS aggregation by chain
- [ ] Implement TVS aggregation by category
- [ ] Implement protocol ranking
- [ ] Implement metric calculations (changes, growth)
- [ ] Write unit tests for all calculations (table-driven)
- [ ] Write benchmark tests

### 14.4 Phase 4: Storage & State
- [ ] Implement state manager (load/save)
- [ ] Implement update detection logic
- [ ] Implement history manager (append/prune)
- [ ] Implement atomic file writer
- [ ] Implement all output formats
- [ ] Write unit tests for storage components

### 14.5 Phase 5: Orchestration
- [ ] Implement main extraction pipeline
- [ ] Add scheduler for periodic updates
- [ ] Implement CLI argument parsing
- [ ] Add structured logging (slog)
- [ ] Implement graceful shutdown
- [ ] Add health check endpoint

### 14.6 Phase 6: Production Readiness
- [ ] Write full integration tests
- [ ] Add Docker support
- [ ] Create systemd service file
- [ ] Implement Prometheus metrics
- [ ] Add alerting integration
- [ ] Performance optimization
- [ ] Security review
- [ ] Documentation

---

## 15. Go-Specific Patterns & Idioms

### 15.1 Structured Logging with slog (Go 1.24+)

```go
// internal/logging/logger.go

package logging

import (
    "log/slog"
    "os"
)

// NewLogger creates a structured logger based on configuration
func NewLogger(level, format, output string) *slog.Logger {
    var handler slog.Handler

    opts := &slog.HandlerOptions{
        Level: parseLevel(level),
        AddSource: level == "debug",
    }

    var w *os.File
    switch output {
    case "stderr":
        w = os.Stderr
    default:
        w = os.Stdout
    }

    switch format {
    case "json":
        handler = slog.NewJSONHandler(w, opts)
    default:
        handler = slog.NewTextHandler(w, opts)
    }

    return slog.New(handler)
}

func parseLevel(level string) slog.Level {
    switch level {
    case "debug":
        return slog.LevelDebug
    case "warn":
        return slog.LevelWarn
    case "error":
        return slog.LevelError
    default:
        return slog.LevelInfo
    }
}

// Usage examples:
// logger.Info("extraction started", slog.String("oracle", "Switchboard"))
// logger.Debug("fetching API", slog.String("url", url), slog.Int("attempt", 1))
// logger.Error("fetch failed", slog.String("error", err.Error()))
// logger.With(slog.String("component", "aggregator")).Info("processing")
```

### 15.2 Context Propagation

```go
// Always pass context through the entire call chain

func (e *Extractor) Run(ctx context.Context) error {
    // Create child context with timeout for API calls
    apiCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
    defer cancel()

    result, err := e.aggregator.Process(apiCtx, e.history)
    if err != nil {
        return err
    }

    // Check for cancellation before expensive operations
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    return e.writer.WriteAll(ctx, result)
}

// In HTTP client
func (c *Client) doRequest(ctx context.Context, url string, target interface{}) error {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    // ... context cancellation is automatically handled
}
```

### 15.3 Dependency Injection Pattern

```go
// Define interfaces for all dependencies
type APIClient interface {
    FetchOracles(ctx context.Context) (*models.OracleAPIResponse, error)
    FetchProtocols(ctx context.Context) (*models.ProtocolAPIResponse, error)
    FetchBoth(ctx context.Context) (*models.OracleAPIResponse, *models.ProtocolAPIResponse, error)
}

type StateManager interface {
    Load() (*models.State, error)
    Save(state *models.State) error
    ShouldUpdate(state *models.State, latestTimestamp int64) bool
}

type OutputWriter interface {
    WriteAll(output *models.FullOutput) error
}

// Extractor uses interfaces, not concrete types
type Extractor struct {
    client     APIClient      // Interface
    state      StateManager   // Interface
    writer     OutputWriter   // Interface
    aggregator *Aggregator
    logger     *slog.Logger
}

// Constructor accepts interfaces for easy testing
func NewExtractor(
    client APIClient,
    state StateManager,
    writer OutputWriter,
    oracleName string,
    logger *slog.Logger,
) *Extractor {
    return &Extractor{
        client:     client,
        state:      state,
        writer:     writer,
        aggregator: NewAggregator(client, oracleName, logger),
        logger:     logger,
    }
}
```

---

## 16. Operational Concerns

### 16.1 Graceful Shutdown

```go
// cmd/extractor/main.go (signal handling portion)

func main() {
    // ... setup code ...

    // Create context that cancels on interrupt
    ctx, cancel := context.WithCancel(context.Background())

    // Handle shutdown signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        sig := <-sigChan
        logger.Info("received shutdown signal", slog.String("signal", sig.String()))
        cancel()
    }()

    // Run extractor
    if err := extractor.Run(ctx); err != nil {
        if errors.Is(err, context.Canceled) {
            logger.Info("extraction cancelled by signal")
        } else {
            logger.Error("extraction failed", slog.String("error", err.Error()))
            os.Exit(1)
        }
    }

    // Cleanup
    logger.Info("shutdown complete")
}
```

### 16.2 Prometheus Metrics

```go
// internal/monitoring/metrics.go

package monitoring

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    ExtractionDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "switchboard_extraction_duration_seconds",
            Help:    "Duration of extraction cycles",
            Buckets: []float64{1, 5, 10, 30, 60, 120},
        },
        []string{"status"},
    )

    ProtocolCount = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "switchboard_protocol_count",
            Help: "Number of protocols using Switchboard oracle",
        },
    )

    TotalValueSecured = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "switchboard_total_value_secured",
            Help: "Total value secured by Switchboard oracle in USD",
        },
    )

    APIRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "switchboard_api_requests_total",
            Help: "Total API requests by endpoint and status",
        },
        []string{"endpoint", "status"},
    )

    APIRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "switchboard_api_request_duration_seconds",
            Help:    "API request duration by endpoint",
            Buckets: []float64{0.1, 0.5, 1, 2, 5, 10},
        },
        []string{"endpoint"},
    )

    LastExtractionTimestamp = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "switchboard_last_extraction_timestamp",
            Help: "Unix timestamp of last successful extraction",
        },
    )

    ExtractionErrors = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "switchboard_extraction_errors_total",
            Help: "Total extraction errors by type",
        },
        []string{"error_type"},
    )
)

// RecordExtraction updates metrics after an extraction cycle
func RecordExtraction(duration float64, protocolCount int, tvs float64, success bool) {
    status := "success"
    if !success {
        status = "failure"
    }
    ExtractionDuration.WithLabelValues(status).Observe(duration)

    if success {
        ProtocolCount.Set(float64(protocolCount))
        TotalValueSecured.Set(tvs)
        LastExtractionTimestamp.SetToCurrentTime()
    }
}
```

### 16.3 Health Check Endpoint

```go
// internal/monitoring/health.go

package monitoring

import (
    "encoding/json"
    "net/http"
    "sync"
    "time"
)

type HealthChecker struct {
    mu              sync.RWMutex
    lastSuccess     time.Time
    lastError       error
    consecutiveFails int
}

type HealthStatus struct {
    Status           string    `json:"status"`
    LastSuccess      time.Time `json:"last_success,omitempty"`
    LastError        string    `json:"last_error,omitempty"`
    ConsecutiveFails int       `json:"consecutive_fails"`
    Uptime           string    `json:"uptime"`
}

func NewHealthChecker() *HealthChecker {
    return &HealthChecker{}
}

func (h *HealthChecker) RecordSuccess() {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.lastSuccess = time.Now()
    h.lastError = nil
    h.consecutiveFails = 0
}

func (h *HealthChecker) RecordFailure(err error) {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.lastError = err
    h.consecutiveFails++
}

func (h *HealthChecker) Handler(startTime time.Time) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        h.mu.RLock()
        defer h.mu.RUnlock()

        status := HealthStatus{
            Status:           "healthy",
            LastSuccess:      h.lastSuccess,
            ConsecutiveFails: h.consecutiveFails,
            Uptime:           time.Since(startTime).String(),
        }

        // Unhealthy if more than 3 consecutive failures
        // or no success in last 30 minutes
        if h.consecutiveFails > 3 {
            status.Status = "unhealthy"
        }
        if !h.lastSuccess.IsZero() && time.Since(h.lastSuccess) > 30*time.Minute {
            status.Status = "unhealthy"
        }

        if h.lastError != nil {
            status.LastError = h.lastError.Error()
        }

        w.Header().Set("Content-Type", "application/json")
        if status.Status != "healthy" {
            w.WriteHeader(http.StatusServiceUnavailable)
        }
        json.NewEncoder(w).Encode(status)
    }
}
```

### 16.4 API Schema Change Detection

```go
// internal/api/validation.go

package api

import (
    "fmt"

    "github.com/yourorg/switchboard-extractor/internal/models"
)

// ValidateOracleResponse checks if the API response has expected structure
func ValidateOracleResponse(resp *models.OracleAPIResponse) error {
    if resp.Oracles == nil {
        return fmt.Errorf("missing 'oracles' field in response")
    }
    if resp.Chart == nil {
        return fmt.Errorf("missing 'chart' field in response")
    }
    if resp.ChainsByOracle == nil {
        return fmt.Errorf("missing 'chainsByOracle' field in response")
    }

    // Check for empty data (possible API issue)
    if len(resp.Oracles) == 0 {
        return fmt.Errorf("'oracles' field is empty - possible API issue")
    }
    if len(resp.Chart) == 0 {
        return fmt.Errorf("'chart' field is empty - possible API issue")
    }

    return nil
}

// ValidateProtocolResponse checks protocol response structure
func ValidateProtocolResponse(resp *models.ProtocolAPIResponse) error {
    if resp.Protocols == nil {
        return fmt.Errorf("missing 'protocols' field in response")
    }

    // Validate at least some protocols have expected fields
    for i, p := range resp.Protocols {
        if i > 10 {
            break // Sample check only
        }
        if p.Name == "" {
            return fmt.Errorf("protocol at index %d has empty name", i)
        }
        if p.Slug == "" {
            return fmt.Errorf("protocol %s has empty slug", p.Name)
        }
    }

    return nil
}
```

---

## 17. Complete Main.go Implementation

```go
// cmd/extractor/main.go

package main

import (
    "context"
    "errors"
    "flag"
    "fmt"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/prometheus/client_golang/prometheus/promhttp"

    "github.com/yourorg/switchboard-extractor/internal/aggregator"
    "github.com/yourorg/switchboard-extractor/internal/api"
    "github.com/yourorg/switchboard-extractor/internal/config"
    "github.com/yourorg/switchboard-extractor/internal/logging"
    "github.com/yourorg/switchboard-extractor/internal/models"
    "github.com/yourorg/switchboard-extractor/internal/monitoring"
    "github.com/yourorg/switchboard-extractor/internal/storage"
)

var (
    Version   = "dev"
    BuildTime = "unknown"
)

func main() {
    // Parse command line flags
    configPath := flag.String("config", "config.yaml", "Path to config file")
    runOnce := flag.Bool("once", false, "Run once and exit")
    dryRun := flag.Bool("dry-run", false, "Fetch data but don't write files")
    showVersion := flag.Bool("version", false, "Print version and exit")
    flag.Parse()

    if *showVersion {
        fmt.Printf("switchboard-extractor %s (built %s)\n", Version, BuildTime)
        os.Exit(0)
    }

    // Load configuration
    cfg, err := config.Load(*configPath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
        os.Exit(1)
    }

    // Initialize logger
    logger := logging.NewLogger(cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.Output)
    logger.Info("starting extractor",
        slog.String("version", Version),
        slog.String("oracle", cfg.Oracle.Name),
    )

    // Create context with cancellation
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle shutdown signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        sig := <-sigChan
        logger.Info("received shutdown signal", slog.String("signal", sig.String()))
        cancel()
    }()

    // Initialize components
    apiClient := api.NewClient(api.ClientConfig{
        OracleURL:      cfg.API.OracleURL,
        ProtocolsURL:   cfg.API.ProtocolsURL,
        Timeout:        cfg.API.Timeout,
        MaxRetries:     cfg.API.MaxRetries,
        RetryBaseDelay: cfg.API.RetryBaseDelay,
        RetryMaxDelay:  cfg.API.RetryMaxDelay,
        UserAgent:      cfg.API.UserAgent,
    }, logger)

    stateManager := storage.NewStateManager(cfg.Output.Directory)
    historyManager := storage.NewHistoryManager(cfg.History.RetentionDays)

    writer, err := storage.NewWriter(cfg.Output.Directory)
    if err != nil {
        logger.Error("failed to create writer", slog.String("error", err.Error()))
        os.Exit(1)
    }

    agg := aggregator.NewAggregator(apiClient, cfg.Oracle.Name, logger)
    healthChecker := monitoring.NewHealthChecker()
    startTime := time.Now()

    // Start monitoring server
    if cfg.Monitoring.Enabled {
        go startMonitoringServer(cfg, healthChecker, startTime, logger)
    }

    // Create extraction function
    extract := func() error {
        return runExtraction(ctx, cfg, agg, stateManager, historyManager, writer, healthChecker, logger, *dryRun)
    }

    // Run extraction
    if *runOnce || !cfg.Scheduler.Enabled {
        if err := extract(); err != nil {
            if !errors.Is(err, context.Canceled) {
                logger.Error("extraction failed", slog.String("error", err.Error()))
                os.Exit(1)
            }
        }
    } else {
        // Run scheduler
        runScheduler(ctx, cfg.Scheduler.Interval, cfg.Scheduler.StartImmediately, extract, logger)
    }

    logger.Info("shutdown complete")
}

func runExtraction(
    ctx context.Context,
    cfg *config.Config,
    agg *aggregator.Aggregator,
    stateManager *storage.StateManager,
    historyManager *storage.HistoryManager,
    writer *storage.Writer,
    healthChecker *monitoring.HealthChecker,
    logger *slog.Logger,
    dryRun bool,
) error {
    startTime := time.Now()
    logger.Info("starting extraction cycle")

    // Load state
    state, err := stateManager.Load()
    if err != nil {
        if errors.Is(err, models.ErrStateCorrupted) {
            logger.Warn("state file corrupted, starting fresh")
            state = &models.State{}
        } else {
            return fmt.Errorf("loading state: %w", err)
        }
    }

    // Load history
    outputPath := fmt.Sprintf("%s/%s", cfg.Output.Directory, cfg.Output.FullFile)
    history, err := historyManager.LoadFromOutput(outputPath)
    if err != nil {
        logger.Warn("failed to load history, starting fresh", slog.String("error", err.Error()))
        history = []models.Snapshot{}
    }

    // Run aggregation
    result, err := agg.Process(ctx, history)
    if err != nil {
        healthChecker.RecordFailure(err)
        monitoring.ExtractionErrors.WithLabelValues(errorType(err)).Inc()
        return fmt.Errorf("aggregation failed: %w", err)
    }

    // Check if we should update
    if !stateManager.ShouldUpdate(state, result.LatestTimestamp) {
        logger.Info("no new data available, skipping")
        monitoring.RecordExtraction(time.Since(startTime).Seconds(), 0, 0, true)
        return nil
    }

    // Update history
    history = historyManager.Append(history, result.Snapshot)
    history = historyManager.Prune(history)
    history = historyManager.Deduplicate(history)

    // Build output
    output := buildOutput(cfg, result, history)

    // Write files (unless dry run)
    if !dryRun {
        if err := writer.WriteAll(output); err != nil {
            healthChecker.RecordFailure(err)
            return fmt.Errorf("writing output: %w", err)
        }

        // Update state
        newState := stateManager.UpdateState(
            state,
            cfg.Oracle.Name,
            result.LatestTimestamp,
            len(result.Protocols),
            result.Metrics.CurrentTVS,
            history,
        )
        if err := stateManager.Save(newState); err != nil {
            return fmt.Errorf("saving state: %w", err)
        }
    }

    // Record success
    duration := time.Since(startTime).Seconds()
    healthChecker.RecordSuccess()
    monitoring.RecordExtraction(duration, len(result.Protocols), result.Metrics.CurrentTVS, true)

    logger.Info("extraction complete",
        slog.Duration("duration", time.Since(startTime)),
        slog.Int("protocols", len(result.Protocols)),
        slog.Float64("tvs", result.Metrics.CurrentTVS),
    )

    return nil
}

func buildOutput(cfg *config.Config, result *aggregator.Result, history []models.Snapshot) *models.FullOutput {
    return &models.FullOutput{
        Version: "1.0.0",
        Oracle: models.OracleInfo{
            Name:          cfg.Oracle.Name,
            Website:       cfg.Oracle.Website,
            Documentation: cfg.Oracle.Documentation,
        },
        Metadata: models.OutputMetadata{
            LastUpdated:      time.Now().UTC().Format(time.RFC3339),
            DataSource:       "DefiLlama API",
            UpdateFrequency:  cfg.Scheduler.Interval.String(),
            ExtractorVersion: Version,
        },
        Summary: models.Summary{
            TotalValueSecured: result.Metrics.CurrentTVS,
            TotalProtocols:    result.Metrics.ProtocolCount,
            ActiveChains:      result.Metrics.ChainCount,
            Categories:        result.Metrics.Categories,
        },
        Metrics:    result.Metrics,
        Breakdown: models.Breakdown{
            ByChain:    result.ChainBreakdown,
            ByCategory: result.CategoryBreakdown,
        },
        Protocols:  result.Protocols,
        Historical: history,
    }
}

func runScheduler(ctx context.Context, interval time.Duration, startImmediately bool, fn func() error, logger *slog.Logger) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    if startImmediately {
        if err := fn(); err != nil && !errors.Is(err, context.Canceled) {
            logger.Error("extraction failed", slog.String("error", err.Error()))
        }
    }

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := fn(); err != nil && !errors.Is(err, context.Canceled) {
                logger.Error("extraction failed", slog.String("error", err.Error()))
            }
        }
    }
}

func startMonitoringServer(cfg *config.Config, healthChecker *monitoring.HealthChecker, startTime time.Time, logger *slog.Logger) {
    mux := http.NewServeMux()
    mux.Handle(cfg.Monitoring.Path, promhttp.Handler())
    mux.HandleFunc("/health", healthChecker.Handler(startTime))

    addr := fmt.Sprintf(":%d", cfg.Monitoring.Port)
    logger.Info("starting monitoring server", slog.String("addr", addr))

    if err := http.ListenAndServe(addr, mux); err != nil && !errors.Is(err, http.ErrServerClosed) {
        logger.Error("monitoring server failed", slog.String("error", err.Error()))
    }
}

func errorType(err error) string {
    var apiErr *models.APIError
    if errors.As(err, &apiErr) {
        return "api_error"
    }
    if errors.Is(err, models.ErrOracleNotFound) {
        return "oracle_not_found"
    }
    if errors.Is(err, models.ErrInvalidResponse) {
        return "invalid_response"
    }
    return "unknown"
}
```

---

## Appendix A: Go Dependencies

```go
// go.mod
module github.com/yourorg/switchboard-oracle-extractor

go 1.24

require (
    gopkg.in/yaml.v3 v3.0.1
    github.com/prometheus/client_golang v1.20.0
)
```

---

## Appendix B: Quick Reference

### B.1 API Endpoints

| Endpoint | Purpose |
|----------|---------|
| `GET /oracles` | Oracle TVS data, charts, protocol lists |
| `GET /lite/protocols2?b=2` | Protocol metadata |

### B.2 Key Constants

| Constant | Value |
|----------|-------|
| Oracle Name | `"Switchboard"` |
| API Base URL | `https://api.llama.fi` |
| Update Interval | 15 minutes |
| History Retention | 90 days |
| HTTP Timeout | 30 seconds |
| Max Retries | 3 |

### B.3 Output Files

| File | Content |
|------|---------|
| `switchboard-oracle-data.json` | Full data with history |
| `switchboard-oracle-data.min.json` | Minified version |
| `switchboard-summary.json` | Current snapshot only |
| `state.json` | Incremental update state |

---

**Document Version:** 1.2.0 (Revised)
**Last Updated:** 2025-11-29
**Author:** Claude Code Assistant

### Changelog
- **v1.2.0**: Updated to latest stable versions - Go 1.24, Alpine 3.22, prometheus/client_golang v1.20.0
- **v1.1.0**: Added complete Go implementations for aggregation, enhanced testing section, added operational concerns, complete main.go, graceful shutdown, Prometheus metrics, health checks, API validation
- **v1.0.0**: Initial specification
