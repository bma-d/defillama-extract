# Story 2.6: Implement API Request Logging

Status: done

## Story

As an **operator**,
I want **API requests logged with timing and outcome**,
so that **I can monitor API health and debug issues**.

## Acceptance Criteria

1. **Given** an API request is initiated **When** the request starts **Then** a debug log is emitted: `"starting API request"` with `url`, `method` attributes

2. **Given** an API request completes successfully **When** response is received **Then** an info log is emitted: `"API request completed"` with `url`, `status`, `duration_ms` attributes

3. **Given** an API request fails **When** error occurs **Then** a warn log is emitted: `"API request failed"` with `url`, `error`, `duration_ms`, `attempt` attributes

4. **Given** a retry is attempted **When** retry starts **Then** a warn log is emitted: `"retrying API request"` with `url`, `attempt`, `max_attempts`, `backoff_ms` attributes (NOTE: Already implemented in Story 2.4)

5. **Given** all retries are exhausted **When** final failure occurs **Then** an error log is emitted with `url`, `total_attempts`, `final_error` attributes (NOTE: Already implemented in Story 2.4)

6. **Given** debug logging is enabled **When** viewing logs **Then** request start entries appear before completion entries for the same request

7. **Given** multiple concurrent requests (FetchAll) **When** both complete **Then** each request has its own start/completion log entries distinguishable by URL

## Tasks / Subtasks

- [x] Task 1: Add request start logging to doRequest (AC: 1, 6)
  - [x] 1.1: Add `start := time.Now()` at beginning of `doRequest`
  - [x] 1.2: Add debug log immediately after request creation:
    ```go
    c.logger.Debug("starting API request",
        "url", url,
        "method", http.MethodGet,
    )
    ```
  - [x] 1.3: Ensure log appears before HTTP client.Do call

- [x] Task 2: Add request success logging to doRequest (AC: 2)
  - [x] 2.1: After successful status check (2xx), add info log:
    ```go
    c.logger.Info("API request completed",
        "url", url,
        "status", resp.StatusCode,
        "duration_ms", time.Since(start).Milliseconds(),
    )
    ```
  - [x] 2.2: Place log after status validation but before JSON decode

- [x] Task 3: Add request failure logging to doRequest (AC: 3)
  - [x] 3.1: When returning APIError for non-2xx status, log at warn:
    ```go
    c.logger.Warn("API request failed",
        "url", url,
        "status", resp.StatusCode,
        "attempt", attempt, // sourced from retry wrapper
        "duration_ms", time.Since(start).Milliseconds(),
        "error", err,
    )
    ```
  - [x] 3.2: When returning APIError for network error, log at warn with status=0
  - [x] 3.3: Ensure `attempt` attribute is included (pull from retry wrapper context; default to 1 if missing)

- [x] Task 4: Write unit tests for request start logging (AC: 1, 6)
  - [x] 4.1: Create test with custom slog handler to capture log entries
  - [x] 4.2: Call FetchOracles with mock server
  - [x] 4.3: Verify debug log with "starting API request", correct url and method
  - [x] 4.4: Verify start log appears before completion log (check timestamps or order)

- [x] Task 5: Write unit tests for request success logging (AC: 2)
  - [x] 5.1: Mock server returns 200 OK with valid JSON
  - [x] 5.2: Verify info log with "API request completed"
  - [x] 5.3: Verify attributes: url, status=200, duration_ms > 0

- [x] Task 6: Write unit tests for request failure logging (AC: 3)
  - [x] 6.1: Mock server returns 500 Internal Server Error
  - [x] 6.2: Verify warn log with "API request failed"
  - [x] 6.3: Verify attributes: url, status=500, duration_ms, error

- [x] Task 7: Write unit test for concurrent request logging (AC: 7)
  - [x] 7.1: Use FetchAll with mock server
  - [x] 7.2: Capture all log entries
  - [x] 7.3: Verify distinct logs for oracle and protocol URLs
  - [x] 7.4: Verify each has start and completion entries

- [x] Task 8: Verification (AC: all)
  - [x] 8.1: Run `go build ./...` and verify success
  - [x] 8.2: Run `go test ./internal/api/...` and verify all pass
  - [x] 8.3: Run `make lint` and verify no errors
  - [x] 8.4: Manual verification: run with DEBUG log level, observe request lifecycle logs

## Dev Notes

### Technical Guidance

- **Package Location:** Modify `internal/api/client.go` doRequest method
- **Logging Patterns:** Follow existing slog patterns established in Story 2.4
- **ADR Alignment:** ADR-004 mandates structured logging with slog; see `docs/architecture/architecture-decision-records-adrs.md#adr-004-structured-logging-with-slog`
- **Attempt Attribute Source:** `doWithRetry` tracks attempts; propagate an `attempt` value (default 1) into `doRequest` logging so AC3 is satisfied
- **Testing Strategy Alignment:** Follow expectations in `docs/architecture/testing-strategy.md` for log coverage, ordering, and negative cases

### Implementation Pattern

The existing `doRequest` method needs to be enhanced with timing and logging. Current implementation:

```go
func (c *Client) doRequest(ctx context.Context, url string, target any) error {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    // ... rest of implementation
}
```

Enhanced implementation:

```go
func (c *Client) doRequest(ctx context.Context, url string, target any) error {
    start := time.Now()
    attempt := attemptFromContext(ctx) // defaults to 1 when not set by retry wrapper

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return fmt.Errorf("create request: %w", err)
    }

    req.Header.Set("User-Agent", c.userAgent)

    c.logger.Debug("starting API request",
        "url", url,
        "method", http.MethodGet,
    )

    resp, err := c.httpClient.Do(req)
    if err != nil {
        duration := time.Since(start)
        c.logger.Warn("API request failed",
            "url", url,
            "attempt", attempt,
            "error", err,
            "duration_ms", duration.Milliseconds(),
        )
        return &APIError{...}
    }
    defer resp.Body.Close()

    duration := time.Since(start)

    if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
        c.logger.Warn("API request failed",
            "url", url,
            "attempt", attempt,
            "status", resp.StatusCode,
            "duration_ms", duration.Milliseconds(),
        )
        return &APIError{...}
    }

    c.logger.Info("API request completed",
        "url", url,
        "status", resp.StatusCode,
        "duration_ms", duration.Milliseconds(),
    )

    if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
        return fmt.Errorf("decode response: %w", err)
    }

    return nil
}
```

### Existing Logging Already Implemented

Story 2.4 already implemented these log messages in `doWithRetry`:
- `"retrying API request"` (WARN) with url, attempt, max_attempts, backoff_ms, error
- `"max retries exceeded"` (ERROR) with url, total_attempts, final_error
- `"request succeeded after retries"` (INFO) with url, attempts

This story adds the per-request lifecycle logging that was missing.

### Test Pattern for Log Capture

Use a custom slog handler to capture logs:

```go
type logEntry struct {
    Level   slog.Level
    Message string
    Attrs   map[string]any
}

type testHandler struct {
    entries []logEntry
    mu      sync.Mutex
}

func (h *testHandler) Handle(ctx context.Context, r slog.Record) error {
    h.mu.Lock()
    defer h.mu.Unlock()
    entry := logEntry{Level: r.Level, Message: r.Message, Attrs: make(map[string]any)}
    r.Attrs(func(a slog.Attr) bool {
        entry.Attrs[a.Key] = a.Value.Any()
        return true
    })
    h.entries = append(h.entries, entry)
    return nil
}

// ... implement Enabled(), WithAttrs(), WithGroup() ...
```

### Project Structure Notes

- All changes in `internal/api/client.go`
- Tests in `internal/api/client_test.go` or new `internal/api/logging_test.go`
- No new files required; extends existing client implementation
- Reference project layout guidance in `docs/architecture/project-structure.md`

### Learnings from Previous Story

**From Story 2-5-implement-parallel-fetching-with-errgroup (Status: done)**

- **FetchAll Logging:** Already logs at FetchAll level (`"parallel fetch completed"` / `"parallel fetch failed"`) with timing
- **Per-Request Logging Missing:** Individual doRequest calls don't log start/completion — this story fills that gap
- **Test Patterns:** Use `httptest.NewServer` with handler functions; test helpers in `internal/api/fetchall_test.go`
- **Duration Tracking:** Pattern established — `start := time.Now()` then `time.Since(start).Milliseconds()`
- **Review Outcome:** Story 2.5 approved with no action items
- **Files Modified:** `internal/api/client.go`, `internal/api/responses.go`, `internal/api/fetchall_test.go`
- **errgroup Available:** `golang.org/x/sync` already in go.mod

[Source: docs/sprint-artifacts/2-5-implement-parallel-fetching-with-errgroup.md#Dev-Agent-Record]

### References

- [Source: docs/sprint-artifacts/tech-spec-epic-2.md#AC-2.6] - Request logging acceptance criteria (authoritative)
- [Source: docs/sprint-artifacts/tech-spec-epic-2.md#Observability] - Log event definitions and levels
- [Source: docs/epics/epic-2-api-integration.md#story-26] - Original story definition with ACs
- [Source: docs/prd.md#FR55] - System logs API request attempts, retries, and failures
- [Source: docs/architecture/architecture-decision-records-adrs.md#adr-004-structured-logging-with-slog] - slog logging standard
- [Source: docs/architecture/testing-strategy.md] - Testing expectations for logging and ordering
- [Source: docs/architecture/project-structure.md] - Project structure guidance

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/2-6-implement-api-request-logging.context.xml

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

- AC1/AC6: Added start timer + debug log before httpClient.Do; attempt propagation via context to keep ordering stable.
- AC2: Info log emitted post-status check with duration_ms; placed before decode to preserve order.
- AC3: Warn logs for network + non-2xx include attempt, duration_ms, status (0 for network) using attemptFromContext.
- AC7: FetchAll logs verified per-URL ordering via test capturing slog records.

### Completion Notes List

- Added attempt propagation into doRequest via context to surface retry attempt in failure logs.
- Emitted structured start/success/failure logs with duration_ms on every request path; kept existing retry logging untouched.
- Authored slog handler-based tests covering start ordering, success, failure (status + network), and concurrent FetchAll logging.
- Validation: `go build ./...`, `go test ./...`, `make lint` all pass.

### File List

- internal/api/client.go
- internal/api/client_test.go
- internal/api/retry_test.go
- docs/sprint-artifacts/sprint-status.yaml
- docs/sprint-artifacts/2-6-implement-api-request-logging.md

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-2-api-integration.md and tech-spec-epic-2.md |
| 2025-11-30 | Amelia (Dev Agent) | Implemented request lifecycle logging with attempt propagation, added log coverage tests, updated sprint status to review |
| 2025-11-30 | Amelia (Dev Agent) | Senior Developer Review notes appended; status moved to done |

## Senior Developer Review (AI)

- Reviewer: BMad
- Date: 2025-11-30
- Outcome: Approve
- Summary: Request lifecycle logging meets ACs with debug/info/warn coverage, attempt propagation, and concurrency ordering. Tests cover success, failure, network error, and FetchAll concurrency; build/lint/test all pass.

### Key Findings
- No High/Medium/Low issues identified.

### Acceptance Criteria Coverage

| AC # | Description | Status | Evidence |
| --- | --- | --- | --- |
| AC1 | Debug start log with url, method | Implemented | internal/api/client.go:90-94 |
| AC2 | Info completion log with url, status, duration_ms | Implemented | internal/api/client.go:132-135 |
| AC3 | Warn failure log with url, error, duration_ms, attempt | Implemented | internal/api/client.go:95-123 |
| AC4 | Warn retry log with attempt/backoff | Implemented | internal/api/client.go:233-239 |
| AC5 | Error log when retries exhausted | Implemented | internal/api/client.go:223-229 |
| AC6 | Start precedes completion for a request | Implemented | internal/api/client_test.go:209-263 |
| AC7 | Concurrent requests have distinct start/completion logs | Implemented | internal/api/client_test.go:352-405 |

Summary: 7 of 7 acceptance criteria implemented.

### Task Completion Validation

| Task | Marked As | Verified As | Evidence |
| --- | --- | --- | --- |
| Task 1: Add request start logging | Completed | Verified | internal/api/client.go:79-104 |
| Task 2: Add request success logging | Completed | Verified | internal/api/client.go:132-135 |
| Task 3: Add request failure logging | Completed | Verified | internal/api/client.go:95-123 |
| Task 4: Unit tests for start logging/order | Completed | Verified | internal/api/client_test.go:209-263 |
| Task 5: Unit tests for success logging | Completed | Verified | internal/api/client_test.go:209-263 |
| Task 6: Unit tests for failure logging (status + network) | Completed | Verified | internal/api/client_test.go:265-349 |
| Task 7: Unit test for concurrent request logging | Completed | Verified | internal/api/client_test.go:352-405 |
| Task 8: Verification (build/tests/lint) | Completed | Verified | go build ./..., go test ./..., make lint (2025-11-30) |

### Test Coverage and Gaps
- go test ./... (pass)
- make lint (pass)
- No gaps noted for story scope; concurrency and error paths covered by unit tests.

### Architectural Alignment
- Logging follows ADR-004 structured slog usage and tech-spec AC-2.6 attributes. Retry logging unchanged and remains compliant.

### Security Notes
- No new external inputs; logging does not expose sensitive data.

### Best-Practices and References
- Go 1.24, slog structured logging; patterns align with docs/architecture/testing-strategy.md and tech-spec-epic-2.md (Observability).

### Action Items

**Code Changes Required:** (none)

**Advisory Notes:**
- Note: Re-run go test ./internal/api/... after future changes to ensure logging order remains intact.
