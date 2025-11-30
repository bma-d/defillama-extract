package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"log/slog"

	"github.com/switchboard-xyz/defillama-extract/internal/config"
)

func newFetchAllClient(t *testing.T, oracleHandler, protocolHandler http.HandlerFunc, timeout time.Duration) (*Client, func()) {
	t.Helper()

	oracleSrv := httptest.NewServer(oracleHandler)
	protocolSrv := httptest.NewServer(protocolHandler)

	cfg := &config.APIConfig{
		OraclesURL:   oracleSrv.URL,
		ProtocolsURL: protocolSrv.URL,
		Timeout:      timeout,
		MaxRetries:   0,
		RetryDelay:   10 * time.Millisecond,
	}

	client := NewClient(cfg, slog.Default())

	cleanup := func() {
		oracleSrv.Close()
		protocolSrv.Close()
	}

	return client, cleanup
}

func testFilePath(name string) string {
	return filepath.Join("..", "..", "testdata", name)
}

func TestFetchAll_BothSucceed(t *testing.T) {
	oracleHandler := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, testFilePath("oracle_response.json"))
	}
	protocolHandler := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, testFilePath("protocol_response.json"))
	}

	client, cleanup := newFetchAllClient(t, oracleHandler, protocolHandler, 2*time.Second)
	t.Cleanup(cleanup)

	res, err := client.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("FetchAll returned error: %v", err)
	}

	if res == nil || res.OracleResponse == nil || len(res.Protocols) == 0 {
		t.Fatalf("expected populated FetchResult, got %+v", res)
	}
}

func TestFetchAll_OracleFails(t *testing.T) {
	oracleHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}
	protocolHandler := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, testFilePath("protocol_response.json"))
	}

	client, cleanup := newFetchAllClient(t, oracleHandler, protocolHandler, 2*time.Second)
	t.Cleanup(cleanup)

	_, err := client.FetchAll(context.Background())
	if err == nil {
		t.Fatalf("expected oracle error, got nil")
	}
	if !strings.Contains(err.Error(), "fetch oracles") {
		t.Fatalf("expected oracle fetch error, got %v", err)
	}
}

func TestFetchAll_ProtocolFails(t *testing.T) {
	oracleHandler := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, testFilePath("oracle_response.json"))
	}
	protocolHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}

	client, cleanup := newFetchAllClient(t, oracleHandler, protocolHandler, 2*time.Second)
	t.Cleanup(cleanup)

	_, err := client.FetchAll(context.Background())
	if err == nil {
		t.Fatalf("expected protocol error, got nil")
	}
	if !strings.Contains(err.Error(), "fetch protocols") {
		t.Fatalf("expected protocol fetch error, got %v", err)
	}
}

func TestFetchAll_BothFailReturnsFirst(t *testing.T) {
	oracleHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	protocolHandler := func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusInternalServerError)
	}

	client, cleanup := newFetchAllClient(t, oracleHandler, protocolHandler, 2*time.Second)
	t.Cleanup(cleanup)

	_, err := client.FetchAll(context.Background())
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "fetch oracles") {
		t.Fatalf("expected first error to be oracle failure, got %v", err)
	}
}

func TestFetchAll_ContextCancelled(t *testing.T) {
	oracleHandler := func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(testPayload{Value: "oracle"})
	}
	protocolHandler := func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode([]Protocol{})
	}

	client, cleanup := newFetchAllClient(t, oracleHandler, protocolHandler, 500*time.Millisecond)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.FetchAll(ctx)
	if err == nil {
		t.Fatalf("expected context cancellation error")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context error, got %v", err)
	}
}

func TestFetchAll_ParallelTiming(t *testing.T) {
	delay := 120 * time.Millisecond
	oracleHandler := func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		http.ServeFile(w, r, testFilePath("oracle_response.json"))
	}
	protocolHandler := func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		http.ServeFile(w, r, testFilePath("protocol_response.json"))
	}

	client, cleanup := newFetchAllClient(t, oracleHandler, protocolHandler, 2*time.Second)
	t.Cleanup(cleanup)

	start := time.Now()
	_, err := client.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	elapsed := time.Since(start)
	if elapsed >= 200*time.Millisecond {
		t.Fatalf("expected parallel execution under 200ms, got %v", elapsed)
	}
}

func TestFetchAll_LogsSuccess(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	oracleHandler := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, testFilePath("oracle_response.json"))
	}
	protocolHandler := func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, testFilePath("protocol_response.json"))
	}

	client, cleanup := newFetchAllClient(t, oracleHandler, protocolHandler, 2*time.Second)
	t.Cleanup(cleanup)
	client.logger = logger

	if _, err := client.FetchAll(context.Background()); err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	logText := buf.String()
	if !strings.Contains(logText, "parallel fetch completed") {
		t.Fatalf("expected success log, got %s", logText)
	}
	if !strings.Contains(logText, "total_duration_ms") {
		t.Fatalf("expected duration fields in log, got %s", logText)
	}
}
