package tvl

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/switchboard-xyz/defillama-extract/internal/api"
	"github.com/switchboard-xyz/defillama-extract/internal/models"
)

func TestCustomDataLoader_LoadsValidFiles_ExistingProtocol(t *testing.T) {
	dir := t.TempDir()
	// alpha is a known slug - only needs slug + tvl_history
	writeFile(t, filepath.Join(dir, "alpha.json"), `{"slug":"alpha","tvl_history":[{"date":"2024-01-01","timestamp":1704067200,"tvl":1.5}]}`)
	writeFile(t, filepath.Join(dir, "_example.json.template"), `{}`)

	handler := &recordingHandler{}
	loader := NewCustomDataLoader(dir, slog.New(handler))

	knownSlugs := map[string]struct{}{"alpha": {}}
	customProtocolSlugs := map[string]struct{}{}

	result, err := loader.Load(context.Background(), knownSlugs, customProtocolSlugs)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(result.History) != 1 {
		t.Fatalf("expected 1 protocol, got %d", len(result.History))
	}
	if len(result.History["alpha"]) != 1 || result.History["alpha"][0].TVL != 1.5 {
		t.Fatalf("unexpected history %+v", result.History["alpha"])
	}
	if len(result.NewProtocols) != 0 {
		t.Fatalf("expected no new protocols, got %d", len(result.NewProtocols))
	}

	stats := loader.Stats()
	if stats.FilesLoaded != 1 || stats.InvalidFiles != 0 || stats.EntriesLoaded != 1 {
		t.Fatalf("stats %+v not expected", stats)
	}
}

func TestCustomDataLoader_InvalidJSONContinues(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "bad.json"), `{"slug":`)
	writeFile(t, filepath.Join(dir, "good.json"), `{"slug":"ok","tvl_history":[{"date":"2024-02-01","timestamp":1706745600,"tvl":2}]}`)

	handler := &recordingHandler{}
	loader := NewCustomDataLoader(dir, slog.New(handler))

	knownSlugs := map[string]struct{}{"ok": {}}
	result, err := loader.Load(context.Background(), knownSlugs, nil)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(result.History) != 1 {
		t.Fatalf("expected 1 valid protocol, got %d", len(result.History))
	}
	if stats := loader.Stats(); stats.InvalidFiles != 1 || stats.FilesLoaded != 1 {
		t.Fatalf("stats %+v not expected", stats)
	}
}

func TestCustomDataLoader_EmptyDirectory_NoErrorAndLogsSummary(t *testing.T) {
	dir := t.TempDir()
	handler := &recordingHandler{}
	loader := NewCustomDataLoader(dir, slog.New(handler))

	result, err := loader.Load(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(result.History) != 0 {
		t.Fatalf("expected no protocols, got %d", len(result.History))
	}

	stats := loader.Stats()
	if stats.FilesScanned != 0 || stats.FilesLoaded != 0 || stats.EntriesLoaded != 0 || stats.InvalidFiles != 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	summary := findRecord(handler.records, "custom_data_loaded")
	if summary == nil {
		t.Fatalf("expected custom_data_loaded log")
	}
	attrs := attrsMap(*summary)
	if attrs["files_scanned"] != int64(0) || attrs["files_loaded"] != int64(0) || attrs["entries_loaded"] != int64(0) || attrs["invalid_files"] != int64(0) {
		t.Fatalf("unexpected summary attrs: %+v", attrs)
	}
}

func TestCustomDataLoader_InvalidSchema_WarnsAndSkips(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "bad.json"), `{"slug":"", "tvl_history":[]}`)

	handler := &recordingHandler{}
	loader := NewCustomDataLoader(dir, slog.New(handler))

	result, err := loader.Load(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(result.History) != 0 {
		t.Fatalf("expected no valid protocols, got %d", len(result.History))
	}

	stats := loader.Stats()
	if stats.InvalidFiles != 1 || stats.FilesLoaded != 0 || stats.FilesScanned != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	if findRecord(handler.records, "custom_data_invalid_schema") == nil {
		t.Fatalf("expected custom_data_invalid_schema log")
	}
	if findRecord(handler.records, "custom_data_loaded") == nil {
		t.Fatalf("expected final summary log")
	}
}

func TestCustomDataLoader_NewProtocol_WithFullMetadata(t *testing.T) {
	dir := t.TempDir()
	// New protocol with all required metadata
	writeFile(t, filepath.Join(dir, "newproto.json"), `{
		"slug": "newproto",
		"is-ongoing": false,
		"live": true,
		"simple-tvs-ratio": 0.75,
		"is-defillama": false,
		"docs_proof": "https://example.com/docs",
		"tvl_history": [{"date":"2024-01-01","timestamp":1704067200,"tvl":100}]
	}`)

	handler := &recordingHandler{}
	loader := NewCustomDataLoader(dir, slog.New(handler))

	// Empty known slugs - this is a new protocol
	result, err := loader.Load(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(result.History) != 1 {
		t.Fatalf("expected 1 protocol history, got %d", len(result.History))
	}
	if len(result.NewProtocols) != 1 {
		t.Fatalf("expected 1 new protocol, got %d", len(result.NewProtocols))
	}

	p := result.NewProtocols[0]
	if p.Slug != "newproto" {
		t.Fatalf("expected slug newproto, got %s", p.Slug)
	}
	if p.IsOngoing != false {
		t.Fatalf("expected is-ongoing false, got %v", p.IsOngoing)
	}
	if p.Live != true {
		t.Fatalf("expected live true, got %v", p.Live)
	}
	if p.SimpleTVSRatio != 0.75 {
		t.Fatalf("expected simple-tvs-ratio 0.75, got %v", p.SimpleTVSRatio)
	}
	if p.DocsProof == nil || *p.DocsProof != "https://example.com/docs" {
		t.Fatalf("expected docs_proof, got %v", p.DocsProof)
	}
}

func TestCustomDataLoader_NewProtocol_MissingRequiredFields(t *testing.T) {
	dir := t.TempDir()
	// New protocol missing is-ongoing (has partial metadata)
	writeFile(t, filepath.Join(dir, "partial.json"), `{
		"slug": "partial",
		"live": true,
		"tvl_history": [{"date":"2024-01-01","timestamp":1704067200,"tvl":100}]
	}`)

	handler := &recordingHandler{}
	loader := NewCustomDataLoader(dir, slog.New(handler))

	result, err := loader.Load(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	// Should be skipped due to validation error
	if len(result.History) != 0 {
		t.Fatalf("expected 0 protocols (invalid), got %d", len(result.History))
	}
	if stats := loader.Stats(); stats.InvalidFiles != 1 {
		t.Fatalf("expected 1 invalid file, got %d", stats.InvalidFiles)
	}
}

func TestCustomDataLoader_NewProtocol_NoMetadata_Rejected(t *testing.T) {
	dir := t.TempDir()
	// New protocol with only slug + tvl_history (no metadata) - should be rejected
	writeFile(t, filepath.Join(dir, "nometa.json"), `{
		"slug": "nometa",
		"tvl_history": [{"date":"2024-01-01","timestamp":1704067200,"tvl":100}]
	}`)

	handler := &recordingHandler{}
	loader := NewCustomDataLoader(dir, slog.New(handler))

	result, err := loader.Load(context.Background(), nil, nil) // nil = unknown slug
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	// Should be skipped - new protocol needs metadata
	if len(result.History) != 0 {
		t.Fatalf("expected 0 protocols, got %d", len(result.History))
	}
	if stats := loader.Stats(); stats.InvalidFiles != 1 {
		t.Fatalf("expected 1 invalid, got %d", stats.InvalidFiles)
	}
}

func TestCustomDataLoader_ExistingProtocol_HistoryOnly(t *testing.T) {
	dir := t.TempDir()
	// Existing protocol - only slug + tvl_history needed
	writeFile(t, filepath.Join(dir, "existing.json"), `{
		"slug": "existing",
		"tvl_history": [{"date":"2024-01-01","timestamp":1704067200,"tvl":50}]
	}`)

	handler := &recordingHandler{}
	loader := NewCustomDataLoader(dir, slog.New(handler))

	knownSlugs := map[string]struct{}{"existing": {}}
	result, err := loader.Load(context.Background(), knownSlugs, nil)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(result.History) != 1 {
		t.Fatalf("expected 1 protocol, got %d", len(result.History))
	}
	if len(result.NewProtocols) != 0 {
		t.Fatalf("expected 0 new protocols, got %d", len(result.NewProtocols))
	}
}

func TestCustomDataLoader_DuplicateSlug_Panics(t *testing.T) {
	dir := t.TempDir()
	// Slug exists in custom-protocols.json AND custom-data has metadata -> panic
	writeFile(t, filepath.Join(dir, "dupe.json"), `{
		"slug": "dupe",
		"is-ongoing": false,
		"live": true,
		"simple-tvs-ratio": 1.0,
		"tvl_history": [{"date":"2024-01-01","timestamp":1704067200,"tvl":100}]
	}`)

	handler := &recordingHandler{}
	loader := NewCustomDataLoader(dir, slog.New(handler))

	knownSlugs := map[string]struct{}{"dupe": {}}
	customProtocolSlugs := map[string]struct{}{"dupe": {}}

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic for duplicate slug, got none")
		}
	}()

	loader.Load(context.Background(), knownSlugs, customProtocolSlugs)
}

func TestCustomDataLoader_DuplicateSlug_HistoryOnly_NoPanic(t *testing.T) {
	dir := t.TempDir()
	// Slug in custom-protocols.json but custom-data has NO metadata -> OK (history-only)
	writeFile(t, filepath.Join(dir, "dupe.json"), `{
		"slug": "dupe",
		"tvl_history": [{"date":"2024-01-01","timestamp":1704067200,"tvl":100}]
	}`)

	handler := &recordingHandler{}
	loader := NewCustomDataLoader(dir, slog.New(handler))

	knownSlugs := map[string]struct{}{"dupe": {}}
	customProtocolSlugs := map[string]struct{}{"dupe": {}}

	result, err := loader.Load(context.Background(), knownSlugs, customProtocolSlugs)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	// Should succeed - history only, no duplicate
	if len(result.History) != 1 {
		t.Fatalf("expected 1 protocol, got %d", len(result.History))
	}
}

func TestMergeTVLHistory_CustomOverridesAndSorts(t *testing.T) {
	apiData := []models.TVLHistoryItem{
		{Date: "2024-01-01", Timestamp: 1, TVL: 10},
		{Date: "2024-01-02", Timestamp: 2, TVL: 20},
	}
	custom := []models.TVLHistoryItem{
		{Date: "2024-01-02", Timestamp: 3, TVL: 25},
		{Date: "2023-12-31", Timestamp: 0, TVL: 5},
	}

	got := MergeTVLHistory(apiData, custom)
	if len(got) != 3 {
		t.Fatalf("expected 3 items, got %d", len(got))
	}
	if got[2].TVL != 25 {
		t.Fatalf("expected custom value to override, got %+v", got[2])
	}
	if got[0].Date != "2023-12-31" {
		t.Fatalf("expected earliest date first, got %s", got[0].Date)
	}
}

func TestMergeCustomTVLData_CustomOnlyProtocol(t *testing.T) {
	custom := map[string][]models.TVLHistoryItem{
		"custom": {
			{Date: "2024-03-01", Timestamp: 1709251200, TVL: 30},
		},
	}

	result, stats := mergeCustomTVLData(nil, custom, nil)
	if len(result) != 1 {
		t.Fatalf("expected 1 protocol, got %d", len(result))
	}
	resp := result["custom"]
	if resp == nil || len(resp.TVL) != 1 {
		t.Fatalf("unexpected response %+v", resp)
	}
	if stats.CustomOnlyProtocols != 1 || stats.ProtocolsWithCustomData != 1 || stats.EntriesMerged != 1 {
		t.Fatalf("unexpected stats %+v", stats)
	}
}

func TestMergeCustomTVLData_PreservesAPIName(t *testing.T) {
	apiResp := &api.ProtocolTVLResponse{
		Name: "ApiName",
		TVL: []api.TVLDataPoint{
			{Date: 10, TotalLiquidityUSD: 1},
		},
	}
	custom := map[string][]models.TVLHistoryItem{
		"slug": {
			{Date: "1970-01-01", Timestamp: 0, TVL: 5},
		},
	}

	result, _ := mergeCustomTVLData(map[string]*api.ProtocolTVLResponse{"slug": apiResp}, custom, nil)
	if result["slug"].Name != "ApiName" {
		t.Fatalf("expected API name preserved, got %q", result["slug"].Name)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

func findRecord(recs []slog.Record, msg string) *slog.Record {
	for i := range recs {
		if recs[i].Message == msg {
			return &recs[i]
		}
	}
	return nil
}

func attrsMap(rec slog.Record) map[string]interface{} {
	out := make(map[string]interface{})
	rec.Attrs(func(a slog.Attr) bool {
		out[a.Key] = a.Value.Any()
		return true
	})
	return out
}
