# Sprint Change Proposal - 2025-12-07

**Author:** PM Agent (John)
**Status:** Draft - Pending Approval

---

## Section 1: Issue Summary

### Problem Statement

The MVP (Epics 1-5) successfully extracts Switchboard oracle data from DefiLlama's auto-tagged protocols. However, two gaps exist:

1. **Untagged Integrations** - Known Switchboard integrations exist that DefiLlama doesn't auto-tag in the `/oracles` endpoint
2. **No Per-Protocol Charting** - Dashboard needs historical TVL time-series per protocol, not just aggregate snapshots

### Context

- MVP delivered and operational
- User has identified specific protocols using Switchboard that aren't captured
- DefiLlama provides per-protocol historical TVL via `/protocol/{slug}` endpoint
- Need to support user-defined protocol lists with integration metadata

### Evidence

- User-provided protocol list with known integration dates and TVS ratios
- API documentation confirms `/protocol/{slug}` returns historical TVL array
- Reference: `docs-from-llm/protocol-query.md`

---

## Section 2: Impact Analysis

### Epic Impact

| Epic | Status | Impact |
|------|--------|--------|
| Epic 1-5 | Complete | None - MVP unaffected |
| Epic 6 | Active | Mark as ongoing maintenance epic |
| Epic 7 | **NEW** | Custom Protocols & TVL Charting |

### Artifact Conflicts

| Artifact | Impact | Action |
|----------|--------|--------|
| PRD | None | Post-MVP growth feature |
| Architecture | Low | Add new API endpoint, output file, config file to docs |
| Epics Index | Low | Add Epic 7 reference |

### Technical Impact

- **New API Integration:** `GET /protocol/{slug}` for per-protocol TVL history
- **New Output File:** `tvl-data.json` (separate from main pipeline)
- **New Config File:** `config/custom-protocols.json`
- **No Breaking Changes:** Additive only, existing outputs unchanged

---

## Section 3: Recommended Approach

### Selected Path: Direct Adjustment

Add Epic 7 as new feature epic, mark Epic 6 as ongoing.

### Rationale

1. **Low Risk** - Separate output file prevents any regression to existing data
2. **Clean Separation** - New pipeline runs alongside existing, no coupling
3. **Reuse Existing Patterns** - HTTP client, retry logic, atomic writes already implemented
4. **Additive Only** - No modifications to working MVP code

### Effort Estimate: Medium
### Risk Level: Low

---

## Section 4: Detailed Change Proposals

### Change #1: Create Epic 7 - Custom Protocols & TVL Charting

**File:** `docs/epics/epic-7-custom-protocols-tvl-charting.md`

**Goal:** Enable tracking of known Switchboard integrations not tagged by DefiLlama, and provide historical TVL charting data for all tracked protocols.

**Input:** `config/custom-protocols.json`
```json
[
  {
    "slug": "drift-trade",
    "is-ongoing": true,
    "live": true,
    "date": 1700000000,
    "simple-tvs-ratio": 0.85,
    "docs_proof": "https://docs.drift.trade/oracles#switchboard",
    "github_proof": "https://github.com/drift-labs/..."
  }
]
```

**Schema:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `slug` | string | Yes | DefiLlama protocol slug |
| `is-ongoing` | boolean | Yes | Whether integration is ongoing |
| `live` | boolean | Yes | If false, skip this protocol entirely |
| `date` | number | No | Unix timestamp of integration |
| `simple-tvs-ratio` | number | Yes | 0-1 decimal for downstream TVS calculation |
| `docs_proof` | string | No | URL to documentation proving Switchboard integration |
| `github_proof` | string | No | URL to code proving Switchboard integration |

**Output:** `data/tvl-data.json`
```json
{
  "version": "1.0.0",
  "metadata": {
    "last_updated": "2025-12-07T12:00:00Z",
    "protocol_count": 25,
    "custom_protocol_count": 4
  },
  "protocols": {
    "drift-trade": {
      "name": "Drift Trade",
      "slug": "drift-trade",
      "source": "custom",
      "is_ongoing": true,
      "simple_tvs_ratio": 0.85,
      "integration_date": 1700000000,
      "docs_proof": "https://...",
      "github_proof": "https://...",
      "current_tvl": 677000000,
      "tvl_history": [
        {
          "date": "2024-01-01",
          "timestamp": 1704067200,
          "tvl": 150000000
        }
      ]
    }
  }
}
```

**Stories:**

| Story | Title | Description |
|-------|-------|-------------|
| 7.1 | Load Custom Protocols Configuration | Parse and validate `config/custom-protocols.json`, filtering out `live: false` entries |
| 7.2 | Implement Protocol TVL Fetcher | Fetch historical TVL from `GET /protocol/{slug}` with retry logic |
| 7.3 | Merge Protocol Lists | Combine auto-detected + custom protocols, dedupe by slug |
| 7.4 | Include Integration Date in Output | Pass through `date` field as `integration_date` (no filtering - downstream handles) |
| 7.5 | Generate tvl-data.json Output | Build and write output file with atomic writes |
| 7.6 | Integrate TVL Pipeline into Main Cycle | Run alongside main extraction in 2-hour cycle |

---

### Change #2: Mark Epic 6 as Ongoing Maintenance Epic

**File:** `docs/epics/epic-6-maintenance.md`

**OLD:**
```markdown
# Epic 6: Maintenance & Data Quality

**Goal:** Address data quality issues and operational improvements discovered post-MVP.

**Nature:** This is a **flexible maintenance epic** - stories are created ad-hoc...
```

**NEW:**
```markdown
# Epic 6: Maintenance & Data Quality

**Goal:** Address data quality issues and operational improvements discovered post-MVP.

**Type:** ONGOING MAINTENANCE EPIC

**Nature:** This is a **flexible maintenance epic** - stories are created ad-hoc...

> **Note:** This epic remains open indefinitely. New maintenance stories will be added as issues are discovered during operation. Epic 6 is never "complete" - it serves as the permanent home for maintenance work.
```

**Rationale:** Explicit marker clarifies Epic 6 has no completion state.

---

## Section 5: Implementation Handoff

### Change Scope: Moderate

New feature epic with 6 stories, plus minor doc update.

### Handoff Plan

| Role | Responsibility |
|------|----------------|
| **SM/PO** | Create Epic 7 file, update Epic 6, update epics index |
| **Architect** | Update architecture docs (api-contracts, project-structure, data-architecture) |
| **Dev Team** | Implement Stories 7.1-7.6 |

### Success Criteria

- [ ] `config/custom-protocols.json` loaded and validated
- [ ] `live: false` protocols skipped
- [ ] All protocols (auto + custom) have TVL history fetched
- [ ] `integration_date` included in output (no filtering)
- [ ] `tvl-data.json` written atomically
- [ ] Logging shows custom vs auto-detected counts
- [ ] Main pipeline (`switchboard-oracle-data.json`) unchanged

### Next Steps

1. Approve this proposal
2. SM creates Epic 7 file from draft
3. SM updates Epic 6 with ongoing marker
4. Architect updates architecture docs
5. Dev begins Story 7.1

---

**Generated:** 2025-12-07
**Workflow:** Correct Course (PM Agent)
