package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/google/uuid"
)

// EmailTokenRepository handles email verification and password reset tokens.
type EmailTokenRepository struct {
	db *db.PostgresDB
}

// NewEmailTokenRepository creates a new email token repository.
func NewEmailTokenRepository(pgDB *db.PostgresDB) *EmailTokenRepository {
	return &EmailTokenRepository{db: pgDB}
}

// StoreVerificationToken stores an email verification token.
func (r *EmailTokenRepository) StoreVerificationToken(userID uuid.UUID, token string, expiresAt time.Time) error {
	query := `
		INSERT INTO email_verification_tokens (user_id, token, expires_at)
		VALUES ($1, $2, $3)
	`
	_, err := r.db.Exec(query, userID, token, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to store verification token: %w", err)
	}
	return nil
}

// VerifyVerificationToken verifies and marks an email verification token as used.
func (r *EmailTokenRepository) VerifyVerificationToken(token string) (uuid.UUID, error) {
	var userID uuid.UUID
	var expiresAt time.Time
	var usedAt sql.NullTime

	query := `
		SELECT user_id, expires_at, used_at
		FROM email_verification_tokens
		WHERE token = $1
	`
	err := r.db.QueryRow(query, token).Scan(&userID, &expiresAt, &usedAt)
	if err == sql.ErrNoRows {
		return uuid.Nil, fmt.Errorf("invalid verification token")
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to query verification token: %w", err)
	}

	if usedAt.Valid {
		return uuid.Nil, fmt.Errorf("verification token already used")
	}

	if time.Now().After(expiresAt) {
		return uuid.Nil, fmt.Errorf("verification token expired")
	}

	// Mark as used
	updateQuery := `UPDATE email_verification_tokens SET used_at = NOW() WHERE token = $1`
	_, err = r.db.Exec(updateQuery, token)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to mark token as used: %w", err)
	}

	return userID, nil
}

// StorePasswordResetToken stores a password reset token.
func (r *EmailTokenRepository) StorePasswordResetToken(userID uuid.UUID, token string, expiresAt time.Time) error {
	query := `
		INSERT INTO password_reset_tokens (user_id, token, expires_at)
		VALUES ($1, $2, $3)
	`
	_, err := r.db.Exec(query, userID, token, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to store password reset token: %w", err)
	}
	return nil
}

// VerifyPasswordResetToken verifies and marks a password reset token as used.
func (r *EmailTokenRepository) VerifyPasswordResetToken(token string) (uuid.UUID, error) {
	var userID uuid.UUID
	var expiresAt time.Time
	var usedAt sql.NullTime

	query := `
		SELECT user_id, expires_at, used_at
		FROM password_reset_tokens
		WHERE token = $1
	`
	err := r.db.QueryRow(query, token).Scan(&userID, &expiresAt, &usedAt)
	if err == sql.ErrNoRows {
		return uuid.Nil, fmt.Errorf("invalid reset token")
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to query reset token: %w", err)
	}

	if usedAt.Valid {
		return uuid.Nil, fmt.Errorf("reset token already used")
	}

	if time.Now().After(expiresAt) {
		return uuid.Nil, fmt.Errorf("reset token expired")
	}

	// Mark as used
	updateQuery := `UPDATE password_reset_tokens SET used_at = NOW() WHERE token = $1`
	_, err = r.db.Exec(updateQuery, token)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to mark token as used: %w", err)
	}

	return userID, nil
}
