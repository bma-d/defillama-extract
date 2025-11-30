# Story 2.4: Implement Retry Logic with Exponential Backoff

Status: ready-for-dev

## Story

As a **developer**,
I want **automatic retries with exponential backoff for transient API failures**,
so that **temporary DefiLlama API issues don't cause extraction failures and the system recovers gracefully from rate limits and server errors**.

## Acceptance Criteria

1. **Given** API configuration with `max_retries: 3` and `retry_delay: 1s` **When** a request fails with a retryable error (timeout, 429, 5xx) **Then** the request is retried up to 3 times

2. **Given** a retryable failure **When** retries are attempted **Then** delays between retries follow exponential backoff: 1s, 2s, 4s (delay * 2^attempt)

3. **Given** exponential backoff delays **When** calculating sleep duration **Then** jitter of +/-25% is added to prevent thundering herd: `delay * (0.75 + rand*0.5)`

4. **Given** a request that fails with 429 (rate limit) or 5xx (500, 502, 503, 504) **When** retries are attempted **Then** each retry is logged at WARN level with: `url`, `attempt`, `max_attempts`, `backoff_ms`

5. **Given** max retries exhausted **When** the final attempt fails **Then** failure is logged at ERROR level with: `url`, `total_attempts`, `final_error`

6. **Given** a request that fails with 4xx (except 429) such as 400, 401, 403, 404 **When** the error is detected **Then** no retry is attempted (client error, not transient) and error is returned immediately

7. **Given** a request that fails on attempt 1 and 2 but succeeds on attempt 3 **When** the response is received **Then** the successful response is returned and info log indicates success after retries

8. **Given** a context with cancellation **When** the context is cancelled during retry sleep or request **Then** the retry loop exits immediately returning `context.Canceled`

9. **Given** a network timeout (request exceeds http.Client.Timeout) **When** the timeout occurs **Then** it is treated as retryable error

## Tasks / Subtasks

- [ ] Task 1: Implement isRetryable helper function (AC: 1, 6, 9)
  - [ ] 1.1: Add `isRetryable(statusCode int, err error) bool` function to `internal/api/client.go`
  - [ ] 1.2: Return `true` for: timeout errors, HTTP 429, 500, 502, 503, 504
  - [ ] 1.3: Return `false` for: HTTP 400, 401, 403, 404 and other 4xx
  - [ ] 1.4: Return `true` for network errors (connection refused, DNS failure)

- [ ] Task 2: Implement calculateBackoff helper (AC: 2, 3)
  - [ ] 2.1: Add `calculateBackoff(attempt int, baseDelay time.Duration) time.Duration` function
  - [ ] 2.2: Calculate exponential delay: `baseDelay * (1 << attempt)` (1s, 2s, 4s)
  - [ ] 2.3: Apply jitter: `delay * (0.75 + rand.Float64()*0.5)` for +/-25% variation
  - [ ] 2.4: Use `math/rand` for jitter calculation

- [ ] Task 3: Implement doWithRetry wrapper method (AC: 1, 2, 3, 4, 5, 7, 8)
  - [ ] 3.1: Add `doWithRetry(ctx context.Context, fn func() error) error` method to `Client`
  - [ ] 3.2: Loop up to `c.maxRetries + 1` attempts (initial + retries)
  - [ ] 3.3: On success, return nil immediately
  - [ ] 3.4: On non-retryable error, return error immediately without retry
  - [ ] 3.5: On retryable error, log at WARN with `url`, `attempt`, `max_attempts`, `backoff_ms`
  - [ ] 3.6: Sleep with backoff, checking context cancellation before and after sleep
  - [ ] 3.7: On max retries exhausted, log at ERROR with `url`, `total_attempts`, `final_error`
  - [ ] 3.8: Return the final error with wrapped context

- [ ] Task 4: Enhance doRequest to capture status code for retry decisions (AC: 1, 6)
  - [ ] 4.1: Create `APIError` struct with `StatusCode`, `Endpoint`, `Message`, `Err` fields
  - [ ] 4.2: Update `doRequest` to return `*APIError` for non-2xx responses
  - [ ] 4.3: Add `IsRetryable() bool` method to `APIError`

- [ ] Task 5: Integrate retry logic into FetchOracles and FetchProtocols (AC: 1, 7)
  - [ ] 5.1: Wrap `doRequest` call with `doWithRetry` in `FetchOracles`
  - [ ] 5.2: Wrap `doRequest` call with `doWithRetry` in `FetchProtocols`
  - [ ] 5.3: Ensure proper error propagation

- [ ] Task 6: Write unit tests for isRetryable (AC: 1, 6, 9)
  - [ ] 6.1: Test returns true for status codes: 429, 500, 502, 503, 504
  - [ ] 6.2: Test returns false for status codes: 400, 401, 403, 404
  - [ ] 6.3: Test returns true for timeout errors
  - [ ] 6.4: Test returns false for JSON decode errors

- [ ] Task 7: Write unit tests for calculateBackoff (AC: 2, 3)
  - [ ] 7.1: Test exponential progression: 1s base -> 1s, 2s, 4s for attempts 0, 1, 2
  - [ ] 7.2: Test jitter is applied (result differs from exact exponential)
  - [ ] 7.3: Test jitter bounds are within +/-25% of base exponential value

- [ ] Task 8: Write integration tests for doWithRetry with mock server (AC: 1, 4, 5, 6, 7, 8)
  - [ ] 8.1: Test retryable error (429) retries up to max then fails
  - [ ] 8.2: Test non-retryable error (404) returns immediately without retry
  - [ ] 8.3: Test success on attempt 3 after 2 failures returns success
  - [ ] 8.4: Test context cancellation exits retry loop
  - [ ] 8.5: Test retry logs are emitted at WARN level (capture log output)
  - [ ] 8.6: Test final failure logs at ERROR level

- [ ] Task 9: Verification (AC: all)
  - [ ] 9.1: Run `go build ./...` and verify success
  - [ ] 9.2: Run `go test ./internal/api/...` and verify all pass including new retry tests
  - [ ] 9.3: Run `make lint` and verify no errors

## Dev Notes

### Technical Guidance

- **Package Location:** All retry logic in `internal/api/client.go`
- **Dependencies:** stdlib only (`net/http`, `context`, `time`, `math/rand`, `log/slog`, `errors`)
- **ADR Alignment:** ADR-001 mandates stdlib; no external retry libraries

### Retry Implementation Pattern

Per tech-spec-epic-2.md, the retry wrapper should:

```go
// doWithRetry wraps a function call with exponential backoff retry logic.
func (c *Client) doWithRetry(ctx context.Context, operation string, fn func() error) error {
    var lastErr error

    for attempt := 0; attempt <= c.maxRetries; attempt++ {
        lastErr = fn()
        if lastErr == nil {
            if attempt > 0 {
                c.logger.Info("request succeeded after retries",
                    "operation", operation,
                    "attempts", attempt+1,
                )
            }
            return nil
        }

        // Check if retryable
        var apiErr *APIError
        if errors.As(lastErr, &apiErr) && !apiErr.IsRetryable() {
            return lastErr // Non-retryable, exit immediately
        }

        if attempt == c.maxRetries {
            break // No more retries
        }

        // Calculate backoff with jitter
        backoff := c.calculateBackoff(attempt, c.retryDelay)

        c.logger.Warn("retrying API request",
            "operation", operation,
            "attempt", attempt+1,
            "max_attempts", c.maxRetries+1,
            "backoff_ms", backoff.Milliseconds(),
            "error", lastErr,
        )

        // Sleep with context cancellation check
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(backoff):
        }
    }

    c.logger.Error("max retries exceeded",
        "operation", operation,
        "total_attempts", c.maxRetries+1,
        "final_error", lastErr,
    )

    return fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

### APIError Type

```go
// APIError represents an HTTP API error with status code for retry decisions.
type APIError struct {
    Endpoint   string
    StatusCode int
    Message    string
    Err        error
}

func (e *APIError) Error() string {
    return fmt.Sprintf("API error %d on %s: %s", e.StatusCode, e.Endpoint, e.Message)
}

func (e *APIError) Unwrap() error {
    return e.Err
}

func (e *APIError) IsRetryable() bool {
    switch e.StatusCode {
    case 429, 500, 502, 503, 504:
        return true
    default:
        return false
    }
}
```

### Backoff Calculation

```go
func (c *Client) calculateBackoff(attempt int, baseDelay time.Duration) time.Duration {
    // Exponential: baseDelay * 2^attempt
    exponential := baseDelay * (1 << attempt)

    // Jitter: +/-25% -> multiply by (0.75 + rand*0.5)
    jitterMultiplier := 0.75 + rand.Float64()*0.5

    return time.Duration(float64(exponential) * jitterMultiplier)
}
```

### Retryable Error Detection

```go
func isRetryable(err error) bool {
    // Check for APIError with retryable status
    var apiErr *APIError
    if errors.As(err, &apiErr) {
        return apiErr.IsRetryable()
    }

    // Check for timeout errors
    if errors.Is(err, context.DeadlineExceeded) {
        return true
    }

    // Check for network errors (URL errors are usually retryable)
    var urlErr *url.Error
    if errors.As(err, &urlErr) {
        return urlErr.Timeout() || urlErr.Temporary()
    }

    return false
}
```

### Project Structure Notes

- All changes in existing file: `internal/api/client.go`
- New test file: `internal/api/retry_test.go`
- Existing Client struct already has `maxRetries` and `retryDelay` fields from config
- APIError type can be added to `internal/api/responses.go` alongside other API types

### Learnings from Previous Story

**From Story 2-3-implement-protocol-endpoint-fetcher (Status: done)**

- **doRequest Helper Available:** Use existing `c.doRequest(ctx, url, &target)` which handles User-Agent, timeout, JSON decode, error wrapping
- **Client Fields Ready:** `c.maxRetries` and `c.retryDelay` already stored from config in `NewClient()`
- **Test Patterns Established:**
  - Use `httptest.NewServer` for mock server tests
  - Cover: success, header verification, status errors, malformed JSON, context cancellation
  - Test fixtures in `testdata/` directory
- **Non-2xx Handling:** `doRequest` returns `fmt.Errorf("unexpected status: %d", resp.StatusCode)` - needs enhancement to expose status code for retry decisions
- **Review Outcome:** Approved with no action items

[Source: docs/sprint-artifacts/2-3-implement-protocol-endpoint-fetcher.md#Dev-Agent-Record]

### References

- [Source: docs/sprint-artifacts/tech-spec-epic-2.md#AC-2.4] - Retry logic acceptance criteria (authoritative)
- [Source: docs/sprint-artifacts/tech-spec-epic-2.md#Workflows-and-Sequencing] - Single request flow with retry
- [Source: docs/sprint-artifacts/tech-spec-epic-2.md#Reliability/Availability] - Retryable vs non-retryable conditions
- [Source: docs/epics/epic-2-api-integration.md#story-24] - Original story definition with ACs
- [Source: docs/prd.md#FR5] - System retries failed API requests with exponential backoff and jitter
- [Source: docs/prd.md#FR7] - System handles API errors gracefully (429, 5xx) with retry
- [Source: docs/prd.md#FR8] - System detects non-retryable errors (4xx client errors)
- [Source: docs/architecture/implementation-patterns.md#Context-Propagation] - Context for cancellation
- [Source: docs/architecture/testing-strategy.md#Test-Organization] - Mock server tests for HTTP retry logic

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/2-4-implement-retry-logic-with-exponential-backoff.context.xml

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-2-api-integration.md and tech-spec-epic-2.md |
