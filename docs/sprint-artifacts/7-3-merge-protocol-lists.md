# Story 7.3: Merge Protocol Lists

Status: done

## Story

As a **TVL pipeline developer**,
I want **to combine auto-detected protocols (from `/oracles` endpoint) with custom protocols (from config), deduplicating by slug with custom taking precedence**,
So that **the TVL charting pipeline has a complete, unified list of all tracked protocols with correct source attribution and metadata**.

## Acceptance Criteria

Source: [Source: docs/sprint-artifacts/tech-spec-epic-7.md#AC-7.3]; [Source: docs/epics/epic-7-custom-protocols-tvl-charting.md#Story-7.3]; [Source: docs/prd.md#additional-protocol-sources]

**AC1: Combine Auto-Detected and Custom Protocols**
**Given** a list of auto-detected protocol slugs (from existing aggregator filtering)
**And** a list of custom protocols (loaded via `CustomLoader`)
**When** `MergeProtocolLists()` is called
**Then** it returns a combined list containing protocols from both sources

**AC2: Auto-Detected Protocol Defaults**
**Given** a protocol slug appears only in the auto-detected list
**When** it is added to the merged result
**Then** it is assigned `source: "auto"`
**And** `simple_tvs_ratio: 1.0`
**And** `integration_date: nil`
**And** `is_ongoing: true`
**And** `docs_proof: "https://defillama.com/protocol/{slug}"` (generated from slug)

**AC3: Custom Protocol Metadata**
**Given** a protocol appears in the custom protocols list
**When** it is added to the merged result
**Then** it is assigned `source: "custom"`
**And** preserves `simple_tvs_ratio` from config
**And** preserves `integration_date` from config `date` field (nil if not set)
**And** preserves `is_ongoing` from config
**And** preserves `docs_proof` and `github_proof` from config

**AC4: Deduplication with Custom Precedence**
**Given** a protocol slug exists in both auto-detected and custom lists
**When** the lists are merged
**Then** the custom protocol's metadata takes precedence (overwrites auto)
**And** the protocol appears exactly once in the result

**AC5: Result Sorted Alphabetically**
**Given** the merged list contains multiple protocols
**When** `MergeProtocolLists()` returns
**Then** the result is sorted alphabetically by slug (ascending)

**AC6: Return Type**
**Given** any combination of inputs
**When** `MergeProtocolLists()` completes
**Then** it returns `[]MergedProtocol` (never nil, may be empty)

## Tasks / Subtasks

- [x] Task 1: Define MergedProtocol Model (AC: 2, 3, 6)
  - [x] 1.1: Add `MergedProtocol` struct to `internal/models/tvl.go`:
    - `Slug string` - Protocol identifier
    - `Name string` - Display name (populated later by TVL fetch)
    - `Source string` - "auto" | "custom"
    - `IsOngoing bool` - Whether integration is ongoing
    - `SimpleTVSRatio float64` - TVS multiplier [0,1]
    - `IntegrationDate *int64` - Unix timestamp, nullable
    - `DocsProof *string` - Documentation URL, nullable
    - `GitHubProof *string` - GitHub proof URL, nullable
  - [x] 1.2: Add JSON tags matching output schema

- [x] Task 2: Create merger.go File (AC: 1)
  - [x] 2.1: Create `internal/tvl/merger.go` with package declaration
  - [x] 2.2: Add imports: `sort`, `fmt`, `github.com/switchboard-xyz/defillama-extract/internal/models`

- [x] Task 3: Implement MergeProtocolLists Function (AC: 1, 2, 3, 4, 5, 6)
  - [x] 3.1: Define signature: `func MergeProtocolLists(autoSlugs []string, custom []models.CustomProtocol) []models.MergedProtocol`
  - [x] 3.2: Create `map[string]models.MergedProtocol` for deduplication
  - [x] 3.3: Iterate auto-detected slugs, add to map with auto defaults (source="auto", ratio=1.0, date=nil, ongoing=true, docs_proof generated)
  - [x] 3.4: Iterate custom protocols, upsert to map with custom values (overwrites if exists)
  - [x] 3.5: Convert map values to slice
  - [x] 3.6: Sort slice by slug (ascending, alphabetical)
  - [x] 3.7: Return sorted slice (empty slice if no inputs, never nil)

- [x] Task 4: Helper for Auto DocsProof URL (AC: 2)
  - [x] 4.1: Create helper function to generate DefiLlama URL: `fmt.Sprintf("https://defillama.com/protocol/%s", slug)`

- [x] Task 5: Write Unit Tests (AC: all)
  - [x] 5.1: Create `internal/tvl/merger_test.go`
  - [x] 5.2: Test: Auto slugs only → all get auto defaults
  - [x] 5.3: Test: Custom protocols only → all get custom metadata
  - [x] 5.4: Test: Overlap (auto + custom with same slug) → custom wins
  - [x] 5.5: Test: Mixed (some overlap, some unique) → correct merge
  - [x] 5.6: Test: Empty inputs → empty slice returned
  - [x] 5.7: Test: Result sorted alphabetically by slug
  - [x] 5.8: Test: Auto docs_proof URL generated correctly
  - [x] 5.9: Test: Custom with nil date → integration_date is nil
  - [x] 5.10: Test: Custom with date → integration_date populated

- [x] Task 6: Build and Lint Verification (AC: all)
  - [x] 6.1: Run `go build ./...` and verify success
  - [x] 6.2: Run `go test ./internal/tvl/...` and verify all pass
  - [x] 6.3: Run `make lint` and fix any issues

### Review Follow-ups (AI)

- [x] [AI-Review][High] Add unit test covering custom protocol with nil `date` to assert `IntegrationDate` remains nil in `MergeProtocolLists` (AC3, Task 5.9) [file: internal/tvl/merger_test.go]

## Dev Notes

### Technical Guidance

**Files to Modify:**
- `internal/models/tvl.go` - Add `MergedProtocol` struct

**Files to Create:**
- `internal/tvl/merger.go` - Protocol merging logic
- `internal/tvl/merger_test.go` - Unit tests

### Learnings from Previous Story

**From Story 7-2-implement-protocol-tvl-fetcher (Status: done)**

- **New API Method Created**: `FetchProtocolTVL(ctx, slug)` available at `internal/api/client.go` - returns `*ProtocolTVLResponse` with Name, TVL[], CurrentChainTvls
- **Response Models Available**: `ProtocolTVLResponse` and `TVLDataPoint` at `internal/api/responses.go:30-41`
- **Rate Limiter**: Built-in 200ms rate limiting already implemented in `FetchProtocolTVL` - caller does NOT need to add throttling
- **404 Handling Pattern**: Returns `nil, nil` on 404 (not error) with warning log - useful pattern for missing protocols
- **Advisory**: Consider URL-escaping slugs (low severity)
- **New/Modified Files (for traceability)**:
  - `internal/api/client.go` — FetchProtocolTVL implementation (404 handling, rate limiter)
  - `internal/api/responses.go` — ProtocolTVLResponse, TVLDataPoint structs
  - `internal/api/endpoints.go` — Protocol TVL endpoint template
  - `internal/api/tvl_test.go` — TVL fetcher unit tests
  - `testdata/protocol_tvl_response.json` — sample success payload
  - `testdata/protocol_404_response.json` — sample 404 payload
  - Use these as source evidence when integrating merger output with TVL fetcher. [Source: docs/sprint-artifacts/7-2-implement-protocol-tvl-fetcher.md#Dev-Agent-Record]

**From Story 7-1 (inferred from existing code)**

- **CustomProtocol Model**: Available at `internal/models/tvl.go` with fields: Slug, IsOngoing, Live, Date, SimpleTVSRatio, DocsProof, GitHubProof
- **CustomLoader**: Available at `internal/tvl/custom.go` - use `Load(ctx)` to get `[]models.CustomProtocol`
- **Validation**: Already handles missing file (returns empty list), invalid JSON (returns error), filters `live: false`

[Source: docs/sprint-artifacts/7-2-implement-protocol-tvl-fetcher.md#Dev-Agent-Record]

### Architecture Patterns and Constraints

- **ADR-003**: Explicit error returns; wrap errors with context (`fmt.Errorf("...: %w", err)`) [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-003]
- **ADR-005**: No new external dependencies; use standard library [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-005]
- **Pattern**: Follow existing `internal/tvl/` package structure established in Story 7.1

### Data Model Reference

**MergedProtocol** (to add in `models/tvl.go`):
```go
// MergedProtocol represents a protocol in the merged list with source attribution.
// Used as the unified representation after combining auto-detected and custom protocols.
type MergedProtocol struct {
    Slug            string   `json:"slug"`
    Name            string   `json:"name"`              // Populated later by TVL fetch
    Source          string   `json:"source"`            // "auto" | "custom"
    IsOngoing       bool     `json:"is_ongoing"`
    SimpleTVSRatio  float64  `json:"simple_tvs_ratio"`
    IntegrationDate *int64   `json:"integration_date"`  // Unix timestamp, nullable
    DocsProof       *string  `json:"docs_proof"`
    GitHubProof     *string  `json:"github_proof"`
}
```

### Merge Algorithm Reference

From tech spec [Source: docs/sprint-artifacts/tech-spec-epic-7.md#Protocol-Merge-Flow]:
```
MergeProtocolLists(autoSlugs, customProtocols)
    │
    ├─► Create map[slug]MergedProtocol
    │
    ├─► For each autoSlug:
    │       └─► Add to map with:
    │           - Source: "auto"
    │           - SimpleTVSRatio: 1.0
    │           - IntegrationDate: nil
    │           - IsOngoing: true
    │
    ├─► For each customProtocol:
    │       └─► Upsert to map (overwrites auto if exists):
    │           - Source: "custom"
    │           - SimpleTVSRatio: from config
    │           - IntegrationDate: from config (nil if not set)
    │           - Proof URLs: from config
    │
    └─► Return sorted slice (by slug)
```

### Test Scenarios Reference

From tech spec [Source: docs/sprint-artifacts/tech-spec-epic-7.md#Protocol-Merging-7.3]:

| Auto Slugs | Custom Slugs | Expected Result |
|------------|--------------|-----------------|
| [a, b] | [] | [a(auto), b(auto)] |
| [] | [c, d] | [c(custom), d(custom)] |
| [a, b] | [b, c] | [a(auto), b(custom), c(custom)] |
| [a] | [a] | [a(custom)] - custom wins |

### Getting Auto-Detected Slugs

Auto-detected protocols are obtained by:
1. Fetching all protocols from `/lite/protocols2` via `api.Client.FetchProtocols(ctx)`
2. Filtering by oracle name using `aggregator.FilterByOracle(protocols, "Switchboard")`
3. Extracting slugs from the filtered `[]api.Protocol` results

For this story, the merger function takes pre-extracted `[]string` slugs as input. The caller (Story 7.6) is responsible for obtaining them from the existing aggregator pipeline.

### Project Structure Notes

- Add `MergedProtocol` to existing `internal/models/tvl.go` (alongside `CustomProtocol`)
- Create `merger.go` in `internal/tvl/` package (alongside `custom.go`)
- Test file follows naming convention: `merger_test.go` in same package

### Testing Strategy

- Table-driven tests with named cases [Source: docs/architecture/testing-strategy.md]
- No HTTP mocking needed - pure logic function
- Test edge cases: empty inputs, duplicates, sorting verification
- Verify pointer fields (DocsProof, GitHubProof, IntegrationDate) handled correctly for both nil and non-nil cases

### References

- [Source: docs/sprint-artifacts/tech-spec-epic-7.md#AC-7.3] - Acceptance criteria definition
- [Source: docs/sprint-artifacts/tech-spec-epic-7.md#Data-Models-and-Contracts] - MergedProtocol struct definition
- [Source: docs/sprint-artifacts/tech-spec-epic-7.md#Protocol-Merging-7.3] - Test scenarios
- [Source: docs/epics/epic-7-custom-protocols-tvl-charting.md#Story-7.3] - Epic story definition
- [Source: docs/prd.md#additional-protocol-sources] - Product requirement for covering protocols DefiLlama may miss
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-003] - Error handling
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-005] - Dependency constraints
- [Source: docs/architecture/testing-strategy.md] - Testing standards

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/7-3-merge-protocol-lists.context.xml

### Agent Model Used

gpt-5-codex

### Debug Log References
- Plan: map auto+custom merge, ensure custom overwrite, sort, keep slice non-nil.
- go build ./... → pass.
- go test ./internal/tvl/... → pass.
- make lint → pass.

### Completion Notes List

- Implemented MergeProtocolLists with auto defaults, custom overwrite, sorted output; added MergedProtocol model.
- Added helper for auto docs_proof URL and comprehensive unit coverage per ACs.
- Build, targeted tests, and lint all passing.
### File List

- internal/models/tvl.go
- internal/tvl/merger.go
- internal/tvl/merger_test.go
- docs/sprint-artifacts/sprint-status.yaml
- docs/sprint-artifacts/7-3-merge-protocol-lists.md
## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-12-08 | SM Agent (Bob) | Initial story draft created from Epic 7 / Tech Spec |
| 2025-12-08 | Amelia (Dev) | Implemented protocol merge logic, model, tests, and updated sprint status |
| 2025-12-08 | BMad | Senior Developer Review (AI) — Systematic validation; status moved to done |

## Senior Developer Review (AI)

Reviewer: BMad  
Date: 2025-12-08  
Outcome: Approve (all ACs implemented; no findings)

### Summary
- AC1–AC6 fully met; merge logic uses map dedupe with custom precedence and alphabetic sort; returns non-nil slice.
- Tests exercise auto defaults, custom metadata (nil/non-nil dates), overlap precedence, sorting, empty inputs, and docs proof generation.
- Build, package tests, and lint all pass on current commit.

### Key Findings
- None (no defects or gaps identified).

### Acceptance Criteria Coverage
| AC | Status | Evidence |
|----|--------|----------|
| AC1 Combine lists | Implemented | map merge of auto + custom (internal/tvl/merger.go:13-44) |
| AC2 Auto defaults | Implemented | defaults set for auto slugs (internal/tvl/merger.go:16-25); validated in TestMergeProtocolLists_AutoOnly (internal/tvl/merger_test.go:11-31) |
| AC3 Custom metadata | Implemented | custom fields preserved incl. nil date (internal/tvl/merger.go:28-37); tests cover nil/non-nil dates (internal/tvl/merger_test.go:33-80,139-152) |
| AC4 Custom precedence | Implemented | custom overwrites auto entry (internal/tvl/merger.go:28-37); confirmed in CustomOverridesAuto test (internal/tvl/merger_test.go:82-108) |
| AC5 Sorted result | Implemented | sort by slug ascending (internal/tvl/merger.go:45-47); Sorting test (internal/tvl/merger_test.go:110-125) |
| AC6 Return type | Implemented | returns slice not nil (internal/tvl/merger.go:13-49); empty-input test (internal/tvl/merger_test.go:128-137) |

### Task Completion Validation
| Task | Status | Evidence |
|------|--------|----------|
| 1: MergedProtocol model | Verified | struct with JSON tags added (internal/models/tvl.go:17-28) |
| 2: merger.go scaffolding | Verified | file with fmt/sort imports (internal/tvl/merger.go:1-8) |
| 3: MergeProtocolLists impl | Verified | map dedupe, defaults, overwrite, sort, non-nil return (internal/tvl/merger.go:13-49) |
| 4: Auto docs_proof helper | Verified | autoDocsProof generates URL (internal/tvl/merger.go:52-54) |
| 5: Unit tests | Verified | coverage across scenarios incl. nil date and docs proof (internal/tvl/merger_test.go:11-152) |
| 6: Build/test/lint | Verified | `go test ./...`, `make lint` on 2025-12-08 (all pass) |

### Test Coverage and Gaps
- Executed: `go test ./...` (pass); `make lint` (pass). No uncovered AC cases remain.

### Architectural Alignment
- No new deps (ADR-005); pure function, deterministic, stdlib only; package placement matches tvl patterns.

### Security Notes
- Pure in-memory merge; no I/O or network surface.

### Best-Practices and References
- Testing: docs/architecture/testing-strategy.md.
- Dependencies: docs/architecture/architecture-decision-records-adrs.md#ADR-005.

### Action Items
- None.
