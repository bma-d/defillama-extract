# Story 3.3: Calculate Total TVS and Chain Breakdown

Status: ready-for-dev

## Story

As a **developer**,
I want **to calculate total TVS and breakdown by chain**,
so that **I can show Switchboard's presence across different blockchains**.

## Acceptance Criteria

Source: Epic 3.3 / PRD FR15, FR16

1. **Given** aggregated protocol data **When** `CalculateChainBreakdown(protocols []AggregatedProtocol)` is called **Then** a `ChainBreakdown` slice is returned with each unique chain represented, including `TVS` sum for that chain, `Percentage` of total TVS, and `ProtocolCount` on that chain

2. **Given** protocols with TVS: Solana=$500M, Sui=$300M, Aptos=$200M **When** calculating breakdown **Then** total TVS = $1B **And** Solana percentage = 50% **And** Sui percentage = 30% **And** Aptos percentage = 20%

3. **Given** chain breakdown results **When** sorting **Then** chains are ordered by TVS descending (highest first)

4. **Given** zero total TVS (no protocols or all protocols have zero TVS) **When** calculating breakdown **Then** percentages are set to 0 (no division by zero panic)

5. **Given** valid inputs **When** `CalculateChainBreakdown` completes **Then** total TVS across all returned chain breakdowns equals the sum of all protocol TVS values

## Tasks / Subtasks

- [ ] Task 1: Define ChainBreakdown struct (AC: 1)
  - [ ] 1.1: Add `ChainBreakdown` struct to `internal/aggregator/models.go`
  - [ ] 1.2: Include fields: `Chain` (string), `TVS` (float64), `Percentage` (float64), `ProtocolCount` (int)
  - [ ] 1.3: Add JSON struct tags for output serialization

- [ ] Task 2: Implement CalculateChainBreakdown function (AC: 1, 2, 3, 4, 5)
  - [ ] 2.1: Create `internal/aggregator/metrics.go` with function signature `func CalculateChainBreakdown(protocols []AggregatedProtocol) []ChainBreakdown`
  - [ ] 2.2: Iterate over protocols and aggregate TVS by chain from `TVSByChain` maps
  - [ ] 2.3: Track protocol count per chain (count protocols that have non-zero TVS on that chain)
  - [ ] 2.4: Calculate total TVS across all chains
  - [ ] 2.5: Calculate percentage for each chain as `(chainTVS / totalTVS) * 100`
  - [ ] 2.6: Handle zero total TVS gracefully (return percentages as 0)
  - [ ] 2.7: Sort results by TVS descending using `sort.Slice()`

- [ ] Task 3: Write unit tests (AC: 1, 2, 3, 4, 5)
  - [ ] 3.1: Create `internal/aggregator/metrics_test.go`
  - [ ] 3.2: Test: ChainBreakdown returned with correct fields populated
  - [ ] 3.3: Test: TVS sums correctly across multiple protocols on same chain
  - [ ] 3.4: Test: Percentages calculate correctly (verify 50/30/20 split example)
  - [ ] 3.5: Test: Results sorted by TVS descending
  - [ ] 3.6: Test: Zero total TVS returns empty slice or zero percentages (no panic)
  - [ ] 3.7: Test: Empty protocols slice returns empty result
  - [ ] 3.8: Test: Protocol count reflects number of protocols per chain

- [ ] Task 4: Verification (AC: all)
  - [ ] 4.1: Run `go build ./...` and verify success
  - [ ] 4.2: Run `go test ./internal/aggregator/...` and verify all pass
  - [ ] 4.3: Run `make lint` and verify no errors

## Dev Notes

### Technical Guidance

- **Package Location:** Add to `internal/aggregator/` package (same as filter.go, extractor.go)
- **Input Type:** `[]AggregatedProtocol` - output from Story 3.2's ExtractProtocolData
- **Output Type:** `[]ChainBreakdown` - sorted slice of chain breakdown data

### Testing Standards

- Follow project testing conventions in `docs/architecture/testing-strategy.md` (Go table-driven tests, arrange/act/assert, must cover success and edge cases) [Source: docs/architecture/testing-strategy.md]

### ChainBreakdown Struct

```go
// ChainBreakdown represents TVS metrics for a single blockchain.
type ChainBreakdown struct {
    Chain         string  `json:"chain"`
    TVS           float64 `json:"tvs"`
    Percentage    float64 `json:"percentage"`
    ProtocolCount int     `json:"protocol_count"`
}
```

### Implementation Pattern

```go
func CalculateChainBreakdown(protocols []AggregatedProtocol) []ChainBreakdown {
    if len(protocols) == 0 {
        return []ChainBreakdown{}
    }

    // Aggregate TVS and protocol counts by chain
    chainData := make(map[string]struct {
        tvs           float64
        protocolCount int
    })

    var totalTVS float64
    for _, p := range protocols {
        for chain, tvs := range p.TVSByChain {
            data := chainData[chain]
            data.tvs += tvs
            data.protocolCount++
            chainData[chain] = data
            totalTVS += tvs
        }
    }

    // Build result slice
    result := make([]ChainBreakdown, 0, len(chainData))
    for chain, data := range chainData {
        pct := 0.0
        if totalTVS > 0 {
            pct = (data.tvs / totalTVS) * 100
        }
        result = append(result, ChainBreakdown{
            Chain:         chain,
            TVS:           data.tvs,
            Percentage:    pct,
            ProtocolCount: data.protocolCount,
        })
    }

    // Sort by TVS descending
    sort.Slice(result, func(i, j int) bool {
        return result[i].TVS > result[j].TVS
    })

    return result
}
```

### Testing Strategy

Follow table-driven test pattern established in `internal/aggregator/filter_test.go` and `internal/aggregator/extractor_test.go`:

```go
func TestCalculateChainBreakdown(t *testing.T) {
    tests := []struct {
        name              string
        protocols         []AggregatedProtocol
        wantChains        int
        wantTotalTVS      float64
        wantFirstChain    string
        wantFirstPct      float64
    }{
        // test cases...
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := CalculateChainBreakdown(tt.protocols)
            // assertions
        })
    }
}
```

### Project Structure Notes

- New file: `internal/aggregator/metrics.go` - CalculateChainBreakdown function
- New file: `internal/aggregator/metrics_test.go` - tests
- Add ChainBreakdown struct to: `internal/aggregator/models.go`
- Aligns with fr-category-to-architecture-mapping.md: Aggregation & Metrics (FR15-FR24) -> `internal/aggregator`

### Learnings from Previous Story

**From Story 3-2-extract-protocol-metadata-and-tvs-data (Status: done)**

- **AggregatedProtocol Available:** Use `AggregatedProtocol` struct from `internal/aggregator/models.go` as input - already has `TVSByChain` map populated
- **ExtractProtocolData Function:** The input for this story comes from `ExtractProtocolData(protocols, oracleResp, oracleName)` which returns `[]AggregatedProtocol`
- **TVSByChain Structure:** Each AggregatedProtocol has `TVSByChain map[string]float64` - iterate this for chain breakdown
- **Test Patterns:** Table-driven tests with mock data established in `internal/aggregator/extractor_test.go` - follow same pattern
- **Build/Lint Commands:** `go build ./...`, `go test ./internal/aggregator/...`, `make lint` all verified working
- **Review Outcome:** Story 3.2 approved with no action items - clean implementation

[Source: docs/sprint-artifacts/3-2-extract-protocol-metadata-and-tvs-data.md]

### References

- [Source: docs/epics/epic-3-data-processing-pipeline.md#story-33-calculate-total-tvs-and-chain-breakdown] - Story definition and acceptance criteria
- [Source: docs/prd.md#fr15] - FR15: System calculates total TVS across all protocols using the oracle
- [Source: docs/prd.md#fr16] - FR16: System calculates TVS breakdown by chain with percentage of total
- [Source: docs/architecture/testing-strategy.md] - Testing standards for Go components
- [Source: internal/aggregator/models.go] - AggregatedProtocol struct with TVSByChain field
- [Source: internal/aggregator/extractor.go] - ExtractProtocolData function that provides input

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/3-3-calculate-total-tvs-and-chain-breakdown.context.xml

### Agent Model Used

- BMad SM Agent (Bob) using GPT-5 class model

### Debug Log References

- None yet (story still drafted)

### Completion Notes List

- 2025-11-30: Story draft validated; Dev Agent Record populated; epic citation anchor corrected.

### File List

- docs/sprint-artifacts/3-3-calculate-total-tvs-and-chain-breakdown.md

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-3-data-processing-pipeline.md |
