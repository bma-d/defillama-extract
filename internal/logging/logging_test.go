package logging

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/switchboard-xyz/defillama-extract/internal/config"
)

func TestSetupJSONHandlerProducesStructuredOutput(t *testing.T) {
	cfg := config.LoggingConfig{Level: "info", Format: "json"}
	buf := &bytes.Buffer{}

	logger := SetupWithWriter(cfg, buf)
	logger.Info("application started", "key", "value")

	if buf.Len() == 0 {
		t.Fatal("expected JSON output, got empty buffer")
	}

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid JSON output, got error: %v", err)
	}

	if payload["msg"] != "application started" {
		t.Fatalf("msg mismatch, got %v", payload["msg"])
	}
	if payload["key"] != "value" {
		t.Fatalf("structured attribute missing, got %v", payload["key"])
	}
	if _, ok := payload["time"]; !ok {
		t.Fatal("expected timestamp field in JSON output")
	}
	if _, ok := payload["level"]; !ok {
		t.Fatal("expected level field in JSON output")
	}
}

func TestSetupTextHandlerProducesReadableOutput(t *testing.T) {
	cfg := config.LoggingConfig{Level: "debug", Format: "text"}
	buf := &bytes.Buffer{}

	logger := SetupWithWriter(cfg, buf)
	logger.Debug("debug message", "oracle", "Switchboard")

	out := buf.String()
	if !strings.Contains(out, "DEBUG") {
		t.Fatalf("expected DEBUG level in text output, got: %s", out)
	}
	if !strings.Contains(out, "debug message") {
		t.Fatalf("expected message in text output, got: %s", out)
	}
	if !strings.Contains(out, "oracle=Switchboard") {
		t.Fatalf("expected structured attr in text output, got: %s", out)
	}
}

func TestParseLevelMapping(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"INFO", slog.LevelInfo},
		{"unknown", slog.LevelInfo},
	}

	for _, tt := range tests {
		if got := ParseLevel(tt.input); got != tt.want {
			t.Fatalf("ParseLevel(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestLevelFilteringSuppressesBelowThreshold(t *testing.T) {
	cfg := config.LoggingConfig{Level: "warn", Format: "json"}
	buf := &bytes.Buffer{}

	logger := SetupWithWriter(cfg, buf)
	logger.Info("info message should be suppressed")
	logger.Warn("warn message should appear")

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	if len(lines) != 1 {
		t.Fatalf("expected only warn message to be logged, got %d lines", len(lines))
	}

	var payload map[string]any
	if err := json.Unmarshal(lines[0], &payload); err != nil {
		t.Fatalf("expected valid JSON for warn message, got error: %v", err)
	}
	if payload["msg"] != "warn message should appear" {
		t.Fatalf("unexpected message logged: %v", payload["msg"])
	}
}

func TestSetDefaultUsesConfiguredHandler(t *testing.T) {
	cfg := config.LoggingConfig{Level: "info", Format: "json"}
	buf := &bytes.Buffer{}

	logger := SetupWithWriter(cfg, buf)
	defer slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{})))
	slog.SetDefault(logger)

	slog.Info("default logger message", "log_level", cfg.Level)

	if buf.Len() == 0 {
		t.Fatal("expected default logger to write to buffer")
	}

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("expected valid JSON from default logger, got error: %v", err)
	}
	if payload["msg"] != "default logger message" {
		t.Fatalf("msg mismatch from default logger: %v", payload["msg"])
	}
	if payload["log_level"] != cfg.Level {
		t.Fatalf("expected structured attribute on default logger, got %v", payload["log_level"])
	}
}
