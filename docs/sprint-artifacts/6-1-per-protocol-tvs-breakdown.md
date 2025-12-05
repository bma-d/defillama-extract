# Story 6.1: Per-Protocol TVS Breakdown

Status: done

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

- [x] Task 1: Investigate API Response Structure (AC: 1, 2)
  - [x] 1.1: Fetch raw `/oracles` API response and examine `oraclesTVS` structure
  - [x] 1.2: Verify key format (protocol slug vs ID) in `oraclesTVS` mapping
  - [x] 1.3: Cross-reference protocol slugs between `/oracles` and `/lite/protocols2`
  - [x] 1.4: Document findings in this story's Dev Notes section
  - [x] 1.5: Update test fixture `testdata/oracle_response.json` with realistic `oraclesTVS` data

- [x] Task 2: Implement Per-Protocol TVS Extraction (AC: 1, 2)
  - [x] 2.1: Create helper function `ExtractProtocolTVS(oraclesTVS, oracleName, protocolSlug)` in `internal/aggregator/`
  - [x] 2.2: Return `(totalTVS float64, tvsByChain map[string]float64, found bool)`
  - [x] 2.3: Handle case where protocol not found in `oraclesTVS` (return 0, empty map, false)
  - [x] 2.4: Integrate TVS extraction into existing protocol aggregation loop

- [x] Task 3: Populate Protocol TVS Fields (AC: 1, 2)
  - [x] 3.1: Modify protocol aggregation to call TVS extraction for each protocol
  - [x] 3.2: Set `AggregatedProtocol.TVS` = sum of chain values
  - [x] 3.3: Set `AggregatedProtocol.TVSByChain` = per-chain breakdown map
  - [x] 3.4: Ensure existing fields (TVL, rank, category) remain unchanged

- [x] Task 4: Add Warning Logging (AC: 4, 5)
  - [x] 4.1: Log WARNING for each protocol without TVS data (include slug)
  - [x] 4.2: Track count of protocols with/without TVS during extraction
  - [x] 4.3: Log summary at extraction completion: `protocols_with_tvs=N protocols_without_tvs=M`

- [x] Task 5: Add TVS Sum Validation (AC: 3)
  - [x] 5.1: After all protocols processed, calculate sum of `protocols[].tvs`
  - [x] 5.2: Compare to `summary.total_value_secured`
  - [x] 5.3: If discrepancy > 5%, log WARNING with both values and percentage difference
  - [x] 5.4: Document expected discrepancy sources (rounding, timing, upstream gaps)

- [x] Task 6: Write Unit Tests (AC: all)
  - [x] 6.1: Test TVS extraction with mock `oraclesTVS` containing multiple protocols
  - [x] 6.2: Test TVS extraction when protocol missing from `oraclesTVS`
  - [x] 6.3: Test sum validation within and outside 5% tolerance
  - [x] 6.4: Test warning log generation for missing protocols

- [x] Task 7: Integration Testing (AC: all)
  - [x] 7.1: Run extraction with `--once` against live API
  - [x] 7.2: Verify previously zero-TVS protocols now have values (where data exists)
  - [x] 7.3: Verify `tvs_by_chain` populated for protocols with TVS
  - [x] 7.4: Check logs for any `protocol_tvs_unavailable` warnings
  - [x] 7.5: Verify sum validation output

- [x] Task 8: Verification (AC: all)
  - [x] 8.1: Run `go build ./...` and verify success
  - [x] 8.2: Run `go test ./...` and verify all pass
  - [x] 8.3: Run `make lint`
  - [x] 8.4: Compare output before/after to verify no schema breaking changes

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
2025-12-05: Implemented per-protocol TVS extraction helper, integrated counts/logging, updated fixture and unit tests; tests: `go build ./...`, `go test ./...`, `make lint`.
2025-12-05: Investigated live `/oracles` and `/lite/protocols2`; `oraclesTVS` keys are protocol display names (not slugs). Switchboard has 21 TVS entries vs 31 protocols in /lite list; 10 missing entries logged as warnings. Name fallback added covers spaces/case. Discrepancy sources: upstream missing TVS, name/slug mismatch, timing of data refresh.
2025-12-05: Ran `go run ./cmd/extractor --once --config configs/config.yaml`; warnings emitted for 10 protocols without TVS; `tvs_by_chain` populated for available entries; no sum validation warning (diff <5%); outputs written under `data/`.

### Completion Notes List
2025-12-05: Added ExtractProtocolTVS helper, warning/summary logging, sum validation hook, refreshed oracle fixture, and tests covering TVS extraction, counts, and validation tolerance. Live run populated TVS/TVSByChain for 21 protocols; logged 10 warnings for missing upstream TVS; schema unchanged (protocol keys: category, chains, name, rank, slug, tvl, tvs, tvs_by_chain, url).

### File List
- internal/aggregator/tvs.go
- internal/aggregator/tvs_test.go
- internal/aggregator/extractor.go
- internal/aggregator/extractor_test.go
- internal/aggregator/aggregator.go
- internal/aggregator/aggregator_test.go
- internal/aggregator/models.go
- cmd/extractor/main.go
- cmd/extractor/main_test.go
- internal/storage/writer_test.go
- testdata/oracle_response.json
- docs/sprint-artifacts/sprint-status.yaml
- data/switchboard-oracle-data.json
- data/switchboard-oracle-data.min.json
- data/switchboard-summary.json
- data/state.json

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-12-05 | BMad | Senior Developer Review (AI) – Approved; status -> done |
| 2025-12-05 | Amelia (Dev Agent) | Senior Developer Review (AI) updated after fixes; outcome: Ready for re-review |
| 2025-12-05 | Amelia (Dev Agent) | Added TVS extraction helper, logging, validation hooks, fixtures, and tests |
| 2025-12-04 | SM Agent (Bob) | Initial story draft created from M-001 issue |

## Senior Developer Review (AI)

- Reviewer: BMad
- Date: 2025-12-05
- Outcome: Approve — all ACs satisfied; no blocking findings.

### Summary
- AC1–AC5 implemented; warnings emitted when TVS missing; summary logs include TVS counts.
- Tests/build/lint all pass (`go test ./...`, `go build ./...`, `make lint` on 2025-12-05).

### Key Findings
- None blocking. Name/slug fallback present; TVS sum validation warns when protocol sum deviates >5% from chart reference (cmd/extractor/main.go:155-189).

### Acceptance Criteria Coverage
| AC | Status | Evidence |
|----|--------|----------|
| AC1 | Implemented | internal/aggregator/tvs.go:6-28; internal/aggregator/extractor.go:39-78 |
| AC2 | Implemented | internal/aggregator/tvs.go:21-28; internal/aggregator/extractor.go:39-78 |
| AC3 | Implemented | cmd/extractor/main.go:155-189; cmd/extractor/main_test.go:201-295 |
| AC4 | Implemented | internal/aggregator/extractor.go:70-74; internal/aggregator/extractor_test.go:231-253 |
| AC5 | Implemented | cmd/extractor/main.go:252-264 |

### Task Validation
| Task | Verified As | Evidence |
|------|-------------|----------|
| 1.1–1.5 Investigation & fixture update | Verified | testdata/oracle_response.json:1-20; Dev Notes debug log (2025-12-05) |
| 2.x TVS helper | Verified | internal/aggregator/tvs.go:6-28 |
| 3.x Populate protocol TVS fields | Verified | internal/aggregator/extractor.go:39-78; internal/aggregator/aggregator.go:26-60 |
| 4.x Warning & summary logging | Verified | internal/aggregator/extractor.go:70-74; cmd/extractor/main.go:252-264 |
| 5.x TVS sum validation | Verified | cmd/extractor/main.go:155-189; cmd/extractor/main_test.go:201-295 |
| 6.x Unit tests | Verified | internal/aggregator/tvs_test.go:5-42; internal/aggregator/extractor_test.go:13-254 |
| 7.x Integration testing (`--once`) | Partially Verified | data/switchboard-*.json artifacts present; latest run not re-executed during this review |
| 8.x Build/test/lint | Verified | `go build ./...`; `go test ./...`; `make lint` (2025-12-05) |

### Test Coverage and Gaps
- Executed: `go test ./...` (2025-12-05); TVS extraction and warning paths covered.
- Executed: `go build ./...`; `make lint` (2025-12-05).
- Not re-run in-review: live `--once` extraction; rely on existing data artifacts.

### Architectural Alignment
- Respects ADR-002 atomic writes, ADR-004 slog warnings, ADR-005 no new deps.

### Security Notes
- No new inputs; only slog warnings. No secrets touched.

### Best-Practices and References
- Go stdlib (`log/slog`, `flag`); table-driven tests per testing-strategy.md.

### Action Items
**Code Changes Required:**
- [ ] None — no blocking issues.

**Advisory Notes:**
- Note: Consider rerunning `go run ./cmd/extractor --once --config configs/config.yaml` to refresh data with latest upstream TVS before release.
