# defillama-extract - Product Requirements Document

**Author:** BMad
**Date:** 2025-11-29
**Version:** 1.0

---

## Executive Summary

Switchboard oracle's Total Value Secured (TVS) is being grossly misrepresented on DefiLlama. The platform fails to capture all protocols that actually use Switchboard as their oracle provider, creating an inaccurate picture of market presence and undermining Switchboard's ability to demonstrate true adoption to stakeholders, potential integrators, and the broader DeFi community.

This project builds a Go-based data extraction service that fetches comprehensive oracle and protocol data from DefiLlama's public APIs, filters and aggregates Switchboard-specific metrics, and outputs structured JSON files that power a corrected analytics dashboard - surfacing the truth about Switchboard's real market position.

### What Makes This Special

**Truth in data.** This isn't about marketing spin or optimistic projections - it's about extracting accurate, verifiable metrics that show Switchboard's actual adoption. The service provides an authoritative, automatically-updated source of truth that corrects the market's incomplete understanding of Switchboard's presence in the DeFi ecosystem.

---

## Project Classification

**Technical Type:** CLI Tool (Data Extraction Pipeline)
**Domain:** General (Internal Infrastructure Tooling)
**Complexity:** Low

This is a backend CLI service with no user interface. It operates on public API data from DefiLlama, requiring no authentication, handling no user data, and subject to no regulatory requirements. The complexity lies in robust data processing and incremental update logic, not domain-specific concerns.

---

## Success Criteria

### Primary Success Metrics

1. **Prove the Misrepresentation Gap**
   - Dashboard displays a TVS figure meaningfully higher than DefiLlama's default oracle view
   - The delta between "what DefiLlama shows" vs "what Switchboard actually secures" is quantified and visible

2. **Become the Authoritative Reference**
   - This service's output becomes the default source provided to potential integrators
   - Stakeholders reference these metrics instead of DefiLlama's incomplete view
   - Data is trusted because it's verifiable (sourced from DefiLlama's own APIs, just aggregated correctly)

3. **Complete Protocol Capture**
   - Zero false negatives: every protocol that DefiLlama marks as using Switchboard is captured
   - All chains where Switchboard operates are represented (Solana primary, Sui/Aptos secondary, Arbitrum/Ethereum tertiary)

### Technical Acceptance Criteria

4. **Schema Compliance** - JSON output matches the defined schema for dashboard consumption
5. **Reliable Incremental Updates** - No duplicate processing, no data gaps, state recovery works
6. **History Integrity** - 90-day rolling window maintained automatically without manual intervention
7. **Operational Stability** - Runs in local environment with minimal resource usage, handles API failures gracefully

### Data Validation Boundary

This service trusts DefiLlama's `oracles` field mapping. It extracts and aggregates what DefiLlama knows about Switchboard usage - it does not independently verify oracle usage on-chain. This is an intentional scope boundary for MVP.

---

## Product Scope

### MVP - Minimum Viable Product

**Core Data Pipeline:**
- Fetch oracle data from DefiLlama `GET /oracles` endpoint
- Fetch protocol metadata from DefiLlama `GET /lite/protocols2?b=2` endpoint
- Parallel fetching with retry logic and exponential backoff
- Filter protocols using Switchboard oracle
- Aggregate TVS by chain and by category
- Calculate derived metrics (24h/7d/30d changes, growth rates, protocol rankings)

**Incremental Update System:**
- Track last processed timestamp in state file
- Skip processing when no new data available
- 2-hour update cycle (conservative to respect API limits)

**Historical Tracking:**
- Retain all historical snapshots (no automatic pruning)
- Deduplication of snapshots

**Output:**
- Full data file with history (`switchboard-oracle-data.json`) - human-readable formatted JSON
- Minified version (`switchboard-oracle-data.min.json`) - same data, whitespace removed for smaller file size
- Summary file - current snapshot only (`switchboard-summary.json`) - lightweight for quick reads
- State file for incremental updates (`state.json`)

**CLI Operation:**
- Run-once mode (`--once` flag)
- Scheduled daemon mode (2-hour intervals)
- YAML config file + environment variable overrides
- Structured logging (slog)

### Out of Scope for MVP

- Prometheus metrics endpoint
- Health check HTTP endpoint
- Docker/containerization
- Systemd service integration
- Alerting integrations
- Web UI or API serving
- Multi-oracle support
- Additional data sources beyond DefiLlama

### Growth Features (Post-MVP)

- **Prometheus Metrics** - Expose operational metrics for monitoring
- **Health Check Endpoint** - HTTP endpoint for orchestration systems
- **Containerization** - Docker support for deployment flexibility
- **Alerting** - Notifications when new protocols adopt Switchboard or significant TVS changes occur

### Vision (Future Platform)

This MVP is the foundation for a comprehensive oracle data platform:

1. **Multi-Oracle Comparison** - Expand beyond Switchboard to track Chainlink, Pyth, and other oracle providers, enabling competitive analysis

2. **Real-Time Streaming** - Move from polling-based updates to real-time data streaming for live dashboards

3. **Public API** - Serve the aggregated data via REST/GraphQL API instead of static JSON files

4. **On-Chain Verification** - Add a verification layer that validates DefiLlama's oracle mappings against actual on-chain data

5. **Additional Protocol Sources** - Integrate data sources beyond DefiLlama to capture protocols they may miss

---

## Reference Documentation

This PRD is based on a comprehensive implementation specification. The full specification has been sharded for detailed reference:

| Section | Reference | Description |
|---------|-----------|-------------|
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

## CLI Tool Specific Requirements

### Command Structure

The CLI operates in two primary modes:

**Run-Once Mode (`--once`)**
- Execute a single extraction cycle and exit
- Useful for manual runs, testing, and cron-based scheduling
- Exit code 0 on success, non-zero on failure

**Daemon Mode (default)**
- Run continuously with scheduled extraction cycles
- 2-hour interval between extractions (configurable)
- Graceful shutdown on SIGINT/SIGTERM
- Option to start immediately or wait for first interval

**Additional Flags:**
- `--config <path>` - Path to YAML configuration file (default: `config.yaml`)
- `--dry-run` - Fetch and process data but don't write output files
- `--version` - Print version and exit

### Configuration Method

**Layered configuration (lowest to highest priority):**
1. Built-in defaults
2. YAML configuration file
3. Environment variable overrides

**Key Configuration Sections:**
- `oracle` - Target oracle name, website, documentation URL
- `api` - DefiLlama endpoint URLs, timeout, retry settings
- `output` - Output directory and file names
- `history` - Retention settings (Note: MVP keeps all history, no pruning)
- `scheduler` - Interval, start_immediately flag
- `logging` - Level (debug/info/warn/error), format (json/text), output (stdout/file)

**Environment Variables:**
- `ORACLE_NAME` - Override oracle name
- `OUTPUT_DIR` - Override output directory
- `LOG_LEVEL` - Override logging level

### Output Formats

All outputs are JSON files written atomically (temp file + rename):

| File | Purpose | Content |
|------|---------|---------|
| `switchboard-oracle-data.json` | Full data with history | Indented, human-readable |
| `switchboard-oracle-data.min.json` | Same data, compact | No whitespace, smaller transfer size |
| `switchboard-summary.json` | Current snapshot only | Lightweight for quick reads |
| `state.json` | Incremental update tracking | Last timestamp, counts |

### Logging

Structured logging via Go's `slog` package:
- JSON format for machine parsing (default in daemon mode)
- Text format for human readability (useful in dev/debug)
- Log levels: debug, info, warn, error
- Key fields: timestamp, level, message, contextual attributes

### Error Handling & Exit Codes

| Exit Code | Meaning |
|-----------|---------|
| 0 | Success |
| 1 | Configuration error |
| 1 | API fetch failure (after retries exhausted) |
| 1 | File write failure |

Daemon mode logs errors but continues running; only exits on shutdown signal.

---

## Functional Requirements

### API Integration

> **Spec Reference:** [3-data-sources-api-specifications.md](../docs-from-user/seed-doc/3-data-sources-api-specifications.md), [5-core-components.md](../docs-from-user/seed-doc/5-core-components.md)

- **FR1:** System fetches oracle data from DefiLlama `GET /oracles` endpoint
- **FR2:** System fetches protocol metadata from DefiLlama `GET /lite/protocols2?b=2` endpoint
- **FR3:** System fetches both endpoints in parallel to minimize total fetch time
- **FR4:** System includes proper User-Agent header identifying the extractor
- **FR5:** System retries failed API requests with exponential backoff and jitter
- **FR6:** System respects configurable timeout for API requests (default 30s)
- **FR7:** System handles API errors gracefully (429, 5xx) with appropriate retry logic
- **FR8:** System detects and reports non-retryable errors (4xx client errors)

### Data Filtering & Extraction

> **Spec Reference:** [4-data-models-structures.md](../docs-from-user/seed-doc/4-data-models-structures.md), [7-custom-aggregation-logic-go-implementation.md#72-protocol-filtering](../docs-from-user/seed-doc/7-custom-aggregation-logic-go-implementation.md#72-protocol-filtering)

- **FR9:** System filters protocols by exact oracle name match ("Switchboard")
- **FR10:** System checks both `oracles` array and legacy `oracle` field for matching
- **FR11:** System extracts TVS (Total Value Secured) data per protocol per chain
- **FR12:** System extracts protocol metadata (name, slug, category, TVL, chains, URL)
- **FR13:** System identifies all chains where the target oracle is used
- **FR14:** System extracts timestamp of latest data point from chart data

### Aggregation & Metrics

> **Spec Reference:** [7-custom-aggregation-logic-go-implementation.md](../docs-from-user/seed-doc/7-custom-aggregation-logic-go-implementation.md)

- **FR15:** System calculates total TVS across all protocols using the oracle
- **FR16:** System calculates TVS breakdown by chain with percentage of total
- **FR17:** System calculates TVS breakdown by protocol category with percentage of total
- **FR18:** System ranks protocols by TVL in descending order
- **FR19:** System calculates 24-hour TVS change percentage (when historical data available)
- **FR20:** System calculates 7-day TVS change percentage (when historical data available)
- **FR21:** System calculates 30-day TVS change percentage (when historical data available)
- **FR22:** System calculates protocol count growth over 7-day and 30-day periods
- **FR23:** System identifies largest protocol by TVL
- **FR24:** System extracts unique categories across all filtered protocols

### Incremental Updates

> **Spec Reference:** [6-incremental-update-strategy.md](../docs-from-user/seed-doc/6-incremental-update-strategy.md)

- **FR25:** System tracks last successfully processed timestamp in state file
- **FR26:** System compares latest API timestamp against last processed timestamp
- **FR27:** System skips processing when no new data is available
- **FR28:** System recovers gracefully from corrupted state file (starts fresh)
- **FR29:** System updates state file atomically after successful extraction

### Historical Data Management

> **Spec Reference:** [6-incremental-update-strategy.md#63-history-manager-implementation](../docs-from-user/seed-doc/6-incremental-update-strategy.md#63-history-manager-implementation)

- **FR30:** System maintains historical snapshots of TVS data over time
- **FR31:** System stores timestamp, date, TVS, TVS by chain, protocol count, and chain count per snapshot
- **FR32:** System deduplicates snapshots with identical timestamps
- **FR33:** System retains all historical snapshots (no automatic pruning)
- **FR34:** System loads existing history from output file on startup

### Output Generation

> **Spec Reference:** [8-storage-caching.md](../docs-from-user/seed-doc/8-storage-caching.md), [4-data-models-structures.md#44-output-models](../docs-from-user/seed-doc/4-data-models-structures.md#44-output-models)

- **FR35:** System generates full output JSON with all data and complete history
- **FR36:** System generates minified output JSON (same data, no whitespace)
- **FR37:** System generates summary output JSON with current snapshot only (no history)
- **FR38:** System writes all output files atomically (temp file + rename)
- **FR39:** System creates output directory if it doesn't exist
- **FR40:** System includes version, oracle info, and metadata in all outputs
- **FR41:** System includes extraction timestamp and data source attribution in metadata

### CLI Operation

> **Spec Reference:** [17-complete-maingo-implementation.md](../docs-from-user/seed-doc/17-complete-maingo-implementation.md)

- **FR42:** System runs in single-extraction mode with `--once` flag
- **FR43:** System runs in daemon mode with configurable interval (default 2 hours)
- **FR44:** System accepts configuration file path via `--config` flag
- **FR45:** System supports dry-run mode that fetches but doesn't write files
- **FR46:** System prints version information with `--version` flag
- **FR47:** System shuts down gracefully on SIGINT and SIGTERM signals
- **FR48:** System logs extraction start, completion, duration, and key metrics

### Configuration

> **Spec Reference:** [10-configuration-environment.md](../docs-from-user/seed-doc/10-configuration-environment.md)

- **FR49:** System loads configuration from YAML file
- **FR50:** System applies environment variable overrides to configuration
- **FR51:** System provides sensible defaults for all configuration values
- **FR52:** System validates configuration on startup

### Logging & Observability

> **Spec Reference:** [15-go-specific-patterns-idioms.md#151-structured-logging-with-slog](../docs-from-user/seed-doc/15-go-specific-patterns-idioms.md#151-structured-logging-with-slog)

- **FR53:** System logs in structured format (JSON or text, configurable)
- **FR54:** System supports configurable log levels (debug, info, warn, error)
- **FR55:** System logs API request attempts, retries, and failures
- **FR56:** System logs extraction cycle results with protocol count and TVS

---

## Non-Functional Requirements

### Performance

> **Spec Reference:** [5-core-components.md](../docs-from-user/seed-doc/5-core-components.md), [8-storage-caching.md](../docs-from-user/seed-doc/8-storage-caching.md)

- **NFR1:** API requests complete within 30 seconds (configurable timeout)
- **NFR2:** Full extraction cycle completes within 2 minutes under normal conditions
- **NFR3:** Parallel API fetching reduces total fetch time vs sequential
- **NFR4:** Atomic file writes prevent partial/corrupted output files
- **NFR5:** Memory usage remains stable over extended daemon operation (no leaks)

### Reliability & Resilience

> **Spec Reference:** [9-error-handling-resilience.md](../docs-from-user/seed-doc/9-error-handling-resilience.md)

- **NFR6:** System continues operating after transient API failures (retry logic)
- **NFR7:** System preserves last known good output when current extraction fails
- **NFR8:** System recovers automatically from corrupted state file
- **NFR9:** System handles unexpected API response shapes without crashing
- **NFR10:** Graceful degradation: API failure → log error → retry next cycle

### Integration

> **Spec Reference:** [3-data-sources-api-specifications.md#32-api-response-timing](../docs-from-user/seed-doc/3-data-sources-api-specifications.md#32-api-response-timing)

- **NFR11:** Respect DefiLlama API etiquette (conservative polling interval, proper User-Agent)
- **NFR12:** 2-hour minimum polling interval to avoid rate limiting
- **NFR13:** Handle API schema changes gracefully (log warnings for unexpected fields)
- **NFR14:** Output JSON schema remains stable for downstream dashboard consumption

### Maintainability

> **Spec Reference:** [2-architecture-design.md](../docs-from-user/seed-doc/2-architecture-design.md), [15-go-specific-patterns-idioms.md](../docs-from-user/seed-doc/15-go-specific-patterns-idioms.md)

- **NFR15:** Structured logging enables easy debugging and monitoring
- **NFR16:** Configuration externalized (no hardcoded values for tunables)
- **NFR17:** Clean separation of concerns (API client, aggregator, storage, CLI)
- **NFR18:** Code follows Go idioms and standard project layout

### Operational

> **Spec Reference:** [appendix-a-go-dependencies.md](../docs-from-user/seed-doc/appendix-a-go-dependencies.md)

- **NFR19:** Single binary deployment (no external runtime dependencies)
- **NFR20:** Runs on local machine without elevated privileges
- **NFR21:** Output files readable by any JSON parser
- **NFR22:** State file human-readable for debugging

---

## Technical Reference

### DefiLlama API Endpoints

| Endpoint | Purpose | Update Frequency |
|----------|---------|------------------|
| `GET https://api.llama.fi/oracles` | Oracle TVS data, protocol lists, chain mappings | ~hourly |
| `GET https://api.llama.fi/lite/protocols2?b=2` | Protocol metadata (name, category, TVL, chains) | ~hourly |

### Expected Data Characteristics

- **Oracle Name:** `Switchboard` (exact match, case-sensitive)
- **Expected Protocol Count:** ~21 protocols
- **Primary Chain:** Solana (highest TVS concentration)
- **Secondary Chains:** Sui, Aptos
- **Tertiary Chains:** Arbitrum, Ethereum
- **Protocol Categories:** Lending, CDP, Liquid Staking, Dexes, Derivatives

### Output Schema (FullOutput)

```
{
  "version": "1.0.0",
  "oracle": { "name", "website", "documentation" },
  "metadata": { "last_updated", "data_source", "update_frequency", "extractor_version" },
  "summary": { "total_value_secured", "total_protocols", "active_chains", "categories" },
  "metrics": { "current_tvs", "change_24h", "change_7d", "change_30d", ... },
  "breakdown": { "by_chain": [...], "by_category": [...] },
  "protocols": [ { "rank", "name", "slug", "category", "tvl", "tvs", "chains", ... } ],
  "historical": [ { "timestamp", "date", "tvs", "tvs_by_chain", "protocol_count", ... } ]
}
```

---

## PRD Summary

**Project:** defillama-extract
**Type:** CLI Tool (Data Extraction Pipeline)
**Domain:** Internal Infrastructure Tooling

**Core Value:** Truth in data - surfacing accurate Switchboard oracle metrics that correct DefiLlama's incomplete representation of market presence.

**MVP Delivers:**
- Complete Switchboard protocol capture from DefiLlama APIs
- TVS aggregation by chain and category with derived metrics
- Historical tracking with all snapshots retained
- JSON output files for dashboard consumption
- CLI with daemon and run-once modes

**56 Functional Requirements** covering API integration, data processing, incremental updates, output generation, and CLI operation.

**22 Non-Functional Requirements** covering performance, reliability, integration, maintainability, and operations.

---

_This PRD captures the requirements for defillama-extract - a Go service that extracts and aggregates Switchboard oracle metrics to provide accurate, authoritative data for stakeholders and potential integrators._

_Created through collaborative discovery between BMad and AI facilitator._
