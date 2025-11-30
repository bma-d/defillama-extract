package storage

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestNewStateManager_PathConstruction(t *testing.T) {
	tmpDir := t.TempDir()
	logger := newTestLogger(&bytes.Buffer{})

	sm := NewStateManager(tmpDir, logger)

	if sm.outputDir != tmpDir {
		t.Fatalf("outputDir = %s, want %s", sm.outputDir, tmpDir)
	}
	if sm.stateFile != filepath.Join(tmpDir, "state.json") {
		t.Fatalf("stateFile = %s, want %s", sm.stateFile, filepath.Join(tmpDir, "state.json"))
	}
	if sm.outputFile != filepath.Join(tmpDir, "switchboard-oracle-data.json") {
		t.Fatalf("outputFile = %s, want %s", sm.outputFile, filepath.Join(tmpDir, "switchboard-oracle-data.json"))
	}
	if sm.logger == nil {
		t.Fatal("logger should be set")
	}
}

func TestLoadState_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	fixturePath := filepath.Join("testdata", "valid_state.json")
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed reading fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "state.json"), data, 0o644); err != nil {
		t.Fatalf("failed writing state file: %v", err)
	}

	sm := NewStateManager(tmpDir, logger)
	state, err := sm.LoadState()
	if err != nil {
		t.Fatalf("LoadState returned error: %v", err)
	}

	var expected State
	if err := json.Unmarshal(data, &expected); err != nil {
		t.Fatalf("fixture unmarshal failed: %v", err)
	}

	if !reflect.DeepEqual(state, &expected) {
		t.Fatalf("state mismatch\ngot:  %+v\nwant: %+v", state, expected)
	}

	if buf.Len() == 0 {
		t.Fatal("expected debug log for loaded state")
	}
}

func TestLoadState_MissingFileReturnsZeroValue(t *testing.T) {
	tmpDir := t.TempDir()
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	sm := NewStateManager(tmpDir, logger)
	state, err := sm.LoadState()
	if err != nil {
		t.Fatalf("LoadState returned error: %v", err)
	}
	if *state != (State{}) {
		t.Fatalf("expected zero-value State, got %+v", state)
	}

	if !bytes.Contains(buf.Bytes(), []byte("first run")) {
		t.Fatal("expected debug log indicating first run")
	}
}

func TestLoadState_CorruptedJSONReturnsZeroValueAndWarns(t *testing.T) {
	tmpDir := t.TempDir()
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	fixturePath := filepath.Join("testdata", "corrupted_state.json")
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed reading corrupted fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "state.json"), data, 0o644); err != nil {
		t.Fatalf("failed writing corrupted state file: %v", err)
	}

	sm := NewStateManager(tmpDir, logger)
	state, err := sm.LoadState()
	if err != nil {
		t.Fatalf("LoadState returned error: %v", err)
	}
	if *state != (State{}) {
		t.Fatalf("expected zero-value State on corruption, got %+v", state)
	}

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	if len(lines) == 0 {
		t.Fatal("expected warning log for corrupted state")
	}

	var payload map[string]any
	if err := json.Unmarshal(lines[len(lines)-1], &payload); err != nil {
		t.Fatalf("failed to parse log payload: %v", err)
	}
	if lvl, ok := payload["level"].(string); !ok || lvl != "WARN" {
		t.Fatalf("expected WARN level log, got %v", payload["level"])
	}
}

func TestLoadState_PartialJSONReturnsZeroValue(t *testing.T) {
	tmpDir := t.TempDir()
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	fixturePath := filepath.Join("testdata", "partial_state.json")
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed reading partial fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "state.json"), data, 0o644); err != nil {
		t.Fatalf("failed writing partial state file: %v", err)
	}

	sm := NewStateManager(tmpDir, logger)
	state, err := sm.LoadState()
	if err != nil {
		t.Fatalf("LoadState returned error: %v", err)
	}
	if *state != (State{}) {
		t.Fatalf("expected zero-value State for partial JSON, got %+v", state)
	}
}

func TestStateJSONRoundTrip(t *testing.T) {
	original := State{
		OracleName:        "switchboard",
		LastUpdated:       1700001234,
		LastUpdatedISO:    "2023-11-14T22:00:00Z",
		LastProtocolCount: 42,
		LastTVS:           123.45,
		SnapshotCount:     7,
		OldestSnapshot:    1699000000,
		NewestSnapshot:    1700001234,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var restored State
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if !reflect.DeepEqual(original, restored) {
		t.Fatalf("round-trip mismatch\ngot:  %+v\nwant: %+v", restored, original)
	}
}
