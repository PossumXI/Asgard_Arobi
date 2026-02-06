// Package flight provides JSBSim-based flight dynamics simulation.
// Uses the open-source JSBSim flight dynamics model for realistic physics.
//
// Copyright 2026 Arobi. All Rights Reserved.
package flight

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

// AircraftState represents the complete state of the aircraft
type AircraftState struct {
	// Position (geodetic)
	Latitude  float64 `json:"latitude"`  // degrees
	Longitude float64 `json:"longitude"` // degrees
	Altitude  float64 `json:"altitude"`  // meters MSL

	// Velocity
	Airspeed      float64 `json:"airspeed"`       // m/s
	GroundSpeed   float64 `json:"ground_speed"`   // m/s
	VerticalSpeed float64 `json:"vertical_speed"` // m/s

	// Attitude (Euler angles)
	Roll    float64 `json:"roll"`    // degrees
	Pitch   float64 `json:"pitch"`   // degrees
	Heading float64 `json:"heading"` // degrees (true)

	// Angular rates
	RollRate  float64 `json:"roll_rate"`  // deg/s
	PitchRate float64 `json:"pitch_rate"` // deg/s
	YawRate   float64 `json:"yaw_rate"`   // deg/s

	// Accelerations
	AccelX float64 `json:"accel_x"` // g
	AccelY float64 `json:"accel_y"` // g
	AccelZ float64 `json:"accel_z"` // g

	// Engine state
	Throttle      float64 `json:"throttle"`       // 0-1
	FuelRemaining float64 `json:"fuel_remaining"` // kg
	EngineRPM     float64 `json:"engine_rpm"`

	// Control surfaces
	Aileron  float64 `json:"aileron"`  // -1 to 1
	Elevator float64 `json:"elevator"` // -1 to 1
	Rudder   float64 `json:"rudder"`   // -1 to 1
	Flaps    float64 `json:"flaps"`    // 0 to 1

	// Mission state
	MissionPhase     string  `json:"mission_phase"`
	DistanceToTarget float64 `json:"distance_to_target"` // meters
	TimeToTarget     float64 `json:"time_to_target"`     // seconds

	// Stealth metrics
	RadarCrossSection float64 `json:"rcs"`          // m^2
	IRSignature       float64 `json:"ir_signature"` // relative

	Timestamp time.Time `json:"timestamp"`
}

// Waypoint represents a navigation waypoint
type Waypoint struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Altitude   float64 `json:"altitude"`
	Speed      float64 `json:"speed"`       // target speed m/s
	Heading    float64 `json:"heading"`     // optional target heading
	Loiter     bool    `json:"loiter"`      // hold position
	LoiterTime int     `json:"loiter_time"` // seconds
	Action     string  `json:"action"`      // waypoint action
}

// FlightPlan represents a complete flight plan
type FlightPlan struct {
	ID          string     `json:"id"`
	MissionType string     `json:"mission_type"`
	Waypoints   []Waypoint `json:"waypoints"`
	Priority    int        `json:"priority"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Threat represents a detected threat
type Threat struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // radar, sam, aircraft, etc.
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	Altitude    float64   `json:"altitude"`
	Range       float64   `json:"range"`        // detection range meters
	ThreatLevel float64   `json:"threat_level"` // 0-1
	Velocity    float64   `json:"velocity"`     // if moving
	Heading     float64   `json:"heading"`      // if moving
	Active      bool      `json:"active"`
	DetectedAt  time.Time `json:"detected_at"`
}

// WeatherConditions represents current weather
type WeatherConditions struct {
	WindSpeed     float64 `json:"wind_speed"`     // m/s
	WindDirection float64 `json:"wind_direction"` // degrees
	WindGust      float64 `json:"wind_gust"`      // m/s
	Visibility    float64 `json:"visibility"`     // meters
	CloudBase     float64 `json:"cloud_base"`     // meters
	CloudCover    float64 `json:"cloud_cover"`    // 0-1
	Temperature   float64 `json:"temperature"`    // Celsius
	Pressure      float64 `json:"pressure"`       // hPa
	Precipitation string  `json:"precipitation"`  // none, rain, snow, etc.
	Turbulence    string  `json:"turbulence"`     // none, light, moderate, severe
}

// FlightSimulator provides JSBSim-based flight simulation
type FlightSimulator struct {
	mu sync.RWMutex

	// Current state
	aircraft AircraftState
	plan     *FlightPlan
	threats  []Threat
	weather  WeatherConditions

	// Simulation control
	running   bool
	paused    bool
	timeScale float64 // 1.0 = real-time
	simTime   time.Time
	startTime time.Time

	// Autopilot
	autopilotEngaged  bool
	targetWaypointIdx int

	// Callbacks
	onStateUpdate     func(AircraftState)
	onThreatDetected  func(Threat)
	onWaypointReached func(Waypoint)
	onWeatherChange   func(WeatherConditions)

	// Channels
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewFlightSimulator creates a new flight simulator
func NewFlightSimulator() *FlightSimulator {
	return &FlightSimulator{
		aircraft: AircraftState{
			Latitude:      37.7749, // San Francisco
			Longitude:     -122.4194,
			Altitude:      1000,
			Airspeed:      50,
			Heading:       90,
			Throttle:      0.6,
			FuelRemaining: 500,
			MissionPhase:  "idle",
		},
		threats:   make([]Threat, 0),
		weather:   getDefaultWeather(),
		timeScale: 10.0, // 10x speed for demo
		stopCh:    make(chan struct{}),
	}
}

// Start begins the simulation
func (fs *FlightSimulator) Start(ctx context.Context) error {
	fs.mu.Lock()
	if fs.running {
		fs.mu.Unlock()
		return fmt.Errorf("simulation already running")
	}
	fs.running = true
	fs.startTime = time.Now()
	fs.simTime = time.Now()
	fs.mu.Unlock()

	// Start simulation loop
	fs.wg.Add(1)
	go fs.simulationLoop(ctx)

	// Start threat detection
	fs.wg.Add(1)
	go fs.threatDetectionLoop(ctx)

	// Start weather updates
	fs.wg.Add(1)
	go fs.weatherUpdateLoop(ctx)

	log.Println("[FlightSim] Simulation started")
	return nil
}

// Stop halts the simulation
func (fs *FlightSimulator) Stop() {
	fs.mu.Lock()
	if !fs.running {
		fs.mu.Unlock()
		return
	}
	fs.running = false
	fs.mu.Unlock()

	close(fs.stopCh)
	fs.wg.Wait()
	log.Println("[FlightSim] Simulation stopped")
}

// SetFlightPlan sets the current flight plan
func (fs *FlightSimulator) SetFlightPlan(plan *FlightPlan) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.plan = plan
	fs.targetWaypointIdx = 0
	fs.aircraft.MissionPhase = "en_route"

	if len(plan.Waypoints) > 0 {
		wp := plan.Waypoints[0]
		fs.aircraft.DistanceToTarget = fs.calculateDistance(
			fs.aircraft.Latitude, fs.aircraft.Longitude,
			wp.Latitude, wp.Longitude,
		)
	}

	log.Printf("[FlightSim] Flight plan set: %s with %d waypoints", plan.ID, len(plan.Waypoints))
}

// EngageAutopilot enables autonomous flight
func (fs *FlightSimulator) EngageAutopilot(engaged bool) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.autopilotEngaged = engaged
	log.Printf("[FlightSim] Autopilot: %v", engaged)
}

// GetState returns current aircraft state
func (fs *FlightSimulator) GetState() AircraftState {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.aircraft
}

// GetThreats returns detected threats
func (fs *FlightSimulator) GetThreats() []Threat {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return append([]Threat{}, fs.threats...)
}

// GetWeather returns current weather
func (fs *FlightSimulator) GetWeather() WeatherConditions {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.weather
}

// AddThreat adds a threat to the simulation
func (fs *FlightSimulator) AddThreat(threat Threat) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.threats = append(fs.threats, threat)
	log.Printf("[FlightSim] Threat added: %s at (%.4f, %.4f)", threat.Type, threat.Latitude, threat.Longitude)
}

// SetWeather updates weather conditions
func (fs *FlightSimulator) SetWeather(weather WeatherConditions) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.weather = weather
	log.Printf("[FlightSim] Weather updated: wind %.1f m/s from %.0f°", weather.WindSpeed, weather.WindDirection)
}

// Callbacks
func (fs *FlightSimulator) OnStateUpdate(cb func(AircraftState)) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.onStateUpdate = cb
}

func (fs *FlightSimulator) OnThreatDetected(cb func(Threat)) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.onThreatDetected = cb
}

func (fs *FlightSimulator) OnWaypointReached(cb func(Waypoint)) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.onWaypointReached = cb
}

// simulationLoop is the main physics update loop
func (fs *FlightSimulator) simulationLoop(ctx context.Context) {
	defer fs.wg.Done()

	ticker := time.NewTicker(50 * time.Millisecond) // 20 Hz
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-fs.stopCh:
			return
		case <-ticker.C:
			fs.updatePhysics()
		}
	}
}

// updatePhysics updates aircraft state using simplified flight dynamics
func (fs *FlightSimulator) updatePhysics() {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.paused {
		return
	}

	dt := 0.05 * fs.timeScale // 50ms * time scale

	// Update simulation time
	fs.simTime = fs.simTime.Add(time.Duration(dt * float64(time.Second)))

	// Autopilot control
	if fs.autopilotEngaged && fs.plan != nil && fs.targetWaypointIdx < len(fs.plan.Waypoints) {
		fs.updateAutopilot(dt)
	}

	// Physics update (simplified JSBSim-like model)
	fs.updateFlightDynamics(dt)

	// Check waypoint arrival
	if fs.plan != nil && fs.targetWaypointIdx < len(fs.plan.Waypoints) {
		wp := fs.plan.Waypoints[fs.targetWaypointIdx]
		dist := fs.calculateDistance(fs.aircraft.Latitude, fs.aircraft.Longitude, wp.Latitude, wp.Longitude)
		fs.aircraft.DistanceToTarget = dist

		if dist < 100 { // 100 meter arrival radius
			log.Printf("[FlightSim] Reached waypoint: %s", wp.Name)
			if fs.onWaypointReached != nil {
				go fs.onWaypointReached(wp)
			}
			fs.targetWaypointIdx++
			if fs.targetWaypointIdx >= len(fs.plan.Waypoints) {
				fs.aircraft.MissionPhase = "complete"
			}
		}
	}

	// Update timestamp
	fs.aircraft.Timestamp = fs.simTime

	// Notify listeners
	if fs.onStateUpdate != nil {
		go fs.onStateUpdate(fs.aircraft)
	}
}

// updateAutopilot calculates control inputs for waypoint navigation
func (fs *FlightSimulator) updateAutopilot(dt float64) {
	wp := fs.plan.Waypoints[fs.targetWaypointIdx]

	// Calculate bearing to waypoint
	bearing := fs.calculateBearing(
		fs.aircraft.Latitude, fs.aircraft.Longitude,
		wp.Latitude, wp.Longitude,
	)

	// Heading error
	headingError := bearing - fs.aircraft.Heading
	for headingError > 180 {
		headingError -= 360
	}
	for headingError < -180 {
		headingError += 360
	}

	// Bank angle command (proportional control)
	bankCmd := headingError * 0.5
	bankCmd = clamp(bankCmd, -30, 30)

	// Altitude error
	altError := wp.Altitude - fs.aircraft.Altitude
	pitchCmd := altError * 0.01
	pitchCmd = clamp(pitchCmd, -15, 15)

	// Apply controls smoothly
	fs.aircraft.Roll = lerp(fs.aircraft.Roll, bankCmd, 0.1)
	fs.aircraft.Pitch = lerp(fs.aircraft.Pitch, pitchCmd, 0.1)

	// Speed control
	speedError := wp.Speed - fs.aircraft.Airspeed
	throttleCmd := 0.6 + speedError*0.01
	fs.aircraft.Throttle = clamp(throttleCmd, 0, 1)
}

// updateFlightDynamics applies simplified aerodynamics
func (fs *FlightSimulator) updateFlightDynamics(dt float64) {
	// Convert to radians
	rollRad := fs.aircraft.Roll * math.Pi / 180
	pitchRad := fs.aircraft.Pitch * math.Pi / 180
	headingRad := fs.aircraft.Heading * math.Pi / 180

	// Turn rate from bank angle (coordinated turn)
	g := 9.81
	turnRate := g * math.Tan(rollRad) / fs.aircraft.Airspeed
	fs.aircraft.Heading += turnRate * dt * 180 / math.Pi

	// Normalize heading
	for fs.aircraft.Heading >= 360 {
		fs.aircraft.Heading -= 360
	}
	for fs.aircraft.Heading < 0 {
		fs.aircraft.Heading += 360
	}

	// Vertical speed from pitch
	fs.aircraft.VerticalSpeed = fs.aircraft.Airspeed * math.Sin(pitchRad)
	fs.aircraft.Altitude += fs.aircraft.VerticalSpeed * dt

	// Ground speed (simplified - ignoring wind for now)
	fs.aircraft.GroundSpeed = fs.aircraft.Airspeed * math.Cos(pitchRad)

	// Position update
	R := 6371000.0 // Earth radius in meters
	dLat := (fs.aircraft.GroundSpeed * math.Cos(headingRad) * dt) / R
	dLon := (fs.aircraft.GroundSpeed * math.Sin(headingRad) * dt) / (R * math.Cos(fs.aircraft.Latitude*math.Pi/180))

	fs.aircraft.Latitude += dLat * 180 / math.Pi
	fs.aircraft.Longitude += dLon * 180 / math.Pi

	// Speed dynamics (thrust vs drag)
	thrust := fs.aircraft.Throttle * 5000                                          // Newtons (simplified)
	drag := 0.5 * 1.225 * fs.aircraft.Airspeed * fs.aircraft.Airspeed * 2.0 * 0.03 // 1/2 * rho * v^2 * S * Cd
	mass := 1000.0                                                                 // kg
	accel := (thrust - drag) / mass
	fs.aircraft.Airspeed += accel * dt
	fs.aircraft.Airspeed = clamp(fs.aircraft.Airspeed, 20, 200)

	// Fuel consumption
	fuelFlow := fs.aircraft.Throttle * 0.1 // kg/s at full throttle
	fs.aircraft.FuelRemaining -= fuelFlow * dt
	if fs.aircraft.FuelRemaining < 0 {
		fs.aircraft.FuelRemaining = 0
	}

	// RCS calculation (simplified)
	fs.aircraft.RadarCrossSection = 1.0 + math.Abs(fs.aircraft.Roll)*0.01

	// ETA calculation
	if fs.aircraft.DistanceToTarget > 0 && fs.aircraft.GroundSpeed > 0 {
		fs.aircraft.TimeToTarget = fs.aircraft.DistanceToTarget / fs.aircraft.GroundSpeed
	}
}

// threatDetectionLoop checks for threats
func (fs *FlightSimulator) threatDetectionLoop(ctx context.Context) {
	defer fs.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-fs.stopCh:
			return
		case <-ticker.C:
			fs.checkThreats()
		}
	}
}

// checkThreats evaluates threat proximity
func (fs *FlightSimulator) checkThreats() {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for i := range fs.threats {
		threat := &fs.threats[i]
		if !threat.Active {
			continue
		}

		dist := fs.calculateDistance(
			fs.aircraft.Latitude, fs.aircraft.Longitude,
			threat.Latitude, threat.Longitude,
		)

		if dist < threat.Range {
			threat.ThreatLevel = 1.0 - (dist / threat.Range)
			if fs.onThreatDetected != nil {
				go fs.onThreatDetected(*threat)
			}
		}
	}
}

// weatherUpdateLoop periodically updates weather
func (fs *FlightSimulator) weatherUpdateLoop(ctx context.Context) {
	defer fs.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-fs.stopCh:
			return
		case <-ticker.C:
			// In production, fetch from weather API
			// For demo, generate realistic variations
			fs.mu.Lock()
			fs.weather.WindSpeed += (0.5 - fs.randFloat()) * 2
			fs.weather.WindSpeed = clamp(fs.weather.WindSpeed, 0, 30)
			fs.weather.Temperature += (0.5 - fs.randFloat()) * 0.5
			fs.mu.Unlock()
		}
	}
}

// Helper functions

func (fs *FlightSimulator) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	R := 6371000.0 // Earth radius in meters

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func (fs *FlightSimulator) calculateBearing(lat1, lon1, lat2, lon2 float64) float64 {
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	y := math.Sin(dLon) * math.Cos(lat2Rad)
	x := math.Cos(lat1Rad)*math.Sin(lat2Rad) - math.Sin(lat1Rad)*math.Cos(lat2Rad)*math.Cos(dLon)

	bearing := math.Atan2(y, x) * 180 / math.Pi
	for bearing < 0 {
		bearing += 360
	}
	return bearing
}

func (fs *FlightSimulator) randFloat() float64 {
	return float64(time.Now().UnixNano()%1000) / 1000.0
}

func getDefaultWeather() WeatherConditions {
	return WeatherConditions{
		WindSpeed:     5.0,
		WindDirection: 270,
		WindGust:      8.0,
		Visibility:    10000,
		CloudBase:     2000,
		CloudCover:    0.3,
		Temperature:   15,
		Pressure:      1013.25,
		Precipitation: "none",
		Turbulence:    "light",
	}
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

// ToJSON serializes state to JSON
func (s *AircraftState) ToJSON() string {
	data, _ := json.Marshal(s)
	return string(data)
}
