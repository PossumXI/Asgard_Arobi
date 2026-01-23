// Package repositories provides data access layer for database operations.
package repositories

import (
	"database/sql"
	"fmt"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// MissionRepository handles mission database operations.
type MissionRepository struct {
	db *db.PostgresDB
}

// NewMissionRepository creates a new mission repository.
func NewMissionRepository(pgDB *db.PostgresDB) *MissionRepository {
	return &MissionRepository{db: pgDB}
}

// GetAll retrieves all missions.
func (r *MissionRepository) GetAll() ([]*db.Mission, error) {
	query := `
		SELECT id, mission_type, priority, status, assigned_hunoid_ids,
		       target_location, description, created_by, created_at,
		       started_at, completed_at
		FROM missions
		ORDER BY priority DESC, created_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query missions: %w", err)
	}
	defer rows.Close()

	var missions []*db.Mission
	for rows.Next() {
		mission := &db.Mission{}
		var description sql.NullString
		var createdBy sql.NullString
		var startedAt sql.NullTime
		var completedAt sql.NullTime
		var targetLocation []byte
		var hunoidIDs pq.StringArray

		err := rows.Scan(
			&mission.ID,
			&mission.MissionType,
			&mission.Priority,
			&mission.Status,
			&hunoidIDs,
			&targetLocation,
			&description,
			&createdBy,
			&mission.CreatedAt,
			&startedAt,
			&completedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan mission: %w", err)
		}

		if description.Valid {
			mission.Description = &description.String
		}
		if createdBy.Valid {
			mission.CreatedBy = &createdBy.String
		}
		if startedAt.Valid {
			mission.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			mission.CompletedAt = &completedAt.Time
		}

		mission.AssignedHunoidIDs = []string(hunoidIDs)
		mission.TargetLocation = targetLocation

		missions = append(missions, mission)
	}

	return missions, nil
}

// GetByID retrieves a mission by ID.
func (r *MissionRepository) GetByID(id string) (*db.Mission, error) {
	missionID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid mission ID: %w", err)
	}

	query := `
		SELECT id, mission_type, priority, status, assigned_hunoid_ids,
		       target_location, description, created_by, created_at,
		       started_at, completed_at
		FROM missions
		WHERE id = $1
	`

	mission := &db.Mission{}
	var description sql.NullString
	var createdBy sql.NullString
	var startedAt sql.NullTime
	var completedAt sql.NullTime
	var targetLocation []byte
	var hunoidIDs pq.StringArray

	err = r.db.QueryRow(query, missionID).Scan(
		&mission.ID,
		&mission.MissionType,
		&mission.Priority,
		&mission.Status,
		&hunoidIDs,
		&targetLocation,
		&description,
		&createdBy,
		&mission.CreatedAt,
		&startedAt,
		&completedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("mission not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query mission: %w", err)
	}

	if description.Valid {
		mission.Description = &description.String
	}
	if createdBy.Valid {
		mission.CreatedBy = &createdBy.String
	}
	if startedAt.Valid {
		mission.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		mission.CompletedAt = &completedAt.Time
	}

	mission.AssignedHunoidIDs = []string(hunoidIDs)
	mission.TargetLocation = targetLocation

	return mission, nil
}

// GetActiveCount returns the count of active missions.
func (r *MissionRepository) GetActiveCount() (int, error) {
	query := `SELECT COUNT(*) FROM missions WHERE status = 'active'`
	var count int
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count missions: %w", err)
	}
	return count, nil
}
