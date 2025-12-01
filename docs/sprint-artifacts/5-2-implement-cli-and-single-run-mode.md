# Story 5.2: Implement CLI and Single-Run Mode

Status: ready-for-dev

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

- [ ] Task 1: Define CLIOptions struct (AC: 1-5)
  - [ ] 1.1: Create `CLIOptions` struct in `cmd/extractor/main.go` with fields: `Once bool`, `ConfigPath string`, `DryRun bool`, `Version bool`
  - [ ] 1.2: Add `ParseCLI() CLIOptions` function using `flag` package
  - [ ] 1.3: Register flags: `--once`, `--config` (default "config.yaml"), `--dry-run`, `--version`
  - [ ] 1.4: Call `flag.Parse()` and populate struct

- [ ] Task 2: Implement --version handling (AC: 4)
  - [ ] 2.1: Define `const Version = "1.0.0"` at package level
  - [ ] 2.2: In `main()`, after parsing flags, check `opts.Version`
  - [ ] 2.3: If true: print "defillama-extract v1.0.0" to stdout and `os.Exit(0)`

- [ ] Task 3: Implement RunOnce function (AC: 6-10)
  - [ ] 3.1: Create `RunOnce(ctx context.Context, cfg *config.Config, opts CLIOptions) error`
  - [ ] 3.2: Create API client using `api.NewClient(cfg)`
  - [ ] 3.3: Create aggregator using `aggregator.New(cfg.Oracle.Name)`
  - [ ] 3.4: Create state manager using `storage.NewStateManager(cfg)`
  - [ ] 3.5: Load state via `stateManager.LoadState()`
  - [ ] 3.6: Fetch oracles and protocols in parallel using errgroup (reuse Epic 2 pattern)
  - [ ] 3.7: Check for new data: compare API timestamp with `state.LastUpdated`
  - [ ] 3.8: If no new data: log "no new data, skipping extraction" and return nil
  - [ ] 3.9: Run aggregation via `aggregator.Aggregate(oracleData, protocols)`
  - [ ] 3.10: Load history via `stateManager.LoadHistory()`
  - [ ] 3.11: Generate outputs using `storage.GenerateFullOutput()` and `storage.GenerateSummaryOutput()`
  - [ ] 3.12: If NOT dry-run: write outputs via `storage.WriteAllOutputs()`
  - [ ] 3.13: If dry-run: log "dry-run mode, skipping file writes"
  - [ ] 3.14: Save state via `stateManager.SaveState()`
  - [ ] 3.15: Return nil on success, error on failure

- [ ] Task 4: Implement extraction logging (AC: 11-14)
  - [ ] 4.1: Record start time with `time.Now()` at extraction start
  - [ ] 4.2: Log INFO "extraction started" with timestamp at cycle start
  - [ ] 4.3: On success: calculate duration, log INFO "extraction completed" with `duration_ms`, `protocol_count`, `tvs`, `chains`
  - [ ] 4.4: On skip: log INFO "extraction skipped, no new data" with `last_updated`
  - [ ] 4.5: On error: log ERROR "extraction failed" with `error`, `duration_ms`

- [ ] Task 5: Wire main() for single-run mode (AC: 7, 8)
  - [ ] 5.1: After version check, load config using `config.Load(opts.ConfigPath)`
  - [ ] 5.2: Apply environment overrides using existing `config.ApplyEnv()`
  - [ ] 5.3: Validate config using `config.Validate()`
  - [ ] 5.4: Initialize slog logger based on config.Logging settings
  - [ ] 5.5: If `opts.Once`: call `RunOnce(ctx, cfg, opts)`
  - [ ] 5.6: If RunOnce returns error: log error, `os.Exit(1)`
  - [ ] 5.7: If RunOnce returns nil: `os.Exit(0)`
  - [ ] 5.8: If NOT `opts.Once`: placeholder for daemon mode (Story 5.3)

- [ ] Task 6: Write unit tests (AC: all)
  - [ ] 6.1: Test `ParseCLI` parses all flags correctly
  - [ ] 6.2: Test `ParseCLI` default values (ConfigPath="config.yaml", others=false)
  - [ ] 6.3: Test --version output format and exit
  - [ ] 6.4: Test RunOnce success path returns nil
  - [ ] 6.5: Test RunOnce with no new data returns nil (skip path)
  - [ ] 6.6: Test RunOnce dry-run mode skips file writes
  - [ ] 6.7: Test RunOnce failure returns error
  - [ ] 6.8: Test log messages contain required attributes (duration_ms, protocol_count, etc.)
  - [ ] 6.9: Test default (no-flag) execution path drops into daemon stub (AC: 5)

- [ ] Task 7: Verification (AC: all)
  - [ ] 7.1: Run `go build ./...` and verify success
  - [ ] 7.2: Run `go test ./cmd/extractor/...` and verify all pass
  - [ ] 7.3: Run `make lint` and verify no errors
  - [ ] 7.4: Manual smoke test: `./extractor --version`
  - [ ] 7.5: Manual smoke test: `./extractor --once --dry-run`

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

- (not yet generated; run *create-story-context to produce XML and add here)

### Agent Model Used

- Bob (GPT-5)

### Debug Log References

- Validation: docs/sprint-artifacts/validation-report-story-5-2-2025-12-01T19-43-28Z.md

### Completion Notes List

- Initialized Dev Agent Record after validation; no implementation work started

### File List

- docs/sprint-artifacts/5-2-implement-cli-and-single-run-mode.md (updated)
- docs/sprint-artifacts/validation-report-story-5-2-2025-12-01T19-43-28Z.md

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-12-01 | SM Agent (Bob) | Initial story draft created from epic-5 and tech-spec-epic-5.md |
| 2025-12-01 | SM Agent (Bob) | Initialized Dev Agent Record after validation |
