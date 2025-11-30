# Epic Technical Specification: API Integration

Date: 2025-11-30
Author: BMad
Epic ID: 2
Status: Approved

---

## Overview

Epic 2 builds the HTTP client layer that fetches data from DefiLlama's public APIs. This epic transforms the application from a configured shell (Epic 1) into a system that can retrieve real oracle and protocol data from external sources. The client must handle network failures gracefully, identify itself properly to the API, and execute requests efficiently through parallel fetching.

This epic directly implements the data acquisition layer referenced in the architecture's integration diagram, positioning `internal/api` as the bridge between DefiLlama's REST endpoints and the aggregation pipeline that will process the data in subsequent epics.

## Objectives and Scope

**In Scope:**
- HTTP client with configurable timeout (default 30s)
- User-Agent header identification (`defillama-extract/1.0`)
- Oracle endpoint fetcher (`GET /oracles`)
- Protocol endpoint fetcher (`GET /lite/protocols2?b=2`)
- Retry logic with exponential backoff and jitter
- Parallel fetching using `errgroup`
- Request/response logging with timing metrics

**Out of Scope:**
- Protocol filtering by oracle name (Epic 3)
- Data aggregation and metric calculation (Epic 3)
- Response caching or persistence (not required per architecture)
- Authentication (DefiLlama APIs are public)
- Rate limiting implementation (handled via conservative polling interval in scheduler)

## System Architecture Alignment

This epic implements the `internal/api` package as defined in the project structure:

```
internal/api/
├── client.go        # HTTP client with retry logic
├── client_test.go   # Client tests with mock server
├── endpoints.go     # API URL constants
└── responses.go     # DefiLlama API response structs
```

**Architectural Constraints:**
- Uses Go standard library `net/http` (ADR-001: No external HTTP frameworks)
- Context propagation required for all I/O operations
- Explicit error returns over exceptions (ADR-003)
- Structured logging with `slog` (ADR-004)
- Minimal external dependencies - only `golang.org/x/sync/errgroup` added (ADR-005)

## Detailed Design

### Services and Modules

| Module | File | Responsibility | Inputs | Outputs |
|--------|------|----------------|--------|---------|
| Client | `client.go` | HTTP client wrapper with timeout, User-Agent, retry logic | `*config.APIConfig` | HTTP responses |
| Endpoints | `endpoints.go` | URL constants for DefiLlama APIs | None | String constants |
| Responses | `responses.go` | Type definitions for API responses | JSON bytes | Parsed structs |

**Client Struct:**
```go
type Client struct {
    httpClient *http.Client
    baseURL    string
    userAgent  string
    maxRetries int
    retryDelay time.Duration
    logger     *slog.Logger
}
```

**Constructor:**
```go
func NewClient(cfg *config.APIConfig, logger *slog.Logger) *Client
```

### Data Models and Contracts

**OracleAPIResponse** (from `GET /oracles`):
```go
type OracleAPIResponse struct {
    Oracles        map[string][]string                       `json:"oracles"`        // oracle name → protocol slugs
    Chart          map[string]map[string]map[string]float64  `json:"chart"`          // historical TVS by oracle/chain/timestamp
    OraclesTVS     map[string]map[string]map[string]float64  `json:"oraclesTVS"`     // current TVS by oracle/chain/period
    ChainsByOracle map[string][]string                       `json:"chainsByOracle"` // oracle name → chains
}
```

**Protocol** (from `GET /lite/protocols2?b=2`):
```go
type Protocol struct {
    ID       string   `json:"id"`
    Name     string   `json:"name"`
    Slug     string   `json:"slug"`
    Category string   `json:"category"`
    TVL      float64  `json:"tvl,omitempty"`
    Chains   []string `json:"chains,omitempty"`
    Oracles  []string `json:"oracles,omitempty"`  // Array field (preferred)
    Oracle   string   `json:"oracle,omitempty"`   // Legacy single field
    URL      string   `json:"url,omitempty"`
}
```

**FetchResult** (combined response):
```go
type FetchResult struct {
    OracleResponse *OracleAPIResponse
    Protocols      []Protocol
}
```

**APIError** (custom error type):
```go
type APIError struct {
    Endpoint   string
    StatusCode int
    Message    string
    Err        error
}

func (e *APIError) IsRetryable() bool {
    return e.StatusCode == 429 || e.StatusCode >= 500
}
```

### APIs and Interfaces

**Client Methods:**

| Method | Signature | Purpose |
|--------|-----------|---------|
| FetchOracles | `(ctx context.Context) (*OracleAPIResponse, error)` | Fetch oracle TVS data |
| FetchProtocols | `(ctx context.Context) ([]Protocol, error)` | Fetch protocol metadata |
| FetchAll | `(ctx context.Context) (*FetchResult, error)` | Parallel fetch both endpoints |

**Internal Helpers:**

| Method | Signature | Purpose |
|--------|-----------|---------|
| doRequest | `(ctx context.Context, url string, target any) error` | Execute HTTP GET with JSON decode |
| doWithRetry | `(ctx context.Context, fn func() error) error` | Retry wrapper with backoff |
| isRetryable | `(statusCode int) bool` | Determine if error is transient |

**DefiLlama Endpoints:**

| Constant | Value | Response Type |
|----------|-------|---------------|
| OraclesEndpoint | `https://api.llama.fi/oracles` | OracleAPIResponse |
| ProtocolsEndpoint | `https://api.llama.fi/lite/protocols2?b=2` | []Protocol |

### Workflows and Sequencing

**Single Request Flow:**
```
FetchOracles(ctx)
    │
    ├─► doWithRetry(ctx, fn)
    │       │
    │       ├─► doRequest(ctx, url, target)
    │       │       │
    │       │       ├─► http.NewRequestWithContext(ctx, "GET", url, nil)
    │       │       ├─► req.Header.Set("User-Agent", userAgent)
    │       │       ├─► httpClient.Do(req)
    │       │       │       │
    │       │       │       ├─► [Success 2xx] json.Decode → return nil
    │       │       │       ├─► [Retryable 429/5xx] return APIError
    │       │       │       └─► [Non-retryable 4xx] return APIError (no retry)
    │       │       │
    │       │       └─► Log request completion with duration
    │       │
    │       ├─► [Error + Retryable] sleep(delay * 2^attempt + jitter) → retry
    │       ├─► [Error + Non-retryable] return error immediately
    │       └─► [Max retries exceeded] return error
    │
    └─► Return (*OracleAPIResponse, error)
```

**Parallel Fetch Flow (FetchAll):**
```
FetchAll(ctx)
    │
    ├─► g, ctx := errgroup.WithContext(ctx)
    │
    ├─► g.Go(func() { oracleResp = FetchOracles(ctx) })
    │
    ├─► g.Go(func() { protocols = FetchProtocols(ctx) })
    │
    ├─► g.Wait()
    │       │
    │       ├─► [Both succeed] return &FetchResult{...}, nil
    │       ├─► [One fails] context cancelled, return first error
    │       └─► [Context cancelled] return context.Canceled
    │
    └─► Return (*FetchResult, error)
```

**Retry Timing (exponential backoff with jitter):**
```
Attempt 1: immediate
Attempt 2: 1s × (0.75 + rand×0.5) = 750ms - 1250ms
Attempt 3: 2s × (0.75 + rand×0.5) = 1500ms - 2500ms
Attempt 4: 4s × (0.75 + rand×0.5) = 3000ms - 5000ms
[Max retries exceeded → fail]
```

## Non-Functional Requirements

### Performance

| Requirement | Target | Implementation |
|-------------|--------|----------------|
| NFR1: Request timeout | 30 seconds | `http.Client.Timeout = 30 * time.Second` |
| NFR3: Parallel fetching | Total time ≈ max(oracle_time, protocol_time) | `errgroup.WithContext` for concurrent requests |

**Measurable Targets:**
- Individual API requests complete within 30 seconds
- Combined `FetchAll` operation completes within 35 seconds (accounting for overhead)
- No blocking operations outside of HTTP I/O

### Security

| Concern | Mitigation |
|---------|------------|
| API identification | User-Agent header: `defillama-extract/1.0` |
| No sensitive data | DefiLlama APIs are public, no auth required |
| Input validation | JSON decoder handles malformed responses gracefully |
| TLS enforcement | All endpoints use HTTPS (enforced by URL constants) |

**Note:** No credentials, tokens, or secrets are handled by this epic. All API calls are unauthenticated reads from public endpoints.

### Reliability/Availability

| Requirement | Implementation |
|-------------|----------------|
| NFR6: Transient failure recovery | Automatic retries (max 3) with exponential backoff |
| NFR9: Unexpected response handling | JSON decoder returns error, doesn't panic |
| NFR10: Graceful degradation | Errors logged and propagated, daemon continues next cycle |

**Retryable Conditions:**
- HTTP 429 (Rate Limited)
- HTTP 500 (Internal Server Error)
- HTTP 502 (Bad Gateway)
- HTTP 503 (Service Unavailable)
- HTTP 504 (Gateway Timeout)
- Network timeouts
- Connection refused (transient)

**Non-Retryable Conditions:**
- HTTP 400 (Bad Request)
- HTTP 401 (Unauthorized)
- HTTP 403 (Forbidden)
- HTTP 404 (Not Found)
- JSON decode errors (malformed response)

### Observability

| Log Event | Level | Attributes |
|-----------|-------|------------|
| Request start | Debug | `url`, `method` |
| Request success | Info | `url`, `status`, `duration_ms` |
| Request failure | Warn | `url`, `error`, `duration_ms`, `attempt` |
| Retry attempt | Warn | `url`, `attempt`, `max_attempts`, `backoff_ms` |
| Max retries exceeded | Error | `url`, `total_attempts`, `final_error` |

**Logging Implementation (FR55):**
```go
slog.Info("API request completed",
    "url", url,
    "status", resp.StatusCode,
    "duration_ms", time.Since(start).Milliseconds(),
)
```

## Dependencies and Integrations

### Go Module Dependencies

**Current (go.mod):**
```go
module github.com/switchboard-xyz/defillama-extract

go 1.23

require gopkg.in/yaml.v3 v3.0.1
```

**New Dependency for This Epic:**
```go
require golang.org/x/sync v0.10.0  // For errgroup
```

### Standard Library Packages Used

| Package | Purpose |
|---------|---------|
| `net/http` | HTTP client and request handling |
| `context` | Request cancellation and timeout |
| `encoding/json` | JSON response decoding |
| `time` | Timeout and backoff timing |
| `log/slog` | Structured logging |
| `math/rand` | Jitter calculation |
| `errors` | Error wrapping and sentinel errors |
| `fmt` | Error formatting |

### External Integration Points

| System | Endpoint | Protocol | Auth |
|--------|----------|----------|------|
| DefiLlama | `https://api.llama.fi/oracles` | HTTPS/REST | None |
| DefiLlama | `https://api.llama.fi/lite/protocols2?b=2` | HTTPS/REST | None |

### Internal Dependencies

| Depends On | From Epic | Purpose |
|------------|-----------|---------|
| `internal/config` | Epic 1 | API configuration (timeout, retries, URLs) |
| `*slog.Logger` | Epic 1 | Structured logging instance |

### Downstream Consumers

| Consumer | Epic | Purpose |
|----------|------|---------|
| `internal/aggregator` | Epic 3 | Receives `FetchResult` for filtering and aggregation |

## Acceptance Criteria (Authoritative)

### AC-2.1: HTTP Client Configuration
1. HTTP client uses configurable timeout from `config.api.timeout` (default: 30s)
2. All requests include `User-Agent: defillama-extract/1.0` header
3. Requests exceeding timeout are cancelled and return timeout error
4. Client is instantiated via `NewClient(cfg, logger)` constructor

### AC-2.2: Oracle Endpoint Fetcher
1. `FetchOracles(ctx)` sends GET to `https://api.llama.fi/oracles`
2. Response is decoded into `OracleAPIResponse` struct with all fields populated
3. Returns `(*OracleAPIResponse, nil)` on success
4. Returns `(nil, error)` on HTTP or decode failure

### AC-2.3: Protocol Endpoint Fetcher
1. `FetchProtocols(ctx)` sends GET to `https://api.llama.fi/lite/protocols2?b=2`
2. Response is decoded into `[]Protocol` slice
3. Protocols with missing optional fields (TVL, Chains, URL) have zero values
4. Returns `([]Protocol, nil)` on success
5. Returns `(nil, error)` on HTTP or decode failure

### AC-2.4: Retry Logic
1. Failed requests are retried up to `config.api.max_retries` times (default: 3)
2. Backoff follows exponential pattern: `delay * 2^attempt`
3. Jitter of ±25% is applied to backoff delay
4. Retries occur for: timeout, 429, 500, 502, 503, 504
5. No retry for: 400, 401, 403, 404 (immediate failure)
6. Each retry is logged at WARN level with attempt number
7. Final failure after max retries is logged at ERROR level

### AC-2.5: Parallel Fetching
1. `FetchAll(ctx)` fetches oracle and protocol endpoints concurrently
2. Total duration approximates max(oracle_time, protocol_time)
3. If one request fails, context is cancelled for the other
4. Returns `(*FetchResult, nil)` when both succeed
5. Returns `(nil, error)` describing first failure

### AC-2.6: Request Logging
1. Request start logged at DEBUG: `"starting API request"` with `url`, `method`
2. Request success logged at INFO: `"API request completed"` with `url`, `status`, `duration_ms`
3. Request failure logged at WARN: `"API request failed"` with `url`, `error`, `duration_ms`, `attempt`
4. Retry logged at WARN: `"retrying API request"` with `url`, `attempt`, `max_attempts`, `backoff_ms`

## Traceability Mapping

| AC | Spec Section | Component/API | Test Idea |
|----|--------------|---------------|-----------|
| AC-2.1 | NFR1, NFR11 | `Client`, `NewClient()` | Unit: verify timeout applied; Unit: verify User-Agent header present |
| AC-2.2 | FR1, FR4 | `FetchOracles()`, `OracleAPIResponse` | Integration: mock server returns sample JSON; Unit: verify struct fields populated |
| AC-2.3 | FR2, FR4 | `FetchProtocols()`, `Protocol` | Integration: mock server returns sample JSON; Unit: optional fields default to zero |
| AC-2.4 | FR5, FR7, FR8 | `doWithRetry()`, `APIError` | Unit: verify retry count; Unit: verify backoff timing; Unit: verify non-retryable stops immediately |
| AC-2.5 | FR3 | `FetchAll()`, `errgroup` | Integration: measure parallel vs sequential timing; Unit: verify one failure cancels other |
| AC-2.6 | FR55 | `doRequest()`, `doWithRetry()` | Unit: capture log output; verify correct levels and attributes |

### FR to Story Mapping

| FR | Story | Description |
|----|-------|-------------|
| FR1 | 2.2 | System fetches oracle data from `/oracles` endpoint |
| FR2 | 2.3 | System fetches protocol metadata from `/lite/protocols2` endpoint |
| FR3 | 2.5 | System fetches both endpoints in parallel |
| FR4 | 2.1 | System includes proper User-Agent header |
| FR5 | 2.4 | System retries failed requests with exponential backoff |
| FR6 | 2.1 | System respects configurable timeout (default 30s) |
| FR7 | 2.4 | System handles API errors gracefully (429, 5xx) with retry |
| FR8 | 2.4 | System detects non-retryable errors (4xx) |
| FR55 | 2.6 | System logs API request attempts, retries, and failures |

## Risks, Assumptions, Open Questions

### Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| DefiLlama API schema changes | Low | Medium | JSON decoder ignores unknown fields; log warnings for missing expected fields |
| DefiLlama API downtime | Low | High | Retry logic handles transient failures; daemon mode continues next cycle |
| Rate limiting by DefiLlama | Low | Medium | Conservative 2-hour polling interval; exponential backoff on 429 |
| Large response payloads causing memory pressure | Low | Low | Stream decode directly from response body; no buffering |

### Assumptions

| Assumption | Validation |
|------------|------------|
| DefiLlama APIs remain publicly accessible without authentication | Verified via API documentation and current behavior |
| Response schema matches documented structure | Test fixtures based on actual API responses |
| Network latency typically under 5 seconds | Timeout set to 30s provides generous margin |
| `golang.org/x/sync/errgroup` is a stable, well-maintained package | Part of official Go extended libraries |

### Open Questions

| Question | Owner | Resolution Path |
|----------|-------|-----------------|
| Should we add circuit breaker for repeated failures? | Developer | Defer to post-MVP; current retry logic sufficient for MVP |
| Should response bodies be logged at DEBUG level? | Developer | Implement if debugging proves difficult; consider size limits |

## Test Strategy Summary

### Test Types

| Type | Location | Coverage Target |
|------|----------|-----------------|
| Unit Tests | `internal/api/*_test.go` | All public methods, error paths |
| Integration Tests | `internal/api/client_test.go` | Mock server with realistic responses |
| Table-Driven Tests | All test files | Multiple scenarios per test function |

### Test Fixtures

| Fixture | Location | Purpose |
|---------|----------|---------|
| `oracle_response.json` | `testdata/` | Sample `/oracles` response |
| `protocol_response.json` | `testdata/` | Sample `/lite/protocols2` response |

### Key Test Scenarios

**Client Configuration:**
- Verify timeout is applied to HTTP client
- Verify User-Agent header is sent
- Verify context cancellation propagates

**Oracle Fetcher:**
- Success: mock server returns valid JSON → struct populated
- Failure: mock server returns 500 → error returned
- Malformed: mock server returns invalid JSON → decode error

**Protocol Fetcher:**
- Success: mock server returns array of protocols
- Optional fields: protocols with missing TVL/Chains → zero values
- Empty response: mock server returns `[]` → empty slice, no error

**Retry Logic:**
- Retryable: 429 → retries up to max, then fails
- Non-retryable: 404 → immediate failure, no retry
- Success on retry: fails twice, succeeds third → returns success
- Timing: verify backoff delays approximate expected values

**Parallel Fetching:**
- Both succeed: verify both responses returned
- One fails: verify error returned, context cancelled
- Performance: verify parallel is faster than sequential

### Coverage Requirements

| Component | Target |
|-----------|--------|
| `client.go` | 90%+ line coverage |
| `endpoints.go` | 100% (constants only) |
| `responses.go` | N/A (type definitions only) |
| Error handling paths | All paths tested |
| Retry logic | All branches tested |

### Mock Server Pattern

```go
func TestFetchOracles(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify User-Agent
        assert.Equal(t, "defillama-extract/1.0", r.Header.Get("User-Agent"))

        // Return fixture
        data, _ := os.ReadFile("testdata/oracle_response.json")
        w.Header().Set("Content-Type", "application/json")
        w.Write(data)
    }))
    defer server.Close()

    client := NewClient(&config.APIConfig{
        OraclesURL: server.URL,
        Timeout:    30 * time.Second,
    }, slog.Default())

    resp, err := client.FetchOracles(context.Background())
    assert.NoError(t, err)
    assert.NotNil(t, resp.Oracles)
}
```
