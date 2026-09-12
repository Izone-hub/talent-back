# ============================================================
# Stage 1: Build Go Backend Binary and Tools
# ============================================================
FROM golang:1.24-bookworm AS builder

WORKDIR /app

# Install Goose for migration management
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.24.1

# Download Go dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source files
COPY . .

# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server main.go

# ============================================================
# Stage 2: Production Runtime
# ============================================================
FROM debian:bookworm-slim AS runner

WORKDIR /app

# Install runtime prerequisites:
# - ca-certificates: TLS verification for external APIs (GitHub, etc.)
# - curl: Container health checks
# - netcat-openbsd: TCP connectivity wait-for-database check in entrypoint
# - docker.io: Docker client CLI for code sandbox isolation
# - gosu: Dropping privileges cleanly to appuser after socket setup
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        netcat-openbsd \
        docker.io \
        gosu \
    && rm -rf /var/lib/apt/lists/*

# Create dedicated non-root application user and group
RUN groupadd -r -g 10001 appuser \
    && useradd -r -u 10001 -g appuser -d /app appuser

# Copy Goose migration tool from builder
COPY --from=builder /go/bin/goose /usr/local/bin/goose

# Copy server binary and database migrations
COPY --from=builder --chown=appuser:appuser /app/server /app/server
COPY --from=builder --chown=appuser:appuser /app/sql/schema /app/sql/schema
COPY --from=builder --chown=appuser:appuser /app/templates /app/templates
COPY --from=builder --chown=appuser:appuser /app/entrypoint.sh /app/entrypoint.sh

RUN chmod +x /app/entrypoint.sh /usr/local/bin/goose \
    && mkdir -p /app/uploads/cv \
    && chown -R appuser:appuser /app

EXPOSE 5000

HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:5000/healthz || exit 1

ENTRYPOINT ["/app/entrypoint.sh"]
