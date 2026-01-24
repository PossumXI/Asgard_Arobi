// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/asgard/pandora/internal/services"
	"github.com/go-chi/chi/v5"
)

// AdminHandler handles admin-only endpoints.
type AdminHandler struct {
	userService *services.UserService
}

// NewAdminHandler creates a new admin handler.
func NewAdminHandler(userService *services.UserService) *AdminHandler {
	return &AdminHandler{userService: userService}
}

type adminUser struct {
	ID               string `json:"id"`
	Email            string `json:"email"`
	FullName         string `json:"fullName"`
	SubscriptionTier string `json:"subscriptionTier"`
	IsGovernment     bool   `json:"isGovernment"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

type adminUserUpdate struct {
	FullName         *string `json:"fullName"`
	SubscriptionTier *string `json:"subscriptionTier"`
	IsGovernment     *bool   `json:"isGovernment"`
}

// ListUsers handles GET /api/admin/users
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	limit := 200
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed := parseInt(limitStr); parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	users, err := h.userService.ListUsers(limit)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error(), "USER_LIST_ERROR")
		return
	}

	response := make([]adminUser, 0, len(users))
	for _, user := range users {
		fullName := ""
		if user.FullName.Valid {
			fullName = user.FullName.String
		}
		response = append(response, adminUser{
			ID:               user.ID.String(),
			Email:            user.Email,
			FullName:         fullName,
			SubscriptionTier: user.SubscriptionTier,
			IsGovernment:     user.IsGovernment,
			CreatedAt:        user.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:        user.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}

	jsonResponse(w, http.StatusOK, response)
}

// UpdateUser handles PATCH /api/admin/users/{userId}
func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if strings.TrimSpace(userID) == "" {
		jsonError(w, http.StatusBadRequest, "User ID required", "INVALID_REQUEST")
		return
	}

	var req adminUserUpdate
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	updated, err := h.userService.UpdateAdminUser(userID, services.AdminUserUpdate{
		FullName:         req.FullName,
		SubscriptionTier: req.SubscriptionTier,
		IsGovernment:     req.IsGovernment,
	})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			jsonError(w, http.StatusNotFound, "User not found", "NOT_FOUND")
			return
		}
		if strings.Contains(err.Error(), "invalid subscription tier") {
			jsonError(w, http.StatusBadRequest, "Invalid subscription tier", "INVALID_TIER")
			return
		}
		jsonError(w, http.StatusInternalServerError, err.Error(), "USER_UPDATE_ERROR")
		return
	}

	fullName := ""
	if updated.FullName.Valid {
		fullName = updated.FullName.String
	}

	jsonResponse(w, http.StatusOK, adminUser{
		ID:               updated.ID.String(),
		Email:            updated.Email,
		FullName:         fullName,
		SubscriptionTier: updated.SubscriptionTier,
		IsGovernment:     updated.IsGovernment,
		CreatedAt:        updated.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:        updated.UpdatedAt.UTC().Format(time.RFC3339),
	})
}
