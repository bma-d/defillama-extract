# Story 2.2: Implement Oracle Endpoint Fetcher

Status: done

## Story

As a **developer**,
I want **to fetch oracle data from the `/oracles` endpoint**,
so that **I can retrieve TVS data and protocol-to-oracle mappings for downstream aggregation**.

## Acceptance Criteria

1. **Given** a configured API client **When** `FetchOracles(ctx context.Context)` is called **Then** a GET request is made to `https://api.llama.fi/oracles` with User-Agent header

2. **Given** a successful API response **When** the response is parsed **Then** the function returns `(*OracleAPIResponse, nil)` with all fields populated:
   - `Oracles`: map of oracle name to protocol slugs (`map[string][]string`)
   - `Chart`: historical TVS data by oracle/chain/timestamp (`map[string]map[string]map[string]float64`)
   - `OraclesTVS`: current TVS by oracle/protocol/chain (fallback to legacy oracle/timestamp/chain) (`map[string]map[string]map[string]float64`)
   - `ChainsByOracle`: chains where each oracle operates (`map[string][]string`)

3. **Given** an HTTP error (network failure, timeout, non-2xx status) **When** the request fails **Then** the function returns `(nil, error)` with descriptive error wrapping

4. **Given** a malformed JSON response **When** decoding fails **Then** the function returns `(nil, error)` with decode error details

5. **Given** a context with cancellation **When** the context is cancelled during the request **Then** the request is aborted and returns `context.Canceled` error

## Tasks / Subtasks

- [x] Task 1: Define OracleAPIResponse struct (AC: 2)
  - [x] 1.1: Create `internal/api/responses.go` file (or add to existing)
  - [x] 1.2: Define `OracleAPIResponse` struct with JSON tags matching API response:
    ```go
    type OracleAPIResponse struct {
        Oracles        map[string][]string                       `json:"oracles"`
        Chart          map[string]map[string]map[string]float64  `json:"chart"`
        OraclesTVS     map[string]map[string]map[string]float64  `json:"oraclesTVS"` // oracle → protocol → chain (or legacy oracle → timestamp → chain)
        ChainsByOracle map[string][]string                       `json:"chainsByOracle"`
    }
    ```

- [x] Task 2: Define endpoints constant (AC: 1)
  - [x] 2.1: Create `internal/api/endpoints.go` file (or add to existing)
  - [x] 2.2: Define `OraclesEndpoint = "https://api.llama.fi/oracles"` constant

- [x] Task 3: Implement FetchOracles method (AC: 1, 2, 3, 4, 5)
  - [x] 3.1: Add `FetchOracles(ctx context.Context) (*OracleAPIResponse, error)` method to `Client`
  - [x] 3.2: Call `c.doRequest(ctx, OraclesEndpoint, &response)` using existing helper
  - [x] 3.3: Return `(*OracleAPIResponse, nil)` on success
  - [x] 3.4: Return `(nil, error)` on failure with wrapped error context

- [x] Task 4: Write unit tests for OracleAPIResponse struct (AC: 2)
  - [x] 4.1: Create test fixture `testdata/oracle_response.json` with sample response
  - [x] 4.2: Test JSON unmarshaling populates all fields correctly
  - [x] 4.3: Test optional/missing fields don't cause panic

- [x] Task 5: Write integration tests with mock server (AC: 1, 2, 3, 4, 5)
  - [x] 5.1: Test successful fetch returns populated OracleAPIResponse
  - [x] 5.2: Test User-Agent header is present in request
  - [x] 5.3: Test HTTP 500 returns wrapped error
  - [x] 5.4: Test HTTP 404 returns wrapped error
  - [x] 5.5: Test malformed JSON returns decode error
  - [x] 5.6: Test context cancellation aborts request

- [x] Task 6: Verification (AC: all)
  - [x] 6.1: Run `go build ./...` and verify success
  - [x] 6.2: Run `go test ./internal/api/...` and verify all pass
  - [x] 6.3: Run `make lint` and verify no errors

## Dev Notes

### Technical Guidance

- **Package Location:** `internal/api/responses.go` for struct, method in `internal/api/client.go`
- **Dependencies:** stdlib only (`net/http`, `context`, `encoding/json`)
- **ADR Alignment:** ADR-001 mandates `net/http` standard library usage

### Implementation Pattern

The `FetchOracles` method should leverage the existing `doRequest` helper from Story 2.1:

```go
// internal/api/client.go

// FetchOracles retrieves oracle TVS data from DefiLlama /oracles endpoint.
func (c *Client) FetchOracles(ctx context.Context) (*OracleAPIResponse, error) {
    var response OracleAPIResponse
    if err := c.doRequest(ctx, OraclesEndpoint, &response); err != nil {
        return nil, fmt.Errorf("fetch oracles: %w", err)
    }
    return &response, nil
}
```

### Response Structure Notes

The `/oracles` endpoint returns a complex nested structure:

```json
{
  "oracles": {
    "Chainlink": ["aave", "compound", ...],
    "Switchboard": ["marinade-finance", "solend", ...]
  },
  "chart": {
    "Chainlink": {
      "Ethereum": {
        "1699574400": 45000000000
      }
    }
  },
  "oraclesTVS": {
    "Chainlink": {
      "Ethereum": {
        "tvs": 45000000000
      }
    }
  },
  "chainsByOracle": {
    "Chainlink": ["Ethereum", "Arbitrum", ...],
    "Switchboard": ["Solana", "Sui", ...]
  }
}
```

### Test Fixture

Create `testdata/oracle_response.json` with realistic sample data for Switchboard:

```json
{
  "oracles": {
    "Switchboard": ["marinade-finance", "solend", "drift-protocol"]
  },
  "chart": {
    "Switchboard": {
      "Solana": {
        "1699574400": 500000000
      }
    }
  },
  "oraclesTVS": {
    "Switchboard": {
      "Solana": {
        "tvs": 500000000
      }
    }
  },
  "chainsByOracle": {
    "Switchboard": ["Solana", "Sui", "Aptos"]
  }
}
```

### Project Structure Notes

- New file: `internal/api/responses.go` - API response type definitions
- New file: `internal/api/endpoints.go` - URL constants
- Existing: `internal/api/client.go` - add FetchOracles method
- New file: `testdata/oracle_response.json` - test fixture

### Learnings from Previous Story

**From Story 2-1-implement-base-http-client-with-timeout-and-user-agent (Status: done)**

- **Client Infrastructure Ready:** `internal/api/client.go` with `NewClient()`, `doRequest()` helper, timeout, and User-Agent
- **doRequest Helper Available:** Use `c.doRequest(ctx, url, &target)` - handles User-Agent, timeout, JSON decode, error wrapping
- **Test Patterns Established:** Use `httptest.NewServer` for mock server tests, capture User-Agent header
- **Non-2xx Handling:** `doRequest` already returns error for non-2xx status codes
- **No Review Issues:** Story 2.1 passed review with no action items

[Source: docs/sprint-artifacts/2-1-implement-base-http-client-with-timeout-and-user-agent.md#Dev-Agent-Record]

### References

- [Source: docs/sprint-artifacts/tech-spec-epic-2.md#AC-2.2] - Oracle endpoint acceptance criteria
- [Source: docs/sprint-artifacts/tech-spec-epic-2.md#Data-Models-and-Contracts] - OracleAPIResponse struct definition
- [Source: docs/epics/epic-2-api-integration.md#story-22] - Original story definition
- [Source: docs/prd.md#FR1] - System fetches oracle data from `/oracles` endpoint
- [Source: docs/architecture/data-architecture.md] - API response models reference
- [Source: docs/architecture/testing-strategy.md#Test-Fixtures] - testdata directory for fixtures

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/2-2-implement-oracle-endpoint-fetcher.context.xml

### Agent Model Used

- gpt-4o (2025-11-30)

### Debug Log References

- Draft validation via *validate-create-story at 2025-11-30T08:18:41Z (see validation-report-story-2-2-2025-11-30T08-18-41Z.md)

### Debug Log

- Planned scope: define Oracle response model, add endpoint constant, implement FetchOracles using existing doRequest helper, and cover AC1-AC5 with unit+integration tests. Confirmed config fallback to default endpoint.

### Completion Notes List

- Implemented OracleAPIResponse model, OraclesEndpoint constant, and FetchOracles with error/context propagation using existing doRequest helper.
- Added fixture `testdata/oracle_response.json` plus unit tests for struct decoding and integration tests covering success, headers, non-2xx, malformed JSON, and context cancellation.
- Verification: `go build ./...`, `go test ./...`, `make lint` (all passing on 2025-11-30).

### File List

- docs/sprint-artifacts/2-2-implement-oracle-endpoint-fetcher.md
- docs/sprint-artifacts/validation-report-story-2-2-2025-11-30T08-18-41Z.md
- docs/sprint-artifacts/sprint-status.yaml
- internal/api/client.go
- internal/api/endpoints.go
- internal/api/responses.go
- internal/api/client_test.go
- internal/api/oracles_test.go
- internal/api/responses_test.go
- testdata/oracle_response.json

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-2-api-integration.md and tech-spec-epic-2.md |
| 2025-11-30 | Amelia | Implemented Oracle fetcher, response model, constants, tests, and verified build/test/lint |
| 2025-11-30 | Amelia | Senior Developer Review (AI) appended; outcome Approve |

## Senior Developer Review (AI)

- **Reviewer:** BMad
- **Date:** 2025-11-30
- **Outcome:** Approve — All ACs satisfied; tests and lint pass
- **Summary:** FetchOracles uses configured `/oracles` URL with User-Agent injection, returns typed response, and surfaces status/decode/context errors with wrapping. Unit and integration tests cover happy path, status errors, decode failures, and cancellation; `go build`, `go test ./...`, and `make lint` pass.

**Key Findings (by severity)**
- None.

**Acceptance Criteria Coverage**

| AC | Description | Status | Evidence |
| --- | --- | --- | --- |
| AC1 | GET https://api.llama.fi/oracles invoked with User-Agent | IMPLEMENTED | `internal/api/client.go:49-65` request with UA; `internal/api/endpoints.go:5` endpoint; `internal/api/oracles_test.go:56-74` asserts header |
| AC2 | Successful response populates OracleAPIResponse fields | IMPLEMENTED | `internal/api/responses.go:3-8` struct; `internal/api/oracles_test.go:27-53` verifies map contents; `internal/api/responses_test.go:10-33` fixture decode |
| AC3 | HTTP errors return wrapped error | IMPLEMENTED | `internal/api/client.go:63-79` status guard + wrapping; `internal/api/oracles_test.go:77-96` 500/404 cases |
| AC4 | Malformed JSON returns decode error | IMPLEMENTED | `internal/api/client.go:67-79` decode wrap; `internal/api/oracles_test.go:99-115` malformed JSON case |
| AC5 | Context cancellation aborts request and surfaces context.Canceled | IMPLEMENTED | `internal/api/client.go:50-60` context-bound request; `internal/api/oracles_test.go:118-139` cancellation scenario |

**Task Completion Validation**

| Task | Marked As | Verified As | Evidence |
| --- | --- | --- | --- |
| Task 1: OracleAPIResponse struct | [x] | Verified complete | `internal/api/responses.go:3-8`; `internal/api/responses_test.go:10-50` |
| Task 2: OraclesEndpoint constant | [x] | Verified complete | `internal/api/endpoints.go:3-5` |
| Task 3: FetchOracles method | [x] | Verified complete | `internal/api/client.go:74-81`; `internal/api/oracles_test.go:27-74` |
| Task 4: Unit tests for struct | [x] | Verified complete | `internal/api/responses_test.go:10-50`; `testdata/oracle_response.json` |
| Task 5: Integration tests with mock server | [x] | Verified complete | `internal/api/oracles_test.go:27-139` |
| Task 6: Verification commands | [x] | Verified complete | `go build ./...`, `go test ./...`, `make lint` (2025-11-30) |

**Test Coverage and Gaps**
- `go test ./...` (pass) — covers success, header injection, status errors, malformed JSON, and context cancellation for FetchOracles; fixture decode paths covered. No gaps identified for ACs.

**Architectural Alignment**
- Uses stdlib `net/http` with context propagation; HTTPS endpoint constant; error wrapping uses `%w`; no external HTTP deps.

**Security Notes**
- HTTPS endpoint; no secrets handled; no additional risks observed.

**Best-Practices and References**
- Data model matches `docs/architecture/data-architecture.md` (OracleAPIResponse).

### Action Items

**Code Changes Required:**
- [ ] None (no changes requested).

**Advisory Notes:**
- Note: No additional advisories.
