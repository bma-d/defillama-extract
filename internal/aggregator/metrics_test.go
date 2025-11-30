package aggregator

import (
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
