package api

import (
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
