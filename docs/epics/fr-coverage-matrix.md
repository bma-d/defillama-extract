# FR Coverage Matrix

| FR | Description | Epic | Story |
|----|-------------|------|-------|
| FR1 | Fetch oracle data from `/oracles` endpoint | Epic 2 | 2.2 |
| FR2 | Fetch protocol metadata from `/lite/protocols2` endpoint | Epic 2 | 2.3 |
| FR3 | Parallel fetching of both endpoints | Epic 2 | 2.5 |
| FR4 | Include proper User-Agent header | Epic 2 | 2.1 |
| FR5 | Retry with exponential backoff | Epic 2 | 2.4 |
| FR6 | Configurable timeout (default 30s) | Epic 2 | 2.1 |
| FR7 | Handle API errors (429, 5xx) with retry | Epic 2 | 2.4 |
| FR8 | Detect non-retryable errors (4xx) | Epic 2 | 2.4 |
| FR9 | Filter protocols by oracle name | Epic 3 | 3.1 |
| FR10 | Check both `oracles` array and legacy `oracle` field | Epic 3 | 3.1 |
| FR11 | Extract TVS per protocol per chain | Epic 3 | 3.2 |
| FR12 | Extract protocol metadata | Epic 3 | 3.2 |
| FR13 | Identify all chains for target oracle | Epic 3 | 3.2 |
| FR14 | Extract timestamp from chart data | Epic 3 | 3.2 |
| FR15 | Calculate total TVS | Epic 3 | 3.3 |
| FR16 | Calculate TVS breakdown by chain | Epic 3 | 3.3 |
| FR17 | Calculate TVS breakdown by category | Epic 3 | 3.4 |
| FR18 | Rank protocols by TVL | Epic 3 | 3.5 |
| FR19 | Calculate 24h TVS change | Epic 3 | 3.6 |
| FR20 | Calculate 7d TVS change | Epic 3 | 3.6 |
| FR21 | Calculate 30d TVS change | Epic 3 | 3.6 |
| FR22 | Calculate protocol count growth | Epic 3 | 3.6 |
| FR23 | Identify largest protocol | Epic 3 | 3.5 |
| FR24 | Extract unique categories | Epic 3 | 3.4 |
| FR25 | Track last processed timestamp | Epic 4 | 4.1 |
| FR26 | Compare timestamps for new data | Epic 4 | 4.2 |
| FR27 | Skip when no new data | Epic 4 | 4.2 |
| FR28 | Recover from corrupted state | Epic 4 | 4.1 |
| FR29 | Update state atomically | Epic 4 | 4.3 |
| FR30 | Maintain historical snapshots | Epic 4 | 4.4 |
| FR31 | Store snapshot fields | Epic 4 | 4.4 |
| FR32 | Deduplicate snapshots | Epic 4 | 4.6 |
| FR33 | Retain all snapshots (no pruning) | Epic 4 | 4.7 |
| FR34 | Load history from output file | Epic 4 | 4.5 |
| FR35 | Generate full output JSON | Epic 5 | 5.1 |
| FR36 | Generate minified output JSON | Epic 5 | 5.2 |
| FR37 | Generate summary output JSON | Epic 5 | 5.3 |
| FR38 | Write files atomically | Epic 5 | 5.4 |
| FR39 | Create output directory | Epic 5 | 5.4 |
| FR40 | Include version/oracle info/metadata | Epic 5 | 5.1 |
| FR41 | Include extraction timestamp/attribution | Epic 5 | 5.1 |
| FR42 | Run-once mode (`--once`) | Epic 5 | 5.5, 5.6 |
| FR43 | Daemon mode with interval | Epic 5 | 5.7 |
| FR44 | Config file path (`--config`) | Epic 5 | 5.5 |
| FR45 | Dry-run mode (`--dry-run`) | Epic 5 | 5.5 |
| FR46 | Version flag (`--version`) | Epic 5 | 5.5 |
| FR47 | Graceful shutdown | Epic 5 | 5.8 |
| FR48 | Log extraction metrics | Epic 5 | 5.6, 5.9 |
| FR49 | Load config from YAML | Epic 1 | 1.2 |
| FR50 | Environment variable overrides | Epic 1 | 1.3 |
| FR51 | Sensible defaults | Epic 1 | 1.2 |
| FR52 | Validate config on startup | Epic 1 | 1.2 |
| FR53 | Structured logging (JSON/text) | Epic 1 | 1.4 |
| FR54 | Configurable log levels | Epic 1 | 1.4 |
| FR55 | Log API requests/retries/failures | Epic 2 | 2.6 |
| FR56 | Log extraction results | Epic 5 | 5.9 |

**Coverage Validation:** All 56 FRs mapped to stories ✓

---
