# 16. Operational Concerns

## 16.1 Graceful Shutdown

```go
// cmd/extractor/main.go (signal handling portion)

func main() {
    // ... setup code ...

    // Create context that cancels on interrupt
    ctx, cancel := context.WithCancel(context.Background())

    // Handle shutdown signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        sig := <-sigChan
        logger.Info("received shutdown signal", slog.String("signal", sig.String()))
        cancel()
    }()

    // Run extractor
    if err := extractor.Run(ctx); err != nil {
        if errors.Is(err, context.Canceled) {
            logger.Info("extraction cancelled by signal")
        } else {
            logger.Error("extraction failed", slog.String("error", err.Error()))
            os.Exit(1)
        }
    }

    // Cleanup
    logger.Info("shutdown complete")
}
```

## 16.2 Prometheus Metrics

```go
// internal/monitoring/metrics.go

package monitoring

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    ExtractionDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "switchboard_extraction_duration_seconds",
            Help:    "Duration of extraction cycles",
            Buckets: []float64{1, 5, 10, 30, 60, 120},
        },
        []string{"status"},
    )

    ProtocolCount = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "switchboard_protocol_count",
            Help: "Number of protocols using Switchboard oracle",
        },
    )

    TotalValueSecured = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "switchboard_total_value_secured",
            Help: "Total value secured by Switchboard oracle in USD",
        },
    )

    APIRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "switchboard_api_requests_total",
            Help: "Total API requests by endpoint and status",
        },
        []string{"endpoint", "status"},
    )

    APIRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "switchboard_api_request_duration_seconds",
            Help:    "API request duration by endpoint",
            Buckets: []float64{0.1, 0.5, 1, 2, 5, 10},
        },
        []string{"endpoint"},
    )

    LastExtractionTimestamp = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "switchboard_last_extraction_timestamp",
            Help: "Unix timestamp of last successful extraction",
        },
    )

    ExtractionErrors = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "switchboard_extraction_errors_total",
            Help: "Total extraction errors by type",
        },
        []string{"error_type"},
    )
)

// RecordExtraction updates metrics after an extraction cycle
func RecordExtraction(duration float64, protocolCount int, tvs float64, success bool) {
    status := "success"
    if !success {
        status = "failure"
    }
    ExtractionDuration.WithLabelValues(status).Observe(duration)

    if success {
        ProtocolCount.Set(float64(protocolCount))
        TotalValueSecured.Set(tvs)
        LastExtractionTimestamp.SetToCurrentTime()
    }
}
```

## 16.3 Health Check Endpoint

```go
// internal/monitoring/health.go

package monitoring

import (
    "encoding/json"
    "net/http"
    "sync"
    "time"
)

type HealthChecker struct {
    mu              sync.RWMutex
    lastSuccess     time.Time
    lastError       error
    consecutiveFails int
}

type HealthStatus struct {
    Status           string    `json:"status"`
    LastSuccess      time.Time `json:"last_success,omitempty"`
    LastError        string    `json:"last_error,omitempty"`
    ConsecutiveFails int       `json:"consecutive_fails"`
    Uptime           string    `json:"uptime"`
}

func NewHealthChecker() *HealthChecker {
    return &HealthChecker{}
}

func (h *HealthChecker) RecordSuccess() {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.lastSuccess = time.Now()
    h.lastError = nil
    h.consecutiveFails = 0
}

func (h *HealthChecker) RecordFailure(err error) {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.lastError = err
    h.consecutiveFails++
}

func (h *HealthChecker) Handler(startTime time.Time) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        h.mu.RLock()
        defer h.mu.RUnlock()

        status := HealthStatus{
            Status:           "healthy",
            LastSuccess:      h.lastSuccess,
            ConsecutiveFails: h.consecutiveFails,
            Uptime:           time.Since(startTime).String(),
        }

        // Unhealthy if more than 3 consecutive failures
        // or no success in last 30 minutes
        if h.consecutiveFails > 3 {
            status.Status = "unhealthy"
        }
        if !h.lastSuccess.IsZero() && time.Since(h.lastSuccess) > 30*time.Minute {
            status.Status = "unhealthy"
        }

        if h.lastError != nil {
            status.LastError = h.lastError.Error()
        }

        w.Header().Set("Content-Type", "application/json")
        if status.Status != "healthy" {
            w.WriteHeader(http.StatusServiceUnavailable)
        }
        json.NewEncoder(w).Encode(status)
    }
}
```

## 16.4 API Schema Change Detection

```go
// internal/api/validation.go

package api

import (
    "fmt"

    "github.com/yourorg/switchboard-extractor/internal/models"
)

// ValidateOracleResponse checks if the API response has expected structure
func ValidateOracleResponse(resp *models.OracleAPIResponse) error {
    if resp.Oracles == nil {
        return fmt.Errorf("missing 'oracles' field in response")
    }
    if resp.Chart == nil {
        return fmt.Errorf("missing 'chart' field in response")
    }
    if resp.ChainsByOracle == nil {
        return fmt.Errorf("missing 'chainsByOracle' field in response")
    }

    // Check for empty data (possible API issue)
    if len(resp.Oracles) == 0 {
        return fmt.Errorf("'oracles' field is empty - possible API issue")
    }
    if len(resp.Chart) == 0 {
        return fmt.Errorf("'chart' field is empty - possible API issue")
    }

    return nil
}

// ValidateProtocolResponse checks protocol response structure
func ValidateProtocolResponse(resp *models.ProtocolAPIResponse) error {
    if resp.Protocols == nil {
        return fmt.Errorf("missing 'protocols' field in response")
    }

    // Validate at least some protocols have expected fields
    for i, p := range resp.Protocols {
        if i > 10 {
            break // Sample check only
        }
        if p.Name == "" {
            return fmt.Errorf("protocol at index %d has empty name", i)
        }
        if p.Slug == "" {
            return fmt.Errorf("protocol %s has empty slug", p.Name)
        }
    }

    return nil
}
```

---
