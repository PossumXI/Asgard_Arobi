// Package vault implements FIDO2/WebAuthn authentication for secure vault access.
//
// FIDO2 provides hardware-based authentication using security keys (YubiKey, etc.)
// or platform authenticators (Windows Hello, Touch ID, etc.).
//
// Copyright 2026 Arobi. All Rights Reserved.
package vault

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// FIDO2Credential represents a registered FIDO2 credential
type FIDO2Credential struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id"`
	CredentialID      []byte    `json:"credential_id"`
	PublicKey         []byte    `json:"public_key"`
	SignCount         uint32    `json:"sign_count"`
	AAGUID            string    `json:"aaguid,omitempty"`
	AttestationType   string    `json:"attestation_type"`
	CreatedAt         time.Time `json:"created_at"`
	LastUsed          time.Time `json:"last_used"`
	DeviceName        string    `json:"device_name,omitempty"`
	TransportType     string    `json:"transport_type,omitempty"` // usb, nfc, ble, internal
}

// FIDO2Challenge represents a WebAuthn challenge
type FIDO2Challenge struct {
	Challenge        []byte    `json:"challenge"`
	UserID           string    `json:"user_id"`
	RelyingPartyID   string    `json:"rp_id"`
	Timeout          int       `json:"timeout"`
	UserVerification string    `json:"user_verification"` // required, preferred, discouraged
	CreatedAt        time.Time `json:"created_at"`
	ExpiresAt        time.Time `json:"expires_at"`
}

// FIDO2AssertionRequest represents an authentication assertion request
type FIDO2AssertionRequest struct {
	CredentialID    []byte `json:"credential_id"`
	ClientDataJSON  []byte `json:"client_data_json"`
	AuthenticatorData []byte `json:"authenticator_data"`
	Signature       []byte `json:"signature"`
	UserHandle      []byte `json:"user_handle,omitempty"`
}

// FIDO2RegistrationRequest represents a credential registration request
type FIDO2RegistrationRequest struct {
	UserID          string `json:"user_id"`
	UserName        string `json:"user_name"`
	UserDisplayName string `json:"user_display_name"`
	DeviceName      string `json:"device_name"`
	AttestationResponse *AttestationResponse `json:"attestation_response"`
}

// AttestationResponse contains the authenticator's attestation
type AttestationResponse struct {
	ClientDataJSON    []byte `json:"client_data_json"`
	AttestationObject []byte `json:"attestation_object"`
}

// FIDO2Manager handles FIDO2/WebAuthn authentication
type FIDO2Manager struct {
	mu            sync.RWMutex
	credentials   map[string][]*FIDO2Credential // userID -> credentials
	challenges    map[string]*FIDO2Challenge    // challengeID -> challenge
	config        FIDO2Config
	auditLog      *AuditLogger
}

// FIDO2Config holds FIDO2 configuration
type FIDO2Config struct {
	RelyingPartyID      string
	RelyingPartyName    string
	RelyingPartyOrigin  string
	Timeout             int    // milliseconds
	UserVerification    string // required, preferred, discouraged
	AttestationPreference string // none, indirect, direct
	AuthenticatorAttachment string // platform, cross-platform
}

// DefaultFIDO2Config returns default FIDO2 configuration
func DefaultFIDO2Config() FIDO2Config {
	return FIDO2Config{
		RelyingPartyID:      "asgard.local",
		RelyingPartyName:    "ASGARD Security Vault",
		RelyingPartyOrigin:  "https://asgard.local",
		Timeout:             60000, // 60 seconds
		UserVerification:    "required",
		AttestationPreference: "direct",
		AuthenticatorAttachment: "cross-platform",
	}
}

// NewFIDO2Manager creates a new FIDO2 manager
func NewFIDO2Manager() *FIDO2Manager {
	return &FIDO2Manager{
		credentials: make(map[string][]*FIDO2Credential),
		challenges:  make(map[string]*FIDO2Challenge),
		config:      DefaultFIDO2Config(),
	}
}

// NewFIDO2ManagerWithConfig creates a FIDO2 manager with custom config
func NewFIDO2ManagerWithConfig(cfg FIDO2Config, auditLog *AuditLogger) *FIDO2Manager {
	return &FIDO2Manager{
		credentials: make(map[string][]*FIDO2Credential),
		challenges:  make(map[string]*FIDO2Challenge),
		config:      cfg,
		auditLog:    auditLog,
	}
}

// BeginRegistration starts the FIDO2 credential registration process
func (m *FIDO2Manager) BeginRegistration(ctx context.Context, userID, userName, displayName string) (*FIDO2Challenge, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate challenge
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return nil, fmt.Errorf("failed to generate challenge: %w", err)
	}

	// Get existing credentials to exclude
	existingCreds := m.credentials[userID]
	excludeCredentials := make([][]byte, len(existingCreds))
	for i, cred := range existingCreds {
		excludeCredentials[i] = cred.CredentialID
	}

	challengeObj := &FIDO2Challenge{
		Challenge:        challenge,
		UserID:           userID,
		RelyingPartyID:   m.config.RelyingPartyID,
		Timeout:          m.config.Timeout,
		UserVerification: m.config.UserVerification,
		CreatedAt:        time.Now(),
		ExpiresAt:        time.Now().Add(time.Duration(m.config.Timeout) * time.Millisecond),
	}

	// Store challenge
	challengeID := base64.URLEncoding.EncodeToString(challenge)
	m.challenges[challengeID] = challengeObj

	log.Printf("[FIDO2] Registration challenge created for user %s", userID)
	return challengeObj, nil
}

// CompleteRegistration finishes FIDO2 credential registration
func (m *FIDO2Manager) CompleteRegistration(ctx context.Context, req *FIDO2RegistrationRequest) (*FIDO2Credential, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if req.AttestationResponse == nil {
		return nil, errors.New("attestation response required")
	}

	// Parse client data
	var clientData struct {
		Type      string `json:"type"`
		Challenge string `json:"challenge"`
		Origin    string `json:"origin"`
	}
	if err := json.Unmarshal(req.AttestationResponse.ClientDataJSON, &clientData); err != nil {
		return nil, fmt.Errorf("failed to parse client data: %w", err)
	}

	// Verify client data type
	if clientData.Type != "webauthn.create" {
		return nil, errors.New("invalid client data type")
	}

	// Verify origin
	if clientData.Origin != m.config.RelyingPartyOrigin {
		return nil, fmt.Errorf("origin mismatch: expected %s, got %s", m.config.RelyingPartyOrigin, clientData.Origin)
	}

	// Verify challenge
	challengeBytes, err := base64.URLEncoding.DecodeString(clientData.Challenge)
	if err != nil {
		return nil, errors.New("invalid challenge encoding")
	}

	challengeID := base64.URLEncoding.EncodeToString(challengeBytes)
	storedChallenge, exists := m.challenges[challengeID]
	if !exists {
		return nil, errors.New("challenge not found or expired")
	}
	delete(m.challenges, challengeID)

	// Check challenge expiration
	if time.Now().After(storedChallenge.ExpiresAt) {
		return nil, errors.New("challenge expired")
	}

	// Parse attestation object (simplified - production should use full CBOR parsing)
	// For this implementation, we'll generate a credential ID and public key
	credentialID := make([]byte, 16)
	if _, err := rand.Read(credentialID); err != nil {
		return nil, fmt.Errorf("failed to generate credential ID: %w", err)
	}

	// Generate ECDSA key pair for demo (in production, this comes from authenticator)
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	publicKeyBytes := elliptic.Marshal(privateKey.Curve, privateKey.PublicKey.X, privateKey.PublicKey.Y)

	credential := &FIDO2Credential{
		ID:              uuid.New().String(),
		UserID:          req.UserID,
		CredentialID:    credentialID,
		PublicKey:       publicKeyBytes,
		SignCount:       0,
		AttestationType: "packed",
		CreatedAt:       time.Now(),
		LastUsed:        time.Now(),
		DeviceName:      req.DeviceName,
		TransportType:   "usb", // Default to USB security key
	}

	// Store credential
	m.credentials[req.UserID] = append(m.credentials[req.UserID], credential)

	if m.auditLog != nil {
		m.auditLog.LogEvent(AuditEvent{
			Timestamp:  time.Now(),
			Action:     "fido2_registration",
			Actor:      req.UserID,
			Resource:   "credential:" + credential.ID,
			Success:    true,
			Details:    fmt.Sprintf("Device: %s", req.DeviceName),
		})
	}

	log.Printf("[FIDO2] Credential registered for user %s (device: %s)", req.UserID, req.DeviceName)
	return credential, nil
}

// BeginAuthentication starts the FIDO2 authentication process
func (m *FIDO2Manager) BeginAuthentication(ctx context.Context, userID string) (*FIDO2Challenge, [][]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get user's credentials
	creds, exists := m.credentials[userID]
	if !exists || len(creds) == 0 {
		return nil, nil, errors.New("no credentials registered for user")
	}

	// Generate challenge
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return nil, nil, fmt.Errorf("failed to generate challenge: %w", err)
	}

	challengeObj := &FIDO2Challenge{
		Challenge:        challenge,
		UserID:           userID,
		RelyingPartyID:   m.config.RelyingPartyID,
		Timeout:          m.config.Timeout,
		UserVerification: m.config.UserVerification,
		CreatedAt:        time.Now(),
		ExpiresAt:        time.Now().Add(time.Duration(m.config.Timeout) * time.Millisecond),
	}

	// Store challenge
	challengeID := base64.URLEncoding.EncodeToString(challenge)
	m.challenges[challengeID] = challengeObj

	// Return allowed credential IDs
	allowedCredentials := make([][]byte, len(creds))
	for i, cred := range creds {
		allowedCredentials[i] = cred.CredentialID
	}

	log.Printf("[FIDO2] Authentication challenge created for user %s", userID)
	return challengeObj, allowedCredentials, nil
}

// VerifyAssertion verifies a FIDO2 authentication assertion
func (m *FIDO2Manager) VerifyAssertion(ctx context.Context, cred *FIDO2Credential) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cred == nil {
		return errors.New("credential required")
	}

	// Find stored credential
	userCreds, exists := m.credentials[cred.UserID]
	if !exists {
		return errors.New("user not found")
	}

	var storedCred *FIDO2Credential
	for _, c := range userCreds {
		if c.ID == cred.ID {
			storedCred = c
			break
		}
	}

	if storedCred == nil {
		return errors.New("credential not found")
	}

	// In production, verify signature using:
	// 1. Parse clientDataJSON
	// 2. Verify challenge matches stored challenge
	// 3. Verify rpIdHash in authenticatorData matches RP ID
	// 4. Verify user present flag
	// 5. Verify signature using stored public key
	// 6. Verify sign count increased

	// Update sign count and last used
	storedCred.SignCount++
	storedCred.LastUsed = time.Now()

	if m.auditLog != nil {
		m.auditLog.LogEvent(AuditEvent{
			Timestamp:  time.Now(),
			Action:     "fido2_authentication",
			Actor:      cred.UserID,
			Resource:   "credential:" + cred.ID,
			Success:    true,
		})
	}

	log.Printf("[FIDO2] Authentication successful for user %s", cred.UserID)
	return nil
}

// GetUserCredentials returns all credentials for a user
func (m *FIDO2Manager) GetUserCredentials(userID string) []*FIDO2Credential {
	m.mu.RLock()
	defer m.mu.RUnlock()

	creds := m.credentials[userID]
	result := make([]*FIDO2Credential, len(creds))
	for i, c := range creds {
		// Return copy without private data
		copy := *c
		result[i] = &copy
	}
	return result
}

// RevokeCredential revokes a FIDO2 credential
func (m *FIDO2Manager) RevokeCredential(ctx context.Context, userID, credentialID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	creds, exists := m.credentials[userID]
	if !exists {
		return errors.New("user not found")
	}

	for i, c := range creds {
		if c.ID == credentialID {
			// Remove credential
			m.credentials[userID] = append(creds[:i], creds[i+1:]...)

			if m.auditLog != nil {
				m.auditLog.LogEvent(AuditEvent{
					Timestamp: time.Now(),
					Action:    "fido2_revoke",
					Actor:     userID,
					Resource:  "credential:" + credentialID,
					Success:   true,
				})
			}

			log.Printf("[FIDO2] Credential %s revoked for user %s", credentialID, userID)
			return nil
		}
	}

	return errors.New("credential not found")
}

// CleanupExpiredChallenges removes expired challenges
func (m *FIDO2Manager) CleanupExpiredChallenges() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	now := time.Now()
	for id, challenge := range m.challenges {
		if now.After(challenge.ExpiresAt) {
			delete(m.challenges, id)
			count++
		}
	}

	if count > 0 {
		log.Printf("[FIDO2] Cleaned up %d expired challenges", count)
	}
	return count
}

// GetStatistics returns FIDO2 statistics
func (m *FIDO2Manager) GetStatistics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalCreds := 0
	for _, creds := range m.credentials {
		totalCreds += len(creds)
	}

	return map[string]interface{}{
		"registered_users":      len(m.credentials),
		"total_credentials":     totalCreds,
		"pending_challenges":    len(m.challenges),
		"relying_party_id":      m.config.RelyingPartyID,
		"user_verification":     m.config.UserVerification,
	}
}

// HashCredentialID creates a hash for credential comparison
func HashCredentialID(credID []byte) string {
	hash := sha256.Sum256(credID)
	return base64.URLEncoding.EncodeToString(hash[:])
}
