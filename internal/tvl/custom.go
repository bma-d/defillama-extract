package tvl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/switchboard-xyz/defillama-extract/internal/models"
)

// CustomLoader loads and validates custom protocol definitions from a JSON file.
type CustomLoader struct {
	configPath string
	logger     *slog.Logger
}

// NewCustomLoader constructs a CustomLoader. If logger is nil, slog.Default() is used.
func NewCustomLoader(configPath string, logger *slog.Logger) *CustomLoader {
	if logger == nil {
		logger = slog.Default()
	}
	return &CustomLoader{
		configPath: configPath,
		logger:     logger,
	}
}

// Load reads the custom protocols JSON, validates entries, filters non-live items,
// and returns the slice of active protocols. Missing files are treated as an
// empty list with an INFO log.
func (l *CustomLoader) Load(ctx context.Context) ([]models.CustomProtocol, error) {
	if l == nil {
		return nil, errors.New("nil CustomLoader")
	}
	if strings.TrimSpace(l.configPath) == "" {
		return nil, errors.New("custom protocols path must not be empty")
	}

	data, err := os.ReadFile(l.configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			l.logger.InfoContext(ctx, "custom_protocols_not_found", "path", l.configPath, "reason", "file not found")
			return []models.CustomProtocol{}, nil
		}
		return nil, fmt.Errorf("read custom protocols: %w", err)
	}

	var rawEntries []json.RawMessage
	if err := json.Unmarshal(data, &rawEntries); err != nil {
		return nil, fmt.Errorf("parse custom protocols: %w", err)
	}

	filtered := 0
	active := make([]models.CustomProtocol, 0, len(rawEntries))
	for i, raw := range rawEntries {
		var p models.CustomProtocol
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, fmt.Errorf("parse custom protocols: %w", err)
		}

		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			return nil, fmt.Errorf("parse custom protocols: %w", err)
		}

		hasIsOngoing := hasField(fields, "is-ongoing")
		hasLive := hasField(fields, "live")

		if err := l.Validate(p, hasIsOngoing, hasLive); err != nil {
			slug := strings.TrimSpace(p.Slug)
			if slug == "" {
				slug = fmt.Sprintf("index_%d", i)
			}
			return nil, fmt.Errorf("invalid protocol %s: %w", slug, err)
		}

		if !p.Live {
			filtered++
			continue
		}

		active = append(active, p)
	}

	l.logger.InfoContext(ctx, "custom_protocols_loaded",
		"total", len(active),
		"filtered", filtered,
		"config_path", l.configPath,
	)

	return active, nil
}

func hasField(fields map[string]json.RawMessage, key string) bool {
	_, ok := fields[key]
	return ok
}

// Validate enforces required fields, presence, and ranges for a CustomProtocol.
func (l *CustomLoader) Validate(p models.CustomProtocol, hasIsOngoing bool, hasLive bool) error {
	if strings.TrimSpace(p.Slug) == "" {
		return errors.New("slug must not be empty")
	}
	if !hasIsOngoing {
		return errors.New("is-ongoing missing")
	}
	if !hasLive {
		return errors.New("live missing")
	}
	if p.SimpleTVSRatio < 0 || p.SimpleTVSRatio > 1 {
		return fmt.Errorf("simple-tvs-ratio must be between 0 and 1")
	}
	return nil
}
