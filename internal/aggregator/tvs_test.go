package aggregator

import "testing"

func TestExtractProtocolTVS_Found(t *testing.T) {
	oraclesTVS := map[string]map[string]map[string]float64{
		"Switchboard": {
			"proto-a": {
				"Solana": 1_000,
				"Sui":    500,
			},
		},
	}

	total, byChain, found := ExtractProtocolTVS(oraclesTVS, "Switchboard", "proto-a")
	if !found {
		t.Fatalf("expected TVS to be found")
	}
	if total != 1500 {
		t.Fatalf("total = %f, want 1500", total)
	}
	if len(byChain) != 2 || byChain["Solana"] != 1000 || byChain["Sui"] != 500 {
		t.Fatalf("unexpected byChain: %+v", byChain)
	}
}

func TestExtractProtocolTVS_NotFound(t *testing.T) {
	oraclesTVS := map[string]map[string]map[string]float64{
		"Switchboard": {},
	}

	total, byChain, found := ExtractProtocolTVS(oraclesTVS, "Switchboard", "missing")
	if found || total != 0 || byChain != nil {
		t.Fatalf("expected not found result, got found=%v total=%f byChain=%v", found, total, byChain)
	}
}

func TestExtractProtocolTVS_BlankSlug(t *testing.T) {
	total, byChain, found := ExtractProtocolTVS(nil, "Switchboard", "")
	if found || total != 0 || byChain != nil {
		t.Fatalf("expected not found for blank slug, got found=%v", found)
	}
}

func TestExtractProtocolTVS_ExcludesBorrowed(t *testing.T) {
	// Borrowed amounts should be excluded entirely from TVS since they are
	// derived from deposits and would cause double-counting.
	oraclesTVS := map[string]map[string]map[string]float64{
		"Switchboard": {
			"proto-a": {
				"Solana":          757_000_000, // deposits
				"Solana-borrowed": 442_000_000, // borrowed (excluded)
				"borrowed":        442_000_000, // total borrowed (excluded)
				"Sui":             50_000_000,  // deposits
				"Sui-borrowed":    30_000_000,  // borrowed (excluded)
			},
		},
	}

	total, byChain, found := ExtractProtocolTVS(oraclesTVS, "Switchboard", "proto-a")
	if !found {
		t.Fatalf("expected TVS to be found")
	}

	// Only non-borrowed values should be counted
	expectedTotal := 757_000_000.0 + 50_000_000.0
	if total != expectedTotal {
		t.Fatalf("total = %f, want %f (borrowed should be excluded)", total, expectedTotal)
	}

	if len(byChain) != 2 {
		t.Fatalf("expected 2 chains (Solana + Sui), got %d: %+v", len(byChain), byChain)
	}

	if byChain["Solana"] != 757_000_000 {
		t.Fatalf("Solana tvs = %f, want 757000000", byChain["Solana"])
	}

	if byChain["Sui"] != 50_000_000 {
		t.Fatalf("Sui tvs = %f, want 50000000", byChain["Sui"])
	}

	// Borrowed should NOT be in byChain
	if _, exists := byChain["borrowed"]; exists {
		t.Fatalf("borrowed should not be in byChain: %+v", byChain)
	}
}

func TestExtractProtocolTVS_OnlyBorrowedReturnsNotFound(t *testing.T) {
	// If a protocol only has borrowed entries, it should return not found
	oraclesTVS := map[string]map[string]map[string]float64{
		"Switchboard": {
			"proto-a": {
				"borrowed":        100,
				"Solana-borrowed": 100,
			},
		},
	}

	total, byChain, found := ExtractProtocolTVS(oraclesTVS, "Switchboard", "proto-a")
	if found {
		t.Fatalf("expected not found when only borrowed entries exist, got total=%f byChain=%+v", total, byChain)
	}
}
