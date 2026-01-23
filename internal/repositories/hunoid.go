// Package repositories provides data access layer for database operations.
package repositories

import (
	"database/sql"
	"fmt"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/google/uuid"
)

// HunoidRepository handles hunoid database operations.
type HunoidRepository struct {
	db *db.PostgresDB
}

// NewHunoidRepository creates a new hunoid repository.
func NewHunoidRepository(pgDB *db.PostgresDB) *HunoidRepository {
	return &HunoidRepository{db: pgDB}
}

// GetAll retrieves all hunoids.
func (r *HunoidRepository) GetAll() ([]*db.Hunoid, error) {
	query := `
		SELECT id, serial_number, current_location, current_mission_id,
		       hardware_config, battery_percent, status, vla_model_version,
		       ethical_score, last_telemetry, created_at, updated_at
		FROM hunoids
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query hunoids: %w", err)
	}
	defer rows.Close()

	var hunoids []*db.Hunoid
	for rows.Next() {
		hunoid := &db.Hunoid{}
		var missionID sql.NullString
		var battery sql.NullFloat64
		var vlaModel sql.NullString
		var lastTelemetry sql.NullTime
		var location, hardwareConfig []byte

		err := rows.Scan(
			&hunoid.ID,
			&hunoid.SerialNumber,
			&location,
			&missionID,
			&hardwareConfig,
			&battery,
			&hunoid.Status,
			&vlaModel,
			&hunoid.EthicalScore,
			&lastTelemetry,
			&hunoid.CreatedAt,
			&hunoid.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan hunoid: %w", err)
		}

		if missionID.Valid {
			missionUUID, err := uuid.Parse(missionID.String)
			if err == nil {
				missionIDStr := missionUUID.String()
				hunoid.CurrentMissionID = &missionIDStr
			}
		}
		if battery.Valid {
			hunoid.BatteryPercent = &battery.Float64
		}
		if vlaModel.Valid {
			hunoid.VLAModelVersion = &vlaModel.String
		}
		if lastTelemetry.Valid {
			hunoid.LastTelemetry = &lastTelemetry.Time
		}

		hunoid.CurrentLocation = location
		hunoid.HardwareConfig = hardwareConfig

		hunoids = append(hunoids, hunoid)
	}

	return hunoids, nil
}

// GetByID retrieves a hunoid by ID.
func (r *HunoidRepository) GetByID(id string) (*db.Hunoid, error) {
	hunoidID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid hunoid ID: %w", err)
	}

	query := `
		SELECT id, serial_number, current_location, current_mission_id,
		       hardware_config, battery_percent, status, vla_model_version,
		       ethical_score, last_telemetry, created_at, updated_at
		FROM hunoids
		WHERE id = $1
	`

	hunoid := &db.Hunoid{}
	var missionID sql.NullString
	var battery sql.NullFloat64
	var vlaModel sql.NullString
	var lastTelemetry sql.NullTime
	var location, hardwareConfig []byte

	err = r.db.QueryRow(query, hunoidID).Scan(
		&hunoid.ID,
		&hunoid.SerialNumber,
		&location,
		&missionID,
		&hardwareConfig,
		&battery,
		&hunoid.Status,
		&vlaModel,
		&hunoid.EthicalScore,
		&lastTelemetry,
		&hunoid.CreatedAt,
		&hunoid.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("hunoid not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query hunoid: %w", err)
	}

	if missionID.Valid {
		missionUUID, err := uuid.Parse(missionID.String)
		if err == nil {
			missionIDStr := missionUUID.String()
			hunoid.CurrentMissionID = &missionIDStr
		}
	}
	if battery.Valid {
		hunoid.BatteryPercent = &battery.Float64
	}
	if vlaModel.Valid {
		hunoid.VLAModelVersion = &vlaModel.String
	}
	if lastTelemetry.Valid {
		hunoid.LastTelemetry = &lastTelemetry.Time
	}

	hunoid.CurrentLocation = location
	hunoid.HardwareConfig = hardwareConfig

	return hunoid, nil
}

// GetActiveCount returns the count of active hunoids.
func (r *HunoidRepository) GetActiveCount() (int, error) {
	query := `SELECT COUNT(*) FROM hunoids WHERE status = 'active'`
	var count int
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count hunoids: %w", err)
	}
	return count, nil
}
