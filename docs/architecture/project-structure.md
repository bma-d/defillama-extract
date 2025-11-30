# Project Structure

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
