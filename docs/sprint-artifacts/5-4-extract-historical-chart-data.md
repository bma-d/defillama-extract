# Story 5.4: Extract Historical Chart Data for Graphing

Status: review

## Story

As a **dashboard consumer**,
I want **historical TVS chart data extracted from DefiLlama**,
So that **I can render time-series graphs showing Switchboard's TVS over time**.

## Acceptance Criteria

Source: [Source: docs/epics/epic-5-output-cli.md#Story-5.4]; [Source: docs/sprint-artifacts/tech-spec-epic-5.md#story-5-4-extract-historical-chart-data-for-graphing]; [Source: docs/prd.md#historical-data-management]

**AC1: Extract Chart Data from API**
**Given** the `/oracles` API response contains `chart` field
**When** extraction runs
**Then** all Switchboard entries from `chart[timestamp]["Switchboard"]` are extracted
**And** each entry includes: timestamp, date, tvl (TVS), borrowed, staking

**AC2: Chart History in Output**
**Given** extracted chart data
**When** output JSON is generated
**Then** a `chart_history` array is included with entries:
  - `timestamp`: Unix timestamp (int64)
  - `date`: ISO date string (YYYY-MM-DD)
  - `tvs`: Total value secured (float64)
  - `borrowed`: Borrowed value (float64, optional)
  - `staking`: Staking value (float64, optional)
**And** entries are sorted by timestamp ascending
**And** array is included in both full output and summary output

**AC3: Chart Data Date Range**
**Given** chart history is generated
**When** output is written
**Then** all available historical data points are included (full history from API)
**And** chart_history contains 1000+ entries (DefiLlama has 4+ years of data)

**AC4: Output Schema Update**
**Given** updated output schema
**When** JSON is generated
**Then** `chart_history` array appears at top level alongside `historical`
**And** `historical` continues to contain extractor-run snapshots (protocol-level detail)
**And** `chart_history` contains API-sourced daily TVS data (for graphing)

## Tasks / Subtasks

- [x] Task 1: Add ChartEntry and ChartDataPoint models (AC: 1, 2)
  - [x] 1.1: Create `internal/aggregator/chart.go` with `ChartDataPoint` struct
  - [x] 1.2: Add `ChartHistory []ChartDataPoint` field to `FullOutput` in `internal/models/output.go`
  - [x] 1.3: Add `ChartHistory []ChartDataPoint` field to `SummaryOutput` in `internal/models/output.go`

- [x] Task 2: Implement Chart Data Extraction (AC: 1, 3)
  - [x] 2.1: Create `ExtractChartHistory(oracleResp, oracleName)` function in `internal/aggregator/chart.go`
  - [x] 2.2: Parse timestamp strings from chart map keys
  - [x] 2.3: Filter entries for target oracle name (e.g., "Switchboard")
  - [x] 2.4: Extract tvl, borrowed, staking from each chart entry
  - [x] 2.5: Sort results by timestamp ascending
  - [x] 2.6: Convert timestamps to ISO date strings

- [x] Task 3: Integrate Chart Data into Output Generation (AC: 2, 4)
  - [x] 3.1: Update `GenerateFullOutput` to accept chart history parameter
  - [x] 3.2: Update `GenerateSummaryOutput` to accept chart history parameter
  - [x] 3.3: Update `runOnce` to extract chart history and pass to output generation
  - [x] 3.4: Ensure chart_history is serialized in both full and summary outputs

- [x] Task 4: Write Unit Tests (AC: all)
  - [x] 4.1: Test chart extraction with sample API response
  - [x] 4.2: Test filtering for specific oracle name
  - [x] 4.3: Test sorting by timestamp
  - [x] 4.4: Test empty chart data handling
  - [x] 4.5: Test output JSON includes chart_history array

- [x] Task 5: Integration Testing (AC: all)
  - [x] 5.1: Run extraction against real API
  - [x] 5.2: Verify chart_history has 1000+ entries
  - [x] 5.3: Verify date range spans from 2021-11-29 to present
  - [x] 5.4: Verify data is suitable for graphing (sequential, valid values)

- [x] Task 6: Verification (AC: all)
  - [x] 6.1: Run `go build ./...` and verify success
  - [x] 6.2: Run `go test ./...` and verify all pass
  - [x] 6.3: Run `make lint` and verify no errors
  - [x] 6.4: Verify output JSON schema matches AC requirements

## Dev Notes

### Technical Guidance

- **Files to Create:**
  - CREATE: `internal/aggregator/chart.go` - Chart data extraction logic
  - CREATE: `internal/aggregator/chart_test.go` - Unit tests

- **Files to Modify:**
  - MODIFY: `internal/models/output.go` - Add ChartHistory field to output structs
  - MODIFY: `internal/storage/writer.go` - Pass chart history to output generation
  - MODIFY: `cmd/extractor/main.go` - Extract chart data in runOnce flow

### Architecture Patterns and Constraints
- Use aggregator-layer helper `internal/aggregator/chart.go` for chart extraction to keep business logic out of CLI wiring [Source: docs/architecture/implementation-patterns.md#dependency-injection]
- Preserve atomic writes via existing writer pattern (temp + rename) for chart_history outputs [Source: docs/architecture/architecture-decision-records-adrs.md#adr-002-atomic-file-writes]
- Ensure context-aware cancellation is threaded through extraction and writes (aligns with daemon/once shutdown rules) [Source: docs/architecture/consistency-rules.md#consistency-rules]

- **API Response Structure:**
  The `/oracles` endpoint returns:
  ```json
  {
    "chart": {
      "1638144000": {
        "Switchboard": {
          "tvl": 6289642.707399444,
          "borrowed": 0,
          "staking": 0
        }
      }
    }
  }
  ```

- **ChartDataPoint Struct:**
  ```go
  type ChartDataPoint struct {
      Timestamp int64   `json:"timestamp"`
      Date      string  `json:"date"`
      TVS       float64 `json:"tvs"`
      Borrowed  float64 `json:"borrowed,omitempty"`
      Staking   float64 `json:"staking,omitempty"`
  }
  ```

- **Expected Data Volume:**
  - ~1,466 data points for Switchboard [Source: docs/sprint-artifacts/tech-spec-epic-5.md#story-5-4-extract-historical-chart-data-for-graphing]
  - Date range: 2021-11-29 to present [Source: docs/sprint-artifacts/tech-spec-epic-5.md#story-5-4-extract-historical-chart-data-for-graphing]
  - Daily granularity

### Learnings from Previous Story
- Previous story: 5.3 Implement Daemon Mode (Status: done) [Source: docs/sprint-artifacts/5-3-implement-daemon-mode.md#story-53-implement-daemon-mode-and-complete-main-entry-point]
- Review follow-ups in 5.3 are already completed; carry forward the practices: keep `--once` writes fully context-aware and maintain daemon signal/scheduler test coverage. [Source: docs/sprint-artifacts/5-3-implement-daemon-mode.md#action-items]
- Reference new/modified files from 5.3 when integrating chart extraction: `cmd/extractor/main.go`, `cmd/extractor/main_test.go`, `internal/storage/writer.go`, `internal/api/responses.go`, `internal/api/responses_test.go` [Source: docs/sprint-artifacts/5-3-implement-daemon-mode.md#file-list]
- Carry forward: honor context-aware writes and graceful shutdown semantics when adding chart extraction
- Open review items: none; all AI review actions from 5.3 were closed in that story.

### Testing Guidance
- Follow project testing standards for table-driven unit tests and integration coverage [Source: docs/architecture/testing-strategy.md]
- Add unit tests for chart extraction edge cases (empty chart, sorting) and ensure chart_history appears in full/summary outputs

### Background

This story was added after discovering that the `chart` field from the DefiLlama API was being fetched but not extracted. The chart data is required to render time-series graphs showing TVS history, as shown on DefiLlama's oracle page.

The existing `historical` array captures snapshots from when the extractor runs (every 2 hours), while `chart_history` captures the full historical TVS data from DefiLlama's records.

### Smoke Test Guide

1. Run `./bin/extractor --once --config configs/config.yaml`
2. Check `data/switchboard-oracle-data.json` for `chart_history` array
3. Verify array has 1000+ entries
4. Verify first entry: `{"timestamp": 1638144000, "date": "2021-11-29", "tvs": 6289642.70...}`
5. Verify last entry matches current date
6. Check `data/switchboard-summary.json` also has `chart_history`

### References

- [Source: docs-from-user/seed-doc/3-data-sources-api-specifications.md#line-28] - Chart field specification
- [Source: docs/epics/epic-5-output-cli.md#Story-5.4] - Story definition
- [Source: docs/sprint-artifacts/tech-spec-epic-5.md#story-5-4-extract-historical-chart-data-for-graphing] - Tech spec source
- [Source: docs/prd.md#historical-data-management] - PRD linkage for historical data output
- [Source: docs/architecture/implementation-patterns.md#dependency-injection] - Aggregator placement
- [Source: docs/architecture/architecture-decision-records-adrs.md#adr-002-atomic-file-writes] - Atomic writes
- [Source: docs/architecture/consistency-rules.md#consistency-rules] - Context and cancellation
- [Source: docs/architecture/testing-strategy.md#Test-Organization] - Testing standards

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/5-4-extract-historical-chart-data.context.xml

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References
- Implemented chart extraction helper and output wiring; updated outputs/tests to carry chart_history per AC4. Build/test/lint executed. Live extraction run produced chart_history=1466 entries (2021-11-29 to 2025-12-03), summary contains chart_history, historical preserved.

### Completion Notes List
- Chart history extracted from /oracles chart for Switchboard, sorted ascending, wired into full/summary outputs; all unit tests/build/lint passing. Live API run confirmed chart_history (1466 points) spans 2021-11-29..2025-12-03 and is sorted.

### File List
- internal/aggregator/chart.go
- internal/aggregator/chart_test.go
- internal/models/output.go
- internal/storage/writer.go
- internal/storage/writer_test.go
- cmd/extractor/main.go
- cmd/extractor/main_test.go
- docs/sprint-artifacts/sprint-status.yaml
- data/switchboard-oracle-data.json
- data/switchboard-summary.json

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-12-02 | SM Agent (Bob) | Initial story draft created - discovered during Epic 5 retrospective |
| 2025-12-03 | Amelia | Added chart_history models, extraction pipeline, output wiring, and unit tests |
| 2025-12-03 | Amelia | Ran live extraction, validated chart_history length/date range, completed build/test/lint |
| 2025-12-03 | Amelia (AI Reviewer) | Senior Developer Review completed — outcome: Approve |

## Senior Developer Review (AI)

- Reviewer: BMad  
- Date: 2025-12-03  
- Outcome: Approve (all ACs satisfied; no blocking issues)

### Summary
- Chart history is extracted, sorted, and threaded through full/summary outputs; live run produced 1,466 points spanning 2021-11-29 to 2025-12-03. No regressions found. Evidence below.

### Key Findings
- HIGH: None
- MEDIUM: None
- LOW: None

### Acceptance Criteria Coverage
| AC | Status | Evidence |
|----|--------|----------|
| AC1 | Implemented | Chart extraction parses timestamps, filters oracle, and captures tvl/borrowed/staking, then sorts ascending — `internal/aggregator/chart.go:20-56`; validated by `internal/aggregator/chart_test.go:10-47`. |
| AC2 | Implemented | `chart_history` added to both outputs and populated during runOnce — `internal/models/output.go:45-66`, `internal/storage/writer.go:42-139`, `cmd/extractor/main.go:68-178`. |
| AC3 | Implemented | Live output contains 1,466 points from 2021-11-29 to 2025-12-03 — `data/switchboard-oracle-data.json` (first: timestamp 1638144000, last: 1764720000) verified via local check. |
| AC4 | Implemented | `chart_history` top-level alongside `historical`; summary omits `historical` while keeping `chart_history` — `internal/models/output.go:45-66`; summary exclusion asserted in `internal/storage/writer_test.go:198-244`. |

AC coverage: 4 / 4 implemented.

### Task Completion Validation
| Task | Marked | Verified | Evidence |
|------|--------|----------|----------|
| Task 1 (models) | [x] | Verified | Output structs include `ChartHistory` — `internal/models/output.go:45-66`. |
| Task 2 (extraction) | [x] | Verified | `ExtractChartHistory` filters and sorts chart data — `internal/aggregator/chart.go:20-56`; tests `chart_test.go:10-47`. |
| Task 3 (integration) | [x] | Verified | runOnce extracts chart history and passes to output generators — `cmd/extractor/main.go:150-178`; output generation threads through writes — `internal/storage/writer.go:42-139`. |
| Task 4 (unit tests) | [x] | Verified | Chart extraction tests and writer coverage for chart_history — `internal/aggregator/chart_test.go`; `internal/storage/writer_test.go:90-244`. |
| Task 5 (integration testing) | [x] | Verified | Live data files generated with 1,466 chart points covering full range — `data/switchboard-oracle-data.json`, `data/switchboard-summary.json`. |
| Task 6 (verification) | [x] | Verified | `go test ./...` passed locally (2025-12-03). |

### Test Coverage and Gaps
- Executed: `go test ./...` (pass).  
- Gaps: None observed for story scope.

### Architectural Alignment
- Uses aggregator-layer helper per implementation-patterns; preserves atomic writes; chart_history included without altering historical arrays — `internal/aggregator/chart.go`, `internal/storage/writer.go`.

### Security Notes
- No new inputs; chart data handled as numeric fields. No secrets added.

### Best-Practices & References
- Atomic file writes preserved (ADR-002); chart extraction isolated to aggregator layer per architecture guidance.

### Action Items
**Code Changes Required:** None  
**Advisory Notes:**  
- Note: Monitor chart_history file size growth in future runs to ensure write times remain acceptable as history expands.
