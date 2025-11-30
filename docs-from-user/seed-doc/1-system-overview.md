# 1. System Overview

## 1.1 Objective

Build a production-ready Go service that:
1. Fetches Switchboard oracle data from DefiLlama APIs
2. Implements custom aggregation logic for enhanced metrics
3. Uses incremental updates to minimize API calls and bandwidth
4. Outputs structured JSON files for consumption by other services/websites
5. Provides real-time metrics and historical data tracking

## 1.2 Key Features

| Feature | Description |
|---------|-------------|
| Custom Aggregation | Calculate derived metrics (7d/30d changes, growth rates, rankings) |
| Incremental Updates | Only fetch/process new data since last successful update |
| Multi-Source Correlation | Combine oracle data with protocol metadata |
| Historical Tracking | Maintain 90-day rolling history of snapshots |
| Fault Tolerance | Retry logic, graceful degradation, state recovery |
| JSON Output | Multiple output formats (full, compact, summary) |
| Graceful Shutdown | Proper signal handling and cleanup |
| Monitoring | Prometheus metrics and health endpoints |

## 1.3 Data Flow Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         DefiLlama APIs                               │
│  ┌─────────────────────┐  ┌─────────────────────────────────────┐  │
│  │ GET /oracles        │  │ GET /lite/protocols2?b=2            │  │
│  │ (Oracle TVS data)   │  │ (Protocol metadata)                 │  │
│  └──────────┬──────────┘  └──────────────────┬──────────────────┘  │
└─────────────┼────────────────────────────────┼──────────────────────┘
              │                                │
              ▼                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      Go Extraction Service                          │
│                                                                     │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────────────┐   │
│  │   Fetcher    │──▶│  Aggregator  │──▶│  Incremental Store   │   │
│  │  (HTTP Client)│   │  (Business   │   │  (State Management)  │   │
│  │              │   │   Logic)     │   │                      │   │
│  └──────────────┘   └──────────────┘   └──────────┬───────────┘   │
│                                                    │               │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │                     JSON File Writer                          │ │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────────────────┐  │ │
│  │  │ Full JSON  │  │ Min JSON   │  │ Summary JSON           │  │ │
│  │  └────────────┘  └────────────┘  └────────────────────────┘  │ │
│  └──────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        Output Directory                             │
│  ./data/                                                            │
│  ├── switchboard-oracle-data.json      (Full data with history)    │
│  ├── switchboard-oracle-data.min.json  (Minified version)          │
│  ├── switchboard-summary.json          (Current snapshot only)     │
│  └── state.json                        (Incremental update state)  │
└─────────────────────────────────────────────────────────────────────┘
```

## 1.4 Update Cycle

```
┌─────────────────────────────────────────────────────────────────┐
│                    15-Minute Update Cycle                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. Load State (state.json)                                     │
│     └─▶ Get last_updated timestamp                              │
│                                                                 │
│  2. Fetch API Data (parallel)                                   │
│     ├─▶ GET /oracles                                            │
│     └─▶ GET /lite/protocols2?b=2                                │
│                                                                 │
│  3. Check for New Data                                          │
│     └─▶ Compare latest chart timestamp vs last_updated          │
│         └─▶ If no new data: EXIT (skip processing)              │
│                                                                 │
│  4. Filter & Aggregate                                          │
│     ├─▶ Filter protocols using Switchboard                      │
│     ├─▶ Calculate TVS metrics                                   │
│     ├─▶ Compute derived metrics (changes, growth)               │
│     └─▶ Generate rankings                                       │
│                                                                 │
│  5. Merge with History                                          │
│     ├─▶ Load existing snapshots                                 │
│     ├─▶ Append new snapshot                                     │
│     └─▶ Prune snapshots older than 90 days                      │
│                                                                 │
│  6. Write Output Files                                          │
│     ├─▶ Write full JSON                                         │
│     ├─▶ Write minified JSON                                     │
│     ├─▶ Write summary JSON                                      │
│     └─▶ Update state.json                                       │
│                                                                 │
│  7. Log Metrics                                                 │
│     └─▶ Protocol count, TVS, duration, etc.                     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---
