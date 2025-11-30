# Story 2.2: Implement Oracle Endpoint Fetcher

Status: ready-for-dev

## Story

As a **developer**,
I want **to fetch oracle data from the `/oracles` endpoint**,
so that **I can retrieve TVS data and protocol-to-oracle mappings for downstream aggregation**.

## Acceptance Criteria

1. **Given** a configured API client **When** `FetchOracles(ctx context.Context)` is called **Then** a GET request is made to `https://api.llama.fi/oracles` with User-Agent header

2. **Given** a successful API response **When** the response is parsed **Then** the function returns `(*OracleAPIResponse, nil)` with all fields populated:
   - `Oracles`: map of oracle name to protocol slugs (`map[string][]string`)
   - `Chart`: historical TVS data by oracle/chain/timestamp (`map[string]map[string]map[string]float64`)
   - `OraclesTVS`: current TVS by oracle/chain (`map[string]map[string]map[string]float64`)
   - `ChainsByOracle`: chains where each oracle operates (`map[string][]string`)

3. **Given** an HTTP error (network failure, timeout, non-2xx status) **When** the request fails **Then** the function returns `(nil, error)` with descriptive error wrapping

4. **Given** a malformed JSON response **When** decoding fails **Then** the function returns `(nil, error)` with decode error details

5. **Given** a context with cancellation **When** the context is cancelled during the request **Then** the request is aborted and returns `context.Canceled` error

## Tasks / Subtasks

- [ ] Task 1: Define OracleAPIResponse struct (AC: 2)
  - [ ] 1.1: Create `internal/api/responses.go` file (or add to existing)
  - [ ] 1.2: Define `OracleAPIResponse` struct with JSON tags matching API response:
    ```go
    type OracleAPIResponse struct {
        Oracles        map[string][]string                       `json:"oracles"`
        Chart          map[string]map[string]map[string]float64  `json:"chart"`
        OraclesTVS     map[string]map[string]map[string]float64  `json:"oraclesTVS"`
        ChainsByOracle map[string][]string                       `json:"chainsByOracle"`
    }
    ```

- [ ] Task 2: Define endpoints constant (AC: 1)
  - [ ] 2.1: Create `internal/api/endpoints.go` file (or add to existing)
  - [ ] 2.2: Define `OraclesEndpoint = "https://api.llama.fi/oracles"` constant

- [ ] Task 3: Implement FetchOracles method (AC: 1, 2, 3, 4, 5)
  - [ ] 3.1: Add `FetchOracles(ctx context.Context) (*OracleAPIResponse, error)` method to `Client`
  - [ ] 3.2: Call `c.doRequest(ctx, OraclesEndpoint, &response)` using existing helper
  - [ ] 3.3: Return `(*OracleAPIResponse, nil)` on success
  - [ ] 3.4: Return `(nil, error)` on failure with wrapped error context

- [ ] Task 4: Write unit tests for OracleAPIResponse struct (AC: 2)
  - [ ] 4.1: Create test fixture `testdata/oracle_response.json` with sample response
  - [ ] 4.2: Test JSON unmarshaling populates all fields correctly
  - [ ] 4.3: Test optional/missing fields don't cause panic

- [ ] Task 5: Write integration tests with mock server (AC: 1, 2, 3, 4, 5)
  - [ ] 5.1: Test successful fetch returns populated OracleAPIResponse
  - [ ] 5.2: Test User-Agent header is present in request
  - [ ] 5.3: Test HTTP 500 returns wrapped error
  - [ ] 5.4: Test HTTP 404 returns wrapped error
  - [ ] 5.5: Test malformed JSON returns decode error
  - [ ] 5.6: Test context cancellation aborts request

- [ ] Task 6: Verification (AC: all)
  - [ ] 6.1: Run `go build ./...` and verify success
  - [ ] 6.2: Run `go test ./internal/api/...` and verify all pass
  - [ ] 6.3: Run `make lint` and verify no errors

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

### Completion Notes List

- Story drafted from epic/tech spec; awaiting context XML and ready-for-dev promotion.
- No code changes yet; implementation pending.

### File List

- docs/sprint-artifacts/2-2-implement-oracle-endpoint-fetcher.md
- docs/sprint-artifacts/validation-report-story-2-2-2025-11-30T08-18-41Z.md

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-2-api-integration.md and tech-spec-epic-2.md |
