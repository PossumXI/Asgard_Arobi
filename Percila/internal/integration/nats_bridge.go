// Package integration provides NATS real-time event integration for PERCILA.
package integration

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// ============================================================================
// PERCILA NATS EVENT TYPES
// ============================================================================

// PercilaEventType represents PERCILA-specific event types.
type PercilaEventType string

const (
	// Inbound events (subscribed from ASGARD systems)
	EventTypeThreatDetected     PercilaEventType = "threat_detected"
	EventTypeSatellitePosition  PercilaEventType = "satellite_position"
	EventTypeTelemetryUpdate    PercilaEventType = "telemetry_update"
	EventTypeHunoidState        PercilaEventType = "hunoid_state"
	EventTypeMissionAssigned    PercilaEventType = "mission_assigned"
	EventTypeContactWindow      PercilaEventType = "contact_window"
	EventTypeWeatherUpdate      PercilaEventType = "weather_update"
	EventTypeNoFlyZoneUpdate    PercilaEventType = "no_fly_zone_update"

	// Outbound events (published by PERCILA)
	EventTypeTrajectoryUpdate   PercilaEventType = "trajectory_update"
	EventTypeMissionUpdate      PercilaEventType = "percila_mission_update"
	EventTypeThreatAlert        PercilaEventType = "percila_threat_alert"
	EventTypeGuidanceCommand    PercilaEventType = "guidance_command"
	EventTypePayloadStatus      PercilaEventType = "payload_status"
	EventTypeEvasiveManeuver    PercilaEventType = "evasive_maneuver"
	EventTypeArrivalEstimate    PercilaEventType = "arrival_estimate"
	EventTypeRouteDeviation     PercilaEventType = "route_deviation"
)

// ============================================================================
// NATS SUBJECTS
// ============================================================================

const (
	// ASGARD inbound subjects (PERCILA subscribes to these)
	SubjectGiruThreats          = "asgard.giru.threats"
	SubjectGiruThreatZones      = "asgard.giru.threat_zones"
	SubjectSilenusPositions     = "asgard.silenus.positions"
	SubjectSilenusAlerts        = "asgard.silenus.alerts"
	SubjectSatNetTelemetry      = "asgard.satnet.telemetry.>"
	SubjectSatNetContactWindows = "asgard.satnet.contact_windows"
	SubjectHunoidStates         = "asgard.hunoid.states"
	SubjectNysusMissions        = "asgard.nysus.missions"
	SubjectWeatherUpdates       = "asgard.weather.updates"
	SubjectNoFlyZones           = "asgard.airspace.no_fly_zones"

	// PERCILA outbound subjects (PERCILA publishes to these)
	SubjectPercilaTrajectory    = "asgard.percila.trajectory"
	SubjectPercilaMission       = "asgard.percila.mission"
	SubjectPercilaThreatAlert   = "asgard.percila.threat_alert"
	SubjectPercilaGuidance      = "asgard.percila.guidance"
	SubjectPercilaPayloadStatus = "asgard.percila.payload_status"
	SubjectPercilaEvasion       = "asgard.percila.evasion"
	SubjectPercilaArrival       = "asgard.percila.arrival"
	SubjectPercilaDeviation     = "asgard.percila.deviation"
)

// ============================================================================
// PERCILA EVENT STRUCTURES
// ============================================================================

// PercilaEvent represents a PERCILA real-time event.
type PercilaEvent struct {
	ID        string                 `json:"id"`
	Type      PercilaEventType       `json:"type"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
	Priority  int                    `json:"priority"` // 1-10, 10 being highest
	PayloadID string                 `json:"payloadId,omitempty"`
	MissionID string                 `json:"missionId,omitempty"`
}

// TrajectoryUpdateEvent contains trajectory change information.
type TrajectoryUpdateEvent struct {
	PayloadID    string     `json:"payloadId"`
	MissionID    string     `json:"missionId"`
	OldWaypoints []Waypoint `json:"oldWaypoints,omitempty"`
	NewWaypoints []Waypoint `json:"newWaypoints"`
	Reason       string     `json:"reason"`
	EstimatedETA time.Time  `json:"estimatedEta"`
	Timestamp    time.Time  `json:"timestamp"`
}

// ThreatAlertEvent contains threat detection information from PERCILA.
type ThreatAlertEvent struct {
	PayloadID      string   `json:"payloadId"`
	ThreatID       string   `json:"threatId"`
	ThreatType     string   `json:"threatType"`
	ThreatLocation GeoCoord `json:"threatLocation"`
	DistanceKm     float64  `json:"distanceKm"`
	Severity       string   `json:"severity"` // low, medium, high, critical
	Action         string   `json:"action"`   // evade, monitor, proceed
	Timestamp      time.Time `json:"timestamp"`
}

// GuidanceCommandEvent contains guidance commands for payloads.
type GuidanceCommandEvent struct {
	PayloadID   string                 `json:"payloadId"`
	CommandType string                 `json:"commandType"`
	Parameters  map[string]interface{} `json:"parameters"`
	Priority    int                    `json:"priority"`
	ValidUntil  time.Time              `json:"validUntil"`
	Timestamp   time.Time              `json:"timestamp"`
}

// PayloadStatusEvent contains payload status updates.
type PayloadStatusEvent struct {
	PayloadID       string   `json:"payloadId"`
	Position        Vector3D `json:"position"`
	Velocity        Vector3D `json:"velocity"`
	Heading         float64  `json:"heading"`
	Altitude        float64  `json:"altitude"`
	Speed           float64  `json:"speed"`
	Fuel            float64  `json:"fuel"`
	Battery         float64  `json:"battery"`
	Status          string   `json:"status"`
	CurrentWaypoint int      `json:"currentWaypoint"`
	ETA             time.Time `json:"eta"`
	Timestamp       time.Time `json:"timestamp"`
}

// EvasiveManeuverEvent contains evasive action details.
type EvasiveManeuverEvent struct {
	PayloadID    string   `json:"payloadId"`
	ManeuverType string   `json:"maneuverType"` // terrain_following, altitude_change, route_deviation
	ThreatID     string   `json:"threatId,omitempty"`
	OldPosition  Vector3D `json:"oldPosition"`
	NewPosition  Vector3D `json:"newPosition"`
	Duration     int      `json:"durationSeconds"`
	Timestamp    time.Time `json:"timestamp"`
}

// ============================================================================
// NATS BRIDGE CONFIGURATION
// ============================================================================

// NATSBridgeConfig holds configuration for the PERCILA NATS bridge.
type NATSBridgeConfig struct {
	NATSURL           string        `json:"natsUrl"`
	ClusterID         string        `json:"clusterId"`
	ClientID          string        `json:"clientId"`
	ReconnectWait     time.Duration `json:"reconnectWait"`
	MaxReconnects     int           `json:"maxReconnects"`
	PingInterval      time.Duration `json:"pingInterval"`
	MaxPendingEvents  int           `json:"maxPendingEvents"`
	EventBufferSize   int           `json:"eventBufferSize"`
	EnableCompression bool          `json:"enableCompression"`
}

// DefaultNATSBridgeConfig returns a default configuration for PERCILA NATS bridge.
func DefaultNATSBridgeConfig() NATSBridgeConfig {
	return NATSBridgeConfig{
		NATSURL:           "nats://localhost:4222",
		ClusterID:         "asgard-cluster",
		ClientID:          "percila-" + generateShortID(),
		ReconnectWait:     2 * time.Second,
		MaxReconnects:     -1, // Unlimited reconnects
		PingInterval:      30 * time.Second,
		MaxPendingEvents:  1000,
		EventBufferSize:   500,
		EnableCompression: true,
	}
}

// ============================================================================
// NATS BRIDGE IMPLEMENTATION
// ============================================================================

// NATSBridge provides NATS real-time event integration for PERCILA.
type NATSBridge struct {
	mu            sync.RWMutex
	nc            *nats.Conn
	subscriptions []*nats.Subscription
	config        NATSBridgeConfig
	running       bool
	ctx           context.Context
	cancel        context.CancelFunc

	// Event handlers
	threatHandler          func(Threat)
	satelliteHandler       func(SatellitePosition)
	telemetryHandler       func(Telemetry)
	hunoidHandler          func(HunoidState)
	missionHandler         func(Mission)
	contactWindowHandler   func(ContactWindow)
	threatZoneHandler      func(ThreatZone)

	// Event channels for internal consumption
	threatChan    chan Threat
	positionChan  chan SatellitePosition
	telemetryChan chan Telemetry

	// Statistics
	stats BridgeStats
}

// BridgeStats contains NATS bridge statistics.
type BridgeStats struct {
	mu               sync.RWMutex
	MessagesReceived int64
	MessagesSent     int64
	Reconnects       int64
	Errors           int64
	LastError        string
	LastErrorTime    time.Time
	ConnectedSince   time.Time
}

// NewNATSBridge creates a new NATS bridge for PERCILA.
func NewNATSBridge(cfg NATSBridgeConfig) (*NATSBridge, error) {
	if cfg.NATSURL == "" {
		cfg = DefaultNATSBridgeConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	bridge := &NATSBridge{
		config:        cfg,
		running:       false,
		ctx:           ctx,
		cancel:        cancel,
		subscriptions: make([]*nats.Subscription, 0),
		threatChan:    make(chan Threat, cfg.EventBufferSize),
		positionChan:  make(chan SatellitePosition, cfg.EventBufferSize),
		telemetryChan: make(chan Telemetry, cfg.EventBufferSize),
	}

	return bridge, nil
}

// Connect establishes connection to the NATS server.
func (b *NATSBridge) Connect() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	opts := []nats.Option{
		nats.Name(b.config.ClientID),
		nats.ReconnectWait(b.config.ReconnectWait),
		nats.MaxReconnects(b.config.MaxReconnects),
		nats.PingInterval(b.config.PingInterval),
		nats.ReconnectHandler(b.onReconnect),
		nats.DisconnectErrHandler(b.onDisconnect),
		nats.ErrorHandler(b.onError),
		nats.ClosedHandler(b.onClosed),
	}

	if b.config.EnableCompression {
		opts = append(opts, nats.Compression(true))
	}

	nc, err := nats.Connect(b.config.NATSURL, opts...)
	if err != nil {
		b.recordError(fmt.Errorf("failed to connect to NATS: %w", err))
		return fmt.Errorf("failed to connect to NATS server at %s: %w", b.config.NATSURL, err)
	}

	b.nc = nc
	b.stats.ConnectedSince = time.Now()
	log.Printf("[PERCILA NATS] Connected to %s", nc.ConnectedUrl())

	return nil
}

// Start begins subscribing to NATS subjects and routing events.
func (b *NATSBridge) Start() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.running {
		return nil
	}

	if b.nc == nil || !b.nc.IsConnected() {
		return fmt.Errorf("NATS connection not established")
	}

	// Subscribe to all ASGARD event subjects
	subscriptions := []struct {
		subject string
		handler nats.MsgHandler
	}{
		// Giru threat events
		{SubjectGiruThreats, b.handleThreatMessage},
		{SubjectGiruThreatZones, b.handleThreatZoneMessage},

		// Silenus satellite events
		{SubjectSilenusPositions, b.handleSatellitePositionMessage},
		{SubjectSilenusAlerts, b.handleSilenusAlertMessage},

		// Sat_Net telemetry
		{SubjectSatNetTelemetry, b.handleTelemetryMessage},
		{SubjectSatNetContactWindows, b.handleContactWindowMessage},

		// Hunoid states
		{SubjectHunoidStates, b.handleHunoidStateMessage},

		// Nysus missions
		{SubjectNysusMissions, b.handleMissionMessage},

		// Environmental updates
		{SubjectWeatherUpdates, b.handleWeatherMessage},
		{SubjectNoFlyZones, b.handleNoFlyZoneMessage},
	}

	for _, s := range subscriptions {
		sub, err := b.nc.Subscribe(s.subject, s.handler)
		if err != nil {
			log.Printf("[PERCILA NATS] Failed to subscribe to %s: %v", s.subject, err)
			b.recordError(err)
			continue
		}
		b.subscriptions = append(b.subscriptions, sub)
		log.Printf("[PERCILA NATS] Subscribed to %s", s.subject)
	}

	b.running = true
	log.Println("[PERCILA NATS] Bridge started successfully")
	return nil
}

// Stop stops the bridge and closes all subscriptions.
func (b *NATSBridge) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.running {
		return nil
	}

	b.cancel()

	// Unsubscribe from all subjects
	for _, sub := range b.subscriptions {
		if err := sub.Unsubscribe(); err != nil {
			log.Printf("[PERCILA NATS] Error unsubscribing from %s: %v", sub.Subject, err)
		}
	}
	b.subscriptions = nil

	// Drain and close connection
	if b.nc != nil {
		if err := b.nc.Drain(); err != nil {
			log.Printf("[PERCILA NATS] Error draining connection: %v", err)
		}
	}

	b.running = false
	log.Println("[PERCILA NATS] Bridge stopped")
	return nil
}

// ============================================================================
// MESSAGE HANDLERS
// ============================================================================

func (b *NATSBridge) handleThreatMessage(msg *nats.Msg) {
	b.stats.mu.Lock()
	b.stats.MessagesReceived++
	b.stats.mu.Unlock()

	var threat Threat
	if err := json.Unmarshal(msg.Data, &threat); err != nil {
		b.recordError(fmt.Errorf("failed to unmarshal threat: %w", err))
		return
	}

	log.Printf("[PERCILA NATS] Received threat: %s (severity: %s)", threat.ID, threat.Severity)

	// Call registered handler
	if b.threatHandler != nil {
		go b.threatHandler(threat)
	}

	// Send to channel for async processing
	select {
	case b.threatChan <- threat:
	default:
		log.Printf("[PERCILA NATS] Threat channel full, dropping threat %s", threat.ID)
	}
}

func (b *NATSBridge) handleThreatZoneMessage(msg *nats.Msg) {
	b.stats.mu.Lock()
	b.stats.MessagesReceived++
	b.stats.mu.Unlock()

	var zone ThreatZone
	if err := json.Unmarshal(msg.Data, &zone); err != nil {
		b.recordError(fmt.Errorf("failed to unmarshal threat zone: %w", err))
		return
	}

	if b.threatZoneHandler != nil {
		go b.threatZoneHandler(zone)
	}
}

func (b *NATSBridge) handleSatellitePositionMessage(msg *nats.Msg) {
	b.stats.mu.Lock()
	b.stats.MessagesReceived++
	b.stats.mu.Unlock()

	var position SatellitePosition
	if err := json.Unmarshal(msg.Data, &position); err != nil {
		b.recordError(fmt.Errorf("failed to unmarshal satellite position: %w", err))
		return
	}

	if b.satelliteHandler != nil {
		go b.satelliteHandler(position)
	}

	select {
	case b.positionChan <- position:
	default:
	}
}

func (b *NATSBridge) handleSilenusAlertMessage(msg *nats.Msg) {
	b.stats.mu.Lock()
	b.stats.MessagesReceived++
	b.stats.mu.Unlock()

	var alert Alert
	if err := json.Unmarshal(msg.Data, &alert); err != nil {
		b.recordError(fmt.Errorf("failed to unmarshal Silenus alert: %w", err))
		return
	}

	log.Printf("[PERCILA NATS] Received Silenus alert: %s (type: %s)", alert.ID, alert.Type)
}

func (b *NATSBridge) handleTelemetryMessage(msg *nats.Msg) {
	b.stats.mu.Lock()
	b.stats.MessagesReceived++
	b.stats.mu.Unlock()

	var telemetry Telemetry
	if err := json.Unmarshal(msg.Data, &telemetry); err != nil {
		b.recordError(fmt.Errorf("failed to unmarshal telemetry: %w", err))
		return
	}

	if b.telemetryHandler != nil {
		go b.telemetryHandler(telemetry)
	}

	select {
	case b.telemetryChan <- telemetry:
	default:
	}
}

func (b *NATSBridge) handleContactWindowMessage(msg *nats.Msg) {
	b.stats.mu.Lock()
	b.stats.MessagesReceived++
	b.stats.mu.Unlock()

	var window ContactWindow
	if err := json.Unmarshal(msg.Data, &window); err != nil {
		b.recordError(fmt.Errorf("failed to unmarshal contact window: %w", err))
		return
	}

	if b.contactWindowHandler != nil {
		go b.contactWindowHandler(window)
	}
}

func (b *NATSBridge) handleHunoidStateMessage(msg *nats.Msg) {
	b.stats.mu.Lock()
	b.stats.MessagesReceived++
	b.stats.mu.Unlock()

	var state HunoidState
	if err := json.Unmarshal(msg.Data, &state); err != nil {
		b.recordError(fmt.Errorf("failed to unmarshal hunoid state: %w", err))
		return
	}

	if b.hunoidHandler != nil {
		go b.hunoidHandler(state)
	}
}

func (b *NATSBridge) handleMissionMessage(msg *nats.Msg) {
	b.stats.mu.Lock()
	b.stats.MessagesReceived++
	b.stats.mu.Unlock()

	var mission Mission
	if err := json.Unmarshal(msg.Data, &mission); err != nil {
		b.recordError(fmt.Errorf("failed to unmarshal mission: %w", err))
		return
	}

	log.Printf("[PERCILA NATS] Received mission update: %s (status: %s)", mission.ID, mission.Status)

	if b.missionHandler != nil {
		go b.missionHandler(mission)
	}
}

func (b *NATSBridge) handleWeatherMessage(msg *nats.Msg) {
	b.stats.mu.Lock()
	b.stats.MessagesReceived++
	b.stats.mu.Unlock()

	var weather map[string]interface{}
	if err := json.Unmarshal(msg.Data, &weather); err != nil {
		b.recordError(fmt.Errorf("failed to unmarshal weather: %w", err))
		return
	}

	log.Printf("[PERCILA NATS] Received weather update for region: %v", weather["region"])
}

func (b *NATSBridge) handleNoFlyZoneMessage(msg *nats.Msg) {
	b.stats.mu.Lock()
	b.stats.MessagesReceived++
	b.stats.mu.Unlock()

	var zone map[string]interface{}
	if err := json.Unmarshal(msg.Data, &zone); err != nil {
		b.recordError(fmt.Errorf("failed to unmarshal no-fly zone: %w", err))
		return
	}

	log.Printf("[PERCILA NATS] Received no-fly zone update: %v", zone["id"])
}

// ============================================================================
// PUBLISH METHODS
// ============================================================================

// PublishTrajectoryUpdate publishes a trajectory update event.
func (b *NATSBridge) PublishTrajectoryUpdate(update TrajectoryUpdateEvent) error {
	data, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal trajectory update: %w", err)
	}

	if err := b.publish(SubjectPercilaTrajectory, data); err != nil {
		return err
	}

	log.Printf("[PERCILA NATS] Published trajectory update for payload %s", update.PayloadID)
	return nil
}

// PublishThreatAlert publishes a threat alert from PERCILA.
func (b *NATSBridge) PublishThreatAlert(alert ThreatAlertEvent) error {
	data, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal threat alert: %w", err)
	}

	if err := b.publish(SubjectPercilaThreatAlert, data); err != nil {
		return err
	}

	log.Printf("[PERCILA NATS] Published threat alert for payload %s (threat: %s)", alert.PayloadID, alert.ThreatID)
	return nil
}

// PublishGuidanceCommand publishes a guidance command.
func (b *NATSBridge) PublishGuidanceCommand(cmd GuidanceCommandEvent) error {
	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal guidance command: %w", err)
	}

	if err := b.publish(SubjectPercilaGuidance, data); err != nil {
		return err
	}

	log.Printf("[PERCILA NATS] Published guidance command for payload %s (type: %s)", cmd.PayloadID, cmd.CommandType)
	return nil
}

// PublishPayloadStatus publishes payload status update.
func (b *NATSBridge) PublishPayloadStatus(status PayloadStatusEvent) error {
	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal payload status: %w", err)
	}

	if err := b.publish(SubjectPercilaPayloadStatus, data); err != nil {
		return err
	}

	return nil
}

// PublishEvasiveManeuver publishes an evasive maneuver event.
func (b *NATSBridge) PublishEvasiveManeuver(maneuver EvasiveManeuverEvent) error {
	data, err := json.Marshal(maneuver)
	if err != nil {
		return fmt.Errorf("failed to marshal evasive maneuver: %w", err)
	}

	if err := b.publish(SubjectPercilaEvasion, data); err != nil {
		return err
	}

	log.Printf("[PERCILA NATS] Published evasive maneuver for payload %s (type: %s)", maneuver.PayloadID, maneuver.ManeuverType)
	return nil
}

// PublishMissionUpdate publishes a PERCILA mission update.
func (b *NATSBridge) PublishMissionUpdate(missionID string, status string, details map[string]interface{}) error {
	event := PercilaEvent{
		ID:        generateEventID(),
		Type:      EventTypeMissionUpdate,
		Source:    "percila",
		Timestamp: time.Now().UTC(),
		MissionID: missionID,
		Payload: map[string]interface{}{
			"missionId": missionID,
			"status":    status,
			"details":   details,
		},
		Priority: 5,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal mission update: %w", err)
	}

	if err := b.publish(SubjectPercilaMission, data); err != nil {
		return err
	}

	log.Printf("[PERCILA NATS] Published mission update: %s (status: %s)", missionID, status)
	return nil
}

// PublishArrivalEstimate publishes an estimated arrival time update.
func (b *NATSBridge) PublishArrivalEstimate(payloadID string, missionID string, eta time.Time, remainingDistance float64) error {
	event := PercilaEvent{
		ID:        generateEventID(),
		Type:      EventTypeArrivalEstimate,
		Source:    "percila",
		Timestamp: time.Now().UTC(),
		PayloadID: payloadID,
		MissionID: missionID,
		Payload: map[string]interface{}{
			"eta":               eta,
			"remainingDistance": remainingDistance,
		},
		Priority: 3,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal arrival estimate: %w", err)
	}

	return b.publish(SubjectPercilaArrival, data)
}

// PublishRouteDeviation publishes a route deviation notification.
func (b *NATSBridge) PublishRouteDeviation(payloadID string, reason string, deviationKm float64, newRoute []Waypoint) error {
	event := PercilaEvent{
		ID:        generateEventID(),
		Type:      EventTypeRouteDeviation,
		Source:    "percila",
		Timestamp: time.Now().UTC(),
		PayloadID: payloadID,
		Payload: map[string]interface{}{
			"reason":      reason,
			"deviationKm": deviationKm,
			"newRoute":    newRoute,
		},
		Priority: 7,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal route deviation: %w", err)
	}

	if err := b.publish(SubjectPercilaDeviation, data); err != nil {
		return err
	}

	log.Printf("[PERCILA NATS] Published route deviation for payload %s (reason: %s, deviation: %.2f km)", payloadID, reason, deviationKm)
	return nil
}

// PublishEvent publishes a generic PERCILA event.
func (b *NATSBridge) PublishEvent(subject string, event PercilaEvent) error {
	if event.ID == "" {
		event.ID = generateEventID()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.Source == "" {
		event.Source = "percila"
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return b.publish(subject, data)
}

// publish is a helper method to publish data to a NATS subject.
func (b *NATSBridge) publish(subject string, data []byte) error {
	b.mu.RLock()
	nc := b.nc
	b.mu.RUnlock()

	if nc == nil || !nc.IsConnected() {
		return fmt.Errorf("not connected to NATS")
	}

	if err := nc.Publish(subject, data); err != nil {
		b.recordError(err)
		return fmt.Errorf("failed to publish to %s: %w", subject, err)
	}

	b.stats.mu.Lock()
	b.stats.MessagesSent++
	b.stats.mu.Unlock()

	return nil
}

// ============================================================================
// HANDLER REGISTRATION
// ============================================================================

// SetThreatHandler sets the handler for incoming threat events.
func (b *NATSBridge) SetThreatHandler(handler func(Threat)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.threatHandler = handler
}

// SetSatelliteHandler sets the handler for incoming satellite position events.
func (b *NATSBridge) SetSatelliteHandler(handler func(SatellitePosition)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.satelliteHandler = handler
}

// SetTelemetryHandler sets the handler for incoming telemetry events.
func (b *NATSBridge) SetTelemetryHandler(handler func(Telemetry)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.telemetryHandler = handler
}

// SetHunoidHandler sets the handler for incoming Hunoid state events.
func (b *NATSBridge) SetHunoidHandler(handler func(HunoidState)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.hunoidHandler = handler
}

// SetMissionHandler sets the handler for incoming mission events.
func (b *NATSBridge) SetMissionHandler(handler func(Mission)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.missionHandler = handler
}

// SetContactWindowHandler sets the handler for incoming contact window events.
func (b *NATSBridge) SetContactWindowHandler(handler func(ContactWindow)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.contactWindowHandler = handler
}

// SetThreatZoneHandler sets the handler for incoming threat zone events.
func (b *NATSBridge) SetThreatZoneHandler(handler func(ThreatZone)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.threatZoneHandler = handler
}

// ============================================================================
// EVENT CHANNELS
// ============================================================================

// Threats returns a channel for receiving threat events.
func (b *NATSBridge) Threats() <-chan Threat {
	return b.threatChan
}

// SatellitePositions returns a channel for receiving satellite position events.
func (b *NATSBridge) SatellitePositions() <-chan SatellitePosition {
	return b.positionChan
}

// Telemetry returns a channel for receiving telemetry events.
func (b *NATSBridge) TelemetryUpdates() <-chan Telemetry {
	return b.telemetryChan
}

// ============================================================================
// CONNECTION CALLBACKS
// ============================================================================

func (b *NATSBridge) onReconnect(nc *nats.Conn) {
	b.stats.mu.Lock()
	b.stats.Reconnects++
	b.stats.ConnectedSince = time.Now()
	b.stats.mu.Unlock()

	log.Printf("[PERCILA NATS] Reconnected to %s (reconnect #%d)", nc.ConnectedUrl(), b.stats.Reconnects)

	// Re-establish subscriptions if bridge was running
	if b.running {
		go func() {
			if err := b.Start(); err != nil {
				log.Printf("[PERCILA NATS] Failed to restart subscriptions after reconnect: %v", err)
			}
		}()
	}
}

func (b *NATSBridge) onDisconnect(nc *nats.Conn, err error) {
	if err != nil {
		b.recordError(err)
		log.Printf("[PERCILA NATS] Disconnected from %s: %v", nc.ConnectedUrl(), err)
	} else {
		log.Printf("[PERCILA NATS] Disconnected from %s", nc.ConnectedUrl())
	}
}

func (b *NATSBridge) onError(nc *nats.Conn, sub *nats.Subscription, err error) {
	b.recordError(err)
	subject := ""
	if sub != nil {
		subject = sub.Subject
	}
	log.Printf("[PERCILA NATS] Error on subject %s: %v", subject, err)
}

func (b *NATSBridge) onClosed(nc *nats.Conn) {
	log.Println("[PERCILA NATS] Connection closed")
}

// ============================================================================
// STATUS AND STATISTICS
// ============================================================================

// IsConnected returns whether the bridge is connected to NATS.
func (b *NATSBridge) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.nc != nil && b.nc.IsConnected()
}

// IsRunning returns whether the bridge is running.
func (b *NATSBridge) IsRunning() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.running
}

// Stats returns current bridge statistics.
func (b *NATSBridge) Stats() map[string]interface{} {
	b.mu.RLock()
	nc := b.nc
	running := b.running
	b.mu.RUnlock()

	b.stats.mu.RLock()
	defer b.stats.mu.RUnlock()

	result := map[string]interface{}{
		"running":           running,
		"connected":         nc != nil && nc.IsConnected(),
		"messages_received": b.stats.MessagesReceived,
		"messages_sent":     b.stats.MessagesSent,
		"reconnects":        b.stats.Reconnects,
		"errors":            b.stats.Errors,
		"subscriptions":     len(b.subscriptions),
	}

	if nc != nil {
		natsStats := nc.Stats()
		result["nats_in_msgs"] = natsStats.InMsgs
		result["nats_out_msgs"] = natsStats.OutMsgs
		result["nats_in_bytes"] = natsStats.InBytes
		result["nats_out_bytes"] = natsStats.OutBytes
		result["nats_reconnects"] = natsStats.Reconnects
	}

	if !b.stats.ConnectedSince.IsZero() {
		result["connected_since"] = b.stats.ConnectedSince
		result["uptime_seconds"] = time.Since(b.stats.ConnectedSince).Seconds()
	}

	if !b.stats.LastErrorTime.IsZero() {
		result["last_error"] = b.stats.LastError
		result["last_error_time"] = b.stats.LastErrorTime
	}

	return result
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func (b *NATSBridge) recordError(err error) {
	b.stats.mu.Lock()
	defer b.stats.mu.Unlock()
	b.stats.Errors++
	b.stats.LastError = err.Error()
	b.stats.LastErrorTime = time.Now()
}

func generateEventID() string {
	return time.Now().Format("20060102150405") + "-" + generateShortID()
}

func generateShortID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// ============================================================================
// REQUEST-REPLY PATTERN
// ============================================================================

// Request sends a request and waits for a reply.
func (b *NATSBridge) Request(subject string, data []byte, timeout time.Duration) ([]byte, error) {
	b.mu.RLock()
	nc := b.nc
	b.mu.RUnlock()

	if nc == nil || !nc.IsConnected() {
		return nil, fmt.Errorf("not connected to NATS")
	}

	msg, err := nc.Request(subject, data, timeout)
	if err != nil {
		b.recordError(err)
		return nil, fmt.Errorf("request failed: %w", err)
	}

	b.stats.mu.Lock()
	b.stats.MessagesSent++
	b.stats.MessagesReceived++
	b.stats.mu.Unlock()

	return msg.Data, nil
}

// RequestThreatAssessment requests a threat assessment from Giru.
func (b *NATSBridge) RequestThreatAssessment(location GeoCoord, radius float64) ([]ThreatZone, error) {
	request := map[string]interface{}{
		"location": location,
		"radiusKm": radius,
	}

	data, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	reply, err := b.Request("asgard.giru.assess_threats", data, 5*time.Second)
	if err != nil {
		return nil, err
	}

	var zones []ThreatZone
	if err := json.Unmarshal(reply, &zones); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return zones, nil
}

// RequestSatelliteCoverage requests satellite coverage information from Silenus.
func (b *NATSBridge) RequestSatelliteCoverage(location GeoCoord, timeWindow time.Duration) ([]SatellitePosition, error) {
	request := map[string]interface{}{
		"location":      location,
		"windowMinutes": int(timeWindow.Minutes()),
	}

	data, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	reply, err := b.Request("asgard.silenus.coverage", data, 5*time.Second)
	if err != nil {
		return nil, err
	}

	var positions []SatellitePosition
	if err := json.Unmarshal(reply, &positions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return positions, nil
}
