# Functional Requirements Inventory

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
