# Story 7.6: Integrate TVL Pipeline into Main Extraction Cycle

Status: ready-for-dev

## Story

As a **system operator**,
I want **the TVL charting pipeline to run automatically alongside the main oracle extraction in the 2-hour cycle**,
So that **both datasets are updated simultaneously with consistent timestamps, while ensuring main pipeline reliability is not affected by TVL pipeline failures**.

## Acceptance Criteria

Source: [Source: docs/epics/epic-7-custom-protocols-tvl-charting.md#Story-7.6]; [Source: docs/sprint-artifacts/tech-spec-epic-7.md#AC-7.6]

**AC1: TVL Pipeline Runs After Main Extraction**
**Given** the main oracle extraction completes successfully
**When** the extraction cycle continues
**Then** the TVL pipeline runs after main extraction
**And** both pipelines share the same extraction start timestamp for consistency
([Source: docs/sprint-artifacts/tech-spec-epic-7.md#AC-7.6]; [Source: docs/epics/epic-7-custom-protocols-tvl-charting.md#Story-7.6])

**AC2: TVL Pipeline Failures Are Isolated**
**Given** the TVL pipeline encounters an error (API failure, file write error, etc.)
**When** the error is handled
**Then** the main pipeline output (`switchboard-oracle-data.json`) is NOT affected
**And** the error is logged with `pipeline: tvl` attribute for filtering
**And** the extraction cycle completes with a partial success (main succeeded, TVL failed)
([Source: docs/sprint-artifacts/tech-spec-epic-7.md#AC-7.6])

**AC3: Logging Distinguishes Pipelines**
**Given** any log message during extraction
**When** the message relates to a specific pipeline
**Then** the log includes a `pipeline` attribute with value `main` or `tvl`
**And** TVL-specific operations are clearly identifiable in logs
([Source: docs/sprint-artifacts/tech-spec-epic-7.md#AC-7.6])

**AC4: Once Mode Runs Both Pipelines**
**Given** the CLI is invoked with `--once` flag
**When** the extraction runs
**Then** both main and TVL pipelines execute sequentially
**And** exit code is 0 if main succeeds (even if TVL fails)
**And** exit code is non-zero only if main pipeline fails
([Source: docs/sprint-artifacts/tech-spec-epic-7.md#AC-7.6])

**AC5: TVL Pipeline State Tracking**
**Given** the TVL pipeline runs
**When** determining whether to process
**Then** TVL state is tracked separately from main state (e.g., `tvl-state.json`)
**And** TVL pipeline skips processing if no changes detected
([Source: docs/epics/epic-7-custom-protocols-tvl-charting.md#Story-7.6])

**AC6: TVL Runs Even If Main Pipeline Fails**
**Given** the main pipeline fails or skips output
**When** RunOnce completes main execution path
**Then** the TVL pipeline still runs using cached/empty slugs if available
**And** any TVL errors remain isolated from main pipeline error handling
([Source: docs/sprint-artifacts/tech-spec-epic-7.md#AC-7.6])

**AC7: Dry Run Skips Writes**
**Given** the CLI runs with `--dry-run`
**When** both pipelines execute
**Then** no files are written for either pipeline
**And** logs still show both pipelines with shared extraction timestamp
([Source: docs/sprint-artifacts/tech-spec-epic-7.md#AC-7.6])

**AC8: Rate Limiting Enforced**
**Given** TVL protocol fetches are performed
**When** issuing successive TVL API calls
**Then** a minimum 200ms delay is enforced between fetches
([Source: docs/sprint-artifacts/tech-spec-epic-7.md#AC-7.6])

## Tasks / Subtasks

- [ ] Task 1: Add TVL Pipeline Runner Function (AC: 1, 2, 3, 6)
  - [ ] 1.1: Create `RunTVLPipeline(ctx, cfg, logger)` function in new file `internal/tvl/pipeline.go`
  - [ ] 1.2: Accept timestamp parameter from main extraction for consistency
  - [ ] 1.3: Orchestrate: LoadCustom -> GetAutoSlugs -> Merge -> FetchTVL -> Generate -> Write
  - [ ] 1.4: Add `pipeline: tvl` attribute to all log messages
  - [ ] 1.5: Return error on failure but design for caller to handle gracefully
  - [ ] 1.6: Respect context cancellation throughout

- [ ] Task 2: Implement TVL State Manager (AC: 5)
  - [ ] 2.1: Create `TVLStateManager` in `internal/tvl/state.go` (similar pattern to storage.StateManager)
  - [ ] 2.2: State file at `data/tvl-state.json` with fields: last_updated, protocol_count, custom_count
  - [ ] 2.3: Implement `ShouldProcess(currentTS)` for skip-if-no-changes logic
  - [ ] 2.4: Implement `LoadState()` and `SaveState()` with atomic writes
  - [ ] 2.5: Handle missing/corrupted state file gracefully (treat as first run)

- [ ] Task 3: Extract Auto-Detected Protocol Slugs (AC: 1)
  - [ ] 3.1: Create `GetAutoDetectedSlugs(oracleResp, oracleName)` function in `internal/tvl/slugs.go`
  - [ ] 3.2: Extract protocol slugs from oracle response that match the configured oracle name
  - [ ] 3.3: Return deduplicated slice of slugs for TVL fetching

- [ ] Task 4: Implement TVL Data Fetcher (AC: 1, 8)
  - [ ] 4.1: Create `FetchAllTVL(ctx, client, slugs)` function in `internal/tvl/fetcher.go`
  - [ ] 4.2: Iterate over merged protocol slugs and call `client.FetchProtocolTVL(slug)`
  - [ ] 4.3: Respect rate limiting (200ms between calls) via existing client logic
  - [ ] 4.4: Return `map[string]*api.ProtocolTVLResponse` for successful fetches
  - [ ] 4.5: Log warnings for 404 (protocol not found) but continue with others
  - [ ] 4.6: Count and log fetch statistics (total, success, not_found, failed)

- [ ] Task 5: Integrate TVL Pipeline into RunOnce (AC: 1, 2, 4, 6)
  - [ ] 5.1: Modify `runOnceWithDeps` in `cmd/extractor/main.go` to call TVL pipeline after main completes
  - [ ] 5.2: Add `tvlRunner` to `runDeps` struct for dependency injection
  - [ ] 5.3: Check `cfg.TVL.Enabled` before running TVL pipeline
  - [ ] 5.4: Wrap TVL pipeline call in error handler that logs but doesn't return error
  - [ ] 5.5: Pass oracle response to TVL pipeline for extracting auto-detected slugs
  - [ ] 5.6: Log TVL pipeline outcome (success/failure) with duration

- [ ] Task 6: Integrate TVL Pipeline into Daemon Mode (supporting AC: 1, 2, 3) 
  - [ ] 6.1: Verify daemon mode inherits TVL integration through RunOnce
  - [ ] 6.2: Confirm TVL failures don't affect daemon loop continuation
  - [ ] 6.3: Ensure "next extraction at" logs after both pipelines complete

- [ ] Task 7: Add Pipeline Logging Context (AC: 3)
  - [ ] 7.1: Add `pipeline` attribute to main extraction log messages where appropriate
  - [ ] 7.2: Ensure TVL pipeline logs include `pipeline: tvl` throughout
  - [ ] 7.3: Add summary log at end: `extraction_cycle_complete` with both pipeline statuses

- [ ] Task 8: Write Unit Tests (AC: all)
  - [ ] 8.1: Test: TVL pipeline runs after main extraction completes (AC1)
  - [ ] 8.2: Test: TVL pipeline failure does not affect main extraction success (AC2)
  - [ ] 8.3: Test: Logging includes `pipeline` attribute for both pipelines (AC3)
  - [ ] 8.4: Test: TVL pipeline executes in `--once` mode and respects exit-code rules (AC4)
  - [ ] 8.5: Test: TVL state tracking skip logic (`ShouldProcess`) works and state persists (AC5)
  - [ ] 8.6: Test: TVL runs even when main pipeline reports failure (AC6)
  - [ ] 8.7: Test: Dry-run mode skips file writes for both pipelines (AC7)
  - [ ] 8.8: Test: Rate limiting enforces ≥200ms between TVL fetch calls (AC8)
  - [ ] 8.9: Test: GetAutoDetectedSlugs extracts correct slugs from oracle response (supporting AC1)
  - [ ] 8.10: Test: FetchAllTVL handles mixed success/failure/404 responses (supporting AC1, AC8)

- [ ] Task 9: Build and Test Verification (AC: all)
  - [ ] 9.1: Run `go build ./...` and verify success
  - [ ] 9.2: Run `go test ./...` and verify all pass
  - [ ] 9.3: Run `./extractor --once` with TVL enabled and verify both outputs generated
  - [ ] 9.4: Run with TVL disabled and verify only main output generated
  - [ ] 9.5: Verify tvl-state.json created with correct structure

## Dev Notes

### Technical Guidance

**Integration Architecture:**

The TVL pipeline must be integrated into the existing extraction flow in `cmd/extractor/main.go`. The key integration point is after the main extraction writes complete, before the function returns. This ensures:

1. Main pipeline runs first and completes independently
2. TVL pipeline has access to oracle response for auto-detected slug extraction
3. TVL failures are logged but don't affect main pipeline success/exit codes

**Key Files to Create:**
- `internal/tvl/pipeline.go` - Main orchestrator for TVL extraction
- `internal/tvl/state.go` - TVL-specific state management
- `internal/tvl/slugs.go` - Auto-detected slug extraction from oracle response
- `internal/tvl/fetcher.go` - Parallel/sequential TVL data fetching

**Key Files to Modify:**
- `cmd/extractor/main.go` - Add TVL pipeline integration to `runOnceWithDeps`
- Potentially add TVL runner to `runDeps` struct for testability

**Existing Infrastructure to Reuse:**
- `internal/tvl/custom.go` - `CustomLoader` for loading custom protocols (Story 7.1)
- `internal/tvl/merger.go` - `MergeProtocolLists` for combining auto + custom (Story 7.3)
- `internal/tvl/output.go` - `GenerateTVLOutput`, `WriteTVLOutputs` (Stories 7.4, 7.5)
- `internal/api/client.go` - `FetchProtocolTVL` with rate limiting (Story 7.2)
- `internal/storage/state.go` - Pattern for `StateManager` implementation

**Pipeline Flow:**
```
RunOnce() {
  1. Main extraction pipeline (existing)
  2. If cfg.TVL.Enabled {
       RunTVLPipeline(ctx, cfg, oracleResp, logger)
     }
  3. Return (main pipeline result only)
}

RunTVLPipeline(ctx, cfg, oracleResp, logger) error {
  1. Load TVL state
  2. Load custom protocols (CustomLoader)
  3. Extract auto-detected slugs from oracleResp
  4. Merge protocol lists
  5. Check if processing needed (state comparison)
  6. If skip -> log and return nil
  7. Fetch TVL data for all protocols
  8. Generate TVL output
  9. Write TVL outputs (tvl-data.json, tvl-data.min.json)
  10. Update and save TVL state
  11. Return nil on success, error on failure
}
```

### Learnings from Previous Story

**From Story 7-5-generate-tvl-data-json-output (Status: done)**

- **GenerateTVLOutput Available**: `internal/tvl/output.go:59-82` - builds complete output from protocols + TVL data
- **WriteTVLOutputs Available**: `internal/tvl/output.go:86-138` - writes both indented and minified atomically
- **TVLOutput Schema**: `internal/models/tvl.go:39-45` - Version, Metadata, Protocols map
- **Atomic Write Pattern**: Uses `storage.WriteJSON` which internally uses `WriteAtomic`
- **Context Cancellation**: `WriteTVLOutputs` respects context and cleans up on cancellation
- **Metadata Calculation**: Protocol counts calculated during generation, timestamp set to now
- **Completion Notes (2025-12-08)**: Story 7.5 marked done; Senior Developer Review approved with no outstanding action items; completion notes captured for handoff to this story. [Source: docs/sprint-artifacts/7-5-generate-tvl-data-json-output.md#Completion-Notes-List]

**From Story 7.3-merge-protocol-lists:**
- `MergeProtocolLists(autoSlugs, customProtocols)` deduplicates by slug, custom takes precedence

**From Story 7.2-implement-protocol-tvl-fetcher:**
- `client.FetchProtocolTVL(ctx, slug)` with 200ms rate limiting built in
- 404 returns nil, nil (not error) - logged as warning

**From Story 7.1-load-custom-protocols:**
- `CustomLoader.Load(ctx)` handles missing file gracefully (empty list)

[Source: docs/sprint-artifacts/7-5-generate-tvl-data-json-output.md#Dev-Agent-Record]

### Architecture Patterns and Constraints

- **ADR-002**: Atomic file writes (temp + rename) for tvl-state.json [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-002]
- **ADR-003**: Explicit error returns; TVL pipeline returns error for logging but main ignores it [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-003]
- **ADR-004**: Structured logging with `pipeline` attribute for filtering [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-004]
- **ADR-005**: No new external dependencies [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-005]
- **Error Isolation**: TVL failures must not affect main pipeline - use pattern: `if err := runTVL(...); err != nil { logger.Error(...); }` (no return)

### Configuration Reference

**Existing TVLConfig (internal/config/config.go:56-59):**
```go
type TVLConfig struct {
    CustomProtocolsPath string `yaml:"custom_protocols_path"`
    Enabled             bool   `yaml:"enabled"`
}
```

**Default values:**
- `custom_protocols_path`: `config/custom-protocols.json`
- `enabled`: `true`

**Environment overrides:**
- `TVL_CUSTOM_PROTOCOLS_PATH`
- `TVL_ENABLED`

### Project Structure Notes

- New files in `internal/tvl/` package following existing patterns
- State file at `data/tvl-state.json` alongside `data/state.json`
- Output files at `data/tvl-data.json` and `data/tvl-data.min.json`
- Follow existing `StateManager` pattern from `internal/storage/state.go`
- Align directory placement with established layout [Source: docs/architecture/project-structure.md]

### Testing Strategy

- Table-driven tests with named cases [Source: docs/architecture/testing-strategy.md]
- Mock the API client using interfaces for TVL fetcher tests
- Test error isolation: TVL failure + main success = overall success
- Integration test: full RunOnce with TVL enabled produces both output files

### References

- [Source: docs/epics/epic-7-custom-protocols-tvl-charting.md#Story-7.6] - Epic story definition
- [Source: docs/sprint-artifacts/tech-spec-epic-7.md#AC-7.6] - Tech spec acceptance criteria
- [Source: docs/prd.md#CLI-Operation] - Daemon and once mode behavior (FR42-FR48)
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-002] - Atomic file writes
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-003] - Error handling
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-004] - Structured logging
- [Source: docs/sprint-artifacts/7-5-generate-tvl-data-json-output.md#Dev-Agent-Record] - Previous story learnings
- [Source: cmd/extractor/main.go:96-267] - RunOnce and runOnceWithDeps functions
- [Source: internal/tvl/output.go:59-138] - GenerateTVLOutput and WriteTVLOutputs
- [Source: internal/tvl/merger.go:13-54] - MergeProtocolLists function
- [Source: internal/tvl/custom.go:35-96] - CustomLoader.Load function
- [Source: internal/api/client.go:330-358] - FetchProtocolTVL with rate limiting
- [Source: internal/storage/state.go:29-173] - StateManager pattern to follow
- [Source: internal/config/config.go:56-59] - TVLConfig structure
- [Source: docs/architecture/project-structure.md] - Directory conventions and placement

## Dev Agent Record

### Context Reference

docs/sprint-artifacts/7-6-integrate-tvl-pipeline-into-main-extraction-cycle.context.xml

### Agent Model Used

gpt-5 (Scrum Master auto-fix)

### Debug Log References

- Pending — log entries to be added during implementation/test runs

### Completion Notes List

- 2025-12-08: Auto-fix applied to align ACs with tech spec, add citations, and initialize Dev Agent Record

### File List

- docs/sprint-artifacts/7-6-integrate-tvl-pipeline-into-main-extraction-cycle.md

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-12-08 | SM Agent (Bob) | Initial story draft created from Epic 7 |
