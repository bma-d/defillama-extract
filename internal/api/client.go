package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"log/slog"

	"github.com/switchboard-xyz/defillama-extract/internal/config"
)

const userAgentValue = "defillama-extract/1.0"

// Client wraps http.Client with configuration needed for DefiLlama requests.
type Client struct {
	httpClient   *http.Client
	oraclesURL   string
	protocolsURL string
	userAgent    string
	maxRetries   int
	retryDelay   time.Duration
	logger       *slog.Logger
}

// NewClient constructs a Client using API configuration. Nil logger falls back to slog.Default().
func NewClient(cfg *config.APIConfig, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}

	oraclesURL := cfg.OraclesURL
	if oraclesURL == "" {
		oraclesURL = OraclesEndpoint
	}

	protocolsURL := cfg.ProtocolsURL
	if protocolsURL == "" {
		protocolsURL = ProtocolsEndpoint
	}

	return &Client{
		httpClient:   &http.Client{Timeout: cfg.Timeout},
		oraclesURL:   oraclesURL,
		protocolsURL: protocolsURL,
		userAgent:    userAgentValue,
		maxRetries:   cfg.MaxRetries,
		retryDelay:   cfg.RetryDelay,
		logger:       logger,
	}
}

// doRequest performs a GET request with User-Agent injection and JSON decoding.
func (c *Client) doRequest(ctx context.Context, url string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

// FetchOracles retrieves oracle TVS data from DefiLlama /oracles endpoint.
func (c *Client) FetchOracles(ctx context.Context) (*OracleAPIResponse, error) {
	var response OracleAPIResponse
	if err := c.doRequest(ctx, c.oraclesURL, &response); err != nil {
		return nil, fmt.Errorf("fetch oracles: %w", err)
	}

	return &response, nil
}

// FetchProtocols retrieves protocol metadata from DefiLlama /lite/protocols2 endpoint.
func (c *Client) FetchProtocols(ctx context.Context) ([]Protocol, error) {
	var protocols []Protocol
	if err := c.doRequest(ctx, c.protocolsURL, &protocols); err != nil {
		return nil, fmt.Errorf("fetch protocols: %w", err)
	}

	return protocols, nil
}
