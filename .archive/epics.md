# defillama-extract - Epic Breakdown

**Author:** BMad
**Date:** 2025-11-29
**Project Level:** MVP
**Target Scale:** Internal Tooling

---

## Overview

This document provides the complete epic and story breakdown for defillama-extract, decomposing the requirements from the [PRD](./prd.md) into implementable stories.

**Living Document Notice:** This is the initial version created from PRD + Architecture context.

---

## Functional Requirements Inventory

| FR | Description | Category |
|----|-------------|----------|
| FR1 | System fetches oracle data from DefiLlama `GET /oracles` endpoint | API Integration |
| FR2 | System fetches protocol metadata from DefiLlama `GET /lite/protocols2?b=2` endpoint | API Integration |
| FR3 | System fetches both endpoints in parallel to minimize total fetch time | API Integration |
| FR4 | System includes proper User-Agent header identifying the extractor | API Integration |
| FR5 | System retries failed API requests with exponential backoff and jitter | API Integration |
| FR6 | System respects configurable timeout for API requests (default 30s) | API Integration |
| FR7 | System handles API errors gracefully (429, 5xx) with appropriate retry logic | API Integration |
| FR8 | System detects and reports non-retryable errors (4xx client errors) | API Integration |
| FR9 | System filters protocols by exact oracle name match ("Switchboard") | Data Filtering |
| FR10 | System checks both `oracles` array and legacy `oracle` field for matching | Data Filtering |
| FR11 | System extracts TVS (Total Value Secured) data per protocol per chain | Data Filtering |
| FR12 | System extracts protocol metadata (name, slug, category, TVL, chains, URL) | Data Filtering |
| FR13 | System identifies all chains where the target oracle is used | Data Filtering |
| FR14 | System extracts timestamp of latest data point from chart data | Data Filtering |
| FR15 | System calculates total TVS across all protocols using the oracle | Aggregation |
| FR16 | System calculates TVS breakdown by chain with percentage of total | Aggregation |
| FR17 | System calculates TVS breakdown by protocol category with percentage of total | Aggregation |
| FR18 | System ranks protocols by TVL in descending order | Aggregation |
| FR19 | System calculates 24-hour TVS change percentage (when historical data available) | Aggregation |
| FR20 | System calculates 7-day TVS change percentage (when historical data available) | Aggregation |
| FR21 | System calculates 30-day TVS change percentage (when historical data available) | Aggregation |
| FR22 | System calculates protocol count growth over 7-day and 30-day periods | Aggregation |
| FR23 | System identifies largest protocol by TVL | Aggregation |
| FR24 | System extracts unique categories across all filtered protocols | Aggregation |
| FR25 | System tracks last successfully processed timestamp in state file | Incremental Updates |
| FR26 | System compares latest API timestamp against last processed timestamp | Incremental Updates |
| FR27 | System skips processing when no new data is available | Incremental Updates |
| FR28 | System recovers gracefully from corrupted state file (starts fresh) | Incremental Updates |
| FR29 | System updates state file atomically after successful extraction | Incremental Updates |
| FR30 | System maintains historical snapshots of TVS data over time | Historical Data |
| FR31 | System stores timestamp, date, TVS, TVS by chain, protocol count, and chain count per snapshot | Historical Data |
| FR32 | System deduplicates snapshots with identical timestamps | Historical Data |
| FR33 | System retains all historical snapshots (no automatic pruning) | Historical Data |
| FR34 | System loads existing history from output file on startup | Historical Data |
| FR35 | System generates full output JSON with all data and complete history | Output Generation |
| FR36 | System generates minified output JSON (same data, no whitespace) | Output Generation |
| FR37 | System generates summary output JSON with current snapshot only (no history) | Output Generation |
| FR38 | System writes all output files atomically (temp file + rename) | Output Generation |
| FR39 | System creates output directory if it doesn't exist | Output Generation |
| FR40 | System includes version, oracle info, and metadata in all outputs | Output Generation |
| FR41 | System includes extraction timestamp and data source attribution in metadata | Output Generation |
| FR42 | System runs in single-extraction mode with `--once` flag | CLI Operation |
| FR43 | System runs in daemon mode with configurable interval (default 2 hours) | CLI Operation |
| FR44 | System accepts configuration file path via `--config` flag | CLI Operation |
| FR45 | System supports dry-run mode that fetches but doesn't write files | CLI Operation |
| FR46 | System prints version information with `--version` flag | CLI Operation |
| FR47 | System shuts down gracefully on SIGINT and SIGTERM signals | CLI Operation |
| FR48 | System logs extraction start, completion, duration, and key metrics | CLI Operation |
| FR49 | System loads configuration from YAML file | Configuration |
| FR50 | System applies environment variable overrides to configuration | Configuration |
| FR51 | System provides sensible defaults for all configuration values | Configuration |
| FR52 | System validates configuration on startup | Configuration |
| FR53 | System logs in structured format (JSON or text, configurable) | Logging |
| FR54 | System supports configurable log levels (debug, info, warn, error) | Logging |
| FR55 | System logs API request attempts, retries, and failures | Logging |
| FR56 | System logs extraction cycle results with protocol count and TVS | Logging |

---

## FR Coverage Map

| Epic | FRs Covered | Description |
|------|-------------|-------------|
| Epic 1: Foundation | FR49-FR52, FR53-FR54 | Project setup, configuration, logging infrastructure |
| Epic 2: API Integration | FR1-FR8, FR55 | DefiLlama API client with retry logic |
| Epic 3: Data Processing Pipeline | FR9-FR24 | Filtering, aggregation, metrics calculation |
| Epic 4: State & History Management | FR25-FR34 | Incremental updates, historical snapshots |
| Epic 5: Output & CLI | FR35-FR48, FR56 | JSON output, CLI modes, graceful shutdown |

---

## Epic 1: Foundation

**Goal:** Establish project structure with working configuration and logging infrastructure so developers can start building features on a solid base.

**User Value:** After this epic, the project compiles, loads configuration from YAML/env vars, and outputs structured logs - the foundation for all subsequent development.

**FRs Covered:** FR49, FR50, FR51, FR52, FR53, FR54

---

### Story 1.1: Initialize Go Module and Project Structure

As a **developer**,
I want **a properly initialized Go module with the standard project layout**,
So that **I have a clean foundation to build the extraction service**.

**Acceptance Criteria:**

**Given** a new project directory
**When** I run `go mod init` and create the directory structure
**Then** the following structure exists:
- `cmd/extractor/main.go` (minimal entry point that compiles)
- `internal/config/` directory
- `internal/models/` directory
- `internal/api/` directory
- `internal/aggregator/` directory
- `internal/storage/` directory
- `go.mod` with module `github.com/switchboard-xyz/defillama-extract`
- `Makefile` with `build`, `test`, `lint` targets
- `.gitignore` excluding `data/`, `*.exe`, vendor/
- `.golangci.yml` with standard linter config

**And** running `go build ./...` succeeds with no errors

**Prerequisites:** None (first story)

**Technical Notes:**
- Follow Go standard project layout per Architecture doc
- Use Go 1.21+ for slog support
- Entry point at `cmd/extractor/main.go` per project-structure.md

---

### Story 1.2: Implement Configuration Loading from YAML

As a **developer**,
I want **configuration loaded from a YAML file with sensible defaults**,
So that **the service behavior can be customized without code changes**.

**Acceptance Criteria:**

**Given** a YAML configuration file at `configs/config.yaml`
**When** the application starts
**Then** configuration is loaded into a typed `Config` struct with sections:
- `oracle`: name (default: "Switchboard"), website, documentation URL
- `api`: base URLs, timeout (default: 30s), max retries (default: 3), retry delay
- `output`: directory (default: "data"), file names
- `scheduler`: interval (default: 2h), start_immediately flag
- `logging`: level (default: "info"), format (default: "json")

**And** missing optional fields use default values
**And** missing required fields cause startup failure with clear error message
**And** invalid values (e.g., negative timeout) cause validation errors

**Given** no config file exists at specified path
**When** the application starts with `--config nonexistent.yaml`
**Then** application exits with error code 1 and logs "config file not found"

**Prerequisites:** Story 1.1

**Technical Notes:**
- Package: `internal/config/config.go`
- Use `gopkg.in/yaml.v3` for YAML parsing
- Config struct fields use `yaml` tags
- Implement `Load(path string) (*Config, error)` function
- Implement `Validate() error` method on Config
- Reference: architecture/fr-category-to-architecture-mapping.md → FR49-FR52

---

### Story 1.3: Implement Environment Variable Overrides

As an **operator**,
I want **environment variables to override YAML configuration**,
So that **I can customize behavior in different environments without modifying files**.

**Acceptance Criteria:**

**Given** a loaded YAML configuration
**When** environment variables are set
**Then** the following overrides are applied (env var → config field):
- `ORACLE_NAME` → `oracle.name`
- `OUTPUT_DIR` → `output.directory`
- `LOG_LEVEL` → `logging.level`
- `LOG_FORMAT` → `logging.format`
- `API_TIMEOUT` → `api.timeout` (parsed as duration, e.g., "45s")
- `SCHEDULER_INTERVAL` → `scheduler.interval`

**And** environment variables take precedence over YAML values
**And** invalid environment variable values log a warning and use YAML/default value
**And** unset environment variables do not affect configuration

**Prerequisites:** Story 1.2

**Technical Notes:**
- Extend `config.Load()` to call `applyEnvOverrides()`
- Use `os.Getenv()` for reading env vars
- Parse duration strings with `time.ParseDuration()`
- Log applied overrides at debug level

---

### Story 1.4: Implement Structured Logging with slog

As a **developer**,
I want **structured logging using Go's slog package**,
So that **logs are machine-parseable and include consistent contextual information**.

**Acceptance Criteria:**

**Given** logging configuration with `format: "json"` and `level: "info"`
**When** the logger is initialized
**Then** a JSON handler is configured writing to stdout
**And** log entries include: timestamp, level, message, and any additional attributes

**Given** logging configuration with `format: "text"` and `level: "debug"`
**When** the logger is initialized
**Then** a text handler is configured for human-readable output
**And** debug-level messages are included in output

**Given** an initialized logger
**When** code logs with `slog.Info("message", "key", "value")`
**Then** output includes the key-value pair as structured data

**And** the following log levels are supported: debug, info, warn, error
**And** messages below configured level are suppressed

**Prerequisites:** Story 1.2

**Technical Notes:**
- Create `internal/logging/logging.go` or initialize in main.go
- Use `slog.New()` with `slog.NewJSONHandler()` or `slog.NewTextHandler()`
- Set as default logger with `slog.SetDefault()`
- Map config level string to `slog.Level` constant
- Reference: 15-go-specific-patterns-idioms.md section 15.1

---

## Epic 2: API Integration

**Goal:** Build a robust HTTP client that fetches data from DefiLlama APIs with proper error handling, retries, and parallel execution.

**User Value:** After this epic, the system can reliably fetch real oracle and protocol data from DefiLlama, handling transient failures automatically.

**FRs Covered:** FR1, FR2, FR3, FR4, FR5, FR6, FR7, FR8, FR55

---

### Story 2.1: Implement Base HTTP Client with Timeout and User-Agent

As a **developer**,
I want **a configured HTTP client with proper timeout and identification headers**,
So that **API requests are well-behaved and identifiable**.

**Acceptance Criteria:**

**Given** API configuration with `timeout: 30s`
**When** the HTTP client is initialized
**Then** all requests use a 30-second timeout
**And** requests include header `User-Agent: defillama-extract/1.0`

**Given** an API request in progress
**When** the timeout duration elapses without response
**Then** the request is cancelled and returns a timeout error
**And** the error message indicates timeout occurred

**Given** a request to any DefiLlama endpoint
**When** the request is sent
**Then** the User-Agent header is present in the request

**Prerequisites:** Story 1.2 (config loading)

**Technical Notes:**
- Package: `internal/api/client.go`
- Create `Client` struct with `*http.Client` and config
- Constructor: `NewClient(cfg *config.APIConfig) *Client`
- Set `http.Client.Timeout` from config
- Create custom `RoundTripper` or add header in request helper
- Reference: 5-core-components.md section 5.1

---

### Story 2.2: Implement Oracle Endpoint Fetcher

As a **developer**,
I want **to fetch oracle data from the `/oracles` endpoint**,
So that **I can retrieve TVS data and protocol-to-oracle mappings**.

**Acceptance Criteria:**

**Given** a configured API client
**When** `FetchOracles(ctx context.Context)` is called
**Then** a GET request is made to `https://api.llama.fi/oracles`
**And** the response is parsed into `OracleAPIResponse` struct containing:
  - `Oracles`: map of oracle name → protocol slugs
  - `Chart`: historical TVS data by oracle/chain/timestamp
  - `OraclesTVS`: current TVS by oracle/chain
  - `ChainsByOracle`: chains where each oracle operates

**Given** a successful API response
**When** parsing completes
**Then** the function returns `(*OracleAPIResponse, error)` with nil error

**Given** an HTTP error (network failure, non-2xx status)
**When** the request fails
**Then** the function returns nil response and descriptive error

**Prerequisites:** Story 2.1

**Technical Notes:**
- Method: `func (c *Client) FetchOracles(ctx context.Context) (*models.OracleAPIResponse, error)`
- Use `http.NewRequestWithContext()` for cancellation support
- Response struct in `internal/models/api.go`
- Decode with `json.NewDecoder(resp.Body).Decode()`
- Reference: 3-data-sources-api-specifications.md, data-architecture.md

---

### Story 2.3: Implement Protocol Endpoint Fetcher

As a **developer**,
I want **to fetch protocol metadata from the `/lite/protocols2` endpoint**,
So that **I can retrieve protocol details like name, category, TVL, and chains**.

**Acceptance Criteria:**

**Given** a configured API client
**When** `FetchProtocols(ctx context.Context)` is called
**Then** a GET request is made to `https://api.llama.fi/lite/protocols2?b=2`
**And** the response is parsed into a slice of `Protocol` structs containing:
  - `ID`, `Name`, `Slug`: protocol identifiers
  - `Category`: protocol type (Lending, CDP, etc.)
  - `TVL`: total value locked
  - `Chains`: list of chains where protocol operates
  - `Oracles`: list of oracles used (array field)
  - `Oracle`: legacy single oracle field (string)
  - `URL`: protocol website

**Given** a successful API response
**When** parsing completes
**Then** the function returns `([]Protocol, error)` with nil error

**Given** protocols with missing optional fields (TVL, Chains, URL)
**When** parsing completes
**Then** those fields are zero-valued (0, nil, "") without error

**Prerequisites:** Story 2.1

**Technical Notes:**
- Method: `func (c *Client) FetchProtocols(ctx context.Context) ([]models.Protocol, error)`
- Protocol struct in `internal/models/protocol.go`
- Use `omitempty` JSON tags for optional fields
- Reference: 3-data-sources-api-specifications.md, data-architecture.md

---

### Story 2.4: Implement Retry Logic with Exponential Backoff

As a **developer**,
I want **automatic retries with exponential backoff for transient failures**,
So that **temporary API issues don't cause extraction failures**.

**Acceptance Criteria:**

**Given** API configuration with `max_retries: 3` and `retry_delay: 1s`
**When** a request fails with a retryable error (timeout, 429, 5xx)
**Then** the request is retried up to 3 times
**And** delays between retries follow exponential backoff: 1s, 2s, 4s
**And** jitter of ±25% is added to prevent thundering herd

**Given** a request that fails with 429 (rate limit)
**When** retries are attempted
**Then** each retry is logged at warn level with attempt number
**And** final failure after exhausting retries is logged at error level

**Given** a request that fails with 4xx (except 429)
**When** the error is detected
**Then** no retry is attempted (client error, not transient)
**And** error is returned immediately

**Given** a request that succeeds on retry attempt 2
**When** the response is received
**Then** the successful response is returned
**And** info log indicates "request succeeded after N retries"

**Prerequisites:** Story 2.1, Story 1.4 (logging)

**Technical Notes:**
- Add `doWithRetry(ctx, fn)` helper method to Client
- Retryable: timeout, 429, 500, 502, 503, 504
- Non-retryable: 400, 401, 403, 404
- Use `time.Sleep()` with jitter: `delay * (0.75 + rand.Float64()*0.5)`
- Log with slog: `slog.Warn("retrying request", "attempt", n, "error", err)`
- Reference: 9-error-handling-resilience.md

---

### Story 2.5: Implement Parallel Fetching with errgroup

As a **developer**,
I want **oracle and protocol data fetched in parallel**,
So that **total fetch time is minimized**.

**Acceptance Criteria:**

**Given** a need to fetch both oracle and protocol data
**When** `FetchAll(ctx context.Context)` is called
**Then** both API requests are initiated concurrently
**And** the function waits for both to complete
**And** total fetch time is approximately max(oracle_time, protocol_time), not sum

**Given** both requests succeed
**When** `FetchAll` returns
**Then** both responses are returned in a combined struct
**And** no error is returned

**Given** the oracle request fails but protocol succeeds
**When** `FetchAll` returns
**Then** an error is returned describing the oracle failure
**And** the context is cancelled for the protocol request (if still in progress)

**Given** context cancellation during fetch
**When** the parent context is cancelled
**Then** both in-flight requests are cancelled
**And** the function returns context.Canceled error

**Prerequisites:** Story 2.2, Story 2.3

**Technical Notes:**
- Use `golang.org/x/sync/errgroup` for coordination
- Create `FetchResult` struct with `OracleResponse` and `Protocols` fields
- Method: `func (c *Client) FetchAll(ctx context.Context) (*FetchResult, error)`
- `g, ctx := errgroup.WithContext(ctx)` for cancellation propagation
- Reference: implementation-patterns.md "Parallel Fetching" section

---

### Story 2.6: Implement API Request Logging

As an **operator**,
I want **API requests logged with timing and outcome**,
So that **I can monitor API health and debug issues**.

**Acceptance Criteria:**

**Given** an API request is initiated
**When** the request starts
**Then** debug log is emitted: `"starting API request"` with `url`, `method` attributes

**Given** an API request completes successfully
**When** response is received
**Then** info log is emitted: `"API request completed"` with `url`, `status`, `duration_ms` attributes

**Given** an API request fails
**When** error occurs
**Then** warn log is emitted: `"API request failed"` with `url`, `error`, `duration_ms`, `attempt` attributes

**Given** retry is attempted
**When** retry starts
**Then** warn log is emitted: `"retrying API request"` with `url`, `attempt`, `max_attempts`, `backoff_ms` attributes

**Prerequisites:** Story 2.4, Story 1.4

**Technical Notes:**
- Add logging to `doWithRetry` and fetch methods
- Use `time.Since(start).Milliseconds()` for duration
- Log at appropriate levels: debug (start), info (success), warn (retry/fail)
- Reference: FR55 - "System logs API request attempts, retries, and failures"

---

## Epic 3: Data Processing Pipeline

**Goal:** Implement filtering, aggregation, and metrics calculation to transform raw API data into meaningful Switchboard oracle metrics.

**User Value:** After this epic, the system correctly identifies all Switchboard protocols and calculates TVS breakdowns, rankings, and change metrics - the core value proposition of surfacing the "truth gap."

**FRs Covered:** FR9, FR10, FR11, FR12, FR13, FR14, FR15, FR16, FR17, FR18, FR19, FR20, FR21, FR22, FR23, FR24

---

### Story 3.1: Implement Protocol Filtering by Oracle Name

As a **developer**,
I want **to filter protocols that use Switchboard as their oracle**,
So that **only relevant protocols are included in aggregations**.

**Acceptance Criteria:**

**Given** a list of protocols from the API
**When** `FilterByOracle(protocols []Protocol, oracleName string)` is called with "Switchboard"
**Then** only protocols where:
  - `oracles` array contains "Switchboard" (exact match, case-sensitive), OR
  - `oracle` field equals "Switchboard" (legacy field check)
are returned

**Given** a protocol with `oracles: ["Chainlink", "Switchboard"]`
**When** filtering for "Switchboard"
**Then** the protocol IS included (multi-oracle protocol)

**Given** a protocol with `oracle: "Switchboard"` but empty `oracles` array
**When** filtering for "Switchboard"
**Then** the protocol IS included (legacy field fallback)

**Given** a protocol with `oracles: ["Chainlink"]` and `oracle: ""`
**When** filtering for "Switchboard"
**Then** the protocol is NOT included

**Given** ~500 protocols from the API
**When** filtering for "Switchboard"
**Then** approximately 21 protocols are returned (expected count per PRD)

**Prerequisites:** Story 2.3 (protocol fetcher)

**Technical Notes:**
- Package: `internal/aggregator/filter.go`
- Function: `FilterByOracle(protocols []models.Protocol, oracleName string) []models.Protocol`
- Check both `Oracles` slice and `Oracle` string field
- Use case-sensitive exact match
- Reference: 7-custom-aggregation-logic-go-implementation.md section 7.2

---

### Story 3.2: Extract Protocol Metadata and TVS Data

As a **developer**,
I want **to extract relevant metadata and TVS for each filtered protocol**,
So that **I have the data needed for aggregation and output**.

**Acceptance Criteria:**

**Given** filtered Switchboard protocols and oracle API response
**When** `ExtractProtocolData(protocols, oracleResp, oracleName)` is called
**Then** for each protocol, an `AggregatedProtocol` struct is created with:
  - `Name`, `Slug`, `Category`, `URL` from protocol metadata
  - `TVL` from protocol metadata
  - `Chains` list from protocol metadata
  - `TVS` calculated from oracle response data

**Given** oracle response with `OraclesTVS["Switchboard"]["Solana"] = 1000000`
**When** extracting TVS for a protocol on Solana
**Then** the protocol's TVS includes the Solana contribution

**Given** a protocol operating on multiple chains
**When** extracting TVS
**Then** `TVSByChain` map contains TVS for each chain

**Given** oracle response chart data
**When** extracting timestamp
**Then** the latest timestamp from chart data is extracted (FR14)

**Prerequisites:** Story 3.1, Story 2.2

**Technical Notes:**
- Package: `internal/aggregator/aggregator.go`
- Create `AggregatedProtocol` struct in `internal/models/protocol.go`
- Cross-reference protocol chains with `OraclesTVS` data
- Extract timestamp from chart data keys (Unix timestamps as strings)
- Reference: data-architecture.md output models

---

### Story 3.3: Calculate Total TVS and Chain Breakdown

As a **developer**,
I want **to calculate total TVS and breakdown by chain**,
So that **I can show Switchboard's presence across different blockchains**.

**Acceptance Criteria:**

**Given** aggregated protocol data
**When** `CalculateChainBreakdown(protocols []AggregatedProtocol)` is called
**Then** a `ChainBreakdown` slice is returned with:
  - Each unique chain represented
  - `TVS` sum for that chain
  - `Percentage` of total TVS
  - `ProtocolCount` on that chain

**Given** protocols with TVS: Solana=$500M, Sui=$300M, Aptos=$200M
**When** calculating breakdown
**Then** total TVS = $1B
**And** Solana percentage = 50%
**And** Sui percentage = 30%
**And** Aptos percentage = 20%

**Given** chain breakdown results
**When** sorting
**Then** chains are ordered by TVS descending (highest first)

**Prerequisites:** Story 3.2

**Technical Notes:**
- Package: `internal/aggregator/metrics.go`
- `ChainBreakdown` struct: `Chain`, `TVS`, `Percentage`, `ProtocolCount`
- Use `float64` for TVS values (can be large numbers)
- Calculate percentage as `(chainTVS / totalTVS) * 100`
- Reference: FR15, FR16

---

### Story 3.4: Calculate Category Breakdown

As a **developer**,
I want **to calculate TVS breakdown by protocol category**,
So that **I can show which DeFi sectors use Switchboard most**.

**Acceptance Criteria:**

**Given** aggregated protocol data
**When** `CalculateCategoryBreakdown(protocols []AggregatedProtocol)` is called
**Then** a `CategoryBreakdown` slice is returned with:
  - Each unique category represented
  - `TVS` sum for that category
  - `Percentage` of total TVS
  - `ProtocolCount` in that category

**Given** protocols in categories: Lending (3 protocols, $600M), CDP (2 protocols, $300M), Dexes (1 protocol, $100M)
**When** calculating breakdown
**Then** Lending percentage = 60%, count = 3
**And** CDP percentage = 30%, count = 2
**And** Dexes percentage = 10%, count = 1

**Given** category breakdown results
**When** sorting
**Then** categories are ordered by TVS descending

**Given** all protocols
**When** extracting categories
**Then** unique categories list is returned (FR24)

**Prerequisites:** Story 3.2

**Technical Notes:**
- Package: `internal/aggregator/metrics.go`
- `CategoryBreakdown` struct: `Category`, `TVS`, `Percentage`, `ProtocolCount`
- Handle empty/missing category as "Uncategorized"
- Reference: FR17, FR24

---

### Story 3.5: Rank Protocols and Identify Largest

As a **developer**,
I want **protocols ranked by TVL and the largest protocol identified**,
So that **I can show protocol importance and highlight top contributors**.

**Acceptance Criteria:**

**Given** aggregated protocol data
**When** `RankProtocols(protocols []AggregatedProtocol)` is called
**Then** protocols are sorted by TVL descending
**And** each protocol is assigned a `Rank` field (1, 2, 3...)

**Given** ranked protocols
**When** identifying largest protocol
**Then** protocol with rank 1 is returned
**And** `LargestProtocol` struct contains: Name, Slug, TVL, TVS

**Given** two protocols with identical TVL
**When** ranking
**Then** alphabetical order by name is used as tiebreaker

**Prerequisites:** Story 3.2

**Technical Notes:**
- Package: `internal/aggregator/metrics.go`
- Use `sort.Slice()` with custom comparison
- Rank starts at 1 (not 0)
- Reference: FR18, FR23

---

### Story 3.6: Calculate Historical Change Metrics

As a **developer**,
I want **24h, 7d, and 30d TVS change percentages calculated**,
So that **I can show growth trends over time**.

**Acceptance Criteria:**

**Given** current TVS and historical snapshots
**When** `CalculateChangeMetrics(currentTVS float64, history []Snapshot)` is called
**Then** a `ChangeMetrics` struct is returned with:
  - `Change24h`: percentage change from 24 hours ago
  - `Change7d`: percentage change from 7 days ago
  - `Change30d`: percentage change from 30 days ago

**Given** current TVS = $1.1B and TVS 24h ago = $1.0B
**When** calculating 24h change
**Then** `Change24h` = 10.0 (representing 10% increase)

**Given** current TVS = $900M and TVS 7d ago = $1.0B
**When** calculating 7d change
**Then** `Change7d` = -10.0 (representing 10% decrease)

**Given** no historical data available for a time period
**When** calculating that period's change
**Then** the change value is `nil` or 0 with a flag indicating "no data"

**Given** history with protocol counts
**When** calculating growth
**Then** `ProtocolCountChange7d` and `ProtocolCountChange30d` are calculated (FR22)

**Prerequisites:** Story 3.3 (needs TVS calculation)

**Technical Notes:**
- Package: `internal/aggregator/metrics.go`
- Find snapshot closest to target time (24h, 7d, 30d ago)
- Use pointer types `*float64` for optional values
- Formula: `((current - previous) / previous) * 100`
- Handle division by zero (previous = 0)
- Reference: FR19, FR20, FR21, FR22

---

### Story 3.7: Build Complete Aggregation Pipeline

As a **developer**,
I want **a single function that orchestrates all data processing**,
So that **I have a clean interface for the extraction pipeline**.

**Acceptance Criteria:**

**Given** raw API responses (oracle and protocols)
**When** `Aggregate(ctx, oracleResp, protocols, history, oracleName)` is called
**Then** a complete `AggregationResult` is returned containing:
  - `TotalTVS`: sum across all protocols
  - `TotalProtocols`: count of filtered protocols
  - `ActiveChains`: list of chains with Switchboard presence
  - `Categories`: unique category list
  - `ChainBreakdown`: TVS by chain
  - `CategoryBreakdown`: TVS by category
  - `Protocols`: ranked protocol list
  - `LargestProtocol`: top protocol by TVL
  - `ChangeMetrics`: 24h/7d/30d changes
  - `Timestamp`: latest data timestamp

**Given** valid API data
**When** aggregation completes
**Then** all FRs 9-24 are satisfied by the result

**Prerequisites:** Stories 3.1-3.6

**Technical Notes:**
- Package: `internal/aggregator/aggregator.go`
- `Aggregator` struct with `NewAggregator(cfg)` constructor
- Main method: `func (a *Aggregator) Aggregate(...) (*AggregationResult, error)`
- Orchestrates: filter → extract → chain breakdown → category breakdown → rank → metrics
- Reference: fr-category-to-architecture-mapping.md

---

## Epic 4: State & History Management

**Goal:** Implement incremental update tracking and historical snapshot management to enable efficient polling and time-series metrics.

**User Value:** After this epic, the system runs incrementally (skipping when no new data), maintains historical trends automatically, and enables 24h/7d/30d change calculations.

**FRs Covered:** FR25, FR26, FR27, FR28, FR29, FR30, FR31, FR32, FR33, FR34

---

### Story 4.1: Implement State File Structure and Loading

As a **developer**,
I want **to load the last extraction state from a JSON file**,
So that **I can determine if new data is available**.

**Acceptance Criteria:**

**Given** a state file exists at `data/state.json`
**When** `LoadState(path string)` is called
**Then** a `State` struct is returned containing:
  - `OracleName`: the oracle being tracked
  - `LastUpdated`: Unix timestamp of last processed data
  - `LastUpdatedISO`: human-readable ISO 8601 timestamp
  - `LastProtocolCount`: number of protocols in last extraction
  - `LastTVS`: total TVS from last extraction

**Given** no state file exists
**When** `LoadState` is called
**Then** a zero-value `State` is returned (not an error)
**And** `LastUpdated` = 0 indicates first run

**Given** a corrupted/invalid state file
**When** `LoadState` is called
**Then** a warning is logged
**And** a zero-value `State` is returned (graceful recovery, FR28)
**And** extraction proceeds as if first run

**Prerequisites:** Story 1.1 (project structure)

**Technical Notes:**
- Package: `internal/storage/state.go`
- `State` struct in `internal/models/output.go`
- Use `os.ReadFile()` + `json.Unmarshal()`
- Handle `os.ErrNotExist` gracefully
- Reference: 6-incremental-update-strategy.md, data-architecture.md

---

### Story 4.2: Implement State Comparison for Skip Logic

As a **developer**,
I want **to compare current API timestamp against last processed timestamp**,
So that **I can skip processing when no new data is available**.

**Acceptance Criteria:**

**Given** current API timestamp = 1700000000 and state.LastUpdated = 1700000000
**When** `ShouldProcess(currentTimestamp, state)` is called
**Then** returns `false` (no new data)
**And** info log: "skipping extraction, no new data available"

**Given** current API timestamp = 1700003600 and state.LastUpdated = 1700000000
**When** `ShouldProcess` is called
**Then** returns `true` (new data available)
**And** debug log: "new data available, proceeding with extraction"

**Given** state.LastUpdated = 0 (first run)
**When** `ShouldProcess` is called with any timestamp
**Then** returns `true` (always process on first run)

**Given** current timestamp is OLDER than last processed (clock skew or API issue)
**When** `ShouldProcess` is called
**Then** returns `false`
**And** warn log: "API timestamp older than last processed, possible clock skew"

**Prerequisites:** Story 4.1

**Technical Notes:**
- Package: `internal/storage/state.go`
- Function: `func (s *StateManager) ShouldProcess(currentTS int64) bool`
- Compare Unix timestamps directly
- Reference: FR26, FR27

---

### Story 4.3: Implement Atomic State File Updates

As a **developer**,
I want **state updates written atomically**,
So that **interrupted writes don't corrupt the state file**.

**Acceptance Criteria:**

**Given** a successful extraction with new data
**When** `SaveState(state State, path string)` is called
**Then** state is written to a temp file first (`state.json.tmp`)
**And** temp file is renamed to target (`state.json`)
**And** the operation is atomic (no partial writes)

**Given** the output directory doesn't exist
**When** `SaveState` is called
**Then** the directory is created
**And** state file is written successfully

**Given** a write failure (disk full, permissions)
**When** `SaveState` fails
**Then** an error is returned with descriptive message
**And** any temp file is cleaned up
**And** original state file (if exists) is preserved

**Given** successful state save
**When** operation completes
**Then** info log: "state saved" with `timestamp`, `protocol_count`, `tvs` attributes

**Prerequisites:** Story 4.1

**Technical Notes:**
- Package: `internal/storage/state.go`
- Use `os.CreateTemp()` in same directory for atomic rename
- `os.Rename()` is atomic on POSIX systems
- Clean up temp file in defer on error
- Reference: FR29, implementation-patterns.md "Atomic File Writes"

---

### Story 4.4: Implement Historical Snapshot Structure

As a **developer**,
I want **historical snapshots stored with required fields**,
So that **I can track TVS trends over time**.

**Acceptance Criteria:**

**Given** current extraction results
**When** `CreateSnapshot(result *AggregationResult)` is called
**Then** a `Snapshot` struct is created with:
  - `Timestamp`: Unix timestamp
  - `Date`: ISO 8601 date string (YYYY-MM-DD)
  - `TVS`: total value secured
  - `TVSByChain`: map of chain → TVS
  - `ProtocolCount`: number of protocols
  - `ChainCount`: number of active chains

**Given** snapshot creation
**When** fields are populated
**Then** all fields match the current aggregation result exactly

**Prerequisites:** Story 3.7 (aggregation result)

**Technical Notes:**
- Package: `internal/storage/history.go`
- `Snapshot` struct in `internal/models/snapshot.go`
- Use `time.Unix(ts, 0).Format("2006-01-02")` for date
- Reference: FR31, data-architecture.md

---

### Story 4.5: Implement History Loading from Output File

As a **developer**,
I want **existing history loaded from the output file on startup**,
So that **historical data is preserved across runs**.

**Acceptance Criteria:**

**Given** output file `switchboard-oracle-data.json` exists with `historical` array
**When** `LoadHistory(outputPath string)` is called
**Then** the `historical` array is extracted and returned as `[]Snapshot`
**And** snapshots are sorted by timestamp ascending (oldest first)

**Given** output file doesn't exist
**When** `LoadHistory` is called
**Then** empty slice is returned (not an error)
**And** debug log: "no existing history found, starting fresh"

**Given** output file exists but `historical` is empty or missing
**When** `LoadHistory` is called
**Then** empty slice is returned

**Given** output file is corrupted
**When** `LoadHistory` is called
**Then** warn log: "failed to load history, starting fresh"
**And** empty slice is returned (graceful degradation)

**Prerequisites:** Story 4.4

**Technical Notes:**
- Package: `internal/storage/history.go`
- Only load the `historical` field, not entire file
- Use `json.RawMessage` for partial parsing if needed
- Reference: FR34

---

### Story 4.6: Implement Snapshot Deduplication

As a **developer**,
I want **duplicate snapshots prevented**,
So that **history doesn't contain redundant entries**.

**Acceptance Criteria:**

**Given** existing history with snapshot at timestamp 1700000000
**When** `AppendSnapshot(history, newSnapshot)` is called with same timestamp
**Then** the new snapshot replaces the existing one (update in place)
**And** history length remains unchanged

**Given** existing history with snapshots at [1700000000, 1700003600]
**When** `AppendSnapshot` is called with timestamp 1700007200
**Then** new snapshot is appended
**And** history length increases by 1

**Given** history is unsorted after operations
**When** history is finalized
**Then** snapshots are sorted by timestamp ascending

**Prerequisites:** Story 4.5

**Technical Notes:**
- Package: `internal/storage/history.go`
- Function: `func (h *HistoryManager) AppendSnapshot(snapshot Snapshot) []Snapshot`
- Check for duplicate by timestamp before appending
- Use `sort.Slice()` to maintain order
- Reference: FR32

---

### Story 4.7: Implement History Retention (Keep All)

As a **developer**,
I want **all historical snapshots retained without pruning**,
So that **complete history is available for analysis**.

**Acceptance Criteria:**

**Given** history with 1000 snapshots spanning 90+ days
**When** a new snapshot is added
**Then** all existing snapshots are retained
**And** new snapshot is appended
**And** no automatic pruning occurs

**Given** MVP requirements
**When** history management is implemented
**Then** there is NO automatic pruning logic (FR33 - retain all)
**And** a comment notes "pruning may be added in future version"

**Prerequisites:** Story 4.6

**Technical Notes:**
- Package: `internal/storage/history.go`
- MVP explicitly requires NO pruning per PRD
- Future: could add configurable retention window
- Reference: FR33

---

### Story 4.8: Build State Manager Component

As a **developer**,
I want **a unified StateManager that handles all state operations**,
So that **I have a clean interface for incremental updates**.

**Acceptance Criteria:**

**Given** configuration with output directory
**When** `NewStateManager(cfg)` is called
**Then** a `StateManager` is created with paths configured:
  - State file: `{output_dir}/state.json`
  - Output file: `{output_dir}/switchboard-oracle-data.json`

**Given** a StateManager instance
**When** extraction cycle starts
**Then** `LoadState()` returns current state
**And** `LoadHistory()` returns existing snapshots
**And** `ShouldProcess(timestamp)` determines if processing needed

**Given** a successful extraction
**When** `SaveState(state)` and `AppendSnapshot(snapshot)` are called
**Then** both operations complete atomically
**And** state and history are consistent

**Prerequisites:** Stories 4.1-4.7

**Technical Notes:**
- Package: `internal/storage/state.go`
- Combine state and history management
- `StateManager` struct with all required methods
- Reference: fr-category-to-architecture-mapping.md

---

## Epic 5: Output & CLI

**Goal:** Implement JSON output generation and complete CLI with daemon mode, graceful shutdown, and all operational features.

**User Value:** After this epic, the complete working tool produces JSON files ready for dashboard consumption, runs as a daemon with 2-hour intervals, and handles all operational scenarios gracefully.

**FRs Covered:** FR35, FR36, FR37, FR38, FR39, FR40, FR41, FR42, FR43, FR44, FR45, FR46, FR47, FR48, FR56

---

### Story 5.1: Implement Full Output JSON Generation

As a **developer**,
I want **complete output JSON generated with all data and history**,
So that **dashboards have all the information they need**.

**Acceptance Criteria:**

**Given** aggregation results and historical snapshots
**When** `GenerateFullOutput(result, history, config)` is called
**Then** a `FullOutput` struct is created with:
  - `version`: "1.0.0"
  - `oracle`: name, website, documentation URL from config
  - `metadata`: last_updated, data_source, update_frequency, extractor_version
  - `summary`: total_value_secured, total_protocols, active_chains, categories
  - `metrics`: current_tvs, change_24h, change_7d, change_30d, growth metrics
  - `breakdown`: by_chain array, by_category array
  - `protocols`: ranked protocol list with all metadata
  - `historical`: complete snapshot history

**Given** full output struct
**When** serialized to JSON
**Then** output is human-readable with 2-space indentation
**And** file is written to `{output_dir}/switchboard-oracle-data.json`

**Prerequisites:** Story 3.7, Story 4.8

**Technical Notes:**
- Package: `internal/storage/writer.go`
- Use `json.MarshalIndent(data, "", "  ")` for formatting
- `FullOutput` struct in `internal/models/output.go`
- Reference: FR35, FR40, FR41, data-architecture.md

---

### Story 5.2: Implement Minified Output JSON Generation

As a **developer**,
I want **a minified version of the output JSON**,
So that **file transfer size is minimized**.

**Acceptance Criteria:**

**Given** the same `FullOutput` data
**When** minified output is generated
**Then** JSON is serialized without whitespace or indentation
**And** file is written to `{output_dir}/switchboard-oracle-data.min.json`
**And** content is identical to full output (just formatting differs)

**Given** full output is 500KB with formatting
**When** minified output is generated
**Then** minified size is significantly smaller (typically 60-70% of formatted size)

**Prerequisites:** Story 5.1

**Technical Notes:**
- Package: `internal/storage/writer.go`
- Use `json.Marshal()` (no indent) for minified
- Write to separate file, same directory
- Reference: FR36

---

### Story 5.3: Implement Summary Output JSON Generation

As a **developer**,
I want **a lightweight summary JSON with current snapshot only**,
So that **quick reads don't require loading full history**.

**Acceptance Criteria:**

**Given** aggregation results (no history needed)
**When** `GenerateSummaryOutput(result, config)` is called
**Then** a `SummaryOutput` struct is created with:
  - `version`: "1.0.0"
  - `oracle`: name, website, documentation URL
  - `metadata`: last_updated, data_source
  - `summary`: total_value_secured, total_protocols, active_chains, categories
  - `metrics`: current snapshot metrics only
  - `breakdown`: by_chain, by_category
  - `top_protocols`: top 10 protocols by TVL (subset)
  - NO `historical` array

**Given** summary output
**When** written to file
**Then** file is `{output_dir}/switchboard-summary.json`
**And** file size is much smaller than full output

**Prerequisites:** Story 5.1

**Technical Notes:**
- Package: `internal/storage/writer.go`
- `SummaryOutput` struct - subset of FullOutput fields
- Include top 10 protocols only to keep size small
- Reference: FR37

---

### Story 5.4: Implement Atomic File Writer

As a **developer**,
I want **all output files written atomically**,
So that **partial writes don't corrupt output files**.

**Acceptance Criteria:**

**Given** output data to write
**When** `WriteJSON(path string, data any)` is called
**Then** data is written to temp file first (`{path}.tmp`)
**And** temp file is renamed to target path
**And** operation is atomic (readers never see partial content)

**Given** output directory doesn't exist
**When** `WriteJSON` is called
**Then** directory is created with `os.MkdirAll()`
**And** file is written successfully

**Given** a write failure mid-operation
**When** error occurs
**Then** temp file is cleaned up
**And** original file (if exists) is preserved
**And** error is returned with context

**Given** multiple output files to write
**When** `WriteAllOutputs(full, minified, summary)` is called
**Then** all three files are written atomically
**And** if any write fails, error indicates which file failed

**Prerequisites:** Story 1.1

**Technical Notes:**
- Package: `internal/storage/writer.go`
- `Writer` struct with `NewWriter(outputDir)` constructor
- Use `os.CreateTemp()` in same directory for atomic rename guarantee
- Defer cleanup of temp files on error
- Reference: FR38, FR39, implementation-patterns.md

---

### Story 5.5: Implement CLI Flag Parsing

As an **operator**,
I want **command-line flags for controlling execution**,
So that **I can run the tool in different modes**.

**Acceptance Criteria:**

**Given** CLI invocation with `--once`
**When** application starts
**Then** single extraction is performed and application exits

**Given** CLI invocation with `--config /path/to/config.yaml`
**When** application starts
**Then** configuration is loaded from specified path

**Given** CLI invocation with `--dry-run`
**When** extraction completes
**Then** data is fetched and processed but NOT written to files
**And** log indicates "dry-run mode, skipping file writes"

**Given** CLI invocation with `--version`
**When** application starts
**Then** version string is printed (e.g., "defillama-extract v1.0.0")
**And** application exits with code 0

**Given** CLI invocation with no flags
**When** application starts
**Then** daemon mode is activated with default config path

**Prerequisites:** Story 1.2

**Technical Notes:**
- Package: `cmd/extractor/main.go`
- Use `flag` package from standard library
- Flags: `--once`, `--config`, `--dry-run`, `--version`
- Store in `CLIOptions` struct
- Reference: FR42, FR44, FR45, FR46

---

### Story 5.6: Implement Single Extraction Mode

As an **operator**,
I want **to run a single extraction and exit**,
So that **I can use cron or manual runs for scheduling**.

**Acceptance Criteria:**

**Given** `--once` flag is set
**When** application runs
**Then** one complete extraction cycle executes:
  1. Load config
  2. Load state
  3. Fetch API data
  4. Check if new data (skip if not)
  5. Aggregate data
  6. Write outputs (unless dry-run)
  7. Save state
  8. Exit

**Given** successful extraction in `--once` mode
**When** extraction completes
**Then** exit code is 0
**And** log: "extraction completed" with protocol_count, tvs, duration_ms

**Given** extraction failure in `--once` mode
**When** error occurs
**Then** exit code is 1
**And** error is logged with context

**Given** `--once` with no new data available
**When** skip logic triggers
**Then** exit code is 0 (not an error)
**And** log: "no new data, skipping extraction"

**Prerequisites:** Story 5.5, Story 3.7, Story 4.8, Story 5.4

**Technical Notes:**
- Package: `cmd/extractor/main.go`
- Create `runOnce(ctx, cfg)` function
- Wire together: API client → Aggregator → StateManager → Writer
- Reference: FR42, FR48

---

### Story 5.7: Implement Daemon Mode with Scheduler

As an **operator**,
I want **the service to run continuously with scheduled extractions**,
So that **data is automatically kept up to date**.

**Acceptance Criteria:**

**Given** daemon mode (no `--once` flag) with `scheduler.interval: 2h`
**When** application starts
**Then** extraction runs on schedule every 2 hours
**And** log: "daemon started, interval: 2h"

**Given** `scheduler.start_immediately: true`
**When** daemon starts
**Then** first extraction runs immediately
**And** subsequent extractions follow interval

**Given** `scheduler.start_immediately: false`
**When** daemon starts
**Then** first extraction waits for interval
**And** log: "waiting for first scheduled extraction"

**Given** daemon is running
**When** extraction cycle completes
**Then** next extraction is scheduled
**And** log: "next extraction at {timestamp}"

**Given** extraction fails in daemon mode
**When** error occurs
**Then** error is logged
**And** daemon continues running (doesn't exit)
**And** next extraction is scheduled normally

**Prerequisites:** Story 5.6

**Technical Notes:**
- Package: `cmd/extractor/main.go`
- Use `time.Ticker` for scheduling
- Create `runDaemon(ctx, cfg)` function
- Handle errors gracefully - log and continue
- Reference: FR43

---

### Story 5.8: Implement Graceful Shutdown

As an **operator**,
I want **graceful shutdown on SIGINT/SIGTERM**,
So that **in-progress operations complete cleanly**.

**Acceptance Criteria:**

**Given** daemon is running an extraction cycle
**When** SIGINT (Ctrl+C) is received
**Then** current extraction is allowed to complete
**And** log: "shutdown signal received, finishing current extraction"
**And** after completion, daemon exits cleanly with code 0

**Given** daemon is waiting for next scheduled extraction
**When** SIGTERM is received
**Then** wait is cancelled immediately
**And** log: "shutdown signal received, exiting"
**And** daemon exits cleanly with code 0

**Given** `--once` mode with extraction in progress
**When** SIGINT is received
**Then** extraction is cancelled via context
**And** partial results are NOT written
**And** exit code is 1

**Given** shutdown signal
**When** processing
**Then** signal is handled only once (no duplicate handling)

**Prerequisites:** Story 5.7

**Technical Notes:**
- Package: `cmd/extractor/main.go`
- Use `signal.NotifyContext()` for clean context cancellation
- Listen for `os.Interrupt` and `syscall.SIGTERM`
- Pass cancellable context to all operations
- Reference: FR47

---

### Story 5.9: Implement Extraction Cycle Logging

As an **operator**,
I want **extraction cycles logged with key metrics**,
So that **I can monitor system health**.

**Acceptance Criteria:**

**Given** extraction cycle starts
**When** processing begins
**Then** info log: "extraction started" with timestamp

**Given** extraction completes successfully
**When** results are available
**Then** info log: "extraction completed" with:
  - `duration_ms`: total extraction time
  - `protocol_count`: number of protocols found
  - `tvs`: total value secured
  - `chains`: number of active chains

**Given** extraction is skipped (no new data)
**When** skip occurs
**Then** info log: "extraction skipped, no new data" with:
  - `last_updated`: timestamp of existing data

**Given** extraction fails
**When** error occurs
**Then** error log: "extraction failed" with:
  - `error`: error message
  - `duration_ms`: time until failure

**Prerequisites:** Story 1.4, Story 5.6

**Technical Notes:**
- Use `slog` with structured attributes
- Track start time with `time.Now()`, calculate duration at end
- Include all relevant metrics in log attributes
- Reference: FR48, FR56

---

### Story 5.10: Build Complete Main Entry Point

As a **developer**,
I want **a complete main.go that wires everything together**,
So that **the application is fully functional**.

**Acceptance Criteria:**

**Given** application starts
**When** `main()` executes
**Then** the following sequence occurs:
  1. Parse CLI flags
  2. Handle `--version` (print and exit)
  3. Load configuration from file
  4. Apply environment overrides
  5. Validate configuration
  6. Initialize logger
  7. Create components (API client, Aggregator, StateManager, Writer)
  8. Set up signal handling
  9. Run in appropriate mode (once vs daemon)
  10. Exit with appropriate code

**Given** any initialization failure
**When** error occurs during startup
**Then** error is logged
**And** application exits with code 1

**Given** successful run
**When** application completes/terminates
**Then** exit code reflects success (0) or failure (1)

**Prerequisites:** All previous stories in Epic 5

**Technical Notes:**
- Package: `cmd/extractor/main.go`
- Use dependency injection pattern per architecture
- Wire: config → logger → client → aggregator → stateManager → writer → runner
- Reference: 17-complete-maingo-implementation.md

---

## FR Coverage Matrix

| FR | Description | Epic | Story |
|----|-------------|------|-------|
| FR1 | Fetch oracle data from `/oracles` endpoint | Epic 2 | 2.2 |
| FR2 | Fetch protocol metadata from `/lite/protocols2` endpoint | Epic 2 | 2.3 |
| FR3 | Parallel fetching of both endpoints | Epic 2 | 2.5 |
| FR4 | Include proper User-Agent header | Epic 2 | 2.1 |
| FR5 | Retry with exponential backoff | Epic 2 | 2.4 |
| FR6 | Configurable timeout (default 30s) | Epic 2 | 2.1 |
| FR7 | Handle API errors (429, 5xx) with retry | Epic 2 | 2.4 |
| FR8 | Detect non-retryable errors (4xx) | Epic 2 | 2.4 |
| FR9 | Filter protocols by oracle name | Epic 3 | 3.1 |
| FR10 | Check both `oracles` array and legacy `oracle` field | Epic 3 | 3.1 |
| FR11 | Extract TVS per protocol per chain | Epic 3 | 3.2 |
| FR12 | Extract protocol metadata | Epic 3 | 3.2 |
| FR13 | Identify all chains for target oracle | Epic 3 | 3.2 |
| FR14 | Extract timestamp from chart data | Epic 3 | 3.2 |
| FR15 | Calculate total TVS | Epic 3 | 3.3 |
| FR16 | Calculate TVS breakdown by chain | Epic 3 | 3.3 |
| FR17 | Calculate TVS breakdown by category | Epic 3 | 3.4 |
| FR18 | Rank protocols by TVL | Epic 3 | 3.5 |
| FR19 | Calculate 24h TVS change | Epic 3 | 3.6 |
| FR20 | Calculate 7d TVS change | Epic 3 | 3.6 |
| FR21 | Calculate 30d TVS change | Epic 3 | 3.6 |
| FR22 | Calculate protocol count growth | Epic 3 | 3.6 |
| FR23 | Identify largest protocol | Epic 3 | 3.5 |
| FR24 | Extract unique categories | Epic 3 | 3.4 |
| FR25 | Track last processed timestamp | Epic 4 | 4.1 |
| FR26 | Compare timestamps for new data | Epic 4 | 4.2 |
| FR27 | Skip when no new data | Epic 4 | 4.2 |
| FR28 | Recover from corrupted state | Epic 4 | 4.1 |
| FR29 | Update state atomically | Epic 4 | 4.3 |
| FR30 | Maintain historical snapshots | Epic 4 | 4.4 |
| FR31 | Store snapshot fields | Epic 4 | 4.4 |
| FR32 | Deduplicate snapshots | Epic 4 | 4.6 |
| FR33 | Retain all snapshots (no pruning) | Epic 4 | 4.7 |
| FR34 | Load history from output file | Epic 4 | 4.5 |
| FR35 | Generate full output JSON | Epic 5 | 5.1 |
| FR36 | Generate minified output JSON | Epic 5 | 5.2 |
| FR37 | Generate summary output JSON | Epic 5 | 5.3 |
| FR38 | Write files atomically | Epic 5 | 5.4 |
| FR39 | Create output directory | Epic 5 | 5.4 |
| FR40 | Include version/oracle info/metadata | Epic 5 | 5.1 |
| FR41 | Include extraction timestamp/attribution | Epic 5 | 5.1 |
| FR42 | Run-once mode (`--once`) | Epic 5 | 5.5, 5.6 |
| FR43 | Daemon mode with interval | Epic 5 | 5.7 |
| FR44 | Config file path (`--config`) | Epic 5 | 5.5 |
| FR45 | Dry-run mode (`--dry-run`) | Epic 5 | 5.5 |
| FR46 | Version flag (`--version`) | Epic 5 | 5.5 |
| FR47 | Graceful shutdown | Epic 5 | 5.8 |
| FR48 | Log extraction metrics | Epic 5 | 5.6, 5.9 |
| FR49 | Load config from YAML | Epic 1 | 1.2 |
| FR50 | Environment variable overrides | Epic 1 | 1.3 |
| FR51 | Sensible defaults | Epic 1 | 1.2 |
| FR52 | Validate config on startup | Epic 1 | 1.2 |
| FR53 | Structured logging (JSON/text) | Epic 1 | 1.4 |
| FR54 | Configurable log levels | Epic 1 | 1.4 |
| FR55 | Log API requests/retries/failures | Epic 2 | 2.6 |
| FR56 | Log extraction results | Epic 5 | 5.9 |

**Coverage Validation:** All 56 FRs mapped to stories ✓

---

## Summary

**Project:** defillama-extract
**Total Epics:** 5
**Total Stories:** 35

| Epic | Stories | FRs Covered |
|------|---------|-------------|
| Epic 1: Foundation | 4 | FR49-54 (6) |
| Epic 2: API Integration | 6 | FR1-8, FR55 (9) |
| Epic 3: Data Processing Pipeline | 7 | FR9-24 (16) |
| Epic 4: State & History Management | 8 | FR25-34 (10) |
| Epic 5: Output & CLI | 10 | FR35-48, FR56 (15) |

**Epic Sequencing:**
1. **Epic 1** establishes foundation (config, logging) - no dependencies
2. **Epic 2** builds API layer - depends on Epic 1 config
3. **Epic 3** implements data processing - depends on Epic 2 API client
4. **Epic 4** adds state management - depends on Epic 3 aggregation
5. **Epic 5** completes CLI and output - depends on all previous epics

**Key Characteristics:**
- Each epic delivers incremental value
- All stories are vertically sliced
- No forward dependencies (only backward references)
- Stories sized for single dev agent sessions
- BDD acceptance criteria for testability
- Technical notes reference architecture docs

---

_This epic breakdown transforms the PRD's 56 functional requirements into 35 implementable stories across 5 epics, ready for Phase 4 implementation._

_Created by PM agent through collaborative workflow with BMad._

