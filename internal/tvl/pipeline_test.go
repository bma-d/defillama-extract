package tvl

import (
	"context"
	"encoding/json"
	"log/slog"
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

type stubTVLClientCustomOnly struct{}

func (stubTVLClientCustomOnly) FetchProtocolTVL(ctx context.Context, slug string) (*api.ProtocolTVLResponse, error) {
	if slug == "custom" {
		return nil, nil
	}
	return &api.ProtocolTVLResponse{Name: slug, TVL: []api.TVLDataPoint{{Date: time.Now().Unix(), TotalLiquidityUSD: 1}}}, nil
}

type stubTVLClientNoCustom struct {
	t     *testing.T
	calls []string
}

func (s *stubTVLClientNoCustom) FetchProtocolTVL(ctx context.Context, slug string) (*api.ProtocolTVLResponse, error) {
	if slug == "custom" {
		s.t.Fatalf("unexpected fetch for custom slug")
	}
	s.calls = append(s.calls, slug)
	return &api.ProtocolTVLResponse{Name: slug, TVL: []api.TVLDataPoint{{Date: time.Now().Unix(), TotalLiquidityUSD: 1}}}, nil
}

func TestRunTVLPipelineWritesStateAndOutputs(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{API: config.APIConfig{}, Output: config.OutputConfig{Directory: dir}, TVL: config.TVLConfig{CustomProtocolsPath: filepath.Join(dir, "custom.json"), CustomDataPath: filepath.Join(dir, "custom-data"), Enabled: true}, Oracle: config.OracleConfig{Name: "Switchboard"}}

	loader := stubLoader{protocols: mustJSON([]map[string]interface{}{{"slug": "custom", "is-ongoing": true, "live": true, "simple-tvs-ratio": 1}})}
	if err := loader.toFile(cfg.TVL.CustomProtocolsPath); err != nil {
		t.Fatalf("setup custom file: %v", err)
	}

	// Use protocols slice with Name (slug derived) - mirrors /lite/protocols2 response
	protocols := []api.Protocol{
		{Name: "Auto Protocol", Slug: "auto-protocol", Oracles: []string{"Switchboard"}},
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
	if _, err := os.Stat(filepath.Join(dir, "custom-data.json")); err != nil {
		t.Fatalf("expected custom-data.json, got %v", err)
	}
}

func TestRunTVLPipelineDryRunSkipsWritesAndState(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{API: config.APIConfig{}, Output: config.OutputConfig{Directory: dir}, TVL: config.TVLConfig{CustomProtocolsPath: filepath.Join(dir, "custom.json"), CustomDataPath: filepath.Join(dir, "custom-data"), Enabled: true}, Oracle: config.OracleConfig{Name: "Switchboard"}}

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
	if _, err := os.Stat(filepath.Join(dir, "custom-data.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no custom-data.json on dry-run, got %v", err)
	}
}

func TestRunTVLPipeline_MergesCustomData(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{API: config.APIConfig{}, Output: config.OutputConfig{Directory: dir}, TVL: config.TVLConfig{CustomProtocolsPath: filepath.Join(dir, "custom.json"), CustomDataPath: filepath.Join(dir, "custom-data"), Enabled: true}, Oracle: config.OracleConfig{Name: "Switchboard"}}

	if err := os.MkdirAll(cfg.TVL.CustomDataPath, 0o755); err != nil {
		t.Fatalf("mkdir custom data: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfg.TVL.CustomDataPath, "custom.json"), []byte(`{"slug":"custom","url":"https://custom.example","is-ongoing":false,"live":true,"simple-tvs-ratio":1,"category":"Lending","chains":["Solana"],"tvl_history":[{"date":"2024-01-01","timestamp":1704067200,"tvl":42}]}`), 0o644); err != nil {
		t.Fatalf("write custom data: %v", err)
	}

	customProtocols := stubLoader{protocols: mustJSON([]map[string]interface{}{})}
	if err := customProtocols.toFile(cfg.TVL.CustomProtocolsPath); err != nil {
		t.Fatalf("setup custom file: %v", err)
	}

	protocols := []api.Protocol{
		{Name: "Auto Protocol", Oracles: []string{"Switchboard"}, URL: "https://auto.example"},
	}

	client := &stubTVLClientNoCustom{t: t}

	err := RunTVLPipeline(context.Background(), cfg, protocols, time.Unix(100, 0), false, nil, RunnerDeps{
		Client:           client,
		OutputDir:        dir,
		CustomDataLoader: NewCustomDataLoader(cfg.TVL.CustomDataPath, slog.Default()),
	})
	if err != nil {
		t.Fatalf("pipeline returned error: %v", err)
	}

	// custom protocol should be emitted to custom-data.json, not tvl-data.json
	tvlData, readErr := os.ReadFile(filepath.Join(dir, "tvl-data.json"))
	if readErr != nil {
		t.Fatalf("read tvl-data: %v", readErr)
	}
	var tvlOutput map[string]interface{}
	if err := json.Unmarshal(tvlData, &tvlOutput); err != nil {
		t.Fatalf("parse tvl output: %v", err)
	}
	protocolsMap, ok := tvlOutput["protocols"].(map[string]interface{})
	if !ok {
		t.Fatalf("protocols map missing")
	}
	if _, ok := protocolsMap["custom"]; ok {
		t.Fatalf("expected custom protocol excluded from tvl-data.json")
	}

	customBytes, readErr := os.ReadFile(filepath.Join(dir, "custom-data.json"))
	if readErr != nil {
		t.Fatalf("read custom-data.json: %v", readErr)
	}
	var customOutput map[string]interface{}
	if err := json.Unmarshal(customBytes, &customOutput); err != nil {
		t.Fatalf("parse custom-data output: %v", err)
	}
	customProtocolsMap, ok := customOutput["protocols"].(map[string]interface{})
	if !ok {
		t.Fatalf("custom protocols map missing")
	}
	customEntry, ok := customProtocolsMap["custom"].(map[string]interface{})
	if !ok {
		t.Fatalf("custom protocol missing in custom-data.json")
	}
	history, ok := customEntry["tvl_history"].([]interface{})
	if !ok || len(history) != 1 {
		t.Fatalf("unexpected tvl_history: %+v", customEntry["tvl_history"])
	}
	if cat, ok := customEntry["category"].(string); !ok || cat != "Lending" {
		t.Fatalf("expected category Lending, got %v", customEntry["category"])
	}
	chains, ok := customEntry["chains"].([]interface{})
	if !ok || len(chains) != 1 || chains[0].(string) != "Solana" {
		t.Fatalf("expected chains [Solana], got %v", customEntry["chains"])
	}
	if url, ok := customEntry["url"].(string); !ok || url != "https://custom.example" {
		t.Fatalf("expected url https://custom.example, got %v", customEntry["url"])
	}

	if len(client.calls) != 1 || client.calls[0] != "auto-protocol" {
		t.Fatalf("expected only auto protocol fetch, got %v", client.calls)
	}
}

func mustJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
