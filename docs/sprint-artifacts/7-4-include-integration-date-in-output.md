# Story 7.4: Include Integration Date in Output

Status: ready-for-dev

## Story

As a **TVL data consumer**,
I want **the integration_date field to be included in the tvl-data.json output for all protocols**,
So that **downstream applications can filter TVL history by integration date if needed, while receiving the complete unfiltered time-series data**.

## Acceptance Criteria

Source: [Source: docs/sprint-artifacts/tech-spec-epic-7.md#AC-7.4]; [Source: docs/epics/epic-7-custom-protocols-tvl-charting.md#Story-7.4]; [Source: docs/prd.md#data-filtering--extraction]

**AC1: Custom Protocols Integration Date**
**Given** a custom protocol with a `date` field in config
**When** the protocol is included in tvl-data.json output
**Then** `integration_date` equals the config `date` value (Unix timestamp)

**AC2: Custom Protocols Without Date**
**Given** a custom protocol without a `date` field in config
**When** the protocol is included in tvl-data.json output
**Then** `integration_date` is null

**AC3: Auto-Detected Protocols**
**Given** an auto-detected protocol (from /oracles endpoint)
**When** the protocol is included in tvl-data.json output
**Then** `integration_date` is null

**AC4: Full TVL History Included**
**Given** any protocol (custom or auto-detected)
**When** tvl-data.json is generated
**Then** the full `tvl_history` array is included regardless of `integration_date`
**And** no filtering of TVL data points by date occurs

## Tasks / Subtasks

- [ ] Task 1: Verify MergedProtocol Model Has IntegrationDate (AC: 1, 2, 3)
  - [ ] 1.1: Confirm `IntegrationDate *int64` field exists in `internal/models/tvl.go` MergedProtocol struct
  - [ ] 1.2: Confirm field uses pointer type to distinguish null from zero

- [ ] Task 2: Verify Merger Sets IntegrationDate Correctly (AC: 1, 2, 3)
  - [ ] 2.1: Review `internal/tvl/merger.go` MergeProtocolLists function
  - [ ] 2.2: Confirm auto-detected protocols get `IntegrationDate: nil`
  - [ ] 2.3: Confirm custom protocols get `IntegrationDate` from config `Date` field (nil if not set)

- [ ] Task 3: Define TVLOutputProtocol Schema (AC: 1, 2, 3, 4)
  - [ ] 3.1: Add/update `TVLOutputProtocol` struct in `internal/models/tvl.go` (or `internal/tvl/output.go`) with:
    - `IntegrationDate *int64 json:"integration_date"` - Unix timestamp, nullable
    - `TVLHistory []TVLHistoryItem json:"tvl_history"` - Full history array
  - [ ] 3.2: Ensure JSON serialization outputs `null` for nil IntegrationDate (not omitted)

- [ ] Task 4: Implement Output Mapping (AC: 1, 2, 3, 4)
  - [ ] 4.1: Create function to map `MergedProtocol` + TVL data to `TVLOutputProtocol`
  - [ ] 4.2: Pass through `IntegrationDate` directly from MergedProtocol
  - [ ] 4.3: Include full `tvl_history` array without filtering

- [ ] Task 5: Write Unit Tests (AC: all)
  - [ ] 5.1: Test: Custom protocol with date → integration_date populated in output
  - [ ] 5.2: Test: Custom protocol without date → integration_date is null in output
  - [ ] 5.3: Test: Auto-detected protocol → integration_date is null in output
  - [ ] 5.4: Test: TVL history included in full regardless of integration_date value
  - [ ] 5.5: Test: JSON serialization outputs null (not omits) for nil integration_date

- [ ] Task 6: Build and Test Verification (AC: all)
  - [ ] 6.1: Run `go build ./...` and verify success
  - [ ] 6.2: Run `go test ./internal/tvl/...` and verify all pass
  - [ ] 6.3: Run `go test ./internal/models/...` if applicable

## Dev Notes

### Technical Guidance

**Key Insight:** The `MergedProtocol` model and merger logic (Story 7.3) already handle integration_date correctly. This story focuses on ensuring the **output generation** layer (to be implemented in Story 7.5) properly passes through this field.

**Files to Verify (already exist from 7.3):**
- `internal/models/tvl.go` - Contains `MergedProtocol` with `IntegrationDate *int64`
- `internal/tvl/merger.go` - Sets IntegrationDate from custom config or nil for auto

**Files to Create/Modify:**
- `internal/models/tvl.go` or `internal/tvl/output.go` - Add `TVLOutputProtocol` struct
- `internal/tvl/output_test.go` - Tests for integration_date in output

### Learnings from Previous Story

**From Story 7-3-merge-protocol-lists (Status: done)**

- **MergedProtocol Model Available**: `internal/models/tvl.go:17-28` - already has `IntegrationDate *int64` field
- **Merger Already Handles Integration Date**:
  - Auto protocols: `IntegrationDate: nil` (internal/tvl/merger.go:16-25)
  - Custom protocols: `IntegrationDate` = config `Date` field, nil if not set (internal/tvl/merger.go:28-37)
- **Tests Exist**: Nil date handling tested in `internal/tvl/merger_test.go:139-152`
- **No New Dependencies**: Pure logic, stdlib only (ADR-005 compliant)

[Source: docs/sprint-artifacts/7-3-merge-protocol-lists.md#Dev-Agent-Record]

### Architecture Patterns and Constraints

- **ADR-003**: Explicit error returns; no panics [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-003]
- **ADR-005**: No new external dependencies [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-005]
- **JSON Null Handling**: Use pointer types (`*int64`) to ensure JSON serialization outputs `null` rather than omitting the field or using zero value

### Data Model Reference

**Existing MergedProtocol (internal/models/tvl.go:17-28):**
```go
type MergedProtocol struct {
    Slug            string  `json:"slug"`
    Name            string  `json:"name"`
    Source          string  `json:"source"`
    IsOngoing       bool    `json:"is_ongoing"`
    SimpleTVSRatio  float64 `json:"simple_tvs_ratio"`
    IntegrationDate *int64  `json:"integration_date"` // <-- Already exists
    DocsProof       *string `json:"docs_proof"`
    GitHubProof     *string `json:"github_proof"`
}
```

**TVLOutputProtocol (to add for Story 7.5, referenced here):**
```go
type TVLOutputProtocol struct {
    Name            string           `json:"name"`
    Slug            string           `json:"slug"`
    Source          string           `json:"source"`
    IsOngoing       bool             `json:"is_ongoing"`
    SimpleTVSRatio  float64          `json:"simple_tvs_ratio"`
    IntegrationDate *int64           `json:"integration_date"` // Passed through from MergedProtocol
    DocsProof       *string          `json:"docs_proof"`
    GitHubProof     *string          `json:"github_proof"`
    CurrentTVL      float64          `json:"current_tvl"`
    TVLHistory      []TVLHistoryItem `json:"tvl_history"`     // Full history, no filtering
}
```

### Relationship to Story 7.5

This story (7.4) establishes the **contract** for how `integration_date` flows through the pipeline. The actual output file generation is in Story 7.5. The key constraints are:

1. `integration_date` MUST be included in output JSON (not omitted)
2. Value is `null` for auto-detected and custom protocols without date
3. Value is Unix timestamp for custom protocols with date
4. `tvl_history` is NOT filtered by integration_date (downstream responsibility)

### Project Structure Notes

- No new files required if TVLOutputProtocol is added to existing `internal/models/tvl.go`
- Alternatively, output-specific structs can go in `internal/tvl/output.go` (Story 7.5 will create this)
- Follow existing package organization from Story 7.1 and 7.3

### Testing Strategy

- Table-driven tests with named cases [Source: docs/architecture/testing-strategy.md]
- Test JSON serialization to verify `null` output (not field omission)
- Verify pointer semantics for nullable int64

### References

- [Source: docs/sprint-artifacts/tech-spec-epic-7.md#AC-7.4] - Acceptance criteria definition
- [Source: docs/sprint-artifacts/tech-spec-epic-7.md#Data-Models-and-Contracts] - TVLOutputProtocol struct definition
- [Source: docs/epics/epic-7-custom-protocols-tvl-charting.md#Story-7.4] - Epic story definition
- [Source: docs/prd.md#data-filtering--extraction] - Product requirement for TVL metadata extraction
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-003] - Error handling
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-005] - Dependency constraints
- [Source: docs/sprint-artifacts/7-3-merge-protocol-lists.md#Dev-Agent-Record] - Previous story learnings
- [Source: internal/models/tvl.go:17-28] - MergedProtocol with IntegrationDate field

## Dev Agent Record

### Context Reference

<!-- Path(s) to story context XML will be added here by context workflow -->

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-12-08 | SM Agent (Bob) | Initial story draft created from Epic 7 / Tech Spec |
