package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/switchboard-xyz/defillama-extract/internal/aggregator"
	"github.com/switchboard-xyz/defillama-extract/internal/api"
	"github.com/switchboard-xyz/defillama-extract/internal/config"
	"github.com/switchboard-xyz/defillama-extract/internal/models"
	"github.com/switchboard-xyz/defillama-extract/internal/storage"
)

type stubClient struct {
	res *api.FetchResult
	err error
}

func (s stubClient) FetchAll(ctx context.Context) (*api.FetchResult, error) {
	return s.res, s.err
}

type stubAgg struct {
	result *aggregator.AggregationResult
}

func (s stubAgg) Aggregate(ctx context.Context, oracleResp *api.OracleAPIResponse, protocols []api.Protocol, history []aggregator.Snapshot) *aggregator.AggregationResult {
	return s.result
}

type stubState struct {
	state         *storage.State
	shouldProcess bool
	loadErr       error
	history       []aggregator.Snapshot
	historyErr    error
	savedState    *storage.State
	saveErr       error
	appendHistory []aggregator.Snapshot
	saveStateHook func(*storage.State) error
}

func (s *stubState) LoadState() (*storage.State, error) {
	return s.state, s.loadErr
}

func (s *stubState) ShouldProcess(currentTS int64, _ *storage.State) bool {
	return s.shouldProcess
}

func (s *stubState) LoadHistory() ([]aggregator.Snapshot, error) {
	return s.history, s.historyErr
}

func (s *stubState) AppendSnapshot(history []aggregator.Snapshot, snapshot aggregator.Snapshot) []aggregator.Snapshot {
	s.appendHistory = append(history, snapshot)
	return s.appendHistory
}

func (s *stubState) UpdateState(oracleName string, ts int64, count int, tvs float64, snapshots []aggregator.Snapshot) *storage.State {
	return &storage.State{LastUpdated: ts, LastProtocolCount: count, LastTVS: tvs}
}

func (s *stubState) SaveState(state *storage.State) error {
	s.savedState = state
	if s.saveStateHook != nil {
		if err := s.saveStateHook(state); err != nil {
			return err
		}
	}
	return s.saveErr
}

func baseConfig() *config.Config {
	return &config.Config{
		Oracle:  config.OracleConfig{Name: "Switchboard"},
		Output:  config.OutputConfig{Directory: "data"},
		Logging: config.LoggingConfig{Level: "info", Format: "text"},
	}
}

func newLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func TestParseCLIDefaults(t *testing.T) {
	got, usage, err := ParseCLI([]string{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if usage != "" {
		t.Fatalf("expected empty usage on success, got %q", usage)
	}
	if got.Once || got.DryRun || got.Version {
		t.Fatalf("expected all flags false by default, got %+v", got)
	}
	if got.ConfigPath != "config.yaml" {
		t.Fatalf("expected default config path config.yaml, got %s", got.ConfigPath)
	}
}

func TestParseCLIFlags(t *testing.T) {
	got, _, err := ParseCLI([]string{"--once", "--config", "/tmp/app.yaml", "--dry-run", "--version"})
	if err != nil {
		t.Fatalf("unexpected error parsing flags: %v", err)
	}

	if !got.Once || !got.DryRun || !got.Version {
		t.Fatalf("expected boolean flags true, got %+v", got)
	}
	if got.ConfigPath != "/tmp/app.yaml" {
		t.Fatalf("expected config path /tmp/app.yaml, got %s", got.ConfigPath)
	}
}

func TestParseCLIRejectsUnknownFlag(t *testing.T) {
	_, usage, err := ParseCLI([]string{"--onxe"})

	if err == nil {
		t.Fatalf("expected parse error for unknown flag")
	}
	if usage == "" {
		t.Fatalf("expected usage output when parse fails")
	}
	if !strings.Contains(usage, "-once") {
		t.Fatalf("expected usage to mention known flags, got: %s", usage)
	}
}

func TestRunVersionOutput(t *testing.T) {
	var out bytes.Buffer
	code := run([]string{"--version"}, &out, &out)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if out.String() != "defillama-extract v1.0.0\n" {
		t.Fatalf("unexpected version output: %q", out.String())
	}
}

func TestRunOnceSuccess(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newLogger(buf)
	cfg := baseConfig()

	client := stubClient{res: &api.FetchResult{OracleResponse: &api.OracleAPIResponse{}, Protocols: []api.Protocol{}}}
	aggResult := &aggregator.AggregationResult{Timestamp: 100, TotalProtocols: 2, TotalTVS: 42.0, ActiveChains: []string{"sol"}}
	agg := stubAgg{result: aggResult}
	state := &stubState{state: &storage.State{}, shouldProcess: true}

	fullCalled := false
	summaryCalled := false
	wroteOutputs := false

	deps := runDeps{
		client: client,
		agg:    agg,
		sm:     state,
		generateFull: func(*aggregator.AggregationResult, []aggregator.Snapshot, []aggregator.ChartDataPoint, *config.Config) *models.FullOutput {
			fullCalled = true
			return &models.FullOutput{}
		},
		generateSummary: func(*aggregator.AggregationResult, *config.Config) *models.SummaryOutput {
			summaryCalled = true
			return &models.SummaryOutput{}
		},
		writeOutputs: func(_ context.Context, _ string, _ *config.Config, _ *models.FullOutput, _ *models.SummaryOutput) error {
			wroteOutputs = true
			return nil
		},
		now:    func() time.Time { return time.Unix(200, 0) },
		logger: logger,
	}

	if err := runOnceWithDeps(context.Background(), cfg, CLIOptions{}, deps); err != nil {
		t.Fatalf("runOnceWithDeps returned error: %v", err)
	}

	if !fullCalled || !summaryCalled || !wroteOutputs {
		t.Fatalf("expected outputs and writes to be invoked")
	}
	if state.savedState == nil || state.savedState.LastUpdated != 100 {
		t.Fatalf("state not saved with correct timestamp: %+v", state.savedState)
	}
	if !strings.Contains(buf.String(), "extraction completed") {
		t.Fatalf("expected completion log, got: %s", buf.String())
	}
}

func TestRunOnceLogsSumValidationWarning(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newLogger(buf)
	cfg := baseConfig()

	oracleResp := &api.OracleAPIResponse{
		Chart: map[string]map[string]map[string]float64{
			"1733000000": {
				"Switchboard": {"tvl": 200},
			},
		},
	}

	aggResult := &aggregator.AggregationResult{
		Timestamp:      100,
		TotalProtocols: 2,
		TotalTVS:       100,
		Protocols: []aggregator.AggregatedProtocol{
			{Name: "A", TVS: 40},
			{Name: "B", TVS: 40},
		},
	}

	deps := runDeps{
		client: stubClient{res: &api.FetchResult{OracleResponse: oracleResp, Protocols: []api.Protocol{}}},
		agg:    stubAgg{result: aggResult},
		sm:     &stubState{state: &storage.State{}, shouldProcess: true},
		generateFull: func(*aggregator.AggregationResult, []aggregator.Snapshot, []aggregator.ChartDataPoint, *config.Config) *models.FullOutput {
			return &models.FullOutput{}
		},
		generateSummary: func(*aggregator.AggregationResult, *config.Config) *models.SummaryOutput {
			return &models.SummaryOutput{}
		},
		writeOutputs: func(context.Context, string, *config.Config, *models.FullOutput, *models.SummaryOutput) error {
			return nil
		},
		now:    func() time.Time { return time.Unix(200, 0) },
		logger: logger,
	}

	if err := runOnceWithDeps(context.Background(), cfg, CLIOptions{}, deps); err != nil {
		t.Fatalf("runOnceWithDeps returned error: %v", err)
	}

	if !strings.Contains(buf.String(), "tvs_sum_validation") || !strings.Contains(buf.String(), "status=fail") {
		t.Fatalf("expected sum validation failure log, got %s", buf.String())
	}
}

func TestRunOnceSkipsSumWarningWithinTolerance(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newLogger(buf)
	cfg := baseConfig()

	oracleResp := &api.OracleAPIResponse{
		Chart: map[string]map[string]map[string]float64{
			"1733000000": {
				"Switchboard": {"tvl": 100},
			},
		},
	}

	aggResult := &aggregator.AggregationResult{
		Timestamp:      100,
		TotalProtocols: 2,
		TotalTVS:       100,
		Protocols: []aggregator.AggregatedProtocol{
			{Name: "A", TVS: 50},
			{Name: "B", TVS: 49},
		},
	}

	deps := runDeps{
		client: stubClient{res: &api.FetchResult{OracleResponse: oracleResp, Protocols: []api.Protocol{}}},
		agg:    stubAgg{result: aggResult},
		sm:     &stubState{state: &storage.State{}, shouldProcess: true},
		generateFull: func(*aggregator.AggregationResult, []aggregator.Snapshot, []aggregator.ChartDataPoint, *config.Config) *models.FullOutput {
			return &models.FullOutput{}
		},
		generateSummary: func(*aggregator.AggregationResult, *config.Config) *models.SummaryOutput {
			return &models.SummaryOutput{}
		},
		writeOutputs: func(context.Context, string, *config.Config, *models.FullOutput, *models.SummaryOutput) error {
			return nil
		},
		now:    func() time.Time { return time.Unix(200, 0) },
		logger: logger,
	}

	if err := runOnceWithDeps(context.Background(), cfg, CLIOptions{}, deps); err != nil {
		t.Fatalf("runOnceWithDeps returned error: %v", err)
	}

	if !strings.Contains(buf.String(), "tvs_sum_validation") || !strings.Contains(buf.String(), "status=pass") {
		t.Fatalf("expected sum validation pass log, got %s", buf.String())
	}
}

func TestRunOnceSkipsWhenNoNewData(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newLogger(buf)
	cfg := baseConfig()

	state := &stubState{state: &storage.State{LastUpdated: 123}, shouldProcess: false}
	deps := runDeps{
		client: stubClient{res: &api.FetchResult{}},
		agg:    stubAgg{result: &aggregator.AggregationResult{Timestamp: 123}},
		sm:     state,
		generateFull: func(*aggregator.AggregationResult, []aggregator.Snapshot, []aggregator.ChartDataPoint, *config.Config) *models.FullOutput {
			t.Fatalf("generateFull should not be called")
			return nil
		},
		generateSummary: func(*aggregator.AggregationResult, *config.Config) *models.SummaryOutput {
			t.Fatalf("generateSummary should not be called")
			return nil
		},
		writeOutputs: func(_ context.Context, _ string, _ *config.Config, _ *models.FullOutput, _ *models.SummaryOutput) error {
			t.Fatalf("writeOutputs should not be called")
			return nil
		},
		now:    func() time.Time { return time.Unix(200, 0) },
		logger: logger,
	}

	if err := runOnceWithDeps(context.Background(), cfg, CLIOptions{}, deps); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if !strings.Contains(buf.String(), "no new data, skipping extraction") {
		t.Fatalf("expected skip log, got %s", buf.String())
	}
}

func TestRunOnceDryRunSkipsWrites(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newLogger(buf)
	cfg := baseConfig()

	state := &stubState{state: &storage.State{}, shouldProcess: true}
	wroteOutputs := false
	deps := runDeps{
		client: stubClient{res: &api.FetchResult{}},
		agg:    stubAgg{result: &aggregator.AggregationResult{Timestamp: 300}},
		sm:     state,
		generateFull: func(*aggregator.AggregationResult, []aggregator.Snapshot, []aggregator.ChartDataPoint, *config.Config) *models.FullOutput {
			return &models.FullOutput{}
		},
		generateSummary: func(*aggregator.AggregationResult, *config.Config) *models.SummaryOutput {
			return &models.SummaryOutput{}
		},
		writeOutputs: func(_ context.Context, _ string, _ *config.Config, _ *models.FullOutput, _ *models.SummaryOutput) error {
			wroteOutputs = true
			return nil
		},
		now:    func() time.Time { return time.Unix(400, 0) },
		logger: logger,
	}

	if err := runOnceWithDeps(context.Background(), cfg, CLIOptions{DryRun: true}, deps); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if wroteOutputs {
		t.Fatalf("expected outputs not to be written in dry-run mode")
	}
	if state.savedState != nil {
		t.Fatalf("expected state not to be saved in dry-run mode, got %+v", state.savedState)
	}
	if !strings.Contains(buf.String(), "dry-run mode, skipping file writes") {
		t.Fatalf("expected dry-run log, got %s", buf.String())
	}
}

func TestRunExitsOnFlagError(t *testing.T) {
	var out bytes.Buffer
	var errBuf bytes.Buffer
	code := run([]string{"--onxe"}, &out, &errBuf)

	if code != 2 {
		t.Fatalf("expected exit code 2 on flag error, got %d", code)
	}
	if out.Len() != 0 {
		t.Fatalf("expected no stdout output on flag error, got %q", out.String())
	}
	if !strings.Contains(errBuf.String(), "invalid flags") {
		t.Fatalf("expected error output, got %q", errBuf.String())
	}
}

func TestRunOncePropagatesWriteError(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newLogger(buf)
	cfg := baseConfig()

	deps := runDeps{
		client: stubClient{res: &api.FetchResult{}},
		agg:    stubAgg{result: &aggregator.AggregationResult{Timestamp: 10}},
		sm:     &stubState{state: &storage.State{}, shouldProcess: true},
		generateFull: func(*aggregator.AggregationResult, []aggregator.Snapshot, []aggregator.ChartDataPoint, *config.Config) *models.FullOutput {
			return &models.FullOutput{}
		},
		generateSummary: func(*aggregator.AggregationResult, *config.Config) *models.SummaryOutput {
			return &models.SummaryOutput{}
		},
		writeOutputs: func(_ context.Context, _ string, _ *config.Config, _ *models.FullOutput, _ *models.SummaryOutput) error {
			return errors.New("write failed")
		},
		now:    func() time.Time { return time.Unix(0, 0) },
		logger: logger,
	}

	err := runOnceWithDeps(context.Background(), cfg, CLIOptions{}, deps)
	if err == nil || !strings.Contains(err.Error(), "write failed") {
		t.Fatalf("expected write error, got %v", err)
	}
	if !strings.Contains(buf.String(), "extraction failed") {
		t.Fatalf("expected failure log, got %s", buf.String())
	}
}

type stubTicker struct {
	ch <-chan time.Time
}

func (t *stubTicker) Chan() <-chan time.Time { return t.ch }
func (t *stubTicker) Stop()                  {}

func TestRunDaemonWithDepsRunsOnTickAndStopsOnCancel(t *testing.T) {
	cfg := baseConfig()
	cfg.Scheduler.Interval = time.Second
	cfg.Scheduler.StartImmediately = false

	tickCh := make(chan time.Time, 1)
	runCalled := make(chan struct{}, 1)

	var runCount int
	buf := &bytes.Buffer{}
	logger := newLogger(buf)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	deps := daemonDeps{
		runOnce: func(context.Context, *config.Config, CLIOptions, *slog.Logger) error {
			runCount++
			runCalled <- struct{}{}
			return nil
		},
		makeTicker: func(time.Duration) ticker {
			return &stubTicker{ch: tickCh}
		},
		now:    func() time.Time { return time.Unix(0, 0) },
		logger: logger,
	}

	done := make(chan struct{})
	go func() {
		_ = runDaemonWithDeps(ctx, cfg, CLIOptions{}, deps)
		close(done)
	}()

	tickCh <- time.Unix(1, 0)
	<-runCalled
	cancel()

	<-done

	if runCount != 1 {
		t.Fatalf("expected runOnce called once, got %d", runCount)
	}
	if !strings.Contains(buf.String(), "next extraction at") {
		t.Fatalf("expected next extraction log, got %s", buf.String())
	}
}

func TestRunDaemonWithDepsContinuesAfterStartImmediatelyFailure(t *testing.T) {
	cfg := baseConfig()
	cfg.Scheduler.Interval = time.Second
	cfg.Scheduler.StartImmediately = true

	tickCh := make(chan time.Time, 1)
	runCalled := make(chan struct{}, 2)

	var runCount int
	buf := &bytes.Buffer{}
	logger := newLogger(buf)

	deps := daemonDeps{
		runOnce: func(context.Context, *config.Config, CLIOptions, *slog.Logger) error {
			runCount++
			runCalled <- struct{}{}
			if runCount == 1 {
				return errors.New("boom")
			}
			return nil
		},
		makeTicker: func(time.Duration) ticker {
			return &stubTicker{ch: tickCh}
		},
		now:    func() time.Time { return time.Unix(0, 0) },
		logger: logger,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = runDaemonWithDeps(ctx, cfg, CLIOptions{}, deps)
		close(done)
	}()

	<-runCalled // start_immediately
	tickCh <- time.Unix(1, 0)
	<-runCalled // tick run
	cancel()

	<-done

	if runCount != 2 {
		t.Fatalf("expected runOnce called twice (failure then tick), got %d", runCount)
	}
	if !strings.Contains(buf.String(), "start_immediately run failed") {
		t.Fatalf("expected start failure log, got %s", buf.String())
	}
	if !strings.Contains(buf.String(), "next extraction at") {
		t.Fatalf("expected next extraction log, got %s", buf.String())
	}
}

func TestRunDaemonLogsNextExtractionAfterFailure(t *testing.T) {
	cfg := baseConfig()
	cfg.Scheduler.Interval = time.Second

	tickCh := make(chan time.Time, 1)
	runCalled := make(chan struct{}, 1)
	buf := &bytes.Buffer{}
	logger := newLogger(buf)

	deps := daemonDeps{
		runOnce: func(context.Context, *config.Config, CLIOptions, *slog.Logger) error {
			runCalled <- struct{}{}
			return errors.New("boom")
		},
		makeTicker: func(time.Duration) ticker { return &stubTicker{ch: tickCh} },
		now:        func() time.Time { return time.Unix(0, 0) },
		logger:     logger,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = runDaemonWithDeps(ctx, cfg, CLIOptions{}, deps)
		close(done)
	}()

	tickCh <- time.Unix(10, 0)
	<-runCalled
	cancel()
	<-done

	if !strings.Contains(buf.String(), "daemon cycle failed") {
		t.Fatalf("expected failure log, got %s", buf.String())
	}
	if !strings.Contains(buf.String(), "next extraction at") {
		t.Fatalf("expected next extraction log even after failure, got %s", buf.String())
	}
}

func TestRunDaemonShutdownDuringExtractionFinishesCurrentRun(t *testing.T) {
	cfg := baseConfig()
	cfg.Scheduler.Interval = time.Second

	tickCh := make(chan time.Time, 1)
	runStarted := make(chan struct{})
	release := make(chan struct{})

	buf := &bytes.Buffer{}
	logger := newLogger(buf)

	deps := daemonDeps{
		runOnce: func(context.Context, *config.Config, CLIOptions, *slog.Logger) error {
			close(runStarted)
			<-release
			return nil
		},
		makeTicker: func(time.Duration) ticker { return &stubTicker{ch: tickCh} },
		now:        func() time.Time { return time.Unix(0, 0) },
		logger:     logger,
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = runDaemonWithDeps(ctx, cfg, CLIOptions{}, deps)
		close(done)
	}()

	tickCh <- time.Unix(5, 0)
	<-runStarted
	cancel()
	close(release)
	<-done

	if !strings.Contains(buf.String(), "shutdown signal received, finishing current extraction") {
		t.Fatalf("expected finishing log, got %s", buf.String())
	}
}

func TestRunOnceWithCanceledContextSkipsWrites(t *testing.T) {
	cfg := baseConfig()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	wroteOutputs := false
	deps := runDeps{
		client: stubClient{res: &api.FetchResult{}},
		agg:    stubAgg{result: &aggregator.AggregationResult{Timestamp: 1}},
		sm:     &stubState{state: &storage.State{}, shouldProcess: true},
		generateFull: func(*aggregator.AggregationResult, []aggregator.Snapshot, []aggregator.ChartDataPoint, *config.Config) *models.FullOutput {
			wroteOutputs = true
			return &models.FullOutput{}
		},
		generateSummary: func(*aggregator.AggregationResult, *config.Config) *models.SummaryOutput {
			wroteOutputs = true
			return &models.SummaryOutput{}
		},
		writeOutputs: func(_ context.Context, _ string, _ *config.Config, _ *models.FullOutput, _ *models.SummaryOutput) error {
			wroteOutputs = true
			return nil
		},
		now:    func() time.Time { return time.Unix(0, 0) },
		logger: newLogger(&bytes.Buffer{}),
	}

	if err := runOnceWithDeps(ctx, cfg, CLIOptions{}, deps); err == nil {
		t.Fatalf("expected cancellation error")
	}

	if wroteOutputs {
		t.Fatalf("expected writes to be skipped on cancellation")
	}
}

func TestRunOnceCancellationDuringProcessingSkipsWrites(t *testing.T) {
	cfg := baseConfig()
	ctx, cancel := context.WithCancel(context.Background())

	wroteOutputs := false
	savedState := false

	state := &stubState{
		state:         &storage.State{},
		shouldProcess: true,
		saveStateHook: func(*storage.State) error {
			savedState = true
			return nil
		},
	}
	deps := runDeps{
		client: stubClient{res: &api.FetchResult{OracleResponse: &api.OracleAPIResponse{}, Protocols: []api.Protocol{}}},
		agg:    stubAgg{result: &aggregator.AggregationResult{Timestamp: 10, TotalProtocols: 1}},
		sm:     state,
		generateFull: func(*aggregator.AggregationResult, []aggregator.Snapshot, []aggregator.ChartDataPoint, *config.Config) *models.FullOutput {
			cancel()
			return &models.FullOutput{}
		},
		generateSummary: func(*aggregator.AggregationResult, *config.Config) *models.SummaryOutput {
			return &models.SummaryOutput{}
		},
		writeOutputs: func(_ context.Context, _ string, _ *config.Config, _ *models.FullOutput, _ *models.SummaryOutput) error {
			wroteOutputs = true
			return nil
		},
		now:    func() time.Time { return time.Unix(0, 0) },
		logger: newLogger(&bytes.Buffer{}),
	}

	err := runOnceWithDeps(ctx, cfg, CLIOptions{}, deps)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if wroteOutputs {
		t.Fatalf("expected writes to be skipped when context cancels during processing")
	}
	if savedState {
		t.Fatalf("expected state save to be skipped on cancellation")
	}
}

func TestRunOnceCancellationDuringWriteOutputsLeavesNoFiles(t *testing.T) {
	cfg := baseConfig()
	dir := t.TempDir()
	cfg.Output.Directory = dir
	cfg.Output.FullFile = "full.json"
	cfg.Output.MinFile = "min.json"
	cfg.Output.SummaryFile = "summary.json"

	ctx, cancel := context.WithCancel(context.Background())

	state := &stubState{state: &storage.State{}, shouldProcess: true}
	full := &models.FullOutput{Version: "v1", Metadata: models.OutputMetadata{LastUpdated: time.Now().Format(time.RFC3339)}}
	summary := &models.SummaryOutput{Version: "v1", Metadata: models.OutputMetadata{LastUpdated: time.Now().Format(time.RFC3339)}}

	deps := runDeps{
		client: stubClient{res: &api.FetchResult{OracleResponse: &api.OracleAPIResponse{}, Protocols: []api.Protocol{}}},
		agg:    stubAgg{result: &aggregator.AggregationResult{Timestamp: 50, TotalProtocols: 1}},
		sm:     state,
		generateFull: func(*aggregator.AggregationResult, []aggregator.Snapshot, []aggregator.ChartDataPoint, *config.Config) *models.FullOutput {
			return full
		},
		generateSummary: func(*aggregator.AggregationResult, *config.Config) *models.SummaryOutput {
			return summary
		},
		writeOutputs: func(ctx context.Context, outputDir string, cfg *config.Config, full *models.FullOutput, summary *models.SummaryOutput) error {
			cancel()
			return storage.WriteAllOutputs(ctx, outputDir, cfg, full, summary)
		},
		now:    func() time.Time { return time.Unix(0, 0) },
		logger: newLogger(&bytes.Buffer{}),
	}

	err := runOnceWithDeps(ctx, cfg, CLIOptions{}, deps)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation error, got %v", err)
	}

	for _, name := range []string{cfg.Output.FullFile, cfg.Output.MinFile, cfg.Output.SummaryFile} {
		if _, statErr := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(statErr) {
			t.Fatalf("expected no file written on cancel: %s", name)
		}
	}

	if state.savedState != nil {
		t.Fatalf("expected state not to be saved after cancellation")
	}
}

func TestRunDaemonWithDepsStartImmediatelyThenTickHonorsContext(t *testing.T) {
	cfg := baseConfig()
	cfg.Scheduler.Interval = time.Second
	cfg.Scheduler.StartImmediately = true

	tickCh := make(chan time.Time, 1)
	runCalled := make(chan struct{}, 2)

	buf := &bytes.Buffer{}
	logger := newLogger(buf)

	var runCount int
	deps := daemonDeps{
		runOnce: func(context.Context, *config.Config, CLIOptions, *slog.Logger) error {
			runCount++
			runCalled <- struct{}{}
			return nil
		},
		makeTicker: func(time.Duration) ticker { return &stubTicker{ch: tickCh} },
		now:        func() time.Time { return time.Unix(0, 0) },
		logger:     logger,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		_ = runDaemonWithDeps(ctx, cfg, CLIOptions{}, deps)
		close(done)
	}()

	<-runCalled // start_immediately

	tickCh <- time.Unix(1, 0)
	<-runCalled // first tick

	cancel()
	<-done

	if runCount != 2 {
		t.Fatalf("expected 2 runs (immediate + tick), got %d", runCount)
	}
	if !strings.Contains(buf.String(), "daemon started") {
		t.Fatalf("expected daemon start log, got %s", buf.String())
	}
	if !strings.Contains(buf.String(), "next extraction at") {
		t.Fatalf("expected next extraction log, got %s", buf.String())
	}
}

func TestRunDaemonWithDepsCancelsWhileWaiting(t *testing.T) {
	cfg := baseConfig()
	cfg.Scheduler.Interval = time.Second

	tickCh := make(chan time.Time)
	var runCount int

	deps := daemonDeps{
		runOnce: func(context.Context, *config.Config, CLIOptions, *slog.Logger) error {
			runCount++
			return nil
		},
		makeTicker: func(time.Duration) ticker { return &stubTicker{ch: tickCh} },
		now:        func() time.Time { return time.Unix(0, 0) },
		logger:     newLogger(&bytes.Buffer{}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = runDaemonWithDeps(ctx, cfg, CLIOptions{}, deps)
		close(done)
	}()

	cancel()
	<-done

	if runCount != 0 {
		t.Fatalf("expected no runs when cancelled while waiting, got %d", runCount)
	}
}

func TestRunOnceCancellationBetweenOutputsAndWritesSkipsSideEffects(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := baseConfig()

	wroteOutputs := false
	savedState := false

	deps := runDeps{
		client: stubClient{res: &api.FetchResult{}},
		agg:    stubAgg{result: &aggregator.AggregationResult{Timestamp: 5, TotalProtocols: 1}},
		sm: &stubState{
			state:         &storage.State{},
			shouldProcess: true,
			saveStateHook: func(*storage.State) error {
				savedState = true
				return nil
			},
		},
		generateFull: func(*aggregator.AggregationResult, []aggregator.Snapshot, []aggregator.ChartDataPoint, *config.Config) *models.FullOutput {
			return &models.FullOutput{}
		},
		generateSummary: func(*aggregator.AggregationResult, *config.Config) *models.SummaryOutput {
			cancel() // cancel after outputs produced, before writes
			return &models.SummaryOutput{}
		},
		writeOutputs: func(_ context.Context, _ string, _ *config.Config, _ *models.FullOutput, _ *models.SummaryOutput) error {
			wroteOutputs = true
			return nil
		},
		now:    func() time.Time { return time.Unix(0, 0) },
		logger: newLogger(&bytes.Buffer{}),
	}

	err := runOnceWithDeps(ctx, cfg, CLIOptions{}, deps)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation error, got %v", err)
	}
	if wroteOutputs {
		t.Fatalf("expected writeOutputs not to be called after cancellation")
	}
	if savedState {
		t.Fatalf("expected SaveState not to be called after cancellation")
	}
}

func TestDaemonIntegration_StartImmediatelyFalse_ShutdownWhileWaiting(t *testing.T) {
	cfg := baseConfig()
	cfg.Scheduler.Interval = 10 * time.Millisecond

	var runCount int
	runCalled := make(chan struct{}, 1)
	loggerBuf := &bytes.Buffer{}

	deps := daemonDeps{
		runOnce: func(context.Context, *config.Config, CLIOptions, *slog.Logger) error {
			runCount++
			runCalled <- struct{}{}
			return nil
		},
		makeTicker: func(d time.Duration) ticker { return timeTicker{t: time.NewTicker(d)} },
		now:        time.Now,
		logger:     newLogger(loggerBuf),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	done := make(chan struct{})
	go func() {
		_ = runDaemonWithDeps(ctx, cfg, CLIOptions{}, deps)
		close(done)
	}()

	<-runCalled // first tick execution
	stop()      // simulate SIGINT while waiting for next tick
	<-done

	if runCount != 1 {
		t.Fatalf("expected exactly one run before shutdown, got %d", runCount)
	}
	if !strings.Contains(loggerBuf.String(), "shutdown signal received") {
		t.Fatalf("expected shutdown log, got %s", loggerBuf.String())
	}
}

func TestDaemonIntegration_StartImmediatelyTrue_ShutdownDuringRun(t *testing.T) {
	cfg := baseConfig()
	cfg.Scheduler.Interval = 50 * time.Millisecond
	cfg.Scheduler.StartImmediately = true

	runStarted := make(chan struct{})
	release := make(chan struct{})
	loggerBuf := &bytes.Buffer{}

	deps := daemonDeps{
		runOnce: func(context.Context, *config.Config, CLIOptions, *slog.Logger) error {
			close(runStarted)
			<-release
			return nil
		},
		makeTicker: func(d time.Duration) ticker { return timeTicker{t: time.NewTicker(d)} },
		now:        time.Now,
		logger:     newLogger(loggerBuf),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		_ = runDaemonWithDeps(ctx, cfg, CLIOptions{}, deps)
		close(done)
	}()

	<-runStarted
	stop() // cancel while runOnce is executing
	close(release)
	<-done

	if !strings.Contains(loggerBuf.String(), "shutdown signal received") {
		t.Fatalf("expected shutdown log, got %s", loggerBuf.String())
	}
}

func TestDaemonIntegration_ErrorRecoveryContinues(t *testing.T) {
	cfg := baseConfig()
	cfg.Scheduler.Interval = 10 * time.Millisecond

	var runCount int
	runCalled := make(chan struct{}, 2)
	loggerBuf := &bytes.Buffer{}

	deps := daemonDeps{
		runOnce: func(context.Context, *config.Config, CLIOptions, *slog.Logger) error {
			runCount++
			runCalled <- struct{}{}
			if runCount == 1 {
				return errors.New("boom")
			}
			return nil
		},
		makeTicker: func(d time.Duration) ticker { return timeTicker{t: time.NewTicker(d)} },
		now:        time.Now,
		logger:     newLogger(loggerBuf),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	done := make(chan struct{})
	go func() {
		_ = runDaemonWithDeps(ctx, cfg, CLIOptions{}, deps)
		close(done)
	}()

	<-runCalled // first (fails)
	<-runCalled // second (succeeds)
	stop()
	<-done

	if runCount < 2 {
		t.Fatalf("expected at least two runs despite first failure, got %d", runCount)
	}
	if !strings.Contains(loggerBuf.String(), "daemon cycle failed") {
		t.Fatalf("expected daemon failure log, got %s", loggerBuf.String())
	}
	if !strings.Contains(loggerBuf.String(), "next extraction at") {
		t.Fatalf("expected next extraction log, got %s", loggerBuf.String())
	}
}
