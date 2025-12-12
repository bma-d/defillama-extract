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
	Client           TVLClient
	State            *TVLStateManager
	Loader           *CustomLoader
	CustomDataLoader *CustomDataLoader
	OutputDir        string
	Now              func() time.Time
}

// RunTVLPipeline orchestrates the TVL pipeline end-to-end. It is designed to
// be called after the main extraction pipeline and shares the same start
// timestamp for consistency (AC1). Errors are returned for logging but callers
// may choose to ignore them to isolate failures (AC2, AC6).
//
// The protocols parameter should be the full list from /lite/protocols2,
// which includes ALL protocols using the oracle (not just those in /oracles).
func RunTVLPipeline(ctx context.Context, cfg *config.Config, protocols []api.Protocol, start time.Time, dryRun bool, logger *slog.Logger, deps RunnerDeps) error {
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

	customDataLoader := deps.CustomDataLoader
	if customDataLoader == nil {
		customDataLoader = NewCustomDataLoader(cfg.TVL.CustomDataPath, tvlLogger)
	}

	state, err := stateMgr.LoadState()
	if err != nil {
		return err
	}

	currentTS := start.Unix()
	if !stateMgr.ShouldProcess(currentTS, state) {
		return nil
	}

	autoSlugs := GetAutoDetectedSlugs(protocols, cfg.Oracle.Name)

	customProtocols, err := loader.Load(ctx)
	if err != nil {
		return err
	}

	// Build known slugs set (auto-detected + custom-protocols.json)
	knownSlugs := make(map[string]struct{}, len(autoSlugs)+len(customProtocols))
	for _, slug := range autoSlugs {
		knownSlugs[slug] = struct{}{}
	}
	customProtocolSlugs := make(map[string]struct{}, len(customProtocols))
	for _, p := range customProtocols {
		knownSlugs[p.Slug] = struct{}{}
		customProtocolSlugs[p.Slug] = struct{}{}
	}

	customDataResult, err := customDataLoader.Load(ctx, knownSlugs, customProtocolSlugs)
	if err != nil {
		return err
	}

	// Merge custom-data new protocols into customProtocols list
	allCustomProtocols := append(customProtocols, customDataResult.NewProtocols...)

	merged := MergeProtocolLists(autoSlugs, allCustomProtocols)
	if len(merged) == 0 {
		tvlLogger.Info("tvl_no_protocols", "reason", "no auto or custom slugs")
		return nil
	}

	tvlLogger.Info("tvl_pipeline_started",
		"timestamp", start.Format(time.RFC3339),
		"auto_slugs", len(autoSlugs),
		"custom_protocols", len(customProtocols),
		"custom_data_new_protocols", len(customDataResult.NewProtocols),
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

	mergedTVLData, mergeStats := mergeCustomTVLData(tvlData, customDataResult.History, tvlLogger)
	tvlLogger.Info("custom_data_merge_summary",
		"protocols_with_custom_data", mergeStats.ProtocolsWithCustomData,
		"entries_merged", mergeStats.EntriesMerged,
		"custom_only_protocols", mergeStats.CustomOnlyProtocols,
	)

	output := GenerateTVLOutput(merged, mergedTVLData)

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
