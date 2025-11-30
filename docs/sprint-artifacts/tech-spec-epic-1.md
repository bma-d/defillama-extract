# Epic Technical Specification: Foundation

Date: 2025-11-29
Author: BMad
Epic ID: 1
Status: Complete

---

## Overview

Epic 1 establishes the foundational infrastructure for the defillama-extract CLI tool. This epic delivers a properly structured Go project with working configuration loading (YAML + environment variable overrides) and structured logging via Go's `slog` package. These foundational components are prerequisites for all subsequent epics - the API client, aggregation pipeline, state management, and CLI all depend on configuration and logging being in place.

The foundation follows Go idioms and standard project layout, prioritizing simplicity and minimal dependencies per ADR-001 (stdlib over frameworks) and ADR-005 (minimal external dependencies). The only external dependency introduced is `gopkg.in/yaml.v3` for configuration parsing.

## Objectives and Scope

**In Scope:**
- Initialize Go module with standard project layout (`cmd/`, `internal/` structure)
- Implement typed configuration loading from YAML files with sensible defaults
- Implement environment variable overrides for deployment flexibility
- Implement structured logging with configurable format (JSON/text) and level (debug/info/warn/error)
- Create Makefile with build, test, lint targets
- Create `.gitignore` and `.golangci.yml` for development tooling

**Out of Scope:**
- API client implementation (Epic 2)
- Data processing logic (Epic 3)
- State/history management (Epic 4)
- Output generation and CLI modes (Epic 5)
- Any HTTP endpoints or network operations

## System Architecture Alignment

This epic maps to the following architectural components per the FR-to-Architecture mapping:

| Component | Package | Purpose |
|-----------|---------|---------|
| Configuration | `internal/config` | Config struct, YAML loading, env overrides, validation |
| Logging | `cmd/extractor` or `internal/logging` | slog initialization with handler configuration |
| Entry Point | `cmd/extractor/main.go` | Minimal main.go that compiles |

**Architectural Decisions Applied:**
- **ADR-001:** Use `flag` (stdlib) for future CLI parsing, minimal framework usage
- **ADR-004:** Structured logging with `log/slog` (stdlib, Go 1.21+)
- **ADR-005:** Single external dependency (`gopkg.in/yaml.v3`)

## Detailed Design

### Services and Modules

| Module | Location | Responsibility | Dependencies |
|--------|----------|----------------|--------------|
| **Config** | `internal/config/config.go` | Load YAML config, apply env overrides, validate settings | `gopkg.in/yaml.v3`, stdlib `os`, `time` |
| **Logging** | `cmd/extractor/main.go` | Initialize slog handler based on config | stdlib `log/slog` |
| **Entry Point** | `cmd/extractor/main.go` | Bootstrap application, wire components | `internal/config` |

**Directory Structure Created:**
```
defillama-extract/
├── cmd/
│   └── extractor/
│       └── main.go           # Entry point (minimal, compiles)
├── internal/
│   ├── config/
│   │   └── config.go         # Config loading and validation
│   ├── models/               # (empty, placeholder for Epic 2+)
│   ├── api/                  # (empty, placeholder for Epic 2)
│   ├── aggregator/           # (empty, placeholder for Epic 3)
│   └── storage/              # (empty, placeholder for Epic 4)
├── configs/
│   └── config.yaml           # Default configuration file
├── go.mod
├── go.sum
├── Makefile
├── .gitignore
└── .golangci.yml
```

### Data Models and Contracts

**Config Struct** (`internal/config/config.go`):

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
    Directory    string `yaml:"directory"`      // Default: "data"
    FullFile     string `yaml:"full_file"`      // Default: "switchboard-oracle-data.json"
    MinFile      string `yaml:"min_file"`       // Default: "switchboard-oracle-data.min.json"
    SummaryFile  string `yaml:"summary_file"`   // Default: "switchboard-summary.json"
    StateFile    string `yaml:"state_file"`     // Default: "state.json"
}

type SchedulerConfig struct {
    Interval         time.Duration `yaml:"interval"`          // Default: 2h
    StartImmediately bool          `yaml:"start_immediately"` // Default: true
}

type LoggingConfig struct {
    Level  string `yaml:"level"`  // Default: "info" (debug|info|warn|error)
    Format string `yaml:"format"` // Default: "json" (json|text)
}
```

### APIs and Interfaces

**Config Package Public Interface:**

```go
// Load reads configuration from YAML file at path, applies defaults,
// applies environment variable overrides, and validates the result.
func Load(path string) (*Config, error)

// Validate checks that all configuration values are valid.
// Returns error describing first validation failure, or nil if valid.
func (c *Config) Validate() error
```

**Environment Variable Mapping:**

| Env Var | Config Field | Type | Notes |
|---------|--------------|------|-------|
| `ORACLE_NAME` | `oracle.name` | string | Override oracle name |
| `OUTPUT_DIR` | `output.directory` | string | Override output directory |
| `LOG_LEVEL` | `logging.level` | string | debug/info/warn/error |
| `LOG_FORMAT` | `logging.format` | string | json/text |
| `API_TIMEOUT` | `api.timeout` | duration | e.g., "45s" |
| `SCHEDULER_INTERVAL` | `scheduler.interval` | duration | e.g., "1h30m" |

### Workflows and Sequencing

**Application Startup Sequence (Epic 1 scope):**

```
1. main() starts
   │
2. Parse CLI flags (--config path)
   │
3. Load config from YAML file
   │  └─> If file not found: exit(1) with error
   │
4. Apply environment variable overrides
   │  └─> Invalid env values: log warning, keep YAML/default
   │
5. Validate configuration
   │  └─> If validation fails: exit(1) with error
   │
6. Initialize slog logger with config settings
   │
7. Log "application started" with config summary
   │
8. [Epic 1 ends here - subsequent epics add remaining logic]
```

**Config Loading Flow:**

```
                    ┌─────────────────┐
                    │ Built-in Defaults│
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │ YAML File Values │ (override defaults)
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │ Env Var Overrides│ (highest priority)
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │    Validation    │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  Final Config    │
                    └─────────────────┘
```

## Non-Functional Requirements

### Performance

| Requirement | Target | Source |
|-------------|--------|--------|
| Config load time | < 50ms | Reasonable for YAML parsing |
| Logger initialization | < 10ms | slog handler setup is trivial |
| Application startup (Epic 1 scope) | < 100ms | Config + logging only |
| Memory footprint | < 10MB | Config struct is small |

**Notes:**
- Performance is not a primary concern for Epic 1 (foundation setup)
- Config is loaded once at startup, not on hot path
- No I/O operations beyond reading config file

### Security

| Concern | Implementation | Source |
|---------|----------------|--------|
| No secrets in config | Config contains no API keys or credentials | PRD: Public APIs only |
| File permissions | Config file readable by application user only (recommendation) | Security Architecture |
| Input validation | Validate config values before use | FR52 |
| No command injection | Config values are typed, not interpolated into shell commands | Best practice |

**Notes:**
- This is a read-only extraction tool with minimal security surface
- No user data collected or stored
- No authentication required for DefiLlama APIs

### Reliability/Availability

| Requirement | Implementation | Source |
|-------------|----------------|--------|
| Graceful config errors | Clear error messages on missing/invalid config | FR52 |
| Default fallbacks | Sensible defaults for all optional fields | FR51 |
| No crash on bad env vars | Log warning and use YAML/default value | Story 1.3 |

**Notes:**
- Epic 1 establishes error handling patterns used throughout the application
- Explicit error returns per ADR-003

### Observability

| Requirement | Implementation | Source |
|-------------|----------------|--------|
| Structured logging | `log/slog` with JSON or text output | FR53, ADR-004 |
| Log levels | debug, info, warn, error configurable | FR54 |
| Startup logging | Log config summary on application start | Best practice |
| Config override logging | Log applied env var overrides at debug level | Story 1.3 |

**Logging Format Examples:**

JSON format:
```json
{"time":"2025-11-29T10:00:00Z","level":"INFO","msg":"application started","oracle":"Switchboard","log_level":"info"}
```

Text format:
```
2025-11-29T10:00:00Z INFO application started oracle=Switchboard log_level=info
```

## Dependencies and Integrations

### Go Module Dependencies

| Dependency | Version | Purpose | Source |
|------------|---------|---------|--------|
| `gopkg.in/yaml.v3` | latest stable | YAML configuration parsing | ADR-005, Story 1.2 |

**Standard Library Usage (no external deps):**

| Package | Purpose |
|---------|---------|
| `os` | File reading, environment variables |
| `time` | Duration parsing and types |
| `log/slog` | Structured logging (Go 1.21+) |
| `flag` | CLI flag parsing (minimal use in Epic 1) |
| `fmt` | Error formatting |
| `strings` | String manipulation for env var parsing |

### Development Dependencies

| Tool | Purpose | Configuration File |
|------|---------|-------------------|
| `golangci-lint` | Static analysis and linting | `.golangci.yml` |
| `go test` | Unit testing | Built-in |
| `go build` | Compilation | `Makefile` |

### Go Version Requirement

**Minimum: Go 1.21**

Rationale: `log/slog` package was added in Go 1.21. This is a hard requirement per ADR-004.

### Integration Points

| Integration | Direction | Notes |
|-------------|-----------|-------|
| File System | Read | Config file at startup |
| Environment | Read | Env var overrides |
| Stdout/Stderr | Write | Log output |

**Note:** Epic 1 has no network integrations. Network operations (DefiLlama API) are added in Epic 2.

## Acceptance Criteria (Authoritative)

### AC1: Project Structure Compiles
**Given** the project is initialized with standard Go layout
**When** `go build ./...` is executed
**Then** the build succeeds with zero errors

### AC2: Config Loads from YAML
**Given** a valid YAML config file exists at `configs/config.yaml`
**When** the application starts with `--config configs/config.yaml`
**Then** all configuration values are loaded into typed Config struct
**And** missing optional fields use default values

### AC3: Config File Not Found Error
**Given** no config file exists at specified path
**When** the application starts with `--config nonexistent.yaml`
**Then** application exits with code 1
**And** logs "config file not found" error message

### AC4: Config Validation Rejects Invalid Values
**Given** a config file with invalid values (e.g., `timeout: -5s`)
**When** the application starts
**Then** application exits with code 1
**And** logs validation error describing the invalid field

### AC5: Environment Variables Override YAML
**Given** YAML config has `oracle.name: "Switchboard"`
**And** environment variable `ORACLE_NAME=TestOracle` is set
**When** the application starts
**Then** `config.Oracle.Name` equals "TestOracle"

### AC6: Invalid Env Vars Use Fallback
**Given** environment variable `API_TIMEOUT=invalid` is set
**When** the application starts
**Then** a warning is logged
**And** `config.API.Timeout` uses YAML or default value

### AC7: JSON Logging Format
**Given** config with `logging.format: "json"` and `logging.level: "info"`
**When** the logger is initialized and a message is logged
**Then** output is valid JSON with `time`, `level`, `msg` fields

### AC8: Text Logging Format
**Given** config with `logging.format: "text"`
**When** the logger is initialized and a message is logged
**Then** output is human-readable text format

### AC9: Log Level Filtering
**Given** config with `logging.level: "warn"`
**When** code logs at info level
**Then** the message is NOT output
**When** code logs at warn or error level
**Then** the message IS output

### AC10: Makefile Targets Work
**Given** the project is initialized
**When** `make build` is executed
**Then** binary is created at `bin/extractor`
**When** `make test` is executed
**Then** tests run (may be empty initially)
**When** `make lint` is executed
**Then** golangci-lint runs against codebase

## Traceability Mapping

| AC | FR | Spec Section | Component | Test Idea |
|----|-----|--------------|-----------|-----------|
| AC1 | - | Story 1.1 | `cmd/extractor/main.go` | Build verification test |
| AC2 | FR49, FR51 | Story 1.2 | `internal/config/config.go` | Unit test: Load valid YAML |
| AC3 | FR49 | Story 1.2 | `internal/config/config.go` | Unit test: Load missing file |
| AC4 | FR52 | Story 1.2 | `internal/config/config.go` | Unit test: Validate() rejects invalid |
| AC5 | FR50 | Story 1.3 | `internal/config/config.go` | Unit test: Env override applied |
| AC6 | FR50 | Story 1.3 | `internal/config/config.go` | Unit test: Invalid env fallback |
| AC7 | FR53 | Story 1.4 | `cmd/extractor/main.go` | Integration test: JSON output format |
| AC8 | FR53 | Story 1.4 | `cmd/extractor/main.go` | Integration test: Text output format |
| AC9 | FR54 | Story 1.4 | `cmd/extractor/main.go` | Unit test: Level filtering |
| AC10 | - | Story 1.1 | `Makefile` | Manual verification |

### FR Coverage Summary

| FR | Description | Covered By |
|----|-------------|------------|
| FR49 | Load config from YAML | AC2, AC3 |
| FR50 | Environment variable overrides | AC5, AC6 |
| FR51 | Sensible defaults | AC2 |
| FR52 | Validate config on startup | AC4 |
| FR53 | Structured logging (JSON/text) | AC7, AC8 |
| FR54 | Configurable log levels | AC9 |

## Risks, Assumptions, Open Questions

### Risks

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| **R1:** Go version < 1.21 on dev machine | High - slog unavailable | Low | Document Go 1.21+ requirement in README, fail fast with version check |
| **R2:** YAML parsing edge cases | Medium - config fails to load | Low | Use well-tested yaml.v3, add comprehensive test cases |
| **R3:** Duration parsing inconsistencies | Medium - unexpected timeouts | Low | Document expected formats (e.g., "30s", "2h"), validate parsed values |

### Assumptions

| ID | Assumption | Validation |
|----|------------|------------|
| **A1** | Developer has Go 1.21+ installed | Verified at build time |
| **A2** | Config file is UTF-8 encoded YAML | Standard for YAML files |
| **A3** | Environment variables are ASCII strings | OS standard |
| **A4** | golangci-lint is available for `make lint` | Optional, lint target can fail gracefully |
| **A5** | Output directory is writable (for future epics) | Validated in Epic 4/5 |

### Open Questions

| ID | Question | Status | Decision |
|----|----------|--------|----------|
| **Q1** | Should we support config reload without restart? | Resolved | No - not needed for MVP, config is static |
| **Q2** | Should invalid env vars fail startup or warn? | Resolved | Warn and use fallback per Story 1.3 |
| **Q3** | Config file location convention? | Resolved | `configs/config.yaml` default, `--config` override |

## Test Strategy Summary

### Test Levels

| Level | Scope | Framework | Coverage Target |
|-------|-------|-----------|-----------------|
| Unit Tests | Config loading, validation, env overrides | `go test` | 90%+ for `internal/config` |
| Integration Tests | Logger initialization, startup sequence | `go test` | Key paths covered |
| Build Verification | Project compiles | `go build` | Pass/fail |

### Test Cases by Story

**Story 1.1 - Project Structure:**
- [ ] `go build ./...` succeeds
- [ ] `make build` creates binary
- [ ] `make test` runs without error
- [ ] `make lint` runs without error (warnings OK)

**Story 1.2 - Config Loading:**
- [ ] Load valid YAML with all fields
- [ ] Load valid YAML with minimal fields (defaults applied)
- [ ] Load missing file returns error
- [ ] Load invalid YAML returns parse error
- [ ] Validate rejects negative timeout
- [ ] Validate rejects empty oracle name
- [ ] Validate rejects invalid log level
- [ ] Validate accepts valid config

**Story 1.3 - Environment Overrides:**
- [ ] ORACLE_NAME overrides yaml value
- [ ] OUTPUT_DIR overrides yaml value
- [ ] LOG_LEVEL overrides yaml value
- [ ] LOG_FORMAT overrides yaml value
- [ ] API_TIMEOUT parses duration correctly
- [ ] SCHEDULER_INTERVAL parses duration correctly
- [ ] Invalid duration logs warning, uses fallback
- [ ] Unset env vars don't affect config

**Story 1.4 - Structured Logging:**
- [ ] JSON handler produces valid JSON
- [ ] Text handler produces readable text
- [ ] Debug level shows all messages
- [ ] Info level filters debug messages
- [ ] Warn level filters info and debug
- [ ] Error level filters warn, info, debug
- [ ] Log includes structured attributes

### Test Patterns

Per Go idioms and project testing strategy:

```go
// Table-driven tests for config validation
func TestConfig_Validate(t *testing.T) {
    tests := []struct {
        name    string
        config  Config
        wantErr bool
    }{
        {"valid config", validConfig(), false},
        {"negative timeout", configWithTimeout(-1), true},
        {"empty oracle name", configWithOracleName(""), true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.config.Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Definition of Done

Epic 1 is complete when:
- [ ] All 4 stories implemented
- [ ] All 10 acceptance criteria pass
- [ ] Unit test coverage ≥ 90% for `internal/config`
- [ ] `go build ./...` succeeds
- [ ] `make lint` passes with no errors
- [ ] Config loads from YAML and applies env overrides
- [ ] Logger outputs in configured format and level
