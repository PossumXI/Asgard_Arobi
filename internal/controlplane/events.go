// Package controlplane provides a unified control plane for coordinating DTN, security, and autonomy systems.
package controlplane

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventDomain represents the source domain of a cross-domain event.
type EventDomain string

const (
	DomainSecurity EventDomain = "security"
	DomainDTN      EventDomain = "dtn"
	DomainAutonomy EventDomain = "autonomy"
	DomainEthics   EventDomain = "ethics"
	DomainControl  EventDomain = "controlplane"
)

// CrossDomainEventType categorizes cross-domain events.
type CrossDomainEventType string

const (
	// Security events
	EventSecurityThreat     CrossDomainEventType = "security.threat"
	EventSecurityMitigated  CrossDomainEventType = "security.mitigated"
	EventSecurityEscalation CrossDomainEventType = "security.escalation"

	// DTN events
	EventDTNBundleReceived  CrossDomainEventType = "dtn.bundle.received"
	EventDTNBundleForwarded CrossDomainEventType = "dtn.bundle.forwarded"
	EventDTNCongestion      CrossDomainEventType = "dtn.congestion"
	EventDTNLinkChange      CrossDomainEventType = "dtn.link.change"

	// Autonomy events
	EventAutonomyStatus       CrossDomainEventType = "autonomy.status"
	EventAutonomyMissionStart CrossDomainEventType = "autonomy.mission.start"
	EventAutonomyMissionEnd   CrossDomainEventType = "autonomy.mission.end"
	EventAutonomyHalted       CrossDomainEventType = "autonomy.halted"
	EventAutonomyResumed      CrossDomainEventType = "autonomy.resumed"

	// Ethics events
	EventEthicsDecision   CrossDomainEventType = "ethics.decision"
	EventEthicsEscalation CrossDomainEventType = "ethics.escalation"
	EventEthicsOverride   CrossDomainEventType = "ethics.override"

	// Control plane events
	EventControlCommand  CrossDomainEventType = "control.command"
	EventControlResponse CrossDomainEventType = "control.response"
	EventControlAlert    CrossDomainEventType = "control.alert"
)

// Severity levels for cross-domain events.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// CrossDomainEvent is the base structure for all cross-domain events.
type CrossDomainEvent struct {
	ID            uuid.UUID              `json:"id"`
	CorrelationID uuid.UUID              `json:"correlation_id"`
	Type          CrossDomainEventType   `json:"type"`
	Domain        EventDomain            `json:"domain"`
	Source        string                 `json:"source"`
	Timestamp     time.Time              `json:"timestamp"`
	Severity      Severity               `json:"severity"`
	Description   string                 `json:"description"`
	Payload       map[string]interface{} `json:"payload"`
	Tags          []string               `json:"tags,omitempty"`
	RequiresAck   bool                   `json:"requires_ack"`
}

// SecurityThreatEvent represents a security threat that requires cross-domain attention.
type SecurityThreatEvent struct {
	CrossDomainEvent
	ThreatType        string   `json:"threat_type"`
	ThreatID          string   `json:"threat_id"`
	SourceIP          string   `json:"source_ip,omitempty"`
	TargetSystem      string   `json:"target_system"`
	Confidence        float64  `json:"confidence"`
	AffectedNodes     []string `json:"affected_nodes"`
	RecommendedAction string   `json:"recommended_action"`
}

// DTNBundleEvent represents significant DTN bundle routing events.
type DTNBundleEvent struct {
	CrossDomainEvent
	BundleID       string        `json:"bundle_id"`
	SourceEID      string        `json:"source_eid"`
	DestinationEID string        `json:"destination_eid"`
	Priority       uint8         `json:"priority"`
	Size           int64         `json:"size"`
	HopCount       int           `json:"hop_count"`
	Latency        time.Duration `json:"latency_ns"`
	NodeID         string        `json:"node_id"`
}

// DTNCongestionEvent represents DTN network congestion.
type DTNCongestionEvent struct {
	CrossDomainEvent
	NodeID           string  `json:"node_id"`
	QueueUtilization float64 `json:"queue_utilization"` // 0.0 to 1.0
	BundlesQueued    int     `json:"bundles_queued"`
	DropRate         float64 `json:"drop_rate"`
	Recommendation   string  `json:"recommendation"`
}

// AutonomyStatusEvent represents status updates from autonomous systems.
type AutonomyStatusEvent struct {
	CrossDomainEvent
	SystemID      string             `json:"system_id"`
	SystemType    AutonomySystemType `json:"system_type"`
	State         AutonomyState      `json:"state"`
	MissionID     string             `json:"mission_id,omitempty"`
	Location      *GeoLocation       `json:"location,omitempty"`
	BatteryLevel  float64            `json:"battery_level"` // 0-100
	HealthStatus  string             `json:"health_status"`
	LastHeartbeat time.Time          `json:"last_heartbeat"`
}

// AutonomySystemType identifies the type of autonomous system.
type AutonomySystemType string

const (
	SystemTypeSilenus   AutonomySystemType = "silenus"   // Satellite vision system
	SystemTypeHunoid    AutonomySystemType = "hunoid"    // Humanoid robot
	SystemTypeSatellite AutonomySystemType = "satellite" // General satellite
	SystemTypeDrone     AutonomySystemType = "drone"     // Autonomous drone
)

// AutonomyState represents the operational state of an autonomous system.
type AutonomyState string

const (
	StateIdle        AutonomyState = "idle"
	StateActive      AutonomyState = "active"
	StatePaused      AutonomyState = "paused"
	StateEmergency   AutonomyState = "emergency"
	StateMaintenance AutonomyState = "maintenance"
	StateOffline     AutonomyState = "offline"
)

// EthicsDecisionEvent represents a decision from the ethics kernel.
type EthicsDecisionEvent struct {
	CrossDomainEvent
	DecisionID    uuid.UUID     `json:"decision_id"`
	ActionType    string        `json:"action_type"`
	SystemID      string        `json:"system_id"`
	Decision      string        `json:"decision"` // approved, rejected, escalated
	Reasoning     string        `json:"reasoning"`
	RulesChecked  []string      `json:"rules_checked"`
	Score         float64       `json:"score"`
	RequiresHuman bool          `json:"requires_human"`
	Timeout       time.Duration `json:"timeout_ns,omitempty"`
}

// GeoLocation represents geographic coordinates.
type GeoLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude,omitempty"`
}

// ControlCommand represents a cross-domain control command.
type ControlCommand struct {
	ID           uuid.UUID              `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	Source       string                 `json:"source"` // Who issued the command
	TargetDomain EventDomain            `json:"target_domain"`
	TargetSystem string                 `json:"target_system,omitempty"`
	CommandType  string                 `json:"command_type"`
	Parameters   map[string]interface{} `json:"parameters"`
	Priority     int                    `json:"priority"` // 0-10, higher is more urgent
	ExpiresAt    time.Time              `json:"expires_at,omitempty"`
}

// ControlResponse represents the response to a control command.
type ControlResponse struct {
	CommandID  uuid.UUID              `json:"command_id"`
	Success    bool                   `json:"success"`
	Error      string                 `json:"error,omitempty"`
	Result     map[string]interface{} `json:"result,omitempty"`
	ExecutedAt time.Time              `json:"executed_at"`
	Duration   time.Duration          `json:"duration_ns"`
}

// NewCrossDomainEvent creates a new cross-domain event with required fields.
func NewCrossDomainEvent(
	eventType CrossDomainEventType,
	domain EventDomain,
	source string,
	severity Severity,
	description string,
) CrossDomainEvent {
	return CrossDomainEvent{
		ID:            uuid.New(),
		CorrelationID: uuid.New(),
		Type:          eventType,
		Domain:        domain,
		Source:        source,
		Timestamp:     time.Now().UTC(),
		Severity:      severity,
		Description:   description,
		Payload:       make(map[string]interface{}),
	}
}

// NewSecurityThreatEvent creates a new security threat event.
func NewSecurityThreatEvent(
	source string,
	threatType string,
	targetSystem string,
	severity Severity,
	confidence float64,
) SecurityThreatEvent {
	base := NewCrossDomainEvent(
		EventSecurityThreat,
		DomainSecurity,
		source,
		severity,
		"Security threat detected: "+threatType,
	)
	base.RequiresAck = severity == SeverityCritical || severity == SeverityHigh

	return SecurityThreatEvent{
		CrossDomainEvent: base,
		ThreatType:       threatType,
		ThreatID:         uuid.New().String(),
		TargetSystem:     targetSystem,
		Confidence:       confidence,
		AffectedNodes:    make([]string, 0),
	}
}

// NewDTNBundleEvent creates a new DTN bundle event.
func NewDTNBundleEvent(
	eventType CrossDomainEventType,
	nodeID string,
	bundleID string,
	sourceEID string,
	destEID string,
	priority uint8,
) DTNBundleEvent {
	base := NewCrossDomainEvent(
		eventType,
		DomainDTN,
		"dtn://"+nodeID,
		SeverityInfo,
		"DTN bundle event",
	)

	return DTNBundleEvent{
		CrossDomainEvent: base,
		BundleID:         bundleID,
		SourceEID:        sourceEID,
		DestinationEID:   destEID,
		Priority:         priority,
		NodeID:           nodeID,
	}
}

// NewDTNCongestionEvent creates a new DTN congestion event.
func NewDTNCongestionEvent(
	nodeID string,
	queueUtilization float64,
	bundlesQueued int,
) DTNCongestionEvent {
	severity := SeverityInfo
	if queueUtilization > 0.9 {
		severity = SeverityCritical
	} else if queueUtilization > 0.75 {
		severity = SeverityHigh
	} else if queueUtilization > 0.5 {
		severity = SeverityMedium
	}

	base := NewCrossDomainEvent(
		EventDTNCongestion,
		DomainDTN,
		"dtn://"+nodeID,
		severity,
		"DTN node experiencing congestion",
	)
	base.RequiresAck = severity == SeverityCritical

	return DTNCongestionEvent{
		CrossDomainEvent: base,
		NodeID:           nodeID,
		QueueUtilization: queueUtilization,
		BundlesQueued:    bundlesQueued,
	}
}

// NewAutonomyStatusEvent creates a new autonomy status event.
func NewAutonomyStatusEvent(
	systemID string,
	systemType AutonomySystemType,
	state AutonomyState,
) AutonomyStatusEvent {
	severity := SeverityInfo
	if state == StateEmergency {
		severity = SeverityCritical
	} else if state == StateOffline {
		severity = SeverityHigh
	}

	base := NewCrossDomainEvent(
		EventAutonomyStatus,
		DomainAutonomy,
		systemID,
		severity,
		"Autonomy system status update",
	)

	return AutonomyStatusEvent{
		CrossDomainEvent: base,
		SystemID:         systemID,
		SystemType:       systemType,
		State:            state,
		HealthStatus:     "nominal",
		LastHeartbeat:    time.Now().UTC(),
	}
}

// NewEthicsDecisionEvent creates a new ethics decision event.
func NewEthicsDecisionEvent(
	systemID string,
	decision string,
	reasoning string,
	requiresHuman bool,
) EthicsDecisionEvent {
	severity := SeverityInfo
	eventType := EventEthicsDecision
	if requiresHuman {
		severity = SeverityHigh
		eventType = EventEthicsEscalation
	}
	if decision == "rejected" {
		severity = SeverityMedium
	}

	base := NewCrossDomainEvent(
		eventType,
		DomainEthics,
		"ethics-kernel",
		severity,
		"Ethics kernel decision: "+decision,
	)
	base.RequiresAck = requiresHuman

	return EthicsDecisionEvent{
		CrossDomainEvent: base,
		DecisionID:       uuid.New(),
		SystemID:         systemID,
		Decision:         decision,
		Reasoning:        reasoning,
		RequiresHuman:    requiresHuman,
		RulesChecked:     make([]string, 0),
	}
}

// NewControlCommand creates a new control command.
func NewControlCommand(
	source string,
	targetDomain EventDomain,
	commandType string,
	params map[string]interface{},
	priority int,
) ControlCommand {
	return ControlCommand{
		ID:           uuid.New(),
		Timestamp:    time.Now().UTC(),
		Source:       source,
		TargetDomain: targetDomain,
		CommandType:  commandType,
		Parameters:   params,
		Priority:     priority,
	}
}

// ToJSON serializes the event to JSON.
func (e *CrossDomainEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// ToMap converts the event to a map.
func (e *CrossDomainEvent) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":             e.ID.String(),
		"correlation_id": e.CorrelationID.String(),
		"type":           string(e.Type),
		"domain":         string(e.Domain),
		"source":         e.Source,
		"timestamp":      e.Timestamp.Format(time.RFC3339Nano),
		"severity":       string(e.Severity),
		"description":    e.Description,
		"payload":        e.Payload,
		"tags":           e.Tags,
		"requires_ack":   e.RequiresAck,
	}
}

// EventHandler processes cross-domain events.
type EventHandler func(event CrossDomainEvent) error

// EventFilter determines if an event should be processed.
type EventFilter func(event CrossDomainEvent) bool

// FilterByDomain creates a filter for events from a specific domain.
func FilterByDomain(domain EventDomain) EventFilter {
	return func(event CrossDomainEvent) bool {
		return event.Domain == domain
	}
}

// FilterBySeverity creates a filter for events at or above a severity level.
func FilterBySeverity(minSeverity Severity) EventFilter {
	severityOrder := map[Severity]int{
		SeverityInfo:     0,
		SeverityLow:      1,
		SeverityMedium:   2,
		SeverityHigh:     3,
		SeverityCritical: 4,
	}

	return func(event CrossDomainEvent) bool {
		return severityOrder[event.Severity] >= severityOrder[minSeverity]
	}
}

// FilterByType creates a filter for specific event types.
func FilterByType(types ...CrossDomainEventType) EventFilter {
	typeSet := make(map[CrossDomainEventType]bool)
	for _, t := range types {
		typeSet[t] = true
	}

	return func(event CrossDomainEvent) bool {
		return typeSet[event.Type]
	}
}
