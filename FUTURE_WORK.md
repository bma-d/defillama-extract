# Future Work

This document captures potential scope extensions beyond the MVP.

> **Note:** Story 5.4 (Extract Historical Chart Data) was added to Epic 5 after discovering the chart data gap during retrospective. Epic 5 is not yet complete.

---

## Priority 1: True TVS Calculation (Recommended Next)

### Problem Statement

The current implementation trusts DefiLlama's `oracles` field mapping. However, DefiLlama doesn't know about all protocols that use Switchboard, leading to an undercount of the true Total Value Secured.

### Proposed Solution

Add a curated list of "additional protocols" that use Switchboard but aren't tagged correctly in DefiLlama, and calculate the actual TVS across all protocols.

### Implementation Options

#### Protocol Source Options

| Option | Description | Complexity | Accuracy |
|--------|-------------|------------|----------|
| **A) Config File** | Manual YAML/JSON file with protocol slugs | Low | Depends on maintenance |
| **B) On-Chain Verification** | Query Solana/Sui/Aptos for actual oracle usage | High | Highest |
| **C) Hybrid** | Config for known ones + on-chain discovery | Medium-High | High |

#### TVS Calculation Options

| Option | Description | Complexity |
|--------|-------------|------------|
| **A) DefiLlama by Slug** | Fetch TVL from DefiLlama using protocol slug (they have data, just wrong oracle tag) | Low |
| **B) On-Chain Direct** | Calculate TVS from on-chain data directly | High |
| **C) Manual Override** | Config file with manual TVS values | Low (but stale) |

#### Output Format Options

| Option | Description |
|--------|-------------|
| **A) Merged** | Seamlessly merge with DefiLlama protocols |
| **B) Separate** | Show "Additional Protocols" vs "DefiLlama Protocols" sections |
| **C) Both Views** | Merged total + breakdown by source |

### Recommended Approach

**Config-based with DefiLlama TVL fetch (Options A, A, C):**

1. Create `configs/additional-protocols.yaml`:
   ```yaml
   # Protocols using Switchboard that DefiLlama doesn't tag correctly
   additional_protocols:
     - slug: "protocol-name"
       reason: "Uses Switchboard for price feeds on Solana"
       verified_date: "2025-12-01"
       chains: ["Solana"]
     - slug: "another-protocol"
       reason: "Confirmed Switchboard integration via GitHub"
       verified_date: "2025-12-01"
       chains: ["Sui"]
   ```

2. Fetch TVL for these protocols from DefiLlama's protocol endpoint
3. Add to aggregation with source attribution
4. Output shows both merged view and source breakdown

### Stories (Estimated)

1. **Config schema and loading** - Define and parse additional protocols config
2. **Protocol TVL fetching** - Fetch individual protocol data from DefiLlama
3. **Aggregation integration** - Merge additional protocols into aggregation
4. **Output enhancement** - Add source attribution and breakdown views
5. **Deduplication** - Handle protocols that appear in both sources

---

## Priority 2: Growth Features (From PRD)

These were explicitly marked as "Post-MVP" in the PRD.

### Prometheus Metrics Endpoint

Expose operational metrics for monitoring:
- `extraction_duration_seconds` - Time per extraction cycle
- `protocols_count` - Number of protocols captured
- `tvs_total` - Current total value secured
- `api_requests_total` - API request counts by endpoint/status
- `api_request_duration_seconds` - API latency histogram

**Complexity:** Low-Medium
**Value:** High for production observability

### Health Check HTTP Endpoint

Simple HTTP endpoint for orchestration systems:
- `GET /health` - Returns 200 if healthy
- `GET /ready` - Returns 200 if ready to serve (after first extraction)

**Complexity:** Low
**Value:** High for container orchestration (K8s, Docker Compose)

### Containerization (Docker)

- Multi-stage Dockerfile for minimal image size
- Docker Compose for easy local deployment
- Environment variable configuration
- Volume mounts for data persistence

**Complexity:** Low
**Value:** Medium (deployment flexibility)

### Alerting

Notifications when:
- New protocol adopts Switchboard
- Significant TVS changes (>10% in 24h)
- Protocol drops Switchboard oracle
- Extraction failures

Integration options: Slack, Discord, PagerDuty, webhook

**Complexity:** Medium
**Value:** Medium-High for proactive monitoring

---

## Priority 3: Vision Features (Long-term)

From the PRD's Vision section.

### Multi-Oracle Comparison

Expand beyond Switchboard to track Chainlink, Pyth, Redstone, and other oracle providers. Enable competitive analysis dashboards.

**Complexity:** Medium (mostly configuration, same code paths)
**Value:** High for market positioning

### Real-Time Streaming

Move from polling-based updates to real-time data streaming for live dashboards.

**Complexity:** High (architecture change)
**Value:** Medium (2-hour updates sufficient for most use cases)

### Public API

Serve aggregated data via REST/GraphQL API instead of static JSON files.

**Complexity:** Medium-High
**Value:** Depends on consumer demand

### On-Chain Verification

Validate DefiLlama's oracle mappings against actual on-chain data. Query program accounts to verify which oracle a protocol actually uses.

**Complexity:** High (chain-specific implementations)
**Value:** Highest accuracy possible

### Additional Data Sources

Integrate data sources beyond DefiLlama to capture protocols they may miss entirely.

**Complexity:** Variable (depends on source)
**Value:** Higher coverage

---

## Implementation Recommendations

### If Extending Scope

1. **Update PRD** - Run `*prd` workflow to add new requirements formally
2. **Create Epic** - Run `*create-epics-and-stories` to generate properly traced stories
3. **Tech Spec** - Draft tech spec before story creation (lesson from Epic 4 retro)
4. **Maintain Patterns** - Follow established patterns (atomic writes, structured logging, table-driven tests)

### BMAD Workflow Commands

| Task | Command |
|------|---------|
| Update PRD | `/bmad:bmm:agents:pm` → `*prd` |
| Create Epics | `/bmad:bmm:agents:sm` → `*create-epics-and-stories` |
| Architecture Review | `/bmad:bmm:agents:architect` |

---

## Notes

- Current MVP captures 31 protocols with ~$988M TVS
- DefiLlama updates approximately hourly
- 2-hour polling interval respects API etiquette
- All historical snapshots retained (no pruning)

---

*Document created: 2025-12-02*
*Last updated: 2025-12-02*
