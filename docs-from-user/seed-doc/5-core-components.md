# 5. Core Components

## 5.1 HTTP Client Component

### 5.1.1 Complete Implementation

```go
// internal/api/client.go

package api

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "log/slog"
    "math/rand"
    "net/http"
    "sync"
    "time"

    "github.com/yourorg/switchboard-extractor/internal/models"
)

// ClientConfig holds HTTP client configuration
type ClientConfig struct {
    OracleURL      string
    ProtocolsURL   string
    Timeout        time.Duration
    MaxRetries     int
    RetryBaseDelay time.Duration
    RetryMaxDelay  time.Duration
    UserAgent      string
}

// DefaultClientConfig returns sensible defaults
func DefaultClientConfig() ClientConfig {
    return ClientConfig{
        OracleURL:      "https://api.llama.fi/oracles",
        ProtocolsURL:   "https://api.llama.fi/lite/protocols2?b=2",
        Timeout:        30 * time.Second,
        MaxRetries:     3,
        RetryBaseDelay: 1 * time.Second,
        RetryMaxDelay:  30 * time.Second,
        UserAgent:      "SwitchboardOracleExtractor/1.0 (Go)",
    }
}

// Client implements the APIClient interface
type Client struct {
    httpClient *http.Client
    config     ClientConfig
    logger     *slog.Logger
}

// NewClient creates a new API client
func NewClient(config ClientConfig, logger *slog.Logger) *Client {
    return &Client{
        httpClient: &http.Client{
            Timeout: config.Timeout,
        },
        config: config,
        logger: logger,
    }
}

// FetchOracles retrieves oracle data from /oracles endpoint
func (c *Client) FetchOracles(ctx context.Context) (*models.OracleAPIResponse, error) {
    var response models.OracleAPIResponse
    if err := c.fetchWithRetry(ctx, c.config.OracleURL, &response); err != nil {
        return nil, fmt.Errorf("fetching oracles: %w", err)
    }
    return &response, nil
}

// FetchProtocols retrieves protocol metadata from /lite/protocols2
func (c *Client) FetchProtocols(ctx context.Context) (*models.ProtocolAPIResponse, error) {
    var response models.ProtocolAPIResponse
    if err := c.fetchWithRetry(ctx, c.config.ProtocolsURL, &response); err != nil {
        return nil, fmt.Errorf("fetching protocols: %w", err)
    }
    return &response, nil
}

// FetchBoth fetches both endpoints in parallel
func (c *Client) FetchBoth(ctx context.Context) (*models.OracleAPIResponse, *models.ProtocolAPIResponse, error) {
    var (
        oracleResp   *models.OracleAPIResponse
        protocolResp *models.ProtocolAPIResponse
        oracleErr    error
        protocolErr  error
        wg           sync.WaitGroup
    )

    wg.Add(2)

    go func() {
        defer wg.Done()
        oracleResp, oracleErr = c.FetchOracles(ctx)
    }()

    go func() {
        defer wg.Done()
        protocolResp, protocolErr = c.FetchProtocols(ctx)
    }()

    wg.Wait()

    // Return first error encountered
    if oracleErr != nil {
        return nil, nil, fmt.Errorf("oracle fetch failed: %w", oracleErr)
    }
    if protocolErr != nil {
        return nil, nil, fmt.Errorf("protocol fetch failed: %w", protocolErr)
    }

    return oracleResp, protocolResp, nil
}

// fetchWithRetry performs HTTP GET with retry logic
func (c *Client) fetchWithRetry(ctx context.Context, url string, target interface{}) error {
    var lastErr error

    for attempt := 0; attempt < c.config.MaxRetries; attempt++ {
        if attempt > 0 {
            delay := c.calculateBackoff(attempt)
            c.logger.Warn("retrying request",
                slog.String("url", url),
                slog.Int("attempt", attempt+1),
                slog.Duration("delay", delay),
            )

            select {
            case <-ctx.Done():
                return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
            case <-time.After(delay):
            }
        }

        err := c.doRequest(ctx, url, target)
        if err == nil {
            return nil
        }

        lastErr = err

        // Check if error is retryable
        var apiErr *models.APIError
        if errors.As(err, &apiErr) && !apiErr.IsRetryable() {
            return err // Don't retry non-retryable errors
        }

        c.logger.Error("request failed",
            slog.String("url", url),
            slog.Int("attempt", attempt+1),
            slog.String("error", err.Error()),
        )
    }

    return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// doRequest performs a single HTTP GET request
func (c *Client) doRequest(ctx context.Context, url string, target interface{}) error {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return fmt.Errorf("creating request: %w", err)
    }

    req.Header.Set("User-Agent", c.config.UserAgent)
    req.Header.Set("Accept", "application/json")

    start := time.Now()
    resp, err := c.httpClient.Do(req)
    duration := time.Since(start)

    if err != nil {
        return &models.APIError{
            Endpoint:   url,
            StatusCode: 0,
            Message:    "request failed",
            Err:        err,
        }
    }
    defer resp.Body.Close()

    c.logger.Debug("request completed",
        slog.String("url", url),
        slog.Int("status", resp.StatusCode),
        slog.Duration("duration", duration),
    )

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
        return &models.APIError{
            Endpoint:   url,
            StatusCode: resp.StatusCode,
            Message:    string(body),
        }
    }

    if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
        return fmt.Errorf("decoding response: %w", err)
    }

    return nil
}

// calculateBackoff returns the delay for exponential backoff with jitter
func (c *Client) calculateBackoff(attempt int) time.Duration {
    // Exponential backoff: baseDelay * 2^attempt
    delay := c.config.RetryBaseDelay * time.Duration(1<<attempt)

    // Add jitter (±10%)
    jitter := time.Duration(float64(delay) * (rand.Float64()*0.2 - 0.1))
    delay += jitter

    // Cap at max delay
    if delay > c.config.RetryMaxDelay {
        delay = c.config.RetryMaxDelay
    }

    return delay
}
```

## 5.2 Aggregator Component Interface

```go
// internal/aggregator/aggregator.go

package aggregator

import (
    "context"
    "fmt"
    "log/slog"
    "sort"
    "time"

    "github.com/yourorg/switchboard-extractor/internal/api"
    "github.com/yourorg/switchboard-extractor/internal/models"
)

// Aggregator orchestrates the data processing pipeline
type Aggregator struct {
    client     *api.Client
    oracleName string
    logger     *slog.Logger
}

// NewAggregator creates a new aggregator instance
func NewAggregator(client *api.Client, oracleName string, logger *slog.Logger) *Aggregator {
    return &Aggregator{
        client:     client,
        oracleName: oracleName,
        logger:     logger,
    }
}

// Result contains the aggregated data
type Result struct {
    Protocols       []models.AggregatedProtocol
    ChainBreakdown  []models.ChainBreakdown
    CategoryBreakdown []models.CategoryBreakdown
    Metrics         models.Metrics
    Snapshot        models.Snapshot
    LatestTimestamp int64
}

// Process fetches and aggregates all oracle data
func (a *Aggregator) Process(ctx context.Context, history []models.Snapshot) (*Result, error) {
    // Fetch both APIs in parallel
    oracleResp, protocolResp, err := a.client.FetchBoth(ctx)
    if err != nil {
        return nil, fmt.Errorf("fetching API data: %w", err)
    }

    // Extract latest timestamp
    latestTimestamp, err := GetLatestTimestamp(oracleResp.Chart)
    if err != nil {
        return nil, fmt.Errorf("extracting timestamp: %w", err)
    }

    // Validate oracle exists
    if _, exists := oracleResp.Oracles[a.oracleName]; !exists {
        return nil, fmt.Errorf("%w: %s", models.ErrOracleNotFound, a.oracleName)
    }

    // Filter protocols
    protocols := FilterProtocolsByOracle(protocolResp.Protocols, a.oracleName)
    a.logger.Info("filtered protocols",
        slog.Int("count", len(protocols)),
        slog.String("oracle", a.oracleName),
    )

    // Get TVS data for this oracle
    oracleTVS := oracleResp.OraclesTVS[a.oracleName]

    // Aggregate protocols with TVS
    aggregated := a.aggregateProtocols(protocols, oracleTVS)

    // Calculate breakdowns
    chainBreakdown := CalculateChainBreakdown(oracleTVS)
    categoryBreakdown := CalculateCategoryBreakdown(aggregated)

    // Calculate current snapshot
    currentTVS := oracleResp.Chart[fmt.Sprintf("%d", latestTimestamp)][a.oracleName]
    snapshot := CreateSnapshot(latestTimestamp, currentTVS, len(aggregated))

    // Calculate metrics with historical comparison
    metrics := CalculateMetrics(snapshot, history, aggregated)

    return &Result{
        Protocols:         aggregated,
        ChainBreakdown:    chainBreakdown,
        CategoryBreakdown: categoryBreakdown,
        Metrics:           metrics,
        Snapshot:          snapshot,
        LatestTimestamp:   latestTimestamp,
    }, nil
}

// aggregateProtocols merges protocol metadata with TVS data
func (a *Aggregator) aggregateProtocols(
    protocols []models.Protocol,
    oracleTVS map[string]map[string]float64,
) []models.AggregatedProtocol {
    result := make([]models.AggregatedProtocol, 0, len(protocols))

    for _, p := range protocols {
        tvsByChain := oracleTVS[p.Name]
        totalTVS := sumMapValues(tvsByChain)

        chains := p.Chains
        if len(chains) == 0 && p.Chain != "" {
            chains = []string{p.Chain}
        }

        result = append(result, models.AggregatedProtocol{
            Name:       p.Name,
            Slug:       p.Slug,
            Category:   p.Category,
            TVL:        p.TVL,
            Chains:     chains,
            URL:        fmt.Sprintf("https://defillama.com/protocol/%s", p.Slug),
            TVS:        totalTVS,
            TVSByChain: tvsByChain,
        })
    }

    // Sort by TVL descending and assign ranks
    sort.Slice(result, func(i, j int) bool {
        return result[i].TVL > result[j].TVL
    })

    for i := range result {
        result[i].Rank = i + 1
    }

    return result
}

func sumMapValues(m map[string]float64) float64 {
    var sum float64
    for _, v := range m {
        sum += v
    }
    return sum
}
```

---
