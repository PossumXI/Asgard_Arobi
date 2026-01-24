// Package physics - Precision Interceptor Module
// High-precision payload delivery to moving targets in atmosphere and space
package physics

import (
	"context"
	"math"
	"sync"
	"time"
)

// ================================================================================
// PRECISION INTERCEPTOR
// Provides ultra-precise guidance for any payload type to any target environment
// ================================================================================

// PayloadType defines the payload being guided
type PayloadType string

const (
	PayloadBallistic    PayloadType = "ballistic"    // Unguided after launch
	PayloadCruise       PayloadType = "cruise"       // Air-breathing, maneuvering
	PayloadHypersonic   PayloadType = "hypersonic"   // Mach 5+ maneuvering
	PayloadOrbitalKV    PayloadType = "orbital_kv"   // Orbital kill vehicle
	PayloadReentry      PayloadType = "reentry"      // Re-entry vehicle
	PayloadSpacecraft   PayloadType = "spacecraft"   // General spacecraft
	PayloadRocket       PayloadType = "rocket"       // Rocket-powered
	PayloadDrone        PayloadType = "drone"        // UAV/Drone
	PayloadRobot        PayloadType = "robot"        // Ground robot
	PayloadSubmarine    PayloadType = "submarine"    // Underwater vehicle
)

// TargetType defines the target being tracked
type TargetType string

const (
	TargetStationary   TargetType = "stationary"   // Fixed position
	TargetLinear       TargetType = "linear"       // Constant velocity
	TargetManeuvering  TargetType = "maneuvering"  // Accelerating/evasive
	TargetOrbital      TargetType = "orbital"      // In orbit
	TargetBallistic    TargetType = "ballistic"    // Ballistic trajectory
	TargetAirborne     TargetType = "airborne"     // Aircraft
	TargetSeaborne     TargetType = "seaborne"     // Ship
)

// EnvironmentType defines the operating environment
type EnvironmentType string

const (
	EnvGroundLevel     EnvironmentType = "ground"      // Surface level
	EnvLowAtmosphere   EnvironmentType = "low_atmo"    // < 20 km
	EnvHighAtmosphere  EnvironmentType = "high_atmo"   // 20-100 km
	EnvLEO             EnvironmentType = "leo"         // 100-2000 km
	EnvMEO             EnvironmentType = "meo"         // 2000-35786 km
	EnvGEO             EnvironmentType = "geo"         // 35786 km
	EnvCislunar        EnvironmentType = "cislunar"    // Earth-Moon space
	EnvDeepSpace       EnvironmentType = "deep_space"  // Beyond Moon
	EnvUnderwater      EnvironmentType = "underwater"  // Submarine
)

// PrecisionInterceptor is the main guidance engine
type PrecisionInterceptor struct {
	mu sync.RWMutex

	// Configuration
	config InterceptorConfig

	// State tracking
	payloadState  OrbitalState
	targetTracker *TargetTracker
	
	// Environment models
	gravityModel    GravityModel
	atmosphereModel AtmosphereModel
	
	// Guidance algorithms
	guidanceLaw GuidanceLaw
	
	// Performance metrics
	metrics InterceptorMetrics
}

// InterceptorConfig holds configuration
type InterceptorConfig struct {
	PayloadType     PayloadType
	PayloadMass     float64 // kg
	PayloadThrust   float64 // N (max thrust)
	PayloadISP      float64 // seconds (specific impulse)
	DragArea        float64 // m²
	DragCoeff       float64
	MaxG            float64 // Maximum g-load
	MaxMach         float64 // Maximum Mach number
	SteeringRate    float64 // rad/s max steering
	SeekerFOV       float64 // radians (seeker field of view)
	SeekerRange     float64 // meters (max detection range)
	MinIntercept    float64 // meters (minimum intercept distance)
	
	// Environment
	CentralBody     string // "earth", "moon", "mars"
	GravityFidelity GravityModel
	AtmoFidelity    AtmosphereModel
	
	// Guidance
	GuidanceLaw     GuidanceLaw
	UpdateRate      time.Duration
	LookAheadTime   time.Duration
}

// GuidanceLaw defines the guidance algorithm
type GuidanceLaw string

const (
	GuidanceProNav          GuidanceLaw = "pronav"           // Proportional Navigation
	GuidanceAugProNav       GuidanceLaw = "aug_pronav"       // Augmented ProNav
	GuidanceTPN             GuidanceLaw = "tpn"              // True Proportional Navigation
	GuidanceZEM             GuidanceLaw = "zem"              // Zero Effort Miss
	GuidanceOptimal         GuidanceLaw = "optimal"          // Optimal control
	GuidanceSlidingMode     GuidanceLaw = "sliding_mode"     // Sliding mode control
	GuidanceAdaptive        GuidanceLaw = "adaptive"         // Adaptive guidance
	GuidancePredictive      GuidanceLaw = "predictive"       // Model predictive control
)

// TargetTracker tracks and predicts target motion
type TargetTracker struct {
	mu sync.RWMutex
	
	targetID      string
	targetType    TargetType
	
	// State history for filtering
	stateHistory  []TargetState
	maxHistory    int
	
	// Kalman filter state (9-state: pos, vel, acc)
	kalmanState   []float64
	kalmanCov     [][]float64
	processNoise  [][]float64
	
	// Maneuver detection
	maneuverProb  float64
	lastManeuver  time.Time
	
	// Prediction
	predictedPath []TargetState
}

// TargetState represents target state at a time
type TargetState struct {
	Position     Vector3D
	Velocity     Vector3D
	Acceleration Vector3D
	Timestamp    time.Time
	Confidence   float64
}

// InterceptorMetrics tracks performance
type InterceptorMetrics struct {
	MissDistance     float64       // meters
	TimeToIntercept  time.Duration
	DeltaVUsed       float64       // m/s
	MaxGExperienced  float64       // g's
	GuidanceUpdates  int
	ManeuversDetected int
	FinalClosingSpeed float64      // m/s
}

// GuidanceCommand represents a guidance update
type GuidanceCommand struct {
	AccelCommand    Vector3D      // Commanded acceleration (m/s²)
	ThrustVector    Vector3D      // Thrust direction (unit vector)
	ThrustLevel     float64       // 0-1 throttle
	TimeToGo        time.Duration // Estimated time to intercept
	PredictedMiss   float64       // meters
	Confidence      float64       // 0-1
	Timestamp       time.Time
}

// NewPrecisionInterceptor creates a new interceptor
func NewPrecisionInterceptor(config InterceptorConfig) *PrecisionInterceptor {
	if config.UpdateRate == 0 {
		config.UpdateRate = 10 * time.Millisecond // 100 Hz default
	}
	if config.LookAheadTime == 0 {
		config.LookAheadTime = 30 * time.Second
	}
	if config.GuidanceLaw == "" {
		config.GuidanceLaw = GuidanceAugProNav
	}
	
	return &PrecisionInterceptor{
		config:          config,
		gravityModel:    config.GravityFidelity,
		atmosphereModel: config.AtmoFidelity,
		guidanceLaw:     config.GuidanceLaw,
		targetTracker:   NewTargetTracker("", TargetManeuvering),
	}
}

// NewTargetTracker creates a new target tracker
func NewTargetTracker(targetID string, targetType TargetType) *TargetTracker {
	// Initialize 9-state Kalman filter
	kalmanState := make([]float64, 9) // [x, y, z, vx, vy, vz, ax, ay, az]
	
	// Initial covariance (large uncertainty)
	kalmanCov := make([][]float64, 9)
	for i := range kalmanCov {
		kalmanCov[i] = make([]float64, 9)
		kalmanCov[i][i] = 10000 // Large initial variance
	}
	
	// Process noise
	processNoise := make([][]float64, 9)
	for i := range processNoise {
		processNoise[i] = make([]float64, 9)
	}
	// Position noise: 1 m²/s⁴
	processNoise[0][0] = 1
	processNoise[1][1] = 1
	processNoise[2][2] = 1
	// Velocity noise: 10 m²/s⁴
	processNoise[3][3] = 10
	processNoise[4][4] = 10
	processNoise[5][5] = 10
	// Acceleration noise: 100 m²/s⁴ (allows for maneuvers)
	processNoise[6][6] = 100
	processNoise[7][7] = 100
	processNoise[8][8] = 100
	
	return &TargetTracker{
		targetID:     targetID,
		targetType:   targetType,
		stateHistory: make([]TargetState, 0, 100),
		maxHistory:   100,
		kalmanState:  kalmanState,
		kalmanCov:    kalmanCov,
		processNoise: processNoise,
	}
}

// UpdateTarget updates target tracking with new observation
func (tt *TargetTracker) UpdateTarget(observation TargetState) {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	
	// Add to history
	tt.stateHistory = append(tt.stateHistory, observation)
	if len(tt.stateHistory) > tt.maxHistory {
		tt.stateHistory = tt.stateHistory[1:]
	}
	
	// Get time delta
	dt := 0.1 // Default 100ms
	if len(tt.stateHistory) >= 2 {
		last := tt.stateHistory[len(tt.stateHistory)-2]
		dt = observation.Timestamp.Sub(last.Timestamp).Seconds()
	}
	if dt <= 0 {
		dt = 0.1
	}
	
	// Kalman prediction step
	tt.kalmanPredict(dt)
	
	// Kalman update step
	tt.kalmanUpdate(observation)
	
	// Maneuver detection
	tt.detectManeuver(observation)
}

// kalmanPredict performs the Kalman filter prediction step
func (tt *TargetTracker) kalmanPredict(dt float64) {
	// State transition matrix for constant acceleration model
	// x_new = x + v*dt + 0.5*a*dt²
	// v_new = v + a*dt
	// a_new = a
	
	x := tt.kalmanState
	dt2 := dt * dt / 2
	
	// Predict state
	newState := make([]float64, 9)
	// Position
	newState[0] = x[0] + x[3]*dt + x[6]*dt2
	newState[1] = x[1] + x[4]*dt + x[7]*dt2
	newState[2] = x[2] + x[5]*dt + x[8]*dt2
	// Velocity
	newState[3] = x[3] + x[6]*dt
	newState[4] = x[4] + x[7]*dt
	newState[5] = x[5] + x[8]*dt
	// Acceleration (assumed constant)
	newState[6] = x[6]
	newState[7] = x[7]
	newState[8] = x[8]
	
	tt.kalmanState = newState
	
	// Predict covariance: P = F*P*F' + Q
	// Simplified: just add process noise
	for i := 0; i < 9; i++ {
		tt.kalmanCov[i][i] += tt.processNoise[i][i] * dt
	}
}

// kalmanUpdate performs the Kalman filter update step
func (tt *TargetTracker) kalmanUpdate(obs TargetState) {
	// Observation: position and velocity
	z := []float64{
		obs.Position.X, obs.Position.Y, obs.Position.Z,
		obs.Velocity.X, obs.Velocity.Y, obs.Velocity.Z,
	}
	
	// Measurement noise (assuming 10m position, 1 m/s velocity accuracy)
	R := []float64{100, 100, 100, 1, 1, 1}
	
	// Innovation
	y := make([]float64, 6)
	for i := 0; i < 6; i++ {
		y[i] = z[i] - tt.kalmanState[i]
	}
	
	// Kalman gain (simplified diagonal)
	K := make([]float64, 9)
	for i := 0; i < 6; i++ {
		S := tt.kalmanCov[i][i] + R[i]
		if S > 0 {
			K[i] = tt.kalmanCov[i][i] / S
		}
	}
	
	// Update state
	for i := 0; i < 6; i++ {
		tt.kalmanState[i] += K[i] * y[i]
	}
	
	// Update covariance
	for i := 0; i < 9; i++ {
		if i < 6 {
			tt.kalmanCov[i][i] *= (1 - K[i])
		}
	}
}

// detectManeuver detects target maneuvers
func (tt *TargetTracker) detectManeuver(obs TargetState) {
	if len(tt.stateHistory) < 3 {
		return
	}
	
	// Check acceleration change
	n := len(tt.stateHistory)
	prevAccel := tt.stateHistory[n-2].Acceleration
	currAccel := obs.Acceleration
	
	accelChange := Vector3D{
		X: currAccel.X - prevAccel.X,
		Y: currAccel.Y - prevAccel.Y,
		Z: currAccel.Z - prevAccel.Z,
	}.Magnitude()
	
	// Maneuver threshold: 5 m/s² change
	if accelChange > 5.0 {
		tt.maneuverProb = math.Min(1.0, tt.maneuverProb+0.3)
		tt.lastManeuver = obs.Timestamp
	} else {
		tt.maneuverProb = math.Max(0, tt.maneuverProb-0.1)
	}
}

// GetPredictedState returns predicted target state at future time
func (tt *TargetTracker) GetPredictedState(tAhead time.Duration) TargetState {
	tt.mu.RLock()
	defer tt.mu.RUnlock()
	
	dt := tAhead.Seconds()
	dt2 := dt * dt / 2
	
	x := tt.kalmanState
	
	return TargetState{
		Position: Vector3D{
			X: x[0] + x[3]*dt + x[6]*dt2,
			Y: x[1] + x[4]*dt + x[7]*dt2,
			Z: x[2] + x[5]*dt + x[8]*dt2,
		},
		Velocity: Vector3D{
			X: x[3] + x[6]*dt,
			Y: x[4] + x[7]*dt,
			Z: x[5] + x[8]*dt,
		},
		Acceleration: Vector3D{
			X: x[6],
			Y: x[7],
			Z: x[8],
		},
		Timestamp:  time.Now().Add(tAhead),
		Confidence: math.Max(0, 1.0-0.1*dt), // Confidence decreases with time
	}
}

// GetCurrentState returns current filtered target state
func (tt *TargetTracker) GetCurrentState() TargetState {
	tt.mu.RLock()
	defer tt.mu.RUnlock()
	
	x := tt.kalmanState
	return TargetState{
		Position: Vector3D{X: x[0], Y: x[1], Z: x[2]},
		Velocity: Vector3D{X: x[3], Y: x[4], Z: x[5]},
		Acceleration: Vector3D{X: x[6], Y: x[7], Z: x[8]},
		Timestamp: time.Now(),
		Confidence: 1.0,
	}
}

// ================================================================================
// GUIDANCE LAWS
// ================================================================================

// ComputeGuidance computes the next guidance command
func (pi *PrecisionInterceptor) ComputeGuidance(ctx context.Context) (*GuidanceCommand, error) {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	
	// Get current states
	payload := pi.payloadState
	target := pi.targetTracker.GetCurrentState()
	
	// Line of sight (LOS) vector
	los := Vector3D{
		X: target.Position.X - payload.Position.X,
		Y: target.Position.Y - payload.Position.Y,
		Z: target.Position.Z - payload.Position.Z,
	}
	losMag := los.Magnitude()
	losUnit := los.Normalize()
	
	// Relative velocity
	vRel := Vector3D{
		X: target.Velocity.X - payload.Velocity.X,
		Y: target.Velocity.Y - payload.Velocity.Y,
		Z: target.Velocity.Z - payload.Velocity.Z,
	}
	
	// Closing velocity (negative means closing)
	closingSpeed := -vRel.Dot(losUnit)
	
	// Time to go
	var timeToGo time.Duration
	if closingSpeed > 0 {
		timeToGo = time.Duration(losMag/closingSpeed*1e9) * time.Nanosecond
	} else {
		timeToGo = 999 * time.Second // Not closing
	}
	
	// LOS rate
	omega := los.Cross(vRel).Scale(1 / (losMag * losMag))
	
	var accelCmd Vector3D
	
	switch pi.guidanceLaw {
	case GuidanceProNav:
		accelCmd = pi.proportionalNavigation(closingSpeed, omega, 3.0)
	case GuidanceAugProNav:
		accelCmd = pi.augmentedProNav(closingSpeed, omega, target.Acceleration, 4.0)
	case GuidanceTPN:
		accelCmd = pi.trueProNav(losUnit, omega, closingSpeed, 4.0)
	case GuidanceZEM:
		accelCmd = pi.zeroEffortMiss(los, vRel, timeToGo)
	case GuidanceOptimal:
		accelCmd = pi.optimalGuidance(los, vRel, target.Acceleration, timeToGo)
	default:
		accelCmd = pi.augmentedProNav(closingSpeed, omega, target.Acceleration, 4.0)
	}
	
	// Apply g-limit
	accelMag := accelCmd.Magnitude()
	maxAccel := pi.config.MaxG * 9.81
	if accelMag > maxAccel {
		accelCmd = accelCmd.Scale(maxAccel / accelMag)
	}
	
	// Add gravity compensation
	gravAccel := CalculateGravity(payload.Position, pi.gravityModel, pi.config.CentralBody)
	accelCmd = accelCmd.Sub(gravAccel) // Compensate for gravity
	
	// Predict miss distance
	missDistance := pi.predictMissDistance(payload, target, timeToGo)
	
	// Calculate thrust vector and level
	thrustDir := accelCmd.Normalize()
	thrustLevel := math.Min(1.0, accelCmd.Magnitude()*payload.Mass/pi.config.PayloadThrust)
	
	cmd := &GuidanceCommand{
		AccelCommand:  accelCmd,
		ThrustVector:  thrustDir,
		ThrustLevel:   thrustLevel,
		TimeToGo:      timeToGo,
		PredictedMiss: missDistance,
		Confidence:    target.Confidence,
		Timestamp:     time.Now(),
	}
	
	pi.metrics.GuidanceUpdates++
	
	return cmd, nil
}

// proportionalNavigation implements basic PN guidance
func (pi *PrecisionInterceptor) proportionalNavigation(vc float64, omega Vector3D, N float64) Vector3D {
	// a = N * Vc * omega
	return omega.Scale(N * vc)
}

// augmentedProNav implements augmented PN with target acceleration
func (pi *PrecisionInterceptor) augmentedProNav(vc float64, omega Vector3D, targetAccel Vector3D, N float64) Vector3D {
	// a = N * Vc * omega + 0.5 * N * at
	pnAccel := omega.Scale(N * vc)
	augAccel := targetAccel.Scale(0.5 * N)
	return pnAccel.Add(augAccel)
}

// trueProNav implements true proportional navigation
func (pi *PrecisionInterceptor) trueProNav(losUnit, omega Vector3D, vc, N float64) Vector3D {
	// Acceleration perpendicular to LOS
	omegaMag := omega.Magnitude()
	perpDir := losUnit.Cross(omega).Normalize()
	return perpDir.Scale(N * vc * omegaMag)
}

// zeroEffortMiss implements ZEM guidance
func (pi *PrecisionInterceptor) zeroEffortMiss(los, vRel Vector3D, tgo time.Duration) Vector3D {
	t := tgo.Seconds()
	if t <= 0 {
		t = 0.1
	}
	
	// ZEM = los + vRel * tgo
	zem := los.Add(vRel.Scale(t))
	
	// a = -2 * ZEM / tgo²
	return zem.Scale(-2 / (t * t))
}

// optimalGuidance implements optimal control guidance
func (pi *PrecisionInterceptor) optimalGuidance(los, vRel, targetAccel Vector3D, tgo time.Duration) Vector3D {
	t := tgo.Seconds()
	if t <= 0 {
		t = 0.1
	}
	
	// Optimal guidance for maneuvering target
	// a = 6*ZEM/tgo² + 2*ZEM_dot/tgo + at/2
	
	zem := los.Add(vRel.Scale(t)).Add(targetAccel.Scale(t * t / 2))
	zemDot := vRel.Add(targetAccel.Scale(t))
	
	accel := zem.Scale(6 / (t * t))
	accel = accel.Add(zemDot.Scale(2 / t))
	accel = accel.Add(targetAccel.Scale(0.5))
	
	return accel
}

// predictMissDistance estimates miss distance
func (pi *PrecisionInterceptor) predictMissDistance(payload OrbitalState, target TargetState, tgo time.Duration) float64 {
	t := tgo.Seconds()
	if t <= 0 {
		return 0
	}
	
	// Simplified prediction: linear extrapolation
	payloadFuture := Vector3D{
		X: payload.Position.X + payload.Velocity.X*t,
		Y: payload.Position.Y + payload.Velocity.Y*t,
		Z: payload.Position.Z + payload.Velocity.Z*t,
	}
	
	targetFuture := Vector3D{
		X: target.Position.X + target.Velocity.X*t + 0.5*target.Acceleration.X*t*t,
		Y: target.Position.Y + target.Velocity.Y*t + 0.5*target.Acceleration.Y*t*t,
		Z: target.Position.Z + target.Velocity.Z*t + 0.5*target.Acceleration.Z*t*t,
	}
	
	miss := payloadFuture.Sub(targetFuture)
	return miss.Magnitude()
}

// UpdatePayloadState updates the payload state
func (pi *PrecisionInterceptor) UpdatePayloadState(state OrbitalState) {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	pi.payloadState = state
}

// UpdateTargetObservation updates the target with new observation
func (pi *PrecisionInterceptor) UpdateTargetObservation(obs TargetState) {
	pi.targetTracker.UpdateTarget(obs)
}

// GetMetrics returns performance metrics
func (pi *PrecisionInterceptor) GetMetrics() InterceptorMetrics {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return pi.metrics
}

// ================================================================================
// MISSION PLANNING
// ================================================================================

// InterceptMission represents a complete intercept mission
type InterceptMission struct {
	ID            string
	PayloadType   PayloadType
	TargetType    TargetType
	Environment   EnvironmentType
	LaunchPos     Vector3D
	LaunchTime    time.Time
	Waypoints     []MissionWaypoint
	InterceptPoint Vector3D
	InterceptTime time.Time
	DeltaVBudget  float64
	Constraints   MissionConstraints
	Status        string
}

// MissionWaypoint represents a waypoint in the mission
type MissionWaypoint struct {
	Position    Vector3D
	Velocity    Vector3D
	Time        time.Time
	Purpose     string // "launch", "burn", "coast", "terminal"
	DeltaV      float64
}

// MissionConstraints contains mission constraints
type MissionConstraints struct {
	MaxFlightTime   time.Duration
	MaxDeltaV       float64
	MaxGLoad        float64
	MinAltitude     float64
	MaxAltitude     float64
	NoFlyZones      []Vector3D
	WeatherLimits   bool
	RadiationLimits bool
}

// PlanIntercept creates an optimal intercept plan
func PlanIntercept(
	launchState OrbitalState,
	targetTracker *TargetTracker,
	payloadType PayloadType,
	constraints MissionConstraints,
) (*InterceptMission, error) {
	
	mission := &InterceptMission{
		ID:          time.Now().Format("20060102-150405"),
		PayloadType: payloadType,
		LaunchPos:   launchState.Position,
		LaunchTime:  time.Now(),
		Constraints: constraints,
		Status:      "planning",
	}
	
	// Get current target state
	targetState := targetTracker.GetCurrentState()
	
	// Determine environment
	altitude := launchState.Position.Magnitude() - R_Earth
	if altitude < 0 {
		mission.Environment = EnvGroundLevel
	} else if altitude < 20e3 {
		mission.Environment = EnvLowAtmosphere
	} else if altitude < KarmanLine {
		mission.Environment = EnvHighAtmosphere
	} else if altitude < 2000e3 {
		mission.Environment = EnvLEO
	} else {
		mission.Environment = EnvMEO
	}
	
	// Calculate intercept
	maxDeltaV := constraints.MaxDeltaV
	if maxDeltaV == 0 {
		maxDeltaV = 5000 // Default 5 km/s
	}
	
	interceptSolution, err := CalculateMovingTargetIntercept(
		launchState.Position,
		launchState.Velocity,
		targetState.Position,
		targetState.Velocity,
		targetState.Acceleration,
		maxDeltaV,
		constraints.MaxFlightTime,
	)
	if err != nil {
		return nil, err
	}
	
	mission.InterceptPoint = interceptSolution.InterceptPoint
	mission.InterceptTime = time.Now().Add(interceptSolution.FlightTime)
	mission.DeltaVBudget = interceptSolution.DeltaV
	
	// Generate waypoints
	mission.Waypoints = generateWaypoints(launchState, interceptSolution, mission.Environment)
	
	mission.Status = "ready"
	return mission, nil
}

// generateWaypoints creates mission waypoints
func generateWaypoints(launchState OrbitalState, intercept *InterceptSolution, env EnvironmentType) []MissionWaypoint {
	waypoints := make([]MissionWaypoint, 0)
	
	// Launch waypoint
	waypoints = append(waypoints, MissionWaypoint{
		Position: launchState.Position,
		Velocity: launchState.Velocity,
		Time:     time.Now(),
		Purpose:  "launch",
		DeltaV:   0,
	})
	
	// Initial burn
	waypoints = append(waypoints, MissionWaypoint{
		Position: launchState.Position,
		Velocity: intercept.LaunchVelocity,
		Time:     time.Now().Add(10 * time.Second),
		Purpose:  "burn",
		DeltaV:   intercept.DeltaV * 0.8, // 80% of delta-v at launch
	})
	
	// Midcourse (coast or adjustment)
	midTime := intercept.FlightTime / 2
	midPos := Vector3D{
		X: launchState.Position.X + intercept.LaunchVelocity.X*midTime.Seconds(),
		Y: launchState.Position.Y + intercept.LaunchVelocity.Y*midTime.Seconds(),
		Z: launchState.Position.Z + intercept.LaunchVelocity.Z*midTime.Seconds(),
	}
	waypoints = append(waypoints, MissionWaypoint{
		Position: midPos,
		Velocity: intercept.LaunchVelocity,
		Time:     time.Now().Add(midTime),
		Purpose:  "coast",
		DeltaV:   0,
	})
	
	// Terminal guidance
	waypoints = append(waypoints, MissionWaypoint{
		Position: intercept.InterceptPoint,
		Velocity: intercept.ImpactVelocity,
		Time:     time.Now().Add(intercept.FlightTime),
		Purpose:  "terminal",
		DeltaV:   intercept.DeltaV * 0.2, // 20% for terminal maneuvers
	})
	
	return waypoints
}

// ================================================================================
// ACCURACY METRICS
// ================================================================================

// PayloadAccuracySpec defines expected accuracy for each payload type
type PayloadAccuracySpec struct {
	PayloadType       PayloadType
	CEP               float64 // meters (Circular Error Probable)
	SEP               float64 // meters (Spherical Error Probable)
	MaxRange          float64 // meters
	TerminalGuidance  bool
	AllWeather        bool
	NightCapable      bool
	MovingTargetCapable bool
	SpaceCapable      bool
}

// GetAccuracySpec returns accuracy specifications for a payload type
func GetAccuracySpec(payloadType PayloadType) PayloadAccuracySpec {
	specs := map[PayloadType]PayloadAccuracySpec{
		PayloadBallistic: {
			PayloadType:       PayloadBallistic,
			CEP:               300,  // 300m CEP
			SEP:               400,
			MaxRange:          10000e3, // 10,000 km
			TerminalGuidance:  false,
			AllWeather:        true,
			NightCapable:      true,
			MovingTargetCapable: false,
			SpaceCapable:      true,
		},
		PayloadCruise: {
			PayloadType:       PayloadCruise,
			CEP:               3,    // 3m CEP with GPS/INS
			SEP:               5,
			MaxRange:          2500e3, // 2,500 km
			TerminalGuidance:  true,
			AllWeather:        true,
			NightCapable:      true,
			MovingTargetCapable: true,
			SpaceCapable:      false,
		},
		PayloadHypersonic: {
			PayloadType:       PayloadHypersonic,
			CEP:               5,    // 5m CEP
			SEP:               8,
			MaxRange:          5000e3, // 5,000 km
			TerminalGuidance:  true,
			AllWeather:        true,
			NightCapable:      true,
			MovingTargetCapable: true,
			SpaceCapable:      true,
		},
		PayloadOrbitalKV: {
			PayloadType:       PayloadOrbitalKV,
			CEP:               0.5,  // 0.5m CEP (hit-to-kill)
			SEP:               1,
			MaxRange:          1000e3, // 1,000 km
			TerminalGuidance:  true,
			AllWeather:        true,
			NightCapable:      true,
			MovingTargetCapable: true,
			SpaceCapable:      true,
		},
		PayloadReentry: {
			PayloadType:       PayloadReentry,
			CEP:               100,  // 100m CEP
			SEP:               150,
			MaxRange:          15000e3, // 15,000 km (ICBM class)
			TerminalGuidance:  true,
			AllWeather:        true,
			NightCapable:      true,
			MovingTargetCapable: false,
			SpaceCapable:      true,
		},
		PayloadDrone: {
			PayloadType:       PayloadDrone,
			CEP:               1,    // 1m CEP
			SEP:               2,
			MaxRange:          500e3, // 500 km
			TerminalGuidance:  true,
			AllWeather:        false, // Weather limited
			NightCapable:      true,
			MovingTargetCapable: true,
			SpaceCapable:      false,
		},
		PayloadRobot: {
			PayloadType:       PayloadRobot,
			CEP:               0.1,  // 10cm CEP
			SEP:               0.2,
			MaxRange:          100e3, // 100 km operational
			TerminalGuidance:  true,
			AllWeather:        false,
			NightCapable:      true,
			MovingTargetCapable: true,
			SpaceCapable:      false,
		},
	}
	
	if spec, ok := specs[payloadType]; ok {
		return spec
	}
	
	// Default
	return PayloadAccuracySpec{
		PayloadType: payloadType,
		CEP:         10,
		SEP:         15,
		MaxRange:    1000e3,
	}
}
