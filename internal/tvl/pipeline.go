package tvl

import (
	"context"
	"log/slog"
	"time"

	"github.com/switchboard-xyz/defillama-extract/internal/api"
	"github.com/switchboard-xyz/defillama-extract/internal/config"
)

// RunnerDeps captures injectable collaborators for testing.
type RunnerDeps struct {
	Client    TVLClient
	State     *TVLStateManager
	Loader    *CustomLoader
	OutputDir string
	Now       func() time.Time
}

// RunTVLPipeline orchestrates the TVL pipeline end-to-end. It is designed to
// be called after the main extraction pipeline and shares the same start
// timestamp for consistency (AC1). Errors are returned for logging but callers
// may choose to ignore them to isolate failures (AC2, AC6).
func RunTVLPipeline(ctx context.Context, cfg *config.Config, oracleResp *api.OracleAPIResponse, start time.Time, dryRun bool, logger *slog.Logger, deps RunnerDeps) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if logger == nil {
		logger = slog.Default()
	}
	if cfg == nil {
		return nil
	}

	tvlLogger := logger.With("pipeline", "tvl")

	outputDir := cfg.Output.Directory
	if deps.OutputDir != "" {
		outputDir = deps.OutputDir
	}

	stateMgr := deps.State
	if stateMgr == nil {
		stateMgr = NewTVLStateManager(outputDir, tvlLogger)
	}

	nowFn := deps.Now
	if nowFn == nil {
		nowFn = time.Now
	}

	client := deps.Client
	if client == nil {
		client = api.NewClient(&cfg.API, tvlLogger)
	}

	loader := deps.Loader
	if loader == nil {
		loader = NewCustomLoader(cfg.TVL.CustomProtocolsPath, tvlLogger)
	}

	state, err := stateMgr.LoadState()
	if err != nil {
		return err
	}

	currentTS := start.Unix()
	if !stateMgr.ShouldProcess(currentTS, state) {
		return nil
	}

	autoSlugs := GetAutoDetectedSlugs(oracleResp, cfg.Oracle.Name)

	customProtocols, err := loader.Load(ctx)
	if err != nil {
		return err
	}

	merged := MergeProtocolLists(autoSlugs, customProtocols)
	if len(merged) == 0 {
		tvlLogger.Info("tvl_no_protocols", "reason", "no auto or custom slugs")
		return nil
	}

	tvlLogger.Info("tvl_pipeline_started",
		"timestamp", start.Format(time.RFC3339),
		"auto_slugs", len(autoSlugs),
		"custom_protocols", len(customProtocols),
		"merged", len(merged),
	)

	slugs := make([]string, 0, len(merged))
	for _, p := range merged {
		slugs = append(slugs, p.Slug)
	}

	tvlData, fetchErr := FetchAllTVL(ctx, client, slugs, tvlLogger)
	if fetchErr != nil {
		tvlLogger.Warn("tvl_fetch_errors_present", "error", fetchErr)
	}

	output := GenerateTVLOutput(merged, tvlData)

	if dryRun {
		tvlLogger.Info("tvl_dry_run_skip_writes_and_state")
		return fetchErr
	}

	if err := WriteTVLOutputs(ctx, outputDir, output); err != nil {
		return err
	}

	customCount := 0
	for _, p := range merged {
		if p.Source == "custom" {
			customCount++
		}
	}

	saveErr := stateMgr.SaveState(&TVLState{
		LastUpdated:       currentTS,
		ProtocolCount:     len(merged),
		CustomProtocolCnt: customCount,
	})
	if saveErr != nil {
		return saveErr
	}

	tvlLogger.Info("tvl_pipeline_complete",
		"duration_ms", nowFn().Sub(start).Milliseconds(),
		"protocols", len(merged),
		"custom_protocols", customCount,
	)

	return fetchErr
}
