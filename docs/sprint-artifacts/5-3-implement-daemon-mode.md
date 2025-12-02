# Story 5.3: Implement Daemon Mode and Complete Main Entry Point

Status: ready-for-dev

## Story

As an **operator**,
I want **continuous daemon operation with graceful shutdown**,
so that **data stays automatically updated in production**.

## Acceptance Criteria

Source: [Source: docs/sprint-artifacts/tech-spec-epic-5.md#Story-5.3]

**AC1: Daemon Mode with Scheduler**
**Given** daemon mode (no `--once` flag) with `scheduler.interval: 2h`
**When** application starts
**Then** extraction runs on schedule every 2 hours
**And** log: "daemon started, interval: 2h"

**AC2: Start Immediately True**
**Given** `scheduler.start_immediately: true`
**When** daemon starts
**Then** first extraction runs immediately, subsequent follow interval

**AC3: Start Immediately False**
**Given** `scheduler.start_immediately: false`
**When** daemon starts
**Then** first extraction waits for interval

**AC4: Next Extraction Log**
**Given** extraction cycle completes in daemon mode
**When** waiting for next interval
**Then** log: "next extraction at {timestamp}"

**AC5: Daemon Error Recovery**
**Given** extraction fails in daemon mode
**When** error occurs
**Then** error is logged, daemon continues running, next extraction scheduled normally

**AC6: Graceful Shutdown During Extraction**
**Given** daemon running an extraction
**When** SIGINT/SIGTERM received
**Then** current extraction completes
**And** log: "shutdown signal received, finishing current extraction"
**And** exits cleanly with code 0

**AC7: Graceful Shutdown While Waiting**
**Given** daemon waiting for next extraction
**When** SIGINT/SIGTERM received
**Then** wait is cancelled immediately, exits with code 0

**AC8: Once Mode Interrupt**
**Given** `--once` mode with extraction in progress
**When** SIGINT received
**Then** extraction cancelled via context, partial results NOT written, exit code 1

**AC9: Complete Main Entry Point**
**Given** application starts
**When** `main()` executes
**Then** sequence is:
  1. Parse CLI flags
  2. Handle `--version` (print and exit)
  3. Load configuration from file
  4. Apply environment overrides
  5. Validate configuration
  6. Initialize logger
  7. Create components (API client, Aggregator, StateManager, Writer)
  8. Set up signal handling
  9. Run in appropriate mode (once vs daemon)
  10. Exit with appropriate code

**AC10: Initialization Failure**
**Given** any initialization failure (bad config, etc.)
**When** error occurs
**Then** error logged, exit code 1

**AC11: Daemon Resilience for Start Immediately Failures**
**Given** `scheduler.start_immediately: true` and the initial extraction fails
**When** error occurs during the boot cycle
**Then** error is logged, daemon continues waiting for the next interval instead of exiting

**AC12: Signal Handling Setup**
**Given** daemon starts
**When** setting up signal handler
**Then** listen for `os.Interrupt` and `syscall.SIGTERM`
**And** use context cancellation for clean shutdown

## Tasks / Subtasks

- [ ] Task 1: Complete Daemon Scheduler Implementation (AC: 1-4, 11)
  - [ ] 1.1: Verify `runDaemonWithDeps` exists in `cmd/extractor/main.go` (from Story 5.2)
  - [ ] 1.2: Add INFO log "daemon started, interval: {interval}" at daemon start
  - [ ] 1.3: Implement `time.Ticker` with `cfg.Scheduler.Interval` (default 2h)
  - [ ] 1.4: If `cfg.Scheduler.StartImmediately == true`: run extraction immediately before entering ticker loop
  - [ ] 1.5: After each extraction cycle, log INFO "next extraction at {timestamp}" with next scheduled time
  - [ ] 1.6: Verify start_immediately failures log error and continue to next tick (already implemented in 5.2)

- [ ] Task 2: Implement Graceful Shutdown (AC: 6-8, 12)
  - [ ] 2.1: Use `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)` for context cancellation
  - [ ] 2.2: In daemon loop: check `ctx.Done()` to detect shutdown signal
  - [ ] 2.3: On shutdown during extraction: log "shutdown signal received, finishing current extraction"
  - [ ] 2.4: Allow current extraction to complete via context propagation
  - [ ] 2.5: On shutdown while waiting: ticker select receives context.Done(), exits immediately
  - [ ] 2.6: Ensure daemon exit code is 0 on graceful shutdown
  - [ ] 2.7: For `--once` mode: if context cancelled mid-extraction, skip writes and exit 1

- [ ] Task 3: Wire Complete Main Entry Point (AC: 9-10)
  - [ ] 3.1: Verify main() sequence follows tech spec order
  - [ ] 3.2: Verify error handling at each initialization step returns exit 1
  - [ ] 3.3: Ensure all components are created before signal handler setup
  - [ ] 3.4: Route to `runDaemonWithDeps` when no `--once` flag

- [ ] Task 4: Implement Daemon Error Recovery (AC: 5)
  - [ ] 4.1: Wrap extraction call in error handling within daemon loop
  - [ ] 4.2: On extraction failure: log ERROR with error message and duration
  - [ ] 4.3: Continue to next scheduled extraction (do not exit)
  - [ ] 4.4: Log "next extraction at {timestamp}" even after failure

- [ ] Task 5: Write Unit Tests (AC: all)
  - [ ] 5.1: Test daemon scheduler runs at interval
  - [ ] 5.2: Test start_immediately=true runs extraction immediately
  - [ ] 5.3: Test start_immediately=false waits for first interval
  - [ ] 5.4: Test daemon logs "daemon started" with interval
  - [ ] 5.5: Test "next extraction at" log after each cycle
  - [ ] 5.6: Test daemon continues after extraction failure
  - [ ] 5.7: Test graceful shutdown during extraction completes current work
  - [ ] 5.8: Test graceful shutdown while waiting exits immediately
  - [ ] 5.9: Test main() initialization failure exits 1
  - [ ] 5.10: Test `--once` interrupted exits 1 with partial results not written

- [ ] Task 6: Integration Testing (AC: all)
  - [ ] 6.1: Run daemon with short interval (e.g., 5s for testing)
  - [ ] 6.2: Verify scheduled extractions occur
  - [ ] 6.3: Send SIGINT during wait, verify immediate exit
  - [ ] 6.4: Send SIGINT during extraction, verify completion then exit
  - [ ] 6.5: Cause extraction failure, verify daemon continues

- [ ] Task 7: Verification (AC: all)
  - [ ] 7.1: Run `go build ./...` and verify success
  - [ ] 7.2: Run `go test ./cmd/extractor/...` and verify all pass
  - [ ] 7.3: Run `make lint` and verify no errors
  - [ ] 7.4: Manual smoke test: Run without `--once`, verify daemon starts and logs interval
  - [ ] 7.5: Manual smoke test: Wait for scheduled extraction, verify runs and logs next time
  - [ ] 7.6: Manual smoke test: Send SIGINT during wait, verify immediate clean exit
  - [ ] 7.7: Manual smoke test: Cause init failure (bad config path), verify exit 1

## Dev Notes

### Technical Guidance

- **Files to Modify:**
  - MODIFY: `cmd/extractor/main.go` - Complete daemon implementation
  - MODIFY: `cmd/extractor/main_test.go` - Add/extend daemon tests

- **ADR Compliance:**
  - ADR-001: Use `time.Ticker` from stdlib (not third-party scheduler)
  - ADR-004: Use `log/slog` for all logging

- **Signal Handling Pattern** per tech spec:
  ```go
  ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
  defer stop()

  // Use ctx in all operations for clean cancellation
  ```

- **Daemon Loop Pattern**:
  ```go
  ticker := time.NewTicker(cfg.Scheduler.Interval)
  defer ticker.Stop()

  if cfg.Scheduler.StartImmediately {
      if err := runOnce(ctx, ...); err != nil {
          logger.Error("extraction failed", "error", err)
          // Continue, don't exit
      }
  }

  for {
      select {
      case <-ctx.Done():
          logger.Info("shutdown signal received")
          return nil
      case <-ticker.C:
          if err := runOnce(ctx, ...); err != nil {
              logger.Error("extraction failed", "error", err)
          }
          logger.Info("next extraction at", "time", time.Now().Add(cfg.Scheduler.Interval))
      }
  }
  ```

- **FR Coverage:** FR43 (daemon mode), FR47 (graceful shutdown)

### Learnings from Previous Story

**From Story 5-2-implement-cli-and-single-run-mode (Status: done)**

- **Daemon Foundation Already Built**:
  - `runDaemonWithDeps()` exists at `cmd/extractor/main.go:186-241`
  - Signal-aware context with `signal.NotifyContext()` implemented
  - Ticker scheduling with `time.Ticker` implemented
  - Start_immediately resiliency implemented (logs error, continues to next tick)

- **Daemon Unit Tests Exist**:
  - `cmd/extractor/main_test.go:320-413` covers daemon tick execution and startup failure

- **Key Patterns Established**:
  - Dependency injection for testability (`runOnceWithDeps` pattern)
  - Error logging with `slog` structured attributes
  - Context propagation for cancellation

- **Remaining Work for Story 5.3**:
  - Add "daemon started, interval: X" log message (AC1)
  - Add "next extraction at {timestamp}" log after each cycle (AC4)
  - Verify/add "shutdown signal received" log message (AC6)
  - Add integration tests for signal handling (Task 6)
  - Complete smoke test verification (Task 7)

- **Carry-forward from Completion Notes (Story 5.2)**:
  - New/modified files to reference: `cmd/extractor/main.go`, `cmd/extractor/main_test.go`, `internal/api/client.go`, `internal/api/responses.go`, `internal/api/responses_test.go` [Source: docs/sprint-artifacts/5-2-implement-cli-and-single-run-mode.md#File-List]
  - Key learnings: dry-run must not persist state; start_immediately failures should log and continue; default no-flag path must enter daemon scheduler; protocol envelope regression tests guard decoding [Source: docs/sprint-artifacts/5-2-implement-cli-and-single-run-mode.md#Completion-Notes-List]

[Source: docs/sprint-artifacts/5-2-implement-cli-and-single-run-mode.md#Dev-Agent-Record]

### Project Structure Notes

- **Package Location**: `cmd/extractor/` for main entry point per Go standard layout
- **Import Dependencies**:
  - `os/signal` - Signal handling
  - `syscall` - SIGTERM constant
  - `time` - Ticker for scheduling
  - `context` - Cancellation propagation
- Reference: docs/architecture/project-structure.md

### Testing Standards

- Follow table-driven test pattern per [Source: docs/architecture/testing-strategy.md#Test-Organization]
- Use mock/fake dependencies for daemon tests (avoid real timers)
- Test signal handling via context cancellation
- Use short intervals (milliseconds) in tests to avoid slow test suites

### Post-Review Follow-ups from Tech Spec

The following items from the tech spec's Post-Review Follow-ups section apply to this story:

- [x] [Story 5.2][High] Implement default daemon execution so running without flags enters the scheduler instead of exiting immediately (`cmd/extractor/main.go:187-190`). **STATUS: Done in 5.2**
- [x] [Story 5.2][Med] Keep daemon start_immediately failures from exiting the process; log the error and wait for the next interval instead (`cmd/extractor/main.go:222-225`). **STATUS: Done in 5.2**

### Smoke Test Guide

1. Run without `--once` flag: daemon starts, logs interval
2. Wait for scheduled extraction: runs and logs next time
3. Send SIGINT during extraction: completes, then exits 0
4. Send SIGINT while waiting: exits immediately with 0
5. Cause init failure (bad config path): exits 1 with error

### References

- [Source: docs/prd.md#FR43] - Daemon mode with configurable interval
- [Source: docs/prd.md#FR47] - Graceful shutdown on SIGINT/SIGTERM
- [Source: docs/epics/epic-5-output-cli.md#story-53] - Story definition
- [Source: docs/sprint-artifacts/tech-spec-epic-5.md#Story-5.3] - Authoritative acceptance criteria
- [Source: docs/sprint-artifacts/tech-spec-epic-5.md#Daemon-Mode-Sequence] - Daemon workflow
- [Source: docs/sprint-artifacts/tech-spec-epic-5.md#Graceful-Shutdown] - Shutdown behavior
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-001] - Use stdlib
- [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-004] - Use slog for logging
- [Source: docs/sprint-artifacts/5-2-implement-cli-and-single-run-mode.md#Dev-Agent-Record] - Previous story learnings

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/5-3-implement-daemon-mode.context.xml

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-12-02 | SM Agent (Bob) | Initial story draft created from epic-5, tech-spec-epic-5.md, and previous story learnings |
