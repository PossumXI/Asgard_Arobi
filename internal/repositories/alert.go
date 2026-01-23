// Package repositories provides data access layer for database operations.
package repositories

import (
	"database/sql"
	"fmt"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/google/uuid"
)

// AlertRepository handles alert database operations.
type AlertRepository struct {
	db *db.PostgresDB
}

// NewAlertRepository creates a new alert repository.
func NewAlertRepository(pgDB *db.PostgresDB) *AlertRepository {
	return &AlertRepository{db: pgDB}
}

// GetAll retrieves all alerts.
func (r *AlertRepository) GetAll() ([]*db.Alert, error) {
	query := `
		SELECT id, satellite_id, alert_type, confidence_score, detection_location,
		       video_segment_url, metadata, status, created_at
		FROM alerts
		ORDER BY created_at DESC
		LIMIT 100
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query alerts: %w", err)
	}
	defer rows.Close()

	var alerts []*db.Alert
	for rows.Next() {
		alert := &db.Alert{}
		var satelliteID sql.NullString
		var videoURL sql.NullString
		var location, metadata []byte

		err := rows.Scan(
			&alert.ID,
			&satelliteID,
			&alert.AlertType,
			&alert.ConfidenceScore,
			&location,
			&videoURL,
			&metadata,
			&alert.Status,
			&alert.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}

		if satelliteID.Valid {
			alert.SatelliteID = &satelliteID.String
		}
		if videoURL.Valid {
			alert.VideoSegmentURL = &videoURL.String
		}

		alert.DetectionLocation = location
		alert.Metadata = metadata

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// GetByID retrieves an alert by ID.
func (r *AlertRepository) GetByID(id string) (*db.Alert, error) {
	alertID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid alert ID: %w", err)
	}

	query := `
		SELECT id, satellite_id, alert_type, confidence_score, detection_location,
		       video_segment_url, metadata, status, created_at
		FROM alerts
		WHERE id = $1
	`

	alert := &db.Alert{}
	var satelliteID sql.NullString
	var videoURL sql.NullString
	var location, metadata []byte

	err = r.db.QueryRow(query, alertID).Scan(
		&alert.ID,
		&satelliteID,
		&alert.AlertType,
		&alert.ConfidenceScore,
		&location,
		&videoURL,
		&metadata,
		&alert.Status,
		&alert.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("alert not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query alert: %w", err)
	}

	if satelliteID.Valid {
		alert.SatelliteID = &satelliteID.String
	}
	if videoURL.Valid {
		alert.VideoSegmentURL = &videoURL.String
	}

	alert.DetectionLocation = location
	alert.Metadata = metadata

	return alert, nil
}

// GetPendingCount returns the count of pending alerts.
func (r *AlertRepository) GetPendingCount() (int, error) {
	query := `SELECT COUNT(*) FROM alerts WHERE status = 'new'`
	var count int
	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count alerts: %w", err)
	}
	return count, nil
}
