# Story 7.2: Implement Protocol TVL Fetcher

Status: done

## Story

As a **service developer**,
I want **the API client to fetch historical TVL data from DefiLlama's per-protocol endpoint**,
So that **the TVL charting pipeline can retrieve TVL time-series data for all tracked protocols (auto-detected and custom)**.

## Acceptance Criteria

Source: [Source: docs/sprint-artifacts/tech-spec-epic-7.md#AC-7.2]; [Source: docs/epics/epic-7-custom-protocols-tvl-charting.md#Story-7.2]; [Source: docs/prd.md#growth-features-post-mvp]

**AC1: Fetch Protocol TVL Endpoint**
**Given** a valid protocol slug
**When** `FetchProtocolTVL(ctx, slug)` is called
**Then** it performs a GET request to `https://api.llama.fi/protocol/{slug}`
**And** uses the existing HTTP client configuration (timeout, User-Agent)

**AC2: Reuse Retry Logic**
**Given** a transient failure occurs (429, 5xx, network timeout)
**When** the request fails
**Then** it retries using the existing exponential backoff with jitter
**And** respects the configured `maxRetries` and `retryDelay` settings

**AC3: Extract Response Fields**
**Given** a successful API response
**When** the response is parsed
**Then** it extracts `name` (string) from the response
**And** extracts `tvl[]` array containing historical TVL data points
**And** extracts `currentChainTvls` map with per-chain current TVL values
**And** returns a `*ProtocolTVLResponse` struct with these fields populated

**AC4: Handle 404 Gracefully**
**Given** the protocol slug does not exist in DefiLlama
**When** the API returns HTTP 404
**Then** it returns `nil, nil` (not an error)
**And** logs a WARNING: `protocol_not_found slug=<slug> status_code=404`
**And** does NOT retry the request (404 is not retryable)

**AC5: Return Error on Other Failures**
**Given** the API returns a non-404 client error (401, 403) or server error persists after retries
**When** all retry attempts are exhausted
**Then** it returns an error wrapping the underlying failure
**And** the error message includes the endpoint and status code

**AC6: Respect Context Cancellation**
**Given** the context is cancelled during the request
**When** `ctx.Done()` fires
**Then** the request is aborted immediately
**And** the function returns `ctx.Err()` without additional retries

**AC7: Built-in Rate Limiting**
**Given** multiple protocols need to be fetched
**When** `FetchProtocolTVL` is called repeatedly
**Then** the fetcher enforces a minimum 200ms delay between successive calls (per process)
**And** no extra throttling is required by the caller

## Tasks / Subtasks

- [x] Task 1: Define TVL Response Models (AC: 3)
  - [x] 1.1: Create `ProtocolTVLResponse` struct in `internal/api/responses.go`:
    - `Name string` - Protocol display name
    - `TVL []TVLDataPoint` - Historical TVL array
    - `CurrentChainTvls map[string]float64` - Per-chain current TVL
  - [x] 1.2: Create `TVLDataPoint` struct:
    - `Date int64` - Unix timestamp
    - `TotalLiquidityUSD float64` - TVL value at that point
  - [x] 1.3: Add JSON tags matching DefiLlama response format

- [x] Task 2: Add Protocol Endpoint Constant (AC: 1)
  - [x] 2.1: Add `ProtocolTVLEndpointTemplate = "https://api.llama.fi/protocol/%s"` to `internal/api/endpoints.go`

- [x] Task 3: Implement FetchProtocolTVL Method (AC: 1, 2, 3, 5, 6)
  - [x] 3.1: Add `FetchProtocolTVL(ctx context.Context, slug string) (*ProtocolTVLResponse, error)` method to `Client`
  - [x] 3.2: Construct URL using `fmt.Sprintf(ProtocolTVLEndpointTemplate, slug)`
  - [x] 3.3: Use `doWithRetry` wrapper for retry logic
  - [x] 3.4: Parse response into `*ProtocolTVLResponse`
  - [x] 3.5: Return `nil, fmt.Errorf("fetch protocol TVL %s: %w", slug, err)` on failure

- [x] Task 4: Implement 404 Handling (AC: 4)
  - [x] 4.1: Create internal helper to detect 404 status from `APIError`
  - [x] 4.2: In `FetchProtocolTVL`, check if error is 404 before wrapping:
    - If 404: log warning, return `nil, nil`
    - If other error: return wrapped error
  - [x] 4.3: Add warning log with fields: `slug`, `status_code`

- [x] Task 5: Write Unit Tests (AC: all)
  - [x] 5.1: Create `internal/api/tvl_test.go`
  - [x] 5.2: Test: Successful fetch returns populated `ProtocolTVLResponse`
  - [x] 5.3: Test: 404 response returns `nil, nil` and logs warning
  - [x] 5.4: Test: 500 response triggers retries and returns error
  - [x] 5.5: Test: Context cancellation returns `context.Canceled`
  - [x] 5.6: Test: Invalid JSON response returns error
  - [x] 5.7: Test: Empty `tvl` array is handled (valid response, empty data)
  - [x] 5.8: Create test fixture `testdata/protocol_tvl_response.json`
  - [x] 5.9: Create test fixture `testdata/protocol_404_response.json`

- [x] Task 6: Build and Lint Verification (AC: all)
  - [x] 6.1: Run `go build ./...` and verify success
  - [x] 6.2: Run `go test ./internal/api/...` and verify all pass
  - [x] 6.3: Run `make lint` and fix any issues

- [x] Task 7: Implement Built-in Rate Limiting (AC: 7)
  - [x] 7.1: Add per-client rate limiter to `Client` (200ms minimum gap between calls to protocol endpoint)
  - [x] 7.2: Ensure rate limiter does not block cancellation (honor ctx)
  - [x] 7.3: Add unit test verifying ~200ms delay between sequential calls (use fake clock)
  - [x] 7.4: Add note in Dev Notes that caller no longer needs to throttle this endpoint

## Dev Notes

### Technical Guidance

**Files to Modify:**
- `internal/api/responses.go` - Add `ProtocolTVLResponse` and `TVLDataPoint` structs
- `internal/api/endpoints.go` - Add protocol TVL endpoint template
- `internal/api/client.go` - Add `FetchProtocolTVL` method

**Files to Create:**
- `internal/api/tvl_test.go` - Unit tests for TVL fetching
- `testdata/protocol_tvl_response.json` - Sample API response
- `testdata/protocol_404_response.json` - 404 error response

### Testing Strategy
- Follow project testing standards for table-driven tests and mock servers [Source: docs/architecture/testing-strategy.md]
- Cover success, 404 (no retry, warning log), retryable 5xx, invalid JSON, context cancellation, empty `tvl` array, and rate-limiter timing

**Rate Limiting Implementation Notes:**
- Use a per-client ticker or token bucket to enforce >=200ms between protocol requests
- Rate limiter must respect `ctx.Done()` to avoid blocking cancellation
- Callers do NOT add extra sleep for this endpoint; built-in limiter suffices

### Architecture Patterns and Constraints

- **ADR-001**: Reuse existing `Client` struct and `doWithRetry` wrapper - no new HTTP client [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-001]
- **ADR-003**: Explicit error returns; wrap errors with context (`fmt.Errorf("...: %w", err)`) [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-003]
- **ADR-004**: Structured logging with `slog` and fields for observability [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-004]
- **ADR-005**: No new external dependencies; use standard library [Source: docs/architecture/architecture-decision-records-adrs.md#ADR-005]

### API Response Reference

**Endpoint:** `GET https://api.llama.fi/protocol/{slug}`

**Sample Response (relevant fields):**
```json
{
  "name": "Drift Trade",
  "tvl": [
    {"date": 1704067200, "totalLiquidityUSD": 150000000},
    {"date": 1704153600, "totalLiquidityUSD": 155000000}
  ],
  "currentChainTvls": {
    "Solana": 677000000
  }
}
```

**Error Response (404):**
```json
{
  "message": "Protocol not found"
}
```

### Data Model Reference

**ProtocolTVLResponse** (to add in `responses.go`):
```go
// ProtocolTVLResponse represents the payload from GET /protocol/{slug}.
type ProtocolTVLResponse struct {
    Name            string             `json:"name"`
    TVL             []TVLDataPoint     `json:"tvl"`
    CurrentChainTvls map[string]float64 `json:"currentChainTvls"`
}

// TVLDataPoint represents a single point in TVL history.
type TVLDataPoint struct {
    Date             int64   `json:"date"`
    TotalLiquidityUSD float64 `json:"totalLiquidityUSD"`
}
```

### Project Structure Notes

- This story adds a new method to the existing `api.Client` struct - no new package needed
- Follow existing patterns in `client.go` for `FetchOracles` and `FetchProtocols` methods
- Test file follows naming convention: `tvl_test.go` in same package
- Test fixtures go in `testdata/` directory at project root or package level
- Aligns to documented layout in project structure [Source: docs/architecture/project-structure.md#project-structure]

### Dependency on Story 7.1

Story 7.1 (Load Custom Protocols Configuration) defines the `CustomProtocol` model and loader. This story (7.2) is independent and can be implemented in parallel since it only adds a fetch method to the API client.

**Integration point:** Story 7.3 (Merge Protocol Lists) will combine the protocol slugs from this fetcher with the custom protocols from Story 7.1.

### Existing Code to Reuse

From `internal/api/client.go`:
- `doWithRetry(ctx, fn)` - Retry wrapper with exponential backoff
- `doRequest(ctx, url, target)` - HTTP GET with JSON decode
- `APIError` - Error type with `IsRetryable()` method
- `isRetryable(statusCode, err)` - Retry decision logic

### 404 Handling Pattern

The 404 case requires special handling because:
1. `doWithRetry` will NOT retry 404 (it's a client error)
2. We want to return `nil, nil` (not an error) so the caller can continue with other protocols
3. Warning log is required for observability

**Implementation approach:**
```go
func (c *Client) FetchProtocolTVL(ctx context.Context, slug string) (*ProtocolTVLResponse, error) {
    url := fmt.Sprintf(ProtocolTVLEndpointTemplate, slug)
    var response ProtocolTVLResponse

    err := c.doWithRetry(ctx, func(ctx context.Context) error {
        return c.doRequest(ctx, url, &response)
    })

    if err != nil {
        var apiErr *APIError
        if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
            c.logger.Warn("protocol_not_found",
                "slug", slug,
                "status_code", apiErr.StatusCode,
            )
            return nil, nil
        }
        return nil, fmt.Errorf("fetch protocol TVL %s: %w", slug, err)
    }

    return &response, nil
}
```

### Testing Guidance

- Use `httptest.NewServer` for HTTP mocking (existing pattern in `internal/api/*_test.go`)
- Table-driven tests with named cases [Source: docs/architecture/testing-strategy.md]
- Test edge cases:
  - Empty `tvl` array (protocol exists but no history)
  - Missing `currentChainTvls` field
  - Large `tvl` array (1000+ data points)
  - Unicode characters in protocol name

### Known Risks

| Risk | Mitigation |
|------|------------|
| Response schema varies by protocol | Use `omitempty` for optional fields; don't fail on missing fields |
| Very large TVL arrays cause memory issues | Not a concern for <1000 protocols; revisit if scale increases |
| Slug encoding issues (special characters) | URL-encode slug; test with edge cases |
| Rate limiter starvation under bursty callers | Use non-blocking wait that respects context cancellation |

### References

- [Source: docs/sprint-artifacts/tech-spec-epic-7.md#AC-7.2] - Acceptance criteria definition
- [Source: docs/sprint-artifacts/tech-spec-epic-7.md#APIs-and-Interfaces] - New API client method spec
- [Source: docs/epics/epic-7-custom-protocols-tvl-charting.md#Story-7.2] - Epic story definition
- [Source: docs/epics/epic-7-custom-protocols-tvl-charting.md#API-Reference] - Protocol endpoint reference
- [Source: docs/architecture/architecture-decision-records-adrs.md] - ADRs governing implementation
- [Source: docs/architecture/implementation-patterns.md#Context-Propagation] - Context handling pattern
- [Source: docs/prd.md#growth-features-post-mvp] - PRD linkage for post-MVP growth features
- [Source: docs/architecture/project-structure.md#project-structure] - Project structure reference for placement
- [Source: docs/architecture/testing-strategy.md] - Testing standards and patterns

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/7-2-implement-protocol-tvl-fetcher.context.xml

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

- Implemented ProtocolTVLResponse models and endpoint constant; added FetchProtocolTVL with 404 handling and built-in 200ms rate limiter.
- Added comprehensive tvl tests (success, 404 warning, retries, cancel, invalid JSON, empty data, rate limiting) plus fixtures.
- Verified go test ./... and make lint.

### Completion Notes List

- Added TVL response models, protocol endpoint template, and FetchProtocolTVL with retry, context cancel, 404 warning handling, and per-client 200ms rate limiting.
- Created tvl unit tests with fixtures covering success, 404, retryable 500, invalid JSON, context cancellation, empty tvl, and rate limiter timing.
- Ran go test ./... and make lint; all passing.

### File List

- internal/api/endpoints.go
- internal/api/responses.go
- internal/api/client.go
- internal/api/tvl_test.go
- testdata/protocol_tvl_response.json
- testdata/protocol_404_response.json
- docs/sprint-artifacts/sprint-status.yaml
- docs/sprint-artifacts/7-2-implement-protocol-tvl-fetcher.md

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-12-08 | SM Agent (Bob) | Initial story draft created from Epic 7 / Tech Spec |
| 2025-12-08 | Amelia | Implemented protocol TVL fetcher, rate limiting, tests, and updated status |
| 2025-12-08 | Amelia | Senior Developer Review (AI) approved |

## Senior Developer Review (AI)

Reviewer: BMad  
Date: 2025-12-08  
Outcome: Approve (all ACs satisfied; no outstanding issues)

### Summary
- Implementation meets AC1-AC7 with built-in 200ms rate limiter, 404 handling, retry reuse, and structured logging.
- Comprehensive tests cover success, 404, retryable 500, invalid JSON, cancellation, empty data, and rate limiting; `go test ./...` passes.

### Key Findings
- No blocking or medium issues identified.
- Advisory: Consider URL-escaping slugs in `FetchProtocolTVL` to guard against unexpected characters (low severity).

### Acceptance Criteria Coverage

| AC | Description | Status | Evidence |
|---|---|---|---|
| AC1 | Uses GET https://api.llama.fi/protocol/{slug} with existing client config | IMPLEMENTED | Endpoint template added and used in fetcher (`internal/api/endpoints.go:5-7`, `internal/api/client.go:335-340`). |
| AC2 | Retries transient 429/5xx with exponential backoff, respects settings | IMPLEMENTED | `doWithRetry` + `isRetryable` reused for TVL fetch (`internal/api/client.go:152-193,240-303`). Server error test verifies retries (`internal/api/tvl_test.go:124-147`). |
| AC3 | Extracts name, tvl[], currentChainTvls into struct | IMPLEMENTED | Response models added (`internal/api/responses.go:30-41`); success test asserts fields (`internal/api/tvl_test.go:49-92`). |
| AC4 | 404 returns nil,nil, warns, no retry | IMPLEMENTED | 404 branch with warning log and early return (`internal/api/client.go:341-352`); test validates (`internal/api/tvl_test.go:94-122`). |
| AC5 | Non-404 failures wrap error with endpoint/status after retries | IMPLEMENTED | Error wrapping on failure (`internal/api/client.go:354-355`); 500 test asserts wrapped message and retries (`internal/api/tvl_test.go:124-147`). |
| AC6 | Honors context cancellation mid-request | IMPLEMENTED | Rate-limit wait and retry short-circuit on ctx (`internal/api/client.go:204-229,331-344`); cancellation test passes (`internal/api/tvl_test.go:169-188`). |
| AC7 | Enforces 200ms min gap between protocol calls | IMPLEMENTED | Per-client rate limiter (`internal/api/client.go:204-229,331-334`); timing test ensures delay (`internal/api/tvl_test.go:209-230`). |

### Task Completion Validation

| Task | Marked As | Verified As | Evidence |
|---|---|---|---|
| 1: Response models | [x] | VERIFIED COMPLETE | Added `ProtocolTVLResponse` & `TVLDataPoint` (`internal/api/responses.go:30-41`). |
| 2: Endpoint constant | [x] | VERIFIED COMPLETE | `ProtocolTVLEndpointTemplate` defined (`internal/api/endpoints.go:5-7`). |
| 3: FetchProtocolTVL method | [x] | VERIFIED COMPLETE | Method with retry, ctx handling (`internal/api/client.go:329-358`). |
| 4: 404 handling | [x] | VERIFIED COMPLETE | 404 detection helper + warning branch (`internal/api/client.go:195-202,341-352`). |
| 5: Unit tests + fixtures | [x] | VERIFIED COMPLETE | New tests and fixtures (`internal/api/tvl_test.go:49-230`, `testdata/protocol_tvl_response.json`, `testdata/protocol_404_response.json`). |
| 6: Build/test/lint verification | [x] | VERIFIED COMPLETE | `go test ./...` executed (2025-12-08). |
| 7: Rate limiting | [x] | VERIFIED COMPLETE | Per-client limiter and timing test (`internal/api/client.go:204-229`, `internal/api/tvl_test.go:209-230`). |

### Test Coverage and Gaps
- `go test ./...` passing (2025-12-08).
- Tests cover success, 404, retryable 500, invalid JSON, cancellation, empty data, and rate limiting. No gaps identified.

### Architectural Alignment
- Reuses existing client, retry wrapper, structured logging, and no new deps, matching ADR-001/003/004/005.
- Tech spec AC-7.2 requirements satisfied; architecture.md not present in repo (not loaded).

### Security Notes
- No secrets handled; low risk. Advisory to URL-escape slugs (low severity hardening).

### Best-Practices and References
- Go 1.24, stdlib HTTP client with context and retries; structured logging via `slog` aligns with project patterns.

### Action Items
- None.
