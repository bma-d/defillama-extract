package tvl

import (
	"testing"

	"github.com/switchboard-xyz/defillama-extract/internal/api"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"Kamino Lend", "kamino-lend"},
		{"Jito Liquid Staking", "jito-liquid-staking"},
		{"Drift Trade", "drift-trade"},
		{"marginfi Lending", "marginfi-lending"},
		{"Rain.fi", "rain.fi"},
		{"Mango Markets V4 Perps", "mango-markets-v4-perps"},
		{"Aptin Finance V1", "aptin-finance-v1"},
		{"  Spaced  Name  ", "spaced-name"},
		{"", ""},
		{"UPPERCASE", "uppercase"},
		{"Special!@#Chars", "specialchars"},
	}

	for _, tt := range tests {
		got := slugify(tt.name)
		if got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestGetAutoDetectedSlugs(t *testing.T) {
	// Test with explicit slugs
	protocols := []api.Protocol{
		{Name: "Protocol A", Slug: "a", Oracles: []string{"Switchboard"}},
		{Name: "Protocol B", Slug: "b", Oracles: []string{"Switchboard", "Chainlink"}},
		{Name: "Protocol A Dup", Slug: "a", Oracles: []string{"Switchboard"}}, // duplicate slug
		{Name: "No Slug", Slug: "", Oracles: []string{"Switchboard"}},         // derives slug from name
		{Name: "Other", Slug: "c", Oracles: []string{"Chainlink"}},            // different oracle
	}

	slugs := GetAutoDetectedSlugs(protocols, "Switchboard")

	// Should get: a, b, no-slug (derived from name)
	if len(slugs) != 3 {
		t.Fatalf("expected 3 slugs, got %d: %#v", len(slugs), slugs)
	}
	if slugs[0] != "a" || slugs[1] != "b" || slugs[2] != "no-slug" {
		t.Errorf("unexpected slugs: %#v", slugs)
	}
}

func TestGetAutoDetectedSlugsDerivedFromNames(t *testing.T) {
	// Test deriving slugs from names (like /lite/protocols2 which has empty slugs)
	protocols := []api.Protocol{
		{Name: "Kamino Lend", Slug: "", Oracles: []string{"Switchboard"}},
		{Name: "Jito Liquid Staking", Slug: "", Oracles: []string{"Switchboard"}},
		{Name: "Drift Trade", Slug: "", Oracles: []string{"Switchboard"}},
		{Name: "marginfi Lending", Slug: "", Oracles: []string{"Switchboard"}},
		{Name: "Other Protocol", Slug: "", Oracles: []string{"Chainlink"}}, // different oracle
	}

	slugs := GetAutoDetectedSlugs(protocols, "Switchboard")

	expected := []string{"drift-trade", "jito-liquid-staking", "kamino-lend", "marginfi-lending"}
	if len(slugs) != len(expected) {
		t.Fatalf("expected %d slugs, got %d: %#v", len(expected), len(slugs), slugs)
	}
	for i, want := range expected {
		if slugs[i] != want {
			t.Errorf("slugs[%d] = %q, want %q", i, slugs[i], want)
		}
	}
}

func TestGetAutoDetectedSlugsWithSingleOracleField(t *testing.T) {
	// Some protocols use Oracle (singular) instead of Oracles array
	protocols := []api.Protocol{
		{Name: "Kamino", Slug: "kamino-lend", Oracle: "Switchboard"},
		{Name: "Drift", Slug: "drift", Oracle: "switchboard"}, // case insensitive
		{Name: "Other", Slug: "other", Oracle: "Chainlink"},
	}

	slugs := GetAutoDetectedSlugs(protocols, "Switchboard")

	if len(slugs) != 2 {
		t.Fatalf("expected 2 slugs, got %d: %#v", len(slugs), slugs)
	}
	if slugs[0] != "drift" || slugs[1] != "kamino-lend" {
		t.Errorf("unexpected slugs: %#v", slugs)
	}
}

func TestGetAutoDetectedSlugsWithMixedOracleFields(t *testing.T) {
	// Test protocols with both Oracle and Oracles fields
	protocols := []api.Protocol{
		{Name: "Jito", Slug: "jito", Oracles: []string{"Switchboard"}},
		{Name: "Save", Slug: "save", Oracle: "Switchboard"},
		{Name: "marginfi", Slug: "marginfi", Oracles: []string{"Pyth", "Switchboard"}},
	}

	slugs := GetAutoDetectedSlugs(protocols, "Switchboard")

	expected := []string{"jito", "marginfi", "save"}
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

func TestGetAutoDetectedSlugsEmptyOracleName(t *testing.T) {
	protocols := []api.Protocol{
		{Name: "Test", Slug: "test", Oracles: []string{"Switchboard"}},
	}
	slugs := GetAutoDetectedSlugs(protocols, "")
	if len(slugs) != 0 {
		t.Fatalf("expected empty slice for empty oracle name, got %#v", slugs)
	}
}
