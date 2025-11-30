# Development Environment

> **Spec Reference:** [14-implementation-checklist.md](../docs-from-user/seed-doc/14-implementation-checklist.md)

## Prerequisites

- Go 1.24 or later
- `golangci-lint` (for linting)
- Make (for build targets)

## Setup Commands

```bash
# Clone and enter directory
git clone https://github.com/switchboard-xyz/defillama-extract.git
cd defillama-extract

# Download dependencies
go mod download

# Run tests
make test

# Build binary
make build

# Run linter
make lint

# Run once (development)
./bin/extractor --once --config configs/config.yaml
```
