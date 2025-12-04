package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/switchboard-xyz/defillama-extract/internal/aggregator"
	"github.com/switchboard-xyz/defillama-extract/internal/config"
)

func sampleAggregationResult() *aggregator.AggregationResult {
	change24 := 1.5
	change7 := 7.2
	change30 := 30.4
	pc7 := 2
	pc30 := 4

	return &aggregator.AggregationResult{
		TotalTVS:       123.45,
		TotalProtocols: 3,
		ActiveChains:   []string{"Ethereum", "Solana"},
		Categories:     []string{"Lending", "DEX"},
		ChainBreakdown: []aggregator.ChainBreakdown{
			{Chain: "Ethereum", TVS: 100, Percentage: 81.0, ProtocolCount: 2},
			{Chain: "Solana", TVS: 23.45, Percentage: 19.0, ProtocolCount: 1},
		},
		CategoryBreakdown: []aggregator.CategoryBreakdown{
			{Category: "Lending", TVS: 80, Percentage: 64.8, ProtocolCount: 1},
			{Category: "DEX", TVS: 43.45, Percentage: 35.2, ProtocolCount: 2},
		},
		Protocols: []aggregator.AggregatedProtocol{
			{Name: "A", Slug: "a", TVS: 50, Rank: 1},
			{Name: "B", Slug: "b", TVS: 40, Rank: 2},
			{Name: "C", Slug: "c", TVS: 33.45, Rank: 3},
		},
		ChangeMetrics: aggregator.ChangeMetrics{
			Change24h:              &change24,
			Change7d:               &change7,
			Change30d:              &change30,
			ProtocolCountChange7d:  &pc7,
			ProtocolCountChange30d: &pc30,
		},
		Timestamp: time.Now().Unix(),
	}
}

func sampleConfig() *config.Config {
	cfg := config.Config{
		Oracle: config.OracleConfig{
			Name:          "Switchboard",
			Website:       "https://switchboard.xyz",
			Documentation: "https://docs.switchboard.xyz",
		},
		Output: config.OutputConfig{
			Directory:   "data",
			FullFile:    "switchboard-oracle-data.json",
			MinFile:     "switchboard-oracle-data.min.json",
			SummaryFile: "switchboard-summary.json",
		},
		Scheduler: config.SchedulerConfig{
			Interval: 2 * time.Hour,
		},
	}
	return &cfg
}

func chartHistorySample() []aggregator.ChartDataPoint {
	return []aggregator.ChartDataPoint{
		{Timestamp: 1, Date: "2021-01-01", TVS: 10},
		{Timestamp: 2, Date: "2021-01-02", TVS: 20, Borrowed: 1},
	}
}

func TestGenerateFullOutput_PopulatesAllFields(t *testing.T) {
	result := sampleAggregationResult()
	history := []aggregator.Snapshot{
		{Timestamp: 1, Date: "2025-01-01", TVS: 10},
		{Timestamp: 2, Date: "2025-01-02", TVS: 20},
	}

	cfg := sampleConfig()
	out := GenerateFullOutput(result, history, chartHistorySample(), cfg)

	if out.Version != outputVersion {
		t.Fatalf("version = %s, want %s", out.Version, outputVersion)
	}
	if out.Oracle.Name == "" || out.Oracle.Website == "" || out.Oracle.Documentation == "" {
		t.Fatalf("oracle info missing fields: %+v", out.Oracle)
	}
	if out.Metadata.DataSource != dataSource || out.Metadata.UpdateFrequency != cfg.Scheduler.Interval.String() || out.Metadata.ExtractorVersion != extractorVersion {
		t.Fatalf("metadata mismatch: %+v", out.Metadata)
	}
	if _, err := time.Parse(time.RFC3339, out.Metadata.LastUpdated); err != nil {
		t.Fatalf("last_updated not RFC3339: %v", err)
	}
	if out.Summary.TotalValueSecured != result.TotalTVS || out.Summary.TotalProtocols != result.TotalProtocols {
		t.Fatalf("summary mismatch: %+v", out.Summary)
	}
	if out.Metrics.CurrentTVS != result.TotalTVS {
		t.Fatalf("metrics current_tvs mismatch: %+v", out.Metrics)
	}
	if !reflect.DeepEqual(out.Breakdown.ByChain, result.ChainBreakdown) || !reflect.DeepEqual(out.Breakdown.ByCategory, result.CategoryBreakdown) {
		t.Fatalf("breakdown mismatch: %+v", out.Breakdown)
	}
	if !reflect.DeepEqual(out.Protocols, result.Protocols) {
		t.Fatalf("protocols mismatch")
	}
	if !reflect.DeepEqual(out.Historical, history) {
		t.Fatalf("history mismatch: %+v", out.Historical)
	}
}

func TestGenerateSummaryOutput_TopProtocolsLimitedAndNoHistory(t *testing.T) {
	result := sampleAggregationResult()
	// extend protocols to 12 to test trimming
	for i := 4; i <= 12; i++ {
		result.Protocols = append(result.Protocols, aggregator.AggregatedProtocol{
			Name: fmt.Sprintf("P%d", i),
			Slug: fmt.Sprintf("p%d", i),
			TVS:  float64(100 - i),
			Rank: i,
		})
	}

out := GenerateSummaryOutput(result, sampleConfig())

	if len(out.TopProtocols) != 10 {
		t.Fatalf("top protocol count = %d, want 10", len(out.TopProtocols))
	}
	if out.Metadata.LastUpdated == "" {
		t.Fatalf("expected metadata timestamp set")
	}
	// Ensure original slice not mutated beyond length check
	if len(result.Protocols) != 12 {
		t.Fatalf("original protocols mutated, len=%d", len(result.Protocols))
	}
}

func TestWriteJSON_IndentedAndCompact(t *testing.T) {
	dir := t.TempDir()
	pathIndented := filepath.Join(dir, "formatted.json")
	pathCompact := filepath.Join(dir, "compact.json")

	data := map[string]string{"hello": "world"}

	if err := WriteJSON(pathIndented, data, true); err != nil {
		t.Fatalf("WriteJSON indented error: %v", err)
	}
	if err := WriteJSON(pathCompact, data, false); err != nil {
		t.Fatalf("WriteJSON compact error: %v", err)
	}

	indentedBytes, _ := os.ReadFile(pathIndented)
	compactBytes, _ := os.ReadFile(pathCompact)

	if !strings.Contains(string(indentedBytes), "\n") || !strings.Contains(string(indentedBytes), "  ") {
		t.Fatalf("indented JSON missing expected formatting: %q", string(indentedBytes))
	}
	if strings.Contains(string(compactBytes), "\n") || strings.Contains(string(compactBytes), "  ") {
		t.Fatalf("compact JSON contains whitespace: %q", string(compactBytes))
	}
}

type badJSON struct{}

func (badJSON) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("marshal failure")
}

func TestWriteJSON_CleansUpOnMarshalError(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "bad.json")

	err := WriteJSON(target, badJSON{}, true)
	if err == nil {
		t.Fatalf("expected error from marshal failure")
	}

	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("expected no file written on marshal error")
	}
}

func TestWriteAllOutputs_WritesAllFilesAndMatchesData(t *testing.T) {
	dir := t.TempDir()
	result := sampleAggregationResult()
	history := []aggregator.Snapshot{{Timestamp: 1, Date: "2025-01-01", TVS: 10}}
	cfg := sampleConfig()

	full := GenerateFullOutput(result, history, chartHistorySample(), cfg)
	summary := GenerateSummaryOutput(result, cfg)

	if err := WriteAllOutputs(context.Background(), dir, cfg, full, summary); err != nil {
		t.Fatalf("WriteAllOutputs error: %v", err)
	}

	fullPath := filepath.Join(dir, fullOutputFileName)
	minPath := filepath.Join(dir, minifiedOutputFileName)
	summaryPath := filepath.Join(dir, summaryOutputFileName)

	for _, p := range []string{fullPath, minPath, summaryPath} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected file written: %s, err: %v", p, err)
		}
	}

	fullDataIndented, _ := os.ReadFile(fullPath)
	fullDataMin, _ := os.ReadFile(minPath)

	var parsedIndented map[string]interface{}
	var parsedMin map[string]interface{}
	if err := json.Unmarshal(fullDataIndented, &parsedIndented); err != nil {
		t.Fatalf("unmarshal indented: %v", err)
	}
	if err := json.Unmarshal(fullDataMin, &parsedMin); err != nil {
		t.Fatalf("unmarshal minified: %v", err)
	}
	if !reflect.DeepEqual(parsedIndented, parsedMin) {
		t.Fatalf("minified and indented outputs differ")
	}

	if strings.Contains(string(fullDataMin), "\n") || strings.Contains(string(fullDataMin), "  ") {
		t.Fatalf("minified output contains whitespace")
	}

	summaryBytes, _ := os.ReadFile(summaryPath)
	if strings.Contains(string(summaryBytes), "historical") {
		t.Fatalf("summary output should not include historical data")
	}
}

func TestWriteAllOutputs_RespectsConfigFilenames(t *testing.T) {
	dir := t.TempDir()
	cfg := sampleConfig()
	cfg.Output.FullFile = "custom-full.json"
	cfg.Output.MinFile = "custom-min.json"
	cfg.Output.SummaryFile = "custom-summary.json"

full := GenerateFullOutput(sampleAggregationResult(), nil, chartHistorySample(), cfg)
summary := GenerateSummaryOutput(sampleAggregationResult(), cfg)

	if err := WriteAllOutputs(context.Background(), dir, cfg, full, summary); err != nil {
		t.Fatalf("WriteAllOutputs error: %v", err)
	}

	customPaths := []string{
		filepath.Join(dir, cfg.Output.FullFile),
		filepath.Join(dir, cfg.Output.MinFile),
		filepath.Join(dir, cfg.Output.SummaryFile),
	}

	for _, p := range customPaths {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected custom file written: %s", p)
		}
	}

	defaultPaths := []string{
		filepath.Join(dir, fullOutputFileName),
		filepath.Join(dir, minifiedOutputFileName),
		filepath.Join(dir, summaryOutputFileName),
	}

	for _, p := range defaultPaths {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("expected default path not used: %s", p)
		}
	}
}

func TestWriteAllOutputs_CancelsBeforeWritesAndLeavesNoFiles(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dir := t.TempDir()
	cfg := sampleConfig()
full := GenerateFullOutput(sampleAggregationResult(), nil, chartHistorySample(), cfg)
summary := GenerateSummaryOutput(sampleAggregationResult(), cfg)

	err := WriteAllOutputs(ctx, dir, cfg, full, summary)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}

	paths := []string{
		filepath.Join(dir, resolveFileName(cfg.Output.FullFile, fullOutputFileName)),
		filepath.Join(dir, resolveFileName(cfg.Output.MinFile, minifiedOutputFileName)),
		filepath.Join(dir, resolveFileName(cfg.Output.SummaryFile, summaryOutputFileName)),
	}

	for _, p := range paths {
		if _, statErr := os.Stat(p); !os.IsNotExist(statErr) {
			t.Fatalf("expected no file written on cancel: %s", p)
		}
	}
}

func TestUpdateFrequency_UsesSchedulerInterval(t *testing.T) {
	cfg := sampleConfig()
	cfg.Scheduler.Interval = 30 * time.Minute

	full := GenerateFullOutput(sampleAggregationResult(), nil, chartHistorySample(), cfg)
	if full.Metadata.UpdateFrequency != cfg.Scheduler.Interval.String() {
		t.Fatalf("update_frequency = %s, want %s", full.Metadata.UpdateFrequency, cfg.Scheduler.Interval.String())
	}

summary := GenerateSummaryOutput(sampleAggregationResult(), cfg)
	if summary.Metadata.UpdateFrequency != cfg.Scheduler.Interval.String() {
		t.Fatalf("summary update_frequency = %s, want %s", summary.Metadata.UpdateFrequency, cfg.Scheduler.Interval.String())
	}
}

func TestWriteAtomic_Success(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "state.json")
	data := []byte("hello")

	if err := WriteAtomic(target, data, 0o644); err != nil {
		t.Fatalf("WriteAtomic returned error: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("failed reading target: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("content mismatch: got %q, want %q", got, data)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("permissions = %v, want %v", info.Mode().Perm(), os.FileMode(0o644))
	}

	tmpFiles, err := filepath.Glob(filepath.Join(dir, ".tmp-*"))
	if err != nil {
		t.Fatalf("glob failed: %v", err)
	}
	if len(tmpFiles) != 0 {
		t.Fatalf("expected temp files cleaned up, found %v", tmpFiles)
	}
}

func TestWriteAtomic_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "nested", "state.json")

	if err := WriteAtomic(target, []byte("data"), 0o644); err != nil {
		t.Fatalf("WriteAtomic returned error: %v", err)
	}

	if _, err := os.Stat(target); err != nil {
		t.Fatalf("target missing: %v", err)
	}

	info, err := os.Stat(filepath.Dir(target))
	if err != nil {
		t.Fatalf("stat dir failed: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("dir perm = %v, want %v", info.Mode().Perm(), os.FileMode(0o755))
	}
}

func TestWriteAtomic_CleanupAndPreserveOnError(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "state.json")
	if err := os.WriteFile(target, []byte("original"), 0o644); err != nil {
		t.Fatalf("seed state file: %v", err)
	}

	// Make directory read-only to force failure during temp file creation.
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}

	err := WriteAtomic(target, []byte("new"), 0o644)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Restore permissions to inspect contents.
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatalf("restore chmod: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read original: %v", err)
	}
	if string(data) != "original" {
		t.Fatalf("original file modified on error: %q", data)
	}

	tmpFiles, err := filepath.Glob(filepath.Join(dir, ".tmp-*"))
	if err != nil {
		t.Fatalf("glob failed: %v", err)
	}
	if len(tmpFiles) != 0 {
		t.Fatalf("expected temp cleanup on error, found %v", tmpFiles)
	}
}
