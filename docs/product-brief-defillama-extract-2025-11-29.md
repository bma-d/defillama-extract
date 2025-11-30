# Product Brief: defillama-extract

**Date:** 2025-11-29
**Author:** BMad
**Context:** Internal tooling / Data infrastructure

---

## Reference Documentation

This product brief is based on a comprehensive implementation specification. The full specification has been sharded for easier reference:

| Section | Reference | Description |
|---------|-----------|-------------|
| **Full Index** | [seed-doc/index.md](../docs-from-user/seed-doc/index.md) | Complete table of contents |
| **System Overview** | [1-system-overview.md](../docs-from-user/seed-doc/1-system-overview.md) | Objectives, key features, data flow |
| **Architecture** | [2-architecture-design.md](../docs-from-user/seed-doc/2-architecture-design.md) | Package structure, components |
| **API Specs** | [3-data-sources-api-specifications.md](../docs-from-user/seed-doc/3-data-sources-api-specifications.md) | DefiLlama API endpoints |
| **Data Models** | [4-data-models-structures.md](../docs-from-user/seed-doc/4-data-models-structures.md) | Go structs for API and internal models |
| **Core Components** | [5-core-components.md](../docs-from-user/seed-doc/5-core-components.md) | HTTP client, aggregator implementations |
| **Incremental Updates** | [6-incremental-update-strategy.md](../docs-from-user/seed-doc/6-incremental-update-strategy.md) | State and history management |
| **Aggregation Logic** | [7-custom-aggregation-logic-go-implementation.md](../docs-from-user/seed-doc/7-custom-aggregation-logic-go-implementation.md) | Metric calculations, filtering |
| **Storage** | [8-storage-caching.md](../docs-from-user/seed-doc/8-storage-caching.md) | Atomic file writer |
| **Error Handling** | [9-error-handling-resilience.md](../docs-from-user/seed-doc/9-error-handling-resilience.md) | Retry logic, graceful degradation |
| **Configuration** | [10-configuration-environment.md](../docs-from-user/seed-doc/10-configuration-environment.md) | YAML config, env vars |
| **Testing** | [11-testing-strategy-complete-implementation.md](../docs-from-user/seed-doc/11-testing-strategy-complete-implementation.md) | Table-driven tests, mocks, benchmarks |
| **Implementation Checklist** | [14-implementation-checklist.md](../docs-from-user/seed-doc/14-implementation-checklist.md) | Phased implementation guide |
| **Go Patterns** | [15-go-specific-patterns-idioms.md](../docs-from-user/seed-doc/15-go-specific-patterns-idioms.md) | slog, context, DI patterns |
| **Main.go** | [17-complete-maingo-implementation.md](../docs-from-user/seed-doc/17-complete-maingo-implementation.md) | Complete entry point implementation |
| **Quick Reference** | [appendix-b-quick-reference.md](../docs-from-user/seed-doc/appendix-b-quick-reference.md) | API endpoints, constants, output files |

---

## Executive Summary

A Go-based data extraction service that retrieves Switchboard oracle metrics from DefiLlama's public APIs, enabling accurate representation of Switchboard's protocol adoption and Total Value Secured (TVS) across the DeFi ecosystem. This powers an analytics dashboard that corrects DefiLlama's underrepresentation of Switchboard's true market position.

> **Spec Reference:** See [1-system-overview.md#11-objective](../docs-from-user/seed-doc/1-system-overview.md#11-objective) for full objectives.

---

## Core Vision

### Problem Statement

Switchboard oracle is being misrepresented on DefiLlama - the platform is not capturing all protocols that actually use Switchboard as their oracle provider. This creates an inaccurate picture of Switchboard's market presence and undermines the ability to demonstrate true adoption to stakeholders, potential integrators, and the broader DeFi community.

### Proposed Solution

Build a production-ready Go extraction service that:
1. Fetches comprehensive oracle and protocol data from DefiLlama APIs
2. Filters and aggregates Switchboard-specific metrics with custom calculations (7d/30d changes, growth rates, rankings)
3. Uses incremental updates to efficiently track changes over time
4. Outputs structured JSON files that power a corrected analytics dashboard

> **Spec Reference:** See [1-system-overview.md#13-data-flow-overview](../docs-from-user/seed-doc/1-system-overview.md#13-data-flow-overview) for detailed data flow diagram.

---

## Target Users

### Primary Users

**Downstream Systems (Machine Consumers)**
- Analytics dashboard (separate project) that reads the JSON output files
- Any future services requiring Switchboard oracle metrics

### Secondary Users

**DevOps / Maintainers**
- Engineers responsible for deploying, monitoring, and maintaining the extraction service
- Need visibility into health status, logs, and Prometheus metrics

---

## MVP Scope

### Core Features

1. **API Integration**
   - Fetch oracle data from `GET /oracles` endpoint
   - Fetch protocol metadata from `GET /lite/protocols2?b=2` endpoint
   - Parallel fetching for efficiency
   - Retry logic with exponential backoff

   > **Spec Reference:** [3-data-sources-api-specifications.md](../docs-from-user/seed-doc/3-data-sources-api-specifications.md) - Complete API endpoint specs, response fields, timing
   > **Spec Reference:** [5-core-components.md#51-http-client-component](../docs-from-user/seed-doc/5-core-components.md#51-http-client-component) - HTTP client implementation with retry

2. **Data Processing**
   - Filter protocols using Switchboard oracle
   - Aggregate TVS by chain and by category
   - Calculate derived metrics (24h/7d/30d changes, growth rates)
   - Rank protocols by TVL

   > **Spec Reference:** [7-custom-aggregation-logic-go-implementation.md](../docs-from-user/seed-doc/7-custom-aggregation-logic-go-implementation.md) - Complete Go implementations for all aggregation algorithms
   > **Spec Reference:** [4-data-models-structures.md](../docs-from-user/seed-doc/4-data-models-structures.md) - All data models (API response, internal, output)

3. **Incremental Updates**
   - Track last processed timestamp in state file
   - Skip processing when no new data available
   - 2-hour update cycle (conservative to respect API limits)

   > **Spec Reference:** [6-incremental-update-strategy.md](../docs-from-user/seed-doc/6-incremental-update-strategy.md) - State manager and history manager implementations

4. **Historical Tracking**
   - Maintain 90-day rolling window of snapshots
   - Automatic pruning of old data
   - Deduplication of snapshots

   > **Spec Reference:** [6-incremental-update-strategy.md#63-history-manager-implementation](../docs-from-user/seed-doc/6-incremental-update-strategy.md#63-history-manager-implementation) - History manager with append/prune/deduplicate

5. **JSON Output**
   - Full data file with history (`switchboard-oracle-data.json`)
   - Minified version (`switchboard-oracle-data.min.json`)
   - Summary file - current snapshot only (`switchboard-summary.json`)
   - State file for incremental updates (`state.json`)

   > **Spec Reference:** [8-storage-caching.md](../docs-from-user/seed-doc/8-storage-caching.md) - Atomic file writer implementation
   > **Spec Reference:** [4-data-models-structures.md#44-output-models](../docs-from-user/seed-doc/4-data-models-structures.md#44-output-models) - FullOutput, SummaryOutput, State models

6. **CLI Operation**
   - Run once mode (`--once` flag)
   - Scheduled daemon mode (2-hour intervals)
   - Configurable via YAML config file and environment variables
   - Structured logging (slog)

   > **Spec Reference:** [10-configuration-environment.md](../docs-from-user/seed-doc/10-configuration-environment.md) - YAML config structure, environment overrides
   > **Spec Reference:** [15-go-specific-patterns-idioms.md#151-structured-logging-with-slog](../docs-from-user/seed-doc/15-go-specific-patterns-idioms.md#151-structured-logging-with-slog) - slog implementation
   > **Spec Reference:** [17-complete-maingo-implementation.md](../docs-from-user/seed-doc/17-complete-maingo-implementation.md) - Complete main.go with CLI, scheduler, signal handling

### Out of Scope for MVP

- Prometheus metrics endpoint
- Health check HTTP endpoint
- Docker/containerization
- Systemd service integration
- Alerting integrations
- Web UI or API serving

### Technical Constraints

- **Runtime Environment:** Local machine only
- **Language:** Go 1.21+
- **External Dependencies:** DefiLlama public APIs only
- **Output:** Local filesystem JSON files

> **Spec Reference:** [2-architecture-design.md](../docs-from-user/seed-doc/2-architecture-design.md) - Package structure, component responsibilities, dependency graph
> **Spec Reference:** [appendix-a-go-dependencies.md](../docs-from-user/seed-doc/appendix-a-go-dependencies.md) - go.mod dependencies

---

## Success Criteria

1. Service correctly identifies all Switchboard protocols from DefiLlama data
2. JSON output matches expected schema for dashboard consumption
3. Incremental updates work reliably (no duplicate processing)
4. 90-day history maintained without manual intervention
5. Service runs stably in local environment with minimal resource usage

> **Spec Reference:** [11-testing-strategy-complete-implementation.md](../docs-from-user/seed-doc/11-testing-strategy-complete-implementation.md) - Testing strategy for validating success criteria

---

## Risks and Assumptions

### Critical Assumptions

1. **DefiLlama API Stability** - The public API remains available without authentication and maintains its current response schema
2. **Data Accuracy** - DefiLlama's protocol-to-oracle mappings are reasonably accurate (this service extracts, not corrects)
3. **Expected Scale** - ~21 protocols using Switchboard, primarily on Solana with presence on Sui, Aptos, Arbitrum, Ethereum

### Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| DefiLlama API schema changes | High - breaks extraction | Validate response structure, fail gracefully with clear errors |
| API rate limiting or blocking | Medium - interrupts updates | Conservative 2-hour polling, proper User-Agent header |
| API downtime | Low - temporary data staleness | Retry logic, keep last known good state |

> **Spec Reference:** [9-error-handling-resilience.md](../docs-from-user/seed-doc/9-error-handling-resilience.md) - Error wrapping patterns, retry config, graceful degradation strategies
> **Spec Reference:** [16-operational-concerns.md#164-api-schema-change-detection](../docs-from-user/seed-doc/16-operational-concerns.md#164-api-schema-change-detection) - API validation for schema change detection

### Chain Priority

1. **Solana** - Primary chain, highest TVS concentration
2. **Sui, Aptos** - Secondary chains with growing adoption
3. **Arbitrum, Ethereum** - Tertiary, include but lower priority

> **Spec Reference:** [3-data-sources-api-specifications.md#33-switchboard-specific-data](../docs-from-user/seed-doc/3-data-sources-api-specifications.md#33-switchboard-specific-data) - Expected chains and protocol categories

---

## Output Schema Ownership

This project **defines the JSON output schema** that downstream consumers (dashboard) must conform to. The schema follows the structure in the seed specification:

- `FullOutput` - complete data with historical snapshots
- `SummaryOutput` - current snapshot metrics only
- `State` - incremental update tracking

> **Spec Reference:** [4-data-models-structures.md#44-output-models](../docs-from-user/seed-doc/4-data-models-structures.md#44-output-models) - Complete FullOutput, OracleInfo, OutputMetadata, Summary, Breakdown structs
> **Spec Reference:** [appendix-b-quick-reference.md#b3-output-files](../docs-from-user/seed-doc/appendix-b-quick-reference.md#b3-output-files) - Output file names and purposes

---

## Implementation Roadmap

The seed specification includes a phased implementation checklist:

| Phase | Focus | Reference |
|-------|-------|-----------|
| Phase 1 | Foundation - Go module, config, models, HTTP client | [14-implementation-checklist.md#141-phase-1-foundation](../docs-from-user/seed-doc/14-implementation-checklist.md#141-phase-1-foundation) |
| Phase 2 | API Integration - Fetchers, parallel fetch, error handling | [14-implementation-checklist.md#142-phase-2-api-integration](../docs-from-user/seed-doc/14-implementation-checklist.md#142-phase-2-api-integration) |
| Phase 3 | Aggregation Logic - Filtering, TVS aggregation, metrics | [14-implementation-checklist.md#143-phase-3-aggregation-logic](../docs-from-user/seed-doc/14-implementation-checklist.md#143-phase-3-aggregation-logic) |
| Phase 4 | Storage & State - State manager, history, file writer | [14-implementation-checklist.md#144-phase-4-storage-state](../docs-from-user/seed-doc/14-implementation-checklist.md#144-phase-4-storage-state) |
| Phase 5 | Orchestration - Pipeline, scheduler, CLI, logging | [14-implementation-checklist.md#145-phase-5-orchestration](../docs-from-user/seed-doc/14-implementation-checklist.md#145-phase-5-orchestration) |
| Phase 6 | Production Readiness (Future) - Docker, Prometheus, alerts | [14-implementation-checklist.md#146-phase-6-production-readiness](../docs-from-user/seed-doc/14-implementation-checklist.md#146-phase-6-production-readiness) |

> **Note:** Phase 6 is out of scope for MVP but documented for future reference.

---

_This Product Brief captures the vision and requirements for defillama-extract._

_It was created through collaborative discovery and reflects the unique needs of this internal tooling project._

_Next: PRD workflow will transform this brief into detailed planning artifacts._
