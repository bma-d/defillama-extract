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
