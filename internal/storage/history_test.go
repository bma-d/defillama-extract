package storage

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/switchboard-xyz/defillama-extract/internal/aggregator"
)

func TestCreateSnapshot_PopulatesFields(t *testing.T) {
	result := &aggregator.AggregationResult{
		TotalTVS:       1_000_000.5,
		TotalProtocols: 42,
		ActiveChains:   []string{"solana", "ethereum", "arbitrum"},
		ChainBreakdown: []aggregator.ChainBreakdown{
			{Chain: "solana", TVS: 600_000.25},
			{Chain: "ethereum", TVS: 300_000.0},
			{Chain: "arbitrum", TVS: 100_000.25},
		},
		Timestamp: 1700000000,
	}

	snapshot := CreateSnapshot(result)

	expectedMap := map[string]float64{
		"solana":   600_000.25,
		"ethereum": 300_000.0,
		"arbitrum": 100_000.25,
	}

	if snapshot.Timestamp != result.Timestamp {
		t.Fatalf("timestamp mismatch: got %d want %d", snapshot.Timestamp, result.Timestamp)
	}

	if snapshot.Date != "2023-11-14" {
		t.Fatalf("date mismatch: got %s want %s", snapshot.Date, "2023-11-14")
	}

	if snapshot.TVS != result.TotalTVS {
		t.Fatalf("tvs mismatch: got %f want %f", snapshot.TVS, result.TotalTVS)
	}

	if !reflect.DeepEqual(snapshot.TVSByChain, expectedMap) {
		t.Fatalf("tvsByChain mismatch: got %+v want %+v", snapshot.TVSByChain, expectedMap)
	}

	if snapshot.ProtocolCount != result.TotalProtocols {
		t.Fatalf("protocol count mismatch: got %d want %d", snapshot.ProtocolCount, result.TotalProtocols)
	}

	if snapshot.ChainCount != len(result.ActiveChains) {
		t.Fatalf("chain count mismatch: got %d want %d", snapshot.ChainCount, len(result.ActiveChains))
	}
}

func TestCreateSnapshot_EmptyChainBreakdown(t *testing.T) {
	result := &aggregator.AggregationResult{
		TotalTVS:       0,
		TotalProtocols: 0,
		ActiveChains:   []string{},
		ChainBreakdown: []aggregator.ChainBreakdown{},
		Timestamp:      1,
	}

	snapshot := CreateSnapshot(result)

	if snapshot.TVSByChain == nil {
		t.Fatalf("tvsByChain is nil; expected empty map")
	}

	if len(snapshot.TVSByChain) != 0 {
		t.Fatalf("expected empty tvsByChain map, got len=%d", len(snapshot.TVSByChain))
	}

	if snapshot.ChainCount != 0 {
		t.Fatalf("expected chain count 0, got %d", snapshot.ChainCount)
	}
}

func TestCreateSnapshot_NilSlices(t *testing.T) {
	result := &aggregator.AggregationResult{Timestamp: 10}

	snapshot := CreateSnapshot(result)

	if snapshot.TVSByChain == nil {
		t.Fatalf("tvsByChain is nil; expected empty map when ChainBreakdown is nil")
	}

	if len(snapshot.TVSByChain) != 0 {
		t.Fatalf("expected empty map when ChainBreakdown is nil, got len=%d", len(snapshot.TVSByChain))
	}

	if snapshot.ChainCount != 0 {
		t.Fatalf("expected chain count 0 when ActiveChains is nil, got %d", snapshot.ChainCount)
	}
}

func TestCreateSnapshot_NilResult(t *testing.T) {
	snapshot := CreateSnapshot(nil)

	if snapshot.TVSByChain == nil {
		t.Fatalf("tvsByChain is nil; expected empty map for nil result")
	}

	if snapshot.Timestamp != 0 || snapshot.TVS != 0 || snapshot.ProtocolCount != 0 || snapshot.ChainCount != 0 || snapshot.Date != "" {
		t.Fatalf("expected zero-value fields for nil result, got %+v", snapshot)
	}
}

func TestCreateSnapshot_DateFormatting(t *testing.T) {
	tests := []struct {
		name      string
		timestamp int64
		wantDate  string
	}{
		{name: "AC timestamp", timestamp: 1700000000, wantDate: "2023-11-14"},
		{name: "year boundary", timestamp: 1703980800, wantDate: "2023-12-31"},
		{name: "new year", timestamp: 1704067200, wantDate: "2024-01-01"},
		{name: "leap day", timestamp: 1582934400, wantDate: "2020-02-29"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &aggregator.AggregationResult{Timestamp: tt.timestamp}
			snapshot := CreateSnapshot(result)

			if snapshot.Date != tt.wantDate {
				t.Fatalf("date mismatch for %s: got %s want %s", tt.name, snapshot.Date, tt.wantDate)
			}
		})
	}
}

func TestLoadFromOutput_ValidFileSorted(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	path := filepath.Join("testdata", "output_with_history.json")
	snapshots, err := LoadFromOutput(path, logger)
	if err != nil {
		t.Fatalf("LoadFromOutput returned error: %v", err)
	}

	if len(snapshots) != 3 {
		t.Fatalf("expected 3 snapshots, got %d", len(snapshots))
	}

	if snapshots[0].Timestamp != 1700000000 || snapshots[1].Timestamp != 1700003600 || snapshots[2].Timestamp != 1700007200 {
		t.Fatalf("snapshots not sorted ascending: %+v", snapshots)
	}

	if snapshots[0].TVS != 900000.0 || snapshots[2].TVS != 1100000.0 {
		t.Fatalf("unexpected TVS values in snapshots: %+v", snapshots)
	}

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	var payload map[string]any
	if err := json.Unmarshal(lines[len(lines)-1], &payload); err != nil {
		t.Fatalf("failed to parse log payload: %v", err)
	}
	if lvl, ok := payload["level"].(string); !ok || lvl != "DEBUG" {
		t.Fatalf("expected DEBUG log for history loaded, got %v", payload["level"])
	}
}

func TestLoadFromOutput_MissingFileReturnsEmpty(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)

	path := filepath.Join(t.TempDir(), "missing.json")

	snapshots, err := LoadFromOutput(path, logger)
	if err != nil {
		t.Fatalf("LoadFromOutput returned error: %v", err)
	}
	if len(snapshots) != 0 {
		t.Fatalf("expected empty slice for missing file, got %d", len(snapshots))
	}

	if !bytes.Contains(buf.Bytes(), []byte("no existing history found")) {
		t.Fatal("expected debug log for missing history file")
	}
}

func TestLoadFromOutput_EmptyHistoricalReturnsEmpty(t *testing.T) {
	logger := newTestLogger(&bytes.Buffer{})
	path := filepath.Join("testdata", "output_no_history.json")

	snapshots, err := LoadFromOutput(path, logger)
	if err != nil {
		t.Fatalf("LoadFromOutput returned error: %v", err)
	}
	if len(snapshots) != 0 {
		t.Fatalf("expected empty slice when historical is empty, got %d", len(snapshots))
	}
}

func TestLoadFromOutput_CorruptedReturnsEmptyAndWarns(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newTestLogger(buf)
	path := filepath.Join("testdata", "output_corrupted.json")

	snapshots, err := LoadFromOutput(path, logger)
	if err != nil {
		t.Fatalf("LoadFromOutput returned error: %v", err)
	}
	if len(snapshots) != 0 {
		t.Fatalf("expected empty slice for corrupted file, got %d", len(snapshots))
	}

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	var payload map[string]any
	if err := json.Unmarshal(lines[len(lines)-1], &payload); err != nil {
		t.Fatalf("failed to parse log payload: %v", err)
	}
	if lvl, ok := payload["level"].(string); !ok || lvl != "WARN" {
		t.Fatalf("expected WARN log for corrupted history, got %v", payload["level"])
	}
}
