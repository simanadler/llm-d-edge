# Multi-stage build for llm-d Edge Router
# Stage 1: Build the Go binary
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o edge-router \
    ./cmd/edge-router

# Stage 2: Create minimal runtime image
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    python3 \
    py3-pip

# Create non-root user
RUN addgroup -g 1000 edgerouter && \
    adduser -D -u 1000 -G edgerouter edgerouter

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/edge-router /app/edge-router

# Copy example config
COPY config.example.yaml /app/config.example.yaml

# Create directories for models and cache
RUN mkdir -p /app/models /app/cache && \
    chown -R edgerouter:edgerouter /app

# Switch to non-root user
USER edgerouter

# Expose API port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Set entrypoint
ENTRYPOINT ["/app/edge-router"]

# Default command (can be overridden)
CMD ["--config", "/app/config.yaml", "--port", "8080"]