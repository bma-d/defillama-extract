package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"log/slog"

	"github.com/switchboard-xyz/defillama-extract/internal/config"
)

func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

type fakeNetError struct {
	timeout   bool
	temporary bool
}

func (e fakeNetError) Error() string   { return "net error" }
func (e fakeNetError) Timeout() bool   { return e.timeout }
func (e fakeNetError) Temporary() bool { return e.temporary }

func TestIsRetryable_StatusCodes(t *testing.T) {
	cases := []struct {
		name string
		code int
		want bool
	}{
		{"429 retryable", http.StatusTooManyRequests, true},
		{"500 retryable", http.StatusInternalServerError, true},
		{"502 retryable", http.StatusBadGateway, true},
		{"503 retryable", http.StatusServiceUnavailable, true},
		{"504 retryable", http.StatusGatewayTimeout, true},
		{"400 non-retryable", http.StatusBadRequest, false},
		{"401 non-retryable", http.StatusUnauthorized, false},
		{"403 non-retryable", http.StatusForbidden, false},
		{"404 non-retryable", http.StatusNotFound, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := &APIError{StatusCode: tc.code}
			if got := isRetryable(tc.code, err); got != tc.want {
				t.Fatalf("isRetryable(%d) = %v, want %v", tc.code, got, tc.want)
			}
		})
	}
}

func TestIsRetryable_TimeoutAndNetwork(t *testing.T) {
	if !isRetryable(0, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline to be retryable")
	}

	if !isRetryable(0, fakeNetError{timeout: true, temporary: true}) {
		t.Fatalf("expected timeout network error to be retryable")
	}

	if !isRetryable(0, &net.OpError{}) {
		t.Fatalf("expected op error to be retryable")
	}
}

func TestIsRetryable_DecodeError(t *testing.T) {
	decodeErr := fmt.Errorf("decode response: %w", io.ErrUnexpectedEOF)
	if isRetryable(0, decodeErr) {
		t.Fatalf("decode errors should not be retryable")
	}
}

func TestCalculateBackoff_ExponentialWithJitter(t *testing.T) {
	client := NewClient(&config.APIConfig{Timeout: time.Second, RetryDelay: time.Second}, slog.Default())

	t.Run("attempt 0", func(t *testing.T) {
		const seed = int64(1)
		client.rng = rand.New(rand.NewSource(seed))
		expectedRng := rand.New(rand.NewSource(seed))
		expected := time.Duration(float64(time.Second) * (0.75 + expectedRng.Float64()*0.5))
		if got := client.calculateBackoff(0, time.Second); got != expected {
			t.Fatalf("attempt0 backoff = %v, want %v", got, expected)
		}
	})

	t.Run("attempt 2 bounds", func(t *testing.T) {
		client.rng = rand.New(rand.NewSource(42))
		delay := client.calculateBackoff(2, time.Second)
		exp := 4 * time.Second
		min := time.Duration(float64(exp) * 0.75)
		max := time.Duration(float64(exp) * 1.25)
		if delay < min || delay > max {
			t.Fatalf("backoff out of jitter bounds: %v (want %v-%v)", delay, min, max)
		}
	})
}

func TestDoWithRetry_RetryableErrorsUpToMax(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(server.Close)

	buf := &bytes.Buffer{}
	cfg := &config.APIConfig{OraclesURL: server.URL, Timeout: time.Second, MaxRetries: 3, RetryDelay: 2 * time.Millisecond}
	client := NewClient(cfg, newTestLogger(buf))

	var payload testPayload
	err := client.doWithRetry(context.Background(), func(ctx context.Context) error {
		return client.doRequest(ctx, server.URL, &payload)
	})

	if err == nil {
		t.Fatalf("expected error after retries, got nil")
	}

	if got := atomic.LoadInt32(&attempts); got != 4 {
		t.Fatalf("expected 4 attempts, got %d", got)
	}

	logText := buf.String()
	if count := strings.Count(logText, "retrying API request"); count != 3 {
		t.Fatalf("expected 3 retry logs, got %d\nlogs: %s", count, logText)
	}
	if !strings.Contains(logText, "max retries exceeded") {
		t.Fatalf("expected max retries error log, got logs: %s", logText)
	}
}

func TestDoWithRetry_NonRetryableReturnsImmediately(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	buf := &bytes.Buffer{}
	cfg := &config.APIConfig{OraclesURL: server.URL, Timeout: time.Second, MaxRetries: 3, RetryDelay: 2 * time.Millisecond}
	client := NewClient(cfg, newTestLogger(buf))

	var payload testPayload
	err := client.doWithRetry(context.Background(), func(ctx context.Context) error {
		return client.doRequest(ctx, server.URL, &payload)
	})

	if err == nil {
		t.Fatalf("expected non-retryable error")
	}
	if got := atomic.LoadInt32(&attempts); got != 1 {
		t.Fatalf("expected single attempt, got %d", got)
	}
	if strings.Contains(buf.String(), "retrying API request") {
		t.Fatalf("should not log retries for non-retryable error")
	}
}

func TestDoWithRetry_SucceedsAfterRetries(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attempts, 1)
		if attempt <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"value":"ok"}`)); err != nil {
			t.Fatalf("write response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	buf := &bytes.Buffer{}
	cfg := &config.APIConfig{OraclesURL: server.URL, Timeout: time.Second, MaxRetries: 3, RetryDelay: 2 * time.Millisecond}
	client := NewClient(cfg, newTestLogger(buf))

	var payload testPayload
	if err := client.doWithRetry(context.Background(), func(ctx context.Context) error {
		return client.doRequest(ctx, server.URL, &payload)
	}); err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}

	if payload.Value != "ok" {
		t.Fatalf("expected decoded payload 'ok', got %q", payload.Value)
	}

	if got := atomic.LoadInt32(&attempts); got != 3 {
		t.Fatalf("expected 3 attempts (2 failures then success), got %d", got)
	}

	if !strings.Contains(buf.String(), "request succeeded after retries") {
		t.Fatalf("expected success log after retries, logs: %s", buf.String())
	}
}

func TestDoWithRetry_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	buf := &bytes.Buffer{}
	cfg := &config.APIConfig{OraclesURL: server.URL, Timeout: time.Second, MaxRetries: 5, RetryDelay: 50 * time.Millisecond}
	client := NewClient(cfg, newTestLogger(buf))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		var payload testPayload
		err := client.doWithRetry(ctx, func(ctx context.Context) error {
			return client.doRequest(ctx, server.URL, &payload)
		})
		done <- err
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatalf("timeout waiting for retry loop to exit on cancellation")
	}
}
