# Story 3.4: Calculate Category Breakdown

Status: ready-for-dev

## Story

As a **developer**,
I want **to calculate TVS breakdown by protocol category**,
so that **I can show which DeFi sectors use Switchboard most**.

## Acceptance Criteria

Source: Epic 3.4 / PRD FR17, FR24

1. **Given** aggregated protocol data **When** `CalculateCategoryBreakdown(protocols []AggregatedProtocol)` is called **Then** a `CategoryBreakdown` slice is returned with each unique category represented, including `TVS` sum for that category, `Percentage` of total TVS, and `ProtocolCount` in that category

2. **Given** protocols in categories: Lending (3 protocols, $600M), CDP (2 protocols, $300M), Dexes (1 protocol, $100M) **When** calculating breakdown **Then** Lending percentage = 60%, count = 3 **And** CDP percentage = 30%, count = 2 **And** Dexes percentage = 10%, count = 1

3. **Given** category breakdown results **When** sorting **Then** categories are ordered by TVS descending (highest first)

4. **Given** a protocol with empty or missing category field **When** calculating breakdown **Then** the protocol is grouped under "Uncategorized"

5. **Given** zero total TVS (no protocols or all protocols have zero TVS) **When** calculating breakdown **Then** percentages are set to 0 (no division by zero panic)

6. **Given** all protocols **When** extracting categories **Then** unique categories list is correctly derived (FR24)

## Tasks / Subtasks

- [ ] Task 1: Define CategoryBreakdown struct (AC: 1)
  - [ ] 1.1: Add `CategoryBreakdown` struct to `internal/aggregator/models.go`
  - [ ] 1.2: Include fields: `Category` (string), `TVS` (float64), `Percentage` (float64), `ProtocolCount` (int)
  - [ ] 1.3: Add JSON struct tags for output serialization

- [ ] Task 2: Implement CalculateCategoryBreakdown function (AC: 1, 2, 3, 4, 5, 6)
  - [ ] 2.1: Add function signature `func CalculateCategoryBreakdown(protocols []AggregatedProtocol) []CategoryBreakdown` to `internal/aggregator/metrics.go`
  - [ ] 2.2: Iterate over protocols and aggregate TVS by category
  - [ ] 2.3: Handle empty/missing category as "Uncategorized"
  - [ ] 2.4: Track protocol count per category
  - [ ] 2.5: Calculate total TVS across all categories
  - [ ] 2.6: Calculate percentage for each category as `(categoryTVS / totalTVS) * 100`
  - [ ] 2.7: Handle zero total TVS gracefully (return percentages as 0)
  - [ ] 2.8: Sort results by TVS descending using `sort.Slice()`

- [ ] Task 3: Write unit tests (AC: 1, 2, 3, 4, 5, 6)
  - [ ] 3.1: Add tests to `internal/aggregator/metrics_test.go`
  - [ ] 3.2: Test: CategoryBreakdown returned with correct fields populated
  - [ ] 3.3: Test: TVS sums correctly across multiple protocols in same category
  - [ ] 3.4: Test: Percentages calculate correctly (verify 60/30/10 split example)
  - [ ] 3.5: Test: Results sorted by TVS descending
  - [ ] 3.6: Test: Zero total TVS returns empty slice or zero percentages (no panic)
  - [ ] 3.7: Test: Empty protocols slice returns empty result
  - [ ] 3.8: Test: Protocol count reflects number of protocols per category
  - [ ] 3.9: Test: Empty/missing category grouped as "Uncategorized"

- [ ] Task 4: Verification (AC: all)
  - [ ] 4.1: Run `go build ./...` and verify success
  - [ ] 4.2: Run `go test ./internal/aggregator/...` and verify all pass
  - [ ] 4.3: Run `make lint` and verify no errors

## Dev Notes

### Technical Guidance

- **Package Location:** Add to `internal/aggregator/` package (same as filter.go, extractor.go, metrics.go)
- **Input Type:** `[]AggregatedProtocol` - output from Story 3.2's ExtractProtocolData
- **Output Type:** `[]CategoryBreakdown` - sorted slice of category breakdown data
- **Pattern:** Follow the established `CalculateChainBreakdown` pattern from Story 3.3

### Testing Standards

- Follow project testing conventions in `docs/architecture/testing-strategy.md` (Go table-driven tests, arrange/act/assert, must cover success and edge cases) [Source: docs/architecture/testing-strategy.md]

### CategoryBreakdown Struct

```go
// CategoryBreakdown represents TVS metrics for a protocol category.
type CategoryBreakdown struct {
    Category      string  `json:"category"`
    TVS           float64 `json:"tvs"`
    Percentage    float64 `json:"percentage"`
    ProtocolCount int     `json:"protocol_count"`
}
```

### Implementation Pattern

```go
func CalculateCategoryBreakdown(protocols []AggregatedProtocol) []CategoryBreakdown {
    if len(protocols) == 0 {
        return []CategoryBreakdown{}
    }

    categoryData := make(map[string]struct {
        tvs           float64
        protocolCount int
    })

    var totalTVS float64
    for _, p := range protocols {
        category := p.Category
        if category == "" {
            category = "Uncategorized"
        }

        data := categoryData[category]
        data.tvs += p.TVS
        data.protocolCount++
        categoryData[category] = data
        totalTVS += p.TVS
    }

    if len(categoryData) == 0 {
        return []CategoryBreakdown{}
    }

    result := make([]CategoryBreakdown, 0, len(categoryData))
    for category, data := range categoryData {
        pct := 0.0
        if totalTVS > 0 {
            pct = (data.tvs / totalTVS) * 100
        }
        result = append(result, CategoryBreakdown{
            Category:      category,
            TVS:           data.tvs,
            Percentage:    pct,
            ProtocolCount: data.protocolCount,
        })
    }

    sort.Slice(result, func(i, j int) bool {
        return result[i].TVS > result[j].TVS
    })

    return result
}
```

### Testing Strategy

Follow table-driven test pattern established in `internal/aggregator/metrics_test.go`:

```go
func TestCalculateCategoryBreakdown(t *testing.T) {
    tests := []struct {
        name              string
        protocols         []AggregatedProtocol
        wantCategories    int
        wantTotalTVS      float64
        wantFirstCategory string
        wantFirstPct      float64
    }{
        // test cases...
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := CalculateCategoryBreakdown(tt.protocols)
            // assertions
        })
    }
}
```

### Project Structure Notes

- Modify file: `internal/aggregator/models.go` - Add CategoryBreakdown struct
- Modify file: `internal/aggregator/metrics.go` - Add CalculateCategoryBreakdown function
- Modify file: `internal/aggregator/metrics_test.go` - Add category breakdown tests
- Aligns with fr-category-to-architecture-mapping.md: Aggregation & Metrics (FR15-FR24) -> `internal/aggregator`

### Learnings from Previous Story

**From Story 3-3-calculate-total-tvs-and-chain-breakdown (Status: review)**

- **ChainBreakdown Pattern Available:** Use `ChainBreakdown` struct in `internal/aggregator/models.go` as template for `CategoryBreakdown` - same structure with `Category` instead of `Chain`
- **CalculateChainBreakdown Function:** The implementation pattern in `internal/aggregator/metrics.go:6-55` provides the exact blueprint - aggregate by key, calculate percentages, sort by TVS descending
- **Zero TVS Handling:** Story 3.3 properly handles zero total TVS by returning empty slice when no data, and setting percentage to 0 when dividing - follow same pattern
- **Test Patterns:** Table-driven tests with mock AggregatedProtocol data established in `internal/aggregator/metrics_test.go` - follow same pattern
- **Build/Lint Commands:** `go build ./...`, `go test ./internal/aggregator/...`, `make lint` all verified working
- **Review Outcome:** Story 3.3 approved with no action items - clean implementation pattern to follow

[Source: docs/sprint-artifacts/3-3-calculate-total-tvs-and-chain-breakdown.md#Dev-Agent-Record]

### Key Differences from Chain Breakdown

1. **Aggregation Key:** Use `p.Category` instead of iterating `p.TVSByChain`
2. **Empty Handling:** Handle empty category as "Uncategorized" (chains are never empty)
3. **TVS Source:** Use `p.TVS` (total protocol TVS) instead of chain-specific TVS
4. **Protocol Count:** Count protocols in each category, not protocols on each chain

### References

- [Source: docs/epics/epic-3-data-processing-pipeline.md#story-34-calculate-category-breakdown] - Story definition and acceptance criteria
- [Source: docs/prd.md#fr17] - FR17: System calculates TVS breakdown by protocol category with percentage of total
- [Source: docs/prd.md#fr24] - FR24: System extracts unique categories across all filtered protocols
- [Source: docs/architecture/testing-strategy.md] - Testing standards for Go components
- [Source: internal/aggregator/models.go] - AggregatedProtocol struct with Category field
- [Source: internal/aggregator/metrics.go] - CalculateChainBreakdown pattern to follow

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/3-4-calculate-category-breakdown.context.xml

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-3-data-processing-pipeline.md |
