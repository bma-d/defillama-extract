# Story 4.6: Implement Snapshot Deduplication

Status: done

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

- [x] Task 1: Implement AppendSnapshot function (AC: 1, 2, 3)
  - [x] 1.1: Add `AppendSnapshot(history []aggregator.Snapshot, snapshot aggregator.Snapshot, logger *slog.Logger) []aggregator.Snapshot` to `internal/storage/history.go`
  - [x] 1.2: Iterate through existing history to find matching timestamp
  - [x] 1.3: If matching timestamp found, replace snapshot in place and log debug message
  - [x] 1.4: If no matching timestamp, append new snapshot to slice
  - [x] 1.5: Sort result by timestamp ascending using `sort.Slice`
  - [x] 1.6: Add doc comment explaining deduplication behavior and sort guarantee

- [x] Task 2: Handle duplicate timestamp replacement (AC: 1, 4)
  - [x] 2.1: When duplicate found, update slice element in place: `history[i] = snapshot`
  - [x] 2.2: Log debug message: "duplicate snapshot replaced" with timestamp attribute
  - [x] 2.3: Return history (length unchanged)

- [x] Task 3: Handle new timestamp append (AC: 2)
  - [x] 3.1: When no duplicate found, append snapshot: `history = append(history, snapshot)`
  - [x] 3.2: Sort slice by timestamp ascending
  - [x] 3.3: Return extended history

- [x] Task 4: Write unit tests for AppendSnapshot (AC: 1-4)
  - [x] 4.1: Test: empty history + new snapshot returns slice with 1 element
  - [x] 4.2: Test: existing history + duplicate timestamp replaces in place (length unchanged)
  - [x] 4.3: Test: existing history + new timestamp appends (length increases by 1)
  - [x] 4.4: Test: result is always sorted by timestamp ascending
  - [x] 4.5: Test: multiple appends maintain sorted order
  - [x] 4.6: Test: verify debug log emitted for duplicate replacement

- [x] Task 5: Verification (AC: all)
  - [x] 5.1: Run `go build ./...` and verify success
  - [x] 5.2: Run `go test ./internal/storage/...` and verify all pass
  - [x] 5.3: Run `make lint` and verify no errors

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
- Implemented AppendSnapshot with in-place dedup and sorted result; added debug log on replacement; preserved slog.Default fallback.
- Added table-driven AppendSnapshot tests covering duplicate replacement, append, sort invariants, and log emission; reused newTestLogger for JSON log capture.
- Validated via `go build ./...`, `go test ./internal/storage/...`, and `make lint`.

### Completion Notes List
- AC-1..4 satisfied: duplicate timestamps replace in place with debug log; unique timestamps append; history always sorted.
- New unit tests ensure deduplication, append path, ordering across multiple appends, and logging behavior.
- All storage tests, build, and lint pass locally on 2025-12-01.

### File List
- internal/storage/history.go
- internal/storage/history_test.go
- docs/sprint-artifacts/sprint-status.yaml
- docs/sprint-artifacts/4-6-implement-snapshot-deduplication.md

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-4 and tech-spec-epic-4.md |
| 2025-12-01 | Amelia (Dev) | Implemented AppendSnapshot with deduplication + sorting; added unit tests; updated sprint status to review |
| 2025-12-01 | Amelia (Dev Reviewer) | Senior Developer Review (AI) appended; outcome Approved |

## Senior Developer Review (AI)

- Reviewer: BMad
- Date: 2025-12-01
- Outcome: Approve — All ACs and completed tasks verified with evidence; no findings

### Summary
- AppendSnapshot meets deduplication, append, and sort guarantees; logging conforms to ADR-004; tests cover AC paths.

### Key Findings
- None (no High/Med/Low issues identified).

### Acceptance Criteria Coverage

| AC # | Description | Status | Evidence |
|------|-------------|--------|----------|
| AC-1 | Duplicate timestamp replaces existing snapshot; length unchanged | Implemented | internal/storage/history.go:98-105; internal/storage/history_test.go:244-276 |
| AC-2 | Unique timestamp appends; length increases by 1 | Implemented | internal/storage/history.go:109-113; internal/storage/history_test.go:279-299 |
| AC-3 | History returned sorted ascending after append/replace | Implemented | internal/storage/history.go:102-104,110-112; internal/storage/history_test.go:279-320 |
| AC-4 | Debug log "duplicate snapshot replaced" with timestamp attribute | Implemented | internal/storage/history.go:101; internal/storage/history_test.go:259-276 |

Summary: 4 / 4 ACs implemented.

### Task Completion Validation

| Task | Marked As | Verified As | Evidence |
|------|-----------|-------------|----------|
| Task 1: AppendSnapshot function with dedup + sort | Complete | Verified | internal/storage/history.go:90-114 |
| Task 2: Duplicate replacement with debug log | Complete | Verified | internal/storage/history.go:98-105; history_test.go:244-276 |
| Task 3: Append new timestamp path | Complete | Verified | internal/storage/history.go:109-113; history_test.go:279-300 |
| Task 4: Unit tests for append/dedup/sort/logging | Complete | Verified | internal/storage/history_test.go:224-320 |
| Task 5: Build/test/lint run | Complete | Verified | go build ./...; go test ./internal/storage/...; make lint (2025-12-01) |

Summary: 5 / 5 completed tasks verified.

### Test Coverage and Gaps
- go test ./internal/storage/... (passes) covers duplicate replacement, append, ordering, log emission; no gaps relative to ACs.

### Architectural Alignment
- Uses slog debug log per ADR-004; maintains sorted invariant via sort.Slice; scope contained to internal/storage per project-structure.md.

### Security Notes
- No security-relevant changes; in-memory slice operations only.

### Best-Practices and References
- Go 1.24 module; logging via slog JSON handler; adheres to tech-spec-epic-4 AC-4.6 and testing-strategy.md table-driven tests.

### Action Items

**Code Changes Required:** None

**Advisory Notes:** None
