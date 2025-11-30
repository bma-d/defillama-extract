# Story 3.5: Rank Protocols and Identify Largest

Status: ready-for-dev

## Story

As a **developer**,
I want **protocols ranked by TVL and the largest protocol identified**,
so that **I can show protocol importance and highlight top contributors**.

## Acceptance Criteria

Source: Epic 3.5 / PRD FR18, FR23

1. **Given** aggregated protocol data **When** `RankProtocols(protocols []AggregatedProtocol)` is called **Then** protocols are sorted by TVL descending **And** each protocol is assigned a `Rank` field (1, 2, 3...)

2. **Given** ranked protocols **When** identifying largest protocol **Then** protocol with rank 1 is returned **And** `LargestProtocol` struct contains: Name, Slug, TVL, TVS

3. **Given** two protocols with identical TVL **When** ranking **Then** alphabetical order by name is used as tiebreaker

4. **Given** an empty protocols slice **When** ranking **Then** an empty slice is returned (no panic)

5. **Given** an empty protocols slice **When** identifying largest protocol **Then** nil or zero-value `LargestProtocol` is returned with appropriate handling (no panic)

6. **Given** ranked protocols **When** output is serialized **Then** `Rank` field appears in JSON output for each protocol

## Tasks / Subtasks

- [ ] Task 1: Define LargestProtocol struct (AC: 2)
  - [ ] 1.1: Add `LargestProtocol` struct to `internal/aggregator/models.go`
  - [ ] 1.2: Include fields: `Name` (string), `Slug` (string), `TVL` (float64), `TVS` (float64)
  - [ ] 1.3: Add JSON struct tags for output serialization

- [ ] Task 2: Add Rank field to AggregatedProtocol (AC: 1, 6)
  - [ ] 2.1: Add `Rank` field (int) with JSON tag to `AggregatedProtocol` struct in `internal/aggregator/models.go`

- [ ] Task 3: Implement RankProtocols function (AC: 1, 3, 4)
  - [ ] 3.1: Add function signature `func RankProtocols(protocols []AggregatedProtocol) []AggregatedProtocol` to `internal/aggregator/metrics.go`
  - [ ] 3.2: Handle empty input gracefully (return empty slice)
  - [ ] 3.3: Sort protocols by TVL descending using `sort.Slice()`
  - [ ] 3.4: Use alphabetical name order as tiebreaker for equal TVL
  - [ ] 3.5: Assign Rank field starting from 1 (not 0)
  - [ ] 3.6: Return the sorted and ranked slice

- [ ] Task 4: Implement GetLargestProtocol function (AC: 2, 5)
  - [ ] 4.1: Add function signature `func GetLargestProtocol(protocols []AggregatedProtocol) *LargestProtocol` to `internal/aggregator/metrics.go`
  - [ ] 4.2: Handle empty input gracefully (return nil)
  - [ ] 4.3: Return protocol with Rank 1 (assumes protocols already ranked, or find max TVL)
  - [ ] 4.4: Create and return `LargestProtocol` with Name, Slug, TVL, TVS from top protocol

- [ ] Task 5: Write unit tests (AC: 1, 2, 3, 4, 5, 6)
  - [ ] 5.1: Add tests to `internal/aggregator/metrics_test.go`
  - [ ] 5.2: Test: Protocols sorted by TVL descending
  - [ ] 5.3: Test: Rank field assigned correctly (1, 2, 3...)
  - [ ] 5.4: Test: Tiebreaker uses alphabetical name order
  - [ ] 5.5: Test: Empty protocols slice returns empty result (no panic)
  - [ ] 5.6: Test: GetLargestProtocol returns correct protocol
  - [ ] 5.7: Test: GetLargestProtocol with empty slice returns nil (no panic)
  - [ ] 5.8: Test: Rank field appears in JSON serialization

- [ ] Task 6: Verification (AC: all)
  - [ ] 6.1: Run `go build ./...` and verify success
  - [ ] 6.2: Run `go test ./internal/aggregator/...` and verify all pass
  - [ ] 6.3: Run `make lint` and verify no errors

## Dev Notes

### Technical Guidance

- **Package Location:** Add to `internal/aggregator/` package (same as filter.go, extractor.go, metrics.go)
- **Input Type:** `[]AggregatedProtocol` - output from Story 3.2's ExtractProtocolData
- **Output Types:**
  - `RankProtocols`: `[]AggregatedProtocol` with Rank field populated
  - `GetLargestProtocol`: `*LargestProtocol` (pointer for nil handling)
- **Pattern:** Follow established breakdown patterns from Stories 3.3/3.4

### Testing Standards

- Follow project testing conventions in `docs/architecture/testing-strategy.md` (Go table-driven tests, arrange/act/assert, must cover success and edge cases) [Source: docs/architecture/testing-strategy.md]

### LargestProtocol Struct

```go
// LargestProtocol represents the top protocol by TVL.
type LargestProtocol struct {
    Name string  `json:"name"`
    Slug string  `json:"slug"`
    TVL  float64 `json:"tvl"`
    TVS  float64 `json:"tvs"`
}
```

### AggregatedProtocol Addition

```go
// Add Rank field to existing AggregatedProtocol struct:
type AggregatedProtocol struct {
    Name       string             `json:"name"`
    Slug       string             `json:"slug"`
    Category   string             `json:"category"`
    URL        string             `json:"url"`
    TVL        float64            `json:"tvl"`
    Chains     []string           `json:"chains"`
    TVS        float64            `json:"tvs"`
    TVSByChain map[string]float64 `json:"tvs_by_chain"`
    Rank       int                `json:"rank"`  // NEW: Protocol rank by TVL
}
```

### RankProtocols Implementation Pattern

```go
// RankProtocols sorts protocols by TVL descending and assigns rank starting from 1.
// Uses alphabetical name order as tiebreaker for equal TVL.
func RankProtocols(protocols []AggregatedProtocol) []AggregatedProtocol {
    if len(protocols) == 0 {
        return []AggregatedProtocol{}
    }

    // Create a copy to avoid mutating the input
    ranked := make([]AggregatedProtocol, len(protocols))
    copy(ranked, protocols)

    sort.Slice(ranked, func(i, j int) bool {
        if ranked[i].TVL != ranked[j].TVL {
            return ranked[i].TVL > ranked[j].TVL // Descending by TVL
        }
        return ranked[i].Name < ranked[j].Name // Alphabetical tiebreaker
    })

    for i := range ranked {
        ranked[i].Rank = i + 1 // Rank starts at 1
    }

    return ranked
}
```

### GetLargestProtocol Implementation Pattern

```go
// GetLargestProtocol returns the protocol with the highest TVL.
// Returns nil if protocols slice is empty.
func GetLargestProtocol(protocols []AggregatedProtocol) *LargestProtocol {
    if len(protocols) == 0 {
        return nil
    }

    // Find max TVL protocol (or use Rank 1 if already ranked)
    var largest *AggregatedProtocol
    for i := range protocols {
        if largest == nil || protocols[i].TVL > largest.TVL {
            largest = &protocols[i]
        }
    }

    return &LargestProtocol{
        Name: largest.Name,
        Slug: largest.Slug,
        TVL:  largest.TVL,
        TVS:  largest.TVS,
    }
}
```

### Testing Strategy

Follow table-driven test pattern established in `internal/aggregator/metrics_test.go`:

```go
func TestRankProtocols(t *testing.T) {
    tests := []struct {
        name           string
        protocols      []AggregatedProtocol
        wantCount      int
        wantFirstRank  int
        wantFirstName  string
        wantFirstTVL   float64
    }{
        {
            name:      "empty input",
            protocols: []AggregatedProtocol{},
            wantCount: 0,
        },
        {
            name: "sorted by TVL descending",
            protocols: []AggregatedProtocol{
                {Name: "Small", TVL: 100},
                {Name: "Large", TVL: 500},
                {Name: "Medium", TVL: 300},
            },
            wantCount:     3,
            wantFirstRank: 1,
            wantFirstName: "Large",
            wantFirstTVL:  500,
        },
        {
            name: "tiebreaker alphabetical",
            protocols: []AggregatedProtocol{
                {Name: "Zebra", TVL: 100},
                {Name: "Alpha", TVL: 100},
            },
            wantCount:     2,
            wantFirstRank: 1,
            wantFirstName: "Alpha",
        },
    }
    // ... test implementation
}
```

### Project Structure Notes

- Modify file: `internal/aggregator/models.go` - Add LargestProtocol struct, add Rank field to AggregatedProtocol
- Modify file: `internal/aggregator/metrics.go` - Add RankProtocols and GetLargestProtocol functions
- Modify file: `internal/aggregator/metrics_test.go` - Add ranking tests
- Aligns with fr-category-to-architecture-mapping.md: Aggregation & Metrics (FR15-FR24) -> `internal/aggregator`

### Learnings from Previous Story

**From Story 3-4-calculate-category-breakdown (Status: done)**

- **CategoryBreakdown Pattern Available:** Story 3.4 established solid patterns for aggregation functions with proper empty handling and sorting
- **Test Patterns:** Table-driven tests with mock AggregatedProtocol data in `internal/aggregator/metrics_test.go` - follow same structure
- **AggregatedProtocol Model:** Current model at `internal/aggregator/models.go:3-13` has Name, Slug, Category, URL, TVL, Chains, TVS, TVSByChain - need to add Rank field
- **Existing Functions:** `CalculateChainBreakdown` and `CalculateCategoryBreakdown` in `internal/aggregator/metrics.go` provide patterns for sort.Slice usage and empty input handling
- **Build/Lint Commands:** `go build ./...`, `go test ./internal/aggregator/...`, `make lint` all verified working
- **Review Outcome:** Story 3.4 approved with no action items - clean implementation to build upon

[Source: docs/sprint-artifacts/3-4-calculate-category-breakdown.md#Dev-Agent-Record]

### Key Implementation Considerations

1. **Rank vs Sort:** RankProtocols both sorts AND assigns rank - returns modified slice
2. **Copy Input:** Create a copy of the input slice to avoid mutating original data
3. **Pointer Return:** GetLargestProtocol returns pointer to handle nil case elegantly
4. **TVL vs TVS:** Ranking is by TVL (Total Value Locked), but LargestProtocol includes both TVL and TVS
5. **Tiebreaker Logic:** When TVL is equal, alphabetical by Name provides deterministic ordering

### References

- [Source: docs/epics/epic-3-data-processing-pipeline.md#story-35-rank-protocols-and-identify-largest] - Story definition and acceptance criteria
- [Source: docs/prd.md#fr18] - FR18: System ranks protocols by TVL in descending order
- [Source: docs/prd.md#fr23] - FR23: System identifies largest protocol by TVL
- [Source: docs/architecture/testing-strategy.md] - Testing standards for Go components
- [Source: internal/aggregator/models.go] - AggregatedProtocol struct to extend
- [Source: internal/aggregator/metrics.go] - Existing breakdown patterns to follow

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/3-5-rank-protocols-and-identify-largest.context.xml

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-3-data-processing-pipeline.md |
