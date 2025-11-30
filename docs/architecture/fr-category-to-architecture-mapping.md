# FR Category to Architecture Mapping

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
