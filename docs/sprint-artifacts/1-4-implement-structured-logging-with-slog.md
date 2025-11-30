# Story 1.4: Implement Structured Logging with slog

Status: ready-for-dev

## Story

As a **developer**,
I want **structured logging using Go's slog package**,
so that **logs are machine-parseable and include consistent contextual information**.

## Acceptance Criteria

1. **Given** logging configuration with `format: "json"` and `level: "info"` **When** the logger is initialized **Then** a JSON handler is configured writing to stdout **And** log entries include: timestamp, level, message, and any additional attributes

2. **Given** logging configuration with `format: "text"` and `level: "debug"` **When** the logger is initialized **Then** a text handler is configured for human-readable output **And** debug-level messages are included in output

3. **Given** an initialized logger **When** code logs with `slog.Info("message", "key", "value")` **Then** output includes the key-value pair as structured data

4. **Given** logging configuration **When** the logger is initialized **Then** the following log levels are supported: debug, info, warn, error

5. **Given** logging configuration with a specific level **When** messages are logged below that level **Then** those messages are suppressed (e.g., `level: "warn"` suppresses debug and info)

6. **Given** the logger is initialized **When** `slog.SetDefault(logger)` is called **Then** all subsequent `slog.Info()`, `slog.Debug()`, etc. calls use the configured handler

7. **Given** configuration loading completes **When** logging is initialized **Then** a startup message is logged with oracle name and log level as structured attributes

## Tasks / Subtasks

- [ ] Task 1: Create logging initialization function (AC: 1, 2, 4, 6)
  - [ ] 1.1: Create `internal/logging/logging.go` file
  - [ ] 1.2: Implement `Setup(cfg config.LoggingConfig) *slog.Logger` function
  - [ ] 1.3: Map config level string ("debug", "info", "warn", "error") to `slog.Level` constant
  - [ ] 1.4: Create `slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})` when format is "json"
  - [ ] 1.5: Create `slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})` when format is "text"
  - [ ] 1.6: Return the configured logger

- [ ] Task 2: Integrate logger setup into main.go (AC: 6, 7)
  - [ ] 2.1: Import `internal/logging` package in `cmd/extractor/main.go`
  - [ ] 2.2: After config loads successfully, call `logging.Setup(cfg.Logging)`
  - [ ] 2.3: Call `slog.SetDefault(logger)` to set as global default
  - [ ] 2.4: Log startup message: `slog.Info("application started", "oracle", cfg.Oracle.Name, "log_level", cfg.Logging.Level)`

- [ ] Task 3: Write unit tests for logging setup (AC: 1, 2, 3, 4, 5)
  - [ ] 3.1: Create `internal/logging/logging_test.go`
  - [ ] 3.2: Test JSON handler produces valid JSON output with timestamp, level, msg fields
  - [ ] 3.3: Test text handler produces human-readable output
  - [ ] 3.4: Test level mapping: "debug" -> `slog.LevelDebug`, "info" -> `slog.LevelInfo`, etc.
  - [ ] 3.5: Test level filtering: warn level suppresses info and debug messages
  - [ ] 3.6: Test structured attributes appear in output (`"key": "value"` in JSON)
  - [ ] 3.7: Use `bytes.Buffer` as output destination to capture and verify log output

- [ ] Task 4: Update main.go to load config and initialize logging (AC: all)
  - [ ] 4.1: Add `flag` package for `--config` flag parsing
  - [ ] 4.2: Parse `--config` flag with default value "configs/config.yaml"
  - [ ] 4.3: Call `config.Load(configPath)` and handle errors
  - [ ] 4.4: On config error, log to stderr and exit with code 1
  - [ ] 4.5: After logger initialized, replace placeholder `fmt.Println` with proper startup log

- [ ] Task 5: Verification (AC: all)
  - [ ] 5.1: Run `go build ./...` and verify success
  - [ ] 5.2: Run `go test ./internal/logging/...` and verify all pass
  - [ ] 5.3: Run `make lint` and verify no errors
  - [ ] 5.4: Manual test: run binary with sample config, verify JSON log output
  - [ ] 5.5: Manual test: run binary with `LOG_FORMAT=text` env var, verify text output

## Dev Notes

### Technical Guidance

- **Package Location:** Create new `internal/logging/logging.go`
- **Dependencies:** stdlib only (`log/slog`, `os`, `strings`)
- **Go Version:** Requires Go 1.21+ for `log/slog` package (per ADR-004)

### slog Level Mapping

| Config String | slog Constant | Numeric Value |
|---------------|---------------|---------------|
| "debug" | `slog.LevelDebug` | -4 |
| "info" | `slog.LevelInfo` | 0 |
| "warn" | `slog.LevelWarn` | 4 |
| "error" | `slog.LevelError` | 8 |

### Implementation Approach

```go
// internal/logging/logging.go
package logging

import (
    "log/slog"
    "os"
    "strings"

    "github.com/switchboard-xyz/defillama-extract/internal/config"
)

// Setup creates and returns a configured slog.Logger based on logging config.
func Setup(cfg config.LoggingConfig) *slog.Logger {
    level := parseLevel(cfg.Level)
    opts := &slog.HandlerOptions{Level: level}

    var handler slog.Handler
    if strings.ToLower(cfg.Format) == "text" {
        handler = slog.NewTextHandler(os.Stdout, opts)
    } else {
        handler = slog.NewJSONHandler(os.Stdout, opts)
    }

    return slog.New(handler)
}

func parseLevel(s string) slog.Level {
    switch strings.ToLower(s) {
    case "debug":
        return slog.LevelDebug
    case "warn":
        return slog.LevelWarn
    case "error":
        return slog.LevelError
    default:
        return slog.LevelInfo
    }
}
```

### main.go Integration Pattern

```go
package main

import (
    "flag"
    "log"
    "log/slog"
    "os"

    "github.com/switchboard-xyz/defillama-extract/internal/config"
    "github.com/switchboard-xyz/defillama-extract/internal/logging"
)

func main() {
    configPath := flag.String("config", "configs/config.yaml", "path to config file")
    flag.Parse()

    cfg, err := config.Load(*configPath)
    if err != nil {
        log.Fatalf("failed to load config: %v", err)
    }

    logger := logging.Setup(cfg.Logging)
    slog.SetDefault(logger)

    slog.Info("application started",
        "oracle", cfg.Oracle.Name,
        "log_level", cfg.Logging.Level,
    )

    // Future epics add remaining logic here
}
```

### Expected Output Examples

**JSON format (default):**
```json
{"time":"2025-11-30T10:00:00.000Z","level":"INFO","msg":"application started","oracle":"Switchboard","log_level":"info"}
```

**Text format:**
```
2025-11-30T10:00:00.000Z INFO application started oracle=Switchboard log_level=info
```

### Testing Strategy Alignment

Per testing-strategy.md:
- Use table-driven tests for level mapping and format selection
- Capture output to `bytes.Buffer` for verification
- Test both success paths and edge cases (invalid level defaults to info)

### Project Structure Notes

- New file: `internal/logging/logging.go` - follows project structure pattern
- New file: `internal/logging/logging_test.go` - tests co-located per convention
- Modify: `cmd/extractor/main.go` - add config loading and logging setup
- Config package already has `LoggingConfig` struct with `Level` and `Format` fields

### Learnings from Previous Story

**From Story 1-3-implement-environment-variable-overrides (Status: done)**

- **Config Package Ready:** `internal/config/config.go` with `LoggingConfig` struct containing `Level` and `Format` fields
- **Validation in Place:** Config validation already ensures `logging.level` is one of debug/info/warn/error and `logging.format` is one of json/text
- **Load Function:** `config.Load(path)` returns `*Config` with validated settings ready for logging setup
- **Test Patterns:** Table-driven tests with `t.Setenv()` for env var testing - follow similar patterns for logging tests
- **Env Overrides Work:** `LOG_LEVEL` and `LOG_FORMAT` env vars already override YAML values before logging setup
- **Files Added/Modified in Story 1-3:** `internal/config/config.go`, `internal/config/config_test.go`, `docs/sprint-artifacts/sprint-status.yaml`, `docs/sprint-artifacts/1-3-implement-environment-variable-overrides.md` (used for continuity checks)

[Source: docs/sprint-artifacts/1-3-implement-environment-variable-overrides.md#Dev-Agent-Record]

### References

- [Source: docs/sprint-artifacts/tech-spec-epic-1.md#AC7] - JSON handler produces valid JSON with time, level, msg fields
- [Source: docs/sprint-artifacts/tech-spec-epic-1.md#AC8] - Text handler produces human-readable format
- [Source: docs/sprint-artifacts/tech-spec-epic-1.md#AC9] - Log level filtering (warn suppresses info and debug)
- [Source: docs/epics/epic-1-foundation.md#story-14-implement-structured-logging-with-slog] - Original story definition with acceptance criteria
- [Source: docs/prd.md#logging] - FR53 (structured logging), FR54 (configurable log levels)
- [Source: docs/architecture/architecture-decision-records-adrs.md#adr-004] - ADR-004: Structured Logging with slog
- [Source: docs/architecture/consistency-rules.md#logging-strategy] - slog usage patterns and log level guidance
- [Source: docs/architecture/testing-strategy.md] - Test organization and coverage requirements

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/1-4-implement-structured-logging-with-slog.context.xml

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-1-foundation.md and tech-spec-epic-1.md |
