package aggregator

import (
	"reflect"
	"testing"

	"github.com/switchboard-xyz/defillama-extract/internal/api"
)

func TestExtractProtocolData_PopulatesMetadataAndTVS(t *testing.T) {
	oracleResp := &api.OracleAPIResponse{
		Chart: map[string]map[string]map[string]float64{
			"1732924800": {},
		},
		OraclesTVS: map[string]map[string]map[string]float64{
			"Switchboard": {
				"1732924800": {
					"Solana": 1_000_000,
				},
			},
		},
	}

	protocols := []api.Protocol{{
		Name:     "Jupiter",
		Slug:     "jupiter",
		Category: "dex",
		URL:      "https://jup.ag",
		TVL:      2_500_000,
		Chains:   []string{"Solana"},
	}}

	got, ts := ExtractProtocolData(protocols, oracleResp, "Switchboard")

	if ts != 1732924800 {
		t.Fatalf("timestamp mismatch: got %d, want %d", ts, 1732924800)
	}

	if len(got) != 1 {
		t.Fatalf("result length = %d, want 1", len(got))
	}

	agg := got[0]
	if agg.Name != "Jupiter" || agg.Slug != "jupiter" || agg.Category != "dex" || agg.URL != "https://jup.ag" || agg.TVL != 2_500_000 {
		t.Fatalf("metadata not copied correctly: %+v", agg)
	}

	if agg.TVS != 1_000_000 {
		t.Fatalf("TVS = %f, want 1000000", agg.TVS)
	}

	wantByChain := map[string]float64{"Solana": 1_000_000}
	if !reflect.DeepEqual(agg.TVSByChain, wantByChain) {
		t.Fatalf("TVSByChain = %+v, want %+v", agg.TVSByChain, wantByChain)
	}
}

func TestExtractProtocolData_MultiChainTVS(t *testing.T) {
	oracleResp := &api.OracleAPIResponse{
		Chart: map[string]map[string]map[string]float64{"1733000000": {}},
		OraclesTVS: map[string]map[string]map[string]float64{
			"Switchboard": {
				"1733000000": {
					"Solana": 750_000,
					"Sui":    250_000,
				},
			},
		},
	}

	protocols := []api.Protocol{{
		Slug:   "multi",
		Chains: []string{"Solana", "Sui"},
	}}

	got, _ := ExtractProtocolData(protocols, oracleResp, "Switchboard")

	agg := got[0]
	if agg.TVS != 1_000_000 {
		t.Fatalf("TVS = %f, want 1000000", agg.TVS)
	}

	wantByChain := map[string]float64{"Solana": 750_000, "Sui": 250_000}
	if !reflect.DeepEqual(agg.TVSByChain, wantByChain) {
		t.Fatalf("TVSByChain = %+v, want %+v", agg.TVSByChain, wantByChain)
	}
}

func TestExtractProtocolData_HandlesMissingChains(t *testing.T) {
	oracleResp := &api.OracleAPIResponse{
		Chart:      map[string]map[string]map[string]float64{"1733000000": {}},
		OraclesTVS: map[string]map[string]map[string]float64{},
	}

	protocols := []api.Protocol{{
		Slug: "no-chains",
	}}

	got, ts := ExtractProtocolData(protocols, oracleResp, "Switchboard")

	if ts != 1733000000 {
		t.Fatalf("timestamp = %d, want 1733000000", ts)
	}

	agg := got[0]
	if agg.TVS != 0 {
		t.Fatalf("TVS = %f, want 0", agg.TVS)
	}
	if len(agg.TVSByChain) != 0 {
		t.Fatalf("TVSByChain should be empty, got %+v", agg.TVSByChain)
	}
}

func TestExtractProtocolData_MissingOracleData(t *testing.T) {
	oracleResp := &api.OracleAPIResponse{
		Chart: map[string]map[string]map[string]float64{"1733000000": {}},
		OraclesTVS: map[string]map[string]map[string]float64{
			"Other": {"1733000000": {"Solana": 10}},
		},
	}

	protocols := []api.Protocol{{
		Slug:   "no-oracle-match",
		Chains: []string{"Solana"},
	}}

	got, ts := ExtractProtocolData(protocols, oracleResp, "Switchboard")

	if ts != 1733000000 {
		t.Fatalf("timestamp = %d, want 1733000000", ts)
	}

	agg := got[0]
	if agg.TVS != 0 {
		t.Fatalf("TVS = %f, want 0", agg.TVS)
	}
	if len(agg.TVSByChain) != 0 {
		t.Fatalf("TVSByChain should be empty, got %+v", agg.TVSByChain)
	}
}

func TestExtractProtocolData_EmptyInputs(t *testing.T) {
	oracleResp := &api.OracleAPIResponse{Chart: map[string]map[string]map[string]float64{"1733000000": {}}}

	got, ts := ExtractProtocolData(nil, oracleResp, "Switchboard")
	if ts != 1733000000 {
		t.Fatalf("timestamp = %d, want 1733000000", ts)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty result, got %d items", len(got))
	}
}

func TestExtractLatestTimestamp(t *testing.T) {
	tests := []struct {
		name       string
		oracleResp *api.OracleAPIResponse
		want       int64
	}{
		{
			name: "selects max timestamp",
			oracleResp: &api.OracleAPIResponse{
				Chart: map[string]map[string]map[string]float64{
					"1732924800": {},
					"1733000000": {},
				},
			},
			want: 1733000000,
		},
		{
			name:       "nil response returns zero",
			oracleResp: nil,
			want:       0,
		},
		{
			name:       "empty chart returns zero",
			oracleResp: &api.OracleAPIResponse{Chart: map[string]map[string]map[string]float64{}},
			want:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractLatestTimestamp(tt.oracleResp)
			if got != tt.want {
				t.Fatalf("ExtractLatestTimestamp() = %d, want %d", got, tt.want)
			}
		})
	}
}
