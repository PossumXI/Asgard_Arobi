// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/asgard/pandora/internal/services"
)

const authClaimsKey = "auth_claims"

// jsonResponse sends a JSON response with the given status code and data.
func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// jsonError sends a JSON error response.
func jsonError(w http.ResponseWriter, status int, message string, code string) {
	jsonResponse(w, status, map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"code":    code,
			"status":  status,
		},
	})
}

// getUserIDFromContext extracts the user ID from the request context.
func getUserIDFromContext(r *http.Request) string {
	claims, ok := getAuthClaimsFromContext(r)
	if !ok {
		return ""
	}
	return claims.UserID
}

// contextWithAuthClaims adds auth claims to the context.
func contextWithAuthClaims(ctx context.Context, claims services.TokenClaims) context.Context {
	return context.WithValue(ctx, authClaimsKey, claims)
}

// getAuthClaimsFromContext extracts auth claims from the request context.
func getAuthClaimsFromContext(r *http.Request) (services.TokenClaims, bool) {
	value := r.Context().Value(authClaimsKey)
	if value == nil {
		return services.TokenClaims{}, false
	}
	claims, ok := value.(services.TokenClaims)
	return claims, ok
}

// parsePaginationParams extracts pagination parameters from the request.
func parsePaginationParams(r *http.Request) (limit, offset int) {
	limit = 20
	offset = 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l := parseInt(limitStr); l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o := parseInt(offsetStr); o >= 0 {
			offset = o
		}
	}

	return limit, offset
}

// parseInt safely parses an integer string.
func parseInt(s string) int {
	var result int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		result = result*10 + int(c-'0')
	}
	return result
}

// validateEmail performs basic email validation.
func validateEmail(email string) bool {
	if len(email) < 3 || len(email) > 255 {
		return false
	}
	hasAt := false
	hasDot := false
	for i, c := range email {
		if c == '@' {
			if hasAt || i == 0 || i == len(email)-1 {
				return false
			}
			hasAt = true
		}
		if c == '.' && hasAt && i < len(email)-1 {
			hasDot = true
		}
	}
	return hasAt && hasDot
}

// validatePassword performs basic password validation.
func validatePassword(password string) bool {
	if len(password) < 8 || len(password) > 128 {
		return false
	}
	return true
}
