# 17. Complete Main.go Implementation

```go
// cmd/extractor/main.go

package main

import (
    "context"
    "errors"
    "flag"
    "fmt"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/prometheus/client_golang/prometheus/promhttp"

    "github.com/yourorg/switchboard-extractor/internal/aggregator"
    "github.com/yourorg/switchboard-extractor/internal/api"
    "github.com/yourorg/switchboard-extractor/internal/config"
    "github.com/yourorg/switchboard-extractor/internal/logging"
    "github.com/yourorg/switchboard-extractor/internal/models"
    "github.com/yourorg/switchboard-extractor/internal/monitoring"
    "github.com/yourorg/switchboard-extractor/internal/storage"
)

var (
    Version   = "dev"
    BuildTime = "unknown"
)

func main() {
    // Parse command line flags
    configPath := flag.String("config", "config.yaml", "Path to config file")
    runOnce := flag.Bool("once", false, "Run once and exit")
    dryRun := flag.Bool("dry-run", false, "Fetch data but don't write files")
    showVersion := flag.Bool("version", false, "Print version and exit")
    flag.Parse()

    if *showVersion {
        fmt.Printf("switchboard-extractor %s (built %s)\n", Version, BuildTime)
        os.Exit(0)
    }

    // Load configuration
    cfg, err := config.Load(*configPath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
        os.Exit(1)
    }

    // Initialize logger
    logger := logging.NewLogger(cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.Output)
    logger.Info("starting extractor",
        slog.String("version", Version),
        slog.String("oracle", cfg.Oracle.Name),
    )

    // Create context with cancellation
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle shutdown signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        sig := <-sigChan
        logger.Info("received shutdown signal", slog.String("signal", sig.String()))
        cancel()
    }()

    // Initialize components
    apiClient := api.NewClient(api.ClientConfig{
        OracleURL:      cfg.API.OracleURL,
        ProtocolsURL:   cfg.API.ProtocolsURL,
        Timeout:        cfg.API.Timeout,
        MaxRetries:     cfg.API.MaxRetries,
        RetryBaseDelay: cfg.API.RetryBaseDelay,
        RetryMaxDelay:  cfg.API.RetryMaxDelay,
        UserAgent:      cfg.API.UserAgent,
    }, logger)

    stateManager := storage.NewStateManager(cfg.Output.Directory)
    historyManager := storage.NewHistoryManager(cfg.History.RetentionDays)

    writer, err := storage.NewWriter(cfg.Output.Directory)
    if err != nil {
        logger.Error("failed to create writer", slog.String("error", err.Error()))
        os.Exit(1)
    }

    agg := aggregator.NewAggregator(apiClient, cfg.Oracle.Name, logger)
    healthChecker := monitoring.NewHealthChecker()
    startTime := time.Now()

    // Start monitoring server
    if cfg.Monitoring.Enabled {
        go startMonitoringServer(cfg, healthChecker, startTime, logger)
    }

    // Create extraction function
    extract := func() error {
        return runExtraction(ctx, cfg, agg, stateManager, historyManager, writer, healthChecker, logger, *dryRun)
    }

    // Run extraction
    if *runOnce || !cfg.Scheduler.Enabled {
        if err := extract(); err != nil {
            if !errors.Is(err, context.Canceled) {
                logger.Error("extraction failed", slog.String("error", err.Error()))
                os.Exit(1)
            }
        }
    } else {
        // Run scheduler
        runScheduler(ctx, cfg.Scheduler.Interval, cfg.Scheduler.StartImmediately, extract, logger)
    }

    logger.Info("shutdown complete")
}

func runExtraction(
    ctx context.Context,
    cfg *config.Config,
    agg *aggregator.Aggregator,
    stateManager *storage.StateManager,
    historyManager *storage.HistoryManager,
    writer *storage.Writer,
    healthChecker *monitoring.HealthChecker,
    logger *slog.Logger,
    dryRun bool,
) error {
    startTime := time.Now()
    logger.Info("starting extraction cycle")

    // Load state
    state, err := stateManager.Load()
    if err != nil {
        if errors.Is(err, models.ErrStateCorrupted) {
            logger.Warn("state file corrupted, starting fresh")
            state = &models.State{}
        } else {
            return fmt.Errorf("loading state: %w", err)
        }
    }

    // Load history
    outputPath := fmt.Sprintf("%s/%s", cfg.Output.Directory, cfg.Output.FullFile)
    history, err := historyManager.LoadFromOutput(outputPath)
    if err != nil {
        logger.Warn("failed to load history, starting fresh", slog.String("error", err.Error()))
        history = []models.Snapshot{}
    }

    // Run aggregation
    result, err := agg.Process(ctx, history)
    if err != nil {
        healthChecker.RecordFailure(err)
        monitoring.ExtractionErrors.WithLabelValues(errorType(err)).Inc()
        return fmt.Errorf("aggregation failed: %w", err)
    }

    // Check if we should update
    if !stateManager.ShouldUpdate(state, result.LatestTimestamp) {
        logger.Info("no new data available, skipping")
        monitoring.RecordExtraction(time.Since(startTime).Seconds(), 0, 0, true)
        return nil
    }

    // Update history
    history = historyManager.Append(history, result.Snapshot)
    history = historyManager.Prune(history)
    history = historyManager.Deduplicate(history)

    // Build output
    output := buildOutput(cfg, result, history)

    // Write files (unless dry run)
    if !dryRun {
        if err := writer.WriteAll(output); err != nil {
            healthChecker.RecordFailure(err)
            return fmt.Errorf("writing output: %w", err)
        }

        // Update state
        newState := stateManager.UpdateState(
            state,
            cfg.Oracle.Name,
            result.LatestTimestamp,
            len(result.Protocols),
            result.Metrics.CurrentTVS,
            history,
        )
        if err := stateManager.Save(newState); err != nil {
            return fmt.Errorf("saving state: %w", err)
        }
    }

    // Record success
    duration := time.Since(startTime).Seconds()
    healthChecker.RecordSuccess()
    monitoring.RecordExtraction(duration, len(result.Protocols), result.Metrics.CurrentTVS, true)

    logger.Info("extraction complete",
        slog.Duration("duration", time.Since(startTime)),
        slog.Int("protocols", len(result.Protocols)),
        slog.Float64("tvs", result.Metrics.CurrentTVS),
    )

    return nil
}

func buildOutput(cfg *config.Config, result *aggregator.Result, history []models.Snapshot) *models.FullOutput {
    return &models.FullOutput{
        Version: "1.0.0",
        Oracle: models.OracleInfo{
            Name:          cfg.Oracle.Name,
            Website:       cfg.Oracle.Website,
            Documentation: cfg.Oracle.Documentation,
        },
        Metadata: models.OutputMetadata{
            LastUpdated:      time.Now().UTC().Format(time.RFC3339),
            DataSource:       "DefiLlama API",
            UpdateFrequency:  cfg.Scheduler.Interval.String(),
            ExtractorVersion: Version,
        },
        Summary: models.Summary{
            TotalValueSecured: result.Metrics.CurrentTVS,
            TotalProtocols:    result.Metrics.ProtocolCount,
            ActiveChains:      result.Metrics.ChainCount,
            Categories:        result.Metrics.Categories,
        },
        Metrics:    result.Metrics,
        Breakdown: models.Breakdown{
            ByChain:    result.ChainBreakdown,
            ByCategory: result.CategoryBreakdown,
        },
        Protocols:  result.Protocols,
        Historical: history,
    }
}

func runScheduler(ctx context.Context, interval time.Duration, startImmediately bool, fn func() error, logger *slog.Logger) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    if startImmediately {
        if err := fn(); err != nil && !errors.Is(err, context.Canceled) {
            logger.Error("extraction failed", slog.String("error", err.Error()))
        }
    }

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := fn(); err != nil && !errors.Is(err, context.Canceled) {
                logger.Error("extraction failed", slog.String("error", err.Error()))
            }
        }
    }
}

func startMonitoringServer(cfg *config.Config, healthChecker *monitoring.HealthChecker, startTime time.Time, logger *slog.Logger) {
    mux := http.NewServeMux()
    mux.Handle(cfg.Monitoring.Path, promhttp.Handler())
    mux.HandleFunc("/health", healthChecker.Handler(startTime))

    addr := fmt.Sprintf(":%d", cfg.Monitoring.Port)
    logger.Info("starting monitoring server", slog.String("addr", addr))

    if err := http.ListenAndServe(addr, mux); err != nil && !errors.Is(err, http.ErrServerClosed) {
        logger.Error("monitoring server failed", slog.String("error", err.Error()))
    }
}

func errorType(err error) string {
    var apiErr *models.APIError
    if errors.As(err, &apiErr) {
        return "api_error"
    }
    if errors.Is(err, models.ErrOracleNotFound) {
        return "oracle_not_found"
    }
    if errors.Is(err, models.ErrInvalidResponse) {
        return "invalid_response"
    }
    return "unknown"
}
```

---
