package aggregator

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/switchboard-xyz/defillama-extract/internal/api"
)

func TestNewAggregator(t *testing.T) {
	agg := NewAggregator("Switchboard")
	if agg.oracleName != "Switchboard" {
		t.Fatalf("oracleName = %s, want Switchboard", agg.oracleName)
	}
}

func TestAggregate_CompletesPipeline(t *testing.T) {
	ctx := context.Background()
	timestamp := int64(1735689600) // 2025-12-01T00:00:00Z
	tsStr := "1735689600"

	oracleResp := &api.OracleAPIResponse{
		Chart: map[string]map[string]map[string]float64{
			tsStr: {},
		},
		OraclesTVS: map[string]map[string]map[string]float64{
			"Switchboard": {
				"proto-a": {
					"Solana": 1000,
					"Sui":    500,
				},
				"proto-b": {
					"Aptos": 250,
				},
			},
		},
	}

	protocols := []api.Protocol{
		{
			Name:     "ProtoA",
			Slug:     "proto-a",
			Category: "Lending",
			URL:      "https://a.example",
			TVL:      2000,
			Chains:   []string{"Solana", "Sui"},
			Oracles:  []string{"Switchboard"},
		},
		{
			Name:     "ProtoB",
			Slug:     "proto-b",
			Category: "Dexes",
			URL:      "https://b.example",
			TVL:      1500,
			Chains:   []string{"Aptos"},
			Oracle:   "Switchboard", // legacy field
		},
		{
			Name:    "IgnoreMe",
			Slug:    "ignore",
			TVL:     999,
			Chains:  []string{"Solana"},
			Oracles: []string{"Other"},
		},
	}

	now := time.Now().Unix()
	history := []Snapshot{
		{Timestamp: now - Hours24, TVS: 1500, ProtocolCount: 2},
		{Timestamp: now - Days7, TVS: 1000, ProtocolCount: 1},
	}

	agg := NewAggregator("Switchboard")
	result := agg.Aggregate(ctx, oracleResp, protocols, history)

	if result.TotalProtocols != 2 {
		t.Fatalf("TotalProtocols = %d, want 2", result.TotalProtocols)
	}

	if result.ProtocolsWithTVS != 2 || result.ProtocolsWithoutTVS != 0 {
		t.Fatalf("unexpected TVS counts: with=%d without=%d", result.ProtocolsWithTVS, result.ProtocolsWithoutTVS)
	}

	if !almostEqual(result.TotalTVS, 1750) {
		t.Fatalf("TotalTVS = %f, want 1750", result.TotalTVS)
	}

	wantChains := []string{"Aptos", "Solana", "Sui"}
	if !reflect.DeepEqual(result.ActiveChains, wantChains) {
		t.Fatalf("ActiveChains = %v, want %v", result.ActiveChains, wantChains)
	}

	wantCategories := []string{"Dexes", "Lending"}
	if !reflect.DeepEqual(result.Categories, wantCategories) {
		t.Fatalf("Categories = %v, want %v", result.Categories, wantCategories)
	}

	if len(result.ChainBreakdown) != 3 {
		t.Fatalf("ChainBreakdown length = %d, want 3", len(result.ChainBreakdown))
	}
	if result.ChainBreakdown[0].Chain != "Solana" || !almostEqual(result.ChainBreakdown[0].TVS, 1000) {
		t.Fatalf("first chain %+v, want Solana/1000", result.ChainBreakdown[0])
	}

	if len(result.CategoryBreakdown) != 2 {
		t.Fatalf("CategoryBreakdown length = %d, want 2", len(result.CategoryBreakdown))
	}
	if result.CategoryBreakdown[0].Category != "Lending" || !almostEqual(result.CategoryBreakdown[0].TVS, 1500) {
		t.Fatalf("first category %+v, want Lending/1500", result.CategoryBreakdown[0])
	}

	if len(result.Protocols) != 2 {
		t.Fatalf("Protocols length = %d, want 2", len(result.Protocols))
	}
	if result.Protocols[0].Name != "ProtoA" || result.Protocols[0].Rank != 1 {
		t.Fatalf("protocol[0] %+v, want ProtoA rank 1", result.Protocols[0])
	}
	if result.Protocols[1].Name != "ProtoB" || result.Protocols[1].Rank != 2 {
		t.Fatalf("protocol[1] %+v, want ProtoB rank 2", result.Protocols[1])
	}

	if result.LargestProtocol == nil || result.LargestProtocol.Name != "ProtoA" {
		t.Fatalf("LargestProtocol %+v, want ProtoA", result.LargestProtocol)
	}

	assertFloatPtr(t, result.ChangeMetrics.Change24h, 16.67)
	assertFloatPtr(t, result.ChangeMetrics.Change7d, 75)
	assertIntPtr(t, result.ChangeMetrics.ProtocolCountChange7d, 1)

	if result.Timestamp != timestamp {
		t.Fatalf("Timestamp = %d, want %d", result.Timestamp, timestamp)
	}
}

func TestAggregate_GracefulOnEmptyInputs(t *testing.T) {
	agg := NewAggregator("Switchboard")
	result := agg.Aggregate(context.Background(), nil, nil, nil)

	if result.TotalTVS != 0 || result.TotalProtocols != 0 {
		t.Fatalf("unexpected totals: tvs=%f protocols=%d", result.TotalTVS, result.TotalProtocols)
	}
	if len(result.ActiveChains) != 0 || len(result.Categories) != 0 {
		t.Fatalf("expected empty slices, got chains=%v categories=%v", result.ActiveChains, result.Categories)
	}
	if len(result.ChainBreakdown) != 0 || len(result.CategoryBreakdown) != 0 || len(result.Protocols) != 0 {
		t.Fatalf("expected empty breakdowns/protocols, got %+v %+v %+v", result.ChainBreakdown, result.CategoryBreakdown, result.Protocols)
	}
	if result.LargestProtocol != nil {
		t.Fatalf("LargestProtocol should be nil, got %+v", result.LargestProtocol)
	}
	if result.ChangeMetrics.Change24h != nil || result.ChangeMetrics.Change7d != nil || result.ChangeMetrics.Change30d != nil {
		t.Fatalf("expected nil change metrics, got %+v", result.ChangeMetrics)
	}
	if result.Timestamp != 0 {
		t.Fatalf("Timestamp = %d, want 0", result.Timestamp)
	}
}
