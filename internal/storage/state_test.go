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
