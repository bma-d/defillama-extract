package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"sync"
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
	rng          *rand.Rand
	rngMu        sync.Mutex
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
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
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
		return &APIError{
			Endpoint:   url,
			StatusCode: 0,
			Message:    fmt.Sprintf("execute request: %v", err),
			Err:        err,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return &APIError{
			Endpoint:   url,
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("unexpected status: %d", resp.StatusCode),
			Err:        fmt.Errorf("unexpected status: %d", resp.StatusCode),
		}
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

func isRetryable(statusCode int, err error) bool {
	if err == nil {
		return false
	}

	switch statusCode {
	case http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound:
		return false
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.IsRetryable()
	}

	if errors.Is(err, context.Canceled) {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	var opErr *net.OpError
	return errors.As(err, &opErr)
}

func (c *Client) calculateBackoff(attempt int, baseDelay time.Duration) time.Duration {
	exponential := baseDelay * time.Duration(1<<attempt)
	c.rngMu.Lock()
	jitterMultiplier := 0.75 + c.rng.Float64()*0.5
	c.rngMu.Unlock()
	return time.Duration(float64(exponential) * jitterMultiplier)
}

func (c *Client) doWithRetry(ctx context.Context, fn func() error) error {
	maxAttempts := c.maxRetries + 1
	var lastErr error
	var lastEndpoint string

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		err := fn()
		if err == nil {
			if attempt > 0 {
				c.logger.Info("request succeeded after retries",
					"url", lastEndpoint,
					"attempts", attempt+1,
				)
			}
			return nil
		}

		lastErr = err
		statusCode := 0

		var apiErr *APIError
		if errors.As(err, &apiErr) {
			statusCode = apiErr.StatusCode
			lastEndpoint = apiErr.Endpoint
		}

		if errors.Is(err, context.Canceled) {
			return err
		}

		if !isRetryable(statusCode, err) || attempt == c.maxRetries {
			if attempt == c.maxRetries {
				c.logger.Error("max retries exceeded",
					"url", lastEndpoint,
					"total_attempts", attempt+1,
					"final_error", err,
				)
			}
			return err
		}

		backoff := c.calculateBackoff(attempt, c.retryDelay)
		c.logger.Warn("retrying API request",
			"url", lastEndpoint,
			"attempt", attempt+1,
			"max_attempts", maxAttempts,
			"backoff_ms", backoff.Milliseconds(),
			"error", err,
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}

	return lastErr
}

// FetchOracles retrieves oracle TVS data from DefiLlama /oracles endpoint.
func (c *Client) FetchOracles(ctx context.Context) (*OracleAPIResponse, error) {
	var response OracleAPIResponse
	if err := c.doWithRetry(ctx, func() error {
		return c.doRequest(ctx, c.oraclesURL, &response)
	}); err != nil {
		return nil, fmt.Errorf("fetch oracles: %w", err)
	}

	return &response, nil
}

// FetchProtocols retrieves protocol metadata from DefiLlama /lite/protocols2 endpoint.
func (c *Client) FetchProtocols(ctx context.Context) ([]Protocol, error) {
	var protocols []Protocol
	if err := c.doWithRetry(ctx, func() error {
		return c.doRequest(ctx, c.protocolsURL, &protocols)
	}); err != nil {
		return nil, fmt.Errorf("fetch protocols: %w", err)
	}

	return protocols, nil
}
