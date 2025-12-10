# Story 6.2: Custom Data Folder for Non-DefiLlama Protocols

Status: ready-for-dev

## Story

As a **data pipeline operator**,
I want **a custom-data folder where I can supply manual TVL history for protocols not on DefiLlama**,
so that **protocols without API data can still be tracked with accurate historical information**.

## Acceptance Criteria

| AC# | Criteria | Verification |
|-----|----------|--------------|
| AC1 | `custom-data/` directory exists at project root and is recognized by the pipeline | Directory created, .gitkeep present |
| AC2 | Per-protocol JSON files follow schema: `{"slug": string, "tvl_history": [{"date": "YYYY-MM-DD", "timestamp": int64, "tvl": float64}]}` | Schema validation in loader |
| AC3 | Custom data loader reads all `*.json` files from `custom-data/` directory | Unit test with multiple files |
| AC4 | Invalid JSON files are logged as warnings but don't fail the pipeline | Error handling test |
| AC5 | Custom data is merged with API data - custom entries win on date conflicts | Merge logic unit test |
| AC6 | Protocols with only custom data (no API response) produce valid output | Integration test |
| AC7 | Config supports optional `custom_data_path` with default `custom-data/` | Config field added |
| AC8 | Pipeline logs custom data load statistics (files loaded, entries merged) | Log output verification |

**AC Source Mapping:** AC1, AC2, AC5, AC6, AC7 align to Epic 6 story M-002 requirements. AC3, AC4, AC8 are internal quality extensions for resilience and observability. [Source: docs/epics/epic-6-maintenance.md#Active-Stories] [Source: docs/sprint-artifacts/tech-spec-epic-6.md#Overview] [Source: docs/prd.md#Success-Criteria]

## Tasks / Subtasks

- [ ] **Task 1: Create custom-data directory structure** (AC: 1)
  - [ ] 1.1 Create `custom-data/` directory at project root
  - [ ] 1.2 Add `.gitkeep` to preserve empty directory
  - [ ] 1.3 Add example file `custom-data/_example.json.template` showing schema
  - [ ] 1.4 Test: dir discovered by pipeline when empty (AC: 1) — follow testing-strategy table-driven pattern [Source: docs/architecture/testing-strategy.md]

- [ ] **Task 2: Define custom data schema and loader** (AC: 2, 3, 4)
  - [ ] 2.1 Create `internal/tvl/customdata.go` with `CustomDataLoader` struct
  - [ ] 2.2 Implement `Load(ctx) (map[string][]models.TVLHistoryItem, error)`
  - [ ] 2.3 Add schema validation for required fields
  - [ ] 2.4 Handle missing directory gracefully (return empty map, log info)
  - [ ] 2.5 Handle invalid JSON files (log warning, skip file, continue)
  - [ ] 2.6 Write unit tests in `internal/tvl/customdata_test.go`
  - [ ] 2.7 Test: invalid schema file yields warning but continues (AC: 4)

- [ ] **Task 3: Implement merge logic** (AC: 5, 6)
  - [ ] 3.1 Create `MergeTVLHistory(apiData, customData []TVLHistoryItem) []TVLHistoryItem`
  - [ ] 3.2 Dedupe by date, custom data wins on conflicts
  - [ ] 3.3 Sort result by timestamp ascending
  - [ ] 3.4 Write unit tests for merge scenarios (overlap, no overlap, custom-only)
  - [ ] 3.5 Test: custom-only protocol produces valid merged output (AC: 6)

- [ ] **Task 4: Update config** (AC: 7)
  - [ ] 4.1 Add `CustomDataPath string` to `TVLConfig` struct
  - [ ] 4.2 Set default to `custom-data/`
  - [ ] 4.3 Add env override `TVL_CUSTOM_DATA_PATH`
  - [ ] 4.4 Update `configs/config.yaml` with new field
  - [ ] 4.5 Test: env override respected and default retained when unset (AC: 7)

- [ ] **Task 5: Integrate into pipeline** (AC: 5, 6, 8)
  - [ ] 5.1 Add `CustomDataLoader` to `RunnerDeps` struct
  - [ ] 5.2 Load custom data after API fetch in `RunTVLPipeline`
  - [ ] 5.3 Merge custom data into `tvlData` map before output generation
  - [ ] 5.4 Add logging for custom data statistics
  - [ ] 5.5 Update pipeline tests
  - [ ] 5.6 Test: log contains files loaded / entries merged metrics (AC: 8)

- [ ] **Task 6: Update Epic 6 with story 6.2 definition** (AC: N/A - process)
  - [ ] 6.1 Add story 6.2 to epic-6-maintenance.md Known Issues / Completed Stories
  - [ ] 6.2 Add story key `6-2-custom-data-folder` to sprint-status.yaml
  - [ ] 6.3 Test: story shows in sprint-status.yaml and epic markdown (consistency check)

## Dev Notes

### Architecture Constraints

- Follow existing patterns from `internal/tvl/custom.go` (CustomLoader)
- Use `models.TVLHistoryItem` for history data structure consistency
- Maintain atomic file operations pattern if writing any files
- Use `log/slog` structured logging per ADR-004
- Align tests with guidance in testing strategy [Source: docs/architecture/testing-strategy.md]
- When touching file IO, follow ADR-002 atomic write rules [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-002-atomic-file-writes]

### Integration Points

| Component | File | Modification |
|-----------|------|--------------|
| Custom Data Loader | `internal/tvl/customdata.go` | NEW |
| Config | `internal/config/config.go` | Add `CustomDataPath` to `TVLConfig` |
| Pipeline | `internal/tvl/pipeline.go` | Load and merge custom data |
| Output | `internal/tvl/output.go` | No change - uses existing merge result |

### Data Flow

```
1. Load custom-protocols.json (existing)
2. Merge with auto-detected slugs (existing)
3. Fetch TVL from DefiLlama API (existing)
4. Load custom-data/*.json files (NEW)
5. Merge API data with custom data per protocol (NEW)
6. Generate output (existing)
```

### Custom Data File Schema

```json
{
  "slug": "protocol-slug",
  "tvl_history": [
    {
      "date": "2024-01-15",
      "timestamp": 1705276800,
      "tvl": 1500000.00
    }
  ]
}
```

### Project Structure Notes

- `custom-data/` at root level (same level as `configs/`, `data/`)
- One JSON file per protocol, named `{slug}.json`
- Files starting with `_` are ignored (allows `_example.json.template`)
- No unified-project-structure doc present; follow current repo layout and keep new files under `internal/tvl/` for loaders and `internal/config/` for config changes.

### References

- [Source: docs/epics/epic-6-maintenance.md#Active-Stories] - Epic source and AC baseline
- [Source: docs/sprint-artifacts/tech-spec-epic-6.md#Overview] - Tech spec context for Epic 6
- [Source: internal/tvl/custom.go] - Pattern for file loading with validation
- [Source: internal/models/tvl.go#TVLHistoryItem] - History item structure
- [Source: internal/tvl/pipeline.go#RunTVLPipeline] - Integration point
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-004] - Structured logging
- [Source: docs/architecture/testing-strategy.md] - Testing approach to follow
- [Source: docs/architecture/data-architecture.md#API-Response-Models] - Data model alignment for TVL/TVS history
- [Source: docs/prd.md#Success-Criteria] - PRD linkage and success metrics

### Learnings from Previous Story

- Previous story 6-1 (Per-Protocol TVS Breakdown) is **done**. Key takeaways: added TVS extraction and logging patterns, verified via `go test ./...` and live run; no unresolved review items. [Source: docs/sprint-artifacts/6-1-per-protocol-tvs-breakdown.md]
- New/modified files to reuse patterns from 6-1: `internal/aggregator/tvs.go`, `internal/aggregator/tvs_test.go`, `internal/aggregator/extractor.go`, `cmd/extractor/main.go` (logging); see File List in story 6-1 for full set. [Source: docs/sprint-artifacts/6-1-per-protocol-tvs-breakdown.md]
- Reuse structured logging style and table-driven tests from 6-1 changes (see internal/aggregator/*). No open review follow-ups to carry over.

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/6-2-custom-data-folder.context.xml

### Agent Model Used

Agent Model: ChatGPT (gpt-5.1)  

### Debug Log References
- 2025-12-10: Auto-fix applied after validation findings; story updated for traceability/testing.

### Completion Notes List
- Initialized continuity, citations, testing subtasks, and records per validation feedback.

### File List
- docs/sprint-artifacts/6-2-custom-data-folder.md (this file)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-12-10 | Bob (SM) | Auto-fix after validation: added continuity, citations, testing subtasks, records |
