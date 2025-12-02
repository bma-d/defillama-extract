package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
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
		generateFull: func(*aggregator.AggregationResult, []aggregator.Snapshot, *config.Config) *models.FullOutput {
			fullCalled = true
			return &models.FullOutput{}
		},
		generateSummary: func(*aggregator.AggregationResult, *config.Config) *models.SummaryOutput {
			summaryCalled = true
			return &models.SummaryOutput{}
		},
		writeOutputs: func(string, *config.Config, *models.FullOutput, *models.SummaryOutput) error {
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

func TestRunOnceSkipsWhenNoNewData(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := newLogger(buf)
	cfg := baseConfig()

	state := &stubState{state: &storage.State{LastUpdated: 123}, shouldProcess: false}
	deps := runDeps{
		client: stubClient{res: &api.FetchResult{}},
		agg:    stubAgg{result: &aggregator.AggregationResult{Timestamp: 123}},
		sm:     state,
		generateFull: func(*aggregator.AggregationResult, []aggregator.Snapshot, *config.Config) *models.FullOutput {
			t.Fatalf("generateFull should not be called")
			return nil
		},
		generateSummary: func(*aggregator.AggregationResult, *config.Config) *models.SummaryOutput {
			t.Fatalf("generateSummary should not be called")
			return nil
		},
		writeOutputs: func(string, *config.Config, *models.FullOutput, *models.SummaryOutput) error {
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
		generateFull: func(*aggregator.AggregationResult, []aggregator.Snapshot, *config.Config) *models.FullOutput {
			return &models.FullOutput{}
		},
		generateSummary: func(*aggregator.AggregationResult, *config.Config) *models.SummaryOutput {
			return &models.SummaryOutput{}
		},
		writeOutputs: func(string, *config.Config, *models.FullOutput, *models.SummaryOutput) error {
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
		generateFull: func(*aggregator.AggregationResult, []aggregator.Snapshot, *config.Config) *models.FullOutput {
			return &models.FullOutput{}
		},
		generateSummary: func(*aggregator.AggregationResult, *config.Config) *models.SummaryOutput {
			return &models.SummaryOutput{}
		},
		writeOutputs: func(string, *config.Config, *models.FullOutput, *models.SummaryOutput) error {
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
	if !strings.Contains(buf.String(), "daemon cycle completed") {
		t.Fatalf("expected completion log, got %s", buf.String())
	}
}

func TestRunDaemonWithDepsContinuesAfterStartImmediatelyFailure(t *testing.T) {
	cfg := baseConfig()
	cfg.Scheduler.Interval = time.Second
	cfg.Scheduler.StartImmediately = true

	tickCh := make(chan time.Time, 1)

	var runCount int
	buf := &bytes.Buffer{}
	logger := newLogger(buf)

	deps := daemonDeps{
		runOnce: func(context.Context, *config.Config, CLIOptions, *slog.Logger) error {
			runCount++
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

	tickCh <- time.Unix(1, 0)
	cancel()

	<-done

	if runCount != 2 {
		t.Fatalf("expected runOnce called twice (failure then tick), got %d", runCount)
	}
	if !strings.Contains(buf.String(), "start_immediately run failed") {
		t.Fatalf("expected start failure log, got %s", buf.String())
	}
	if !strings.Contains(buf.String(), "daemon cycle completed") {
		t.Fatalf("expected daemon cycle completion log, got %s", buf.String())
	}
}
