# Story 4.5: Implement History Loading from Output File

Status: done

## Story

As a **developer**,
I want **existing history loaded from the output file on startup**,
so that **historical data is preserved across runs**.

## Acceptance Criteria

Source: [Source: docs/sprint-artifacts/tech-spec-epic-4.md#AC-4.5] / [Source: docs/epics/epic-4-state-history-management.md#Story-4.5]

1. **Given** output file `switchboard-oracle-data.json` exists with `historical` array **When** `LoadFromOutput(outputPath string)` is called **Then** the `historical` array is extracted and returned as `[]aggregator.Snapshot`

2. **Given** a valid output file with historical data **When** `LoadFromOutput` is called **Then** snapshots are returned sorted by timestamp ascending (oldest first)

3. **Given** output file doesn't exist **When** `LoadFromOutput` is called **Then** empty slice is returned (not an error) **And** debug log: "no existing history found, starting fresh"

4. **Given** output file exists but `historical` field is empty or missing **When** `LoadFromOutput` is called **Then** empty slice is returned (not an error)

5. **Given** output file is corrupted (invalid JSON) **When** `LoadFromOutput` is called **Then** warn log: "failed to load history, starting fresh" **And** empty slice is returned (graceful degradation, not error)

## Tasks / Subtasks

- [x] Task 1: Implement LoadFromOutput function (AC: 1, 2)
  - [x] 1.1: Add `LoadFromOutput(outputPath string, logger *slog.Logger) ([]aggregator.Snapshot, error)` to `internal/storage/history.go`
  - [x] 1.2: Read file using `os.ReadFile(outputPath)`
  - [x] 1.3: Define minimal struct to extract only `historical` field: `type outputHistoryExtract struct { Historical []aggregator.Snapshot json:"historical" }`
  - [x] 1.4: Unmarshal JSON into extract struct
  - [x] 1.5: Sort snapshots by timestamp ascending using `sort.Slice`
  - [x] 1.6: Add doc comment explaining the function's purpose and graceful handling

- [x] Task 2: Handle missing file gracefully (AC: 3)
  - [x] 2.1: Check for `os.IsNotExist(err)` after `os.ReadFile`
  - [x] 2.2: Log debug message: "no existing history found, starting fresh" with path attribute
  - [x] 2.3: Return empty `[]aggregator.Snapshot{}` and nil error

- [x] Task 3: Handle empty/missing historical field (AC: 4)
  - [x] 3.1: After successful unmarshal, check if `Historical` slice is nil or empty
  - [x] 3.2: Return empty slice (already initialized by Go) - no special handling needed

- [x] Task 4: Handle corrupted file gracefully (AC: 5)
  - [x] 4.1: If `json.Unmarshal` returns error, log warn: "failed to load history, starting fresh" with path and error attributes
  - [x] 4.2: Return empty `[]aggregator.Snapshot{}` and nil error (graceful degradation)

- [x] Task 5: Write unit tests for LoadFromOutput (AC: 1-5)
  - [x] 5.1: Create test fixtures in `internal/storage/testdata/` directory
  - [x] 5.2: Create `output_with_history.json` fixture with valid historical data (multiple snapshots, unsorted)
  - [x] 5.3: Create `output_no_history.json` fixture with valid JSON but no/empty historical field
  - [x] 5.4: Create `output_corrupted.json` fixture with invalid JSON
  - [x] 5.5: Test: valid output file returns sorted snapshots
  - [x] 5.6: Test: missing file returns empty slice, no error
  - [x] 5.7: Test: empty historical field returns empty slice
  - [x] 5.8: Test: corrupted file returns empty slice, no error, warn logged
  - [x] 5.9: Test: verify sort order is timestamp ascending

- [x] Task 6: Verification (AC: all)
  - [x] 6.1: Run `go build ./...` and verify success
  - [x] 6.2: Run `go test ./internal/storage/...` and verify all pass
  - [x] 6.3: Run `make lint` and verify no errors

## Dev Notes

### Technical Guidance

- **Package Location:** `internal/storage/history.go` (add to existing file)
- **Dependencies:**
  - `encoding/json` for unmarshaling
  - `log/slog` for structured logging
  - `os` for file operations
  - `sort` for sorting snapshots
- **Type Reuse:** Use existing `aggregator.Snapshot` from `internal/aggregator/models.go:41-48`
- **Pattern:** Graceful degradation - never return error for missing/corrupted files, just empty slice with appropriate logging

### Partial JSON Parsing Strategy

Only extract the `historical` field to minimize parsing overhead:

```go
// outputHistoryExtract is used for partial parsing of output file
// to extract only the historical snapshots field.
type outputHistoryExtract struct {
    Historical []aggregator.Snapshot `json:"historical"`
}
```

### Implementation Pattern

```go
// LoadFromOutput extracts historical snapshots from an existing output file.
// Returns empty slice (not error) if file is missing, empty, or corrupted.
// This enables graceful degradation - extraction continues even without history.
func LoadFromOutput(outputPath string, logger *slog.Logger) ([]aggregator.Snapshot, error) {
    data, err := os.ReadFile(outputPath)
    if err != nil {
        if os.IsNotExist(err) {
            logger.Debug("no existing history found, starting fresh", "path", outputPath)
            return []aggregator.Snapshot{}, nil
        }
        // Other read errors - treat as corrupted
        logger.Warn("failed to read history file", "path", outputPath, "error", err)
        return []aggregator.Snapshot{}, nil
    }

    var extract outputHistoryExtract
    if err := json.Unmarshal(data, &extract); err != nil {
        logger.Warn("failed to load history, starting fresh", "path", outputPath, "error", err)
        return []aggregator.Snapshot{}, nil
    }

    // Handle nil/empty historical field
    if len(extract.Historical) == 0 {
        return []aggregator.Snapshot{}, nil
    }

    // Sort by timestamp ascending (oldest first)
    sort.Slice(extract.Historical, func(i, j int) bool {
        return extract.Historical[i].Timestamp < extract.Historical[j].Timestamp
    })

    logger.Debug("history loaded",
        "snapshot_count", len(extract.Historical),
        "oldest", extract.Historical[0].Timestamp,
        "newest", extract.Historical[len(extract.Historical)-1].Timestamp,
    )

    return extract.Historical, nil
}
```

### Project Structure Notes

- **File:** `internal/storage/history.go` (add function to existing file)
- **Test File:** `internal/storage/history_test.go` (add test cases)
- **Test Fixtures:** `internal/storage/testdata/` (create directory and fixture files)
- **Structure Compliance:** All changes scoped to `internal/storage/` per [Source: docs/architecture/project-structure.md]

### Learnings from Previous Story

**From Story 4-4-implement-historical-snapshot-structure (Status: done)**

- **CreateSnapshot Available:** `CreateSnapshot(result *aggregator.AggregationResult) aggregator.Snapshot` in `internal/storage/history.go:12-34`
- **Test Patterns:** Table-driven tests established - follow same patterns
- **Type Reuse:** Confirmed `aggregator.Snapshot` type should be reused
- **Nil Guards:** Defensive programming encouraged - handle nil inputs gracefully
- **Review Outcome:** Approved with no blocking findings

[Source: docs/sprint-artifacts/4-4-implement-historical-snapshot-structure.md#Dev-Agent-Record]

### Testing Standards

- Follow Go table-driven tests pattern per [Source: docs/architecture/testing-strategy.md#Test-Organization]
- Use `t.TempDir()` for ephemeral test files where appropriate
- Use testdata fixtures for predefined test scenarios
- Test both success and error paths
- Verify logging occurs with expected attributes

### Smoke Test Guide

**Manual verification after implementation:**

1. Create a test output file manually or use fixture:
   ```json
   {
     "version": "1.0.0",
     "historical": [
       {"timestamp": 1700003600, "date": "2023-11-14", "tvs": 1000000.0, "tvs_by_chain": {"solana": 1000000.0}, "protocol_count": 10, "chain_count": 1},
       {"timestamp": 1700000000, "date": "2023-11-14", "tvs": 900000.0, "tvs_by_chain": {"solana": 900000.0}, "protocol_count": 9, "chain_count": 1}
     ]
   }
   ```

2. Run: `go test -v ./internal/storage/... -run TestLoadFromOutput`

3. Verify:
   - Returns 2 snapshots
   - Sorted by timestamp (1700000000 first, 1700003600 second)
   - Debug log shows snapshot_count=2

4. Test missing file:
   ```go
   snapshots, err := storage.LoadFromOutput("/nonexistent/path.json", slog.Default())
   // Should return empty slice, nil error
   ```

5. Test corrupted file:
   ```bash
   echo "invalid json {" > /tmp/corrupted.json
   # LoadFromOutput should return empty slice with warn log
   ```

### FR Coverage

| FR | Description | Satisfied By |
|----|-------------|--------------|
| FR34 | Load existing history from output file on startup | `LoadFromOutput` extracts and returns historical snapshots |
| NFR10 | Graceful degradation | Missing/corrupted files return empty slice, not error |

### References

- [Source: docs/prd.md#FR34] - Load existing history from output file on startup
- [Source: docs/epics/epic-4-state-history-management.md#story-45] - Story definition and acceptance criteria
- [Source: docs/sprint-artifacts/tech-spec-epic-4.md#AC-4.5] - Authoritative acceptance criteria
- [Source: docs/sprint-artifacts/tech-spec-epic-4.md#Workflows-and-Sequencing] - History loading flow
- [Source: docs/architecture/data-architecture.md#Output-Models] - FullOutput.Historical field definition
- [Source: internal/aggregator/models.go:41-48] - Snapshot struct definition
- [Source: internal/storage/history.go] - Existing CreateSnapshot function
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-003] - Error handling patterns
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-004] - slog logging patterns
- [Source: docs/architecture/testing-strategy.md#Test-Organization] - Testing standards
- [Source: docs/sprint-artifacts/4-4-implement-historical-snapshot-structure.md] - Previous story patterns

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/4-5-implement-history-loading-from-output-file.context.xml

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References
- Implemented `LoadFromOutput` with partial parse + graceful degradation (AC-1..5)
- Sorted snapshots ascending with `sort.Slice` and debug telemetry (AC-2)

### Completion Notes List
- AC-1..5 satisfied: history loads from output, handles missing/empty/corrupted with logs, returns sorted snapshots.
- Tests/verification: `go build ./...`, `go test ./...`, `make lint`.

### File List
- internal/storage/history.go
- internal/storage/history_test.go
- internal/storage/testdata/output_with_history.json
- internal/storage/testdata/output_no_history.json
- internal/storage/testdata/output_corrupted.json
- docs/sprint-artifacts/sprint-status.yaml

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-4 and tech-spec-epic-4.md |
| 2025-11-30 | Amelia | Implemented `LoadFromOutput`, added fixtures/tests, marked story ready for review |
| 2025-11-30 | Amelia | Senior Developer Review (AI) appended; status updated to done |

## Senior Developer Review (AI)

Reviewer: BMad  
Date: 2025-11-30  
Outcome: Approve – all ACs satisfied; no code changes required.

### Summary
- LoadFromOutput meets AC-1..AC-5 with graceful handling for missing/empty/corrupted output files and sorted history return.
- Unit tests cover all acceptance paths; go build/test and make lint executed successfully.
- Repository contains additional pending changes (state.go, writer.go) from other stories; ensure merge hygiene when integrating.

### Key Findings
- No blocking or medium findings for this story.
- Low: Additional modified/untracked files outside story scope present; verify they belong to other stories before merge.

### Acceptance Criteria Coverage
| AC# | Status | Evidence |
|-----|--------|----------|
| AC-1 | Implemented | internal/storage/history.go:55-87; internal/storage/history_test.go:137-167 |
| AC-2 | Implemented | internal/storage/history.go:76-79; internal/storage/history_test.go:151-153 |
| AC-3 | Implemented | internal/storage/history.go:55-63; internal/storage/history_test.go:169-185 |
| AC-4 | Implemented | internal/storage/history.go:72-74; internal/storage/history_test.go:188-199 |
| AC-5 | Implemented | internal/storage/history.go:66-70; internal/storage/history_test.go:201-221; internal/storage/testdata/output_corrupted.json |
**Coverage:** 5/5 acceptance criteria fully implemented.

### Task Completion Validation
| Task | Marked As | Verified As | Evidence |
|------|-----------|-------------|----------|
| 1. Implement LoadFromOutput | Done | Verified | internal/storage/history.go:46-87 |
| 2. Handle missing file | Done | Verified | internal/storage/history.go:55-63; history_test.go:169-185 |
| 3. Handle empty/missing historical | Done | Verified | internal/storage/history.go:72-74; history_test.go:188-199 |
| 4. Handle corrupted file | Done | Verified | internal/storage/history.go:66-70; history_test.go:201-221 |
| 5. Write unit tests | Done | Verified | internal/storage/history_test.go:137-221; testdata/output_*.json |
| 6. Verification (build/test/lint) | Done | Verified | `go build ./...`; `go test ./...`; `make lint` (all pass) |
**Tasks:** 6/6 completed and verified.

### Test Coverage and Gaps
- go test ./... (pass); targeted LoadFromOutput scenarios cover AC-1..AC-5 including missing, empty, corrupted, sorting.
- No additional gaps identified for this story.

### Architectural Alignment
- Follows ADR-004 structured logging with slog (history.go log calls).  
- Uses partial parse per tech spec to extract only `historical`; retains graceful degradation per NFR10 (tech-spec-epic-4.md).

### Security Notes
- No secrets or external IO beyond local file read; permission/read errors degrade with warn log, no crash.

### Best-Practices and References
- Stack: Go 1.24 (go.mod); stdlib-only per ADR-005.  
- Logging: ADR-004 structured slog; messages emitted at DEBUG/WARN for history load paths.  
- Error handling: ADR-003 explicit errors; function returns nil error with graceful fallbacks as required by ACs.

### Action Items
**Code Changes Required:** None.

**Advisory Notes:**
- Note: Repository has additional modified/untracked files (e.g., internal/storage/state.go, internal/storage/writer.go); ensure they are scoped to their respective stories before merge.
