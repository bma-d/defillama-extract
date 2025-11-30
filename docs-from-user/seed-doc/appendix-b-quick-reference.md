# Appendix B: Quick Reference

## B.1 API Endpoints

| Endpoint | Purpose |
|----------|---------|
| `GET /oracles` | Oracle TVS data, charts, protocol lists |
| `GET /lite/protocols2?b=2` | Protocol metadata |

## B.2 Key Constants

| Constant | Value |
|----------|-------|
| Oracle Name | `"Switchboard"` |
| API Base URL | `https://api.llama.fi` |
| Update Interval | 15 minutes |
| History Retention | 90 days |
| HTTP Timeout | 30 seconds |
| Max Retries | 3 |

## B.3 Output Files

| File | Content |
|------|---------|
| `switchboard-oracle-data.json` | Full data with history |
| `switchboard-oracle-data.min.json` | Minified version |
| `switchboard-summary.json` | Current snapshot only |
| `state.json` | Incremental update state |

---

**Document Version:** 1.2.0 (Revised)
**Last Updated:** 2025-11-29
**Author:** Claude Code Assistant

## Changelog
- **v1.2.0**: Updated to latest stable versions - Go 1.24, Alpine 3.22, prometheus/client_golang v1.20.0
- **v1.1.0**: Added complete Go implementations for aggregation, enhanced testing section, added operational concerns, complete main.go, graceful shutdown, Prometheus metrics, health checks, API validation
- **v1.0.0**: Initial specification
