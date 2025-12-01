# Story 4.8: Build State Manager Component

Status: done

## Story

As a **developer**,
I want **a unified StateManager that coordinates all state and history operations**,
so that **I have a clean, consistent interface for managing incremental updates during extraction cycles**.

## Acceptance Criteria

Source: [Source: docs/sprint-artifacts/tech-spec-epic-4.md#AC-4.8] / [Source: docs/epics/epic-4-state-history-management.md#story-48-build-state-manager-component]

1. **Given** configuration with output directory **When** `NewStateManager(outputDir, logger)` is called **Then** a `StateManager` is created with paths configured:
   - State file: `{outputDir}/state.json`
   - Output file: `{outputDir}/switchboard-oracle-data.json`

2. **Given** a StateManager instance **When** extraction cycle starts **Then** `LoadState()` returns current state **And** state can be used with `ShouldProcess(timestamp)` to determine if processing needed

3. **Given** a successful extraction producing an `AggregationResult` **When** `UpdateState(oracleName, timestamp, count, tvs, snapshots)` is called **Then** a new `*State` is returned with all fields populated:
   - `OracleName`: the oracle being tracked
   - `LastUpdated`: Unix timestamp of current extraction
   - `LastUpdatedISO`: human-readable ISO 8601 timestamp
   - `LastProtocolCount`: number of protocols in current extraction
   - `LastTVS`: total TVS from current extraction
   - `SnapshotCount`: length of snapshots slice
   - `OldestSnapshot`: timestamp of first snapshot
   - `NewestSnapshot`: timestamp of last snapshot

4. **Given** a StateManager instance **When** history operations are needed **Then** `LoadFromOutput` and `AppendSnapshot` are available (via HistoryManager methods or StateManager methods)

5. **Given** a complete extraction cycle **When** `SaveState(state)` is called **Then** state is persisted atomically **And** state and history are consistent

## Tasks / Subtasks

- [x] Task 1: Add `UpdateState` method to StateManager (AC: 3)
  - [x] 1.1: Implement `UpdateState(oracleName string, ts int64, count int, tvs float64, snapshots []Snapshot) *State` in `internal/storage/state.go`
  - [x] 1.2: Populate all State fields including `SnapshotCount`, `OldestSnapshot`, `NewestSnapshot` from snapshots slice
  - [x] 1.3: Format `LastUpdatedISO` as ISO 8601 using `time.Unix(ts, 0).UTC().Format(time.RFC3339)`
  - [x] 1.4: Handle edge case: empty snapshots slice (set snapshot metadata to 0)

- [x] Task 2: Add history accessor methods to StateManager (AC: 4)
  - [x] 2.1: Add `LoadHistory() ([]aggregator.Snapshot, error)` method that delegates to `LoadFromOutput(sm.outputFile, sm.logger)`
  - [x] 2.2: Add `AppendSnapshot(history []aggregator.Snapshot, snapshot aggregator.Snapshot) []aggregator.Snapshot` method that delegates to package-level `AppendSnapshot`
  - [x] 2.3: Ensure StateManager provides unified interface for both state and history operations

- [x] Task 3: Add `OutputFile()` accessor method (AC: 1, 4)
  - [x] 3.1: Add `OutputFile() string` method to return the output file path for use by downstream consumers (Epic 5 writer)

- [x] Task 4: Write unit tests for new methods (AC: 1-5)
  - [x] 4.1: Test `UpdateState` with valid inputs - verify all fields populated correctly
  - [x] 4.2: Test `UpdateState` with empty snapshots slice - verify snapshot metadata is 0
  - [x] 4.3: Test `UpdateState` ISO 8601 timestamp formatting
  - [x] 4.4: Test `LoadHistory` delegation (uses existing LoadFromOutput tests)
  - [x] 4.5: Test `AppendSnapshot` delegation (uses existing tests)
  - [x] 4.6: Test `OutputFile` returns correct path

- [x] Task 5: Write integration test for full extraction cycle (AC: 5)
  - [x] 5.1: Test complete cycle: LoadState -> ShouldProcess -> (simulate extraction) -> UpdateState -> SaveState
  - [x] 5.2: Verify state file contains correct values after cycle
  - [x] 5.3: Verify state and history consistency after multiple cycles

- [x] Task 6: Verification (AC: all)
  - [x] 6.1: Run `go build ./...` and verify success
  - [x] 6.2: Run `go test ./internal/storage/...` and verify all pass
  - [x] 6.3: Run `make lint` and verify no errors

## Dev Notes

### Technical Guidance

- **Package Location:** `internal/storage/state.go` (add methods to existing StateManager)
- **Story Nature:** This story completes the StateManager interface by adding the `UpdateState` method and history accessor methods for a unified API
- **Key Insight:** Most functionality already exists from Stories 4.1-4.7. This story unifies the interface.
- **FR Coverage:** FR25-FR34 (all State & History Management FRs consolidated into unified component)

### Current StateManager Status (from Stories 4.1-4.3)

The following already exists in `internal/storage/state.go`:
- `State` struct with all fields
- `NewStateManager(outputDir, logger)` constructor
- `LoadState() (*State, error)` - loads state from disk
- `SaveState(state *State) error` - atomic save
- `ShouldProcess(currentTS int64, state *State) bool` - skip logic

### Current History Functions (from Stories 4.4-4.7)

The following already exists in `internal/storage/history.go`:
- `CreateSnapshot(result *AggregationResult) Snapshot` - creates snapshot from aggregation result
- `LoadFromOutput(outputPath string, logger *slog.Logger) ([]Snapshot, error)` - loads history from output file
- `AppendSnapshot(history []Snapshot, snapshot Snapshot, logger *slog.Logger) []Snapshot` - appends with deduplication

### What This Story Adds

1. **`UpdateState` method** - Creates a fully populated State from extraction results:
   ```go
   func (sm *StateManager) UpdateState(oracleName string, ts int64, count int, tvs float64, snapshots []aggregator.Snapshot) *State {
       state := &State{
           OracleName:        oracleName,
           LastUpdated:       ts,
           LastUpdatedISO:    time.Unix(ts, 0).UTC().Format(time.RFC3339),
           LastProtocolCount: count,
           LastTVS:           tvs,
           SnapshotCount:     len(snapshots),
       }
       if len(snapshots) > 0 {
           state.OldestSnapshot = snapshots[0].Timestamp
           state.NewestSnapshot = snapshots[len(snapshots)-1].Timestamp
       }
       return state
   }
   ```

2. **History accessor methods** - Convenience methods that delegate to package-level functions:
   ```go
   func (sm *StateManager) LoadHistory() ([]aggregator.Snapshot, error) {
       return LoadFromOutput(sm.outputFile, sm.logger)
   }

   func (sm *StateManager) AppendSnapshot(history []aggregator.Snapshot, snapshot aggregator.Snapshot) []aggregator.Snapshot {
       return AppendSnapshot(history, snapshot, sm.logger)
   }
   ```

3. **`OutputFile` accessor** - Returns output file path for downstream consumers:
   ```go
   func (sm *StateManager) OutputFile() string {
       return sm.outputFile
   }
   ```

### Learnings from Previous Story

**From Story 4-7-implement-history-retention-keep-all (Status: done)**

- **Files Modified:** `internal/storage/history.go`, `internal/storage/history_test.go`
- **Key Implementation:** Retention policy documented; no pruning code exists
- **Patterns Established:**
  - Table-driven tests with fixtures
  - `slog.Default()` fallback for nil logger
  - History functions maintain timestamp sort order
  - Use `aggregator.Snapshot` from `internal/aggregator/models.go`
- **Review Outcome:** Approved with no blocking findings
- **Build Status:** All build, test, and lint checks pass

[Source: docs/sprint-artifacts/4-7-implement-history-retention-keep-all.md#Dev-Agent-Record]

### Project Structure Notes

- **Files to Modify:** `internal/storage/state.go`, `internal/storage/state_test.go`
- **No New Files:** All additions go into existing state.go
- **Import:** May need to add `time` import for RFC3339 formatting
- **Structure Compliance:** All changes scoped to `internal/storage/` per [Source: docs/architecture/project-structure.md]

### Testing Standards

- Follow Go table-driven tests pattern per [Source: docs/architecture/testing-strategy.md#Test-Organization]
- Use `aggregator.Snapshot{}` literals for test fixtures
- Test edge cases: empty snapshots, nil inputs
- Integration test for full cycle: load -> process decision -> update -> save

### Smoke Test Guide

**Manual verification after implementation:**

1. Run unit tests:
   ```bash
   go test -v ./internal/storage/... -run TestUpdateState
   go test -v ./internal/storage/... -run TestStateManager
   ```

2. Verify full cycle in a test:
   ```go
   sm := NewStateManager("/tmp/test-output", nil)

   // Load initial state (should be zero-value)
   state, _ := sm.LoadState()

   // Check if processing needed (should be true for first run)
   if sm.ShouldProcess(1700000000, state) {
       // Simulate extraction...
       snapshots := []aggregator.Snapshot{
           {Timestamp: 1700000000, TVS: 1000000.0},
       }

       // Update state
       newState := sm.UpdateState("Switchboard", 1700000000, 21, 1000000.0, snapshots)

       // Save state
       sm.SaveState(newState)
   }

   // Reload and verify
   reloaded, _ := sm.LoadState()
   // reloaded.LastUpdated should be 1700000000
   // reloaded.SnapshotCount should be 1
   ```

3. Verify method availability:
   ```bash
   go doc github.com/switchboard-xyz/defillama-extract/internal/storage StateManager
   # Should list: LoadState, SaveState, ShouldProcess, UpdateState, LoadHistory, AppendSnapshot, OutputFile
   ```

### FR Coverage

| FR | Description | Satisfied By |
|----|-------------|--------------|
| FR25 | Track last processed timestamp in state file | `UpdateState` populates `LastUpdated` |
| FR26 | Compare latest API timestamp against last processed | `ShouldProcess` (existing) |
| FR27 | Skip processing when no new data | `ShouldProcess` (existing) |
| FR28 | Recover gracefully from corrupted state file | `LoadState` (existing) |
| FR29 | Update state file atomically after successful extraction | `SaveState` (existing) |
| FR30-FR34 | Historical snapshot management | `LoadHistory`, `AppendSnapshot` accessors |

### References

- [Source: docs/prd.md#FR25-FR34] - State & History Management requirements
- [Source: docs/epics/epic-4-state-history-management.md#story-48] - Story definition and acceptance criteria
- [Source: docs/sprint-artifacts/tech-spec-epic-4.md#AC-4.8] - Authoritative acceptance criteria
- [Source: docs/sprint-artifacts/tech-spec-epic-4.md#Services-and-Modules] - StateManager interface design
- [Source: internal/storage/state.go] - Existing StateManager implementation
- [Source: internal/storage/history.go] - Existing history functions
- [Source: docs/sprint-artifacts/4-7-implement-history-retention-keep-all.md] - Previous story implementation
- [Source: docs/architecture/testing-strategy.md#Test-Organization] - Testing standards

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/4-8-build-state-manager-component.context.xml

### Agent Model Used
Amelia (GPT-5)

### Debug Log References

- Implemented AC3/AC4 methods in `internal/storage/state.go` (UpdateState, LoadHistory, AppendSnapshot, OutputFile)
- Added unit and integration coverage for UpdateState, history delegation, output path accessor, and full cycle flow in `internal/storage/state_test.go`
- Executed validations: `go build ./...`, `go test ./internal/storage/...`, `make lint`

### Completion Notes List

- Added StateManager UpdateState with RFC3339 ISO formatting and snapshot metadata handling (AC3)
- Added history/output accessors to unify API surface (AC1, AC4)
- Expanded storage tests with delegation checks and full cycle integration; all storage tests passing (AC5)

### File List

- internal/storage/state.go
- internal/storage/state_test.go

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-12-01 | SM Agent (Bob) | Initial story draft created from epic-4 and tech-spec-epic-4.md |
| 2025-12-01 | Amelia | Implemented StateManager update/history accessors and tests; build/test/lint passing |
| 2025-12-01 | Amelia | Senior Developer Review (AI) outcome: Approved after adding multi-cycle integration test coverage |
| 2025-12-01 | Amelia | Senior Developer Review (AI) outcome: Approved with no new findings; status moved to done |

## Senior Developer Review (AI)
Reviewer: BMad
Date: 2025-12-01
Outcome: Approve — all ACs satisfied with multi-cycle coverage added

Summary
- Multi-cycle integration test added; state/history consistency validated across successive runs.

Key Findings
- None.

Acceptance Criteria Coverage
| AC | Status | Evidence |
|----|--------|----------|
| AC1 | Implemented | internal/storage/state.go:37-49; internal/storage/state_test.go:21-35 |
| AC2 | Implemented | internal/storage/state.go:51-136; internal/storage/state_test.go:155-255 |
| AC3 | Implemented | internal/storage/state.go:139-156; internal/storage/state_test.go:393-434 |
| AC4 | Implemented | internal/storage/state.go:159-167; internal/storage/state_test.go:436-485 |
| AC5 | Implemented | internal/storage/state.go:82-104; internal/storage/state_test.go:497-555,558-641 |

Task Completion Validation
| Task | Marked As | Verified As | Evidence |
|------|-----------|-------------|----------|
| 1.1 UpdateState implemented | [x] | Verified | internal/storage/state.go:139-156 |
| 1.2 Populate State fields incl snapshot metadata | [x] | Verified | internal/storage/state.go:139-156 |
| 1.3 ISO 8601 formatting | [x] | Verified | internal/storage/state.go:143-146; internal/storage/state_test.go:393-420 |
| 1.4 Empty snapshots edge case | [x] | Verified | internal/storage/state.go:151-154; internal/storage/state_test.go:423-434 |
| 2.1 LoadHistory accessor | [x] | Verified | internal/storage/state.go:159-162; internal/storage/state_test.go:436-459 |
| 2.2 AppendSnapshot accessor | [x] | Verified | internal/storage/state.go:164-167; internal/storage/state_test.go:462-485 |
| 2.3 Unified interface for state/history | [x] | Verified | internal/storage/state.go:159-171 |
| 3.1 OutputFile accessor | [x] | Verified | internal/storage/state.go:169-171; internal/storage/state_test.go:487-495 |
| 4.1 Test UpdateState valid inputs | [x] | Verified | internal/storage/state_test.go:393-420 |
| 4.2 Test UpdateState empty snapshots | [x] | Verified | internal/storage/state_test.go:423-434 |
| 4.3 Test UpdateState ISO formatting | [x] | Verified | internal/storage/state_test.go:393-420 |
| 4.4 Test LoadHistory delegation | [x] | Verified | internal/storage/state_test.go:436-459 |
| 4.5 Test AppendSnapshot delegation | [x] | Verified | internal/storage/state_test.go:462-485 |
| 4.6 Test OutputFile accessor | [x] | Verified | internal/storage/state_test.go:487-495 |
| 5.1 Integration: LoadState → ShouldProcess → UpdateState → SaveState | [x] | Verified | internal/storage/state_test.go:497-555 |
| 5.2 Verify state file values after cycle | [x] | Verified | internal/storage/state_test.go:520-529 |
| 5.3 Verify state/history consistency after multiple cycles | [x] | Verified | internal/storage/state_test.go:558-641 |
| 6.1 Run `go build ./...` | [x] | Verified | Executed 2025-12-01 |
| 6.2 Run `go test ./internal/storage/...` | [x] | Verified | Executed 2025-12-01 |
| 6.3 Run `make lint` | [x] | Verified | Executed 2025-12-01 |

Test Coverage and Gaps
- go test ./internal/storage/... (pass, 2025-12-01)
- go build ./... (pass, 2025-12-01)
- make lint (pass, 2025-12-01)
- No gaps remaining for ACs or declared tasks.

Architectural Alignment
- Atomic write pattern preserved (internal/storage/writer.go) per docs/architecture/implementation-patterns.md#atomic-file-writes.
- Structured logging maintained per docs/architecture/architecture-decision-records-adrs.md#adr-004-structured-logging-with-slog.

Security Notes
- No new security surface; file I/O confined to configured outputDir.

Best-Practices and References
- Go 1.24 stdlib stack with slog; minimal deps (gopkg.in/yaml.v3). See docs/architecture/implementation-patterns.md and docs/architecture/architecture-decision-records-adrs.md.

Action Items
**Code Changes Required:**
- None (all review findings resolved).

**Advisory Notes:**
- None.

## Senior Developer Review (AI)
Reviewer: BMad  
Date: 2025-12-01  
Outcome: Approve — all ACs and tasks verified; no new findings.

Summary
- Verified AC1–AC5 against code/tests; repository build, tests, and lint all pass (`go build ./...`, `go test ./...`, `make lint` on 2025-12-01).
- Architecture reviewed via `docs/architecture/index.md` and linked shards (`project-structure.md`, `implementation-patterns.md`, tech spec).

Key Findings
- None.

Acceptance Criteria Coverage (5/5 implemented)
| AC | Description | Status | Evidence |
|----|-------------|--------|----------|
| AC1 | StateManager uses configured paths | Implemented | internal/storage/state.go:37-49; internal/storage/state_test.go:21-38 |
| AC2 | LoadState/ShouldProcess enable cycle gating | Implemented | internal/storage/state.go:51-136; internal/storage/state_test.go:41-170,497-533 |
| AC3 | UpdateState populates state metadata incl. ISO timestamp | Implemented | internal/storage/state.go:139-156; internal/storage/state_test.go:393-434 |
| AC4 | History accessors exposed | Implemented | internal/storage/state.go:159-167; internal/storage/state_test.go:436-485 |
| AC5 | State persists atomically and aligns with history across cycles | Implemented | internal/storage/state.go:82-104; internal/storage/state_test.go:497-555,557-660 |

Task Completion Validation (20/20 verified, 0 questionable, 0 false completions)
| Task/Subtask | Marked As | Verified As | Evidence |
|--------------|-----------|-------------|----------|
| 1.1 UpdateState implementation | [x] | Verified | internal/storage/state.go:139-156 |
| 1.2 Snapshot metadata populated | [x] | Verified | internal/storage/state.go:151-154; internal/storage/state_test.go:393-434 |
| 1.3 ISO 8601 formatting | [x] | Verified | internal/storage/state.go:145-146; internal/storage/state_test.go:393-420 |
| 1.4 Empty snapshots edge case | [x] | Verified | internal/storage/state.go:151-154; internal/storage/state_test.go:423-434 |
| 2.1 LoadHistory accessor | [x] | Verified | internal/storage/state.go:159-162; internal/storage/state_test.go:436-459 |
| 2.2 AppendSnapshot accessor | [x] | Verified | internal/storage/state.go:164-167; internal/storage/state_test.go:462-485 |
| 2.3 Unified state/history interface | [x] | Verified | internal/storage/state.go:159-171 |
| 3.1 OutputFile accessor | [x] | Verified | internal/storage/state.go:169-171; internal/storage/state_test.go:487-495 |
| 4.1 UpdateState valid inputs test | [x] | Verified | internal/storage/state_test.go:393-420 |
| 4.2 UpdateState empty snapshots test | [x] | Verified | internal/storage/state_test.go:423-434 |
| 4.3 UpdateState ISO formatting test | [x] | Verified | internal/storage/state_test.go:393-420 |
| 4.4 LoadHistory delegation test | [x] | Verified | internal/storage/state_test.go:436-459 |
| 4.5 AppendSnapshot delegation test | [x] | Verified | internal/storage/state_test.go:462-485 |
| 4.6 OutputFile accessor test | [x] | Verified | internal/storage/state_test.go:487-495 |
| 5.1 Full cycle integration test | [x] | Verified | internal/storage/state_test.go:497-555 |
| 5.2 State values persisted after cycle | [x] | Verified | internal/storage/state_test.go:520-529 |
| 5.3 Multi-cycle consistency test | [x] | Verified | internal/storage/state_test.go:557-660 |
| 6.1 go build ./... | [x] | Verified | manual run 2025-12-01 |
| 6.2 go test ./internal/storage/... | [x] | Verified | manual run 2025-12-01 |
| 6.3 make lint | [x] | Verified | manual run 2025-12-01 |

Test Coverage and Gaps
- Commands run 2025-12-01: `go build ./...`, `go test ./...`, `make lint` — all passing.
- No uncovered AC-related scenarios identified.

Architectural Alignment
- Uses atomic writes via `WriteAtomic` (state.go:82-104) per implementation patterns.
- Maintains slog logging and path scoping to configured output directory.

Security Notes
- No new I/O surfaces beyond existing outputDir files; permissions 0644 maintained.

Best-Practices and References
- Go 1.24 toolchain (go.mod).
- Patterns from `docs/architecture/implementation-patterns.md` and `testing-strategy.md` adhered to (table-driven tests, atomic writes).

Action Items

**Code Changes Required:**
- None.

**Advisory Notes:**
- Note: Architecture references pulled from `docs/architecture/index.md` and linked shards.
