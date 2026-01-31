// Package repositories provides data access layer for database operations.
package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
)

// BaseRepository provides common database operations.
type BaseRepository struct {
	db *db.PostgresDB
}

// NewBaseRepository creates a new base repository.
func NewBaseRepository(pgDB *db.PostgresDB) *BaseRepository {
	return &BaseRepository{db: pgDB}
}

// Query executes a query and returns rows.
func (r *BaseRepository) Query(query string, args ...interface{}) (*sql.Rows, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return r.db.QueryContext(ctx, query, args...)
}

// QueryRow executes a query and returns a single row.
func (r *BaseRepository) QueryRow(query string, args ...interface{}) *sql.Row {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return r.db.QueryRowContext(ctx, query, args...)
}

// Exec executes a query without returning rows.
func (r *BaseRepository) Exec(query string, args ...interface{}) (sql.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return r.db.ExecContext(ctx, query, args...)
}
