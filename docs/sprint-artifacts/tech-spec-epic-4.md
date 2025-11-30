# Epic Technical Specification: State & History Management

Date: 2025-11-30
Author: BMad
Epic ID: 4
Status: Draft

---

## Overview

Epic 4 implements the incremental update tracking and historical snapshot management system that enables efficient polling and time-series metrics. This epic transforms the extraction service from a stateless data fetcher (Epics 1-3) into an intelligent system that tracks what has been processed, skips redundant work when no new data is available, and maintains a complete historical record of TVS snapshots over time.

The state management layer serves as the bridge between extraction cycles, allowing the daemon mode to operate efficiently by comparing timestamps before processing. The history management layer accumulates snapshots that power the 24h/7d/30d change calculations already stubbed in Epic 3's `ChangeMetrics` structure.

This epic directly addresses PRD requirements for incremental updates (FR25-FR29) and historical data management (FR30-FR34), establishing the foundation for the output generation and CLI operation in Epic 5.

## Objectives and Scope

**In Scope:**
- State file structure with `State` struct (`state.json`)
- State loading with graceful handling of missing/corrupted files
- State comparison logic for skip decisions (`ShouldProcess`)
- Atomic state file updates using temp-file-then-rename pattern
- Snapshot structure matching `aggregator.Snapshot` fields
- History loading from existing output file (`switchboard-oracle-data.json`)
- Snapshot deduplication by timestamp
- History retention with no automatic pruning (MVP keeps all)
- Unified `StateManager` component coordinating all operations

**Out of Scope:**
- Output file generation (Epic 5, Stories 5.1-5.4)
- CLI flag parsing and daemon mode (Epic 5, Stories 5.5-5.8)
- Automatic history pruning (explicitly deferred per PRD FR33)
- Prometheus metrics or health endpoints (post-MVP)
- Database or external storage backends

## System Architecture Alignment

This epic implements the `internal/storage` package as defined in the project structure:

```
internal/storage/
├── doc.go           # Package documentation (exists)
├── state.go         # StateManager implementation
├── state_test.go    # State tests
├── history.go       # HistoryManager implementation
├── history_test.go  # History tests
└── writer.go        # Atomic file writer utility
```

**Architectural Constraints:**
- Uses Go standard library only (`encoding/json`, `os`, `path/filepath`, `time`, `sort`)
- No external dependencies required for this epic
- Atomic file writes via temp file + `os.Rename()` (POSIX atomic on same filesystem)
- Context propagation for future cancellation support
- Structured logging with `slog` (ADR-004)
- Explicit error returns over exceptions (ADR-003)

**Data Flow:**
```
┌─────────────────────────────────────────────────────────────────┐
│                     Extraction Cycle                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────┐    ┌──────────────┐    ┌──────────────────────┐  │
│  │  state   │───►│ StateManager │───►│ ShouldProcess(ts)?   │  │
│  │  .json   │    │   .Load()    │    │ true = new data      │  │
│  └──────────┘    └──────────────┘    │ false = skip cycle   │  │
│                                       └──────────────────────┘  │
│                                                 │                │
│                                                 ▼                │
│  ┌────────────────────────────────┐   ┌─────────────────────┐  │
│  │ switchboard-oracle-data.json  │◄──│  HistoryManager     │  │
│  │   (historical[] array)        │   │  .LoadFromOutput()  │  │
│  └────────────────────────────────┘   └─────────────────────┘  │
│                                                 │                │
│                                                 ▼                │
│                                       ┌─────────────────────┐  │
│                                       │ Aggregator.Aggregate│  │
│                                       │   (with history)    │  │
│                                       └─────────────────────┘  │
│                                                 │                │
│                                                 ▼                │
│  ┌──────────┐    ┌──────────────┐    ┌─────────────────────┐  │
│  │  state   │◄───│ StateManager │◄───│ CreateSnapshot()    │  │
│  │  .json   │    │   .Save()    │    │ AppendSnapshot()    │  │
│  └──────────┘    └──────────────┘    └─────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Detailed Design

### Services and Modules

| Module | File | Responsibility | Inputs | Outputs |
|--------|------|----------------|--------|---------|
| StateManager | `state.go` | Load/save state, determine if processing needed | `outputDir` path, timestamp | `*State`, bool, error |
| HistoryManager | `history.go` | Load/append/deduplicate snapshots | Output file path, snapshots | `[]Snapshot`, error |
| WriteAtomic | `writer.go` | Atomic file writes | Path, data, permissions | error |

**StateManager Struct:**
```go
type StateManager struct {
    outputDir    string
    stateFile    string        // {outputDir}/state.json
    outputFile   string        // {outputDir}/switchboard-oracle-data.json
    logger       *slog.Logger
}
```

**Constructor:**
```go
func NewStateManager(outputDir string, logger *slog.Logger) *StateManager
```

### Data Models and Contracts

**State** (stored in `state.json`):
```go
type State struct {
    OracleName        string  `json:"oracle_name"`
    LastUpdated       int64   `json:"last_updated"`        // Unix timestamp
    LastUpdatedISO    string  `json:"last_updated_iso"`    // ISO 8601 for humans
    LastProtocolCount int     `json:"last_protocol_count"`
    LastTVS           float64 `json:"last_tvs"`
    SnapshotCount     int     `json:"snapshot_count"`
    OldestSnapshot    int64   `json:"oldest_snapshot"`
    NewestSnapshot    int64   `json:"newest_snapshot"`
}
```

**Snapshot** (reuses `aggregator.Snapshot`):
```go
// Already defined in internal/aggregator/models.go
type Snapshot struct {
    Timestamp     int64              `json:"timestamp"`
    Date          string             `json:"date"`           // YYYY-MM-DD
    TVS           float64            `json:"tvs"`
    TVSByChain    map[string]float64 `json:"tvs_by_chain"`
    ProtocolCount int                `json:"protocol_count"`
    ChainCount    int                `json:"chain_count"`
}
```

**Sentinel Errors:**
```go
var (
    ErrStateCorrupted = errors.New("state file corrupted")
)
```

### Entity Relationships

- **State → Output**: `state.json` fields (`LastUpdated`, `LastUpdatedISO`, `SnapshotCount`) summarize the most recent snapshot persisted to `switchboard-oracle-data.json` (the `outputFile` used by HistoryManager).
- **State ↔ History**: `OldestSnapshot` and `NewestSnapshot` mirror the bounds of the `historical[]` slice maintained by HistoryManager, enabling skip decisions without reloading the entire history.
- **HistoryManager → StateManager**: HistoryManager loads `historical[]`; after appending/deduping a new `Snapshot`, StateManager updates metadata and writes `state.json`.
- **Aggregator → HistoryManager**: Aggregator emits an `AggregationResult`; HistoryManager converts it to `Snapshot`, merges into `historical[]`, and returns the updated slice for downstream persistence.

### APIs and Interfaces

**StateManager Methods:**

| Method | Signature | Purpose |
|--------|-----------|---------|
| LoadState | `() (*State, error)` | Load state from disk, zero-value if missing |
| SaveState | `(state *State) error` | Atomic save to state.json |
| ShouldProcess | `(currentTS int64, state *State) bool` | Determine if new data available |
| UpdateState | `(oracleName string, ts int64, count int, tvs float64, snapshots []Snapshot) *State` | Create updated state from results |

**HistoryManager Methods:**

| Method | Signature | Purpose |
|--------|-----------|---------|
| LoadFromOutput | `(outputPath string) ([]Snapshot, error)` | Extract historical[] from output file |
| AppendSnapshot | `(history []Snapshot, snapshot Snapshot) []Snapshot` | Add snapshot, deduplicate |
| CreateSnapshot | `(result *AggregationResult) Snapshot` | Convert aggregation result to snapshot |

**WriteAtomic Function:**

| Function | Signature | Purpose |
|----------|-----------|---------|
| WriteAtomic | `(path string, data []byte, perm os.FileMode) error` | Temp file + rename atomic write |

### Workflows and Sequencing

**State Loading Flow:**
```
LoadState()
    │
    ├─► os.ReadFile(stateFile)
    │       │
    │       ├─► [os.ErrNotExist] → return &State{}, nil (first run)
    │       ├─► [Other error] → return nil, error
    │       └─► [Success] → continue
    │
    ├─► json.Unmarshal(data, &state)
    │       │
    │       ├─► [Error] → log.Warn("corrupted state") → return &State{}, nil
    │       └─► [Success] → return &state, nil
    │
    └─► Result: State struct or zero-value
```

**Skip Decision Flow:**
```
ShouldProcess(currentTS, state)
    │
    ├─► [state.LastUpdated == 0] → return true (first run)
    │
    ├─► [currentTS > state.LastUpdated]
    │       → log.Debug("new data available") → return true
    │
    ├─► [currentTS == state.LastUpdated]
    │       → log.Info("no new data") → return false
    │
    └─► [currentTS < state.LastUpdated]
            → log.Warn("clock skew detected") → return false
```

**Atomic Write Flow:**
```
WriteAtomic(path, data, perm)
    │
    ├─► os.MkdirAll(dir, 0755) if needed
    │
    ├─► os.CreateTemp(dir, ".tmp-*")
    │       │
    │       └─► [Error] → return error
    │
    ├─► tmpFile.Write(data)
    │       │
    │       └─► [Error] → cleanup tmp → return error
    │
    ├─► tmpFile.Sync()
    │
    ├─► tmpFile.Close()
    │
    ├─► os.Chmod(tmpPath, perm)
    │
    ├─► os.Rename(tmpPath, path)   ← Atomic!
    │       │
    │       └─► [Error] → cleanup tmp → return error
    │
    └─► return nil (success, defer cleanup skipped)
```

**History Append Flow:**
```
AppendSnapshot(history, newSnapshot)
    │
    ├─► For each existing snapshot:
    │       │
    │       └─► [timestamp == newSnapshot.Timestamp]
    │               → Replace in place → return history
    │
    ├─► history = append(history, newSnapshot)
    │
    ├─► sort.Slice(history, by timestamp ascending)
    │
    └─► return sorted history
```

## Non-Functional Requirements

### Performance

| Requirement | Target | Implementation |
|-------------|--------|----------------|
| State file load | < 10ms | Single `os.ReadFile` + `json.Unmarshal` |
| State file save | < 50ms | Atomic write includes fsync |
| History load | < 500ms for 1000 snapshots | Partial JSON parse (only `historical` field) |
| Skip decision | O(1) | Simple timestamp comparison |

**Measurable Targets:**
- State operations complete within 100ms under normal conditions
- History operations scale linearly with snapshot count
- No blocking operations outside file I/O

### Security

| Concern | Mitigation |
|---------|------------|
| File permissions | State and output files written with 0644 |
| Path traversal | All paths derived from configured `outputDir` |
| No sensitive data | State contains only timestamps, counts, TVS values |
| Atomic updates | Prevents partial/corrupted state files |

### Reliability/Availability

| Requirement | Implementation |
|-------------|----------------|
| NFR7: Preserve last good output | State save only after successful extraction |
| NFR8: Corrupted state recovery | Returns zero-value state, logs warning |
| NFR10: Graceful degradation | Missing history → empty slice, extraction continues |
| FR28: Graceful recovery | Corrupted state treated as first run |

**Error Recovery:**
- Missing state file → First run behavior (process everything)
- Corrupted state file → Warning logged, first run behavior
- Missing output file → Empty history, snapshots start fresh
- Corrupted output file → Warning logged, empty history

### Observability

| Log Event | Level | Attributes |
|-----------|-------|------------|
| State loaded | Debug | `oracle_name`, `last_updated`, `snapshot_count` |
| No state file | Debug | `path`, `reason: "first run"` |
| State corrupted | Warn | `path`, `error` |
| No new data (skip) | Info | `current_ts`, `last_ts` |
| New data available | Debug | `current_ts`, `last_ts`, `delta_seconds` |
| Clock skew detected | Warn | `current_ts`, `last_ts` |
| State saved | Info | `timestamp`, `protocol_count`, `tvs` |
| History loaded | Debug | `snapshot_count`, `oldest`, `newest` |
| Snapshot appended | Debug | `timestamp`, `total_snapshots` |
| Duplicate replaced | Debug | `timestamp` |

**Sample log lines (slog key/value):**
- `level=DEBUG msg="state loaded" oracle_name=defillama last_updated=1701302400 snapshot_count=42`
- `level=INFO msg="no new data" current_ts=1701388800 last_ts=1701388800`
- `level=WARN msg="clock skew detected" current_ts=1701300000 last_ts=1701388800`

## Dependencies and Integrations

### Go Module Dependencies

**No new dependencies required.** This epic uses only Go standard library (built and tested with Go 1.22.x):

```go
import (
    "encoding/json"
    "errors"
    "fmt"
    "log/slog"
    "os"
    "path/filepath"
    "sort"
    "time"
)
```

### Standard Library Packages Used

| Package | Purpose |
|---------|---------|
| `encoding/json` | JSON marshaling/unmarshaling |
| `errors` | Sentinel errors |
| `fmt` | Error formatting |
| `log/slog` | Structured logging |
| `os` | File I/O, temp files, rename |
| `path/filepath` | Path manipulation |
| `sort` | Snapshot sorting |
| `time` | Timestamp formatting |

### Internal Dependencies

| Depends On | From Epic | Purpose |
|------------|-----------|---------|
| `internal/aggregator.Snapshot` | Epic 3 | Snapshot data structure |
| `internal/aggregator.AggregationResult` | Epic 3 | Source for creating snapshots |
| `*slog.Logger` | Epic 1 | Structured logging |
| `internal/config.Config` | Epic 1 | Output directory configuration |

### Downstream Consumers

| Consumer | Epic | Purpose |
|----------|------|---------|
| `cmd/extractor/main.go` | Epic 5 | Orchestrates state/history in extraction cycle |
| `internal/storage/writer.go` | Epic 5 | Uses WriteAtomic for output files |

## Acceptance Criteria (Authoritative)

### AC-4.1: State File Structure and Loading (Story 4.1)
1. `LoadState()` reads from `{outputDir}/state.json`
2. Returns `*State` with all fields populated on success
3. Returns zero-value `*State` when file doesn't exist (not error)
4. Returns zero-value `*State` with warning log when file is corrupted
5. `State` struct contains: `OracleName`, `LastUpdated`, `LastUpdatedISO`, `LastProtocolCount`, `LastTVS`

### AC-4.2: State Comparison for Skip Logic (Story 4.2)
1. `ShouldProcess(currentTS, state)` returns `true` when `state.LastUpdated == 0`
2. Returns `true` when `currentTS > state.LastUpdated`
3. Returns `false` when `currentTS == state.LastUpdated` with info log
4. Returns `false` when `currentTS < state.LastUpdated` with warn log (clock skew)
5. All decisions logged with appropriate level and attributes

### AC-4.3: Atomic State File Updates (Story 4.3)
1. `SaveState(state)` writes to temp file first
2. Temp file renamed to `state.json` atomically
3. Directory created if doesn't exist
4. Temp file cleaned up on any error
5. Original state preserved on write failure
6. Success logged with `timestamp`, `protocol_count`, `tvs`

### AC-4.4: Historical Snapshot Structure (Story 4.4)
1. `CreateSnapshot(result)` creates `Snapshot` from `AggregationResult`
2. `Timestamp` matches `result.Timestamp`
3. `Date` formatted as `YYYY-MM-DD`
4. `TVS`, `TVSByChain`, `ProtocolCount`, `ChainCount` match result

### AC-4.5: History Loading from Output File (Story 4.5)
1. `LoadFromOutput(path)` extracts `historical[]` from output JSON
2. Returns empty slice when file doesn't exist (not error)
3. Returns empty slice when `historical` is missing/empty
4. Returns empty slice with warning when file is corrupted
5. Snapshots returned sorted by timestamp ascending

### AC-4.6: Snapshot Deduplication (Story 4.6)
1. `AppendSnapshot(history, snapshot)` replaces existing snapshot with same timestamp
2. History length unchanged when replacing
3. New snapshot appended when timestamp is unique
4. History always sorted by timestamp ascending after operation

### AC-4.7: History Retention (Story 4.7)
1. No automatic pruning logic exists
2. All snapshots retained regardless of age
3. Code comment documents "pruning may be added in future version"

### AC-4.8: Unified StateManager (Story 4.8)
1. `NewStateManager(outputDir, logger)` creates configured instance
2. State file path: `{outputDir}/state.json`
3. Output file path: `{outputDir}/switchboard-oracle-data.json`
4. All methods available: `LoadState`, `SaveState`, `ShouldProcess`, `UpdateState`
5. `LoadFromOutput` and `AppendSnapshot` available via HistoryManager or embedded

## Traceability Mapping

| AC | FR | Spec Section | Component/API | Test Idea |
|----|-----|--------------|---------------|-----------|
| AC-4.1 | FR25, FR28 | Data Models, Workflows | `StateManager.LoadState()` | Unit: missing file returns zero; Unit: corrupted file returns zero with log |
| AC-4.2 | FR26, FR27 | Workflows | `StateManager.ShouldProcess()` | Unit: table-driven tests for all timestamp scenarios |
| AC-4.3 | FR29 | Workflows | `StateManager.SaveState()`, `WriteAtomic()` | Unit: verify atomic rename; Unit: verify cleanup on error |
| AC-4.4 | FR31 | Data Models | `CreateSnapshot()` | Unit: verify all fields populated from AggregationResult |
| AC-4.5 | FR34 | Workflows | `HistoryManager.LoadFromOutput()` | Unit: missing file; Unit: corrupted file; Unit: valid file |
| AC-4.6 | FR32 | Workflows | `HistoryManager.AppendSnapshot()` | Unit: duplicate replacement; Unit: new append; Unit: sort order |
| AC-4.7 | FR33 | NFRs | N/A (absence of code) | Review: no prune function exists |
| AC-4.8 | FR25-FR34 | Services | `StateManager` struct | Integration: full cycle load→process→save |

### FR to Story Mapping

| FR | Story | Description |
|----|-------|-------------|
| FR25 | 4.1, 4.3 | Track last processed timestamp in state file |
| FR26 | 4.2 | Compare latest API timestamp against last processed |
| FR27 | 4.2 | Skip processing when no new data |
| FR28 | 4.1 | Recover gracefully from corrupted state file |
| FR29 | 4.3 | Update state file atomically after successful extraction |
| FR30 | 4.4 | Maintain historical snapshots over time |
| FR31 | 4.4 | Store timestamp, date, TVS, TVSByChain, counts per snapshot |
| FR32 | 4.6 | Deduplicate snapshots with identical timestamps |
| FR33 | 4.7 | Retain all historical snapshots (no pruning) |
| FR34 | 4.5 | Load existing history from output file on startup |

## Risks, Assumptions, Open Questions

### Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Disk full during atomic write | Low | Medium | Error returned, original preserved, logged |
| Output file grows unbounded | Medium | Low | MVP accepts this; post-MVP can add optional pruning |
| Race condition on state file | Low | Low | Single-threaded extraction; future: file locking |
| Clock skew on system | Low | Low | Detection + warning log; no data loss |

### Assumptions

| Assumption | Validation |
|------------|------------|
| `os.Rename` is atomic on target filesystem | POSIX guarantee; documented limitation on cross-filesystem |
| Output directory is writable | Verified at startup in Epic 5 |
| Snapshot count won't exceed memory limits | ~1000 snapshots × ~1KB = ~1MB; acceptable for years |
| `aggregator.Snapshot` struct is stable | Defined in Epic 3, no changes expected |

### Open Questions

| Question | Owner | Resolution Path |
|----------|-------|-----------------|
| Should history be stored in separate file? | Developer | Defer; current design matches PRD output schema |
| File locking for concurrent access? | Developer | Defer to post-MVP; single process assumed |

## Test Strategy Summary

### Test Types

| Type | Location | Coverage Target |
|------|----------|-----------------|
| Unit Tests | `internal/storage/*_test.go` | All public methods, error paths |
| Table-Driven Tests | All test files | Multiple scenarios per function |
| Integration Tests | `internal/storage/state_test.go` | Full load→modify→save cycle |

### Key Test Scenarios

**State Loading (4.1):**
- File exists with valid JSON → State populated
- File doesn't exist → Zero-value State, no error
- File contains invalid JSON → Zero-value State, warning logged
- File contains partial JSON → Zero-value State, warning logged

**Skip Logic (4.2):**
| CurrentTS | LastUpdated | Expected | Log Level |
|-----------|-------------|----------|-----------|
| 1700003600 | 0 | true | Debug |
| 1700003600 | 1700000000 | true | Debug |
| 1700000000 | 1700000000 | false | Info |
| 1700000000 | 1700003600 | false | Warn |

**Atomic Write (4.3):**
- Successful write → file contains correct data
- Write to new directory → directory created, file written
- Disk error during write → temp cleaned, original preserved
- Permission error → returns error, no partial file

**Snapshot Creation (4.4):**
- Valid AggregationResult → Snapshot with all fields
- Empty TVSByChain → Empty map (not nil)
- Timestamp formatting → Correct YYYY-MM-DD

**History Loading (4.5):**
- Valid output file → Snapshots extracted, sorted
- Missing file → Empty slice
- Corrupted JSON → Empty slice, warning
- Missing `historical` field → Empty slice

**Deduplication (4.6):**
- Duplicate timestamp → Replace, same length
- New timestamp → Append, length + 1
- Multiple operations → Maintained sort order

### Coverage Requirements

| Component | Target |
|-----------|--------|
| `state.go` | 90%+ line coverage |
| `history.go` | 90%+ line coverage |
| `writer.go` | 90%+ line coverage |
| Error paths | All paths tested |
| Edge cases | Empty inputs, nil handling |

### Test Fixtures

| Fixture | Location | Purpose |
|---------|----------|---------|
| `valid_state.json` | `testdata/` | Valid state file |
| `corrupted_state.json` | `testdata/` | Invalid JSON for error testing |
| `output_with_history.json` | `testdata/` | Output file with historical data |
| `output_no_history.json` | `testdata/` | Output file without historical field |
