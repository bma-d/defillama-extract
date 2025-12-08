package tvl

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/switchboard-xyz/defillama-extract/internal/api"
)

type stubTVLClient struct {
	responses map[string]*stubTVLResult
}

type stubTVLResult struct {
	resp *api.ProtocolTVLResponse
	err  error
}

func (s stubTVLClient) FetchProtocolTVL(ctx context.Context, slug string) (*api.ProtocolTVLResponse, error) {
	if res, ok := s.responses[slug]; ok {
		return res.resp, res.err
	}
	return nil, nil
}

func TestFetchAllTVLStats(t *testing.T) {
	client := stubTVLClient{responses: map[string]*stubTVLResult{
		"a": {resp: &api.ProtocolTVLResponse{Name: "A"}},
		"b": {resp: nil},
		"c": {resp: nil, err: errors.New("boom")},
	}}

	var logBuf strings.Builder
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	result, err := FetchAllTVL(context.Background(), client, []string{"a", "b", "c"}, logger)

	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected aggregated error, got %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected one success, got %d", len(result))
	}
	if !strings.Contains(logBuf.String(), "tvl_fetch_complete") {
		t.Fatalf("expected stats log, got %s", logBuf.String())
	}
}

type slowTVLClient struct {
	delay time.Duration
}

func (s slowTVLClient) FetchProtocolTVL(ctx context.Context, slug string) (*api.ProtocolTVLResponse, error) {
	time.Sleep(s.delay)
	return &api.ProtocolTVLResponse{Name: slug}, nil
}

func TestFetchAllTVLSequentialRate(t *testing.T) {
	delay := 220 * time.Millisecond
	client := slowTVLClient{delay: delay}

	start := time.Now()
	_, err := FetchAllTVL(context.Background(), client, []string{"a", "b", "c"}, slog.Default())
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	minExpected := 2 * delay // at least two intervals between three sequential calls
	if elapsed < minExpected {
		t.Fatalf("expected sequential fetch taking >= %v, got %v", minExpected, elapsed)
	}
}
