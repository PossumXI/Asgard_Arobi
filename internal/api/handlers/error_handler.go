// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"log"
	"net/http"

	"github.com/asgard/pandora/internal/api/validation"
	"github.com/asgard/pandora/internal/utils"
)

// handleError processes errors and sends appropriate HTTP responses.
func handleError(w http.ResponseWriter, err error) {
	if apiErr, ok := err.(*utils.APIError); ok {
		jsonError(w, apiErr.Status, apiErr.Message, apiErr.Code)
		return
	}

	if valErr, ok := err.(*validation.ValidationError); ok {
		jsonError(w, http.StatusBadRequest, valErr.Message, "VALIDATION_ERROR")
		return
	}

	// Log unexpected errors
	log.Printf("Unexpected error: %v", err)
	jsonError(w, http.StatusInternalServerError, "Internal server error", "INTERNAL_ERROR")
}
