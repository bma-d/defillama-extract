# Story 1.1: Initialize Go Module and Project Structure

Status: ready-for-dev

## Story

As a **developer**,
I want **a properly initialized Go module with the standard project layout**,
so that **I have a clean foundation to build the extraction service**.

## Acceptance Criteria

1. **Given** a new project directory **When** `go mod init github.com/switchboard-xyz/defillama-extract` is run **Then** `go.mod` file is created with the correct module path
2. **Given** an initialized Go module **When** the standard directory structure is created **Then** the following directories exist:
   - `cmd/extractor/` (entry point location)
   - `internal/config/` (configuration package)
   - `internal/models/` (data structures)
   - `internal/api/` (HTTP client)
   - `internal/aggregator/` (data processing)
   - `internal/storage/` (file output)
   - `configs/` (default configuration)
   - `testdata/` (test fixtures)
3. **Given** the directory structure exists **When** a minimal `cmd/extractor/main.go` is created **Then** it compiles successfully with `go build ./...`
4. **Given** the project is initialized **When** `Makefile` is created **Then** the following targets work:
   - `make build` creates binary at `bin/extractor`
   - `make test` runs tests (passes with no tests)
   - `make lint` runs golangci-lint (requires `.golangci.yml`)
5. **Given** the project is initialized **When** `.gitignore` is created **Then** it excludes:
   - `data/` (output files)
   - `bin/` (build output)
   - `*.exe` (Windows binaries)
   - `vendor/` (vendored dependencies)
   - `.DS_Store` (macOS metadata)
6. **Given** the project is initialized **When** `.golangci.yml` is created **Then** it configures standard linters for Go projects
7. **Given** all files are created **When** `go build ./...` is executed **Then** build succeeds with zero errors and zero warnings

## Tasks / Subtasks

- [ ] Task 1: Initialize Go module (AC: 1)
  - [ ] 1.1: Run `go mod init github.com/switchboard-xyz/defillama-extract`
  - [ ] 1.2: Verify `go.mod` contains correct module path and Go version (1.21+)

- [ ] Task 2: Create directory structure (AC: 2)
  - [ ] 2.1: Create `cmd/extractor/` directory
  - [ ] 2.2: Create `internal/config/` directory
  - [ ] 2.3: Create `internal/models/` directory
  - [ ] 2.4: Create `internal/api/` directory
  - [ ] 2.5: Create `internal/aggregator/` directory
  - [ ] 2.6: Create `internal/storage/` directory
  - [ ] 2.7: Create `configs/` directory
  - [ ] 2.8: Create `testdata/` directory

- [ ] Task 3: Create minimal entry point (AC: 3)
  - [ ] 3.1: Create `cmd/extractor/main.go` with minimal `main()` function
  - [ ] 3.2: Verify it compiles: `go build ./cmd/extractor`

- [ ] Task 4: Create Makefile with build targets (AC: 4)
  - [ ] 4.1: Add `build` target that outputs to `bin/extractor`
  - [ ] 4.2: Add `test` target that runs `go test ./...`
  - [ ] 4.3: Add `lint` target that runs `golangci-lint run`
  - [ ] 4.4: Add `clean` target to remove build artifacts
  - [ ] 4.5: Add `all` target as default (lint, test, build)

- [ ] Task 5: Create .gitignore file (AC: 5)
  - [ ] 5.1: Add `data/` exclusion
  - [ ] 5.2: Add `bin/` exclusion
  - [ ] 5.3: Add `*.exe` exclusion
  - [ ] 5.4: Add `vendor/` exclusion
  - [ ] 5.5: Add `.DS_Store` exclusion

- [ ] Task 6: Create .golangci.yml configuration (AC: 6)
  - [ ] 6.1: Configure standard linters (gofmt, govet, errcheck, staticcheck)
  - [ ] 6.2: Set appropriate timeouts and exclusions

- [ ] Task 7: Final verification (AC: 7)
  - [ ] 7.1: Run `go build ./...` and verify success
  - [ ] 7.2: Run `make build` and verify binary created
  - [ ] 7.3: Run `make test` and verify passes
  - [ ] 7.4: Run `make lint` and verify passes (warnings OK)

## Dev Notes

### Technical Guidance

- **Go Version**: Minimum Go 1.21 required for `log/slog` package (ADR-004)
- **Module Path**: Use `github.com/switchboard-xyz/defillama-extract` as per PRD
- **Standard Layout**: Follow Go project layout conventions with `cmd/` and `internal/`
- The `internal/` directory is enforced by Go compiler - packages inside cannot be imported by external projects

### Minimal main.go Template

```go
package main

import "fmt"

func main() {
    fmt.Println("defillama-extract starting...")
}
```

### Project Structure Notes

Per architecture/project-structure.md:

```
defillama-extract/
├── cmd/
│   └── extractor/
│       └── main.go           # Entry point
├── internal/
│   ├── config/               # Config loading
│   ├── models/               # Data structures
│   ├── api/                  # HTTP client
│   ├── aggregator/           # Data processing
│   └── storage/              # File operations
├── configs/
│   └── config.yaml           # Default config (created in Story 1.2)
├── testdata/                 # Test fixtures
├── go.mod
├── Makefile
├── .gitignore
└── .golangci.yml
```

### References

- [Source: docs/architecture/project-structure.md] - Complete directory structure
- [Source: docs/sprint-artifacts/tech-spec-epic-1.md#detailed-design] - Directory structure details
- [Source: docs/epics.md#story-11] - Original story definition
- [Source: docs/prd.md#cli-operation] - CLI requirements
- [Source: docs/architecture/testing-strategy.md] - Test organization, fixtures location, coverage requirements
- [Source: docs/architecture/architecture-decision-records-adrs.md#adr-004] - Structured logging with slog (Go 1.21+ requirement)

## Dev Agent Record

### Context Reference

- docs/sprint-artifacts/1-1-initialize-go-module-and-project-structure.context.xml

### Agent Model Used

### Debug Log References

### Completion Notes List

### File List

## Change Log

| Date | Author | Change |
|------|--------|--------|
| 2025-11-29 | SM Agent (Bob) | Initial story draft created from epics.md and tech-spec-epic-1.md |
| 2025-11-29 | SM Agent (Bob) | Added testing-strategy.md and ADR-004 citations per validation review |
| 2025-11-29 | SM Agent (Bob) | Generated story context XML, status changed to ready-for-dev |
