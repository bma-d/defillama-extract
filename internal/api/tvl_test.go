package api

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"log/slog"

	"github.com/switchboard-xyz/defillama-extract/internal/config"
)

func newTestTVLClient(t *testing.T, template string, retries int) *Client {
	t.Helper()

	cfg := &config.APIConfig{
		Timeout:    200 * time.Millisecond,
		MaxRetries: retries,
		RetryDelay: 10 * time.Millisecond,
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
	client := NewClient(cfg, logger)
	client.protocolTVLEndpointTemplate = template

	return client
}

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", name, err)
	}

	return data
}

func TestFetchProtocolTVL_Success(t *testing.T) {
	fixture := loadFixture(t, "protocol_tvl_response.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if got := r.Header.Get("User-Agent"); got != userAgentValue {
			t.Fatalf("expected User-Agent %q, got %q", userAgentValue, got)
		}
		if !strings.HasSuffix(r.URL.Path, "/drift") {
			t.Fatalf("expected slug drift, got path %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	t.Cleanup(server.Close)

	template := server.URL + "/%s"
	client := newTestTVLClient(t, template, 1)

	resp, err := client.FetchProtocolTVL(context.Background(), "drift")
	if err != nil {
		t.Fatalf("FetchProtocolTVL returned error: %v", err)
	}

	if resp == nil {
		t.Fatalf("expected response, got nil")
	}

	if resp.Name != "Drift Trade" {
		t.Fatalf("expected name Drift Trade, got %s", resp.Name)
	}

	if len(resp.TVL) != 2 {
		t.Fatalf("expected 2 tvl points, got %d", len(resp.TVL))
	}

	if resp.CurrentChainTvls["Solana"] != 677000000 {
		t.Fatalf("unexpected chain tvl: %v", resp.CurrentChainTvls)
	}
}

func TestFetchProtocolTVL_NotFound(t *testing.T) {
	fixture := loadFixture(t, "protocol_404_response.json")

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(fixture)
	}))
	t.Cleanup(server.Close)

	template := server.URL + "/%s"
	client := NewClient(&config.APIConfig{Timeout: 100 * time.Millisecond}, logger)
	client.protocolTVLEndpointTemplate = template

	resp, err := client.FetchProtocolTVL(context.Background(), "missing-protocol")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if resp != nil {
		t.Fatalf("expected nil response for 404, got %#v", resp)
	}

	if !strings.Contains(logBuf.String(), "protocol_not_found") {
		t.Fatalf("expected warning log for protocol_not_found, got %s", logBuf.String())
	}
}

func TestFetchProtocolTVL_ServerErrorRetries(t *testing.T) {
	// Override cache dir to ensure no fallback during error tests
	origCacheDir := protocolTVLCacheDir
	protocolTVLCacheDir = filepath.Join(t.TempDir(), "nonexistent")
	t.Cleanup(func() { protocolTVLCacheDir = origCacheDir })

	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	template := server.URL + "/%s"
	client := newTestTVLClient(t, template, 1)

	_, err := client.FetchProtocolTVL(context.Background(), "drift")
	if err == nil {
		t.Fatalf("expected error for 500 response, got nil")
	}

	if attempts < 2 {
		t.Fatalf("expected at least 2 attempts, got %d", attempts)
	}

	if !strings.Contains(err.Error(), "fetch protocol TVL") {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

func TestFetchProtocolTVL_InvalidJSON(t *testing.T) {
	// Override cache dir to ensure no fallback during error tests
	origCacheDir := protocolTVLCacheDir
	protocolTVLCacheDir = filepath.Join(t.TempDir(), "nonexistent")
	t.Cleanup(func() { protocolTVLCacheDir = origCacheDir })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":`))
	}))
	t.Cleanup(server.Close)

	client := newTestTVLClient(t, server.URL+"/%s", 0)

	_, err := client.FetchProtocolTVL(context.Background(), "drift")
	if err == nil {
		t.Fatalf("expected decode error, got nil")
	}

	if !strings.Contains(err.Error(), "decode response") {
		t.Fatalf("expected decode response error, got %v", err)
	}
}

func TestFetchProtocolTVL_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)

	client := newTestTVLClient(t, server.URL+"/%s", 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.FetchProtocolTVL(ctx, "drift")
	if err == nil {
		t.Fatalf("expected cancellation error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestFetchProtocolTVL_EmptyTVLArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"Empty","tvl":[],"currentChainTvls":{}}`))
	}))
	t.Cleanup(server.Close)

	client := newTestTVLClient(t, server.URL+"/%s", 0)

	resp, err := client.FetchProtocolTVL(context.Background(), "empty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil || len(resp.TVL) != 0 {
		t.Fatalf("expected empty tvl slice, got %#v", resp)
	}
}

func TestFetchProtocolTVL_RateLimiting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"name":"X","tvl":[],"currentChainTvls":{}}`))
	}))
	t.Cleanup(server.Close)

	client := newTestTVLClient(t, server.URL+"/%s", 0)

	start := time.Now()
	if _, err := client.FetchProtocolTVL(context.Background(), "a"); err != nil {
		t.Fatalf("first call error: %v", err)
	}

	if _, err := client.FetchProtocolTVL(context.Background(), "b"); err != nil {
		t.Fatalf("second call error: %v", err)
	}

	elapsed := time.Since(start)
	if elapsed < 180*time.Millisecond {
		t.Fatalf("expected rate limiter to enforce delay, elapsed %v", elapsed)
	}
}

func TestFetchProtocolTVL_FallbackCache(t *testing.T) {
	// Set up temp cache directory with a valid cache file
	tmpDir := t.TempDir()
	origCacheDir := protocolTVLCacheDir
	protocolTVLCacheDir = tmpDir
	t.Cleanup(func() { protocolTVLCacheDir = origCacheDir })

	// Write cache file for slug "cached-proto"
	cacheData := `{"name":"Cached Protocol","tvl":[{"date":1704067200,"totalLiquidityUSD":12345.67}],"currentChainTvls":{"Ethereum":12345.67}}`
	cachePath := filepath.Join(tmpDir, "cached-proto.json")
	if err := os.WriteFile(cachePath, []byte(cacheData), 0o644); err != nil {
		t.Fatalf("failed to write cache: %v", err)
	}

	// Server returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	client := newTestTVLClient(t, server.URL+"/%s", 0)

	resp, err := client.FetchProtocolTVL(context.Background(), "cached-proto")
	if err != nil {
		t.Fatalf("expected fallback to cache, got error: %v", err)
	}

	if resp == nil || resp.Name != "Cached Protocol" {
		t.Fatalf("expected cached protocol, got %+v", resp)
	}

	if len(resp.TVL) != 1 || resp.TVL[0].TotalLiquidityUSD != 12345.67 {
		t.Fatalf("expected cached TVL data, got %+v", resp.TVL)
	}
}
