package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/switchboard-xyz/defillama-extract/internal/config"
)

func newProtocolTestClient(protocolsURL string) *Client {
	cfg := &config.APIConfig{
		ProtocolsURL: protocolsURL,
		Timeout:      200 * time.Millisecond,
		MaxRetries:   0,
	}

	return NewClient(cfg, nil)
}

func TestFetchProtocols_Success(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "testdata", "protocol_response.json")
	fixture, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	t.Cleanup(server.Close)

	client := newProtocolTestClient(server.URL)

	resp, err := client.FetchProtocols(context.Background())
	if err != nil {
		t.Fatalf("FetchProtocols returned error: %v", err)
	}

	if len(resp) != 3 {
		t.Fatalf("expected 3 protocols, got %d", len(resp))
	}

	if resp[0].ID != "marinade-finance" || resp[0].Category != "Liquid Staking" {
		t.Fatalf("unexpected first protocol decoded: %+v", resp[0])
	}
}

func TestFetchProtocols_SetsUserAgent(t *testing.T) {
	var capturedUA string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(server.Close)

	client := newProtocolTestClient(server.URL)

	if _, err := client.FetchProtocols(context.Background()); err != nil {
		t.Fatalf("FetchProtocols returned error: %v", err)
	}

	if capturedUA != userAgentValue {
		t.Fatalf("expected User-Agent %q, got %q", userAgentValue, capturedUA)
	}
}

func TestFetchProtocols_StatusErrors(t *testing.T) {
	// Temporarily override cache path to ensure no fallback during error tests
	origCachePath := protocolsCachePath
	protocolsCachePath = filepath.Join(t.TempDir(), "nonexistent", "protocols.json")
	t.Cleanup(func() { protocolsCachePath = origCachePath })

	cases := []int{http.StatusInternalServerError, http.StatusNotFound}

	for _, status := range cases {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		}))
		t.Cleanup(server.Close)

		client := newProtocolTestClient(server.URL)

		_, err := client.FetchProtocols(context.Background())
		if err == nil {
			t.Fatalf("expected error for status %d, got nil", status)
		}

		if !strings.Contains(err.Error(), "unexpected status") || !strings.Contains(err.Error(), "fetch protocols") {
			t.Fatalf("expected wrapped unexpected status error, got %v", err)
		}
	}
}

func TestFetchProtocols_MalformedJSON(t *testing.T) {
	// Temporarily override cache path to ensure no fallback during error tests
	origCachePath := protocolsCachePath
	protocolsCachePath = filepath.Join(t.TempDir(), "nonexistent", "protocols.json")
	t.Cleanup(func() { protocolsCachePath = origCachePath })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":`))
	}))
	t.Cleanup(server.Close)

	client := newProtocolTestClient(server.URL)

	_, err := client.FetchProtocols(context.Background())
	if err == nil {
		t.Fatalf("expected decode error, got nil")
	}

	if !strings.Contains(err.Error(), "decode response") {
		t.Fatalf("expected decode response error, got %v", err)
	}
}

func TestFetchProtocols_ContextCancellation(t *testing.T) {
	// Temporarily override cache path to ensure no fallback during error tests
	origCachePath := protocolsCachePath
	protocolsCachePath = filepath.Join(t.TempDir(), "nonexistent", "protocols.json")
	t.Cleanup(func() { protocolsCachePath = origCachePath })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)

	client := newProtocolTestClient(server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := client.FetchProtocols(ctx)
	if err == nil {
		t.Fatalf("expected cancellation error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestFetchProtocols_EmptyArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(server.Close)

	client := newProtocolTestClient(server.URL)

	resp, err := client.FetchProtocols(context.Background())
	if err != nil {
		t.Fatalf("FetchProtocols returned error: %v", err)
	}

	if len(resp) != 0 {
		t.Fatalf("expected empty slice, got %d entries", len(resp))
	}
}

func TestFetchProtocols_FallbackCache(t *testing.T) {
	// Set up temp cache directory with a valid cache file
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "protocols.json")

	origCachePath := protocolsCachePath
	protocolsCachePath = cachePath
	t.Cleanup(func() { protocolsCachePath = origCachePath })

	// Write cache file
	cacheData := `[{"id":"cached-proto","name":"Cached Protocol","symbol":"CACHE","category":"Cache"}]`
	if err := os.WriteFile(cachePath, []byte(cacheData), 0o644); err != nil {
		t.Fatalf("failed to write cache: %v", err)
	}

	// Server returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	client := newProtocolTestClient(server.URL)

	resp, err := client.FetchProtocols(context.Background())
	if err != nil {
		t.Fatalf("expected fallback to cache, got error: %v", err)
	}

	if len(resp) != 1 || resp[0].ID != "cached-proto" {
		t.Fatalf("expected cached protocol, got %+v", resp)
	}
}
