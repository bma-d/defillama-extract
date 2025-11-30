# Architecture

## Executive Summary

This architecture defines a Go-based CLI data extraction service that fetches Switchboard oracle data from DefiLlama's public APIs, aggregates TVS (Total Value Secured) metrics, and outputs structured JSON files for dashboard consumption. The design prioritizes simplicity, reliability, and minimal dependencies - leveraging Go's excellent standard library for HTTP, JSON, and structured logging.

The architecture follows standard Go project layout conventions with clear package boundaries, explicit dependency injection, and comprehensive error handling patterns that ensure AI agents can implement consistently.

## Reference Documentation

This architecture is based on a comprehensive implementation specification. For detailed implementation guidance, refer to these seed documents:

| Section | Reference | Description |
|---------|-----------|-------------|
| **System Overview** | [1-system-overview.md](../docs-from-user/seed-doc/1-system-overview.md) | Objectives, key features, data flow |
| **Architecture Design** | [2-architecture-design.md](../docs-from-user/seed-doc/2-architecture-design.md) | Package structure, component responsibilities |
| **API Specifications** | [3-data-sources-api-specifications.md](../docs-from-user/seed-doc/3-data-sources-api-specifications.md) | DefiLlama API endpoints and response timing |
| **Data Models** | [4-data-models-structures.md](../docs-from-user/seed-doc/4-data-models-structures.md) | Go structs for API and internal models |
| **Core Components** | [5-core-components.md](../docs-from-user/seed-doc/5-core-components.md) | HTTP client, aggregator implementations |
| **Incremental Updates** | [6-incremental-update-strategy.md](../docs-from-user/seed-doc/6-incremental-update-strategy.md) | State and history management |
| **Aggregation Logic** | [7-custom-aggregation-logic-go-implementation.md](../docs-from-user/seed-doc/7-custom-aggregation-logic-go-implementation.md) | Metric calculations, protocol filtering |
| **Storage & Caching** | [8-storage-caching.md](../docs-from-user/seed-doc/8-storage-caching.md) | Atomic file writer implementation |
| **Error Handling** | [9-error-handling-resilience.md](../docs-from-user/seed-doc/9-error-handling-resilience.md) | Retry logic, graceful degradation |
| **Configuration** | [10-configuration-environment.md](../docs-from-user/seed-doc/10-configuration-environment.md) | YAML config, environment variables |
| **Testing Strategy** | [11-testing-strategy-complete-implementation.md](../docs-from-user/seed-doc/11-testing-strategy-complete-implementation.md) | Table-driven tests, mocks, benchmarks |
| **Deployment** | [12-deployment-considerations.md](../docs-from-user/seed-doc/12-deployment-considerations.md) | Deployment options and considerations |
| **API Response Examples** | [13-api-response-examples.md](../docs-from-user/seed-doc/13-api-response-examples.md) | Sample API responses for testing |
| **Implementation Checklist** | [14-implementation-checklist.md](../docs-from-user/seed-doc/14-implementation-checklist.md) | Phased implementation guide |
| **Go Patterns** | [15-go-specific-patterns-idioms.md](../docs-from-user/seed-doc/15-go-specific-patterns-idioms.md) | slog, context, dependency injection patterns |
| **Operational Concerns** | [16-operational-concerns.md](../docs-from-user/seed-doc/16-operational-concerns.md) | Monitoring, logging, maintenance |
| **Main.go Implementation** | [17-complete-maingo-implementation.md](../docs-from-user/seed-doc/17-complete-maingo-implementation.md) | Complete entry point implementation |
| **Go Dependencies** | [appendix-a-go-dependencies.md](../docs-from-user/seed-doc/appendix-a-go-dependencies.md) | go.mod dependencies |
| **Quick Reference** | [appendix-b-quick-reference.md](../docs-from-user/seed-doc/appendix-b-quick-reference.md) | API endpoints, constants, output files |

## Decision Summary

| Category | Decision | Version | Affects FRs | Rationale |
|----------|----------|---------|-------------|-----------|
| Language | Go | 1.24 | All | Compiled binary, excellent concurrency, comprehensive stdlib |
| Module Path | `github.com/switchboard-xyz/defillama-extract` | - | All | Standard Go module naming convention |
| HTTP Client | `net/http` (stdlib) | Go 1.24 | FR1-FR8 | No external dep needed; full control over retries |
| JSON | `encoding/json` (stdlib) | Go 1.24 | FR9-FR14, FR35-FR41 | Standard, sufficient performance |
| YAML Config | `gopkg.in/yaml.v3` | v3.0.1 | FR49-FR52 | De facto standard for Go YAML parsing |
| Logging | `log/slog` (stdlib) | Go 1.24 | FR53-FR56 | Structured logging built-in; JSON/text output |
| CLI Flags | `flag` (stdlib) | Go 1.24 | FR42-FR48 | Simple flags; no subcommands needed |
| Testing | `testing` (stdlib) | Go 1.24 | NFR17-18 | Table-driven tests; no framework needed |
| Linting | `golangci-lint` | latest | NFR18 | Standard Go linter aggregator |
| Build System | Makefile | - | NFR19 | Standard for Go projects |
| Monitoring | `prometheus/client_golang` | v1.20.0 | Post-MVP | Industry standard (scoped to post-MVP) |

## Project Structure

```
defillama-extract/
├── cmd/
│   └── extractor/
│       └── main.go                 # Entry point: CLI parsing, DI, signal handling
│
├── internal/                       # Private packages (Go compiler enforced)
│   ├── api/
│   │   ├── client.go               # HTTP client with retry logic
│   │   ├── client_test.go          # Client tests with mock server
│   │   ├── endpoints.go            # API URL constants
│   │   └── responses.go            # DefiLlama API response structs
│   │
│   ├── aggregator/
│   │   ├── aggregator.go           # Main orchestration: fetch -> filter -> aggregate
│   │   ├── aggregator_test.go      # Integration tests
│   │   ├── filter.go               # Protocol filtering by oracle name
│   │   ├── filter_test.go          # Filter tests
│   │   ├── metrics.go              # Derived metric calculations
│   │   └── metrics_test.go         # Metrics tests
│   │
│   ├── storage/
│   │   ├── writer.go               # Atomic JSON file writer
│   │   ├── writer_test.go          # Writer tests
│   │   ├── state.go                # Incremental state management
│   │   ├── state_test.go           # State tests
│   │   └── history.go              # Historical snapshot management
│   │
│   ├── models/
│   │   ├── api.go                  # API response type definitions
│   │   ├── oracle.go               # Oracle-related data structures
│   │   ├── protocol.go             # Protocol data structures
│   │   ├── snapshot.go             # Snapshot data structures
│   │   ├── output.go               # Output format structures
│   │   └── errors.go               # Custom error types
│   │
│   └── config/
│       ├── config.go               # Config loading & validation
│       └── config_test.go          # Config tests
│
├── testdata/                       # Test fixtures
│   ├── oracle_response.json        # Sample DefiLlama /oracles response
│   ├── protocol_response.json      # Sample DefiLlama /lite/protocols2 response
│   └── config.yaml                 # Test configuration
│
├── configs/
│   └── config.yaml                 # Default configuration template
│
├── data/                           # Output directory (gitignored)
│
├── .github/
│   └── workflows/
│       └── ci.yml                  # GitHub Actions: test, lint, build
│
├── go.mod                          # Module definition
├── go.sum                          # Dependency lockfile
├── Makefile                        # Build targets
├── .gitignore
├── .golangci.yml                   # Linter configuration
└── README.md
```

## FR Category to Architecture Mapping

> **Spec Reference:** [2-architecture-design.md](../docs-from-user/seed-doc/2-architecture-design.md)

| FR Category | Package(s) | Key Files | Spec Reference |
|-------------|-----------|-----------|----------------|
| API Integration (FR1-FR8) | `internal/api` | `client.go`, `endpoints.go` | [5-core-components.md](../docs-from-user/seed-doc/5-core-components.md) |
| Data Filtering (FR9-FR14) | `internal/aggregator` | `filter.go` | [7-custom-aggregation-logic-go-implementation.md](../docs-from-user/seed-doc/7-custom-aggregation-logic-go-implementation.md) |
| Aggregation & Metrics (FR15-FR24) | `internal/aggregator` | `aggregator.go`, `metrics.go` | [7-custom-aggregation-logic-go-implementation.md](../docs-from-user/seed-doc/7-custom-aggregation-logic-go-implementation.md) |
| Incremental Updates (FR25-FR29) | `internal/storage` | `state.go` | [6-incremental-update-strategy.md](../docs-from-user/seed-doc/6-incremental-update-strategy.md) |
| Historical Data (FR30-FR34) | `internal/storage` | `history.go` | [6-incremental-update-strategy.md](../docs-from-user/seed-doc/6-incremental-update-strategy.md) |
| Output Generation (FR35-FR41) | `internal/storage` | `writer.go` | [8-storage-caching.md](../docs-from-user/seed-doc/8-storage-caching.md) |
| CLI Operation (FR42-FR48) | `cmd/extractor` | `main.go` | [17-complete-maingo-implementation.md](../docs-from-user/seed-doc/17-complete-maingo-implementation.md) |
| Configuration (FR49-FR52) | `internal/config` | `config.go` | [10-configuration-environment.md](../docs-from-user/seed-doc/10-configuration-environment.md) |
| Logging & Observability (FR53-FR56) | All packages | Use `slog` throughout | [15-go-specific-patterns-idioms.md](../docs-from-user/seed-doc/15-go-specific-patterns-idioms.md) |

## Technology Stack Details

> **Spec Reference:** [appendix-a-go-dependencies.md](../docs-from-user/seed-doc/appendix-a-go-dependencies.md)

### Core Technologies

**Go 1.24**
- Compiled to single binary (NFR19)
- Built-in concurrency with goroutines
- Comprehensive standard library
- `slog` for structured logging (since Go 1.21)

**Dependencies (go.mod)**
```go
module github.com/switchboard-xyz/defillama-extract

go 1.24

require (
    gopkg.in/yaml.v3 v3.0.1
)

// Post-MVP
// github.com/prometheus/client_golang v1.20.0
```

### Integration Points

```
                    ┌─────────────────────────────────────┐
                    │         cmd/extractor/main.go       │
                    │  (CLI, scheduler, signal handling)  │
                    └──────────────────┬──────────────────┘
                                       │
                    ┌──────────────────▼──────────────────┐
                    │       internal/aggregator           │
                    │   (orchestrates the pipeline)       │
                    └───┬──────────────────────────────┬──┘
                        │                              │
         ┌──────────────▼──────────────┐    ┌─────────▼─────────┐
         │       internal/api          │    │  internal/storage │
         │  (fetch from DefiLlama)     │    │  (write outputs)  │
         └──────────────┬──────────────┘    └─────────┬─────────┘
                        │                              │
                        ▼                              ▼
              DefiLlama APIs                    data/*.json
              - GET /oracles                    - oracle-data.json
              - GET /lite/protocols2            - summary.json
                                                - state.json
```

## Implementation Patterns

> **Spec Reference:** [15-go-specific-patterns-idioms.md](../docs-from-user/seed-doc/15-go-specific-patterns-idioms.md)

These patterns ensure consistent implementation across all AI agents:

### Dependency Injection

Use constructor functions, not global state:

```go
// Good: explicit dependencies
func NewAggregator(client *api.Client, config *config.Config) *Aggregator {
    return &Aggregator{
        client: client,
        config: config,
    }
}

// main.go wires everything
func main() {
    cfg := config.Load()
    client := api.NewClient(cfg.API)
    agg := aggregator.NewAggregator(client, cfg)
    writer := storage.NewWriter(cfg.Output)
    // ... run
}
```

### Context Propagation

All I/O functions accept `context.Context` as first parameter:

```go
func (c *Client) FetchOracles(ctx context.Context) (*OracleResponse, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", c.oraclesURL, nil)
    // ...
}
```

### Parallel Fetching

Use `errgroup` for concurrent API calls:

```go
import "golang.org/x/sync/errgroup"

func (a *Aggregator) Fetch(ctx context.Context) (*Data, error) {
    var oracleResp *OracleResponse
    var protocolResp *ProtocolResponse

    g, ctx := errgroup.WithContext(ctx)

    g.Go(func() error {
        var err error
        oracleResp, err = a.client.FetchOracles(ctx)
        return err
    })

    g.Go(func() error {
        var err error
        protocolResp, err = a.client.FetchProtocols(ctx)
        return err
    })

    if err := g.Wait(); err != nil {
        return nil, err
    }

    return &Data{Oracles: oracleResp, Protocols: protocolResp}, nil
}
```

### Atomic File Writes

Never write directly to target file:

```go
func (w *Writer) WriteJSON(path string, data any) error {
    tmpPath := path + ".tmp"
    f, err := os.Create(tmpPath)
    // ... write and close ...
    return os.Rename(tmpPath, path)  // atomic
}
```

## Consistency Rules

### Naming Conventions

| Category | Convention | Example |
|----------|------------|---------|
| Packages | lowercase, single word | `api`, `storage`, `models` |
| Files | lowercase, underscores | `client.go`, `client_test.go` |
| Exported types | PascalCase | `OracleAPIResponse`, `Protocol` |
| Unexported types | camelCase | `httpClient`, `retryConfig` |
| Constants | PascalCase (exported) | `DefaultTimeout`, `MaxRetries` |
| Interfaces | PascalCase, `-er` suffix | `Reader`, `Writer`, `Aggregator` |
| Test functions | `Test` prefix + PascalCase | `TestCalculateChange` |
| JSON fields | snake_case | `"total_value_secured"` |

### Code Organization

| Rule | Pattern |
|------|---------|
| One primary type per file | `protocol.go` contains `Protocol` struct |
| Tests co-located | `client.go` → `client_test.go` same directory |
| Interfaces near usage | Define where used, not where implemented |

### Error Handling

> **Spec Reference:** [9-error-handling-resilience.md](../docs-from-user/seed-doc/9-error-handling-resilience.md)

**Sentinel errors for expected conditions:**
```go
var (
    ErrNoNewData       = errors.New("no new data available")
    ErrOracleNotFound  = errors.New("oracle not found in response")
    ErrInvalidResponse = errors.New("invalid API response")
)
```

**Custom error types for API errors:**
```go
type APIError struct {
    Endpoint   string
    StatusCode int
    Message    string
    Err        error
}

func (e *APIError) Error() string {
    return fmt.Sprintf("API error for %s (status %d): %s",
        e.Endpoint, e.StatusCode, e.Message)
}

func (e *APIError) Unwrap() error { return e.Err }

func (e *APIError) IsRetryable() bool {
    return e.StatusCode == 429 || e.StatusCode >= 500
}
```

**Error wrapping with context:**
```go
result, err := doSomething()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}
```

### Logging Strategy

**Use `slog` with structured fields:**
```go
slog.Info("extraction complete",
    "protocols", count,
    "tvs", totalTVS,
    "duration_ms", elapsed.Milliseconds(),
)
```

**Log levels:**
| Level | Use For |
|-------|---------|
| Debug | Detailed tracing, request/response bodies |
| Info | Normal operations, cycle start/end, metrics |
| Warn | Recoverable issues, retries, degraded state |
| Error | Failures requiring attention |

## Data Architecture

> **Spec Reference:** [4-data-models-structures.md](../docs-from-user/seed-doc/4-data-models-structures.md)

### API Response Models

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

### Output Models

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

## API Contracts

> **Spec Reference:** [3-data-sources-api-specifications.md](../docs-from-user/seed-doc/3-data-sources-api-specifications.md), [13-api-response-examples.md](../docs-from-user/seed-doc/13-api-response-examples.md)

### DefiLlama Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `https://api.llama.fi/oracles` | GET | Oracle TVS data, protocol lists |
| `https://api.llama.fi/lite/protocols2?b=2` | GET | Protocol metadata |

### Output Files

| File | Content | Format |
|------|---------|--------|
| `switchboard-oracle-data.json` | Full data with history | Indented JSON |
| `switchboard-oracle-data.min.json` | Same data, compact | Minified JSON |
| `switchboard-summary.json` | Current snapshot only | Indented JSON |
| `state.json` | Incremental update state | Indented JSON |

### JSON Output Schema

```json
{
  "version": "1.0.0",
  "oracle": {
    "name": "Switchboard",
    "website": "https://switchboard.xyz",
    "documentation": "https://docs.switchboard.xyz"
  },
  "metadata": {
    "last_updated": "2025-11-29T22:51:16Z",
    "data_source": "DefiLlama",
    "update_frequency": "2 hours",
    "extractor_version": "1.0.0"
  },
  "summary": {
    "total_value_secured": 180000000,
    "total_protocols": 21,
    "active_chains": 5,
    "categories": ["Lending", "CDP", "Liquid Staking"]
  },
  "metrics": { ... },
  "breakdown": { ... },
  "protocols": [ ... ],
  "historical": [ ... ]
}
```

## Security Architecture

This is a read-only data extraction tool with minimal security surface:

| Concern | Approach |
|---------|----------|
| API Access | Public APIs only, no authentication required |
| User Data | None collected or stored |
| Secrets | None required (no API keys) |
| File Permissions | Default user permissions for output files |
| Input Validation | Validate API response shapes, reject malformed data |
| Dependencies | Minimal (1 external dep) to reduce supply chain risk |

## Performance Considerations

> **Spec Reference:** [9-error-handling-resilience.md](../docs-from-user/seed-doc/9-error-handling-resilience.md), [5-core-components.md](../docs-from-user/seed-doc/5-core-components.md)

| NFR | Implementation |
|-----|----------------|
| NFR1: 30s request timeout | `http.Client.Timeout = 30 * time.Second` |
| NFR2: 2min extraction cycle | Parallel API fetches, efficient aggregation |
| NFR3: Parallel fetching | `errgroup` for concurrent API calls |
| NFR4: Atomic writes | Temp file + rename pattern |
| NFR5: Stable memory | No growing buffers, process and discard |

### Retry Configuration

> **Spec Reference:** [9-error-handling-resilience.md](../docs-from-user/seed-doc/9-error-handling-resilience.md)

```
Attempt 1: immediate
Attempt 2: 1s + jitter (0-500ms)
Attempt 3: 2s + jitter
Attempt 4: 4s + jitter
Attempt 5: fail
```

## Deployment Architecture

> **Spec Reference:** [12-deployment-considerations.md](../docs-from-user/seed-doc/12-deployment-considerations.md), [16-operational-concerns.md](../docs-from-user/seed-doc/16-operational-concerns.md)

### Local Execution

```bash
# Run once
./extractor --once --config config.yaml

# Run as daemon
./extractor --config config.yaml
```

### Build Targets (Makefile)

```makefile
.PHONY: build test lint clean

build:
	go build -o bin/extractor ./cmd/extractor

test:
	go test -v ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/ data/
```

## Development Environment

> **Spec Reference:** [14-implementation-checklist.md](../docs-from-user/seed-doc/14-implementation-checklist.md)

### Prerequisites

- Go 1.24 or later
- `golangci-lint` (for linting)
- Make (for build targets)

### Setup Commands

```bash
# Clone and enter directory
git clone https://github.com/switchboard-xyz/defillama-extract.git
cd defillama-extract

# Download dependencies
go mod download

# Run tests
make test

# Build binary
make build

# Run linter
make lint

# Run once (development)
./bin/extractor --once --config configs/config.yaml
```

## Testing Strategy

> **Spec Reference:** [11-testing-strategy-complete-implementation.md](../docs-from-user/seed-doc/11-testing-strategy-complete-implementation.md)

### Test Organization

| Test Type | Location | Purpose |
|-----------|----------|---------|
| Unit Tests | `*_test.go` co-located | Test individual functions/methods |
| Table-Driven Tests | All test files | Multiple inputs/outputs per test |
| Integration Tests | `aggregator_test.go` | Test component interactions |
| Mock Server Tests | `internal/api/client_test.go` | Test HTTP client with `httptest` |

### Test Fixtures

Test data lives in `testdata/` directory:
- `oracle_response.json` - Sample DefiLlama `/oracles` response
- `protocol_response.json` - Sample DefiLlama `/lite/protocols2` response
- `config.yaml` - Test configuration file

> **Sample Responses:** See [13-api-response-examples.md](../docs-from-user/seed-doc/13-api-response-examples.md) for example API responses

### Coverage Requirements

- Aggregation logic: High coverage (business-critical)
- Error handling paths: Must be tested
- HTTP retry logic: Test with mock server
- Configuration loading: Test all layers (defaults, YAML, env)

## Architecture Decision Records (ADRs)

### ADR-001: Use Go Standard Library Over Frameworks

**Context:** CLI applications in Go can use frameworks like Cobra/Viper or the standard library.

**Decision:** Use `flag` (stdlib) for CLI parsing, `net/http` for HTTP, `encoding/json` for JSON.

**Rationale:**
- Simple CLI with 4 flags doesn't warrant framework overhead
- Fewer dependencies = smaller attack surface, easier maintenance
- Go stdlib is well-tested and stable

### ADR-002: Atomic File Writes

**Context:** Output files must never be corrupted, even on crash.

**Decision:** Write to temp file, then atomic rename.

**Rationale:**
- `os.Rename` is atomic on POSIX systems
- Readers always see complete previous version or complete new version
- No partial writes possible

### ADR-003: Explicit Error Returns Over Exceptions

**Context:** Go uses error values, not exceptions.

**Decision:** Return errors explicitly, wrap with context, use sentinel errors for expected conditions.

**Rationale:**
- Go idiom - errors are values
- Explicit error handling at each call site
- Error wrapping provides stack context
- Sentinel errors enable type-safe error checking

### ADR-004: Structured Logging with slog

**Context:** Need machine-parseable logs for operations.

**Decision:** Use `log/slog` (stdlib) with JSON output in daemon mode.

**Rationale:**
- Built into Go 1.21+ (no dependency)
- Structured fields enable log aggregation
- Text mode available for development

### ADR-005: Minimal External Dependencies

**Context:** Each dependency is a maintenance and security burden.

**Decision:** Only external dependency is `gopkg.in/yaml.v3` for config parsing.

**Rationale:**
- Go stdlib provides HTTP, JSON, logging, testing
- Prometheus client scoped to post-MVP
- Fewer deps = faster builds, smaller binary, less CVE exposure

---

_Generated by BMAD Decision Architecture Workflow v1.0_
_Date: 2025-11-29_
_For: BMad_
