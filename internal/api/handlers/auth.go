// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/asgard/pandora/internal/services"
	"github.com/asgard/pandora/internal/utils"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authService *services.AuthService
}

// NewAuthHandler creates a new auth handler.
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// SignIn handles POST /api/auth/signin
func (h *AuthHandler) SignIn(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handleError(w, utils.ErrBadRequest)
		return
	}

	if !validateEmail(req.Email) {
		handleError(w, utils.NewAPIError("INVALID_EMAIL", "Invalid email address", http.StatusBadRequest))
		return
	}

	if !validatePassword(req.Password) {
		handleError(w, utils.NewAPIError("INVALID_PASSWORD", "Password must be at least 8 characters", http.StatusBadRequest))
		return
	}

	user, token, err := h.authService.SignIn(req.Email, req.Password)
	if err != nil {
		switch err {
		case services.ErrEmailNotVerified:
			handleError(w, utils.NewAPIError("EMAIL_NOT_VERIFIED", "Email verification required for government access", http.StatusForbidden))
		case services.ErrFido2Required:
			handleError(w, utils.NewAPIError("FIDO2_REQUIRED", "FIDO2 authentication required for government access", http.StatusForbidden))
		default:
			handleError(w, utils.WrapAPIError(err, "INVALID_CREDENTIALS", "Invalid email or password", http.StatusUnauthorized))
		}
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"user":  user,
		"token": token,
	})
}

// SignUp handles POST /api/auth/signup
func (h *AuthHandler) SignUp(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email            string `json:"email"`
		Password         string `json:"password"`
		FullName         string `json:"fullName"`
		OrganizationType string `json:"organizationType,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	isGovernment := req.OrganizationType == "government"
	user, token, err := h.authService.SignUp(req.Email, req.Password, req.FullName, isGovernment)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"user":  user,
		"token": token,
	})
}

// SignOut handles POST /api/auth/signout
func (h *AuthHandler) SignOut(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token != "" {
		claims, err := h.authService.ValidateToken(token)
		if err == nil {
			if err := h.authService.RevokeToken(claims.TokenID, claims.UserID); err != nil {
				http.Error(w, "Failed to revoke token", http.StatusInternalServerError)
				return
			}
		}
	}
	jsonResponse(w, http.StatusOK, map[string]string{"message": "Signed out successfully"})
}

// RefreshToken handles POST /api/auth/refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	claims, err := h.authService.ValidateToken(token)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	newToken, err := h.authService.RefreshToken(claims)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"token": newToken})
}

// RequestPasswordReset handles POST /api/auth/password-reset/request
func (h *AuthHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.authService.RequestPasswordReset(req.Email); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"message": "Password reset email sent"})
}

// ResetPassword handles POST /api/auth/password-reset/confirm
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.authService.ResetPassword(req.Token, req.Password); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"message": "Password reset successfully"})
}

// VerifyEmail handles POST /api/auth/verify-email
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.authService.VerifyEmail(req.Token); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"message": "Email verified successfully"})
}

// StartFido2Registration handles POST /api/auth/fido2/register/start
func (h *AuthHandler) StartFido2Registration(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	options, err := h.authService.StartFido2Registration(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, options)
}

// CompleteFido2Registration handles POST /api/auth/fido2/register/complete
func (h *AuthHandler) CompleteFido2Registration(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.authService.CompleteFido2Registration(userID, r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"message": "FIDO2 registration completed"})
}

// StartFido2Auth handles POST /api/auth/fido2/auth/start
func (h *AuthHandler) StartFido2Auth(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}

	_ = json.NewDecoder(r.Body).Decode(&req)

	email := req.Email
	if email == "" {
		email = r.URL.Query().Get("email")
	}
	if email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	options, err := h.authService.StartFido2Auth(email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, options)
}

// CompleteFido2Auth handles POST /api/auth/fido2/auth/complete
func (h *AuthHandler) CompleteFido2Auth(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email      string                 `json:"email"`
		Credential map[string]interface{} `json:"credential"`
	}

	_ = json.NewDecoder(r.Body).Decode(&req)

	email := req.Email
	if email == "" {
		email = r.URL.Query().Get("email")
	}
	if email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	user, token, err := h.authService.CompleteFido2Auth(email, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"user":  user,
		"token": token,
	})
}

// RequireAuth is middleware that requires authentication.
func (h *AuthHandler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := h.authService.ValidateToken(token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := contextWithAuthClaims(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth is middleware that optionally authenticates.
func (h *AuthHandler) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token != "" {
			if claims, err := h.authService.ValidateToken(token); err == nil {
				ctx := contextWithAuthClaims(r.Context(), claims)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Helper functions

func extractToken(r *http.Request) string {
	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}

	// Check query parameter
	return r.URL.Query().Get("token")
}
