# 9. Error Handling & Resilience

## 9.1 Error Wrapping Patterns

```go
// Example of proper error wrapping throughout the codebase

// In API client
func (c *Client) FetchOracles(ctx context.Context) (*models.OracleAPIResponse, error) {
    var response models.OracleAPIResponse
    if err := c.fetchWithRetry(ctx, c.config.OracleURL, &response); err != nil {
        // Wrap with context about what operation failed
        return nil, fmt.Errorf("fetching oracles from %s: %w", c.config.OracleURL, err)
    }
    return &response, nil
}

// In aggregator
func (a *Aggregator) Process(ctx context.Context, history []models.Snapshot) (*Result, error) {
    oracleResp, protocolResp, err := a.client.FetchBoth(ctx)
    if err != nil {
        // Wrap to add aggregator context
        return nil, fmt.Errorf("aggregator.Process: %w", err)
    }
    // ...
}

// In main extraction loop
func (e *Extractor) Run(ctx context.Context) error {
    result, err := e.aggregator.Process(ctx, history)
    if err != nil {
        // Check for specific error types
        if errors.Is(err, models.ErrNoNewData) {
            e.logger.Info("no new data, skipping extraction")
            return nil
        }
        if errors.Is(err, models.ErrOracleNotFound) {
            e.logger.Error("oracle not found", slog.String("oracle", e.oracleName))
            return err // Don't retry this
        }
        // Wrap for context
        return fmt.Errorf("extraction failed: %w", err)
    }
    // ...
}

// Checking error types
func handleError(err error) {
    var apiErr *models.APIError
    if errors.As(err, &apiErr) {
        if apiErr.IsRetryable() {
            // Schedule retry
        } else {
            // Log and alert
        }
    }

    var validationErr *models.ValidationError
    if errors.As(err, &validationErr) {
        // Log validation failure with field details
    }
}
```

## 9.2 Retry Configuration

```go
type RetryConfig struct {
    MaxAttempts   int           // Maximum retry attempts (default: 3)
    BaseDelay     time.Duration // Initial delay (default: 1s)
    MaxDelay      time.Duration // Maximum delay (default: 30s)
    Multiplier    float64       // Backoff multiplier (default: 2.0)
    RetryableHTTP []int         // HTTP codes to retry (default: 429, 500, 502, 503, 504)
}
```

## 9.3 Graceful Degradation Table

| Failure | Degradation Strategy |
|---------|---------------------|
| Oracle API fails | Use cached data, skip update |
| Protocol API fails | Use cached data, skip update |
| Both APIs fail | Keep existing files, retry next cycle |
| State file corrupt | Delete and restart |
| Output write fails | Keep previous output, log error |

---
