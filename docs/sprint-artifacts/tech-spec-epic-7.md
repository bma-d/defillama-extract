# Epic Technical Specification: Custom Protocols & Per-Protocol TVL Charting

Date: 2025-12-07
Author: BMad
Epic ID: 7
Status: Draft

---

## Overview

Epic 7 extends the defillama-extract service with two complementary capabilities: a custom protocol input system for tracking known Switchboard integrations that DefiLlama doesn't auto-detect, and a per-protocol TVL charting pipeline that fetches historical TVL time-series data for all tracked protocols (both auto-detected and custom).

This epic addresses a critical business need: DefiLlama's `/oracles` endpoint misses several protocols that actively use Switchboard as their oracle provider. The custom protocol input allows manual specification of these integrations with proof references (documentation URLs, GitHub links), ensuring comprehensive coverage. The TVL charting data enables downstream dashboards to show protocol-level historical trends and calculate TVS using configurable `simple_tvs_ratio` multipliers.

The implementation produces a new output file (`tvl-data.json`) that is independent from the main `switchboard-oracle-data.json`, following the same atomic write patterns and 2-hour polling cycle established in earlier epics.

## Objectives and Scope

**In Scope:**
- Custom protocols configuration file (`config/custom-protocols.json`) parsing and validation
- Protocol TVL fetcher using `GET https://api.llama.fi/protocol/{slug}` endpoint
- Merging of auto-detected protocols (from `/oracles`) with custom protocols
- Deduplication by protocol slug with custom taking precedence for metadata
- Pass-through of `integration_date` field (no filtering of TVL history)
- `tvl-data.json` and `tvl-data.min.json` output generation with atomic writes
- Integration of TVL pipeline into main extraction cycle
- Rate limiting for per-protocol API calls (200ms delay)
- State tracking for TVL pipeline to enable skip-when-unchanged behavior

**Out of Scope:**
- TVS calculation in output (downstream responsibility using `simple_tvs_ratio`)
- Date-based filtering of TVL history (downstream responsibility using `integration_date`)
- Modification of existing `switchboard-oracle-data.json` output
- New CLI flags beyond existing `--once`, `--config`, `--dry-run`
- Prometheus metrics or health endpoints (post-MVP)
- On-chain verification of oracle usage

## System Architecture Alignment

This epic extends the existing package structure with new components in `internal/tvl/`:

```
defillama-extract/
├── cmd/extractor/
│   └── main.go               # Updated: orchestrate TVL pipeline after main extraction
│
├── internal/
│   ├── api/
│   │   └── client.go         # Extended: add FetchProtocolTVL method
│   │
│   ├── tvl/                   # NEW package for TVL pipeline
│   │   ├── doc.go            # Package documentation
│   │   ├── custom.go         # Custom protocol loading and validation
│   │   ├── custom_test.go    # Custom protocol tests
│   │   ├── fetcher.go        # Per-protocol TVL fetching with rate limiting
│   │   ├── fetcher_test.go   # Fetcher tests
│   │   ├── merger.go         # Auto + custom protocol merging
│   │   ├── merger_test.go    # Merger tests
│   │   ├── output.go         # TVL output generation
│   │   └── output_test.go    # Output tests
│   │
│   ├── models/
│   │   └── tvl.go            # NEW: TVL-specific data structures
│   │
│   └── config/
│       └── config.go         # Extended: add TVL configuration section
│
├── config/
│   └── custom-protocols.json # Custom protocol definitions
│
└── data/
    ├── tvl-data.json         # NEW output: TVL history per protocol
    └── tvl-data.min.json     # NEW output: minified version
```

**Architectural Constraints:**
- Reuse existing HTTP client (`internal/api.Client`) with retry logic (ADR-001)
- Follow atomic file write pattern from `internal/storage.WriteAtomic()` (ADR-002)
- Explicit error returns, no panics (ADR-003)
- Structured logging with `slog` (ADR-004)
- Minimal external dependencies - standard library preferred (ADR-005)
- Context propagation for cancellation support
- Independent pipeline: TVL failures don't affect main oracle extraction

**Data Flow:**
```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      TVL Charting Pipeline (New)                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌────────────────────┐    ┌─────────────────────────────────────────────┐  │
│  │ custom-protocols   │───►│ LoadCustomProtocols()                       │  │
│  │     .json          │    │   - Parse JSON                              │  │
│  │                    │    │   - Validate required fields                │  │
│  └────────────────────┘    │   - Filter live: false                      │  │
│                            └─────────────────────────────────────────────┘  │
│                                              │                              │
│  ┌────────────────────┐                      ▼                              │
│  │  Main Extraction   │    ┌─────────────────────────────────────────────┐  │
│  │  (Oracle slugs)    │───►│ MergeProtocolLists()                        │  │
│  └────────────────────┘    │   - Dedupe by slug                          │  │
│                            │   - Custom takes precedence                 │  │
│                            │   - Set source: "auto" | "custom"           │  │
│                            └─────────────────────────────────────────────┘  │
│                                              │                              │
│                                              ▼                              │
│                            ┌─────────────────────────────────────────────┐  │
│                            │ FetchProtocolTVL() per slug                 │  │
│                            │   - GET /protocol/{slug}                    │  │
│                            │   - 200ms rate limit between calls          │  │
│                            │   - Extract tvl[] + currentChainTvls        │  │
│                            │   - Handle 404 gracefully                   │  │
│                            └─────────────────────────────────────────────┘  │
│                                              │                              │
│                                              ▼                              │
│  ┌────────────────────┐    ┌─────────────────────────────────────────────┐  │
│  │ tvl-data.json      │◄───│ GenerateTVLOutput()                         │  │
│  │ tvl-data.min.json  │    │   - Build output structure                  │  │
│  └────────────────────┘    │   - WriteAtomic()                           │  │
│                            └─────────────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Detailed Design

### Services and Modules

| Module | File | Responsibility | Inputs | Outputs |
|--------|------|----------------|--------|---------|
| CustomLoader | `tvl/custom.go` | Load/validate custom protocols JSON | File path | `[]CustomProtocol`, error |
| ProtocolTVLFetcher | `tvl/fetcher.go` | Fetch TVL history from DefiLlama | Slug, context | `*ProtocolTVLResponse`, error |
| ProtocolMerger | `tvl/merger.go` | Combine auto + custom protocol lists | Auto slugs, custom list | `[]MergedProtocol` |
| TVLOutputGenerator | `tvl/output.go` | Build and write tvl-data.json | Merged protocols, TVL data | error |

**CustomLoader:**
```go
type CustomLoader struct {
    configPath string
    logger     *slog.Logger
}

func NewCustomLoader(configPath string, logger *slog.Logger) *CustomLoader
```

**ProtocolTVLFetcher:**
```go
type ProtocolTVLFetcher struct {
    client    *api.Client
    rateLimit time.Duration  // 200ms default
    logger    *slog.Logger
}

func NewProtocolTVLFetcher(client *api.Client, logger *slog.Logger) *ProtocolTVLFetcher
```

### Data Models and Contracts

**CustomProtocol** (input from `custom-protocols.json`):
```go
type CustomProtocol struct {
    Slug           string   `json:"slug"`
    IsOngoing      bool     `json:"is-ongoing"`
    Live           bool     `json:"live"`
    Date           *int64   `json:"date,omitempty"`         // Unix timestamp, optional
    SimpleTVSRatio float64  `json:"simple-tvs-ratio"`
    DocsProof      string   `json:"docs_proof,omitempty"`
    GitHubProof    string   `json:"github_proof,omitempty"`
}
```

**ProtocolTVLResponse** (from `GET /protocol/{slug}`):
```go
type ProtocolTVLResponse struct {
    Name            string                 `json:"name"`
    TVL             []TVLDataPoint         `json:"tvl"`
    CurrentChainTvls map[string]float64    `json:"currentChainTvls"`
}

type TVLDataPoint struct {
    Date             int64   `json:"date"`              // Unix timestamp
    TotalLiquidityUSD float64 `json:"totalLiquidityUSD"`
}
```

**MergedProtocol** (internal representation after merge):
```go
type MergedProtocol struct {
    Slug            string   `json:"slug"`
    Name            string   `json:"name"`
    Source          string   `json:"source"`            // "auto" | "custom"
    IsOngoing       bool     `json:"is_ongoing"`
    SimpleTVSRatio  float64  `json:"simple_tvs_ratio"`
    IntegrationDate *int64   `json:"integration_date"`  // Unix timestamp, nullable
    DocsProof       *string  `json:"docs_proof"`
    GitHubProof     *string  `json:"github_proof"`
}
```

**TVLOutputProtocol** (per-protocol entry in output):
```go
type TVLOutputProtocol struct {
    Name            string           `json:"name"`
    Slug            string           `json:"slug"`
    Source          string           `json:"source"`
    IsOngoing       bool             `json:"is_ongoing"`
    SimpleTVSRatio  float64          `json:"simple_tvs_ratio"`
    IntegrationDate *int64           `json:"integration_date"`
    DocsProof       *string          `json:"docs_proof"`
    GitHubProof     *string          `json:"github_proof"`
    CurrentTVL      float64          `json:"current_tvl"`
    TVLHistory      []TVLHistoryItem `json:"tvl_history"`
}

type TVLHistoryItem struct {
    Date      string  `json:"date"`      // YYYY-MM-DD
    Timestamp int64   `json:"timestamp"` // Unix
    TVL       float64 `json:"tvl"`
}
```

**TVLOutput** (root output structure):
```go
type TVLOutput struct {
    Version   string                        `json:"version"`
    Metadata  TVLOutputMetadata             `json:"metadata"`
    Protocols map[string]TVLOutputProtocol  `json:"protocols"` // Keyed by slug
}

type TVLOutputMetadata struct {
    LastUpdated         string `json:"last_updated"`          // ISO 8601
    ProtocolCount       int    `json:"protocol_count"`
    CustomProtocolCount int    `json:"custom_protocol_count"`
}
```

### APIs and Interfaces

**New API Client Method:**

| Method | Signature | Purpose |
|--------|-----------|---------|
| FetchProtocolTVL | `(ctx context.Context, slug string) (*ProtocolTVLResponse, error)` | Fetch single protocol's TVL history |

**Endpoint:** `GET https://api.llama.fi/protocol/{slug}`

**Response Shape (relevant fields):**
```json
{
  "name": "Drift Trade",
  "tvl": [
    {"date": 1704067200, "totalLiquidityUSD": 150000000},
    {"date": 1704153600, "totalLiquidityUSD": 155000000}
  ],
  "currentChainTvls": {
    "Solana": 677000000
  }
}
```

**Error Handling:**
- 404: Protocol not found → warning log, skip protocol, continue
- 429/5xx: Retryable → existing retry logic with backoff
- 4xx (other): Non-retryable → error, skip protocol

**CustomLoader Methods:**

| Method | Signature | Purpose |
|--------|-----------|---------|
| Load | `(ctx context.Context) ([]CustomProtocol, error)` | Load and validate custom protocols |
| Validate | `(p CustomProtocol) error` | Validate single protocol entry |

**ProtocolMerger Methods:**

| Method | Signature | Purpose |
|--------|-----------|---------|
| Merge | `(autoSlugs []string, custom []CustomProtocol) []MergedProtocol` | Combine and deduplicate lists |

**TVLOutputGenerator Methods:**

| Method | Signature | Purpose |
|--------|-----------|---------|
| Generate | `(protocols []MergedProtocol, tvlData map[string]*ProtocolTVLResponse) *TVLOutput` | Build output structure |
| WriteAll | `(ctx context.Context, outputDir string, output *TVLOutput) error` | Write both JSON files |

### Workflows and Sequencing

**Custom Protocol Loading Flow:**
```
LoadCustomProtocols(configPath)
    │
    ├─► os.ReadFile(configPath)
    │       │
    │       ├─► [os.ErrNotExist] → log.Info("no custom protocols") → return [], nil
    │       ├─► [Other error] → return nil, error
    │       └─► [Success] → continue
    │
    ├─► json.Unmarshal(data, &protocols)
    │       │
    │       └─► [Error] → return nil, fmt.Errorf("parse custom protocols: %w", err)
    │
    ├─► For each protocol:
    │       │
    │       ├─► Validate required fields (slug, is-ongoing, live, simple-tvs-ratio)
    │       │       └─► [Invalid] → return nil, fmt.Errorf("invalid protocol %s: %w", slug, err)
    │       │
    │       └─► [live == false] → skip (don't add to result)
    │
    ├─► log.Info("loaded custom protocols", "total", len(loaded), "filtered", filtered)
    │
    └─► return loaded, nil
```

**Protocol Merge Flow:**
```
MergeProtocolLists(autoSlugs, customProtocols)
    │
    ├─► Create map[slug]MergedProtocol
    │
    ├─► For each autoSlug:
    │       │
    │       └─► Add to map with:
    │           - Source: "auto"
    │           - SimpleTVSRatio: 1.0
    │           - IntegrationDate: nil
    │           - IsOngoing: true
    │
    ├─► For each customProtocol:
    │       │
    │       └─► Upsert to map (overwrites auto if exists):
    │           - Source: "custom"
    │           - SimpleTVSRatio: from config
    │           - IntegrationDate: from config (nil if not set)
    │           - Proof URLs: from config
    │
    └─► Return sorted slice (by slug)
```

**TVL Fetching Flow:**
```
FetchAllProtocolTVL(ctx, protocols)
    │
    ├─► results := make(map[string]*ProtocolTVLResponse)
    │
    ├─► For each protocol (sequential with rate limit):
    │       │
    │       ├─► [ctx cancelled] → return results, ctx.Err()
    │       │
    │       ├─► FetchProtocolTVL(ctx, slug)
    │       │       │
    │       │       ├─► [404] → log.Warn("protocol not found") → continue
    │       │       ├─► [Other error] → log.Error → continue
    │       │       └─► [Success] → results[slug] = response
    │       │
    │       └─► time.Sleep(200ms)  // Rate limit
    │
    └─► return results, nil
```

**Main Integration Flow:**
```
RunExtractionCycle(ctx)
    │
    ├─► [Existing main extraction]
    │       │
    │       └─► Produces: oracleSlugs []string, state updated
    │
    ├─► [TVL Pipeline - runs regardless of main extraction skip]
    │       │
    │       ├─► LoadCustomProtocols()
    │       │
    │       ├─► MergeProtocolLists(oracleSlugs, customProtocols)
    │       │
    │       ├─► FetchAllProtocolTVL(ctx, mergedList)
    │       │
    │       ├─► GenerateTVLOutput(mergedList, tvlData)
    │       │
    │       └─► WriteTVLOutputs(ctx, outputDir, output)
    │               │
    │               ├─► WriteAtomic(tvl-data.json)
    │               └─► WriteAtomic(tvl-data.min.json)
    │
    └─► Return combined status (main success + TVL status)
```

## Non-Functional Requirements

### Performance

| Requirement | Target | Implementation |
|-------------|--------|----------------|
| Per-protocol fetch | < 5s per call | Existing 30s timeout, retries |
| Rate limiting | 200ms between calls | Sequential fetching with sleep |
| Total TVL pipeline | < 10 min for 50 protocols | 50 × (avg 3s + 0.2s) = ~160s typical |
| Output generation | < 500ms | Single JSON marshal + atomic write |
| Memory usage | < 50MB for TVL data | Streaming not needed; in-memory is fine |

**Measurable Targets:**
- Individual protocol fetch completes within timeout (30s default)
- Pipeline completes before next 2-hour cycle
- No memory growth between cycles (TVL data rebuilt each cycle)

### Security

| Concern | Mitigation |
|---------|------------|
| Custom config path traversal | Config path from trusted YAML config only |
| Malicious custom protocol data | Validate all fields; no code execution |
| Proof URL validation | URLs stored as strings; no fetching/rendering |
| File permissions | Output files written with 0644 (existing pattern) |
| API rate limiting | 200ms delay prevents abuse; User-Agent identifies service |

### Reliability/Availability

| Requirement | Implementation |
|-------------|----------------|
| Missing custom config | Empty list returned, not error; pipeline continues |
| Protocol 404 | Warning logged, protocol skipped, others processed |
| Partial TVL fetch failure | Successful fetches saved; failures logged |
| Main extraction failure | TVL pipeline can still run with cached auto slugs |
| TVL pipeline failure | Main extraction output preserved; error logged |

**Error Recovery:**
- Missing `custom-protocols.json` → No custom protocols; auto-detected only
- Invalid JSON in custom config → Error logged, pipeline halts (config error = stop)
- API timeout on protocol → Retry with backoff, then skip
- All fetches fail → Empty protocols in output; error logged

### Observability

| Log Event | Level | Attributes |
|-----------|-------|------------|
| Custom protocols loaded | Info | `total`, `filtered`, `config_path` |
| No custom config | Info | `path`, `reason: "file not found"` |
| Invalid custom config | Error | `path`, `error` |
| Protocol lists merged | Info | `auto_count`, `custom_count`, `total` |
| Protocol TVL fetch started | Debug | `slug`, `source` |
| Protocol TVL fetch complete | Debug | `slug`, `tvl_points`, `duration_ms` |
| Protocol not found (404) | Warn | `slug`, `status_code` |
| Protocol fetch failed | Error | `slug`, `error`, `attempts` |
| TVL pipeline complete | Info | `protocols_fetched`, `protocols_failed`, `duration_ms` |
| TVL output written | Info | `path`, `protocol_count`, `bytes` |

## Dependencies and Integrations

### Go Module Dependencies

**No new external dependencies required.** Existing versions (from `go.mod`, as of 2025-12-07):

| Module | Version | Notes |
|--------|---------|-------|
| go toolchain | go1.24.10 | Minimum required to build/test |
| golang.org/x/sync | v0.18.0 | errgroup used for concurrency | 
| gopkg.in/yaml.v3 | v3.0.1 | Config parsing |
| golang.org/x/net | v0.27.0 (indirect) | HTTP/URL helpers via deps |
| golang.org/x/text | v0.16.0 (indirect) | Encoding/text utilities via deps |

### Internal Dependencies

| Depends On | From Epic | Purpose |
|------------|-----------|---------|
| `internal/api.Client` | Epic 2 | HTTP client with retry logic |
| `internal/api.APIError` | Epic 2 | Error types for retry decisions |
| `internal/storage.WriteAtomic` | Epic 4/5 | Atomic file writing |
| `internal/config.Config` | Epic 1 | Configuration loading |
| `*slog.Logger` | Epic 1 | Structured logging |

### New Configuration

Extend `internal/config.Config` with TVL section:

```go
type TVLConfig struct {
    CustomProtocolsPath string        `yaml:"custom_protocols_path"`
    OutputFile          string        `yaml:"output_file"`
    MinOutputFile       string        `yaml:"min_output_file"`
    RateLimit           time.Duration `yaml:"rate_limit"`
    Enabled             bool          `yaml:"enabled"`
}

// Defaults
TVLConfig{
    CustomProtocolsPath: "config/custom-protocols.json",
    OutputFile:          "tvl-data.json",
    MinOutputFile:       "tvl-data.min.json",
    RateLimit:           200 * time.Millisecond,
    Enabled:             true,
}
```

### API Endpoint

| Endpoint | Method | Rate Limit | Purpose | Version/SLA |
|----------|--------|------------|---------|-------------|
| `https://api.llama.fi/protocol/{slug}` | GET | 200ms between calls | Per-protocol TVL history | Public DefiLlama API (unversioned); schema validated as of 2025-12-07; best-effort availability |

**Response Size:** Varies by protocol; typically 10KB-500KB depending on history length.

## Acceptance Criteria (Authoritative)

### AC-7.1: Load Custom Protocols Configuration (Story 7.1)
1. `LoadCustomProtocols()` reads from configurable path (default: `config/custom-protocols.json`)
2. Returns `[]CustomProtocol` with all validated entries where `live: true`
3. Returns empty slice (not error) when file doesn't exist
4. Returns error when file exists but contains invalid JSON
5. Validates required fields: `slug` (non-empty), `is-ongoing` (bool), `live` (bool), `simple-tvs-ratio` (0-1)
6. Filters out entries where `live: false`
7. Logs count of loaded vs filtered protocols

### AC-7.2: Implement Protocol TVL Fetcher (Story 7.2)
1. `FetchProtocolTVL(ctx, slug)` calls `GET https://api.llama.fi/protocol/{slug}`
2. Reuses existing HTTP client with retry/backoff from Epic 2
3. Extracts `name`, `tvl[]` array, and `currentChainTvls` from response
4. Returns `*ProtocolTVLResponse` on success
5. Returns `nil, nil` (not error) on 404 with warning log
6. Returns error on other failures (after retries exhausted)
7. Respects context cancellation

### AC-7.3: Merge Protocol Lists (Story 7.3)
1. `MergeProtocolLists()` combines auto-detected slugs with custom protocols
2. Auto-detected protocols get `source: "auto"`, `simple_tvs_ratio: 1.0`, `integration_date: nil`
3. Custom protocols get `source: "custom"` with config values
4. Deduplication by slug; custom takes precedence (overwrites auto)
5. Result sorted alphabetically by slug
6. Returns `[]MergedProtocol`

### AC-7.4: Include Integration Date in Output (Story 7.4)
1. Custom protocols: `integration_date` = config `date` value (null if not set)
2. Auto-detected protocols: `integration_date` = null
3. Full `tvl_history` array included regardless of integration date
4. No filtering of TVL data points by date

### AC-7.5: Generate tvl-data.json Output (Story 7.5)
1. Output structure matches schema in epic definition
2. `metadata.last_updated` in ISO 8601 format
3. `metadata.protocol_count` = total protocols in output
4. `metadata.custom_protocol_count` = protocols with `source: "custom"`
5. `protocols` keyed by slug
6. Each protocol includes all fields from `TVLOutputProtocol`
7. `tvl_history` entries have `date` (YYYY-MM-DD), `timestamp` (Unix), `tvl` (float)
8. Written atomically using `WriteAtomic()`
9. Minified version written to `tvl-data.min.json`

### AC-7.6: Integrate TVL Pipeline into Main Extraction Cycle (Story 7.6)
1. TVL pipeline runs after main oracle extraction completes
2. Both pipelines share same extraction timestamp in logs
3. Failures in TVL pipeline don't affect main pipeline output
4. Failures in main pipeline don't prevent TVL pipeline from running (uses cached slugs or empty)
5. Logging distinguishes main vs TVL pipeline operations
6. `--once` mode runs both pipelines
7. `--dry-run` mode skips file writes for both pipelines
8. Rate limiting enforced (200ms between protocol fetches)

## Traceability Mapping

| AC | Story | PRD Ref | Component | Test Idea |
|----|-------|---------|-----------|-----------|
| AC-7.1 | 7.1 | Growth Feature | `tvl/custom.go` | Unit: missing file returns []; Unit: invalid JSON returns error; Unit: live:false filtered |
| AC-7.2 | 7.2 | Growth Feature | `api/client.go`, `tvl/fetcher.go` | Unit: mock 200/404/500; Integration: real API call |
| AC-7.3 | 7.3 | Growth Feature | `tvl/merger.go` | Unit: auto only; Unit: custom only; Unit: overlap (custom wins) |
| AC-7.4 | 7.4 | Growth Feature | `tvl/output.go` | Unit: date set; Unit: date null; Unit: full history included |
| AC-7.5 | 7.5 | Growth Feature | `tvl/output.go` | Unit: schema compliance; Unit: atomic write; Unit: minified version |
| AC-7.6 | 7.6 | FR42-FR48 | `cmd/extractor/main.go` | Integration: full cycle; Unit: isolation between pipelines |

## Risks, Assumptions, Open Questions

### Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| DefiLlama rate limits TVL endpoint | Medium | Medium | 200ms delay; monitor 429s; increase delay if needed |
| Protocol slug mismatch | Low | Low | Validate slugs exist before bulk fetch |
| Large TVL history (memory) | Low | Low | ~50 protocols × 500KB = 25MB; acceptable |
| API schema changes | Low | Medium | Flexible parsing; log unknown fields |
| Long fetch time blocks cycle | Medium | Low | Sequential fetch bounded by protocol count |

### Assumptions

| Assumption | Validation |
|------------|------------|
| DefiLlama protocol endpoint stable | Used by DefiLlama UI; unlikely to change |
| 200ms rate limit sufficient | Test in production; adjust if 429s occur |
| Custom protocols file manually maintained | Documented; no auto-discovery needed |
| ~25 protocols typical count | Current auto-detected + expected custom |
| TVL history always available | Some protocols may have sparse data; handle empty |

### Open Questions

| Question | Owner | Resolution Path |
|----------|-------|-----------------|
| Should TVL state tracking be separate from main state? | Developer | Start with shared cycle; separate if needed |
| Retry count for 429 on TVL endpoint? | Developer | Use existing retry config; monitor |
| Max protocols before parallel fetch needed? | Developer | Sequential fine for <100; revisit if scale increases |

## Test Strategy Summary

### Test Types

| Type | Location | Coverage Target |
|------|----------|-----------------|
| Unit Tests | `internal/tvl/*_test.go` | All public functions, error paths |
| Integration Tests | `internal/tvl/fetcher_test.go` | Real API calls (tagged) |
| Table-Driven Tests | All test files | Multiple scenarios per function |

### Key Test Scenarios

**Custom Protocol Loading (7.1):**
- Valid JSON with multiple protocols → All loaded
- File not found → Empty slice, no error
- Invalid JSON → Error returned
- Missing required field → Error returned
- `live: false` entries → Filtered out
- `simple-tvs-ratio` out of range → Error returned

**Protocol TVL Fetching (7.2):**
- Successful fetch → `ProtocolTVLResponse` with tvl array
- 404 response → `nil, nil` returned, warning logged
- 500 response → Retry, then error
- Context cancelled → Immediate return
- Rate limit respected → 200ms between calls

**Protocol Merging (7.3):**
| Auto Slugs | Custom Slugs | Expected Result |
|------------|--------------|-----------------|
| [a, b] | [] | [a(auto), b(auto)] |
| [] | [c, d] | [c(custom), d(custom)] |
| [a, b] | [b, c] | [a(auto), b(custom), c(custom)] |
| [a] | [a] | [a(custom)] - custom wins |

**Output Generation (7.5):**
- All fields populated correctly
- Timestamps formatted correctly (ISO, Unix, YYYY-MM-DD)
- Empty TVL history handled
- Minified version has no whitespace
- Atomic write successful

**Integration (7.6):**
- Main extraction success + TVL success
- Main extraction failure + TVL success (TVL still runs)
- Main extraction success + TVL failure (main output preserved)
- `--once` runs both pipelines
- `--dry-run` skips both file writes

### Coverage Requirements

| Component | Target |
|-----------|--------|
| `tvl/custom.go` | 90%+ |
| `tvl/fetcher.go` | 85%+ (mock API) |
| `tvl/merger.go` | 95%+ |
| `tvl/output.go` | 90%+ |
| Error paths | All paths tested |

### Test Fixtures

| Fixture | Location | Purpose |
|---------|----------|---------|
| `valid_custom_protocols.json` | `testdata/` | Valid custom protocol config |
| `invalid_custom_protocols.json` | `testdata/` | Malformed JSON for error testing |
| `empty_custom_protocols.json` | `testdata/` | Empty array |
| `protocol_tvl_response.json` | `testdata/` | Sample /protocol/{slug} response |
| `protocol_404_response.json` | `testdata/` | 404 error response |

### Post-Review Follow-ups

- RESOLVED 2025-12-08: Presence validation for required booleans (`is-ongoing`, `live`) added with fixtures/tests (Story 7.1; internal/tvl/custom.go, internal/tvl/custom_test.go, testdata/tvl/missing_*.json).
