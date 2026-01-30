package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/asgard/pandora/internal/services"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// SignInRequest represents a sign in request.
type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	AccessCode string `json:"accessCode,omitempty"`
}

// SignUpRequest represents a sign up request.
type SignUpRequest struct {
	Email            string `json:"email"`
	Password         string `json:"password"`
	FullName         string `json:"fullName"`
	OrganizationType string `json:"organizationType,omitempty"`
}

// AuthResponse represents an auth response.
type AuthResponse struct {
	User  UserResponse `json:"user"`
	Token string       `json:"token"`
}

// UserResponse represents a user in API responses.
type UserResponse struct {
	ID               string  `json:"id"`
	Email            string  `json:"email"`
	FullName         string  `json:"fullName"`
	SubscriptionTier string  `json:"subscriptionTier"`
	IsGovernment     bool    `json:"isGovernment"`
	EmailVerified    bool    `json:"emailVerified"`
	CreatedAt        string  `json:"createdAt"`
	UpdatedAt        string  `json:"updatedAt"`
	LastLogin        *string `json:"lastLogin"`
}

// handleSignIn handles POST /api/auth/signin
func (s *Server) handleSignIn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	var req SignInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	ctx := r.Context()

	// Query user from database
	var userID, passwordHash, fullName, subscriptionTier string
	var isGovernment bool
	var createdAt, updatedAt time.Time

	if s.pgDB == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Database not configured", "DB_NOT_CONFIGURED")
		return
	}

	err := s.pgDB.QueryRowContext(ctx,
		`SELECT id, password_hash, full_name, subscription_tier, is_government, created_at, updated_at
		 FROM users WHERE email = $1`, req.Email).Scan(
		&userID, &passwordHash, &fullName, &subscriptionTier, &isGovernment, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusUnauthorized, "Invalid credentials", "INVALID_CREDENTIALS")
			return
		}
		s.writeError(w, http.StatusInternalServerError, "Failed to query user", "DB_ERROR")
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		s.writeError(w, http.StatusUnauthorized, "Invalid credentials", "INVALID_CREDENTIALS")
		return
	}

	if s.accessCodeService != nil {
		required, err := s.accessCodeService.RequiresAccessCode(ctx, userID)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "Failed to validate access code", "ACCESS_CODE_ERROR")
			return
		}
		if required {
			record, err := s.accessCodeService.ValidateForUser(ctx, req.AccessCode, userID, "portal")
			if err != nil {
				s.writeAccessCodeError(w, err)
				return
			}
			_ = record
		}
	}

	// Update last login
	_, _ = s.pgDB.ExecContext(ctx, "UPDATE users SET last_login = $1 WHERE id = $2", time.Now().UTC(), userID)

	token, err := generateTokenForUser(userID, req.Email, subscriptionTier, isGovernment)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to create token", "TOKEN_ERROR")
		return
	}

	s.writeJSON(w, http.StatusOK, AuthResponse{
		User: UserResponse{
			ID:               userID,
			Email:            req.Email,
			FullName:         fullName,
			SubscriptionTier: subscriptionTier,
			IsGovernment:     isGovernment,
			EmailVerified:    true,
			CreatedAt:        createdAt.Format(time.RFC3339),
			UpdatedAt:        updatedAt.Format(time.RFC3339),
		},
		Token: token,
	})
}

// handleSignUp handles POST /api/auth/signup
func (s *Server) handleSignUp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	var req SignUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body", "INVALID_REQUEST")
		return
	}

	// Validate
	if req.Email == "" || req.Password == "" || req.FullName == "" {
		s.writeError(w, http.StatusBadRequest, "Email, password, and full name are required", "MISSING_FIELDS")
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to process password", "INTERNAL_ERROR")
		return
	}

	if s.pgDB == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Database not configured", "DB_NOT_CONFIGURED")
		return
	}

	// Insert user
	userID := uuid.New()
	now := time.Now().UTC()
	isGov := req.OrganizationType == "government"

	_, err = s.pgDB.ExecContext(r.Context(),
		`INSERT INTO users (id, email, password_hash, full_name, subscription_tier, is_government, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		userID, req.Email, string(hashedPassword), req.FullName, "free", isGov, now, now)

	if err != nil {
		// Handle duplicate email
		s.writeError(w, http.StatusConflict, "Email already exists", "EMAIL_EXISTS")
		return
	}

	token, err := generateTokenForUser(userID.String(), req.Email, "free", isGov)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to create token", "TOKEN_ERROR")
		return
	}

	s.writeJSON(w, http.StatusCreated, AuthResponse{
		User: UserResponse{
			ID:               userID.String(),
			Email:            req.Email,
			FullName:         req.FullName,
			SubscriptionTier: "free",
			IsGovernment:     isGov,
			EmailVerified:    false,
			CreatedAt:        now.Format(time.RFC3339),
			UpdatedAt:        now.Format(time.RFC3339),
		},
		Token: token,
	})
}

// handleSignOut handles POST /api/auth/signout
func (s *Server) handleSignOut(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"message": "Signed out successfully"})
}

// handleRefreshToken handles POST /api/auth/refresh
func (s *Server) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	if s.pgDB == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Database not configured", "DB_NOT_CONFIGURED")
		return
	}

	rawToken := extractToken(r)
	if rawToken == "" {
		s.writeError(w, http.StatusUnauthorized, "Missing token", "TOKEN_REQUIRED")
		return
	}
	userID, _, tier, isGovernment, err := parseJWTClaims(rawToken)
	if err != nil || userID == "" {
		s.writeError(w, http.StatusUnauthorized, "Invalid token", "INVALID_TOKEN")
		return
	}

	var email string
	err = s.pgDB.QueryRowContext(r.Context(), "SELECT email FROM users WHERE id = $1", userID).Scan(&email)
	if err != nil {
		s.writeError(w, http.StatusUnauthorized, "Invalid token", "INVALID_TOKEN")
		return
	}

	token, err := generateTokenForUser(userID, email, tier, isGovernment)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to create token", "TOKEN_ERROR")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (s *Server) writeAccessCodeError(w http.ResponseWriter, err error) {
	switch err {
	case services.ErrAccessCodeRequired:
		s.writeError(w, http.StatusForbidden, "Access code required", "ACCESS_CODE_REQUIRED")
	case services.ErrAccessCodeExpired:
		s.writeError(w, http.StatusForbidden, "Access code expired", "ACCESS_CODE_EXPIRED")
	case services.ErrAccessCodeRevoked:
		s.writeError(w, http.StatusForbidden, "Access code revoked", "ACCESS_CODE_REVOKED")
	case services.ErrAccessCodeScopeMismatch:
		s.writeError(w, http.StatusForbidden, "Access code scope mismatch", "ACCESS_CODE_SCOPE")
	case services.ErrAccessCodeUsageExceeded:
		s.writeError(w, http.StatusForbidden, "Access code usage exceeded", "ACCESS_CODE_EXHAUSTED")
	default:
		s.writeError(w, http.StatusUnauthorized, "Invalid access code", "ACCESS_CODE_INVALID")
	}
}

func generateTokenForUser(userID, email, tier string, isGovernment bool) (string, error) {
	now := time.Now().UTC()
	role := "user"
	if isGovernment {
		role = "government"
	}

	claims := jwt.MapClaims{
		"user_id":          userID,
		"email":            email,
		"subscription_tier": tier,
		"is_government":    isGovernment,
		"role":             role,
		"iat":              now.Unix(),
		"exp":              now.Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(getJWTSecret())
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return signed, nil
}
