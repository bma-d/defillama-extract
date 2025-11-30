# Story 2.1: Implement Base HTTP Client with Timeout and User-Agent

Status: done

## Story

As a **developer**,
I want **a configured HTTP client with proper timeout and identification headers**,
so that **API requests are well-behaved and identifiable to DefiLlama's servers**.

## Acceptance Criteria

1. **Given** API configuration with `timeout: 30s` **When** the HTTP client is initialized via `NewClient(cfg, logger)` **Then** the underlying `*http.Client` uses a 30-second timeout

2. **Given** any HTTP request made by the client **When** the request is sent **Then** the `User-Agent` header is set to `defillama-extract/1.0`

3. **Given** an API request in progress **When** the timeout duration elapses without response **Then** the request is cancelled **And** returns an error indicating timeout occurred

4. **Given** valid `*config.APIConfig` and `*slog.Logger` **When** `NewClient(cfg, logger)` is called **Then** a properly configured `*Client` is returned with the timeout and User-Agent applied

5. **Given** a context with cancellation **When** the context is cancelled during a request **Then** the request is aborted and returns `context.Canceled` error

## Tasks / Subtasks

- [x] Task 1: Create Client struct and constructor (AC: 1, 2, 4)
  - [x] 1.1: Create `internal/api/client.go` file
  - [x] 1.2: Define `Client` struct with fields: `httpClient *http.Client`, `baseURL string`, `userAgent string`, `maxRetries int`, `retryDelay time.Duration`, `logger *slog.Logger`
  - [x] 1.3: Implement `NewClient(cfg *config.APIConfig, logger *slog.Logger) *Client` constructor
  - [x] 1.4: Set `http.Client.Timeout` from `cfg.Timeout` (default 30s)
  - [x] 1.5: Store User-Agent string as `defillama-extract/1.0`

- [x] Task 2: Implement request helper with User-Agent injection (AC: 2, 3, 5)
  - [x] 2.1: Create `doRequest(ctx context.Context, url string, target any) error` method
  - [x] 2.2: Use `http.NewRequestWithContext(ctx, "GET", url, nil)` for context support
  - [x] 2.3: Set `req.Header.Set("User-Agent", c.userAgent)` on all requests
  - [x] 2.4: Execute request with `c.httpClient.Do(req)` and check for errors
  - [x] 2.5: Decode response body with `json.NewDecoder(resp.Body).Decode(target)`
  - [x] 2.6: Close response body in defer statement

- [x] Task 3: Write unit tests for client initialization (AC: 1, 4)
  - [x] 3.1: Create `internal/api/client_test.go` file
  - [x] 3.2: Test `NewClient` sets correct timeout from config
  - [x] 3.3: Test `NewClient` stores User-Agent string correctly
  - [x] 3.4: Test `NewClient` handles nil logger (should use slog.Default)

- [x] Task 4: Write integration tests with mock server (AC: 2, 3, 5)
  - [x] 4.1: Test User-Agent header is present using `httptest.NewServer`
  - [x] 4.2: Test timeout behavior with delayed mock server response
  - [x] 4.3: Test context cancellation aborts request
  - [x] 4.4: Test successful JSON decode into target struct

- [x] Task 5: Verification (AC: all)
  - [x] 5.1: Run `go build ./...` and verify success
  - [x] 5.2: Run `go test ./internal/api/...` and verify all pass
  - [x] 5.3: Run `make lint` and verify no errors

## Dev Notes

### Technical Guidance

- **Package Location:** `internal/api/client.go`
- **Dependencies:** stdlib only (`net/http`, `context`, `encoding/json`, `time`, `log/slog`)
- **ADR Alignment:** ADR-001 mandates `net/http` standard library usage, no external HTTP frameworks

### Client Struct Design

```go
// internal/api/client.go
package api

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "net/http"
    "time"

    "github.com/switchboard-xyz/defillama-extract/internal/config"
)

const userAgentValue = "defillama-extract/1.0"

// Client wraps http.Client with configured timeout, User-Agent, and logging.
type Client struct {
    httpClient *http.Client
    baseURL    string
    userAgent  string
    maxRetries int
    retryDelay time.Duration
    logger     *slog.Logger
}

// NewClient creates a new API client with configuration from APIConfig.
func NewClient(cfg *config.APIConfig, logger *slog.Logger) *Client {
    if logger == nil {
        logger = slog.Default()
    }
    return &Client{
        httpClient: &http.Client{
            Timeout: cfg.Timeout,
        },
        userAgent:  userAgentValue,
        maxRetries: cfg.MaxRetries,
        retryDelay: cfg.RetryDelay,
        logger:     logger,
    }
}
```

### Request Helper Pattern

```go
// doRequest performs a GET request with User-Agent and JSON decoding.
func (c *Client) doRequest(ctx context.Context, url string, target any) error {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return fmt.Errorf("create request: %w", err)
    }
    req.Header.Set("User-Agent", c.userAgent)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("execute request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }

    if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
        return fmt.Errorf("decode response: %w", err)
    }

    return nil
}
```

### Testing Strategy Alignment

Per testing-strategy.md and patterns from Story 1.4:
- Use `httptest.NewServer` for integration tests
- Capture and verify User-Agent header in mock handler
- Use table-driven tests for configuration variations
- Test timeout with `http.HandlerFunc` that sleeps beyond timeout duration

### Test Pattern Example

```go
func TestNewClient_SetsTimeout(t *testing.T) {
    cfg := &config.APIConfig{
        Timeout:    15 * time.Second,
        MaxRetries: 3,
        RetryDelay: 1 * time.Second,
    }
    client := NewClient(cfg, nil)

    // Verify internal httpClient has correct timeout
    if client.httpClient.Timeout != 15*time.Second {
        t.Errorf("expected timeout 15s, got %v", client.httpClient.Timeout)
    }
}

func TestDoRequest_SetsUserAgent(t *testing.T) {
    var capturedUA string
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        capturedUA = r.Header.Get("User-Agent")
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"test": "data"}`))
    }))
    defer server.Close()

    cfg := &config.APIConfig{Timeout: 5 * time.Second}
    client := NewClient(cfg, nil)

    var result map[string]string
    err := client.doRequest(context.Background(), server.URL, &result)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if capturedUA != "defillama-extract/1.0" {
        t.Errorf("expected User-Agent 'defillama-extract/1.0', got %q", capturedUA)
    }
}
```

### Project Structure Notes

- New file: `internal/api/client.go` - HTTP client wrapper
- New file: `internal/api/client_test.go` - tests co-located per convention
- Existing: `internal/api/doc.go` - package documentation placeholder (update if needed)
- Config package already has `APIConfig` struct with `Timeout`, `MaxRetries`, `RetryDelay` fields

### Learnings from Previous Story

**From Story 1-4-implement-structured-logging-with-slog (Status: done)**

- **Logging Package Ready:** `internal/logging.Setup(cfg.Logging)` returns configured `*slog.Logger`
- **slog.SetDefault Called:** Global logger already wired in `cmd/extractor/main.go`
- **Config Package Ready:** `internal/config.APIConfig` contains `Timeout`, `MaxRetries`, `RetryDelay` fields
- **Test Patterns:** Table-driven tests using stdlib testing package - follow similar patterns
- **Module Path:** `github.com/switchboard-xyz/defillama-extract` for imports
- **Files from Epic 1:** `internal/config/config.go`, `internal/logging/logging.go`, `cmd/extractor/main.go` all ready

[Source: docs/sprint-artifacts/1-4-implement-structured-logging-with-slog.md#Dev-Agent-Record]

### References

- [Source: docs/sprint-artifacts/tech-spec-epic-2.md#AC-2.1] - HTTP client configuration acceptance criteria
- [Source: docs/sprint-artifacts/tech-spec-epic-2.md#detailed-design] - Client struct and constructor design
- [Source: docs/epics/epic-2-api-integration.md#story-21] - Original story definition
- [Source: docs/prd.md#FR4] - User-Agent header requirement
- [Source: docs/prd.md#FR6] - Configurable timeout requirement (default 30s)
- [Source: docs/architecture/architecture-decision-records-adrs.md#adr-001] - ADR-001: Use net/http standard library
- [Source: docs/architecture/testing-strategy.md#Test-Organization] - Testing approach and mock server expectations
 - [Source: docs/architecture/project-structure.md#project-structure] - Package layout for placing client and tests

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/2-1-implement-base-http-client-with-timeout-and-user-agent.context.xml

### Agent Model Used

gpt-4o (2025-11-30)

### Debug Log References

- Planned and executed HTTP client implementation per ACs; ensured User-Agent constant injected on all requests and timeout sourced from config.
- Test matrix: unit (constructor defaults, user-agent, nil logger), integration (User-Agent capture, timeout net.Error, context cancellation) executed via `go test ./...`.
- Validation commands run: `go build ./...`, `go test ./internal/api/...`, `make lint`.

### Completion Notes List

- Implemented `internal/api/client.go` with configured timeouts, User-Agent constant, nil-logger fallback, and JSON request helper.
- Added comprehensive `client_test.go` covering constructor behavior, header injection, timeout and cancellation paths.

### File List

- internal/api/client.go
- internal/api/client_test.go
- docs/sprint-artifacts/sprint-status.yaml
- docs/sprint-artifacts/2-1-implement-base-http-client-with-timeout-and-user-agent.md
## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-2-api-integration.md and tech-spec-epic-2.md |
| 2025-11-30 | Amelia (Dev) | Implemented base HTTP client with User-Agent/timeout, added tests, updated sprint status |
| 2025-11-30 | Amelia (Reviewer) | Senior Developer Review (AI) - Approved |
| 2025-11-30 | Amelia (Dev) | Added non-2xx status test per review advisory |

## Senior Developer Review (AI)

Reviewer: BMad  
Date: 2025-11-30  
Outcome: Approve

### Summary
- ACs 5/5 implemented; all completed tasks verified; non-2xx status test added; no defects found.

### Key Findings
- None (no High/Med/Low issues identified).

### Acceptance Criteria Coverage

| AC# | Description | Status | Evidence |
| --- | ----------- | ------ | -------- |
| AC1 | http.Client uses configured timeout | IMPLEMENTED | internal/api/client.go:34; internal/api/client_test.go:20-27 |
| AC2 | User-Agent set to defillama-extract/1.0 on requests | IMPLEMENTED | internal/api/client.go:15,50; internal/api/client_test.go:50-76 |
| AC3 | Requests time out and surface timeout error | IMPLEMENTED | internal/api/client.go:34; internal/api/client_test.go:79-102 |
| AC4 | NewClient returns configured Client with timeout & UA | IMPLEMENTED | internal/api/client.go:27-40; internal/api/client_test.go:20-48 |
| AC5 | Context cancellation aborts request with context.Canceled | IMPLEMENTED | internal/api/client.go:45; internal/api/client_test.go:104-127 |

Summary: 5 of 5 acceptance criteria implemented.

### Task Completion Validation

| Task | Marked As | Verified As | Evidence |
| ---- | --------- | ----------- | -------- |
| Task 1: Client struct & constructor | complete | verified | internal/api/client.go:17-40 |
| Task 2: Request helper with UA injection | complete | verified | internal/api/client.go:43-66 |
| Task 3: Unit tests for initialization | complete | verified | internal/api/client_test.go:20-48 |
| Task 4: Integration tests (UA, timeout, cancel, decode) | complete | verified | internal/api/client_test.go:50-128 |
| Task 5: Verification commands (build, tests, lint) | complete | verified | go build ./...; go test ./...; make lint (2025-11-30) |

Summary: 5 of 5 completed tasks verified; 0 questionable; 0 false completions.

### Test Coverage and Gaps
- go test ./... passes; make lint passes; coverage includes timeout and cancellation paths. No gaps noted for scoped ACs.

### Architectural Alignment
- Uses stdlib net/http with context propagation; no external HTTP libs (ADR-001). User-Agent constant matches spec.

### Security Notes
- No secrets handled; requests use HTTPS; no additional concerns for this scope.

### Best-Practices and References
- Go stdlib http.Client timeout handling; context cancellation via http.NewRequestWithContext.

### Action Items

**Code Changes Required:**
- None.

**Advisory Notes:**
- None.
