package tvl

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/switchboard-xyz/defillama-extract/internal/api"
	"github.com/switchboard-xyz/defillama-extract/internal/config"
)

type stubLoader struct {
	protocols []byte
}

func (s stubLoader) toFile(path string) error {
	return os.WriteFile(path, s.protocols, 0o644)
}

type stubTVLClientSuccess struct{}

func (stubTVLClientSuccess) FetchProtocolTVL(ctx context.Context, slug string) (*api.ProtocolTVLResponse, error) {
	return &api.ProtocolTVLResponse{Name: slug, TVL: []api.TVLDataPoint{{Date: time.Now().Unix(), TotalLiquidityUSD: 1}}}, nil
}

func TestRunTVLPipelineWritesStateAndOutputs(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{API: config.APIConfig{}, Output: config.OutputConfig{Directory: dir}, TVL: config.TVLConfig{CustomProtocolsPath: filepath.Join(dir, "custom.json"), Enabled: true}, Oracle: config.OracleConfig{Name: "Switchboard"}}

	loader := stubLoader{protocols: mustJSON([]map[string]interface{}{{"slug": "custom", "is-ongoing": true, "live": true, "simple-tvs-ratio": 1}})}
	if err := loader.toFile(cfg.TVL.CustomProtocolsPath); err != nil {
		t.Fatalf("setup custom file: %v", err)
	}

	// Use protocols slice with Name (slug derived) - mirrors /lite/protocols2 response
	protocols := []api.Protocol{
		{Name: "Auto Protocol", Oracles: []string{"Switchboard"}},
	}

	err := RunTVLPipeline(context.Background(), cfg, protocols, time.Unix(100, 0), false, nil, RunnerDeps{
		Client:    stubTVLClientSuccess{},
		OutputDir: dir,
	})

	if err != nil {
		t.Fatalf("pipeline returned error: %v", err)
	}

	statePath := filepath.Join(dir, "tvl-state.json")
	data, readErr := os.ReadFile(statePath)
	if readErr != nil {
		t.Fatalf("expected state file: %v", readErr)
	}

	var st TVLState
	if err := json.Unmarshal(data, &st); err != nil {
		t.Fatalf("parse state: %v", err)
	}
	if st.ProtocolCount != 2 {
		t.Fatalf("expected protocol count 2, got %d", st.ProtocolCount)
	}

	// outputs should exist when not dry-run
	if _, err := os.Stat(filepath.Join(dir, "tvl-data.json")); err != nil {
		t.Fatalf("expected tvl-data.json, got %v", err)
	}
}

func TestRunTVLPipelineDryRunSkipsWritesAndState(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{API: config.APIConfig{}, Output: config.OutputConfig{Directory: dir}, TVL: config.TVLConfig{CustomProtocolsPath: filepath.Join(dir, "custom.json"), Enabled: true}, Oracle: config.OracleConfig{Name: "Switchboard"}}

	loader := stubLoader{protocols: mustJSON([]map[string]interface{}{{"slug": "custom", "is-ongoing": true, "live": true, "simple-tvs-ratio": 1}})}
	if err := loader.toFile(cfg.TVL.CustomProtocolsPath); err != nil {
		t.Fatalf("setup custom file: %v", err)
	}

	// Use protocols slice with Name (slug derived) - mirrors /lite/protocols2 response
	protocols := []api.Protocol{
		{Name: "Auto Protocol", Oracles: []string{"Switchboard"}},
	}

	err := RunTVLPipeline(context.Background(), cfg, protocols, time.Unix(100, 0), true, nil, RunnerDeps{
		Client:    stubTVLClientSuccess{},
		OutputDir: dir,
	})
	if err != nil {
		t.Fatalf("pipeline returned error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "tvl-state.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no state file on dry-run, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "tvl-data.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no tvl-data.json on dry-run, got %v", err)
	}
}

func mustJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
