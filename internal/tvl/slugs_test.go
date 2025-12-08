package tvl

import (
	"testing"

	"github.com/switchboard-xyz/defillama-extract/internal/api"
)

func TestGetAutoDetectedSlugs(t *testing.T) {
	resp := &api.OracleAPIResponse{Oracles: map[string][]string{
		"Switchboard": {"a", "b", "a", ""},
	}}

	slugs := GetAutoDetectedSlugs(resp, "Switchboard")

	if len(slugs) != 2 || slugs[0] != "a" || slugs[1] != "b" {
		t.Fatalf("unexpected slugs: %#v", slugs)
	}
}

func TestGetAutoDetectedSlugsWithWhitespace(t *testing.T) {
	resp := &api.OracleAPIResponse{Oracles: map[string][]string{
		"Switchboard": {"Kamino Lend", "Mango Markets V4 Perps", "drift", "Kamino Lend"},
	}}

	slugs := GetAutoDetectedSlugs(resp, "Switchboard")

	// Slugs with spaces are preserved as-is (not normalized to kebab-case)
	expected := []string{"Kamino Lend", "Mango Markets V4 Perps", "drift"}
	if len(slugs) != len(expected) {
		t.Fatalf("expected %d slugs, got %d: %#v", len(expected), len(slugs), slugs)
	}

	for i, want := range expected {
		if slugs[i] != want {
			t.Errorf("slugs[%d] = %q, want %q", i, slugs[i], want)
		}
	}
}

func TestGetAutoDetectedSlugsNil(t *testing.T) {
	slugs := GetAutoDetectedSlugs(nil, "")
	if len(slugs) != 0 {
		t.Fatalf("expected empty slice, got %#v", slugs)
	}
}
