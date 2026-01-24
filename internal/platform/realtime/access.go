// Package realtime provides access control rules for real-time events.
package realtime

import (
	"sync"
)

// AccessRules defines which event types are accessible at each access level.
type AccessRules struct {
	rules map[AccessLevel][]EventType
	mu    sync.RWMutex
}

// NewAccessRules creates default access rules.
func NewAccessRules() *AccessRules {
	ar := &AccessRules{
		rules: make(map[AccessLevel][]EventType),
	}
	ar.initializeDefaults()
	return ar
}

// initializeDefaults sets up the default access control rules.
func (ar *AccessRules) initializeDefaults() {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	// Public: Limited to general streams and public alerts
	ar.rules[AccessLevelPublic] = []EventType{
		EventTypeStreamUpdate,
		EventTypeStreamChat,
		EventTypeSystemHealth,
	}

	// Civilian: Authenticated users can see satellite status and general telemetry
	ar.rules[AccessLevelCivilian] = []EventType{
		EventTypeStreamUpdate,
		EventTypeStreamChat,
		EventTypeSystemHealth,
		EventTypeAlert,
		EventTypeTelemetry,
		EventTypeSatelliteStatus,
	}

	// Military: Access to hunoid and mission data
	ar.rules[AccessLevelMilitary] = []EventType{
		EventTypeStreamUpdate,
		EventTypeStreamChat,
		EventTypeSystemHealth,
		EventTypeAlert,
		EventTypeTelemetry,
		EventTypeSatelliteStatus,
		EventTypeHunoidStatus,
		EventTypeMissionUpdate,
	}

	// Interstellar: Commander tier - access to interstellar streams and advanced telemetry
	ar.rules[AccessLevelInterstellar] = []EventType{
		EventTypeStreamUpdate,
		EventTypeStreamChat,
		EventTypeSystemHealth,
		EventTypeAlert,
		EventTypeTelemetry,
		EventTypeSatelliteStatus,
		EventTypeHunoidStatus,
		EventTypeMissionUpdate,
	}

	// Government: Full access including security events
	ar.rules[AccessLevelGovernment] = []EventType{
		EventTypeStreamUpdate,
		EventTypeStreamChat,
		EventTypeSystemHealth,
		EventTypeAlert,
		EventTypeTelemetry,
		EventTypeSatelliteStatus,
		EventTypeHunoidStatus,
		EventTypeMissionUpdate,
		EventTypeControlCommand,
		EventTypeThreat,
		EventTypeSecurityFinding,
	}

	// Admin: All event types
	ar.rules[AccessLevelAdmin] = []EventType{
		EventTypeStreamUpdate,
		EventTypeStreamChat,
		EventTypeSystemHealth,
		EventTypeAlert,
		EventTypeTelemetry,
		EventTypeSatelliteStatus,
		EventTypeHunoidStatus,
		EventTypeMissionUpdate,
		EventTypeControlCommand,
		EventTypeThreat,
		EventTypeSecurityFinding,
	}
}

// CanAccess checks if a given access level can access an event type.
func (ar *AccessRules) CanAccess(level AccessLevel, eventType EventType) bool {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	allowedTypes, ok := ar.rules[level]
	if !ok {
		return false
	}

	for _, t := range allowedTypes {
		if t == eventType {
			return true
		}
	}
	return false
}

// GetAllowedEventTypes returns all event types accessible at a given level.
func (ar *AccessRules) GetAllowedEventTypes(level AccessLevel) []EventType {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	types, ok := ar.rules[level]
	if !ok {
		return nil
	}

	// Return a copy to prevent modification
	result := make([]EventType, len(types))
	copy(result, types)
	return result
}

// SubjectChannels defines the mapping between NATS subjects and access levels.
type SubjectChannels struct {
	channels map[string]AccessLevel
	mu       sync.RWMutex
}

// NewSubjectChannels creates default subject channel mappings.
func NewSubjectChannels() *SubjectChannels {
	sc := &SubjectChannels{
		channels: make(map[string]AccessLevel),
	}
	sc.initializeDefaults()
	return sc
}

// initializeDefaults sets up the default subject channel mappings.
func (sc *SubjectChannels) initializeDefaults() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Public channels
	sc.channels["asgard.alerts.public"] = AccessLevelPublic
	sc.channels["asgard.streams.update"] = AccessLevelPublic
	sc.channels["asgard.system.health"] = AccessLevelPublic

	// Civilian channels
	sc.channels["asgard.alerts.general"] = AccessLevelCivilian
	sc.channels["asgard.telemetry.*"] = AccessLevelCivilian
	sc.channels["asgard.satellites.status"] = AccessLevelCivilian

	// Military channels
	sc.channels["asgard.military.alerts"] = AccessLevelMilitary
	sc.channels["asgard.military.missions"] = AccessLevelMilitary
	sc.channels["asgard.hunoids.status"] = AccessLevelMilitary

	// Government channels
	sc.channels["asgard.gov.alerts"] = AccessLevelGovernment
	sc.channels["asgard.gov.threats"] = AccessLevelGovernment
	sc.channels["asgard.security.findings"] = AccessLevelGovernment

	// Admin channels
	sc.channels["asgard.admin.*"] = AccessLevelAdmin
}

// GetRequiredLevel returns the required access level for a subject.
func (sc *SubjectChannels) GetRequiredLevel(subject string) AccessLevel {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// Direct match
	if level, ok := sc.channels[subject]; ok {
		return level
	}

	// Wildcard matching (simplified)
	for pattern, level := range sc.channels {
		if matchesPattern(pattern, subject) {
			return level
		}
	}

	return AccessLevelAdmin // Default to highest level for unknown subjects
}

// matchesPattern checks if a subject matches a wildcard pattern.
func matchesPattern(pattern, subject string) bool {
	if pattern == subject {
		return true
	}

	// Handle * wildcard at end
	if len(pattern) > 2 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(subject) >= len(prefix) && subject[:len(prefix)] == prefix
	}

	// Handle > wildcard (matches any remaining segments)
	if len(pattern) > 2 && pattern[len(pattern)-1] == '>' {
		prefix := pattern[:len(pattern)-1]
		return len(subject) >= len(prefix) && subject[:len(prefix)] == prefix
	}

	return false
}

// AccessLevelFromString parses an access level from a string.
func AccessLevelFromString(s string) AccessLevel {
	switch s {
	case "public":
		return AccessLevelPublic
	case "civilian":
		return AccessLevelCivilian
	case "military":
		return AccessLevelMilitary
	case "government":
		return AccessLevelGovernment
	case "admin":
		return AccessLevelAdmin
	default:
		return AccessLevelPublic
	}
}

// AccessLevelFromUserRole maps a user role to an access level.
func AccessLevelFromUserRole(role string, isGovernment bool) AccessLevel {
	if role == "admin" {
		return AccessLevelAdmin
	}
	if isGovernment {
		return AccessLevelGovernment
	}
	switch role {
	case "interstellar":
		return AccessLevelInterstellar
	case "military":
		return AccessLevelMilitary
	case "subscriber", "user", "civilian":
		return AccessLevelCivilian
	default:
		return AccessLevelPublic
	}
}
