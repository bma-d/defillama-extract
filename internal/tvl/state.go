package tvl

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/switchboard-xyz/defillama-extract/internal/storage"
)

// TVLState tracks the last successful TVL pipeline run for skip logic and metadata.
// A zero-value state represents a first run (no tvl-state.json present).
type TVLState struct {
	LastUpdated       int64  `json:"last_updated"`
	LastUpdatedISO    string `json:"last_updated_iso"`
	ProtocolCount     int    `json:"protocol_count"`
	CustomProtocolCnt int    `json:"custom_count"`
}

// TVLStateManager handles persistence of tvl-state.json using the same atomic
// write semantics as the main pipeline state manager.
type TVLStateManager struct {
	outputDir string
	stateFile string
	logger    *slog.Logger
}

// NewTVLStateManager constructs a TVLStateManager rooted at outputDir.
func NewTVLStateManager(outputDir string, logger *slog.Logger) *TVLStateManager {
	if logger == nil {
		logger = slog.Default()
	}

	return &TVLStateManager{
		outputDir: outputDir,
		stateFile: filepath.Join(outputDir, "tvl-state.json"),
		logger:    logger,
	}
}

// LoadState reads tvl-state.json, returning an empty state when the file is
// missing or corrupted to keep the pipeline resilient (AC5).
func (m *TVLStateManager) LoadState() (*TVLState, error) {
	data, err := os.ReadFile(m.stateFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			m.logger.Debug("tvl_state_missing", "path", m.stateFile)
			return &TVLState{}, nil
		}
		return nil, fmt.Errorf("read tvl state: %w", err)
	}

	var state TVLState
	if err := json.Unmarshal(data, &state); err != nil {
		m.logger.Warn("tvl_state_corrupted", "path", m.stateFile, "error", err.Error())
		return &TVLState{}, nil
	}

	m.logger.Debug("tvl_state_loaded",
		"path", m.stateFile,
		"last_updated", state.LastUpdated,
		"protocol_count", state.ProtocolCount,
		"custom_count", state.CustomProtocolCnt,
	)

	return &state, nil
}

// ShouldProcess decides whether to run the TVL pipeline based on the current
// extraction timestamp versus the last successful run timestamp.
func (m *TVLStateManager) ShouldProcess(currentTS int64, state *TVLState) bool {
	switch {
	case state == nil:
		return true
	case state.LastUpdated == 0:
		m.logger.Debug("tvl_first_run")
		return true
	case currentTS > state.LastUpdated:
		m.logger.Debug("tvl_new_data", "current_ts", currentTS, "last_ts", state.LastUpdated)
		return true
	case currentTS == state.LastUpdated:
		m.logger.Info("tvl_skip_no_change", "current_ts", currentTS)
		return false
	default:
		m.logger.Warn("tvl_timestamp_regression", "current_ts", currentTS, "last_ts", state.LastUpdated)
		return false
	}
}

// SaveState persists the TVL state atomically, including ISO timestamp for
// human readability.
func (m *TVLStateManager) SaveState(state *TVLState) error {
	if state == nil {
		return errors.New("nil state")
	}

	state.LastUpdatedISO = time.Unix(state.LastUpdated, 0).UTC().Format(time.RFC3339)

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tvl state: %w", err)
	}

	if err := os.MkdirAll(m.outputDir, 0o755); err != nil {
		return fmt.Errorf("create tvl output dir: %w", err)
	}

	if err := storage.WriteAtomic(m.stateFile, data, 0o644); err != nil {
		return fmt.Errorf("write tvl state: %w", err)
	}

	m.logger.Info("tvl_state_saved",
		"last_updated", state.LastUpdated,
		"protocol_count", state.ProtocolCount,
		"custom_count", state.CustomProtocolCnt,
	)

	return nil
}
