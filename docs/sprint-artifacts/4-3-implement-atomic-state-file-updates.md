# Story 4.3: Implement Atomic State File Updates

Status: done

## Story

As a **developer**,
I want **state updates written atomically**,
so that **interrupted writes don't corrupt the state file**.

## Acceptance Criteria

Source: [Source: docs/sprint-artifacts/tech-spec-epic-4.md#AC-4.3] / [Source: docs/epics/epic-4-state-history-management.md#Story-4.3]

1. **Given** a valid `*State` struct **When** `SaveState(state *State)` is called **Then** state is written to a temp file first (`.tmp-*` pattern in same directory) **And** temp file is renamed to `state.json` atomically

2. **Given** the output directory doesn't exist **When** `SaveState` is called **Then** the directory is created with `0755` permissions **And** state file is written successfully

3. **Given** a write failure (disk full, permissions) **When** `SaveState` fails **Then** an error is returned with descriptive message **And** any temp file is cleaned up **And** original state file (if exists) is preserved

4. **Given** successful state save **When** operation completes **Then** info log: "state saved" with attributes: `timestamp`, `protocol_count`, `tvs`

5. **Given** a `WriteAtomic(path string, data []byte, perm os.FileMode)` utility function **When** called with valid inputs **Then** writes to temp file, syncs, and atomically renames to target **And** can be reused for other atomic file operations

## Tasks / Subtasks

- [x] Task 1: Implement WriteAtomic utility function (AC: 5)
  - [x] 1.1: Create `internal/storage/writer.go` with `WriteAtomic(path string, data []byte, perm os.FileMode) error`
  - [x] 1.2: Use `os.CreateTemp(dir, ".tmp-*")` to create temp file in same directory as target
  - [x] 1.3: Write data to temp file with `tmpFile.Write(data)`
  - [x] 1.4: Call `tmpFile.Sync()` to ensure data is flushed to disk
  - [x] 1.5: Close temp file
  - [x] 1.6: Set permissions with `os.Chmod(tmpPath, perm)`
  - [x] 1.7: Atomically rename with `os.Rename(tmpPath, path)`
  - [x] 1.8: Use `defer` with cleanup flag to ensure temp file cleanup on error
  - [x] 1.9: Add doc comment explaining atomic write guarantees and POSIX limitations

- [x] Task 2: Implement SaveState method on StateManager (AC: 1, 2, 4)
  - [x] 2.1: Add `func (sm *StateManager) SaveState(state *State) error` to `internal/storage/state.go`
  - [x] 2.2: Marshal state to JSON with indentation (`json.MarshalIndent`)
  - [x] 2.3: Ensure output directory exists: `os.MkdirAll(sm.outputDir, 0755)`
  - [x] 2.4: Call `WriteAtomic(sm.stateFile, data, 0644)`
  - [x] 2.5: Log info "state saved" with `timestamp`, `protocol_count`, `tvs` attributes on success
  - [x] 2.6: Return wrapped error with context on failure

- [x] Task 3: Implement error handling and cleanup (AC: 3)
  - [x] 3.1: In WriteAtomic: use `defer` with `cleanupNeeded` flag to remove temp file on error paths
  - [x] 3.2: In WriteAtomic: wrap errors with context (e.g., "create temp file", "write data", "sync", "rename")
  - [x] 3.3: Verify original file untouched when write fails (atomic rename only happens on success)

- [x] Task 4: Write unit tests for WriteAtomic (AC: 3, 5)
  - [x] 4.1: Create `internal/storage/writer_test.go`
  - [x] 4.2: Test: successful write creates file with correct content
  - [x] 4.3: Test: write to new directory creates directory
  - [x] 4.4: Test: correct permissions applied to final file
  - [x] 4.5: Test: temp file cleaned up on write error (simulate with read-only dir)
  - [x] 4.6: Test: original file preserved on rename error (if feasible to simulate)

- [x] Task 5: Write unit tests for SaveState (AC: 1, 2, 4)
  - [x] 5.1: Add tests to `internal/storage/state_test.go`
  - [x] 5.2: Test: SaveState creates state.json with correct JSON content
  - [x] 5.3: Test: SaveState creates directory if not exists
  - [x] 5.4: Test: SaveState logs "state saved" with correct attributes
  - [x] 5.5: Test: SaveState returns error on write failure

- [x] Task 6: Verification (AC: all)
  - [x] 6.1: Run `go build ./...` and verify success
  - [x] 6.2: Run `go test ./internal/storage/...` and verify all pass
  - [x] 6.3: Run `make lint` and verify no errors

## Dev Notes

### Technical Guidance

- **Package Location:** `internal/storage/state.go` (add SaveState method), `internal/storage/writer.go` (new utility)
- **Pattern:** Temp file + atomic rename per [Source: docs/architecture/implementation-patterns.md#Atomic-File-Writes]
- **Dependencies:** Go stdlib only (`encoding/json`, `os`, `path/filepath`)
- **Error Handling:** Explicit returns with wrapped errors per ADR-003
- **Logging:** ADR-004 structured logging with slog, Info level for success

### Atomic Write Pattern (from Architecture)

```go
// WriteAtomic writes data to path atomically using temp file + rename.
// This is atomic on POSIX systems when temp and target are on the same filesystem.
func WriteAtomic(path string, data []byte, perm os.FileMode) error {
    dir := filepath.Dir(path)

    // Ensure directory exists
    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Errorf("create directory %s: %w", dir, err)
    }

    // Create temp file in same directory for atomic rename
    tmpFile, err := os.CreateTemp(dir, ".tmp-*")
    if err != nil {
        return fmt.Errorf("create temp file: %w", err)
    }
    tmpPath := tmpFile.Name()

    // Cleanup on error
    cleanupNeeded := true
    defer func() {
        if cleanupNeeded {
            os.Remove(tmpPath)
        }
    }()

    // Write data
    if _, err := tmpFile.Write(data); err != nil {
        tmpFile.Close()
        return fmt.Errorf("write data: %w", err)
    }

    // Sync to ensure data on disk
    if err := tmpFile.Sync(); err != nil {
        tmpFile.Close()
        return fmt.Errorf("sync file: %w", err)
    }

    if err := tmpFile.Close(); err != nil {
        return fmt.Errorf("close temp file: %w", err)
    }

    // Set permissions before rename
    if err := os.Chmod(tmpPath, perm); err != nil {
        return fmt.Errorf("set permissions: %w", err)
    }

    // Atomic rename
    if err := os.Rename(tmpPath, path); err != nil {
        return fmt.Errorf("rename %s to %s: %w", tmpPath, path, err)
    }

    cleanupNeeded = false
    return nil
}
```

### SaveState Method Pattern

```go
// SaveState persists the state atomically to disk.
func (sm *StateManager) SaveState(state *State) error {
    data, err := json.MarshalIndent(state, "", "  ")
    if err != nil {
        return fmt.Errorf("marshal state: %w", err)
    }

    if err := WriteAtomic(sm.stateFile, data, 0644); err != nil {
        return fmt.Errorf("write state file: %w", err)
    }

    sm.logger.Info("state saved",
        "timestamp", state.LastUpdated,
        "protocol_count", state.LastProtocolCount,
        "tvs", state.LastTVS,
    )

    return nil
}
```

### Project Structure Notes

- **New File:** `internal/storage/writer.go` for reusable `WriteAtomic` function
- **Existing File:** `internal/storage/state.go` - add `SaveState` method
- **Test Files:** `internal/storage/writer_test.go` (new), `internal/storage/state_test.go` (extend)
- **Structure Compliance:** All changes scoped to `internal/storage/` per [Source: docs/architecture/project-structure.md]

### Learnings from Previous Story

**From Story 4-2-implement-state-comparison-for-skip-logic (Status: done)**

- **StateManager Pattern:** Constructor and methods established; `SaveState` follows same method-on-struct pattern
- **Structured Logging:** Debug/Info/Warn levels with key-value pairs; use Info for "state saved" per tech spec
- **Table-Driven Tests:** Pattern established in `state_test.go`; extend with SaveState tests
- **Files to Reference:** `internal/storage/state.go` (84-111 for logging style), `internal/storage/state_test.go` (test patterns)
- **Review Outcome:** Approved with no action items - clean patterns to follow

[Source: docs/sprint-artifacts/4-2-implement-state-comparison-for-skip-logic.md#Dev-Agent-Record]

### Testing Standards

- Follow Go table-driven tests pattern per [Source: docs/architecture/testing-strategy.md#Test-Organization]
- Use temp directories (`t.TempDir()`) for isolated file system tests
- Test both success and error paths for WriteAtomic
- Verify log output using log capture pattern from 4.2
- Verify file permissions with `os.Stat()` and `Mode().Perm()`

### Smoke Test Guide

**Manual verification after implementation:**

1. Create a test program or use existing test:
   ```go
   sm := storage.NewStateManager("/tmp/test-state", nil)
   state := &storage.State{
       OracleName:        "defillama",
       LastUpdated:       1700000000,
       LastUpdatedISO:    "2023-11-15T00:00:00Z",
       LastProtocolCount: 42,
       LastTVS:           1000000.0,
   }

   // Test SaveState
   err := sm.SaveState(state)
   if err != nil {
       fmt.Printf("SaveState error: %v\n", err)
   }

   // Verify file exists and content
   data, _ := os.ReadFile("/tmp/test-state/state.json")
   fmt.Printf("File content:\n%s\n", string(data))

   // Verify permissions
   info, _ := os.Stat("/tmp/test-state/state.json")
   fmt.Printf("Permissions: %v (expected: -rw-r--r--)\n", info.Mode())
   ```

2. Run: `go test -v ./internal/storage/... -run TestSaveState`

3. Verify: File created atomically, correct JSON content, correct permissions, "state saved" log appears

### FR Coverage

| FR | Description | Satisfied By |
|----|-------------|--------------|
| FR29 | Update state file atomically after successful extraction | `SaveState` using `WriteAtomic` temp-file-then-rename |

### References

- [Source: docs/prd.md#FR29] - Atomic state file updates requirement
- [Source: docs/epics/epic-4-state-history-management.md#story-43] - Story definition and acceptance criteria
- [Source: docs/sprint-artifacts/tech-spec-epic-4.md#AC-4.3] - Authoritative acceptance criteria
- [Source: docs/sprint-artifacts/tech-spec-epic-4.md#workflows-and-sequencing] - Atomic write flow diagram
- [Source: docs/architecture/implementation-patterns.md#Atomic-File-Writes] - Atomic file write pattern
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-003] - Error handling patterns
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-004] - Structured logging with slog
- [Source: docs/architecture/testing-strategy.md#Test-Organization] - Testing standards
- [Source: docs/sprint-artifacts/4-2-implement-state-comparison-for-skip-logic.md] - Previous story patterns

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/4-3-implement-atomic-state-file-updates.context.xml

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References


- Plan (AC1-5): implement WriteAtomic + SaveState per context; cover cleanup/error handling; add tests for success/error/logs.
- Execution: writer.go with atomic temp+rename+chmod+cleanup; SaveState with MarshalIndent, MkdirAll 0755, WriteAtomic, slog info attrs timestamp/protocol_count/tvs.
- Tests: writer_test.go covers success, dir creation, cleanup/original preservation; state_test.go covers save success, dir creation, log payload, error on readonly dir.
- Verification: go build ./..., go test ./internal/storage/..., make lint.

### Completion Notes List

- AC1-5 satisfied via WriteAtomic utility and SaveState method using temp-file-then-rename, chmod, cleanup, slog info log with required attrs.
- Error handling wrapped with context; directories created 0755 before atomic writes; original preserved on failure via cleanup flag.
- Added unit tests for WriteAtomic and SaveState (content, permissions, directory creation, log attributes, error propagation); all pass.
- Validations executed: `go build ./...`; `go test ./internal/storage/...`; `make lint`.

### File List

- internal/storage/writer.go
- internal/storage/state.go
- internal/storage/writer_test.go
- internal/storage/state_test.go
- docs/sprint-artifacts/sprint-status.yaml
- docs/sprint-artifacts/4-3-implement-atomic-state-file-updates.md

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | Developer Agent (Amelia) | Senior Developer Review (AI) appended; status set to done |
| 2025-11-30 | Amelia (Dev Agent) | Implemented atomic state save, tests, and marked story ready for review |
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-4 and tech-spec-epic-4.md |

## Senior Developer Review (AI)

Reviewer: BMad  
Date: 2025-11-30  
Outcome: Approve (all ACs implemented and tested; fsync hardening added)

### Summary
- AC1–AC5 implemented via `WriteAtomic` and `SaveState`; atomic temp+rename with cleanup and structured logging.
- Tests cover success/error paths for writes and SaveState logging; `go test ./...` and `make lint` passing.
- No security issues observed; alignment with ADR-002 (atomic writes) and ADR-004 (slog).
- Added hardening: fsync parent dir after rename for crash durability.

### Key Findings
- None blocking. Durability hardening implemented (fsync parent dir).

### Acceptance Criteria Coverage (5/5 implemented)
| AC# | Description | Status | Evidence |
| --- | ----------- | ------ | -------- |
| 1 | SaveState writes via temp file then atomic rename | Implemented | internal/storage/writer.go:12-53; internal/storage/state.go:81-99 |
| 2 | Creates output dir 0755 before writing | Implemented | internal/storage/writer.go:13-16; internal/storage/state.go:87-93; internal/storage/writer_test.go:43-61 |
| 3 | Errors return with temp cleanup; original preserved | Implemented | internal/storage/writer.go:24-50; internal/storage/writer_test.go:64-100 |
| 4 | Success logs "state saved" with timestamp, protocol_count, tvs | Implemented | internal/storage/state.go:95-99; internal/storage/state_test.go:311-345 |
| 5 | Reusable WriteAtomic syncs then renames atomically | Implemented | internal/storage/writer.go:18-50; internal/storage/writer_test.go:9-40 |

### Task Validation
| Task | Marked As | Verified As | Evidence |
| ---- | --------- | ----------- | -------- |
| Task 1: WriteAtomic | [x] | Verified complete | internal/storage/writer.go:12-53; internal/storage/writer_test.go:9-40 |
| Task 2: SaveState | [x] | Verified complete | internal/storage/state.go:81-101; internal/storage/state_test.go:254-308 |
| Task 3: Error handling & cleanup | [x] | Verified complete | internal/storage/writer.go:24-50; internal/storage/writer_test.go:64-100 |
| Task 4: Tests for WriteAtomic | [x] | Verified complete | internal/storage/writer_test.go:9-100 |
| Task 5: Tests for SaveState | [x] | Verified complete | internal/storage/state_test.go:254-360 |
| Task 6: Verification commands | [x] | Verified complete | `go test ./...` (2025-11-30); `make lint` (2025-11-30) |

### Test Coverage and Gaps
- `go test ./...` passing (includes storage suite).
- `make lint` passing.
- No additional gaps noted for story scope.

### Architectural Alignment
- Follows ADR-002 atomic write pattern (`WriteAtomic` temp + rename).
- Uses ADR-004 structured logging with slog at Info for successful save.

### Security Notes
- No secrets or unsafe file permissions observed (state files 0644, dirs 0755).

### Best-Practices and References
- Atomic write pattern: docs/architecture/implementation-patterns.md#Atomic-File-Writes.
- Structured logging: docs/architecture/architecture-decision-records-adrs.md#ADR-004.
- Stack: Go 1.24, stdlib only; tests table-driven.

### Action Items
**Code Changes Required:**
- [x] [Low] Fsync parent directory after rename in `internal/storage/writer.go` to harden against sudden power loss (done).
