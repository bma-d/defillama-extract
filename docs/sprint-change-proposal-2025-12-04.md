# Sprint Change Proposal - Data Quality Maintenance Epic

**Date:** 2025-12-04
**Triggered By:** Post-implementation data quality discovery
**Change Scope:** Minor - Direct implementation by development team

---

## 1. Issue Summary

### Problem Statement

Post-MVP deployment revealed that **per-protocol TVS (Total Value Secured) breakdown is incomplete**. While the extractor correctly aggregates total TVS (~$1.02B across 31 protocols), individual protocol entries often show `tvs: 0` and empty `tvs_by_chain: {}` despite having substantial TVL.

### Evidence

From `data/switchboard-summary.json` (2025-12-04):

| Protocol | TVL | TVS | Status |
|----------|-----|-----|--------|
| Kamino Lend | $2.37B | $791M | Has breakdown |
| Jito Liquid Staking | $2.05B | **$0** | Missing |
| Drift Trade | $677M | **$0** | Missing |
| Save | $145M | **$0** | Missing |
| marginfi Lending | $116M | $116M | Has breakdown |

The dashboard requires per-protocol TVS attribution to display "where the total TVS comes from" - this is currently impossible for protocols with missing data.

### Discovery Context

- Discovered during dashboard UI integration
- MVP functional requirements technically met (total TVS captured)
- Data granularity insufficient for intended dashboard visualization

---

## 2. Impact Analysis

### Epic Impact

| Epic | Status | Impact |
|------|--------|--------|
| Epic 1-5 | DONE | No changes required |
| Epic 6 (NEW) | Proposed | Maintenance epic for ongoing data quality |

### Artifact Impact

| Artifact | Change Required |
|----------|-----------------|
| PRD | No change - MVP scope achieved; maintenance is operational concern |
| Architecture | No change - existing patterns support new extraction logic |
| Epics | ADD Epic 6 - Maintenance & Data Quality |
| Sprint Status | ADD Epic 6 tracking |
| Backlog | Reference Epic 6 for data quality items |

### Technical Impact

- May require additional API calls to get per-protocol TVS
- DefiLlama API structure analysis needed to find TVS source
- Existing aggregation pipeline may need enhancement

---

## 3. Recommended Approach

**Selected Path:** Direct Adjustment - Create Maintenance Epic

### Rationale

1. **MVP is complete** - all 5 epics delivered, core functionality works
2. **Additive work** - enhancing data quality, not fixing broken functionality
3. **Ongoing nature** - data quality issues will continue to surface; needs permanent home
4. **Low risk** - doesn't affect existing stable code unless explicitly addressed

### Approach Details

Create **Epic 6: Maintenance & Data Quality** as a flexible, living epic that:
- Captures data quality issues as they're discovered
- Does NOT pre-populate with stories
- Stories created ad-hoc when issues are prioritized
- Serves as the permanent home for operational improvements

---

## 4. Detailed Change Proposals

### Epic 6: Maintenance & Data Quality

**File:** `docs/epics/epic-6-maintenance.md`

```markdown
# Epic 6: Maintenance & Data Quality

**Goal:** Address data quality issues and operational improvements discovered post-MVP.

**Nature:** This is a **flexible maintenance epic** - stories are created ad-hoc as issues are discovered and prioritized. Unlike feature epics, this epic intentionally does NOT have a predefined story list.

## Why This Epic Exists

The MVP (Epics 1-5) delivered a working data extraction pipeline. However, real-world usage reveals data quality gaps, edge cases, and enhancement opportunities that weren't apparent during initial development.

This epic provides a structured home for:
- Data quality fixes (missing fields, incorrect mappings)
- API response handling improvements
- Output schema enhancements
- Operational reliability improvements

## Story Creation Process

1. Issue discovered (user report, dashboard integration, monitoring)
2. Issue documented in this epic's "Known Issues" section
3. When prioritized, issue is converted to a story using standard template
4. Story implemented following existing dev workflow

## Known Issues

| ID | Issue | Severity | Status | Story |
|----|-------|----------|--------|-------|
| M-001 | Per-protocol TVS breakdown missing for many protocols | High | Ready for Story | - |

## Completed Stories

_None yet - stories will be added as issues are addressed._

---

## Issue: M-001 - Per-Protocol TVS Breakdown Missing

**Discovered:** 2025-12-04
**Severity:** High
**Impact:** Dashboard cannot display TVS attribution per protocol

### Problem

Many protocols in output have `tvs: 0` and `tvs_by_chain: {}` despite having substantial TVL. Examples:
- Jito Liquid Staking: TVL $2.05B, TVS $0
- Drift Trade: TVL $677M, TVS $0
- Save: TVL $145M, TVS $0

### Expected Outcome

When this issue is resolved:
- Each protocol should have non-zero `tvs` value (where data exists in API)
- Each protocol should have populated `tvs_by_chain` breakdown
- Sum of all protocol TVS should approximately match `summary.total_value_secured` (within 5% tolerance)
- Protocols without upstream TVS data should be logged as warnings
- Existing output schema preserved (no breaking changes)

### Investigation Required

- [ ] Investigate `/oracles` API response structure for per-protocol TVS source
- [ ] Check if TVS data exists in different API field
- [ ] Determine if additional API calls required
- [ ] Assess DefiLlama data completeness (may be upstream issue)

### Technical Context

- Current issue: Many protocols show `tvs: 0` despite having TVL
- The `oracles` endpoint returns `tvl` per protocol in the oracle's protocol list
- May need to cross-reference with protocol-specific chain data
- Package: `internal/aggregator/` (likely modifications to existing files)
- Reference: `data/switchboard-summary.json` for current state

### Verification Steps (for eventual story)

1. Run extraction with `--once`
2. Verify protocols previously showing `tvs: 0` now have values (where API provides data)
3. Verify `tvs_by_chain` is populated for protocols with TVS
4. Sum all `protocols[].tvs` and compare to `summary.total_value_secured`
5. Check logs for any protocols where TVS data unavailable upstream

---
```

### Sprint Status Update

**File:** `docs/sprint-artifacts/sprint-status.yaml`

Add after Epic 5 section:

```yaml
  # Epic 6: Maintenance & Data Quality (flexible - stories created ad-hoc)
  epic-6: active
  # Stories added here by SM as issues are converted via *create-story
```

---

## 5. Implementation Handoff

### Change Scope Classification

**Minor** - Can be implemented directly by development team

### Deliverables

1. Create `docs/epics/epic-6-maintenance.md` with flexible epic structure
2. Update `docs/sprint-artifacts/sprint-status.yaml` to track Epic 6
3. M-001 issue documented with expected outcomes, technical context, and verification steps

### Next Steps

1. **Immediate:** Create Epic 6 file and update sprint status
2. **SM:** Use `*create-story` to convert M-001 into Story 6.1 when ready
3. **Ongoing:** Add future maintenance issues to Epic 6 as they arise

### Success Criteria

- Epic 6 exists as permanent home for maintenance work
- M-001 documented with sufficient detail for SM to create story
- SM can convert issues to stories using standard workflow

---

## Approval

- [ ] Sprint Change Proposal reviewed
- [ ] Epic 6 structure approved
- [ ] Ready for implementation

---

_Generated by Correct Course workflow - 2025-12-04_
