package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/asgard/pandora/Pricilla/internal/integration"
	"github.com/asgard/pandora/Pricilla/internal/metrics"
	"github.com/asgard/pandora/Pricilla/internal/sensors"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PRICILLA - Precision Engagement & Routing Control with Integrated Learning Architecture
// The most advanced AI guidance system in ASGARD

const (
	Version = "2.0.0"
	AppName = "PRICILLA"
)

// =============================================================================
// CLEARANCE & ACCESS CONTROL
// =============================================================================

// ClearanceLevel defines access tiers
type ClearanceLevel int

const (
	ClearancePublic    ClearanceLevel = 0
	ClearanceCivilian  ClearanceLevel = 1
	ClearanceMilitary  ClearanceLevel = 2
	ClearanceGov       ClearanceLevel = 3
	ClearanceSecret    ClearanceLevel = 4
	ClearanceUltra     ClearanceLevel = 5
)

func getClearanceName(level ClearanceLevel) string {
	names := []string{"PUBLIC", "CIVILIAN", "MILITARY", "GOVERNMENT", "SECRET", "ULTRA"}
	if int(level) < len(names) {
		return names[level]
	}
	return "UNKNOWN"
}

// User represents a system user
type User struct {
	ID          string         `json:"id"`
	Username    string         `json:"username"`
	Clearance   ClearanceLevel `json:"clearance"`
	AccessTypes []string       `json:"accessTypes"`
	Active      bool           `json:"active"`
}

// Session represents an authenticated session
type Session struct {
	ID        string         `json:"id"`
	UserID    string         `json:"userId"`
	Token     string         `json:"token"`
	Clearance ClearanceLevel `json:"clearance"`
	ExpiresAt time.Time      `json:"expiresAt"`
}

// Terminal represents an access terminal
type Terminal struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Type      string         `json:"type"`
	Location  string         `json:"location"`
	Clearance ClearanceLevel `json:"clearance"`
	Status    string         `json:"status"`
}

// =============================================================================
// LIVE FEED SYSTEM
// =============================================================================

// LiveFeed represents a live data stream
type LiveFeed struct {
	ID          string         `json:"id"`
	MissionID   string         `json:"missionId"`
	PayloadID   string         `json:"payloadId"`
	PayloadType string         `json:"payloadType"`
	StreamType  string         `json:"streamType"`
	Clearance   ClearanceLevel `json:"clearance"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Status      string         `json:"status"`
	ViewerCount int            `json:"viewerCount"`
	Quality     string         `json:"quality"`
	StartedAt   time.Time      `json:"startedAt"`
}

// TelemetryFrame represents a telemetry update
type TelemetryFrame struct {
	PayloadID    string    `json:"payloadId"`
	Position     Vector3D  `json:"position"`
	GeoPosition  *GeoCoord `json:"geoPosition,omitempty"`
	Velocity     Vector3D  `json:"velocity"`
	Heading      float64   `json:"heading"`
	Speed        float64   `json:"speed"`
	Fuel         float64   `json:"fuel"`
	Battery      float64   `json:"battery"`
	Status       string    `json:"status"`
	MissionPhase string    `json:"missionPhase"`
	ETA          string    `json:"eta"`
	Distance     float64   `json:"distanceRemaining"`
	Timestamp    time.Time `json:"timestamp"`
}

// GeoCoord represents geographic coordinates
type GeoCoord struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"`
}

// Config holds application configuration
type Config struct {
	HTTPPort           int    `json:"httpPort"`
	MetricsPort        int    `json:"metricsPort"`
	NysusEndpoint      string `json:"nysusEndpoint"`
	SatNetEndpoint     string `json:"satnetEndpoint"`
	GiruEndpoint       string `json:"giruEndpoint"`
	NATSURL            string `json:"natsUrl"`
	EnableStealth      bool   `json:"enableStealth"`
	EnablePrediction   bool   `json:"enablePrediction"`
	EnableMultiPayload bool   `json:"enableMultiPayload"`
	EnableNATS         bool   `json:"enableNats"`
	EnableSensorFusion bool   `json:"enableSensorFusion"`
	EnableWiFiImaging  bool   `json:"enableWiFiImaging"`
	ReplanInterval     time.Duration `json:"replanInterval"`
	LogLevel           string `json:"logLevel"`
}

// Vector3D represents 3D coordinates
type Vector3D struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// Waypoint represents a navigation waypoint
type Waypoint struct {
	ID        string    `json:"id"`
	Position  Vector3D  `json:"position"`
	Velocity  Vector3D  `json:"velocity"`
	Timestamp time.Time `json:"timestamp"`
	Stealth   bool      `json:"stealth"`
}

// Trajectory represents a flight path
type Trajectory struct {
	ID           string     `json:"id"`
	PayloadID    string     `json:"payloadId"`
	PayloadType  string     `json:"payloadType"`
	Waypoints    []Waypoint `json:"waypoints"`
	StealthScore float64    `json:"stealthScore"`
	Confidence   float64    `json:"confidence"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"createdAt"`
}

// Mission represents a guidance mission
type Mission struct {
	ID              string    `json:"id"`
	Type            string    `json:"type"`
	PayloadID       string    `json:"payloadId"`
	PayloadType     string    `json:"payloadType"`
	StartPosition   Vector3D  `json:"startPosition"`
	TargetPosition  Vector3D  `json:"targetPosition"`
	Priority        int       `json:"priority"`
	StealthRequired bool      `json:"stealthRequired"`
	Status          string    `json:"status"`
	Trajectory      *Trajectory `json:"trajectory,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// PayloadState represents current state of a payload
type PayloadState struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Position     Vector3D  `json:"position"`
	Velocity     Vector3D  `json:"velocity"`
	Heading      float64   `json:"heading"`
	Fuel         float64   `json:"fuel"`
	Battery      float64   `json:"battery"`
	Health       float64   `json:"health"`
	Status       string    `json:"status"`
	LastUpdate   time.Time `json:"lastUpdate"`
}

// TargetingMetrics captures real-time targeting performance data
type TargetingMetrics struct {
	TargetUpdates       int       `json:"targetUpdates"`
	ReplanCount         int       `json:"replanCount"`
	LastReplanReason    string    `json:"lastReplanReason"`
	LastReplanAt        time.Time `json:"lastReplanAt"`
	LastTargetUpdateAt  time.Time `json:"lastTargetUpdateAt"`
	LastTargetPosition  Vector3D  `json:"lastTargetPosition"`
	LastPayloadPosition Vector3D  `json:"lastPayloadPosition"`
	LastTrajectoryID    string    `json:"lastTrajectoryId"`
	LastMissionID       string    `json:"lastMissionId"`
	CompletionDistance  float64   `json:"completionDistance"`
	MissionCompletedAt  time.Time `json:"missionCompletedAt"`
	// Enhanced targeting metrics
	HitProbability      float64   `json:"hitProbability"`      // Estimated probability of hit (0-1)
	CEP                 float64   `json:"cep"`                 // Circular Error Probable in meters
	TerminalGuidanceOn  bool      `json:"terminalGuidanceOn"`  // Terminal guidance mode active
	TimeToImpact        float64   `json:"timeToImpact"`        // Seconds until impact
	ClosingVelocity     float64   `json:"closingVelocity"`     // m/s closing rate
	CrossTrackError     float64   `json:"crossTrackError"`     // meters off optimal path
	ECMDetected         bool      `json:"ecmDetected"`         // Electronic countermeasures detected
	WeatherImpact       float64   `json:"weatherImpact"`       // Weather degradation factor (0-1)
}

// WeatherCondition represents current weather affecting guidance
type WeatherCondition struct {
	WindSpeed       float64 `json:"windSpeed"`       // m/s
	WindDirection   float64 `json:"windDirection"`   // radians
	Visibility      float64 `json:"visibility"`      // meters
	Precipitation   float64 `json:"precipitation"`   // mm/hr
	Temperature     float64 `json:"temperature"`     // Celsius
	Turbulence      float64 `json:"turbulence"`      // 0-1 severity
	IcingRisk       float64 `json:"icingRisk"`       // 0-1 risk level
}

// ECMThreat represents electronic countermeasure detection
type ECMThreat struct {
	ID             string    `json:"id"`
	Type           string    `json:"type"`           // jamming, spoofing, deception
	Position       Vector3D  `json:"position"`
	EffectRadius   float64   `json:"effectRadius"`
	Strength       float64   `json:"strength"`       // 0-1
	FrequencyBand  string    `json:"frequencyBand"`  // GPS, radar, comms
	DetectedAt     time.Time `json:"detectedAt"`
	Active         bool      `json:"active"`
}

// TerminalGuidanceConfig configures precision terminal approach
type TerminalGuidanceConfig struct {
	Enabled            bool    `json:"enabled"`
	ActivationDistance float64 `json:"activationDistance"` // meters to target when activated
	UpdateRateHz       float64 `json:"updateRateHz"`       // Hz for terminal updates
	MaxCorrection      float64 `json:"maxCorrection"`      // max correction angle rad/s
	PredictorHorizon   float64 `json:"predictorHorizon"`   // seconds ahead prediction
	ProNavGain         float64 `json:"proNavGain"`         // proportional navigation gain
}

// GuidanceEngine is the core AI guidance system
type GuidanceEngine struct {
	mu sync.RWMutex

	missions     map[string]*Mission
	trajectories map[string]*Trajectory
	payloads     map[string]*PayloadState
	
	// Access Control
	users        map[string]*User
	sessions     map[string]*Session
	terminals    map[string]*Terminal
	
	// Live Feeds
	feeds        map[string]*LiveFeed
	telemetry    map[string]*TelemetryFrame
	
	// Integration components
	natsBridge   *integration.NATSBridge
	sensorFusion *sensors.SensorFusion
	metrics      *metrics.Metrics
	replanInterval time.Duration
	lastReplan      time.Time
	wifiModel       *sensors.WiFiImagingModel
	wifiRouters     map[string]sensors.WiFiRouter
	targetingMetrics TargetingMetrics
	
	// Enhanced guidance systems
	weather           *WeatherCondition
	ecmThreats        map[string]*ECMThreat
	terminalConfig    TerminalGuidanceConfig
	abortedMissions   map[string]string // missionID -> abort reason
	
	config       Config
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewGuidanceEngine creates a new guidance engine
func NewGuidanceEngine(config Config) *GuidanceEngine {
	replanInterval := config.ReplanInterval
	if replanInterval <= 0 {
		replanInterval = 250 * time.Millisecond
	}
	ge := &GuidanceEngine{
		missions:     make(map[string]*Mission),
		trajectories: make(map[string]*Trajectory),
		payloads:     make(map[string]*PayloadState),
		users:        make(map[string]*User),
		sessions:     make(map[string]*Session),
		terminals:    make(map[string]*Terminal),
		feeds:        make(map[string]*LiveFeed),
		telemetry:    make(map[string]*TelemetryFrame),
		config:       config,
		replanInterval: replanInterval,
		wifiRouters:  make(map[string]sensors.WiFiRouter),
		targetingMetrics: TargetingMetrics{HitProbability: 0.0, CEP: 100.0},
		// Enhanced systems initialization
		weather: &WeatherCondition{
			WindSpeed:     0,
			Visibility:    10000, // 10km default visibility
			Temperature:   15,    // 15C default
			Turbulence:    0,
		},
		ecmThreats:      make(map[string]*ECMThreat),
		abortedMissions: make(map[string]string),
		terminalConfig: TerminalGuidanceConfig{
			Enabled:            true,
			ActivationDistance: 1000,  // 1km terminal phase
			UpdateRateHz:       50,    // 50Hz terminal updates
			MaxCorrection:      0.5,   // 0.5 rad/s max correction
			PredictorHorizon:   5,     // 5 second prediction
			ProNavGain:         4.0,   // Standard PN gain
		},
	}

	if config.EnableWiFiImaging {
		ge.wifiModel = sensors.NewWiFiImagingModel()
	}

	// Initialize Prometheus metrics
	ge.metrics = metrics.GetMetrics()

	return ge
}

// Start begins the guidance engine
func (ge *GuidanceEngine) Start(ctx context.Context) error {
	ge.ctx, ge.cancel = context.WithCancel(ctx)

	log.Printf("[%s] Starting Guidance Engine v%s", AppName, Version)

	// Initialize NATS bridge if enabled
	if ge.config.EnableNATS && ge.config.NATSURL != "" {
		if err := ge.initNATSBridge(); err != nil {
			log.Printf("[%s] Warning: Failed to initialize NATS bridge: %v", AppName, err)
		} else {
			log.Printf("[%s] NATS bridge initialized", AppName)
		}
	}

	// Initialize sensor fusion if enabled
	if ge.config.EnableSensorFusion {
		if err := ge.initSensorFusion(); err != nil {
			log.Printf("[%s] Warning: Failed to initialize sensor fusion: %v", AppName, err)
		} else {
			log.Printf("[%s] Sensor fusion initialized", AppName)
		}
	}

	// Start background processes
	go ge.telemetryProcessor()
	go ge.missionMonitor()
	go ge.trajectoryOptimizer()

	log.Printf("[%s] Guidance Engine started successfully", AppName)
	return nil
}

// initNATSBridge initializes and connects the NATS bridge
func (ge *GuidanceEngine) initNATSBridge() error {
	cfg := integration.NATSBridgeConfig{
		NATSURL:           ge.config.NATSURL,
		ClusterID:         "asgard-cluster",
		ClientID:          "pricilla-guidance",
		ReconnectWait:     2 * time.Second,
		MaxReconnects:     -1,
		PingInterval:      30 * time.Second,
		MaxPendingEvents:  1000,
		EventBufferSize:   500,
		EnableCompression: true,
	}

	bridge, err := integration.NewNATSBridge(cfg)
	if err != nil {
		return fmt.Errorf("failed to create NATS bridge: %w", err)
	}

	// Connect to NATS
	if err := bridge.Connect(); err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Set up event handlers
	bridge.SetThreatHandler(ge.handleThreatEvent)
	bridge.SetTelemetryHandler(ge.handleTelemetryEvent)
	bridge.SetMissionHandler(ge.handleMissionEvent)
	bridge.SetHunoidHandler(ge.handleHunoidEvent)
	bridge.SetSatelliteHandler(ge.handleSatelliteEvent)

	// Start the bridge
	if err := bridge.Start(); err != nil {
		return fmt.Errorf("failed to start NATS bridge: %w", err)
	}

	ge.natsBridge = bridge
	return nil
}

// initSensorFusion initializes the sensor fusion system
func (ge *GuidanceEngine) initSensorFusion() error {
	cfg := sensors.DefaultFusionConfig()
	cfg.MinSensorsRequired = 1 // Allow single sensor operation
	cfg.UpdateRate = 50 * time.Millisecond

	sf := sensors.NewSensorFusion(cfg)

	// Set up callbacks
	sf.OnStateUpdate(ge.handleFusedStateUpdate)
	sf.OnSensorFailure(ge.handleSensorFailure)
	sf.OnAnomalyDetect(ge.handleSensorAnomaly)
	sf.OnFailoverEvent(ge.handleSensorFailover)

	// Register default sensors
	sf.RegisterSensor("gps-primary", sensors.SensorGPS, nil)
	sf.RegisterSensor("ins-primary", sensors.SensorINS, nil)
	sf.RegisterSensor("radar-primary", sensors.SensorRADAR, nil)
	if ge.config.EnableWiFiImaging {
		sf.RegisterSensor("wifi-imaging-primary", sensors.SensorWiFi, nil)
	}

	// Start sensor fusion
	if err := sf.Start(ge.ctx); err != nil {
		return fmt.Errorf("failed to start sensor fusion: %w", err)
	}

	ge.sensorFusion = sf
	return nil
}

// NATS event handlers
func (ge *GuidanceEngine) handleThreatEvent(threat integration.Threat) {
	log.Printf("[%s] Received threat event: %s (severity: %s)", AppName, threat.ID, threat.Severity)
	
	// Record metric
	metrics.RecordThreatAssessment(threat.Severity, "received")
	
	// Update service connection status
	metrics.UpdateServiceConnectionStatus("giru", true)
}

func (ge *GuidanceEngine) handleTelemetryEvent(telemetry integration.Telemetry) {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	// Update payload state from telemetry
	state := &PayloadState{
		ID:         telemetry.PayloadID,
		Position:   Vector3D{X: telemetry.Position.X, Y: telemetry.Position.Y, Z: telemetry.Position.Z},
		Velocity:   Vector3D{X: telemetry.Velocity.X, Y: telemetry.Velocity.Y, Z: telemetry.Velocity.Z},
		Fuel:       telemetry.Fuel,
		Battery:    telemetry.Battery,
		Health:     telemetry.Health,
		Status:     telemetry.Status,
		LastUpdate: telemetry.Timestamp,
	}
	ge.payloads[telemetry.PayloadID] = state

	// Record metric
	metrics.RecordPositionUpdate()

	ge.requestFastReplanLocked(telemetry.PayloadID, "telemetry_update")
}

func (ge *GuidanceEngine) handleMissionEvent(mission integration.Mission) {
	log.Printf("[%s] Received mission event: %s (status: %s)", AppName, mission.ID, mission.Status)
	
	// Update service connection status
	metrics.UpdateServiceConnectionStatus("nysus", true)
}

func (ge *GuidanceEngine) handleHunoidEvent(state integration.HunoidState) {
	log.Printf("[%s] Received Hunoid state: %s (status: %s)", AppName, state.HunoidID, state.Status)
	
	// Update service connection status
	metrics.UpdateServiceConnectionStatus("hunoid", true)
}

func (ge *GuidanceEngine) handleSatelliteEvent(position integration.SatellitePosition) {
	log.Printf("[%s] Received satellite position: %s", AppName, position.SatelliteID)
	
	// Update service connection status
	metrics.UpdateServiceConnectionStatus("silenus", true)
}

// Sensor fusion callbacks
func (ge *GuidanceEngine) handleFusedStateUpdate(state sensors.FusedState) {
	// Update navigation accuracy metric
	metrics.RecordNavigationFix(string(state.PrimarySensor), fmt.Sprintf("%.2f", state.Confidence))
}

func (ge *GuidanceEngine) handleSensorFailure(sensorID string, health sensors.SensorHealth) {
	log.Printf("[%s] Sensor failure detected: %s (status: %s)", AppName, sensorID, health.Status)
	
	// Record detection event for sensor failure
	metrics.RecordDetectionEvent("sensor", "failure")
}

func (ge *GuidanceEngine) handleSensorAnomaly(anomaly sensors.AnomalyReport) {
	log.Printf("[%s] Sensor anomaly detected: %s on %s (severity: %.2f)", 
		AppName, anomaly.AnomalyType, anomaly.SensorID, anomaly.Severity)
}

func (ge *GuidanceEngine) handleSensorFailover(from, to sensors.SensorType) {
	log.Printf("[%s] Sensor failover: %s -> %s", AppName, from, to)
}

// PublishTrajectoryUpdate publishes a trajectory update via NATS
func (ge *GuidanceEngine) PublishTrajectoryUpdate(payloadID, missionID string, waypoints []Waypoint) error {
	if ge.natsBridge == nil || !ge.natsBridge.IsConnected() {
		return fmt.Errorf("NATS bridge not connected")
	}

	// Convert waypoints to integration format
	intWaypoints := make([]integration.Waypoint, len(waypoints))
	for i, wp := range waypoints {
		intWaypoints[i] = integration.Waypoint{
			ID:        wp.ID,
			Position:  integration.Vector3D{X: wp.Position.X, Y: wp.Position.Y, Z: wp.Position.Z},
			Velocity:  integration.Vector3D{X: wp.Velocity.X, Y: wp.Velocity.Y, Z: wp.Velocity.Z},
			Timestamp: wp.Timestamp,
		}
	}

	update := integration.TrajectoryUpdateEvent{
		PayloadID:    payloadID,
		MissionID:    missionID,
		NewWaypoints: intWaypoints,
		Reason:       "trajectory_update",
		EstimatedETA: time.Now().Add(30 * time.Minute),
		Timestamp:    time.Now(),
	}

	return ge.natsBridge.PublishTrajectoryUpdate(update)
}

// PublishPayloadStatus publishes payload status via NATS
func (ge *GuidanceEngine) PublishPayloadStatus(state *PayloadState) error {
	if ge.natsBridge == nil || !ge.natsBridge.IsConnected() {
		return fmt.Errorf("NATS bridge not connected")
	}

	status := integration.PayloadStatusEvent{
		PayloadID: state.ID,
		Position:  integration.Vector3D{X: state.Position.X, Y: state.Position.Y, Z: state.Position.Z},
		Velocity:  integration.Vector3D{X: state.Velocity.X, Y: state.Velocity.Y, Z: state.Velocity.Z},
		Heading:   state.Heading,
		Altitude:  state.Position.Z,
		Speed:     math.Sqrt(state.Velocity.X*state.Velocity.X + state.Velocity.Y*state.Velocity.Y + state.Velocity.Z*state.Velocity.Z),
		Fuel:      state.Fuel,
		Battery:   state.Battery,
		Status:    state.Status,
		Timestamp: time.Now(),
	}

	return ge.natsBridge.PublishPayloadStatus(status)
}

// Stop halts the guidance engine
func (ge *GuidanceEngine) Stop() {
	log.Printf("[%s] Stopping Guidance Engine...", AppName)

	// Stop sensor fusion
	if ge.sensorFusion != nil {
		ge.sensorFusion.Stop()
		log.Printf("[%s] Sensor fusion stopped", AppName)
	}

	// Stop NATS bridge
	if ge.natsBridge != nil {
		if err := ge.natsBridge.Stop(); err != nil {
			log.Printf("[%s] Error stopping NATS bridge: %v", AppName, err)
		} else {
			log.Printf("[%s] NATS bridge stopped", AppName)
		}
	}

	// Cancel context
	if ge.cancel != nil {
		ge.cancel()
	}

	log.Printf("[%s] Guidance Engine stopped", AppName)
}

// GetNATSBridge returns the NATS bridge instance
func (ge *GuidanceEngine) GetNATSBridge() *integration.NATSBridge {
	return ge.natsBridge
}

// GetSensorFusion returns the sensor fusion instance
func (ge *GuidanceEngine) GetSensorFusion() *sensors.SensorFusion {
	return ge.sensorFusion
}

// GetMetrics returns the metrics instance
func (ge *GuidanceEngine) GetMetrics() *metrics.Metrics {
	return ge.metrics
}

// CreateMission creates a new guidance mission
func (ge *GuidanceEngine) CreateMission(mission *Mission) error {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	if mission.ID == "" {
		mission.ID = uuid.New().String()
	}
	mission.Status = "pending"
	mission.CreatedAt = time.Now()
	mission.UpdatedAt = time.Now()

	// Generate trajectory
	trajectory, err := ge.generateTrajectory(mission)
	if err != nil {
		return fmt.Errorf("failed to generate trajectory: %w", err)
	}

	mission.Trajectory = trajectory
	ge.missions[mission.ID] = mission
	ge.trajectories[trajectory.ID] = trajectory

	log.Printf("[%s] Created mission %s for payload %s", AppName, mission.ID, mission.PayloadID)
	return nil
}

// GetMission retrieves a mission by ID
func (ge *GuidanceEngine) GetMission(missionID string) (*Mission, error) {
	ge.mu.RLock()
	defer ge.mu.RUnlock()

	mission, exists := ge.missions[missionID]
	if !exists {
		return nil, fmt.Errorf("mission not found: %s", missionID)
	}
	return mission, nil
}

// GetAllMissions returns all missions
func (ge *GuidanceEngine) GetAllMissions() []*Mission {
	ge.mu.RLock()
	defer ge.mu.RUnlock()

	missions := make([]*Mission, 0, len(ge.missions))
	for _, m := range ge.missions {
		missions = append(missions, m)
	}
	return missions
}

// UpdatePayloadState updates the state of a payload
func (ge *GuidanceEngine) UpdatePayloadState(state *PayloadState) {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	state.LastUpdate = time.Now()
	ge.payloads[state.ID] = state
	ge.targetingMetrics.LastPayloadPosition = state.Position

	for _, mission := range ge.missions {
		if mission.PayloadID != state.ID {
			continue
		}

		if mission.Status == "pending" {
			mission.Status = "active"
			mission.UpdatedAt = time.Now()
		}

		if mission.Status != "completed" {
			distance := vectorDistance(state.Position, mission.TargetPosition)
			if distance <= 25.0 {
				mission.Status = "completed"
				mission.UpdatedAt = time.Now()
				ge.targetingMetrics.CompletionDistance = distance
				ge.targetingMetrics.MissionCompletedAt = time.Now()
				ge.targetingMetrics.LastMissionID = mission.ID
				log.Printf("[%s] Mission %s completed (distance %.2fm)", AppName, mission.ID, distance)
			}
		}
	}

	ge.requestFastReplanLocked(state.ID, "payload_update")
}

func (ge *GuidanceEngine) UpdateMissionTarget(missionID string, target Vector3D) (*Trajectory, error) {
	ge.mu.RLock()
	mission, exists := ge.missions[missionID]
	var payload *PayloadState
	if exists {
		payload = ge.payloads[mission.PayloadID]
	}
	ge.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("mission not found: %s", missionID)
	}

	missionCopy := *mission
	missionCopy.TargetPosition = target
	if payload != nil {
		missionCopy.StartPosition = payload.Position
	}
	missionCopy.UpdatedAt = time.Now()

	trajectory, err := ge.generateTrajectory(&missionCopy)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	ge.mu.Lock()
	if liveMission, ok := ge.missions[missionID]; ok {
		liveMission.TargetPosition = target
		if liveMission.Status == "pending" {
			liveMission.Status = "active"
		}
		if payload != nil {
			liveMission.StartPosition = payload.Position
		}
		liveMission.Trajectory = trajectory
		liveMission.UpdatedAt = now
		ge.trajectories[trajectory.ID] = trajectory
		ge.targetingMetrics.TargetUpdates++
		ge.targetingMetrics.LastTargetUpdateAt = now
		ge.targetingMetrics.LastTargetPosition = target
		ge.targetingMetrics.LastReplanReason = "target_update"
		ge.targetingMetrics.LastReplanAt = now
		ge.targetingMetrics.ReplanCount++
		ge.targetingMetrics.LastTrajectoryID = trajectory.ID
		ge.targetingMetrics.LastMissionID = missionID
	}
	ge.mu.Unlock()

	log.Printf("[%s] Target updated for mission %s", AppName, missionID)
	return trajectory, nil
}

func (ge *GuidanceEngine) GetTargetingMetrics() TargetingMetrics {
	ge.mu.RLock()
	defer ge.mu.RUnlock()
	return ge.targetingMetrics
}

func vectorDistance(a, b Vector3D) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	dz := a.Z - b.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func (ge *GuidanceEngine) RegisterWiFiRouter(router sensors.WiFiRouter) {
	ge.mu.Lock()
	defer ge.mu.Unlock()
	ge.wifiRouters[router.ID] = router
}

func (ge *GuidanceEngine) GetWiFiRouters() []sensors.WiFiRouter {
	ge.mu.RLock()
	defer ge.mu.RUnlock()
	routers := make([]sensors.WiFiRouter, 0, len(ge.wifiRouters))
	for _, router := range ge.wifiRouters {
		routers = append(routers, router)
	}
	return routers
}

func (ge *GuidanceEngine) ProcessWiFiImaging(frame sensors.WiFiImagingFrame) ([]sensors.ThroughWallObservation, error) {
	if ge.wifiModel == nil || !ge.config.EnableWiFiImaging {
		return nil, fmt.Errorf("wifi imaging disabled")
	}

	ge.mu.RLock()
	routers := make([]sensors.WiFiRouter, 0, len(ge.wifiRouters))
	for _, router := range ge.wifiRouters {
		routers = append(routers, router)
	}
	ge.mu.RUnlock()

	observations, confidence := ge.wifiModel.EstimateThroughWall(frame, routers)
	if len(observations) == 0 {
		return observations, nil
	}

	reading := sensors.SensorReading{
		SensorID:   frame.RouterID + "-wifi",
		SensorType: sensors.SensorWiFi,
		Position:   observations[0].EstimatedPosition,
		Velocity:   sensors.Vector3D{},
		Covariance: sensors.Matrix3x3{{2, 0, 0}, {0, 2, 0}, {0, 0, 2}},
		Timestamp:  frame.Timestamp,
		Confidence: confidence,
		IsValid:    confidence >= 0.2,
	}

	if ge.sensorFusion != nil && ge.config.EnableSensorFusion {
		_ = ge.sensorFusion.ProcessReading(reading)
	}

	ge.mu.Lock()
	for _, mission := range ge.missions {
		if mission.Status == "active" {
			ge.requestFastReplanLocked(mission.PayloadID, "wifi_imaging")
		}
	}
	ge.mu.Unlock()

	return observations, nil
}

func (ge *GuidanceEngine) requestFastReplanLocked(payloadID, reason string) {
	if ge.replanInterval <= 0 {
		return
	}
	if time.Since(ge.lastReplan) < ge.replanInterval {
		return
	}

	var activeMission *Mission
	for _, mission := range ge.missions {
		if mission.PayloadID == payloadID && mission.Status == "active" {
			activeMission = mission
			break
		}
	}
	if activeMission == nil {
		return
	}

	ge.lastReplan = time.Now()
	missionID := activeMission.ID
	go ge.replanMission(missionID, payloadID, reason)
}

func (ge *GuidanceEngine) replanMission(missionID, payloadID, reason string) {
	ge.mu.RLock()
	mission, exists := ge.missions[missionID]
	payload, hasPayload := ge.payloads[payloadID]
	ge.mu.RUnlock()
	if !exists || !hasPayload {
		return
	}

	missionCopy := *mission
	missionCopy.StartPosition = payload.Position
	missionCopy.UpdatedAt = time.Now()

	trajectory, err := ge.generateTrajectory(&missionCopy)
	if err != nil {
		log.Printf("[%s] Rapid replan failed for mission %s: %v", AppName, missionID, err)
		return
	}

	ge.mu.Lock()
	if liveMission, ok := ge.missions[missionID]; ok {
		liveMission.StartPosition = payload.Position
		liveMission.Trajectory = trajectory
		liveMission.UpdatedAt = time.Now()
		ge.trajectories[trajectory.ID] = trajectory
		ge.targetingMetrics.ReplanCount++
		ge.targetingMetrics.LastReplanReason = reason
		ge.targetingMetrics.LastReplanAt = time.Now()
		ge.targetingMetrics.LastTrajectoryID = trajectory.ID
		ge.targetingMetrics.LastMissionID = missionID
		log.Printf("[%s] Rapid replan complete for mission %s (%s)", AppName, missionID, reason)
	}
	ge.mu.Unlock()
}

// =============================================================================
// ENHANCED GUIDANCE SYSTEMS
// =============================================================================

// CalculateHitProbability computes estimated hit probability based on current conditions
func (ge *GuidanceEngine) CalculateHitProbability(missionID string) float64 {
	ge.mu.RLock()
	defer ge.mu.RUnlock()

	mission, exists := ge.missions[missionID]
	if !exists || mission.Trajectory == nil {
		return 0.0
	}

	payload, hasPayload := ge.payloads[mission.PayloadID]
	if !hasPayload {
		return 0.0
	}

	// Calculate distance to target
	distance := vectorDistance(payload.Position, mission.TargetPosition)

	// Base probability from trajectory confidence
	baseProbability := mission.Trajectory.Confidence

	// Distance factor (closer = higher probability)
	distanceFactor := 1.0 / (1.0 + distance/10000.0)

	// Weather impact
	weatherFactor := 1.0
	if ge.weather != nil {
		weatherFactor = ge.calculateWeatherImpact()
	}

	// ECM impact
	ecmFactor := 1.0
	if len(ge.ecmThreats) > 0 {
		ecmFactor = ge.calculateECMImpact(payload.Position)
	}

	// Terminal guidance boost (if in terminal phase)
	terminalBoost := 1.0
	if ge.terminalConfig.Enabled && distance < ge.terminalConfig.ActivationDistance {
		terminalBoost = 1.2 // 20% boost in terminal phase
	}

	// Payload health factor
	healthFactor := payload.Health / 100.0
	if healthFactor == 0 {
		healthFactor = 1.0
	}

	// Calculate final probability
	hitProbability := baseProbability * distanceFactor * weatherFactor * ecmFactor * terminalBoost * healthFactor

	// Clamp to valid range
	if hitProbability > 1.0 {
		hitProbability = 1.0
	}
	if hitProbability < 0.0 {
		hitProbability = 0.0
	}

	return hitProbability
}

// CalculateCEP computes Circular Error Probable based on current conditions
func (ge *GuidanceEngine) CalculateCEP(missionID string) float64 {
	ge.mu.RLock()
	defer ge.mu.RUnlock()

	mission, exists := ge.missions[missionID]
	if !exists {
		return 1000.0 // Default 1km CEP
	}

	// Base CEP based on guidance type
	baseCEP := 50.0 // 50m base CEP with terminal guidance

	if !ge.terminalConfig.Enabled {
		baseCEP = 100.0 // 100m without terminal guidance
	}

	// Weather degradation
	weatherDegradation := 1.0
	if ge.weather != nil {
		if ge.weather.WindSpeed > 20 {
			weatherDegradation += (ge.weather.WindSpeed - 20) * 0.05
		}
		if ge.weather.Visibility < 5000 {
			weatherDegradation += (5000 - ge.weather.Visibility) / 5000 * 0.5
		}
		if ge.weather.Turbulence > 0.3 {
			weatherDegradation += ge.weather.Turbulence * 0.3
		}
	}

	// ECM degradation
	ecmDegradation := 1.0
	for _, ecm := range ge.ecmThreats {
		if ecm.Active {
			ecmDegradation += ecm.Strength * 0.5
		}
	}

	// Stealth mode provides better CEP (less detection, more optimal path)
	stealthBonus := 1.0
	if mission.StealthRequired {
		stealthBonus = 0.9
	}

	return baseCEP * weatherDegradation * ecmDegradation * stealthBonus
}

// UpdateWeather updates current weather conditions
func (ge *GuidanceEngine) UpdateWeather(weather WeatherCondition) {
	ge.mu.Lock()
	defer ge.mu.Unlock()
	ge.weather = &weather
	ge.targetingMetrics.WeatherImpact = ge.calculateWeatherImpactLocked()
	log.Printf("[%s] Weather updated: wind=%.1fm/s, vis=%.0fm, turb=%.2f",
		AppName, weather.WindSpeed, weather.Visibility, weather.Turbulence)
}

// GetWeather returns current weather conditions
func (ge *GuidanceEngine) GetWeather() *WeatherCondition {
	ge.mu.RLock()
	defer ge.mu.RUnlock()
	if ge.weather == nil {
		return nil
	}
	w := *ge.weather
	return &w
}

func (ge *GuidanceEngine) calculateWeatherImpact() float64 {
	ge.mu.RLock()
	defer ge.mu.RUnlock()
	return ge.calculateWeatherImpactLocked()
}

func (ge *GuidanceEngine) calculateWeatherImpactLocked() float64 {
	if ge.weather == nil {
		return 1.0
	}
	impact := 1.0
	// High wind reduces accuracy
	if ge.weather.WindSpeed > 10 {
		impact -= (ge.weather.WindSpeed - 10) * 0.01
	}
	// Low visibility reduces accuracy
	if ge.weather.Visibility < 5000 {
		impact -= (5000 - ge.weather.Visibility) / 10000
	}
	// Turbulence
	impact -= ge.weather.Turbulence * 0.2
	// Icing risk
	impact -= ge.weather.IcingRisk * 0.15
	if impact < 0.3 {
		impact = 0.3
	}
	return impact
}

// RegisterECMThreat registers a detected ECM threat
func (ge *GuidanceEngine) RegisterECMThreat(ecm ECMThreat) {
	ge.mu.Lock()
	defer ge.mu.Unlock()
	if ecm.ID == "" {
		ecm.ID = uuid.New().String()
	}
	ecm.DetectedAt = time.Now()
	ecm.Active = true
	ge.ecmThreats[ecm.ID] = &ecm
	ge.targetingMetrics.ECMDetected = true
	log.Printf("[%s] ECM threat registered: %s (type=%s, strength=%.2f)",
		AppName, ecm.ID, ecm.Type, ecm.Strength)
}

// ClearECMThreat removes an ECM threat
func (ge *GuidanceEngine) ClearECMThreat(ecmID string) {
	ge.mu.Lock()
	defer ge.mu.Unlock()
	delete(ge.ecmThreats, ecmID)
	ge.targetingMetrics.ECMDetected = len(ge.ecmThreats) > 0
}

// GetECMThreats returns all active ECM threats
func (ge *GuidanceEngine) GetECMThreats() []ECMThreat {
	ge.mu.RLock()
	defer ge.mu.RUnlock()
	threats := make([]ECMThreat, 0, len(ge.ecmThreats))
	for _, ecm := range ge.ecmThreats {
		threats = append(threats, *ecm)
	}
	return threats
}

func (ge *GuidanceEngine) calculateECMImpact(position Vector3D) float64 {
	if len(ge.ecmThreats) == 0 {
		return 1.0
	}
	impact := 1.0
	for _, ecm := range ge.ecmThreats {
		if !ecm.Active {
			continue
		}
		dist := vectorDistance(position, ecm.Position)
		if dist < ecm.EffectRadius {
			// Calculate impact based on distance and strength
			distanceFactor := 1.0 - (dist / ecm.EffectRadius)
			impact -= distanceFactor * ecm.Strength * 0.3
		}
	}
	if impact < 0.3 {
		impact = 0.3
	}
	return impact
}

// EnableTerminalGuidance enables/configures terminal guidance
func (ge *GuidanceEngine) EnableTerminalGuidance(config TerminalGuidanceConfig) {
	ge.mu.Lock()
	defer ge.mu.Unlock()
	ge.terminalConfig = config
	log.Printf("[%s] Terminal guidance configured: enabled=%v, activation=%.0fm, rate=%.0fHz",
		AppName, config.Enabled, config.ActivationDistance, config.UpdateRateHz)
}

// GetTerminalGuidanceConfig returns current terminal guidance configuration
func (ge *GuidanceEngine) GetTerminalGuidanceConfig() TerminalGuidanceConfig {
	ge.mu.RLock()
	defer ge.mu.RUnlock()
	return ge.terminalConfig
}

// CheckTerminalPhase checks if a mission has entered terminal phase
func (ge *GuidanceEngine) CheckTerminalPhase(missionID string) (bool, float64) {
	ge.mu.RLock()
	defer ge.mu.RUnlock()

	mission, exists := ge.missions[missionID]
	if !exists {
		return false, 0
	}

	payload, hasPayload := ge.payloads[mission.PayloadID]
	if !hasPayload {
		return false, 0
	}

	distance := vectorDistance(payload.Position, mission.TargetPosition)
	inTerminal := ge.terminalConfig.Enabled && distance < ge.terminalConfig.ActivationDistance

	return inTerminal, distance
}

// AbortMission aborts a mission and optionally initiates RTB
func (ge *GuidanceEngine) AbortMission(missionID, reason string, returnToBase bool) error {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	mission, exists := ge.missions[missionID]
	if !exists {
		return fmt.Errorf("mission not found: %s", missionID)
	}

	mission.Status = "aborted"
	mission.UpdatedAt = time.Now()
	ge.abortedMissions[missionID] = reason

	log.Printf("[%s] Mission %s ABORTED: %s", AppName, missionID, reason)

	if returnToBase && mission.Trajectory != nil && len(mission.Trajectory.Waypoints) > 0 {
		// Generate RTB trajectory
		rtbMission := &Mission{
			ID:             uuid.New().String(),
			Type:           "rtb",
			PayloadID:      mission.PayloadID,
			PayloadType:    mission.PayloadType,
			StartPosition:  mission.Trajectory.Waypoints[len(mission.Trajectory.Waypoints)/2].Position, // Current approx position
			TargetPosition: mission.StartPosition, // Return to launch position
			Priority:       mission.Priority + 1,  // Higher priority
			StealthRequired: true,
			Status:         "active",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		ge.missions[rtbMission.ID] = rtbMission
		log.Printf("[%s] RTB mission created: %s for payload %s", AppName, rtbMission.ID, mission.PayloadID)
	}

	return nil
}

// GetAbortedMissions returns list of aborted missions with reasons
func (ge *GuidanceEngine) GetAbortedMissions() map[string]string {
	ge.mu.RLock()
	defer ge.mu.RUnlock()
	result := make(map[string]string)
	for k, v := range ge.abortedMissions {
		result[k] = v
	}
	return result
}

// UpdateEnhancedTargetingMetrics updates all enhanced targeting metrics for a mission
func (ge *GuidanceEngine) UpdateEnhancedTargetingMetrics(missionID string) {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	mission, exists := ge.missions[missionID]
	if !exists {
		return
	}

	payload, hasPayload := ge.payloads[mission.PayloadID]
	if !hasPayload {
		return
	}

	// Calculate distance and closing velocity
	distance := vectorDistance(payload.Position, mission.TargetPosition)

	// Calculate closing velocity (assuming target is stationary for now)
	speed := math.Sqrt(payload.Velocity.X*payload.Velocity.X +
		payload.Velocity.Y*payload.Velocity.Y +
		payload.Velocity.Z*payload.Velocity.Z)

	// Estimate time to impact
	timeToImpact := 0.0
	if speed > 0 {
		timeToImpact = distance / speed
	}

	// Calculate cross-track error if we have trajectory
	crossTrackError := 0.0
	if mission.Trajectory != nil && len(mission.Trajectory.Waypoints) > 1 {
		crossTrackError = ge.calculateCrossTrackError(payload.Position, mission.Trajectory)
	}

	// Check terminal phase
	inTerminal := ge.terminalConfig.Enabled && distance < ge.terminalConfig.ActivationDistance

	// Update metrics (these need unlocked versions since we already have the lock)
	ge.targetingMetrics.HitProbability = ge.calculateHitProbabilityLocked(mission, payload, distance)
	ge.targetingMetrics.CEP = ge.calculateCEPLocked(mission)
	ge.targetingMetrics.TerminalGuidanceOn = inTerminal
	ge.targetingMetrics.TimeToImpact = timeToImpact
	ge.targetingMetrics.ClosingVelocity = speed
	ge.targetingMetrics.CrossTrackError = crossTrackError
	ge.targetingMetrics.WeatherImpact = ge.calculateWeatherImpactLocked()
	ge.targetingMetrics.ECMDetected = len(ge.ecmThreats) > 0
}

func (ge *GuidanceEngine) calculateHitProbabilityLocked(mission *Mission, payload *PayloadState, distance float64) float64 {
	if mission.Trajectory == nil {
		return 0.0
	}
	baseProbability := mission.Trajectory.Confidence
	distanceFactor := 1.0 / (1.0 + distance/10000.0)
	weatherFactor := ge.calculateWeatherImpactLocked()
	ecmFactor := ge.calculateECMImpactLocked(payload.Position)
	terminalBoost := 1.0
	if ge.terminalConfig.Enabled && distance < ge.terminalConfig.ActivationDistance {
		terminalBoost = 1.2
	}
	healthFactor := payload.Health / 100.0
	if healthFactor == 0 {
		healthFactor = 1.0
	}
	hitProbability := baseProbability * distanceFactor * weatherFactor * ecmFactor * terminalBoost * healthFactor
	if hitProbability > 1.0 {
		hitProbability = 1.0
	}
	if hitProbability < 0.0 {
		hitProbability = 0.0
	}
	return hitProbability
}

func (ge *GuidanceEngine) calculateCEPLocked(mission *Mission) float64 {
	baseCEP := 50.0
	if !ge.terminalConfig.Enabled {
		baseCEP = 100.0
	}
	weatherDegradation := 1.0
	if ge.weather != nil {
		if ge.weather.WindSpeed > 20 {
			weatherDegradation += (ge.weather.WindSpeed - 20) * 0.05
		}
		if ge.weather.Visibility < 5000 {
			weatherDegradation += (5000 - ge.weather.Visibility) / 5000 * 0.5
		}
		if ge.weather.Turbulence > 0.3 {
			weatherDegradation += ge.weather.Turbulence * 0.3
		}
	}
	ecmDegradation := 1.0
	for _, ecm := range ge.ecmThreats {
		if ecm.Active {
			ecmDegradation += ecm.Strength * 0.5
		}
	}
	stealthBonus := 1.0
	if mission.StealthRequired {
		stealthBonus = 0.9
	}
	return baseCEP * weatherDegradation * ecmDegradation * stealthBonus
}

func (ge *GuidanceEngine) calculateECMImpactLocked(position Vector3D) float64 {
	if len(ge.ecmThreats) == 0 {
		return 1.0
	}
	impact := 1.0
	for _, ecm := range ge.ecmThreats {
		if !ecm.Active {
			continue
		}
		dist := vectorDistance(position, ecm.Position)
		if dist < ecm.EffectRadius {
			distanceFactor := 1.0 - (dist / ecm.EffectRadius)
			impact -= distanceFactor * ecm.Strength * 0.3
		}
	}
	if impact < 0.3 {
		impact = 0.3
	}
	return impact
}

func (ge *GuidanceEngine) calculateCrossTrackError(position Vector3D, trajectory *Trajectory) float64 {
	if len(trajectory.Waypoints) < 2 {
		return 0.0
	}
	// Find closest segment
	minDist := math.MaxFloat64
	for i := 0; i < len(trajectory.Waypoints)-1; i++ {
		wp1 := trajectory.Waypoints[i].Position
		wp2 := trajectory.Waypoints[i+1].Position
		// Calculate point-to-line distance
		dist := pointToLineDistance(position, wp1, wp2)
		if dist < minDist {
			minDist = dist
		}
	}
	return minDist
}

func pointToLineDistance(point, lineStart, lineEnd Vector3D) float64 {
	// Vector from lineStart to lineEnd
	dx := lineEnd.X - lineStart.X
	dy := lineEnd.Y - lineStart.Y
	dz := lineEnd.Z - lineStart.Z

	// Vector from lineStart to point
	px := point.X - lineStart.X
	py := point.Y - lineStart.Y
	pz := point.Z - lineStart.Z

	// Calculate the parameter t for the closest point on the line
	lineLenSq := dx*dx + dy*dy + dz*dz
	if lineLenSq == 0 {
		return math.Sqrt(px*px + py*py + pz*pz)
	}

	t := (px*dx + py*dy + pz*dz) / lineLenSq
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}

	// Closest point on line segment
	closestX := lineStart.X + t*dx
	closestY := lineStart.Y + t*dy
	closestZ := lineStart.Z + t*dz

	// Distance from point to closest point
	errX := point.X - closestX
	errY := point.Y - closestY
	errZ := point.Z - closestZ

	return math.Sqrt(errX*errX + errY*errY + errZ*errZ)
}

// GetPayloadState retrieves payload state
func (ge *GuidanceEngine) GetPayloadState(payloadID string) (*PayloadState, error) {
	ge.mu.RLock()
	defer ge.mu.RUnlock()

	state, exists := ge.payloads[payloadID]
	if !exists {
		return nil, fmt.Errorf("payload not found: %s", payloadID)
	}
	return state, nil
}

// generateTrajectory creates an optimal trajectory for a mission
func (ge *GuidanceEngine) generateTrajectory(mission *Mission) (*Trajectory, error) {
	trajectory := &Trajectory{
		ID:          uuid.New().String(),
		PayloadID:   mission.PayloadID,
		PayloadType: mission.PayloadType,
		Waypoints:   make([]Waypoint, 0),
		Status:      "planned",
		CreatedAt:   time.Now(),
	}

	// Generate waypoints based on mission parameters
	numWaypoints := 5
	for i := 0; i <= numWaypoints; i++ {
		t := float64(i) / float64(numWaypoints)
		
		// Linear interpolation with altitude variation
		wp := Waypoint{
			ID: uuid.New().String(),
			Position: Vector3D{
				X: mission.StartPosition.X + t*(mission.TargetPosition.X-mission.StartPosition.X),
				Y: mission.StartPosition.Y + t*(mission.TargetPosition.Y-mission.StartPosition.Y),
				Z: mission.StartPosition.Z + t*(mission.TargetPosition.Z-mission.StartPosition.Z),
			},
			Timestamp: time.Now().Add(time.Duration(i*60) * time.Second),
			Stealth:   mission.StealthRequired,
		}

		// Add mid-course altitude for terrain clearance
		if i > 0 && i < numWaypoints {
			if wp.Position.Z < 1000 {
				wp.Position.Z = 1000 // Minimum altitude
			}
		}

		trajectory.Waypoints = append(trajectory.Waypoints, wp)
	}

	// Calculate velocities between waypoints
	for i := 0; i < len(trajectory.Waypoints)-1; i++ {
		curr := &trajectory.Waypoints[i]
		next := trajectory.Waypoints[i+1]
		dt := next.Timestamp.Sub(curr.Timestamp).Seconds()
		if dt > 0 {
			curr.Velocity = Vector3D{
				X: (next.Position.X - curr.Position.X) / dt,
				Y: (next.Position.Y - curr.Position.Y) / dt,
				Z: (next.Position.Z - curr.Position.Z) / dt,
			}
		}
	}

	// Calculate stealth score
	if mission.StealthRequired {
		trajectory.StealthScore = 0.85 // Simulated stealth optimization
	} else {
		trajectory.StealthScore = 0.50
	}

	trajectory.Confidence = 0.92

	return trajectory, nil
}

// telemetryProcessor handles incoming telemetry
func (ge *GuidanceEngine) telemetryProcessor() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ge.ctx.Done():
			return
		case <-ticker.C:
			ge.processTelemetry()
		}
	}
}

// processTelemetry processes current telemetry
func (ge *GuidanceEngine) processTelemetry() {
	ge.mu.RLock()
	defer ge.mu.RUnlock()

	for payloadID, state := range ge.payloads {
		// Check if payload is stale
		if time.Since(state.LastUpdate) > 30*time.Second {
			log.Printf("[%s] Warning: Payload %s telemetry stale", AppName, payloadID)
		}

		// Check fuel/battery levels
		if state.Fuel < 10 || state.Battery < 10 {
			log.Printf("[%s] Warning: Payload %s low resources (fuel=%.1f%%, battery=%.1f%%)",
				AppName, payloadID, state.Fuel, state.Battery)
		}
	}
}

// missionMonitor monitors active missions
func (ge *GuidanceEngine) missionMonitor() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ge.ctx.Done():
			return
		case <-ticker.C:
			ge.checkMissions()
		}
	}
}

// checkMissions checks mission status
func (ge *GuidanceEngine) checkMissions() {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	activeMissions := 0
	for _, mission := range ge.missions {
		if mission.Status == "active" {
			activeMissions++

			// Check if payload reached target
			if state, exists := ge.payloads[mission.PayloadID]; exists {
				dx := state.Position.X - mission.TargetPosition.X
				dy := state.Position.Y - mission.TargetPosition.Y
				dz := state.Position.Z - mission.TargetPosition.Z
				distance := dx*dx + dy*dy + dz*dz

				if distance < 100*100 { // Within 100m
					mission.Status = "completed"
					mission.UpdatedAt = time.Now()
					log.Printf("[%s] Mission %s completed", AppName, mission.ID)
				}
			}
		}
	}

	if activeMissions > 0 {
		log.Printf("[%s] Active missions: %d", AppName, activeMissions)
	}
}

// trajectoryOptimizer continuously optimizes active trajectories
func (ge *GuidanceEngine) trajectoryOptimizer() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ge.ctx.Done():
			return
		case <-ticker.C:
			ge.optimizeTrajectories()
		}
	}
}

// optimizeTrajectories optimizes trajectories based on current conditions
func (ge *GuidanceEngine) optimizeTrajectories() {
	ge.mu.Lock()
	defer ge.mu.Unlock()

	for _, mission := range ge.missions {
		if mission.Status != "active" || mission.Trajectory == nil {
			continue
		}

		// In production, this would:
		// 1. Get current threat data from Giru
		// 2. Get terrain data from Silenus
		// 3. Recalculate optimal path
		// 4. Update trajectory

		mission.UpdatedAt = time.Now()
	}
}

// HTTPServer provides REST API
type HTTPServer struct {
	engine *GuidanceEngine
	server *http.Server
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(engine *GuidanceEngine, port int) *HTTPServer {
	mux := http.NewServeMux()
	
	s := &HTTPServer{
		engine: engine,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
	}

	// Register routes
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/api/v1/missions", s.missionsHandler)
	mux.HandleFunc("/api/v1/missions/target/", s.missionTargetHandler)
	mux.HandleFunc("/api/v1/missions/", s.missionHandler)
	mux.HandleFunc("/api/v1/payloads", s.payloadsHandler)
	mux.HandleFunc("/api/v1/payloads/", s.payloadHandler)
	mux.HandleFunc("/api/v1/trajectories/", s.trajectoryHandler)
	mux.HandleFunc("/api/v1/status", s.statusHandler)
	mux.HandleFunc("/api/v1/metrics/targeting", s.targetingMetricsHandler)
	
	// Access Control routes
	mux.HandleFunc("/api/v1/auth/login", s.loginHandler)
	mux.HandleFunc("/api/v1/auth/validate", s.validateSessionHandler)
	mux.HandleFunc("/api/v1/users", s.usersHandler)
	mux.HandleFunc("/api/v1/terminals", s.terminalsHandler)
	mux.HandleFunc("/api/v1/clearance/levels", s.clearanceLevelsHandler)
	
	// Live Feed routes
	mux.HandleFunc("/api/v1/feeds", s.feedsHandler)
	mux.HandleFunc("/api/v1/feeds/", s.feedHandler)
	mux.HandleFunc("/api/v1/telemetry/", s.telemetryHandler)
	mux.HandleFunc("/api/v1/wifi/routers", s.wifiRoutersHandler)
	mux.HandleFunc("/api/v1/wifi/imaging", s.wifiImagingHandler)

	// Enhanced guidance routes
	mux.HandleFunc("/api/v1/guidance/terminal", s.terminalGuidanceHandler)
	mux.HandleFunc("/api/v1/guidance/weather", s.weatherHandler)
	mux.HandleFunc("/api/v1/guidance/ecm", s.ecmHandler)
	mux.HandleFunc("/api/v1/guidance/abort/", s.abortMissionHandler)
	mux.HandleFunc("/api/v1/guidance/probability/", s.hitProbabilityHandler)

	return s
}

// Start begins the HTTP server
func (s *HTTPServer) Start() error {
	log.Printf("[%s] HTTP server listening on %s", AppName, s.server.Addr)
	return s.server.ListenAndServe()
}

// Stop halts the HTTP server
func (s *HTTPServer) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// healthHandler handles health checks
func (s *HTTPServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"version": Version,
		"service": AppName,
	})
}

// statusHandler returns system status
func (s *HTTPServer) statusHandler(w http.ResponseWriter, r *http.Request) {
	missions := s.engine.GetAllMissions()
	
	activeMissions := 0
	completedMissions := 0
	for _, m := range missions {
		switch m.Status {
		case "active":
			activeMissions++
		case "completed":
			completedMissions++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service":           AppName,
		"version":           Version,
		"uptime":            time.Since(startTime).String(),
		"activeMissions":    activeMissions,
		"completedMissions": completedMissions,
		"totalMissions":     len(missions),
	})
}

// missionsHandler handles mission list/create
func (s *HTTPServer) missionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		missions := s.engine.GetAllMissions()
		json.NewEncoder(w).Encode(missions)

	case http.MethodPost:
		var mission Mission
		if err := json.NewDecoder(r.Body).Decode(&mission); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := s.engine.CreateMission(&mission); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(mission)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// missionHandler handles individual mission operations
func (s *HTTPServer) missionHandler(w http.ResponseWriter, r *http.Request) {
	missionID := r.URL.Path[len("/api/v1/missions/"):]
	if missionID == "" {
		http.Error(w, "Mission ID required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		mission, err := s.engine.GetMission(missionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(mission)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// missionTargetHandler updates mission target position
func (s *HTTPServer) missionTargetHandler(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w)
	if r.Method == "OPTIONS" {
		return
	}

	missionID := r.URL.Path[len("/api/v1/missions/target/"):]
	if missionID == "" {
		http.Error(w, "Mission ID required", http.StatusBadRequest)
		return
	}

	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var target Vector3D
	if err := json.Unmarshal(body, &target); err != nil {
		var payload struct {
			TargetPosition Vector3D `json:"targetPosition"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, "Invalid target payload", http.StatusBadRequest)
			return
		}
		target = payload.TargetPosition
	}

	trajectory, err := s.engine.UpdateMissionTarget(missionID, target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"missionId":   missionID,
		"trajectory":  trajectory,
		"target":      target,
	})
}

// payloadsHandler handles payload list
func (s *HTTPServer) payloadsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		s.engine.mu.RLock()
		payloads := make([]*PayloadState, 0, len(s.engine.payloads))
		for _, p := range s.engine.payloads {
			payloads = append(payloads, p)
		}
		s.engine.mu.RUnlock()
		json.NewEncoder(w).Encode(payloads)

	case http.MethodPost:
		var state PayloadState
		if err := json.NewDecoder(r.Body).Decode(&state); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.engine.UpdatePayloadState(&state)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(state)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// payloadHandler handles individual payload operations
func (s *HTTPServer) payloadHandler(w http.ResponseWriter, r *http.Request) {
	payloadID := r.URL.Path[len("/api/v1/payloads/"):]
	if payloadID == "" {
		http.Error(w, "Payload ID required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		state, err := s.engine.GetPayloadState(payloadID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(state)

	case http.MethodPut:
		var state PayloadState
		if err := json.NewDecoder(r.Body).Decode(&state); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		state.ID = payloadID
		s.engine.UpdatePayloadState(&state)
		json.NewEncoder(w).Encode(state)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// trajectoryHandler handles trajectory operations
func (s *HTTPServer) trajectoryHandler(w http.ResponseWriter, r *http.Request) {
	trajectoryID := r.URL.Path[len("/api/v1/trajectories/"):]
	if trajectoryID == "" {
		http.Error(w, "Trajectory ID required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		s.engine.mu.RLock()
		trajectory, exists := s.engine.trajectories[trajectoryID]
		s.engine.mu.RUnlock()

		if !exists {
			http.Error(w, "Trajectory not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(trajectory)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// targetingMetricsHandler returns live targeting performance metrics
func (s *HTTPServer) targetingMetricsHandler(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w)
	if r.Method == "OPTIONS" {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := s.engine.GetTargetingMetrics()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// =============================================================================
// ACCESS CONTROL HANDLERS
// =============================================================================

// loginHandler handles user login
func (s *HTTPServer) loginHandler(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w)
	if r.Method == "OPTIONS" {
		return
	}
	
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Find user
	var foundUser *User
	s.engine.mu.RLock()
	for _, user := range s.engine.users {
		if user.Username == req.Username {
			foundUser = user
			break
		}
	}
	s.engine.mu.RUnlock()

	if foundUser == nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Create session
	session := &Session{
		ID:        uuid.New().String(),
		UserID:    foundUser.ID,
		Token:     uuid.New().String(),
		Clearance: foundUser.Clearance,
		ExpiresAt: time.Now().Add(4 * time.Hour),
	}

	s.engine.mu.Lock()
	s.engine.sessions[session.ID] = session
	s.engine.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"sessionId":     session.ID,
		"token":         session.Token,
		"userId":        foundUser.ID,
		"username":      foundUser.Username,
		"clearance":     foundUser.Clearance,
		"clearanceName": getClearanceName(foundUser.Clearance),
		"expiresAt":     session.ExpiresAt,
	})
}

// validateSessionHandler validates a session
func (s *HTTPServer) validateSessionHandler(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w)
	if r.Method == "OPTIONS" {
		return
	}

	sessionID := r.Header.Get("X-Session-ID")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	s.engine.mu.RLock()
	session, exists := s.engine.sessions[sessionID]
	s.engine.mu.RUnlock()

	if !exists || time.Now().After(session.ExpiresAt) {
		http.Error(w, "Invalid or expired session", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

// usersHandler handles user list
func (s *HTTPServer) usersHandler(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w)
	if r.Method == "OPTIONS" {
		return
	}

	w.Header().Set("Content-Type", "application/json")

	s.engine.mu.RLock()
	users := make([]*User, 0, len(s.engine.users))
	for _, user := range s.engine.users {
		users = append(users, user)
	}
	s.engine.mu.RUnlock()

	json.NewEncoder(w).Encode(users)
}

// terminalsHandler handles terminal list
func (s *HTTPServer) terminalsHandler(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w)
	if r.Method == "OPTIONS" {
		return
	}

	w.Header().Set("Content-Type", "application/json")

	clearance := getClearanceFromHeader(r)

	s.engine.mu.RLock()
	terminals := make([]*Terminal, 0)
	for _, terminal := range s.engine.terminals {
		if terminal.Clearance <= clearance {
			terminals = append(terminals, terminal)
		}
	}
	s.engine.mu.RUnlock()

	json.NewEncoder(w).Encode(terminals)
}

// clearanceLevelsHandler returns clearance level information
func (s *HTTPServer) clearanceLevelsHandler(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w)
	if r.Method == "OPTIONS" {
		return
	}

	w.Header().Set("Content-Type", "application/json")

	levels := []map[string]interface{}{
		{"level": 0, "name": "PUBLIC", "description": "Public access - basic info", "color": "#22c55e"},
		{"level": 1, "name": "CIVILIAN", "description": "Civilian operations", "color": "#3b82f6"},
		{"level": 2, "name": "MILITARY", "description": "Military operations", "color": "#f59e0b"},
		{"level": 3, "name": "GOVERNMENT", "description": "Government classified", "color": "#8b5cf6"},
		{"level": 4, "name": "SECRET", "description": "Secret operations", "color": "#ef4444"},
		{"level": 5, "name": "ULTRA", "description": "Highest classification", "color": "#ec4899"},
	}

	json.NewEncoder(w).Encode(levels)
}

// =============================================================================
// LIVE FEED HANDLERS
// =============================================================================

// feedsHandler handles feed list/create
func (s *HTTPServer) feedsHandler(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w)
	if r.Method == "OPTIONS" {
		return
	}

	w.Header().Set("Content-Type", "application/json")

	clearance := getClearanceFromHeader(r)

	switch r.Method {
	case http.MethodGet:
		s.engine.mu.RLock()
		feeds := make([]*LiveFeed, 0)
		for _, feed := range s.engine.feeds {
			if feed.Clearance <= clearance && feed.Status == "active" {
				feeds = append(feeds, feed)
			}
		}
		s.engine.mu.RUnlock()

		json.NewEncoder(w).Encode(feeds)

	case http.MethodPost:
		var feed LiveFeed
		if err := json.NewDecoder(r.Body).Decode(&feed); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if feed.ID == "" {
			feed.ID = uuid.New().String()
		}
		feed.Status = "active"
		feed.StartedAt = time.Now()

		s.engine.mu.Lock()
		s.engine.feeds[feed.ID] = &feed
		s.engine.mu.Unlock()

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(feed)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// feedHandler handles individual feed operations
func (s *HTTPServer) feedHandler(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w)
	if r.Method == "OPTIONS" {
		return
	}

	feedID := r.URL.Path[len("/api/v1/feeds/"):]
	if feedID == "" {
		http.Error(w, "Feed ID required", http.StatusBadRequest)
		return
	}

	clearance := getClearanceFromHeader(r)

	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		s.engine.mu.RLock()
		feed, exists := s.engine.feeds[feedID]
		s.engine.mu.RUnlock()

		if !exists {
			http.Error(w, "Feed not found", http.StatusNotFound)
			return
		}

		if feed.Clearance > clearance {
			http.Error(w, "Insufficient clearance", http.StatusForbidden)
			return
		}

		json.NewEncoder(w).Encode(feed)

	case http.MethodDelete:
		s.engine.mu.Lock()
		if feed, exists := s.engine.feeds[feedID]; exists {
			feed.Status = "ended"
		}
		s.engine.mu.Unlock()

		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// telemetryHandler handles telemetry requests
func (s *HTTPServer) telemetryHandler(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w)
	if r.Method == "OPTIONS" {
		return
	}

	payloadID := r.URL.Path[len("/api/v1/telemetry/"):]
	if payloadID == "" {
		http.Error(w, "Payload ID required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	state, err := s.engine.GetPayloadState(payloadID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	speed := math.Sqrt(state.Velocity.X*state.Velocity.X + state.Velocity.Y*state.Velocity.Y + state.Velocity.Z*state.Velocity.Z)
	telemetry := TelemetryFrame{
		PayloadID:  state.ID,
		Position:   state.Position,
		Velocity:   state.Velocity,
		Heading:    state.Heading,
		Speed:      speed,
		Fuel:       state.Fuel,
		Battery:    state.Battery,
		Status:     state.Status,
		Timestamp:  state.LastUpdate,
	}

	json.NewEncoder(w).Encode(telemetry)
}

// wifiRoutersHandler registers or lists WiFi imaging routers
func (s *HTTPServer) wifiRoutersHandler(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w)
	if r.Method == "OPTIONS" {
		return
	}

	switch r.Method {
	case http.MethodGet:
		routers := s.engine.GetWiFiRouters()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(routers)

	case http.MethodPost:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		var routers []sensors.WiFiRouter
		if err := json.Unmarshal(body, &routers); err != nil {
			var single sensors.WiFiRouter
			if err := json.Unmarshal(body, &single); err != nil {
				http.Error(w, "Invalid router payload", http.StatusBadRequest)
				return
			}
			routers = []sensors.WiFiRouter{single}
		}

		for _, router := range routers {
			s.engine.RegisterWiFiRouter(router)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"registered": len(routers),
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// wifiImagingHandler ingests WiFi CSI frames for through-wall estimation
func (s *HTTPServer) wifiImagingHandler(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w)
	if r.Method == "OPTIONS" {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var frame sensors.WiFiImagingFrame
	if err := json.NewDecoder(r.Body).Decode(&frame); err != nil {
		http.Error(w, "Invalid imaging payload", http.StatusBadRequest)
		return
	}
	if frame.Timestamp.IsZero() {
		frame.Timestamp = time.Now()
	}
	observations, err := s.engine.ProcessWiFiImaging(frame)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"observations": observations,
	})
}

// =============================================================================
// ENHANCED GUIDANCE HANDLERS
// =============================================================================

// terminalGuidanceHandler manages terminal guidance configuration
func (s *HTTPServer) terminalGuidanceHandler(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w)
	if r.Method == "OPTIONS" {
		return
	}

	switch r.Method {
	case http.MethodGet:
		config := s.engine.GetTerminalGuidanceConfig()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)

	case http.MethodPost, http.MethodPut:
		var config TerminalGuidanceConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, "Invalid config payload", http.StatusBadRequest)
			return
		}
		s.engine.EnableTerminalGuidance(config)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "configured",
			"config":  config,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// weatherHandler manages weather condition updates
func (s *HTTPServer) weatherHandler(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w)
	if r.Method == "OPTIONS" {
		return
	}

	switch r.Method {
	case http.MethodGet:
		weather := s.engine.GetWeather()
		w.Header().Set("Content-Type", "application/json")
		if weather == nil {
			json.NewEncoder(w).Encode(map[string]string{"status": "no weather data"})
		} else {
			json.NewEncoder(w).Encode(weather)
		}

	case http.MethodPost, http.MethodPut:
		var weather WeatherCondition
		if err := json.NewDecoder(r.Body).Decode(&weather); err != nil {
			http.Error(w, "Invalid weather payload", http.StatusBadRequest)
			return
		}
		s.engine.UpdateWeather(weather)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":        "updated",
			"weather":       weather,
			"weatherImpact": s.engine.GetTargetingMetrics().WeatherImpact,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ecmHandler manages ECM threat registration
func (s *HTTPServer) ecmHandler(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w)
	if r.Method == "OPTIONS" {
		return
	}

	switch r.Method {
	case http.MethodGet:
		threats := s.engine.GetECMThreats()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ecmThreats":  threats,
			"ecmDetected": len(threats) > 0,
		})

	case http.MethodPost:
		var ecm ECMThreat
		if err := json.NewDecoder(r.Body).Decode(&ecm); err != nil {
			http.Error(w, "Invalid ECM payload", http.StatusBadRequest)
			return
		}
		s.engine.RegisterECMThreat(ecm)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "registered",
			"ecm":    ecm,
		})

	case http.MethodDelete:
		ecmID := r.URL.Query().Get("id")
		if ecmID == "" {
			http.Error(w, "ECM ID required", http.StatusBadRequest)
			return
		}
		s.engine.ClearECMThreat(ecmID)
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// abortMissionHandler handles mission abort requests
func (s *HTTPServer) abortMissionHandler(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w)
	if r.Method == "OPTIONS" {
		return
	}

	missionID := r.URL.Path[len("/api/v1/guidance/abort/"):]
	if missionID == "" {
		// Return list of aborted missions
		aborted := s.engine.GetAbortedMissions()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(aborted)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Reason       string `json:"reason"`
		ReturnToBase bool   `json:"returnToBase"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Reason = "manual_abort"
		req.ReturnToBase = false
	}

	if err := s.engine.AbortMission(missionID, req.Reason, req.ReturnToBase); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":       "aborted",
		"missionId":    missionID,
		"reason":       req.Reason,
		"returnToBase": req.ReturnToBase,
	})
}

// hitProbabilityHandler returns hit probability for a mission
func (s *HTTPServer) hitProbabilityHandler(w http.ResponseWriter, r *http.Request) {
	s.setCORSHeaders(w)
	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	missionID := r.URL.Path[len("/api/v1/guidance/probability/"):]
	if missionID == "" {
		http.Error(w, "Mission ID required", http.StatusBadRequest)
		return
	}

	// Update enhanced metrics for this mission
	s.engine.UpdateEnhancedTargetingMetrics(missionID)

	hitProb := s.engine.CalculateHitProbability(missionID)
	cep := s.engine.CalculateCEP(missionID)
	inTerminal, distance := s.engine.CheckTerminalPhase(missionID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"missionId":       missionID,
		"hitProbability":  hitProb,
		"cep":             cep,
		"inTerminalPhase": inTerminal,
		"distanceToTarget": distance,
		"metrics":         s.engine.GetTargetingMetrics(),
	})
}

// setCORSHeaders sets CORS headers
func (s *HTTPServer) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Session-ID, X-Clearance")
}

// getClearanceFromHeader extracts clearance level from header
func getClearanceFromHeader(r *http.Request) ClearanceLevel {
	clearanceStr := r.Header.Get("X-Clearance")
	switch clearanceStr {
	case "PUBLIC", "0":
		return ClearancePublic
	case "CIVILIAN", "1":
		return ClearanceCivilian
	case "MILITARY", "2":
		return ClearanceMilitary
	case "GOVERNMENT", "GOV", "3":
		return ClearanceGov
	case "SECRET", "4":
		return ClearanceSecret
	case "ULTRA", "5":
		return ClearanceUltra
	default:
		return ClearancePublic
	}
}

var startTime time.Time

func main() {
	startTime = time.Now()

	// Parse command-line flags
	httpPort := flag.Int("http-port", 8089, "HTTP API port")
	metricsPort := flag.Int("metrics-port", 9089, "Metrics port")
	nysusEndpoint := flag.String("nysus", "http://localhost:8080", "Nysus API endpoint")
	satnetEndpoint := flag.String("satnet", "http://localhost:8081", "Sat_Net endpoint")
	giruEndpoint := flag.String("giru", "http://localhost:9090", "Giru endpoint")
	natsURL := flag.String("nats-url", "nats://localhost:4222", "NATS server URL")
	enableStealth := flag.Bool("stealth", true, "Enable stealth optimization")
	enablePrediction := flag.Bool("prediction", true, "Enable trajectory prediction")
	enableNATS := flag.Bool("enable-nats", true, "Enable NATS integration")
	enableSensorFusion := flag.Bool("enable-sensors", true, "Enable sensor fusion")
	enableWiFiImaging := flag.Bool("enable-wifi", true, "Enable WiFi CSI imaging")
	replanMs := flag.Int("replan-ms", 250, "Rapid replan interval in milliseconds")
	flag.Parse()

	log.Printf("╔══════════════════════════════════════════════════════════════╗")
	log.Printf("║  PRICILLA - Precision Engagement & Routing Control           ║")
	log.Printf("║           with Integrated Learning Architecture              ║")
	log.Printf("║                      Version %s                            ║", Version)
	log.Printf("╚══════════════════════════════════════════════════════════════╝")

	config := Config{
		HTTPPort:           *httpPort,
		MetricsPort:        *metricsPort,
		NysusEndpoint:      *nysusEndpoint,
		SatNetEndpoint:     *satnetEndpoint,
		GiruEndpoint:       *giruEndpoint,
		NATSURL:            *natsURL,
		EnableStealth:      *enableStealth,
		EnablePrediction:   *enablePrediction,
		EnableMultiPayload: true,
		EnableNATS:         *enableNATS,
		EnableSensorFusion: *enableSensorFusion,
		EnableWiFiImaging:  *enableWiFiImaging,
		ReplanInterval:     time.Duration(*replanMs) * time.Millisecond,
		LogLevel:           "info",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create guidance engine
	engine := NewGuidanceEngine(config)
	if err := engine.Start(ctx); err != nil {
		log.Fatalf("Failed to start guidance engine: %v", err)
	}

	// Create HTTP server
	httpServer := NewHTTPServer(engine, config.HTTPPort)
	go func() {
		if err := httpServer.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Create metrics server with Prometheus handler
	metricsMux := http.NewServeMux()
	
	// Use Prometheus handler for /metrics endpoint
	metricsMux.Handle("/metrics", promhttp.Handler())
	
	// Health check endpoint
	metricsMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	// Readiness check endpoint (checks NATS and sensor fusion status)
	metricsMux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		status := map[string]interface{}{
			"status": "ready",
			"uptime": time.Since(startTime).String(),
		}
		
		// Check NATS connection
		if engine.GetNATSBridge() != nil {
			status["nats_connected"] = engine.GetNATSBridge().IsConnected()
			status["nats_running"] = engine.GetNATSBridge().IsRunning()
		} else {
			status["nats_enabled"] = false
		}
		
		// Check sensor fusion
		if engine.GetSensorFusion() != nil {
			status["sensor_fusion_running"] = engine.GetSensorFusion().IsRunning()
			activeSensors, _ := engine.GetSensorFusion().GetActiveSensors()
			status["active_sensors"] = activeSensors
		} else {
			status["sensor_fusion_enabled"] = false
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(status)
	})
	
	// NATS stats endpoint
	metricsMux.HandleFunc("/nats/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if engine.GetNATSBridge() != nil {
			json.NewEncoder(w).Encode(engine.GetNATSBridge().Stats())
		} else {
			json.NewEncoder(w).Encode(map[string]string{"status": "nats_disabled"})
		}
	})
	
	// Sensor fusion stats endpoint
	metricsMux.HandleFunc("/sensors/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if engine.GetSensorFusion() != nil {
			json.NewEncoder(w).Encode(engine.GetSensorFusion().GetAllSensorHealth())
		} else {
			json.NewEncoder(w).Encode(map[string]string{"status": "sensor_fusion_disabled"})
		}
	})
	
	metricsMux.HandleFunc("/sensors/state", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if engine.GetSensorFusion() != nil {
			json.NewEncoder(w).Encode(engine.GetSensorFusion().GetFusedState())
		} else {
			json.NewEncoder(w).Encode(map[string]string{"status": "sensor_fusion_disabled"})
		}
	})

	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.MetricsPort),
		Handler: metricsMux,
	}
	
	go func() {
		log.Printf("[%s] Metrics server listening on :%d", AppName, config.MetricsPort)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	log.Printf("[%s] System ready", AppName)
	log.Printf("[%s] HTTP API: http://localhost:%d", AppName, config.HTTPPort)
	log.Printf("[%s] Prometheus Metrics: http://localhost:%d/metrics", AppName, config.MetricsPort)

	// Print capabilities
	log.Printf("[%s] Capabilities:", AppName)
	log.Printf("[%s]   - Multi-payload guidance (Hunoid, UAV, Rocket, Missile, Spacecraft)", AppName)
	log.Printf("[%s]   - Stealth trajectory optimization: %v", AppName, config.EnableStealth)
	log.Printf("[%s]   - AI trajectory prediction: %v", AppName, config.EnablePrediction)
	log.Printf("[%s]   - NATS real-time integration: %v", AppName, config.EnableNATS)
	log.Printf("[%s]   - Multi-sensor fusion (EKF): %v", AppName, config.EnableSensorFusion)
	log.Printf("[%s]   - Through-wall WiFi imaging: %v", AppName, config.EnableWiFiImaging)
	log.Printf("[%s]   - Rapid replanning: %s", AppName, config.ReplanInterval)
	log.Printf("[%s]   - Terminal guidance with precision approach: enabled", AppName)
	log.Printf("[%s]   - Hit probability estimation: enabled", AppName)
	log.Printf("[%s]   - ECM/Jamming detection & adaptation: enabled", AppName)
	log.Printf("[%s]   - Weather impact modeling: enabled", AppName)
	log.Printf("[%s]   - Mission abort/RTB capability: enabled", AppName)
	log.Printf("[%s]   - Full ASGARD integration (Silenus, Hunoid, Sat_Net, Giru, Nysus)", AppName)
	log.Printf("[%s]   - Live Feed Streaming with tiered access", AppName)
	log.Printf("[%s]   - Mission Hub with 6 clearance levels (PUBLIC to ULTRA)", AppName)
	log.Printf("[%s]   - Access terminals and command interfaces", AppName)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Printf("[%s] Shutting down...", AppName)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	httpServer.Stop(shutdownCtx)
	metricsServer.Shutdown(shutdownCtx)
	engine.Stop()

	log.Printf("[%s] Shutdown complete", AppName)
}
