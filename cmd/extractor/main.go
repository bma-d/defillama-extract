package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"time"

	"syscall"

	"github.com/switchboard-xyz/defillama-extract/internal/aggregator"
	"github.com/switchboard-xyz/defillama-extract/internal/api"
	"github.com/switchboard-xyz/defillama-extract/internal/config"
	"github.com/switchboard-xyz/defillama-extract/internal/logging"
	"github.com/switchboard-xyz/defillama-extract/internal/models"
	"github.com/switchboard-xyz/defillama-extract/internal/storage"
)

const Version = "1.0.0"

type CLIOptions struct {
	Once       bool
	ConfigPath string
	DryRun     bool
	Version    bool
}

// ParseCLI parses command-line flags into CLIOptions using the stdlib flag package.
// It returns the parsed options, usage text, and any parse error.
func ParseCLI(args []string) (CLIOptions, string, error) {
	fs := flag.NewFlagSet("defillama-extract", flag.ContinueOnError)
	var usage bytes.Buffer
	fs.SetOutput(&usage)

	opts := CLIOptions{}
	fs.BoolVar(&opts.Once, "once", false, "Run single extraction and exit")
	fs.StringVar(&opts.ConfigPath, "config", "config.yaml", "Path to config file")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "Fetch and process but do not write files")
	fs.BoolVar(&opts.Version, "version", false, "Print version and exit")

	err := fs.Parse(args)
	return opts, strings.TrimSpace(usage.String()), err
}

type apiClient interface {
	FetchAll(ctx context.Context) (*api.FetchResult, error)
}

type aggregationPipeline interface {
	Aggregate(ctx context.Context, oracleResp *api.OracleAPIResponse, protocols []api.Protocol, history []aggregator.Snapshot) *aggregator.AggregationResult
}

type stateManager interface {
	LoadState() (*storage.State, error)
	ShouldProcess(currentTS int64, state *storage.State) bool
	LoadHistory() ([]aggregator.Snapshot, error)
	AppendSnapshot(history []aggregator.Snapshot, snapshot aggregator.Snapshot) []aggregator.Snapshot
	UpdateState(oracleName string, ts int64, count int, tvs float64, snapshots []aggregator.Snapshot) *storage.State
	SaveState(state *storage.State) error
}

type runDeps struct {
	client          apiClient
	agg             aggregationPipeline
	sm              stateManager
	generateFull    func(*aggregator.AggregationResult, []aggregator.Snapshot, *config.Config) *models.FullOutput
	generateSummary func(*aggregator.AggregationResult, *config.Config) *models.SummaryOutput
	writeOutputs    func(string, *config.Config, *models.FullOutput, *models.SummaryOutput) error
	now             func() time.Time
	logger          *slog.Logger
}

// RunOnce executes a single extraction cycle according to Story 5.2.
func RunOnce(ctx context.Context, cfg *config.Config, opts CLIOptions, logger *slog.Logger) error {
	deps := runDeps{
		client:          api.NewClient(&cfg.API, logger),
		agg:             aggregator.NewAggregator(cfg.Oracle.Name),
		sm:              storage.NewStateManager(cfg.Output.Directory, logger),
		generateFull:    storage.GenerateFullOutput,
		generateSummary: storage.GenerateSummaryOutput,
		writeOutputs:    storage.WriteAllOutputs,
		now:             time.Now,
		logger:          logger,
	}

	return runOnceWithDeps(ctx, cfg, opts, deps)
}

func runOnceWithDeps(ctx context.Context, cfg *config.Config, opts CLIOptions, d runDeps) error {
	if ctx == nil {
		ctx = context.Background()
	}

	logger := d.logger
	if logger == nil {
		logger = slog.Default()
	}

	start := d.now()
	logger.Info("extraction started", "timestamp", start.Format(time.RFC3339))

	state, err := d.sm.LoadState()
	if err != nil {
		logger.Error("extraction failed", "error", err, "duration_ms", d.now().Sub(start).Milliseconds())
		return err
	}

	result, err := d.client.FetchAll(ctx)
	if err != nil {
		logger.Error("extraction failed", "error", err, "duration_ms", d.now().Sub(start).Milliseconds())
		return err
	}

	history, err := d.sm.LoadHistory()
	if err != nil {
		logger.Error("extraction failed", "error", err, "duration_ms", d.now().Sub(start).Milliseconds())
		return err
	}

	aggResult := d.agg.Aggregate(ctx, result.OracleResponse, result.Protocols, history)

	if !d.sm.ShouldProcess(aggResult.Timestamp, state) {
		logger.Info("no new data, skipping extraction",
			"last_updated", state.LastUpdated,
			"current_ts", aggResult.Timestamp,
		)
		return nil
	}

	snapshot := storage.CreateSnapshot(aggResult)
	history = d.sm.AppendSnapshot(history, snapshot)

	if opts.DryRun {
		logger.Info("dry-run mode, skipping file writes")
	} else {
		full := d.generateFull(aggResult, history, cfg)
		summary := d.generateSummary(aggResult, cfg)

		if err := d.writeOutputs(cfg.Output.Directory, cfg, full, summary); err != nil {
			logger.Error("extraction failed", "error", err, "duration_ms", d.now().Sub(start).Milliseconds())
			return err
		}

		newState := d.sm.UpdateState(cfg.Oracle.Name, aggResult.Timestamp, aggResult.TotalProtocols, aggResult.TotalTVS, history)
		if err := d.sm.SaveState(newState); err != nil {
			logger.Error("extraction failed", "error", err, "duration_ms", d.now().Sub(start).Milliseconds())
			return err
		}
	}

	logger.Info("extraction completed",
		"duration_ms", d.now().Sub(start).Milliseconds(),
		"protocol_count", aggResult.TotalProtocols,
		"tvs", aggResult.TotalTVS,
		"chains", len(aggResult.ActiveChains),
	)

	return nil
}

type ticker interface {
	Chan() <-chan time.Time
	Stop()
}

type timeTicker struct {
	t *time.Ticker
}

func (t timeTicker) Chan() <-chan time.Time { return t.t.C }
func (t timeTicker) Stop()                  { t.t.Stop() }

type daemonDeps struct {
	runOnce    func(context.Context, *config.Config, CLIOptions, *slog.Logger) error
	makeTicker func(time.Duration) ticker
	now        func() time.Time
	logger     *slog.Logger
}

func runDaemonWithDeps(ctx context.Context, cfg *config.Config, opts CLIOptions, d daemonDeps) error {
	if ctx == nil {
		ctx = context.Background()
	}

	logger := d.logger
	if logger == nil {
		logger = slog.Default()
	}

	runOnceFn := d.runOnce
	if runOnceFn == nil {
		runOnceFn = RunOnce
	}

	makeTicker := d.makeTicker
	if makeTicker == nil {
		makeTicker = func(interval time.Duration) ticker {
			return timeTicker{t: time.NewTicker(interval)}
		}
	}

	now := d.now
	if now == nil {
		now = time.Now
	}

	if cfg.Scheduler.Interval <= 0 {
		return fmt.Errorf("invalid scheduler interval: %s", cfg.Scheduler.Interval)
	}

	logger.Info("daemon started", "interval", cfg.Scheduler.Interval.String(), "start_immediately", cfg.Scheduler.StartImmediately)

	t := makeTicker(cfg.Scheduler.Interval)
	defer t.Stop()

	if cfg.Scheduler.StartImmediately {
		if err := runOnceFn(ctx, cfg, opts, logger); err != nil {
			logger.Error("start_immediately run failed", "error", err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			logger.Info("shutdown signal received, exiting daemon")
			return nil
		case <-t.Chan():
			if err := runOnceFn(ctx, cfg, opts, logger); err != nil {
				logger.Error("daemon cycle failed", "error", err)
				continue
			}
			logger.Info("daemon cycle completed", "next_run_at", now().Add(cfg.Scheduler.Interval).Format(time.RFC3339))
		}
	}
}

func run(args []string, stdout, stderr io.Writer) int {
	opts, usage, err := ParseCLI(args)
	if err != nil {
		fmt.Fprintf(stderr, "invalid flags: %v\n", err)
		if usage != "" {
			fmt.Fprintln(stderr, usage)
		}
		return 2
	}
	if opts.Version {
		fmt.Fprintf(stdout, "defillama-extract v%s\n", Version)
		return 0
	}

	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		fmt.Fprintf(stderr, "failed to load config: %v\n", err)
		return 1
	}

	logger := logging.Setup(cfg.Logging)
	slog.SetDefault(logger)

	if !opts.Once {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		deps := daemonDeps{
			now:    time.Now,
			logger: logger,
		}

		if err := runDaemonWithDeps(ctx, cfg, opts, deps); err != nil {
			logger.Error("daemon failed", "error", err)
			return 1
		}

		return 0
	}

	if err := RunOnce(context.Background(), cfg, opts, logger); err != nil {
		logger.Error("extraction failed", "error", err)
		return 1
	}

	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
