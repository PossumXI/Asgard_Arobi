// Package repositories provides data access layer for database operations.
package repositories

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/google/uuid"
)

// SatelliteRepository handles satellite database operations.
type SatelliteRepository struct {
	db *db.PostgresDB
}

// NewSatelliteRepository creates a new satellite repository.
func NewSatelliteRepository(pgDB *db.PostgresDB) *SatelliteRepository {
	return &SatelliteRepository{db: pgDB}
}

// GetAll retrieves all satellites.
func (r *SatelliteRepository) GetAll() ([]*db.Satellite, error) {
	query := `
		SELECT id, norad_id, name, orbital_elements, hardware_config,
		       current_battery_percent, status, last_telemetry, firmware_version,
		       created_at, updated_at
		FROM satellites
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query satellites: %w", err)
	}
	defer rows.Close()

	var satellites []*db.Satellite
	for rows.Next() {
		sat := &db.Satellite{}
		var noradID sql.NullInt32
		var battery sql.NullFloat64
		var lastTelemetry sql.NullTime
		var firmware sql.NullString
		var orbitalElementsJSON, hardwareConfigJSON []byte

		err := rows.Scan(
			&sat.ID,
			&noradID,
			&sat.Name,
			&orbitalElementsJSON,
			&hardwareConfigJSON,
			&battery,
			&sat.Status,
			&lastTelemetry,
			&firmware,
			&sat.CreatedAt,
			&sat.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan satellite: %w", err)
		}

		sat.NoradID = noradID
		sat.CurrentBatteryPercent = battery
		sat.LastTelemetry = lastTelemetry
		sat.FirmwareVersion = firmware
		sat.OrbitalElements = orbitalElementsJSON
		sat.HardwareConfig = hardwareConfigJSON

		satellites = append(satellites, sat)
	}

	return satellites, nil
}

// GetByID retrieves a satellite by ID.
func (r *SatelliteRepository) GetByID(id string) (*db.Satellite, error) {
	satID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid satellite ID: %w", err)
	}

	query := `
		SELECT id, norad_id, name, orbital_elements, hardware_config,
		       current_battery_percent, status, last_telemetry, firmware_version,
		       created_at, updated_at
		FROM satellites
		WHERE id = $1
	`

	sat := &db.Satellite{}
	var noradID sql.NullInt32
	var battery sql.NullFloat64
	var lastTelemetry sql.NullTime
	var firmware sql.NullString
	var orbitalElementsJSON, hardwareConfigJSON []byte

	err = r.db.QueryRow(query, satID).Scan(
		&sat.ID,
		&noradID,
		&sat.Name,
		&orbitalElementsJSON,
		&hardwareConfigJSON,
		&battery,
		&sat.Status,
		&lastTelemetry,
		&firmware,
		&sat.CreatedAt,
		&sat.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("satellite not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query satellite: %w", err)
	}

	sat.NoradID = noradID
	sat.CurrentBatteryPercent = battery
	sat.LastTelemetry = lastTelemetry
	sat.FirmwareVersion = firmware
	sat.OrbitalElements = orbitalElementsJSON
	sat.HardwareConfig = hardwareConfigJSON

	return sat, nil
}

// GetActiveCount returns the count of active satellites.
func (r *SatelliteRepository) GetActiveCount() (int, error) {
	query := `SELECT COUNT(*) FROM satellites WHERE status = 'operational'`
	var count int
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count satellites: %w", err)
	}
	return count, nil
}

// SatelliteTelemetry represents satellite telemetry fields from the API view.
type SatelliteTelemetry struct {
	BatteryPercent sql.NullFloat64
	Status         string
	LastTelemetry  sql.NullTime
}

// GetTelemetry returns telemetry fields for a satellite.
func (r *SatelliteRepository) GetTelemetry(id string) (*SatelliteTelemetry, error) {
	if r.db == nil {
		return nil, fmt.Errorf("postgres database not configured")
	}

	satID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid satellite ID: %w", err)
	}

	var telemetry SatelliteTelemetry
	err = r.db.QueryRow(`
		SELECT current_battery_percent, status, last_telemetry
		FROM satellites_api
		WHERE id = $1
	`, satID).Scan(&telemetry.BatteryPercent, &telemetry.Status, &telemetry.LastTelemetry)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("satellite not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query satellite telemetry: %w", err)
	}

	return &telemetry, nil
}

// ParseOrbitalElements parses JSONB orbital elements into a map.
func (r *SatelliteRepository) ParseOrbitalElements(data []byte) (map[string]interface{}, error) {
	var elements map[string]interface{}
	if err := json.Unmarshal(data, &elements); err != nil {
		return nil, fmt.Errorf("failed to parse orbital elements: %w", err)
	}
	return elements, nil
}

// ParseHardwareConfig parses JSONB hardware config into a map.
func (r *SatelliteRepository) ParseHardwareConfig(data []byte) (map[string]interface{}, error) {
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse hardware config: %w", err)
	}
	return config, nil
}
