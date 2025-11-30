# 15. Go-Specific Patterns & Idioms

## 15.1 Structured Logging with slog (Go 1.23+)

```go
// internal/logging/logger.go

package logging

import (
    "log/slog"
    "os"
)

// NewLogger creates a structured logger based on configuration
func NewLogger(level, format, output string) *slog.Logger {
    var handler slog.Handler

    opts := &slog.HandlerOptions{
        Level: parseLevel(level),
        AddSource: level == "debug",
    }

    var w *os.File
    switch output {
    case "stderr":
        w = os.Stderr
    default:
        w = os.Stdout
    }

    switch format {
    case "json":
        handler = slog.NewJSONHandler(w, opts)
    default:
        handler = slog.NewTextHandler(w, opts)
    }

    return slog.New(handler)
}

func parseLevel(level string) slog.Level {
    switch level {
    case "debug":
        return slog.LevelDebug
    case "warn":
        return slog.LevelWarn
    case "error":
        return slog.LevelError
    default:
        return slog.LevelInfo
    }
}

// Usage examples:
// logger.Info("extraction started", slog.String("oracle", "Switchboard"))
// logger.Debug("fetching API", slog.String("url", url), slog.Int("attempt", 1))
// logger.Error("fetch failed", slog.String("error", err.Error()))
// logger.With(slog.String("component", "aggregator")).Info("processing")
```

## 15.2 Context Propagation

```go
// Always pass context through the entire call chain

func (e *Extractor) Run(ctx context.Context) error {
    // Create child context with timeout for API calls
    apiCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
    defer cancel()

    result, err := e.aggregator.Process(apiCtx, e.history)
    if err != nil {
        return err
    }

    // Check for cancellation before expensive operations
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    return e.writer.WriteAll(ctx, result)
}

// In HTTP client
func (c *Client) doRequest(ctx context.Context, url string, target interface{}) error {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    // ... context cancellation is automatically handled
}
```

## 15.3 Dependency Injection Pattern

```go
// Define interfaces for all dependencies
type APIClient interface {
    FetchOracles(ctx context.Context) (*models.OracleAPIResponse, error)
    FetchProtocols(ctx context.Context) (*models.ProtocolAPIResponse, error)
    FetchBoth(ctx context.Context) (*models.OracleAPIResponse, *models.ProtocolAPIResponse, error)
}

type StateManager interface {
    Load() (*models.State, error)
    Save(state *models.State) error
    ShouldUpdate(state *models.State, latestTimestamp int64) bool
}

type OutputWriter interface {
    WriteAll(output *models.FullOutput) error
}

// Extractor uses interfaces, not concrete types
type Extractor struct {
    client     APIClient      // Interface
    state      StateManager   // Interface
    writer     OutputWriter   // Interface
    aggregator *Aggregator
    logger     *slog.Logger
}

// Constructor accepts interfaces for easy testing
func NewExtractor(
    client APIClient,
    state StateManager,
    writer OutputWriter,
    oracleName string,
    logger *slog.Logger,
) *Extractor {
    return &Extractor{
        client:     client,
        state:      state,
        writer:     writer,
        aggregator: NewAggregator(client, oracleName, logger),
        logger:     logger,
    }
}
```

---
