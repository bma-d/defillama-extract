# Sprint Change Proposal: Epic 5 Story Consolidation

**Date:** 2025-11-30
**Scope:** Minor
**Status:** Approved

---

## 1. Issue Summary

**Problem:** Epic 5 (Output & CLI) was over-fragmented into 10 stories, creating unnecessary overhead and artificial boundaries between tightly-coupled functionality.

**Evidence:**
- Stories 5.1-5.4 all modify `internal/storage/writer.go` - same package, same review
- Story 5.5 (CLI flags) has no value without Story 5.6 (using them)
- Stories 5.7-5.10 are all pieces of `cmd/extractor/main.go`

**Discovery:** Pre-implementation review identified fragmentation as anti-pattern.

---

## 2. Impact Analysis

| Artifact | Impact |
|----------|--------|
| Epic 5 | 10 stories → 3 stories |
| PRD | No changes |
| Architecture | No changes |
| Sprint Status | Will need story ID updates |
| Tech Spec | Not yet created - will use new story structure |

---

## 3. Recommended Approach

**Direct Adjustment** - Consolidate stories within existing epic structure.

**Rationale:**
- No scope change - same functionality, better packaging
- Reduces ceremony (fewer reviews, fewer status updates)
- Clearer deliverable boundaries
- Each consolidated story = one testable, deployable unit

---

## 4. Detailed Changes

### Story Mapping

| New Story | Combines Old | Focus |
|-----------|--------------|-------|
| 5.1 | 5.1 + 5.2 + 5.3 + 5.4 | Output file generation (full/minified/summary + atomic writes) |
| 5.2 | 5.5 + 5.6 + 5.9 | CLI flags + single-run mode + logging |
| 5.3 | 5.7 + 5.8 + 5.10 | Daemon mode + graceful shutdown + complete main |

### FR Coverage (unchanged)

All original FRs remain covered:
- FR35-FR41: Output generation (Story 5.1)
- FR42, FR44-FR46, FR48, FR56: CLI and logging (Story 5.2)
- FR43, FR47: Daemon and shutdown (Story 5.3)

---

## 5. Implementation Handoff

**Scope Classification:** Minor

**Route to:** Development team

**Deliverables:**
- Updated `docs/epics/epic-5-output-cli.md` (complete)
- Sprint status file update needed before Epic 5 begins

**Success Criteria:**
- Tech spec created using 3-story structure
- Stories implemented in order: 5.1 → 5.2 → 5.3
- All original FRs validated in acceptance testing

---

**Approved by:** BMad (2025-11-30)
