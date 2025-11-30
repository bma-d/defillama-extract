# Story 1.4: Implement Structured Logging with slog

Status: done

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

- [x] Task 1: Create logging initialization function (AC: 1, 2, 4, 6)
  - [x] 1.1: Create `internal/logging/logging.go` file
  - [x] 1.2: Implement `Setup(cfg config.LoggingConfig) *slog.Logger` function
  - [x] 1.3: Map config level string ("debug", "info", "warn", "error") to `slog.Level` constant
  - [x] 1.4: Create `slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})` when format is "json"
  - [x] 1.5: Create `slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})` when format is "text"
  - [x] 1.6: Return the configured logger

- [x] Task 2: Integrate logger setup into main.go (AC: 6, 7)
  - [x] 2.1: Import `internal/logging` package in `cmd/extractor/main.go`
  - [x] 2.2: After config loads successfully, call `logging.Setup(cfg.Logging)`
  - [x] 2.3: Call `slog.SetDefault(logger)` to set as global default
  - [x] 2.4: Log startup message: `slog.Info("application started", "oracle", cfg.Oracle.Name, "log_level", cfg.Logging.Level)`

- [x] Task 3: Write unit tests for logging setup (AC: 1, 2, 3, 4, 5)
  - [x] 3.1: Create `internal/logging/logging_test.go`
  - [x] 3.2: Test JSON handler produces valid JSON output with timestamp, level, msg fields
  - [x] 3.3: Test text handler produces human-readable output
  - [x] 3.4: Test level mapping: "debug" -> `slog.LevelDebug`, "info" -> `slog.LevelInfo`, etc.
  - [x] 3.5: Test level filtering: warn level suppresses info and debug messages
  - [x] 3.6: Test structured attributes appear in output (`"key": "value"` in JSON)
  - [x] 3.7: Use `bytes.Buffer` as output destination to capture and verify log output

- [x] Task 4: Update main.go to load config and initialize logging (AC: all)
  - [x] 4.1: Add `flag` package for `--config` flag parsing
  - [x] 4.2: Parse `--config` flag with default value "configs/config.yaml"
  - [x] 4.3: Call `config.Load(configPath)` and handle errors
  - [x] 4.4: On config error, log to stderr and exit with code 1
  - [x] 4.5: After logger initialized, replace placeholder `fmt.Println` with proper startup log

- [x] Task 5: Verification (AC: all)
  - [x] 5.1: Run `go build ./...` and verify success
  - [x] 5.2: Run `go test ./internal/logging/...` and verify all pass
  - [x] 5.3: Run `make lint` and verify no errors
  - [x] 5.4: Manual test: run binary with sample config, verify JSON log output
  - [x] 5.5: Manual test: run binary with `LOG_FORMAT=text` env var, verify text output

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

- 2025-11-30: Plan — add `internal/logging` with `Setup`/`parse level`, support json/text handlers; integrate config load + logger wiring in `cmd/extractor/main.go` with `--config`; add tests for json/text output, level mapping/filtering, structured attrs, SetDefault; run gofmt, go test ./..., go build ./..., make lint.

### Completion Notes List

- 2025-11-30: Implemented slog setup with JSON/text handlers and level mapping; added tests for formats, structured attrs, filtering, SetDefault; wired config loading and startup log in main; commands run: gofmt, go test ./..., go build ./..., make lint, go run ./cmd/extractor (json + text).

### File List

- cmd/extractor/main.go
- internal/logging/logging.go
- internal/logging/logging_test.go
- docs/sprint-artifacts/sprint-status.yaml
- docs/sprint-artifacts/1-4-implement-structured-logging-with-slog.md

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-1-foundation.md and tech-spec-epic-1.md |
| 2025-11-30 | Amelia (Dev Agent) | Implemented structured logging setup, main integration, tests, and updated sprint status |
| 2025-11-30 | BMad (Reviewer) | Senior Developer Review (AI) appended and story approved |

## Senior Developer Review (AI)

- **Reviewer:** BMad
- **Date:** 2025-11-30
- **Outcome:** Approve — all ACs implemented and verified; no blocking findings

### Summary
- Logging setup cleanly maps config to slog handlers (JSON/Text) with level filtering and global default wiring.
- Tests cover handler selection, level mapping/filtering, structured attributes, and SetDefault behavior.
- Startup log now emits oracle name and configured log level as structured fields after config load.

### Key Findings
- None.

### Acceptance Criteria Coverage
| AC | Status | Evidence |
|----|--------|----------|
| AC1 JSON handler emits structured output | Implemented | internal/logging/logging.go:18-30; internal/logging/logging_test.go:14-42 |
| AC2 Text handler for debug level | Implemented | internal/logging/logging.go:22-27; internal/logging/logging_test.go:44-60 |
| AC3 Structured key-value preserved | Implemented | internal/logging/logging_test.go:18-35 |
| AC4 Supports debug/info/warn/error | Implemented | internal/logging/logging.go:33-44; internal/logging/logging_test.go:63-80 |
| AC5 Messages below level suppressed | Implemented | internal/logging/logging.go:19-21; internal/logging/logging_test.go:83-103 |
| AC6 slog.SetDefault uses configured handler | Implemented | cmd/extractor/main.go:21-28; internal/logging/logging_test.go:105-128 |
| AC7 Startup message with oracle + log level | Implemented | cmd/extractor/main.go:24-28 |

### Task Completion Validation
| Task | Marked As | Verified As | Evidence |
|------|-----------|-------------|----------|
| 1: Logging initialization function | [x] | Verified complete | internal/logging/logging.go:12-30 |
| 2: Integrate logger setup into main.go | [x] | Verified complete | cmd/extractor/main.go:12-28 |
| 3: Unit tests for logging setup | [x] | Verified complete | internal/logging/logging_test.go:14-128 |
| 4: Update main.go with config load + startup log | [x] | Verified complete | cmd/extractor/main.go:12-28 |
| 5.1-5.3: Build/Test/Lint commands | [x] | Verified complete | go build ./...; go test ./...; make lint (2025-11-30) |
| 5.4-5.5: Manual JSON/Text run checks | [x] | Not independently verified (manual) | — |

### Test Coverage and Gaps
- Executed: `go test ./...`, `go build ./...`, `make lint` on 2025-11-30 (all pass).
- Manual run checks (JSON/Text outputs) not re-run in this review; rely on automated tests.

### Architectural Alignment
- Uses Go stdlib `log/slog` per ADR-004; no extra dependencies (logging.go:3-7).
- Handler selection and level mapping follow consistency-rules logging strategy (logging.go:18-44).

### Security Notes
- No secrets introduced; logging writes to stdout only.

### Best-Practices and References
- Go 1.23 module; slog structured logging per ADR-004.

### Action Items
- None.
