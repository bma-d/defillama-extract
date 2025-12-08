# Epic 7: Custom Protocols & Per-Protocol TVL Charting

**Goal:** Enable tracking of known Switchboard integrations not tagged by DefiLlama, and provide historical TVL charting data for all tracked protocols.

**Status:** Ready for Development

## Overview

This epic adds two capabilities:

1. **Custom Protocol Input** - User-defined JSON file specifying Switchboard integrations that DefiLlama doesn't auto-detect
2. **Per-Protocol TVL Charting** - Historical TVL time-series data for ALL protocols (auto-detected + custom)

## Key Design Decisions

- **Separate data pipeline** - `tvl-data.json` is independent from main `switchboard-oracle-data.json`
- **No TVS calculation** - Only TVL stored; TVS calculated downstream using `simple-tvs-ratio`
- **No date filtering** - Full TVL history included; `integration_date` passed through for downstream filtering
- **Same update cycle** - 2-hour polling alongside main extraction

## Input: Custom Protocols JSON

**Location:** `config/custom-protocols.json`

**Schema:**
```json
[
  {
    "slug": "drift-trade",
    "is-ongoing": true,
    "live": true,
    "date": 1700000000,
    "simple-tvs-ratio": 0.85,
    "docs_proof": "https://docs.drift.trade/oracles#switchboard",
    "github_proof": "https://github.com/drift-labs/protocol-v2/blob/main/src/oracle.rs"
  }
]
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `slug` | string | Yes | DefiLlama protocol slug |
| `is-ongoing` | boolean | Yes | Whether integration is ongoing |
| `live` | boolean | Yes | If false, skip this protocol entirely |
| `date` | number | No | Unix timestamp of integration |
| `simple-tvs-ratio` | number | Yes | 0-1 decimal for downstream TVS calculation |
| `docs_proof` | string | No | URL to documentation proving Switchboard integration |
| `github_proof` | string | No | URL to code proving Switchboard integration |

## Output: tvl-data.json

**Location:** `data/tvl-data.json`

**Contains:** ALL protocols (auto-detected from `/oracles` + custom from JSON)

**Schema:**
```json
{
  "version": "1.0.0",
  "metadata": {
    "last_updated": "2025-12-07T12:00:00Z",
    "protocol_count": 25,
    "custom_protocol_count": 4
  },
  "protocols": {
    "drift-trade": {
      "name": "Drift Trade",
      "slug": "drift-trade",
      "source": "custom",
      "is_ongoing": true,
      "simple_tvs_ratio": 0.85,
      "integration_date": 1700000000,
      "docs_proof": "https://...",
      "github_proof": "https://...",
      "current_tvl": 677000000,
      "tvl_history": [
        {
          "date": "2024-01-01",
          "timestamp": 1704067200,
          "tvl": 150000000
        }
      ]
    },
    "marginfi": {
      "name": "marginfi",
      "slug": "marginfi",
      "source": "auto",
      "is_ongoing": true,
      "simple_tvs_ratio": 1.0,
      "integration_date": null,
      "docs_proof": null,
      "github_proof": null,
      "current_tvl": 500000000,
      "tvl_history": [...]
    }
  }
}
```

## Stories

### Story 7.1: Load Custom Protocols Configuration

**Goal:** Parse and validate `config/custom-protocols.json`, filtering out `live: false` entries.

**Acceptance Criteria:**
- [ ] Load JSON from configurable path (default: `config/custom-protocols.json`)
- [ ] Validate required fields (slug, is-ongoing, live, simple-tvs-ratio)
- [ ] Filter out entries where `live: false`
- [ ] Log count of loaded vs filtered protocols
- [ ] Handle missing file gracefully (empty custom list, not error)

---

### Story 7.2: Implement Protocol TVL Fetcher

**Goal:** Fetch historical TVL data from `GET /protocol/{slug}` endpoint with retry logic.

**Acceptance Criteria:**
- [ ] Reuse existing HTTP client with retry/backoff
- [ ] Fetch `GET https://api.llama.fi/protocol/{slug}` for each protocol
- [ ] Extract `tvl[]` array (historical TVL time-series)
- [ ] Extract `name`, `currentChainTvls` for current TVL
- [ ] Handle 404 (protocol not found) gracefully with warning log
- [ ] Rate limit requests (200ms delay between calls)

---

### Story 7.3: Merge Protocol Lists

**Goal:** Combine auto-detected protocols (from `/oracles`) with custom protocols, deduplicating by slug.

**Acceptance Criteria:**
- [ ] Get auto-detected protocol slugs from existing aggregator
- [ ] Merge with custom protocols list
- [ ] Deduplicate by slug (custom takes precedence for metadata)
- [ ] Auto-detected protocols get `source: "auto"`, `simple_tvs_ratio: 1.0`, `docs_proof: "https://defillama.com/protocol/{slug}"`
- [ ] Custom protocols get `source: "custom"` with their config values

---

### Story 7.4: Include Integration Date in Output

**Goal:** Pass through `date` field as `integration_date` in output. No filtering of TVL history.

**Acceptance Criteria:**
- [ ] Custom protocols: `integration_date` = config `date` value (or null if not set)
- [ ] Auto-detected protocols: `integration_date` = null
- [ ] Full `tvl_history` array included regardless of integration date
- [ ] Downstream applications handle date filtering

---

### Story 7.5: Generate tvl-data.json Output

**Goal:** Build and write the `tvl-data.json` output file with atomic writes.

**Acceptance Criteria:**
- [ ] Build output structure per schema above
- [ ] Include metadata (last_updated, protocol_count, custom_protocol_count)
- [ ] Write using existing atomic file writer (temp + rename)
- [ ] Output to configurable path (default: `data/tvl-data.json`)
- [ ] Generate minified version `tvl-data.min.json`

---

### Story 7.6: Integrate TVL Pipeline into Main Extraction Cycle

**Goal:** Run TVL charting pipeline alongside main extraction in the 2-hour cycle.

**Acceptance Criteria:**
- [ ] TVL pipeline runs after main oracle extraction completes
- [ ] Both pipelines share same extraction timestamp
- [ ] Failures in TVL pipeline don't affect main pipeline output
- [ ] Logging distinguishes main vs TVL pipeline operations
- [ ] `--once` mode runs both pipelines
- [ ] State tracking for TVL pipeline (skip if no changes)

---

## Dependencies

- **Epic 2** (API Integration) - reuse HTTP client and retry logic
- **Epic 5** (Output & CLI) - reuse atomic file writer

## API Reference

**Endpoint:** `GET https://api.llama.fi/protocol/{slug}`

**Response (relevant fields):**
```json
{
  "name": "Drift Trade",
  "tvl": [
    {"date": 1704067200, "totalLiquidityUSD": 150000000},
    {"date": 1704153600, "totalLiquidityUSD": 155000000}
  ],
  "currentChainTvls": {
    "Solana": 677000000
  }
}
```

**Reference:** `docs-from-llm/protocol-query.md`

## Definition of Done

- [ ] Custom protocols loaded from `config/custom-protocols.json`
- [ ] `live: false` protocols are skipped
- [ ] All protocols (auto + custom) have TVL history fetched
- [ ] `integration_date` included in output (no filtering)
- [ ] `tvl-data.json` and `tvl-data.min.json` written atomically
- [ ] Logging shows custom vs auto-detected protocol counts
- [ ] Main pipeline (`switchboard-oracle-data.json`) unchanged
- [ ] Unit tests for new components
- [ ] Integration test for full TVL pipeline
