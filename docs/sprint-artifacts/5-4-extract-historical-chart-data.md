# Story 5.4: Extract Historical Chart Data for Graphing

Status: ready-for-dev

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

- [ ] Task 1: Add ChartEntry and ChartDataPoint models (AC: 1, 2)
  - [ ] 1.1: Create `internal/aggregator/chart.go` with `ChartDataPoint` struct
  - [ ] 1.2: Add `ChartHistory []ChartDataPoint` field to `FullOutput` in `internal/models/output.go`
  - [ ] 1.3: Add `ChartHistory []ChartDataPoint` field to `SummaryOutput` in `internal/models/output.go`

- [ ] Task 2: Implement Chart Data Extraction (AC: 1, 3)
  - [ ] 2.1: Create `ExtractChartHistory(oracleResp, oracleName)` function in `internal/aggregator/chart.go`
  - [ ] 2.2: Parse timestamp strings from chart map keys
  - [ ] 2.3: Filter entries for target oracle name (e.g., "Switchboard")
  - [ ] 2.4: Extract tvl, borrowed, staking from each chart entry
  - [ ] 2.5: Sort results by timestamp ascending
  - [ ] 2.6: Convert timestamps to ISO date strings

- [ ] Task 3: Integrate Chart Data into Output Generation (AC: 2, 4)
  - [ ] 3.1: Update `GenerateFullOutput` to accept chart history parameter
  - [ ] 3.2: Update `GenerateSummaryOutput` to accept chart history parameter
  - [ ] 3.3: Update `runOnce` to extract chart history and pass to output generation
  - [ ] 3.4: Ensure chart_history is serialized in both full and summary outputs

- [ ] Task 4: Write Unit Tests (AC: all)
  - [ ] 4.1: Test chart extraction with sample API response
  - [ ] 4.2: Test filtering for specific oracle name
  - [ ] 4.3: Test sorting by timestamp
  - [ ] 4.4: Test empty chart data handling
  - [ ] 4.5: Test output JSON includes chart_history array

- [ ] Task 5: Integration Testing (AC: all)
  - [ ] 5.1: Run extraction against real API
  - [ ] 5.2: Verify chart_history has 1000+ entries
  - [ ] 5.3: Verify date range spans from 2021-11-29 to present
  - [ ] 5.4: Verify data is suitable for graphing (sequential, valid values)

- [ ] Task 6: Verification (AC: all)
  - [ ] 6.1: Run `go build ./...` and verify success
  - [ ] 6.2: Run `go test ./...` and verify all pass
  - [ ] 6.3: Run `make lint` and verify no errors
  - [ ] 6.4: Verify output JSON schema matches AC requirements

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

(to be filled during implementation)

### Completion Notes List

(to be filled on completion)

### File List

(to be filled on completion)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-12-02 | SM Agent (Bob) | Initial story draft created - discovered during Epic 5 retrospective |
