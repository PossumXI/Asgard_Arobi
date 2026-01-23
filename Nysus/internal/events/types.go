// Package events defines the event types and structures for Nysus orchestration.
package events

import (
	"time"

	"github.com/google/uuid"
)

// Event represents a system event flowing through Nysus.
type Event struct {
	ID        uuid.UUID   `json:"id"`
	Type      EventType   `json:"type"`
	Source    string      `json:"source"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
	Priority  int         `json:"priority"`
}

// EventType categorizes system events.
type EventType string

const (
	EventTypeAlert         EventType = "alert"
	EventTypeAlertUpdated  EventType = "alert.updated"
	EventTypeTelemetry     EventType = "telemetry"
	EventTypeCommand       EventType = "command"
	EventTypeThreat        EventType = "threat"
	EventTypeThreatMitigated EventType = "threat.mitigated"
	EventTypeMissionStarted EventType = "mission.started"
	EventTypeMissionCompleted EventType = "mission.completed"
	EventTypeHunoidStatus  EventType = "hunoid.status"
	EventTypeSatelliteTelemetry EventType = "satellite.telemetry"
)

// GeoLocation represents geographic coordinates.
type GeoLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude,omitempty"`
}

// AlertEvent represents a Silenus detection alert.
type AlertEvent struct {
	SatelliteID string      `json:"satelliteId"`
	AlertType   string      `json:"alertType"`
	Confidence  float64     `json:"confidence"`
	Location    GeoLocation `json:"location"`
	VideoURL    string      `json:"videoUrl,omitempty"`
}

// TelemetryEvent contains system health data.
type TelemetryEvent struct {
	ComponentID   string             `json:"componentId"`
	ComponentType string             `json:"componentType"` // satellite, hunoid, ground_station
	Metrics       map[string]float64 `json:"metrics"`
	Status        string             `json:"status"`
}

// CommandEvent represents an action command.
type CommandEvent struct {
	TargetID    string                 `json:"targetId"`
	CommandType string                 `json:"commandType"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ThreatEvent represents a security threat detection.
type ThreatEvent struct {
	ThreatType      string `json:"threatType"`
	Severity        string `json:"severity"`
	SourceIP        string `json:"sourceIp,omitempty"`
	TargetComponent string `json:"targetComponent"`
	AttackVector    string `json:"attackVector,omitempty"`
}

// MissionEvent represents mission state changes.
type MissionEvent struct {
	MissionID   string   `json:"missionId"`
	MissionType string   `json:"missionType"`
	Status      string   `json:"status"`
	HunoidIDs   []string `json:"hunoidIds"`
}
