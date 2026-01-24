// Package events provides event contracts for security telemetry and response actions.
package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType defines the type of security event.
type EventType string

const (
	EventTypeAlert    EventType = "alert"
	EventTypeFinding  EventType = "finding"
	EventTypeResponse EventType = "response"
	EventTypeIncident EventType = "incident"
	EventTypeAudit    EventType = "audit"
)

// Severity levels for security events.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// SecurityEvent is the base structure for all security events.
type SecurityEvent struct {
	ID            uuid.UUID              `json:"id"`
	CorrelationID uuid.UUID              `json:"correlation_id"`
	Type          EventType              `json:"type"`
	Source        string                 `json:"source"`
	Timestamp     time.Time              `json:"timestamp"`
	Severity      Severity               `json:"severity"`
	Description   string                 `json:"description"`
	Payload       map[string]interface{} `json:"payload"`
	Tags          []string               `json:"tags,omitempty"`
}

// AlertEvent represents a security alert from threat detection.
type AlertEvent struct {
	SecurityEvent
	ThreatType string `json:"threat_type"`
	SourceIP   string `json:"source_ip"`
	TargetIP   string `json:"target_ip,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
	Port       int    `json:"port,omitempty"`
	Confidence float64 `json:"confidence"`
}

// FindingEvent represents a security finding from scanning or analysis.
type FindingEvent struct {
	SecurityEvent
	Category     string   `json:"category"`
	Affected     []string `json:"affected"`
	Remediation  string   `json:"remediation,omitempty"`
	CVE          string   `json:"cve,omitempty"`
	CVSS         float64  `json:"cvss,omitempty"`
}

// ResponseEvent represents a response action taken.
type ResponseEvent struct {
	SecurityEvent
	ThreatID     uuid.UUID `json:"threat_id"`
	ActionType   string    `json:"action_type"`
	ActionTarget string    `json:"action_target"`
	Success      bool      `json:"success"`
	Duration     time.Duration `json:"duration_ns"`
	Automated    bool      `json:"automated"`
}

// IncidentEvent represents a security incident that requires investigation.
type IncidentEvent struct {
	SecurityEvent
	Status       string       `json:"status"`
	AssignedTo   string       `json:"assigned_to,omitempty"`
	RelatedAlerts []uuid.UUID `json:"related_alerts"`
	Resolved     bool        `json:"resolved"`
	ResolutionNote string    `json:"resolution_note,omitempty"`
}

// AuditEvent represents a security audit log entry.
type AuditEvent struct {
	SecurityEvent
	Actor      string `json:"actor"`
	Action     string `json:"action"`
	Resource   string `json:"resource"`
	Outcome    string `json:"outcome"`
	ClientIP   string `json:"client_ip,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
}

// NewSecurityEvent creates a new security event with required fields.
func NewSecurityEvent(eventType EventType, source string, severity Severity, description string) SecurityEvent {
	return SecurityEvent{
		ID:            uuid.New(),
		CorrelationID: uuid.New(),
		Type:          eventType,
		Source:        source,
		Timestamp:     time.Now().UTC(),
		Severity:      severity,
		Description:   description,
		Payload:       make(map[string]interface{}),
	}
}

// NewAlertEvent creates a new alert event.
func NewAlertEvent(source, threatType, sourceIP string, severity Severity, confidence float64, description string) AlertEvent {
	base := NewSecurityEvent(EventTypeAlert, source, severity, description)
	return AlertEvent{
		SecurityEvent: base,
		ThreatType:    threatType,
		SourceIP:      sourceIP,
		Confidence:    confidence,
	}
}

// NewFindingEvent creates a new finding event.
func NewFindingEvent(source, category string, severity Severity, affected []string, description string) FindingEvent {
	base := NewSecurityEvent(EventTypeFinding, source, severity, description)
	return FindingEvent{
		SecurityEvent: base,
		Category:      category,
		Affected:      affected,
	}
}

// NewResponseEvent creates a new response event.
func NewResponseEvent(source string, threatID uuid.UUID, actionType, target string, success bool, duration time.Duration) ResponseEvent {
	severity := SeverityInfo
	if !success {
		severity = SeverityMedium
	}
	base := NewSecurityEvent(EventTypeResponse, source, severity, "Response action: "+actionType)
	return ResponseEvent{
		SecurityEvent: base,
		ThreatID:      threatID,
		ActionType:    actionType,
		ActionTarget:  target,
		Success:       success,
		Duration:      duration,
		Automated:     true,
	}
}

// ToJSON serializes the event to JSON.
func (e *SecurityEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// ToMap converts the event to a map for NATS publishing.
func (e *SecurityEvent) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":             e.ID.String(),
		"correlation_id": e.CorrelationID.String(),
		"type":           string(e.Type),
		"source":         e.Source,
		"timestamp":      e.Timestamp.Format(time.RFC3339),
		"severity":       string(e.Severity),
		"description":    e.Description,
		"payload":        e.Payload,
		"tags":           e.Tags,
	}
}

// AlertToMap converts an alert event to a map.
func (e *AlertEvent) ToMap() map[string]interface{} {
	m := e.SecurityEvent.ToMap()
	m["threat_type"] = e.ThreatType
	m["source_ip"] = e.SourceIP
	m["target_ip"] = e.TargetIP
	m["protocol"] = e.Protocol
	m["port"] = e.Port
	m["confidence"] = e.Confidence
	return m
}

// ResponseToMap converts a response event to a map.
func (e *ResponseEvent) ToMap() map[string]interface{} {
	m := e.SecurityEvent.ToMap()
	m["threat_id"] = e.ThreatID.String()
	m["action_type"] = e.ActionType
	m["action_target"] = e.ActionTarget
	m["success"] = e.Success
	m["duration_ns"] = e.Duration.Nanoseconds()
	m["automated"] = e.Automated
	return m
}

// NATS subject constants for security events.
const (
	SubjectSecurityAlerts    = "asgard.security.alerts"
	SubjectSecurityFindings  = "asgard.security.findings"
	SubjectSecurityResponses = "asgard.security.responses"
	SubjectSecurityIncidents = "asgard.security.incidents"
	SubjectSecurityAudit     = "asgard.security.audit"
	SubjectGovThreats        = "asgard.gov.threats"
)

// GetSubjectForEvent returns the appropriate NATS subject for an event type.
func GetSubjectForEvent(eventType EventType, severity Severity) string {
	switch eventType {
	case EventTypeAlert:
		if severity == SeverityCritical || severity == SeverityHigh {
			return SubjectGovThreats
		}
		return SubjectSecurityAlerts
	case EventTypeFinding:
		return SubjectSecurityFindings
	case EventTypeResponse:
		return SubjectSecurityResponses
	case EventTypeIncident:
		return SubjectSecurityIncidents
	case EventTypeAudit:
		return SubjectSecurityAudit
	default:
		return SubjectSecurityAlerts
	}
}
