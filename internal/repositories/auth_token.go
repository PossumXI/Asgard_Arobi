// Package repositories provides data access layer for auth token operations.
package repositories

import (
	"fmt"
	"net"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/google/uuid"
)

// AuthTokenRepository handles token revocations and refresh tokens.
type AuthTokenRepository struct {
	db *db.PostgresDB
}

// NewAuthTokenRepository creates a new auth token repository.
func NewAuthTokenRepository(pgDB *db.PostgresDB) *AuthTokenRepository {
	return &AuthTokenRepository{db: pgDB}
}

// RevokeToken marks a JWT token ID as revoked.
func (r *AuthTokenRepository) RevokeToken(tokenID, userID string) error {
	tokenUUID, err := uuid.Parse(tokenID)
	if err != nil {
		return fmt.Errorf("invalid token ID: %w", err)
	}
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	query := `
		INSERT INTO auth_token_revocations (token_id, user_id)
		VALUES ($1, $2)
	`
	_, err = r.db.Exec(query, tokenUUID, userUUID)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	return nil
}

// IsTokenRevoked checks whether a token ID has been revoked.
func (r *AuthTokenRepository) IsTokenRevoked(tokenID string) (bool, error) {
	tokenUUID, err := uuid.Parse(tokenID)
	if err != nil {
		return false, fmt.Errorf("invalid token ID: %w", err)
	}

	query := `
		SELECT COUNT(1)
		FROM auth_token_revocations
		WHERE token_id = $1
	`

	var count int
	if err := r.db.QueryRow(query, tokenUUID).Scan(&count); err != nil {
		return false, fmt.Errorf("failed to check token revocation: %w", err)
	}
	return count > 0, nil
}

// StoreRefreshToken saves a hashed refresh token.
func (r *AuthTokenRepository) StoreRefreshToken(userID, tokenHash string, expiresAt time.Time, userAgent string, ip net.IP) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	query := `
		INSERT INTO auth_refresh_tokens (user_id, token_hash, expires_at, user_agent, ip_address)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err = r.db.Exec(query, userUUID, tokenHash, expiresAt, userAgent, ip)
	if err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}
	return nil
}

// RevokeRefreshToken revokes a refresh token by hash.
func (r *AuthTokenRepository) RevokeRefreshToken(tokenHash string) error {
	query := `
		UPDATE auth_refresh_tokens
		SET revoked_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`
	_, err := r.db.Exec(query, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}
	return nil
}

// GetRefreshTokenUser returns the user ID for a valid refresh token hash.
func (r *AuthTokenRepository) GetRefreshTokenUser(tokenHash string) (string, error) {
	query := `
		SELECT user_id
		FROM auth_refresh_tokens
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`

	var userID uuid.UUID
	if err := r.db.QueryRow(query, tokenHash).Scan(&userID); err != nil {
		return "", fmt.Errorf("refresh token not found: %w", err)
	}
	return userID.String(), nil
}
