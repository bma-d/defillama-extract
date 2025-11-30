# 11. Testing Strategy (Complete Implementation)

## 11.1 Table-Driven Tests for Metrics

```go
// internal/aggregator/metrics_test.go

package aggregator

import (
    "testing"

    "github.com/yourorg/switchboard-extractor/internal/models"
)

func TestCalculatePercentageChange(t *testing.T) {
    tests := []struct {
        name     string
        old      float64
        new      float64
        expected float64
    }{
        {
            name:     "positive change",
            old:      100,
            new:      110,
            expected: 10.0,
        },
        {
            name:     "negative change",
            old:      100,
            new:      90,
            expected: -10.0,
        },
        {
            name:     "no change",
            old:      100,
            new:      100,
            expected: 0.0,
        },
        {
            name:     "zero old value",
            old:      0,
            new:      100,
            expected: 0.0, // Avoid division by zero
        },
        {
            name:     "decimal result",
            old:      100,
            new:      106.52,
            expected: 6.52,
        },
        {
            name:     "large change",
            old:      1000000,
            new:      2500000,
            expected: 150.0,
        },
        {
            name:     "small values",
            old:      0.001,
            new:      0.002,
            expected: 100.0,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := CalculatePercentageChange(tt.old, tt.new)
            if result != tt.expected {
                t.Errorf("CalculatePercentageChange(%f, %f) = %f, want %f",
                    tt.old, tt.new, result, tt.expected)
            }
        })
    }
}

func TestFindSnapshotAtTime(t *testing.T) {
    snapshots := []models.Snapshot{
        {Timestamp: 1732867200, TVS: 100000000}, // Most recent
        {Timestamp: 1732780800, TVS: 95000000},  // 1 day ago
        {Timestamp: 1732694400, TVS: 90000000},  // 2 days ago
        {Timestamp: 1732262400, TVS: 85000000},  // 7 days ago
    }

    tests := []struct {
        name        string
        targetTime  int64
        tolerance   int64
        expectFound bool
        expectedTVS float64
    }{
        {
            name:        "exact match",
            targetTime:  1732867200,
            tolerance:   3600,
            expectFound: true,
            expectedTVS: 100000000,
        },
        {
            name:        "within tolerance",
            targetTime:  1732867200 + 1800, // 30 min off
            tolerance:   3600,
            expectFound: true,
            expectedTVS: 100000000,
        },
        {
            name:        "outside tolerance",
            targetTime:  1732867200 + 7200, // 2 hours off
            tolerance:   3600,
            expectFound: false,
        },
        {
            name:        "no snapshots match",
            targetTime:  1700000000,
            tolerance:   3600,
            expectFound: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := FindSnapshotAtTime(snapshots, tt.targetTime, tt.tolerance)

            if tt.expectFound {
                if result == nil {
                    t.Errorf("expected to find snapshot, got nil")
                } else if result.TVS != tt.expectedTVS {
                    t.Errorf("expected TVS %f, got %f", tt.expectedTVS, result.TVS)
                }
            } else {
                if result != nil {
                    t.Errorf("expected nil, got snapshot with timestamp %d", result.Timestamp)
                }
            }
        })
    }
}

func TestCalculateChainBreakdown(t *testing.T) {
    oracleTVS := map[string]map[string]float64{
        "Kamino Lend": {"Solana": 145000000},
        "marginfi":    {"Solana": 25000000},
        "Scallop":     {"Sui": 35000000},
        "MovePosition": {"Aptos": 20000000},
    }

    result := CalculateChainBreakdown(oracleTVS)

    // Should be sorted by TVS descending
    if len(result) != 3 {
        t.Errorf("expected 3 chains, got %d", len(result))
    }

    // First should be Solana (highest TVS)
    if result[0].Chain != "Solana" {
        t.Errorf("expected Solana first, got %s", result[0].Chain)
    }
    if result[0].TVS != 170000000 {
        t.Errorf("expected Solana TVS 170000000, got %f", result[0].TVS)
    }
    if result[0].ProtocolCount != 2 {
        t.Errorf("expected Solana protocol count 2, got %d", result[0].ProtocolCount)
    }

    // Verify percentages sum to ~100
    var totalPct float64
    for _, cb := range result {
        totalPct += cb.Percentage
    }
    if totalPct < 99.9 || totalPct > 100.1 {
        t.Errorf("percentages should sum to ~100, got %f", totalPct)
    }
}
```

## 11.2 Mock API Client for Testing

```go
// internal/api/mock_client.go

package api

import (
    "context"
    "encoding/json"
    "os"

    "github.com/yourorg/switchboard-extractor/internal/models"
)

// MockClient implements APIClient for testing
type MockClient struct {
    OracleResponse   *models.OracleAPIResponse
    ProtocolResponse *models.ProtocolAPIResponse
    OracleError      error
    ProtocolError    error
    FetchCount       int
}

// NewMockClient creates a mock client from test fixture files
func NewMockClient(oracleFixture, protocolFixture string) (*MockClient, error) {
    mock := &MockClient{}

    if oracleFixture != "" {
        data, err := os.ReadFile(oracleFixture)
        if err != nil {
            return nil, err
        }
        var resp models.OracleAPIResponse
        if err := json.Unmarshal(data, &resp); err != nil {
            return nil, err
        }
        mock.OracleResponse = &resp
    }

    if protocolFixture != "" {
        data, err := os.ReadFile(protocolFixture)
        if err != nil {
            return nil, err
        }
        var resp models.ProtocolAPIResponse
        if err := json.Unmarshal(data, &resp); err != nil {
            return nil, err
        }
        mock.ProtocolResponse = &resp
    }

    return mock, nil
}

func (m *MockClient) FetchOracles(ctx context.Context) (*models.OracleAPIResponse, error) {
    m.FetchCount++
    if m.OracleError != nil {
        return nil, m.OracleError
    }
    return m.OracleResponse, nil
}

func (m *MockClient) FetchProtocols(ctx context.Context) (*models.ProtocolAPIResponse, error) {
    m.FetchCount++
    if m.ProtocolError != nil {
        return nil, m.ProtocolError
    }
    return m.ProtocolResponse, nil
}

func (m *MockClient) FetchBoth(ctx context.Context) (*models.OracleAPIResponse, *models.ProtocolAPIResponse, error) {
    m.FetchCount += 2
    if m.OracleError != nil {
        return nil, nil, m.OracleError
    }
    if m.ProtocolError != nil {
        return nil, nil, m.ProtocolError
    }
    return m.OracleResponse, m.ProtocolResponse, nil
}
```

## 11.3 Integration Test Example

```go
// internal/aggregator/aggregator_test.go

package aggregator

import (
    "context"
    "log/slog"
    "os"
    "testing"

    "github.com/yourorg/switchboard-extractor/internal/api"
    "github.com/yourorg/switchboard-extractor/internal/models"
)

func TestAggregator_Process(t *testing.T) {
    // Create mock client with test fixtures
    mockClient, err := api.NewMockClient(
        "../../testdata/oracle_response.json",
        "../../testdata/protocol_response.json",
    )
    if err != nil {
        t.Fatalf("failed to create mock client: %v", err)
    }

    logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
    agg := NewAggregator(mockClient, "Switchboard", logger)

    history := []models.Snapshot{
        {Timestamp: 1732780800, TVS: 95000000, ProtocolCount: 20},
        {Timestamp: 1732262400, TVS: 85000000, ProtocolCount: 18},
    }

    ctx := context.Background()
    result, err := agg.Process(ctx, history)
    if err != nil {
        t.Fatalf("Process failed: %v", err)
    }

    // Verify results
    if len(result.Protocols) == 0 {
        t.Error("expected protocols, got none")
    }

    if len(result.ChainBreakdown) == 0 {
        t.Error("expected chain breakdown, got none")
    }

    if result.Metrics.CurrentTVS == 0 {
        t.Error("expected non-zero current TVS")
    }

    // Verify protocols are ranked
    for i, p := range result.Protocols {
        if p.Rank != i+1 {
            t.Errorf("protocol %s has rank %d, expected %d", p.Name, p.Rank, i+1)
        }
    }
}

func TestAggregator_Process_OracleNotFound(t *testing.T) {
    mockClient := &api.MockClient{
        OracleResponse: &models.OracleAPIResponse{
            Oracles: map[string][]string{
                "Chainlink": {"Protocol A"},
            },
        },
        ProtocolResponse: &models.ProtocolAPIResponse{},
    }

    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
    agg := NewAggregator(mockClient, "Switchboard", logger)

    _, err := agg.Process(context.Background(), nil)
    if err == nil {
        t.Error("expected error for missing oracle")
    }
    if !errors.Is(err, models.ErrOracleNotFound) {
        t.Errorf("expected ErrOracleNotFound, got %v", err)
    }
}
```

## 11.4 Test Fixtures

Create `testdata/oracle_response.json`:
```json
{
  "oracles": {
    "Switchboard": ["Kamino Lend", "marginfi Lending", "Scallop Lend"]
  },
  "chart": {
    "1732867200": {
      "Switchboard": {
        "Solana": 170000000,
        "Sui": 35000000
      }
    }
  },
  "oraclesTVS": {
    "Switchboard": {
      "Kamino Lend": {"Solana": 145000000},
      "marginfi Lending": {"Solana": 25000000},
      "Scallop Lend": {"Sui": 35000000}
    }
  },
  "chainsByOracle": {
    "Switchboard": ["Solana", "Sui", "Aptos"]
  }
}
```

Create `testdata/protocol_response.json`:
```json
{
  "protocols": [
    {
      "id": "1",
      "name": "Kamino Lend",
      "slug": "kamino-lend",
      "chain": "Solana",
      "category": "Lending",
      "tvl": 145000000,
      "oracles": ["Switchboard"]
    },
    {
      "id": "2",
      "name": "marginfi Lending",
      "slug": "marginfi-lending",
      "chain": "Solana",
      "category": "Lending",
      "tvl": 25000000,
      "oracles": ["Switchboard", "Pyth"]
    },
    {
      "id": "3",
      "name": "Scallop Lend",
      "slug": "scallop-lend",
      "chain": "Sui",
      "category": "Lending",
      "tvl": 35000000,
      "oracles": ["Switchboard"]
    }
  ],
  "chains": ["Solana", "Sui"]
}
```

## 11.5 Benchmark Tests

```go
// internal/aggregator/metrics_benchmark_test.go

package aggregator

import (
    "testing"

    "github.com/yourorg/switchboard-extractor/internal/models"
)

func BenchmarkCalculateChainBreakdown(b *testing.B) {
    // Create large dataset
    oracleTVS := make(map[string]map[string]float64)
    for i := 0; i < 100; i++ {
        oracleTVS[fmt.Sprintf("Protocol%d", i)] = map[string]float64{
            "Solana": float64(i * 1000000),
            "Sui":    float64(i * 500000),
        }
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        CalculateChainBreakdown(oracleTVS)
    }
}

func BenchmarkFindSnapshotAtTime(b *testing.B) {
    // Create 90 days of snapshots
    snapshots := make([]models.Snapshot, 2160)
    now := time.Now().Unix()
    for i := 0; i < 2160; i++ {
        snapshots[i] = models.Snapshot{
            Timestamp: now - int64(i*3600),
            TVS:       float64(100000000 + i*10000),
        }
    }

    targetTime := now - 7*24*3600 // 7 days ago

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        FindSnapshotAtTime(snapshots, targetTime, 7200)
    }
}
```

---
