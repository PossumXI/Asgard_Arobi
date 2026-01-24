// Package repositories provides data access layer for database operations.
package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/google/uuid"
)

// NotificationSettingsRepository handles notification settings persistence.
type NotificationSettingsRepository struct {
	db *db.PostgresDB
}

// NewNotificationSettingsRepository creates a new repository.
func NewNotificationSettingsRepository(pgDB *db.PostgresDB) *NotificationSettingsRepository {
	return &NotificationSettingsRepository{db: pgDB}
}

// GetByUserID retrieves settings for a user, creating defaults if missing.
func (r *NotificationSettingsRepository) GetByUserID(userID string) (*db.NotificationSettings, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	query := `
		SELECT user_id, email_alerts, push_notifications, weekly_digest,
		       security_alerts, mission_updates, system_status, updated_at
		FROM user_notification_settings
		WHERE user_id = $1
	`

	settings := &db.NotificationSettings{}
	err = r.db.QueryRow(query, uid).Scan(
		&settings.UserID,
		&settings.EmailAlerts,
		&settings.PushNotifications,
		&settings.WeeklyDigest,
		&settings.SecurityAlerts,
		&settings.MissionUpdates,
		&settings.SystemStatus,
		&settings.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		if _, err := r.db.Exec(`INSERT INTO user_notification_settings (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, uid); err != nil {
			return nil, fmt.Errorf("failed to create default settings: %w", err)
		}
		return r.GetByUserID(userID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query notification settings: %w", err)
	}

	return settings, nil
}

// Upsert updates or creates settings for a user.
func (r *NotificationSettingsRepository) Upsert(settings *db.NotificationSettings) error {
	query := `
		INSERT INTO user_notification_settings (
			user_id, email_alerts, push_notifications, weekly_digest,
			security_alerts, mission_updates, system_status, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET
			email_alerts = EXCLUDED.email_alerts,
			push_notifications = EXCLUDED.push_notifications,
			weekly_digest = EXCLUDED.weekly_digest,
			security_alerts = EXCLUDED.security_alerts,
			mission_updates = EXCLUDED.mission_updates,
			system_status = EXCLUDED.system_status,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.Exec(query,
		settings.UserID,
		settings.EmailAlerts,
		settings.PushNotifications,
		settings.WeeklyDigest,
		settings.SecurityAlerts,
		settings.MissionUpdates,
		settings.SystemStatus,
		time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("failed to upsert notification settings: %w", err)
	}
	return nil
}
