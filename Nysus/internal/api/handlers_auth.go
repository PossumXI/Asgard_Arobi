package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// SignInRequest represents a sign in request.
type SignInRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
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

	// In production, this would query the database
	// For now, create a simulated response
	ctx := r.Context()

	// Query user from database
	var userID, passwordHash, fullName, subscriptionTier string
	var isGovernment bool
	var createdAt, updatedAt time.Time

	err := s.pgDB.QueryRowContext(ctx,
		`SELECT id, password_hash, full_name, subscription_tier, is_government, created_at, updated_at 
		 FROM users WHERE email = $1`, req.Email).Scan(
		&userID, &passwordHash, &fullName, &subscriptionTier, &isGovernment, &createdAt, &updatedAt)

	if err != nil {
		// For demo: create a mock user if not found
		userID = uuid.New().String()
		fullName = "Demo User"
		subscriptionTier = "observer"
		isGovernment = false
		createdAt = time.Now().UTC()
		updatedAt = time.Now().UTC()
	} else {
		// Verify password
		if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
			s.writeError(w, http.StatusUnauthorized, "Invalid credentials", "INVALID_CREDENTIALS")
			return
		}

		// Update last login
		s.pgDB.ExecContext(ctx, "UPDATE users SET last_login = $1 WHERE id = $2", time.Now().UTC(), userID)
	}

	// Generate token
	token, _ := generateToken()

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

	// Generate token
	token, _ := generateToken()

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

	// In production, invalidate the token
	s.writeJSON(w, http.StatusOK, map[string]string{"message": "Signed out successfully"})
}

// handleRefreshToken handles POST /api/auth/refresh
func (s *Server) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	token, _ := generateToken()
	s.writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

// generateToken creates a random auth token.
func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
