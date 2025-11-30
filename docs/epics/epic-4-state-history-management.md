# Epic 4: State & History Management

**Goal:** Implement incremental update tracking and historical snapshot management to enable efficient polling and time-series metrics.

**User Value:** After this epic, the system runs incrementally (skipping when no new data), maintains historical trends automatically, and enables 24h/7d/30d change calculations.

**FRs Covered:** FR25, FR26, FR27, FR28, FR29, FR30, FR31, FR32, FR33, FR34

---

## Story 4.1: Implement State File Structure and Loading

As a **developer**,
I want **to load the last extraction state from a JSON file**,
So that **I can determine if new data is available**.

**Acceptance Criteria:**

**Given** a state file exists at `data/state.json`
**When** `LoadState(path string)` is called
**Then** a `State` struct is returned containing:
  - `OracleName`: the oracle being tracked
  - `LastUpdated`: Unix timestamp of last processed data
  - `LastUpdatedISO`: human-readable ISO 8601 timestamp
  - `LastProtocolCount`: number of protocols in last extraction
  - `LastTVS`: total TVS from last extraction

**Given** no state file exists
**When** `LoadState` is called
**Then** a zero-value `State` is returned (not an error)
**And** `LastUpdated` = 0 indicates first run

**Given** a corrupted/invalid state file
**When** `LoadState` is called
**Then** a warning is logged
**And** a zero-value `State` is returned (graceful recovery, FR28)
**And** extraction proceeds as if first run

**Prerequisites:** Story 1.1 (project structure)

**Technical Notes:**
- Package: `internal/storage/state.go`
- `State` struct in `internal/models/output.go`
- Use `os.ReadFile()` + `json.Unmarshal()`
- Handle `os.ErrNotExist` gracefully
- Reference: 6-incremental-update-strategy.md, data-architecture.md

---

## Story 4.2: Implement State Comparison for Skip Logic

As a **developer**,
I want **to compare current API timestamp against last processed timestamp**,
So that **I can skip processing when no new data is available**.

**Acceptance Criteria:**

**Given** current API timestamp = 1700000000 and state.LastUpdated = 1700000000
**When** `ShouldProcess(currentTimestamp, state)` is called
**Then** returns `false` (no new data)
**And** info log: "skipping extraction, no new data available"

**Given** current API timestamp = 1700003600 and state.LastUpdated = 1700000000
**When** `ShouldProcess` is called
**Then** returns `true` (new data available)
**And** debug log: "new data available, proceeding with extraction"

**Given** state.LastUpdated = 0 (first run)
**When** `ShouldProcess` is called with any timestamp
**Then** returns `true` (always process on first run)

**Given** current timestamp is OLDER than last processed (clock skew or API issue)
**When** `ShouldProcess` is called
**Then** returns `false`
**And** warn log: "API timestamp older than last processed, possible clock skew"

**Prerequisites:** Story 4.1

**Technical Notes:**
- Package: `internal/storage/state.go`
- Function: `func (s *StateManager) ShouldProcess(currentTS int64) bool`
- Compare Unix timestamps directly
- Reference: FR26, FR27

---

## Story 4.3: Implement Atomic State File Updates

As a **developer**,
I want **state updates written atomically**,
So that **interrupted writes don't corrupt the state file**.

**Acceptance Criteria:**

**Given** a successful extraction with new data
**When** `SaveState(state State, path string)` is called
**Then** state is written to a temp file first (`state.json.tmp`)
**And** temp file is renamed to target (`state.json`)
**And** the operation is atomic (no partial writes)

**Given** the output directory doesn't exist
**When** `SaveState` is called
**Then** the directory is created
**And** state file is written successfully

**Given** a write failure (disk full, permissions)
**When** `SaveState` fails
**Then** an error is returned with descriptive message
**And** any temp file is cleaned up
**And** original state file (if exists) is preserved

**Given** successful state save
**When** operation completes
**Then** info log: "state saved" with `timestamp`, `protocol_count`, `tvs` attributes

**Prerequisites:** Story 4.1

**Technical Notes:**
- Package: `internal/storage/state.go`
- Use `os.CreateTemp()` in same directory for atomic rename
- `os.Rename()` is atomic on POSIX systems
- Clean up temp file in defer on error
- Reference: FR29, implementation-patterns.md "Atomic File Writes"

---

## Story 4.4: Implement Historical Snapshot Structure

As a **developer**,
I want **historical snapshots stored with required fields**,
So that **I can track TVS trends over time**.

**Acceptance Criteria:**

**Given** current extraction results
**When** `CreateSnapshot(result *AggregationResult)` is called
**Then** a `Snapshot` struct is created with:
  - `Timestamp`: Unix timestamp
  - `Date`: ISO 8601 date string (YYYY-MM-DD)
  - `TVS`: total value secured
  - `TVSByChain`: map of chain → TVS
  - `ProtocolCount`: number of protocols
  - `ChainCount`: number of active chains

**Given** snapshot creation
**When** fields are populated
**Then** all fields match the current aggregation result exactly

**Prerequisites:** Story 3.7 (aggregation result)

**Technical Notes:**
- Package: `internal/storage/history.go`
- `Snapshot` struct in `internal/models/snapshot.go`
- Use `time.Unix(ts, 0).Format("2006-01-02")` for date
- Reference: FR31, data-architecture.md

---

## Story 4.5: Implement History Loading from Output File

As a **developer**,
I want **existing history loaded from the output file on startup**,
So that **historical data is preserved across runs**.

**Acceptance Criteria:**

**Given** output file `switchboard-oracle-data.json` exists with `historical` array
**When** `LoadHistory(outputPath string)` is called
**Then** the `historical` array is extracted and returned as `[]Snapshot`
**And** snapshots are sorted by timestamp ascending (oldest first)

**Given** output file doesn't exist
**When** `LoadHistory` is called
**Then** empty slice is returned (not an error)
**And** debug log: "no existing history found, starting fresh"

**Given** output file exists but `historical` is empty or missing
**When** `LoadHistory` is called
**Then** empty slice is returned

**Given** output file is corrupted
**When** `LoadHistory` is called
**Then** warn log: "failed to load history, starting fresh"
**And** empty slice is returned (graceful degradation)

**Prerequisites:** Story 4.4

**Technical Notes:**
- Package: `internal/storage/history.go`
- Only load the `historical` field, not entire file
- Use `json.RawMessage` for partial parsing if needed
- Reference: FR34

---

## Story 4.6: Implement Snapshot Deduplication

As a **developer**,
I want **duplicate snapshots prevented**,
So that **history doesn't contain redundant entries**.

**Acceptance Criteria:**

**Given** existing history with snapshot at timestamp 1700000000
**When** `AppendSnapshot(history, newSnapshot)` is called with same timestamp
**Then** the new snapshot replaces the existing one (update in place)
**And** history length remains unchanged

**Given** existing history with snapshots at [1700000000, 1700003600]
**When** `AppendSnapshot` is called with timestamp 1700007200
**Then** new snapshot is appended
**And** history length increases by 1

**Given** history is unsorted after operations
**When** history is finalized
**Then** snapshots are sorted by timestamp ascending

**Prerequisites:** Story 4.5

**Technical Notes:**
- Package: `internal/storage/history.go`
- Function: `func (h *HistoryManager) AppendSnapshot(snapshot Snapshot) []Snapshot`
- Check for duplicate by timestamp before appending
- Use `sort.Slice()` to maintain order
- Reference: FR32

---

## Story 4.7: Implement History Retention (Keep All)

As a **developer**,
I want **all historical snapshots retained without pruning**,
So that **complete history is available for analysis**.

**Acceptance Criteria:**

**Given** history with 1000 snapshots spanning 90+ days
**When** a new snapshot is added
**Then** all existing snapshots are retained
**And** new snapshot is appended
**And** no automatic pruning occurs

**Given** MVP requirements
**When** history management is implemented
**Then** there is NO automatic pruning logic (FR33 - retain all)
**And** a comment notes "pruning may be added in future version"

**Prerequisites:** Story 4.6

**Technical Notes:**
- Package: `internal/storage/history.go`
- MVP explicitly requires NO pruning per PRD
- Future: could add configurable retention window
- Reference: FR33

---

## Story 4.8: Build State Manager Component

As a **developer**,
I want **a unified StateManager that handles all state operations**,
So that **I have a clean interface for incremental updates**.

**Acceptance Criteria:**

**Given** configuration with output directory
**When** `NewStateManager(cfg)` is called
**Then** a `StateManager` is created with paths configured:
  - State file: `{output_dir}/state.json`
  - Output file: `{output_dir}/switchboard-oracle-data.json`

**Given** a StateManager instance
**When** extraction cycle starts
**Then** `LoadState()` returns current state
**And** `LoadHistory()` returns existing snapshots
**And** `ShouldProcess(timestamp)` determines if processing needed

**Given** a successful extraction
**When** `SaveState(state)` and `AppendSnapshot(snapshot)` are called
**Then** both operations complete atomically
**And** state and history are consistent

**Prerequisites:** Stories 4.1-4.7

**Technical Notes:**
- Package: `internal/storage/state.go`
- Combine state and history management
- `StateManager` struct with all required methods
- Reference: fr-category-to-architecture-mapping.md

---
