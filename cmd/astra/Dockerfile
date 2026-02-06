# ASGARD Astra - Moltbook Social Agent
# Dockerfile for containerized deployment
#
# Build: docker build -t asgard/astra .
# Run: docker run -d --env-file .env asgard/astra
#
# Copyright 2026 Arobi. All Rights Reserved.

FROM golang:1.22-alpine AS builder

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /astra ./cmd/astra

# Final stage - minimal image
FROM alpine:3.19

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' astra

# Copy binary
COPY --from=builder /astra /usr/local/bin/astra

# Set timezone
ENV TZ=UTC

# Switch to non-root user
USER astra

# Health check
HEALTHCHECK --interval=5m --timeout=10s --start-period=30s --retries=3 \
  CMD pgrep astra || exit 1

# Default command - run scheduled agent
ENTRYPOINT ["/usr/local/bin/astra"]
