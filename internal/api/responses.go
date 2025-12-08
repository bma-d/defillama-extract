package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// OracleAPIResponse represents the payload returned by GET /oracles.
type OracleAPIResponse struct {
	Oracles        map[string][]string                      `json:"oracles"`
	Chart          map[string]map[string]map[string]float64 `json:"chart"`
	OraclesTVS     map[string]map[string]map[string]float64 `json:"oraclesTVS"`
	ChainsByOracle map[string][]string                      `json:"chainsByOracle"`
}

// Protocol represents a protocol returned by GET /lite/protocols2.
type Protocol struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Slug     string   `json:"slug"`
	Category string   `json:"category"`
	TVL      float64  `json:"tvl,omitempty"`
	Chains   []string `json:"chains,omitempty"`
	Oracles  []string `json:"oracles,omitempty"`
	Oracle   string   `json:"oracle,omitempty"`
	URL      string   `json:"url,omitempty"`
}

// ProtocolTVLResponse represents the payload from GET /protocol/{slug}.
type ProtocolTVLResponse struct {
	Name             string             `json:"name"`
	TVL              []TVLDataPoint     `json:"tvl"`
	CurrentChainTvls map[string]float64 `json:"currentChainTvls"`
}

// TVLDataPoint represents a single point in a protocol's TVL history.
type TVLDataPoint struct {
	Date              int64   `json:"date"`
	TotalLiquidityUSD float64 `json:"totalLiquidityUSD"`
}

// protocolList flexibly unmarshals either a bare array or an envelope containing
// a top-level "protocols" field. The DefiLlama endpoint has shipped both shapes
// historically, so decoding must tolerate either form.
type protocolList []Protocol

func (p *protocolList) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	if data[0] == '{' {
		var envelope struct {
			Protocols []Protocol `json:"protocols"`
		}
		if err := json.Unmarshal(data, &envelope); err != nil {
			return err
		}
		*p = envelope.Protocols
		return nil
	}

	var arr []Protocol
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}

	*p = arr
	return nil
}

// FetchResult aggregates oracle and protocol responses from parallel fetch operations.
type FetchResult struct {
	OracleResponse *OracleAPIResponse
	Protocols      []Protocol
}

// APIError represents an HTTP error response with metadata for retry decisions.
type APIError struct {
	Endpoint   string
	StatusCode int
	Message    string
	Err        error
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d on %s: %s", e.StatusCode, e.Endpoint, e.Message)
}

func (e *APIError) Unwrap() error {
	return e.Err
}

func (e *APIError) IsRetryable() bool {
	switch e.StatusCode {
	case http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}
