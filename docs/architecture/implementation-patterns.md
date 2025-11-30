# Implementation Patterns

> **Spec Reference:** [15-go-specific-patterns-idioms.md](../docs-from-user/seed-doc/15-go-specific-patterns-idioms.md)

These patterns ensure consistent implementation across all AI agents:

## Dependency Injection

Use constructor functions, not global state:

```go
// Good: explicit dependencies
func NewAggregator(client *api.Client, config *config.Config) *Aggregator {
    return &Aggregator{
        client: client,
        config: config,
    }
}

// main.go wires everything
func main() {
    cfg := config.Load()
    client := api.NewClient(cfg.API)
    agg := aggregator.NewAggregator(client, cfg)
    writer := storage.NewWriter(cfg.Output)
    // ... run
}
```

## Context Propagation

All I/O functions accept `context.Context` as first parameter:

```go
func (c *Client) FetchOracles(ctx context.Context) (*OracleResponse, error) {
    req, err := http.NewRequestWithContext(ctx, "GET", c.oraclesURL, nil)
    // ...
}
```

## Parallel Fetching

Use `errgroup` for concurrent API calls:

```go
import "golang.org/x/sync/errgroup"

func (a *Aggregator) Fetch(ctx context.Context) (*Data, error) {
    var oracleResp *OracleResponse
    var protocolResp *ProtocolResponse

    g, ctx := errgroup.WithContext(ctx)

    g.Go(func() error {
        var err error
        oracleResp, err = a.client.FetchOracles(ctx)
        return err
    })

    g.Go(func() error {
        var err error
        protocolResp, err = a.client.FetchProtocols(ctx)
        return err
    })

    if err := g.Wait(); err != nil {
        return nil, err
    }

    return &Data{Oracles: oracleResp, Protocols: protocolResp}, nil
}
```

## Atomic File Writes

Never write directly to target file:

```go
func (w *Writer) WriteJSON(path string, data any) error {
    tmpPath := path + ".tmp"
    f, err := os.Create(tmpPath)
    // ... write and close ...
    return os.Rename(tmpPath, path)  // atomic
}
```
