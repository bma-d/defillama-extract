# Technology Stack Details

> **Spec Reference:** [appendix-a-go-dependencies.md](../docs-from-user/seed-doc/appendix-a-go-dependencies.md)

## Core Technologies

**Go 1.23 (pinned for toolchain compatibility)**
- Compiled to single binary (NFR19)
- Built-in concurrency with goroutines
- Comprehensive standard library
- `slog` for structured logging (available since Go 1.21)

**Dependencies (go.mod)**
```go
module github.com/switchboard-xyz/defillama-extract

go 1.23

require (
    gopkg.in/yaml.v3 v3.0.1
)

// Post-MVP
// github.com/prometheus/client_golang v1.20.0
```

## Integration Points

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
