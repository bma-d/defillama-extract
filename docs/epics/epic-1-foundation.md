# Epic 1: Foundation

**Goal:** Establish project structure with working configuration and logging infrastructure so developers can start building features on a solid base.

**User Value:** After this epic, the project compiles, loads configuration from YAML/env vars, and outputs structured logs - the foundation for all subsequent development.

**FRs Covered:** FR49, FR50, FR51, FR52, FR53, FR54

---

## Story 1.1: Initialize Go Module and Project Structure

As a **developer**,
I want **a properly initialized Go module with the standard project layout**,
So that **I have a clean foundation to build the extraction service**.

**Acceptance Criteria:**

**Given** a new project directory
**When** I run `go mod init` and create the directory structure
**Then** the following structure exists:
- `cmd/extractor/main.go` (minimal entry point that compiles)
- `internal/config/` directory
- `internal/models/` directory
- `internal/api/` directory
- `internal/aggregator/` directory
- `internal/storage/` directory
- `go.mod` with module `github.com/switchboard-xyz/defillama-extract`
- `Makefile` with `build`, `test`, `lint` targets
- `.gitignore` excluding `data/`, `*.exe`, vendor/
- `.golangci.yml` with standard linter config

**And** running `go build ./...` succeeds with no errors

**Prerequisites:** None (first story)

**Technical Notes:**
- Follow Go standard project layout per Architecture doc
- Use Go 1.21+ for slog support
- Entry point at `cmd/extractor/main.go` per project-structure.md

---

## Story 1.2: Implement Configuration Loading from YAML

As a **developer**,
I want **configuration loaded from a YAML file with sensible defaults**,
So that **the service behavior can be customized without code changes**.

**Acceptance Criteria:**

**Given** a YAML configuration file at `configs/config.yaml`
**When** the application starts
**Then** configuration is loaded into a typed `Config` struct with sections:
- `oracle`: name (default: "Switchboard"), website, documentation URL
- `api`: base URLs, timeout (default: 30s), max retries (default: 3), retry delay
- `output`: directory (default: "data"), file names
- `scheduler`: interval (default: 2h), start_immediately flag
- `logging`: level (default: "info"), format (default: "json")

**And** missing optional fields use default values
**And** missing required fields cause startup failure with clear error message
**And** invalid values (e.g., negative timeout) cause validation errors

**Given** no config file exists at specified path
**When** the application starts with `--config nonexistent.yaml`
**Then** application exits with error code 1 and logs "config file not found"

**Prerequisites:** Story 1.1

**Technical Notes:**
- Package: `internal/config/config.go`
- Use `gopkg.in/yaml.v3` for YAML parsing
- Config struct fields use `yaml` tags
- Implement `Load(path string) (*Config, error)` function
- Implement `Validate() error` method on Config
- Reference: architecture/fr-category-to-architecture-mapping.md → FR49-FR52

---

## Story 1.3: Implement Environment Variable Overrides

As an **operator**,
I want **environment variables to override YAML configuration**,
So that **I can customize behavior in different environments without modifying files**.

**Acceptance Criteria:**

**Given** a loaded YAML configuration
**When** environment variables are set
**Then** the following overrides are applied (env var → config field):
- `ORACLE_NAME` → `oracle.name`
- `OUTPUT_DIR` → `output.directory`
- `LOG_LEVEL` → `logging.level`
- `LOG_FORMAT` → `logging.format`
- `API_TIMEOUT` → `api.timeout` (parsed as duration, e.g., "45s")
- `SCHEDULER_INTERVAL` → `scheduler.interval`

**And** environment variables take precedence over YAML values
**And** invalid environment variable values log a warning and use YAML/default value
**And** unset environment variables do not affect configuration

**Prerequisites:** Story 1.2

**Technical Notes:**
- Extend `config.Load()` to call `applyEnvOverrides()`
- Use `os.Getenv()` for reading env vars
- Parse duration strings with `time.ParseDuration()`
- Log applied overrides at debug level

---

## Story 1.4: Implement Structured Logging with slog

As a **developer**,
I want **structured logging using Go's slog package**,
So that **logs are machine-parseable and include consistent contextual information**.

**Acceptance Criteria:**

**Given** logging configuration with `format: "json"` and `level: "info"`
**When** the logger is initialized
**Then** a JSON handler is configured writing to stdout
**And** log entries include: timestamp, level, message, and any additional attributes

**Given** logging configuration with `format: "text"` and `level: "debug"`
**When** the logger is initialized
**Then** a text handler is configured for human-readable output
**And** debug-level messages are included in output

**Given** an initialized logger
**When** code logs with `slog.Info("message", "key", "value")`
**Then** output includes the key-value pair as structured data

**And** the following log levels are supported: debug, info, warn, error
**And** messages below configured level are suppressed

**Prerequisites:** Story 1.2

**Technical Notes:**
- Create `internal/logging/logging.go` or initialize in main.go
- Use `slog.New()` with `slog.NewJSONHandler()` or `slog.NewTextHandler()`
- Set as default logger with `slog.SetDefault()`
- Map config level string to `slog.Level` constant
- Reference: 15-go-specific-patterns-idioms.md section 15.1

---
