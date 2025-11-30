# Epic 5: Output & CLI

**Goal:** Implement JSON output generation and complete CLI with daemon mode, graceful shutdown, and all operational features.

**User Value:** After this epic, the complete working tool produces JSON files ready for dashboard consumption, runs as a daemon with 2-hour intervals, and handles all operational scenarios gracefully.

**FRs Covered:** FR35, FR36, FR37, FR38, FR39, FR40, FR41, FR42, FR43, FR44, FR45, FR46, FR47, FR48, FR56

> **MANDATORY:** A Tech Spec MUST be drafted before creating stories for this epic. This requirement was established in the Epic 2+3 retrospective (2025-11-30) after identifying that missing tech specs made AC validation and review more difficult.

> **MANDATORY:** Each story MUST include a **Smoke Test Guide** in Dev Notes (or explicitly mark "Smoke test: N/A" for internal-only functions). Build/test/lint alone do not verify runtime behavior. This requirement was established in the Epic 2+3 retrospective (2025-11-30).

---

## Story 5.1: Implement Full Output JSON Generation

As a **developer**,
I want **complete output JSON generated with all data and history**,
So that **dashboards have all the information they need**.

**Acceptance Criteria:**

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

**Given** full output struct
**When** serialized to JSON
**Then** output is human-readable with 2-space indentation
**And** file is written to `{output_dir}/switchboard-oracle-data.json`

**Prerequisites:** Story 3.7, Story 4.8

**Technical Notes:**
- Package: `internal/storage/writer.go`
- Use `json.MarshalIndent(data, "", "  ")` for formatting
- `FullOutput` struct in `internal/models/output.go`
- Reference: FR35, FR40, FR41, data-architecture.md

---

## Story 5.2: Implement Minified Output JSON Generation

As a **developer**,
I want **a minified version of the output JSON**,
So that **file transfer size is minimized**.

**Acceptance Criteria:**

**Given** the same `FullOutput` data
**When** minified output is generated
**Then** JSON is serialized without whitespace or indentation
**And** file is written to `{output_dir}/switchboard-oracle-data.min.json`
**And** content is identical to full output (just formatting differs)

**Given** full output is 500KB with formatting
**When** minified output is generated
**Then** minified size is significantly smaller (typically 60-70% of formatted size)

**Prerequisites:** Story 5.1

**Technical Notes:**
- Package: `internal/storage/writer.go`
- Use `json.Marshal()` (no indent) for minified
- Write to separate file, same directory
- Reference: FR36

---

## Story 5.3: Implement Summary Output JSON Generation

As a **developer**,
I want **a lightweight summary JSON with current snapshot only**,
So that **quick reads don't require loading full history**.

**Acceptance Criteria:**

**Given** aggregation results (no history needed)
**When** `GenerateSummaryOutput(result, config)` is called
**Then** a `SummaryOutput` struct is created with:
  - `version`: "1.0.0"
  - `oracle`: name, website, documentation URL
  - `metadata`: last_updated, data_source
  - `summary`: total_value_secured, total_protocols, active_chains, categories
  - `metrics`: current snapshot metrics only
  - `breakdown`: by_chain, by_category
  - `top_protocols`: top 10 protocols by TVL (subset)
  - NO `historical` array

**Given** summary output
**When** written to file
**Then** file is `{output_dir}/switchboard-summary.json`
**And** file size is much smaller than full output

**Prerequisites:** Story 5.1

**Technical Notes:**
- Package: `internal/storage/writer.go`
- `SummaryOutput` struct - subset of FullOutput fields
- Include top 10 protocols only to keep size small
- Reference: FR37

---

## Story 5.4: Implement Atomic File Writer

As a **developer**,
I want **all output files written atomically**,
So that **partial writes don't corrupt output files**.

**Acceptance Criteria:**

**Given** output data to write
**When** `WriteJSON(path string, data any)` is called
**Then** data is written to temp file first (`{path}.tmp`)
**And** temp file is renamed to target path
**And** operation is atomic (readers never see partial content)

**Given** output directory doesn't exist
**When** `WriteJSON` is called
**Then** directory is created with `os.MkdirAll()`
**And** file is written successfully

**Given** a write failure mid-operation
**When** error occurs
**Then** temp file is cleaned up
**And** original file (if exists) is preserved
**And** error is returned with context

**Given** multiple output files to write
**When** `WriteAllOutputs(full, minified, summary)` is called
**Then** all three files are written atomically
**And** if any write fails, error indicates which file failed

**Prerequisites:** Story 1.1

**Technical Notes:**
- Package: `internal/storage/writer.go`
- `Writer` struct with `NewWriter(outputDir)` constructor
- Use `os.CreateTemp()` in same directory for atomic rename guarantee
- Defer cleanup of temp files on error
- Reference: FR38, FR39, implementation-patterns.md

---

## Story 5.5: Implement CLI Flag Parsing

As an **operator**,
I want **command-line flags for controlling execution**,
So that **I can run the tool in different modes**.

**Acceptance Criteria:**

**Given** CLI invocation with `--once`
**When** application starts
**Then** single extraction is performed and application exits

**Given** CLI invocation with `--config /path/to/config.yaml`
**When** application starts
**Then** configuration is loaded from specified path

**Given** CLI invocation with `--dry-run`
**When** extraction completes
**Then** data is fetched and processed but NOT written to files
**And** log indicates "dry-run mode, skipping file writes"

**Given** CLI invocation with `--version`
**When** application starts
**Then** version string is printed (e.g., "defillama-extract v1.0.0")
**And** application exits with code 0

**Given** CLI invocation with no flags
**When** application starts
**Then** daemon mode is activated with default config path

**Prerequisites:** Story 1.2

**Technical Notes:**
- Package: `cmd/extractor/main.go`
- Use `flag` package from standard library
- Flags: `--once`, `--config`, `--dry-run`, `--version`
- Store in `CLIOptions` struct
- Reference: FR42, FR44, FR45, FR46

---

## Story 5.6: Implement Single Extraction Mode

As an **operator**,
I want **to run a single extraction and exit**,
So that **I can use cron or manual runs for scheduling**.

**Acceptance Criteria:**

**Given** `--once` flag is set
**When** application runs
**Then** one complete extraction cycle executes:
  1. Load config
  2. Load state
  3. Fetch API data
  4. Check if new data (skip if not)
  5. Aggregate data
  6. Write outputs (unless dry-run)
  7. Save state
  8. Exit

**Given** successful extraction in `--once` mode
**When** extraction completes
**Then** exit code is 0
**And** log: "extraction completed" with protocol_count, tvs, duration_ms

**Given** extraction failure in `--once` mode
**When** error occurs
**Then** exit code is 1
**And** error is logged with context

**Given** `--once` with no new data available
**When** skip logic triggers
**Then** exit code is 0 (not an error)
**And** log: "no new data, skipping extraction"

**Prerequisites:** Story 5.5, Story 3.7, Story 4.8, Story 5.4

**Technical Notes:**
- Package: `cmd/extractor/main.go`
- Create `runOnce(ctx, cfg)` function
- Wire together: API client → Aggregator → StateManager → Writer
- Reference: FR42, FR48

---

## Story 5.7: Implement Daemon Mode with Scheduler

As an **operator**,
I want **the service to run continuously with scheduled extractions**,
So that **data is automatically kept up to date**.

**Acceptance Criteria:**

**Given** daemon mode (no `--once` flag) with `scheduler.interval: 2h`
**When** application starts
**Then** extraction runs on schedule every 2 hours
**And** log: "daemon started, interval: 2h"

**Given** `scheduler.start_immediately: true`
**When** daemon starts
**Then** first extraction runs immediately
**And** subsequent extractions follow interval

**Given** `scheduler.start_immediately: false`
**When** daemon starts
**Then** first extraction waits for interval
**And** log: "waiting for first scheduled extraction"

**Given** daemon is running
**When** extraction cycle completes
**Then** next extraction is scheduled
**And** log: "next extraction at {timestamp}"

**Given** extraction fails in daemon mode
**When** error occurs
**Then** error is logged
**And** daemon continues running (doesn't exit)
**And** next extraction is scheduled normally

**Prerequisites:** Story 5.6

**Technical Notes:**
- Package: `cmd/extractor/main.go`
- Use `time.Ticker` for scheduling
- Create `runDaemon(ctx, cfg)` function
- Handle errors gracefully - log and continue
- Reference: FR43

---

## Story 5.8: Implement Graceful Shutdown

As an **operator**,
I want **graceful shutdown on SIGINT/SIGTERM**,
So that **in-progress operations complete cleanly**.

**Acceptance Criteria:**

**Given** daemon is running an extraction cycle
**When** SIGINT (Ctrl+C) is received
**Then** current extraction is allowed to complete
**And** log: "shutdown signal received, finishing current extraction"
**And** after completion, daemon exits cleanly with code 0

**Given** daemon is waiting for next scheduled extraction
**When** SIGTERM is received
**Then** wait is cancelled immediately
**And** log: "shutdown signal received, exiting"
**And** daemon exits cleanly with code 0

**Given** `--once` mode with extraction in progress
**When** SIGINT is received
**Then** extraction is cancelled via context
**And** partial results are NOT written
**And** exit code is 1

**Given** shutdown signal
**When** processing
**Then** signal is handled only once (no duplicate handling)

**Prerequisites:** Story 5.7

**Technical Notes:**
- Package: `cmd/extractor/main.go`
- Use `signal.NotifyContext()` for clean context cancellation
- Listen for `os.Interrupt` and `syscall.SIGTERM`
- Pass cancellable context to all operations
- Reference: FR47

---

## Story 5.9: Implement Extraction Cycle Logging

As an **operator**,
I want **extraction cycles logged with key metrics**,
So that **I can monitor system health**.

**Acceptance Criteria:**

**Given** extraction cycle starts
**When** processing begins
**Then** info log: "extraction started" with timestamp

**Given** extraction completes successfully
**When** results are available
**Then** info log: "extraction completed" with:
  - `duration_ms`: total extraction time
  - `protocol_count`: number of protocols found
  - `tvs`: total value secured
  - `chains`: number of active chains

**Given** extraction is skipped (no new data)
**When** skip occurs
**Then** info log: "extraction skipped, no new data" with:
  - `last_updated`: timestamp of existing data

**Given** extraction fails
**When** error occurs
**Then** error log: "extraction failed" with:
  - `error`: error message
  - `duration_ms`: time until failure

**Prerequisites:** Story 1.4, Story 5.6

**Technical Notes:**
- Use `slog` with structured attributes
- Track start time with `time.Now()`, calculate duration at end
- Include all relevant metrics in log attributes
- Reference: FR48, FR56

---

## Story 5.10: Build Complete Main Entry Point

As a **developer**,
I want **a complete main.go that wires everything together**,
So that **the application is fully functional**.

**Acceptance Criteria:**

**Given** application starts
**When** `main()` executes
**Then** the following sequence occurs:
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
**When** error occurs during startup
**Then** error is logged
**And** application exits with code 1

**Given** successful run
**When** application completes/terminates
**Then** exit code reflects success (0) or failure (1)

**Prerequisites:** All previous stories in Epic 5

**Technical Notes:**
- Package: `cmd/extractor/main.go`
- Use dependency injection pattern per architecture
- Wire: config → logger → client → aggregator → stateManager → writer → runner
- Reference: 17-complete-maingo-implementation.md

---
