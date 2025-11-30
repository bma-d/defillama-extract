package aggregator

import (
	"fmt"
	"testing"

	"github.com/switchboard-xyz/defillama-extract/internal/api"
)

func TestFilterByOracle(t *testing.T) {
	tests := []struct {
		name       string
		protocols  []api.Protocol
		oracleName string
		wantCount  int
		wantSlugs  []string
	}{
		{
			name: "includes protocol when oracle in Oracles slice",
			protocols: []api.Protocol{
				{Slug: "switchboard-only", Oracles: []string{"Switchboard"}},
			},
			oracleName: "Switchboard",
			wantCount:  1,
			wantSlugs:  []string{"switchboard-only"},
		},
		{
			name: "includes multi-oracle protocol",
			protocols: []api.Protocol{
				{Slug: "multi", Oracles: []string{"Chainlink", "Switchboard"}},
			},
			oracleName: "Switchboard",
			wantCount:  1,
			wantSlugs:  []string{"multi"},
		},
		{
			name: "includes legacy oracle field",
			protocols: []api.Protocol{
				{Slug: "legacy", Oracle: "Switchboard"},
			},
			oracleName: "Switchboard",
			wantCount:  1,
			wantSlugs:  []string{"legacy"},
		},
		{
			name: "excludes non-matching protocol",
			protocols: []api.Protocol{
				{Slug: "other", Oracles: []string{"Chainlink"}},
			},
			oracleName: "Switchboard",
			wantCount:  0,
		},
		{
			name: "enforces case sensitivity",
			protocols: []api.Protocol{
				{Slug: "lower", Oracles: []string{"switchboard"}},
			},
			oracleName: "Switchboard",
			wantCount:  0,
		},
		{
			name:       "empty input returns empty slice",
			protocols:  nil,
			oracleName: "Switchboard",
			wantCount:  0,
		},
		{
			name: "empty oracle name returns empty slice",
			protocols: []api.Protocol{
				{Slug: "noop", Oracles: []string{"Switchboard"}},
			},
			oracleName: "",
			wantCount:  0,
		},
		{
			name:       "realistic dataset returns expected count",
			protocols:  buildProtocolDataset(),
			oracleName: "Switchboard",
			wantCount:  21,
			wantSlugs: []string{
				"switchboard-0", "switchboard-1", "switchboard-2", "switchboard-3", "switchboard-4",
				"switchboard-5", "switchboard-6", "switchboard-7", "switchboard-8", "switchboard-9",
				"multi-10", "multi-11", "multi-12", "multi-13", "multi-14",
				"legacy-15", "legacy-16", "legacy-17", "legacy-18", "legacy-19", "legacy-20",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterByOracle(tt.protocols, tt.oracleName)

			if len(got) != tt.wantCount {
				t.Fatalf("got %d protocols, want %d", len(got), tt.wantCount)
			}

			if len(tt.wantSlugs) > 0 {
				if err := assertContainsSlugs(got, tt.wantSlugs); err != nil {
					t.Fatalf("%v", err)
				}
			}
		})
	}
}

func buildProtocolDataset() []api.Protocol {
	protocols := make([]api.Protocol, 0, 500)
	for i := 0; i < 500; i++ {
		protocol := api.Protocol{
			Slug:    fmt.Sprintf("protocol-%d", i),
			Oracles: []string{"Chainlink"},
		}

		switch {
		case i < 10:
			protocol.Slug = fmt.Sprintf("switchboard-%d", i)
			protocol.Oracles = []string{"Switchboard"}
		case i >= 10 && i < 15:
			protocol.Slug = fmt.Sprintf("multi-%d", i)
			protocol.Oracles = []string{"Chainlink", "Switchboard"}
		case i >= 15 && i < 21:
			protocol.Slug = fmt.Sprintf("legacy-%d", i)
			protocol.Oracles = nil
			protocol.Oracle = "Switchboard"
		}

		protocols = append(protocols, protocol)
	}

	return protocols
}

func assertContainsSlugs(protocols []api.Protocol, want []string) error {
	index := make(map[string]struct{}, len(protocols))
	for _, p := range protocols {
		index[p.Slug] = struct{}{}
	}

	for _, slug := range want {
		if _, ok := index[slug]; !ok {
			return fmt.Errorf("protocol with slug %s not found", slug)
		}
	}

	return nil
}
