# Epic Technical Specification: Maintenance & Data Quality

Date: 2025-12-04
Author: BMad
Epic ID: 6
Status: Draft

---

## Overview

Epic 6 is a **maintenance and data quality epic** that addresses issues discovered post-MVP. Unlike Epics 1-5 which delivered predefined features, this epic operates as a **living container** for ad-hoc fixes and improvements that emerge from real-world usage of the defillama-extract pipeline.

The initial driver is **M-001: Per-Protocol TVS Breakdown Missing** - a high-severity data quality issue where many protocols show `tvs: 0` despite having substantial TVL. This prevents the dashboard from displaying accurate TVS attribution per protocol.

This epic directly supports PRD goals **3. Complete Protocol Capture** and **1. Prove the Misrepresentation Gap** (prd.md, Success Criteria) by ensuring Switchboard-aligned protocols surface correct TVS values across chains.

**Note:** This is an evolving technical specification. Stories will be added to this epic as issues are discovered and prioritized. Each story addition may extend the scope and acceptance criteria documented here.

## Objectives and Scope

### In Scope

- **Data Quality Fixes:** Correcting missing or incorrect data mappings (e.g., per-protocol TVS)
- **API Response Handling:** Improving parsing of DefiLlama API responses to capture all available data
- **Output Schema Enhancements:** Adding fields or improving data completeness (backward-compatible)
- **Operational Reliability:** Handling edge cases, improving error logging for data gaps
- **Investigation Work:** Determining root cause of data issues (upstream vs. internal)

### Out of Scope

- New features unrelated to data quality (those belong in future feature epics)
- Breaking changes to output schema (all changes must be additive/backward-compatible)
- Performance optimization (unless directly caused by a data quality fix)
- UI/dashboard changes (this epic is backend-only)

## System Architecture Alignment

This epic operates within the existing architecture established in Epics 1-5:

| Component | Role in Epic 6 |
|-----------|----------------|
| `internal/aggregator/` | Primary modification target for data quality fixes |
| `internal/api/` | May need updates if additional API fields require parsing |
| `internal/models/` | May need struct updates for new data fields |
| `internal/storage/` | Unlikely to change unless output format needs extension |

**Architectural Constraints:**
- All changes must preserve backward compatibility with existing output consumers
- Logging must be enhanced to surface data quality warnings (protocols with missing upstream data)
- No new external dependencies unless absolutely required

## Detailed Design

### Services and Modules

| Module | File(s) | Responsibility | Epic 6 Impact |
|--------|---------|----------------|---------------|
| **Aggregator** | `internal/aggregator/aggregator.go` | Orchestrates fetch → filter → aggregate | Primary target for TVS mapping fixes |
| **Filter** | `internal/aggregator/filter.go` | Filters protocols by oracle name | May need to extract additional fields |
| **Metrics** | `internal/aggregator/metrics.go` | Calculates derived metrics | May need to compute per-protocol TVS |
| **API Client** | `internal/api/client.go` | Fetches DefiLlama endpoints | May need to parse additional response fields |
| **Models** | `internal/models/*.go` | Data structure definitions | May need struct field additions |

**M-001 Specific Investigation Areas:**
- `OracleAPIResponse.OraclesTVS` structure: Currently `map[string]map[string]map[string]float64` - need to verify if per-protocol TVS is available here
- Cross-reference between `/oracles` and `/lite/protocols2` responses
- Potential need for protocol-specific chain TVS extraction

### Data Models and Contracts

**Current Output Model (protocols array element):**
```go
type AggregatedProtocol struct {
    Rank       int                `json:"rank"`
    Name       string             `json:"name"`
    Slug       string             `json:"slug"`
    Category   string             `json:"category"`
    TVL        float64            `json:"tvl"`
    TVS        float64            `json:"tvs"`         // Currently often 0
    TVSByChain map[string]float64 `json:"tvs_by_chain"` // Currently often empty
    Chains     []string           `json:"chains"`
    URL        string             `json:"url,omitempty"`
}
```

**Expected Data Flow for M-001 Fix:**
1. `/oracles` response contains `oraclesTVS[oracle][protocol][chain]` mapping
2. For each filtered protocol, look up its TVS data from this mapping
3. Populate `TVS` (sum across chains) and `TVSByChain` (per-chain breakdown)
4. Log warning if protocol exists in filter but has no TVS data upstream

**Schema Stability:** No breaking changes to output schema - only populating currently-empty fields.

### Entity Relationships & Data Flow

- **Upstream → Aggregation:** `oraclesTVS["Switchboard"][protocol_slug][chain]` (DefiLlama `/oracles`) → `AggregatedProtocol.TVSByChain`
- **Aggregation → Derived:** `TVS = sum(TVSByChain values)` stored alongside TVL in `AggregatedProtocol`
- **Storage:** Aggregated protocols array persisted to `data/switchboard-oracle-data.json` and `data/switchboard-oracle-data.min.json`; summary metrics to `data/switchboard-summary.json`; state to `data/state.json`
- **Consumers:** Dashboard and analytics jobs read the JSON outputs; no database layer introduced.
- **Logging:** Missing upstream TVS triggers `WARN protocol_tvs_unavailable` with protocol slug; summary counts logged at extraction completion.

### APIs and Interfaces

**DefiLlama API Response Structure (relevant to M-001):**

```
GET /oracles response:
{
  "oracles": { "Switchboard": ["protocol-slug-1", "protocol-slug-2", ...] },
  "oraclesTVS": {
    "Switchboard": {
      "protocol-slug-1": { "Solana": 1234567.89, "Sui": 234567.89 },
      "protocol-slug-2": { "Solana": 345678.90 }
    }
  },
  "chart": { ... },
  "chainsByOracle": { "Switchboard": ["Solana", "Sui", "Aptos", ...] }
}
```

**Investigation Required:** Verify actual API response matches this structure. The `oraclesTVS` mapping may use protocol slugs or protocol IDs as keys.

### Workflows and Sequencing

**Issue-to-Story Workflow:**
```
1. Issue discovered → Documented in epic-6-maintenance.md (Known Issues table)
2. Issue prioritized → SM runs *create-story to convert to story
3. Story implemented → Developer follows existing dev workflow
4. Story completed → Issue row updated with story reference
5. This tech spec updated → Add acceptance criteria for new story
```

**M-001 Implementation Sequence:**
```
1. Investigation Phase
   └─ Fetch raw API responses, analyze oraclesTVS structure
   └─ Determine if data exists or is upstream gap

2. Implementation Phase (if data exists)
   └─ Modify aggregator to extract per-protocol TVS
   └─ Populate TVS and TVSByChain fields
   └─ Add warning logs for protocols without upstream data

3. Verification Phase
   └─ Run extraction, compare before/after output
   └─ Verify sum of protocol TVS ≈ total_value_secured
```

## Non-Functional Requirements

### Performance

- **No degradation:** Data quality fixes must not increase extraction cycle time beyond current 2-minute baseline
- **Memory stability:** Additional data mapping lookups must not introduce memory leaks
- **No additional API calls:** Fixes should leverage existing API responses; no new endpoints unless absolutely required

### Security

- **No new attack surface:** Maintenance fixes do not introduce new inputs or external integrations
- **Data integrity:** All output changes must be deterministic and reproducible
- **No credentials:** No API keys or authentication (DefiLlama APIs remain public)

### Reliability/Availability

- **Graceful degradation for missing data:** If upstream API lacks per-protocol TVS data, log warning and continue with `tvs: 0` (existing behavior)
- **No extraction failures:** Data quality fixes must not convert warnings into hard failures
- **Backward compatibility:** Existing consumers must continue working without modification

### Observability

- **Enhanced logging for data gaps:**
  - Log WARNING for each protocol where TVS data is unavailable upstream
  - Include protocol slug and expected data location in log message
- **Summary metrics:** Log count of protocols with/without TVS data at end of extraction
- **Example log format:**
  ```
  WARN protocol_tvs_unavailable protocol=jito-liquid-staking reason="not found in oraclesTVS"
  INFO extraction_complete protocols_with_tvs=15 protocols_without_tvs=6
  ```

## Dependencies and Integrations

**External Dependencies (unchanged from MVP):**

| Dependency | Version | Purpose |
|------------|---------|---------|
| Go | 1.21+ | Runtime |
| gopkg.in/yaml.v3 | v3.0.1 (pinned) | Config parsing |
| stdlib log | n/a | Structured logging for warnings/summary |

**Integration Points:**

| System | Interface | Epic 6 Impact |
|--------|-----------|---------------|
| DefiLlama `/oracles` | HTTP GET | May need deeper parsing of `oraclesTVS` structure |
| DefiLlama `/lite/protocols2` | HTTP GET | No changes expected |
| Dashboard consumer | JSON files | Schema unchanged; fields populated more completely |

**No new dependencies required for M-001 or anticipated maintenance work.**

## Acceptance Criteria (Authoritative)

> **Note:** This section will be extended as new stories are added to this epic.

### Epic-Level Acceptance Criteria

| ID | Criterion | Verification |
|----|-----------|--------------|
| AC-E6-01 | All fixes maintain backward compatibility with existing output schema | JSON schema validation |
| AC-E6-02 | No extraction failures introduced by data quality fixes | CI tests pass, manual extraction succeeds |
| AC-E6-03 | Data gaps are logged as warnings, not errors | Log inspection |

### M-001: Per-Protocol TVS Breakdown

| ID | Criterion | Verification |
|----|-----------|--------------|
| AC-M001-01 | Protocols with upstream TVS data have non-zero `tvs` field populated | Output inspection |
| AC-M001-02 | Protocols with upstream TVS data have `tvs_by_chain` populated with per-chain breakdown | Output inspection |
| AC-M001-03 | Sum of all `protocols[].tvs` is within 5% of `summary.total_value_secured` (allowing for rounding) | Automated calculation |
| AC-M001-04 | Protocols without upstream TVS data log WARNING with protocol slug | Log inspection |
| AC-M001-05 | Extraction summary includes count of protocols with/without TVS data | Log inspection |

### Future Stories (placeholder)

_Acceptance criteria for additional stories will be added here as issues are converted to stories._

## Traceability Mapping

| Acceptance Criteria | Spec Section | Component(s) | Test Approach |
|---------------------|--------------|--------------|---------------|
| AC-E6-01 | System Architecture Alignment | All | JSON schema diff before/after |
| AC-E6-02 | Reliability/Availability | `cmd/extractor/main.go` | CI integration test |
| AC-E6-03 | Observability | `internal/aggregator/` | Log output inspection |
| AC-M001-01 | Data Models | `internal/aggregator/aggregator.go` | Unit test with mock API response |
| AC-M001-02 | Data Models | `internal/aggregator/aggregator.go` | Unit test with mock API response |
| AC-M001-03 | Detailed Design | `internal/aggregator/metrics.go` | Integration test |
| AC-M001-04 | Observability | `internal/aggregator/aggregator.go` | Log capture test |
| AC-M001-05 | Observability | `internal/aggregator/aggregator.go` | Log capture test |

## Risks, Assumptions, Open Questions

### Risks

| ID | Risk | Likelihood | Impact | Mitigation |
|----|------|------------|--------|------------|
| R-001 | Per-protocol TVS data may not exist in DefiLlama API | Medium | High | Investigation phase before implementation; graceful degradation if data unavailable |
| R-002 | `oraclesTVS` key structure may differ from expected (slug vs ID) | Medium | Medium | Thorough API response analysis; flexible key matching |
| R-003 | Maintenance fixes may inadvertently break existing functionality | Low | High | Comprehensive test coverage; CI validation |

### Assumptions

| ID | Assumption | Validation Approach |
|----|------------|---------------------|
| A-001 | DefiLlama `/oracles` API contains per-protocol TVS data somewhere | API response inspection during investigation |
| A-002 | Protocol slugs in `oraclesTVS` match slugs from `/lite/protocols2` | Cross-reference comparison |
| A-003 | Dashboard consumers can handle newly-populated fields without modification | Schema is unchanged; only values change |

### Open Questions

| ID | Question | Owner | Resolution Target |
|----|----------|-------|-------------------|
| Q-001 | What is the actual structure of `oraclesTVS` in current API responses? | Developer | Story investigation phase |
| Q-002 | Do all protocols have TVS data, or only some? | Developer | Story investigation phase |
| Q-003 | If TVS data is missing upstream, should we estimate from TVL? | Product | Before story implementation |

## Test Strategy Summary

### Test Approach for Maintenance Stories

| Test Type | Scope | Tools |
|-----------|-------|-------|
| **Unit Tests** | Individual functions modified | Go testing, table-driven |
| **Integration Tests** | End-to-end extraction with mock server | `httptest`, fixture files |
| **Regression Tests** | Ensure existing functionality unchanged | Diff output files before/after |
| **Manual Verification** | Spot-check output for specific protocols | `jq` queries on output JSON |

### M-001 Specific Tests

1. **Unit Test:** Mock `oraclesTVS` data → verify protocol TVS extraction
2. **Unit Test:** Missing protocol in `oraclesTVS` → verify warning logged, `tvs: 0` returned
3. **Integration Test:** Full extraction → verify sum of protocol TVS ≈ total TVS
4. **Regression Test:** Compare output schema before/after (no breaking changes)

### Test Data Requirements

- Update `testdata/oracle_response.json` to include realistic `oraclesTVS` structure
- Create test cases for protocols with and without TVS data
- Fixture for edge case: protocol slug mismatch between endpoints

---

**Document Status:** This tech spec will evolve as stories are added to Epic 6. Last updated sections should be timestamped when modified.
