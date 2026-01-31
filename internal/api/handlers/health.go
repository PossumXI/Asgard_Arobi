// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"net/http"
	"time"
)

// HealthHandler handles health check endpoints.
type HealthHandler struct{}

// NewHealthHandler creates a new health handler.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Health handles GET /api/health
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "nysus",
		"version":   "1.0.0",
	})
}
