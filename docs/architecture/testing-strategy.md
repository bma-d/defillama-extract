# Testing Strategy

> **Spec Reference:** [11-testing-strategy-complete-implementation.md](../docs-from-user/seed-doc/11-testing-strategy-complete-implementation.md)

## Test Organization

| Test Type | Location | Purpose |
|-----------|----------|---------|
| Unit Tests | `*_test.go` co-located | Test individual functions/methods |
| Table-Driven Tests | All test files | Multiple inputs/outputs per test |
| Integration Tests | `aggregator_test.go` | Test component interactions |
| Mock Server Tests | `internal/api/client_test.go` | Test HTTP client with `httptest` |

## Test Fixtures

Test data lives in `testdata/` directory:
- `oracle_response.json` - Sample DefiLlama `/oracles` response
- `protocol_response.json` - Sample DefiLlama `/lite/protocols2` response
- `config.yaml` - Test configuration file

> **Sample Responses:** See [13-api-response-examples.md](../docs-from-user/seed-doc/13-api-response-examples.md) for example API responses

## Coverage Requirements

- Aggregation logic: High coverage (business-critical)
- Error handling paths: Must be tested
- HTTP retry logic: Test with mock server
- Configuration loading: Test all layers (defaults, YAML, env)
