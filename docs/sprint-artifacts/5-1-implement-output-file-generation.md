# Story 5.1: Implement Output File Generation

Status: drafted

## Story

As a **developer**,
I want **all output JSON files generated with atomic writes**,
so that **dashboards have reliable, complete data in multiple formats**.

## Acceptance Criteria

Source: [Source: docs/sprint-artifacts/tech-spec-epic-5.md#Story-5.1] / [Source: docs/epics/epic-5-output-cli.md#story-51]

**AC1: Full Output JSON**
**Given** aggregation results and historical snapshots
**When** `GenerateFullOutput(result, history, config)` is called
**Then** a `FullOutput` struct is created with:
  - `version`: "1.0.0"
  - `oracle`: name, website, documentation URL from config
  - `metadata`: last_updated, data_source, update_frequency, extractor_version
  - `summary`: total_value_secured, total_protocols, active_chains, categories
  - `metrics`: current_tvs, change_24h, change_7d, change_30d, growth metrics
  - `breakdown`: by_chain array, by_category array
  - `protocols`: ranked protocol list with all metadata
  - `historical`: complete snapshot history
**And** JSON is human-readable with 2-space indentation
**And** file is written to `{output_dir}/switchboard-oracle-data.json`

**AC2: Minified Output JSON**
**Given** the same `FullOutput` data
**When** minified output is generated
**Then** JSON is serialized without whitespace
**And** file is written to `{output_dir}/switchboard-oracle-data.min.json`

**AC3: Summary Output JSON**
**Given** aggregation results (no history needed)
**When** `GenerateSummaryOutput(result, config)` is called
**Then** `SummaryOutput` struct contains current snapshot only:
  - `version`, `oracle`, `metadata`, `summary`, `metrics`, `breakdown`
  - `top_protocols`: top 10 by TVL
  - NO `historical` array
**And** file is written to `{output_dir}/switchboard-summary.json`

**AC4: Atomic File Writes**
**Given** output data to write
**When** `WriteJSON(path, data, indent)` is called
**Then** data is written to temp file first (using `os.CreateTemp` in same dir)
**And** temp file is renamed to target path atomically
**And** directory is created if missing (`os.MkdirAll`)

**Given** write failure mid-operation
**When** error occurs
**Then** temp file is cleaned up, original preserved, error returned

**Given** all outputs ready
**When** `WriteAllOutputs(outputDir, full, summary)` is called
**Then** all three files are written atomically

## Tasks / Subtasks

- [ ] Task 1: Define output model structs (AC: 1, 3)
  - [ ] 1.1: Create `internal/models/output.go` with `FullOutput`, `SummaryOutput`, `OracleInfo`, `OutputMetadata` structs per tech spec
  - [ ] 1.2: Add JSON tags matching schema exactly (version, oracle, metadata, summary, metrics, breakdown, protocols, historical)
  - [ ] 1.3: Verify `SummaryOutput` uses `top_protocols` (not `protocols`) and has no `Historical` field

- [ ] Task 2: Implement `GenerateFullOutput` function (AC: 1)
  - [ ] 2.1: Create `internal/storage/writer.go` with `GenerateFullOutput(result *aggregator.AggregationResult, history []aggregator.Snapshot, cfg *config.Config) *models.FullOutput`
  - [ ] 2.2: Populate `Version` = "1.0.0"
  - [ ] 2.3: Populate `Oracle` from config (Name, Website, Documentation)
  - [ ] 2.4: Populate `Metadata` with last_updated (ISO 8601), data_source ("DefiLlama API"), update_frequency ("2 hours"), extractor_version ("1.0.0")
  - [ ] 2.5: Populate `Summary` from aggregation result (total_value_secured, total_protocols, active_chains, categories)
  - [ ] 2.6: Populate `Metrics` from aggregation result (current_tvs, change_24h, change_7d, change_30d, growth metrics)
  - [ ] 2.7: Populate `Breakdown` from aggregation result (by_chain, by_category arrays)
  - [ ] 2.8: Populate `Protocols` from aggregation result (ranked list)
  - [ ] 2.9: Populate `Historical` from history slice

- [ ] Task 3: Implement `GenerateSummaryOutput` function (AC: 3)
  - [ ] 3.1: Implement `GenerateSummaryOutput(result *aggregator.AggregationResult, cfg *config.Config) *models.SummaryOutput`
  - [ ] 3.2: Populate same fields as full output except: use `TopProtocols` (top 10 only) and NO `Historical`
  - [ ] 3.3: Extract top 10 protocols by TVL from ranked list

- [ ] Task 4: Implement atomic `WriteJSON` function (AC: 4)
  - [ ] 4.1: Implement `WriteJSON(path string, data interface{}, indent bool) error`
  - [ ] 4.2: Use `os.MkdirAll` to create directory if missing
  - [ ] 4.3: Use `os.CreateTemp` in same directory as target (ensures same filesystem for atomic rename)
  - [ ] 4.4: If `indent=true`: use `json.MarshalIndent(data, "", "  ")` for 2-space indentation
  - [ ] 4.5: If `indent=false`: use `json.Marshal(data)` for compact/minified output
  - [ ] 4.6: Write bytes to temp file and close
  - [ ] 4.7: Use `os.Rename(tmpPath, path)` for atomic move
  - [ ] 4.8: On any error: clean up temp file with `os.Remove(tmpPath)` and return error

- [ ] Task 5: Implement `WriteAllOutputs` function (AC: 2, 4)
  - [ ] 5.1: Implement `WriteAllOutputs(outputDir string, full *models.FullOutput, summary *models.SummaryOutput) error`
  - [ ] 5.2: Write full output (indented) to `{outputDir}/switchboard-oracle-data.json`
  - [ ] 5.3: Write minified output (compact, no whitespace) to `{outputDir}/switchboard-oracle-data.min.json` (AC: 2)
  - [ ] 5.4: Verify minified output preserves the same data as full output (AC: 2)
  - [ ] 5.5: Write summary output (indented) to `{outputDir}/switchboard-summary.json`
  - [ ] 5.6: Return first error encountered (fail fast)

- [ ] Task 6: Write unit tests (AC: 1-4)
  - [ ] 6.1: Test `GenerateFullOutput` populates all required fields correctly
  - [ ] 6.2: Test `GenerateSummaryOutput` has no Historical field and limits to 10 protocols
  - [ ] 6.3: Test `WriteJSON` with indent=true produces formatted JSON
  - [ ] 6.4: Test `WriteJSON` with indent=false produces compact JSON (AC: 2)
  - [ ] 6.5: Test `WriteJSON` creates directory if missing
  - [ ] 6.6: Test `WriteJSON` cleans up temp file on write failure (use error injection)
  - [ ] 6.7: Test `WriteAllOutputs` creates all three files
  - [ ] 6.8: Test minified output file has no whitespace and matches full output data (AC: 2)

- [ ] Task 7: Verification (AC: all)
  - [ ] 7.1: Run `go build ./...` and verify success
  - [ ] 7.2: Run `go test ./internal/storage/... ./internal/models/...` and verify all pass
  - [ ] 7.3: Run `make lint` and verify no errors

## Dev Notes

### Technical Guidance

- **Files to Create/Modify:**
  - NEW: `internal/models/output.go` - Output struct definitions
  - NEW/MODIFY: `internal/storage/writer.go` - Writer functions (may need to create if not exists)
  - NEW: `internal/storage/writer_test.go` - Writer tests

- **Atomic Write Pattern** per [Source: docs/architecture/implementation-patterns.md#atomic-file-writes]:
  ```go
  func WriteJSON(path string, data interface{}, indent bool) error {
      dir := filepath.Dir(path)
      if err := os.MkdirAll(dir, 0755); err != nil {
          return fmt.Errorf("create directory: %w", err)
      }

      tmpFile, err := os.CreateTemp(dir, "*.tmp")
      if err != nil {
          return fmt.Errorf("create temp file: %w", err)
      }
      tmpPath := tmpFile.Name()
      defer os.Remove(tmpPath) // cleanup on any error path

      var jsonData []byte
      if indent {
          jsonData, err = json.MarshalIndent(data, "", "  ")
      } else {
          jsonData, err = json.Marshal(data)
      }
      if err != nil {
          return fmt.Errorf("marshal json: %w", err)
      }

      if _, err := tmpFile.Write(jsonData); err != nil {
          tmpFile.Close()
          return fmt.Errorf("write temp file: %w", err)
      }
      if err := tmpFile.Close(); err != nil {
          return fmt.Errorf("close temp file: %w", err)
      }

      if err := os.Rename(tmpPath, path); err != nil {
          return fmt.Errorf("rename to target: %w", err)
      }
      return nil
  }
  ```

- **JSON Schema Compliance**: Output must match schemas in tech spec section "JSON Schemas (authoritative contracts)"
- **FR Coverage**: FR35 (full), FR36 (minified), FR37 (summary), FR38 (atomic), FR39 (directory creation), FR40 (version/oracle/metadata), FR41 (timestamps)

### Learnings from Previous Story

**From Story 4-8-build-state-manager-component (Status: done)**

- **Unified Interface Pattern**: StateManager provides unified access to both state and history via accessor methods - follow similar pattern for Writer
- **Atomic Write Implementation**: `WriteAtomic` helper exists in storage package - REUSE this pattern, don't recreate
- **Testing Pattern**: Table-driven tests with fixtures; use `slog.Default()` fallback for nil logger
- **Files Available**:
  - `internal/storage/state.go` - Has `OutputFile()` accessor that returns output file path
  - `internal/aggregator/models.go` - Has `Snapshot` struct for historical data
- **StateManager Methods to Use**:
  - `sm.OutputFile()` - Returns the output file path for writing
  - `sm.LoadHistory()` - Loads existing history for appending

[Source: docs/sprint-artifacts/4-8-build-state-manager-component.md#Dev-Agent-Record]

### Project Structure Notes

- **New Files**: `internal/models/output.go`, `internal/storage/writer.go`, `internal/storage/writer_test.go`
- **Package Alignment**: Per [Source: docs/architecture/project-structure.md]:
  - Models go in `internal/models/`
  - Writer goes in `internal/storage/`
- **Existing Dependencies**:
  - Import `internal/aggregator` for `AggregationResult`, `Snapshot`
  - Import `internal/config` for `Config`

### Testing Standards

- Follow table-driven test pattern per [Source: docs/architecture/testing-strategy.md#Test-Organization]
- Test both success and error paths
- Use temp directories for file write tests
- Verify JSON structure matches schema

### Smoke Test Guide

1. Run extraction with valid data (manual or integration test)
2. Verify all 3 files exist in output dir:
   - `switchboard-oracle-data.json` (formatted)
   - `switchboard-oracle-data.min.json` (compact)
   - `switchboard-summary.json` (no history)
3. Verify full JSON has 2-space indentation
4. Verify minified JSON has no whitespace/newlines
5. Verify summary has `top_protocols` (max 10) and NO `historical` key
6. Kill process mid-write (simulate), verify no corrupt files remain

### References

- [Source: docs/prd.md#FR35-FR41] - Output generation requirements
- [Source: docs/epics/epic-5-output-cli.md#story-51] - Story definition
- [Source: docs/sprint-artifacts/tech-spec-epic-5.md#Story-5.1] - Authoritative acceptance criteria
- [Source: docs/sprint-artifacts/tech-spec-epic-5.md#Data-Models-and-Contracts] - Output struct definitions
- [Source: docs/sprint-artifacts/tech-spec-epic-5.md#JSON-Schemas] - JSON schema contracts
- [Source: docs/architecture/implementation-patterns.md#atomic-file-writes] - Atomic write pattern
- [Source: docs/architecture/project-structure.md#project-structure] - File locations
- [Source: docs/architecture/testing-strategy.md#test-organization] - Test patterns
- [Source: docs/sprint-artifacts/4-8-build-state-manager-component.md#dev-agent-record] - Previous story reference

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/5-1-implement-output-file-generation.context.xml

### Agent Model Used
Bob (GPT-5)

### Debug Log References
- TBD — add links to build/test runs after implementation

### Completion Notes List
- Drafted story using epic-5 and tech-spec-epic-5 ACs; mapped tasks to ACs and tests
- Reused atomic write pattern from architecture doc and Story 4-8 learnings
- Captured continuity requirements for previous story outputs and review items

### File List
- internal/models/output.go (new)
- internal/storage/writer.go (new)
- internal/storage/writer_test.go (new)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-12-01 | SM Agent (Bob) | Initial story draft created from epic-5 and tech-spec-epic-5.md |
