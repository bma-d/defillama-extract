# Epic 2: API Integration

**Goal:** Build a robust HTTP client that fetches data from DefiLlama APIs with proper error handling, retries, and parallel execution.

**User Value:** After this epic, the system can reliably fetch real oracle and protocol data from DefiLlama, handling transient failures automatically.

**FRs Covered:** FR1, FR2, FR3, FR4, FR5, FR6, FR7, FR8, FR55

> **MANDATORY:** A Tech Spec MUST be drafted before creating stories for this epic. Skipping the tech spec for Epic 2 was identified as a mistake in the Epic 2+3 retrospective (2025-11-30). The tech spec provides critical traceability for AC validation and review.

> **MANDATORY:** Each story MUST include a **Smoke Test Guide** in Dev Notes (or explicitly mark "Smoke test: N/A" for internal-only functions). Build/test/lint alone do not verify runtime behavior. This requirement was established in the Epic 2+3 retrospective (2025-11-30).

---

## Story 2.1: Implement Base HTTP Client with Timeout and User-Agent

As a **developer**,
I want **a configured HTTP client with proper timeout and identification headers**,
So that **API requests are well-behaved and identifiable**.

**Acceptance Criteria:**

**Given** API configuration with `timeout: 30s`
**When** the HTTP client is initialized
**Then** all requests use a 30-second timeout
**And** requests include header `User-Agent: defillama-extract/1.0`

**Given** an API request in progress
**When** the timeout duration elapses without response
**Then** the request is cancelled and returns a timeout error
**And** the error message indicates timeout occurred

**Given** a request to any DefiLlama endpoint
**When** the request is sent
**Then** the User-Agent header is present in the request

**Prerequisites:** Story 1.2 (config loading)

**Technical Notes:**
- Package: `internal/api/client.go`
- Create `Client` struct with `*http.Client` and config
- Constructor: `NewClient(cfg *config.APIConfig) *Client`
- Set `http.Client.Timeout` from config
- Create custom `RoundTripper` or add header in request helper
- Reference: 5-core-components.md section 5.1

---

## Story 2.2: Implement Oracle Endpoint Fetcher

As a **developer**,
I want **to fetch oracle data from the `/oracles` endpoint**,
So that **I can retrieve TVS data and protocol-to-oracle mappings**.

**Acceptance Criteria:**

**Given** a configured API client
**When** `FetchOracles(ctx context.Context)` is called
**Then** a GET request is made to `https://api.llama.fi/oracles`
**And** the response is parsed into `OracleAPIResponse` struct containing:
  - `Oracles`: map of oracle name → protocol slugs
  - `Chart`: historical TVS data by oracle/chain/timestamp
  - `OraclesTVS`: current TVS by oracle/protocol/chain (legacy timestamp dimension supported as fallback)
  - `ChainsByOracle`: chains where each oracle operates

**Given** a successful API response
**When** parsing completes
**Then** the function returns `(*OracleAPIResponse, error)` with nil error

**Given** an HTTP error (network failure, non-2xx status)
**When** the request fails
**Then** the function returns nil response and descriptive error

**Prerequisites:** Story 2.1

**Technical Notes:**
- Method: `func (c *Client) FetchOracles(ctx context.Context) (*models.OracleAPIResponse, error)`
- Use `http.NewRequestWithContext()` for cancellation support
- Response struct in `internal/models/api.go`
- Decode with `json.NewDecoder(resp.Body).Decode()`
- Reference: 3-data-sources-api-specifications.md, data-architecture.md

---

## Story 2.3: Implement Protocol Endpoint Fetcher

As a **developer**,
I want **to fetch protocol metadata from the `/lite/protocols2` endpoint**,
So that **I can retrieve protocol details like name, category, TVL, and chains**.

**Acceptance Criteria:**

**Given** a configured API client
**When** `FetchProtocols(ctx context.Context)` is called
**Then** a GET request is made to `https://api.llama.fi/lite/protocols2?b=2`
**And** the response is parsed into a slice of `Protocol` structs containing:
  - `ID`, `Name`, `Slug`: protocol identifiers
  - `Category`: protocol type (Lending, CDP, etc.)
  - `TVL`: total value locked
  - `Chains`: list of chains where protocol operates
  - `Oracles`: list of oracles used (array field)
  - `Oracle`: legacy single oracle field (string)
  - `URL`: protocol website

**Given** a successful API response
**When** parsing completes
**Then** the function returns `([]Protocol, error)` with nil error

**Given** protocols with missing optional fields (TVL, Chains, URL)
**When** parsing completes
**Then** those fields are zero-valued (0, nil, "") without error

**Prerequisites:** Story 2.1

**Technical Notes:**
- Method: `func (c *Client) FetchProtocols(ctx context.Context) ([]models.Protocol, error)`
- Protocol struct in `internal/models/protocol.go`
- Use `omitempty` JSON tags for optional fields
- Reference: 3-data-sources-api-specifications.md, data-architecture.md

---

## Story 2.4: Implement Retry Logic with Exponential Backoff

As a **developer**,
I want **automatic retries with exponential backoff for transient failures**,
So that **temporary API issues don't cause extraction failures**.

**Acceptance Criteria:**

**Given** API configuration with `max_retries: 3` and `retry_delay: 1s`
**When** a request fails with a retryable error (timeout, 429, 5xx)
**Then** the request is retried up to 3 times
**And** delays between retries follow exponential backoff: 1s, 2s, 4s
**And** jitter of ±25% is added to prevent thundering herd

**Given** a request that fails with 429 (rate limit)
**When** retries are attempted
**Then** each retry is logged at warn level with attempt number
**And** final failure after exhausting retries is logged at error level

**Given** a request that fails with 4xx (except 429)
**When** the error is detected
**Then** no retry is attempted (client error, not transient)
**And** error is returned immediately

**Given** a request that succeeds on retry attempt 2
**When** the response is received
**Then** the successful response is returned
**And** info log indicates "request succeeded after N retries"

**Prerequisites:** Story 2.1, Story 1.4 (logging)

**Technical Notes:**
- Add `doWithRetry(ctx, fn)` helper method to Client
- Retryable: timeout, 429, 500, 502, 503, 504
- Non-retryable: 400, 401, 403, 404
- Use `time.Sleep()` with jitter: `delay * (0.75 + rand.Float64()*0.5)`
- Log with slog: `slog.Warn("retrying request", "attempt", n, "error", err)`
- Reference: 9-error-handling-resilience.md

---

## Story 2.5: Implement Parallel Fetching with errgroup

As a **developer**,
I want **oracle and protocol data fetched in parallel**,
So that **total fetch time is minimized**.

**Acceptance Criteria:**

**Given** a need to fetch both oracle and protocol data
**When** `FetchAll(ctx context.Context)` is called
**Then** both API requests are initiated concurrently
**And** the function waits for both to complete
**And** total fetch time is approximately max(oracle_time, protocol_time), not sum

**Given** both requests succeed
**When** `FetchAll` returns
**Then** both responses are returned in a combined struct
**And** no error is returned

**Given** the oracle request fails but protocol succeeds
**When** `FetchAll` returns
**Then** an error is returned describing the oracle failure
**And** the context is cancelled for the protocol request (if still in progress)

**Given** context cancellation during fetch
**When** the parent context is cancelled
**Then** both in-flight requests are cancelled
**And** the function returns context.Canceled error

**Prerequisites:** Story 2.2, Story 2.3

**Technical Notes:**
- Use `golang.org/x/sync/errgroup` for coordination
- Create `FetchResult` struct with `OracleResponse` and `Protocols` fields
- Method: `func (c *Client) FetchAll(ctx context.Context) (*FetchResult, error)`
- `g, ctx := errgroup.WithContext(ctx)` for cancellation propagation
- Reference: implementation-patterns.md "Parallel Fetching" section

---

## Story 2.6: Implement API Request Logging

As an **operator**,
I want **API requests logged with timing and outcome**,
So that **I can monitor API health and debug issues**.

**Acceptance Criteria:**

**Given** an API request is initiated
**When** the request starts
**Then** debug log is emitted: `"starting API request"` with `url`, `method` attributes

**Given** an API request completes successfully
**When** response is received
**Then** info log is emitted: `"API request completed"` with `url`, `status`, `duration_ms` attributes

**Given** an API request fails
**When** error occurs
**Then** warn log is emitted: `"API request failed"` with `url`, `error`, `duration_ms`, `attempt` attributes

**Given** retry is attempted
**When** retry starts
**Then** warn log is emitted: `"retrying API request"` with `url`, `attempt`, `max_attempts`, `backoff_ms` attributes

**Prerequisites:** Story 2.4, Story 1.4

**Technical Notes:**
- Add logging to `doWithRetry` and fetch methods
- Use `time.Since(start).Milliseconds()` for duration
- Log at appropriate levels: debug (start), info (success), warn (retry/fail)
- Reference: FR55 - "System logs API request attempts, retries, and failures"

---
