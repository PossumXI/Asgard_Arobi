// Package repositories provides data access layer for database operations.
package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/google/uuid"
)

// ThreatRepository handles threat database operations.
type ThreatRepository struct {
	db *db.PostgresDB
}

// NewThreatRepository creates a new threat repository.
func NewThreatRepository(pgDB *db.PostgresDB) *ThreatRepository {
	return &ThreatRepository{db: pgDB}
}

// GetTodayCount returns the count of threats detected today.
func (r *ThreatRepository) GetTodayCount() (int, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	query := `SELECT COUNT(*) FROM threats WHERE detected_at >= $1`
	var count int
	err := r.db.QueryRow(query, today).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count threats: %w", err)
	}
	return count, nil
}

// GetByID retrieves a threat by ID.
func (r *ThreatRepository) GetByID(id string) (*db.Threat, error) {
	threatID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid threat ID: %w", err)
	}

	query := `
		SELECT id, threat_type, severity, source_ip, target_component,
		       attack_vector, mitigation_action, status, detected_at, resolved_at
		FROM threats
		WHERE id = $1
	`

	threat := &db.Threat{}
	var sourceIP sql.NullString
	var targetComponent sql.NullString
	var attackVector sql.NullString
	var mitigationAction sql.NullString
	var resolvedAt sql.NullTime

	err = r.db.QueryRow(query, threatID).Scan(
		&threat.ID,
		&threat.ThreatType,
		&threat.Severity,
		&sourceIP,
		&targetComponent,
		&attackVector,
		&mitigationAction,
		&threat.Status,
		&threat.DetectedAt,
		&resolvedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("threat not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query threat: %w", err)
	}

	if sourceIP.Valid {
		threat.SourceIP = &sourceIP.String
	}
	if targetComponent.Valid {
		threat.TargetComponent = &targetComponent.String
	}
	if attackVector.Valid {
		threat.AttackVector = &attackVector.String
	}
	if mitigationAction.Valid {
		threat.MitigationAction = &mitigationAction.String
	}
	if resolvedAt.Valid {
		threat.ResolvedAt = &resolvedAt.Time
	}

	return threat, nil
}
