package aggregator

import (
	"sort"
	"strconv"
	"time"

	"github.com/switchboard-xyz/defillama-extract/internal/api"
)

// ChartDataPoint represents a single historical TVS datapoint sourced from DefiLlama chart data.
type ChartDataPoint struct {
	Timestamp int64   `json:"timestamp"`
	Date      string  `json:"date"`
	TVS       float64 `json:"tvs"`
	Borrowed  float64 `json:"borrowed,omitempty"`
	Staking   float64 `json:"staking,omitempty"`
}

// ExtractChartHistory converts oracle chart data into a sorted slice of ChartDataPoint for the given oracle.
// It returns an empty slice when no chart data is available; never nil.
func ExtractChartHistory(oracleResp *api.OracleAPIResponse, oracleName string) []ChartDataPoint {
	if oracleResp == nil || len(oracleResp.Chart) == 0 {
		return []ChartDataPoint{}
	}

	points := make([]ChartDataPoint, 0, len(oracleResp.Chart))
	for tsStr, oracles := range oracleResp.Chart {
		ts, err := strconv.ParseInt(tsStr, 10, 64)
		if err != nil {
			continue
		}

		oracleEntry, ok := oracles[oracleName]
		if !ok {
			continue
		}

		tvs := oracleEntry["tvl"]
		borrowed := oracleEntry["borrowed"]
		staking := oracleEntry["staking"]

		points = append(points, ChartDataPoint{
			Timestamp: ts,
			Date:      time.Unix(ts, 0).UTC().Format("2006-01-02"),
			TVS:       tvs,
			Borrowed:  borrowed,
			Staking:   staking,
		})
	}

	sort.Slice(points, func(i, j int) bool {
		return points[i].Timestamp < points[j].Timestamp
	})

	return points
}
