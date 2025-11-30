# 2. Architecture Design

## 2.1 Package Structure

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

## 2.2 Component Responsibilities

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

## 2.3 Dependency Graph

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
