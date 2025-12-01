# Story 4.8: Build State Manager Component

Status: ready-for-dev

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

- [ ] Task 1: Add `UpdateState` method to StateManager (AC: 3)
  - [ ] 1.1: Implement `UpdateState(oracleName string, ts int64, count int, tvs float64, snapshots []Snapshot) *State` in `internal/storage/state.go`
  - [ ] 1.2: Populate all State fields including `SnapshotCount`, `OldestSnapshot`, `NewestSnapshot` from snapshots slice
  - [ ] 1.3: Format `LastUpdatedISO` as ISO 8601 using `time.Unix(ts, 0).UTC().Format(time.RFC3339)`
  - [ ] 1.4: Handle edge case: empty snapshots slice (set snapshot metadata to 0)

- [ ] Task 2: Add history accessor methods to StateManager (AC: 4)
  - [ ] 2.1: Add `LoadHistory() ([]aggregator.Snapshot, error)` method that delegates to `LoadFromOutput(sm.outputFile, sm.logger)`
  - [ ] 2.2: Add `AppendSnapshot(history []aggregator.Snapshot, snapshot aggregator.Snapshot) []aggregator.Snapshot` method that delegates to package-level `AppendSnapshot`
  - [ ] 2.3: Ensure StateManager provides unified interface for both state and history operations

- [ ] Task 3: Add `OutputFile()` accessor method (AC: 1, 4)
  - [ ] 3.1: Add `OutputFile() string` method to return the output file path for use by downstream consumers (Epic 5 writer)

- [ ] Task 4: Write unit tests for new methods (AC: 1-5)
  - [ ] 4.1: Test `UpdateState` with valid inputs - verify all fields populated correctly
  - [ ] 4.2: Test `UpdateState` with empty snapshots slice - verify snapshot metadata is 0
  - [ ] 4.3: Test `UpdateState` ISO 8601 timestamp formatting
  - [ ] 4.4: Test `LoadHistory` delegation (uses existing LoadFromOutput tests)
  - [ ] 4.5: Test `AppendSnapshot` delegation (uses existing tests)
  - [ ] 4.6: Test `OutputFile` returns correct path

- [ ] Task 5: Write integration test for full extraction cycle (AC: 5)
  - [ ] 5.1: Test complete cycle: LoadState -> ShouldProcess -> (simulate extraction) -> UpdateState -> SaveState
  - [ ] 5.2: Verify state file contains correct values after cycle
  - [ ] 5.3: Verify state and history consistency after multiple cycles

- [ ] Task 6: Verification (AC: all)
  - [ ] 6.1: Run `go build ./...` and verify success
  - [ ] 6.2: Run `go test ./internal/storage/...` and verify all pass
  - [ ] 6.3: Run `make lint` and verify no errors

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

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-12-01 | SM Agent (Bob) | Initial story draft created from epic-4 and tech-spec-epic-4.md |
