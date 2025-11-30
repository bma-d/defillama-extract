.PHONY: all build test lint clean

all: lint test build

build:
	@mkdir -p bin
	go build -o bin/extractor ./cmd/extractor

test:
	go test ./...

lint:
	@export PATH="$$(go env GOBIN 2>/dev/null):$$(go env GOPATH 2>/dev/null)/bin:$$PATH"; \
	GCI=$$(command -v golangci-lint 2>/dev/null || \
		([ -x "$$(go env GOBIN)/golangci-lint" ] && echo "$$(go env GOBIN)/golangci-lint") || \
		([ -x "$$(go env GOPATH)/bin/golangci-lint" ] && echo "$$(go env GOPATH)/bin/golangci-lint")); \
	if [ -z "$$GCI" ] || ! "$$GCI" version >/dev/null 2>&1; then \
		echo "golangci-lint not found or misconfigured; installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.60.3; \
	fi; \
	GCI=$$(command -v golangci-lint 2>/dev/null || \
		([ -x "$$(go env GOBIN)/golangci-lint" ] && echo "$$(go env GOBIN)/golangci-lint") || \
		([ -x "$$(go env GOPATH)/bin/golangci-lint" ] && echo "$$(go env GOPATH)/bin/golangci-lint")); \
	if [ -z "$$GCI" ]; then \
		echo "golangci-lint installation failed"; exit 1; \
	fi; \
	"$$GCI" run

clean:
	rm -rf bin data
