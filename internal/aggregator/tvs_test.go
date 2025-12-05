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

func TestExtractProtocolTVS_NormalizesBorrowed(t *testing.T) {
	oraclesTVS := map[string]map[string]map[string]float64{
		"Switchboard": {
			"proto-a": {
				"Solana":          50,
				"Solana-borrowed": 100,
				"borrowed":        100,
				"Sui-borrowed":    100,
			},
		},
	}

	total, byChain, found := ExtractProtocolTVS(oraclesTVS, "Switchboard", "proto-a")
	if !found {
		t.Fatalf("expected TVS to be found")
	}

	if total != 150 {
		t.Fatalf("total = %f, want 150", total)
	}

	if len(byChain) != 2 {
		t.Fatalf("expected 2 chains (Solana + borrowed), got %d: %+v", len(byChain), byChain)
	}

	if byChain["Solana"] != 50 {
		t.Fatalf("Solana tvs = %f, want 50", byChain["Solana"])
	}

	if byChain["borrowed"] != 100 {
		t.Fatalf("borrowed tvs = %f, want 100", byChain["borrowed"])
	}
}
