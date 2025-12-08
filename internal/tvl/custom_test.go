package tvl

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/switchboard-xyz/defillama-extract/internal/models"
)

type recordingHandler struct {
	records []slog.Record
}

func (h *recordingHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}
func (h *recordingHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *recordingHandler) WithGroup(_ string) slog.Handler      { return h }

func TestLoad_ValidProtocols(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "tvl", "valid_custom_protocols.json")
	h := &recordingHandler{}
	logger := slog.New(h)

	loader := NewCustomLoader(path, logger)
	got, err := loader.Load(context.Background())
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 active protocols, got %d", len(got))
	}

	if got[0].Slug == "" || got[0].SimpleTVSRatio == 0 {
		t.Fatalf("protocol fields not parsed correctly: %+v", got[0])
	}

	foundSummary := false
	for _, rec := range h.records {
		if rec.Message == "custom_protocols_loaded" {
			foundSummary = true
			if rec.NumAttrs() != 3 {
				t.Fatalf("summary log attrs = %d, want 3", rec.NumAttrs())
			}
		}
	}
	if !foundSummary {
		t.Fatalf("expected custom_protocols_loaded log entry")
	}
}

func TestLoad_FileNotFound_ReturnsEmpty(t *testing.T) {
	path := filepath.Join("testdata", "tvl", "nope.json")
	h := &recordingHandler{}
	logger := slog.New(h)

	loader := NewCustomLoader(path, logger)
	got, err := loader.Load(context.Background())
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %d", len(got))
	}

	if len(h.records) == 0 || h.records[0].Message != "custom_protocols_not_found" {
		t.Fatalf("expected custom_protocols_not_found log, got %+v", h.records)
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "tvl", "invalid_custom_protocols.json")
	loader := NewCustomLoader(path, slog.Default())

	_, err := loader.Load(context.Background())
	if err == nil || err.Error() == "" {
		t.Fatal("expected parse error, got nil")
	}
	if !strings.Contains(err.Error(), "parse custom protocols") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_MissingSlugFailsValidation(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "tvl", "missing_slug.json")
	loader := NewCustomLoader(path, slog.Default())

	_, err := loader.Load(context.Background())
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if err == nil || !contains(err.Error(), "slug must not be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_RatioOutOfRange(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "tvl", "ratio_out_of_range.json")
	loader := NewCustomLoader(path, slog.Default())

	_, err := loader.Load(context.Background())
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !contains(err.Error(), "simple-tvs-ratio") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_AllLiveFalseReturnsEmpty(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "tvl", "all_live_false.json")
	h := &recordingHandler{}
	logger := slog.New(h)

	loader := NewCustomLoader(path, logger)
	got, err := loader.Load(context.Background())
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty slice when all live=false, got %d", len(got))
	}

	seen := false
	for _, rec := range h.records {
		if rec.Message == "custom_protocols_loaded" {
			seen = true
		}
	}
	if !seen {
		t.Fatalf("expected summary log for all_live_false scenario")
	}
}

func TestLoad_MissingIsOngoing(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "tvl", "missing_is_ongoing.json")
	loader := NewCustomLoader(path, slog.Default())

	_, err := loader.Load(context.Background())
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !contains(err.Error(), "is-ongoing missing") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_MissingLive(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "tvl", "missing_live.json")
	loader := NewCustomLoader(path, slog.Default())

	_, err := loader.Load(context.Background())
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !contains(err.Error(), "live missing") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_AllowsBoundaryValues(t *testing.T) {
	loader := NewCustomLoader("config.json", slog.Default())
	cp := models.CustomProtocol{
		Slug:           "boundary",
		IsOngoing:      true,
		Live:           true,
		SimpleTVSRatio: 1.0,
	}
	if err := loader.Validate(cp, true, true); err != nil {
		t.Fatalf("Validate returned error for boundary ratio: %v", err)
	}
}

func contains(s, sub string) bool {
	return strings.Contains(s, sub)
}
