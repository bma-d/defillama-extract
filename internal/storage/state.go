package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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

// StateManager handles loading state from disk. Additional state operations
// (save, skip logic) are introduced in subsequent stories of the epic.
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
