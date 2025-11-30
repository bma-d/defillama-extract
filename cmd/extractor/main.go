package main

import (
	"flag"
	"log"
	"log/slog"

	"github.com/switchboard-xyz/defillama-extract/internal/config"
	"github.com/switchboard-xyz/defillama-extract/internal/logging"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger := logging.Setup(cfg.Logging)
	slog.SetDefault(logger)

	slog.Info(
		"application started",
		"oracle", cfg.Oracle.Name,
		"log_level", cfg.Logging.Level,
	)
}
