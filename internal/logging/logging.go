package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/switchboard-xyz/defillama-extract/internal/config"
)

// Setup creates and returns a configured slog.Logger writing to stdout.
func Setup(cfg config.LoggingConfig) *slog.Logger {
	return SetupWithWriter(cfg, os.Stdout)
}

// SetupWithWriter mirrors Setup but allows overriding the output destination (useful for tests).
func SetupWithWriter(cfg config.LoggingConfig, w io.Writer) *slog.Logger {
	level := ParseLevel(cfg.Level)
	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	switch strings.ToLower(cfg.Format) {
	case "text":
		handler = slog.NewTextHandler(w, opts)
	default:
		handler = slog.NewJSONHandler(w, opts)
	}

	return slog.New(handler)
}

// ParseLevel converts a config level string into a slog.Level, defaulting to info.
func ParseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
