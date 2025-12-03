package aggregator

import (
	"reflect"
	"testing"

	"github.com/switchboard-xyz/defillama-extract/internal/api"
)

func TestExtractChartHistory_FiltersOracleAndSorts(t *testing.T) {
	resp := &api.OracleAPIResponse{
		Chart: map[string]map[string]map[string]float64{
			"2": {
				"Switchboard": {"tvl": 20, "borrowed": 2},
			},
			"1": {
				"Switchboard": {"tvl": 10},
			},
			"3": {
				"Other": {"tvl": 30},
			},
		},
	}

	got := ExtractChartHistory(resp, "Switchboard")

	want := []ChartDataPoint{
		{Timestamp: 1, Date: "1970-01-01", TVS: 10},
		{Timestamp: 2, Date: "1970-01-01", TVS: 20, Borrowed: 2},
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d points, got %d", len(want), len(got))
	}

	// Compare without date timezone sensitivity by zeroing dates to want's values
	for i := range got {
		if got[i].Date != want[i].Date {
			t.Fatalf("date mismatch at %d: got %s want %s", i, got[i].Date, want[i].Date)
		}
	}

	gotNormalized := make([]ChartDataPoint, len(got))
	copy(gotNormalized, got)
	if !reflect.DeepEqual(gotNormalized, want) {
		t.Fatalf("chart history mismatch\n got: %+v\nwant: %+v", gotNormalized, want)
	}
}

func TestExtractChartHistory_EmptyWhenNoData(t *testing.T) {
	cases := []struct {
		name string
		resp *api.OracleAPIResponse
	}{
		{"nil", nil},
		{"empty chart", &api.OracleAPIResponse{Chart: map[string]map[string]map[string]float64{}}},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractChartHistory(tt.resp, "Switchboard")
			if got == nil {
				t.Fatalf("expected empty slice, got nil")
			}
			if len(got) != 0 {
				t.Fatalf("expected zero entries, got %d", len(got))
			}
		})
	}
}
