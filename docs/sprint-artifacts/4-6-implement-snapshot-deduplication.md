# Story 4.6: Implement Snapshot Deduplication

Status: ready-for-dev

## Story

As a **developer**,
I want **duplicate snapshots prevented when appending new ones**,
so that **history doesn't contain redundant entries with identical timestamps**.

## Acceptance Criteria

Source: [Source: docs/sprint-artifacts/tech-spec-epic-4.md#AC-4.6] / [Source: docs/epics/epic-4-state-history-management.md#story-46-implement-snapshot-deduplication]

1. **Given** existing history with snapshot at timestamp 1700000000 **When** `AppendSnapshot(history, newSnapshot)` is called with same timestamp **Then** the new snapshot replaces the existing one (update in place) **And** history length remains unchanged

2. **Given** existing history with snapshots at [1700000000, 1700003600] **When** `AppendSnapshot` is called with timestamp 1700007200 **Then** new snapshot is appended **And** history length increases by 1

3. **Given** history after any append operation **When** history is returned **Then** snapshots are sorted by timestamp ascending (oldest first)

4. **Given** `AppendSnapshot` is called with a duplicate timestamp **When** replacement occurs **Then** debug log: "duplicate snapshot replaced" with `timestamp` attribute

## Tasks / Subtasks

- [ ] Task 1: Implement AppendSnapshot function (AC: 1, 2, 3)
  - [ ] 1.1: Add `AppendSnapshot(history []aggregator.Snapshot, snapshot aggregator.Snapshot, logger *slog.Logger) []aggregator.Snapshot` to `internal/storage/history.go`
  - [ ] 1.2: Iterate through existing history to find matching timestamp
  - [ ] 1.3: If matching timestamp found, replace snapshot in place and log debug message
  - [ ] 1.4: If no matching timestamp, append new snapshot to slice
  - [ ] 1.5: Sort result by timestamp ascending using `sort.Slice`
  - [ ] 1.6: Add doc comment explaining deduplication behavior and sort guarantee

- [ ] Task 2: Handle duplicate timestamp replacement (AC: 1, 4)
  - [ ] 2.1: When duplicate found, update slice element in place: `history[i] = snapshot`
  - [ ] 2.2: Log debug message: "duplicate snapshot replaced" with timestamp attribute
  - [ ] 2.3: Return history (length unchanged)

- [ ] Task 3: Handle new timestamp append (AC: 2)
  - [ ] 3.1: When no duplicate found, append snapshot: `history = append(history, snapshot)`
  - [ ] 3.2: Sort slice by timestamp ascending
  - [ ] 3.3: Return extended history

- [ ] Task 4: Write unit tests for AppendSnapshot (AC: 1-4)
  - [ ] 4.1: Test: empty history + new snapshot returns slice with 1 element
  - [ ] 4.2: Test: existing history + duplicate timestamp replaces in place (length unchanged)
  - [ ] 4.3: Test: existing history + new timestamp appends (length increases by 1)
  - [ ] 4.4: Test: result is always sorted by timestamp ascending
  - [ ] 4.5: Test: multiple appends maintain sorted order
  - [ ] 4.6: Test: verify debug log emitted for duplicate replacement

- [ ] Task 5: Verification (AC: all)
  - [ ] 5.1: Run `go build ./...` and verify success
  - [ ] 5.2: Run `go test ./internal/storage/...` and verify all pass
  - [ ] 5.3: Run `make lint` and verify no errors

## Dev Notes

### Technical Guidance

- **Package Location:** `internal/storage/history.go` (add to existing file with `CreateSnapshot` and `LoadFromOutput`)
- **Dependencies:**
  - `log/slog` for structured logging
  - `sort` for sorting snapshots (already imported)
- **Type Reuse:** Use existing `aggregator.Snapshot` from `internal/aggregator/models.go:41-48`
- **Pattern:** Deduplication by timestamp comparison, maintain sorted invariant

### Implementation Pattern

```go
// AppendSnapshot adds a snapshot to history, replacing any existing snapshot
// with the same timestamp (deduplication). The returned slice is always sorted
// by timestamp ascending.
func AppendSnapshot(history []aggregator.Snapshot, snapshot aggregator.Snapshot, logger *slog.Logger) []aggregator.Snapshot {
    if logger == nil {
        logger = slog.Default()
    }

    // Check for duplicate timestamp
    for i, existing := range history {
        if existing.Timestamp == snapshot.Timestamp {
            logger.Debug("duplicate snapshot replaced", "timestamp", snapshot.Timestamp)
            history[i] = snapshot
            // Already sorted, replacement maintains order
            return history
        }
    }

    // Append new snapshot
    history = append(history, snapshot)

    // Sort by timestamp ascending
    sort.Slice(history, func(i, j int) bool {
        return history[i].Timestamp < history[j].Timestamp
    })

    return history
}
```

### Project Structure Notes

- **File:** `internal/storage/history.go` (add function to existing file)
- **Test File:** `internal/storage/history_test.go` (add test cases)
- **Structure Compliance:** All changes scoped to `internal/storage/` per [Source: docs/architecture/project-structure.md]

### Learnings from Previous Story

**From Story 4-5-implement-history-loading-from-output-file (Status: done)**

- **New/Modified Files:** internal/storage/history.go; internal/storage/history_test.go; internal/storage/testdata/output_with_history.json; internal/storage/testdata/output_no_history.json; internal/storage/testdata/output_corrupted.json; docs/sprint-artifacts/sprint-status.yaml.
- **Completion Notes:** AC-1..5 satisfied; go build/test/lint all passed; graceful handling for missing/corrupted files validated.
- **Review Outcome:** Approved with no blocking findings; no outstanding review items.
- **Patterns to Reuse:** Table-driven tests with fixtures; `slog.Default()` fallback for nil logger; history functions keep timestamps sorted ascending; reuse `aggregator.Snapshot`.

[Source: docs/sprint-artifacts/4-5-implement-history-loading-from-output-file.md#Dev-Agent-Record]

### Testing Standards

- Follow Go table-driven tests pattern per [Source: docs/architecture/testing-strategy.md#Test-Organization]
- Use `aggregator.Snapshot{}` literals for test fixtures
- Test both duplicate replacement and new append scenarios
- Verify sort order is maintained across operations
- Test edge case: empty history

### Smoke Test Guide

**Manual verification after implementation:**

1. Run unit tests:
   ```bash
   go test -v ./internal/storage/... -run TestAppendSnapshot
   ```

2. Verify duplicate replacement:
   ```go
   history := []aggregator.Snapshot{
       {Timestamp: 1700000000, TVS: 1000000.0},
       {Timestamp: 1700003600, TVS: 1100000.0},
   }
   newSnapshot := aggregator.Snapshot{Timestamp: 1700000000, TVS: 1200000.0}  // Same timestamp
   result := storage.AppendSnapshot(history, newSnapshot, slog.Default())
   // Should return 2 snapshots (not 3)
   // First snapshot TVS should be 1200000.0 (replaced)
   ```

3. Verify new append:
   ```go
   history := []aggregator.Snapshot{
       {Timestamp: 1700000000, TVS: 1000000.0},
   }
   newSnapshot := aggregator.Snapshot{Timestamp: 1700003600, TVS: 1100000.0}  // New timestamp
   result := storage.AppendSnapshot(history, newSnapshot, slog.Default())
   // Should return 2 snapshots
   // Sorted: [1700000000, 1700003600]
   ```

4. Verify sort order with out-of-order append:
   ```go
   history := []aggregator.Snapshot{
       {Timestamp: 1700003600, TVS: 1100000.0},
   }
   newSnapshot := aggregator.Snapshot{Timestamp: 1700000000, TVS: 1000000.0}  // Earlier timestamp
   result := storage.AppendSnapshot(history, newSnapshot, slog.Default())
   // Should return sorted: [1700000000, 1700003600]
   ```

### FR Coverage

| FR | Description | Satisfied By |
|----|-------------|--------------|
| FR32 | Deduplicate snapshots with identical timestamps | `AppendSnapshot` replaces existing snapshot with same timestamp |

### References

- [Source: docs/prd.md#FR32] - Deduplicate snapshots with identical timestamps
- [Source: docs/epics/epic-4-state-history-management.md#story-46] - Story definition and acceptance criteria
- [Source: docs/sprint-artifacts/tech-spec-epic-4.md#AC-4.6] - Authoritative acceptance criteria
- [Source: docs/sprint-artifacts/tech-spec-epic-4.md#Workflows-and-Sequencing] - History append flow
- [Source: internal/aggregator/models.go:41-48] - Snapshot struct definition
- [Source: internal/storage/history.go] - Existing CreateSnapshot and LoadFromOutput functions
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-003] - Error handling patterns
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-004] - slog logging patterns
- [Source: docs/architecture/testing-strategy.md#Test-Organization] - Testing standards
- [Source: docs/sprint-artifacts/4-5-implement-history-loading-from-output-file.md] - Previous story patterns

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/4-6-implement-snapshot-deduplication.context.xml

### Agent Model Used

- OpenAI GPT-5 (BMAD SM persona)

### Debug Log References

- Story drafted only; implementation pending, no execution logs yet.

### Completion Notes List

- Draft created from epic and tech spec sources; ready for development once validation fixes applied.

### File List

- docs/sprint-artifacts/4-6-implement-snapshot-deduplication.md

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-4 and tech-spec-epic-4.md |
