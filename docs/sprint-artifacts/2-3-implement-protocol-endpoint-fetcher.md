# Story 2.3: Implement Protocol Endpoint Fetcher

Status: done

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

- [x] Task 1: Define Protocol struct (AC: 2, 3)
  - [x] 1.1: Add `Protocol` struct to `internal/api/responses.go`
  - [x] 1.2: Include all fields with proper JSON tags and `omitempty` for optional fields:
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

- [x] Task 2: Define ProtocolsEndpoint constant (AC: 1)
  - [x] 2.1: Add `ProtocolsEndpoint = "https://api.llama.fi/lite/protocols2?b=2"` to `internal/api/endpoints.go`

- [x] Task 3: Implement FetchProtocols method (AC: 1, 2, 3, 4, 5, 6, 7)
  - [x] 3.1: Add `FetchProtocols(ctx context.Context) ([]Protocol, error)` method to `Client` in `internal/api/client.go`
  - [x] 3.2: Call `c.doRequest(ctx, ProtocolsEndpoint, &protocols)` using existing helper
  - [x] 3.3: Return `(protocols, nil)` on success
  - [x] 3.4: Return `(nil, error)` on failure with wrapped error context: `fmt.Errorf("fetch protocols: %w", err)`

- [x] Task 4: Write unit tests for Protocol struct (AC: 2, 3)
  - [x] 4.1: Create test fixture `testdata/protocol_response.json` with sample response including protocols with all fields and some with missing optional fields
  - [x] 4.2: Add tests to `internal/api/responses_test.go` for Protocol struct JSON unmarshaling
  - [x] 4.3: Test JSON unmarshaling populates all fields correctly
  - [x] 4.4: Test protocols with missing optional fields have zero values

- [x] Task 5: Write integration tests with mock server (AC: 1, 2, 3, 4, 5, 6, 7)
  - [x] 5.1: Create `internal/api/protocols_test.go` for FetchProtocols tests
  - [x] 5.2: Test successful fetch returns populated []Protocol slice
  - [x] 5.3: Test User-Agent header is present in request
  - [x] 5.4: Test HTTP 500 returns wrapped error
  - [x] 5.5: Test HTTP 404 returns wrapped error
  - [x] 5.6: Test malformed JSON returns decode error
  - [x] 5.7: Test context cancellation aborts request
  - [x] 5.8: Test empty array response `[]` returns empty slice without error

- [x] Task 6: Verification (AC: all)
  - [x] 6.1: Run `go build ./...` and verify success
  - [x] 6.2: Run `go test ./internal/api/...` and verify all pass
  - [x] 6.3: Run `make lint` and verify no errors

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
- Ran `go build ./...`, `go test ./internal/api/...`, and `make lint` locally; all passed.

### Completion Notes List

- Implemented Protocol struct, endpoint constant, and FetchProtocols using existing doRequest helper with error wrapping.
- Added fixture-driven unit and integration tests covering success, optional fields, user agent, status errors, malformed JSON, context cancellation, and empty responses.
- Confirmed verification commands succeed (build, unit/integration tests, lint).

### File List

- docs/sprint-artifacts/2-3-implement-protocol-endpoint-fetcher.md
- docs/sprint-artifacts/validation-report-story-2-3-2025-11-30T08-44-24Z.md
- docs/sprint-artifacts/sprint-status.yaml
- internal/api/responses.go
- internal/api/endpoints.go
- internal/api/client.go
- internal/api/responses_test.go
- internal/api/protocols_test.go
- testdata/protocol_response.json

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-2-api-integration.md and tech-spec-epic-2.md |
| 2025-11-30 | Dev Agent (Amelia) | Implemented protocol endpoint fetcher, added tests/fixtures, updated status to review |
| 2025-11-30 | Dev Agent (Amelia) | Senior Developer Review (AI) completed, outcome Approved, status moved to done |

## Senior Developer Review (AI)

- Reviewer: BMad
- Date: 2025-11-30
- Outcome: Approve (all ACs implemented; no findings)

### Summary
- Implementation matches tech spec for `/lite/protocols2` fetch; tests cover success, errors, optional fields, cancellation. All reviewed files clean; no action items.

### Key Findings (by severity)
- None.

### Acceptance Criteria Coverage

| AC# | Description | Status | Evidence |
|-----|-------------|--------|----------|
| 1 | GET to https://api.llama.fi/lite/protocols2?b=2 with User-Agent | Implemented | internal/api/endpoints.go:4-6; internal/api/client.go:55-74; internal/api/protocols_test.go:56-74 |
| 2 | Successful response returns []Protocol with fields populated | Implemented | internal/api/responses.go:11-21; internal/api/client.go:92-98; internal/api/protocols_test.go:27-54 |
| 3 | Missing optional fields decode to zero values | Implemented | internal/api/responses.go:17-21; internal/api/responses_test.go:79-101 |
| 4 | HTTP errors return wrapped error | Implemented | internal/api/client.go:70-96; internal/api/protocols_test.go:77-96 |
| 5 | Malformed JSON returns decode error | Implemented | internal/api/client.go:74-76; internal/api/protocols_test.go:99-115 |
| 6 | Context cancellation aborts request | Implemented | internal/api/client.go:55-67; internal/api/protocols_test.go:118-139 |
| 7 | Empty array returns empty slice without error | Implemented | internal/api/protocols_test.go:142-158 |

Summary: 7 of 7 acceptance criteria fully implemented.

### Task Completion Validation

| Task | Marked As | Verified As | Evidence |
|------|-----------|-------------|----------|
| Task 1: Define Protocol struct | [x] | Verified complete | internal/api/responses.go:11-21; internal/api/responses_test.go:52-76 |
| Task 2: Define ProtocolsEndpoint constant | [x] | Verified complete | internal/api/endpoints.go:3-6 |
| Task 3: Implement FetchProtocols method | [x] | Verified complete | internal/api/client.go:92-98; internal/api/protocols_test.go:27-54 |
| Task 4: Unit tests for Protocol struct | [x] | Verified complete | internal/api/responses_test.go:52-101; testdata/protocol_response.json |
| Task 5: Integration tests with mock server | [x] | Verified complete | internal/api/protocols_test.go:27-158 |
| Task 6: Verification commands (build/tests/lint) | [x] | Verified complete | go build ./...; go test ./... on 2025-11-30 (lint not rerun) |

### Test Coverage and Gaps
- go test ./... and go build ./... pass (2025-11-30); protocols unit/integration tests cover success, header, status errors, malformed JSON, cancellation, empty response.
- golangci-lint not rerun during review (developer reported prior pass).

### Architectural Alignment
- Uses stdlib net/http + context with omitempty fields per ADR-001/ADR-003; Protocol model matches data-architecture.md; User-Agent constant aligns with tech-spec-epic-2.

### Security Notes
- Read-only public API calls; no secrets; no new security risks observed.

### Best-Practices and References
- ADR-001 (stdlib HTTP), data-architecture.md (Protocol struct), tech-spec-epic-2 (AC-2.3) remain authoritative; implementation conforms.

### Action Items

**Code Changes Required:**
- None.

**Advisory Notes:**
- None.
