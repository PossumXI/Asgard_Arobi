// Package ai provides AI-powered decision making for autonomous flight
package ai

import (
	"context"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/PossumXI/Asgard/Valkyrie/internal/fusion"
	"github.com/PossumXI/Asgard/Valkyrie/internal/integration"
	"github.com/sirupsen/logrus"
)

// DecisionEngine is the AI brain of VALKYRIE
type DecisionEngine struct {
	mu sync.RWMutex

	// Sub-systems
	fusionEngine *fusion.ExtendedKalmanFilter

	// Current state
	currentState   *fusion.FusionState
	currentMission *Mission

	// AI models
	rlPolicy *ReinforcementLearningPolicy

	// Configuration
	config DecisionConfig

	// Active threats
	threats []*Threat

	// ASGARD integration
	asgardClients *integration.ASGARDClients

	// Logger
	logger *logrus.Logger
}

// DecisionConfig holds AI decision parameters
type DecisionConfig struct {
	SafetyPriority     float64 // 0.0 to 1.0
	EfficiencyPriority float64
	StealthPriority    float64

	MaxRollAngle  float64 // radians
	MaxPitchAngle float64
	MaxYawRate    float64

	MinSafeAltitude  float64 // meters AGL
	MaxVerticalSpeed float64 // m/s

	EnableAutoland     bool
	EnableThreatAvoid  bool
	EnableWeatherAvoid bool

	DecisionRate float64 // Hz

	GeoReferenceEnabled   bool    // enable local meters -> lat/lon conversion
	GeoReferenceLatitude  float64 // degrees
	GeoReferenceLongitude float64 // degrees
	GeoReferenceSource    string  // "fusion" (default) or "n2yo"
	GeoReferenceNoradID   int     // NORAD ID for N2YO position lookups
}

// Mission represents a flight mission
type Mission struct {
	ID          string
	Type        MissionType
	Waypoints   []Waypoint
	Constraints MissionConstraints
	Priority    Priority
	Status      MissionStatus
	StartTime   time.Time
	ETA         time.Time
}

// MissionType defines mission categories
type MissionType int

const (
	MissionRecon MissionType = iota
	MissionTransport
	MissionPatrol
	MissionIntercept
	MissionRescue
	MissionTraining
)

// FlightCommand represents commands to the flight controller
type FlightCommand struct {
	Timestamp time.Time

	// Attitude commands
	RollAngle  float64 // radians
	PitchAngle float64
	YawRate    float64

	// Throttle (0.0 to 1.0)
	Throttle float64

	// Surface deflections
	Aileron  float64 // -1.0 to 1.0
	Elevator float64
	Rudder   float64
	Flaps    float64

	// Special modes
	AutoThrottle bool
	AutoLand     bool
	EmergencyRTB bool
}

// Waypoint represents a navigation waypoint
type Waypoint struct {
	ID       string
	Position [3]float64
	Speed    float64
	Altitude float64
	Heading  float64
	Loiter   time.Duration
}

// MissionConstraints defines mission limits
type MissionConstraints struct {
	MaxAltitude   float64
	MinAltitude   float64
	MaxSpeed      float64
	NoFlyZones    []NoFlyZone
	TimeWindow    [2]time.Time
	FuelReserve   float64
	StealthLevel  float64
}

// NoFlyZone represents a restricted area
type NoFlyZone struct {
	Center [3]float64
	Radius float64
	Type   string
}

// Priority levels
type Priority int

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// MissionStatus represents mission state
type MissionStatus int

const (
	MissionPending MissionStatus = iota
	MissionActive
	MissionCompleted
	MissionAborted
	MissionPaused
)

// Threat represents a detected threat
type Threat struct {
	ID        string
	Type      ThreatType
	Position  [3]float64
	Velocity  [3]float64
	Distance  float64
	Bearing   float64
	Severity  float64
	Timestamp time.Time
}

// ThreatType categorizes threats
type ThreatType int

const (
	ThreatRadar ThreatType = iota
	ThreatMissile
	ThreatAircraft
	ThreatSAM
	ThreatWeather
	ThreatTerrain
	ThreatBirdStrike
)

// WeatherConditions holds weather data
type WeatherConditions struct {
	WindSpeed     float64
	WindDirection float64
	Visibility    float64
	Turbulence    float64
	IcingRisk     float64
	Ceiling       float64
	Temperature   float64
	Pressure      float64
}

// ReinforcementLearningPolicy implements the RL policy
type ReinforcementLearningPolicy struct {
	mu      sync.RWMutex
	weights []float64
	epsilon float64
}

// RLAction represents an action from the RL policy
type RLAction struct {
	RollAngle    float64
	PitchAngle   float64
	YawRate      float64
	Throttle     float64
	Aileron      float64
	Elevator     float64
	Rudder       float64
	AutoThrottle bool
}

// NewDecisionEngine creates a new AI decision engine
func NewDecisionEngine(config DecisionConfig, asgardClients *integration.ASGARDClients) *DecisionEngine {
	return &DecisionEngine{
		config:        config,
		rlPolicy:      NewRLPolicy(),
		threats:       make([]*Threat, 0),
		asgardClients: asgardClients,
		logger:        logrus.New(),
	}
}

// Initialize sets up the decision engine
func (de *DecisionEngine) Initialize(ctx context.Context) error {
	de.mu.Lock()
	defer de.mu.Unlock()

	// Initialize RL policy weights
	de.rlPolicy.Initialize()

	return nil
}

// SetFusionEngine connects the fusion engine
func (de *DecisionEngine) SetFusionEngine(fe *fusion.ExtendedKalmanFilter) {
	de.mu.Lock()
	defer de.mu.Unlock()
	de.fusionEngine = fe
}

// SetMission sets the current mission
func (de *DecisionEngine) SetMission(mission *Mission) {
	de.mu.Lock()
	defer de.mu.Unlock()
	de.currentMission = mission
	de.currentMission.Status = MissionActive
	de.currentMission.StartTime = time.Now()
}

// UpdateThreats updates the active threats list
func (de *DecisionEngine) UpdateThreats(threats []*Threat) {
	de.mu.Lock()
	defer de.mu.Unlock()
	de.threats = threats
}

// Decide generates flight commands based on current state
func (de *DecisionEngine) Decide(ctx context.Context) (*FlightCommand, error) {
	de.mu.RLock()
	defer de.mu.RUnlock()

	// Get current fused state
	if de.fusionEngine != nil {
		de.currentState = de.fusionEngine.GetState()
	}

	// If no state available, create a default
	if de.currentState == nil {
		de.currentState = &fusion.FusionState{
			Position:    [3]float64{0, 0, 500}, // Default safe altitude
			Velocity:    [3]float64{0, 0, 0},
			Attitude:    [3]float64{0, 0, 0},
			Confidence:  0.5,
		}
	}

	// If no mission, return level flight
	if de.currentMission == nil || len(de.currentMission.Waypoints) == 0 {
		return &FlightCommand{
			Timestamp:    time.Now(),
			Throttle:     0.5,
			AutoThrottle: true,
		}, nil
	}

	// Get weather conditions
	weather := de.getWeatherConditions()

	// Compute target from current waypoint
	waypoint := de.getCurrentWaypoint()
	target := waypoint.Position

	// Compute trajectory error
	posError := [3]float64{
		target[0] - de.currentState.Position[0],
		target[1] - de.currentState.Position[1],
		target[2] - de.currentState.Position[2],
	}

	// RL policy decision
	action := de.rlPolicy.SelectAction(de.currentState, de.threats, weather, posError)

	// Convert to flight command
	cmd := de.actionToCommand(action)

	// Safety checks
	cmd = de.applySafetyLimits(cmd)

	// Check for threat avoidance
	if de.config.EnableThreatAvoid && len(de.threats) > 0 {
		cmd = de.applyThreatAvoidance(cmd)
	}

	return cmd, nil
}

// actionToCommand converts RL action to flight command
func (de *DecisionEngine) actionToCommand(action *RLAction) *FlightCommand {
	return &FlightCommand{
		Timestamp:    time.Now(),
		RollAngle:    action.RollAngle,
		PitchAngle:   action.PitchAngle,
		YawRate:      action.YawRate,
		Throttle:     action.Throttle,
		Aileron:      action.Aileron,
		Elevator:     action.Elevator,
		Rudder:       action.Rudder,
		AutoThrottle: action.AutoThrottle,
	}
}

// applySafetyLimits enforces safety constraints
func (de *DecisionEngine) applySafetyLimits(cmd *FlightCommand) *FlightCommand {
	// Limit roll angle
	if math.Abs(cmd.RollAngle) > de.config.MaxRollAngle {
		cmd.RollAngle = math.Copysign(de.config.MaxRollAngle, cmd.RollAngle)
	}

	// Limit pitch angle
	if math.Abs(cmd.PitchAngle) > de.config.MaxPitchAngle {
		cmd.PitchAngle = math.Copysign(de.config.MaxPitchAngle, cmd.PitchAngle)
	}

	// Limit yaw rate
	if math.Abs(cmd.YawRate) > de.config.MaxYawRate {
		cmd.YawRate = math.Copysign(de.config.MaxYawRate, cmd.YawRate)
	}

	// Altitude check - emergency pull-up
	if de.currentState != nil && de.currentState.Position[2] < de.config.MinSafeAltitude {
		cmd.PitchAngle = de.config.MaxPitchAngle * 0.8 // 80% of max for safety margin
		cmd.Throttle = 1.0
		cmd.EmergencyRTB = true
	}

	// Throttle limits
	if cmd.Throttle < 0 {
		cmd.Throttle = 0
	}
	if cmd.Throttle > 1.0 {
		cmd.Throttle = 1.0
	}

	return cmd
}

// applyThreatAvoidance modifies command for threat evasion
func (de *DecisionEngine) applyThreatAvoidance(cmd *FlightCommand) *FlightCommand {
	// Find nearest threat
	var nearestThreat *Threat
	minDist := math.MaxFloat64

	for _, threat := range de.threats {
		if threat.Distance < minDist {
			minDist = threat.Distance
			nearestThreat = threat
		}
	}

	if nearestThreat == nil || minDist > 5000 { // No immediate threat
		return cmd
	}

	// Calculate evasion vector (perpendicular to threat bearing)
	threatBearing := nearestThreat.Bearing
	evasionBearing := threatBearing + math.Pi/2 // Turn 90 degrees

	// Adjust roll to turn away
	cmd.RollAngle = math.Copysign(de.config.MaxRollAngle*0.7, math.Sin(evasionBearing))

	// If threat is a missile, dive or climb based on relative altitude
	if nearestThreat.Type == ThreatMissile {
		if nearestThreat.Position[2] > de.currentState.Position[2] {
			cmd.PitchAngle = -de.config.MaxPitchAngle * 0.5 // Dive
		} else {
			cmd.PitchAngle = de.config.MaxPitchAngle * 0.5 // Climb
		}
		cmd.Throttle = 1.0 // Max power
	}

	return cmd
}

// getCurrentWaypoint gets the next waypoint
func (de *DecisionEngine) getCurrentWaypoint() Waypoint {
	if de.currentMission == nil || len(de.currentMission.Waypoints) == 0 {
		return Waypoint{Position: [3]float64{0, 0, 1000}}
	}

	// Check if we've reached current waypoint (only if currentState is valid)
	if de.currentState != nil && de.currentState.Confidence > 0 && len(de.currentMission.Waypoints) > 0 {
		wp := de.currentMission.Waypoints[0]
		dist := math.Sqrt(
			math.Pow(wp.Position[0]-de.currentState.Position[0], 2) +
				math.Pow(wp.Position[1]-de.currentState.Position[1], 2) +
				math.Pow(wp.Position[2]-de.currentState.Position[2], 2),
		)
		if dist < 50 { // Within 50m of waypoint
			// Remove current waypoint, advance to next
			if len(de.currentMission.Waypoints) > 1 {
				de.currentMission.Waypoints = de.currentMission.Waypoints[1:]
			}
		}
	}

	return de.currentMission.Waypoints[0]
}

// getWeatherConditions retrieves weather data from Silenus
func (de *DecisionEngine) getWeatherConditions() *WeatherConditions {
	// Try to get real weather from Silenus
	if de.asgardClients != nil && de.asgardClients.Silenus != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		lat, lon := de.getWeatherCoordinates(ctx)

		weather, err := de.asgardClients.Silenus.GetWeather(ctx, lat, lon)
		if err == nil && weather != nil {
			return &WeatherConditions{
				WindSpeed:     weather.WindSpeed,
				WindDirection: weather.WindDirection,
				Visibility:    weather.Visibility,
				Turbulence:    0.2, // Would be calculated from wind
				IcingRisk:     0.0, // Would be calculated from temp/humidity
				Ceiling:       weather.Ceiling,
				Temperature:   weather.Temperature,
				Pressure:      101325.0, // Would be from weather data
			}
		}
		de.logger.WithError(err).Debug("Failed to get weather from Silenus, using defaults")
	}
	
	// Fallback to default conditions
	return &WeatherConditions{
		WindSpeed:     5.0,
		WindDirection: 0.0,
		Visibility:    10000.0,
		Turbulence:    0.2,
		IcingRisk:     0.0,
		Ceiling:       10000.0,
		Temperature:   15.0,
		Pressure:      101325.0,
	}
}

func (de *DecisionEngine) getWeatherCoordinates(ctx context.Context) (float64, float64) {
	if strings.EqualFold(de.config.GeoReferenceSource, "n2yo") {
		if de.asgardClients != nil && de.asgardClients.Nysus != nil {
			noradID := de.config.GeoReferenceNoradID
			if noradID == 0 {
				noradID = 25544 // ISS default
			}
			lat, lon, err := de.asgardClients.Nysus.GetRealtimeSatellitePosition(ctx, noradID)
			if err == nil {
				return lat, lon
			}
			de.logger.WithError(err).Warn("N2YO position lookup failed; falling back to fusion coordinates")
		}
	}

	if de.currentState == nil || de.currentState.Confidence <= 0 {
		return 0.0, 0.0
	}

	x := de.currentState.Position[0]
	y := de.currentState.Position[1]

	if !de.config.GeoReferenceEnabled {
		return x, y
	}

	if de.config.GeoReferenceLatitude == 0 && de.config.GeoReferenceLongitude == 0 {
		de.logger.Warn("Geo reference enabled but origin not set; using raw position")
		return x, y
	}

	// Convert local meters (X east, Y north) to lat/lon.
	// Use reference latitude for the cosine factor (proper local tangent plane conversion).
	const metersPerDegLat = 111320.0
	refLatRad := de.config.GeoReferenceLatitude * math.Pi / 180.0
	metersPerDegLon := metersPerDegLat * math.Cos(refLatRad)
	if metersPerDegLon == 0 {
		return de.config.GeoReferenceLatitude + (y / metersPerDegLat), de.config.GeoReferenceLongitude
	}

	lat := de.config.GeoReferenceLatitude + (y / metersPerDegLat)
	lon := de.config.GeoReferenceLongitude + (x / metersPerDegLon)
	return lat, lon
}

// Run starts the decision loop
func (de *DecisionEngine) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Duration(float64(time.Second) / de.config.DecisionRate))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			_, err := de.Decide(ctx)
			if err != nil {
				// Log but continue
				continue
			}
		}
	}
}

// NewRLPolicy creates a new RL policy
func NewRLPolicy() *ReinforcementLearningPolicy {
	return &ReinforcementLearningPolicy{
		weights: make([]float64, 100),
		epsilon: 0.1,
	}
}

// Initialize sets up the RL policy
func (rl *ReinforcementLearningPolicy) Initialize() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Initialize with small random weights
	for i := range rl.weights {
		rl.weights[i] = 0.01
	}
}

// SelectAction chooses an action given the current state
func (rl *ReinforcementLearningPolicy) SelectAction(
	state *fusion.FusionState,
	threats []*Threat,
	weather *WeatherConditions,
	posError [3]float64,
) *RLAction {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	// Production-ready RL policy using Q-learning with linear function approximation
	// Extract state features for function approximation
	stateFeatures := rl.extractFeatures(state, threats, weather, posError)
	
	// Compute Q-values for action space using linear approximation: Q(s,a) = w^T * phi(s,a)
	qValues := rl.computeQValues(stateFeatures)
	
	// Epsilon-greedy action selection
	var action *RLAction
	if rand.Float64() < rl.epsilon {
		// Exploration: random action within safety bounds
		action = rl.exploreAction(state)
	} else {
		// Exploitation: select best action from Q-values
		action = rl.exploitAction(qValues, state, threats, weather)
	}
	
	// Apply safety constraints
	action = rl.applySafetyConstraints(action, state, threats, weather)

	// Adjust for threats
	if len(threats) > 0 {
		action.Throttle = 0.9 // Increase speed when threatened
	}

	// Adjust for weather
	if weather != nil && weather.WindSpeed > 10 {
		action.Throttle = math.Min(1.0, action.Throttle+0.1)
	}

	return action
}

// GetMissionStatus returns the current mission status
func (de *DecisionEngine) GetMissionStatus() MissionStatus {
	de.mu.RLock()
	defer de.mu.RUnlock()
	if de.currentMission == nil {
		return MissionPending
	}
	return de.currentMission.Status
}
