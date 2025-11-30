# 6. Incremental Update Strategy

## 6.1 Overview

The incremental update strategy minimizes redundant processing by:
1. Tracking the timestamp of the last processed data
2. Comparing with the latest available timestamp
3. Only processing when new data is detected
4. Maintaining a rolling window of historical snapshots

## 6.2 State Manager Implementation

```go
// internal/storage/state.go

package storage

import (
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "time"

    "github.com/yourorg/switchboard-extractor/internal/models"
)

// StateManager handles incremental update state
type StateManager struct {
    filePath string
}

// NewStateManager creates a new state manager
func NewStateManager(outputDir string) *StateManager {
    return &StateManager{
        filePath: filepath.Join(outputDir, "state.json"),
    }
}

// Load reads state from disk, returns empty state if not found
func (sm *StateManager) Load() (*models.State, error) {
    data, err := os.ReadFile(sm.filePath)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            return &models.State{}, nil
        }
        return nil, fmt.Errorf("reading state file: %w", err)
    }

    var state models.State
    if err := json.Unmarshal(data, &state); err != nil {
        return nil, fmt.Errorf("%w: %v", models.ErrStateCorrupted, err)
    }

    return &state, nil
}

// Save writes state to disk atomically
func (sm *StateManager) Save(state *models.State) error {
    state.LastUpdatedISO = time.Unix(state.LastUpdated, 0).UTC().Format(time.RFC3339)

    data, err := json.MarshalIndent(state, "", "  ")
    if err != nil {
        return fmt.Errorf("marshaling state: %w", err)
    }

    return WriteAtomic(sm.filePath, data, 0644)
}

// ShouldUpdate determines if new data should be processed
func (sm *StateManager) ShouldUpdate(state *models.State, latestTimestamp int64) bool {
    return latestTimestamp > state.LastUpdated
}

// UpdateState creates a new state from extraction result
func (sm *StateManager) UpdateState(
    state *models.State,
    oracleName string,
    timestamp int64,
    protocolCount int,
    tvs float64,
    snapshots []models.Snapshot,
) *models.State {
    var oldest, newest int64
    if len(snapshots) > 0 {
        oldest = snapshots[len(snapshots)-1].Timestamp
        newest = snapshots[0].Timestamp
    }

    return &models.State{
        OracleName:        oracleName,
        LastUpdated:       timestamp,
        LastProtocolCount: protocolCount,
        LastTVS:           tvs,
        SnapshotCount:     len(snapshots),
        OldestSnapshot:    oldest,
        NewestSnapshot:    newest,
    }
}
```

## 6.3 History Manager Implementation

```go
// internal/storage/history.go

package storage

import (
    "encoding/json"
    "os"
    "sort"
    "time"

    "github.com/yourorg/switchboard-extractor/internal/models"
)

const (
    DefaultRetentionDays = 90
    SecondsPerDay        = 24 * 60 * 60
)

// HistoryManager handles historical snapshot data
type HistoryManager struct {
    retentionDays int
}

// NewHistoryManager creates a new history manager
func NewHistoryManager(retentionDays int) *HistoryManager {
    if retentionDays <= 0 {
        retentionDays = DefaultRetentionDays
    }
    return &HistoryManager{retentionDays: retentionDays}
}

// LoadFromOutput reads historical snapshots from the full output file
func (hm *HistoryManager) LoadFromOutput(outputPath string) ([]models.Snapshot, error) {
    data, err := os.ReadFile(outputPath)
    if err != nil {
        if os.IsNotExist(err) {
            return []models.Snapshot{}, nil
        }
        return nil, err
    }

    var output models.FullOutput
    if err := json.Unmarshal(data, &output); err != nil {
        return nil, err
    }

    return output.Historical, nil
}

// Append adds a new snapshot, maintaining sort order (newest first)
func (hm *HistoryManager) Append(snapshots []models.Snapshot, newSnapshot models.Snapshot) []models.Snapshot {
    // Check for duplicate
    for i, s := range snapshots {
        if s.Timestamp == newSnapshot.Timestamp {
            // Replace existing with newer data
            snapshots[i] = newSnapshot
            return snapshots
        }
    }

    // Append and re-sort
    snapshots = append(snapshots, newSnapshot)
    sort.Slice(snapshots, func(i, j int) bool {
        return snapshots[i].Timestamp > snapshots[j].Timestamp
    })

    return snapshots
}

// Prune removes snapshots older than retentionDays
func (hm *HistoryManager) Prune(snapshots []models.Snapshot) []models.Snapshot {
    cutoffTime := time.Now().Unix() - int64(hm.retentionDays*SecondsPerDay)

    result := make([]models.Snapshot, 0, len(snapshots))
    for _, s := range snapshots {
        if s.Timestamp >= cutoffTime {
            result = append(result, s)
        }
    }

    return result
}

// Deduplicate removes duplicate timestamps, keeping the one with newer extraction time
func (hm *HistoryManager) Deduplicate(snapshots []models.Snapshot) []models.Snapshot {
    seen := make(map[int64]bool)
    result := make([]models.Snapshot, 0, len(snapshots))

    for _, s := range snapshots {
        if !seen[s.Timestamp] {
            seen[s.Timestamp] = true
            result = append(result, s)
        }
    }

    return result
}
```

---
