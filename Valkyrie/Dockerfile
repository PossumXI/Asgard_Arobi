# VALKYRIE - Autonomous Flight System
# Multi-stage Docker build
#
# NOTE: Build context must be the repository root (.)
# because Valkyrie imports from the root Go module.

# Stage 1: Build
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files from root (Valkyrie is part of main module)
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy entire source (needed for internal packages)
COPY . .

# Build with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=1.0.0 -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o valkyrie \
    ./Valkyrie/cmd/valkyrie/main.go

# Stage 2: Runtime
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' valkyrie

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/valkyrie .

# Copy configuration
COPY --from=builder /app/Valkyrie/configs ./configs

# Create directories
RUN mkdir -p logs data && chown -R valkyrie:valkyrie /app

# Switch to non-root user
USER valkyrie

# Expose ports
EXPOSE 8093 9093

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8093/health || exit 1

# Default command
ENTRYPOINT ["./valkyrie"]

# Default arguments
CMD ["-sim", "-ai", "-security", "-failsafe", "-livefeed"]
