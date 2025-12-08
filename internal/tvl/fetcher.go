package tvl

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/switchboard-xyz/defillama-extract/internal/api"
)

// TVLClient defines the subset of api.Client used by the TVL pipeline to fetch
// per-protocol TVL history. The real client already satisfies this interface.
type TVLClient interface {
	FetchProtocolTVL(ctx context.Context, slug string) (*api.ProtocolTVLResponse, error)
}

// FetchAllTVL iterates protocol slugs sequentially, honoring the client's
// built-in rate limiting (200ms). It returns a map of successful responses and
// captures statistics for logging (AC8).
func FetchAllTVL(ctx context.Context, client TVLClient, slugs []string, logger *slog.Logger) (map[string]*api.ProtocolTVLResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if logger == nil {
		logger = slog.Default()
	}
	if client == nil {
		return nil, fmt.Errorf("nil TVL client")
	}

	results := make(map[string]*api.ProtocolTVLResponse, len(slugs))
	var firstErr error
	stats := struct {
		total    int
		success  int
		notFound int
		failed   int
		duration time.Duration
	}{}

	start := time.Now()
	for _, slug := range slugs {
		if err := ctx.Err(); err != nil {
			return results, err
		}

		stats.total++
		resp, err := client.FetchProtocolTVL(ctx, slug)
		switch {
		case err != nil:
			stats.failed++
			if firstErr == nil {
				firstErr = err
			}
			logger.Error("tvl_fetch_failed", "slug", slug, "error", err)
		case resp == nil:
			stats.notFound++
			logger.Warn("tvl_protocol_not_found", "slug", slug)
		default:
			stats.success++
			results[slug] = resp
		}
	}
	stats.duration = time.Since(start)

	logger.Info("tvl_fetch_complete",
		"total", stats.total,
		"success", stats.success,
		"not_found", stats.notFound,
		"failed", stats.failed,
		"duration_ms", stats.duration.Milliseconds(),
	)

	return results, firstErr
}
