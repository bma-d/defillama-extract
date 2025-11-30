# Story 1.2: Implement Configuration Loading from YAML

Status: done

## Story

As a **developer**,
I want **configuration loaded from a YAML file with sensible defaults**,
so that **the service behavior can be customized without code changes**.

## Acceptance Criteria

1. **Given** a YAML configuration file at the specified path **When** `config.Load(path)` is called **Then** configuration is loaded into a typed `Config` struct with all sections:
   - `oracle`: name (default: "Switchboard"), website, documentation URL
   - `api`: base URLs (oracles, protocols), timeout (default: 30s), max_retries (default: 3), retry_delay (default: 1s)
   - `output`: directory (default: "data"), file names (full, min, summary, state)
   - `scheduler`: interval (default: 2h), start_immediately flag (default: true)
   - `logging`: level (default: "info"), format (default: "json")

2. **Given** a YAML file with missing optional fields **When** configuration is loaded **Then** default values are applied for missing fields and the Config struct is fully populated

3. **Given** a YAML file with all fields specified **When** configuration is loaded **Then** all YAML values override defaults

4. **Given** no config file exists at the specified path **When** `config.Load(nonexistent.yaml)` is called **Then** an error is returned with message containing "config file not found" or "no such file"

5. **Given** an invalid/malformed YAML file **When** configuration is loaded **Then** a parse error is returned with descriptive message

6. **Given** a Config struct with invalid values (e.g., negative timeout, empty oracle name, invalid log level) **When** `config.Validate()` is called **Then** an error is returned describing the first validation failure

7. **Given** a Config struct with valid values **When** `config.Validate()` is called **Then** nil is returned (no error)

## Tasks / Subtasks

- [x] Task 1: Create Config struct with YAML tags (AC: 1)
  - [x] 1.1: Create `internal/config/config.go` with main Config struct
  - [x] 1.2: Create `OracleConfig` struct with `name`, `website`, `documentation` fields
  - [x] 1.3: Create `APIConfig` struct with `oracles_url`, `protocols_url`, `timeout`, `max_retries`, `retry_delay` fields
  - [x] 1.4: Create `OutputConfig` struct with `directory`, `full_file`, `min_file`, `summary_file`, `state_file` fields
  - [x] 1.5: Create `SchedulerConfig` struct with `interval`, `start_immediately` fields
  - [x] 1.6: Create `LoggingConfig` struct with `level`, `format` fields
  - [x] 1.7: Add appropriate `yaml` struct tags to all fields

- [x] Task 2: Implement Load function with defaults (AC: 1, 2, 3, 4, 5)
  - [x] 2.1: Implement `defaultConfig()` function returning Config with all defaults
  - [x] 2.2: Implement `Load(path string) (*Config, error)` that reads YAML file
  - [x] 2.3: Handle file not found error with clear message
  - [x] 2.4: Handle YAML parse errors with descriptive message
  - [x] 2.5: Merge YAML values over defaults (YAML overrides defaults)

- [x] Task 3: Implement Validate method (AC: 6, 7)
  - [x] 3.1: Implement `(c *Config) Validate() error` method
  - [x] 3.2: Validate `oracle.name` is not empty
  - [x] 3.3: Validate `api.timeout` is positive
  - [x] 3.4: Validate `api.max_retries` is non-negative
  - [x] 3.5: Validate `logging.level` is one of: debug, info, warn, error
  - [x] 3.6: Validate `logging.format` is one of: json, text
  - [x] 3.7: Validate `scheduler.interval` is positive

- [x] Task 4: Create default config.yaml file (AC: 1)
  - [x] 4.1: Create `configs/config.yaml` with documented defaults
  - [x] 4.2: Include comments explaining each configuration section

- [x] Task 5: Add gopkg.in/yaml.v3 dependency (AC: 1)
  - [x] 5.1: Run `go get gopkg.in/yaml.v3`
  - [x] 5.2: Verify `go.sum` is updated

- [x] Task 6: Write unit tests (AC: all)
  - [x] 6.1: Test `Load` with valid YAML file (all fields)
  - [x] 6.2: Test `Load` with minimal YAML file (defaults applied)
  - [x] 6.3: Test `Load` with missing file (error returned)
  - [x] 6.4: Test `Load` with invalid YAML (parse error)
  - [x] 6.5: Test `Validate` with valid config
  - [x] 6.6: Test `Validate` rejects negative timeout
  - [x] 6.7: Test `Validate` rejects empty oracle name
  - [x] 6.8: Test `Validate` rejects invalid log level
  - [x] 6.9: Use table-driven tests per Go idioms

- [x] Task 7: Verification (AC: all)
  - [x] 7.1: Run `go build ./...` and verify success
  - [x] 7.2: Run `go test ./internal/config/...` and verify all pass
  - [x] 7.3: Run `make lint` and verify no errors

## Dev Notes

### Technical Guidance

- **Package Location:** `internal/config/config.go`
- **Dependency:** `gopkg.in/yaml.v3` (only external dependency per ADR-005)
- **Go Version:** Go 1.21+ required for `log/slog` (Story 1.4), but config has no slog dependency
- **Pattern:** Load function returns pointer to avoid copying, Validate is a method on Config

### Config Struct Reference (from tech-spec-epic-1.md)

```go
type Config struct {
    Oracle    OracleConfig    `yaml:"oracle"`
    API       APIConfig       `yaml:"api"`
    Output    OutputConfig    `yaml:"output"`
    Scheduler SchedulerConfig `yaml:"scheduler"`
    Logging   LoggingConfig   `yaml:"logging"`
}

type OracleConfig struct {
    Name          string `yaml:"name"`           // Default: "Switchboard"
    Website       string `yaml:"website"`
    Documentation string `yaml:"documentation"`
}

type APIConfig struct {
    OraclesURL   string        `yaml:"oracles_url"`    // Default: "https://api.llama.fi/oracles"
    ProtocolsURL string        `yaml:"protocols_url"`  // Default: "https://api.llama.fi/lite/protocols2?b=2"
    Timeout      time.Duration `yaml:"timeout"`        // Default: 30s
    MaxRetries   int           `yaml:"max_retries"`    // Default: 3
    RetryDelay   time.Duration `yaml:"retry_delay"`    // Default: 1s
}

type OutputConfig struct {
    Directory   string `yaml:"directory"`      // Default: "data"
    FullFile    string `yaml:"full_file"`      // Default: "switchboard-oracle-data.json"
    MinFile     string `yaml:"min_file"`       // Default: "switchboard-oracle-data.min.json"
    SummaryFile string `yaml:"summary_file"`   // Default: "switchboard-summary.json"
    StateFile   string `yaml:"state_file"`     // Default: "state.json"
}

type SchedulerConfig struct {
    Interval         time.Duration `yaml:"interval"`          // Default: 2h
    StartImmediately bool          `yaml:"start_immediately"` // Default: true
}

type LoggingConfig struct {
    Level  string `yaml:"level"`  // Default: "info"
    Format string `yaml:"format"` // Default: "json"
}
```

### Default Values Summary

| Config Path | Default Value |
|-------------|---------------|
| `oracle.name` | "Switchboard" |
| `api.oracles_url` | "https://api.llama.fi/oracles" |
| `api.protocols_url` | "https://api.llama.fi/lite/protocols2?b=2" |
| `api.timeout` | 30s |
| `api.max_retries` | 3 |
| `api.retry_delay` | 1s |
| `output.directory` | "data" |
| `output.full_file` | "switchboard-oracle-data.json" |
| `output.min_file` | "switchboard-oracle-data.min.json" |
| `output.summary_file` | "switchboard-summary.json" |
| `output.state_file` | "state.json" |
| `scheduler.interval` | 2h |
| `scheduler.start_immediately` | true |
| `logging.level` | "info" |
| `logging.format` | "json" |

### Validation Rules

| Field | Rule |
|-------|------|
| `oracle.name` | Must not be empty |
| `api.timeout` | Must be > 0 |
| `api.max_retries` | Must be >= 0 |
| `api.retry_delay` | Must be >= 0 |
| `scheduler.interval` | Must be > 0 |
| `logging.level` | Must be one of: "debug", "info", "warn", "error" |
| `logging.format` | Must be one of: "json", "text" |

### Project Structure Notes

Per Story 1.1, the `internal/config/` directory already exists with `doc.go` placeholder. Replace with actual implementation:
- Replace `internal/config/doc.go` with `internal/config/config.go`
- Add `internal/config/config_test.go` for unit tests

### Learnings from Previous Story

**From Story 1-1 (Status: done)**

- **Project Layout Established:** Standard Go layout with `cmd/extractor/`, `internal/` packages, `configs/` for default config
- **Build Tooling Ready:** Makefile with `build`, `test`, `lint` targets verified working
- **Go Version:** Project uses Go 1.23 (aligned with golangci-lint compatibility)
- **Placeholder Pattern:** Package directories have `doc.go` placeholders - replace with actual implementation files

[Source: docs/sprint-artifacts/1-1-initialize-go-module-and-project-structure.md#Dev-Agent-Record]

### References

- [Source: docs/sprint-artifacts/tech-spec-epic-1.md#detailed-design] - Config struct definitions and defaults
- [Source: docs/epics.md#story-12] - Original story definition with acceptance criteria
- [Source: docs/prd.md#configuration] - FR49, FR51, FR52 requirements
- [Source: docs/architecture/testing-strategy.md] - Test organization and coverage requirements
- [Source: docs/architecture/architecture-decision-records-adrs.md#adr-005] - Minimal external dependencies (yaml.v3 only)

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/1-2-implement-configuration-loading-from-yaml.context.xml

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References
 - Planned against ACs; built Config structs and defaults; Load merges YAML over defaults with specific error messaging; Validate enforces required fields and enums; authored table-driven tests for load/validate success and error cases; executed build, unit tests, and lint (all passing).

### Completion Notes List
 - Implemented YAML-backed configuration with defaults, validation, documented template, and coverage for success/error paths; project builds, tests, and lint all pass.

### File List
 - internal/config/config.go
 - internal/config/config_test.go
 - internal/config/testdata/config_all.yaml
 - internal/config/testdata/config_minimal.yaml
 - internal/config/testdata/config_invalid.yaml
 - configs/config.yaml
 - go.mod
 - go.sum

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epics.md and tech-spec-epic-1.md |
| 2025-11-30 | Amelia (Dev Agent) | Implemented config loading, defaults, validation, tests, defaults template; moved story to review |
| 2025-11-30 | Amelia (Dev Agent) | Senior Developer Review (AI) appended; outcome: Approve |
| 2025-11-30 | Amelia (Dev Agent) | Updated go.mod to mark yaml.v3 as direct dependency |

## Senior Developer Review (AI)

- Reviewer: BMad
- Date: 2025-11-30
- Outcome: Approve

### Summary
- Config loading, defaults, validation, and error handling implemented per AC1–AC7; build, tests, and lint all pass.

### Key Findings
- LOW: go.mod lists `gopkg.in/yaml.v3` as `// indirect` even though it is imported directly; mark as direct to avoid accidental removal (go.mod).

### Acceptance Criteria Coverage (7/7 implemented)
| AC | Description | Status | Evidence |
|----|-------------|--------|----------|
| AC1 | Load YAML into typed Config with all sections | Implemented | internal/config/config.go:13-52,87-108; internal/config/config_test.go:10-41 |
| AC2 | Defaults applied when fields missing | Implemented | internal/config/config.go:54-85; internal/config/config_test.go:44-63 |
| AC3 | YAML overrides defaults | Implemented | internal/config/config.go:87-101; internal/config/config_test.go:10-41 |
| AC4 | Missing file returns not-found error | Implemented | internal/config/config.go:91-97; internal/config/config_test.go:66-73 |
| AC5 | Invalid YAML surfaces parse error | Implemented | internal/config/config.go:99-101; internal/config/config_test.go:76-84 |
| AC6 | Validate rejects invalid values | Implemented | internal/config/config.go:110-145; internal/config/config_test.go:86-137 |
| AC7 | Validate passes for valid config | Implemented | internal/config/config.go:146; internal/config/config_test.go:141-146 |

### Task Completion Validation
| Task | Marked As | Verified As | Evidence |
|------|-----------|-------------|----------|
| Task 1: Config structs with YAML tags | Completed | Verified | internal/config/config.go:13-52 |
| Task 2: Load with defaults, errors | Completed | Verified | internal/config/config.go:54-108 |
| Task 3: Validate method | Completed | Verified | internal/config/config.go:110-146 |
| Task 4: Default config.yaml | Completed | Verified | configs/config.yaml:1-43 |
| Task 5: Add yaml.v3 dependency | Completed | Verified | go.mod:1-3; go.sum |
| Task 6: Unit tests | Completed | Verified | internal/config/config_test.go:10-146; internal/config/testdata/*.yaml |
| Task 7: Build/test/lint run | Completed | Verified | go build ./...; go test ./...; make lint (2025-11-30) |

### Test Coverage and Gaps
- go test ./... (2025-11-30): pass; config table-driven coverage for success and error paths.

### Architectural Alignment
- Adheres to ADR-005 minimal deps; structs and defaults match tech-spec-epic-1.md; validation follows constraints in story context.

### Security Notes
- No secrets handled; validation ensures durations and enums bounded; dependency surface minimal.

### Best-Practices and References
- Go 1.23; config parsing via yaml.v3; validation uses explicit errors per ADR-003.

### Action Items
- [x] [Low] Mark `gopkg.in/yaml.v3` as a direct dependency in go.mod to reflect direct import and prevent pruning (done 2025-11-30).
