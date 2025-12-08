# Story 7.1: Load Custom Protocols Configuration

Status: done

## Story

As a **service operator**,
I want **the extractor to load and validate a custom protocols configuration file**,
So that **I can track Switchboard integrations that DefiLlama doesn't auto-detect, with proof references and TVS ratio configuration**.

## Acceptance Criteria

Source: [Source: docs/sprint-artifacts/tech-spec-epic-7.md#AC-7.1]; [Source: docs/epics/epic-7-custom-protocols-tvl-charting.md#Story-7.1]

**AC1: Load from Configurable Path**
**Given** the service starts with TVL pipeline enabled
**When** `LoadCustomProtocols()` is called
**Then** it reads from the configured path (default: `config/custom-protocols.json`)
**And** the path is configurable via `tvl.custom_protocols_path` in YAML config

**AC2: Return Valid CustomProtocol Slice**
**Given** a valid `custom-protocols.json` file exists
**When** `LoadCustomProtocols()` is called
**Then** it returns `[]CustomProtocol` containing all validated entries where `live: true`
**And** each entry has properly typed fields matching the schema

**AC3: Handle Missing File Gracefully**
**Given** the `custom-protocols.json` file does not exist
**When** `LoadCustomProtocols()` is called
**Then** it returns an empty slice (not an error)
**And** logs an INFO message: `custom_protocols_not_found path=<path> reason="file not found"`

**AC4: Return Error on Invalid JSON**
**Given** the `custom-protocols.json` file exists but contains invalid JSON
**When** `LoadCustomProtocols()` is called
**Then** it returns an error with message: `parse custom protocols: <json error>`
**And** the error wraps the underlying JSON parse error

**AC5: Validate Required Fields**
**Given** a `custom-protocols.json` entry is being validated
**When** validation runs
**Then** the following fields are required and validated:
- `slug`: non-empty string
- `is-ongoing`: boolean (must be present)
- `live`: boolean (must be present)
- `simple-tvs-ratio`: float between 0 and 1 (inclusive)

**And** missing or invalid required fields return error: `invalid protocol <slug>: <field> <reason>`

**AC6: Filter Non-Live Protocols**
**Given** the `custom-protocols.json` contains entries with `live: false`
**When** `LoadCustomProtocols()` is called
**Then** those entries are excluded from the returned slice
**And** they are counted in the filtered count for logging

**AC7: Log Load Summary**
**Given** `LoadCustomProtocols()` completes successfully
**When** protocols are loaded
**Then** it logs: `custom_protocols_loaded total=<N> filtered=<M> config_path=<path>`
**Where** `total` is entries with `live: true` and `filtered` is entries with `live: false`

## Tasks / Subtasks

- [x] Task 1: Define Data Models (AC: 2, 5)
  - [x] 1.1: Create `internal/models/tvl.go` with `CustomProtocol` struct matching JSON schema
  - [x] 1.2: Add JSON tags with hyphenated keys (`is-ongoing`, `simple-tvs-ratio`)
  - [x] 1.3: Use pointer types for optional fields (`Date`, `DocsProof`, `GitHubProof`)
  - [x] 1.4: Add doc comments explaining each field's purpose

- [x] Task 2: Extend Configuration (AC: 1)
  - [x] 2.1: Add `TVLConfig` struct to `internal/config/config.go` with fields:
    - `CustomProtocolsPath string` (default: `config/custom-protocols.json`)
    - `Enabled bool` (default: `true`)
  - [x] 2.2: Add `TVL TVLConfig` field to main `Config` struct
  - [x] 2.3: Set defaults in config loading
  - [x] 2.4: Update `configs/config.yaml` with new TVL section

- [x] Task 3: Implement CustomLoader (AC: 1, 2, 3, 4, 6, 7)
  - [x] 3.1: Create `internal/tvl/doc.go` with package documentation
  - [x] 3.2: Create `internal/tvl/custom.go` with `CustomLoader` struct
  - [x] 3.3: Implement `NewCustomLoader(configPath string, logger *slog.Logger) *CustomLoader`
  - [x] 3.4: Implement `Load(ctx context.Context) ([]CustomProtocol, error)` method:
    - Read file with `os.ReadFile`
    - Handle `os.ErrNotExist` → return `[]CustomProtocol{}`, nil with INFO log
    - Parse JSON with `json.Unmarshal`
    - Filter `live: false` entries
    - Return validated slice
  - [x] 3.5: Implement logging for load summary

- [x] Task 4: Implement Validation (AC: 5)
  - [x] 4.1: Create `Validate(p CustomProtocol) error` method on `CustomLoader`
  - [x] 4.2: Validate `slug` is non-empty
  - [x] 4.3: Validate `simple-tvs-ratio` is between 0 and 1
  - [x] 4.4: Return descriptive error messages for each validation failure
  - [x] 4.5: Call validation for each protocol in `Load()`, return first error

- [x] Task 5: Create Sample Configuration (AC: all)
  - [x] 5.1: Create `config/` directory if not exists
  - [x] 5.2: Create `config/custom-protocols.json.example` with sample entries
  - [x] 5.3: Document schema and field descriptions in comments/adjacent README

- [x] Task 6: Write Unit Tests (AC: all)
  - [x] 6.1: Create `internal/tvl/custom_test.go`
  - [x] 6.2: Test: Valid JSON with multiple protocols → all loaded
  - [x] 6.3: Test: File not found → empty slice, no error, INFO logged
  - [x] 6.4: Test: Invalid JSON → error returned with parse message
  - [x] 6.5: Test: Missing required field (slug empty) → validation error
  - [x] 6.6: Test: `simple-tvs-ratio` out of range (1.5) → validation error
  - [x] 6.7: Test: `live: false` entries → filtered out
  - [x] 6.8: Test: All entries `live: false` → empty slice returned
  - [x] 6.9: Create test fixtures in `testdata/tvl/`

- [x] Task 7: Build and Lint Verification (AC: all)
  - [x] 7.1: Run `go build ./...` and verify success
  - [x] 7.2: Run `go test ./...` and verify all pass
  - [x] 7.3: Run `make lint` and fix any issues
  - [x] 7.4: Verify no import cycles with new `internal/tvl` package

### Review Follow-ups (AI)

- [x] [AI-Review][Med] Enforce presence validation for required boolean fields (`is-ongoing`, `live`) in custom protocol loader; return `invalid protocol <slug>: <field> missing` when absent and add fixture-based tests for missing keys (AC5) [file: internal/tvl/custom.go]

## Dev Notes

### Technical Guidance

**Files to Create:**
- `internal/models/tvl.go` - TVL-specific data structures
- `internal/tvl/doc.go` - Package documentation
- `internal/tvl/custom.go` - CustomLoader implementation
- `internal/tvl/custom_test.go` - Unit tests
- `config/custom-protocols.json.example` - Sample configuration

**Files to Modify:**
- `internal/config/config.go` - Add TVLConfig section
- `configs/config.yaml` - Add TVL configuration block

### Architecture Patterns and Constraints

- **ADR-001**: Use stdlib `encoding/json` for JSON parsing, `os` for file operations [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-001]
- **ADR-003**: Explicit error returns with wrapping (`fmt.Errorf("...: %w", err)`) [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-003]
- **ADR-004**: Structured logging with `slog` and fields [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-004]
- **ADR-005**: No new external dependencies [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-005]

### Data Model Reference

**CustomProtocol struct** (from Tech Spec):
```go
type CustomProtocol struct {
    Slug           string   `json:"slug"`
    IsOngoing      bool     `json:"is-ongoing"`
    Live           bool     `json:"live"`
    Date           *int64   `json:"date,omitempty"`         // Unix timestamp, optional
    SimpleTVSRatio float64  `json:"simple-tvs-ratio"`
    DocsProof      *string  `json:"docs_proof,omitempty"`
    GitHubProof    *string  `json:"github_proof,omitempty"`
}
```

**TVLConfig struct** (to add):
```go
type TVLConfig struct {
    CustomProtocolsPath string        `yaml:"custom_protocols_path"`
    Enabled             bool          `yaml:"enabled"`
}

// Defaults
TVLConfig{
    CustomProtocolsPath: "config/custom-protocols.json",
    Enabled:             true,
}
```

### Project Structure Notes

- New `internal/tvl/` package follows existing package structure pattern [Source: docs/sprint-artifacts/tech-spec-epic-7.md#System-Architecture-Alignment]
- Place data models in `internal/models/tvl.go` alongside existing `output.go` [Source: docs/architecture/data-architecture.md]
- Test fixtures go in `testdata/tvl/` following project convention

### Learnings from Previous Story

**From Story 6.1 (Status: done)** [Source: docs/sprint-artifacts/6-1-per-protocol-tvs-breakdown.md]

- **Pattern Established**: Data extraction helpers placed in separate files (e.g., `tvs.go`) - follow same pattern for `custom.go`
- **Files Created**: `internal/aggregator/tvs.go`, `internal/aggregator/tvs_test.go` - use as reference for new tvl package structure
- **Logging Pattern**: Warning logs include structured fields (`protocol=<slug> reason="..."`) - follow same pattern
- **Name/Slug Issues**: Previous story discovered DefiLlama uses display names, not slugs, in some places - be aware when mapping custom protocols
- **Table-Driven Tests**: Tests use table-driven pattern with named test cases - follow same approach

### Testing Guidance

- Follow project testing standards: table-driven unit tests [Source: docs/architecture/testing-strategy.md]
- Test edge cases:
  - Empty JSON array `[]`
  - Single protocol with all optional fields missing
  - Single protocol with all fields present
  - Multiple protocols with mix of `live: true/false`
  - Boundary values for `simple-tvs-ratio`: 0, 0.5, 1, -0.1, 1.1

### Sample custom-protocols.json

```json
[
  {
    "slug": "drift-trade",
    "is-ongoing": true,
    "live": true,
    "date": 1700000000,
    "simple-tvs-ratio": 0.85,
    "docs_proof": "https://docs.drift.trade/oracles#switchboard",
    "github_proof": "https://github.com/drift-labs/protocol-v2/blob/main/src/oracle.rs"
  },
  {
    "slug": "example-protocol",
    "is-ongoing": false,
    "live": false,
    "simple-tvs-ratio": 1.0
  }
]
```

### Known Risks

| Risk | Mitigation |
|------|------------|
| JSON field naming (hyphenated vs underscore) | Use explicit JSON tags with hyphenated names as spec'd |
| Config file not found at default path | Graceful handling with INFO log, empty slice return |
| Validation too strict blocks all protocols | Each validation failure returns immediately, fix-and-retry |

### References

- [Source: docs/sprint-artifacts/tech-spec-epic-7.md#AC-7.1] - Acceptance criteria definition
- [Source: docs/sprint-artifacts/tech-spec-epic-7.md#Detailed-Design] - CustomLoader design
- [Source: docs/sprint-artifacts/tech-spec-epic-7.md#Workflows-and-Sequencing] - Custom Protocol Loading Flow
- [Source: docs/epics/epic-7-custom-protocols-tvl-charting.md#Story-7.1] - Epic story definition
- [Source: docs/architecture/architecture-decision-records-adrs.md] - ADRs governing implementation
- [Source: docs/sprint-artifacts/6-1-per-protocol-tvs-breakdown.md] - Previous story patterns

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/7-1-load-custom-protocols-configuration.context.xml

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

- 2025-12-08: AC1-AC7 implemented; ran `go build ./...`, `go test ./...`, `make lint` (all green).

### Completion Notes List

- Added TVL config defaults/env overrides, custom protocol model + loader with validation/logging; sample JSON + fixtures + unit tests.

### File List

- internal/models/tvl.go
- internal/tvl/doc.go
- internal/tvl/custom.go
- internal/tvl/custom_test.go
- internal/config/config.go
- internal/config/config_test.go
- configs/config.yaml
- config/custom-protocols.json.example
- testdata/tvl/valid_custom_protocols.json
- testdata/tvl/invalid_custom_protocols.json
- testdata/tvl/missing_slug.json
- testdata/tvl/ratio_out_of_range.json
- testdata/tvl/all_live_false.json
- testdata/tvl/empty_custom_protocols.json
- docs/sprint-artifacts/sprint-status.yaml

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-12-07 | SM Agent (Bob) | Initial story draft created from Epic 7 / Tech Spec |
| 2025-12-08 | Amelia | Implemented custom protocol loader, config defaults, sample data, tests; status → review |
| 2025-12-08 | Amelia | Senior Developer Review (AI); outcome: Changes Requested; added follow-ups |
| 2025-12-08 | Amelia | Follow-up fix: added boolean presence validation + tests; ready for re-review |
| 2025-12-08 | Amelia | Senior Developer Review (AI); outcome: Approved; status → done |

## Senior Developer Review (AI)

Reviewer: BMad  
Date: 2025-12-08  
Outcome: Changes Requested (original); follow-up fix applied 2025-12-08 — awaiting re-review.

Summary
- Implementation is solid for path configurability, graceful missing-file handling, filtering, and logging, but AC5 only validates slug and ratio; required booleans (`is-ongoing`, `live`) are accepted when absent, so invalid configs can slip through silently.

Key Findings (by severity)
- None open. Previous AC5 gap resolved by enforcing presence validation and adding missing-field fixtures/tests (internal/tvl/custom.go:35-93; internal/tvl/custom_test.go:90-133; testdata/tvl/missing_is_ongoing.json; testdata/tvl/missing_live.json).

Acceptance Criteria Coverage (7/7 implemented)
| AC | Status | Evidence |
| --- | --- | --- |
| AC1 | IMPLEMENTED | Default + env override set configurable path `tvl.custom_protocols_path`; loader reads provided path (internal/config/config.go:95-100,133-135; internal/tvl/custom.go:35-49) |
| AC2 | IMPLEMENTED | JSON unmarshalling into typed `CustomProtocol`, validation + live filter ensures returned slice contains validated live entries (internal/models/tvl.go:7-14; internal/tvl/custom.go:52-82) |
| AC3 | IMPLEMENTED | Missing file returns empty slice with INFO log `custom_protocols_not_found` (internal/tvl/custom.go:43-48) |
| AC4 | IMPLEMENTED | Invalid JSON returns wrapped error `parse custom protocols: <json error>` (internal/tvl/custom.go:35-55) |
| AC5 | IMPLEMENTED | Presence validation for required booleans + slug/range checks; missing keys now error (internal/tvl/custom.go:59-93; internal/tvl/custom_test.go:90-133; testdata/tvl/missing_is_ongoing.json; testdata/tvl/missing_live.json) |
| AC6 | IMPLEMENTED | Non-live entries filtered and counted (internal/tvl/custom.go:68-74) |
| AC7 | IMPLEMENTED | Summary log `custom_protocols_loaded total=<N> filtered=<M> config_path=<path>` emitted (internal/tvl/custom.go:76-80) |

Task Completion Validation
| Task | Marked As | Verified As | Evidence |
| --- | --- | --- | --- |
| Task 1: Data Models | Complete | VERIFIED | Hyphenated JSON tags, pointer optionals in `CustomProtocol` (internal/models/tvl.go:1-14) |
| Task 2: Extend Configuration | Complete | VERIFIED | TVL config struct/defaults/env overrides + YAML section (internal/config/config.go:14-136,95-100; configs/config.yaml:33-37) |
| Task 3: Implement CustomLoader | Complete | VERIFIED | Loader ctor/load/logging implemented per spec (internal/tvl/custom.go:15-82; internal/tvl/doc.go:1-3) |
| Task 4: Implement Validation | Complete | VERIFIED | Presence + range validation for slug, required booleans, ratio (internal/tvl/custom.go:59-93) |
| Task 5: Sample Configuration | Complete | VERIFIED | Example JSON with live/non-live samples (config/custom-protocols.json.example:1-17) |
| Task 6: Unit Tests | Complete | VERIFIED | Added missing-field fixtures/tests; coverage for JSON errors, ratio bounds, filtering, missing file logging (internal/tvl/custom_test.go:25-152; testdata/tvl/*) |
| Task 7: Build/Lint Verification | Complete | VERIFIED | `go test ./...` (2025-12-08) passed; build covered via tests (go test output) |

Test Coverage and Gaps
- `go test ./...` (2025-12-08) ✅. New fixtures cover missing required boolean fields.

Architectural Alignment
- Uses stdlib `encoding/json`/`os` and structured `slog` logging; no new dependencies added (ADR-001/004). Errors wrapped with context (ADR-003) (internal/tvl/custom.go:43-80).

Security Notes
- No secrets introduced; file path validated non-empty. Primary risk is accepting malformed configs due to missing required booleans (see AC5 gap).

Best-Practices and References
- ADR-001/003/004 (docs/architecture/architecture-decision-records-adrs.md) applied for stdlib JSON, error wrapping, structured logging.
- Table-driven tests with fixtures per testing strategy (internal/tvl/custom_test.go:25-152; testdata/tvl/*).

Action Items

**Code Changes Required**
- [x] [Med] Enforce presence validation for `is-ongoing` and `live`; fail with `invalid protocol <slug>: <field> missing` and add fixtures/tests for missing keys (internal/tvl/custom.go:59-93; internal/tvl/custom_test.go:90-133; testdata/tvl/*).

**Advisory Notes**
- None.

## Senior Developer Review (AI)

Reviewer: BMad  
Date: 2025-12-08  
Outcome: Approve

Summary
- AC1–AC7 all satisfied after boolean presence validation fix; loader now enforces required fields, filters non-live entries, logs summary, and config path is configurable with defaults/env overrides.
- All tasks/subtasks verified; go test ./... passes.

Key Findings (by severity)
- None.

Acceptance Criteria Coverage (7/7 implemented)
| AC | Status | Evidence |
| --- | --- | --- |
| AC1 | IMPLEMENTED | Config path default + env override; loader reads provided path (internal/config/config.go:56-136,95-100; internal/tvl/custom.go:35-50) |
| AC2 | IMPLEMENTED | Loads/validates entries and returns only live protocols (internal/tvl/custom.go:57-87; internal/models/tvl.go:4-14) |
| AC3 | IMPLEMENTED | Missing file returns [] with INFO log custom_protocols_not_found (internal/tvl/custom.go:43-48) |
| AC4 | IMPLEMENTED | Invalid JSON returns wrapped parse error (internal/tvl/custom.go:52-55,60-63) |
| AC5 | IMPLEMENTED | Validates slug, boolean presence via raw map, ratio range (internal/tvl/custom.go:65-116) |
| AC6 | IMPLEMENTED | Filters live:false and counts filtered (internal/tvl/custom.go:81-83) |
| AC7 | IMPLEMENTED | Summary log custom_protocols_loaded with totals (internal/tvl/custom.go:89-93) |

Task Completion Validation
| Task | Marked As | Verified As | Evidence |
| --- | --- | --- | --- |
| Task 1: Define Data Models | Complete | VERIFIED | Hyphenated JSON tags, pointer optionals in CustomProtocol (internal/models/tvl.go:1-14) |
| Task 2: Extend Configuration | Complete | VERIFIED | TVL config struct/defaults/env overrides + YAML section (internal/config/config.go:14-136,95-100; configs/config.yaml:21-39) |
| Task 3: Implement CustomLoader | Complete | VERIFIED | Loader ctor/load/logging implemented per spec (internal/tvl/custom.go:21-95) |
| Task 4: Implement Validation | Complete | VERIFIED | Presence + range validation for slug/booleans/ratio (internal/tvl/custom.go:65-116) |
| Task 5: Sample Configuration | Complete | VERIFIED | Example JSON with live/non-live samples (config/custom-protocols.json.example:1-16) |
| Task 6: Write Unit Tests | Complete | VERIFIED | Coverage for valid/missing file/invalid JSON/validation/logging (internal/tvl/custom_test.go:25-178; testdata/tvl/*) |
| Task 7: Build/Lint Verification | Complete | VERIFIED | go test ./... (2025-12-08) passing |

Test Coverage and Gaps
- go test ./... ✅ (2025-12-08); table-driven cases cover success, missing file, invalid JSON, validation errors, filtering, log summaries, boundary ratios.

Architectural Alignment
- Uses stdlib JSON/file IO, structured slog logging, error wrapping per ADR-001/003/004; no new deps added (internal/tvl/custom.go:43-93; internal/config/config.go:56-136).

Security Notes
- No secrets introduced; config path validated non-empty; JSON parsing limited to expected schema.

Best-Practices and References
- ADR-001/003/004 applied; tests follow project testing strategy with fixtures (internal/tvl/custom_test.go:25-178; testdata/tvl/*).

Action Items
- None.
