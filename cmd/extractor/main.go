package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math"
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
	"github.com/switchboard-xyz/defillama-extract/internal/tvl"
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
	tvlClient       tvl.TVLClient
	agg             aggregationPipeline
	sm              stateManager
	generateFull    func(*aggregator.AggregationResult, []aggregator.Snapshot, []aggregator.ChartDataPoint, *config.Config) *models.FullOutput
	generateSummary func(*aggregator.AggregationResult, *config.Config) *models.SummaryOutput
	writeOutputs    func(context.Context, string, *config.Config, *models.FullOutput, *models.SummaryOutput) error
	now             func() time.Time
	logger          *slog.Logger
	tvlRunner       func(context.Context, *config.Config, []api.Protocol, time.Time, CLIOptions, tvl.TVLClient, *slog.Logger) error
}

// RunOnce executes a single extraction cycle according to Story 5.2.
func RunOnce(ctx context.Context, cfg *config.Config, opts CLIOptions, logger *slog.Logger) error {
	deps := runDeps{
		client:          api.NewClient(&cfg.API, logger),
		tvlClient:       nil,
		agg:             aggregator.NewAggregator(cfg.Oracle.Name),
		sm:              storage.NewStateManager(cfg.Output.Directory, logger),
		generateFull:    storage.GenerateFullOutput,
		generateSummary: storage.GenerateSummaryOutput,
		writeOutputs:    storage.WriteAllOutputs,
		now:             time.Now,
		logger:          logger,
		tvlRunner:       nil,
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

	mainLogger := logger.With("pipeline", "main")
	tvlLogger := logger.With("pipeline", "tvl")

	checkCtx := func(stage string) error {
		if err := ctx.Err(); err != nil {
			mainLogger.Info("extraction cancelled", "stage", stage, "error", err)
			return err
		}
		return nil
	}

	start := d.now()
	mainLogger.Info("extraction started", "timestamp", start.Format(time.RFC3339))

	if err := checkCtx("start"); err != nil {
		return err
	}

	var (
		mainErr    error
		mainStatus = "success"
		protocols  []api.Protocol
		aggResult  *aggregator.AggregationResult
	)

	for {
		state, err := d.sm.LoadState()
		if err != nil {
			mainLogger.Error("extraction failed", "error", err, "duration_ms", d.now().Sub(start).Milliseconds())
			mainErr = err
			mainStatus = "failed"
			break
		}

		if err := checkCtx("after_load_state"); err != nil {
			mainErr = err
			mainStatus = "failed"
			break
		}

		result, err := d.client.FetchAll(ctx)
		if err != nil {
			mainLogger.Error("extraction failed", "error", err, "duration_ms", d.now().Sub(start).Milliseconds())
			mainErr = err
			mainStatus = "failed"
			break
		}
		protocols = result.Protocols

		if err := checkCtx("after_fetch"); err != nil {
			mainErr = err
			mainStatus = "failed"
			break
		}

		history, err := d.sm.LoadHistory()
		if err != nil {
			mainLogger.Error("extraction failed", "error", err, "duration_ms", d.now().Sub(start).Milliseconds())
			mainErr = err
			mainStatus = "failed"
			break
		}

		if err := checkCtx("after_load_history"); err != nil {
			mainErr = err
			mainStatus = "failed"
			break
		}

		chartHistory := aggregator.ExtractChartHistory(result.OracleResponse, cfg.Oracle.Name)

		aggResult = d.agg.Aggregate(ctx, result.OracleResponse, result.Protocols, history)

		// AC3: Validate protocol TVS sum vs independent reference total_value_secured (5% tolerance)
		if aggResult != nil {
			var protocolSum float64
			for _, p := range aggResult.Protocols {
				protocolSum += p.TVS
			}

			var referenceTVS float64
			if n := len(chartHistory); n > 0 {
				referenceTVS = chartHistory[n-1].TVS
			}

			switch {
			case referenceTVS <= 0:
				mainLogger.Info("tvs_sum_validation_skipped",
					"reason", "no reference total_value_secured",
					"protocols_with_tvs", aggResult.ProtocolsWithTVS,
					"protocols_without_tvs", aggResult.ProtocolsWithoutTVS,
				)
			default:
				diffPct := math.Abs(protocolSum-referenceTVS) / referenceTVS * 100
				if diffPct > 5 {
					mainLogger.Warn("tvs_sum_validation",
						"status", "fail",
						"protocols_total_tvs", protocolSum,
						"reference_total_value_secured", referenceTVS,
						"pct_diff", diffPct,
					)
				} else {
					mainLogger.Info("tvs_sum_validation",
						"status", "pass",
						"protocols_total_tvs", protocolSum,
						"reference_total_value_secured", referenceTVS,
						"pct_diff", diffPct,
					)
				}
			}
		}

		if aggResult == nil {
			mainErr = fmt.Errorf("nil aggregation result")
			mainStatus = "failed"
			break
		}

		if !d.sm.ShouldProcess(aggResult.Timestamp, state) {
			mainLogger.Info("no new data, skipping extraction",
				"last_updated", state.LastUpdated,
				"current_ts", aggResult.Timestamp,
			)
			mainStatus = "skipped"
			break
		}

		if err := checkCtx("before_snapshot"); err != nil {
			mainErr = err
			mainStatus = "failed"
			break
		}

		snapshot := storage.CreateSnapshot(aggResult)
		history = d.sm.AppendSnapshot(history, snapshot)

		if opts.DryRun {
			mainLogger.Info("dry-run mode, skipping file writes")
			mainStatus = "success"
			break
		}

		if err := checkCtx("before_writes"); err != nil {
			mainErr = err
			mainStatus = "failed"
			break
		}

		full := d.generateFull(aggResult, history, chartHistory, cfg)
		summary := d.generateSummary(aggResult, cfg)

		if err := checkCtx("after_generate_outputs"); err != nil {
			mainErr = err
			mainStatus = "failed"
			break
		}

		if err := checkCtx("before_write_outputs"); err != nil {
			mainErr = err
			mainStatus = "failed"
			break
		}

		if err := d.writeOutputs(ctx, cfg.Output.Directory, cfg, full, summary); err != nil {
			mainLogger.Error("extraction failed", "error", err, "duration_ms", d.now().Sub(start).Milliseconds())
			mainErr = err
			mainStatus = "failed"
			break
		}

		if err := checkCtx("after_write_outputs"); err != nil {
			mainErr = err
			mainStatus = "failed"
			break
		}

		newState := d.sm.UpdateState(cfg.Oracle.Name, aggResult.Timestamp, aggResult.TotalProtocols, aggResult.TotalTVS, history)

		if err := checkCtx("before_save_state"); err != nil {
			mainErr = err
			mainStatus = "failed"
			break
		}

		if err := d.sm.SaveState(newState); err != nil {
			mainLogger.Error("extraction failed", "error", err, "duration_ms", d.now().Sub(start).Milliseconds())
			mainErr = err
			mainStatus = "failed"
			break
		}

		if err := checkCtx("after_save_state"); err != nil {
			mainErr = err
			mainStatus = "failed"
			break
		}

		mainLogger.Info("extraction completed",
			"duration_ms", d.now().Sub(start).Milliseconds(),
			"protocol_count", aggResult.TotalProtocols,
			"tvs", aggResult.TotalTVS,
			"chains", len(aggResult.ActiveChains),
			"protocols_with_tvs", aggResult.ProtocolsWithTVS,
			"protocols_without_tvs", aggResult.ProtocolsWithoutTVS,
		)

		mainLogger.Info("extraction_complete",
			"protocols_with_tvs", aggResult.ProtocolsWithTVS,
			"protocols_without_tvs", aggResult.ProtocolsWithoutTVS,
		)

		mainStatus = "success"
		break
	}

	tvlStatus := "skipped"
	var tvlErr error

	if cfg != nil && cfg.TVL.Enabled {
		tvlClient := d.tvlClient
		if tvlClient == nil {
			if c, ok := d.client.(tvl.TVLClient); ok {
				tvlClient = c
			}
		}
		if tvlClient == nil {
			tvlClient = api.NewClient(&cfg.API, tvlLogger)
		}

		runner := d.tvlRunner
		if runner == nil {
			runner = func(c context.Context, cfg *config.Config, protos []api.Protocol, ts time.Time, opts CLIOptions, client tvl.TVLClient, logger *slog.Logger) error {
				return tvl.RunTVLPipeline(c, cfg, protos, ts, opts.DryRun, logger, tvl.RunnerDeps{
					Client:    client,
					OutputDir: cfg.Output.Directory,
				})
			}
		}

		tvlErr = runner(ctx, cfg, protocols, start, opts, tvlClient, tvlLogger)
		if tvlErr != nil {
			tvlStatus = "failed"
			mainLogger.Warn("tvl_pipeline_failed", "error", tvlErr)
		} else {
			tvlStatus = "success"
		}
	} else {
		tvlStatus = "disabled"
	}

	logger.Info("extraction_cycle_complete",
		"main_status", mainStatus,
		"tvl_status", tvlStatus,
		"duration_ms", d.now().Sub(start).Milliseconds(),
	)

	if mainErr != nil {
		return mainErr
	}

	return checkCtx("complete")
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

	logger.Info(fmt.Sprintf("daemon started, interval: %s", cfg.Scheduler.Interval), "interval", cfg.Scheduler.Interval, "start_immediately", cfg.Scheduler.StartImmediately)

	t := makeTicker(cfg.Scheduler.Interval)
	defer t.Stop()

	nextExtraction := func(reference time.Time) time.Time {
		return reference.Add(cfg.Scheduler.Interval)
	}

	if cfg.Scheduler.StartImmediately {
		runErr := runOnceFn(ctx, cfg, opts, logger)
		if runErr != nil {
			logger.Error("start_immediately run failed", "error", runErr)
		}
		logger.Info("next extraction at", "time", nextExtraction(now()).Format(time.RFC3339))

		if ctx.Err() != nil {
			logger.Info("shutdown signal received, finishing current extraction")
			return nil
		}
	}

	for {
		select {
		case <-ctx.Done():
			logger.Info("shutdown signal received, exiting daemon")
			return nil
		case tickTime := <-t.Chan():
			runErr := runOnceFn(ctx, cfg, opts, logger)
			if runErr != nil {
				logger.Error("daemon cycle failed", "error", runErr)
			}

			logger.Info("next extraction at", "time", nextExtraction(tickTime).Format(time.RFC3339))

			if ctx.Err() != nil {
				logger.Info("shutdown signal received, finishing current extraction")
				return nil
			}
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := RunOnce(ctx, cfg, opts, logger); err != nil {
		logger.Error("extraction failed", "error", err)
		return 1
	}

	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
