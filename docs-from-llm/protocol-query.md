# DefiLlama Protocol Query Guide

Complete guide for querying individual protocol data from DefiLlama API, including metadata, documentation links, and time-series TVL data for charts.

---

## Table of Contents
1. [Quick Start](#quick-start)
2. [Primary Endpoint](#primary-endpoint)
3. [Response Structure](#response-structure)
4. [Extracting Required Data](#extracting-required-data)
5. [Batch Querying Multiple Protocols](#batch-querying-multiple-protocols)
6. [Finding Protocol Slugs](#finding-protocol-slugs)
7. [Chart Data Formatting](#chart-data-formatting)
8. [Real Examples](#real-examples)
9. [Additional Endpoints](#additional-endpoints)
10. [Best Practices](#best-practices)

---

## Quick Start

**Single endpoint returns everything you need:**

```bash
GET https://api.llama.fi/protocol/{protocol-slug}
```

**Example:**
```bash
curl https://api.llama.fi/protocol/jupiter-aggregator
curl https://api.llama.fi/protocol/drift-trade
```

**Returns:**
- Protocol metadata (name, description, category)
- Website & documentation links
- Social links (Twitter, GitHub)
- Audit information
- Time-series TVL data (for charts)
- Current TVL by chain
- Token composition

---

## Primary Endpoint

### GET /protocol/{slug}

**Base URL:** `https://api.llama.fi`

**Parameters:**
- `slug` (required): Protocol slug name (lowercase, hyphens for spaces)

**Query Parameters:**
- `restrictResponseSize` (optional): Set to `false` to get full token breakdown details

**Examples:**
```bash
# Standard request
GET https://api.llama.fi/protocol/jupiter-aggregator

# Full data (no size restrictions)
GET https://api.llama.fi/protocol/drift-trade?restrictResponseSize=false
```

**Cache Headers:**
- `Cache-Control: public, max-age=1200` (20 minutes)

---

## Response Structure

### Complete Response Schema

```typescript
{
  // Basic Metadata
  id: string;                    // Unique protocol ID
  name: string;                  // Display name
  symbol: string;                // Token symbol
  url: string;                   // Official website
  description: string;           // Protocol description
  logo: string;                  // Logo URL
  address?: string;              // Token contract address
  chain: string;                 // Primary chain
  chains: string[];              // All chains
  category?: string;             // Protocol category

  // Documentation & Links
  twitter?: string;              // Twitter handle (without @)
  github?: string[];             // GitHub repos
  gecko_id?: string;             // CoinGecko ID
  cmcId?: string;                // CoinMarketCap ID
  referralUrl?: string;          // Referral link

  // Audit Information
  audits: string;                // Number of audits
  audit_links?: string[];        // Audit report URLs
  audit_note?: string;           // Additional audit notes

  // Time-Series TVL Data (Main Chart Data)
  tvl: Array<{
    date: number;                // Unix timestamp (seconds)
    totalLiquidityUSD: number;   // TVL in USD
  }>;

  // Current TVL by Chain
  currentChainTvls: {
    [chainName: string]: number; // Current TVL in USD per chain
  };

  // Historical TVL by Chain
  chainTvls: {
    [chainName: string]: {
      tvl: Array<{
        date: number;
        totalLiquidityUSD: number;
      }>;
      tokensInUsd?: Array<{
        date: number;
        tokens: { [symbol: string]: number };
      }>;
      tokens?: Array<{
        date: number;
        tokens: { [symbol: string]: number };
      }>;
    };
  };

  // Token Composition (Optional - may be restricted)
  tokensInUsd?: Array<{
    date: number;
    tokens: {
      [tokenSymbol: string]: number;  // USD value
    };
  }>;

  tokens?: Array<{
    date: number;
    tokens: {
      [tokenSymbol: string]: number;  // Raw token amounts
    };
  }>;

  // Additional Metadata
  mcap?: number;                      // Market cap
  methodology?: string;               // TVL calculation methodology
  parentProtocol?: string;            // Parent protocol slug
  otherProtocols?: string[];          // Related protocols
  hallmarks?: Array<[number, string]>; // Notable events [timestamp, label]

  // Fundraising Data
  raises?: Array<{
    date: number;
    name: string;
    amount: number;
    round: string;
    valuation: number;
    leadInvestors: string[];
  }>;

  // Oracle Information (if applicable)
  oracles?: string[];                 // Oracle names
  oraclesByChain?: {
    [chain: string]: string[];
  };
  oraclesBreakdown?: Array<{
    name: string;
    type: string;  // "Primary", "Secondary", "Fallback", "Aggregator"
  }>;
}
```

---

## Extracting Required Data

### Website & Documentation Links

```javascript
const response = await fetch('https://api.llama.fi/protocol/jupiter-aggregator')
const data = await response.json()

// Website
const website = data.url
// "https://jup.ag/"

// Documentation (usually same as website or in GitHub)
const docs = data.url
const github = data.github?.[0] ? `https://github.com/${data.github[0]}` : null

// Social Links
const twitter = data.twitter ? `https://twitter.com/${data.twitter}` : null
// Example: "https://twitter.com/JupiterExchange"

// Audit Reports
const audits = data.audit_links || []
// ["https://github.com/...audit-report.pdf"]
```

### Time-Series TVL Data for Charts

```javascript
// Historical TVL data points
const historicalTvl = data.tvl.map(point => ({
  date: new Date(point.date * 1000).toISOString(),
  timestamp: point.date,
  tvlUSD: point.totalLiquidityUSD
}))

// Example output:
// [
//   {
//     date: "2022-01-01T00:00:00.000Z",
//     timestamp: 1640995200,
//     tvlUSD: 125000000
//   },
//   {
//     date: "2022-01-02T00:00:00.000Z",
//     timestamp: 1641081600,
//     tvlUSD: 130000000
//   }
//   // ... hundreds more points
// ]
```

### Current TVL

```javascript
// Method 1: From currentChainTvls
const currentTvl = Object.values(data.currentChainTvls || {})
  .reduce((sum, tvl) => sum + tvl, 0)

// Method 2: From latest tvl array entry
const latestTvl = data.tvl[data.tvl.length - 1]?.totalLiquidityUSD || 0

// Method 3: By specific chain
const solanaTvl = data.currentChainTvls?.Solana || 0
```

### Protocol Metadata

```javascript
const metadata = {
  id: data.id,
  name: data.name,
  symbol: data.symbol,
  description: data.description,
  logo: data.logo,
  category: data.category,
  chains: data.chains,

  // Token/Market Data
  geckoId: data.gecko_id,
  cmcId: data.cmcId,
  marketCap: data.mcap,

  // Methodology
  methodology: data.methodology
}
```

---

## Batch Querying Multiple Protocols

### Node.js Example

```javascript
const fetch = require('node-fetch')
const fs = require('fs').promises

const protocols = [
  'jupiter-aggregator',
  'drift-trade',
  'kamino-lend',
  'marginfi',
  'scallop',
  'raydium',
  'orca',
  'mango-markets',
  'solend',
  'francium'
  // ... add up to 20 protocol slugs
]

async function fetchProtocolData(slug) {
  try {
    const response = await fetch(`https://api.llama.fi/protocol/${slug}`)

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: ${response.statusText}`)
    }

    const data = await response.json()

    return {
      slug: slug,
      name: data.name,
      symbol: data.symbol,
      website: data.url,
      docs: data.url,
      description: data.description,
      logo: data.logo,

      // Links
      twitter: data.twitter ? `https://twitter.com/${data.twitter}` : null,
      github: data.github?.[0] ? `https://github.com/${data.github[0]}` : null,

      // Audit Information
      auditCount: parseInt(data.audits || '0'),
      auditLinks: data.audit_links || [],

      // TVL Data
      currentTvl: Object.values(data.currentChainTvls || {})
        .reduce((a, b) => a + b, 0),

      historicalTvl: data.tvl.map(point => ({
        date: new Date(point.date * 1000).toISOString(),
        timestamp: point.date,
        tvl: point.totalLiquidityUSD
      })),

      // Chain & Category
      chains: data.chains,
      category: data.category,

      // Additional Metadata
      geckoId: data.gecko_id,
      cmcId: data.cmcId,
      methodology: data.methodology
    }
  } catch (error) {
    console.error(`Failed to fetch ${slug}:`, error.message)
    return null
  }
}

async function fetchAllProtocols(slugs) {
  const results = []

  for (const slug of slugs) {
    console.log(`Fetching ${slug}...`)
    const data = await fetchProtocolData(slug)

    if (data) {
      results.push(data)
    }

    // Rate limiting - wait 200ms between requests
    await new Promise(resolve => setTimeout(resolve, 200))
  }

  return results
}

async function main() {
  const data = await fetchAllProtocols(protocols)

  // Save to JSON file
  await fs.writeFile(
    './protocols-data.json',
    JSON.stringify(data, null, 2)
  )

  console.log(`\nFetched ${data.length} protocols`)
  console.log(`Saved to protocols-data.json`)
}

main().catch(console.error)
```

### Output Format

```json
[
  {
    "slug": "jupiter-aggregator",
    "name": "Jupiter Aggregator",
    "symbol": "JUP",
    "website": "https://jup.ag/",
    "docs": "https://jup.ag/",
    "description": "The best swap aggregator & infrastructure for Solana",
    "logo": "https://icons.llama.fi/jupiter-aggregator.jpg",
    "twitter": "https://twitter.com/JupiterExchange",
    "github": "https://github.com/jup-ag/jupiter-core",
    "auditCount": 2,
    "auditLinks": ["https://..."],
    "currentTvl": 2500000000,
    "historicalTvl": [
      {
        "date": "2022-01-01T00:00:00.000Z",
        "timestamp": 1640995200,
        "tvl": 125000000
      }
      // ... more data points
    ],
    "chains": ["Solana"],
    "category": "DEX Aggregator",
    "geckoId": "jupiter-exchange-solana",
    "cmcId": "29210"
  }
]
```

---

## Finding Protocol Slugs

### Method 1: Search All Protocols

```javascript
const fetch = require('node-fetch')

async function findProtocolSlugs(searchTerms) {
  const response = await fetch('https://api.llama.fi/protocols')
  const allProtocols = await response.json()

  const slugMap = {}

  for (const term of searchTerms) {
    const protocol = allProtocols.find(p =>
      p.name.toLowerCase().includes(term.toLowerCase())
    )

    if (protocol) {
      slugMap[term] = {
        slug: protocol.slug,
        name: protocol.name,
        tvl: protocol.tvl
      }
    } else {
      console.warn(`Protocol not found: ${term}`)
      slugMap[term] = null
    }
  }

  return slugMap
}

// Usage
const searchTerms = ['Jupiter', 'Drift', 'Kamino', 'Marginfi']
findProtocolSlugs(searchTerms).then(console.log)
```

**Output:**
```json
{
  "Jupiter": {
    "slug": "jupiter-aggregator",
    "name": "Jupiter Aggregator",
    "tvl": 2500000000
  },
  "Drift": {
    "slug": "drift-trade",
    "name": "Drift Trade",
    "tvl": 150000000
  }
}
```

### Method 2: Direct API Query

```bash
# Search for protocols containing "jupiter"
curl https://api.llama.fi/protocols | \
  jq '.[] | select(.name | ascii_downcase | contains("jupiter")) | {name, slug, tvl}'
```

### Common Protocol Slug Patterns

| Protocol Name | Slug |
|---------------|------|
| Jupiter Aggregator | `jupiter-aggregator` |
| Drift Trade | `drift-trade` |
| Kamino Lend | `kamino-lend` |
| marginfi Lending | `marginfi` |
| Raydium | `raydium` |
| Orca | `orca` |
| Mango Markets V4 | `mango-markets` |
| Solend | `solend` |
| Scallop Lend | `scallop` |
| Phoenix | `phoenix-trade` |

**Slug Rules:**
- Lowercase only
- Spaces → hyphens
- Special characters removed
- May omit "Lending", "V2", etc. suffixes

---

## Chart Data Formatting

### For Chart.js

```javascript
const response = await fetch('https://api.llama.fi/protocol/jupiter-aggregator')
const data = await response.json()

const chartData = {
  labels: data.tvl.map(point => new Date(point.date * 1000)),
  datasets: [{
    label: `${data.name} TVL`,
    data: data.tvl.map(point => point.totalLiquidityUSD),
    borderColor: 'rgb(75, 192, 192)',
    tension: 0.1
  }]
}
```

### For Recharts (React)

```jsx
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer } from 'recharts'

function ProtocolChart({ slug }) {
  const [data, setData] = useState(null)

  useEffect(() => {
    fetch(`https://api.llama.fi/protocol/${slug}`)
      .then(res => res.json())
      .then(protocol => {
        const chartData = protocol.tvl.map(point => ({
          date: point.date * 1000,
          tvl: point.totalLiquidityUSD
        }))
        setData(chartData)
      })
  }, [slug])

  if (!data) return <div>Loading...</div>

  return (
    <ResponsiveContainer width="100%" height={400}>
      <LineChart data={data}>
        <XAxis
          dataKey="date"
          type="number"
          domain={['dataMin', 'dataMax']}
          tickFormatter={(unixTime) => new Date(unixTime).toLocaleDateString()}
        />
        <YAxis
          tickFormatter={(value) => `$${(value / 1e6).toFixed(0)}M`}
        />
        <Tooltip
          labelFormatter={(unixTime) => new Date(unixTime).toLocaleDateString()}
          formatter={(value) => [`$${(value / 1e6).toFixed(2)}M`, 'TVL']}
        />
        <Line type="monotone" dataKey="tvl" stroke="#8884d8" />
      </LineChart>
    </ResponsiveContainer>
  )
}
```

### For D3.js

```javascript
const response = await fetch('https://api.llama.fi/protocol/jupiter-aggregator')
const data = await response.json()

const chartData = data.tvl.map(point => ({
  date: new Date(point.date * 1000),
  value: point.totalLiquidityUSD
}))

// Use with D3 scale and line generators
const x = d3.scaleTime()
  .domain(d3.extent(chartData, d => d.date))
  .range([0, width])

const y = d3.scaleLinear()
  .domain([0, d3.max(chartData, d => d.value)])
  .range([height, 0])

const line = d3.line()
  .x(d => x(d.date))
  .y(d => y(d.value))
```

---

## Real Examples

### Jupiter Aggregator

```bash
curl https://api.llama.fi/protocol/jupiter-aggregator | jq '{
  name: .name,
  website: .url,
  twitter: .twitter,
  currentTvl: .currentChainTvls,
  dataPoints: .tvl | length,
  category: .category,
  chains: .chains
}'
```

**Output:**
```json
{
  "name": "Jupiter Aggregator",
  "website": "https://jup.ag/",
  "twitter": "JupiterExchange",
  "currentTvl": {
    "Solana": 2500000000
  },
  "dataPoints": 450,
  "category": "DEX Aggregator",
  "chains": ["Solana"]
}
```

### Drift Trade

```bash
curl https://api.llama.fi/protocol/drift-trade | jq '{
  name: .name,
  website: .url,
  twitter: .twitter,
  github: .github,
  audits: .audit_links,
  currentTvl: .currentChainTvls.Solana,
  category: .category,
  oracles: .oraclesBreakdown
}'
```

**Output:**
```json
{
  "name": "Drift Trade",
  "website": "https://app.drift.trade/ref/defillama",
  "twitter": "DriftProtocol",
  "github": ["drift-labs/protocol-v2"],
  "audits": [
    "https://github.com/Zellic/publications/blob/master/Drift%20-%20Audit%20Report.pdf"
  ],
  "currentTvl": 150000000,
  "category": "Derivatives",
  "oracles": [
    {
      "name": "Pyth",
      "type": "Primary"
    },
    {
      "name": "Switchboard",
      "type": "Secondary"
    }
  ]
}
```

### Kamino Lend

```bash
curl https://api.llama.fi/protocol/kamino-lend | jq '{
  name: .name,
  website: .url,
  description: .description,
  currentTvl: .currentChainTvls.Solana,
  last30Days: .tvl[-30:] | map(.totalLiquidityUSD)
}'
```

---

## Additional Endpoints

### Get Current TVL Only

**Endpoint:**
```
GET https://api.llama.fi/tvl/{slug}
```

**Example:**
```bash
curl https://api.llama.fi/tvl/jupiter-aggregator
# Output: 2500000000
```

**Returns:** Single number (current TVL in USD)

**Use Case:** Faster if you only need current TVL without historical data

---

### Get Protocol Configuration

**Endpoint:**
```
GET https://api.llama.fi/config/smol/{slug}
```

**Example:**
```bash
curl https://api.llama.fi/config/smol/drift-trade
```

**Returns:** Protocol configuration metadata (minimal response)

---

### Get Inflows/Outflows Data

**Endpoint:**
```
GET https://api.llama.fi/inflows/{slug}/{startTimestamp}?end={endTimestamp}
```

**Example:**
```bash
curl "https://api.llama.fi/inflows/jupiter-aggregator/1700000000?end=1700086400"
```

**Response:**
```json
{
  "outflows": 50000000,
  "oldTokens": {
    "date": 1700000000,
    "tvl": {
      "SOL": 1000000,
      "USDC": 2000000
    }
  },
  "currentTokens": {
    "date": 1700086400,
    "tvl": {
      "SOL": 1050000,
      "USDC": 1950000
    }
  }
}
```

**Use Case:** Calculate net inflows/outflows between two time periods

---

### Get All Protocols (List)

**Endpoint:**
```
GET https://api.llama.fi/protocols
```

**Query Parameters:**
- `includeChains=true` - Include chain details

**Example:**
```bash
curl "https://api.llama.fi/protocols?includeChains=true"
```

**Returns:** Array of all protocols with summary data

---

### Get Protocol CSV Dataset

**Endpoint:**
```
GET https://api.llama.fi/dataset/{slug}
GET https://api.llama.fi/dataset/{slug}.csv
```

**Example:**
```bash
curl https://api.llama.fi/dataset/jupiter-aggregator.csv > jupiter-tvl.csv
```

**Returns:** CSV file with historical TVL data

**Format:**
```csv
date,totalLiquidityUSD
1640995200,125000000
1641081600,130000000
```

---

## Best Practices

### 1. Rate Limiting

```javascript
class RateLimitedFetcher {
  constructor(delayMs = 200) {
    this.delayMs = delayMs
    this.lastRequest = 0
  }

  async fetch(url) {
    const now = Date.now()
    const timeSinceLastRequest = now - this.lastRequest

    if (timeSinceLastRequest < this.delayMs) {
      await new Promise(resolve =>
        setTimeout(resolve, this.delayMs - timeSinceLastRequest)
      )
    }

    this.lastRequest = Date.now()
    return fetch(url)
  }
}

const fetcher = new RateLimitedFetcher(200)
```

**Recommendations:**
- Minimum 200ms delay between requests
- DefiLlama has no official rate limit, but be respectful
- Cache responses locally

---

### 2. Error Handling

```javascript
async function fetchProtocolWithRetry(slug, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      const response = await fetch(`https://api.llama.fi/protocol/${slug}`)

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }

      return await response.json()
    } catch (error) {
      console.error(`Attempt ${i + 1}/${maxRetries} failed:`, error.message)

      if (i === maxRetries - 1) {
        throw new Error(`Failed to fetch ${slug} after ${maxRetries} attempts`)
      }

      // Exponential backoff
      await new Promise(resolve =>
        setTimeout(resolve, Math.pow(2, i) * 1000)
      )
    }
  }
}
```

---

### 3. Response Validation

```javascript
function validateProtocolResponse(data, slug) {
  const required = ['id', 'name', 'tvl', 'currentChainTvls']
  const missing = required.filter(field => !(field in data))

  if (missing.length > 0) {
    throw new Error(
      `Protocol ${slug} response missing fields: ${missing.join(', ')}`
    )
  }

  if (!Array.isArray(data.tvl) || data.tvl.length === 0) {
    throw new Error(`Protocol ${slug} has no TVL data`)
  }

  return true
}
```

---

### 4. Caching Strategy

```javascript
const NodeCache = require('node-cache')
const cache = new NodeCache({ stdTTL: 1200 }) // 20 minutes

async function getCachedProtocol(slug) {
  // Check cache first
  const cached = cache.get(slug)
  if (cached) {
    console.log(`Cache hit: ${slug}`)
    return cached
  }

  // Fetch from API
  console.log(`Cache miss: ${slug} - fetching from API`)
  const response = await fetch(`https://api.llama.fi/protocol/${slug}`)
  const data = await response.json()

  // Store in cache
  cache.set(slug, data)

  return data
}
```

---

### 5. Handling Large Responses

Some protocols have very large responses (10MB+) due to extensive token breakdowns.

```javascript
async function fetchProtocol(slug, options = {}) {
  const {
    includeTokens = false,
    maxSize = 10 * 1024 * 1024  // 10MB limit
  } = options

  const url = includeTokens
    ? `https://api.llama.fi/protocol/${slug}?restrictResponseSize=false`
    : `https://api.llama.fi/protocol/${slug}`

  const response = await fetch(url)
  const contentLength = response.headers.get('content-length')

  if (contentLength && parseInt(contentLength) > maxSize) {
    console.warn(`Response for ${slug} is large (${contentLength} bytes)`)
  }

  return response.json()
}
```

---

### 6. Timestamp Handling

```javascript
// Unix timestamp (seconds) to JavaScript Date
function unixToDate(timestamp) {
  return new Date(timestamp * 1000)
}

// JavaScript Date to Unix timestamp (seconds)
function dateToUnix(date) {
  return Math.floor(date.getTime() / 1000)
}

// Format for display
function formatDate(timestamp) {
  return new Date(timestamp * 1000).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  })
}
```

---

### 7. Data Transformation Pipeline

```javascript
class ProtocolDataTransformer {
  constructor(rawData) {
    this.raw = rawData
  }

  getMetadata() {
    return {
      id: this.raw.id,
      name: this.raw.name,
      symbol: this.raw.symbol,
      description: this.raw.description,
      category: this.raw.category,
      chains: this.raw.chains
    }
  }

  getLinks() {
    return {
      website: this.raw.url,
      twitter: this.raw.twitter ? `https://twitter.com/${this.raw.twitter}` : null,
      github: this.raw.github?.[0] ? `https://github.com/${this.raw.github[0]}` : null,
      coingecko: this.raw.gecko_id
        ? `https://www.coingecko.com/en/coins/${this.raw.gecko_id}`
        : null
    }
  }

  getCurrentTVL() {
    return Object.values(this.raw.currentChainTvls || {})
      .reduce((sum, tvl) => sum + tvl, 0)
  }

  getHistoricalTVL(options = {}) {
    const { days, format = 'object' } = options

    let tvlData = this.raw.tvl

    if (days) {
      tvlData = tvlData.slice(-days)
    }

    if (format === 'csv') {
      return this.toCSV(tvlData)
    }

    return tvlData.map(point => ({
      date: new Date(point.date * 1000).toISOString(),
      timestamp: point.date,
      tvl: point.totalLiquidityUSD
    }))
  }

  toCSV(tvlData) {
    const header = 'date,timestamp,tvl\n'
    const rows = tvlData.map(point =>
      `${new Date(point.date * 1000).toISOString()},${point.date},${point.totalLiquidityUSD}`
    ).join('\n')

    return header + rows
  }

  toJSON() {
    return {
      metadata: this.getMetadata(),
      links: this.getLinks(),
      currentTVL: this.getCurrentTVL(),
      historicalTVL: this.getHistoricalTVL()
    }
  }
}

// Usage
const response = await fetch('https://api.llama.fi/protocol/jupiter-aggregator')
const rawData = await response.json()
const transformer = new ProtocolDataTransformer(rawData)

console.log(transformer.toJSON())
```

---

## Summary

### Key Points

1. **Single endpoint:** `https://api.llama.fi/protocol/{slug}` returns all data
2. **No authentication required**
3. **Rate limit:** No official limit, but use 200ms delays
4. **Cache duration:** API caches for 20 minutes
5. **Timestamp format:** Unix seconds (not milliseconds)
6. **Response size:** Can be large (5-10MB) for some protocols

### Quick Reference

| Need | Endpoint | Returns |
|------|----------|---------|
| Full protocol data | `/protocol/{slug}` | Everything |
| Current TVL only | `/tvl/{slug}` | Number |
| All protocols list | `/protocols` | Array |
| CSV export | `/dataset/{slug}.csv` | CSV file |
| Inflows/outflows | `/inflows/{slug}/{timestamp}` | Flow data |

### Recommended Workflow

```bash
# 1. Find protocol slug
curl https://api.llama.fi/protocols | jq '.[] | select(.name | contains("Jupiter")) | .slug'

# 2. Get full protocol data
curl https://api.llama.fi/protocol/jupiter-aggregator > jupiter.json

# 3. Extract specific fields
jq '{name, website: .url, tvl: .currentChainTvls, charts: .tvl}' jupiter.json

# 4. Cache locally for 20 minutes
# 5. Display on your website
```

---

**Generated:** 2024-12-07
**API Base URL:** https://api.llama.fi
**Documentation Source:** DefiLlama codebase analysis
