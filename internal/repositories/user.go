// Package repositories provides data access layer for database operations.
package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/google/uuid"
)

// UserRepository handles user database operations.
type UserRepository struct {
	db *db.PostgresDB
}

// NewUserRepository creates a new user repository.
func NewUserRepository(pgDB *db.PostgresDB) *UserRepository {
	return &UserRepository{db: pgDB}
}

// GetByID retrieves a user by ID.
func (r *UserRepository) GetByID(id string) (*db.User, error) {
	userID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	query := `
		SELECT id, email, password_hash, email_verified, email_verified_at, full_name, 
		       subscription_tier, is_government, created_at, updated_at, last_login
		FROM users
		WHERE id = $1
	`

	user := &db.User{}
	var fullName sql.NullString
	var lastLogin sql.NullTime
	var emailVerifiedAt sql.NullTime

	err = r.db.QueryRow(query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.EmailVerified,
		&emailVerifiedAt,
		&fullName,
		&user.SubscriptionTier,
		&user.IsGovernment,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLogin,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	user.FullName = fullName
	user.LastLogin = lastLogin
	user.EmailVerifiedAt = emailVerifiedAt

	return user, nil
}

// GetByEmail retrieves a user by email.
func (r *UserRepository) GetByEmail(email string) (*db.User, error) {
	query := `
		SELECT id, email, password_hash, email_verified, email_verified_at, full_name, 
		       subscription_tier, is_government, created_at, updated_at, last_login
		FROM users
		WHERE email = $1
	`

	user := &db.User{}
	var fullName sql.NullString
	var lastLogin sql.NullTime
	var emailVerifiedAt sql.NullTime

	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.EmailVerified,
		&emailVerifiedAt,
		&fullName,
		&user.SubscriptionTier,
		&user.IsGovernment,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLogin,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	user.FullName = fullName
	user.LastLogin = lastLogin
	user.EmailVerifiedAt = emailVerifiedAt

	return user, nil
}

// Create creates a new user.
func (r *UserRepository) Create(user *db.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, full_name, subscription_tier, 
		                  is_government, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.Exec(query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.FullName,
		user.SubscriptionTier,
		user.IsGovernment,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// Update updates an existing user.
func (r *UserRepository) Update(user *db.User) error {
	query := `
		UPDATE users
		SET email = $2, password_hash = $3, email_verified = $4, email_verified_at = $5,
		    full_name = $6, subscription_tier = $7, is_government = $8, updated_at = $9, last_login = $10
		WHERE id = $1
	`

	_, err := r.db.Exec(query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.EmailVerified,
		user.EmailVerifiedAt,
		user.FullName,
		user.SubscriptionTier,
		user.IsGovernment,
		time.Now(),
		user.LastLogin,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// SetEmailVerified sets the email verification status for a user.
func (r *UserRepository) SetEmailVerified(userID string, verified bool) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	var query string
	if verified {
		query = `
			UPDATE users
			SET email_verified = true, email_verified_at = NOW(), updated_at = NOW()
			WHERE id = $1
		`
	} else {
		query = `
			UPDATE users
			SET email_verified = false, email_verified_at = NULL, updated_at = NOW()
			WHERE id = $1
		`
	}

	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to update email verification status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// ListUsers returns a list of recent users for admin views.
func (r *UserRepository) ListUsers(limit int) ([]*db.User, error) {
	if limit <= 0 || limit > 200 {
		limit = 200
	}

	query := `
		SELECT id, email, password_hash, email_verified, email_verified_at, full_name,
		       subscription_tier, is_government, created_at, updated_at, last_login
		FROM users
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	users := make([]*db.User, 0, limit)
	for rows.Next() {
		user := &db.User{}
		var fullName sql.NullString
		var lastLogin sql.NullTime
		var emailVerifiedAt sql.NullTime

		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&user.EmailVerified,
			&emailVerifiedAt,
			&fullName,
			&user.SubscriptionTier,
			&user.IsGovernment,
			&user.CreatedAt,
			&user.UpdatedAt,
			&lastLogin,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		user.FullName = fullName
		user.LastLogin = lastLogin
		user.EmailVerifiedAt = emailVerifiedAt
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate users: %w", err)
	}

	return users, nil
}
