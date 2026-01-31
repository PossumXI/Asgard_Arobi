// Package services provides business logic services for the API.
package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/asgard/pandora/internal/repositories"
	"github.com/asgard/pandora/internal/utils"
)

// AuditService handles audit logging business logic.
type AuditService struct {
	auditRepo  *repositories.AuditLogRepository
	ethicsRepo *repositories.EthicalDecisionRepository
	logger     *utils.Logger
	fileLogger *FileAuditLogger
	enableFile bool
	mu         sync.RWMutex
}

// FileAuditLogger handles file-based audit logging.
type FileAuditLogger struct {
	logDir   string
	mu       sync.Mutex
	file     *os.File
	fileName string
}

// NewAuditService creates a new audit service.
func NewAuditService(
	auditRepo *repositories.AuditLogRepository,
	ethicsRepo *repositories.EthicalDecisionRepository,
	logDir string,
	enableFileLogging bool,
) *AuditService {
	var fileLogger *FileAuditLogger
	if enableFileLogging && logDir != "" {
		fileLogger = NewFileAuditLogger(logDir)
	}

	return &AuditService{
		auditRepo:  auditRepo,
		ethicsRepo: ethicsRepo,
		logger:     utils.NewLogger(),
		fileLogger: fileLogger,
		enableFile: enableFileLogging,
	}
}

// NewFileAuditLogger creates a new file audit logger.
func NewFileAuditLogger(logDir string) *FileAuditLogger {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil
	}
	return &FileAuditLogger{
		logDir: logDir,
	}
}

// LogEntry represents a structured audit log entry for file logging.
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Component string                 `json:"component"`
	Action    string                 `json:"action"`
	UserID    string                 `json:"user_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// CreateAuditLog creates a new audit log entry in both database and file.
func (s *AuditService) CreateAuditLog(ctx context.Context, component, action string, userID *string, metadata map[string]interface{}) error {
	// Prepare the database log entry
	log := &db.AuditLog{
		Component: component,
		Action:    action,
		CreatedAt: time.Now().UTC(),
	}

	if userID != nil && *userID != "" {
		log.UserID = sql.NullString{String: *userID, Valid: true}
	}

	if metadata != nil {
		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			s.logger.Error("Failed to marshal audit metadata: %v", err)
		} else {
			log.Metadata = metadataJSON
		}
	}

	// Write to database
	if err := s.auditRepo.Create(ctx, log); err != nil {
		s.logger.Error("Failed to create audit log in database: %v", err)
		// Continue to write to file even if database fails
	}

	// Write to file if enabled
	if s.enableFile && s.fileLogger != nil {
		entry := LogEntry{
			Timestamp: log.CreatedAt,
			Component: component,
			Action:    action,
			Metadata:  metadata,
		}
		if userID != nil {
			entry.UserID = *userID
		}
		if err := s.fileLogger.Write(entry); err != nil {
			s.logger.Error("Failed to write audit log to file: %v", err)
		}
	}

	return nil
}

// LogUserAction logs a user-initiated action.
func (s *AuditService) LogUserAction(ctx context.Context, userID, action string, details map[string]interface{}) error {
	metadata := map[string]interface{}{
		"type": "user_action",
	}
	for k, v := range details {
		metadata[k] = v
	}
	return s.CreateAuditLog(ctx, "user", action, &userID, metadata)
}

// LogSystemEvent logs a system event.
func (s *AuditService) LogSystemEvent(ctx context.Context, component, action string, details map[string]interface{}) error {
	metadata := map[string]interface{}{
		"type": "system_event",
	}
	for k, v := range details {
		metadata[k] = v
	}
	return s.CreateAuditLog(ctx, component, action, nil, metadata)
}

// LogSecurityEvent logs a security-related event.
func (s *AuditService) LogSecurityEvent(ctx context.Context, action string, userID *string, details map[string]interface{}) error {
	metadata := map[string]interface{}{
		"type":     "security_event",
		"severity": "high",
	}
	for k, v := range details {
		metadata[k] = v
	}
	return s.CreateAuditLog(ctx, "security", action, userID, metadata)
}

// LogEthicalDecision logs an ethical decision event.
func (s *AuditService) LogEthicalDecision(ctx context.Context, hunoidID, decision, reasoning string, details map[string]interface{}) error {
	metadata := map[string]interface{}{
		"type":      "ethical_decision",
		"hunoid_id": hunoidID,
		"decision":  decision,
		"reasoning": reasoning,
	}
	for k, v := range details {
		metadata[k] = v
	}
	return s.CreateAuditLog(ctx, "ethics", "ethical_decision_made", nil, metadata)
}

// GetAuditLogs retrieves audit logs with filters.
func (s *AuditService) GetAuditLogs(ctx context.Context, filters repositories.AuditLogFilters) ([]*db.AuditLog, error) {
	logs, err := s.auditRepo.GetWithFilters(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}
	return logs, nil
}

// GetAuditLogByID retrieves a specific audit log.
func (s *AuditService) GetAuditLogByID(ctx context.Context, id int64) (*db.AuditLog, error) {
	log, err := s.auditRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}
	return log, nil
}

// GetAuditLogsByComponent retrieves audit logs for a component.
func (s *AuditService) GetAuditLogsByComponent(ctx context.Context, component string, since time.Time) ([]*db.AuditLog, error) {
	logs, err := s.auditRepo.GetByComponent(ctx, component, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs by component: %w", err)
	}
	return logs, nil
}

// GetAuditLogsByUser retrieves audit logs for a user.
func (s *AuditService) GetAuditLogsByUser(ctx context.Context, userID string, limit int) ([]*db.AuditLog, error) {
	logs, err := s.auditRepo.GetByUserID(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs by user: %w", err)
	}
	return logs, nil
}

// GetAuditLogsByDateRange retrieves audit logs within a date range.
func (s *AuditService) GetAuditLogsByDateRange(ctx context.Context, start, end time.Time) ([]*db.AuditLog, error) {
	logs, err := s.auditRepo.GetByDateRange(ctx, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs by date range: %w", err)
	}
	return logs, nil
}

// GetRecentAuditLogs retrieves recent audit logs.
func (s *AuditService) GetRecentAuditLogs(ctx context.Context, limit int) ([]*db.AuditLog, error) {
	logs, err := s.auditRepo.GetRecent(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent audit logs: %w", err)
	}
	return logs, nil
}

// GetAuditStats returns audit log statistics.
func (s *AuditService) GetAuditStats(ctx context.Context, since time.Time) (map[string]interface{}, error) {
	counts, err := s.auditRepo.CountByComponent(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit stats: %w", err)
	}

	total := 0
	for _, count := range counts {
		total += count
	}

	return map[string]interface{}{
		"by_component": counts,
		"total":        total,
		"since":        since,
	}, nil
}

// GetEthicalDecisions retrieves ethical decisions with filters.
func (s *AuditService) GetEthicalDecisions(ctx context.Context, limit int) ([]*db.EthicalDecision, error) {
	decisions, err := s.ethicsRepo.GetRecent(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get ethical decisions: %w", err)
	}
	return decisions, nil
}

// GetEthicalDecisionByID retrieves a specific ethical decision.
func (s *AuditService) GetEthicalDecisionByID(ctx context.Context, id string) (*db.EthicalDecision, error) {
	decision, err := s.ethicsRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get ethical decision: %w", err)
	}
	return decision, nil
}

// GetEthicalDecisionsByHunoid retrieves ethical decisions for a hunoid.
func (s *AuditService) GetEthicalDecisionsByHunoid(ctx context.Context, hunoidID string, limit int) ([]*db.EthicalDecision, error) {
	decisions, err := s.ethicsRepo.GetByHunoidID(ctx, hunoidID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get ethical decisions by hunoid: %w", err)
	}
	return decisions, nil
}

// GetEthicalDecisionsByMission retrieves ethical decisions for a mission.
func (s *AuditService) GetEthicalDecisionsByMission(ctx context.Context, missionID string) ([]*db.EthicalDecision, error) {
	decisions, err := s.ethicsRepo.GetByMissionID(ctx, missionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ethical decisions by mission: %w", err)
	}
	return decisions, nil
}

// GetEthicalDecisionsByType retrieves ethical decisions by decision type.
func (s *AuditService) GetEthicalDecisionsByType(ctx context.Context, decisionType string, limit int) ([]*db.EthicalDecision, error) {
	decisions, err := s.ethicsRepo.GetByDecisionType(ctx, decisionType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get ethical decisions by type: %w", err)
	}
	return decisions, nil
}

// GetEthicsStats returns ethical decision statistics.
func (s *AuditService) GetEthicsStats(ctx context.Context) (map[string]interface{}, error) {
	counts, err := s.ethicsRepo.CountByDecisionType(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get ethics stats: %w", err)
	}

	total := 0
	for _, count := range counts {
		total += count
	}

	approvalRate := 0.0
	if total > 0 {
		approved := counts["approved"]
		approvalRate = float64(approved) / float64(total) * 100
	}

	return map[string]interface{}{
		"by_decision":   counts,
		"total":         total,
		"approval_rate": approvalRate,
	}, nil
}

// CleanupOldLogs removes audit logs older than the retention period.
func (s *AuditService) CleanupOldLogs(ctx context.Context, retentionDays int) (int64, error) {
	before := time.Now().AddDate(0, 0, -retentionDays)
	count, err := s.auditRepo.DeleteOlderThan(ctx, before)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old logs: %w", err)
	}

	// Log the cleanup action
	s.LogSystemEvent(ctx, "audit", "logs_cleaned", map[string]interface{}{
		"deleted_count":  count,
		"retention_days": retentionDays,
	})

	return count, nil
}

// Write writes a log entry to the file.
func (f *FileAuditLogger) Write(entry LogEntry) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Rotate file daily
	today := time.Now().Format("2006-01-02")
	expectedFileName := fmt.Sprintf("audit_%s.log", today)

	if f.fileName != expectedFileName {
		if f.file != nil {
			f.file.Close()
		}

		filePath := filepath.Join(f.logDir, expectedFileName)
		file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open audit log file: %w", err)
		}

		f.file = file
		f.fileName = expectedFileName
	}

	// Write JSON line
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	if _, err := f.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}

	return nil
}

// Close closes the file logger.
func (f *FileAuditLogger) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.file != nil {
		return f.file.Close()
	}
	return nil
}

// Close closes the audit service resources.
func (s *AuditService) Close() error {
	if s.fileLogger != nil {
		return s.fileLogger.Close()
	}
	return nil
}
