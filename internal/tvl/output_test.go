package tvl

import (
	"encoding/json"
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
