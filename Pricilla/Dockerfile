# PRICILLA - Precision Guidance System
# Multi-stage Docker build
#
# Copyright 2026 Arobi. All Rights Reserved.

# Stage 1: Build
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy go mod files from root (Pricilla is part of main module)
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy entire source (needed for internal packages)
COPY . .

# Ensure configs directory exists (may not have config files yet)
RUN mkdir -p Pricilla/configs

# Build Pricilla with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=1.0.0 -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o pricilla \
    ./Pricilla/cmd/pricilla/main.go

# Stage 2: Runtime
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' pricilla

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/pricilla .

# Copy Pricilla configs
COPY --from=builder /app/Pricilla/configs ./configs

# Create directories
RUN mkdir -p logs data && chown -R pricilla:pricilla /app

# Switch to non-root user
USER pricilla

# Expose ports (HTTP API and Metrics)
EXPOSE 8089 9089

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8089/health || exit 1

# Default command
ENTRYPOINT ["./pricilla"]

# Default arguments
CMD ["-http-port", "8089", "-metrics-port", "9089"]
