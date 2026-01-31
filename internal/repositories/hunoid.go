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

		hunoid.CurrentMissionID = missionID
		hunoid.BatteryPercent = battery
		hunoid.VLAModelVersion = vlaModel
		hunoid.LastTelemetry = lastTelemetry
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

	hunoid.CurrentMissionID = missionID
	hunoid.BatteryPercent = battery
	hunoid.VLAModelVersion = vlaModel
	hunoid.LastTelemetry = lastTelemetry
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

// GetLocation returns the hunoid's current location if available.
func (r *HunoidRepository) GetLocation(id string) (*GeoLocation, error) {
	if r.db == nil {
		return nil, fmt.Errorf("postgres database not configured")
	}

	hunoidID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid hunoid ID: %w", err)
	}

	var lat sql.NullFloat64
	var lon sql.NullFloat64
	var alt sql.NullFloat64
	err = r.db.QueryRow(`
		SELECT latitude, longitude, altitude
		FROM hunoids_api
		WHERE id = $1
	`, hunoidID).Scan(&lat, &lon, &alt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("hunoid not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query hunoid location: %w", err)
	}

	if !lat.Valid || !lon.Valid {
		return nil, nil
	}

	location := &GeoLocation{
		Latitude:  lat.Float64,
		Longitude: lon.Float64,
	}
	if alt.Valid {
		location.Altitude = alt.Float64
	}

	return location, nil
}

// HunoidTelemetry represents hunoid telemetry fields from the API view.
type HunoidTelemetry struct {
	BatteryPercent sql.NullFloat64
	Status         string
	LastTelemetry  sql.NullTime
	Latitude       sql.NullFloat64
	Longitude      sql.NullFloat64
	Altitude       sql.NullFloat64
}

// GetTelemetry returns telemetry fields for a hunoid.
func (r *HunoidRepository) GetTelemetry(id string) (*HunoidTelemetry, error) {
	if r.db == nil {
		return nil, fmt.Errorf("postgres database not configured")
	}

	hunoidID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid hunoid ID: %w", err)
	}

	var telemetry HunoidTelemetry
	err = r.db.QueryRow(`
		SELECT battery_percent, status, last_telemetry, latitude, longitude, altitude
		FROM hunoids_api
		WHERE id = $1
	`, hunoidID).Scan(
		&telemetry.BatteryPercent,
		&telemetry.Status,
		&telemetry.LastTelemetry,
		&telemetry.Latitude,
		&telemetry.Longitude,
		&telemetry.Altitude,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("hunoid not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query hunoid telemetry: %w", err)
	}

	return &telemetry, nil
}
