package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOracleAPIResponse_UnmarshalFixture(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "testdata", "oracle_response.json")
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	var resp OracleAPIResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("failed to unmarshal fixture: %v", err)
	}

	if got := resp.Oracles["Switchboard"]; len(got) != 3 {
		t.Fatalf("expected 3 protocols for Switchboard, got %d", len(got))
	}

	if chains := resp.ChainsByOracle["Switchboard"]; len(chains) != 3 {
		t.Fatalf("expected 3 chains for Switchboard, got %d", len(chains))
	}

	if resp.Chart["Switchboard"]["Solana"]["1699574400"] != 500000000 {
		t.Fatalf("expected chart value 500000000, got %v", resp.Chart["Switchboard"]["Solana"]["1699574400"])
	}
}

func TestOracleAPIResponse_UnmarshalMissingFields(t *testing.T) {
	data := []byte(`{"oracles":{"Switchboard":["one"]}}`)

	var resp OracleAPIResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("unexpected error unmarshaling with missing fields: %v", err)
	}

	if resp.Oracles == nil || len(resp.Oracles["Switchboard"]) != 1 {
		t.Fatalf("expected oracle entry retained when optional fields missing")
	}

	if resp.Chart != nil {
		t.Fatalf("expected Chart nil when omitted")
	}
}

func TestProtocol_UnmarshalFixture(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "testdata", "protocol_response.json")
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	var protocols []Protocol
	if err := json.Unmarshal(data, &protocols); err != nil {
		t.Fatalf("failed to unmarshal fixture: %v", err)
	}

	if len(protocols) != 3 {
		t.Fatalf("expected 3 protocols, got %d", len(protocols))
	}

	first := protocols[0]
	if first.ID != "marinade-finance" || first.Category != "Liquid Staking" {
		t.Fatalf("unexpected first protocol decoded: %+v", first)
	}

	second := protocols[1]
	if second.Oracle != "Switchboard" || len(second.Oracles) != 2 {
		t.Fatalf("expected oracle fields populated, got %+v", second)
	}
}

func TestProtocol_UnmarshalMissingFields(t *testing.T) {
	data := []byte(`[
		{
			"id": "minimal-protocol",
			"name": "Minimal Protocol",
			"slug": "minimal-protocol",
			"category": "DEX"
		}
	]`)

	var protocols []Protocol
	if err := json.Unmarshal(data, &protocols); err != nil {
		t.Fatalf("unexpected error unmarshaling protocol with missing optional fields: %v", err)
	}

	if len(protocols) != 1 {
		t.Fatalf("expected one protocol decoded, got %d", len(protocols))
	}

	p := protocols[0]
	if p.TVL != 0 || p.Chains != nil || p.URL != "" || p.Oracles != nil || p.Oracle != "" {
		t.Fatalf("expected zero values for optional fields, got %+v", p)
	}
}
