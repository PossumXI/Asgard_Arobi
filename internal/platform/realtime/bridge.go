// Package realtime provides NATS-to-WebSocket bridging for real-time events.
package realtime

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/asgard/pandora/internal/platform/observability"
	"github.com/nats-io/nats.go"
)

// EventType represents the type of real-time event.
type EventType string

const (
	EventTypeAlert           EventType = "alert"
	EventTypeThreat          EventType = "threat"
	EventTypeTelemetry       EventType = "telemetry"
	EventTypeSatelliteStatus EventType = "satellite_status"
	EventTypeHunoidStatus    EventType = "hunoid_status"
	EventTypeMissionUpdate   EventType = "mission_update"
	EventTypeStreamUpdate    EventType = "stream_update"
	EventTypeStreamChat      EventType = "stream_chat"
	EventTypeControlCommand  EventType = "control_command"
	EventTypeSecurityFinding EventType = "security_finding"
	EventTypeSystemHealth    EventType = "system_health"
)

// AccessLevel defines user access levels for event subscription.
type AccessLevel string

const (
	AccessLevelPublic       AccessLevel = "public"
	AccessLevelCivilian     AccessLevel = "civilian"
	AccessLevelMilitary     AccessLevel = "military"
	AccessLevelInterstellar AccessLevel = "interstellar"
	AccessLevelGovernment   AccessLevel = "government"
	AccessLevelAdmin        AccessLevel = "admin"
)

// Event represents a real-time event from NATS.
type Event struct {
	ID          string                 `json:"id"`
	Type        EventType              `json:"type"`
	Source      string                 `json:"source"`
	Timestamp   time.Time              `json:"timestamp"`
	Payload     map[string]interface{} `json:"payload"`
	AccessLevel AccessLevel            `json:"access_level"`
	Priority    int                    `json:"priority"`
}

// Bridge connects NATS subjects to WebSocket clients.
type Bridge struct {
	nc            *nats.Conn
	subscriptions []*nats.Subscription
	wsManager     *WebSocketManager
	mu            sync.RWMutex
	running       bool
	ctx           context.Context
	cancel        context.CancelFunc
}

// BridgeConfig holds configuration for the NATS bridge.
type BridgeConfig struct {
	NATSURL          string
	ReconnectWait    time.Duration
	MaxReconnects    int
	PingInterval     time.Duration
	MaxPendingEvents int
}

// DefaultBridgeConfig returns a default configuration.
func DefaultBridgeConfig() BridgeConfig {
	return BridgeConfig{
		NATSURL:          "nats://localhost:4222",
		ReconnectWait:    2 * time.Second,
		MaxReconnects:    60,
		PingInterval:     30 * time.Second,
		MaxPendingEvents: 1000,
	}
}

// NewBridge creates a new NATS-to-WebSocket bridge.
func NewBridge(cfg BridgeConfig, wsManager *WebSocketManager) (*Bridge, error) {
	opts := []nats.Option{
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.PingInterval(cfg.PingInterval),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("[NATS Bridge] Reconnected to %s", nc.ConnectedUrl())
			observability.UpdateNATSConnectionStatus(true)
		}),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				log.Printf("[NATS Bridge] Disconnected: %v", err)
			}
			observability.UpdateNATSConnectionStatus(false)
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			log.Printf("[NATS Bridge] Error: %v", err)
		}),
	}

	nc, err := nats.Connect(cfg.NATSURL, opts...)
	if err != nil {
		observability.UpdateNATSConnectionStatus(false)
		return nil, err
	}
	observability.UpdateNATSConnectionStatus(true)

	ctx, cancel := context.WithCancel(context.Background())

	bridge := &Bridge{
		nc:            nc,
		subscriptions: make([]*nats.Subscription, 0),
		wsManager:     wsManager,
		running:       false,
		ctx:           ctx,
		cancel:        cancel,
	}

	return bridge, nil
}

// Start begins subscribing to NATS subjects and routing events.
func (b *Bridge) Start() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.running {
		return nil
	}

	// Subscribe to all ASGARD event subjects
	subjects := []struct {
		subject     string
		eventType   EventType
		accessLevel AccessLevel
	}{
		// Public events
		{"asgard.alerts.public", EventTypeAlert, AccessLevelPublic},
		{"asgard.streams.update", EventTypeStreamUpdate, AccessLevelPublic},
		{"asgard.system.health", EventTypeSystemHealth, AccessLevelPublic},

		// Civilian events (authenticated users)
		{"asgard.alerts.>", EventTypeAlert, AccessLevelCivilian},
		{"asgard.telemetry.>", EventTypeTelemetry, AccessLevelCivilian},
		{"asgard.satellites.status", EventTypeSatelliteStatus, AccessLevelCivilian},

		// Military events
		{"asgard.military.alerts", EventTypeAlert, AccessLevelMilitary},
		{"asgard.military.missions", EventTypeMissionUpdate, AccessLevelMilitary},
		{"asgard.hunoids.status", EventTypeHunoidStatus, AccessLevelMilitary},

		// Government events
		{"asgard.gov.alerts", EventTypeAlert, AccessLevelGovernment},
		{"asgard.gov.threats", EventTypeThreat, AccessLevelGovernment},
		{"asgard.security.findings", EventTypeSecurityFinding, AccessLevelGovernment},

		// Admin events (all)
		{"asgard.admin.>", EventTypeSystemHealth, AccessLevelAdmin},
	}

	for _, s := range subjects {
		sub, err := b.nc.Subscribe(s.subject, b.createHandler(s.eventType, s.accessLevel))
		if err != nil {
			log.Printf("[NATS Bridge] Failed to subscribe to %s: %v", s.subject, err)
			continue
		}
		b.subscriptions = append(b.subscriptions, sub)
		log.Printf("[NATS Bridge] Subscribed to %s (access: %s)", s.subject, s.accessLevel)
	}

	b.running = true
	log.Println("[NATS Bridge] Started successfully")
	return nil
}

// createHandler creates a NATS message handler for a specific event type.
func (b *Bridge) createHandler(eventType EventType, accessLevel AccessLevel) nats.MsgHandler {
	return func(msg *nats.Msg) {
		start := time.Now()
		var payload map[string]interface{}
		if err := json.Unmarshal(msg.Data, &payload); err != nil {
			log.Printf("[NATS Bridge] Failed to unmarshal message: %v", err)
			return
		}
		observability.GetMetrics().NATSMessagesReceived.WithLabelValues(msg.Subject).Inc()

		event := Event{
			ID:          generateEventID(),
			Type:        eventType,
			Source:      msg.Subject,
			Timestamp:   time.Now().UTC(),
			Payload:     payload,
			AccessLevel: accessLevel,
			Priority:    getPriorityFromPayload(payload),
		}

		// Broadcast to WebSocket clients with appropriate access level
		b.wsManager.Broadcast(event)
		observability.RecordEventProcessed(string(event.Type), event.Source)
		observability.RecordEventLatency(string(event.Type), time.Since(start))
	}
}

// Stop stops the bridge and closes all subscriptions.
func (b *Bridge) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.running {
		return nil
	}

	b.cancel()
	observability.UpdateNATSConnectionStatus(false)

	for _, sub := range b.subscriptions {
		if err := sub.Unsubscribe(); err != nil {
			log.Printf("[NATS Bridge] Error unsubscribing: %v", err)
		}
	}

	b.subscriptions = nil
	b.nc.Close()
	b.running = false

	log.Println("[NATS Bridge] Stopped")
	return nil
}

// Publish publishes an event to NATS.
func (b *Bridge) Publish(subject string, event Event) error {
	data, err := json.Marshal(event.Payload)
	if err != nil {
		return err
	}
	if err := b.nc.Publish(subject, data); err != nil {
		return err
	}
	observability.GetMetrics().NATSMessagesPublished.WithLabelValues(subject).Inc()
	return nil
}

// PublishAlert publishes an alert event.
func (b *Bridge) PublishAlert(alert map[string]interface{}, level AccessLevel) error {
	var subject string
	switch level {
	case AccessLevelPublic:
		subject = "asgard.alerts.public"
	case AccessLevelMilitary:
		subject = "asgard.military.alerts"
	case AccessLevelGovernment:
		subject = "asgard.gov.alerts"
	default:
		subject = "asgard.alerts.general"
	}

	event := Event{
		ID:          generateEventID(),
		Type:        EventTypeAlert,
		Source:      "nysus",
		Timestamp:   time.Now().UTC(),
		Payload:     alert,
		AccessLevel: level,
		Priority:    getPriorityFromPayload(alert),
	}

	return b.Publish(subject, event)
}

// PublishTelemetry publishes telemetry data.
func (b *Bridge) PublishTelemetry(componentID string, telemetry map[string]interface{}) error {
	subject := "asgard.telemetry." + componentID
	event := Event{
		ID:          generateEventID(),
		Type:        EventTypeTelemetry,
		Source:      componentID,
		Timestamp:   time.Now().UTC(),
		Payload:     telemetry,
		AccessLevel: AccessLevelCivilian,
		Priority:    1,
	}

	return b.Publish(subject, event)
}

// PublishThreat publishes a security threat.
func (b *Bridge) PublishThreat(threat map[string]interface{}) error {
	subject := "asgard.gov.threats"
	event := Event{
		ID:          generateEventID(),
		Type:        EventTypeThreat,
		Source:      "giru",
		Timestamp:   time.Now().UTC(),
		Payload:     threat,
		AccessLevel: AccessLevelGovernment,
		Priority:    getPriorityFromPayload(threat),
	}

	return b.Publish(subject, event)
}

// IsConnected returns whether the bridge is connected to NATS.
func (b *Bridge) IsConnected() bool {
	return b.nc != nil && b.nc.IsConnected()
}

// Stats returns connection statistics.
func (b *Bridge) Stats() map[string]interface{} {
	if b.nc == nil {
		return map[string]interface{}{
			"connected": false,
		}
	}

	stats := b.nc.Stats()
	return map[string]interface{}{
		"connected":     b.nc.IsConnected(),
		"reconnects":    stats.Reconnects,
		"in_msgs":       stats.InMsgs,
		"out_msgs":      stats.OutMsgs,
		"in_bytes":      stats.InBytes,
		"out_bytes":     stats.OutBytes,
		"subscriptions": len(b.subscriptions),
	}
}

// Helper functions

func generateEventID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[time.Now().UnixNano()%int64(len(chars))]
		time.Sleep(time.Nanosecond)
	}
	return string(result)
}

func getPriorityFromPayload(payload map[string]interface{}) int {
	if p, ok := payload["priority"].(float64); ok {
		return int(p)
	}
	if p, ok := payload["priority"].(int); ok {
		return p
	}
	if severity, ok := payload["severity"].(string); ok {
		switch severity {
		case "critical":
			return 10
		case "high":
			return 7
		case "medium":
			return 5
		case "low":
			return 2
		}
	}
	return 5 // Default priority
}
