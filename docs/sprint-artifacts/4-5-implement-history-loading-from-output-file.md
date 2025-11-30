# Story 4.5: Implement History Loading from Output File

Status: ready-for-dev

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

- [ ] Task 1: Implement LoadFromOutput function (AC: 1, 2)
  - [ ] 1.1: Add `LoadFromOutput(outputPath string, logger *slog.Logger) ([]aggregator.Snapshot, error)` to `internal/storage/history.go`
  - [ ] 1.2: Read file using `os.ReadFile(outputPath)`
  - [ ] 1.3: Define minimal struct to extract only `historical` field: `type outputHistoryExtract struct { Historical []aggregator.Snapshot json:"historical" }`
  - [ ] 1.4: Unmarshal JSON into extract struct
  - [ ] 1.5: Sort snapshots by timestamp ascending using `sort.Slice`
  - [ ] 1.6: Add doc comment explaining the function's purpose and graceful handling

- [ ] Task 2: Handle missing file gracefully (AC: 3)
  - [ ] 2.1: Check for `os.IsNotExist(err)` after `os.ReadFile`
  - [ ] 2.2: Log debug message: "no existing history found, starting fresh" with path attribute
  - [ ] 2.3: Return empty `[]aggregator.Snapshot{}` and nil error

- [ ] Task 3: Handle empty/missing historical field (AC: 4)
  - [ ] 3.1: After successful unmarshal, check if `Historical` slice is nil or empty
  - [ ] 3.2: Return empty slice (already initialized by Go) - no special handling needed

- [ ] Task 4: Handle corrupted file gracefully (AC: 5)
  - [ ] 4.1: If `json.Unmarshal` returns error, log warn: "failed to load history, starting fresh" with path and error attributes
  - [ ] 4.2: Return empty `[]aggregator.Snapshot{}` and nil error (graceful degradation)

- [ ] Task 5: Write unit tests for LoadFromOutput (AC: 1-5)
  - [ ] 5.1: Create test fixtures in `internal/storage/testdata/` directory
  - [ ] 5.2: Create `output_with_history.json` fixture with valid historical data (multiple snapshots, unsorted)
  - [ ] 5.3: Create `output_no_history.json` fixture with valid JSON but no/empty historical field
  - [ ] 5.4: Create `output_corrupted.json` fixture with invalid JSON
  - [ ] 5.5: Test: valid output file returns sorted snapshots
  - [ ] 5.6: Test: missing file returns empty slice, no error
  - [ ] 5.7: Test: empty historical field returns empty slice
  - [ ] 5.8: Test: corrupted file returns empty slice, no error, warn logged
  - [ ] 5.9: Test: verify sort order is timestamp ascending

- [ ] Task 6: Verification (AC: all)
  - [ ] 6.1: Run `go build ./...` and verify success
  - [ ] 6.2: Run `go test ./internal/storage/...` and verify all pass
  - [ ] 6.3: Run `make lint` and verify no errors

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

### Completion Notes List

### File List

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-4 and tech-spec-epic-4.md |
