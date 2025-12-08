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

func newTestClient(oraclesURL string) *Client {
	cfg := &config.APIConfig{
		OraclesURL: oraclesURL,
		Timeout:    200 * time.Millisecond,
		MaxRetries: 0,
	}

	return NewClient(cfg, nil)
}

func TestFetchOracles_Success(t *testing.T) {
	oldPath := oraclesCachePath
	oraclesCachePath = filepath.Join(t.TempDir(), "oracles.json")
	prevInterval := oraclesMinInterval
	oraclesMinInterval = 0
	t.Cleanup(func() {
		oraclesCachePath = oldPath
		oraclesMinInterval = prevInterval
	})

	fixturePath := filepath.Join("..", "..", "testdata", "oracle_response.json")
	fixture, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	t.Cleanup(server.Close)

	client := newTestClient(server.URL)

	resp, err := client.FetchOracles(context.Background())
	if err != nil {
		t.Fatalf("FetchOracles returned error: %v", err)
	}

	if len(resp.Oracles["Switchboard"]) != 3 {
		t.Fatalf("expected 3 protocol entries, got %d", len(resp.Oracles["Switchboard"]))
	}

	if resp.ChainsByOracle["Switchboard"][0] != "Solana" {
		t.Fatalf("expected first chain Solana, got %s", resp.ChainsByOracle["Switchboard"][0])
	}
}

func TestFetchOracles_SetsUserAgent(t *testing.T) {
	oldPath := oraclesCachePath
	oraclesCachePath = filepath.Join(t.TempDir(), "oracles.json")
	prevInterval := oraclesMinInterval
	oraclesMinInterval = 0
	t.Cleanup(func() {
		oraclesCachePath = oldPath
		oraclesMinInterval = prevInterval
	})

	var capturedUA string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"oracles":{},"chart":{},"oraclesTVS":{},"chainsByOracle":{}}`))
	}))
	t.Cleanup(server.Close)

	client := newTestClient(server.URL)

	if _, err := client.FetchOracles(context.Background()); err != nil {
		t.Fatalf("FetchOracles returned error: %v", err)
	}

	if capturedUA != userAgentValue {
		t.Fatalf("expected User-Agent %q, got %q", userAgentValue, capturedUA)
	}
}

func TestFetchOracles_StatusErrors(t *testing.T) {
	oldPath := oraclesCachePath
	oraclesCachePath = filepath.Join(t.TempDir(), "oracles.json")
	prevInterval := oraclesMinInterval
	oraclesMinInterval = 0
	t.Cleanup(func() {
		oraclesCachePath = oldPath
		oraclesMinInterval = prevInterval
	})

	cases := []int{http.StatusInternalServerError, http.StatusNotFound}

	for _, status := range cases {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		}))
		t.Cleanup(server.Close)

		client := newTestClient(server.URL)

		_, err := client.FetchOracles(context.Background())
		if err == nil {
			t.Fatalf("expected error for status %d, got nil", status)
		}

		if !strings.Contains(err.Error(), "unexpected status") {
			t.Fatalf("expected unexpected status error, got %v", err)
		}
	}
}

func TestFetchOracles_MalformedJSON(t *testing.T) {
	oldPath := oraclesCachePath
	oraclesCachePath = filepath.Join(t.TempDir(), "oracles.json")
	prevInterval := oraclesMinInterval
	oraclesMinInterval = 0
	t.Cleanup(func() {
		oraclesCachePath = oldPath
		oraclesMinInterval = prevInterval
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"oracles": `))
	}))
	t.Cleanup(server.Close)

	client := newTestClient(server.URL)

	_, err := client.FetchOracles(context.Background())
	if err == nil {
		t.Fatalf("expected decode error, got nil")
	}

	if !strings.Contains(err.Error(), "decode response") {
		t.Fatalf("expected decode response error, got %v", err)
	}
}

func TestFetchOracles_ContextCancellation(t *testing.T) {
	oldPath := oraclesCachePath
	oraclesCachePath = filepath.Join(t.TempDir(), "oracles.json")
	prevInterval := oraclesMinInterval
	oraclesMinInterval = 0
	t.Cleanup(func() {
		oraclesCachePath = oldPath
		oraclesMinInterval = prevInterval
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)

	client := newTestClient(server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := client.FetchOracles(ctx)
	if err == nil {
		t.Fatalf("expected cancellation error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestFetchOracles_FallbackCache(t *testing.T) {
	// point cache to temp file
	oldPath := oraclesCachePath
	t.Cleanup(func() { oraclesCachePath = oldPath })
	tmpDir := t.TempDir()
	oraclesCachePath = filepath.Join(tmpDir, "oracles.json")

	fixturePath := filepath.Join("..", "..", "testdata", "oracle_response.json")
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := os.WriteFile(oraclesCachePath, data, 0o644); err != nil {
		t.Fatalf("write cache: %v", err)
	}

	// server always fails
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	client := newTestClient(server.URL)

	resp, err := client.FetchOracles(context.Background())
	if err != nil {
		t.Fatalf("expected fallback without error, got %v", err)
	}
	if len(resp.Oracles) == 0 {
		t.Fatalf("expected data from cache, got empty response")
	}
}

func TestFetchOracles_WritesCacheOnSuccess(t *testing.T) {
	oldPath := oraclesCachePath
	t.Cleanup(func() { oraclesCachePath = oldPath })
	tmpDir := t.TempDir()
	oraclesCachePath = filepath.Join(tmpDir, "oracles.json")

	fixturePath := filepath.Join("..", "..", "testdata", "oracle_response.json")
	fixture, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	t.Cleanup(server.Close)

	client := newTestClient(server.URL)
	if _, err := client.FetchOracles(context.Background()); err != nil {
		t.Fatalf("FetchOracles returned error: %v", err)
	}

	info, err := os.Stat(oraclesCachePath)
	if err != nil {
		t.Fatalf("expected cache file, got error: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("expected cache file to be written, got size 0")
	}
}
