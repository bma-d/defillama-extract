package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/switchboard-xyz/defillama-extract/internal/aggregator"
)

// State represents the last extraction state for incremental update tracking.
// A zero-value State (LastUpdated == 0) indicates the first run or recovery
// from a missing/corrupted state file.
type State struct {
	OracleName        string  `json:"oracle_name"`
	LastUpdated       int64   `json:"last_updated"`
	LastUpdatedISO    string  `json:"last_updated_iso"`
	LastProtocolCount int     `json:"last_protocol_count"`
	LastTVS           float64 `json:"last_tvs"`
	SnapshotCount     int     `json:"snapshot_count"`
	OldestSnapshot    int64   `json:"oldest_snapshot"`
	NewestSnapshot    int64   `json:"newest_snapshot"`
}

// StateManager handles state and history operations for incremental extraction.
type StateManager struct {
	outputDir  string
	stateFile  string
	outputFile string
	logger     *slog.Logger
}

// NewStateManager creates a StateManager for the given output directory.
func NewStateManager(outputDir string, logger *slog.Logger) *StateManager {
	if logger == nil {
		logger = slog.Default()
	}

	return &StateManager{
		outputDir:  outputDir,
		stateFile:  filepath.Join(outputDir, "state.json"),
		outputFile: filepath.Join(outputDir, "switchboard-oracle-data.json"),
		logger:     logger,
	}
}

// LoadState reads the state file and returns the parsed State.
// Missing files return a zero-value State (first run). Corrupted files log a
// warning and return a zero-value State to allow graceful recovery (FR28).
func (sm *StateManager) LoadState() (*State, error) {
	data, err := os.ReadFile(sm.stateFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			sm.logger.Debug("no state file found, treating as first run", "path", sm.stateFile)
			return &State{}, nil
		}
		return nil, fmt.Errorf("read state file %s: %w", sm.stateFile, err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		sm.logger.Warn("state file corrupted, treating as first run", "path", sm.stateFile, "error", err.Error())
		return &State{}, nil
	}

	sm.logger.Debug("state loaded",
		"path", sm.stateFile,
		"oracle_name", state.OracleName,
		"last_updated", state.LastUpdated,
		"last_protocol_count", state.LastProtocolCount,
		"last_tvs", state.LastTVS,
		"snapshot_count", state.SnapshotCount,
	)

	return &state, nil
}

// SaveState persists the state to disk using atomic write semantics.
func (sm *StateManager) SaveState(state *State) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	if err := os.MkdirAll(sm.outputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory %s: %w", sm.outputDir, err)
	}

	if err := WriteAtomic(sm.stateFile, data, 0o644); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}

	sm.logger.Info("state saved",
		"timestamp", state.LastUpdated,
		"protocol_count", state.LastProtocolCount,
		"tvs", state.LastTVS,
	)

	return nil
}

// ShouldProcess determines whether extraction should proceed based on the API
// timestamp compared to the last processed timestamp in state. It returns true
// when processing is required (first run or newer data) and false when work
// can be skipped (no new data or clock skew). All decisions are logged with
// appropriate levels and attributes per ADR-004.
func (sm *StateManager) ShouldProcess(currentTS int64, state *State) bool {
	switch {
	case state.LastUpdated == 0:
		sm.logger.Debug("first run, processing required")
		return true
	case currentTS > state.LastUpdated:
		delta := currentTS - state.LastUpdated
		sm.logger.Debug("new data available",
			"current_ts", currentTS,
			"last_ts", state.LastUpdated,
			"delta_seconds", delta,
		)
		return true
	case currentTS == state.LastUpdated:
		sm.logger.Info("no new data, skipping extraction",
			"current_ts", currentTS,
			"last_ts", state.LastUpdated,
		)
		return false
	default:
		sm.logger.Warn("clock skew detected, API timestamp older than last processed",
			"current_ts", currentTS,
			"last_ts", state.LastUpdated,
		)
		return false
	}
}

// UpdateState creates a new State populated from extraction outputs and history snapshots.
// Snapshot metadata falls back to zero when no snapshots are supplied.
func (sm *StateManager) UpdateState(oracleName string, ts int64, count int, tvs float64, snapshots []aggregator.Snapshot) *State {
	state := &State{
		OracleName:        oracleName,
		LastUpdated:       ts,
		LastUpdatedISO:    time.Unix(ts, 0).UTC().Format(time.RFC3339),
		LastProtocolCount: count,
		LastTVS:           tvs,
		SnapshotCount:     len(snapshots),
	}

	if len(snapshots) > 0 {
		state.OldestSnapshot = snapshots[0].Timestamp
		state.NewestSnapshot = snapshots[len(snapshots)-1].Timestamp
	}

	return state
}

// LoadHistory returns historical snapshots via the configured output file.
func (sm *StateManager) LoadHistory() ([]aggregator.Snapshot, error) {
	return LoadFromOutput(sm.outputFile, sm.logger)
}

// AppendSnapshot delegates snapshot deduplication and ordering to the package-level helper.
func (sm *StateManager) AppendSnapshot(history []aggregator.Snapshot, snapshot aggregator.Snapshot) []aggregator.Snapshot {
	return AppendSnapshot(history, snapshot, sm.logger)
}

// OutputFile exposes the configured output path for downstream consumers.
func (sm *StateManager) OutputFile() string {
	return sm.outputFile
}
