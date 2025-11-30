package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"log/slog"

	"github.com/switchboard-xyz/defillama-extract/internal/config"
)

type testPayload struct {
	Value string `json:"value"`
}

func TestNewClient_SetsTimeout(t *testing.T) {
	cfg := &config.APIConfig{Timeout: 15 * time.Second}

	client := NewClient(cfg, nil)

	if client.httpClient.Timeout != 15*time.Second {
		t.Fatalf("expected timeout 15s, got %v", client.httpClient.Timeout)
	}
}

func TestNewClient_StoresUserAgent(t *testing.T) {
	cfg := &config.APIConfig{Timeout: 10 * time.Second}

	client := NewClient(cfg, slog.Default())

	if client.userAgent != userAgentValue {
		t.Fatalf("expected user agent %q, got %q", userAgentValue, client.userAgent)
	}
}

func TestNewClient_NilLoggerUsesDefault(t *testing.T) {
	cfg := &config.APIConfig{Timeout: 5 * time.Second}

	client := NewClient(cfg, nil)

	if client.logger != slog.Default() {
		t.Fatalf("expected slog.Default() when logger is nil")
	}
}

func TestDoRequest_SetsUserAgentAndDecodes(t *testing.T) {
	var capturedUA string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUA = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"value":"ok"}`)); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	cfg := &config.APIConfig{Timeout: 2 * time.Second}
	client := NewClient(cfg, nil)

	var payload testPayload
	if err := client.doRequest(context.Background(), server.URL, &payload); err != nil {
		t.Fatalf("doRequest returned error: %v", err)
	}

	if capturedUA != userAgentValue {
		t.Fatalf("expected User-Agent %q, got %q", userAgentValue, capturedUA)
	}

	if payload.Value != "ok" {
		t.Fatalf("expected decoded payload 'ok', got %q", payload.Value)
	}
}

func TestDoRequest_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"value":"slow"}`)); err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	cfg := &config.APIConfig{Timeout: 50 * time.Millisecond}
	client := NewClient(cfg, nil)

	var payload testPayload
	err := client.doRequest(context.Background(), server.URL, &payload)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}

	var netErr interface{ Timeout() bool }
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func TestDoRequest_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)

	cfg := &config.APIConfig{Timeout: 2 * time.Second}
	client := NewClient(cfg, nil)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	var payload testPayload
	err := client.doRequest(ctx, server.URL, &payload)
	if err == nil {
		t.Fatalf("expected cancellation error, got nil")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestDoRequest_NonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	cfg := &config.APIConfig{Timeout: 2 * time.Second}
	client := NewClient(cfg, nil)

	var payload testPayload
	err := client.doRequest(context.Background(), server.URL, &payload)
	if err == nil {
		t.Fatalf("expected error for non-2xx status, got nil")
	}

	if !strings.Contains(err.Error(), "unexpected status: 500") {
		t.Fatalf("expected unexpected status error, got %v", err)
	}
}
