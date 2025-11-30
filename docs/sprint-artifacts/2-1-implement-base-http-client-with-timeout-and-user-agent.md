# Story 2.1: Implement Base HTTP Client with Timeout and User-Agent

Status: ready-for-dev

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

- [ ] Task 1: Create Client struct and constructor (AC: 1, 2, 4)
  - [ ] 1.1: Create `internal/api/client.go` file
  - [ ] 1.2: Define `Client` struct with fields: `httpClient *http.Client`, `baseURL string`, `userAgent string`, `maxRetries int`, `retryDelay time.Duration`, `logger *slog.Logger`
  - [ ] 1.3: Implement `NewClient(cfg *config.APIConfig, logger *slog.Logger) *Client` constructor
  - [ ] 1.4: Set `http.Client.Timeout` from `cfg.Timeout` (default 30s)
  - [ ] 1.5: Store User-Agent string as `defillama-extract/1.0`

- [ ] Task 2: Implement request helper with User-Agent injection (AC: 2, 3, 5)
  - [ ] 2.1: Create `doRequest(ctx context.Context, url string, target any) error` method
  - [ ] 2.2: Use `http.NewRequestWithContext(ctx, "GET", url, nil)` for context support
  - [ ] 2.3: Set `req.Header.Set("User-Agent", c.userAgent)` on all requests
  - [ ] 2.4: Execute request with `c.httpClient.Do(req)` and check for errors
  - [ ] 2.5: Decode response body with `json.NewDecoder(resp.Body).Decode(target)`
  - [ ] 2.6: Close response body in defer statement

- [ ] Task 3: Write unit tests for client initialization (AC: 1, 4)
  - [ ] 3.1: Create `internal/api/client_test.go` file
  - [ ] 3.2: Test `NewClient` sets correct timeout from config
  - [ ] 3.3: Test `NewClient` stores User-Agent string correctly
  - [ ] 3.4: Test `NewClient` handles nil logger (should use slog.Default)

- [ ] Task 4: Write integration tests with mock server (AC: 2, 3, 5)
  - [ ] 4.1: Test User-Agent header is present using `httptest.NewServer`
  - [ ] 4.2: Test timeout behavior with delayed mock server response
  - [ ] 4.3: Test context cancellation aborts request
  - [ ] 4.4: Test successful JSON decode into target struct

- [ ] Task 5: Verification (AC: all)
  - [ ] 5.1: Run `go build ./...` and verify success
  - [ ] 5.2: Run `go test ./internal/api/...` and verify all pass
  - [ ] 5.3: Run `make lint` and verify no errors

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

- Validation pass: none recorded (story draft only)

### Completion Notes List

- Draft story created; awaiting architecture/testing citation updates and Dev Agent Record completion

### File List

- docs/sprint-artifacts/2-1-implement-base-http-client-with-timeout-and-user-agent.md
## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-2-api-integration.md and tech-spec-epic-2.md |
