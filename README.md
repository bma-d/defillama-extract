# defillama-extract

A Go-based data extraction service that fetches Switchboard oracle metrics from DefiLlama's public APIs, aggregates Total Value Secured (TVS) data, and outputs structured JSON files for dashboard consumption.

## Why This Exists

Switchboard oracle's Total Value Secured (TVS) is misrepresented on DefiLlama. The platform fails to capture all protocols that actually use Switchboard as their oracle provider. This tool extracts accurate, verifiable metrics that show Switchboard's actual adoption by:

- Fetching comprehensive oracle and protocol data from DefiLlama APIs
- Filtering protocols that use Switchboard oracle
- Aggregating TVS by chain and category
- Calculating derived metrics (24h/7d/30d changes, rankings)
- Outputting structured JSON for dashboard consumption

## Quick Start

### Prerequisites

- Go 1.21 or later

### Installation

```bash
# Clone the repository
git clone https://github.com/switchboard-xyz/defillama-extract.git
cd defillama-extract

# Build the binary
make build

# Or build directly with Go
go build -o bin/extractor ./cmd/extractor
```

### Run Once

```bash
# Run a single extraction using default config
./bin/extractor --once --config configs/config.yaml

# Or run directly with Go
go run ./cmd/extractor --once --config configs/config.yaml
```

### Run as Daemon

```bash
# Run continuously with 2-hour extraction intervals
./bin/extractor --config configs/config.yaml
```

## CLI Usage

```
defillama-extract [flags]

Flags:
  --once          Run single extraction and exit (default: daemon mode)
  --config PATH   Path to YAML configuration file (default: config.yaml)
  --dry-run       Fetch and process data but don't write output files
  --version       Print version and exit
```

### Examples

```bash
# Check version
./bin/extractor --version

# Single extraction with custom config
./bin/extractor --once --config /path/to/config.yaml

# Dry run (fetch data but don't write files)
./bin/extractor --once --dry-run --config configs/config.yaml

# Daemon mode (runs every 2 hours by default)
./bin/extractor --config configs/config.yaml
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Error (configuration, API failure, file write failure) |

### Graceful Shutdown

In daemon mode, send `SIGINT` (Ctrl+C) or `SIGTERM` to gracefully shut down:
- If extracting: completes current extraction, then exits
- If waiting: exits immediately

## Configuration

Configuration uses a layered approach (lowest to highest priority):
1. Built-in defaults
2. YAML configuration file
3. Environment variable overrides

### Sample Configuration

See `configs/config.yaml` for the full configuration:

```yaml
oracle:
  name: Switchboard
  website: https://switchboard.xyz
  documentation: https://docs.switchboard.xyz

api:
  oracles_url: https://api.llama.fi/oracles
  protocols_url: https://api.llama.fi/lite/protocols2?b=2
  timeout: 30s
  max_retries: 3
  retry_delay: 1s

output:
  directory: data
  full_file: switchboard-oracle-data.json
  min_file: switchboard-oracle-data.min.json
  summary_file: switchboard-summary.json
  state_file: state.json

scheduler:
  interval: 2h
  start_immediately: true

logging:
  level: info      # debug | info | warn | error
  format: json     # json | text
```

### Environment Variables

Override configuration values with environment variables:

| Variable | Description |
|----------|-------------|
| `ORACLE_NAME` | Override oracle name |
| `OUTPUT_DIR` | Override output directory |
| `LOG_LEVEL` | Override logging level |
| `API_TIMEOUT` | Override API timeout (e.g., "60s") |

Example:
```bash
LOG_LEVEL=debug OUTPUT_DIR=/tmp/data ./bin/extractor --once --config configs/config.yaml
```

## Output Files

All outputs are written atomically (temp file + rename) to prevent corruption.

| File | Purpose | Content |
|------|---------|---------|
| `switchboard-oracle-data.json` | Full data with history | Indented, human-readable JSON with chart_history + historical |
| `switchboard-oracle-data.min.json` | Same data, compact | No whitespace, smaller file size |
| `switchboard-summary.json` | Current snapshot + chart history | Lightweight for quick reads and graphing |
| `state.json` | Incremental update tracking | Last timestamp, protocol count |

### Output Schema

```json
{
  "version": "1.0.0",
  "oracle": {
    "name": "Switchboard",
    "website": "https://switchboard.xyz",
    "documentation": "https://docs.switchboard.xyz"
  },
  "metadata": {
    "last_updated": "2025-12-02T22:54:39Z",
    "data_source": "DefiLlama API",
    "update_frequency": "2h0m0s",
    "extractor_version": "1.0.0"
  },
  "summary": {
    "total_value_secured": 988531925.97,
    "total_protocols": 31,
    "active_chains": ["Solana", "Sui", "Aptos", "Movement", "RENEC"],
    "categories": ["Lending", "CDP", "Liquid Staking", ...]
  },
  "metrics": {
    "current_tvs": 988531925.97,
    "change_24h": 5.44,
    "change_7d": null,
    "change_30d": null
  },
  "breakdown": {
    "by_chain": [
      {"chain": "Solana", "tvs": 924169627.85, "percentage": 93.49, "protocol_count": 12}
    ],
    "by_category": [
      {"category": "Lending", "tvs": 500000000.00, "percentage": 50.5, "protocol_count": 5}
    ]
  },
  "protocols": [
    {"rank": 1, "name": "Protocol Name", "slug": "protocol-name", "category": "Lending", "tvl": 100000000, "tvs": 50000000, "chains": ["Solana"]}
  ],
  "chart_history": [
    {"timestamp": 1638144000, "date": "2021-11-29", "tvs": 6289642.70, "borrowed": 0, "staking": 0}
  ],
  "historical": [
    {"timestamp": 1764720000, "date": "2025-12-03", "tvs": 988531925.97, "tvs_by_chain": {...}, "protocol_count": 31, "chain_count": 5}
  ]
}
```

**Output Arrays:**
- `chart_history`: Daily TVS data from DefiLlama (4+ years, ~1,466 data points) - for time-series graphing
- `historical`: Extractor-run snapshots (every 2 hours) - detailed protocol-level data per extraction

Note: `switchboard-summary.json` includes `chart_history` for graphing but excludes `historical` and limits `protocols` to top 10.

## Development

### Project Structure

```
defillama-extract/
├── cmd/
│   └── extractor/          # CLI entry point
│       ├── main.go
│       └── main_test.go
├── internal/
│   ├── aggregator/         # Data aggregation logic
│   ├── api/                # DefiLlama API client
│   ├── config/             # Configuration loading
│   ├── logging/            # Structured logging setup
│   ├── models/             # Output data models
│   └── storage/            # State management & file writing
├── configs/
│   └── config.yaml         # Default configuration
├── data/                   # Output directory (created on first run)
├── docs/                   # Project documentation
├── Makefile
├── go.mod
└── README.md
```

### Building

```bash
# Build binary
make build

# Build and run all checks
make all
```

### Testing

```bash
# Run all tests
make test

# Or directly with Go
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package tests
go test ./internal/aggregator/...
```

### Linting

```bash
# Run linter (installs golangci-lint if needed)
make lint
```

### Clean

```bash
# Remove build artifacts and data directory
make clean
```

## How It Works

1. **Fetch**: Parallel API requests to DefiLlama `/oracles` and `/lite/protocols2` endpoints
2. **Filter**: Extract protocols using Switchboard oracle (checks both `oracles` array and legacy `oracle` field)
3. **Aggregate**: Calculate TVS totals, breakdowns by chain/category, derived metrics
4. **Compare**: Check if new data available (skip if timestamps match)
5. **Output**: Write JSON files atomically with historical snapshots
6. **State**: Save state for incremental updates

### Data Flow

```
DefiLlama API ──► Fetch (parallel) ──► Filter (Switchboard) ──► Aggregate
                                                                    │
                                                                    ▼
                    ◄── Write JSON (atomic) ◄── Generate Output ◄──┘
                    │
                    ▼
            Output Files (data/)
```

## Operational Notes

### Incremental Updates

The extractor tracks the last processed timestamp in `state.json`. If the API returns data with the same timestamp, extraction is skipped to avoid duplicate processing.

### History Retention

All historical snapshots are retained (no automatic pruning). The `historical` array in the full output grows with each new data point.

### Error Handling

- **API failures**: Retries with exponential backoff (configurable)
- **Corrupted state file**: Starts fresh (graceful degradation)
- **Daemon mode errors**: Logs error, continues to next scheduled extraction
- **Atomic writes**: Prevents partial/corrupted output files

### Logging

Structured logging via Go's `slog` package:

```json
{"time":"2025-12-02T19:54:37Z","level":"INFO","msg":"extraction started","timestamp":"2025-12-02T19:54:37Z"}
{"time":"2025-12-02T19:54:39Z","level":"INFO","msg":"extraction completed","duration_ms":1886,"protocol_count":31,"tvs":988531925.97,"chains":5}
```

## API Endpoints Used

| Endpoint | Purpose |
|----------|---------|
| `GET https://api.llama.fi/oracles` | Oracle TVS data, protocol lists |
| `GET https://api.llama.fi/lite/protocols2?b=2` | Protocol metadata |

## License

See LICENSE file for details.

---

Built with Go. Data sourced from [DefiLlama](https://defillama.com/).
