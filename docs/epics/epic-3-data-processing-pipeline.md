# Epic 3: Data Processing Pipeline

**Goal:** Implement filtering, aggregation, and metrics calculation to transform raw API data into meaningful Switchboard oracle metrics.

**User Value:** After this epic, the system correctly identifies all Switchboard protocols and calculates TVS breakdowns, rankings, and change metrics - the core value proposition of surfacing the "truth gap."

**FRs Covered:** FR9, FR10, FR11, FR12, FR13, FR14, FR15, FR16, FR17, FR18, FR19, FR20, FR21, FR22, FR23, FR24

> **MANDATORY:** A Tech Spec MUST be drafted before creating stories for this epic. Skipping the tech spec for Epic 3 was identified as a mistake in the Epic 2+3 retrospective (2025-11-30). The tech spec provides critical traceability for AC validation and review.

> **MANDATORY:** Each story MUST include a **Smoke Test Guide** in Dev Notes (or explicitly mark "Smoke test: N/A" for internal-only functions). Build/test/lint alone do not verify runtime behavior. This requirement was established in the Epic 2+3 retrospective (2025-11-30).

---

## Story 3.1: Implement Protocol Filtering by Oracle Name

As a **developer**,
I want **to filter protocols that use Switchboard as their oracle**,
So that **only relevant protocols are included in aggregations**.

**Acceptance Criteria:**

**Given** a list of protocols from the API
**When** `FilterByOracle(protocols []Protocol, oracleName string)` is called with "Switchboard"
**Then** only protocols where:
  - `oracles` array contains "Switchboard" (exact match, case-sensitive), OR
  - `oracle` field equals "Switchboard" (legacy field check)
are returned

**Given** a protocol with `oracles: ["Chainlink", "Switchboard"]`
**When** filtering for "Switchboard"
**Then** the protocol IS included (multi-oracle protocol)

**Given** a protocol with `oracle: "Switchboard"` but empty `oracles` array
**When** filtering for "Switchboard"
**Then** the protocol IS included (legacy field fallback)

**Given** a protocol with `oracles: ["Chainlink"]` and `oracle: ""`
**When** filtering for "Switchboard"
**Then** the protocol is NOT included

**Given** ~500 protocols from the API
**When** filtering for "Switchboard"
**Then** approximately 21 protocols are returned (expected count per PRD)

**Prerequisites:** Story 2.3 (protocol fetcher)

**Technical Notes:**
- Package: `internal/aggregator/filter.go`
- Function: `FilterByOracle(protocols []models.Protocol, oracleName string) []models.Protocol`
- Check both `Oracles` slice and `Oracle` string field
- Use case-sensitive exact match
- Reference: 7-custom-aggregation-logic-go-implementation.md section 7.2

---

## Story 3.2: Extract Protocol Metadata and TVS Data

As a **developer**,
I want **to extract relevant metadata and TVS for each filtered protocol**,
So that **I have the data needed for aggregation and output**.

**Acceptance Criteria:**

**Given** filtered Switchboard protocols and oracle API response
**When** `ExtractProtocolData(protocols, oracleResp, oracleName)` is called
**Then** for each protocol, an `AggregatedProtocol` struct is created with:
  - `Name`, `Slug`, `Category`, `URL` from protocol metadata
  - `TVL` from protocol metadata
  - `Chains` list from protocol metadata
  - `TVS` calculated from oracle response data

**Given** oracle response with `OraclesTVS["Switchboard"]["<protocol>"]["Solana"] = 1000000` (timestamp key only as legacy fallback)
**When** extracting TVS for a protocol on Solana
**Then** the protocol's TVS includes the Solana contribution

**Given** a protocol operating on multiple chains
**When** extracting TVS
**Then** `TVSByChain` map contains TVS for each chain

**Given** oracle response chart data
**When** extracting timestamp
**Then** the latest timestamp from chart data is extracted (FR14)

**Prerequisites:** Story 3.1, Story 2.2

**Technical Notes:**
- Package: `internal/aggregator/aggregator.go`
- Create `AggregatedProtocol` struct in `internal/models/protocol.go`
- Cross-reference protocol chains with `OraclesTVS` data using protocol-keyed map; fall back to timestamp-keyed map when protocol entries are missing
- Extract timestamp from chart data keys (Unix timestamps as strings)
- Reference: data-architecture.md output models

---

## Story 3.3: Calculate Total TVS and Chain Breakdown

As a **developer**,
I want **to calculate total TVS and breakdown by chain**,
So that **I can show Switchboard's presence across different blockchains**.

**Acceptance Criteria:**

**Given** aggregated protocol data
**When** `CalculateChainBreakdown(protocols []AggregatedProtocol)` is called
**Then** a `ChainBreakdown` slice is returned with:
  - Each unique chain represented
  - `TVS` sum for that chain
  - `Percentage` of total TVS
  - `ProtocolCount` on that chain

**Given** protocols with TVS: Solana=$500M, Sui=$300M, Aptos=$200M
**When** calculating breakdown
**Then** total TVS = $1B
**And** Solana percentage = 50%
**And** Sui percentage = 30%
**And** Aptos percentage = 20%

**Given** chain breakdown results
**When** sorting
**Then** chains are ordered by TVS descending (highest first)

**Prerequisites:** Story 3.2

**Technical Notes:**
- Package: `internal/aggregator/metrics.go`
- `ChainBreakdown` struct: `Chain`, `TVS`, `Percentage`, `ProtocolCount`
- Use `float64` for TVS values (can be large numbers)
- Calculate percentage as `(chainTVS / totalTVS) * 100`
- Reference: FR15, FR16

---

## Story 3.4: Calculate Category Breakdown

As a **developer**,
I want **to calculate TVS breakdown by protocol category**,
So that **I can show which DeFi sectors use Switchboard most**.

**Acceptance Criteria:**

**Given** aggregated protocol data
**When** `CalculateCategoryBreakdown(protocols []AggregatedProtocol)` is called
**Then** a `CategoryBreakdown` slice is returned with:
  - Each unique category represented
  - `TVS` sum for that category
  - `Percentage` of total TVS
  - `ProtocolCount` in that category

**Given** protocols in categories: Lending (3 protocols, $600M), CDP (2 protocols, $300M), Dexes (1 protocol, $100M)
**When** calculating breakdown
**Then** Lending percentage = 60%, count = 3
**And** CDP percentage = 30%, count = 2
**And** Dexes percentage = 10%, count = 1

**Given** category breakdown results
**When** sorting
**Then** categories are ordered by TVS descending

**Given** all protocols
**When** extracting categories
**Then** unique categories list is returned (FR24)

**Prerequisites:** Story 3.2

**Technical Notes:**
- Package: `internal/aggregator/metrics.go`
- `CategoryBreakdown` struct: `Category`, `TVS`, `Percentage`, `ProtocolCount`
- Handle empty/missing category as "Uncategorized"
- Reference: FR17, FR24

---

## Story 3.5: Rank Protocols and Identify Largest

As a **developer**,
I want **protocols ranked by TVL and the largest protocol identified**,
So that **I can show protocol importance and highlight top contributors**.

**Acceptance Criteria:**

**Given** aggregated protocol data
**When** `RankProtocols(protocols []AggregatedProtocol)` is called
**Then** protocols are sorted by TVL descending
**And** each protocol is assigned a `Rank` field (1, 2, 3...)

**Given** ranked protocols
**When** identifying largest protocol
**Then** protocol with rank 1 is returned
**And** `LargestProtocol` struct contains: Name, Slug, TVL, TVS

**Given** two protocols with identical TVL
**When** ranking
**Then** alphabetical order by name is used as tiebreaker

**Prerequisites:** Story 3.2

**Technical Notes:**
- Package: `internal/aggregator/metrics.go`
- Use `sort.Slice()` with custom comparison
- Rank starts at 1 (not 0)
- Reference: FR18, FR23

---

## Story 3.6: Calculate Historical Change Metrics

As a **developer**,
I want **24h, 7d, and 30d TVS change percentages calculated**,
So that **I can show growth trends over time**.

**Acceptance Criteria:**

**Given** current TVS and historical snapshots
**When** `CalculateChangeMetrics(currentTVS float64, history []Snapshot)` is called
**Then** a `ChangeMetrics` struct is returned with:
  - `Change24h`: percentage change from 24 hours ago
  - `Change7d`: percentage change from 7 days ago
  - `Change30d`: percentage change from 30 days ago

**Given** current TVS = $1.1B and TVS 24h ago = $1.0B
**When** calculating 24h change
**Then** `Change24h` = 10.0 (representing 10% increase)

**Given** current TVS = $900M and TVS 7d ago = $1.0B
**When** calculating 7d change
**Then** `Change7d` = -10.0 (representing 10% decrease)

**Given** no historical data available for a time period
**When** calculating that period's change
**Then** the change value is `nil` or 0 with a flag indicating "no data"

**Given** history with protocol counts
**When** calculating growth
**Then** `ProtocolCountChange7d` and `ProtocolCountChange30d` are calculated (FR22)

**Prerequisites:** Story 3.3 (needs TVS calculation)

**Technical Notes:**
- Package: `internal/aggregator/metrics.go`
- Find snapshot closest to target time (24h, 7d, 30d ago)
- Use pointer types `*float64` for optional values
- Formula: `((current - previous) / previous) * 100`
- Handle division by zero (previous = 0)
- Reference: FR19, FR20, FR21, FR22

---

## Story 3.7: Build Complete Aggregation Pipeline

As a **developer**,
I want **a single function that orchestrates all data processing**,
So that **I have a clean interface for the extraction pipeline**.

**Acceptance Criteria:**

**Given** raw API responses (oracle and protocols)
**When** `Aggregate(ctx, oracleResp, protocols, history, oracleName)` is called
**Then** a complete `AggregationResult` is returned containing:
  - `TotalTVS`: sum across all protocols
  - `TotalProtocols`: count of filtered protocols
  - `ActiveChains`: list of chains with Switchboard presence
  - `Categories`: unique category list
  - `ChainBreakdown`: TVS by chain
  - `CategoryBreakdown`: TVS by category
  - `Protocols`: ranked protocol list
  - `LargestProtocol`: top protocol by TVL
  - `ChangeMetrics`: 24h/7d/30d changes
  - `Timestamp`: latest data timestamp

**Given** valid API data
**When** aggregation completes
**Then** all FRs 9-24 are satisfied by the result

**Prerequisites:** Stories 3.1-3.6

**Technical Notes:**
- Package: `internal/aggregator/aggregator.go`
- `Aggregator` struct with `NewAggregator(cfg)` constructor
- Main method: `func (a *Aggregator) Aggregate(...) (*AggregationResult, error)`
- Orchestrates: filter → extract → chain breakdown → category breakdown → rank → metrics
- Reference: fr-category-to-architecture-mapping.md

---
