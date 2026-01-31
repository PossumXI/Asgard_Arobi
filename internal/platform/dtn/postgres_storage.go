package dtn

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/asgard/pandora/pkg/bundle"
	"github.com/google/uuid"
)

// PostgresBundleStorage implements persistent bundle storage using PostgreSQL.
type PostgresBundleStorage struct {
	db *db.PostgresDB
}

// NewPostgresBundleStorage creates a new PostgreSQL-backed bundle storage.
func NewPostgresBundleStorage(pgDB *db.PostgresDB) (*PostgresBundleStorage, error) {
	storage := &PostgresBundleStorage{db: pgDB}

	// Create table if it doesn't exist
	if err := storage.createTable(); err != nil {
		return nil, fmt.Errorf("failed to create bundle storage table: %w", err)
	}

	return storage, nil
}

// createTable creates the bundles table if it doesn't exist.
func (s *PostgresBundleStorage) createTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS dtn_bundles (
			id UUID PRIMARY KEY,
			version INTEGER NOT NULL,
			bundle_flags INTEGER NOT NULL,
			destination_eid TEXT NOT NULL,
			source_eid TEXT NOT NULL,
			report_to TEXT,
			creation_timestamp BIGINT NOT NULL,
			lifetime BIGINT NOT NULL,
			payload BYTEA NOT NULL,
			crc_type INTEGER NOT NULL,
			previous_node TEXT,
			hop_count INTEGER NOT NULL DEFAULT 0,
			priority INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'pending',
			stored_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
		
		CREATE INDEX IF NOT EXISTS idx_bundles_destination ON dtn_bundles(destination_eid);
		CREATE INDEX IF NOT EXISTS idx_bundles_source ON dtn_bundles(source_eid);
		CREATE INDEX IF NOT EXISTS idx_bundles_status ON dtn_bundles(status);
		CREATE INDEX IF NOT EXISTS idx_bundles_priority ON dtn_bundles(priority DESC);
		CREATE INDEX IF NOT EXISTS idx_bundles_stored_at ON dtn_bundles(stored_at);
	`

	_, err := s.db.Exec(query)
	return err
}

// Store persists a bundle to PostgreSQL.
func (s *PostgresBundleStorage) Store(ctx context.Context, b *bundle.Bundle) error {
	if err := b.Validate(); err != nil {
		return fmt.Errorf("invalid bundle: %w", err)
	}

	query := `
		INSERT INTO dtn_bundles (
			id, version, bundle_flags, destination_eid, source_eid,
			report_to, creation_timestamp, lifetime, payload, crc_type,
			previous_node, hop_count, priority, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			updated_at = NOW()
	`

	_, err := s.db.ExecContext(ctx, query,
		b.ID,
		b.Version,
		b.BundleFlags,
		b.DestinationEID,
		b.SourceEID,
		b.ReportTo,
		b.CreationTimestamp,
		b.Lifetime,
		b.Payload,
		b.CRCType,
		b.PreviousNode,
		b.HopCount,
		b.Priority,
		StatusPending,
	)

	return err
}

// Retrieve fetches a bundle by ID.
func (s *PostgresBundleStorage) Retrieve(ctx context.Context, id uuid.UUID) (*bundle.Bundle, error) {
	query := `
		SELECT id, version, bundle_flags, destination_eid, source_eid,
		       report_to, creation_timestamp, lifetime, payload, crc_type,
		       previous_node, hop_count, priority
		FROM dtn_bundles
		WHERE id = $1
	`

	var b bundle.Bundle
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&b.ID,
		&b.Version,
		&b.BundleFlags,
		&b.DestinationEID,
		&b.SourceEID,
		&b.ReportTo,
		&b.CreationTimestamp,
		&b.Lifetime,
		&b.Payload,
		&b.CRCType,
		&b.PreviousNode,
		&b.HopCount,
		&b.Priority,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("bundle not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve bundle: %w", err)
	}

	return &b, nil
}

// Delete removes a bundle from storage.
func (s *PostgresBundleStorage) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM dtn_bundles WHERE id = $1`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete bundle: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("bundle not found: %s", id)
	}

	return nil
}

// List returns bundles matching the filter criteria.
func (s *PostgresBundleStorage) List(ctx context.Context, filter BundleFilter) ([]*bundle.Bundle, error) {
	query := `SELECT id, version, bundle_flags, destination_eid, source_eid,
	                 report_to, creation_timestamp, lifetime, payload, crc_type,
	                 previous_node, hop_count, priority
	          FROM dtn_bundles
	          WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if filter.DestinationEID != "" {
		query += fmt.Sprintf(" AND destination_eid = $%d", argIndex)
		args = append(args, filter.DestinationEID)
		argIndex++
	}

	if filter.SourceEID != "" {
		query += fmt.Sprintf(" AND source_eid = $%d", argIndex)
		args = append(args, filter.SourceEID)
		argIndex++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, string(filter.Status))
		argIndex++
	}

	if filter.MinPriority > 0 {
		query += fmt.Sprintf(" AND priority >= $%d", argIndex)
		args = append(args, filter.MinPriority)
		argIndex++
	}

	if filter.MaxAge > 0 {
		query += fmt.Sprintf(" AND stored_at > NOW() - INTERVAL '%d seconds'", int(filter.MaxAge.Seconds()))
	}

	// Order by
	switch filter.OrderBy {
	case "priority":
		query += " ORDER BY priority DESC, stored_at ASC"
	case "age":
		query += " ORDER BY stored_at ASC"
	case "size":
		query += " ORDER BY LENGTH(payload) ASC"
	default:
		query += " ORDER BY priority DESC, stored_at ASC"
	}

	// Limit
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list bundles: %w", err)
	}
	defer rows.Close()

	var bundles []*bundle.Bundle
	for rows.Next() {
		var b bundle.Bundle
		if err := rows.Scan(
			&b.ID,
			&b.Version,
			&b.BundleFlags,
			&b.DestinationEID,
			&b.SourceEID,
			&b.ReportTo,
			&b.CreationTimestamp,
			&b.Lifetime,
			&b.Payload,
			&b.CRCType,
			&b.PreviousNode,
			&b.HopCount,
			&b.Priority,
		); err != nil {
			return nil, fmt.Errorf("failed to scan bundle: %w", err)
		}
		bundles = append(bundles, &b)
	}

	return bundles, rows.Err()
}

// UpdateStatus changes the status of a bundle.
func (s *PostgresBundleStorage) UpdateStatus(ctx context.Context, id uuid.UUID, status BundleStatus) error {
	query := `UPDATE dtn_bundles SET status = $1, updated_at = NOW() WHERE id = $2`
	result, err := s.db.ExecContext(ctx, query, string(status), id)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("bundle not found: %s", id)
	}

	return nil
}

// GetStatus returns the current status of a bundle.
func (s *PostgresBundleStorage) GetStatus(ctx context.Context, id uuid.UUID) (BundleStatus, error) {
	query := `SELECT status FROM dtn_bundles WHERE id = $1`
	var status string
	err := s.db.QueryRowContext(ctx, query, id).Scan(&status)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("bundle not found: %s", id)
	}
	if err != nil {
		return "", fmt.Errorf("failed to get status: %w", err)
	}

	return BundleStatus(status), nil
}

// Count returns the total number of bundles in storage.
func (s *PostgresBundleStorage) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM dtn_bundles`
	var count int
	err := s.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

// PurgeExpired removes all expired bundles.
func (s *PostgresBundleStorage) PurgeExpired(ctx context.Context) (int, error) {
	query := `
		DELETE FROM dtn_bundles
		WHERE (creation_timestamp + lifetime) < EXTRACT(EPOCH FROM NOW()) * 1000
		RETURNING id
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to purge expired bundles: %w", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		count++
	}

	return count, rows.Err()
}
