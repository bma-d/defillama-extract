# Story 1.3: Implement Environment Variable Overrides

Status: done

## Story

As an **operator**,
I want **environment variables to override YAML configuration**,
so that **I can customize behavior in different environments without modifying files**.

## Acceptance Criteria

1. **Given** a loaded YAML configuration **When** environment variable `ORACLE_NAME` is set **Then** `config.Oracle.Name` is overridden to the env var value

2. **Given** a loaded YAML configuration **When** environment variable `OUTPUT_DIR` is set **Then** `config.Output.Directory` is overridden to the env var value

3. **Given** a loaded YAML configuration **When** environment variable `LOG_LEVEL` is set to a valid level (debug, info, warn, error) **Then** `config.Logging.Level` is overridden to the env var value

4. **Given** a loaded YAML configuration **When** environment variable `LOG_FORMAT` is set to a valid format (json, text) **Then** `config.Logging.Format` is overridden to the env var value

5. **Given** a loaded YAML configuration **When** environment variable `API_TIMEOUT` is set to a valid duration (e.g., "45s") **Then** `config.API.Timeout` is overridden to the parsed duration value

6. **Given** a loaded YAML configuration **When** environment variable `SCHEDULER_INTERVAL` is set to a valid duration (e.g., "1h30m") **Then** `config.Scheduler.Interval` is overridden to the parsed duration value

7. **Given** environment variables that take precedence **When** both YAML value and env var are set for the same field **Then** the env var value wins (highest priority)

8. **Given** an invalid environment variable value (e.g., `API_TIMEOUT=invalid`) **When** configuration is loaded **Then** a warning is logged and the YAML/default value is used instead

9. **Given** an environment variable is not set **When** configuration is loaded **Then** the YAML/default value is used and no override occurs

## Tasks / Subtasks

- [x] Task 1: Implement applyEnvOverrides function (AC: 1, 2, 3, 4, 5, 6, 7, 9)
  - [x] 1.1: Create `applyEnvOverrides(cfg *Config)` function in `internal/config/config.go`
  - [x] 1.2: Read `ORACLE_NAME` with `os.Getenv()` and override `cfg.Oracle.Name` if set
  - [x] 1.3: Read `OUTPUT_DIR` with `os.Getenv()` and override `cfg.Output.Directory` if set
  - [x] 1.4: Read `LOG_LEVEL` with `os.Getenv()` and override `cfg.Logging.Level` if set
  - [x] 1.5: Read `LOG_FORMAT` with `os.Getenv()` and override `cfg.Logging.Format` if set
  - [x] 1.6: Read `API_TIMEOUT` with `os.Getenv()` and parse with `time.ParseDuration()`, override if valid
  - [x] 1.7: Read `SCHEDULER_INTERVAL` with `os.Getenv()` and parse with `time.ParseDuration()`, override if valid

- [x] Task 2: Handle invalid duration parsing (AC: 8)
  - [x] 2.1: If `time.ParseDuration()` fails for `API_TIMEOUT`, log warning at debug level (slog not yet initialized, use stderr)
  - [x] 2.2: If `time.ParseDuration()` fails for `SCHEDULER_INTERVAL`, log warning at debug level
  - [x] 2.3: On parse failure, retain YAML/default value (do not override)

- [x] Task 3: Integrate into Load function (AC: 7, 9)
  - [x] 3.1: Call `applyEnvOverrides(cfg)` after YAML loading but before validation in `Load()`
  - [x] 3.2: Ensure env vars have highest priority (applied last before validation)

- [x] Task 4: Write unit tests (AC: all)
  - [x] 4.1: Test `ORACLE_NAME` override works
  - [x] 4.2: Test `OUTPUT_DIR` override works
  - [x] 4.3: Test `LOG_LEVEL` override works
  - [x] 4.4: Test `LOG_FORMAT` override works
  - [x] 4.5: Test `API_TIMEOUT` override parses duration correctly
  - [x] 4.6: Test `SCHEDULER_INTERVAL` override parses duration correctly
  - [x] 4.7: Test env var takes precedence over YAML value
  - [x] 4.8: Test invalid `API_TIMEOUT` retains YAML/default value
  - [x] 4.9: Test invalid `SCHEDULER_INTERVAL` retains YAML/default value
  - [x] 4.10: Test unset env vars don't affect config
  - [x] 4.11: Use `t.Setenv()` for test isolation (Go 1.17+)
  - [x] 4.12: Use table-driven tests per Go idioms

- [x] Task 5: Verification (AC: all)
  - [x] 5.1: Run `go build ./...` and verify success
  - [x] 5.2: Run `go test ./internal/config/...` and verify all pass
  - [x] 5.3: Run `make lint` and verify no errors

## Dev Notes

### Technical Guidance

- **Package Location:** `internal/config/config.go` (extend existing file)
- **Dependencies:** stdlib only (`os`, `time`, `log` for stderr warning)
- **Pattern:** Env overrides applied after YAML merge, before validation

### Environment Variable Mapping (from tech-spec-epic-1.md)

| Env Var | Config Field | Type | Notes |
|---------|--------------|------|-------|
| `ORACLE_NAME` | `oracle.name` | string | Override oracle name |
| `OUTPUT_DIR` | `output.directory` | string | Override output directory |
| `LOG_LEVEL` | `logging.level` | string | debug/info/warn/error |
| `LOG_FORMAT` | `logging.format` | string | json/text |
| `API_TIMEOUT` | `api.timeout` | duration | e.g., "45s" |
| `SCHEDULER_INTERVAL` | `scheduler.interval` | duration | e.g., "1h30m" |

### Implementation Approach

```go
func applyEnvOverrides(cfg *Config) {
    // String overrides - simple os.Getenv check
    if v := os.Getenv("ORACLE_NAME"); v != "" {
        cfg.Oracle.Name = v
    }
    if v := os.Getenv("OUTPUT_DIR"); v != "" {
        cfg.Output.Directory = v
    }
    if v := os.Getenv("LOG_LEVEL"); v != "" {
        cfg.Logging.Level = v
    }
    if v := os.Getenv("LOG_FORMAT"); v != "" {
        cfg.Logging.Format = v
    }

    // Duration overrides - parse with error handling
    if v := os.Getenv("API_TIMEOUT"); v != "" {
        if d, err := time.ParseDuration(v); err == nil {
            cfg.API.Timeout = d
        } else {
            log.Printf("warning: invalid API_TIMEOUT '%s', using default", v)
        }
    }
    if v := os.Getenv("SCHEDULER_INTERVAL"); v != "" {
        if d, err := time.ParseDuration(v); err == nil {
            cfg.Scheduler.Interval = d
        } else {
            log.Printf("warning: invalid SCHEDULER_INTERVAL '%s', using default", v)
        }
    }
}
```

### Config Loading Flow (from tech-spec-epic-1.md)

```
Built-in Defaults → YAML File Values → Env Var Overrides → Validation
                                            ↑
                                    (applyEnvOverrides)
```

### Testing with t.Setenv

Go 1.17+ provides `t.Setenv()` which automatically cleans up env vars after the test:

```go
func TestEnvOverride_OracleName(t *testing.T) {
    t.Setenv("ORACLE_NAME", "TestOracle")

    cfg, err := Load("testdata/config_minimal.yaml")
    require.NoError(t, err)
    assert.Equal(t, "TestOracle", cfg.Oracle.Name)
}
```

### Testing Strategy Alignment

- Follow project testing strategy for unit and table-driven tests ([Source: docs/architecture/testing-strategy.md#Test-Organization]).
- Ensure override tests cover success and error paths per coverage requirements in testing-strategy.

### Project Structure Notes

- Config package already established in `internal/config/config.go`
- Test file already exists at `internal/config/config_test.go` - extend with new tests
- Test data files available in `internal/config/testdata/`

### Learnings from Previous Story

**From Story 1-2-implement-configuration-loading-from-yaml (Status: done)**

- **Config Package Ready:** `internal/config/config.go` with Config struct, Load(), Validate() functions
- **Test Structure Established:** `internal/config/config_test.go` with table-driven tests pattern
- **Test Data Available:** `internal/config/testdata/` directory with config_minimal.yaml, config_all.yaml
- **YAML Dependency Added:** `gopkg.in/yaml.v3` available in go.mod (marked as direct dependency)
- **Load Function Structure:** Load() returns Config with defaults applied, then YAML merged - extend this flow
- **Validation After Load:** Validation runs after all config sources merged - env overrides must be applied before validation

[Source: docs/sprint-artifacts/1-2-implement-configuration-loading-from-yaml.md#Dev-Agent-Record]

### References

- [Source: docs/sprint-artifacts/tech-spec-epic-1.md#environment-variable-mapping] - Env var to config field mapping
- [Source: docs/sprint-artifacts/tech-spec-epic-1.md#config-loading-flow] - Priority order: defaults → YAML → env
- [Source: docs/epics.md#story-13] - Original story definition with acceptance criteria
- [Source: docs/prd.md#configuration] - FR50 requirement for environment variable overrides
- [Source: docs/architecture/architecture-decision-records-adrs.md#adr-005] - Minimal external dependencies (stdlib for env)
- [Source: docs/architecture/testing-strategy.md#Test-Organization] - Required unit/table-driven coverage for config overrides

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/1-3-implement-environment-variable-overrides.context.xml

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

- Plan: review config.go/tests; add applyEnvOverrides with string+duration overrides and warnings; wire into Load pre-Validate; add table-driven env override tests covering success, precedence, invalid durations, unset; run go build/tests/lint.

### Completion Notes List

- Env override layer added: `applyEnvOverrides` applies ORACLE_NAME, OUTPUT_DIR, LOG_LEVEL, LOG_FORMAT, API_TIMEOUT, SCHEDULER_INTERVAL with duration parsing and stderr warnings on invalid input; wired into `Load` before validation (AC1-7,9).
- Env precedence validated with table-driven tests; invalid durations fall back to YAML/default while logging warnings; unset env vars leave YAML/default intact (AC7-9).
- Regression checks executed: `go build ./...`, `go test ./internal/config/...`, `make lint` (AC all).

### File List

- internal/config/config.go
- internal/config/config_test.go
- docs/sprint-artifacts/sprint-status.yaml
- docs/sprint-artifacts/1-3-implement-environment-variable-overrides.md

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epics.md and tech-spec-epic-1.md |
| 2025-11-30 | SM Agent (Bob) | Story context generated and story marked ready-for-dev |
| 2025-11-30 | Dev Agent (Amelia) | Implemented env overrides, tests, lint/build/test verification; moved story to review |
| 2025-11-30 | Dev Agent (Amelia) | Senior Developer Review appended; status moved to done |

## Senior Developer Review (AI)

- **Reviewer:** BMad  
- **Date:** 2025-11-30  
- **Outcome:** Approve — all ACs implemented and tests green

### Summary
Env override layer works per spec; durations parse with warnings on invalid input; env values take precedence; defaults preserved when unset. go build/go test succeed. No blocking findings.

### Key Findings
- None (no High/Medium/Low issues)

### Acceptance Criteria Coverage
| AC# | Description | Status | Evidence |
| --- | ----------- | ------ | -------- |
| 1 | ORACLE_NAME overrides config.Oracle.Name | IMPLEMENTED | internal/config/config.go:55-59; internal/config/config_test.go:150-179 |
| 2 | OUTPUT_DIR overrides config.Output.Directory | IMPLEMENTED | internal/config/config.go:61-63; internal/config/config_test.go:150-179 |
| 3 | LOG_LEVEL overrides config.Logging.Level (valid values) | IMPLEMENTED | internal/config/config.go:65-67,148-174; internal/config/config_test.go:150-179 |
| 4 | LOG_FORMAT overrides config.Logging.Format (valid values) | IMPLEMENTED | internal/config/config.go:69-71,176-182; internal/config/config_test.go:150-179 |
| 5 | API_TIMEOUT parses duration override | IMPLEMENTED | internal/config/config.go:73-78; internal/config/config_test.go:150-179 |
| 6 | SCHEDULER_INTERVAL parses duration override | IMPLEMENTED | internal/config/config.go:81-86; internal/config/config_test.go:150-182 |
| 7 | Env vars have highest priority | IMPLEMENTED | internal/config/config.go:123-140; internal/config/config_test.go:185-217 |
| 8 | Invalid duration logs warning, keeps YAML/default | IMPLEMENTED | internal/config/config.go:73-86; internal/config/config_test.go:220-249 |
| 9 | Unset env vars leave YAML/default values | IMPLEMENTED | internal/config/config.go:55-88; internal/config/config_test.go:46-66 |

**AC Coverage:** 9 / 9 implemented.

### Task Completion Validation
| Task | Marked As | Verified As | Evidence |
| ---- | --------- | ----------- | -------- |
| Task 1: Implement applyEnvOverrides | Completed | VERIFIED COMPLETE | internal/config/config.go:55-88 |
| Task 2: Handle invalid duration parsing | Completed | VERIFIED COMPLETE | internal/config/config.go:73-86; internal/config/config_test.go:220-249 |
| Task 3: Integrate into Load | Completed | VERIFIED COMPLETE | internal/config/config.go:123-140 |
| Task 4: Write unit tests | Completed | VERIFIED COMPLETE | internal/config/config_test.go:150-249 |
| Task 5: Verification (build/test/lint) | Completed | VERIFIED COMPLETE (build/test re-run; lint not re-run in review) | go build ./...; go test ./... |

**Task Summary:** 5 / 5 completed; 0 questionable; 0 false completions.

### Test Coverage and Gaps
- Executed: `go build ./...`; `go test ./...` (all pass).  
- Lint: not re-run during review (developer claim only).

### Architectural Alignment
- Env overrides applied after YAML, before Validate per tech spec.  
- Uses stdlib only (ADR-005). No new dependencies introduced.

### Security Notes
- Env inputs parsed and validated; invalid durations fall back safely; no secrets stored.

### Best-Practices and References
- Follows config loading flow (defaults → YAML → env) per tech-spec-epic-1.md.  
- Validation enforces allowed log levels/formats (config.go:148-182).

### Action Items
**Code Changes Required:**
- [ ] None (no changes requested)

**Advisory Notes:**
- Note: `make lint` not re-run during this review; optional to confirm locally.
