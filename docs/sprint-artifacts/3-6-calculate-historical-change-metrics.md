# Story 3.6: Calculate Historical Change Metrics

Status: ready-for-dev

## Story

As a **developer**,
I want **24h, 7d, and 30d TVS change percentages calculated**,
so that **I can show growth trends over time**.

## Acceptance Criteria

Source: Epic 3.6 / PRD FR19, FR20, FR21, FR22

1. **Given** current TVS and historical snapshots **When** `CalculateChangeMetrics(currentTVS float64, history []Snapshot)` is called **Then** a `ChangeMetrics` struct is returned with `Change24h`, `Change7d`, `Change30d` fields

2. **Given** current TVS = $1.1B and TVS 24h ago = $1.0B **When** calculating 24h change **Then** `Change24h` = 10.0 (representing 10% increase)

3. **Given** current TVS = $900M and TVS 7d ago = $1.0B **When** calculating 7d change **Then** `Change7d` = -10.0 (representing 10% decrease)

4. **Given** no historical data available for a time period **When** calculating that period's change **Then** the change value is `nil` (pointer type) indicating "no data available"

5. **Given** history with protocol counts **When** calculating growth **Then** `ProtocolCountChange7d` and `ProtocolCountChange30d` are calculated (FR22)

6. **Given** historical snapshot at exactly the target time (24h/7d/30d ago) **When** searching for snapshot **Then** that exact snapshot is used; if not exact, use closest snapshot within **2-hour tolerance** per seed aggregation logic [Source: docs-from-user/seed-doc/7-custom-aggregation-logic-go-implementation.md#L20-L56]

7. **Given** historical snapshot within the 2-hour tolerance window of the target time **When** searching for snapshot **Then** the closest snapshot within tolerance is used [Source: docs-from-user/seed-doc/7-custom-aggregation-logic-go-implementation.md#L20-L56]

8. **Given** no historical snapshot within the 2-hour tolerance window **When** searching for snapshot **Then** nil is returned for that time period [Source: docs-from-user/seed-doc/7-custom-aggregation-logic-go-implementation.md#L20-L56]

9. **Given** previous TVS value of 0 **When** calculating percentage change **Then** return 0 (avoid division by zero)

10. **Given** an empty history slice **When** calculating change metrics **Then** all change fields are nil (no panic)

## Tasks / Subtasks

- [ ] Task 1: Define Snapshot struct (AC: 1, 6, 7, 8)
  - [ ] 1.1: Add `Snapshot` struct to `internal/aggregator/models.go` with fields: `Timestamp` (int64), `Date` (string), `TVS` (float64), `TVSByChain` (map[string]float64), `ProtocolCount` (int), `ChainCount` (int)
  - [ ] 1.2: Add JSON struct tags for all fields

- [ ] Task 2: Define ChangeMetrics struct (AC: 1, 4, 5)
  - [ ] 2.1: Add `ChangeMetrics` struct to `internal/aggregator/models.go`
  - [ ] 2.2: Include pointer fields for optional values: `Change24h` (*float64), `Change7d` (*float64), `Change30d` (*float64)
  - [ ] 2.3: Include protocol count change fields: `ProtocolCountChange7d` (*int), `ProtocolCountChange30d` (*int)
  - [ ] 2.4: Add JSON struct tags with `omitempty` for pointer fields

- [ ] Task 3: Implement time constants (AC: 6, 7)
  - [ ] 3.1: Add constants to `internal/aggregator/metrics.go`: `Hours24` (24*60*60), `Days7` (7*24*60*60), `Days30` (30*24*60*60), `SnapshotTolerance` (2*60*60)

- [ ] Task 4: Implement FindSnapshotAtTime helper (AC: 6, 7, 8)
  - [ ] 4.1: Add function `func FindSnapshotAtTime(snapshots []Snapshot, targetTime int64, tolerance int64) *Snapshot`
  - [ ] 4.2: Return snapshot closest to targetTime if within tolerance
  - [ ] 4.3: Return nil if no snapshot found within tolerance
  - [ ] 4.4: Handle empty snapshots slice gracefully

- [ ] Task 5: Implement CalculatePercentageChange helper (AC: 2, 3, 9)
  - [ ] 5.1: Add function `func CalculatePercentageChange(oldValue, newValue float64) float64`
  - [ ] 5.2: Use formula: `((new - old) / old) * 100`
  - [ ] 5.3: Return 0 if oldValue is 0 (avoid division by zero)
  - [ ] 5.4: Round result to 2 decimal places

- [ ] Task 6: Implement CalculateChangeMetrics function (AC: 1, 2, 3, 4, 5, 10)
  - [ ] 6.1: Add function `func CalculateChangeMetrics(currentTVS float64, currentProtocolCount int, history []Snapshot) ChangeMetrics`
  - [ ] 6.2: Get current time using `time.Now().Unix()`
  - [ ] 6.3: Find snapshot for 24h ago using FindSnapshotAtTime
  - [ ] 6.4: Find snapshot for 7d ago using FindSnapshotAtTime
  - [ ] 6.5: Find snapshot for 30d ago using FindSnapshotAtTime
  - [ ] 6.6: Calculate Change24h if 24h snapshot found (else nil)
  - [ ] 6.7: Calculate Change7d if 7d snapshot found (else nil)
  - [ ] 6.8: Calculate Change30d if 30d snapshot found (else nil)
  - [ ] 6.9: Calculate ProtocolCountChange7d if 7d snapshot found (else nil)
  - [ ] 6.10: Calculate ProtocolCountChange30d if 30d snapshot found (else nil)
  - [ ] 6.11: Return ChangeMetrics with all populated (or nil) fields

- [ ] Task 7: Write unit tests (AC: 1-10)
  - [ ] 7.1: Add tests to `internal/aggregator/metrics_test.go`
  - [ ] 7.2: Test: CalculatePercentageChange with positive change (10% increase)
  - [ ] 7.3: Test: CalculatePercentageChange with negative change (10% decrease)
  - [ ] 7.4: Test: CalculatePercentageChange with zero old value returns 0
  - [ ] 7.5: Test: FindSnapshotAtTime returns exact match
  - [ ] 7.6: Test: FindSnapshotAtTime returns closest within tolerance
  - [ ] 7.7: Test: FindSnapshotAtTime returns nil when outside tolerance
  - [ ] 7.8: Test: FindSnapshotAtTime returns nil for empty history
  - [ ] 7.9: Test: CalculateChangeMetrics with full history
  - [ ] 7.10: Test: CalculateChangeMetrics with partial history (only 24h available)
  - [ ] 7.11: Test: CalculateChangeMetrics with empty history returns all nil
  - [ ] 7.12: Test: Protocol count change calculation

- [ ] Task 8: Verification (AC: all)
  - [ ] 8.1: Run `go build ./...` and verify success
  - [ ] 8.2: Run `go test ./internal/aggregator/...` and verify all pass
  - [ ] 8.3: Run `make lint` and verify no errors

## Dev Notes

### Technical Guidance

- **Package Location:** Add to `internal/aggregator/` package (same as existing metrics.go)
- **Input Types:**
  - `currentTVS`: float64 - current total TVS value
  - `currentProtocolCount`: int - current number of protocols
  - `history`: []Snapshot - historical snapshots sorted by timestamp descending (newest first)
- **Output Type:** `ChangeMetrics` struct with pointer fields for optional values
- **Pattern:** Follow established patterns from Stories 3.3/3.4/3.5

### Architecture References

- **Data structures and historical snapshot fields:** Align with canonical models in [Source: docs/architecture/data-architecture.md#output-models]
- **Go implementation and testing idioms:** Follow concurrency, immutability, and table-driven test guidance in [Source: docs/architecture/implementation-patterns.md#patterns]

### Testing Standards

- Follow project testing conventions in `docs/architecture/testing-strategy.md` (Go table-driven tests, arrange/act/assert, must cover success and edge cases) [Source: docs/architecture/testing-strategy.md#test-organization]

### Snapshot Struct

```go
// Snapshot represents a point-in-time TVS measurement for historical tracking.
type Snapshot struct {
    Timestamp     int64              `json:"timestamp"`
    Date          string             `json:"date"`
    TVS           float64            `json:"tvs"`
    TVSByChain    map[string]float64 `json:"tvs_by_chain"`
    ProtocolCount int                `json:"protocol_count"`
    ChainCount    int                `json:"chain_count"`
}
```

### ChangeMetrics Struct

```go
// ChangeMetrics represents TVS and protocol count changes over time periods.
// Pointer types are used for optional values (nil = no data available).
type ChangeMetrics struct {
    Change24h             *float64 `json:"change_24h,omitempty"`
    Change7d              *float64 `json:"change_7d,omitempty"`
    Change30d             *float64 `json:"change_30d,omitempty"`
    ProtocolCountChange7d  *int     `json:"protocol_count_change_7d,omitempty"`
    ProtocolCountChange30d *int     `json:"protocol_count_change_30d,omitempty"`
}
```

### Time Constants

```go
const (
    // Time offsets for historical comparison (in seconds)
    Hours24           = 24 * 60 * 60         // 86400
    Days7             = 7 * 24 * 60 * 60     // 604800
    Days30            = 30 * 24 * 60 * 60    // 2592000
    SnapshotTolerance = 2 * 60 * 60          // 7200 (2 hours)
)
```

### FindSnapshotAtTime Implementation Pattern

```go
// FindSnapshotAtTime finds the snapshot closest to the target time within tolerance.
// Returns nil if no snapshot found within tolerance.
func FindSnapshotAtTime(snapshots []Snapshot, targetTime int64, tolerance int64) *Snapshot {
    if len(snapshots) == 0 {
        return nil
    }

    var closest *Snapshot
    minDiff := int64(math.MaxInt64)

    for i := range snapshots {
        diff := abs(snapshots[i].Timestamp - targetTime)
        if diff <= tolerance && diff < minDiff {
            minDiff = diff
            closest = &snapshots[i]
        }
    }

    return closest
}

func abs(x int64) int64 {
    if x < 0 {
        return -x
    }
    return x
}
```

### CalculatePercentageChange Implementation Pattern

```go
// CalculatePercentageChange computes percentage change between old and new values.
// Returns 0 if old is 0 to avoid division by zero.
// Result is rounded to 2 decimal places.
func CalculatePercentageChange(oldValue, newValue float64) float64 {
    if oldValue == 0 {
        return 0
    }
    change := ((newValue - oldValue) / oldValue) * 100
    return math.Round(change*100) / 100
}
```

### CalculateChangeMetrics Implementation Pattern

```go
// CalculateChangeMetrics computes TVS and protocol count changes over 24h, 7d, and 30d periods.
// Returns pointer types for optional values (nil = no historical data available).
func CalculateChangeMetrics(currentTVS float64, currentProtocolCount int, history []Snapshot) ChangeMetrics {
    metrics := ChangeMetrics{}
    now := time.Now().Unix()

    // Find historical snapshots
    snapshot24h := FindSnapshotAtTime(history, now-Hours24, SnapshotTolerance)
    snapshot7d := FindSnapshotAtTime(history, now-Days7, SnapshotTolerance)
    snapshot30d := FindSnapshotAtTime(history, now-Days30, SnapshotTolerance)

    // Calculate TVS changes
    if snapshot24h != nil {
        change := CalculatePercentageChange(snapshot24h.TVS, currentTVS)
        metrics.Change24h = &change
    }

    if snapshot7d != nil {
        change := CalculatePercentageChange(snapshot7d.TVS, currentTVS)
        metrics.Change7d = &change
        protocolChange := currentProtocolCount - snapshot7d.ProtocolCount
        metrics.ProtocolCountChange7d = &protocolChange
    }

    if snapshot30d != nil {
        change := CalculatePercentageChange(snapshot30d.TVS, currentTVS)
        metrics.Change30d = &change
        protocolChange := currentProtocolCount - snapshot30d.ProtocolCount
        metrics.ProtocolCountChange30d = &protocolChange
    }

    return metrics
}
```

### Testing Strategy

Follow table-driven test pattern established in `internal/aggregator/metrics_test.go`:

```go
func TestCalculatePercentageChange(t *testing.T) {
    tests := []struct {
        name     string
        oldValue float64
        newValue float64
        want     float64
    }{
        {
            name:     "positive_10_percent_increase",
            oldValue: 1000,
            newValue: 1100,
            want:     10.0,
        },
        {
            name:     "negative_10_percent_decrease",
            oldValue: 1000,
            newValue: 900,
            want:     -10.0,
        },
        {
            name:     "zero_old_value_returns_zero",
            oldValue: 0,
            newValue: 100,
            want:     0,
        },
    }
    // ... test implementation
}

func TestFindSnapshotAtTime(t *testing.T) {
    now := time.Now().Unix()
    tests := []struct {
        name       string
        snapshots  []Snapshot
        targetTime int64
        tolerance  int64
        wantFound  bool
        wantTVS    float64
    }{
        {
            name: "exact_match",
            snapshots: []Snapshot{
                {Timestamp: now - Hours24, TVS: 1000},
            },
            targetTime: now - Hours24,
            tolerance:  SnapshotTolerance,
            wantFound:  true,
            wantTVS:    1000,
        },
        {
            name: "within_tolerance",
            snapshots: []Snapshot{
                {Timestamp: now - Hours24 + 3600, TVS: 1000}, // 1 hour off
            },
            targetTime: now - Hours24,
            tolerance:  SnapshotTolerance,
            wantFound:  true,
            wantTVS:    1000,
        },
        {
            name: "outside_tolerance",
            snapshots: []Snapshot{
                {Timestamp: now - Hours24 + 10800, TVS: 1000}, // 3 hours off
            },
            targetTime: now - Hours24,
            tolerance:  SnapshotTolerance,
            wantFound:  false,
        },
        {
            name:       "empty_history",
            snapshots:  nil,
            targetTime: now - Hours24,
            tolerance:  SnapshotTolerance,
            wantFound:  false,
        },
    }
    // ... test implementation
}
```

### Project Structure Notes

- Modify file: `internal/aggregator/models.go` - Add Snapshot and ChangeMetrics structs
- Modify file: `internal/aggregator/metrics.go` - Add constants, FindSnapshotAtTime, CalculatePercentageChange, CalculateChangeMetrics
- Modify file: `internal/aggregator/metrics_test.go` - Add comprehensive tests
- Aligns with fr-category-to-architecture-mapping.md: Aggregation & Metrics (FR15-FR24) -> `internal/aggregator`

### Learnings from Previous Story

**From Story 3-5-rank-protocols-and-identify-largest (Status: done)**

- Completion notes: rank logic patterns, tie-breaks, nil-safe returns validated; no outstanding review items [Source: docs/sprint-artifacts/3-5-rank-protocols-and-identify-largest.md#L267-L300]
- File carryover (new/modified in Story 3.5): `internal/aggregator/models.go`, `internal/aggregator/metrics.go`, `internal/aggregator/metrics_test.go`, `docs/sprint-artifacts/sprint-status.yaml`, `docs/sprint-artifacts/3-5-rank-protocols-and-identify-largest.md` — consider impacts to Snapshot/ChangeMetrics additions [Source: docs/sprint-artifacts/3-5-rank-protocols-and-identify-largest.md#L271-L277]
- Patterns to reuse: copy-before-sort, pointer returns for optional results, table-driven tests with edge cases and JSON assertions [Source: docs/sprint-artifacts/3-5-rank-protocols-and-identify-largest.md#L261-L269]
- Build/lint commands previously verified: `go build ./...`, `go test ./internal/aggregator/...`, `make lint`; reuse to validate new changes [Source: docs/sprint-artifacts/3-5-rank-protocols-and-identify-largest.md#L263-L269]

[Source: docs/sprint-artifacts/3-5-rank-protocols-and-identify-largest.md#Dev-Agent-Record]

### Key Implementation Considerations

1. **Pointer Types for Optional Values:** Use `*float64` and `*int` for change fields to distinguish "no data" (nil) from "zero change" (0)
2. **Time Tolerance:** Historical snapshots update hourly; 2-hour tolerance ensures we find the closest match
3. **Snapshot Order:** History is expected to be sorted newest-first; FindSnapshotAtTime handles any order
4. **Division by Zero:** CalculatePercentageChange returns 0 when old value is 0
5. **Rounding:** Round percentage to 2 decimal places for clean output
6. **Protocol Count Changes:** Track protocol adoption growth over time (FR22)

### Seed Documentation References

- Reference implementation in `docs-from-user/seed-doc/7-custom-aggregation-logic-go-implementation.md` section 7.1
- History management patterns in `docs-from-user/seed-doc/6-incremental-update-strategy.md` section 6.3

### References

- [Source: docs/epics/epic-3-data-processing-pipeline.md#story-36-calculate-historical-change-metrics] - Story definition and acceptance criteria
- [Source: docs/prd.md#fr19] - FR19: System calculates 24-hour TVS change percentage
- [Source: docs/prd.md#fr20] - FR20: System calculates 7-day TVS change percentage
- [Source: docs/prd.md#fr21] - FR21: System calculates 30-day TVS change percentage
- [Source: docs/prd.md#fr22] - FR22: System calculates protocol count growth over 7-day and 30-day periods
- [Source: docs/architecture/data-architecture.md#output-models] - Data structures and historical snapshot fields
- [Source: docs/architecture/implementation-patterns.md#patterns] - Go implementation and testing idioms
- [Source: docs/architecture/testing-strategy.md#test-organization] - Testing standards for Go components
- [Source: internal/aggregator/models.go] - Existing model structs to extend
- [Source: internal/aggregator/metrics.go] - Existing metrics patterns to follow
- [Source: docs-from-user/seed-doc/7-custom-aggregation-logic-go-implementation.md#71-metric-calculations] - Reference implementation

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/3-6-calculate-historical-change-metrics.context.xml

### Agent Model Used

- GPT-5 Codex (Scrum Master persona)

### Debug Log References

- docs/sprint-artifacts/validation-report-story-3-6-2025-11-30T19-31-48Z.md

### Completion Notes List

- Added architecture citations (data architecture, implementation patterns) per validation.
- Dev Agent Record populated with model and validation reference.

### File List

- docs/sprint-artifacts/3-6-calculate-historical-change-metrics.md (story)
- docs/sprint-artifacts/validation-report-story-3-6-2025-11-30T19-31-48Z.md (validation report)

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-30 | SM Agent (Bob) | Initial story draft created from epic-3-data-processing-pipeline.md |
| 2025-11-30 | SM Agent (Bob) | Added architecture citations and populated Dev Agent Record |
