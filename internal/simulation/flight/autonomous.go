// Package flight provides autonomous decision-making for flight operations.
// Integrates with GIRU for replanning and threat response.
//
// Copyright 2026 Arobi. All Rights Reserved.
package flight

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

// AutonomousFlightController manages autonomous flight decisions
type AutonomousFlightController struct {
	mu sync.RWMutex

	simulator  *FlightSimulator
	weatherAPI *WeatherAPI

	// Decision state
	currentDecision   DecisionState
	decisionHistory   []DecisionRecord
	threatResponses   []ThreatResponse
	replanCount       int

	// Callbacks
	onDecision      func(DecisionRecord)
	onReplan        func(FlightPlan, string)
	onThreatResponse func(ThreatResponse)

	// Control
	running bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// DecisionState represents the current autonomous decision state
type DecisionState struct {
	Mode             string  `json:"mode"`             // normal, evasive, abort, loiter
	Confidence       float64 `json:"confidence"`       // 0-1
	ThreatLevel      float64 `json:"threat_level"`     // 0-1
	WeatherSafe      bool    `json:"weather_safe"`
	FuelSufficient   bool    `json:"fuel_sufficient"`
	MissionViable    bool    `json:"mission_viable"`
	ReplanRequired   bool    `json:"replan_required"`
	Reason           string  `json:"reason"`
	LastUpdate       time.Time `json:"last_update"`
}

// DecisionRecord logs a decision
type DecisionRecord struct {
	ID          string        `json:"id"`
	Timestamp   time.Time     `json:"timestamp"`
	Type        string        `json:"type"`
	Decision    string        `json:"decision"`
	Reason      string        `json:"reason"`
	Confidence  float64       `json:"confidence"`
	Input       interface{}   `json:"input"`
	Result      interface{}   `json:"result"`
	LatencyMs   float64       `json:"latency_ms"`
}

// ThreatResponse describes how the system responded to a threat
type ThreatResponse struct {
	ThreatID     string    `json:"threat_id"`
	ThreatType   string    `json:"threat_type"`
	ResponseType string    `json:"response_type"` // evade, engage, ignore
	NewHeading   float64   `json:"new_heading"`
	NewAltitude  float64   `json:"new_altitude"`
	Timestamp    time.Time `json:"timestamp"`
	Success      bool      `json:"success"`
}

// WeatherAPI provides weather data integration
type WeatherAPI struct {
	apiKey  string
	baseURL string
}

// NewWeatherAPI creates a new weather API client
func NewWeatherAPI(apiKey string) *WeatherAPI {
	return &WeatherAPI{
		apiKey:  apiKey,
		baseURL: "https://api.openweathermap.org/data/2.5",
	}
}

// GetWeatherAt fetches weather for a location
func (w *WeatherAPI) GetWeatherAt(ctx context.Context, lat, lon float64) (*WeatherConditions, error) {
	// For demo, return simulated weather
	// In production, call OpenWeatherMap API
	weather := &WeatherConditions{
		WindSpeed:     5.0 + float64(time.Now().Unix()%10),
		WindDirection: float64(time.Now().Unix() % 360),
		WindGust:      8.0,
		Visibility:    10000,
		CloudBase:     2000,
		CloudCover:    float64(time.Now().Unix()%50) / 100,
		Temperature:   15 + float64(time.Now().Unix()%10) - 5,
		Pressure:      1013.25,
		Precipitation: "none",
		Turbulence:    "light",
	}

	// Simulate occasional bad weather
	if time.Now().Unix()%30 == 0 {
		weather.Visibility = 3000
		weather.Precipitation = "rain"
		weather.Turbulence = "moderate"
	}

	return weather, nil
}

// NewAutonomousFlightController creates a new autonomous controller
func NewAutonomousFlightController(sim *FlightSimulator) *AutonomousFlightController {
	return &AutonomousFlightController{
		simulator:       sim,
		weatherAPI:      NewWeatherAPI(""), // Free tier doesn't need key for demo
		decisionHistory: make([]DecisionRecord, 0),
		threatResponses: make([]ThreatResponse, 0),
		stopCh:          make(chan struct{}),
	}
}

// Start begins autonomous control
func (afc *AutonomousFlightController) Start(ctx context.Context) error {
	afc.mu.Lock()
	if afc.running {
		afc.mu.Unlock()
		return fmt.Errorf("already running")
	}
	afc.running = true
	afc.mu.Unlock()

	// Set up simulator callbacks
	afc.simulator.OnThreatDetected(afc.handleThreat)

	// Start decision loop
	afc.wg.Add(1)
	go afc.decisionLoop(ctx)

	// Start weather monitoring
	afc.wg.Add(1)
	go afc.weatherMonitorLoop(ctx)

	log.Println("[Autonomous] Controller started")
	return nil
}

// Stop halts autonomous control
func (afc *AutonomousFlightController) Stop() {
	afc.mu.Lock()
	if !afc.running {
		afc.mu.Unlock()
		return
	}
	afc.running = false
	afc.mu.Unlock()

	close(afc.stopCh)
	afc.wg.Wait()
	log.Println("[Autonomous] Controller stopped")
}

// GetDecisionState returns current decision state
func (afc *AutonomousFlightController) GetDecisionState() DecisionState {
	afc.mu.RLock()
	defer afc.mu.RUnlock()
	return afc.currentDecision
}

// GetDecisionHistory returns decision history
func (afc *AutonomousFlightController) GetDecisionHistory() []DecisionRecord {
	afc.mu.RLock()
	defer afc.mu.RUnlock()
	return append([]DecisionRecord{}, afc.decisionHistory...)
}

// GetReplanCount returns number of replans
func (afc *AutonomousFlightController) GetReplanCount() int {
	afc.mu.RLock()
	defer afc.mu.RUnlock()
	return afc.replanCount
}

// Callbacks
func (afc *AutonomousFlightController) OnDecision(cb func(DecisionRecord)) {
	afc.mu.Lock()
	defer afc.mu.Unlock()
	afc.onDecision = cb
}

func (afc *AutonomousFlightController) OnReplan(cb func(FlightPlan, string)) {
	afc.mu.Lock()
	defer afc.mu.Unlock()
	afc.onReplan = cb
}

func (afc *AutonomousFlightController) OnThreatResponse(cb func(ThreatResponse)) {
	afc.mu.Lock()
	defer afc.mu.Unlock()
	afc.onThreatResponse = cb
}

// decisionLoop runs the main decision-making loop
func (afc *AutonomousFlightController) decisionLoop(ctx context.Context) {
	defer afc.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond) // 10 Hz decision rate
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-afc.stopCh:
			return
		case <-ticker.C:
			afc.evaluateSituation()
		}
	}
}

// evaluateSituation makes autonomous decisions
func (afc *AutonomousFlightController) evaluateSituation() {
	startTime := time.Now()

	afc.mu.Lock()
	defer afc.mu.Unlock()

	state := afc.simulator.GetState()
	weather := afc.simulator.GetWeather()
	threats := afc.simulator.GetThreats()

	// Evaluate conditions
	decision := DecisionState{
		Mode:           "normal",
		Confidence:     0.95,
		LastUpdate:     time.Now(),
	}

	// Check fuel
	decision.FuelSufficient = state.FuelRemaining > 50 // 50 kg minimum
	if !decision.FuelSufficient {
		decision.Mode = "abort"
		decision.Reason = "fuel critical"
		decision.Confidence = 0.99
	}

	// Check weather
	decision.WeatherSafe = weather.Visibility > 5000 && weather.Turbulence != "severe"
	if !decision.WeatherSafe {
		if decision.Mode == "normal" {
			decision.Mode = "loiter"
			decision.ReplanRequired = true
			decision.Reason = "weather unsafe - visibility or turbulence"
		}
	}

	// Check threats
	activeThreatLevel := 0.0
	for _, threat := range threats {
		if threat.Active && threat.ThreatLevel > activeThreatLevel {
			activeThreatLevel = threat.ThreatLevel
		}
	}
	decision.ThreatLevel = activeThreatLevel

	if activeThreatLevel > 0.5 {
		decision.Mode = "evasive"
		decision.ReplanRequired = true
		decision.Reason = fmt.Sprintf("high threat level: %.2f", activeThreatLevel)
	}

	// Check mission viability
	decision.MissionViable = decision.FuelSufficient && decision.Mode != "abort"

	// Record if significant decision
	if decision.Mode != afc.currentDecision.Mode || decision.ReplanRequired {
		record := DecisionRecord{
			ID:         fmt.Sprintf("dec_%d", time.Now().UnixNano()),
			Timestamp:  time.Now(),
			Type:       "situation_assessment",
			Decision:   decision.Mode,
			Reason:     decision.Reason,
			Confidence: decision.Confidence,
			Input: map[string]interface{}{
				"altitude":     state.Altitude,
				"fuel":         state.FuelRemaining,
				"threat_level": activeThreatLevel,
				"visibility":   weather.Visibility,
			},
			LatencyMs: float64(time.Since(startTime).Microseconds()) / 1000,
		}
		afc.decisionHistory = append(afc.decisionHistory, record)

		if afc.onDecision != nil {
			go afc.onDecision(record)
		}

		// Trigger replan if needed
		if decision.ReplanRequired {
			afc.replanCount++
			go afc.executeReplan(decision.Reason)
		}
	}

	afc.currentDecision = decision
}

// handleThreat responds to detected threats
func (afc *AutonomousFlightController) handleThreat(threat Threat) {
	afc.mu.Lock()
	defer afc.mu.Unlock()

	log.Printf("[Autonomous] Threat detected: %s at level %.2f", threat.Type, threat.ThreatLevel)

	state := afc.simulator.GetState()

	// Calculate evasion
	response := ThreatResponse{
		ThreatID:   threat.ID,
		ThreatType: threat.Type,
		Timestamp:  time.Now(),
	}

	// Determine response based on threat type and level
	if threat.ThreatLevel > 0.8 {
		// Critical threat - immediate evasion
		response.ResponseType = "evade"

		// Turn away from threat
		threatBearing := afc.simulator.calculateBearing(
			state.Latitude, state.Longitude,
			threat.Latitude, threat.Longitude,
		)
		response.NewHeading = threatBearing + 180
		if response.NewHeading >= 360 {
			response.NewHeading -= 360
		}

		// Descend to avoid radar
		response.NewAltitude = math.Max(state.Altitude-500, 100)
		response.Success = true

	} else if threat.ThreatLevel > 0.5 {
		// Moderate threat - adjust course
		response.ResponseType = "evade"

		threatBearing := afc.simulator.calculateBearing(
			state.Latitude, state.Longitude,
			threat.Latitude, threat.Longitude,
		)
		// Turn 90 degrees away
		response.NewHeading = threatBearing + 90
		if response.NewHeading >= 360 {
			response.NewHeading -= 360
		}
		response.NewAltitude = state.Altitude
		response.Success = true

	} else {
		// Low threat - monitor
		response.ResponseType = "ignore"
		response.NewHeading = state.Heading
		response.NewAltitude = state.Altitude
		response.Success = true
	}

	afc.threatResponses = append(afc.threatResponses, response)

	if afc.onThreatResponse != nil {
		go afc.onThreatResponse(response)
	}
}

// executeReplan generates a new flight plan
func (afc *AutonomousFlightController) executeReplan(reason string) {
	log.Printf("[Autonomous] Replanning due to: %s", reason)

	state := afc.simulator.GetState()
	weather := afc.simulator.GetWeather()

	// Generate new waypoints avoiding known threats
	threats := afc.simulator.GetThreats()

	// Create a safe route
	newPlan := &FlightPlan{
		ID:          fmt.Sprintf("replan_%d", time.Now().UnixNano()),
		MissionType: "evasive",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Add current position as first waypoint
	newPlan.Waypoints = append(newPlan.Waypoints, Waypoint{
		ID:        "wp_current",
		Name:      "Current Position",
		Latitude:  state.Latitude,
		Longitude: state.Longitude,
		Altitude:  state.Altitude,
		Speed:     state.Airspeed,
	})

	// Add safe waypoint away from threats
	safeHeading := state.Heading
	for _, threat := range threats {
		if threat.Active && threat.ThreatLevel > 0.3 {
			threatBearing := afc.simulator.calculateBearing(
				state.Latitude, state.Longitude,
				threat.Latitude, threat.Longitude,
			)
			// Move away from threat
			safeHeading = threatBearing + 180
		}
	}

	// Calculate safe waypoint 5km away
	R := 6371000.0
	d := 5000.0 // 5km
	safeHeadingRad := safeHeading * math.Pi / 180
	lat1Rad := state.Latitude * math.Pi / 180
	lon1Rad := state.Longitude * math.Pi / 180

	lat2Rad := math.Asin(math.Sin(lat1Rad)*math.Cos(d/R) +
		math.Cos(lat1Rad)*math.Sin(d/R)*math.Cos(safeHeadingRad))
	lon2Rad := lon1Rad + math.Atan2(
		math.Sin(safeHeadingRad)*math.Sin(d/R)*math.Cos(lat1Rad),
		math.Cos(d/R)-math.Sin(lat1Rad)*math.Sin(lat2Rad))

	safeAlt := state.Altitude
	if weather.Visibility < 5000 {
		// Climb above clouds if visibility is low
		safeAlt = math.Max(weather.CloudBase+500, state.Altitude)
	}

	newPlan.Waypoints = append(newPlan.Waypoints, Waypoint{
		ID:        "wp_safe",
		Name:      "Safe Point",
		Latitude:  lat2Rad * 180 / math.Pi,
		Longitude: lon2Rad * 180 / math.Pi,
		Altitude:  safeAlt,
		Speed:     80, // Cruise speed
	})

	// Apply new plan
	afc.simulator.SetFlightPlan(newPlan)

	if afc.onReplan != nil {
		go afc.onReplan(*newPlan, reason)
	}
}

// weatherMonitorLoop monitors weather changes
func (afc *AutonomousFlightController) weatherMonitorLoop(ctx context.Context) {
	defer afc.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-afc.stopCh:
			return
		case <-ticker.C:
			afc.checkWeather(ctx)
		}
	}
}

// checkWeather fetches and evaluates weather
func (afc *AutonomousFlightController) checkWeather(ctx context.Context) {
	state := afc.simulator.GetState()

	weather, err := afc.weatherAPI.GetWeatherAt(ctx, state.Latitude, state.Longitude)
	if err != nil {
		log.Printf("[Autonomous] Weather API error: %v", err)
		return
	}

	afc.simulator.SetWeather(*weather)

	// Log significant weather changes
	if weather.Visibility < 5000 {
		log.Printf("[Autonomous] Low visibility warning: %.0fm", weather.Visibility)
	}
	if weather.Turbulence == "severe" {
		log.Printf("[Autonomous] Severe turbulence warning")
	}
}

// GetStatistics returns controller statistics
func (afc *AutonomousFlightController) GetStatistics() map[string]interface{} {
	afc.mu.RLock()
	defer afc.mu.RUnlock()

	return map[string]interface{}{
		"decisions_made":   len(afc.decisionHistory),
		"replans":          afc.replanCount,
		"threat_responses": len(afc.threatResponses),
		"current_mode":     afc.currentDecision.Mode,
		"mission_viable":   afc.currentDecision.MissionViable,
	}
}
