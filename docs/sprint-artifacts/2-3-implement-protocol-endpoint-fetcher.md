# Story 2.3: Implement Protocol Endpoint Fetcher

Status: ready-for-dev

## Story

As a **developer**,
I want **to fetch protocol metadata from the `/lite/protocols2` endpoint**,
so that **I can retrieve protocol details like name, category, TVL, chains, and oracle associations for downstream filtering and aggregation**.

## Acceptance Criteria

1. **Given** a configured API client **When** `FetchProtocols(ctx context.Context)` is called **Then** a GET request is made to `https://api.llama.fi/lite/protocols2?b=2` with User-Agent header

2. **Given** a successful API response **When** the response is parsed **Then** the function returns `([]Protocol, nil)` with all fields populated:
   - `ID`, `Name`, `Slug`: protocol identifiers (string)
   - `Category`: protocol type (Lending, CDP, etc.) (string)
   - `TVL`: total value locked (float64, optional)
   - `Chains`: list of chains where protocol operates ([]string, optional)
   - `Oracles`: list of oracles used - array field ([]string, optional)
   - `Oracle`: legacy single oracle field (string, optional)
   - `URL`: protocol website (string, optional)

3. **Given** protocols with missing optional fields (TVL, Chains, URL, Oracles, Oracle) **When** parsing completes **Then** those fields are zero-valued (0, nil, "", nil) without error

4. **Given** an HTTP error (network failure, timeout, non-2xx status) **When** the request fails **Then** the function returns `(nil, error)` with descriptive error wrapping

5. **Given** a malformed JSON response **When** decoding fails **Then** the function returns `(nil, error)` with decode error details

6. **Given** a context with cancellation **When** the context is cancelled during the request **Then** the request is aborted and returns `context.Canceled` error

7. **Given** an empty response array `[]` **When** parsing completes **Then** the function returns `([]Protocol{}, nil)` - empty slice with no error

## Tasks / Subtasks

- [ ] Task 1: Define Protocol struct (AC: 2, 3)
  - [ ] 1.1: Add `Protocol` struct to `internal/api/responses.go`
  - [ ] 1.2: Include all fields with proper JSON tags and `omitempty` for optional fields:
    ```go
    type Protocol struct {
        ID       string   `json:"id"`
        Name     string   `json:"name"`
        Slug     string   `json:"slug"`
        Category string   `json:"category"`
        TVL      float64  `json:"tvl,omitempty"`
        Chains   []string `json:"chains,omitempty"`
        Oracles  []string `json:"oracles,omitempty"`
        Oracle   string   `json:"oracle,omitempty"`
        URL      string   `json:"url,omitempty"`
    }
    ```

- [ ] Task 2: Define ProtocolsEndpoint constant (AC: 1)
  - [ ] 2.1: Add `ProtocolsEndpoint = "https://api.llama.fi/lite/protocols2?b=2"` to `internal/api/endpoints.go`

- [ ] Task 3: Implement FetchProtocols method (AC: 1, 2, 3, 4, 5, 6, 7)
  - [ ] 3.1: Add `FetchProtocols(ctx context.Context) ([]Protocol, error)` method to `Client` in `internal/api/client.go`
  - [ ] 3.2: Call `c.doRequest(ctx, ProtocolsEndpoint, &protocols)` using existing helper
  - [ ] 3.3: Return `(protocols, nil)` on success
  - [ ] 3.4: Return `(nil, error)` on failure with wrapped error context: `fmt.Errorf("fetch protocols: %w", err)`

- [ ] Task 4: Write unit tests for Protocol struct (AC: 2, 3)
  - [ ] 4.1: Create test fixture `testdata/protocol_response.json` with sample response including protocols with all fields and some with missing optional fields
  - [ ] 4.2: Add tests to `internal/api/responses_test.go` for Protocol struct JSON unmarshaling
  - [ ] 4.3: Test JSON unmarshaling populates all fields correctly
  - [ ] 4.4: Test protocols with missing optional fields have zero values

- [ ] Task 5: Write integration tests with mock server (AC: 1, 2, 3, 4, 5, 6, 7)
  - [ ] 5.1: Create `internal/api/protocols_test.go` for FetchProtocols tests
  - [ ] 5.2: Test successful fetch returns populated []Protocol slice
  - [ ] 5.3: Test User-Agent header is present in request
  - [ ] 5.4: Test HTTP 500 returns wrapped error
  - [ ] 5.5: Test HTTP 404 returns wrapped error
  - [ ] 5.6: Test malformed JSON returns decode error
  - [ ] 5.7: Test context cancellation aborts request
  - [ ] 5.8: Test empty array response `[]` returns empty slice without error

- [ ] Task 6: Verification (AC: all)
  - [ ] 6.1: Run `go build ./...` and verify success
  - [ ] 6.2: Run `go test ./internal/api/...` and verify all pass
  - [ ] 6.3: Run `make lint` and verify no errors

## Dev Notes

### Technical Guidance

- **Package Location:** `Protocol` struct in `internal/api/responses.go`, method in `internal/api/client.go`
- **Dependencies:** stdlib only (`net/http`, `context`, `encoding/json`)
- **ADR Alignment:** ADR-001 mandates `net/http` standard library usage

### Implementation Pattern

The `FetchProtocols` method should follow the same pattern as `FetchOracles` using the existing `doRequest` helper:

```go
// internal/api/client.go

// FetchProtocols retrieves protocol metadata from DefiLlama /lite/protocols2 endpoint.
func (c *Client) FetchProtocols(ctx context.Context) ([]Protocol, error) {
    var protocols []Protocol
    if err := c.doRequest(ctx, ProtocolsEndpoint, &protocols); err != nil {
        return nil, fmt.Errorf("fetch protocols: %w", err)
    }
    return protocols, nil
}
```

### Response Structure Notes

The `/lite/protocols2?b=2` endpoint returns an array of protocol objects:

```json
[
  {
    "id": "marinade-finance",
    "name": "Marinade Finance",
    "slug": "marinade-finance",
    "category": "Liquid Staking",
    "tvl": 1234567890.50,
    "chains": ["Solana"],
    "oracles": ["Switchboard"],
    "url": "https://marinade.finance"
  },
  {
    "id": "protocol-without-oracle",
    "name": "Simple Protocol",
    "slug": "simple-protocol",
    "category": "Lending"
  }
]
```

**Important Field Notes:**
- `oracles` (array) and `oracle` (string) are both optional - protocols may have one, both, or neither
- `TVL`, `Chains`, `URL` are optional and may be missing or null
- The response is a JSON array at the top level, not wrapped in an object

### Test Fixture

Create `testdata/protocol_response.json` with realistic sample data:

```json
[
  {
    "id": "marinade-finance",
    "name": "Marinade Finance",
    "slug": "marinade-finance",
    "category": "Liquid Staking",
    "tvl": 500000000,
    "chains": ["Solana"],
    "oracles": ["Switchboard"],
    "url": "https://marinade.finance"
  },
  {
    "id": "solend",
    "name": "Solend",
    "slug": "solend",
    "category": "Lending",
    "tvl": 150000000,
    "chains": ["Solana"],
    "oracles": ["Switchboard", "Pyth"],
    "oracle": "Switchboard",
    "url": "https://solend.fi"
  },
  {
    "id": "minimal-protocol",
    "name": "Minimal Protocol",
    "slug": "minimal-protocol",
    "category": "DEX"
  }
]
```

### Project Structure Notes

- Existing: `internal/api/responses.go` - add Protocol struct
- Existing: `internal/api/endpoints.go` - add ProtocolsEndpoint constant
- Existing: `internal/api/client.go` - add FetchProtocols method
- New file: `testdata/protocol_response.json` - test fixture
- New file: `internal/api/protocols_test.go` - FetchProtocols tests
- Existing: `internal/api/responses_test.go` - add Protocol struct tests

### Learnings from Previous Story

**From Story 2-2-implement-oracle-endpoint-fetcher (Status: done)**

- **Infrastructure Ready:** `internal/api/client.go` with `NewClient()`, `doRequest()` helper, timeout, and User-Agent - all reusable
- **doRequest Helper Available:** Use `c.doRequest(ctx, url, &target)` - handles User-Agent, timeout, JSON decode, error wrapping
- **Response Files Pattern:** Response structs go in `internal/api/responses.go`
- **Endpoint Constants Pattern:** URL constants go in `internal/api/endpoints.go`
- **Test Patterns Established:**
  - Use `httptest.NewServer` for mock server tests
  - Capture User-Agent header in handler
  - Test fixtures in `testdata/` directory
  - Cover: success, header verification, status errors (500, 404), malformed JSON, context cancellation
- **Non-2xx Handling:** `doRequest` already returns error for non-2xx status codes with wrapped context
- **Review Outcome:** Approved with no action items - follow same patterns

[Source: docs/sprint-artifacts/2-2-implement-oracle-endpoint-fetcher.md#Dev-Agent-Record]

### References

- [Source: docs/sprint-artifacts/tech-spec-epic-2.md#AC-2.3] - Protocol endpoint acceptance criteria
- [Source: docs/sprint-artifacts/tech-spec-epic-2.md#Data-Models-and-Contracts] - Protocol struct definition
- [Source: docs/epics/epic-2-api-integration.md#story-23] - Original story definition
- [Source: docs/prd.md#FR2] - System fetches protocol metadata from `/lite/protocols2` endpoint
- [Source: docs/architecture/data-architecture.md] - Protocol struct definition
- [Source: docs/architecture/testing-strategy.md#Test-Fixtures] - testdata directory for fixtures

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/2-3-implement-protocol-endpoint-fetcher.context.xml

### Agent Model Used

- gpt-4o (2025-11-30)

### Debug Log References

- docs/sprint-artifacts/validation-report-story-2-3-2025-11-30T08-44-24Z.md

### Completion Notes List

- Drafted story from epic-2 API integration and tech-spec AC-2.3; aligned ACs, tasks, and tests.
- Captured learnings from Story 2.2 (helper reuse, patterns, review outcome: no action items).
- Validation pass with minor issue (empty Dev Agent Record fields now populated).

### File List

- docs/sprint-artifacts/2-3-implement-protocol-endpoint-fetcher.md
- docs/sprint-artifacts/validation-report-story-2-3-2025-11-30T08-44-24Z.md

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-2-api-integration.md and tech-spec-epic-2.md |
