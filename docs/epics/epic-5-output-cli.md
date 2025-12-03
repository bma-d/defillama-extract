# Epic 5: Output & CLI

**Goal:** Implement JSON output generation and complete CLI with daemon mode, graceful shutdown, and all operational features.

**User Value:** After this epic, the complete working tool produces JSON files ready for dashboard consumption, runs as a daemon with 2-hour intervals, and handles all operational scenarios gracefully.

**FRs Covered:** FR35, FR36, FR37, FR38, FR39, FR40, FR41, FR42, FR43, FR44, FR45, FR46, FR47, FR48, FR56

> **MANDATORY:** A Tech Spec MUST be drafted before creating stories for this epic. This requirement was established in the Epic 2+3 retrospective (2025-11-30) after identifying that missing tech specs made AC validation and review more difficult.

> **MANDATORY:** Each story MUST include a **Smoke Test Guide** in Dev Notes (or explicitly mark "Smoke test: N/A" for internal-only functions). Build/test/lint alone do not verify runtime behavior. This requirement was established in the Epic 2+3 retrospective (2025-11-30).

> **Note:** Stories consolidated from 10 to 3 via course correction (2025-11-30). Original fragmentation created unnecessary boundaries between tightly-coupled functionality.

---

## Story 5.1: Implement Output File Generation

As a **developer**,
I want **all output JSON files generated with atomic writes**,
So that **dashboards have reliable, complete data in multiple formats**.

**Acceptance Criteria:**

**AC1: Full Output JSON**
**Given** aggregation results and historical snapshots
**When** `GenerateFullOutput(result, history, config)` is called
**Then** a `FullOutput` struct is created with:
  - `version`: "1.0.0"
  - `oracle`: name, website, documentation URL from config
  - `metadata`: last_updated, data_source, update_frequency, extractor_version
  - `summary`: total_value_secured, total_protocols, active_chains, categories
  - `metrics`: current_tvs, change_24h, change_7d, change_30d, growth metrics
  - `breakdown`: by_chain array, by_category array
  - `protocols`: ranked protocol list with all metadata
  - `historical`: complete snapshot history
**And** JSON is human-readable with 2-space indentation
**And** file is written to `{output_dir}/switchboard-oracle-data.json`

**AC2: Minified Output JSON**
**Given** the same `FullOutput` data
**When** minified output is generated
**Then** JSON is serialized without whitespace
**And** file is written to `{output_dir}/switchboard-oracle-data.min.json`

**AC3: Summary Output JSON**
**Given** aggregation results (no history needed)
**When** `GenerateSummaryOutput(result, config)` is called
**Then** `SummaryOutput` struct contains current snapshot only:
  - `version`, `oracle`, `metadata`, `summary`, `metrics`, `breakdown`
  - `top_protocols`: top 10 by TVL
  - NO `historical` array
**And** file is written to `{output_dir}/switchboard-summary.json`

**AC4: Atomic File Writes**
**Given** output data to write
**When** `WriteJSON(path, data)` is called
**Then** data is written to temp file first (`{path}.tmp`)
**And** temp file is renamed to target path atomically
**And** directory is created if missing (`os.MkdirAll`)

**Given** write failure mid-operation
**When** error occurs
**Then** temp file is cleaned up, original preserved, error returned

**Given** all outputs ready
**When** `WriteAllOutputs(full, minified, summary)` is called
**Then** all three files are written atomically

**Prerequisites:** Story 3.7, Story 4.7

**Technical Notes:**
- Package: `internal/storage/writer.go`
- Structs: `FullOutput`, `SummaryOutput` in `internal/models/output.go`
- Use `json.MarshalIndent` for full, `json.Marshal` for minified
- Use `os.CreateTemp()` in same dir for atomic rename guarantee
- Reference: FR35, FR36, FR37, FR38, FR39, FR40, FR41

**Smoke Test Guide:**
1. Run extraction with valid data
2. Verify all 3 files exist in output dir
3. Verify full JSON is formatted, minified is compact
4. Verify summary has no historical array
5. Kill process mid-write, verify no corrupt files remain

---

## Story 5.2: Implement CLI and Single-Run Mode

As an **operator**,
I want **command-line flags and single extraction mode with proper logging**,
So that **I can run manual or cron-scheduled extractions**.

**Acceptance Criteria:**

**AC1: CLI Flag Parsing**
**Given** CLI invocation
**When** application starts
**Then** the following flags are supported:
  - `--once`: single extraction, then exit
  - `--config /path/to/config.yaml`: custom config path
  - `--dry-run`: fetch and process but don't write files
  - `--version`: print version and exit

**Given** `--version` flag
**When** application starts
**Then** prints "defillama-extract v1.0.0" and exits with code 0

**Given** no flags
**When** application starts
**Then** daemon mode is activated (Story 5.3)

**AC2: Single Extraction Mode**
**Given** `--once` flag is set
**When** application runs
**Then** one complete extraction cycle executes:
  1. Load config
  2. Load state
  3. Fetch API data
  4. Check if new data (skip if not)
  5. Aggregate data
  6. Write outputs (unless --dry-run)
  7. Save state
  8. Exit

**Given** successful extraction in `--once` mode
**Then** exit code is 0

**Given** extraction failure in `--once` mode
**Then** exit code is 1 with error logged

**Given** `--once` with no new data available
**Then** exit code is 0, log: "no new data, skipping extraction"

**Given** `--dry-run` flag
**When** extraction completes
**Then** data is fetched and processed but NOT written
**And** log: "dry-run mode, skipping file writes"

**AC3: Extraction Cycle Logging**
**Given** extraction cycle starts
**Then** info log: "extraction started" with timestamp

**Given** extraction completes successfully
**Then** info log: "extraction completed" with:
  - `duration_ms`, `protocol_count`, `tvs`, `chains`

**Given** extraction is skipped
**Then** info log: "extraction skipped, no new data" with `last_updated`

**Given** extraction fails
**Then** error log: "extraction failed" with `error`, `duration_ms`

**Prerequisites:** Story 1.2, Story 1.4, Story 3.7, Story 4.7, Story 5.1

**Technical Notes:**
- Package: `cmd/extractor/main.go`
- Use `flag` package from standard library
- Store flags in `CLIOptions` struct
- Create `runOnce(ctx, cfg)` function
- Use `slog` with structured attributes for all logs
- Track timing with `time.Now()` at start
- Reference: FR42, FR44, FR45, FR46, FR48, FR56

**Smoke Test Guide:**
1. `./extractor --version` → prints version, exits 0
2. `./extractor --once --config ./config.yaml` → runs once, exits
3. `./extractor --once --dry-run` → fetches but no files written
4. Check logs contain duration_ms, protocol_count, tvs
5. Run with stale data → logs "skipping", exits 0

---

## Story 5.3: Implement Daemon Mode and Complete Main Entry Point

As an **operator**,
I want **continuous daemon operation with graceful shutdown**,
So that **data stays automatically updated in production**.

**Acceptance Criteria:**

**AC1: Daemon Mode with Scheduler**
**Given** daemon mode (no `--once` flag) with `scheduler.interval: 2h`
**When** application starts
**Then** extraction runs on schedule every 2 hours
**And** log: "daemon started, interval: 2h"

**Given** `scheduler.start_immediately: true`
**Then** first extraction runs immediately, subsequent follow interval

**Given** `scheduler.start_immediately: false`
**Then** first extraction waits for interval

**Given** extraction cycle completes in daemon mode
**Then** log: "next extraction at {timestamp}"

**Given** extraction fails in daemon mode
**Then** error is logged, daemon continues running, next extraction scheduled normally

**AC2: Graceful Shutdown**
**Given** daemon running an extraction
**When** SIGINT/SIGTERM received
**Then** current extraction completes
**And** log: "shutdown signal received, finishing current extraction"
**And** exits cleanly with code 0

**Given** daemon waiting for next extraction
**When** SIGINT/SIGTERM received
**Then** wait is cancelled immediately, exits with code 0

**Given** `--once` mode with extraction in progress
**When** SIGINT received
**Then** extraction cancelled via context, partial results NOT written, exit code 1

**AC3: Complete Main Entry Point**
**Given** application starts
**When** `main()` executes
**Then** sequence is:
  1. Parse CLI flags
  2. Handle `--version` (print and exit)
  3. Load configuration from file
  4. Apply environment overrides
  5. Validate configuration
  6. Initialize logger
  7. Create components (API client, Aggregator, StateManager, Writer)
  8. Set up signal handling
  9. Run in appropriate mode (once vs daemon)
  10. Exit with appropriate code

**Given** any initialization failure
**Then** error logged, exit code 1

**Prerequisites:** Story 5.1, Story 5.2

**Technical Notes:**
- Package: `cmd/extractor/main.go`
- Use `time.Ticker` for scheduling
- Use `signal.NotifyContext()` for clean context cancellation
- Listen for `os.Interrupt` and `syscall.SIGTERM`
- Wire: config → logger → client → aggregator → stateManager → writer → runner
- Reference: FR43, FR47

**Smoke Test Guide:**
1. Run without `--once` → daemon starts, logs interval
2. Wait for scheduled extraction → runs and logs next time
3. Send SIGINT during extraction → completes, then exits 0
4. Send SIGINT while waiting → exits immediately with 0
5. Cause init failure (bad config) → exits 1 with error

---

## Story 5.4: Extract Historical Chart Data for Graphing

As a **dashboard consumer**,
I want **historical TVS chart data extracted from DefiLlama**,
So that **I can render time-series graphs showing Switchboard's TVS over time**.

**Acceptance Criteria:**

**AC1: Extract Chart Data from API**
**Given** the `/oracles` API response contains `chart` field
**When** extraction runs
**Then** all Switchboard entries from `chart[timestamp]["Switchboard"]` are extracted
**And** each entry includes: timestamp, date, tvl (TVS), borrowed, staking

**AC2: Chart History in Output**
**Given** extracted chart data
**When** output JSON is generated
**Then** a `chart_history` array is included with entries:
  - `timestamp`: Unix timestamp (int64)
  - `date`: ISO date string (YYYY-MM-DD)
  - `tvs`: Total value secured (float64)
  - `borrowed`: Borrowed value (float64, optional)
  - `staking`: Staking value (float64, optional)
**And** entries are sorted by timestamp ascending
**And** array is included in both full output and summary output

**AC3: Chart Data Date Range**
**Given** chart history is generated
**When** output is written
**Then** all available historical data points are included (full history from API)
**And** chart_history contains 1000+ entries (DefiLlama has 4+ years of data)

**AC4: Output Schema Update**
**Given** updated output schema
**When** JSON is generated
**Then** `chart_history` array appears at top level alongside `historical`
**And** `historical` continues to contain extractor-run snapshots (protocol-level detail)
**And** `chart_history` contains API-sourced daily TVS data (for graphing)

**Prerequisites:** Story 5.1, Story 5.3

**Technical Notes:**
- The `chart` field structure is: `map[timestamp_string]map[oracle_name]ChartEntry`
- `ChartEntry` has: `tvl` (float64), `borrowed` (float64), `staking` (float64)
- Filter for `oracleName` (e.g., "Switchboard") from chart data
- Parse timestamp strings to int64 for sorting
- Package: `internal/aggregator/chart.go` (new file)
- Update: `internal/models/output.go` to add `ChartHistory` field
- Update: `internal/storage/writer.go` to include chart data in output generation
- Reference: Seed doc `3-data-sources-api-specifications.md` line 28

**Smoke Test Guide:**
1. Run extraction with `--once`
2. Verify `chart_history` array exists in output JSON
3. Verify array has 1000+ entries
4. Verify first entry date is ~2021-11-29 (when Switchboard data starts)
5. Verify last entry matches current date
6. Verify data can be used to plot a chart (timestamps sequential, values reasonable)

---
