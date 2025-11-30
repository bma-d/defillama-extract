# Story 4.1: Implement State File Structure and Loading

Status: done

## Story

As a **developer**,
I want **to load the last extraction state from a JSON file**,
so that **I can determine if new data is available**.

## Acceptance Criteria

Source: Epic 4.1 / Tech Spec AC-4.1 / [Source: docs/prd.md#fr25-fr29]

1. **Given** a state file exists at `{outputDir}/state.json` **When** `LoadState()` is called **Then** a `*State` struct is returned containing all fields populated:
   - `OracleName`: the oracle being tracked
   - `LastUpdated`: Unix timestamp of last processed data
   - `LastUpdatedISO`: human-readable ISO 8601 timestamp
   - `LastProtocolCount`: number of protocols in last extraction
   - `LastTVS`: total TVS from last extraction

2. **Given** no state file exists **When** `LoadState()` is called **Then** a zero-value `*State` is returned (not an error) **And** `LastUpdated = 0` indicates first run

3. **Given** a corrupted/invalid state file (invalid JSON, partial write) **When** `LoadState()` is called **Then** a warning is logged with path and error details **And** a zero-value `*State` is returned (graceful recovery per FR28) **And** extraction proceeds as if first run

## Tasks / Subtasks

- [x] Task 1: Define State struct (AC: 1)
  - [x] 1.1: Add `State` struct to `internal/storage/state.go` with fields: `OracleName`, `LastUpdated`, `LastUpdatedISO`, `LastProtocolCount`, `LastTVS`, `SnapshotCount`, `OldestSnapshot`, `NewestSnapshot`
  - [x] 1.2: Add JSON struct tags for all fields (snake_case)
  - [x] 1.3: Add doc comment explaining struct purpose and zero-value semantics

- [x] Task 2: Define StateManager struct (AC: 1, implementation guidance)
  - [x] 2.1: Add `StateManager` struct with fields: `outputDir`, `stateFile`, `outputFile`, `logger`
  - [x] 2.2: Implement `NewStateManager(outputDir string, logger *slog.Logger) *StateManager` constructor
  - [x] 2.3: Constructor sets `stateFile = outputDir + "/state.json"` and `outputFile = outputDir + "/switchboard-oracle-data.json"`

- [x] Task 3: Implement LoadState method (AC: 1, 2, 3)
  - [x] 3.1: Add `func (sm *StateManager) LoadState() (*State, error)` method
  - [x] 3.2: Use `os.ReadFile(sm.stateFile)` to read file
  - [x] 3.3: Handle `os.ErrNotExist` → return `&State{}`, nil (first run)
  - [x] 3.4: Handle other read errors → return nil, wrapped error
  - [x] 3.5: Use `json.Unmarshal()` to parse JSON
  - [x] 3.6: Handle JSON parse error → log warning, return `&State{}`, nil (graceful recovery)
  - [x] 3.7: On success → log debug with state attributes, return populated state

- [x] Task 4: Write unit tests (AC: 1-3)
  - [x] 4.1: Create `internal/storage/state_test.go`
  - [x] 4.2: Test: NewStateManager creates instance with correct paths
  - [x] 4.3: Test: LoadState with valid JSON returns populated State
  - [x] 4.4: Test: LoadState with missing file returns zero-value State (no error)
  - [x] 4.5: Test: LoadState with corrupted JSON returns zero-value State, logs warning
  - [x] 4.6: Test: LoadState with partial JSON returns zero-value State, logs warning
  - [x] 4.7: Test: State struct JSON marshaling/unmarshaling round-trip
  - [x] 4.8: Create testdata fixtures: `valid_state.json`, `corrupted_state.json`

- [x] Task 5: Verification (AC: all)
  - [x] 5.1: Run `go build ./...` and verify success
  - [x] 5.2: Run `go test ./internal/storage/...` and verify all pass
  - [x] 5.3: Run `make lint` and verify no errors

## Dev Notes

### Technical Guidance

- **Package Location:** `internal/storage/state.go` (existing `doc.go` placeholder)
- **Pattern:** Manager pattern with encapsulated paths and logger
- **Dependencies:** Go stdlib only (`encoding/json`, `os`, `log/slog`, `errors`)
- **Error Handling:** ADR-003 - explicit error returns, wrap with context
- **Logging:** ADR-004 - structured logging with slog

### State Struct Definition

```go
// State represents the last extraction state for incremental update tracking.
// A zero-value State (LastUpdated == 0) indicates first run or corrupted state recovery.
type State struct {
    OracleName        string  `json:"oracle_name"`
    LastUpdated       int64   `json:"last_updated"`        // Unix timestamp
    LastUpdatedISO    string  `json:"last_updated_iso"`    // ISO 8601 for humans
    LastProtocolCount int     `json:"last_protocol_count"`
    LastTVS           float64 `json:"last_tvs"`
    SnapshotCount     int     `json:"snapshot_count"`
    OldestSnapshot    int64   `json:"oldest_snapshot"`
    NewestSnapshot    int64   `json:"newest_snapshot"`
}
```

### StateManager Implementation Pattern

```go
// StateManager handles state file operations for incremental updates.
type StateManager struct {
    outputDir  string
    stateFile  string        // {outputDir}/state.json
    outputFile string        // {outputDir}/switchboard-oracle-data.json
    logger     *slog.Logger
}

// NewStateManager creates a StateManager for the given output directory.
func NewStateManager(outputDir string, logger *slog.Logger) *StateManager {
    return &StateManager{
        outputDir:  outputDir,
        stateFile:  filepath.Join(outputDir, "state.json"),
        outputFile: filepath.Join(outputDir, "switchboard-oracle-data.json"),
        logger:     logger,
    }
}
```

### LoadState Method Pattern

```go
// LoadState reads the state file and returns the current state.
// Returns zero-value State if file doesn't exist or is corrupted (graceful recovery).
func (sm *StateManager) LoadState() (*State, error) {
    data, err := os.ReadFile(sm.stateFile)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            sm.logger.Debug("no state file found, first run", "path", sm.stateFile)
            return &State{}, nil
        }
        return nil, fmt.Errorf("failed to read state file: %w", err)
    }

    var state State
    if err := json.Unmarshal(data, &state); err != nil {
        sm.logger.Warn("state file corrupted, treating as first run",
            "path", sm.stateFile,
            "error", err.Error(),
        )
        return &State{}, nil
    }

    sm.logger.Debug("state loaded",
        "oracle_name", state.OracleName,
        "last_updated", state.LastUpdated,
        "snapshot_count", state.SnapshotCount,
    )
    return &state, nil
}
```

### Project Structure Notes

- **Existing Package:** `internal/storage/doc.go` exists as placeholder
- **New Files:** `state.go`, `state_test.go` to be created
- **Test Fixtures:** Create `internal/storage/testdata/` directory
- **Shared Types:** `Snapshot` struct exists in `internal/aggregator/models.go` - will be referenced in later stories

### Learnings from Previous Story

**From Story 3-7-build-complete-aggregation-pipeline (Status: done)**

- **New/Modified files to reuse:** `internal/aggregator/aggregator.go`, `internal/aggregator/aggregator_test.go`, `internal/aggregator/models.go`, `docs/sprint-artifacts/sprint-status.yaml`, `docs/sprint-artifacts/3-7-build-complete-aggregation-pipeline.md`.
- **Completion notes carried forward:** Aggregator orchestrator pattern validated; build/test/lint workflow confirmed; no outstanding review items.
- **Orchestrator Pattern:** StateManager should follow similar manager pattern to Aggregator.
- **Struct Location:** State struct stays in implementation file (storage-specific).
- **Test Patterns:** Keep table-driven tests per aggregator package conventions.
- **Nil Safety:** Handle nil/empty inputs gracefully (return zero values, not errors).

[Source: docs/sprint-artifacts/3-7-build-complete-aggregation-pipeline.md#Dev-Agent-Record]

### Architecture References

- **Atomic writes (ADR-002):** Write to temp file, then atomic rename [Source: docs/architecture/architecture-decision-records-adrs.md#adr-002]
- **Error handling (ADR-003):** Explicit error returns, wrap with context [Source: docs/architecture/architecture-decision-records-adrs.md#adr-003]
- **Structured logging (ADR-004):** Use slog with JSON output [Source: docs/architecture/architecture-decision-records-adrs.md#adr-004]
- **State loading flow:** Tech spec defines exact flow [Source: docs/sprint-artifacts/tech-spec-epic-4.md#workflows-and-sequencing]
- **Data model:** State struct fields from tech spec [Source: docs/sprint-artifacts/tech-spec-epic-4.md#data-models-and-contracts]

### Testing Standards

- Follow Go table-driven tests pattern
- Use `t.TempDir()` for test directories
- Test all error paths: missing file, corrupted file, read error
- Verify log output using custom handler or mock logger

### Smoke Test Guide

**Manual verification after implementation:**

1. Create valid `state.json` in `/tmp/test/`:
   ```json
   {"oracle_name":"defillama","last_updated":1700000000,"last_updated_iso":"2023-11-14T22:00:00Z","last_protocol_count":50,"last_tvs":1500000000.0,"snapshot_count":10,"oldest_snapshot":1699000000,"newest_snapshot":1700000000}
   ```
2. Run: `go test -v ./internal/storage/... -run TestLoadState`
3. Verify: State populated with all fields
4. Delete state.json, rerun: Verify zero-value State returned
5. Create invalid JSON `{"oracle_name":`, rerun: Verify warning logged, zero-value returned

### FR Coverage

| FR | Description | Satisfied By |
|----|-------------|--------------|
| FR25 | Track last processed timestamp | `State.LastUpdated` field |
| FR28 | Recover gracefully from corrupted state | `LoadState` returns zero-value on JSON error |

### References

- [Source: docs/prd.md#fr25-fr29] - Functional requirements for state and incremental updates
- [Source: docs/epics/epic-4-state-history-management.md#story-41] - Story definition and acceptance criteria
- [Source: docs/sprint-artifacts/tech-spec-epic-4.md#ac-41] - Authoritative acceptance criteria
- [Source: docs/sprint-artifacts/tech-spec-epic-4.md#data-models-and-contracts] - State struct definition
- [Source: docs/sprint-artifacts/tech-spec-epic-4.md#workflows-and-sequencing] - LoadState flow diagram
- [Source: docs/architecture/architecture-decision-records-adrs.md] - ADRs for patterns
- [Source: docs/architecture/testing-strategy.md#test-organization] - Testing conventions

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/4-1-implement-state-file-structure-and-loading.context.xml

### Agent Model Used

{{agent_model_name_version}}

### Completion Notes
**Completed:** 2025-11-30  
**Definition of Done:** All acceptance criteria met, code reviewed, tests passing

### Debug Log References

- Planned and implemented `State` + `StateManager` per ACs; constructor builds canonical paths, nil logger defaults to slog default.
- `LoadState` handles missing file (first run), read errors (wrapped), and JSON corruption (warn + zero-value) with structured logging.
- Tests cover happy path, missing file, corrupted/partial JSON, path construction, JSON round-trip; fixtures under `internal/storage/testdata`.

### Completion Notes List

- Implemented `state.go` with `State` struct, `StateManager`, and `LoadState` covering AC1–AC3 with structured debug/warn logs.
- Added table-driven unit tests plus fixtures for valid, corrupted, and partial state files; verified constructor path setup and JSON round-trip.
- Verification run: `go build ./...`, `go test ./internal/storage/...`, `make lint`.

### File List

- internal/storage/state.go
- internal/storage/state_test.go
- internal/storage/testdata/valid_state.json
- internal/storage/testdata/corrupted_state.json
- internal/storage/testdata/partial_state.json
- docs/sprint-artifacts/sprint-status.yaml

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-4-state-history-management.md and tech-spec-epic-4.md |
| 2025-11-30 | Amelia (Dev Agent) | Implemented StateManager LoadState with tests and fixtures; marked story ready for review |
| 2025-11-30 | Amelia (Dev Agent) | Senior Developer Review (AI) approved; notes appended |

## Senior Developer Review (AI)

**Reviewer:** BMad  
**Date:** 2025-11-30  
**Outcome:** Approve (all ACs implemented; no issues found)

### Summary
- LoadState meets AC1–AC3; zero regressions detected.
- Tests, build, lint all passing locally.

### Key Findings
- None.

### Acceptance Criteria Coverage
| AC# | Description | Status | Evidence |
|-----|-------------|--------|----------|
| AC1 | LoadState returns populated State when state.json exists | Implemented | internal/storage/state.go:49-77 |
| AC2 | Missing state file returns zero-value State, no error | Implemented | internal/storage/state.go:53-58 |
| AC3 | Corrupted/invalid JSON logs warning and returns zero-value State | Implemented | internal/storage/state.go:62-66 |

Summary: 3/3 ACs implemented.

### Task Completion Validation
| Task | Marked As | Verified As | Evidence |
|------|-----------|-------------|----------|
| Define State struct + JSON tags | Done | Verified complete | internal/storage/state.go:12-24 |
| Define StateManager struct + constructor paths | Done | Verified complete | internal/storage/state.go:26-47 |
| Implement LoadState handling missing/corrupted files | Done | Verified complete | internal/storage/state.go:49-77 |
| Unit tests + fixtures (valid/missing/corrupted/partial, round-trip) | Done | Verified complete | internal/storage/state_test.go:17-175; internal/storage/testdata/*.json |
| Verification commands (build, tests, lint) | Done | Verified complete | go build ./...; go test ./internal/storage/...; make lint |

Completed tasks: 5/5; Questionable: 0; False completions: 0.

### Test Coverage and Gaps
- go test ./internal/storage/... (pass)
- go build ./... (pass)
- make lint (pass)
- No gaps noted for AC1–AC3 paths.

### Architectural Alignment
- Uses slog per ADR-004; stdlib-only per tech spec; path construction via filepath.Join.

### Security Notes
- No secrets handled; file I/O limited to configured outputDir; logs do not leak data.

### Best-Practices and References
- Stack: Go 1.24; slog JSON handler for structured logs.

### Action Items
**Code Changes Required:**
- [ ] None

**Advisory Notes:**
- Note: None
