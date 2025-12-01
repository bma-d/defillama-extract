package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/switchboard-xyz/defillama-extract/internal/aggregator"
	"github.com/switchboard-xyz/defillama-extract/internal/config"
	"github.com/switchboard-xyz/defillama-extract/internal/models"
)

const (
	outputVersion          = "1.0.0"
	dataSource             = "DefiLlama API"
	extractorVersion       = "1.0.0"
	fullOutputFileName     = "switchboard-oracle-data.json"
	minifiedOutputFileName = "switchboard-oracle-data.min.json"
	summaryOutputFileName  = "switchboard-summary.json"
)

var defaultUpdateFrequency = 2 * time.Hour

func schedulerInterval(cfg *config.Config) string {
	if cfg != nil && cfg.Scheduler.Interval > 0 {
		return cfg.Scheduler.Interval.String()
	}
	return defaultUpdateFrequency.String()
}

func resolveFileName(preferred string, fallback string) string {
	if strings.TrimSpace(preferred) != "" {
		return preferred
	}
	return fallback
}

// GenerateFullOutput builds the full output structure using aggregation results,
// historical snapshots, and oracle metadata from config.
func GenerateFullOutput(result *aggregator.AggregationResult, history []aggregator.Snapshot, cfg *config.Config) *models.FullOutput {
	if result == nil {
		result = &aggregator.AggregationResult{}
	}
	if cfg == nil {
		cfg = &config.Config{}
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)

	return &models.FullOutput{
		Version: outputVersion,
		Oracle: models.OracleInfo{
			Name:          cfg.Oracle.Name,
			Website:       cfg.Oracle.Website,
			Documentation: cfg.Oracle.Documentation,
		},
		Metadata: models.OutputMetadata{
			LastUpdated:      timestamp,
			DataSource:       dataSource,
			UpdateFrequency:  schedulerInterval(cfg),
			ExtractorVersion: extractorVersion,
		},
		Summary: models.Summary{
			TotalValueSecured: result.TotalTVS,
			TotalProtocols:    result.TotalProtocols,
			ActiveChains:      result.ActiveChains,
			Categories:        result.Categories,
		},
		Metrics: models.Metrics{
			CurrentTVS:             result.TotalTVS,
			Change24h:              result.ChangeMetrics.Change24h,
			Change7d:               result.ChangeMetrics.Change7d,
			Change30d:              result.ChangeMetrics.Change30d,
			ProtocolCountChange7d:  result.ChangeMetrics.ProtocolCountChange7d,
			ProtocolCountChange30d: result.ChangeMetrics.ProtocolCountChange30d,
		},
		Breakdown: models.Breakdown{
			ByChain:    result.ChainBreakdown,
			ByCategory: result.CategoryBreakdown,
		},
		Protocols:  result.Protocols,
		Historical: history,
	}
}

// GenerateSummaryOutput builds the summary output structure without historical data.
func GenerateSummaryOutput(result *aggregator.AggregationResult, cfg *config.Config) *models.SummaryOutput {
	if result == nil {
		result = &aggregator.AggregationResult{}
	}
	if cfg == nil {
		cfg = &config.Config{}
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)
	topProtocols := result.Protocols
	if len(topProtocols) > 10 {
		topProtocols = topProtocols[:10]
	}

	return &models.SummaryOutput{
		Version: outputVersion,
		Oracle: models.OracleInfo{
			Name:          cfg.Oracle.Name,
			Website:       cfg.Oracle.Website,
			Documentation: cfg.Oracle.Documentation,
		},
		Metadata: models.OutputMetadata{
			LastUpdated:      timestamp,
			DataSource:       dataSource,
			UpdateFrequency:  schedulerInterval(cfg),
			ExtractorVersion: extractorVersion,
		},
		Summary: models.Summary{
			TotalValueSecured: result.TotalTVS,
			TotalProtocols:    result.TotalProtocols,
			ActiveChains:      result.ActiveChains,
			Categories:        result.Categories,
		},
		Metrics: models.Metrics{
			CurrentTVS:             result.TotalTVS,
			Change24h:              result.ChangeMetrics.Change24h,
			Change7d:               result.ChangeMetrics.Change7d,
			Change30d:              result.ChangeMetrics.Change30d,
			ProtocolCountChange7d:  result.ChangeMetrics.ProtocolCountChange7d,
			ProtocolCountChange30d: result.ChangeMetrics.ProtocolCountChange30d,
		},
		Breakdown: models.Breakdown{
			ByChain:    result.ChainBreakdown,
			ByCategory: result.CategoryBreakdown,
		},
		TopProtocols: topProtocols,
	}
}

// WriteJSON marshals data to JSON (indented when requested) and writes it atomically.
func WriteJSON(path string, data interface{}, indent bool) error {
	var (
		payload []byte
		err     error
	)

	if indent {
		payload, err = json.MarshalIndent(data, "", "  ")
	} else {
		payload, err = json.Marshal(data)
	}
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}

	return WriteAtomic(path, payload, 0o644)
}

// WriteAllOutputs writes full (indented), minified (compact), and summary outputs atomically.
func WriteAllOutputs(outputDir string, cfg *config.Config, full *models.FullOutput, summary *models.SummaryOutput) error {
	if cfg == nil {
		cfg = &config.Config{}
	}
	if full == nil {
		return fmt.Errorf("full output is nil")
	}
	if summary == nil {
		return fmt.Errorf("summary output is nil")
	}

	fullFile := resolveFileName(cfg.Output.FullFile, fullOutputFileName)
	minFile := resolveFileName(cfg.Output.MinFile, minifiedOutputFileName)
	summaryFile := resolveFileName(cfg.Output.SummaryFile, summaryOutputFileName)

	fullPath := filepath.Join(outputDir, fullFile)
	if err := WriteJSON(fullPath, full, true); err != nil {
		return err
	}

	minifiedPath := filepath.Join(outputDir, minFile)
	if err := WriteJSON(minifiedPath, full, false); err != nil {
		return err
	}

	summaryPath := filepath.Join(outputDir, summaryFile)
	if err := WriteJSON(summaryPath, summary, true); err != nil {
		return err
	}

	return nil
}

// WriteAtomic writes data to the target path atomically using a temp file in the
// same directory. It ensures data is synced, permissions are set, and the temp
// file is removed on error before renaming.
func WriteAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	tmpFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	cleanupNeeded := true
	defer func() {
		if cleanupNeeded {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write data: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("sync file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Chmod(tmpPath, perm); err != nil {
		return fmt.Errorf("set permissions: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename %s to %s: %w", tmpPath, path, err)
	}

	cleanupNeeded = false

	// Fsync parent directory to ensure directory entry is durably recorded.
	dirFile, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open dir for sync: %w", err)
	}
	if err := dirFile.Sync(); err != nil {
		_ = dirFile.Close()
		return fmt.Errorf("sync dir: %w", err)
	}
	if err := dirFile.Close(); err != nil {
		return fmt.Errorf("close dir: %w", err)
	}

	return nil
}
