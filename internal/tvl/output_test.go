package tvl

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/switchboard-xyz/defillama-extract/internal/api"
	"github.com/switchboard-xyz/defillama-extract/internal/models"
)

func TestMapToOutputProtocol_CustomWithDate(t *testing.T) {
	date := int64(1700000000)
	merged := models.MergedProtocol{
		Slug:            "custom-proto",
		Name:            "Custom Proto",
		Source:          "custom",
		IsOngoing:       true,
		SimpleTVSRatio:  0.5,
		IntegrationDate: &date,
	}

	tvl := &api.ProtocolTVLResponse{
		Name: "Custom Proto",
		TVL: []api.TVLDataPoint{
			{Date: 1704067200, TotalLiquidityUSD: 123.45},
		},
	}

	out := MapToOutputProtocol(merged, tvl)

	if out.IntegrationDate == nil || *out.IntegrationDate != date {
		t.Fatalf("expected integration_date %d, got %v", date, out.IntegrationDate)
	}

	if len(out.TVLHistory) != 1 {
		t.Fatalf("expected 1 history item, got %d", len(out.TVLHistory))
	}

	expectedDate := time.Unix(1704067200, 0).UTC().Format("2006-01-02")
	if out.TVLHistory[0].Date != expectedDate {
		t.Fatalf("expected date %s, got %s", expectedDate, out.TVLHistory[0].Date)
	}

	if out.CurrentTVL != 123.45 {
		t.Fatalf("expected current TVL 123.45, got %f", out.CurrentTVL)
	}
}

func TestMapToOutputProtocol_CustomWithoutDate_NullJSON(t *testing.T) {
	merged := models.MergedProtocol{
		Slug:           "custom-no-date",
		Name:           "No Date",
		Source:         "custom",
		IsOngoing:      true,
		SimpleTVSRatio: 0.7,
	}

	tvl := &api.ProtocolTVLResponse{
		Name: "No Date",
		TVL: []api.TVLDataPoint{
			{Date: 1704153600, TotalLiquidityUSD: 50},
		},
	}

	out := MapToOutputProtocol(merged, tvl)

	if out.IntegrationDate != nil {
		t.Fatalf("expected integration_date to be nil, got %v", out.IntegrationDate)
	}

	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal output: %v", err)
	}

	if !containsJSONNull(string(data), "integration_date") {
		t.Fatalf("expected integration_date to marshal as null, got %s", string(data))
	}
}

func TestMapToOutputProtocol_AutoProtocolIntegrationDateNil(t *testing.T) {
	merged := models.MergedProtocol{
		Slug:           "auto-proto",
		Name:           "Auto Proto",
		Source:         "auto",
		IsOngoing:      true,
		SimpleTVSRatio: 1.0,
	}

	out := MapToOutputProtocol(merged, nil)

	if out.IntegrationDate != nil {
		t.Fatalf("expected nil integration_date for auto protocol, got %v", out.IntegrationDate)
	}

	if len(out.TVLHistory) != 0 {
		t.Fatalf("expected empty history when no TVL data, got %d items", len(out.TVLHistory))
	}
}

func TestMapToOutputProtocol_FullHistoryPreserved(t *testing.T) {
	merged := models.MergedProtocol{Slug: "hist-proto", Source: "custom"}
	tvl := &api.ProtocolTVLResponse{
		Name: "Hist",
		TVL: []api.TVLDataPoint{
			{Date: 1704240000, TotalLiquidityUSD: 10},
			{Date: 1704326400, TotalLiquidityUSD: 20},
		},
	}

	out := MapToOutputProtocol(merged, tvl)

	if len(out.TVLHistory) != 2 {
		t.Fatalf("expected 2 history items, got %d", len(out.TVLHistory))
	}

	if out.TVLHistory[0].Timestamp != 1704240000 || out.TVLHistory[1].Timestamp != 1704326400 {
		t.Fatalf("history timestamps were altered: %#v", out.TVLHistory)
	}
}

func containsJSONNull(jsonStr, field string) bool {
	needle := "\"" + field + "\":null"
	return strings.Contains(jsonStr, needle)
}

func TestGenerateTVLOutput_MixedProtocols(t *testing.T) {
	protocols := []models.MergedProtocol{
		{Slug: "auto-1", Source: "auto", Name: "Auto One"},
		{Slug: "custom-1", Source: "custom", Name: "Custom One"},
		{Slug: "auto-2", Source: "auto", Name: "Auto Two"},
	}

	tvlData := map[string]*api.ProtocolTVLResponse{
		"auto-1": {Name: "Auto One", TVL: []api.TVLDataPoint{{Date: 1, TotalLiquidityUSD: 10}}},
	}

	out := GenerateTVLOutput(protocols, tvlData)

	if out.Version != "1.0.0" {
		t.Fatalf("version mismatch: %s", out.Version)
	}

	if out.Metadata.ProtocolCount != 3 {
		t.Fatalf("protocol_count = %d", out.Metadata.ProtocolCount)
	}

	if out.Metadata.CustomProtocolCount != 1 {
		t.Fatalf("custom_protocol_count = %d", out.Metadata.CustomProtocolCount)
	}

	if _, err := time.Parse(time.RFC3339, out.Metadata.LastUpdated); err != nil {
		t.Fatalf("last_updated not RFC3339: %v", err)
	}

	if len(out.Protocols) != 3 {
		t.Fatalf("expected 3 protocols, got %d", len(out.Protocols))
	}

	if out.Protocols["auto-1"].Name != "Auto One" {
		t.Fatalf("protocol mapping mismatch: %s", out.Protocols["auto-1"].Name)
	}
}

func TestGenerateTVLOutput_EmptyProtocols(t *testing.T) {
	out := GenerateTVLOutput(nil, nil)

	if out.Metadata.ProtocolCount != 0 || out.Metadata.CustomProtocolCount != 0 {
		t.Fatalf("expected zero counts, got %+v", out.Metadata)
	}

	if out.Protocols == nil || len(out.Protocols) != 0 {
		t.Fatalf("protocols map should be empty, got %d", len(out.Protocols))
	}
}

func TestWriteTVLOutputs_WritesFile(t *testing.T) {
	outDir := t.TempDir()
	ctx := context.Background()

	output := &models.TVLOutput{
		Version: "1.0.0",
		Metadata: models.TVLOutputMetadata{
			LastUpdated:         time.Now().UTC().Format(time.RFC3339),
			ProtocolCount:       1,
			CustomProtocolCount: 1,
		},
		Protocols: map[string]models.TVLOutputProtocol{
			"proto": {Slug: "proto", Name: "Proto"},
		},
	}

	if err := WriteTVLOutputs(ctx, outDir, output); err != nil {
		t.Fatalf("write outputs: %v", err)
	}

	fullPath := filepath.Join(outDir, "tvl-data.json")

	fullData, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("read full output: %v", err)
	}

	// Verify indented format
	if !strings.Contains(string(fullData), "\n") {
		t.Fatalf("output should be indented but has no newlines")
	}

	var parsed models.TVLOutput
	if err := json.Unmarshal(fullData, &parsed); err != nil {
		t.Fatalf("unmarshal full output: %v", err)
	}

	if parsed.Protocols["proto"].Name != output.Protocols["proto"].Name {
		t.Fatalf("content mismatch: %s", parsed.Protocols["proto"].Name)
	}
}

func TestWriteTVLOutputs_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	outDir := t.TempDir()
	output := &models.TVLOutput{Protocols: map[string]models.TVLOutputProtocol{}}

	err := WriteTVLOutputs(ctx, outDir, output)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}

	if _, err := os.Stat(filepath.Join(outDir, "tvl-data.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no files when context cancelled, got err=%v", err)
	}
}
