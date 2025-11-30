# Performance Considerations

> **Spec Reference:** [9-error-handling-resilience.md](../docs-from-user/seed-doc/9-error-handling-resilience.md), [5-core-components.md](../docs-from-user/seed-doc/5-core-components.md)

| NFR | Implementation |
|-----|----------------|
| NFR1: 30s request timeout | `http.Client.Timeout = 30 * time.Second` |
| NFR2: 2min extraction cycle | Parallel API fetches, efficient aggregation |
| NFR3: Parallel fetching | `errgroup` for concurrent API calls |
| NFR4: Atomic writes | Temp file + rename pattern |
| NFR5: Stable memory | No growing buffers, process and discard |

## Retry Configuration

> **Spec Reference:** [9-error-handling-resilience.md](../docs-from-user/seed-doc/9-error-handling-resilience.md)

```
Attempt 1: immediate
Attempt 2: 1s + jitter (0-500ms)
Attempt 3: 2s + jitter
Attempt 4: 4s + jitter
Attempt 5: fail
```
