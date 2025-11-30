# Story 2.5: Implement Parallel Fetching with errgroup

Status: ready-for-dev

## Story

As a **developer**,
I want **oracle and protocol data fetched concurrently using errgroup**,
so that **total fetch time is minimized to approximately the duration of the slowest request rather than the sum of both**.

## Acceptance Criteria

1. **Given** a need to fetch both oracle and protocol data **When** `FetchAll(ctx context.Context)` is called **Then** both API requests are initiated concurrently using `golang.org/x/sync/errgroup`

2. **Given** both requests execute concurrently **When** measuring total fetch time **Then** duration approximates `max(oracle_time, protocol_time)` rather than `oracle_time + protocol_time`

3. **Given** both requests succeed **When** `FetchAll` returns **Then** a `*FetchResult` struct is returned containing both `OracleResponse *OracleAPIResponse` and `Protocols []Protocol` with nil error

4. **Given** the oracle request fails but protocol succeeds **When** `FetchAll` returns **Then** an error is returned describing the oracle failure **And** the protocol request's context is cancelled (if still in progress)

5. **Given** the protocol request fails but oracle succeeds **When** `FetchAll` returns **Then** an error is returned describing the protocol failure **And** the oracle request's context is cancelled (if still in progress)

6. **Given** both requests fail **When** `FetchAll` returns **Then** the first error encountered is returned (errgroup behavior)

7. **Given** the parent context is cancelled during fetch **When** cancellation propagates **Then** both in-flight requests are cancelled **And** the function returns `context.Canceled` or `context.DeadlineExceeded` error

8. **Given** parallel fetch completes successfully **When** logging occurs **Then** an info log is emitted with total duration: `"parallel fetch completed"` with `oracle_duration_ms`, `protocol_duration_ms`, `total_duration_ms`

## Tasks / Subtasks

- [ ] Task 1: Add golang.org/x/sync dependency (AC: 1)
  - [ ] 1.1: Run `go get golang.org/x/sync` to add errgroup package
  - [ ] 1.2: Verify `go.mod` includes `golang.org/x/sync` dependency
  - [ ] 1.3: Run `go mod tidy` to clean up dependencies

- [ ] Task 2: Define FetchResult struct (AC: 3)
  - [ ] 2.1: Add `FetchResult` struct to `internal/api/responses.go`:
    ```go
    type FetchResult struct {
        OracleResponse *OracleAPIResponse
        Protocols      []Protocol
    }
    ```
  - [ ] 2.2: Ensure struct fields are exported for external package access

- [ ] Task 3: Implement FetchAll method (AC: 1, 2, 3, 4, 5, 6, 7)
  - [ ] 3.1: Add `FetchAll(ctx context.Context) (*FetchResult, error)` method to `Client` in `internal/api/client.go`
  - [ ] 3.2: Create errgroup with context: `g, ctx := errgroup.WithContext(ctx)`
  - [ ] 3.3: Launch oracle fetch in goroutine via `g.Go()`
  - [ ] 3.4: Launch protocol fetch in goroutine via `g.Go()`
  - [ ] 3.5: Wait for both with `g.Wait()`
  - [ ] 3.6: On success, return `&FetchResult{OracleResponse: oracleResp, Protocols: protocols}`
  - [ ] 3.7: On error, return `nil, err` (errgroup returns first error)

- [ ] Task 4: Add timing and logging (AC: 2, 8)
  - [ ] 4.1: Track start time before launching goroutines
  - [ ] 4.2: Track individual fetch durations within each goroutine
  - [ ] 4.3: On success, log at INFO: `"parallel fetch completed"` with `oracle_duration_ms`, `protocol_duration_ms`, `total_duration_ms`
  - [ ] 4.4: On failure, log at ERROR: `"parallel fetch failed"` with `error`, `total_duration_ms`

- [ ] Task 5: Write unit tests for FetchAll success path (AC: 2, 3, 8)
  - [ ] 5.1: Create test in `internal/api/client_test.go` or new `internal/api/fetchall_test.go`
  - [ ] 5.2: Mock server returns both endpoints successfully
  - [ ] 5.3: Verify both `OracleResponse` and `Protocols` are populated
  - [ ] 5.4: Verify no error returned
  - [ ] 5.5: (Optional) Verify parallel timing is faster than sequential (add delays to mock)

- [ ] Task 6: Write unit tests for partial failure (AC: 4, 5, 6)
  - [ ] 6.1: Test oracle fails (500), protocol succeeds → error returned
  - [ ] 6.2: Test protocol fails (500), oracle succeeds → error returned
  - [ ] 6.3: Test both fail → first error returned
  - [ ] 6.4: Verify error message identifies which endpoint failed

- [ ] Task 7: Write unit test for context cancellation (AC: 7)
  - [ ] 7.1: Create cancellable context with short timeout
  - [ ] 7.2: Mock server adds delay longer than timeout
  - [ ] 7.3: Verify `FetchAll` returns context error
  - [ ] 7.4: Verify both requests are cancelled (no hung goroutines)

- [ ] Task 8: Write parallel performance test (AC: 2)
  - [ ] 8.1: Mock server with 100ms delay on each endpoint
  - [ ] 8.2: Measure FetchAll duration
  - [ ] 8.3: Assert total time < 150ms (parallel) not > 180ms (sequential would be 200ms+)

- [ ] Task 9: Verification (AC: all)
  - [ ] 9.1: Run `go build ./...` and verify success
  - [ ] 9.2: Run `go test ./internal/api/...` and verify all pass including new FetchAll tests
  - [ ] 9.3: Run `make lint` and verify no errors

## Dev Notes

### Technical Guidance

- **Package Location:** `FetchAll` method in `internal/api/client.go`, `FetchResult` struct in `internal/api/responses.go`
- **New Dependency:** `golang.org/x/sync/errgroup` - official Go extended library, stable and well-maintained per ADR-005
- **ADR Alignment:** ADR-001 stdlib preference; errgroup is the exception explicitly allowed in tech-spec
- **Testing Strategy Alignment:** Follow mock-server patterns and coverage expectations from `docs/architecture/testing-strategy.md` (unit + mock server tests for FetchAll paths)

### Implementation Pattern

Per docs/architecture/implementation-patterns.md#Parallel-Fetching:

```go
import "golang.org/x/sync/errgroup"

func (c *Client) FetchAll(ctx context.Context) (*FetchResult, error) {
    start := time.Now()
    var oracleResp *OracleAPIResponse
    var protocols []Protocol
    var oracleDuration, protocolDuration time.Duration

    g, ctx := errgroup.WithContext(ctx)

    g.Go(func() error {
        fetchStart := time.Now()
        var err error
        oracleResp, err = c.FetchOracles(ctx)
        oracleDuration = time.Since(fetchStart)
        return err
    })

    g.Go(func() error {
        fetchStart := time.Now()
        var err error
        protocols, err = c.FetchProtocols(ctx)
        protocolDuration = time.Since(fetchStart)
        return err
    })

    if err := g.Wait(); err != nil {
        c.logger.Error("parallel fetch failed",
            "error", err,
            "total_duration_ms", time.Since(start).Milliseconds(),
        )
        return nil, err
    }

    c.logger.Info("parallel fetch completed",
        "oracle_duration_ms", oracleDuration.Milliseconds(),
        "protocol_duration_ms", protocolDuration.Milliseconds(),
        "total_duration_ms", time.Since(start).Milliseconds(),
    )

    return &FetchResult{
        OracleResponse: oracleResp,
        Protocols:      protocols,
    }, nil
}
```

### errgroup Behavior Notes

- `errgroup.WithContext(ctx)` returns a derived context that is cancelled when any goroutine returns an error
- `g.Wait()` blocks until all goroutines complete or one returns an error
- If one goroutine fails, the context is cancelled, signaling other goroutines to abort
- First error is returned; subsequent errors from other goroutines are discarded

### Project Structure Notes

- `FetchResult` struct added to `internal/api/responses.go` alongside `OracleAPIResponse`, `Protocol`, `APIError`
- `FetchAll` method added to `Client` in `internal/api/client.go`
- Tests can be in existing `client_test.go` or new dedicated `fetchall_test.go`

### Learnings from Previous Story

**From Story 2-4-implement-retry-logic-with-exponential-backoff (Status: done)**

- **Retry Logic Available:** `FetchOracles` and `FetchProtocols` already wrapped with `doWithRetry` - retry/backoff handled automatically
- **Logging Patterns:** Use same slog patterns: `c.logger.Info/Warn/Error` with structured attributes
- **Test Patterns Established:**
  - Use `httptest.NewServer` with handler functions
  - Cover success, failure, timeout, context cancellation paths
  - Track attempt counts and timing in tests
- **APIError Handling:** Errors from fetchers are properly wrapped `*APIError` with status codes
- **Context Propagation:** Both fetchers respect context cancellation
- **Review Outcome:** Approved with no action items

**New/Modified Files from Story 2-4 (for continuity):**
- `internal/api/client.go` — added retry wrapper, logging, context checks
- `internal/api/responses.go` — introduced `APIError` for status-aware retries
- `internal/api/retry_test.go` — retry logic unit/integration tests

**Completion Notes carried forward:**
- Retry stack implemented and validated (go build/test/lint all passing on 2025-11-30)
- No outstanding review action items or follow-ups

[Source: docs/sprint-artifacts/2-4-implement-retry-logic-with-exponential-backoff.md#Dev-Agent-Record]

### References

- [Source: docs/sprint-artifacts/tech-spec-epic-2.md#AC-2.5] - Parallel fetching acceptance criteria (authoritative)
- [Source: docs/sprint-artifacts/tech-spec-epic-2.md#Workflows-and-Sequencing] - Parallel Fetch Flow diagram
- [Source: docs/sprint-artifacts/tech-spec-epic-2.md#Data-Models-and-Contracts] - FetchResult struct definition
- [Source: docs/epics/epic-2-api-integration.md#story-25] - Original story definition with ACs
- [Source: docs/prd.md#FR3] - System fetches both endpoints in parallel to minimize total fetch time
- [Source: docs/architecture/implementation-patterns.md#Parallel-Fetching] - errgroup pattern reference
- [Source: docs/sprint-artifacts/tech-spec-epic-2.md#Dependencies-and-Integrations] - golang.org/x/sync dependency
- [Source: docs/architecture/testing-strategy.md#Test-Organization] - Testing patterns to follow for FetchAll mock server coverage

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/2-5-implement-parallel-fetching-with-errgroup.context.xml

### Agent Model Used

- GPT-5 (Codex) — SM Agent Bob

### Debug Log References

- 2025-11-30: Validation pass (minor metadata); populated Dev Agent Record fields.

### Completion Notes List

- 2025-11-30: Story drafted and validated; ready for Dev Agent implementation of parallel fetch.

### File List

- docs/sprint-artifacts/2-5-implement-parallel-fetching-with-errgroup.md
- docs/sprint-artifacts/2-5-implement-parallel-fetching-with-errgroup.context.xml
## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-2-api-integration.md and tech-spec-epic-2.md |
