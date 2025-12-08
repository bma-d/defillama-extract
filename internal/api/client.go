package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"log/slog"

	"github.com/switchboard-xyz/defillama-extract/internal/config"
	"golang.org/x/net/http2"
	"golang.org/x/sync/errgroup"
)

const userAgentValue = "defillama-extract/1.0"

var oraclesCachePath = filepath.Join("api-cache", "oracles.json")
var oraclesMinInterval = 4 * time.Second

type contextKey string

const attemptContextKey contextKey = "api_attempt"
const protocolRateLimitInterval = 200 * time.Millisecond

// Client wraps http.Client with configuration needed for DefiLlama requests.
type Client struct {
	httpClient                  *http.Client
	oraclesURL                  string
	protocolsURL                string
	protocolTVLEndpointTemplate string
	userAgent                   string
	maxRetries                  int
	retryDelay                  time.Duration
	logger                      *slog.Logger
	rng                         *rand.Rand
	rngMu                       sync.Mutex
	protocolRateMu              sync.Mutex
	nextProtocolAllowedAt       time.Time
	oracleRateMu                sync.Mutex
	nextOracleAllowedAt         time.Time
	minOracleInterval           time.Duration
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
		httpClient:                  &http.Client{Timeout: cfg.Timeout},
		oraclesURL:                  oraclesURL,
		protocolsURL:                protocolsURL,
		protocolTVLEndpointTemplate: ProtocolTVLEndpointTemplate,
		userAgent:                   userAgentValue,
		maxRetries:                  cfg.MaxRetries,
		retryDelay:                  cfg.RetryDelay,
		logger:                      logger,
		rng:                         rand.New(rand.NewSource(time.Now().UnixNano())),
		minOracleInterval:           oraclesMinInterval,
	}
}

// doRequest performs a GET request with User-Agent injection and JSON decoding.
func attemptFromContext(ctx context.Context) int {
	attempt, ok := ctx.Value(attemptContextKey).(int)
	if !ok || attempt < 1 {
		return 1
	}

	return attempt
}

// doRequest performs a GET request with User-Agent injection and JSON decoding.
func (c *Client) doRequest(ctx context.Context, url string, target any) error {
	start := time.Now()
	if url == c.oraclesURL {
		if err := c.waitForOracleRateLimit(ctx); err != nil {
			return err
		}
	}
	attempt := attemptFromContext(ctx)
	method := http.MethodGet

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)

	c.logger.Debug("starting API request",
		"url", url,
		"method", method,
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		duration := time.Since(start)
		c.logger.Warn("API request failed",
			"url", url,
			"attempt", attempt,
			"duration_ms", duration.Milliseconds(),
			"error", err,
		)
		return &APIError{
			Endpoint:   url,
			StatusCode: 0,
			Message:    fmt.Sprintf("execute request: %v", err),
			Err:        err,
		}
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		err := fmt.Errorf("unexpected status: %d", resp.StatusCode)
		c.logger.Warn("API request failed",
			"url", url,
			"status", resp.StatusCode,
			"attempt", attempt,
			"duration_ms", duration.Milliseconds(),
			"error", err,
		)
		return &APIError{
			Endpoint:   url,
			StatusCode: resp.StatusCode,
			Message:    err.Error(),
			Err:        err,
		}
	}

	c.logger.Info("API request completed",
		"url", url,
		"status", resp.StatusCode,
		"duration_ms", duration.Milliseconds(),
	)

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

func (c *Client) loadOraclesCache() (*OracleAPIResponse, error) {
	data, err := os.ReadFile(oraclesCachePath)
	if err != nil {
		return nil, err
	}

	var resp OracleAPIResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *Client) saveOraclesCache(resp *OracleAPIResponse) error {
	if resp == nil {
		return errors.New("nil oracles response")
	}

	if err := os.MkdirAll(filepath.Dir(oraclesCachePath), 0o755); err != nil {
		c.logger.Warn("oracles_cache_write_failed", "path", oraclesCachePath, "error", err)
		return err
	}

	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		c.logger.Warn("oracles_cache_marshal_failed", "error", err)
		return err
	}

	if err := os.WriteFile(oraclesCachePath, data, 0o644); err != nil {
		c.logger.Warn("oracles_cache_write_failed", "path", oraclesCachePath, "error", err)
		return err
	}

	c.logger.Debug("oracles_cache_updated", "path", oraclesCachePath)
	return nil
}

func (c *Client) waitForOracleRateLimit(ctx context.Context) error {
	c.oracleRateMu.Lock()
	now := time.Now()
	wait := time.Until(c.nextOracleAllowedAt)
	if wait <= 0 {
		c.nextOracleAllowedAt = now.Add(c.minOracleInterval)
		c.oracleRateMu.Unlock()
		return nil
	}
	c.oracleRateMu.Unlock()

	t := time.NewTimer(wait)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		c.oracleRateMu.Lock()
		c.nextOracleAllowedAt = time.Now().Add(c.minOracleInterval)
		c.oracleRateMu.Unlock()
		return nil
	}
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

	var h2Err http2.StreamError
	if errors.As(err, &h2Err) {
		return true
	}

	if errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	var opErr *net.OpError
	return errors.As(err, &opErr)
}

func isNotFoundAPIError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}

	return false
}

func (c *Client) waitForProtocolRateLimit(ctx context.Context) error {
	c.protocolRateMu.Lock()
	now := time.Now()
	waitUntil := now

	if c.nextProtocolAllowedAt.After(now) {
		waitUntil = c.nextProtocolAllowedAt
	}

	c.nextProtocolAllowedAt = waitUntil.Add(protocolRateLimitInterval)
	c.protocolRateMu.Unlock()

	wait := time.Until(waitUntil)
	if wait <= 0 {
		return nil
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *Client) calculateBackoff(attempt int, baseDelay time.Duration) time.Duration {
	exponential := baseDelay * time.Duration(1<<attempt)
	c.rngMu.Lock()
	jitterMultiplier := 0.75 + c.rng.Float64()*0.5
	c.rngMu.Unlock()
	return time.Duration(float64(exponential) * jitterMultiplier)
}

func (c *Client) doWithRetry(ctx context.Context, fn func(context.Context) error) error {
	maxAttempts := c.maxRetries + 1
	var lastErr error
	var lastEndpoint string

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		attemptCtx := context.WithValue(ctx, attemptContextKey, attempt+1)
		err := fn(attemptCtx)
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
	if err := c.doWithRetry(ctx, func(ctx context.Context) error {
		return c.doRequest(ctx, c.oraclesURL, &response)
	}); err != nil {
		cached, cacheErr := c.loadOraclesCache()
		if cacheErr == nil {
			c.logger.Warn("oracles_fallback_cache_used",
				"path", oraclesCachePath,
				"error", err,
			)
			return cached, nil
		}

		return nil, fmt.Errorf("fetch oracles: %w", err)
	}

	_ = c.saveOraclesCache(&response)

	return &response, nil
}

// FetchProtocols retrieves protocol metadata from DefiLlama /lite/protocols2 endpoint.
func (c *Client) FetchProtocols(ctx context.Context) ([]Protocol, error) {
	var protocols protocolList
	if err := c.doWithRetry(ctx, func(ctx context.Context) error {
		return c.doRequest(ctx, c.protocolsURL, &protocols)
	}); err != nil {
		return nil, fmt.Errorf("fetch protocols: %w", err)
	}

	return []Protocol(protocols), nil
}

// FetchProtocolTVL retrieves historical TVL data for a protocol from DefiLlama /protocol/{slug} endpoint.
func (c *Client) FetchProtocolTVL(ctx context.Context, slug string) (*ProtocolTVLResponse, error) {
	if err := c.waitForProtocolRateLimit(ctx); err != nil {
		return nil, err
	}

	url := fmt.Sprintf(c.protocolTVLEndpointTemplate, slug)
	var response ProtocolTVLResponse

	err := c.doWithRetry(ctx, func(ctx context.Context) error {
		return c.doRequest(ctx, url, &response)
	})
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, ctx.Err()
		}

		if isNotFoundAPIError(err) {
			c.logger.Warn("protocol_not_found",
				"slug", slug,
				"status_code", http.StatusNotFound,
			)
			return nil, nil
		}

		return nil, fmt.Errorf("fetch protocol TVL %s: %w", slug, err)
	}

	return &response, nil
}

// FetchAll retrieves oracle and protocol data concurrently using errgroup.
func (c *Client) FetchAll(ctx context.Context) (*FetchResult, error) {
	start := time.Now()
	var (
		oracleResp       *OracleAPIResponse
		protocols        []Protocol
		oracleDuration   time.Duration
		protocolDuration time.Duration
	)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		fetchStart := time.Now()
		resp, err := c.FetchOracles(ctx)
		oracleDuration = time.Since(fetchStart)
		if err != nil {
			return err
		}
		oracleResp = resp
		return nil
	})

	g.Go(func() error {
		fetchStart := time.Now()
		resp, err := c.FetchProtocols(ctx)
		protocolDuration = time.Since(fetchStart)
		if err != nil {
			return err
		}
		protocols = resp
		return nil
	})

	if err := g.Wait(); err != nil {
		total := time.Since(start)
		c.logger.Error("parallel fetch failed",
			"error", err,
			"total_duration_ms", total.Milliseconds(),
		)
		return nil, err
	}

	total := time.Since(start)
	c.logger.Info("parallel fetch completed",
		"oracle_duration_ms", oracleDuration.Milliseconds(),
		"protocol_duration_ms", protocolDuration.Milliseconds(),
		"total_duration_ms", total.Milliseconds(),
	)

	return &FetchResult{
		OracleResponse: oracleResp,
		Protocols:      protocols,
	}, nil
}
