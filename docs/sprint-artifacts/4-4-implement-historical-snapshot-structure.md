# Story 4.4: Implement Historical Snapshot Structure

Status: done

## Story

As a **developer**,
I want **historical snapshots stored with required fields**,
so that **I can track TVS trends over time**.

## Acceptance Criteria

Source: [Source: docs/sprint-artifacts/tech-spec-epic-4.md#AC-4.4] / [Source: docs/epics/epic-4-state-history-management.md#Story-4.4]

1. **Given** current extraction results **When** `CreateSnapshot(result *AggregationResult)` is called **Then** a `Snapshot` struct is created and returned (uses existing `aggregator.Snapshot` type)

2. **Given** an `AggregationResult` with valid fields **When** `CreateSnapshot` is called **Then** the returned `Snapshot` contains:
   - `Timestamp`: matches `result.Timestamp` exactly
   - `Date`: formatted as `YYYY-MM-DD` (ISO 8601 date portion)
   - `TVS`: matches `result.TotalTVS`
   - `TVSByChain`: map populated from `result.ChainBreakdown` (chain -> TVS)
   - `ProtocolCount`: matches `result.TotalProtocols`
   - `ChainCount`: equals `len(result.ActiveChains)`

3. **Given** an `AggregationResult` with empty `ChainBreakdown` **When** `CreateSnapshot` is called **Then** `TVSByChain` is an empty map (not nil) to ensure valid JSON output

4. **Given** timestamp 1700000000 **When** `CreateSnapshot` formats the date **Then** `Date` field equals `"2023-11-14"` (UTC conversion)

## Tasks / Subtasks

- [x] Task 1: Implement CreateSnapshot function (AC: 1, 2)
  - [x] 1.1: Create `internal/storage/history.go` with package declaration and imports
  - [x] 1.2: Add `func CreateSnapshot(result *aggregator.AggregationResult) aggregator.Snapshot`
  - [x] 1.3: Set `Timestamp` directly from `result.Timestamp`
  - [x] 1.4: Format `Date` using `time.Unix(result.Timestamp, 0).UTC().Format("2006-01-02")`
  - [x] 1.5: Set `TVS` from `result.TotalTVS`
  - [x] 1.6: Build `TVSByChain` map by iterating `result.ChainBreakdown` and extracting chain -> TVS
  - [x] 1.7: Set `ProtocolCount` from `result.TotalProtocols`
  - [x] 1.8: Set `ChainCount` as `len(result.ActiveChains)`
  - [x] 1.9: Add doc comment explaining the function's purpose and field mapping

- [x] Task 2: Handle empty/nil inputs gracefully (AC: 3)
  - [x] 2.1: Initialize `TVSByChain` as `make(map[string]float64)` before population
  - [x] 2.2: Handle nil `result.ChainBreakdown` slice by leaving map empty
  - [x] 2.3: Handle nil `result.ActiveChains` by setting `ChainCount` to 0

- [x] Task 3: Write unit tests for CreateSnapshot (AC: 1-4)
  - [x] 3.1: Create `internal/storage/history_test.go`
  - [x] 3.2: Test: valid AggregationResult produces Snapshot with all fields populated correctly
  - [x] 3.3: Test: empty ChainBreakdown produces empty map (not nil)
  - [x] 3.4: Test: timestamp 1700000000 produces date "2023-11-14" (UTC)
  - [x] 3.5: Test: timestamp formatting for different dates (edge cases: year boundary, leap year)
  - [x] 3.6: Test: TVSByChain populated correctly from ChainBreakdown

- [x] Task 4: Verification (AC: all)
  - [x] 4.1: Run `go build ./...` and verify success
  - [x] 4.2: Run `go test ./internal/storage/...` and verify all pass
  - [x] 4.3: Run `make lint` and verify no errors

## Dev Notes

### Technical Guidance

- **Package Location:** `internal/storage/history.go` (new file)
- **Type Reuse:** Use existing `aggregator.Snapshot` struct from `internal/aggregator/models.go:41-48` - do NOT duplicate the type
- **Import:** `import "defillama-extract/internal/aggregator"`
- **Dependencies:** Go stdlib only (`time` for date formatting)
- **Pattern:** Pure function (no side effects, deterministic output for given input)

### Snapshot Struct Reference

The `aggregator.Snapshot` type already exists and should be reused:

```go
// From internal/aggregator/models.go:41-48
type Snapshot struct {
    Timestamp     int64              `json:"timestamp"`
    Date          string             `json:"date"`
    TVS           float64            `json:"tvs"`
    TVSByChain    map[string]float64 `json:"tvs_by_chain"`
    ProtocolCount int                `json:"protocol_count"`
    ChainCount    int                `json:"chain_count"`
}
```

### Implementation Pattern

```go
// CreateSnapshot creates a Snapshot from an AggregationResult for historical tracking.
// The snapshot captures point-in-time TVS metrics that power 24h/7d/30d change calculations.
func CreateSnapshot(result *aggregator.AggregationResult) aggregator.Snapshot {
    tvsByChain := make(map[string]float64)
    for _, cb := range result.ChainBreakdown {
        tvsByChain[cb.Chain] = cb.TVS
    }

    chainCount := 0
    if result.ActiveChains != nil {
        chainCount = len(result.ActiveChains)
    }

    return aggregator.Snapshot{
        Timestamp:     result.Timestamp,
        Date:          time.Unix(result.Timestamp, 0).UTC().Format("2006-01-02"),
        TVS:           result.TotalTVS,
        TVSByChain:    tvsByChain,
        ProtocolCount: result.TotalProtocols,
        ChainCount:    chainCount,
    }
}
```

### Date Formatting

Go's time format layout string `"2006-01-02"` produces ISO 8601 date format YYYY-MM-DD. Using UTC ensures consistent date representation regardless of server timezone.

**Verification:**
- `time.Unix(1700000000, 0).UTC().Format("2006-01-02")` = `"2023-11-14"`

### Project Structure Notes

- **New File:** `internal/storage/history.go` - first file for history management
- **Test File:** `internal/storage/history_test.go` (new)
- **Structure Compliance:** All changes scoped to `internal/storage/` per [Source: docs/architecture/project-structure.md]

### Learnings from Previous Story

**From Story 4-3-implement-atomic-state-file-updates (Status: done)**

- **WriteAtomic Utility:** Available at `internal/storage/writer.go` for atomic file operations (will be used in subsequent history stories)
- **Test Patterns:** Table-driven tests established in `internal/storage/state_test.go` and `writer_test.go` - follow same patterns
- **slog Logging:** Not needed for this story (pure function with no side effects)
- **Review Outcome:** Approved with no action items; fsync parent dir hardening added
- **Files Created:** `internal/storage/writer.go`, tests in `writer_test.go`

[Source: docs/sprint-artifacts/4-3-implement-atomic-state-file-updates.md#Dev-Agent-Record]

### Testing Standards

- Follow Go table-driven tests pattern per [Source: docs/architecture/testing-strategy.md#Test-Organization]
- Test both success and edge cases
- Verify date formatting with known timestamps
- Ensure empty inputs produce valid (not nil) maps

### Smoke Test Guide

**Manual verification after implementation:**

1. Create a test in history_test.go or run interactively:
   ```go
   result := &aggregator.AggregationResult{
       TotalTVS:       1000000.0,
       TotalProtocols: 42,
       ActiveChains:   []string{"solana", "ethereum", "arbitrum"},
       ChainBreakdown: []aggregator.ChainBreakdown{
           {Chain: "solana", TVS: 600000.0},
           {Chain: "ethereum", TVS: 300000.0},
           {Chain: "arbitrum", TVS: 100000.0},
       },
       Timestamp: 1700000000,
   }

   snap := storage.CreateSnapshot(result)

   // Verify fields
   fmt.Printf("Timestamp: %d (expected: 1700000000)\n", snap.Timestamp)
   fmt.Printf("Date: %s (expected: 2023-11-14)\n", snap.Date)
   fmt.Printf("TVS: %.2f (expected: 1000000.00)\n", snap.TVS)
   fmt.Printf("TVSByChain: %v\n", snap.TVSByChain)
   fmt.Printf("ProtocolCount: %d (expected: 42)\n", snap.ProtocolCount)
   fmt.Printf("ChainCount: %d (expected: 3)\n", snap.ChainCount)
   ```

2. Run: `go test -v ./internal/storage/... -run TestCreateSnapshot`

3. Verify: All fields correctly populated, date is UTC formatted, TVSByChain map is non-nil

### FR Coverage

| FR | Description | Satisfied By |
|----|-------------|--------------|
| FR30 | Maintain historical snapshots over time | `CreateSnapshot` converts aggregation results to snapshot format |
| FR31 | Store timestamp, date, TVS, TVSByChain, counts per snapshot | All fields populated in returned `Snapshot` struct |

### References

- [Source: docs/prd.md#FR30-FR31] - Historical tracking requirements
- [Source: docs/epics/epic-4-state-history-management.md#story-44] - Story definition and acceptance criteria
- [Source: docs/sprint-artifacts/tech-spec-epic-4.md#AC-4.4] - Authoritative acceptance criteria
- [Source: docs/sprint-artifacts/tech-spec-epic-4.md#data-models-and-contracts] - Snapshot structure definition
- [Source: docs/architecture/data-architecture.md#Output-Models] - Historical snapshots field in output schema
- [Source: internal/aggregator/models.go:41-48] - Existing Snapshot struct (REUSE, do not duplicate)
- [Source: internal/aggregator/models.go:60-72] - AggregationResult struct (input type)
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-003] - Error handling patterns
- [Source: docs/architecture/testing-strategy.md#Test-Organization] - Testing standards
- [Source: docs/sprint-artifacts/4-3-implement-atomic-state-file-updates.md] - Previous story patterns

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/4-4-implement-historical-snapshot-structure.context.xml

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

- Plan: map AggregationResult→Snapshot (AC1-2), enforce non-nil TVSByChain (AC3), UTC date format tests including 1700000000→2023-11-14 (AC4); follow storage package patterns.
- Execution: Added `CreateSnapshot` in `internal/storage/history.go` with UTC date formatting and safe map init; table-driven tests cover population, nil/empty inputs, and date edges; ran gofmt/go build/go test/make lint.

### Completion Notes List

- Implemented `CreateSnapshot` that reuses `aggregator.Snapshot`, initializes TVSByChain, and maps counts directly from AggregationResult.
- Added comprehensive unit tests for field mapping, nil/empty handling, and date formatting across edge timestamps.
- Verified with `go build ./...`, `go test ./internal/storage/...`, and `make lint` (all passing).

### File List

- internal/storage/history.go
- internal/storage/history_test.go
- docs/sprint-artifacts/sprint-status.yaml
- docs/sprint-artifacts/4-4-implement-historical-snapshot-structure.md

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-4 and tech-spec-epic-4.md |
| 2025-11-30 | Amelia | Implemented CreateSnapshot with tests; updated sprint status and story metadata |
| 2025-11-30 | Amelia | Senior Developer Review (AI) approved; no blocking findings |
| 2025-11-30 | Amelia | Added nil guard to CreateSnapshot for defensive safety |

## Senior Developer Review (AI)

**Reviewer:** BMad  
**Date:** 2025-11-30  
**Outcome:** Approve — all ACs and completed tasks verified with evidence.

### Summary
- Implementation matches tech spec for Snapshot creation; map initialization and UTC date formatting verified.
- Tests cover happy path, nil/empty inputs, and date edge cases; build/test/lint all pass locally.

### Key Findings
- High: None
- Medium: None
- Low: None
- Notes: Defensive guard added for nil `AggregationResult` to return zero-value snapshot with empty map.

### Acceptance Criteria Coverage
| AC# | Description | Status | Evidence |
| --- | ----------- | ------ | -------- |
| AC1 | CreateSnapshot returns Snapshot using existing type | Implemented | `internal/storage/history.go:9-27` |
| AC2 | Snapshot fields map exactly from AggregationResult | Implemented | `internal/storage/history.go:20-26`, `internal/storage/history_test.go:10-53` |
| AC3 | Empty ChainBreakdown yields empty (non-nil) map | Implemented | `internal/storage/history.go:13-25`, `internal/storage/history_test.go:56-78` |
| AC4 | Timestamp 1700000000 formats to 2023-11-14 UTC | Implemented | `internal/storage/history.go:22`, `internal/storage/history_test.go:98-119` |

### Task Completion Validation
| Task | Marked As | Verified As | Evidence |
| ---- | --------- | ----------- | -------- |
| Task 1: Implement CreateSnapshot (subs 1.1-1.9) | Complete | Verified complete | `internal/storage/history.go:9-27` |
| Task 2: Handle empty/nil inputs | Complete | Verified complete | `internal/storage/history.go:13-26`, `history_test.go:56-96` |
| Task 3: Unit tests for CreateSnapshot | Complete | Verified complete | `internal/storage/history_test.go:10-119` |
| Task 4: Verification commands (build/test/lint) | Complete | Verified complete | `go build ./...`, `go test ./internal/storage/...`, `make lint` |

### Test Coverage and Gaps
- Executed: `go build ./...`; `go test ./internal/storage/...`; `make lint` (all pass).
- Tests exercise mapping, nil/empty slices, date edge cases including AC timestamp.

### Architectural Alignment
- Reuses `aggregator.Snapshot` per tech spec; pure function, stdlib only; no logging side-effects. Aligns with data-architecture and ADR-003.

### Security Notes
- None identified (pure data mapping, no I/O).

### Best-Practices and References
- Date formatting via `time.Unix(...).UTC().Format("2006-01-02")` aligns with Epic-4 tech spec and `docs/architecture/data-architecture.md`.

### Action Items
**Code Changes Required:** None  
**Advisory Notes:**  
- Note: If upstream might pass nil `AggregationResult`, add a nil guard in `CreateSnapshot` to avoid panic.
