// Package middleware provides HTTP middleware for the API server.
package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// Apply sets up all middleware for the HTTP server.
func Apply(handler http.Handler) http.Handler {
	// Request ID for tracing
	handler = middleware.RequestID(handler)

	// Real IP from proxy headers
	handler = middleware.RealIP(handler)

	// Structured logging
	handler = middleware.Logger(handler)

	// Panic recovery
	handler = middleware.Recoverer(handler)

	// Request timeout
	handler = middleware.Timeout(30 * time.Second)(handler)

	// Compression
	handler = middleware.Compress(5)(handler)

	return handler
}
