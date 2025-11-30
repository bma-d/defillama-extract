# API Contracts

> **Spec Reference:** [3-data-sources-api-specifications.md](../docs-from-user/seed-doc/3-data-sources-api-specifications.md), [13-api-response-examples.md](../docs-from-user/seed-doc/13-api-response-examples.md)

## DefiLlama Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `https://api.llama.fi/oracles` | GET | Oracle TVS data, protocol lists |
| `https://api.llama.fi/lite/protocols2?b=2` | GET | Protocol metadata |

## Output Files

| File | Content | Format |
|------|---------|--------|
| `switchboard-oracle-data.json` | Full data with history | Indented JSON |
| `switchboard-oracle-data.min.json` | Same data, compact | Minified JSON |
| `switchboard-summary.json` | Current snapshot only | Indented JSON |
| `state.json` | Incremental update state | Indented JSON |

## JSON Output Schema

```json
{
  "version": "1.0.0",
  "oracle": {
    "name": "Switchboard",
    "website": "https://switchboard.xyz",
    "documentation": "https://docs.switchboard.xyz"
  },
  "metadata": {
    "last_updated": "2025-11-29T22:51:16Z",
    "data_source": "DefiLlama",
    "update_frequency": "2 hours",
    "extractor_version": "1.0.0"
  },
  "summary": {
    "total_value_secured": 180000000,
    "total_protocols": 21,
    "active_chains": 5,
    "categories": ["Lending", "CDP", "Liquid Staking"]
  },
  "metrics": { ... },
  "breakdown": { ... },
  "protocols": [ ... ],
  "historical": [ ... ]
}
```
