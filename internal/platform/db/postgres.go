package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type PostgresDB struct {
	*sql.DB
}

func NewPostgresDB(cfg *Config) (*PostgresDB, error) {
	db, err := sql.Open("postgres", cfg.PostgresDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres connection: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &PostgresDB{DB: db}, nil
}

func (db *PostgresDB) Health(ctx context.Context) error {
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("postgres health check failed: %w", err)
	}
	return nil
}

func (db *PostgresDB) Close() error {
	if err := db.DB.Close(); err != nil {
		return fmt.Errorf("failed to close postgres connection: %w", err)
	}
	return nil
}
