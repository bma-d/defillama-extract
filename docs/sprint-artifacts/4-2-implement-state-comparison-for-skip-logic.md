# Story 4.2: Implement State Comparison for Skip Logic

Status: ready-for-dev

## Story

As a **developer**,
I want **to compare current API timestamp against last processed timestamp**,
so that **I can skip processing when no new data is available**.

## Acceptance Criteria

Source: Epic 4.2 / Tech Spec AC-4.2 / [Source: docs/prd.md#fr26-fr27]

1. **Given** `state.LastUpdated == 0` (first run) **When** `ShouldProcess(currentTS, state)` is called **Then** returns `true` **And** debug log: "first run, processing required"

2. **Given** `currentTS > state.LastUpdated` (new data available) **When** `ShouldProcess(currentTS, state)` is called **Then** returns `true` **And** debug log: "new data available" with attributes `current_ts`, `last_ts`, `delta_seconds`

3. **Given** `currentTS == state.LastUpdated` (no new data) **When** `ShouldProcess(currentTS, state)` is called **Then** returns `false` **And** info log: "no new data, skipping extraction"

4. **Given** `currentTS < state.LastUpdated` (clock skew) **When** `ShouldProcess(currentTS, state)` is called **Then** returns `false` **And** warn log: "clock skew detected, API timestamp older than last processed" with `current_ts`, `last_ts`

5. **Given** any call to `ShouldProcess` **When** decision is made **Then** appropriate log level is used (debug for process, info for skip, warn for anomaly) **And** all relevant attributes logged per ADR-004

## Tasks / Subtasks

- [ ] Task 1: Implement ShouldProcess method (AC: 1-5)
  - [ ] 1.1: Add `func (sm *StateManager) ShouldProcess(currentTS int64, state *State) bool` method to `internal/storage/state.go`
  - [ ] 1.2: Handle first-run case: `state.LastUpdated == 0` → return `true`, log debug "first run"
  - [ ] 1.3: Handle new-data case: `currentTS > state.LastUpdated` → return `true`, log debug with delta
  - [ ] 1.4: Handle no-new-data case: `currentTS == state.LastUpdated` → return `false`, log info
  - [ ] 1.5: Handle clock-skew case: `currentTS < state.LastUpdated` → return `false`, log warn
  - [ ] 1.6: Add doc comment explaining function behavior and return semantics

- [ ] Task 2: Write unit tests (AC: 1-5)
  - [ ] 2.1: Add tests to `internal/storage/state_test.go`
  - [ ] 2.2: Test: first run (LastUpdated=0) returns true
  - [ ] 2.3: Test: new data (currentTS > LastUpdated) returns true
  - [ ] 2.4: Test: no new data (currentTS == LastUpdated) returns false
  - [ ] 2.5: Test: clock skew (currentTS < LastUpdated) returns false
  - [ ] 2.6: Test: verify log output for each scenario using log capture

- [ ] Task 3: Verification (AC: all)
  - [ ] 3.1: Run `go build ./...` and verify success
  - [ ] 3.2: Run `go test ./internal/storage/...` and verify all pass
  - [ ] 3.3: Run `make lint` and verify no errors

## Dev Notes

### Technical Guidance

- **Package Location:** `internal/storage/state.go` (add to existing file)
- **Pattern:** Method on StateManager struct for consistency
- **Dependencies:** Go stdlib only (`log/slog` for logging)
- **Error Handling:** No errors returned; boolean decision with logging
- **Logging:** ADR-004 - structured logging with slog, level varies by scenario

### ShouldProcess Method Pattern

```go
// ShouldProcess determines if extraction should proceed based on timestamp comparison.
// Returns true if processing is needed (first run or new data), false otherwise.
// All decisions are logged with appropriate level and attributes.
func (sm *StateManager) ShouldProcess(currentTS int64, state *State) bool {
    // First run: always process
    if state.LastUpdated == 0 {
        sm.logger.Debug("first run, processing required")
        return true
    }

    // New data available
    if currentTS > state.LastUpdated {
        delta := currentTS - state.LastUpdated
        sm.logger.Debug("new data available, proceeding with extraction",
            "current_ts", currentTS,
            "last_ts", state.LastUpdated,
            "delta_seconds", delta,
        )
        return true
    }

    // No new data (timestamps match)
    if currentTS == state.LastUpdated {
        sm.logger.Info("no new data, skipping extraction",
            "current_ts", currentTS,
            "last_ts", state.LastUpdated,
        )
        return false
    }

    // Clock skew: current timestamp older than last processed
    sm.logger.Warn("clock skew detected, API timestamp older than last processed",
        "current_ts", currentTS,
        "last_ts", state.LastUpdated,
    )
    return false
}
```

### Test Table Pattern

```go
func TestStateManager_ShouldProcess(t *testing.T) {
    tests := []struct {
        name        string
        currentTS   int64
        lastUpdated int64
        wantResult  bool
        wantLogMsg  string
        wantLevel   slog.Level
    }{
        {
            name:        "first run returns true",
            currentTS:   1700000000,
            lastUpdated: 0,
            wantResult:  true,
            wantLogMsg:  "first run",
            wantLevel:   slog.LevelDebug,
        },
        {
            name:        "new data returns true",
            currentTS:   1700003600,
            lastUpdated: 1700000000,
            wantResult:  true,
            wantLogMsg:  "new data available",
            wantLevel:   slog.LevelDebug,
        },
        {
            name:        "same timestamp returns false",
            currentTS:   1700000000,
            lastUpdated: 1700000000,
            wantResult:  false,
            wantLogMsg:  "no new data",
            wantLevel:   slog.LevelInfo,
        },
        {
            name:        "clock skew returns false",
            currentTS:   1700000000,
            lastUpdated: 1700003600,
            wantResult:  false,
            wantLogMsg:  "clock skew",
            wantLevel:   slog.LevelWarn,
        },
    }
    // ... test implementation
}
```

### Project Structure Notes

- **Existing File:** `internal/storage/state.go` contains `StateManager` and `LoadState`
- **Add Method:** `ShouldProcess` extends StateManager functionality
- **Test File:** `internal/storage/state_test.go` exists with LoadState tests
- **No New Files:** This story only extends existing implementation
- **Structure Compliance:** Keep changes scoped to `internal/storage/` per project structure [Source: docs/architecture/project-structure.md]

### Learnings from Previous Story

**From Story 4-1-implement-state-file-structure-and-loading (Status: done)**

- **StateManager Pattern:** Constructor (`NewStateManager`) with encapsulated paths and logger already established - `ShouldProcess` follows same method pattern [Source: docs/sprint-artifacts/4-1-implement-state-file-structure-and-loading.md#Technical-Guidance]
- **Nil Logger Handling:** Constructor defaults nil logger to `slog.Default()` - reuse this safety pattern [Source: docs/sprint-artifacts/4-1-implement-state-file-structure-and-loading.md#Technical-Guidance]
- **Test Patterns:** Table-driven tests with fixtures used; add `ShouldProcess` tests to same file [Source: docs/sprint-artifacts/4-1-implement-state-file-structure-and-loading.md#Testing-Standards]
- **Structured Logging:** Debug/Warn/Info levels with key-value pairs established in `LoadState` - maintain consistency [Source: docs/sprint-artifacts/4-1-implement-state-file-structure-and-loading.md#Testing-Standards]
- **Files NEW/MODIFIED to reference:** `internal/storage/state.go`, `internal/storage/state_test.go`, `internal/storage/testdata/*.json` → reuse logging style, table-driven layout, and fixtures approach [Source: docs/sprint-artifacts/4-1-implement-state-file-structure-and-loading.md#File-List]

[Source: docs/sprint-artifacts/4-1-implement-state-file-structure-and-loading.md#Dev-Agent-Record]

### Architecture References

- **Error handling (ADR-003):** Explicit returns, but ShouldProcess uses bool+logging (no error return needed for simple decision)
- **Structured logging (ADR-004):** Use slog with appropriate levels per scenario
- **Skip logic flow:** Tech spec defines exact flow [Source: docs/sprint-artifacts/tech-spec-epic-4.md#workflows-and-sequencing]
- **FR26/FR27:** Compare timestamps, skip when no new data [Source: docs/prd.md#fr26-fr27]
- **Testing standards:** Follow project testing guidance; cite testing strategy when describing tests [Source: docs/architecture/testing-strategy.md#Test-Organization]
- **Project structure:** Keep implementations under `internal/storage/` per structure doc [Source: docs/architecture/project-structure.md]

### Testing Standards

- Follow Go table-driven tests pattern (per 4.1) [Source: docs/architecture/testing-strategy.md#Test-Organization]
- Use log capture to verify log output if needed
- Test all four timestamp scenarios
- Verify correct log level for each scenario

### Smoke Test Guide

**Manual verification after implementation:**

1. Create a test program or add to existing test:
   ```go
   sm := storage.NewStateManager("/tmp/test", nil)
   state := &storage.State{LastUpdated: 1700000000}

   // Test new data scenario
   result := sm.ShouldProcess(1700003600, state)
   fmt.Printf("New data: %v (expected: true)\n", result)

   // Test no new data scenario
   result = sm.ShouldProcess(1700000000, state)
   fmt.Printf("Same timestamp: %v (expected: false)\n", result)

   // Test first run
   result = sm.ShouldProcess(1700000000, &storage.State{})
   fmt.Printf("First run: %v (expected: true)\n", result)
   ```

2. Run: `go test -v ./internal/storage/... -run TestStateManager_ShouldProcess`

3. Verify: All scenarios return correct boolean and appropriate log messages appear

### FR Coverage

| FR | Description | Satisfied By |
|----|-------------|--------------|
| FR26 | Compare latest API timestamp against last processed | `ShouldProcess` timestamp comparison |
| FR27 | Skip processing when no new data | Returns `false` when `currentTS == state.LastUpdated` |

### References

- [Source: docs/prd.md#fr26-fr27] - Functional requirements for skip logic
- [Source: docs/epics/epic-4-state-history-management.md#story-42] - Story definition and acceptance criteria
- [Source: docs/sprint-artifacts/tech-spec-epic-4.md#ac-42] - Authoritative acceptance criteria
- [Source: docs/sprint-artifacts/tech-spec-epic-4.md#workflows-and-sequencing] - Skip decision flow diagram
- [Source: docs/architecture/architecture-decision-records-adrs.md#adr-004] - Structured logging with slog
- [Source: docs/sprint-artifacts/4-1-implement-state-file-structure-and-loading.md] - Previous story patterns and context

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/4-2-implement-state-comparison-for-skip-logic.context.xml

### Agent Model Used

SM Auto-Improve v1 (Bob)

### Debug Log References

To be added during implementation

### Completion Notes List

- Pending development updates

### File List

- None yet (implementation not started)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-4-state-history-management.md and tech-spec-epic-4.md |
