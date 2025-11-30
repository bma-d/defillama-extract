# Decision Summary

| Category | Decision | Version | Affects FRs | Rationale |
|----------|----------|---------|-------------|-----------|
| Language | Go | 1.23 | All | Compiled binary, excellent concurrency, comprehensive stdlib |
| Module Path | `github.com/switchboard-xyz/defillama-extract` | - | All | Standard Go module naming convention |
| HTTP Client | `net/http` (stdlib) | Go 1.23 | FR1-FR8 | No external dep needed; full control over retries |
| JSON | `encoding/json` (stdlib) | Go 1.23 | FR9-FR14, FR35-FR41 | Standard, sufficient performance |
| YAML Config | `gopkg.in/yaml.v3` | v3.0.1 | FR49-FR52 | De facto standard for Go YAML parsing |
| Logging | `log/slog` (stdlib) | Go 1.23 | FR53-FR56 | Structured logging built-in; JSON/text output |
| CLI Flags | `flag` (stdlib) | Go 1.23 | FR42-FR48 | Simple flags; no subcommands needed |
| Testing | `testing` (stdlib) | Go 1.23 | NFR17-18 | Table-driven tests; no framework needed |
| Linting | `golangci-lint` | latest | NFR18 | Standard Go linter aggregator |
| Build System | Makefile | - | NFR19 | Standard for Go projects |
| Monitoring | `prometheus/client_golang` | v1.20.0 | Post-MVP | Industry standard (scoped to post-MVP) |
