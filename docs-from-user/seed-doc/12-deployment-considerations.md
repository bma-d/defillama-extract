# 12. Deployment Considerations

## 12.1 Docker Configuration

```dockerfile
# Dockerfile
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.Version=${VERSION:-dev}" \
    -o extractor ./cmd/extractor

# Runtime image
FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/extractor .
COPY configs/config.yaml ./config.yaml

# Create non-root user
RUN adduser -D -g '' extractor
USER extractor

VOLUME /app/data

EXPOSE 9090

ENTRYPOINT ["./extractor"]
CMD ["--config", "config.yaml"]
```

## 12.2 Docker Compose

```yaml
# docker-compose.yml
version: '3.8'

services:
  extractor:
    build: .
    container_name: switchboard-extractor
    restart: unless-stopped
    volumes:
      - ./data:/app/data
      - ./config.yaml:/app/config.yaml:ro
    environment:
      - LOG_LEVEL=info
      - TZ=UTC
    ports:
      - "9090:9090"
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:9090/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
```

---
