// Package vault implements audit logging for vault access tracking.
//
// All vault operations are logged for compliance with DO-178C DAL-B requirements
// and government security standards.
//
// Copyright 2026 Arobi. All Rights Reserved.
package vault

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditEvent represents a security audit log entry
type AuditEvent struct {
	ID            string        `json:"id"`
	Timestamp     time.Time     `json:"timestamp"`
	Action        string        `json:"action"`
	Actor         string        `json:"actor"`
	ActorIP       string        `json:"actor_ip,omitempty"`
	UserAgent     string        `json:"user_agent,omitempty"`
	Resource      string        `json:"resource,omitempty"`
	ResourceID    string        `json:"resource_id,omitempty"`
	SecurityLevel SecurityLevel `json:"security_level,omitempty"`
	Success       bool          `json:"success"`
	Details       string        `json:"details,omitempty"`
	FIDO2Used     bool          `json:"fido2_used,omitempty"`
	SessionID     string        `json:"session_id,omitempty"`
	Duration      time.Duration `json:"duration_ns,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// AuditLogger handles security audit logging
type AuditLogger struct {
	mu          sync.Mutex
	file        *os.File
	filePath    string
	events      []AuditEvent
	maxInMemory int
	rotateSize  int64 // bytes
	retentionDays int
	eventChan   chan AuditEvent
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// AuditLoggerConfig holds audit logger configuration
type AuditLoggerConfig struct {
	FilePath      string
	MaxInMemory   int
	RotateSize    int64
	RetentionDays int
	AsyncWrite    bool
}

// DefaultAuditLoggerConfig returns default configuration
func DefaultAuditLoggerConfig() AuditLoggerConfig {
	return AuditLoggerConfig{
		FilePath:      "./data/vault/audit.log",
		MaxInMemory:   1000,
		RotateSize:    10 * 1024 * 1024, // 10MB
		RetentionDays: 365,              // 1 year
		AsyncWrite:    true,
	}
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(filePath string) (*AuditLogger, error) {
	cfg := DefaultAuditLoggerConfig()
	cfg.FilePath = filePath
	return NewAuditLoggerWithConfig(cfg)
}

// NewAuditLoggerWithConfig creates an audit logger with custom configuration
func NewAuditLoggerWithConfig(cfg AuditLoggerConfig) (*AuditLogger, error) {
	// Create directory if needed
	dir := filepath.Dir(cfg.FilePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	file, err := os.OpenFile(cfg.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}

	al := &AuditLogger{
		file:          file,
		filePath:      cfg.FilePath,
		events:        make([]AuditEvent, 0, cfg.MaxInMemory),
		maxInMemory:   cfg.MaxInMemory,
		rotateSize:    cfg.RotateSize,
		retentionDays: cfg.RetentionDays,
		eventChan:     make(chan AuditEvent, 100),
		stopCh:        make(chan struct{}),
	}

	if cfg.AsyncWrite {
		al.wg.Add(1)
		go al.asyncWriter()
	}

	log.Printf("[Audit] Logger initialized: %s", cfg.FilePath)
	return al, nil
}

// LogEvent logs an audit event
func (al *AuditLogger) LogEvent(event AuditEvent) {
	// Generate ID if not set
	if event.ID == "" {
		event.ID = fmt.Sprintf("audit-%d", time.Now().UnixNano())
	}

	// Ensure timestamp is set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Send to async writer or write directly
	select {
	case al.eventChan <- event:
	default:
		// Channel full, write directly
		al.writeEvent(event)
	}
}

// asyncWriter handles async event writing
func (al *AuditLogger) asyncWriter() {
	defer al.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var batch []AuditEvent

	for {
		select {
		case <-al.stopCh:
			// Flush remaining events
			for {
				select {
				case event := <-al.eventChan:
					batch = append(batch, event)
				default:
					if len(batch) > 0 {
						al.writeBatch(batch)
					}
					return
				}
			}
		case event := <-al.eventChan:
			batch = append(batch, event)
			if len(batch) >= 50 {
				al.writeBatch(batch)
				batch = nil
			}
		case <-ticker.C:
			if len(batch) > 0 {
				al.writeBatch(batch)
				batch = nil
			}
		}
	}
}

// writeEvent writes a single event
func (al *AuditLogger) writeEvent(event AuditEvent) {
	al.mu.Lock()
	defer al.mu.Unlock()

	// Store in memory
	al.events = append(al.events, event)
	if len(al.events) > al.maxInMemory {
		al.events = al.events[1:]
	}

	// Write to file
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("[Audit] Failed to marshal event: %v", err)
		return
	}

	if _, err := al.file.Write(append(data, '\n')); err != nil {
		log.Printf("[Audit] Failed to write event: %v", err)
		return
	}

	// Check for rotation
	al.checkRotation()
}

// writeBatch writes a batch of events
func (al *AuditLogger) writeBatch(events []AuditEvent) {
	al.mu.Lock()
	defer al.mu.Unlock()

	for _, event := range events {
		// Store in memory
		al.events = append(al.events, event)

		// Write to file
		data, err := json.Marshal(event)
		if err != nil {
			log.Printf("[Audit] Failed to marshal event: %v", err)
			continue
		}

		if _, err := al.file.Write(append(data, '\n')); err != nil {
			log.Printf("[Audit] Failed to write event: %v", err)
		}
	}

	// Trim in-memory events
	if len(al.events) > al.maxInMemory {
		al.events = al.events[len(al.events)-al.maxInMemory:]
	}

	// Sync and check rotation
	al.file.Sync()
	al.checkRotation()
}

// checkRotation checks if log rotation is needed
func (al *AuditLogger) checkRotation() {
	stat, err := al.file.Stat()
	if err != nil {
		return
	}

	if stat.Size() >= al.rotateSize {
		al.rotate()
	}
}

// rotate performs log rotation
func (al *AuditLogger) rotate() {
	// Close current file
	al.file.Close()

	// Rename current file with timestamp
	timestamp := time.Now().Format("20060102-150405")
	rotatedPath := fmt.Sprintf("%s.%s", al.filePath, timestamp)
	os.Rename(al.filePath, rotatedPath)

	// Open new file
	file, err := os.OpenFile(al.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Printf("[Audit] Failed to create new log file: %v", err)
		return
	}
	al.file = file

	log.Printf("[Audit] Log rotated: %s", rotatedPath)

	// Schedule cleanup of old logs
	go al.cleanupOldLogs()
}

// cleanupOldLogs removes logs older than retention period
func (al *AuditLogger) cleanupOldLogs() {
	dir := filepath.Dir(al.filePath)
	base := filepath.Base(al.filePath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -al.retentionDays)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if len(name) <= len(base) || name[:len(base)] != base {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			path := filepath.Join(dir, name)
			if err := os.Remove(path); err == nil {
				log.Printf("[Audit] Removed old log: %s", name)
			}
		}
	}
}

// QueryEvents queries audit events with filters
func (al *AuditLogger) QueryEvents(filter AuditQueryFilter) []AuditEvent {
	al.mu.Lock()
	defer al.mu.Unlock()

	result := make([]AuditEvent, 0)

	for _, event := range al.events {
		// Apply filters
		if filter.Actor != "" && event.Actor != filter.Actor {
			continue
		}
		if filter.Action != "" && event.Action != filter.Action {
			continue
		}
		if filter.Resource != "" && event.Resource != filter.Resource {
			continue
		}
		if filter.SecurityLevel != "" && event.SecurityLevel != filter.SecurityLevel {
			continue
		}
		if !filter.StartTime.IsZero() && event.Timestamp.Before(filter.StartTime) {
			continue
		}
		if !filter.EndTime.IsZero() && event.Timestamp.After(filter.EndTime) {
			continue
		}
		if filter.SuccessOnly && !event.Success {
			continue
		}
		if filter.FailureOnly && event.Success {
			continue
		}

		result = append(result, event)

		if filter.Limit > 0 && len(result) >= filter.Limit {
			break
		}
	}

	return result
}

// GetRecentEvents returns the most recent events
func (al *AuditLogger) GetRecentEvents(count int) []AuditEvent {
	al.mu.Lock()
	defer al.mu.Unlock()

	if count <= 0 || count > len(al.events) {
		count = len(al.events)
	}

	start := len(al.events) - count
	result := make([]AuditEvent, count)
	copy(result, al.events[start:])

	return result
}

// GetStatistics returns audit statistics
func (al *AuditLogger) GetStatistics() AuditStatistics {
	al.mu.Lock()
	defer al.mu.Unlock()

	stats := AuditStatistics{
		TotalEvents:   len(al.events),
		ByAction:      make(map[string]int),
		ByActor:       make(map[string]int),
		BySecurityLevel: make(map[SecurityLevel]int),
	}

	for _, event := range al.events {
		stats.ByAction[event.Action]++
		stats.ByActor[event.Actor]++
		if event.SecurityLevel != "" {
			stats.BySecurityLevel[event.SecurityLevel]++
		}
		if !event.Success {
			stats.FailedCount++
		}
		if event.FIDO2Used {
			stats.FIDO2AccessCount++
		}
	}

	return stats
}

// Close closes the audit logger
func (al *AuditLogger) Close() error {
	close(al.stopCh)
	al.wg.Wait()

	al.mu.Lock()
	defer al.mu.Unlock()

	if al.file != nil {
		al.file.Sync()
		return al.file.Close()
	}
	return nil
}

// AuditQueryFilter defines filters for querying audit events
type AuditQueryFilter struct {
	Actor         string
	Action        string
	Resource      string
	SecurityLevel SecurityLevel
	StartTime     time.Time
	EndTime       time.Time
	SuccessOnly   bool
	FailureOnly   bool
	Limit         int
}

// AuditStatistics contains audit statistics
type AuditStatistics struct {
	TotalEvents      int
	FailedCount      int
	FIDO2AccessCount int
	ByAction         map[string]int
	ByActor          map[string]int
	BySecurityLevel  map[SecurityLevel]int
}

// Common audit action constants
const (
	AuditActionStoreSecret     = "store_secret"
	AuditActionRetrieveSecret  = "retrieve_secret"
	AuditActionDeleteSecret    = "delete_secret"
	AuditActionListSecrets     = "list_secrets"
	AuditActionVaultInit       = "vault_init"
	AuditActionVaultSeal       = "vault_seal"
	AuditActionVaultUnseal     = "vault_unseal"
	AuditActionFIDO2Register   = "fido2_register"
	AuditActionFIDO2Authenticate = "fido2_authenticate"
	AuditActionFIDO2Revoke     = "fido2_revoke"
	AuditActionAccessDenied    = "access_denied"
	AuditActionSuspiciousAccess = "suspicious_access"
)

// FormatEvent formats an audit event for display
func (e *AuditEvent) FormatEvent() string {
	status := "SUCCESS"
	if !e.Success {
		status = "FAILED"
	}

	fido2 := ""
	if e.FIDO2Used {
		fido2 = " [FIDO2]"
	}

	return fmt.Sprintf("[%s] %s - %s - %s%s - %s",
		e.Timestamp.Format(time.RFC3339),
		e.Action,
		e.Actor,
		status,
		fido2,
		e.Resource,
	)
}
