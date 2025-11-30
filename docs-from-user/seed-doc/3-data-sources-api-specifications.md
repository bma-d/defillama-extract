# 3. Data Sources & API Specifications

## 3.1 Primary API Endpoints

### 3.1.1 Oracle Data Endpoint

```
Endpoint: GET https://api.llama.fi/oracles
Method: GET
Authentication: None (public API)
Rate Limit: No official limit (recommend 15+ minute intervals)
Cache-Control: public, max-age=600 (10 minutes)
```

**Request Headers:**
```
User-Agent: SwitchboardOracleExtractor/1.0 (Go)
Accept: application/json
```

**Response Content-Type:** `application/json`

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `oracles` | `map[string][]string` | Maps oracle name to list of protocol names |
| `chart` | `map[string]map[string]map[string]float64` | `timestamp -> oracle -> chain -> TVS` |
| `chainChart` | `map[string]map[string]map[string]float64` | Same structure as `chart` (chain-specific) |
| `oraclesTVS` | `map[string]map[string]map[string]float64` | `oracle -> protocol -> chain -> TVS` |
| `chainsByOracle` | `map[string][]string` | Maps oracle name to list of chain names |

### 3.1.2 Protocol Metadata Endpoint

```
Endpoint: GET https://api.llama.fi/lite/protocols2?b=2
Method: GET
Authentication: None (public API)
Rate Limit: No official limit
```

**Request Headers:**
```
User-Agent: SwitchboardOracleExtractor/1.0 (Go)
Accept: application/json
```

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `protocols` | `[]Protocol` | Array of protocol objects |
| `chains` | `[]string` | List of all chain names |
| `parentProtocols` | `[]ParentProtocol` | Parent protocol relationships |

**Protocol Object Fields:**

| Field | Type | Description | Nullable |
|-------|------|-------------|----------|
| `id` | `string` | Unique protocol identifier | No |
| `name` | `string` | Display name | No |
| `slug` | `string` | URL-friendly identifier | No |
| `chain` | `string` | Primary chain | No |
| `chains` | `[]string` | All supported chains | Yes |
| `category` | `string` | Protocol category (Lending, CDP, etc.) | No |
| `tvl` | `float64` | Current total value locked | Yes |
| `oracles` | `[]string` | List of oracle names used | Yes |
| `oracle` | `string` | Single oracle name (legacy) | Yes |
| `symbol` | `string` | Token symbol | Yes |
| `url` | `string` | Protocol website URL | Yes |

## 3.2 API Response Timing

```
DefiLlama Update Schedule:
┌────────────────────────────────────────────────────────────────┐
│ :00 - Protocol adapters start running                          │
│ :20 - Most adapters complete                                   │
│ :21 - Backend cache invalidation                               │
│ :22 - New oracle data generated (writeOracles cron)            │
│ :22 - Data available via API                                   │
│ :32 - API cache expires (10 min TTL)                           │
└────────────────────────────────────────────────────────────────┘

Recommended polling: Every 15 minutes, starting at :07, :22, :37, :52
This aligns with DefiLlama's update cycle while avoiding peak times.
```

## 3.3 Switchboard-Specific Data

**Oracle Name (exact match required):** `"Switchboard"`

**Expected Chains:**
- Solana (primary)
- Sui
- Aptos
- Arbitrum
- Ethereum

**Expected Protocol Count:** ~21 protocols

**Protocol Categories:**
- Lending
- CDP (Collateralized Debt Position)
- Liquid Staking
- Dexes
- Derivatives

---
