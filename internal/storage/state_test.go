package storage

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func TestStateManager_ShouldProcess(t *testing.T) {
	tmpDir := t.TempDir()
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)
	sm := NewStateManager(tmpDir, logger)

	tests := []struct {
		name        string
		currentTS   int64
		lastUpdated int64
		want        bool
		wantLevel   string
		msgContains string
		wantAttrs   map[string]int64
	}{
		{
			name:        "first run returns true",
			currentTS:   1700000000,
			lastUpdated: 0,
			want:        true,
			wantLevel:   "DEBUG",
			msgContains: "first run, processing required",
		},
		{
			name:        "new data returns true",
			currentTS:   1700003600,
			lastUpdated: 1700000000,
			want:        true,
			wantLevel:   "DEBUG",
			msgContains: "new data available",
			wantAttrs: map[string]int64{
				"current_ts":    1700003600,
				"last_ts":       1700000000,
				"delta_seconds": 3600,
			},
		},
		{
			name:        "same timestamp returns false",
			currentTS:   1700000000,
			lastUpdated: 1700000000,
			want:        false,
			wantLevel:   "INFO",
			msgContains: "no new data, skipping extraction",
			wantAttrs: map[string]int64{
				"current_ts": 1700000000,
				"last_ts":    1700000000,
			},
		},
		{
			name:        "clock skew returns false",
			currentTS:   1700000000,
			lastUpdated: 1700003600,
			want:        false,
			wantLevel:   "WARN",
			msgContains: "clock skew detected, API timestamp older than last processed",
			wantAttrs: map[string]int64{
				"current_ts": 1700000000,
				"last_ts":    1700003600,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()

			state := &State{LastUpdated: tt.lastUpdated}
			got := sm.ShouldProcess(tt.currentTS, state)
			if got != tt.want {
				t.Fatalf("ShouldProcess() = %v, want %v", got, tt.want)
			}

			lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
			if len(lines) == 0 {
				t.Fatal("expected log output")
			}

			var payload map[string]any
			if err := json.Unmarshal(lines[len(lines)-1], &payload); err != nil {
				t.Fatalf("failed to parse log payload: %v", err)
			}

			if lvl, ok := payload["level"].(string); !ok || lvl != tt.wantLevel {
				t.Fatalf("log level = %v, want %s", payload["level"], tt.wantLevel)
			}
			if msg, ok := payload["msg"].(string); !ok || !strings.Contains(msg, tt.msgContains) {
				t.Fatalf("log msg = %v, want contains %q", payload["msg"], tt.msgContains)
			}

			for k, v := range tt.wantAttrs {
				gotVal, ok := payload[k]
				if !ok {
					t.Fatalf("expected attribute %s in log", k)
				}
				if gotNum, ok := gotVal.(float64); !ok || int64(gotNum) != v {
					t.Fatalf("attribute %s = %v, want %d", k, gotVal, v)
				}
			}
		})
	}
}

func TestSaveState_Success(t *testing.T) {
	tmpDir := t.TempDir()
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)
	sm := NewStateManager(tmpDir, logger)

	state := &State{
		OracleName:        "switchboard",
		LastUpdated:       1700001234,
		LastProtocolCount: 10,
		LastTVS:           123.45,
	}

	if err := sm.SaveState(state); err != nil {
		t.Fatalf("SaveState returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "state.json"))
	if err != nil {
		t.Fatalf("failed reading state file: %v", err)
	}

	var got State
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal state: %v", err)
	}

	if got.LastUpdated != state.LastUpdated || got.LastTVS != state.LastTVS || got.LastProtocolCount != state.LastProtocolCount {
		t.Fatalf("state mismatch after save: %+v", got)
	}

	info, err := os.Stat(filepath.Join(tmpDir, "state.json"))
	if err != nil {
		t.Fatalf("stat state file: %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("state file perm = %v, want %v", info.Mode().Perm(), os.FileMode(0o644))
	}
}

func TestSaveState_CreatesDirectory(t *testing.T) {
	baseDir := t.TempDir()
	outputDir := filepath.Join(baseDir, "nested", "state")
	sm := NewStateManager(outputDir, newTestLogger(&bytes.Buffer{}))

	if err := sm.SaveState(&State{}); err != nil {
		t.Fatalf("SaveState returned error: %v", err)
	}

	if _, err := os.Stat(outputDir); err != nil {
		t.Fatalf("output dir missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "state.json")); err != nil {
		t.Fatalf("state file missing: %v", err)
	}
}

func TestSaveState_LogsSuccess(t *testing.T) {
	buf := &bytes.Buffer{}
	sm := NewStateManager(t.TempDir(), newTestLogger(buf))
	state := &State{LastUpdated: 1700000000, LastProtocolCount: 5, LastTVS: 42.0}

	if err := sm.SaveState(state); err != nil {
		t.Fatalf("SaveState returned error: %v", err)
	}

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	if len(lines) == 0 {
		t.Fatal("expected log output")
	}

	var payload map[string]any
	if err := json.Unmarshal(lines[len(lines)-1], &payload); err != nil {
		t.Fatalf("parse log payload: %v", err)
	}

	if msg, ok := payload["msg"].(string); !ok || msg != "state saved" {
		t.Fatalf("log msg = %v, want 'state saved'", payload["msg"])
	}
	if lvl, ok := payload["level"].(string); !ok || lvl != "INFO" {
		t.Fatalf("log level = %v, want INFO", payload["level"])
	}

	if ts, ok := payload["timestamp"].(float64); !ok || int64(ts) != state.LastUpdated {
		t.Fatalf("timestamp attr = %v, want %d", payload["timestamp"], state.LastUpdated)
	}
	if pc, ok := payload["protocol_count"].(float64); !ok || int(pc) != state.LastProtocolCount {
		t.Fatalf("protocol_count attr = %v, want %d", payload["protocol_count"], state.LastProtocolCount)
	}
	if tvs, ok := payload["tvs"].(float64); !ok || tvs != state.LastTVS {
		t.Fatalf("tvs attr = %v, want %f", payload["tvs"], state.LastTVS)
	}
}

func TestSaveState_ErrorWhenOutputDirNotWritable(t *testing.T) {
	baseDir := t.TempDir()
	readOnly := filepath.Join(baseDir, "ro")
	if err := os.MkdirAll(readOnly, 0o500); err != nil {
		t.Fatalf("mkdir read-only dir: %v", err)
	}

	outputDir := filepath.Join(readOnly, "child")
	sm := NewStateManager(outputDir, newTestLogger(&bytes.Buffer{}))

	if err := sm.SaveState(&State{}); err == nil {
		t.Fatal("expected error when output dir not writable")
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
