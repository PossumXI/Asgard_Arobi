// Package repositories provides data access layer for database operations.
package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/google/uuid"
)

// AccessCodeRepository handles access code database operations.
type AccessCodeRepository struct {
	db *db.PostgresDB
}

// NewAccessCodeRepository creates a new access code repository.
func NewAccessCodeRepository(pgDB *db.PostgresDB) *AccessCodeRepository {
	return &AccessCodeRepository{db: pgDB}
}

// Create inserts a new access code.
func (r *AccessCodeRepository) Create(ctx context.Context, code *db.AccessCode) error {
	query := `
		INSERT INTO access_codes
			(id, code_hash, code_last4, user_id, created_by, clearance_level, scope,
			 issued_at, expires_at, revoked_at, last_used_at, usage_count, max_uses,
			 rotation_interval_hours, next_rotation_at, note)
		VALUES
			($1, $2, $3, $4, $5, $6, $7,
			 $8, $9, $10, $11, $12, $13,
			 $14, $15, $16)
	`

	_, err := r.db.ExecContext(ctx, query,
		code.ID,
		code.CodeHash,
		code.CodeLast4,
		nullUUID(code.UserID),
		nullUUID(code.CreatedBy),
		code.ClearanceLevel,
		code.Scope,
		code.IssuedAt,
		code.ExpiresAt,
		nullTime(code.RevokedAt),
		nullTime(code.LastUsedAt),
		code.UsageCount,
		nullInt32(code.MaxUses),
		code.RotationIntervalHours,
		code.NextRotationAt,
		nullString(code.Note),
	)
	if err != nil {
		return fmt.Errorf("failed to create access code: %w", err)
	}
	return nil
}

// GetActiveForUser returns the active access code for a user.
func (r *AccessCodeRepository) GetActiveForUser(ctx context.Context, userID string) (*db.AccessCode, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}
	query := `
		SELECT id, code_hash, code_last4, user_id, created_by, clearance_level, scope,
		       issued_at, expires_at, revoked_at, last_used_at, usage_count, max_uses,
		       rotation_interval_hours, next_rotation_at, note
		FROM access_codes
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY issued_at DESC
		LIMIT 1
	`
	code := &db.AccessCode{}
	var userIDNull sql.NullString
	var createdBy sql.NullString
	var revokedAt sql.NullTime
	var lastUsedAt sql.NullTime
	var maxUses sql.NullInt32
	var note sql.NullString
	err = r.db.QueryRowContext(ctx, query, id).Scan(
		&code.ID,
		&code.CodeHash,
		&code.CodeLast4,
		&userIDNull,
		&createdBy,
		&code.ClearanceLevel,
		&code.Scope,
		&code.IssuedAt,
		&code.ExpiresAt,
		&revokedAt,
		&lastUsedAt,
		&code.UsageCount,
		&maxUses,
		&code.RotationIntervalHours,
		&code.NextRotationAt,
		&note,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("access code not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query access code: %w", err)
	}
	code.UserID = userIDNull
	code.CreatedBy = createdBy
	code.RevokedAt = revokedAt
	code.LastUsedAt = lastUsedAt
	code.MaxUses = maxUses
	code.Note = note
	return code, nil
}

// GetByHash returns an access code by hash if active.
func (r *AccessCodeRepository) GetByHash(ctx context.Context, codeHash string) (*db.AccessCode, error) {
	query := `
		SELECT id, code_hash, code_last4, user_id, created_by, clearance_level, scope,
		       issued_at, expires_at, revoked_at, last_used_at, usage_count, max_uses,
		       rotation_interval_hours, next_rotation_at, note
		FROM access_codes
		WHERE code_hash = $1
		LIMIT 1
	`
	code := &db.AccessCode{}
	var userIDNull sql.NullString
	var createdBy sql.NullString
	var revokedAt sql.NullTime
	var lastUsedAt sql.NullTime
	var maxUses sql.NullInt32
	var note sql.NullString
	err := r.db.QueryRowContext(ctx, query, codeHash).Scan(
		&code.ID,
		&code.CodeHash,
		&code.CodeLast4,
		&userIDNull,
		&createdBy,
		&code.ClearanceLevel,
		&code.Scope,
		&code.IssuedAt,
		&code.ExpiresAt,
		&revokedAt,
		&lastUsedAt,
		&code.UsageCount,
		&maxUses,
		&code.RotationIntervalHours,
		&code.NextRotationAt,
		&note,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("access code not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query access code: %w", err)
	}
	code.UserID = userIDNull
	code.CreatedBy = createdBy
	code.RevokedAt = revokedAt
	code.LastUsedAt = lastUsedAt
	code.MaxUses = maxUses
	code.Note = note
	return code, nil
}

// List returns access codes with user emails for admin views.
func (r *AccessCodeRepository) List(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 || limit > 200 {
		limit = 200
	}
	query := `
		SELECT ac.id::text, ac.code_last4, ac.clearance_level, ac.scope, ac.issued_at,
		       ac.expires_at, ac.revoked_at, ac.last_used_at, ac.usage_count, ac.max_uses,
		       ac.rotation_interval_hours, ac.next_rotation_at,
		       u.id::text, u.email, u.full_name
		FROM access_codes ac
		LEFT JOIN users u ON u.id = ac.user_id
		ORDER BY ac.issued_at DESC
		LIMIT $1
	`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list access codes: %w", err)
	}
	defer rows.Close()

	results := []map[string]interface{}{}
	for rows.Next() {
		var id, userID, email, fullName sql.NullString
		var last4, clearance, scope string
		var issuedAt, expiresAt, nextRotationAt time.Time
		var revokedAt, lastUsedAt sql.NullTime
		var usageCount int
		var maxUses sql.NullInt32
		var rotationInterval int
		if scanErr := rows.Scan(
			&id,
			&last4,
			&clearance,
			&scope,
			&issuedAt,
			&expiresAt,
			&revokedAt,
			&lastUsedAt,
			&usageCount,
			&maxUses,
			&rotationInterval,
			&nextRotationAt,
			&userID,
			&email,
			&fullName,
		); scanErr != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"id":                    id.String,
			"codeLast4":             last4,
			"clearanceLevel":        clearance,
			"scope":                 scope,
			"issuedAt":              issuedAt.UTC().Format(time.RFC3339),
			"expiresAt":             expiresAt.UTC().Format(time.RFC3339),
			"revokedAt":             timePtrToString(revokedAt),
			"lastUsedAt":            timePtrToString(lastUsedAt),
			"usageCount":            usageCount,
			"maxUses":               int32PtrToValue(maxUses),
			"rotationIntervalHours": rotationInterval,
			"nextRotationAt":        nextRotationAt.UTC().Format(time.RFC3339),
			"userId":                userID.String,
			"userEmail":             email.String,
			"userFullName":          fullName.String,
		})
	}
	return results, nil
}

// MarkUsed increments usage count and updates last_used_at.
func (r *AccessCodeRepository) MarkUsed(ctx context.Context, codeID string) error {
	id, err := uuid.Parse(codeID)
	if err != nil {
		return fmt.Errorf("invalid code ID: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE access_codes
		SET usage_count = usage_count + 1, last_used_at = NOW()
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("failed to update usage: %w", err)
	}
	return nil
}

// Revoke marks a code as revoked.
func (r *AccessCodeRepository) Revoke(ctx context.Context, codeID string) error {
	id, err := uuid.Parse(codeID)
	if err != nil {
		return fmt.Errorf("invalid code ID: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE access_codes
		SET revoked_at = NOW()
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("failed to revoke access code: %w", err)
	}
	return nil
}

// RevokeByUser revokes active codes for a user.
func (r *AccessCodeRepository) RevokeByUser(ctx context.Context, userID string) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE access_codes
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("failed to revoke access codes: %w", err)
	}
	return nil
}

// ListRotationDue returns access codes due for rotation.
func (r *AccessCodeRepository) ListRotationDue(ctx context.Context) ([]db.AccessCode, error) {
	query := `
		SELECT id, code_hash, code_last4, user_id, created_by, clearance_level, scope,
		       issued_at, expires_at, revoked_at, last_used_at, usage_count, max_uses,
		       rotation_interval_hours, next_rotation_at, note
		FROM access_codes
		WHERE revoked_at IS NULL AND next_rotation_at <= NOW()
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query rotation due codes: %w", err)
	}
	defer rows.Close()

	results := []db.AccessCode{}
	for rows.Next() {
		code := db.AccessCode{}
		var userIDNull sql.NullString
		var createdBy sql.NullString
		var revokedAt sql.NullTime
		var lastUsedAt sql.NullTime
		var maxUses sql.NullInt32
		var note sql.NullString
		if scanErr := rows.Scan(
			&code.ID,
			&code.CodeHash,
			&code.CodeLast4,
			&userIDNull,
			&createdBy,
			&code.ClearanceLevel,
			&code.Scope,
			&code.IssuedAt,
			&code.ExpiresAt,
			&revokedAt,
			&lastUsedAt,
			&code.UsageCount,
			&maxUses,
			&code.RotationIntervalHours,
			&code.NextRotationAt,
			&note,
		); scanErr != nil {
			continue
		}
		code.UserID = userIDNull
		code.CreatedBy = createdBy
		code.RevokedAt = revokedAt
		code.LastUsedAt = lastUsedAt
		code.MaxUses = maxUses
		code.Note = note
		results = append(results, code)
	}
	return results, nil
}

// ListActiveUserIDs returns distinct user IDs with active access codes.
func (r *AccessCodeRepository) ListActiveUserIDs(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT user_id::text
		FROM access_codes
		WHERE user_id IS NOT NULL AND revoked_at IS NULL AND expires_at > NOW()
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list active access code users: %w", err)
	}
	defer rows.Close()

	results := []string{}
	for rows.Next() {
		var userID string
		if scanErr := rows.Scan(&userID); scanErr != nil {
			continue
		}
		results = append(results, userID)
	}
	return results, nil
}

// UpdateRotation updates rotation metadata.
func (r *AccessCodeRepository) UpdateRotation(ctx context.Context, codeID string, nextRotation time.Time) error {
	id, err := uuid.Parse(codeID)
	if err != nil {
		return fmt.Errorf("invalid code ID: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE access_codes
		SET next_rotation_at = $2
		WHERE id = $1
	`, id, nextRotation)
	if err != nil {
		return fmt.Errorf("failed to update rotation: %w", err)
	}
	return nil
}

func nullUUID(value sql.NullString) interface{} {
	if value.Valid {
		return value.String
	}
	return nil
}

func nullTime(value sql.NullTime) interface{} {
	if value.Valid {
		return value.Time
	}
	return nil
}

func nullInt32(value sql.NullInt32) interface{} {
	if value.Valid {
		return value.Int32
	}
	return nil
}

func nullString(value sql.NullString) interface{} {
	if value.Valid {
		return value.String
	}
	return nil
}

func timePtrToString(value sql.NullTime) *string {
	if !value.Valid {
		return nil
	}
	formatted := value.Time.UTC().Format(time.RFC3339)
	return &formatted
}

func int32PtrToValue(value sql.NullInt32) *int32 {
	if !value.Valid {
		return nil
	}
	return &value.Int32
}
