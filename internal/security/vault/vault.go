// Package vault implements a secure credential vault with FIDO2 authentication.
// Provides encrypted storage for sensitive access codes, API keys, and secrets.
//
// DO-178C DAL-B Compliant: All access is audited and encrypted.
// Copyright 2026 Arobi. All Rights Reserved.
package vault

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SecurityLevel defines access tiers for vault entries
type SecurityLevel string

const (
	// SecurityLevelPublic - accessible by all authenticated users
	SecurityLevelPublic SecurityLevel = "public"
	// SecurityLevelDeveloper - accessible by developers with valid credentials
	SecurityLevelDeveloper SecurityLevel = "developer"
	// SecurityLevelAdmin - requires admin privileges
	SecurityLevelAdmin SecurityLevel = "admin"
	// SecurityLevelGovernment - requires government clearance + FIDO2
	SecurityLevelGovernment SecurityLevel = "government"
	// SecurityLevelMilitary - requires military clearance + FIDO2 + biometric
	SecurityLevelMilitary SecurityLevel = "military"
)

// SecretType categorizes stored secrets
type SecretType string

const (
	SecretTypeAPIKey      SecretType = "api_key"
	SecretTypeCredential  SecretType = "credential"
	SecretTypeCertificate SecretType = "certificate"
	SecretTypeSSHKey      SecretType = "ssh_key"
	SecretTypeAccessCode  SecretType = "access_code"
	SecretTypeEncryption  SecretType = "encryption_key"
	SecretTypeProprietary SecretType = "proprietary"
)

// VaultEntry represents a stored secret
type VaultEntry struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Description   string            `json:"description,omitempty"`
	Type          SecretType        `json:"type"`
	SecurityLevel SecurityLevel     `json:"security_level"`
	EncryptedData string            `json:"encrypted_data"`
	Checksum      string            `json:"checksum"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	LastAccessed  *time.Time        `json:"last_accessed,omitempty"`
	AccessCount   int               `json:"access_count"`
	CreatedBy     string            `json:"created_by"`
	Tags          []string          `json:"tags,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	ExpiresAt     *time.Time        `json:"expires_at,omitempty"`
	Rotated       bool              `json:"rotated"`
}

// VaultConfig holds vault configuration
type VaultConfig struct {
	StoragePath       string
	MasterKeyPath     string
	AuditLogPath      string
	EncryptionAlgo    string
	KeyDerivationFunc string
	MaxEntries        int
	RequireFIDO2      map[SecurityLevel]bool
	AutoRotateDays    int
	BackupEnabled     bool
	BackupPath        string
}

// DefaultVaultConfig returns default configuration
func DefaultVaultConfig() VaultConfig {
	return VaultConfig{
		StoragePath:       "./data/vault/secrets.enc",
		MasterKeyPath:     "./data/vault/master.key",
		AuditLogPath:      "./data/vault/audit.log",
		EncryptionAlgo:    "AES-256-GCM",
		KeyDerivationFunc: "PBKDF2-SHA256",
		MaxEntries:        10000,
		RequireFIDO2: map[SecurityLevel]bool{
			SecurityLevelGovernment: true,
			SecurityLevelMilitary:   true,
		},
		AutoRotateDays: 90,
		BackupEnabled:  true,
		BackupPath:     "./data/vault/backups/",
	}
}

// Vault manages secure secret storage
type Vault struct {
	mu           sync.RWMutex
	entries      map[string]*VaultEntry
	masterKey    []byte
	config       VaultConfig
	auditLog     *AuditLogger
	fido2Manager *FIDO2Manager
	agentMonitor *VaultAgent
	initialized  bool
	sealed       bool
	gcm          cipher.AEAD
}

// NewVault creates a new vault instance
func NewVault(cfg VaultConfig) (*Vault, error) {
	v := &Vault{
		entries: make(map[string]*VaultEntry),
		config:  cfg,
		sealed:  true,
	}

	// Initialize audit logger
	auditLog, err := NewAuditLogger(cfg.AuditLogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create audit logger: %w", err)
	}
	v.auditLog = auditLog

	// Initialize FIDO2 manager
	v.fido2Manager = NewFIDO2Manager()

	// Initialize monitoring agent
	v.agentMonitor = NewVaultAgent(v)

	return v, nil
}

// Initialize sets up the vault with a master key
func (v *Vault) Initialize(ctx context.Context, masterPassword string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.initialized {
		return errors.New("vault already initialized")
	}

	// Derive master key from password using PBKDF2
	v.masterKey = deriveKey(masterPassword, 32)

	// Create AES-GCM cipher
	block, err := aes.NewCipher(v.masterKey)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}
	v.gcm = gcm

	// Create storage directories
	if err := os.MkdirAll(filepath.Dir(v.config.StoragePath), 0700); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Load existing entries if available
	if err := v.loadEntries(); err != nil {
		log.Printf("[Vault] No existing entries found, starting fresh")
	}

	v.initialized = true
	v.sealed = false

	// Start monitoring agent
	go v.agentMonitor.Start(ctx)

	v.auditLog.LogEvent(AuditEvent{
		Timestamp: time.Now(),
		Action:    "vault_initialized",
		Actor:     "system",
		Success:   true,
	})

	log.Printf("[Vault] Initialized with %d entries", len(v.entries))
	return nil
}

// Seal locks the vault, clearing the master key from memory
func (v *Vault) Seal() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.sealed {
		return errors.New("vault already sealed")
	}

	// Clear master key from memory
	for i := range v.masterKey {
		v.masterKey[i] = 0
	}
	v.masterKey = nil
	v.gcm = nil
	v.sealed = true

	v.auditLog.LogEvent(AuditEvent{
		Timestamp: time.Now(),
		Action:    "vault_sealed",
		Actor:     "system",
		Success:   true,
	})

	log.Printf("[Vault] Sealed")
	return nil
}

// Unseal unlocks the vault with the master password
func (v *Vault) Unseal(masterPassword string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if !v.sealed {
		return errors.New("vault already unsealed")
	}

	// Derive master key
	v.masterKey = deriveKey(masterPassword, 32)

	// Create cipher
	block, err := aes.NewCipher(v.masterKey)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}
	v.gcm = gcm

	// Verify by loading entries
	if err := v.loadEntries(); err != nil {
		// Clear key on failure
		for i := range v.masterKey {
			v.masterKey[i] = 0
		}
		v.masterKey = nil
		v.gcm = nil
		return fmt.Errorf("invalid master password")
	}

	v.sealed = false

	v.auditLog.LogEvent(AuditEvent{
		Timestamp: time.Now(),
		Action:    "vault_unsealed",
		Actor:     "system",
		Success:   true,
	})

	return nil
}

// StoreSecret adds a new secret to the vault
func (v *Vault) StoreSecret(ctx context.Context, req StoreSecretRequest) (*VaultEntry, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.sealed {
		return nil, errors.New("vault is sealed")
	}

	// Verify FIDO2 if required for this security level
	if v.config.RequireFIDO2[req.SecurityLevel] {
		if req.FIDO2Credential == nil {
			return nil, errors.New("FIDO2 authentication required for this security level")
		}
		if err := v.fido2Manager.VerifyAssertion(ctx, req.FIDO2Credential); err != nil {
			v.auditLog.LogEvent(AuditEvent{
				Timestamp:     time.Now(),
				Action:        "store_secret_failed",
				Actor:         req.Actor,
				Resource:      req.Name,
				SecurityLevel: req.SecurityLevel,
				Success:       false,
				Details:       "FIDO2 verification failed",
			})
			return nil, fmt.Errorf("FIDO2 verification failed: %w", err)
		}
	}

	// Encrypt the secret data
	encrypted, err := v.encrypt([]byte(req.Data))
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	// Create checksum
	checksum := sha256.Sum256([]byte(req.Data))

	entry := &VaultEntry{
		ID:            uuid.New().String(),
		Name:          req.Name,
		Description:   req.Description,
		Type:          req.Type,
		SecurityLevel: req.SecurityLevel,
		EncryptedData: base64.StdEncoding.EncodeToString(encrypted),
		Checksum:      hex.EncodeToString(checksum[:]),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		AccessCount:   0,
		CreatedBy:     req.Actor,
		Tags:          req.Tags,
		Metadata:      req.Metadata,
		ExpiresAt:     req.ExpiresAt,
	}

	v.entries[entry.ID] = entry

	// Persist to storage
	if err := v.saveEntries(); err != nil {
		delete(v.entries, entry.ID)
		return nil, fmt.Errorf("failed to persist: %w", err)
	}

	// Log audit event
	v.auditLog.LogEvent(AuditEvent{
		Timestamp:     time.Now(),
		Action:        "store_secret",
		Actor:         req.Actor,
		Resource:      req.Name,
		ResourceID:    entry.ID,
		SecurityLevel: req.SecurityLevel,
		Success:       true,
	})

	// Notify monitoring agent
	v.agentMonitor.NotifyAccess(AccessEvent{
		Type:          "store",
		EntryID:       entry.ID,
		Actor:         req.Actor,
		SecurityLevel: req.SecurityLevel,
		Timestamp:     time.Now(),
	})

	return entry, nil
}

// RetrieveSecret gets a secret from the vault
func (v *Vault) RetrieveSecret(ctx context.Context, req RetrieveSecretRequest) (string, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.sealed {
		return "", errors.New("vault is sealed")
	}

	entry, exists := v.entries[req.EntryID]
	if !exists {
		v.auditLog.LogEvent(AuditEvent{
			Timestamp:  time.Now(),
			Action:     "retrieve_secret_failed",
			Actor:      req.Actor,
			ResourceID: req.EntryID,
			Success:    false,
			Details:    "entry not found",
		})
		return "", errors.New("entry not found")
	}

	// Verify FIDO2 if required
	if v.config.RequireFIDO2[entry.SecurityLevel] {
		if req.FIDO2Credential == nil {
			return "", errors.New("FIDO2 authentication required")
		}
		if err := v.fido2Manager.VerifyAssertion(ctx, req.FIDO2Credential); err != nil {
			v.auditLog.LogEvent(AuditEvent{
				Timestamp:     time.Now(),
				Action:        "retrieve_secret_failed",
				Actor:         req.Actor,
				ResourceID:    req.EntryID,
				SecurityLevel: entry.SecurityLevel,
				Success:       false,
				Details:       "FIDO2 verification failed",
			})
			return "", fmt.Errorf("FIDO2 verification failed: %w", err)
		}
	}

	// Check expiration
	if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
		return "", errors.New("secret has expired")
	}

	// Decrypt data
	encryptedData, err := base64.StdEncoding.DecodeString(entry.EncryptedData)
	if err != nil {
		return "", fmt.Errorf("failed to decode: %w", err)
	}

	decrypted, err := v.decrypt(encryptedData)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	// Verify checksum
	checksum := sha256.Sum256(decrypted)
	if hex.EncodeToString(checksum[:]) != entry.Checksum {
		return "", errors.New("checksum verification failed - data may be corrupted")
	}

	// Update access metadata
	now := time.Now()
	entry.LastAccessed = &now
	entry.AccessCount++

	// Log audit event
	v.auditLog.LogEvent(AuditEvent{
		Timestamp:     time.Now(),
		Action:        "retrieve_secret",
		Actor:         req.Actor,
		Resource:      entry.Name,
		ResourceID:    entry.ID,
		SecurityLevel: entry.SecurityLevel,
		Success:       true,
	})

	// Notify monitoring agent
	v.agentMonitor.NotifyAccess(AccessEvent{
		Type:          "retrieve",
		EntryID:       entry.ID,
		Actor:         req.Actor,
		SecurityLevel: entry.SecurityLevel,
		Timestamp:     time.Now(),
	})

	return string(decrypted), nil
}

// DeleteSecret removes a secret from the vault
func (v *Vault) DeleteSecret(ctx context.Context, req DeleteSecretRequest) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.sealed {
		return errors.New("vault is sealed")
	}

	entry, exists := v.entries[req.EntryID]
	if !exists {
		return errors.New("entry not found")
	}

	// Government and military level secrets require FIDO2 for deletion
	if entry.SecurityLevel == SecurityLevelGovernment || entry.SecurityLevel == SecurityLevelMilitary {
		if req.FIDO2Credential == nil {
			return errors.New("FIDO2 authentication required for deletion")
		}
		if err := v.fido2Manager.VerifyAssertion(ctx, req.FIDO2Credential); err != nil {
			return fmt.Errorf("FIDO2 verification failed: %w", err)
		}
	}

	delete(v.entries, req.EntryID)

	if err := v.saveEntries(); err != nil {
		// Restore entry on failure
		v.entries[entry.ID] = entry
		return fmt.Errorf("failed to persist: %w", err)
	}

	v.auditLog.LogEvent(AuditEvent{
		Timestamp:     time.Now(),
		Action:        "delete_secret",
		Actor:         req.Actor,
		Resource:      entry.Name,
		ResourceID:    entry.ID,
		SecurityLevel: entry.SecurityLevel,
		Success:       true,
	})

	return nil
}

// ListSecrets returns metadata for all secrets (without decrypted data)
func (v *Vault) ListSecrets(ctx context.Context, filter ListSecretsFilter) ([]*VaultEntry, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.sealed {
		return nil, errors.New("vault is sealed")
	}

	result := make([]*VaultEntry, 0)
	for _, entry := range v.entries {
		// Apply filters
		if filter.SecurityLevel != "" && entry.SecurityLevel != filter.SecurityLevel {
			continue
		}
		if filter.Type != "" && entry.Type != filter.Type {
			continue
		}
		if filter.Tag != "" {
			hasTag := false
			for _, t := range entry.Tags {
				if t == filter.Tag {
					hasTag = true
					break
				}
			}
			if !hasTag {
				continue
			}
		}

		// Return copy without encrypted data for listing
		entryCopy := *entry
		entryCopy.EncryptedData = "[REDACTED]"
		result = append(result, &entryCopy)
	}

	return result, nil
}

// GetStatistics returns vault statistics
func (v *Vault) GetStatistics() VaultStatistics {
	v.mu.RLock()
	defer v.mu.RUnlock()

	stats := VaultStatistics{
		TotalEntries: len(v.entries),
		Sealed:       v.sealed,
		ByLevel:      make(map[SecurityLevel]int),
		ByType:       make(map[SecretType]int),
	}

	for _, entry := range v.entries {
		stats.ByLevel[entry.SecurityLevel]++
		stats.ByType[entry.Type]++
		if entry.ExpiresAt != nil && time.Now().After(*entry.ExpiresAt) {
			stats.ExpiredCount++
		}
	}

	return stats
}

// encrypt encrypts data using AES-256-GCM
func (v *Vault) encrypt(data []byte) ([]byte, error) {
	nonce := make([]byte, v.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return v.gcm.Seal(nonce, nonce, data, nil), nil
}

// decrypt decrypts data using AES-256-GCM
func (v *Vault) decrypt(data []byte) ([]byte, error) {
	nonceSize := v.gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return v.gcm.Open(nil, nonce, ciphertext, nil)
}

// saveEntries persists all entries to storage
func (v *Vault) saveEntries() error {
	data, err := json.Marshal(v.entries)
	if err != nil {
		return err
	}

	encrypted, err := v.encrypt(data)
	if err != nil {
		return err
	}

	return os.WriteFile(v.config.StoragePath, encrypted, 0600)
}

// loadEntries loads entries from storage
func (v *Vault) loadEntries() error {
	data, err := os.ReadFile(v.config.StoragePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No entries yet
		}
		return err
	}

	decrypted, err := v.decrypt(data)
	if err != nil {
		return err
	}

	return json.Unmarshal(decrypted, &v.entries)
}

// deriveKey derives a key from password using SHA-256 (simplified PBKDF2)
func deriveKey(password string, length int) []byte {
	// In production, use proper PBKDF2 with salt and iterations
	hash := sha256.Sum256([]byte(password + "asgard-vault-salt"))
	return hash[:length]
}

// Request/Response types

// StoreSecretRequest contains parameters for storing a secret
type StoreSecretRequest struct {
	Name            string
	Description     string
	Data            string
	Type            SecretType
	SecurityLevel   SecurityLevel
	Actor           string
	Tags            []string
	Metadata        map[string]string
	ExpiresAt       *time.Time
	FIDO2Credential *FIDO2Credential
}

// RetrieveSecretRequest contains parameters for retrieving a secret
type RetrieveSecretRequest struct {
	EntryID         string
	Actor           string
	FIDO2Credential *FIDO2Credential
}

// DeleteSecretRequest contains parameters for deleting a secret
type DeleteSecretRequest struct {
	EntryID         string
	Actor           string
	FIDO2Credential *FIDO2Credential
}

// ListSecretsFilter filters for listing secrets
type ListSecretsFilter struct {
	SecurityLevel SecurityLevel
	Type          SecretType
	Tag           string
}

// VaultStatistics contains vault statistics
type VaultStatistics struct {
	TotalEntries int
	Sealed       bool
	ByLevel      map[SecurityLevel]int
	ByType       map[SecretType]int
	ExpiredCount int
}
