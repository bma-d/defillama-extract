# Story 3.2: Extract Protocol Metadata and TVS Data

Status: ready-for-dev

## Story

As a **developer**,
I want **to extract relevant metadata and TVS for each filtered protocol**,
so that **I have the data needed for aggregation and output**.

## Acceptance Criteria

Source: Epic 3.2 / PRD FR11-FR14

1. **Given** filtered Switchboard protocols and oracle API response **When** `ExtractProtocolData(protocols []api.Protocol, oracleResp *api.OracleAPIResponse, oracleName string)` is called **Then** for each protocol an `AggregatedProtocol` struct is created with: `Name`, `Slug`, `Category`, `URL` from protocol metadata, `TVL` from protocol metadata, `Chains` list from protocol metadata, and `TVS` calculated from oracle response data

2. **Given** oracle response with `OraclesTVS["Switchboard"]["Solana"] = 1000000` **When** extracting TVS for a protocol on Solana **Then** the protocol's TVS includes the Solana contribution

3. **Given** a protocol operating on multiple chains **When** extracting TVS **Then** `TVSByChain` map contains TVS for each chain where the protocol operates AND the oracle has TVS data

4. **Given** oracle response chart data **When** extracting timestamp **Then** the latest timestamp from chart data is extracted (FR14) as Unix timestamp

5. **Given** a protocol with empty or missing `Chains` field **When** extracting TVS **Then** the protocol is included with `TVS = 0` and empty `TVSByChain` map (no crash)

6. **Given** valid inputs **When** `ExtractProtocolData` completes **Then** a slice of `AggregatedProtocol` structs and extracted timestamp are returned

## Tasks / Subtasks

- [ ] Task 1: Define AggregatedProtocol struct (AC: 1, 3)
  - [ ] 1.1: Create `internal/aggregator/models.go` with `AggregatedProtocol` struct
  - [ ] 1.2: Include fields: `Name`, `Slug`, `Category`, `URL`, `TVL`, `Chains`, `TVS` (total), `TVSByChain` (map[string]float64)
  - [ ] 1.3: Add JSON struct tags for output serialization

- [ ] Task 2: Implement ExtractProtocolData function (AC: 1, 2, 3, 5)
  - [ ] 2.1: Create `internal/aggregator/extractor.go` with function signature
  - [ ] 2.2: Iterate over filtered protocols and copy metadata fields
  - [ ] 2.3: For each protocol, cross-reference `Chains` with `oracleResp.OraclesTVS[oracleName]`
  - [ ] 2.4: Sum TVS across all chains for protocol's total TVS
  - [ ] 2.5: Populate `TVSByChain` map with per-chain TVS values
  - [ ] 2.6: Handle empty/missing chains gracefully (return zero TVS)

- [ ] Task 3: Implement timestamp extraction (AC: 4)
  - [ ] 3.1: Create `ExtractLatestTimestamp(oracleResp *api.OracleAPIResponse)` helper function
  - [ ] 3.2: Parse chart data keys as Unix timestamps (strings to int64)
  - [ ] 3.3: Return the maximum timestamp found

- [ ] Task 4: Write unit tests (AC: 1, 2, 3, 4, 5, 6)
  - [ ] 4.1: Create `internal/aggregator/extractor_test.go`
  - [ ] 4.2: Test: ExtractProtocolData populates all metadata fields correctly
  - [ ] 4.3: Test: TVS calculated correctly from OraclesTVS by matching chain
  - [ ] 4.4: Test: TVSByChain map populated for multi-chain protocol
  - [ ] 4.5: Test: Protocol with missing chains returns zero TVS (no panic)
  - [ ] 4.6: Test: ExtractLatestTimestamp returns max timestamp from chart keys
  - [ ] 4.7: Test: Empty oracle response returns zero timestamp

- [ ] Task 5: Verification (AC: all)
  - [ ] 5.1: Run `go build ./...` and verify success
  - [ ] 5.2: Run `go test ./internal/aggregator/...` and verify all pass
  - [ ] 5.3: Run `make lint` and verify no errors

## Dev Notes

### Technical Guidance

- **Package Location:** Add to `internal/aggregator/` package (same as FilterByOracle)
- **Input Types:**
  - `[]api.Protocol` - filtered protocols from Story 3.1's FilterByOracle
  - `*api.OracleAPIResponse` - from API client's FetchOracles
  - `string` - oracle name ("Switchboard")
- **Output Types:**
  - `[]AggregatedProtocol` - enriched protocol data with TVS
  - `int64` - latest timestamp from chart data

### Testing Standards

- Follow project testing conventions in `docs/architecture/testing-strategy.md` (Go table-driven tests, arrange/act/assert, must cover success and edge cases; cite this doc in Dev Notes) [Source: docs/architecture/testing-strategy.md]

### OraclesTVS Data Structure

The `OracleAPIResponse.OraclesTVS` is a nested map:
```go
OraclesTVS map[string]map[string]map[string]float64
// Level 1: oracle name -> map
// Level 2: timestamp (string) -> map
// Level 3: chain name -> TVS value

// Example access:
oracleResp.OraclesTVS["Switchboard"]["1732924800"]["Solana"] // returns float64 TVS
```

To get current TVS by chain, find the latest timestamp key first.

### Chart Data Structure

The `OracleAPIResponse.Chart` contains historical data with Unix timestamps as string keys:
```go
Chart map[string]map[string]map[string]float64
// Keys are Unix timestamps as strings, e.g., "1732924800"
```

Parse keys to int64, find maximum to get latest timestamp.

### AggregatedProtocol Struct

```go
type AggregatedProtocol struct {
    Name       string             `json:"name"`
    Slug       string             `json:"slug"`
    Category   string             `json:"category"`
    URL        string             `json:"url"`
    TVL        float64            `json:"tvl"`
    Chains     []string           `json:"chains"`
    TVS        float64            `json:"tvs"`
    TVSByChain map[string]float64 `json:"tvs_by_chain"`
}
```

### Implementation Pattern

```go
func ExtractProtocolData(protocols []api.Protocol, oracleResp *api.OracleAPIResponse, oracleName string) ([]AggregatedProtocol, int64) {
    timestamp := ExtractLatestTimestamp(oracleResp)

    // Get TVS data for this oracle at latest timestamp
    timestampStr := strconv.FormatInt(timestamp, 10)
    chainTVS := oracleResp.OraclesTVS[oracleName][timestampStr]

    result := make([]AggregatedProtocol, 0, len(protocols))
    for _, p := range protocols {
        agg := AggregatedProtocol{
            Name:       p.Name,
            Slug:       p.Slug,
            Category:   p.Category,
            URL:        p.URL,
            TVL:        p.TVL,
            Chains:     p.Chains,
            TVSByChain: make(map[string]float64),
        }

        // Calculate TVS from chains the protocol operates on
        for _, chain := range p.Chains {
            if tvs, ok := chainTVS[chain]; ok {
                agg.TVSByChain[chain] = tvs
                agg.TVS += tvs
            }
        }

        result = append(result, agg)
    }

    return result, timestamp
}
```

### Testing Strategy

Follow table-driven test pattern established in `internal/aggregator/filter_test.go`:

```go
func TestExtractProtocolData(t *testing.T) {
    tests := []struct {
        name           string
        protocols      []api.Protocol
        oracleResp     *api.OracleAPIResponse
        oracleName     string
        wantCount      int
        wantTotalTVS   float64
        wantTimestamp  int64
    }{
        // test cases...
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, ts := ExtractProtocolData(tt.protocols, tt.oracleResp, tt.oracleName)
            // assertions
        })
    }
}
```

### Project Structure Notes

- New file: `internal/aggregator/models.go` - AggregatedProtocol struct
- New file: `internal/aggregator/extractor.go` - ExtractProtocolData function
- New file: `internal/aggregator/extractor_test.go` - tests
- Aligns with fr-category-to-architecture-mapping.md: Data Filtering (FR9-FR14) → `internal/aggregator`

### Learnings from Previous Story

**From Story 3-1-implement-protocol-filtering-by-oracle-name (Status: done)**

- **FilterByOracle Available:** Use `FilterByOracle(protocols, oracleName)` from `internal/aggregator/filter.go` to get filtered protocols as input
- **api.Protocol Struct:** Protocol struct at `internal/api/responses.go` has all fields needed (Name, Slug, Category, URL, TVL, Chains, Oracles, Oracle)
- **Test Patterns:** Table-driven tests with mock data established in `internal/aggregator/filter_test.go` - follow same pattern
- **Build/Lint Commands:** `go build ./...`, `go test ./internal/aggregator/...`, `make lint` all verified working
- **Review Outcome:** Story 3.1 approved with no action items - clean implementation

[Source: docs/sprint-artifacts/3-1-implement-protocol-filtering-by-oracle-name.md#Dev-Agent-Record]

### References

- [Source: docs/epics/epic-3-data-processing-pipeline.md#story-32] - Story definition and acceptance criteria
- [Source: docs/prd.md#fr11] - FR11: System extracts TVS (Total Value Secured) data per protocol per chain
- [Source: docs/prd.md#fr12] - FR12: System extracts protocol metadata (name, slug, category, TVL, chains, URL)
- [Source: docs/prd.md#fr14] - FR14: System extracts timestamp of latest data point from chart data
- [Source: docs/architecture/data-architecture.md#api-response-models] - API response models and output models
- [Source: docs/architecture/fr-category-to-architecture-mapping.md#fr9-fr14-data-filtering] - Data Filtering (FR9-FR14) maps to `internal/aggregator` package
- [Source: docs/architecture/testing-strategy.md#go-testing-conventions] - Testing standards for Go components
- [Source: internal/api/responses.go] - OracleAPIResponse and Protocol types

## Dev Agent Record

### Context Reference
- docs/sprint-artifacts/3-2-extract-protocol-metadata-and-tvs-data.context.xml

### Agent Model Used
BMad SM Agent (Bob) using GPT-5 class model

### Debug Log References
None captured for this drafting pass
### Completion Notes List
2025-11-30: Draft reviewed and updated Dev Notes/testing reference; added anchored citations; Dev Agent Record populated; pending dev handoff
### File List
- docs/sprint-artifacts/3-2-extract-protocol-metadata-and-tvs-data.md
## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-3-data-processing-pipeline.md |
