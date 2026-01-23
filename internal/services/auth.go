// Package services provides business logic services for the API.
package services

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/asgard/pandora/internal/repositories"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailExists        = errors.New("email already exists")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
)

// AuthService handles authentication and authorization.
type AuthService struct {
	userRepo    *repositories.UserRepository
	jwtSecret   []byte
	tokenExpiry time.Duration
}

// NewAuthService creates a new authentication service.
func NewAuthService(userRepo *repositories.UserRepository) *AuthService {
	// In production, load from environment variable
	secret := []byte("asgard_jwt_secret_change_in_production_2026")
	if len(secret) < 32 {
		panic("JWT secret must be at least 32 bytes")
	}

	return &AuthService{
		userRepo:    userRepo,
		jwtSecret:   secret,
		tokenExpiry: 24 * time.Hour,
	}
}

// SignIn authenticates a user and returns a JWT token.
func (s *AuthService) SignIn(email, password string) (*db.User, string, error) {
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}

	if !s.verifyPassword(user.PasswordHash, password) {
		return nil, "", ErrInvalidCredentials
	}

	// Update last login
	now := time.Now()
	user.LastLogin = &now
	if err := s.userRepo.Update(user); err != nil {
		return nil, "", fmt.Errorf("failed to update last login: %w", err)
	}

	token, err := s.generateToken(user.ID.String())
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return user, token, nil
}

// SignUp creates a new user account.
func (s *AuthService) SignUp(email, password, fullName string, isGovernment bool) (*db.User, string, error) {
	// Check if user exists
	_, err := s.userRepo.GetByEmail(email)
	if err == nil {
		return nil, "", ErrEmailExists
	}

	// Hash password
	passwordHash, err := s.hashPassword(password)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &db.User{
		Email:            email,
		PasswordHash:     passwordHash,
		FullName:         &fullName,
		SubscriptionTier: "free",
		IsGovernment:     isGovernment,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, "", fmt.Errorf("failed to create user: %w", err)
	}

	token, err := s.generateToken(user.ID.String())
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return user, token, nil
}

// ValidateToken validates a JWT token and returns the user ID.
func (s *AuthService) ValidateToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return "", ErrInvalidToken
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID, ok := claims["user_id"].(string)
		if !ok {
			return "", ErrInvalidToken
		}
		return userID, nil
	}

	return "", ErrInvalidToken
}

// RefreshToken generates a new token for an existing user.
func (s *AuthService) RefreshToken(userID string) (string, error) {
	// Verify user exists
	_, err := s.userRepo.GetByID(userID)
	if err != nil {
		return "", ErrUserNotFound
	}

	return s.generateToken(userID)
}

// hashPassword hashes a password using Argon2id.
func (s *AuthService) hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, 64*1024, 1, 4, b64Salt, b64Hash), nil
}

// verifyPassword verifies a password against a hash.
func (s *AuthService) verifyPassword(hash, password string) bool {
	// Parse Argon2id hash
	var version, memory, time, parallelism uint32
	var salt, hashBytes []byte

	_, err := fmt.Sscanf(hash, "$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		&version, &memory, &time, &parallelism, &salt, &hashBytes)
	if err != nil {
		// Fallback: simple comparison for development
		return subtle.ConstantTimeCompare([]byte(hash), []byte(password)) == 1
	}

	decodedSalt, err := base64.RawStdEncoding.DecodeString(string(salt))
	if err != nil {
		return false
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(string(hashBytes))
	if err != nil {
		return false
	}

	computedHash := argon2.IDKey([]byte(password), decodedSalt, time, memory, parallelism, uint32(len(decodedHash)))

	return subtle.ConstantTimeCompare(decodedHash, computedHash) == 1
}

// generateToken generates a JWT token for a user.
func (s *AuthService) generateToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":      time.Now().Add(s.tokenExpiry).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// RequestPasswordReset initiates a password reset flow.
func (s *AuthService) RequestPasswordReset(email string) error {
	_, err := s.userRepo.GetByEmail(email)
	if err != nil {
		// Don't reveal if user exists
		return nil
	}

	// In production, send email with reset token
	// For now, just return success
	return nil
}

// ResetPassword resets a user's password using a reset token.
func (s *AuthService) ResetPassword(token, newPassword string) error {
	// In production, validate reset token from database
	// For now, this is a placeholder
	return errors.New("password reset not fully implemented")
}

// VerifyEmail verifies a user's email address.
func (s *AuthService) VerifyEmail(token string) error {
	// In production, validate verification token
	// For now, this is a placeholder
	return errors.New("email verification not fully implemented")
}

// StartFido2Registration initiates FIDO2/WebAuthn registration.
func (s *AuthService) StartFido2Registration(userID string) (map[string]interface{}, error) {
	// Placeholder for FIDO2 implementation
	return nil, errors.New("FIDO2 registration not fully implemented")
}

// CompleteFido2Registration completes FIDO2/WebAuthn registration.
func (s *AuthService) CompleteFido2Registration(userID string, credential map[string]interface{}) error {
	// Placeholder for FIDO2 implementation
	return errors.New("FIDO2 registration completion not fully implemented")
}

// StartFido2Auth initiates FIDO2/WebAuthn authentication.
func (s *AuthService) StartFido2Auth(email string) (map[string]interface{}, error) {
	// Placeholder for FIDO2 implementation
	return nil, errors.New("FIDO2 authentication not fully implemented")
}

// CompleteFido2Auth completes FIDO2/WebAuthn authentication.
func (s *AuthService) CompleteFido2Auth(email string, credential map[string]interface{}) (*db.User, string, error) {
	// Placeholder for FIDO2 implementation
	return nil, "", errors.New("FIDO2 authentication completion not fully implemented")
}
