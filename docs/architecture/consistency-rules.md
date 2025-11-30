# Consistency Rules

## Naming Conventions

| Category | Convention | Example |
|----------|------------|---------|
| Packages | lowercase, single word | `api`, `storage`, `models` |
| Files | lowercase, underscores | `client.go`, `client_test.go` |
| Exported types | PascalCase | `OracleAPIResponse`, `Protocol` |
| Unexported types | camelCase | `httpClient`, `retryConfig` |
| Constants | PascalCase (exported) | `DefaultTimeout`, `MaxRetries` |
| Interfaces | PascalCase, `-er` suffix | `Reader`, `Writer`, `Aggregator` |
| Test functions | `Test` prefix + PascalCase | `TestCalculateChange` |
| JSON fields | snake_case | `"total_value_secured"` |

## Code Organization

| Rule | Pattern |
|------|---------|
| One primary type per file | `protocol.go` contains `Protocol` struct |
| Tests co-located | `client.go` → `client_test.go` same directory |
| Interfaces near usage | Define where used, not where implemented |

## Error Handling

> **Spec Reference:** [9-error-handling-resilience.md](../docs-from-user/seed-doc/9-error-handling-resilience.md)

**Sentinel errors for expected conditions:**
```go
var (
    ErrNoNewData       = errors.New("no new data available")
    ErrOracleNotFound  = errors.New("oracle not found in response")
    ErrInvalidResponse = errors.New("invalid API response")
)
```

**Custom error types for API errors:**
```go
type APIError struct {
    Endpoint   string
    StatusCode int
    Message    string
    Err        error
}

func (e *APIError) Error() string {
    return fmt.Sprintf("API error for %s (status %d): %s",
        e.Endpoint, e.StatusCode, e.Message)
}

func (e *APIError) Unwrap() error { return e.Err }

func (e *APIError) IsRetryable() bool {
    return e.StatusCode == 429 || e.StatusCode >= 500
}
```

**Error wrapping with context:**
```go
result, err := doSomething()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}
```

## Logging Strategy

**Use `slog` with structured fields:**
```go
slog.Info("extraction complete",
    "protocols", count,
    "tvs", totalTVS,
    "duration_ms", elapsed.Milliseconds(),
)
```

**Log levels:**
| Level | Use For |
|-------|---------|
| Debug | Detailed tracing, request/response bodies |
| Info | Normal operations, cycle start/end, metrics |
| Warn | Recoverable issues, retries, degraded state |
| Error | Failures requiring attention |
