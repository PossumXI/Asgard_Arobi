// Package services provides business logic services for the API.
package services

import (
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func setupTestAuthService(t *testing.T) *AuthService {
	t.Helper()
	// Set required environment for test
	os.Setenv("ASGARD_ENV", "development")
	return NewAuthService(nil, nil, nil, nil)
}

func TestHashPassword(t *testing.T) {
	authService := setupTestAuthService(t)

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "simple password",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "complex password",
			password: "C0mpl3x!P@ssw0rd#2024",
			wantErr:  false,
		},
		{
			name:     "unicode password",
			password: "密码测试😀🔐",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  false, // Empty password should still hash (validation done elsewhere)
		},
		{
			name:     "long password",
			password: strings.Repeat("a", 1000),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := authService.hashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("hashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Verify hash format (Argon2id format)
				if !strings.HasPrefix(hash, "$argon2id$") {
					t.Errorf("hashPassword() hash should start with $argon2id$, got %s", hash[:20])
				}
				// Verify hash is not empty
				if hash == "" {
					t.Error("hashPassword() returned empty hash")
				}
			}
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	authService := setupTestAuthService(t)

	tests := []struct {
		name     string
		password string
		wantOk   bool
	}{
		{
			name:     "correct password",
			password: "correctPassword123",
			wantOk:   true,
		},
		{
			name:     "unicode password",
			password: "密码测试",
			wantOk:   true,
		},
		{
			name:     "empty password",
			password: "",
			wantOk:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First hash the password
			hash, err := authService.hashPassword(tt.password)
			if err != nil {
				t.Fatalf("hashPassword() error = %v", err)
			}

			// Verify the same password
			if got := authService.verifyPassword(hash, tt.password); got != tt.wantOk {
				t.Errorf("verifyPassword() = %v, want %v", got, tt.wantOk)
			}

			// Verify wrong password fails
			if got := authService.verifyPassword(hash, "wrongPassword"); got != false {
				t.Errorf("verifyPassword() with wrong password = %v, want false", got)
			}
		})
	}
}

func TestVerifyPassword_InvalidHash(t *testing.T) {
	authService := setupTestAuthService(t)

	tests := []struct {
		name string
		hash string
	}{
		{
			name: "empty hash",
			hash: "",
		},
		{
			name: "invalid format",
			hash: "not-a-valid-hash",
		},
		{
			name: "wrong algorithm prefix",
			hash: "$bcrypt$...",
		},
		{
			name: "malformed argon2id",
			hash: "$argon2id$invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := authService.verifyPassword(tt.hash, "anypassword"); got != false {
				t.Errorf("verifyPassword() with invalid hash = %v, want false", got)
			}
		})
	}
}

func TestGenerateToken(t *testing.T) {
	authService := setupTestAuthService(t)

	tests := []struct {
		name             string
		user             *db.User
		wantRole         string
		wantIsGovernment bool
	}{
		{
			name: "free tier user",
			user: &db.User{
				ID:               uuid.New(),
				Email:            "free@test.com",
				SubscriptionTier: "free",
				IsGovernment:     false,
			},
			wantRole:         "civilian",
			wantIsGovernment: false,
		},
		{
			name: "observer tier user",
			user: &db.User{
				ID:               uuid.New(),
				Email:            "observer@test.com",
				SubscriptionTier: "observer",
				IsGovernment:     false,
			},
			wantRole:         "civilian",
			wantIsGovernment: false,
		},
		{
			name: "supporter tier user",
			user: &db.User{
				ID:               uuid.New(),
				Email:            "supporter@test.com",
				SubscriptionTier: "supporter",
				IsGovernment:     false,
			},
			wantRole:         "military",
			wantIsGovernment: false,
		},
		{
			name: "commander tier user",
			user: &db.User{
				ID:               uuid.New(),
				Email:            "commander@test.com",
				SubscriptionTier: "commander",
				IsGovernment:     false,
			},
			wantRole:         "interstellar",
			wantIsGovernment: false,
		},
		{
			name: "government user",
			user: &db.User{
				ID:               uuid.New(),
				Email:            "gov@test.gov",
				SubscriptionTier: "free",
				IsGovernment:     true,
			},
			wantRole:         "government",
			wantIsGovernment: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenStr, tokenID, err := authService.generateToken(tt.user)
			if err != nil {
				t.Fatalf("generateToken() error = %v", err)
			}

			if tokenStr == "" {
				t.Error("generateToken() returned empty token string")
			}
			if tokenID == "" {
				t.Error("generateToken() returned empty token ID")
			}

			// Parse and validate the token
			token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
				return authService.jwtSecret, nil
			})
			if err != nil {
				t.Fatalf("failed to parse token: %v", err)
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok || !token.Valid {
				t.Fatal("invalid token claims")
			}

			// Verify claims
			if claims["user_id"] != tt.user.ID.String() {
				t.Errorf("user_id = %v, want %v", claims["user_id"], tt.user.ID.String())
			}
			if claims["role"] != tt.wantRole {
				t.Errorf("role = %v, want %v", claims["role"], tt.wantRole)
			}
			if claims["subscription_tier"] != tt.user.SubscriptionTier {
				t.Errorf("subscription_tier = %v, want %v", claims["subscription_tier"], tt.user.SubscriptionTier)
			}
			if claims["is_government"] != tt.wantIsGovernment {
				t.Errorf("is_government = %v, want %v", claims["is_government"], tt.wantIsGovernment)
			}
			if claims["jti"] != tokenID {
				t.Errorf("jti = %v, want %v", claims["jti"], tokenID)
			}
		})
	}
}

func TestValidateToken_Valid(t *testing.T) {
	authService := setupTestAuthService(t)

	user := &db.User{
		ID:               uuid.New(),
		Email:            "test@test.com",
		SubscriptionTier: "supporter",
		IsGovernment:     false,
	}

	tokenStr, tokenID, err := authService.generateToken(user)
	if err != nil {
		t.Fatalf("generateToken() error = %v", err)
	}

	claims, err := authService.ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if claims.UserID != user.ID.String() {
		t.Errorf("UserID = %v, want %v", claims.UserID, user.ID.String())
	}
	if claims.TokenID != tokenID {
		t.Errorf("TokenID = %v, want %v", claims.TokenID, tokenID)
	}
	if claims.Role != "military" {
		t.Errorf("Role = %v, want military", claims.Role)
	}
	if claims.SubscriptionTier != "supporter" {
		t.Errorf("SubscriptionTier = %v, want supporter", claims.SubscriptionTier)
	}
	if claims.IsGovernment != false {
		t.Errorf("IsGovernment = %v, want false", claims.IsGovernment)
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	authService := setupTestAuthService(t)

	tests := []struct {
		name      string
		token     string
		wantError error
	}{
		{
			name:      "empty token",
			token:     "",
			wantError: ErrInvalidToken,
		},
		{
			name:      "malformed token",
			token:     "not.a.valid.jwt",
			wantError: ErrInvalidToken,
		},
		{
			name:      "invalid signature",
			token:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMTIzIn0.invalidsignature",
			wantError: ErrInvalidToken,
		},
		{
			name:      "missing parts",
			token:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			wantError: ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := authService.ValidateToken(tt.token)
			if err == nil {
				t.Error("ValidateToken() expected error, got nil")
			}
			if err != tt.wantError {
				t.Errorf("ValidateToken() error = %v, want %v", err, tt.wantError)
			}
		})
	}
}

func TestValidateToken_Expired(t *testing.T) {
	authService := setupTestAuthService(t)

	// Create a token that's already expired
	user := &db.User{
		ID:               uuid.New(),
		Email:            "expired@test.com",
		SubscriptionTier: "free",
		IsGovernment:     false,
	}

	// Create token with past expiration
	tokenID := uuid.New().String()
	claims := jwt.MapClaims{
		"user_id":           user.ID.String(),
		"jti":               tokenID,
		"role":              roleForUser(user),
		"subscription_tier": user.SubscriptionTier,
		"is_government":     user.IsGovernment,
		"exp":               time.Now().Add(-1 * time.Hour).Unix(), // Expired 1 hour ago
		"iat":               time.Now().Add(-2 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(authService.jwtSecret)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	_, err = authService.ValidateToken(tokenStr)
	if err == nil {
		t.Error("ValidateToken() expected error for expired token, got nil")
	}
	if err != ErrInvalidToken {
		t.Errorf("ValidateToken() error = %v, want %v", err, ErrInvalidToken)
	}
}

func TestValidateToken_MissingUserID(t *testing.T) {
	authService := setupTestAuthService(t)

	// Create a token without user_id
	claims := jwt.MapClaims{
		"jti":               uuid.New().String(),
		"role":              "civilian",
		"subscription_tier": "free",
		"exp":               time.Now().Add(time.Hour).Unix(),
		"iat":               time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(authService.jwtSecret)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	_, err = authService.ValidateToken(tokenStr)
	if err != ErrInvalidToken {
		t.Errorf("ValidateToken() error = %v, want %v", err, ErrInvalidToken)
	}
}

func TestRoleForUser(t *testing.T) {
	tests := []struct {
		name         string
		user         *db.User
		expectedRole string
	}{
		{
			name: "government user always gets government role",
			user: &db.User{
				SubscriptionTier: "free",
				IsGovernment:     true,
			},
			expectedRole: "government",
		},
		{
			name: "government commander gets government role (not interstellar)",
			user: &db.User{
				SubscriptionTier: "commander",
				IsGovernment:     true,
			},
			expectedRole: "government",
		},
		{
			name: "free tier gets civilian",
			user: &db.User{
				SubscriptionTier: "free",
				IsGovernment:     false,
			},
			expectedRole: "civilian",
		},
		{
			name: "observer tier gets civilian",
			user: &db.User{
				SubscriptionTier: "observer",
				IsGovernment:     false,
			},
			expectedRole: "civilian",
		},
		{
			name: "supporter tier gets military",
			user: &db.User{
				SubscriptionTier: "supporter",
				IsGovernment:     false,
			},
			expectedRole: "military",
		},
		{
			name: "commander tier gets interstellar",
			user: &db.User{
				SubscriptionTier: "commander",
				IsGovernment:     false,
			},
			expectedRole: "interstellar",
		},
		{
			name: "unknown tier gets civilian",
			user: &db.User{
				SubscriptionTier: "unknown_tier",
				IsGovernment:     false,
			},
			expectedRole: "civilian",
		},
		{
			name: "empty tier gets civilian",
			user: &db.User{
				SubscriptionTier: "",
				IsGovernment:     false,
			},
			expectedRole: "civilian",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			role := roleForUser(tt.user)
			if role != tt.expectedRole {
				t.Errorf("roleForUser() = %v, want %v", role, tt.expectedRole)
			}
		})
	}
}

func TestHashPasswordDeterminism(t *testing.T) {
	authService := setupTestAuthService(t)

	password := "testPassword123"

	hash1, err := authService.hashPassword(password)
	if err != nil {
		t.Fatalf("hashPassword() error = %v", err)
	}

	hash2, err := authService.hashPassword(password)
	if err != nil {
		t.Fatalf("hashPassword() error = %v", err)
	}

	// Hashes should be different (due to random salt)
	if hash1 == hash2 {
		t.Error("hashPassword() should produce different hashes for same password (random salt)")
	}

	// But both should verify correctly
	if !authService.verifyPassword(hash1, password) {
		t.Error("verifyPassword() failed for hash1")
	}
	if !authService.verifyPassword(hash2, password) {
		t.Error("verifyPassword() failed for hash2")
	}
}

func TestNewAuthService_DevMode(t *testing.T) {
	os.Setenv("ASGARD_ENV", "development")
	os.Unsetenv("ASGARD_JWT_SECRET")

	service := NewAuthService(nil, nil, nil, nil)
	if service == nil {
		t.Fatal("NewAuthService() returned nil in development mode")
	}
}

func TestIsDevelopmentMode(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		want     bool
	}{
		{
			name:     "development mode",
			envValue: "development",
			want:     true,
		},
		{
			name:     "production mode",
			envValue: "production",
			want:     false,
		},
		{
			name:     "empty value",
			envValue: "",
			want:     false,
		},
		{
			name:     "staging mode",
			envValue: "staging",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue == "" {
				os.Unsetenv("ASGARD_ENV")
			} else {
				os.Setenv("ASGARD_ENV", tt.envValue)
			}

			if got := isDevelopmentMode(); got != tt.want {
				t.Errorf("isDevelopmentMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateToken_WithFullName(t *testing.T) {
	authService := setupTestAuthService(t)

	user := &db.User{
		ID:               uuid.New(),
		Email:            "test@test.com",
		FullName:         sql.NullString{String: "John Doe", Valid: true},
		SubscriptionTier: "observer",
		IsGovernment:     false,
	}

	tokenStr, _, err := authService.generateToken(user)
	if err != nil {
		t.Fatalf("generateToken() error = %v", err)
	}

	claims, err := authService.ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("ValidateToken() error = %v", err)
	}

	if claims.UserID != user.ID.String() {
		t.Errorf("UserID = %v, want %v", claims.UserID, user.ID.String())
	}
}
