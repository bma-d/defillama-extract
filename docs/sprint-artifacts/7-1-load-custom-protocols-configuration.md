# Story 7.1: Load Custom Protocols Configuration

Status: drafted

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

- [ ] Task 1: Define Data Models (AC: 2, 5)
  - [ ] 1.1: Create `internal/models/tvl.go` with `CustomProtocol` struct matching JSON schema
  - [ ] 1.2: Add JSON tags with hyphenated keys (`is-ongoing`, `simple-tvs-ratio`)
  - [ ] 1.3: Use pointer types for optional fields (`Date`, `DocsProof`, `GitHubProof`)
  - [ ] 1.4: Add doc comments explaining each field's purpose

- [ ] Task 2: Extend Configuration (AC: 1)
  - [ ] 2.1: Add `TVLConfig` struct to `internal/config/config.go` with fields:
    - `CustomProtocolsPath string` (default: `config/custom-protocols.json`)
    - `Enabled bool` (default: `true`)
  - [ ] 2.2: Add `TVL TVLConfig` field to main `Config` struct
  - [ ] 2.3: Set defaults in config loading
  - [ ] 2.4: Update `configs/config.yaml` with new TVL section

- [ ] Task 3: Implement CustomLoader (AC: 1, 2, 3, 4, 6, 7)
  - [ ] 3.1: Create `internal/tvl/doc.go` with package documentation
  - [ ] 3.2: Create `internal/tvl/custom.go` with `CustomLoader` struct
  - [ ] 3.3: Implement `NewCustomLoader(configPath string, logger *slog.Logger) *CustomLoader`
  - [ ] 3.4: Implement `Load(ctx context.Context) ([]CustomProtocol, error)` method:
    - Read file with `os.ReadFile`
    - Handle `os.ErrNotExist` → return `[]CustomProtocol{}`, nil with INFO log
    - Parse JSON with `json.Unmarshal`
    - Filter `live: false` entries
    - Return validated slice
  - [ ] 3.5: Implement logging for load summary

- [ ] Task 4: Implement Validation (AC: 5)
  - [ ] 4.1: Create `Validate(p CustomProtocol) error` method on `CustomLoader`
  - [ ] 4.2: Validate `slug` is non-empty
  - [ ] 4.3: Validate `simple-tvs-ratio` is between 0 and 1
  - [ ] 4.4: Return descriptive error messages for each validation failure
  - [ ] 4.5: Call validation for each protocol in `Load()`, return first error

- [ ] Task 5: Create Sample Configuration (AC: all)
  - [ ] 5.1: Create `config/` directory if not exists
  - [ ] 5.2: Create `config/custom-protocols.json.example` with sample entries
  - [ ] 5.3: Document schema and field descriptions in comments/adjacent README

- [ ] Task 6: Write Unit Tests (AC: all)
  - [ ] 6.1: Create `internal/tvl/custom_test.go`
  - [ ] 6.2: Test: Valid JSON with multiple protocols → all loaded
  - [ ] 6.3: Test: File not found → empty slice, no error, INFO logged
  - [ ] 6.4: Test: Invalid JSON → error returned with parse message
  - [ ] 6.5: Test: Missing required field (slug empty) → validation error
  - [ ] 6.6: Test: `simple-tvs-ratio` out of range (1.5) → validation error
  - [ ] 6.7: Test: `live: false` entries → filtered out
  - [ ] 6.8: Test: All entries `live: false` → empty slice returned
  - [ ] 6.9: Create test fixtures in `testdata/tvl/`

- [ ] Task 7: Build and Lint Verification (AC: all)
  - [ ] 7.1: Run `go build ./...` and verify success
  - [ ] 7.2: Run `go test ./...` and verify all pass
  - [ ] 7.3: Run `make lint` and fix any issues
  - [ ] 7.4: Verify no import cycles with new `internal/tvl` package

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

<!-- Path(s) to story context XML will be added here by context workflow -->

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-12-07 | SM Agent (Bob) | Initial story draft created from Epic 7 / Tech Spec |
