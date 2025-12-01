# Story 4.7: Implement History Retention (Keep All)

Status: ready-for-dev

## Story

As a **developer**,
I want **all historical snapshots retained without automatic pruning**,
so that **complete history is available for analysis and no data is lost over time**.

## Acceptance Criteria

Source: [Source: docs/sprint-artifacts/tech-spec-epic-4.md#AC-4.7] / [Source: docs/epics/epic-4-state-history-management.md#story-47-implement-history-retention-keep-all]

1. **Given** history with 1000 snapshots spanning 90+ days **When** a new snapshot is added **Then** all existing snapshots are retained **And** new snapshot is appended **And** no automatic pruning occurs

2. **Given** MVP requirements **When** history management code is reviewed **Then** there is NO automatic pruning logic (FR33 - retain all) **And** a doc comment notes "pruning may be added in future version"

3. **Given** existing `AppendSnapshot` function **When** code is reviewed **Then** it appends without removing any existing snapshots

## Tasks / Subtasks

- [ ] Task 1: Verify no pruning logic exists (AC: 1, 2, 3)
  - [ ] 1.1: Review `internal/storage/history.go` - confirm `AppendSnapshot` only appends/replaces, never removes snapshots
  - [ ] 1.2: Review `internal/storage/history.go` - confirm `LoadFromOutput` does not filter or limit snapshots
  - [ ] 1.3: Search codebase for any "prune", "trim", "limit", "maxHistory", "retention" logic - verify none exists
  - [ ] 1.4: Confirm no snapshot removal code paths in any history-related functions

- [ ] Task 2: Add documentation comment for future pruning (AC: 2)
  - [ ] 2.1: Add package-level doc comment to `internal/storage/history.go` noting: "MVP retains all historical snapshots without automatic pruning. Pruning may be added in a future version."
  - [ ] 2.2: Add comment to `AppendSnapshot` function noting retention policy

- [ ] Task 3: Write unit test verifying retention behavior (AC: 1, 3)
  - [ ] 3.1: Add test: `TestAppendSnapshot_RetainsAllHistory` - append to history with many snapshots, verify all retained
  - [ ] 3.2: Add test: verify AppendSnapshot with 100+ snapshots maintains all entries
  - [ ] 3.3: Verify test confirms no snapshot count reduction after any append operation

- [ ] Task 4: Verification (AC: all)
  - [ ] 4.1: Run `go build ./...` and verify success
  - [ ] 4.2: Run `go test ./internal/storage/...` and verify all pass
  - [ ] 4.3: Run `make lint` and verify no errors

## Dev Notes

### Technical Guidance

- **Package Location:** `internal/storage/history.go`
- **Story Nature:** This story is primarily about verification and documentation rather than new implementation
- **Key Insight:** The existing `AppendSnapshot` function (implemented in Story 4.6) already satisfies the "keep all" requirement - it only appends or replaces (deduplication), never removes snapshots
- **FR Coverage:** FR33 - "System retains all historical snapshots (no automatic pruning)"

### Verification Checklist

1. **No Pruning Code Exists:**
   - `AppendSnapshot` - replaces duplicates or appends, never removes
   - `LoadFromOutput` - loads all historical snapshots from file
   - No `maxHistory`, `retentionDays`, `pruneOlder`, or similar parameters

2. **Documentation Required:**
   - Add comment explaining MVP decision to retain all history
   - Note that pruning may be added in future (configurable retention window)

### Implementation Pattern

The work for this story is minimal because the retention-all policy is already implicit in the existing code. The tasks are:

1. **Verify** the existing code meets the requirement (no removals)
2. **Document** the retention policy explicitly
3. **Test** to ensure the behavior is locked in

```go
// Package storage handles state persistence and historical snapshot management.
//
// History Retention Policy (MVP):
// All historical snapshots are retained without automatic pruning. This ensures
// complete data is available for analysis and historical trend calculations
// (24h/7d/30d changes). Configurable retention/pruning may be added in a
// future version.
package storage
```

### Project Structure Notes

- **File:** `internal/storage/history.go` (add documentation)
- **Test File:** `internal/storage/history_test.go` (add retention tests)
- **Structure Compliance:** All changes scoped to `internal/storage/` per [Source: docs/architecture/project-structure.md]

### Learnings from Previous Story

**From Story 4-6-implement-snapshot-deduplication (Status: done)**

- **Files Modified:** `internal/storage/history.go`, `internal/storage/history_test.go`
- **Key Implementation:** `AppendSnapshot()` function with deduplication and sorting
- **Pattern Established:** Function replaces duplicate timestamps in place, appends new timestamps, maintains sorted order - **does not remove any snapshots**
- **Patterns to Reuse:**
  - Table-driven tests with fixtures
  - `slog.Default()` fallback for nil logger
  - History functions maintain timestamp sort order
  - Use `aggregator.Snapshot` from `internal/aggregator/models.go`
- **Review Outcome:** Approved with no blocking findings

[Source: docs/sprint-artifacts/4-6-implement-snapshot-deduplication.md#Dev-Agent-Record]

### Testing Standards

- Follow Go table-driven tests pattern per [Source: docs/architecture/testing-strategy.md#Test-Organization]
- Use `aggregator.Snapshot{}` literals for test fixtures
- Test with large snapshot counts (100+) to verify retention
- Verify no snapshot count reduction after append operations

### Smoke Test Guide

**Manual verification after implementation:**

1. Run unit tests:
   ```bash
   go test -v ./internal/storage/... -run TestAppendSnapshot
   ```

2. Verify retention with many snapshots:
   ```go
   // Build history with 100 snapshots
   history := make([]aggregator.Snapshot, 100)
   for i := 0; i < 100; i++ {
       history[i] = aggregator.Snapshot{
           Timestamp: int64(1700000000 + i*3600),
           TVS:       float64(1000000 + i*1000),
       }
   }

   // Append new snapshot
   newSnapshot := aggregator.Snapshot{Timestamp: 1700360000, TVS: 2000000.0}
   result := storage.AppendSnapshot(history, newSnapshot, slog.Default())

   // Verify: should have 101 snapshots (all retained + 1 new)
   // No pruning should occur
   ```

3. Verify doc comments added:
   ```bash
   go doc github.com/switchboard-xyz/defillama-extract/internal/storage
   # Should mention retention policy
   ```

### FR Coverage

| FR | Description | Satisfied By |
|----|-------------|--------------|
| FR33 | Retain all historical snapshots (no automatic pruning) | Verification that AppendSnapshot only appends/replaces, documentation of policy |

### References

- [Source: docs/prd.md#FR33] - Retain all historical snapshots
- [Source: docs/epics/epic-4-state-history-management.md#story-47] - Story definition and acceptance criteria
- [Source: docs/sprint-artifacts/tech-spec-epic-4.md#AC-4.7] - Authoritative acceptance criteria
- [Source: docs/sprint-artifacts/tech-spec-epic-4.md#Risks-Assumptions] - "Output file grows unbounded" accepted for MVP
- [Source: internal/storage/history.go] - Existing AppendSnapshot function (Story 4.6)
- [Source: docs/sprint-artifacts/4-6-implement-snapshot-deduplication.md] - Previous story implementation
- [Source: docs/architecture/testing-strategy.md#Test-Organization] - Testing standards

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/4-7-implement-history-retention-keep-all.context.xml

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-4 and tech-spec-epic-4.md |
