# Story 6.1: Per-Protocol TVS Breakdown

Status: ready-for-dev

## Story

As a **dashboard consumer**,
I want **each protocol to have its TVS value and per-chain breakdown populated**,
So that **I can display accurate TVS attribution per protocol and verify data completeness**.

## Acceptance Criteria

Source: [Source: docs/epics/epic-6-maintenance.md#issue-m-001---per-protocol-tvs-breakdown-missing]; [Source: docs/sprint-artifacts/tech-spec-epic-6.md#M-001-Per-Protocol-TVS-Breakdown]; [Source: docs/prd.md#Success-Criteria]

**AC1: Protocols Have Non-Zero TVS (where data exists)**
**Given** the `/oracles` API response contains `oraclesTVS` data for a protocol
**When** extraction runs
**Then** that protocol's `tvs` field is populated with the sum of its chain TVS values
**And** protocols without upstream TVS data retain `tvs: 0` (graceful degradation)

**AC2: Per-Chain TVS Breakdown Populated**
**Given** the `/oracles` API response contains `oraclesTVS[oracle][protocol][chain]` mapping
**When** extraction runs
**Then** each protocol's `tvs_by_chain` map is populated with per-chain values
**And** the structure is `{"Solana": 1234567.89, "Sui": 234567.89, ...}`

**AC3: TVS Sum Validation**
**Given** all protocols have been processed
**When** output is generated
**Then** the sum of all `protocols[].tvs` is within 5% of `summary.total_value_secured`
**And** any discrepancy beyond 5% is logged as WARNING

**AC4: Missing Data Logging**
**Given** a protocol exists in the oracle's protocol list but has no entry in `oraclesTVS`
**When** extraction runs
**Then** a WARNING is logged with format: `protocol_tvs_unavailable protocol=<slug> reason="not found in oraclesTVS"`
**And** extraction continues without failure

**AC5: Extraction Summary Logging**
**Given** extraction completes
**When** summary is logged
**Then** log includes: `extraction_complete protocols_with_tvs=<N> protocols_without_tvs=<M>`

## Tasks / Subtasks

- [ ] Task 1: Investigate API Response Structure (AC: 1, 2)
  - [ ] 1.1: Fetch raw `/oracles` API response and examine `oraclesTVS` structure
  - [ ] 1.2: Verify key format (protocol slug vs ID) in `oraclesTVS` mapping
  - [ ] 1.3: Cross-reference protocol slugs between `/oracles` and `/lite/protocols2`
  - [ ] 1.4: Document findings in this story's Dev Notes section
  - [ ] 1.5: Update test fixture `testdata/oracle_response.json` with realistic `oraclesTVS` data

- [ ] Task 2: Implement Per-Protocol TVS Extraction (AC: 1, 2)
  - [ ] 2.1: Create helper function `ExtractProtocolTVS(oraclesTVS, oracleName, protocolSlug)` in `internal/aggregator/`
  - [ ] 2.2: Return `(totalTVS float64, tvsByChain map[string]float64, found bool)`
  - [ ] 2.3: Handle case where protocol not found in `oraclesTVS` (return 0, empty map, false)
  - [ ] 2.4: Integrate TVS extraction into existing protocol aggregation loop

- [ ] Task 3: Populate Protocol TVS Fields (AC: 1, 2)
  - [ ] 3.1: Modify protocol aggregation to call TVS extraction for each protocol
  - [ ] 3.2: Set `AggregatedProtocol.TVS` = sum of chain values
  - [ ] 3.3: Set `AggregatedProtocol.TVSByChain` = per-chain breakdown map
  - [ ] 3.4: Ensure existing fields (TVL, rank, category) remain unchanged

- [ ] Task 4: Add Warning Logging (AC: 4, 5)
  - [ ] 4.1: Log WARNING for each protocol without TVS data (include slug)
  - [ ] 4.2: Track count of protocols with/without TVS during extraction
  - [ ] 4.3: Log summary at extraction completion: `protocols_with_tvs=N protocols_without_tvs=M`

- [ ] Task 5: Add TVS Sum Validation (AC: 3)
  - [ ] 5.1: After all protocols processed, calculate sum of `protocols[].tvs`
  - [ ] 5.2: Compare to `summary.total_value_secured`
  - [ ] 5.3: If discrepancy > 5%, log WARNING with both values and percentage difference
  - [ ] 5.4: Document expected discrepancy sources (rounding, timing, upstream gaps)

- [ ] Task 6: Write Unit Tests (AC: all)
  - [ ] 6.1: Test TVS extraction with mock `oraclesTVS` containing multiple protocols
  - [ ] 6.2: Test TVS extraction when protocol missing from `oraclesTVS`
  - [ ] 6.3: Test sum validation within and outside 5% tolerance
  - [ ] 6.4: Test warning log generation for missing protocols

- [ ] Task 7: Integration Testing (AC: all)
  - [ ] 7.1: Run extraction with `--once` against live API
  - [ ] 7.2: Verify previously zero-TVS protocols now have values (where data exists)
  - [ ] 7.3: Verify `tvs_by_chain` populated for protocols with TVS
  - [ ] 7.4: Check logs for any `protocol_tvs_unavailable` warnings
  - [ ] 7.5: Verify sum validation output

- [ ] Task 8: Verification (AC: all)
  - [ ] 8.1: Run `go build ./...` and verify success
  - [ ] 8.2: Run `go test ./...` and verify all pass
  - [ ] 8.3: Run `make lint` and verify no errors
  - [ ] 8.4: Compare output before/after to verify no schema breaking changes

## Dev Notes

### Technical Guidance

- **Files to Modify:**
  - MODIFY: `internal/aggregator/aggregator.go` - Add TVS extraction to protocol aggregation loop
  - MODIFY: `internal/aggregator/filter.go` - May need to pass `oraclesTVS` data through
  - MODIFY: `internal/aggregator/metrics.go` - Add sum validation logic
  - MODIFY: `testdata/oracle_response.json` - Add realistic `oraclesTVS` fixture data

- **Files to Create (if needed):**
  - CREATE: `internal/aggregator/tvs.go` - TVS extraction helper (optional, could be in aggregator.go)
  - CREATE: `internal/aggregator/tvs_test.go` - TVS extraction unit tests

### Architecture Patterns and Constraints

- Follow existing aggregator-layer pattern for data extraction (see `chart.go` from Story 5.4) [Source: docs/architecture/implementation-patterns.md]
- Preserve atomic writes via existing writer pattern [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-002-atomic-file-writes]
- Use `log/slog` for structured logging with fields [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-004-structured-logging-with-slog]
- No new external dependencies [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-005]
- All output schema changes must be backward compatible (only populating existing empty fields)

### API Response Structure (from Tech Spec)

The `/oracles` endpoint returns `oraclesTVS` with this expected structure:
```json
{
  "oraclesTVS": {
    "Switchboard": {
      "protocol-slug-1": { "Solana": 1234567.89, "Sui": 234567.89 },
      "protocol-slug-2": { "Solana": 345678.90 }
    }
  }
}
```

**Investigation Required:** Verify actual API response matches this structure. Key may be protocol slug or protocol ID.

### Data Model Reference

Current `AggregatedProtocol` struct (from `internal/aggregator/models.go`):
```go
type AggregatedProtocol struct {
    Rank       int                `json:"rank"`
    Name       string             `json:"name"`
    Slug       string             `json:"slug"`
    Category   string             `json:"category"`
    TVL        float64            `json:"tvl"`
    TVS        float64            `json:"tvs"`           // Currently often 0 - TO FIX
    TVSByChain map[string]float64 `json:"tvs_by_chain"`  // Currently often empty - TO FIX
    Chains     []string           `json:"chains"`
    URL        string             `json:"url,omitempty"`
}
```

### Project Structure Notes

- TVS extraction logic belongs in `internal/aggregator/` alongside existing chart/filter/metrics code [Source: docs/architecture/project-structure.md#project-structure]
- Follow table-driven test pattern established in `filter_test.go`, `metrics_test.go`, `chart_test.go`
- Test fixtures in `testdata/` directory

### Learnings from Previous Story

**From Story 5.4 (Status: done)** [Source: docs/sprint-artifacts/5-4-extract-historical-chart-data.md]

- **Pattern Established**: Chart data extraction placed in aggregator layer (`chart.go`) - follow same pattern for TVS extraction
- **Files Modified**: `internal/models/output.go`, `internal/storage/writer.go`, `cmd/extractor/main.go` - may need similar touches
- **New Capability**: `internal/aggregator/chart.go` demonstrates extraction from `OracleAPIResponse` - use as reference
- **Summary vs Full Output**: Summary output kept lightweight (no chart_history) - TVS fields should appear in both since they're per-protocol, not historical
- **Context-Aware Writes**: Maintain context cancellation threading for graceful shutdown

### Testing Guidance

- Follow project testing standards: table-driven unit tests, integration coverage [Source: docs/architecture/testing-strategy.md]
- Add unit tests for TVS extraction edge cases:
  - Protocol exists in filter but missing from `oraclesTVS`
  - Protocol has TVS for some chains but not all
  - Empty `oraclesTVS` map
  - Sum validation tolerance boundary (4.9% vs 5.1%)

### Known Risks

| Risk | Mitigation |
|------|------------|
| `oraclesTVS` key format differs from expected (slug vs ID) | Investigation phase with flexible key matching |
| Not all protocols have TVS data upstream | Graceful degradation with logging |
| Sum discrepancy > 5% may be normal | Document expected sources, make threshold configurable if needed |

### References

- [Source: docs/epics/epic-6-maintenance.md#issue-m-001---per-protocol-tvs-breakdown-missing] - Issue definition and expected outcome
- [Source: docs/sprint-artifacts/tech-spec-epic-6.md#M-001-Per-Protocol-TVS-Breakdown] - Technical specification
- [Source: docs/sprint-artifacts/tech-spec-epic-6.md#Acceptance-Criteria-Authoritative] - Authoritative ACs
- [Source: docs/prd.md#Success-Criteria] - PRD linkage and success metrics
- [Source: docs/architecture/data-architecture.md#API-Response-Models] - API response models
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-002-atomic-file-writes] - Atomic write constraint
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-004-structured-logging-with-slog] - Logging constraint
- [Source: docs/sprint-artifacts/5-4-extract-historical-chart-data.md#Learnings-from-Previous-Story] - Previous story (patterns to follow)

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/6-1-per-protocol-tvs-breakdown.context.xml

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-12-04 | SM Agent (Bob) | Initial story draft created from M-001 issue |
