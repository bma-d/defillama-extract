# Reference Documentation

This architecture is based on a comprehensive implementation specification. For detailed implementation guidance, refer to these seed documents:

| Section | Reference | Description |
|---------|-----------|-------------|
| **System Overview** | [1-system-overview.md](../docs-from-user/seed-doc/1-system-overview.md) | Objectives, key features, data flow |
| **Architecture Design** | [2-architecture-design.md](../docs-from-user/seed-doc/2-architecture-design.md) | Package structure, component responsibilities |
| **API Specifications** | [3-data-sources-api-specifications.md](../docs-from-user/seed-doc/3-data-sources-api-specifications.md) | DefiLlama API endpoints and response timing |
| **Data Models** | [4-data-models-structures.md](../docs-from-user/seed-doc/4-data-models-structures.md) | Go structs for API and internal models |
| **Core Components** | [5-core-components.md](../docs-from-user/seed-doc/5-core-components.md) | HTTP client, aggregator implementations |
| **Incremental Updates** | [6-incremental-update-strategy.md](../docs-from-user/seed-doc/6-incremental-update-strategy.md) | State and history management |
| **Aggregation Logic** | [7-custom-aggregation-logic-go-implementation.md](../docs-from-user/seed-doc/7-custom-aggregation-logic-go-implementation.md) | Metric calculations, protocol filtering |
| **Storage & Caching** | [8-storage-caching.md](../docs-from-user/seed-doc/8-storage-caching.md) | Atomic file writer implementation |
| **Error Handling** | [9-error-handling-resilience.md](../docs-from-user/seed-doc/9-error-handling-resilience.md) | Retry logic, graceful degradation |
| **Configuration** | [10-configuration-environment.md](../docs-from-user/seed-doc/10-configuration-environment.md) | YAML config, environment variables |
| **Testing Strategy** | [11-testing-strategy-complete-implementation.md](../docs-from-user/seed-doc/11-testing-strategy-complete-implementation.md) | Table-driven tests, mocks, benchmarks |
| **Deployment** | [12-deployment-considerations.md](../docs-from-user/seed-doc/12-deployment-considerations.md) | Deployment options and considerations |
| **API Response Examples** | [13-api-response-examples.md](../docs-from-user/seed-doc/13-api-response-examples.md) | Sample API responses for testing |
| **Implementation Checklist** | [14-implementation-checklist.md](../docs-from-user/seed-doc/14-implementation-checklist.md) | Phased implementation guide |
| **Go Patterns** | [15-go-specific-patterns-idioms.md](../docs-from-user/seed-doc/15-go-specific-patterns-idioms.md) | slog, context, dependency injection patterns |
| **Operational Concerns** | [16-operational-concerns.md](../docs-from-user/seed-doc/16-operational-concerns.md) | Monitoring, logging, maintenance |
| **Main.go Implementation** | [17-complete-maingo-implementation.md](../docs-from-user/seed-doc/17-complete-maingo-implementation.md) | Complete entry point implementation |
| **Go Dependencies** | [appendix-a-go-dependencies.md](../docs-from-user/seed-doc/appendix-a-go-dependencies.md) | go.mod dependencies |
| **Quick Reference** | [appendix-b-quick-reference.md](../docs-from-user/seed-doc/appendix-b-quick-reference.md) | API endpoints, constants, output files |
