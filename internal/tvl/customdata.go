package tvl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/switchboard-xyz/defillama-extract/internal/models"
)

// CustomDataStats captures loader metrics for observability and testing.
type CustomDataStats struct {
	FilesScanned  int
	FilesLoaded   int
	EntriesLoaded int
	InvalidFiles  int
}

// CustomDataLoader loads per-protocol TVL history from a folder of JSON files.
// Files prefixed with "_" are ignored to allow templates (e.g., _example.json.template).
type CustomDataLoader struct {
	path      string
	logger    *slog.Logger
	lastStats CustomDataStats
}

// NewCustomDataLoader constructs a loader rooted at the given path.
func NewCustomDataLoader(path string, logger *slog.Logger) *CustomDataLoader {
	if logger == nil {
		logger = slog.Default()
	}
	return &CustomDataLoader{path: path, logger: logger}
}

// Path returns the configured directory path.
func (l *CustomDataLoader) Path() string {
	return l.path
}

// Stats returns metrics from the most recent Load call.
func (l *CustomDataLoader) Stats() CustomDataStats {
	return l.lastStats
}

// customDataFile represents a custom-data JSON file. It supports two modes:
// 1. History-only: existing protocol (in custom-protocols.json or auto-detected) - only slug + tvl_history required
// 2. Full protocol: NEW protocol - requires slug, is-ongoing, live, simple-tvs-ratio, category, chains, tvl_history
type customDataFile struct {
	Slug           string                  `json:"slug"`
	IsOngoing      *bool                   `json:"is-ongoing,omitempty"`
	Live           *bool                   `json:"live,omitempty"`
	SimpleTVSRatio *float64                `json:"simple-tvs-ratio,omitempty"`
	URL            *string                 `json:"url,omitempty"`
	IsDefillama    *bool                   `json:"is-defillama,omitempty"`
	DocsProof      *string                 `json:"docs_proof,omitempty"`
	GitHubProof    *string                 `json:"github_proof,omitempty"`
	Category       string                  `json:"category,omitempty"`
	Chains         []string                `json:"chains,omitempty"`
	TVLHistory     []models.TVLHistoryItem `json:"tvl_history"`
}

// hasMetadata returns true if any protocol metadata fields are set (beyond slug + tvl_history)
func (f *customDataFile) hasMetadata() bool {
	return f.IsOngoing != nil || f.Live != nil || f.SimpleTVSRatio != nil ||
		f.IsDefillama != nil || f.DocsProof != nil || f.GitHubProof != nil ||
		f.Category != "" || len(f.Chains) > 0
}

// toCustomProtocol converts to models.CustomProtocol. Only call if hasMetadata() is true.
func (f *customDataFile) toCustomProtocol() models.CustomProtocol {
	p := models.CustomProtocol{
		Slug:        f.Slug,
		URL:         derefString(f.URL),
		DocsProof:   f.DocsProof,
		GitHubProof: f.GitHubProof,
		Category:    f.Category,
		Chains:      f.Chains,
	}
	if f.IsOngoing != nil {
		p.IsOngoing = *f.IsOngoing
	}
	if f.Live != nil {
		p.Live = *f.Live
	}
	if f.SimpleTVSRatio != nil {
		p.SimpleTVSRatio = *f.SimpleTVSRatio
	}
	if f.IsDefillama != nil {
		p.IsDefillama = f.IsDefillama
	}
	return p
}

// CustomDataResult holds the output of Load(): TVL history by slug and any new protocols defined.
type CustomDataResult struct {
	History      map[string][]models.TVLHistoryItem
	NewProtocols []models.CustomProtocol
	Metadata     map[string]CustomDataAttributes
}

// CustomDataAttributes preserves optional metadata provided in custom-data
// files for downstream outputs (e.g., category, chains).
type CustomDataAttributes struct {
	Category string
	Chains   []string
	URL      string
}

// Load reads all *.json files under the directory (non-recursive), validates
// schema, and returns TVL history plus any new protocol definitions.
//
// knownSlugs contains slugs already registered (from custom-protocols.json or auto-detected).
// customProtocolSlugs contains slugs specifically from custom-protocols.json (for duplicate detection).
//
// Validation rules:
// - If slug in knownSlugs: only slug + tvl_history required (history-only mode)
// - If slug NOT in knownSlugs: slug, is-ongoing, live, simple-tvs-ratio, tvl_history required (new protocol)
// - If slug in customProtocolSlugs AND file has metadata: PANIC (duplicate registration)
//
// Invalid files are logged as warnings and skipped (AC4).
func (l *CustomDataLoader) Load(ctx context.Context, knownSlugs, customProtocolSlugs map[string]struct{}) (*CustomDataResult, error) {
	if l == nil {
		return nil, errors.New("nil CustomDataLoader")
	}
	l.lastStats = CustomDataStats{}

	if strings.TrimSpace(l.path) == "" {
		return nil, errors.New("custom data path must not be empty")
	}

	result := &CustomDataResult{
		History:      make(map[string][]models.TVLHistoryItem),
		NewProtocols: make([]models.CustomProtocol, 0),
		Metadata:     make(map[string]CustomDataAttributes),
	}

	dirEntries, err := os.ReadDir(l.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			l.logger.InfoContext(ctx, "custom_data_not_found", "path", l.path, "reason", "directory not found")
			return result, nil
		}
		return nil, fmt.Errorf("read custom data dir: %w", err)
	}

	l.lastStats.FilesScanned = len(dirEntries)

	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}
		name := de.Name()
		if strings.HasPrefix(name, "_") || filepath.Ext(name) != ".json" {
			continue
		}

		fullPath := filepath.Join(l.path, name)
		data, readErr := os.ReadFile(fullPath)
		if readErr != nil {
			l.logger.WarnContext(ctx, "custom_data_read_failed", "path", fullPath, "error", readErr)
			l.lastStats.InvalidFiles++
			continue
		}

		var file customDataFile
		if err := json.Unmarshal(data, &file); err != nil {
			l.logger.WarnContext(ctx, "custom_data_invalid_json", "path", fullPath, "error", err)
			l.lastStats.InvalidFiles++
			continue
		}

		var fields map[string]json.RawMessage
		if err := json.Unmarshal(data, &fields); err != nil {
			l.logger.WarnContext(ctx, "custom_data_invalid_json", "path", fullPath, "error", err)
			l.lastStats.InvalidFiles++
			continue
		}
		_, hasURL := fields["url"]

		slug := strings.TrimSpace(file.Slug)
		_, isKnown := knownSlugs[slug]
		_, isCustomProtocol := customProtocolSlugs[slug]

		// Duplicate detection: panic if slug in custom-protocols.json AND has metadata
		if isCustomProtocol && file.hasMetadata() {
			panic(fmt.Sprintf("duplicate protocol registration: slug %q exists in custom-protocols.json AND custom-data file %s has metadata fields; remove metadata from custom-data or remove from custom-protocols.json", slug, fullPath))
		}

		// Validate based on mode
		if err := validateCustomDataFileWithContext(file, isKnown, hasURL); err != nil {
			l.logger.WarnContext(ctx, "custom_data_invalid_schema", "path", fullPath, "error", err)
			l.lastStats.InvalidFiles++
			continue
		}

		normalized := normalizeHistory(file.TVLHistory)
		result.History[slug] = normalized
		result.Metadata[slug] = CustomDataAttributes{
			Category: file.Category,
			Chains:   file.Chains,
			URL:      derefString(file.URL),
		}
		l.lastStats.FilesLoaded++
		l.lastStats.EntriesLoaded += len(normalized)

		// If new protocol with metadata, add to NewProtocols (only if live, matching custom.go filtering)
		if !isKnown && file.hasMetadata() {
			p := file.toCustomProtocol()
			if p.Live {
				result.NewProtocols = append(result.NewProtocols, p)
			} else {
				l.logger.InfoContext(ctx, "custom_data_skipped_not_live", "slug", slug, "path", fullPath)
			}
		}
	}

	l.logger.InfoContext(ctx, "custom_data_loaded",
		"path", l.path,
		"files_scanned", l.lastStats.FilesScanned,
		"files_loaded", l.lastStats.FilesLoaded,
		"entries_loaded", l.lastStats.EntriesLoaded,
		"invalid_files", l.lastStats.InvalidFiles,
		"new_protocols", len(result.NewProtocols),
	)

	return result, nil
}

// validateCustomDataFileWithContext validates a custom data file based on whether it's for a known protocol.
// - Known protocols: slug + tvl_history required, url field must be present (empty string allowed)
// - New protocols: slug, is-ongoing, live, simple-tvs-ratio, category, chains, tvl_history, url required
func validateCustomDataFileWithContext(file customDataFile, isKnown bool, hasURL bool) error {
	if strings.TrimSpace(file.Slug) == "" {
		return errors.New("slug is required")
	}
	if !hasURL {
		return errors.New("url is required (may be empty string if unknown)")
	}
	if len(file.TVLHistory) == 0 {
		return errors.New("tvl_history must not be empty")
	}
	for i, item := range file.TVLHistory {
		if strings.TrimSpace(item.Date) == "" && item.Timestamp == 0 {
			return fmt.Errorf("entry %d missing date and timestamp", i)
		}
		if item.Timestamp < 0 {
			return fmt.Errorf("entry %d has negative timestamp", i)
		}
	}

	// New protocols require mandatory metadata fields
	if !isKnown && file.hasMetadata() {
		if file.IsOngoing == nil {
			return errors.New("is-ongoing is required for new protocols")
		}
		if file.Live == nil {
			return errors.New("live is required for new protocols")
		}
		if file.SimpleTVSRatio == nil {
			return errors.New("simple-tvs-ratio is required for new protocols")
		}
		if *file.SimpleTVSRatio < 0 || *file.SimpleTVSRatio > 1 {
			return errors.New("simple-tvs-ratio must be between 0 and 1")
		}
		if strings.TrimSpace(file.Category) == "" {
			return errors.New("category is required for new protocols")
		}
		if len(file.Chains) == 0 {
			return errors.New("chains is required for new protocols")
		}
	}

	// New protocol without any metadata - require all mandatory fields
	if !isKnown && !file.hasMetadata() {
		return errors.New("new protocol requires is-ongoing, live, simple-tvs-ratio, category, and chains fields")
	}

	return nil
}

func normalizeHistory(history []models.TVLHistoryItem) []models.TVLHistoryItem {
	normalized := make([]models.TVLHistoryItem, 0, len(history))
	for _, item := range history {
		if strings.TrimSpace(item.Date) == "" && item.Timestamp > 0 {
			item.Date = time.Unix(item.Timestamp, 0).UTC().Format("2006-01-02")
		}
		if item.Timestamp == 0 && strings.TrimSpace(item.Date) != "" {
			if t, err := time.Parse("2006-01-02", item.Date); err == nil {
				item.Timestamp = t.Unix()
			}
		}
		normalized = append(normalized, item)
	}

	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].Timestamp < normalized[j].Timestamp
	})

	return normalized
}

// MergeTVLHistory merges API and custom history by date. Custom data overrides
// API data on conflicts and the result is sorted ascending by timestamp.
func MergeTVLHistory(apiData, customData []models.TVLHistoryItem) []models.TVLHistoryItem {
	if len(apiData) == 0 && len(customData) == 0 {
		return nil
	}

	byDate := make(map[string]models.TVLHistoryItem, len(apiData)+len(customData))

	for _, item := range apiData {
		key := strings.TrimSpace(item.Date)
		if key == "" && item.Timestamp > 0 {
			key = time.Unix(item.Timestamp, 0).UTC().Format("2006-01-02")
		}
		if key == "" {
			continue
		}
		byDate[key] = item
	}

	for _, item := range customData {
		key := strings.TrimSpace(item.Date)
		if key == "" && item.Timestamp > 0 {
			key = time.Unix(item.Timestamp, 0).UTC().Format("2006-01-02")
		}
		if key == "" {
			continue
		}
		byDate[key] = item // custom overrides API
	}

	merged := make([]models.TVLHistoryItem, 0, len(byDate))
	for _, item := range byDate {
		merged = append(merged, item)
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Timestamp < merged[j].Timestamp
	})

	return merged
}

func derefString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}
