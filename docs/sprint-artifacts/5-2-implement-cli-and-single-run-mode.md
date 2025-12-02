# Story 5.2: Implement CLI and Single-Run Mode

Status: done

## Story

As an **operator**,
I want **command-line flags and single extraction mode with proper logging**,
so that **I can run manual or cron-scheduled extractions**.

## Acceptance Criteria

Source: [Source: docs/sprint-artifacts/tech-spec-epic-5.md#Story-5.2]

**AC1: CLI Flag Parsing (--once)**
**Given** CLI invocation with `--once` flag
**When** application starts
**Then** single extraction mode is activated
**And** application exits after one complete extraction cycle

**AC2: CLI Flag Parsing (--config)**
**Given** CLI invocation with `--config /path/to/config.yaml`
**When** application starts
**Then** configuration is loaded from the specified path instead of default

**AC3: CLI Flag Parsing (--dry-run)**
**Given** CLI invocation with `--dry-run` flag
**When** extraction completes
**Then** data is fetched and processed but NOT written to files
**And** log message: "dry-run mode, skipping file writes"

**AC4: CLI Flag Parsing (--version)**
**Given** CLI invocation with `--version` flag
**When** application starts
**Then** prints "defillama-extract v1.0.0" and exits with code 0

**AC5: Default Mode (No Flags)**
**Given** no flags provided
**When** application starts
**Then** daemon mode is activated (handled by Story 5.3)

**AC6: Single Mode Sequence**
**Given** `--once` flag is set
**When** application runs
**Then** one complete extraction cycle executes in order:
  1. Load config
  2. Load state
  3. Fetch API data (oracles + protocols in parallel)
  4. Check if new data available (compare timestamps)
  5. If new data: Aggregate data
  6. Write outputs (unless --dry-run)
  7. Save state
  8. Exit

**AC7: Single Mode Success Exit**
**Given** successful extraction in `--once` mode
**When** extraction completes
**Then** exit code is 0

**AC8: Single Mode Failure Exit**
**Given** extraction failure in `--once` mode
**When** error occurs
**Then** exit code is 1 with error logged

**AC9: No New Data Handling**
**Given** `--once` mode with no new data available
**When** timestamp comparison shows stale data
**Then** exit code is 0
**And** log: "no new data, skipping extraction"

**AC10: Dry-Run No Writes**
**Given** `--dry-run` flag
**When** extraction completes successfully
**Then** files are NOT written
**And** log: "dry-run mode, skipping file writes"

**AC11: Log Extraction Started**
**Given** extraction cycle begins
**When** processing starts
**Then** INFO log with timestamp: "extraction started"

**AC12: Log Extraction Completed**
**Given** extraction completes successfully
**When** all operations finish
**Then** INFO log with attributes:
  - `duration_ms`: extraction duration in milliseconds
  - `protocol_count`: number of protocols processed
  - `tvs`: total value secured
  - `chains`: number of active chains

**AC13: Log Extraction Skipped**
**Given** extraction is skipped (no new data)
**When** skip decision is made
**Then** INFO log with `last_updated` timestamp

**AC14: Log Extraction Failed**
**Given** extraction fails
**When** error occurs
**Then** ERROR log with:
  - `error`: error message
  - `duration_ms`: time elapsed before failure

## Tasks / Subtasks

- [x] Task 1: Define CLIOptions struct (AC: 1-5)
  - [x] 1.1: Create `CLIOptions` struct in `cmd/extractor/main.go` with fields: `Once bool`, `ConfigPath string`, `DryRun bool`, `Version bool`
  - [x] 1.2: Add `ParseCLI() CLIOptions` function using `flag` package
  - [x] 1.3: Register flags: `--once`, `--config` (default "config.yaml"), `--dry-run`, `--version`
  - [x] 1.4: Call `flag.Parse()` and populate struct

- [x] Task 2: Implement --version handling (AC: 4)
  - [x] 2.1: Define `const Version = "1.0.0"` at package level
  - [x] 2.2: In `main()`, after parsing flags, check `opts.Version`
  - [x] 2.3: If true: print "defillama-extract v1.0.0" to stdout and `os.Exit(0)`

- [x] Task 3: Implement RunOnce function (AC: 6-10)
  - [x] 3.1: Create `RunOnce(ctx context.Context, cfg *config.Config, opts CLIOptions) error`
  - [x] 3.2: Create API client using `api.NewClient(cfg)`
  - [x] 3.3: Create aggregator using `aggregator.New(cfg.Oracle.Name)`
  - [x] 3.4: Create state manager using `storage.NewStateManager(cfg)`
  - [x] 3.5: Load state via `stateManager.LoadState()`
  - [x] 3.6: Fetch oracles and protocols in parallel using errgroup (reuse Epic 2 pattern)
  - [x] 3.7: Check for new data: compare API timestamp with `state.LastUpdated`
  - [x] 3.8: If no new data: log "no new data, skipping extraction" and return nil
  - [x] 3.9: Run aggregation via `aggregator.Aggregate(oracleData, protocols)`
  - [x] 3.10: Load history via `stateManager.LoadHistory()`
  - [x] 3.11: Generate outputs using `storage.GenerateFullOutput()` and `storage.GenerateSummaryOutput()`
  - [x] 3.12: If NOT dry-run: write outputs via `storage.WriteAllOutputs()`
  - [x] 3.13: If dry-run: log "dry-run mode, skipping file writes"
  - [x] 3.14: Save state via `stateManager.SaveState()`
  - [x] 3.15: Return nil on success, error on failure

- [x] Task 4: Implement extraction logging (AC: 11-14)
  - [x] 4.1: Record start time with `time.Now()` at extraction start
  - [x] 4.2: Log INFO "extraction started" with timestamp at cycle start
  - [x] 4.3: On success: calculate duration, log INFO "extraction completed" with `duration_ms`, `protocol_count`, `tvs`, `chains`
  - [x] 4.4: On skip: log INFO "extraction skipped, no new data" with `last_updated`
  - [x] 4.5: On error: log ERROR "extraction failed" with `error`, `duration_ms`

- [x] Task 5: Wire main() for single-run mode (AC: 7, 8)
  - [x] 5.1: After version check, load config using `config.Load(opts.ConfigPath)`
  - [x] 5.2: Apply environment overrides using existing `config.ApplyEnv()`
  - [x] 5.3: Validate config using `config.Validate()`
  - [x] 5.4: Initialize slog logger based on config.Logging settings
  - [x] 5.5: If `opts.Once`: call `RunOnce(ctx, cfg, opts)`
  - [x] 5.6: If RunOnce returns error: log error, `os.Exit(1)`
  - [x] 5.7: If RunOnce returns nil: `os.Exit(0)`
  - [x] 5.8: If NOT `opts.Once`: placeholder for daemon mode (Story 5.3)

- [x] Task 6: Write unit tests (AC: all)
  - [x] 6.1: Test `ParseCLI` parses all flags correctly
  - [x] 6.2: Test `ParseCLI` default values (ConfigPath="config.yaml", others=false)
  - [x] 6.3: Test --version output format and exit
  - [x] 6.4: Test RunOnce success path returns nil
  - [x] 6.5: Test RunOnce with no new data returns nil (skip path)
  - [x] 6.6: Test RunOnce dry-run mode skips file writes
  - [x] 6.7: Test RunOnce failure returns error
  - [x] 6.8: Test log messages contain required attributes (duration_ms, protocol_count, etc.)
  - [x] 6.9: Test default (no-flag) execution path drops into daemon stub (AC: 5)

- [x] Task 7: Verification (AC: all)
  - [x] 7.1: Run `go build ./...` and verify success
  - [x] 7.2: Run `go test ./cmd/extractor/...` and verify all pass
  - [x] 7.3: Run `make lint` and verify no errors
- [x] 7.4: Manual smoke test: `./extractor --version`
- [x] 7.5: Manual smoke test: `./extractor --once --dry-run`

### Review Follow-ups (AI)

- [x] [AI-Review][High] Prevent `--dry-run` executions from saving new state/history so subsequent real runs do not skip writing outputs (AC10).
- [x] [AI-Review][Med] Surface CLI flag parse errors instead of silently ignoring unknown flags so valid options (e.g., `--once`) still apply per ADR-001.
- [x] [AI-Review][Med] Add regression tests for the DefiLlama `protocols` envelope to lock the new `protocolList` decoder.
- [x] [AI-Review][High] Implement default daemon execution (AC5) so running without flags starts the scheduler or fails loudly instead of returning immediately (`cmd/extractor/main.go:187-190`).
- [x] [AI-Review][Med] Keep daemon mode running after the `start_immediately` boot cycle fails; log the error and wait for the next interval instead of returning from `runDaemonWithDeps` (`cmd/extractor/main.go:222-225`, `docs/sprint-artifacts/tech-spec-epic-5.md:219-289`).

## Dev Notes

### Technical Guidance

- **Files to Create/Modify:**
  - MODIFY: `cmd/extractor/main.go` - CLI parsing, main entry point, RunOnce
  - NEW: `cmd/extractor/main_test.go` - Unit tests for CLI and RunOnce

- **ADR Compliance:**
  - ADR-001: Use `flag` package (stdlib) for CLI parsing, NOT Cobra/Viper
  - ADR-004: Use `log/slog` for all logging

- **CLI Flag Pattern** per tech spec:
  ```go
  type CLIOptions struct {
      Once       bool   // --once flag
      ConfigPath string // --config flag
      DryRun     bool   // --dry-run flag
      Version    bool   // --version flag
  }

  func ParseCLI() CLIOptions {
      opts := CLIOptions{}
      flag.BoolVar(&opts.Once, "once", false, "Run single extraction and exit")
      flag.StringVar(&opts.ConfigPath, "config", "config.yaml", "Path to config file")
      flag.BoolVar(&opts.DryRun, "dry-run", false, "Fetch and process but don't write files")
      flag.BoolVar(&opts.Version, "version", false, "Print version and exit")
      flag.Parse()
      return opts
  }
  ```

- **Extraction Logging Pattern:**
  ```go
  func logExtractionCompleted(logger *slog.Logger, start time.Time, result *aggregator.AggregationResult) {
      logger.Info("extraction completed",
          "duration_ms", time.Since(start).Milliseconds(),
          "protocol_count", len(result.Protocols),
          "tvs", result.Summary.TotalValueSecured,
          "chains", len(result.Breakdown.ByChain),
      )
  }
  ```

- **FR Coverage:** FR42, FR44, FR45, FR46, FR48, FR56

### Learnings from Previous Story

**From Story 5-1-implement-output-file-generation (Status: done)**

- **Output Generation Functions Available**:
  - `storage.GenerateFullOutput(result, history, cfg)` - Creates full output struct
  - `storage.GenerateSummaryOutput(result, cfg)` - Creates summary output struct
  - `storage.WriteAllOutputs(outputDir, cfg, full, summary)` - Writes all three JSON files atomically

- **Model Structs Available** at `internal/models/output.go`:
  - `FullOutput`, `SummaryOutput`, `OracleInfo`, `OutputMetadata`

- **Config Integration**:
  - Output filenames honor `cfg.Output.FullFile/MinFile/SummaryFile`
  - Metadata update_frequency uses `cfg.Scheduler.Interval`

- **Testing Pattern**: Table-driven tests with `slog.Default()` fallback for nil logger

- **Files to Reuse**:
  - `internal/storage/writer.go` - WriteAllOutputs, GenerateFullOutput, GenerateSummaryOutput
  - `internal/storage/state.go` - StateManager with LoadState, SaveState, LoadHistory

[Source: docs/sprint-artifacts/5-1-implement-output-file-generation.md#Dev-Agent-Record]

### Project Structure Notes

- **Package Location**: `cmd/extractor/` for main entry point per Go standard layout
- **Import Dependencies**:
  - `internal/config` - Config loading and validation
  - `internal/api` - API client for fetching
  - `internal/aggregator` - Data aggregation
  - `internal/storage` - State manager and output writing
  - `internal/models` - Output model structs

### Testing Standards

- Follow table-driven test pattern per [Source: docs/architecture/testing-strategy.md#Test-Organization]
- Use test flags for CLI parsing tests
- Mock external dependencies (API client) for RunOnce tests
- Assert log output contains required fields

### Smoke Test Guide

1. `./extractor --version` prints "defillama-extract v1.0.0", exits 0
2. `./extractor --once --config ./config.yaml` runs once, exits
3. `./extractor --once --dry-run` fetches but no files written
4. Check logs contain `duration_ms`, `protocol_count`, `tvs`
5. Run with stale data logs "skipping", exits 0

### References

- [Source: docs/prd.md#FR42-FR48] - CLI operation requirements
- [Source: docs/prd.md#FR56] - Extraction cycle logging
- [Source: docs/epics/epic-5-output-cli.md#story-52] - Story definition
- [Source: docs/sprint-artifacts/tech-spec-epic-5.md#Story-5.2] - Authoritative acceptance criteria
- [Source: docs/sprint-artifacts/tech-spec-epic-5.md#CLI-Options] - CLIOptions struct definition
- [Source: docs/sprint-artifacts/tech-spec-epic-5.md#Workflows-and-Sequencing] - Single extraction cycle sequence
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-001] - Use stdlib flag package
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-004] - Use slog for logging
- [Source: docs/sprint-artifacts/5-1-implement-output-file-generation.md#Dev-Agent-Record] - Previous story reference

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/5-2-implement-cli-and-single-run-mode.context.xml

### Agent Model Used

- Bob (GPT-5)

### Debug Log References

- Validation: docs/sprint-artifacts/validation-report-story-5-2-2025-12-01T19-43-28Z.md

### Debug Log

- 2025-12-02T04:44Z: Implemented daemon resiliency for start_immediately failures (log and continue to next tick); added unit coverage; go test/build/lint green. go run --once --dry-run --config configs/config.yaml failed due to upstream timeout on https://api.llama.fi/oracles (context deadline exceeded).
- 2025-12-02T04:44Z: Implemented daemon resiliency for start_immediately failures (log and continue to next tick); added unit coverage; go test/build/lint green. go run --once --dry-run --config configs/config.yaml failed due to upstream timeout on https://api.llama.fi/oracles (context deadline exceeded).
- 2025-12-02T05:32Z: Retried dry-run with API_TIMEOUT=120s; protocols/oracles completed, no new data so extraction skipped; command exited 0.

- 2025-12-01T21:18Z: Planned implementation (CLI parse, RunOnce flow, tests, protocol decode fix) and set sprint-status to in-progress.
- 2025-12-01T21:40Z: Added CLIOptions + ParseCLI, RunOnce with structured logging, dependency injection for tests, daemon stub in run().
- 2025-12-01T21:55Z: Added protocol response envelope decoder to handle `{protocols:[...]}` shape; resolved real API decode failure.
- 2025-12-01T22:00Z: Tests/build/lint green (`go test ./...`, `go build ./...`, `make lint`). Manual smoke: `go run ./cmd/extractor --version` OK, `--once --dry-run --config configs/config.yaml` succeeds; state saved, outputs skipped per dry-run.
- 2025-12-01T23:10Z: Resuming after review (Blocked). Plan: fix dry-run persistence, surface flag parse errors, add protocol envelope test; sprint-status set to in-progress.
- 2025-12-01T23:35Z: Implemented fixes (dry-run skips state writes, flag parse errors surfaced, protocol envelope test added); `go test ./...` passing.
- 2025-12-01T23:58Z: Implemented daemon default path with signal-aware context + ticker scheduler; added daemon unit tests; `go test ./...` passing; preparing story for review.

### Completion Notes List

- Implemented CLI flag parsing (`--once`, `--config`, `--dry-run`, `--version`) with Version constant and daemon stub for Story 5.3.
- Added RunOnce orchestration using api client, aggregator, state manager, history, outputs, dry-run guard, structured logs for start/skip/success/failure.
- Hardened protocol fetch decoding for DefiLlama `protocols2` envelope; prevents runtime JSON errors.
- Added unit tests for CLI parsing, RunOnce success/skip/dry-run/error, daemon stub, version output; all pass.
- Build/test/lint executed; manual smoke tests for `--version` and `--once --dry-run` completed (dry-run uses configs/config.yaml, no writes).
- Addressed review findings: dry-run now skips state persistence; CLI flag parse errors surfaced with usage and exit code 2; added protocol envelope regression test; `go test ./...` now green post-fix.
- Default (no-flag) execution now enters daemon scheduler with signal-aware shutdown and tick-driven RunOnce; daemon unit tests cover tick execution and startup failure; `go test ./...` (2025-12-01) passing.
- Added daemon resiliency: start_immediately failure now logs and waits for next interval instead of exiting; unit test added. Validation rerun: gofmt, go test ./..., go build ./..., make lint pass. Manual `go run ./cmd/extractor --once --dry-run --config configs/config.yaml` failed due to upstream timeout (api.llama.fi/oracles); documented for follow-up if persists.
- Added daemon resiliency: start_immediately failure now logs and waits for next interval instead of exiting; unit test added. Validation rerun: gofmt, go test ./..., go build ./..., make lint pass. Manual `go run ./cmd/extractor --once --dry-run --config configs/config.yaml` failed due to upstream timeout (api.llama.fi/oracles); documented for follow-up if persists. Retried with `API_TIMEOUT=120s` and command succeeded (no new data, skipped extraction).

### File List

- docs/sprint-artifacts/5-2-implement-cli-and-single-run-mode.md (updated)
- docs/sprint-artifacts/validation-report-story-5-2-2025-12-01T19-43-28Z.md
- docs/sprint-artifacts/validation-report-story-5-2-2025-12-02T05-39-00Z.md
- docs/sprint-artifacts/sprint-status.yaml
- cmd/extractor/main.go
- cmd/extractor/main_test.go
- internal/api/client.go
- internal/api/responses.go
- internal/api/responses_test.go

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-12-02 | Amelia | Senior Developer Review notes appended; outcome Approve. |
| 2025-12-02 | Amelia | Implemented daemon start_immediately resiliency (log and continue to next tick), refreshed tests. |
| 2025-12-02 | Amelia | Senior Developer Review notes appended; outcome Changes Requested (daemon start_immediately resiliency follow-up). |
| 2025-12-01 | Amelia | Default no-flag path starts daemon loop with signal-aware shutdown and scheduler ticker; added daemon unit tests; `go test ./...` passing; status → review. |
| 2025-12-01 | Amelia | Senior Developer Review notes appended; outcome Changes Requested |
| 2025-12-01 | Amelia | Addressed review items: dry-run state guard, CLI flag error surfacing, protocol envelope regression test; `go test ./...` passing |
| 2025-12-01 | Amelia | Implemented CLI + RunOnce, protocol decode fix, tests/build/lint, smoke tested; marked story ready for review |
| 2025-12-01 | SM Agent (Bob) | Initial story draft created from epic-5 and tech-spec-epic-5.md |
| 2025-12-01 | SM Agent (Bob) | Initialized Dev Agent Record after validation |
| 2025-12-01 | SM Agent (Bob) | Generated story context XML and marked ready-for-dev |
| 2025-12-01 | Amelia | Senior Developer Review notes appended; outcome Blocked |

## Senior Developer Review (AI)

**Reviewer:** BMad  
**Date:** 2025-12-01  
**Outcome:** Blocked – high severity dry-run regression and CLI UX gaps.

### Summary
1. `--dry-run` currently mutates `state.json`, so subsequent real runs will skip writing outputs until upstream timestamps advance.
2. CLI silently ignores unknown flags, so typos prevent valid options from being parsed; default invocation still does nothing because daemon mode is not wired.
3. The new protocol envelope decoder has no regression tests, leaving the hotfix fragile.

### Key Findings
**High**
1. `cmd/extractor/main.go:135-147` saves new state even when `opts.DryRun` is true. Combined with `internal/storage/state.go:82-135`, the next real run sees matching timestamps and exits without updating files (violates AC10 / FR45).

**Medium**
1. `ParseCLI` discards parse errors (`cmd/extractor/main.go:29-41`), so any unknown flag stops parsing and other valid flags are ignored with no message (AC1–AC4, ADR-001).
2. Default (no flag) path merely logs "daemon mode not implemented" and exits (`cmd/extractor/main.go:176-179`), so AC5 remains unmet pending Story 5.3.
3. The new `protocolList` decoder lacks tests for the `{"protocols": [...]}` envelope (`internal/api/responses.go:30-57` vs. `internal/api/responses_test.go:1-76`).

**Low**
- None.

### Acceptance Criteria Coverage
12 / 14 ACs implemented.

| AC | Description | Status | Evidence |
|----|-------------|--------|----------|
| AC1 | `--once` flag enables single-run | Pass | `cmd/extractor/main.go:35-41`, `cmd/extractor/main.go:176-186`; `cmd/extractor/main_test.go:122-169` |
| AC2 | `--config` overrides path | Pass | `cmd/extractor/main.go:36`, `cmd/extractor/main.go:167-170`; `cmd/extractor/main_test.go:99-107` |
| AC3 | `--dry-run` skips writes/logs message | Pass | `cmd/extractor/main.go:37`, `cmd/extractor/main.go:135-142`; `cmd/extractor/main_test.go:206-240` |
| AC4 | `--version` prints version | Pass | `cmd/extractor/main.go:20`, `cmd/extractor/main.go:162-165`; `cmd/extractor/main_test.go:110-119` |
| AC5 | Default (no flags) enters daemon | Fail | `cmd/extractor/main.go:176-179` exits immediately instead of starting daemon |
| AC6 | Single-run sequence executed | Pass | `cmd/extractor/main.go:101-150` |
| AC7 | Success exit code 0 | Pass | `cmd/extractor/main.go:176-186` |
| AC8 | Failure exit code 1 | Pass | `cmd/extractor/main.go:167-184` |
| AC9 | No new data skip + log | Pass | `cmd/extractor/main.go:121-127`; `cmd/extractor/main_test.go:171-204` |
| AC10 | Dry-run writes no files | Fail | `cmd/extractor/main.go:135-147` still updates `state.json`, causing future runs to skip |
| AC11 | Start log w/ timestamp | Pass | `cmd/extractor/main.go:98-100` |
| AC12 | Completion log metrics | Pass | `cmd/extractor/main.go:150-154` |
| AC13 | Skip log with `last_updated` | Pass | `cmd/extractor/main.go:121-125` |
| AC14 | Failure log w/ error+duration | Pass | `cmd/extractor/main.go:101-117` |

### Task Completion Validation
| Task / Subtask | Status | Evidence |
|----------------|--------|----------|
| 1 / 1.1–1.4 CLIOptions + ParseCLI | Verified | `cmd/extractor/main.go:22-41` |
| 2 / 2.1–2.3 `--version` handling | Verified | `cmd/extractor/main.go:20`, `cmd/extractor/main.go:160-165`; `cmd/extractor/main_test.go:110-119` |
| 3 / 3.1 RunOnce signature | Verified | `cmd/extractor/main.go:72-87` |
| 3 / 3.2 API client wiring | Verified | `cmd/extractor/main.go:74-76`; `internal/api/client.go:217-270` |
| 3 / 3.3 Aggregator wiring | Verified | `cmd/extractor/main.go:75-79` |
| 3 / 3.4 State manager wiring | Verified | `cmd/extractor/main.go:77-78`; `internal/storage/state.go:23-69` |
| 3 / 3.5 LoadState | Verified | `cmd/extractor/main.go:101-105` |
| 3 / 3.6 Parallel fetch reuse | Verified via FetchAll | `internal/api/client.go:217-270` |
| 3 / 3.7 New data check | Verified | `cmd/extractor/main.go:121-126`; `internal/storage/state.go:106-135` |
| 3 / 3.8 Skip log | Verified | `cmd/extractor/main.go:121-126` |
| 3 / 3.9 Aggregate call | Verified | `cmd/extractor/main.go:119-120` |
| 3 / 3.10 Load history | Verified | `cmd/extractor/main.go:113-117`; `internal/storage/history.go:54-91` |
| 3 / 3.11 Generate outputs | Verified | `cmd/extractor/main.go:132-134` |
| 3 / 3.12 Write outputs | Verified | `cmd/extractor/main.go:135-142` |
| 3 / 3.13 Dry-run log | Verified | `cmd/extractor/main.go:135-137` |
| 3 / 3.14 Save state *violates dry-run intent* | Issue | `cmd/extractor/main.go:144-147` saves state even during dry-run |
| 3 / 3.15 Return codes | Verified | `cmd/extractor/main.go:150-157` |
| 4 / 4.1–4.5 Logging | Verified | `cmd/extractor/main.go:98-154` |
| 5 / 5.1 Config load | Verified | `cmd/extractor/main.go:167-170`; `internal/config/config.go:78-129` |
| 5 / 5.2 Env overrides | Verified | `internal/config/config.go:36-68` |
| 5 / 5.3 Validation | Verified | `internal/config/config.go:131-170` |
| 5 / 5.4 Logger setup | Verified | `cmd/extractor/main.go:172-175`; `internal/logging/logging.go:8-37` |
| 5 / 5.5 RunOnce when `--once` | Verified | `cmd/extractor/main.go:176-186` |
| 5 / 5.6 Error exit 1 | Verified | `cmd/extractor/main.go:181-183` |
| 5 / 5.7 Success exit 0 | Verified | `cmd/extractor/main.go:176-186` |
| 5 / 5.8 Daemon placeholder | Partial | `cmd/extractor/main.go:176-179` logs stub but does not start daemon |
| 6 / 6.1–6.9 Unit tests | Verified | `cmd/extractor/main_test.go:88-284` |
| 7 / 7.1–7.5 Manual verification | Not independently verifiable; only noted in Dev Notes |

### Test Coverage and Gaps
- `go test ./...` (2025-12-01) currently passes locally.
- `cmd/extractor/main_test.go:88-284` exercises CLI flag parsing, RunOnce success/skip/dry-run, daemon stub, and error propagation.
- No automated coverage validates the new `protocolList` envelope decoder, so regressions could reintroduce JSON decoding failures.

### Architectural Alignment
- CLI flag handling follows ADR-001's stdlib-flag requirement but still needs proper error surfacing (`docs/architecture/architecture-decision-records-adrs.md:3-12`).
- Structured logging requirements from ADR-004 are satisfied (`cmd/extractor/main.go:98-154`), and Go-only dependency policy remains intact.

### Security Notes
- No secrets or elevated permissions introduced. Primary risk is operational: dry-run mutates persisted state leading to skipped real writes.

### Best-Practices and References
1. PRD CLI requirements (FR42–FR48) demand both dry-run safety and daemon defaults (`docs/prd.md:152-172`, `docs/prd.md:303-310`).
2. ADR-001 mandates stdlib flag usage with predictable UX (`docs/architecture/architecture-decision-records-adrs.md:3-12`).

### Action Items

**Code Changes Required**
- [x] [High] Gate state/history persistence when `opts.DryRun` is true so FR45/AC10 hold and real runs are not skipped (`cmd/extractor/main.go:135-147`, `internal/storage/state.go:82-135`).
- [x] [Med] Propagate flag parsing errors (show usage, return non-zero) so typos like `--onxe` do not silently drop `--once` and `--config` (`cmd/extractor/main.go:29-41`).
- [x] [Med] Add tests (unit or fixture-driven) covering the `{"protocols": [...]}` envelope to lock the decoder change (`internal/api/responses.go:30-57`, `internal/api/responses_test.go:1-76`).

**Advisory Notes**
- None.

## Senior Developer Review (AI)

**Reviewer:** BMad  
**Date:** 2025-12-01  
**Outcome:** Changes Requested – default/no-flag execution still exits immediately, so AC5 remains unmet.

### Summary
1. CLI flags, RunOnce orchestration, dry-run safety, and the protocol envelope regression tests all behave as specified; `go test ./...` (2025-12-01) passes end-to-end.
2. Architecture, tech spec, and story context were loaded; there were no UX or document_project files under the configured glob, so review relied on the available artifacts.
3. Default execution without `--once` still logs "daemon mode not implemented (Story 5.3)" and returns 0 (`cmd/extractor/main.go:187-190`), conflicting with the deployment contract that `./extractor --config config.yaml` launches the daemon (`docs/architecture/deployment-architecture.md:5-13`).

### Key Findings
**High**
1. AC5 missing: running without flags never enters daemon mode or fails loudly, so operators cannot follow the documented workflow (`cmd/extractor/main.go:187-190`; `cmd/extractor/main_test.go:285-295`).

**Medium**
- None.

**Low**
- None.

### Acceptance Criteria Coverage
13 / 14 acceptance criteria implemented.

| AC | Description | Status | Evidence |
|----|-------------|--------|----------|
| AC1 | `--once` flag enables single run and exit | Pass | `cmd/extractor/main.go:24`; `cmd/extractor/main.go:187`; `cmd/extractor/main_test.go:88` |
| AC2 | `--config` overrides path | Pass | `cmd/extractor/main.go:40`; `cmd/extractor/main.go:167`; `cmd/extractor/main_test.go:105` |
| AC3 | `--dry-run` skips writes and logs notice | Pass | `cmd/extractor/main.go:37`; `cmd/extractor/main.go:136`; `cmd/extractor/main_test.go:229` |
| AC4 | `--version` prints release then exits 0 | Pass | `cmd/extractor/main.go:22`; `cmd/extractor/main.go:173`; `cmd/extractor/main_test.go:133` |
| AC5 | Default (no flags) starts daemon | Fail | `cmd/extractor/main.go:187`; `cmd/extractor/main_test.go:285` |
| AC6 | Single-run executes documented sequence | Pass | `cmd/extractor/main.go:105`; `cmd/extractor/main.go:125`; `cmd/extractor/main.go:142`; `cmd/extractor/main.go:147` |
| AC7 | Success exit code is 0 | Pass | `cmd/extractor/main.go:176` |
| AC8 | Failure exit code is 1 with error logged | Pass | `cmd/extractor/main.go:167`; `cmd/extractor/main.go:192` |
| AC9 | No new data skips work + logs reason | Pass | `cmd/extractor/main.go:125`; `cmd/extractor/main_test.go:194` |
| AC10 | Dry-run performs work but avoids writes/state | Pass | `cmd/extractor/main.go:136`; `cmd/extractor/main.go:147`; `cmd/extractor/main_test.go:229` |
| AC11 | Start log carries timestamp | Pass | `cmd/extractor/main.go:102` |
| AC12 | Completion log includes metrics | Pass | `cmd/extractor/main.go:154` |
| AC13 | Skip log reports `last_updated` | Pass | `cmd/extractor/main.go:125` |
| AC14 | Failure log includes error + duration | Pass | `cmd/extractor/main.go:105` |

### Task Completion Validation
| Task / Subtask | Status | Evidence |
|----------------|--------|----------|
| 1 / 1.1 CLIOptions struct fields | Verified | `cmd/extractor/main.go:24` |
| 1 / 1.2 `ParseCLI()` helper | Verified | `cmd/extractor/main.go:33` |
| 1 / 1.3 Register flags | Verified | `cmd/extractor/main.go:39` |
| 1 / 1.4 Call `flag.Parse()` | Verified | `cmd/extractor/main.go:44` |
| 2 / 2.1 Version constant | Verified | `cmd/extractor/main.go:22` |
| 2 / 2.2 Version flag check | Verified | `cmd/extractor/main.go:173` |
| 2 / 2.3 Print version + exit 0 | Verified | `cmd/extractor/main.go:174` |
| 3 / 3.1 `RunOnce` signature | Verified | `cmd/extractor/main.go:76` |
| 3 / 3.2 API client construction | Verified | `cmd/extractor/main.go:79` |
| 3 / 3.3 Aggregator construction | Verified | `cmd/extractor/main.go:80` |
| 3 / 3.4 State manager construction | Verified | `cmd/extractor/main.go:81` |
| 3 / 3.5 Load state | Verified | `cmd/extractor/main.go:105` |
| 3 / 3.6 Parallel fetch via errgroup | Verified | `internal/api/client.go:276` |
| 3 / 3.7 New-data check | Verified | `cmd/extractor/main.go:125` |
| 3 / 3.8 Skip log when stale | Verified | `cmd/extractor/main.go:125` |
| 3 / 3.9 Aggregate data | Verified | `cmd/extractor/main.go:123` |
| 3 / 3.10 Load history | Verified | `cmd/extractor/main.go:117`; `internal/storage/history.go:50` |
| 3 / 3.11 Generate outputs | Verified | `cmd/extractor/main.go:139`; `internal/storage/writer.go:43` |
| 3 / 3.12 Write outputs | Verified | `cmd/extractor/main.go:142`; `internal/storage/writer.go:159` |
| 3 / 3.13 Dry-run log text | Verified | `cmd/extractor/main.go:136` |
| 3 / 3.14 Save state after successful writes | Verified | `cmd/extractor/main.go:147`; `internal/storage/state.go:82` |
| 3 / 3.15 Return nil on success | Verified | `cmd/extractor/main.go:161` |
| 4 / 4.1 Record start time | Verified | `cmd/extractor/main.go:102` |
| 4 / 4.2 Info log "extraction started" | Verified | `cmd/extractor/main.go:102` |
| 4 / 4.3 Completion log w/ metrics | Verified | `cmd/extractor/main.go:154` |
| 4 / 4.4 Skip log w/ `last_updated` | Verified | `cmd/extractor/main.go:125` |
| 4 / 4.5 Error log w/ duration | Verified | `cmd/extractor/main.go:105` |
| 5 / 5.1 Load config after version check | Verified | `cmd/extractor/main.go:167`; `internal/config/config.go:124` |
| 5 / 5.2 Apply env overrides | Verified | `internal/config/config.go:55` |
| 5 / 5.3 Validate config | Verified | `internal/config/config.go:148` |
| 5 / 5.4 Initialize slog logger | Verified | `cmd/extractor/main.go:184`; `internal/logging/logging.go:12` |
| 5 / 5.5 Execute `RunOnce` when `--once` | Verified | `cmd/extractor/main.go:176` |
| 5 / 5.6 Exit 1 on RunOnce error | Verified | `cmd/extractor/main.go:181` |
| 5 / 5.7 Exit 0 on success | Verified | `cmd/extractor/main.go:176` |
| 5 / 5.8 Daemon placeholder | Verified (stub only) | `cmd/extractor/main.go:187`; `cmd/extractor/main_test.go:285` |
| 6 / 6.1 Flag parsing test coverage | Verified | `cmd/extractor/main_test.go:88` |
| 6 / 6.2 Default flag values test | Verified | `cmd/extractor/main_test.go:88` |
| 6 / 6.3 Version output test | Verified | `cmd/extractor/main_test.go:133` |
| 6 / 6.4 RunOnce success test | Verified | `cmd/extractor/main_test.go:145` |
| 6 / 6.5 No-new-data test | Verified | `cmd/extractor/main_test.go:194` |
| 6 / 6.6 Dry-run test | Verified | `cmd/extractor/main_test.go:229` |
| 6 / 6.7 Failure propagation test | Verified | `cmd/extractor/main_test.go:298` |
| 6 / 6.8 Log attribute assertions | Verified | `cmd/extractor/main_test.go:145` |
| 6 / 6.9 Default mode test (stub) | Verified | `cmd/extractor/main_test.go:285` |
| 7 / 7.1 Build succeeds | Not independently verified | Dev notes only |
| 7 / 7.2 CLI tests run | Not independently verified | Dev notes only |
| 7 / 7.3 Lint run | Not independently verified | Dev notes only |
| 7 / 7.4 Manual `--version` smoke | Not independently verified | Dev notes only |
| 7 / 7.5 Manual `--once --dry-run` smoke | Not independently verified | Dev notes only |

### Test Coverage and Gaps
- Local run: `go test ./...` (2025-12-01) passes across cmd/extractor, internal/api, storage, etc.  
- Unit tests exercise CLI parsing, RunOnce success/skip/dry-run, daemon stub, and protocol envelope decoding (`cmd/extractor/main_test.go:88-327`; `internal/api/responses_test.go:10-123`).  
- Gap: no integration or acceptance coverage for the actual daemon scheduler yet, so AC5 still lacks automated detection beyond the current stub (expected to be addressed once daemon mode ships per `docs/architecture/testing-strategy.md:5-27`).

### Architectural Alignment
- Usage of the stdlib `flag` package and slog aligns with ADR-001 and ADR-004 (`docs/architecture/architecture-decision-records-adrs.md:3-47`).
- Deployment expectations for a no-flag daemon remain unmet (`docs/architecture/deployment-architecture.md:5-13`), so action is required before marking this story done.

### Security Notes
- No secrets are introduced; the tool continues to use public endpoints only, matching the documented security posture (`docs/architecture/security-architecture.md:3-12`).

### Best-Practices and References
1. ADR-001/004 continue to govern CLI and logging choices (`docs/architecture/architecture-decision-records-adrs.md:3-47`).
2. Deployment expectations for `./extractor --config config.yaml` are defined in the deployment architecture doc (`docs/architecture/deployment-architecture.md:5-13`).
3. Test organization guidance in the testing strategy doc underpins the current unit-test approach (`docs/architecture/testing-strategy.md:5-27`).

### Action Items

**Code Changes Required**
- [x] [High] Implement daemon scheduling (or fail fast) for the default execution path so AC5 is satisfied before Story 5.2 can be marked done (`cmd/extractor/main.go:187-190`; `cmd/extractor/main_test.go:285-295`).

**Advisory Notes**
- Note: Once daemon mode lands in Story 5.3, add integration tests that assert the scheduler + signal handling contract to keep future regressions visible.

## Senior Developer Review (AI)

**Reviewer:** BMad  
**Date:** 2025-12-02  
**Outcome:** Changes Requested – a failed `start_immediately` daemon cycle still returns an error instead of waiting for the next interval, so operators must restart manually.

### Summary
1. Story, context XML, tech spec (`docs/sprint-artifacts/tech-spec-epic-5.md`), and architecture set (`docs/architecture/*.md`) were fully loaded; no `ux_design` or `document_project` files matched the configured globs, so those inputs remain unavailable.
2. Stack confirmed as Go 1.24 with stdlib `flag`/`slog` per ADR-001/ADR-004 (`go.mod`, `docs/architecture/architecture-decision-records-adrs.md`); CLI + RunOnce flow now satisfy AC1–AC14, and File List entries point to every touched file.
3. Validation included `go test ./...`, `go build ./...`, `make lint`, `go run ./cmd/extractor --version`, and `go run ./cmd/extractor --once --dry-run --config configs/config.yaml`, confirming CLI behavior, dry-run safety, and that daemon scheduling triggers; the only blocker is the lack of retry after an immediate start failure.

### Key Findings
**Medium**
1. `runDaemonWithDeps` aborts when the startup `start_immediately` invocation fails instead of logging the error and waiting for the ticker, which violates the Epic 5 reliability contract that daemon mode must continue after failures (`cmd/extractor/main.go:222-225`, `docs/sprint-artifacts/tech-spec-epic-5.md:219-289`).

### Acceptance Criteria Coverage
14 / 14 ACs implemented.

| AC | Description | Status | Evidence |
|----|-------------|--------|----------|
| AC1 | `--once` flag enables single-run | Pass | `cmd/extractor/main.go:27-48`, `cmd/extractor/main_test.go:85-114` |
| AC2 | `--config` overrides path | Pass | `cmd/extractor/main.go:42-45`, `cmd/extractor/main.go:257-260`, `cmd/extractor/main_test.go:102-113` |
| AC3 | `--dry-run` skips writes/logs message | Pass | `cmd/extractor/main.go:139-155`, `cmd/extractor/main_test.go:226-264` |
| AC4 | `--version` prints version and exits 0 | Pass | `cmd/extractor/main.go:25,252-255`, `cmd/extractor/main_test.go:130-139`, `go run ./cmd/extractor --version` |
| AC5 | Default (no flags) enters daemon | Pass | `cmd/extractor/main.go:263-281`, `cmd/extractor/main.go:186-241`, `cmd/extractor/main_test.go:320-381` |
| AC6 | Single-run sequence executed | Pass | `cmd/extractor/main.go:105-155`, `internal/api/client.go:252-330`, `internal/storage/state.go:30-146` |
| AC7 | Success exit code 0 | Pass | `cmd/extractor/main.go:283-288`, `cmd/extractor/main_test.go:142-189` |
| AC8 | Failure exit code 1 | Pass | `cmd/extractor/main.go:243-285`, `cmd/extractor/main_test.go:266-310` |
| AC9 | No new data skip + log | Pass | `cmd/extractor/main.go:128-134`, `cmd/extractor/main_test.go:191-224`, `go run ./cmd/extractor --once --dry-run --config configs/config.yaml` |
| AC10 | Dry-run writes no files | Pass | `cmd/extractor/main.go:139-155`, `cmd/extractor/main_test.go:226-264`, `go run ./cmd/extractor --once --dry-run --config configs/config.yaml` |
| AC11 | Start log w/ timestamp | Pass | `cmd/extractor/main.go:105-107`, runtime logs from manual dry-run |
| AC12 | Completion log metrics | Pass | `cmd/extractor/main.go:157-162`, `cmd/extractor/main_test.go:142-189` |
| AC13 | Skip log with `last_updated` | Pass | `cmd/extractor/main.go:128-133`, manual dry-run output |
| AC14 | Failure log w/ error+duration | Pass | `cmd/extractor/main.go:108-117`, `cmd/extractor/main_test.go:266-310` |

### Task Completion Validation
| Task / Subtask | Status | Evidence |
|----------------|--------|----------|
| 1.1 Define CLIOptions struct | Verified | `cmd/extractor/main.go:27-32` |
| 1.2 Add `ParseCLI()` helper | Verified | `cmd/extractor/main.go:34-49` |
| 1.3 Register flags | Verified | `cmd/extractor/main.go:41-45` |
| 1.4 Call `flag.Parse()` and populate struct | Verified | `cmd/extractor/main.go:47-48` |
| 2.1 Define `const Version` | Verified | `cmd/extractor/main.go:25` |
| 2.2 Check `opts.Version` in `main()` | Verified | `cmd/extractor/main.go:252-254` |
| 2.3 Print version + exit 0 | Verified | `cmd/extractor/main.go:253-255`, `cmd/extractor/main_test.go:130-139` |
| 3.1 Create `RunOnce` signature | Verified | `cmd/extractor/main.go:79-93` |
| 3.2 Instantiate API client | Verified | `cmd/extractor/main.go:81-83`, `internal/api/client.go:252-330` |
| 3.3 Instantiate aggregator | Verified | `cmd/extractor/main.go:82-84`, `internal/aggregator/aggregator.go:1-56` |
| 3.4 Instantiate state manager | Verified | `cmd/extractor/main.go:83-85`, `internal/storage/state.go:30-65` |
| 3.5 Load state via `LoadState()` | Verified | `cmd/extractor/main.go:108-112` |
| 3.6 Fetch in parallel via errgroup | Verified | `cmd/extractor/main.go:114-118`, `internal/api/client.go:276-330` |
| 3.7 Check for new data | Verified | `cmd/extractor/main.go:126-134`, `internal/storage/state.go:89-131` |
| 3.8 Skip log when stale | Verified | `cmd/extractor/main.go:128-133`, `cmd/extractor/main_test.go:191-224` |
| 3.9 Aggregate data | Verified | `cmd/extractor/main.go:126-127`, `internal/aggregator/aggregator.go:12-52` |
| 3.10 Load history | Verified | `cmd/extractor/main.go:120-124`, `internal/storage/history.go:26-82` |
| 3.11 Generate outputs | Verified | `cmd/extractor/main.go:142-144`, `internal/storage/writer.go:30-156` |
| 3.12 Write outputs | Verified | `cmd/extractor/main.go:145-149`, `internal/storage/writer.go:158-210` |
| 3.13 Dry-run log message | Verified | `cmd/extractor/main.go:139-141`, `cmd/extractor/main_test.go:226-264` |
| 3.14 Save state | Verified | `cmd/extractor/main.go:150-154`, `internal/storage/state.go:61-86` |
| 3.15 Return nil on success | Verified | `cmd/extractor/main.go:157-165`, `cmd/extractor/main_test.go:142-189` |
| 4.1 Record start time | Verified | `cmd/extractor/main.go:105-107` |
| 4.2 Log "extraction started" | Verified | `cmd/extractor/main.go:105-107`, runtime logs |
| 4.3 Log "extraction completed" with metrics | Verified | `cmd/extractor/main.go:157-162`, `cmd/extractor/main_test.go:142-189` |
| 4.4 Log skip with `last_updated` | Verified | `cmd/extractor/main.go:128-133`, manual dry-run output |
| 4.5 Log failures w/ duration | Verified | `cmd/extractor/main.go:108-117`, `cmd/extractor/main_test.go:266-310` |
| 5.1 Load config after version check | Verified | `cmd/extractor/main.go:257-260` |
| 5.2 Apply env overrides | Verified | `internal/config/config.go:52-78` |
| 5.3 Validate config | Verified | `internal/config/config.go:100-170` |
| 5.4 Initialize slog logger | Verified | `cmd/extractor/main.go:263-265`, `internal/logging/logging.go:8-40` |
| 5.5 Call `RunOnce` when `--once` | Verified | `cmd/extractor/main.go:266-287` |
| 5.6 Exit 1 on RunOnce error | Verified | `cmd/extractor/main.go:283-285`, `cmd/extractor/main_test.go:266-310` |
| 5.7 Exit 0 on success | Verified | `cmd/extractor/main.go:283-288` |
| 5.8 Wire daemon path | Verified (with noted follow-up) | `cmd/extractor/main.go:266-281`, `cmd/extractor/main_test.go:320-381` |
| 6.1 Test ParseCLI flag parsing | Verified | `cmd/extractor/main_test.go:85-114` |
| 6.2 Test default flag values | Verified | `cmd/extractor/main_test.go:85-100` |
| 6.3 Test version output | Verified | `cmd/extractor/main_test.go:130-139` |
| 6.4 Test RunOnce success | Verified | `cmd/extractor/main_test.go:142-189` |
| 6.5 Test no-new-data skip | Verified | `cmd/extractor/main_test.go:191-224` |
| 6.6 Test dry-run behavior | Verified | `cmd/extractor/main_test.go:226-264` |
| 6.7 Test failure propagation | Verified | `cmd/extractor/main_test.go:282-310` |
| 6.8 Assert log attributes | Verified | `cmd/extractor/main_test.go:142-264` |
| 6.9 Validate default mode via daemon deps | Verified | `cmd/extractor/main_test.go:320-381` |
| 7.1 Run `go build ./...` | Verified | Reviewer executed `go build ./...` (2025-12-02) |
| 7.2 Run `go test ./cmd/extractor/...` | Verified | Reviewer executed `go test ./...` (2025-12-02) |
| 7.3 Run `make lint` | Verified | Reviewer executed `make lint` (2025-12-02) |
| 7.4 Manual smoke `./extractor --version` | Verified | Reviewer executed `go run ./cmd/extractor --version` (2025-12-02) |
| 7.5 Manual smoke `--once --dry-run` | Verified | Reviewer executed `go run ./cmd/extractor --once --dry-run --config configs/config.yaml` (2025-12-02) |

### Test Coverage and Gaps
- `go test ./...`, `go build ./...`, and `make lint` all succeed locally (2025-12-02).
- Runtime smokes for `--version` and `--once --dry-run` confirm CLI flag handling, dry-run logging, and skip-path behavior against live DefiLlama endpoints.
- Daemon logic is unit-tested via injected dependencies, but no integration/e2e test yet asserts signal-handling plus scheduler timing; consider adding once daemon mode stabilizes (ties to Story 5.3 scope).

### Architectural Alignment
- CLI implementation stays within ADR-001/ADR-004 mandates for stdlib `flag`/`slog`, matching the architecture references in `docs/architecture/architecture-decision-records-adrs.md`.
- Deployment guidance in `docs/architecture/deployment-architecture.md` expects daemon resiliency; once the immediate-start failure handling is fixed, the current wiring will align with FR43/FR47 requirements.

### Security Notes
- No secrets or privileged operations introduced; all runtime calls hit public DefiLlama endpoints and respect the existing config/output directories.

### Best-Practices and References
1. Stack + dependency posture confirmed from `go.mod` and ADR docs (`go.mod:1-11`, `docs/architecture/architecture-decision-records-adrs.md`).
2. PRD FR42–FR48/FR56 references (`docs/prd.md`) continue to govern CLI and logging expectations satisfied here.
3. Daemon behavior must continue after transient failures per `docs/sprint-artifacts/tech-spec-epic-5.md:219-289`.

### Action Items

**Code Changes Required**
- [x] [Med] Keep daemon mode running after `start_immediately` failures by logging the error and proceeding to the next interval instead of returning from `runDaemonWithDeps` (`cmd/extractor/main.go:222-225`, `docs/sprint-artifacts/tech-spec-epic-5.md:219-289`).

**Advisory Notes**
- None.

## Senior Developer Review (AI)

**Reviewer:** BMad  
**Date:** 2025-12-02  
**Outcome:** Approve – CLI flag coverage, RunOnce orchestration, daemon handoff, and protocol decoder regression tests satisfy AC1–AC14 without outstanding defects.

### Summary
1. CLI stack detected as Go 1.24 with stdlib `flag`/`log/slog` per `go.mod:1-9` and ADR-001/ADR-004; unit + smoke tests exercise `cmd/extractor/main.go` + `cmd/extractor/main_test.go` for all flag combinations and manual `go run` commands cover `--version`/`--once --dry-run` flows.
2. RunOnce/state/history path loads config, runs `internal/api/client.go` parallel fetches, executes aggregation, dry-run guard, and persistence with structured logging; File List entries (`cmd/extractor/main.go`, `cmd/extractor/main_test.go`, `internal/api/{client.go,responses.go,responses_test.go}`, `docs/sprint-artifacts/5-2-implement-cli-and-single-run-mode.md`, `docs/sprint-artifacts/sprint-status.yaml`, validation report) match actual edits.
3. Regression coverage spans CLI parsing, RunOnce success/skip/dry-run/error, daemon ticks, and protocol envelopes; `go build ./...`, `go test ./...`, and `make lint` all pass on 2025-12-02 alongside manual CLI smokes.

### Key Findings
- None – no High/Medium/Low issues observed.

**Acceptance Criteria Coverage (14/14 implemented)**

| AC | Description | Status | Evidence |
|----|-------------|--------|----------|
| AC1 | `--once` flag enables single-run exit | Pass | `cmd/extractor/main.go:42-45`, `cmd/extractor/main.go:266-288`, `cmd/extractor/main_test.go:142-189` |
| AC2 | `--config` overrides config path | Pass | `cmd/extractor/main.go:42-45`, `cmd/extractor/main.go:257-260`, `cmd/extractor/main_test.go:102-113` |
| AC3 | `--dry-run` skips writes and logs notice | Pass | `cmd/extractor/main.go:139-154`, `cmd/extractor/main_test.go:226-264` |
| AC4 | `--version` prints release and exits 0 | Pass | `cmd/extractor/main.go:25`, `cmd/extractor/main.go:252-255`, `cmd/extractor/main_test.go:130-139` |
| AC5 | Default no-flag path enters daemon | Pass | `cmd/extractor/main.go:266-281`, `cmd/extractor/main_test.go:320-365` |
| AC6 | Single-run sequence executes config→state→fetch→aggregate→write→state update | Pass | `cmd/extractor/main.go:105-155`, `internal/api/client.go:252-330` |
| AC7 | Success exit code is 0 | Pass | `cmd/extractor/main.go:283-288` |
| AC8 | Failures exit 1 with logs | Pass | `cmd/extractor/main.go:167-285`, `cmd/extractor/main_test.go:266-310` |
| AC9 | No-new-data path exits 0 with skip log | Pass | `cmd/extractor/main.go:126-134`, `cmd/extractor/main_test.go:191-224` |
| AC10 | Dry-run performs work but avoids writes/state | Pass | `cmd/extractor/main.go:139-154`, `cmd/extractor/main_test.go:226-264` |
| AC11 | Start log with timestamp | Pass | `cmd/extractor/main.go:105-107` |
| AC12 | Completion log with metrics | Pass | `cmd/extractor/main.go:157-162`, `cmd/extractor/main_test.go:142-189` |
| AC13 | Skip log includes `last_updated` | Pass | `cmd/extractor/main.go:128-133` |
| AC14 | Failure log reports error + duration | Pass | `cmd/extractor/main.go:108-117`, `cmd/extractor/main.go:145-153`, `cmd/extractor/main_test.go:282-310` |

**Task Completion Validation (49/49 verified)**

| Task/Subtask | Marked | Verified | Evidence |
|--------------|--------|----------|----------|
| 1.1 CLIOptions struct | ☑ | Pass | `cmd/extractor/main.go:27-32` |
| 1.2 ParseCLI helper | ☑ | Pass | `cmd/extractor/main.go:34-49` |
| 1.3 Flag registration | ☑ | Pass | `cmd/extractor/main.go:42-45` |
| 1.4 flag.Parse invocation | ☑ | Pass | `cmd/extractor/main.go:47-48` |
| 2.1 Version constant | ☑ | Pass | `cmd/extractor/main.go:25` |
| 2.2 Version flag check | ☑ | Pass | `cmd/extractor/main.go:252-255` |
| 2.3 Print + exit 0 | ☑ | Pass | `cmd/extractor/main.go:252-255` |
| 3.1 RunOnce signature | ☑ | Pass | `cmd/extractor/main.go:79-93` |
| 3.2 API client wiring | ☑ | Pass | `cmd/extractor/main.go:81-83` |
| 3.3 Aggregator wiring | ☑ | Pass | `cmd/extractor/main.go:82-84` |
| 3.4 State manager wiring | ☑ | Pass | `cmd/extractor/main.go:83-85` |
| 3.5 Load state | ☑ | Pass | `cmd/extractor/main.go:108-112` |
| 3.6 Parallel fetch via errgroup | ☑ | Pass | `internal/api/client.go:252-330` |
| 3.7 New-data check | ☑ | Pass | `cmd/extractor/main.go:126-134` |
| 3.8 Skip log | ☑ | Pass | `cmd/extractor/main.go:128-133` |
| 3.9 Aggregate data | ☑ | Pass | `cmd/extractor/main.go:126-127` |
| 3.10 Load history | ☑ | Pass | `cmd/extractor/main.go:120-124` |
| 3.11 Generate outputs | ☑ | Pass | `cmd/extractor/main.go:142-144` |
| 3.12 Write outputs | ☑ | Pass | `cmd/extractor/main.go:145-148` |
| 3.13 Dry-run log | ☑ | Pass | `cmd/extractor/main.go:139-141` |
| 3.14 Save state | ☑ | Pass | `cmd/extractor/main.go:150-154` |
| 3.15 Return nil on success | ☑ | Pass | `cmd/extractor/main.go:157-165` |
| 4.1 Record start time | ☑ | Pass | `cmd/extractor/main.go:105-107` |
| 4.2 Start log | ☑ | Pass | `cmd/extractor/main.go:105-107` |
| 4.3 Completion log | ☑ | Pass | `cmd/extractor/main.go:157-162` |
| 4.4 Skip log attributes | ☑ | Pass | `cmd/extractor/main.go:128-133` |
| 4.5 Error log attributes | ☑ | Pass | `cmd/extractor/main.go:108-117`, `cmd/extractor/main.go:145-153` |
| 5.1 Load config post-version | ☑ | Pass | `cmd/extractor/main.go:257-260` |
| 5.2 Apply env overrides | ☑ | Pass | `internal/config/config.go:52-78` |
| 5.3 Validate config | ☑ | Pass | `internal/config/config.go:92-119` |
| 5.4 Initialize slog logger | ☑ | Pass | `cmd/extractor/main.go:263-265` |
| 5.5 RunOnce on `--once` | ☑ | Pass | `cmd/extractor/main.go:283-284` |
| 5.6 Exit 1 on RunOnce error | ☑ | Pass | `cmd/extractor/main.go:283-285` |
| 5.7 Exit 0 on success | ☑ | Pass | `cmd/extractor/main.go:287-288` |
| 5.8 Daemon path when no `--once` | ☑ | Pass | `cmd/extractor/main.go:266-281` |
| 6.1 ParseCLI flag tests | ☑ | Pass | `cmd/extractor/main_test.go:85-114` |
| 6.2 Default value tests | ☑ | Pass | `cmd/extractor/main_test.go:85-100` |
| 6.3 Version output test | ☑ | Pass | `cmd/extractor/main_test.go:130-139` |
| 6.4 RunOnce success test | ☑ | Pass | `cmd/extractor/main_test.go:142-189` |
| 6.5 No-new-data test | ☑ | Pass | `cmd/extractor/main_test.go:191-224` |
| 6.6 Dry-run test | ☑ | Pass | `cmd/extractor/main_test.go:226-264` |
| 6.7 Failure propagation test | ☑ | Pass | `cmd/extractor/main_test.go:282-310` |
| 6.8 Log attribute assertions | ☑ | Pass | `cmd/extractor/main_test.go:142-264` |
| 6.9 Daemon dependency tests | ☑ | Pass | `cmd/extractor/main_test.go:320-413` |
| 7.1 `go build ./...` | ☑ | Pass | Command executed 2025-12-02 (`go build ./...`) |
| 7.2 `go test ./...` | ☑ | Pass | Command executed 2025-12-02 (`go test ./...`) |
| 7.3 `make lint` | ☑ | Pass | Command executed 2025-12-02 (`make lint`) |
| 7.4 Smoke: `go run ./cmd/extractor --version` | ☑ | Pass | Command executed 2025-12-02 (prints version) |
| 7.5 Smoke: `go run ./cmd/extractor --once --dry-run --config configs/config.yaml` | ☑ | Pass | Command executed 2025-12-02 (logs skip path) |

### Test Coverage and Gaps
- `go test ./...`, `go build ./...`, and `make lint` succeeded on 2025-12-02; manual `go run ./cmd/extractor --version` and `--once --dry-run --config configs/config.yaml` confirm CLI behavior against live endpoints.
- Unit suites in `cmd/extractor/main_test.go` and `internal/api/responses_test.go` cover CLI parsing, RunOnce success/skip/dry-run/error, daemon ticker scheduling, and protocol envelopes; remaining integration gap is full daemon signal-handling (Story 5.3 scope).

### Architectural Alignment
- Implementation adheres to ADR-001/ADR-004 for stdlib `flag` + `log/slog` usage (`docs/architecture/architecture-decision-records-adrs.md:3-47`) and matches deployment guidance in `docs/architecture/deployment-architecture.md:1-20`.

### Security Notes
- No new secrets or privileged operations introduced; behavior remains within `docs/architecture/security-architecture.md:1-9` constraints (public DefiLlama endpoints, atomic writes only).

### Best-Practices and References
1. Go stdlib `flag` documentation (pkg.go.dev/flag) confirms canonical CLI parsing semantics.
2. Go `log/slog` handler guidance (pkg.go.dev/log/slog) informs structured logging usage in daemon/single-run paths.
3. Tech spec remains authoritative for sequencing and daemon behavior (`docs/sprint-artifacts/tech-spec-epic-5.md:96-289`).

### Action Items

**Code Changes Required**
- None – story approved.

**Advisory Notes**
- Note: Add end-to-end signal-handling validation once Story 5.3 lands to complement current daemon unit coverage.
