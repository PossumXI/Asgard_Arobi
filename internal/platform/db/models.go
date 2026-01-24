package db

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// User represents a system user (from Websites)
type User struct {
	ID               uuid.UUID      `db:"id"`
	Email            string         `db:"email"`
	PasswordHash     string         `db:"password_hash"`
	EmailVerified    bool           `db:"email_verified"`
	EmailVerifiedAt  sql.NullTime   `db:"email_verified_at"`
	FullName         sql.NullString `db:"full_name"`
	SubscriptionTier string         `db:"subscription_tier"`
	IsGovernment     bool           `db:"is_government"`
	CreatedAt        time.Time      `db:"created_at"`
	UpdatedAt        time.Time      `db:"updated_at"`
	LastLogin        sql.NullTime   `db:"last_login"`
}

// NotificationSettings represents user notification preferences.
type NotificationSettings struct {
	UserID            uuid.UUID `db:"user_id"`
	EmailAlerts       bool      `db:"email_alerts"`
	PushNotifications bool      `db:"push_notifications"`
	WeeklyDigest      bool      `db:"weekly_digest"`
	SecurityAlerts    bool      `db:"security_alerts"`
	MissionUpdates    bool      `db:"mission_updates"`
	SystemStatus      bool      `db:"system_status"`
	UpdatedAt         time.Time `db:"updated_at"`
}

// Satellite represents an orbital vehicle (Silenus)
type Satellite struct {
	ID                    uuid.UUID       `db:"id"`
	NoradID               sql.NullInt32   `db:"norad_id"`
	Name                  string          `db:"name"`
	OrbitalElements       []byte          `db:"orbital_elements"` // JSONB
	HardwareConfig        []byte          `db:"hardware_config"`  // JSONB
	CurrentBatteryPercent sql.NullFloat64 `db:"current_battery_percent"`
	Status                string          `db:"status"`
	LastTelemetry         sql.NullTime    `db:"last_telemetry"`
	FirmwareVersion       sql.NullString  `db:"firmware_version"`
	CreatedAt             time.Time       `db:"created_at"`
	UpdatedAt             time.Time       `db:"updated_at"`
}

// Hunoid represents a humanoid robot
type Hunoid struct {
	ID               uuid.UUID       `db:"id"`
	SerialNumber     string          `db:"serial_number"`
	CurrentLocation  []byte          `db:"current_location"` // PostGIS geography
	CurrentMissionID sql.NullString  `db:"current_mission_id"`
	HardwareConfig   []byte          `db:"hardware_config"` // JSONB
	BatteryPercent   sql.NullFloat64 `db:"battery_percent"`
	Status           string          `db:"status"`
	VLAModelVersion  sql.NullString  `db:"vla_model_version"`
	EthicalScore     float64         `db:"ethical_score"`
	LastTelemetry    sql.NullTime    `db:"last_telemetry"`
	CreatedAt        time.Time       `db:"created_at"`
	UpdatedAt        time.Time       `db:"updated_at"`
}

// Mission represents a task assigned to Hunoids
type Mission struct {
	ID                uuid.UUID      `db:"id"`
	MissionType       string         `db:"mission_type"`
	Priority          int            `db:"priority"`
	Status            string         `db:"status"`
	AssignedHunoidIDs []string       `db:"assigned_hunoid_ids"` // Array
	TargetLocation    []byte         `db:"target_location"`     // PostGIS geography
	Description       sql.NullString `db:"description"`
	CreatedBy         sql.NullString `db:"created_by"`
	CreatedAt         time.Time      `db:"created_at"`
	StartedAt         sql.NullTime   `db:"started_at"`
	CompletedAt       sql.NullTime   `db:"completed_at"`
}

// Alert represents a detection from Silenus
type Alert struct {
	ID                uuid.UUID      `db:"id"`
	SatelliteID       sql.NullString `db:"satellite_id"`
	AlertType         string         `db:"alert_type"`
	ConfidenceScore   float64        `db:"confidence_score"`
	DetectionLocation []byte         `db:"detection_location"` // PostGIS geography
	VideoSegmentURL   sql.NullString `db:"video_segment_url"`
	Metadata          []byte         `db:"metadata"` // JSONB
	Status            string         `db:"status"`
	CreatedAt         time.Time      `db:"created_at"`
}

// Threat represents a security incident (Giru)
type Threat struct {
	ID               uuid.UUID      `db:"id"`
	ThreatType       string         `db:"threat_type"`
	Severity         string         `db:"severity"`
	SourceIP         sql.NullString `db:"source_ip"`
	TargetComponent  sql.NullString `db:"target_component"`
	AttackVector     sql.NullString `db:"attack_vector"`
	MitigationAction sql.NullString `db:"mitigation_action"`
	Status           string         `db:"status"`
	DetectedAt       time.Time      `db:"detected_at"`
	ResolvedAt       sql.NullTime   `db:"resolved_at"`
}

// Subscription represents a user's payment subscription
type Subscription struct {
	ID                   uuid.UUID      `db:"id"`
	UserID               uuid.UUID      `db:"user_id"`
	StripeSubscriptionID sql.NullString `db:"stripe_subscription_id"`
	StripeCustomerID     sql.NullString `db:"stripe_customer_id"`
	Tier                 sql.NullString `db:"tier"`
	Status               string         `db:"status"`
	CurrentPeriodStart   sql.NullTime   `db:"current_period_start"`
	CurrentPeriodEnd     sql.NullTime   `db:"current_period_end"`
	CreatedAt            time.Time      `db:"created_at"`
	UpdatedAt            time.Time      `db:"updated_at"`
}

// AuditLog represents system activity tracking
type AuditLog struct {
	ID        int64          `db:"id"`
	Component string         `db:"component"`
	Action    string         `db:"action"`
	UserID    sql.NullString `db:"user_id"`
	Metadata  []byte         `db:"metadata"` // JSONB
	CreatedAt time.Time      `db:"created_at"`
}

// EthicalDecision represents a Hunoid's ethical assessment
type EthicalDecision struct {
	ID                uuid.UUID      `db:"id"`
	HunoidID          uuid.UUID      `db:"hunoid_id"`
	ProposedAction    string         `db:"proposed_action"`
	EthicalAssessment []byte         `db:"ethical_assessment"` // JSONB
	Decision          string         `db:"decision"`
	Reasoning         sql.NullString `db:"reasoning"`
	HumanOverride     bool           `db:"human_override"`
	CreatedAt         time.Time      `db:"created_at"`
}
