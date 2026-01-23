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
		SELECT id, email, password_hash, full_name, subscription_tier, 
		       is_government, created_at, updated_at, last_login
		FROM users
		WHERE id = $1
	`

	user := &db.User{}
	var fullName sql.NullString
	var lastLogin sql.NullTime

	err = r.db.QueryRow(query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
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

	if fullName.Valid {
		user.FullName = &fullName.String
	}
	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}

	return user, nil
}

// GetByEmail retrieves a user by email.
func (r *UserRepository) GetByEmail(email string) (*db.User, error) {
	query := `
		SELECT id, email, password_hash, full_name, subscription_tier, 
		       is_government, created_at, updated_at, last_login
		FROM users
		WHERE email = $1
	`

	user := &db.User{}
	var fullName sql.NullString
	var lastLogin sql.NullTime

	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
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

	if fullName.Valid {
		user.FullName = &fullName.String
	}
	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}

	return user, nil
}

// Create creates a new user.
func (r *UserRepository) Create(user *db.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, full_name, subscription_tier, 
		                  is_government, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	var fullName interface{}
	if user.FullName != nil {
		fullName = *user.FullName
	} else {
		fullName = nil
	}

	_, err := r.db.Exec(query,
		user.ID,
		user.Email,
		user.PasswordHash,
		fullName,
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
		SET email = $2, password_hash = $3, full_name = $4, subscription_tier = $5,
		    is_government = $6, updated_at = $7, last_login = $8
		WHERE id = $1
	`

	var fullName interface{}
	if user.FullName != nil {
		fullName = *user.FullName
	} else {
		fullName = nil
	}

	var lastLogin interface{}
	if user.LastLogin != nil {
		lastLogin = *user.LastLogin
	} else {
		lastLogin = nil
	}

	_, err := r.db.Exec(query,
		user.ID,
		user.Email,
		user.PasswordHash,
		fullName,
		user.SubscriptionTier,
		user.IsGovernment,
		time.Now(),
		lastLogin,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}
