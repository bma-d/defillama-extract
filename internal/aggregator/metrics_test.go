package aggregator

import (
	"encoding/json"
	"math"
	"testing"
	"time"
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

func TestCalculatePercentageChange(t *testing.T) {
	tests := []struct {
		name     string
		oldValue float64
		newValue float64
		want     float64
	}{
		{name: "positive_increase", oldValue: 1000, newValue: 1100, want: 10},
		{name: "negative_decrease", oldValue: 1000, newValue: 900, want: -10},
		{name: "zero_old_value", oldValue: 0, newValue: 500, want: 0},
		{name: "rounding_two_decimals", oldValue: 100, newValue: 100.555, want: 0.56},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculatePercentageChange(tt.oldValue, tt.newValue)
			if !almostEqual(got, tt.want) {
				t.Fatalf("got %f, want %f", got, tt.want)
			}
		})
	}
}

func TestFindSnapshotAtTime(t *testing.T) {
	now := time.Now().Unix()
	snapshots := []Snapshot{
		{Timestamp: now - Hours24, TVS: 1000},
		{Timestamp: now - Hours24 + 3600, TVS: 900},  // within tolerance
		{Timestamp: now - Hours24 + 10800, TVS: 800}, // outside tolerance
	}

	tests := []struct {
		name       string
		targetTime int64
		tolerance  int64
		wantNil    bool
		wantTVS    float64
	}{
		{name: "exact_match", targetTime: now - Hours24, tolerance: SnapshotTolerance, wantTVS: 1000},
		{name: "within_tolerance_picks_closest", targetTime: now - Hours24 + 1800, tolerance: SnapshotTolerance, wantTVS: 1000},
		{name: "outside_tolerance_returns_nil", targetTime: now - Hours24 - 10800, tolerance: SnapshotTolerance, wantNil: true},
		{name: "empty_history", targetTime: now - Hours24, tolerance: SnapshotTolerance, wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input []Snapshot
			if tt.name != "empty_history" {
				input = snapshots
			}
			got := FindSnapshotAtTime(input, tt.targetTime, tt.tolerance)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %+v", got)
				}
				return
			}

			if got == nil {
				t.Fatalf("expected snapshot, got nil")
			}
			if !almostEqual(got.TVS, tt.wantTVS) {
				t.Fatalf("got TVS %f, want %f", got.TVS, tt.wantTVS)
			}
		})
	}
}

func TestCalculateChangeMetrics(t *testing.T) {
	now := time.Now().Unix()
	buildSnapshot := func(offset int64, tvs float64, protocolCount int) Snapshot {
		return Snapshot{
			Timestamp:     now - offset,
			TVS:           tvs,
			ProtocolCount: protocolCount,
		}
	}

	currentTVS := 1100.0
	currentProtocols := 120
	history := []Snapshot{
		buildSnapshot(Hours24, 1000, 110),
		buildSnapshot(Days7, 900, 100),
		buildSnapshot(Days30, 1000, 90),
	}

	t.Run("full_history", func(t *testing.T) {
		metrics := CalculateChangeMetrics(currentTVS, currentProtocols, history)

		assertFloatPtr(t, metrics.Change24h, 10)
		assertFloatPtr(t, metrics.Change7d, 22.22) // (1100-900)/900*100 = 22.22...
		assertFloatPtr(t, metrics.Change30d, 10)
		assertIntPtr(t, metrics.ProtocolCountChange7d, 20)
		assertIntPtr(t, metrics.ProtocolCountChange30d, 30)
	})

	t.Run("partial_history", func(t *testing.T) {
		partialHistory := []Snapshot{buildSnapshot(Hours24, 1000, 110)}
		metrics := CalculateChangeMetrics(currentTVS, currentProtocols, partialHistory)

		assertFloatPtr(t, metrics.Change24h, 10)
		if metrics.Change7d != nil || metrics.Change30d != nil {
			t.Fatalf("expected nil changes for missing periods")
		}
		if metrics.ProtocolCountChange7d != nil || metrics.ProtocolCountChange30d != nil {
			t.Fatalf("expected nil protocol count changes for missing periods")
		}
	})

	t.Run("empty_history", func(t *testing.T) {
		metrics := CalculateChangeMetrics(currentTVS, currentProtocols, nil)
		if metrics.Change24h != nil || metrics.Change7d != nil || metrics.Change30d != nil {
			t.Fatalf("expected nil change pointers when history empty")
		}
		if metrics.ProtocolCountChange7d != nil || metrics.ProtocolCountChange30d != nil {
			t.Fatalf("expected nil protocol count change pointers when history empty")
		}
	})

	t.Run("division_by_zero_guard", func(t *testing.T) {
		h := []Snapshot{buildSnapshot(Hours24, 0, 50)}
		metrics := CalculateChangeMetrics(currentTVS, currentProtocols, h)
		assertFloatPtr(t, metrics.Change24h, 0)
	})

	t.Run("negative_change_seven_day", func(t *testing.T) {
		current := 900.0
		negHistory := []Snapshot{
			buildSnapshot(Days7, 1000, 95),
		}

		metrics := CalculateChangeMetrics(current, currentProtocols, negHistory)
		assertFloatPtr(t, metrics.Change7d, -10)
		if metrics.Change24h != nil || metrics.Change30d != nil {
			t.Fatalf("expected only 7d change populated for available history")
		}
	})
}

func TestChangeMetricsJSONSerialization(t *testing.T) {
	change := 10.0
	count := 5

	metrics := ChangeMetrics{
		Change24h:             &change,
		ProtocolCountChange7d: &count,
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded["change_24h"] != change {
		t.Fatalf("change_24h missing or incorrect: %v", decoded["change_24h"])
	}
	if decoded["protocol_count_change_7d"] != float64(count) {
		t.Fatalf("protocol_count_change_7d missing or incorrect: %v", decoded["protocol_count_change_7d"])
	}

	if _, ok := decoded["change_7d"]; ok {
		t.Fatalf("expected change_7d to be omitted when nil")
	}
	if _, ok := decoded["change_30d"]; ok {
		t.Fatalf("expected change_30d to be omitted when nil")
	}
	if _, ok := decoded["protocol_count_change_30d"]; ok {
		t.Fatalf("expected protocol_count_change_30d to be omitted when nil")
	}
}

func assertFloatPtr(t *testing.T, got *float64, want float64) {
	t.Helper()
	if got == nil {
		t.Fatalf("nil float pointer, want %f", want)
	}
	if !almostEqual(*got, want) {
		t.Fatalf("value %f, want %f", *got, want)
	}
}

func assertIntPtr(t *testing.T, got *int, want int) {
	t.Helper()
	if got == nil {
		t.Fatalf("nil int pointer, want %d", want)
	}
	if *got != want {
		t.Fatalf("value %d, want %d", *got, want)
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
