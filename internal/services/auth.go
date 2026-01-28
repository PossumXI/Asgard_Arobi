// Package services provides business logic services for the API.
package services

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/asgard/pandora/internal/repositories"
	"github.com/go-webauthn/webauthn/protocol"
	webauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailExists        = errors.New("email already exists")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	ErrEmailNotVerified   = errors.New("email not verified")
	ErrFido2Required      = errors.New("fido2 required")
)

// isDevelopmentMode returns true if ASGARD_ENV is set to "development".
// In development mode, certain security fallbacks are allowed.
func isDevelopmentMode() bool {
	return os.Getenv("ASGARD_ENV") == "development"
}

// AuthService handles authentication and authorization.
type AuthService struct {
	userRepo         *repositories.UserRepository
	tokenRepo        *repositories.AuthTokenRepository
	webauthnRepo     *repositories.WebAuthnRepository
	emailTokenRepo   *repositories.EmailTokenRepository
	emailService     *EmailService
	jwtSecret        []byte
	tokenExpiry      time.Duration
	refreshExpiry    time.Duration
	webAuthn         *webauthn.WebAuthn
}

// TokenClaims represents validated JWT claims.
type TokenClaims struct {
	UserID           string
	TokenID          string
	Role             string
	SubscriptionTier string
	IsGovernment     bool
}

// NewAuthService creates a new authentication service.
// In production (ASGARD_ENV != "development"), ASGARD_JWT_SECRET must be set and >= 32 bytes.
func NewAuthService(
	userRepo *repositories.UserRepository,
	tokenRepo *repositories.AuthTokenRepository,
	webauthnRepo *repositories.WebAuthnRepository,
	emailTokenRepo *repositories.EmailTokenRepository,
) *AuthService {
	secret := []byte(os.Getenv("ASGARD_JWT_SECRET"))
	if len(secret) < 32 {
		if isDevelopmentMode() {
			// Allow fallback secret only in development mode
			secret = []byte("asgard_dev_jwt_secret_not_for_production!")
		} else {
			panic("FATAL: ASGARD_JWT_SECRET environment variable must be set and at least 32 bytes in production")
		}
	}

	service := &AuthService{
		userRepo:       userRepo,
		tokenRepo:      tokenRepo,
		webauthnRepo:   webauthnRepo,
		emailTokenRepo: emailTokenRepo,
		emailService:   NewEmailService(),
		jwtSecret:      secret,
		tokenExpiry:    24 * time.Hour,
		refreshExpiry:  30 * 24 * time.Hour,
	}
	service.initWebAuthn()
	return service
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

	if user.IsGovernment {
		if !user.EmailVerified {
			return nil, "", ErrEmailNotVerified
		}
		if s.webauthnRepo == nil {
			return nil, "", ErrFido2Required
		}
		creds, err := s.webauthnRepo.GetCredentialsByUserID(user.ID.String())
		if err != nil || len(creds) == 0 {
			return nil, "", ErrFido2Required
		}
	}

	// Update last login
	now := time.Now()
	user.LastLogin = sql.NullTime{Time: now, Valid: true}
	if err := s.userRepo.Update(user); err != nil {
		return nil, "", fmt.Errorf("failed to update last login: %w", err)
	}

	token, _, err := s.generateToken(user)
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
		FullName:         sql.NullString{String: fullName, Valid: fullName != ""},
		SubscriptionTier: "free",
		IsGovernment:     isGovernment,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, "", fmt.Errorf("failed to create user: %w", err)
	}

	// Generate email verification token
	verifyToken := uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour)
	if err := s.emailTokenRepo.StoreVerificationToken(user.ID, verifyToken, expiresAt); err == nil {
		// Send verification email (non-blocking)
		go s.emailService.SendEmailVerification(email, verifyToken)
	}

	token, _, err := s.generateToken(user)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return user, token, nil
}

// ValidateToken validates a JWT token and returns claims.
func (s *AuthService) ValidateToken(tokenString string) (TokenClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return TokenClaims{}, ErrInvalidToken
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		userID, _ := claims["user_id"].(string)
		tokenID, _ := claims["jti"].(string)
		role, _ := claims["role"].(string)
		tier, _ := claims["subscription_tier"].(string)
		isGov, _ := claims["is_government"].(bool)

		if userID == "" {
			return TokenClaims{}, ErrInvalidToken
		}

		if s.tokenRepo != nil && tokenID != "" {
			revoked, err := s.tokenRepo.IsTokenRevoked(tokenID)
			if err != nil {
				return TokenClaims{}, ErrInvalidToken
			}
			if revoked {
				return TokenClaims{}, ErrTokenExpired
			}
		}

		return TokenClaims{
			UserID:           userID,
			TokenID:          tokenID,
			Role:             role,
			SubscriptionTier: tier,
			IsGovernment:     isGov,
		}, nil
	}

	return TokenClaims{}, ErrInvalidToken
}

// RefreshToken generates a new token for an existing user.
func (s *AuthService) RefreshToken(claims TokenClaims) (string, error) {
	user, err := s.userRepo.GetByID(claims.UserID)
	if err != nil {
		return "", ErrUserNotFound
	}

	if s.tokenRepo != nil && claims.TokenID != "" {
		if err := s.tokenRepo.RevokeToken(claims.TokenID, claims.UserID); err != nil {
			return "", fmt.Errorf("failed to revoke token: %w", err)
		}
	}

	token, _, err := s.generateToken(user)
	if err != nil {
		return "", err
	}
	return token, nil
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
// Only properly hashed Argon2id passwords are accepted in production.
func (s *AuthService) verifyPassword(hash, password string) bool {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return false
	}
	if parts[1] != "argon2id" {
		return false
	}
	if !strings.HasPrefix(parts[2], "v=") {
		return false
	}

	version, err := strconv.Atoi(strings.TrimPrefix(parts[2], "v="))
	if err != nil || version != argon2.Version {
		return false
	}

	var memory uint32
	var timeCost uint32
	var parallelism uint32
	params := strings.Split(parts[3], ",")
	for _, param := range params {
		kv := strings.SplitN(param, "=", 2)
		if len(kv) != 2 {
			return false
		}
		value, err := strconv.Atoi(kv[1])
		if err != nil {
			return false
		}
		switch kv[0] {
		case "m":
			memory = uint32(value)
		case "t":
			timeCost = uint32(value)
		case "p":
			parallelism = uint32(value)
		}
	}
	if memory == 0 || timeCost == 0 || parallelism == 0 {
		return false
	}

	decodedSalt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	computedHash := argon2.IDKey([]byte(password), decodedSalt, timeCost, memory, uint8(parallelism), uint32(len(decodedHash)))
	return subtle.ConstantTimeCompare(decodedHash, computedHash) == 1
}

// generateToken generates a JWT token for a user.
func (s *AuthService) generateToken(user *db.User) (string, string, error) {
	tokenID := uuid.New().String()
	claims := jwt.MapClaims{
		"user_id":           user.ID.String(),
		"jti":               tokenID,
		"role":              roleForUser(user),
		"subscription_tier": user.SubscriptionTier,
		"is_government":     user.IsGovernment,
		"exp":               time.Now().Add(s.tokenExpiry).Unix(),
		"iat":               time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", "", err
	}
	return signed, tokenID, nil
}

func roleForUser(user *db.User) string {
	if user.IsGovernment {
		return "government"
	}
	// Map subscription tiers to roles for access control
	// Tier hierarchy: free < observer < supporter < commander
	// Role hierarchy: civilian < military < interstellar
	switch user.SubscriptionTier {
	case "commander":
		// Commander tier gets interstellar access (highest non-government level)
		return "interstellar"
	case "supporter":
		// Supporter tier gets military access
		return "military"
	case "observer":
		// Observer tier gets civilian access
		return "civilian"
	case "free":
		// Free tier gets public/limited access
		return "civilian"
	default:
		return "civilian"
	}
}

func (s *AuthService) RevokeToken(tokenID, userID string) error {
	if s.tokenRepo == nil || tokenID == "" {
		return nil
	}
	return s.tokenRepo.RevokeToken(tokenID, userID)
}

// RequestPasswordReset initiates a password reset flow.
func (s *AuthService) RequestPasswordReset(email string) error {
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		// Don't reveal if user exists
		return nil
	}

	// Generate reset token
	token := uuid.New().String()
	expiresAt := time.Now().Add(1 * time.Hour)

	// Store token
	if err := s.emailTokenRepo.StorePasswordResetToken(user.ID, token, expiresAt); err != nil {
		return err
	}

	// Send email
	return s.emailService.SendPasswordResetEmail(email, token)
}

// ResetPassword resets a user's password using a reset token.
func (s *AuthService) ResetPassword(token, newPassword string) error {
	// Verify token
	userID, err := s.emailTokenRepo.VerifyPasswordResetToken(token)
	if err != nil {
		return err
	}

	// Get user
	user, err := s.userRepo.GetByID(userID.String())
	if err != nil {
		return ErrUserNotFound
	}

	// Hash new password
	hashedPassword, err := s.hashPassword(newPassword)
	if err != nil {
		return err
	}

	// Update password
	user.PasswordHash = hashedPassword
	return s.userRepo.Update(user)
}

// VerifyEmail verifies a user's email address.
func (s *AuthService) VerifyEmail(token string) error {
	// Verify token
	userID, err := s.emailTokenRepo.VerifyVerificationToken(token)
	if err != nil {
		return err
	}

	// Mark email as verified with timestamp
	return s.userRepo.SetEmailVerified(userID.String(), true)
}

// StartFido2Registration initiates FIDO2/WebAuthn registration.
func (s *AuthService) StartFido2Registration(userID string) (map[string]interface{}, error) {
	if s.webAuthn == nil || s.webauthnRepo == nil {
		return nil, errors.New("webauthn not configured")
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	creds, err := s.webauthnRepo.GetCredentialsByUserID(user.ID.String())
	if err != nil {
		return nil, err
	}

	webUser := newWebAuthnUser(user, creds)
	options, sessionData, err := s.webAuthn.BeginRegistration(
		webUser,
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementPreferred,
			UserVerification: protocol.VerificationPreferred,
		}),
	)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(5 * time.Minute)
	if err := s.webauthnRepo.StoreSession(user.ID.String(), "registration", *sessionData, expiresAt); err != nil {
		return nil, err
	}

	return optionsToMap(options), nil
}

// CompleteFido2Registration completes FIDO2/WebAuthn registration.
func (s *AuthService) CompleteFido2Registration(userID string, r *http.Request) error {
	if s.webAuthn == nil || s.webauthnRepo == nil {
		return errors.New("webauthn not configured")
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	creds, err := s.webauthnRepo.GetCredentialsByUserID(user.ID.String())
	if err != nil {
		return err
	}
	webUser := newWebAuthnUser(user, creds)

	sessionData, err := s.webauthnRepo.GetLatestSession(user.ID.String(), "registration")
	if err != nil {
		return err
	}

	credential, err := s.webAuthn.FinishRegistration(webUser, sessionData, r)
	if err != nil {
		return err
	}

	if err := s.webauthnRepo.UpsertCredential(user.ID.String(), credential); err != nil {
		return err
	}

	return nil
}

// StartFido2Auth initiates FIDO2/WebAuthn authentication.
func (s *AuthService) StartFido2Auth(email string) (map[string]interface{}, error) {
	if s.webAuthn == nil || s.webauthnRepo == nil {
		return nil, errors.New("webauthn not configured")
	}

	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return nil, ErrUserNotFound
	}

	creds, err := s.webauthnRepo.GetCredentialsByUserID(user.ID.String())
	if err != nil {
		return nil, err
	}

	webUser := newWebAuthnUser(user, creds)
	options, sessionData, err := s.webAuthn.BeginLogin(webUser)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(5 * time.Minute)
	if err := s.webauthnRepo.StoreSession(user.ID.String(), "authentication", *sessionData, expiresAt); err != nil {
		return nil, err
	}

	return optionsToMap(options), nil
}

// CompleteFido2Auth completes FIDO2/WebAuthn authentication.
func (s *AuthService) CompleteFido2Auth(email string, r *http.Request) (*db.User, string, error) {
	if s.webAuthn == nil || s.webauthnRepo == nil {
		return nil, "", errors.New("webauthn not configured")
	}

	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return nil, "", ErrUserNotFound
	}
	if user.IsGovernment && !user.EmailVerified {
		return nil, "", ErrEmailNotVerified
	}

	creds, err := s.webauthnRepo.GetCredentialsByUserID(user.ID.String())
	if err != nil {
		return nil, "", err
	}

	webUser := newWebAuthnUser(user, creds)
	sessionData, err := s.webauthnRepo.GetLatestSession(user.ID.String(), "authentication")
	if err != nil {
		return nil, "", err
	}

	credential, err := s.webAuthn.FinishLogin(webUser, sessionData, r)
	if err != nil {
		return nil, "", err
	}

	if err := s.webauthnRepo.UpdateCredential(user.ID.String(), credential); err != nil {
		return nil, "", err
	}

	token, _, err := s.generateToken(user)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *AuthService) initWebAuthn() {
	rpOrigin := os.Getenv("ASGARD_WEBAUTHN_RP_ORIGIN")
	rpName := os.Getenv("ASGARD_WEBAUTHN_RP_NAME")
	rpID := os.Getenv("ASGARD_WEBAUTHN_RP_ID")

	// In production, require all WebAuthn environment variables
	if !isDevelopmentMode() {
		if rpOrigin == "" || rpName == "" || rpID == "" {
			// WebAuthn not configured - leave s.webAuthn nil
			// This will cause WebAuthn operations to return "webauthn not configured" error
			return
		}
	} else {
		// Development mode: use fallback values
		if rpOrigin == "" {
			rpOrigin = "http://localhost:5173"
		}
		if rpName == "" {
			rpName = "ASGARD Government Portal (Dev)"
		}
		if rpID == "" {
			rpID = "localhost"
		}
	}

	cfg := &webauthn.Config{
		RPDisplayName: rpName,
		RPID:          rpID,
		RPOrigins:     []string{rpOrigin},
	}

	webAuthn, err := webauthn.New(cfg)
	if err != nil {
		return
	}
	s.webAuthn = webAuthn
}

// getEnvOrDefaultShared returns the value of an environment variable or a fallback value.
// Used by other services in this package for non-security-critical configuration.
func getEnvOrDefaultShared(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// optionsToMap converts WebAuthn options to a map for JSON serialization.
func optionsToMap(options interface{}) map[string]interface{} {
	data, err := json.Marshal(options)
	if err != nil {
		return nil
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}

type webAuthnUser struct {
	id          uuid.UUID
	email       string
	displayName string
	credentials []webauthn.Credential
}

func newWebAuthnUser(user *db.User, creds []webauthn.Credential) *webAuthnUser {
	display := user.Email
	if user.FullName.Valid && user.FullName.String != "" {
		display = user.FullName.String
	}
	return &webAuthnUser{
		id:          user.ID,
		email:       user.Email,
		displayName: display,
		credentials: creds,
	}
}

func (u *webAuthnUser) WebAuthnID() []byte {
	return u.id[:]
}

func (u *webAuthnUser) WebAuthnName() string {
	return u.email
}

func (u *webAuthnUser) WebAuthnDisplayName() string {
	return u.displayName
}

func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

func (u *webAuthnUser) WebAuthnIcon() string {
	return ""
}

// UpdateRefreshToken allows rotation using refresh tokens.
func (s *AuthService) UpdateRefreshToken(userID string, refreshToken string, userAgent string, ip net.IP) error {
	if s.tokenRepo == nil {
		return nil
	}
	hash := hashToken(refreshToken)
	expiresAt := time.Now().Add(s.refreshExpiry)
	return s.tokenRepo.StoreRefreshToken(userID, hash, expiresAt, userAgent, ip)
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.RawStdEncoding.EncodeToString(hash[:])
}
