# Story 3.1: Implement Protocol Filtering by Oracle Name

Status: ready-for-dev

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

- [ ] Task 1: Create aggregator package structure (AC: all)
  - [ ] 1.1: Create `internal/aggregator/filter.go` file
  - [ ] 1.2: Add package documentation in `doc.go` if not exists
  - [ ] 1.3: Define `FilterByOracle` function signature

- [ ] Task 2: Implement FilterByOracle function (AC: 1, 2, 3, 4)
  - [ ] 2.1: Implement iteration over protocols slice
  - [ ] 2.2: Check `Oracles` slice for exact case-sensitive match
  - [ ] 2.3: Check `Oracle` string field for exact case-sensitive match (legacy fallback)
  - [ ] 2.4: Return filtered slice containing only matching protocols

- [ ] Task 3: Write unit tests for FilterByOracle (AC: 1, 2, 3, 4, 5)
  - [ ] 3.1: Create `internal/aggregator/filter_test.go`
  - [ ] 3.2: Test: protocol with oracle in `Oracles` array is included
  - [ ] 3.3: Test: protocol with multiple oracles including target is included (AC: 2)
  - [ ] 3.4: Test: protocol with legacy `Oracle` field only is included (AC: 3)
  - [ ] 3.5: Test: protocol without target oracle is excluded (AC: 4)
  - [ ] 3.6: Test: case-sensitive matching (e.g., "switchboard" != "Switchboard")
  - [ ] 3.7: Test: empty input returns empty slice
  - [ ] 3.8: Test: empty oracle name returns empty slice
  - [ ] 3.9: Test: realistic dataset filtering (~500 protocols yields ~21 results) (AC: 5)

- [ ] Task 4: Verification (AC: all)
  - [ ] 4.1: Run `go build ./...` and verify success
  - [ ] 4.2: Run `go test ./internal/aggregator/...` and verify all pass
  - [ ] 4.3: Run `make lint` and verify no errors

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

### Completion Notes List

### File List

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-3-data-processing-pipeline.md |
