# Story 7.5: Generate tvl-data.json Output

Status: done

## Story

As a **TVL data consumer**,
I want **the system to generate a tvl-data.json file containing all protocol TVL data with metadata**,
So that **downstream applications can access comprehensive historical TVL data for all tracked protocols (auto-detected and custom) in a standardized format**.

## Acceptance Criteria

Source: [Source: docs/sprint-artifacts/tech-spec-epic-7.md#AC-7.5]; [Source: docs/epics/epic-7-custom-protocols-tvl-charting.md#Story-7.5]

**AC1: Output Structure Matches Schema**
**Given** merged protocols with TVL data
**When** tvl-data.json is generated
**Then** output structure matches the defined TVLOutput schema:
```json
{
  "version": "1.0.0",
  "metadata": { ... },
  "protocols": { "slug": TVLOutputProtocol, ... }
}
```

**AC2: Metadata Contains Required Fields**
**Given** a TVL output generation
**When** tvl-data.json is written
**Then** `metadata.last_updated` is in ISO 8601 format
**And** `metadata.protocol_count` equals total protocols in output
**And** `metadata.custom_protocol_count` equals protocols with `source: "custom"`

**AC3: Protocols Keyed by Slug**
**Given** merged protocols
**When** tvl-data.json is generated
**Then** `protocols` is a map keyed by protocol slug
**And** each protocol includes all fields from TVLOutputProtocol struct

**AC4: TVL History Format**
**Given** a protocol with TVL data
**When** included in tvl-data.json
**Then** `tvl_history` entries have `date` (YYYY-MM-DD), `timestamp` (Unix), `tvl` (float)
**And** full history is preserved (no filtering)

**AC5: Atomic Write**
**Given** TVL output ready for writing
**When** tvl-data.json is written
**Then** uses `WriteAtomic()` pattern (temp file + rename)
**And** file permissions are 0644

**AC6: Minified Version**
**Given** TVL output written
**When** write completes
**Then** minified version written to `tvl-data.min.json`
**And** contains same data without indentation

## Tasks / Subtasks

- [x] Task 1: Define TVLOutput Root Structure (AC: 1, 2)
  - [x] 1.1: Add `TVLOutput` struct to `internal/models/tvl.go` with Version, Metadata, Protocols fields
  - [x] 1.2: Add `TVLOutputMetadata` struct with LastUpdated, ProtocolCount, CustomProtocolCount
  - [x] 1.3: Ensure JSON tags use snake_case to match schema

- [x] Task 2: Implement GenerateTVLOutput Function (AC: 1, 2, 3, 4)
  - [x] 2.1: Create `GenerateTVLOutput` function in `internal/tvl/output.go`
  - [x] 2.2: Accept `[]MergedProtocol` and `map[string]*api.ProtocolTVLResponse` as inputs
  - [x] 2.3: Build protocols map keyed by slug using existing `MapToOutputProtocol`
  - [x] 2.4: Calculate protocol_count (total) and custom_protocol_count (source == "custom")
  - [x] 2.5: Set last_updated to current UTC time in RFC3339 format
  - [x] 2.6: Set version to "1.0.0"

- [x] Task 3: Implement WriteTVLOutputs Function (AC: 5, 6)
  - [x] 3.1: Create `WriteTVLOutputs` function in `internal/tvl/output.go`
  - [x] 3.2: Accept context, outputDir, output paths (or use defaults), and TVLOutput
  - [x] 3.3: Use `storage.WriteJSON` for indented version (tvl-data.json)
  - [x] 3.4: Use `storage.WriteJSON` for minified version (tvl-data.min.json)
  - [x] 3.5: Respect context cancellation between writes
  - [x] 3.6: Return error on any write failure (no partial state)

- [x] Task 4: Write Unit Tests (AC: all)
  - [x] 4.1: Test: GenerateTVLOutput with mixed auto/custom protocols returns correct counts
  - [x] 4.2: Test: Metadata.last_updated is valid RFC3339 timestamp
  - [x] 4.3: Protocols map keyed by slug correctly
  - [x] 4.4: Empty protocol list produces valid output with zero counts
  - [x] 4.5: TVLHistoryItem dates formatted correctly (YYYY-MM-DD)
  - [x] 4.6: WriteTVLOutputs creates both files atomically
  - [x] 4.7: Context cancellation prevents/rolls back writes

- [x] Task 5: Build and Test Verification (AC: all)
  - [x] 5.1: Run `go build ./...` and verify success
  - [x] 5.2: Run `go test ./internal/tvl/...` and verify all pass
  - [x] 5.3: Run `go test ./internal/models/...` if applicable

## Dev Notes

### Technical Guidance

**Key Insight:** Story 7.4 established the `TVLOutputProtocol` model and `MapToOutputProtocol` function. This story focuses on:
1. Defining the root `TVLOutput` and `TVLOutputMetadata` structures
2. Implementing the generator that builds the complete output from merged protocols
3. Implementing the writer that persists both full and minified versions

**Existing Infrastructure to Reuse:**
- `internal/tvl/output.go` - Already has `MapToOutputProtocol` function
- `internal/storage/writer.go` - Has `WriteJSON` and `WriteAtomic` functions
- `internal/models/tvl.go` - Has `TVLOutputProtocol`, `TVLHistoryItem` structs

**Files to Create/Modify:**
- `internal/models/tvl.go` - Add `TVLOutput` and `TVLOutputMetadata` structs
- `internal/tvl/output.go` - Add `GenerateTVLOutput` and `WriteTVLOutputs` functions
- `internal/tvl/output_test.go` - Add tests for new functions

### Learnings from Previous Story

**From Story 7-4-include-integration-date-in-output (Status: done)**

- **MapToOutputProtocol Available**: `internal/tvl/output.go:14-49` - converts MergedProtocol + TVL data to TVLOutputProtocol
- **TVLOutputProtocol Schema Defined**: `internal/models/tvl.go:39-53` - full output protocol struct with all fields
- **TVLHistoryItem Format**: `internal/models/tvl.go:30-37` - Date (YYYY-MM-DD), Timestamp (Unix), TVL (float64)
- **CurrentTVL Derivation**: Uses last TVL point from history (`tvl.TVL[len(tvl.TVL)-1].TotalLiquidityUSD`)
- **Name Fallback**: If protocol.Name is empty, uses tvl.Name from API response
- **No New Dependencies**: Pure logic, stdlib only (ADR-005 compliant)
- **New/Modified Files from 7.4**: `internal/models/tvl.go`, `internal/tvl/output.go`, `internal/tvl/output_test.go`, `docs/sprint-artifacts/sprint-status.yaml`, `docs/sprint-artifacts/7-4-include-integration-date-in-output.md`
- **Completion Notes Recap**: Added TVL output contract with nullable integration_date, preserved full history, mapped custom/auto/null cases, and verified via unit tests (AC1-AC4).

[Source: docs/sprint-artifacts/7-4-include-integration-date-in-output.md#Dev-Agent-Record]

### Architecture Patterns and Constraints

- **ADR-002**: Atomic file writes (temp + rename) [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-002]
- **ADR-003**: Explicit error returns; no panics [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-003]
- **ADR-005**: No new external dependencies [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-005]
- **Existing Pattern**: Follow `WriteAllOutputs` pattern from `internal/storage/writer.go:162-230`
- **JSON Formatting**: Use `json.MarshalIndent` for full, `json.Marshal` for minified

### Data Model Reference

**TVLOutput (to add to internal/models/tvl.go):**
```go
type TVLOutput struct {
    Version   string                       `json:"version"`
    Metadata  TVLOutputMetadata            `json:"metadata"`
    Protocols map[string]TVLOutputProtocol `json:"protocols"`
}

type TVLOutputMetadata struct {
    LastUpdated         string `json:"last_updated"`
    ProtocolCount       int    `json:"protocol_count"`
    CustomProtocolCount int    `json:"custom_protocol_count"`
}
```

**Expected Output Example:**
```json
{
  "version": "1.0.0",
  "metadata": {
    "last_updated": "2025-12-08T12:00:00Z",
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
        {"date": "2024-01-01", "timestamp": 1704067200, "tvl": 150000000}
      ]
    }
  }
}
```

### Project Structure Notes

- `TVLOutput` and `TVLOutputMetadata` structs go in `internal/models/tvl.go` alongside existing TVL models
- Generator and writer functions go in `internal/tvl/output.go` extending existing mapper
- Follow existing package organization established in Stories 7.1-7.4

### Testing Strategy

- Table-driven tests with named cases [Source: docs/architecture/testing-strategy.md]
- Test edge cases: empty protocols, all custom, all auto, mixed
- Test time formatting (RFC3339 for metadata, YYYY-MM-DD for history)
- Test atomic write by checking file existence and content
- Test context cancellation behavior

### References

- [Source: docs/sprint-artifacts/tech-spec-epic-7.md#AC-7.5] - Acceptance criteria definition
- [Source: docs/sprint-artifacts/tech-spec-epic-7.md#Data-Models-and-Contracts] - TVLOutput schema definition
- [Source: docs/epics/epic-7-custom-protocols-tvl-charting.md#Story-7.5] - Epic story definition
- [Source: docs/prd.md#executive-summary] - Product requirement alignment
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-002] - Atomic file writes
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-003] - Error handling
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-005] - Dependency constraints
- [Source: docs/sprint-artifacts/7-4-include-integration-date-in-output.md#Dev-Agent-Record] - Previous story learnings
- [Source: internal/tvl/output.go:14-49] - MapToOutputProtocol function
- [Source: internal/models/tvl.go:39-53] - TVLOutputProtocol struct
- [Source: internal/storage/writer.go:232-292] - WriteAtomic function
- [Source: internal/storage/writer.go:141-158] - WriteJSON function

## Dev Agent Record

### Context Reference

docs/sprint-artifacts/7-5-generate-tvl-data-json-output.context.xml

### Agent Model Used

gpt-5 (Scrum Master auto-fix)

### Debug Log References

- 2025-12-08: Auto-fix applied after validation to address continuity, PRD citation, and Dev Agent Record completeness.
- 2025-12-08: Implemented TVLOutput root structs, generator, writer, and tests; validated via go test ./internal/tvl/... and go build ./...

### Completion Notes List

- 2025-12-08: Added previous-story file learnings and completion notes, cited PRD, populated Dev Agent Record metadata and file list.
- 2025-12-08: TVL output generation completed (AC1-AC6). Added TVLOutput structures, GenerateTVLOutput, WriteTVLOutputs with atomic writes, and unit coverage for counts, metadata, minified output, and cancellation.

### File List

- docs/sprint-artifacts/7-5-generate-tvl-data-json-output.md
- docs/sprint-artifacts/validation-report-2025-12-08T06-43-36Z.md
- docs/sprint-artifacts/sprint-status.yaml
- internal/models/tvl.go
- internal/tvl/output.go
- internal/tvl/output_test.go

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-12-08 | SM Agent (Bob) | Initial story draft created from Epic 7 / Tech Spec |
| 2025-12-08 | Amelia | Implemented TVL output structures, generator, writer, and tests; set story to review |
| 2025-12-08 | Amelia (AI Review) | Senior Developer Review completed; approved and notes appended |

## Senior Developer Review (AI)

- Reviewer: BMad
- Date: 2025-12-08
- Outcome: Approve — All ACs and completed tasks verified with evidence; no change requests.

### Summary
- TVL output root structs, generator, and atomic writers match Epic 7 schema; minified output produced; tests cover AC-critical paths; build and full test suite passing.

### Key Findings
- No High/Medium findings. No code changes requested.

### Acceptance Criteria Coverage
| AC | Status | Evidence |
| --- | --- | --- |
| AC1 Output structure matches schema | Implemented | internal/models/tvl.go:39-45; internal/tvl/output.go:60-73; internal/tvl/output_test.go:132-168 |
| AC2 Metadata fields + ISO timestamp | Implemented | internal/tvl/output.go:60-80; internal/tvl/output_test.go:149-159 |
| AC3 Protocols keyed by slug, all fields present | Implemented | internal/tvl/output.go:60-73; internal/tvl/output_test.go:132-167 |
| AC4 TVL history format preserved | Implemented | internal/tvl/output.go:23-34; internal/tvl/output_test.go:106-124 |
| AC5 Atomic writes with 0644 perms | Implemented | internal/tvl/output.go:84-135; internal/storage/writer.go:141-158,232-278 |
| AC6 Minified version matches data | Implemented | internal/tvl/output.go:132-135; internal/tvl/output_test.go:182-217 |

### Task Completion Validation
| Task | Marked As | Verified As | Evidence |
| --- | --- | --- | --- |
| 1 Define TVLOutput root structures | Completed | Verified | internal/models/tvl.go:39-53 |
| 2 GenerateTVLOutput implementation | Completed | Verified | internal/tvl/output.go:56-82; internal/tvl/output_test.go:132-180 |
| 3 WriteTVLOutputs (atomic + minified) | Completed | Verified | internal/tvl/output.go:84-135; internal/storage/writer.go:141-158,232-278; internal/tvl/output_test.go:182-243 |
| 4 Unit tests added | Completed | Verified | internal/tvl/output_test.go:16-243 |
| 5 Build & test runs | Completed | Verified | `go build ./...`; `go test ./...` |

### Test Coverage and Gaps
- Executed: `go build ./...`, `go test ./...` (2025-12-08) — all packages pass. Tests cover metadata, slug mapping, history preservation, minified output, and context cancellation. No gaps noted for AC1-AC6.

### Architectural Alignment
- Atomic writes via WriteAtomic with fsync and 0644 perms (ADR-002) — internal/storage/writer.go:232-278.
- Explicit error returns, no panics (ADR-003) — WriteTVLOutputs returns errors for nil output/context cancellation.
- No new dependencies added (ADR-005); stdlib only.

### Security Notes
- Output writing uses WriteAtomic; no new I/O surfaces beyond json marshalling. No secrets or external calls added.

### Best-Practices and References
- Go 1.24; stdlib JSON and context handling. Patterns mirror existing WriteAllOutputs for atomic multi-file writes.

### Action Items
**Code Changes Required:** None.

**Advisory Notes:**
- Note: No additional follow-ups identified.
