# Deployment Architecture

> **Spec Reference:** [12-deployment-considerations.md](../docs-from-user/seed-doc/12-deployment-considerations.md), [16-operational-concerns.md](../docs-from-user/seed-doc/16-operational-concerns.md)

## Local Execution

```bash
# Run once
./extractor --once --config config.yaml

# Run as daemon
./extractor --config config.yaml
```

## Build Targets (Makefile)

```makefile
.PHONY: build test lint clean

build:
	go build -o bin/extractor ./cmd/extractor

test:
	go test -v ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/ data/
```
