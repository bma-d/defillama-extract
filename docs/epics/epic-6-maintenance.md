# Epic 6: Maintenance & Data Quality

**Goal:** Address data quality issues and operational improvements discovered post-MVP.

**Type:** ONGOING MAINTENANCE EPIC

**Nature:** This is a **flexible maintenance epic** - stories are created ad-hoc as issues are discovered and prioritized. Unlike feature epics, this epic intentionally does NOT have a predefined story list.

> **Note:** This epic remains open indefinitely. New maintenance stories will be added as issues are discovered during operation. Epic 6 is never "complete" - it serves as the permanent home for maintenance work.
>
> **Priority Rule:** Stories from Epic 6 are never automatically selected as the "next story" unless explicitly specified by the user. Feature epics (7+) always take precedence in the default story queue.

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
3. When prioritized, SM converts issue to story via `*create-story`
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
