# Story 3.1: Implement Protocol Filtering by Oracle Name

Status: done

## Story

As a **developer**,
I want **to filter protocols that use Switchboard as their oracle**,
so that **only relevant protocols are included in aggregations**.

## Acceptance Criteria

Source: Epic 3.1 / PRD FR9-FR10

1. **Given** a list of protocols from the API **When** `FilterByOracle(protocols []Protocol, oracleName string)` is called with "Switchboard" **Then** only protocols where `oracles` array contains "Switchboard" (exact match, case-sensitive) OR `oracle` field equals "Switchboard" (legacy field check) are returned

2. **Given** a protocol with `oracles: ["Chainlink", "Switchboard"]` **When** filtering for "Switchboard" **Then** the protocol IS included (multi-oracle protocol)

3. **Given** a protocol with `oracle: "Switchboard"` but empty `oracles` array **When** filtering for "Switchboard" **Then** the protocol IS included (legacy field fallback)

4. **Given** a protocol with `oracles: ["Chainlink"]` and `oracle: ""` **When** filtering for "Switchboard" **Then** the protocol is NOT included

5. **Given** ~500 protocols from the API **When** filtering for "Switchboard" **Then** approximately 21 protocols are returned (expected count per PRD)

## Tasks / Subtasks

- [x] Task 1: Create aggregator package structure (AC: all)
  - [x] 1.1: Create `internal/aggregator/filter.go` file
  - [x] 1.2: Add package documentation in `doc.go` if not exists
  - [x] 1.3: Define `FilterByOracle` function signature

- [x] Task 2: Implement FilterByOracle function (AC: 1, 2, 3, 4)
  - [x] 2.1: Implement iteration over protocols slice
  - [x] 2.2: Check `Oracles` slice for exact case-sensitive match
  - [x] 2.3: Check `Oracle` string field for exact case-sensitive match (legacy fallback)
  - [x] 2.4: Return filtered slice containing only matching protocols

- [x] Task 3: Write unit tests for FilterByOracle (AC: 1, 2, 3, 4, 5)
  - [x] 3.1: Create `internal/aggregator/filter_test.go`
  - [x] 3.2: Test: protocol with oracle in `Oracles` array is included
  - [x] 3.3: Test: protocol with multiple oracles including target is included (AC: 2)
  - [x] 3.4: Test: protocol with legacy `Oracle` field only is included (AC: 3)
  - [x] 3.5: Test: protocol without target oracle is excluded (AC: 4)
  - [x] 3.6: Test: case-sensitive matching (e.g., "switchboard" != "Switchboard")
  - [x] 3.7: Test: empty input returns empty slice
  - [x] 3.8: Test: empty oracle name returns empty slice
  - [x] 3.9: Test: realistic dataset filtering (~500 protocols yields ~21 results) (AC: 5)

- [x] Task 4: Verification (AC: all)
  - [x] 4.1: Run `go build ./...` and verify success
  - [x] 4.2: Run `go test ./internal/aggregator/...` and verify all pass
  - [x] 4.3: Run `make lint` and verify no errors

## Dev Notes

### Technical Guidance

- **Package Location:** Create new package at `internal/aggregator/filter.go`
- **Input Type:** Use `[]api.Protocol` from `internal/api/responses.go` - the Protocol struct already has `Oracles []string` and `Oracle string` fields
- **Output Type:** Return `[]api.Protocol` (same type, filtered)
- **Matching Logic:** Exact string match, case-sensitive. Check both fields:
  1. First check `Oracles` slice using a simple loop
  2. If not found in slice, check legacy `Oracle` field
- **No External Dependencies:** Pure Go, no additional packages needed

### Implementation Pattern

```go
// FilterByOracle returns protocols that use the specified oracle.
// Checks both Oracles slice (preferred) and Oracle field (legacy).
func FilterByOracle(protocols []api.Protocol, oracleName string) []api.Protocol {
    if oracleName == "" {
        return nil
    }
    var result []api.Protocol
    for _, p := range protocols {
        if containsOracle(p.Oracles, oracleName) || p.Oracle == oracleName {
            result = append(result, p)
        }
    }
    return result
}

func containsOracle(oracles []string, target string) bool {
    for _, o := range oracles {
        if o == target {
            return true
        }
    }
    return false
}
```

### Testing Strategy

Follow table-driven test pattern established in `internal/api/*_test.go`; adhere to `docs/architecture/testing-strategy.md` expectations for coverage of success/error paths and ordering:

```go
func TestFilterByOracle(t *testing.T) {
    tests := []struct {
        name       string
        protocols  []api.Protocol
        oracleName string
        wantCount  int
        wantSlugs  []string // verify specific protocols matched
    }{
        {
            name: "filters protocols with oracle in Oracles array",
            // ...
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := FilterByOracle(tt.protocols, tt.oracleName)
            // assertions
        })
    }
}
```

### Project Structure Notes

- New file: `internal/aggregator/filter.go` - aligns with fr-category-to-architecture-mapping.md
- Test file: `internal/aggregator/filter_test.go`
- Package doc: `internal/aggregator/doc.go` already exists (placeholder from project init)
- Import path: `github.com/switchboard-xyz/defillama-extract/internal/aggregator`

### Learnings from Previous Story

**From Story 2-6-implement-api-request-logging (Status: done)**

- **Protocol Struct Available:** `api.Protocol` struct in `internal/api/responses.go` has all needed fields (`Oracles []string`, `Oracle string`, `Name`, `Slug`, etc.)
- **FetchAll Available:** Use `client.FetchAll()` to get `FetchResult` containing `Protocols []Protocol` for integration testing
- **Test Patterns:** Table-driven tests with mock server established in `internal/api/*_test.go`
- **Build/Lint Commands:** `go build ./...`, `go test ./...`, `make lint` all verified working
- **Review Outcome:** Story 2.6 approved with no action items

[Source: docs/sprint-artifacts/2-6-implement-api-request-logging.md#Dev-Agent-Record]

### References

- [Source: docs/epics/epic-3-data-processing-pipeline.md#story-31] - Story definition and acceptance criteria
- [Source: docs/prd.md#FR9] - FR9: System filters protocols by exact oracle name match ("Switchboard")
- [Source: docs/prd.md#FR10] - FR10: System checks both `oracles` array and legacy `oracle` field for matching
- [Source: docs/architecture/fr-category-to-architecture-mapping.md#fr-category-to-architecture-mapping] - Data Filtering (FR9-FR14) maps to `internal/aggregator` package
- [Source: docs/architecture/data-architecture.md#L16] - Protocol struct definition
- [Source: internal/api/responses.go#L16] - Protocol type with Oracles/Oracle fields

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/3-1-implement-protocol-filtering-by-oracle-name.context.xml

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

- 2025-11-30: Plan — add `FilterByOracle` + helper in `internal/aggregator/filter.go`, create table-driven tests covering ACs including ~500 protocol dataset; run gofmt, go test (aggregator), go build ./..., make lint; update story checkboxes, File List, Change Log, and sprint status.

### Completion Notes List

- Implemented case-sensitive `FilterByOracle` with legacy fallback and comprehensive table-driven tests covering all AC scenarios including ~500 protocol dataset count; verified via gofmt, go test ./internal/aggregator/..., go build ./..., make lint on 2025-11-30.

### File List

- internal/aggregator/filter.go
- internal/aggregator/filter_test.go
- docs/sprint-artifacts/3-1-implement-protocol-filtering-by-oracle-name.md
- docs/sprint-artifacts/sprint-status.yaml

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-3-data-processing-pipeline.md |
| 2025-11-30 | Dev Agent (Amelia) | Implemented FilterByOracle with tests; updated story status to in-progress and sprint-status.yaml |
| 2025-11-30 | Dev Reviewer (Amelia) | Senior Developer Review completed; story approved and status updated to done |

## Senior Developer Review (AI)

- Reviewer: BMad  
- Date: 2025-11-30  
- Outcome: Approve — All ACs implemented with passing tests; no issues found.

### Summary
- FilterByOracle matches Switchboard via both Oracles slice and legacy Oracle field, returning expected 21 protocols in synthetic dataset; unit tests cover all AC scenarios.

### Key Findings
- No High/Medium/Low findings. Implementation aligns with requirements and coding standards.

### Acceptance Criteria Coverage

| AC# | Description | Status | Evidence |
|-----|-------------|--------|----------|
| AC1 | Filter returns protocols where Oracles contains target or legacy Oracle equals target (case-sensitive) | IMPLEMENTED | internal/aggregator/filter.go:7-19; internal/aggregator/filter_test.go:18-26,89-102 |
| AC2 | Multi-oracle protocol included | IMPLEMENTED | internal/aggregator/filter_test.go:28-35,89-102 |
| AC3 | Legacy Oracle-only protocol included | IMPLEMENTED | internal/aggregator/filter_test.go:37-44,89-102 |
| AC4 | Non-matching protocol excluded (case-sensitive) | IMPLEMENTED | internal/aggregator/filter_test.go:46-60,89-102 |
| AC5 | ~500 protocols yield ~21 Switchboard results | IMPLEMENTED | internal/aggregator/filter_test.go:76-86,106-125 |

Summary: 5 of 5 acceptance criteria fully implemented.

### Task Completion Validation

| Task | Marked As | Verified As | Evidence |
|------|-----------|-------------|----------|
| Task 1: Create aggregator package structure | Complete | VERIFIED COMPLETE | internal/aggregator/filter.go:1-20; internal/aggregator/doc.go:1 |
| Task 2: Implement FilterByOracle function | Complete | VERIFIED COMPLETE | internal/aggregator/filter.go:7-19,22-29 |
| Task 3: Write unit tests for FilterByOracle | Complete | VERIFIED COMPLETE | internal/aggregator/filter_test.go:10-145 |
| Task 4: Verification (build/test/lint) | Complete | VERIFIED COMPLETE | go test ./...; go build ./...; make lint (2025-11-30) |

Summary: 4 of 4 tasks (23 checkboxes) verified complete; 0 questionable; 0 falsely marked complete.

### Test Coverage and Gaps
- Unit tests cover Oracles slice, multi-oracle, legacy Oracle fallback, exclusion, case sensitivity, empty inputs, and dataset count; no gaps noted.

### Architectural Alignment
- Uses api.Protocol from internal/api/responses.go and lives in internal/aggregator per architecture mapping; follows table-driven test convention from testing-strategy.

### Security Notes
- Pure in-memory filtering; no external I/O or inputs beyond provided slices; no security risks observed.

### Best-Practices and References
- Slice preallocation via `make([]api.Protocol, 0, len(protocols))` follows performance guidance to avoid reallocation citeturn0search0.
- Table-driven subtests with named cases align with Go testing best practices for readability and coverage citeturn0search2.

### Action Items

**Code Changes Required:**  
- None.

**Advisory Notes:**  
- Note: No additional actions recommended.
