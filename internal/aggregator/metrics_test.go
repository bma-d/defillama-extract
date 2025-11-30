package aggregator

import (
	"encoding/json"
	"math"
	"testing"
)

func TestCalculateChainBreakdown(t *testing.T) {
	tests := []struct {
		name           string
		protocols      []AggregatedProtocol
		wantChains     int
		wantFirstChain string
		wantFirstTVS   float64
		wantFirstPct   float64
	}{
		{
			name: "percentages_and_sorting",
			protocols: []AggregatedProtocol{
				{TVSByChain: map[string]float64{"solana": 500}},
				{TVSByChain: map[string]float64{"sui": 300}},
				{TVSByChain: map[string]float64{"aptos": 200}},
			},
			wantChains:     3,
			wantFirstChain: "solana",
			wantFirstTVS:   500,
			wantFirstPct:   50,
		},
		{
			name: "multichain_protocol_counts",
			protocols: []AggregatedProtocol{
				{TVSByChain: map[string]float64{"solana": 100, "aptos": 50}},
				{TVSByChain: map[string]float64{"solana": 200}},
				{TVSByChain: map[string]float64{"aptos": 0}},
			},
			wantChains:     2,
			wantFirstChain: "solana",
			wantFirstTVS:   300,
			wantFirstPct:   0, // not asserting percentage exactly here
		},
		{
			name:       "zero_total_tvs_returns_empty",
			protocols:  []AggregatedProtocol{{TVSByChain: map[string]float64{"solana": 0}}},
			wantChains: 0,
		},
		{
			name:       "empty_input",
			protocols:  nil,
			wantChains: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateChainBreakdown(tt.protocols)

			if len(got) != tt.wantChains {
				t.Fatalf("got %d chains, want %d", len(got), tt.wantChains)
			}

			if tt.wantChains == 0 {
				return
			}

			if got[0].Chain != tt.wantFirstChain {
				t.Fatalf("first chain %s, want %s", got[0].Chain, tt.wantFirstChain)
			}

			if !almostEqual(got[0].TVS, tt.wantFirstTVS) {
				t.Fatalf("first chain TVS %f, want %f", got[0].TVS, tt.wantFirstTVS)
			}

			if tt.wantFirstPct > 0 && !almostEqual(got[0].Percentage, tt.wantFirstPct) {
				t.Fatalf("first chain pct %f, want %f", got[0].Percentage, tt.wantFirstPct)
			}

			if tt.name == "percentages_and_sorting" {
				total := sumTVS(got)
				if !almostEqual(total, 1000) {
					t.Fatalf("total tvs %f, want 1000", total)
				}

				pct := got[0].Percentage
				if !almostEqual(pct, 50) {
					t.Fatalf("solana pct %f, want 50", pct)
				}
				if !almostEqual(got[1].Percentage, 30) {
					t.Fatalf("sui pct %f, want 30", got[1].Percentage)
				}
				if !almostEqual(got[2].Percentage, 20) {
					t.Fatalf("aptos pct %f, want 20", got[2].Percentage)
				}
			}

			if tt.name == "multichain_protocol_counts" {
				if got[0].ProtocolCount != 2 {
					t.Fatalf("solana protocol count %d, want 2", got[0].ProtocolCount)
				}

				var aptosCount int
				for _, item := range got {
					if item.Chain == "aptos" {
						aptosCount = item.ProtocolCount
					}
				}
				if aptosCount != 1 {
					t.Fatalf("aptos protocol count %d, want 1", aptosCount)
				}
			}
		})
	}
}

func TestCalculateCategoryBreakdown(t *testing.T) {
	tests := []struct {
		name                   string
		protocols              []AggregatedProtocol
		wantCategories         int
		wantFirstCategory      string
		wantFirstTVS           float64
		wantFirstPct           float64
		wantUncategorized      bool
		wantUncategorizedCount int
	}{
		{
			name: "percentages_and_sorting",
			protocols: []AggregatedProtocol{
				{Category: "Lending", TVS: 300},
				{Category: "Lending", TVS: 300},
				{Category: "CDP", TVS: 200},
				{Category: "CDP", TVS: 100},
				{Category: "Dexes", TVS: 100},
			},
			wantCategories:    3,
			wantFirstCategory: "Lending",
			wantFirstTVS:      600,
			wantFirstPct:      60,
		},
		{
			name: "uncategorized_and_counts",
			protocols: []AggregatedProtocol{
				{Category: "", TVS: 25},
				{Category: "Lending", TVS: 100},
				{Category: "", TVS: 25},
			},
			wantCategories:         2,
			wantFirstCategory:      "Lending",
			wantFirstTVS:           100,
			wantFirstPct:           -1,
			wantUncategorized:      true,
			wantUncategorizedCount: 2,
		},
		{
			name: "zero_total_tvs",
			protocols: []AggregatedProtocol{
				{Category: "Lending", TVS: 0},
			},
			wantCategories:    1,
			wantFirstCategory: "Lending",
			wantFirstTVS:      0,
			wantFirstPct:      0,
		},
		{
			name:           "empty_input",
			protocols:      nil,
			wantCategories: 0,
			wantFirstPct:   -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateCategoryBreakdown(tt.protocols)

			if len(got) != tt.wantCategories {
				t.Fatalf("got %d categories, want %d", len(got), tt.wantCategories)
			}

			if tt.wantCategories == 0 {
				return
			}

			if got[0].Category != tt.wantFirstCategory {
				t.Fatalf("first category %s, want %s", got[0].Category, tt.wantFirstCategory)
			}

			if !almostEqual(got[0].TVS, tt.wantFirstTVS) {
				t.Fatalf("first category TVS %f, want %f", got[0].TVS, tt.wantFirstTVS)
			}

			if tt.wantFirstPct >= 0 && !almostEqual(got[0].Percentage, tt.wantFirstPct) {
				t.Fatalf("first category pct %f, want %f", got[0].Percentage, tt.wantFirstPct)
			}

			if tt.name == "percentages_and_sorting" {
				pct := got[0].Percentage
				if !almostEqual(pct, 60) {
					t.Fatalf("lending pct %f, want 60", pct)
				}

				if !almostEqual(got[1].Percentage, 30) {
					t.Fatalf("cdp pct %f, want 30", got[1].Percentage)
				}

				if !almostEqual(got[2].Percentage, 10) {
					t.Fatalf("dexes pct %f, want 10", got[2].Percentage)
				}
			}

			if tt.wantUncategorized {
				var count int
				for _, item := range got {
					if item.Category == "Uncategorized" {
						count = item.ProtocolCount
					}
				}
				if count != tt.wantUncategorizedCount {
					t.Fatalf("uncategorized protocol count %d, want %d", count, tt.wantUncategorizedCount)
				}
			}
		})
	}
}

func TestRankProtocols(t *testing.T) {
	tests := []struct {
		name        string
		protocols   []AggregatedProtocol
		wantOrder   []string
		wantRanks   []int
		wantEmpty   bool
		expectInput []int
	}{
		{
			name: "sorts_by_tvl_and_assigns_rank",
			protocols: []AggregatedProtocol{
				{Name: "Medium", TVL: 300},
				{Name: "Large", TVL: 900},
				{Name: "Small", TVL: 100},
			},
			wantOrder:   []string{"Large", "Medium", "Small"},
			wantRanks:   []int{1, 2, 3},
			expectInput: []int{0, 0, 0},
		},
		{
			name: "tiebreaker_alphabetical",
			protocols: []AggregatedProtocol{
				{Name: "Zeta", TVL: 500},
				{Name: "Alpha", TVL: 500},
			},
			wantOrder:   []string{"Alpha", "Zeta"},
			wantRanks:   []int{1, 2},
			expectInput: []int{0, 0},
		},
		{
			name:      "empty_input",
			protocols: nil,
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := append([]AggregatedProtocol(nil), tt.protocols...)

			got := RankProtocols(tt.protocols)

			if tt.wantEmpty {
				if len(got) != 0 {
					t.Fatalf("got %d items, want 0", len(got))
				}
				return
			}

			if len(got) != len(tt.wantOrder) {
				t.Fatalf("got %d items, want %d", len(got), len(tt.wantOrder))
			}

			for i, name := range tt.wantOrder {
				if got[i].Name != name {
					t.Fatalf("index %d name %s, want %s", i, got[i].Name, name)
				}
				if got[i].Rank != tt.wantRanks[i] {
					t.Fatalf("index %d rank %d, want %d", i, got[i].Rank, tt.wantRanks[i])
				}
			}

			for i := range original {
				if original[i].Rank != tt.expectInput[i] {
					t.Fatalf("input mutated at %d: rank %d, want %d", i, original[i].Rank, tt.expectInput[i])
				}
			}
		})
	}
}

func TestGetLargestProtocol(t *testing.T) {
	protocols := []AggregatedProtocol{
		{Name: "Beta", Slug: "beta", TVL: 200, TVS: 20},
		{Name: "Alpha", Slug: "alpha", TVL: 500, TVS: 50},
		{Name: "Zeta", Slug: "zeta", TVL: 500, TVS: 40},
	}

	got := GetLargestProtocol(protocols)
	if got == nil {
		t.Fatalf("expected largest protocol, got nil")
	}

	if got.Name != "Alpha" || got.Slug != "alpha" {
		t.Fatalf("got %s, want Alpha", got.Name)
	}
	if !almostEqual(got.TVL, 500) || !almostEqual(got.TVS, 50) {
		t.Fatalf("unexpected TVL/TVS: %f/%f", got.TVL, got.TVS)
	}

	if GetLargestProtocol(nil) != nil {
		t.Fatalf("expected nil for empty input")
	}
}

func TestRankProtocolsJSONSerialization(t *testing.T) {
	protocols := []AggregatedProtocol{
		{Name: "A", Slug: "a", TVL: 10},
		{Name: "B", Slug: "b", TVL: 5},
	}

	ranked := RankProtocols(protocols)
	data, err := json.Marshal(ranked[0])
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded["rank"] != float64(1) {
		t.Fatalf("rank field missing or incorrect: %v", decoded["rank"])
	}
}
func sumTVS(items []ChainBreakdown) float64 {
	var total float64
	for _, item := range items {
		total += item.TVS
	}
	return total
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-6
}
