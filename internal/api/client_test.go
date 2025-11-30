package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"log/slog"

	"github.com/switchboard-xyz/defillama-extract/internal/config"
)

type testPayload struct {
	Value string `json:"value"`
}

type logEntry struct {
	Level   slog.Level
	Message string
	Attrs   map[string]any
}

type memoryHandler struct {
	mu      sync.Mutex
	entries []logEntry
}

func (h *memoryHandler) Enabled(_ context.Context, level slog.Level) bool { return true }

func (h *memoryHandler) Handle(_ context.Context, r slog.Record) error {
	entry := logEntry{Level: r.Level, Message: r.Message, Attrs: make(map[string]any)}
	r.Attrs(func(a slog.Attr) bool {
		entry.Attrs[a.Key] = a.Value.Any()
		return true
	})
	h.mu.Lock()
	h.entries = append(h.entries, entry)
	h.mu.Unlock()
	return nil
}

func (h *memoryHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *memoryHandler) WithGroup(_ string) slog.Handler      { return h }

func (h *memoryHandler) Entries() []logEntry {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]logEntry, len(h.entries))
	copy(out, h.entries)
	return out
}

func intAttr(t *testing.T, attrs map[string]any, key string) int {
	t.Helper()
	val, ok := attrs[key]
	if !ok {
		t.Fatalf("missing attr %s", key)
	}
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case uint64:
		return int(v)
	case float64:
		return int(v)
	default:
		t.Fatalf("attr %s has unexpected type %T", key, val)
	}
	return 0
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

func TestDoRequest_LogsStartAndCompletion(t *testing.T) {
	h := &memoryHandler{}
	logger := slog.New(h)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"value":"ok"}`)); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	client := NewClient(&config.APIConfig{Timeout: time.Second}, logger)
	var payload testPayload
	if err := client.doRequest(context.Background(), server.URL, &payload); err != nil {
		t.Fatalf("doRequest returned error: %v", err)
	}

	entries := h.Entries()
	if len(entries) < 2 {
		t.Fatalf("expected at least 2 log entries, got %d", len(entries))
	}

	startIdx, completeIdx := -1, -1
	for i, e := range entries {
		switch e.Message {
		case "starting API request":
			startIdx = i
			if got := e.Attrs["url"]; got != server.URL {
				t.Fatalf("start url = %v, want %v", got, server.URL)
			}
			if got := e.Attrs["method"]; got != http.MethodGet {
				t.Fatalf("start method = %v, want %s", got, http.MethodGet)
			}
		case "API request completed":
			completeIdx = i
			status := intAttr(t, e.Attrs, "status")
			if status != http.StatusOK {
				t.Fatalf("completion status = %d, want %d", status, http.StatusOK)
			}
			dur, ok := e.Attrs["duration_ms"].(int64)
			if !ok || dur < 0 {
				t.Fatalf("duration_ms missing or negative: %v", e.Attrs["duration_ms"])
			}
		}
	}

	if startIdx == -1 || completeIdx == -1 {
		t.Fatalf("missing start or completion log; entries: %+v", entries)
	}

	if !(startIdx < completeIdx) {
		t.Fatalf("start log should precede completion log (got start=%d complete=%d)", startIdx, completeIdx)
	}
}

func TestDoRequest_LogsFailureWithStatus(t *testing.T) {
	h := &memoryHandler{}
	logger := slog.New(h)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	cfg := &config.APIConfig{Timeout: time.Second, MaxRetries: 0}
	client := NewClient(cfg, logger)
	var payload testPayload

	err := client.doWithRetry(context.Background(), func(ctx context.Context) error {
		return client.doRequest(ctx, server.URL, &payload)
	})
	if err == nil {
		t.Fatalf("expected error for 500 status")
	}

	entries := h.Entries()
	warnIdx := -1
	for i, e := range entries {
		if e.Message == "API request failed" {
			warnIdx = i
			status := intAttr(t, e.Attrs, "status")
			if status != http.StatusInternalServerError {
				t.Fatalf("warn status = %d, want %d", status, http.StatusInternalServerError)
			}
			attempt := intAttr(t, e.Attrs, "attempt")
			if attempt != 1 {
				t.Fatalf("attempt = %d, want 1", attempt)
			}
			dur, ok := e.Attrs["duration_ms"].(int64)
			if !ok || dur < 0 {
				t.Fatalf("duration_ms missing or negative: %v", e.Attrs["duration_ms"])
			}
			break
		}
	}

	if warnIdx == -1 {
		t.Fatalf("expected warn log for failure, entries: %+v", entries)
	}
}

func TestDoRequest_LogsNetworkError(t *testing.T) {
	h := &memoryHandler{}
	logger := slog.New(h)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close()

	client := NewClient(&config.APIConfig{Timeout: time.Second}, logger)
	var payload testPayload

	err := client.doRequest(context.Background(), url, &payload)
	if err == nil {
		t.Fatalf("expected network error, got nil")
	}

	entries := h.Entries()
	var found bool
	for _, e := range entries {
		if e.Message == "API request failed" {
			found = true
			if statusVal, ok := e.Attrs["status"]; ok {
				switch statusVal.(type) {
				case int, int64:
					if intAttr(t, e.Attrs, "status") != 0 {
						t.Fatalf("expected status 0, got %v", statusVal)
					}
				}
			}
			attempt := intAttr(t, e.Attrs, "attempt")
			if attempt != 1 {
				t.Fatalf("attempt = %d, want 1", attempt)
			}
		}
	}

	if !found {
		t.Fatalf("expected failure log for network error, entries: %+v", entries)
	}
}

func TestFetchAll_LogsDistinctRequests(t *testing.T) {
	h := &memoryHandler{}
	logger := slog.New(h)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oracles":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"oracles":{},"chart":{}}`))
		case "/lite/protocols2":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)

	cfg := &config.APIConfig{
		Timeout:      time.Second,
		OraclesURL:   server.URL + "/oracles",
		ProtocolsURL: server.URL + "/lite/protocols2?b=2",
	}
	client := NewClient(cfg, logger)

	if _, err := client.FetchAll(context.Background()); err != nil {
		t.Fatalf("FetchAll returned error: %v", err)
	}

	entries := h.Entries()
	starts := make(map[string]int)
	completions := make(map[string]int)
	for i, e := range entries {
		urlVal, _ := e.Attrs["url"].(string)
		switch e.Message {
		case "starting API request":
			starts[urlVal] = i
		case "API request completed":
			completions[urlVal] = i
		}
	}

	if len(starts) != 2 || len(completions) != 2 {
		t.Fatalf("expected logs for two URLs, got starts=%d completions=%d entries=%+v", len(starts), len(completions), entries)
	}

	for url, startIdx := range starts {
		completeIdx, ok := completions[url]
		if !ok {
			t.Fatalf("missing completion log for url %s", url)
		}
		if !(startIdx < completeIdx) {
			t.Fatalf("start should precede completion for %s (start=%d complete=%d)", url, startIdx, completeIdx)
		}
	}
}
