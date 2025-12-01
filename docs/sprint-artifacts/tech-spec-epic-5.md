# Epic Technical Specification: Output & CLI

Date: 2025-12-01
Author: BMad
Epic ID: 5
Status: Draft

---

## Overview

Epic 5 completes the defillama-extract tool by implementing the final layer: JSON output generation and full CLI operability. This epic transforms the data pipeline (Epics 1-4) into a deployable, production-ready CLI tool that produces dashboard-consumable JSON files, supports both single-run and daemon modes, and handles operational concerns like graceful shutdown.

The epic consolidates what was originally 10 stories into 3 cohesive stories (per course correction 2025-11-30), covering: (1) atomic file output generation for full/minified/summary JSON, (2) CLI flag parsing with single-run mode, and (3) daemon mode with scheduler and graceful shutdown. These deliver FR35-FR48 and FR56 from the PRD.

## Objectives and Scope

**In Scope:**
- Full output JSON generation with history (`switchboard-oracle-data.json`)
- Minified output JSON (`switchboard-oracle-data.min.json`)
- Summary output JSON without history (`switchboard-summary.json`)
- Atomic file writes (temp file + rename) for all outputs
- CLI flags: `--once`, `--config`, `--dry-run`, `--version`
- Single extraction mode (`--once`)
- Daemon mode with configurable 2-hour interval scheduler
- Graceful shutdown on SIGINT/SIGTERM
- Extraction cycle logging with duration, protocol count, TVS

**Out of Scope:**
- Prometheus metrics endpoint (post-MVP)
- Health check HTTP endpoint (post-MVP)
- Docker/containerization (post-MVP)
- Web UI or API serving

## System Architecture Alignment

| Component | Package | Key Files | ADR Alignment |
|-----------|---------|-----------|---------------|
| Output Generation | `internal/storage` | `writer.go` | ADR-002: Atomic File Writes |
| Output Models | `internal/models` | `output.go` | - |
| CLI Entry Point | `cmd/extractor` | `main.go` | ADR-001: Use Go Standard Library |
| Logging | All packages | Use `slog` | ADR-004: Structured Logging |

**Dependencies on Previous Epics:**
- Epic 1: Configuration loading, structured logging
- Epic 3: Aggregator (`AggregationResult`) as input to output generation
- Epic 4: State manager, history manager providing snapshots

## Detailed Design

### Services and Modules

| Module | File | Responsibility | Inputs | Outputs |
|--------|------|----------------|--------|---------|
| **OutputGenerator** | `internal/storage/writer.go` | Generate and serialize output structs | `AggregationResult`, `[]Snapshot`, `Config` | `FullOutput`, `SummaryOutput` |
| **AtomicWriter** | `internal/storage/writer.go` | Write JSON files atomically | File path, data bytes | Written file |
| **CLIParser** | `cmd/extractor/main.go` | Parse command-line flags | `os.Args` | `CLIOptions` struct |
| **Scheduler** | `cmd/extractor/main.go` | Manage extraction intervals | Config interval, context | Timed extraction triggers |
| **Runner** | `cmd/extractor/main.go` | Orchestrate extraction cycle | All components | Exit code |

### Data Models and Contracts

**Output Structs (internal/models/output.go):**

```go
// FullOutput - complete output with history
type FullOutput struct {
    Version    string               `json:"version"`
    Oracle     OracleInfo           `json:"oracle"`
    Metadata   OutputMetadata       `json:"metadata"`
    Summary    Summary              `json:"summary"`
    Metrics    Metrics              `json:"metrics"`
    Breakdown  Breakdown            `json:"breakdown"`
    Protocols  []AggregatedProtocol `json:"protocols"`
    Historical []Snapshot           `json:"historical"`
}

// SummaryOutput - current snapshot only (no history)
type SummaryOutput struct {
    Version      string               `json:"version"`
    Oracle       OracleInfo           `json:"oracle"`
    Metadata     OutputMetadata       `json:"metadata"`
    Summary      Summary              `json:"summary"`
    Metrics      Metrics              `json:"metrics"`
    Breakdown    Breakdown            `json:"breakdown"`
    TopProtocols []AggregatedProtocol `json:"top_protocols"` // Top 10 only
}

// OracleInfo - oracle identification
type OracleInfo struct {
    Name          string `json:"name"`
    Website       string `json:"website"`
    Documentation string `json:"documentation"`
}

// OutputMetadata - extraction metadata
type OutputMetadata struct {
    LastUpdated      string `json:"last_updated"`
    DataSource       string `json:"data_source"`
    UpdateFrequency  string `json:"update_frequency"`
    ExtractorVersion string `json:"extractor_version"`
}
```

### JSON Schemas (authoritative contracts)

```json
// Full output (also used for indented file)
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["version", "oracle", "metadata", "summary", "metrics", "breakdown", "protocols", "historical"],
  "properties": {
    "version": {"type": "string"},
    "oracle": {"type": "object", "required": ["name", "website", "documentation"], "additionalProperties": false},
    "metadata": {"type": "object", "required": ["last_updated", "data_source", "update_frequency", "extractor_version"], "additionalProperties": false},
    "summary": {"type": "object"},
    "metrics": {"type": "object"},
    "breakdown": {"type": "object"},
    "protocols": {"type": "array", "items": {"type": "object"}},
    "historical": {"type": "array", "items": {"type": "object"}}
  },
  "additionalProperties": false
}

// Summary output (no historical array, top 10 protocols only)
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["version", "oracle", "metadata", "summary", "metrics", "breakdown", "top_protocols"],
  "properties": {
    "version": {"type": "string"},
    "oracle": {"type": "object", "required": ["name", "website", "documentation"], "additionalProperties": false},
    "metadata": {"type": "object", "required": ["last_updated", "data_source", "update_frequency", "extractor_version"], "additionalProperties": false},
    "summary": {"type": "object"},
    "metrics": {"type": "object"},
    "breakdown": {"type": "object"},
    "top_protocols": {"type": "array", "maxItems": 10, "items": {"type": "object"}}
  },
  "additionalProperties": false
}
```

Notes:
- Minified output uses the full-output schema (identical fields, no whitespace).
- Schemas are contract source for tests and downstream consumers; any field change requires schema update + version bump.

**CLI Options (cmd/extractor/main.go):**

```go
type CLIOptions struct {
    Once       bool   // --once flag
    ConfigPath string // --config flag
    DryRun     bool   // --dry-run flag
    Version    bool   // --version flag
}
```

### APIs and Interfaces

**OutputGenerator Interface:**

```go
// GenerateFullOutput creates complete output with history
func GenerateFullOutput(result *AggregationResult, history []Snapshot, cfg *Config) *FullOutput

// GenerateSummaryOutput creates current-snapshot-only output
func GenerateSummaryOutput(result *AggregationResult, cfg *Config) *SummaryOutput
```

**AtomicWriter Interface:**

```go
// WriteJSON writes data to path atomically (temp + rename)
func WriteJSON(path string, data interface{}, indent bool) error

// WriteAllOutputs writes full, minified, and summary files using cfg.Output filenames
func WriteAllOutputs(outputDir string, cfg *Config, full *FullOutput, summary *SummaryOutput) error
```

**Runner Interface:**

```go
// RunOnce executes single extraction cycle
func RunOnce(ctx context.Context, cfg *Config, opts CLIOptions) error

// RunDaemon executes scheduled extraction cycles
func RunDaemon(ctx context.Context, cfg *Config) error
```

### Workflows and Sequencing

**Single Extraction Cycle (--once mode):**

```
1. Parse CLI flags
2. Load configuration (YAML + env overrides)
3. Validate configuration
4. Initialize logger (slog)
5. Create components:
   - API Client (from Epic 2)
   - Aggregator (from Epic 3)
   - StateManager (from Epic 4)
   - OutputWriter (this epic)
6. Load existing state
7. Fetch API data (parallel: oracles + protocols)
8. Check if new data available (compare timestamps)
   - If no new data: log "skipping", exit 0
9. Aggregate data
10. Load existing history from output file
11. Append new snapshot (deduplicated)
12. Generate outputs (full, minified, summary)
13. Write outputs atomically (unless --dry-run)
14. Save state
15. Log completion metrics
16. Exit 0
```

**Daemon Mode Sequence:**

```
1-5. Same initialization as single mode
6. Set up signal handler (SIGINT, SIGTERM)
7. Create ticker with scheduler.interval (default 2h)
8. If scheduler.start_immediately: run extraction immediately
9. Loop:
   a. Wait for ticker or shutdown signal
   b. If shutdown: complete current extraction, exit 0
   c. Run extraction cycle
   d. Log "next extraction at {timestamp}"
   e. Continue loop
```

**Graceful Shutdown:**

```
Signal received during extraction:
  → Context cancelled
  → Current operation completes or times out
  → Partial results NOT written
  → Exit 0 (daemon) or exit 1 (--once interrupted)

Signal received while waiting:
  → Ticker cancelled immediately
  → Exit 0
```

## Non-Functional Requirements

### Performance

| Metric | Target | Source |
|--------|--------|--------|
| Atomic file write | < 100ms for typical output (~500KB) | NFR4 |
| Full extraction cycle | < 2 minutes | NFR2 |
| Memory stability | No growth over extended daemon operation | NFR5 |
| JSON serialization | Standard `encoding/json` performance | - |

**Implementation Notes:**
- Use `os.CreateTemp()` in same directory as target for guaranteed atomic rename
- Minified JSON reduces file size by ~40% vs indented
- Summary file is ~10% size of full output (no history)

### Security

| Concern | Approach | Source |
|---------|----------|--------|
| File permissions | Default user permissions (0644) | Security Architecture |
| No secrets | No API keys, no credentials | Security Architecture |
| Input validation | Output data already validated by aggregator | - |
| Atomic writes | Prevents partial/corrupted files | ADR-002 |

**Notes:**
- Output files are world-readable (public data)
- No sensitive data in outputs or logs
- No network listeners (CLI tool only)

### Reliability/Availability

| Requirement | Implementation | Source |
|-------------|----------------|--------|
| Atomic writes prevent corruption | Temp file + rename pattern | NFR4, ADR-002 |
| Daemon continues after extraction failure | Log error, schedule next extraction | NFR10 |
| Original file preserved on write failure | Temp file cleaned up, original untouched | NFR7 |
| Graceful shutdown | Context cancellation, complete current work | FR47 |

**Error Handling:**
- Write failure: Clean up temp file, return error, preserve original
- Daemon extraction failure: Log error, continue to next scheduled run
- Signal during extraction: Complete or timeout, then exit

### Observability

| Signal | Log Level | Attributes | Source |
|--------|-----------|------------|--------|
| Extraction started | INFO | timestamp | FR48 |
| Extraction completed | INFO | duration_ms, protocol_count, tvs, chains | FR48, FR56 |
| Extraction skipped | INFO | last_updated, reason | FR48 |
| Extraction failed | ERROR | error, duration_ms | FR48 |
| Daemon started | INFO | interval | FR43 |
| Next extraction scheduled | INFO | next_timestamp | FR43 |
| Shutdown signal received | INFO | signal | FR47 |
| File written | DEBUG | path, size_bytes | - |
| Output file metrics snapshot | INFO | full_size_bytes, min_size_bytes, summary_size_bytes | Observability Rec 2025-12-01 |

**Log Format:**
- JSON format for daemon mode (machine-parseable)
- Text format for development/debugging
- All logs via `slog` (ADR-004)

## Dependencies and Integrations

**Go Module Dependencies (go.mod):**

| Dependency | Version | Purpose |
|------------|---------|---------|
| `golang.org/x/sync` | v0.18.0 | `errgroup` for parallel API fetching (Epic 2) |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML config file parsing (Epic 1) |

**Standard Library Usage (Epic 5 Specific):**

| Package | Purpose |
|---------|---------|
| `flag` | CLI flag parsing (ADR-001) |
| `encoding/json` | JSON serialization for outputs |
| `os` | File operations, signal handling |
| `os/signal` | Graceful shutdown signal handling |
| `syscall` | SIGTERM constant |
| `time` | Ticker for daemon scheduling |
| `context` | Cancellation propagation |
| `log/slog` | Structured logging (ADR-004) |

**Internal Package Dependencies:**

| This Epic Uses | From Epic | Purpose |
|----------------|-----------|---------|
| `internal/config.Config` | Epic 1 | Configuration struct |
| `internal/config.Load()` | Epic 1 | Config loading |
| `internal/api.Client` | Epic 2 | API fetching |
| `internal/aggregator.Aggregator` | Epic 3 | Data aggregation |
| `internal/aggregator.AggregationResult` | Epic 3 | Aggregation output |
| `internal/storage.StateManager` | Epic 4 | State tracking |
| `internal/storage.Snapshot` | Epic 4 | Historical snapshots |

**Integration Points:**

| System | Integration | Notes |
|--------|-------------|-------|
| Dashboard | JSON file consumption | Files written to configured output_dir |
| Cron/Scheduler | Exit codes | 0=success, 1=failure for external scheduling |
| Shell | Signal handling | SIGINT (Ctrl+C), SIGTERM for graceful shutdown |

## Acceptance Criteria (Authoritative)

### Story 5.1: Output File Generation

| AC ID | Criterion | Testable Statement |
|-------|-----------|-------------------|
| 5.1.1 | Full output JSON generated | `GenerateFullOutput()` produces struct with version, oracle, metadata, summary, metrics, breakdown, protocols, historical |
| 5.1.2 | Full output human-readable | JSON serialized with 2-space indentation |
| 5.1.3 | Full output file path | Written to `{output_dir}/switchboard-oracle-data.json` |
| 5.1.4 | Minified output JSON | Same data as full, serialized without whitespace |
| 5.1.5 | Minified output file path | Written to `{output_dir}/switchboard-oracle-data.min.json` |
| 5.1.6 | Summary output JSON | Contains current snapshot only: version, oracle, metadata, summary, metrics, breakdown, top_protocols (10) |
| 5.1.7 | Summary has no history | NO `historical` array in summary output |
| 5.1.8 | Summary output file path | Written to `{output_dir}/switchboard-summary.json` |
| 5.1.9 | Atomic write temp file | Data written to `{path}.tmp` first |
| 5.1.10 | Atomic write rename | Temp file renamed to target atomically |
| 5.1.11 | Directory creation | `os.MkdirAll` creates output dir if missing |
| 5.1.12 | Write failure cleanup | On error, temp file cleaned up, original preserved |
| 5.1.13 | WriteAllOutputs | All three files written atomically in single call |

### Story 5.2: CLI and Single-Run Mode

| AC ID | Criterion | Testable Statement |
|-------|-----------|-------------------|
| 5.2.1 | --once flag | Single extraction mode activated |
| 5.2.2 | --config flag | Custom config path accepted |
| 5.2.3 | --dry-run flag | Fetch/process but no file writes |
| 5.2.4 | --version flag | Prints "defillama-extract v1.0.0" and exits 0 |
| 5.2.5 | No flags default | Daemon mode activated |
| 5.2.6 | Single mode sequence | Load config → Load state → Fetch → Check new → Aggregate → Write → Save state → Exit |
| 5.2.7 | Single mode success exit | Exit code 0 on success |
| 5.2.8 | Single mode failure exit | Exit code 1 on failure |
| 5.2.9 | No new data handling | Exit 0, log "no new data, skipping extraction" |
| 5.2.10 | Dry-run no writes | Files not written, log "dry-run mode, skipping file writes" |
| 5.2.11 | Log extraction started | INFO log with timestamp |
| 5.2.12 | Log extraction completed | INFO log with duration_ms, protocol_count, tvs, chains |
| 5.2.13 | Log extraction skipped | INFO log with last_updated |
| 5.2.14 | Log extraction failed | ERROR log with error, duration_ms |

### Story 5.3: Daemon Mode and Main Entry Point

| AC ID | Criterion | Testable Statement |
|-------|-----------|-------------------|
| 5.3.1 | Daemon scheduler | Extraction runs every scheduler.interval (default 2h) |
| 5.3.2 | Daemon started log | INFO "daemon started, interval: 2h" |
| 5.3.3 | Start immediately true | First extraction runs immediately |
| 5.3.4 | Start immediately false | First extraction waits for interval |
| 5.3.5 | Next extraction log | "next extraction at {timestamp}" after each cycle |
| 5.3.6 | Daemon error recovery | Error logged, daemon continues, next extraction scheduled |
| 5.3.7 | Shutdown during extraction | Current extraction completes, then exit 0 |
| 5.3.8 | Shutdown log | "shutdown signal received, finishing current extraction" |
| 5.3.9 | Shutdown while waiting | Wait cancelled immediately, exit 0 |
| 5.3.10 | --once interrupted | Extraction cancelled, partial results NOT written, exit 1 |
| 5.3.11 | Main sequence | Parse flags → Version check → Load config → Apply env → Validate → Init logger → Create components → Signal handling → Run mode → Exit |
| 5.3.12 | Init failure | Error logged, exit 1 |

## Traceability Mapping

| AC ID | FR(s) | Spec Section | Component/File | Test Approach |
|-------|-------|--------------|----------------|---------------|
| 5.1.1 | FR35, FR40 | Data Models | `internal/models/output.go` | Unit: verify struct fields |
| 5.1.2 | FR35 | APIs/Interfaces | `internal/storage/writer.go` | Unit: verify indentation |
| 5.1.3 | FR35 | Workflows | `internal/storage/writer.go` | Integration: verify file path |
| 5.1.4 | FR36 | APIs/Interfaces | `internal/storage/writer.go` | Unit: verify no whitespace |
| 5.1.5 | FR36 | Workflows | `internal/storage/writer.go` | Integration: verify file path |
| 5.1.6 | FR37 | Data Models | `internal/models/output.go` | Unit: verify struct fields |
| 5.1.7 | FR37 | Data Models | `internal/models/output.go` | Unit: verify no Historical field |
| 5.1.8 | FR37 | Workflows | `internal/storage/writer.go` | Integration: verify file path |
| 5.1.9 | FR38 | APIs/Interfaces | `internal/storage/writer.go` | Unit: verify temp file created |
| 5.1.10 | FR38 | APIs/Interfaces | `internal/storage/writer.go` | Unit: verify atomic rename |
| 5.1.11 | FR39 | APIs/Interfaces | `internal/storage/writer.go` | Unit: verify MkdirAll |
| 5.1.12 | FR38 | Reliability | `internal/storage/writer.go` | Unit: inject write failure |
| 5.1.13 | FR35-37 | Workflows | `internal/storage/writer.go` | Integration: verify all files |
| 5.2.1 | FR42 | CLI | `cmd/extractor/main.go` | Unit: parse --once |
| 5.2.2 | FR44 | CLI | `cmd/extractor/main.go` | Unit: parse --config |
| 5.2.3 | FR45 | CLI | `cmd/extractor/main.go` | Integration: dry-run mode |
| 5.2.4 | FR46 | CLI | `cmd/extractor/main.go` | Unit: version output |
| 5.2.5 | FR43 | Workflows | `cmd/extractor/main.go` | Integration: default mode |
| 5.2.6 | FR42 | Workflows | `cmd/extractor/main.go` | Integration: full cycle |
| 5.2.7-8 | FR42 | CLI | `cmd/extractor/main.go` | Integration: exit codes |
| 5.2.9 | FR27 | Workflows | `cmd/extractor/main.go` | Integration: skip logic |
| 5.2.10 | FR45 | Workflows | `cmd/extractor/main.go` | Integration: no writes |
| 5.2.11-14 | FR48, FR56 | Observability | `cmd/extractor/main.go` | Unit: log assertions |
| 5.3.1 | FR43 | Daemon | `cmd/extractor/main.go` | Integration: time.Ticker |
| 5.3.2-5 | FR43 | Observability | `cmd/extractor/main.go` | Unit: log assertions |
| 5.3.6 | FR43 | Reliability | `cmd/extractor/main.go` | Integration: inject failure |
| 5.3.7-10 | FR47 | Daemon | `cmd/extractor/main.go` | Integration: signal testing |
| 5.3.11-12 | FR42-48 | Workflows | `cmd/extractor/main.go` | Integration: init sequence |

## Risks, Assumptions, Open Questions

### Risks

| ID | Risk | Impact | Mitigation |
|----|------|--------|------------|
| R1 | Large history arrays cause slow JSON serialization | Performance degradation | Monitor file size; future: consider streaming JSON or compression |
| R2 | Signal handling race conditions | Partial writes or corrupted state | Use context cancellation pattern; atomic writes protect output |
| R3 | Disk full during atomic write | Temp file created but rename fails | Check available disk space; clean up temp on failure |

### Assumptions

| ID | Assumption | Rationale |
|----|------------|-----------|
| A1 | Output directory is writable by process | CLI tool runs with user permissions per Security Architecture |
| A2 | `os.Rename` is atomic on target filesystem | Standard POSIX guarantee; tested on ext4, APFS |
| A3 | History arrays remain manageable size | Keep-all strategy per PRD; ~365 snapshots/year at ~1KB each; benchmark target: serialize 1,000 snapshots (<1MB) in <150ms on dev laptop |
| A4 | Dashboard consumes JSON synchronously | Atomic writes ensure no partial reads |

### Open Questions

| ID | Question | Owner | Resolution Path |
|----|----------|-------|-----------------|
| Q1 | Should we add file size metrics to logs? | Dev | Yes—emit `full_size_bytes`, `min_size_bytes`, `summary_size_bytes` in new INFO log (see Observability table) and track implementation via Story 5.1 follow-up task |
| Q2 | What's the max reasonable history size before performance degrades? | Dev | Benchmark with 1000+ snapshots; document findings |

## Test Strategy Summary

### Unit Tests

| Component | Test File | Key Test Cases |
|-----------|-----------|----------------|
| OutputGenerator | `writer_test.go` | GenerateFullOutput fields, GenerateSummaryOutput fields, top 10 limit |
| AtomicWriter | `writer_test.go` | Temp file creation, atomic rename, failure cleanup, directory creation |
| CLIParser | `main_test.go` | All flag combinations, default values, version output |
| Log formatting | `main_test.go` | Log message assertions for all extraction events |

### Integration Tests

| Scenario | Setup | Verification |
|----------|-------|--------------|
| Full extraction cycle | Mock API responses | All 3 output files exist with correct content |
| Dry-run mode | `--dry-run` flag | No files written, logs confirm |
| Skip when no new data | State with recent timestamp | Exit 0, skip log |
| Daemon shutdown | Send SIGINT during wait | Immediate exit 0 |
| Daemon error recovery | Mock API failure | Error logged, next extraction scheduled |

### Smoke Tests (from Epic file)

**Story 5.1:**
1. Run extraction with valid data
2. Verify all 3 files exist in output dir
3. Verify full JSON is formatted, minified is compact
4. Verify summary has no historical array
5. Kill process mid-write, verify no corrupt files remain

**Story 5.2:**
1. `./extractor --version` → prints version, exits 0
2. `./extractor --once --config ./config.yaml` → runs once, exits
3. `./extractor --once --dry-run` → fetches but no files written
4. Check logs contain duration_ms, protocol_count, tvs
5. Run with stale data → logs "skipping", exits 0

**Story 5.3:**
1. Run without `--once` → daemon starts, logs interval
2. Wait for scheduled extraction → runs and logs next time
3. Send SIGINT during extraction → completes, then exits 0
4. Send SIGINT while waiting → exits immediately with 0
5. Cause init failure (bad config) → exits 1 with error

### Test Coverage Target

- Unit test coverage: 80%+ on `internal/storage/writer.go`, `cmd/extractor/main.go`
- Integration tests: Cover all happy paths and primary error paths
- No external dependencies: All tests use mocks/stubs for API calls

## Post-Review Follow-ups

- [x] [Story 5.1][High] Ensure output writers respect `cfg.Output.FullFile/MinFile/SummaryFile` so configurable filenames take effect rather than the current constants.
- [x] [Story 5.1][Medium] Populate `metadata.update_frequency` from `cfg.Scheduler.Interval` so outputs reflect the actual configured cadence instead of the fixed "2 hours" literal.

---

_Generated by BMAD Epic Tech Context Workflow v6.0_
_Date: 2025-12-01_
_For: BMad_
