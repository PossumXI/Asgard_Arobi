// Package notifications provides access key generation and notification services.
//
// Copyright 2026 Arobi. All Rights Reserved.
package notifications

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/asgard/pandora/internal/notifications/email"
)

// KeyType defines the type of access key
type KeyType string

const (
	KeyTypeFounder    KeyType = "FOUNDER_MASTER"
	KeyTypeGovernment KeyType = "GOVERNMENT_ACCESS"
	KeyTypeMilitary   KeyType = "MILITARY_ACCESS"
	KeyTypeAdmin      KeyType = "ADMIN_ACCESS"
	KeyTypeDeveloper  KeyType = "DEVELOPER_ACCESS"
	KeyTypeAPI        KeyType = "API_ACCESS"
)

// AccessKey represents a generated access key
type AccessKey struct {
	ID          string     `json:"id"`
	KeyType     KeyType    `json:"key_type"`
	KeyHash     string     `json:"key_hash"` // SHA-256 hash of the actual key
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	IssuedTo    string     `json:"issued_to"` // Protected person ID or email
	IssuedBy    string     `json:"issued_by"`
	UsedAt      *time.Time `json:"used_at,omitempty"`
	Revoked     bool       `json:"revoked"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	RevokedBy   string     `json:"revoked_by,omitempty"`
	Description string     `json:"description,omitempty"`
	Permissions []string   `json:"permissions"`
}

// AccessKeyManager manages access key generation and validation
type AccessKeyManager struct {
	mu          sync.RWMutex
	keys        map[string]*AccessKey // keyHash -> key
	emailClient *email.ResendClient
	config      AccessKeyConfig
}

// AccessKeyConfig holds configuration
type AccessKeyConfig struct {
	DefaultExpiration  time.Duration
	FounderEmail       string
	NotifyOnGeneration bool
	NotifyOnUse        bool
	NotifyOnRevoke     bool
}

// DefaultAccessKeyConfig returns default configuration
func DefaultAccessKeyConfig() AccessKeyConfig {
	return AccessKeyConfig{
		DefaultExpiration:  24 * time.Hour,
		FounderEmail:       "Gaetano@aura-genesis.org",
		NotifyOnGeneration: true,
		NotifyOnUse:        true,
		NotifyOnRevoke:     true,
	}
}

// NewAccessKeyManager creates a new access key manager
func NewAccessKeyManager(emailClient *email.ResendClient, cfg AccessKeyConfig) *AccessKeyManager {
	return &AccessKeyManager{
		keys:        make(map[string]*AccessKey),
		emailClient: emailClient,
		config:      cfg,
	}
}

// GenerateAccessKey generates a new access key and optionally sends it via email
func (m *AccessKeyManager) GenerateAccessKey(ctx context.Context, req GenerateKeyRequest) (*AccessKey, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate secure random key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Format: ASGARD-{TYPE}-{RANDOM}
	rawKey := fmt.Sprintf("ASGARD-%s-%s", req.KeyType, base64.URLEncoding.EncodeToString(keyBytes)[:32])

	// Hash the key for storage
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	// Generate unique ID
	idBytes := make([]byte, 8)
	rand.Read(idBytes)
	keyID := fmt.Sprintf("key_%s", hex.EncodeToString(idBytes))

	// Determine expiration
	expiration := m.config.DefaultExpiration
	if req.Expiration > 0 {
		expiration = req.Expiration
	}

	// Create access key record
	key := &AccessKey{
		ID:          keyID,
		KeyType:     req.KeyType,
		KeyHash:     keyHash,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(expiration),
		IssuedTo:    req.IssuedTo,
		IssuedBy:    req.IssuedBy,
		Description: req.Description,
		Permissions: req.Permissions,
	}

	// Store the key
	m.keys[keyHash] = key

	log.Printf("[AccessKey] Generated %s key for %s (expires: %s)", req.KeyType, req.IssuedTo, key.ExpiresAt)

	// Send email notification
	if m.config.NotifyOnGeneration && m.emailClient != nil && req.SendEmail {
		emailTo := req.EmailTo
		if emailTo == "" {
			// Default to founder email for founder keys
			if req.KeyType == KeyTypeFounder {
				emailTo = m.config.FounderEmail
			}
		}

		if emailTo != "" {
			if err := m.emailClient.SendAccessKeyEmail(ctx, emailTo, rawKey, string(req.KeyType)); err != nil {
				log.Printf("[AccessKey] Warning: Failed to send email notification: %v", err)
			} else {
				log.Printf("[AccessKey] Access key emailed to %s", emailTo)
			}
		}
	}

	return key, rawKey, nil
}

// ValidateKey validates an access key and marks it as used
func (m *AccessKeyManager) ValidateKey(ctx context.Context, rawKey string) (*AccessKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Hash the provided key
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	// Look up the key
	key, exists := m.keys[keyHash]
	if !exists {
		return nil, errors.New("invalid access key")
	}

	// Check if revoked
	if key.Revoked {
		return nil, errors.New("access key has been revoked")
	}

	// Check expiration
	if time.Now().After(key.ExpiresAt) {
		return nil, errors.New("access key has expired")
	}

	// Check if already used (one-time keys)
	if key.UsedAt != nil {
		return nil, errors.New("access key has already been used")
	}

	// Mark as used
	now := time.Now()
	key.UsedAt = &now

	log.Printf("[AccessKey] Key %s validated for %s", key.ID, key.IssuedTo)

	// Notify if configured
	if m.config.NotifyOnUse && m.emailClient != nil && key.KeyType == KeyTypeFounder {
		go func() {
			m.emailClient.SendSecurityAlertEmail(
				context.Background(),
				m.config.FounderEmail,
				"Access Key Used",
				fmt.Sprintf("Your %s access key was used at %s", key.KeyType, now.Format(time.RFC3339)),
				"low",
			)
		}()
	}

	return key, nil
}

// RevokeKey revokes an access key
func (m *AccessKeyManager) RevokeKey(ctx context.Context, keyID, revokedBy, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Find key by ID
	var targetKey *AccessKey
	for _, key := range m.keys {
		if key.ID == keyID {
			targetKey = key
			break
		}
	}

	if targetKey == nil {
		return errors.New("key not found")
	}

	if targetKey.Revoked {
		return errors.New("key already revoked")
	}

	// Revoke the key
	now := time.Now()
	targetKey.Revoked = true
	targetKey.RevokedAt = &now
	targetKey.RevokedBy = revokedBy

	log.Printf("[AccessKey] Key %s revoked by %s", keyID, revokedBy)

	// Notify founder if their key was revoked
	if m.config.NotifyOnRevoke && m.emailClient != nil && targetKey.KeyType == KeyTypeFounder {
		go func() {
			m.emailClient.SendSecurityAlertEmail(
				context.Background(),
				m.config.FounderEmail,
				"Access Key Revoked",
				fmt.Sprintf("Your %s access key was revoked. Reason: %s", targetKey.KeyType, reason),
				"high",
			)
		}()
	}

	return nil
}

// GenerateFounderKey is a convenience method to generate a founder access key
func (m *AccessKeyManager) GenerateFounderKey(ctx context.Context, issuedBy string) (*AccessKey, string, error) {
	return m.GenerateAccessKey(ctx, GenerateKeyRequest{
		KeyType:     KeyTypeFounder,
		IssuedTo:    "ASGARD-001", // Founder ID
		IssuedBy:    issuedBy,
		Description: "Founder master access key",
		SendEmail:   true,
		EmailTo:     m.config.FounderEmail,
		Permissions: []string{
			"vault.admin",
			"services.all",
			"founder.protection",
			"systems.override",
		},
	})
}

// ListKeys returns all keys (without sensitive data)
func (m *AccessKeyManager) ListKeys() []*AccessKey {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*AccessKey, 0, len(m.keys))
	for _, key := range m.keys {
		// Return copy
		keyCopy := *key
		result = append(result, &keyCopy)
	}
	return result
}

// GetKey returns a key by ID
func (m *AccessKeyManager) GetKey(keyID string) *AccessKey {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, key := range m.keys {
		if key.ID == keyID {
			keyCopy := *key
			return &keyCopy
		}
	}
	return nil
}

// CleanupExpired removes expired keys from memory
func (m *AccessKeyManager) CleanupExpired() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	count := 0
	for hash, key := range m.keys {
		if now.After(key.ExpiresAt) && key.UsedAt != nil {
			delete(m.keys, hash)
			count++
		}
	}

	if count > 0 {
		log.Printf("[AccessKey] Cleaned up %d expired keys", count)
	}
	return count
}

// GenerateKeyRequest contains parameters for key generation
type GenerateKeyRequest struct {
	KeyType     KeyType
	IssuedTo    string
	IssuedBy    string
	Description string
	Expiration  time.Duration
	Permissions []string
	SendEmail   bool
	EmailTo     string
}

// VerificationCodeManager manages email verification codes
type VerificationCodeManager struct {
	mu          sync.RWMutex
	codes       map[string]*VerificationCode // email -> code
	emailClient *email.ResendClient
}

// VerificationCode represents an email verification code
type VerificationCode struct {
	Email     string
	Code      string
	CreatedAt time.Time
	ExpiresAt time.Time
	Verified  bool
	Attempts  int
}

// NewVerificationCodeManager creates a new verification code manager
func NewVerificationCodeManager(emailClient *email.ResendClient) *VerificationCodeManager {
	return &VerificationCodeManager{
		codes:       make(map[string]*VerificationCode),
		emailClient: emailClient,
	}
}

// GenerateVerificationCode generates and sends a verification code
func (m *VerificationCodeManager) GenerateVerificationCode(ctx context.Context, email string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate 6-digit code
	codeBytes := make([]byte, 3)
	rand.Read(codeBytes)
	code := fmt.Sprintf("%06d", int(codeBytes[0])<<16|int(codeBytes[1])<<8|int(codeBytes[2])%1000000)

	vc := &VerificationCode{
		Email:     email,
		Code:      code,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}

	m.codes[email] = vc

	// Send email
	if m.emailClient != nil {
		if err := m.emailClient.SendVerificationEmail(ctx, email, code); err != nil {
			return fmt.Errorf("failed to send verification email: %w", err)
		}
	}

	log.Printf("[Verification] Code sent to %s", email)
	return nil
}

// VerifyCode verifies an email verification code
func (m *VerificationCodeManager) VerifyCode(email, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	vc, exists := m.codes[email]
	if !exists {
		return errors.New("no verification code found")
	}

	vc.Attempts++

	if vc.Attempts > 5 {
		delete(m.codes, email)
		return errors.New("too many attempts")
	}

	if time.Now().After(vc.ExpiresAt) {
		delete(m.codes, email)
		return errors.New("verification code expired")
	}

	if vc.Code != code {
		return errors.New("invalid verification code")
	}

	vc.Verified = true
	log.Printf("[Verification] Email %s verified", email)
	return nil
}

// IsVerified checks if an email has been verified
func (m *VerificationCodeManager) IsVerified(email string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	vc, exists := m.codes[email]
	return exists && vc.Verified
}
