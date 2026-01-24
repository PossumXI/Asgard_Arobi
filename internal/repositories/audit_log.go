// Package repositories provides data access layer for database operations.
package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
)

// AuditLogRepository handles audit log database operations.
type AuditLogRepository struct {
	db *db.PostgresDB
}

// NewAuditLogRepository creates a new audit log repository.
func NewAuditLogRepository(pgDB *db.PostgresDB) *AuditLogRepository {
	return &AuditLogRepository{db: pgDB}
}

// Create inserts a new audit log record.
func (r *AuditLogRepository) Create(ctx context.Context, log *db.AuditLog) error {
	query := `
		INSERT INTO audit_logs (component, action, user_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now().UTC()
	}

	err := r.db.QueryRowContext(ctx, query,
		log.Component,
		log.Action,
		log.UserID,
		log.Metadata,
		log.CreatedAt,
	).Scan(&log.ID)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// GetByID retrieves an audit log by ID.
func (r *AuditLogRepository) GetByID(ctx context.Context, id int64) (*db.AuditLog, error) {
	query := `
		SELECT id, component, action, user_id, metadata, created_at
		FROM audit_logs
		WHERE id = $1
	`

	log := &db.AuditLog{}
	var userID sql.NullString
	var metadata []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&log.ID,
		&log.Component,
		&log.Action,
		&userID,
		&metadata,
		&log.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("audit log not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query audit log: %w", err)
	}

	log.UserID = userID
	log.Metadata = metadata

	return log, nil
}

// GetByComponent retrieves audit logs for a specific component since a given time.
func (r *AuditLogRepository) GetByComponent(ctx context.Context, component string, since time.Time) ([]*db.AuditLog, error) {
	query := `
		SELECT id, component, action, user_id, metadata, created_at
		FROM audit_logs
		WHERE component = $1 AND created_at >= $2
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, component, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs by component: %w", err)
	}
	defer rows.Close()

	return r.scanLogs(rows)
}

// GetByUserID retrieves audit logs for a specific user.
func (r *AuditLogRepository) GetByUserID(ctx context.Context, userID string, limit int) ([]*db.AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	query := `
		SELECT id, component, action, user_id, metadata, created_at
		FROM audit_logs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs by user: %w", err)
	}
	defer rows.Close()

	return r.scanLogs(rows)
}

// GetByDateRange retrieves audit logs within a date range.
func (r *AuditLogRepository) GetByDateRange(ctx context.Context, start, end time.Time) ([]*db.AuditLog, error) {
	query := `
		SELECT id, component, action, user_id, metadata, created_at
		FROM audit_logs
		WHERE created_at >= $1 AND created_at <= $2
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs by date range: %w", err)
	}
	defer rows.Close()

	return r.scanLogs(rows)
}

// GetByAction retrieves audit logs by action type.
func (r *AuditLogRepository) GetByAction(ctx context.Context, action string, limit int) ([]*db.AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	query := `
		SELECT id, component, action, user_id, metadata, created_at
		FROM audit_logs
		WHERE action = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, action, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs by action: %w", err)
	}
	defer rows.Close()

	return r.scanLogs(rows)
}

// GetRecent retrieves the most recent audit logs.
func (r *AuditLogRepository) GetRecent(ctx context.Context, limit int) ([]*db.AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	query := `
		SELECT id, component, action, user_id, metadata, created_at
		FROM audit_logs
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent audit logs: %w", err)
	}
	defer rows.Close()

	return r.scanLogs(rows)
}

// GetWithFilters retrieves audit logs with multiple filters.
func (r *AuditLogRepository) GetWithFilters(ctx context.Context, filters AuditLogFilters) ([]*db.AuditLog, error) {
	query := `
		SELECT id, component, action, user_id, metadata, created_at
		FROM audit_logs
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if filters.Component != "" {
		query += fmt.Sprintf(" AND component = $%d", argIdx)
		args = append(args, filters.Component)
		argIdx++
	}

	if filters.Action != "" {
		query += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, filters.Action)
		argIdx++
	}

	if filters.UserID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, filters.UserID)
		argIdx++
	}

	if !filters.Since.IsZero() {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, filters.Since)
		argIdx++
	}

	if !filters.Until.IsZero() {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, filters.Until)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	limit := filters.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}
	query += fmt.Sprintf(" LIMIT $%d", argIdx)
	args = append(args, limit)
	argIdx++

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filters.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs with filters: %w", err)
	}
	defer rows.Close()

	return r.scanLogs(rows)
}

// AuditLogFilters contains filter options for querying audit logs.
type AuditLogFilters struct {
	Component string
	Action    string
	UserID    string
	Since     time.Time
	Until     time.Time
	Limit     int
	Offset    int
}

// scanLogs scans rows into audit log structs.
func (r *AuditLogRepository) scanLogs(rows *sql.Rows) ([]*db.AuditLog, error) {
	var logs []*db.AuditLog

	for rows.Next() {
		log := &db.AuditLog{}
		var userID sql.NullString
		var metadata []byte

		err := rows.Scan(
			&log.ID,
			&log.Component,
			&log.Action,
			&userID,
			&metadata,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		log.UserID = userID
		log.Metadata = metadata

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit logs: %w", err)
	}

	return logs, nil
}

// CountByComponent returns counts of logs grouped by component.
func (r *AuditLogRepository) CountByComponent(ctx context.Context, since time.Time) (map[string]int, error) {
	query := `
		SELECT component, COUNT(*) as count
		FROM audit_logs
		WHERE created_at >= $1
		GROUP BY component
	`

	rows, err := r.db.QueryContext(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to count audit logs: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var component string
		var count int
		if err := rows.Scan(&component, &count); err != nil {
			return nil, fmt.Errorf("failed to scan log count: %w", err)
		}
		counts[component] = count
	}

	return counts, nil
}

// DeleteOlderThan removes audit logs older than the specified time.
func (r *AuditLogRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	query := `DELETE FROM audit_logs WHERE created_at < $1`

	result, err := r.db.ExecContext(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old audit logs: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get deleted count: %w", err)
	}

	return count, nil
}
