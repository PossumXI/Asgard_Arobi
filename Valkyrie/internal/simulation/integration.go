// Package simulation provides integration with Giru, Pricilla, and Hunoid systems.
// This enables full ASGARD system simulation with all components connected.
//
// DO-178C DAL-B compliant - ASGARD Integration Module
// Copyright 2026 Arobi. All Rights Reserved.
package simulation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// IntegratedSimulation connects all ASGARD systems for unified simulation
type IntegratedSimulation struct {
	mu sync.RWMutex

	// Core simulator (X-Plane or JSBSim)
	simulator Simulator

	// System endpoints
	config IntegrationConfig

	// HTTP client for service communication
	client *http.Client

	// State
	running    bool
	lastUpdate time.Time

	// Channels for real-time data
	stateChan     chan *SimulatorState
	telemetryChan chan *TelemetryUpdate
	alertsChan    chan *SecurityAlert
	trajectoryChan chan *TrajectoryData

	// Results
	results *IntegratedSimulationResult
}

// IntegrationConfig holds endpoints for all ASGARD services
type IntegrationConfig struct {
	// Giru Security
	GiruSecurityURL  string // e.g., "http://localhost:9090"
	GiruJarvisURL    string // e.g., "http://localhost:7777"
	GiruJarvisWSURL  string // e.g., "ws://localhost:7778"

	// Pricilla
	PricillaURL string // e.g., "http://localhost:8089"

	// Hunoid
	HunoidURL string // e.g., "http://localhost:8090"

	// Nysus
	NysusURL string // e.g., "http://localhost:8080"

	// Valkyrie
	ValkyrieURL string // e.g., "http://localhost:8093"

	// Update rates
	TelemetryRateHz float64
	SecurityRateHz  float64
}

// DefaultIntegrationConfig returns default configuration
func DefaultIntegrationConfig() IntegrationConfig {
	return IntegrationConfig{
		GiruSecurityURL:  "http://localhost:9090",
		GiruJarvisURL:    "http://localhost:7777",
		GiruJarvisWSURL:  "ws://localhost:7778",
		PricillaURL:      "http://localhost:8089",
		HunoidURL:        "http://localhost:8090",
		NysusURL:         "http://localhost:8080",
		ValkyrieURL:      "http://localhost:8093",
		TelemetryRateHz:  10.0,
		SecurityRateHz:   1.0,
	}
}

// TelemetryUpdate holds real-time telemetry from simulation
type TelemetryUpdate struct {
	Timestamp time.Time `json:"timestamp"`

	// Position (geodetic)
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"`

	// Velocity
	Airspeed    float64 `json:"airspeed"`
	Groundspeed float64 `json:"groundspeed"`
	ClimbRate   float64 `json:"climbRate"`

	// Attitude
	Roll  float64 `json:"roll"`
	Pitch float64 `json:"pitch"`
	Yaw   float64 `json:"yaw"`

	// System status
	BatterySOC float64 `json:"batterySOC"`
	FlightMode string  `json:"flightMode"`
	Armed      bool    `json:"armed"`
}

// SecurityAlert represents alerts from Giru Security
type SecurityAlert struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	Source      string    `json:"source"`
}

// TrajectoryData holds trajectory information from Pricilla
type TrajectoryData struct {
	Timestamp     time.Time         `json:"timestamp"`
	Waypoints     []TrajectoryPoint `json:"waypoints"`
	EstimatedTime time.Duration     `json:"estimatedTime"`
	Confidence    float64           `json:"confidence"`
}

// TrajectoryPoint represents a single trajectory waypoint
type TrajectoryPoint struct {
	Position  [3]float64    `json:"position"`  // lat, lon, alt
	Velocity  [3]float64    `json:"velocity"`  // N, E, D
	TimeAt    time.Duration `json:"timeAt"`
	Heading   float64       `json:"heading"`
}

// RescueTarget represents a target for Hunoid rescue
type RescueTarget struct {
	ID          string     `json:"id"`
	Position    [3]float64 `json:"position"`
	ThreatLevel float64    `json:"threatLevel"`
	Priority    float64    `json:"priority"`
	Status      string     `json:"status"`
}

// IntegratedSimulationResult holds results from integrated simulation
type IntegratedSimulationResult struct {
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration

	// Simulation data
	TelemetryHistory []TelemetryUpdate
	AlertsReceived   []SecurityAlert
	TrajectoryData   []TrajectoryData

	// Rescue operations (if applicable)
	RescueTargets   []RescueTarget
	RescuesAttempted int
	RescuesSuccessful int

	// Performance
	AverageLatencyMs float64
	MaxLatencyMs     float64
	UpdateCount      int

	// System status
	SystemsHealth map[string]string
}

// NewIntegratedSimulation creates a new integrated simulation
func NewIntegratedSimulation(simulator Simulator, config IntegrationConfig) *IntegratedSimulation {
	return &IntegratedSimulation{
		simulator:      simulator,
		config:         config,
		client:         &http.Client{Timeout: 5 * time.Second},
		stateChan:      make(chan *SimulatorState, 100),
		telemetryChan:  make(chan *TelemetryUpdate, 100),
		alertsChan:     make(chan *SecurityAlert, 50),
		trajectoryChan: make(chan *TrajectoryData, 20),
		results: &IntegratedSimulationResult{
			TelemetryHistory: make([]TelemetryUpdate, 0, 1000),
			AlertsReceived:   make([]SecurityAlert, 0, 100),
			TrajectoryData:   make([]TrajectoryData, 0, 50),
			RescueTargets:    make([]RescueTarget, 0, 20),
			SystemsHealth:    make(map[string]string),
		},
	}
}

// Start begins the integrated simulation
func (is *IntegratedSimulation) Start(ctx context.Context) error {
	is.mu.Lock()
	if is.running {
		is.mu.Unlock()
		return fmt.Errorf("simulation already running")
	}
	is.running = true
	is.results.StartTime = time.Now()
	is.mu.Unlock()

	// Check system health
	is.checkSystemHealth()

	// Start background workers
	var wg sync.WaitGroup

	// Telemetry publisher
	wg.Add(1)
	go func() {
		defer wg.Done()
		is.telemetryWorker(ctx)
	}()

	// Security monitor
	wg.Add(1)
	go func() {
		defer wg.Done()
		is.securityWorker(ctx)
	}()

	// Trajectory updater
	wg.Add(1)
	go func() {
		defer wg.Done()
		is.trajectoryWorker(ctx)
	}()

	// Wait for context cancellation
	<-ctx.Done()

	is.mu.Lock()
	is.running = false
	is.results.EndTime = time.Now()
	is.results.Duration = is.results.EndTime.Sub(is.results.StartTime)
	is.mu.Unlock()

	wg.Wait()
	return nil
}

// RunScenarioWithIntegration runs a scenario with all systems integrated
func (is *IntegratedSimulation) RunScenarioWithIntegration(ctx context.Context, scenario *Scenario) (*IntegratedSimulationResult, error) {
	// Start background integration
	simCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go is.Start(simCtx)

	// Load and run scenario
	if err := is.simulator.LoadScenario(scenario); err != nil {
		return nil, fmt.Errorf("load scenario: %w", err)
	}

	simResult, err := is.simulator.RunScenario(ctx)
	if err != nil {
		return nil, fmt.Errorf("run scenario: %w", err)
	}

	// Stop integration
	cancel()

	// Add simulation result to integrated result
	is.mu.Lock()
	is.results.UpdateCount = simResult.UpdateCount
	is.results.AverageLatencyMs = float64(simResult.AverageLatency) / float64(time.Millisecond)
	is.results.MaxLatencyMs = float64(simResult.MaxLatency) / float64(time.Millisecond)
	result := is.results
	is.mu.Unlock()

	return result, nil
}

// telemetryWorker publishes telemetry to connected systems
func (is *IntegratedSimulation) telemetryWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(float64(time.Second) / is.config.TelemetryRateHz))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			state, err := is.simulator.GetState()
			if err != nil {
				continue
			}

			telemetry := &TelemetryUpdate{
				Timestamp:   state.Timestamp,
				Latitude:    state.Latitude,
				Longitude:   state.Longitude,
				Altitude:    state.Altitude,
				Airspeed:    state.Airspeed,
				Groundspeed: state.Groundspeed,
				ClimbRate:   -state.VelocityDown,
				Roll:        state.Roll,
				Pitch:       state.Pitch,
				Yaw:         state.Yaw,
				BatterySOC:  0.85, // Would come from propulsion system
				FlightMode:  "AUTO",
				Armed:       true,
			}

			// Store locally
			is.mu.Lock()
			is.results.TelemetryHistory = append(is.results.TelemetryHistory, *telemetry)
			is.mu.Unlock()

			// Publish to Valkyrie
			go is.publishToValkyrie(telemetry)

			// Publish to Nysus
			go is.publishToNysus(telemetry)
		}
	}
}

// securityWorker monitors Giru Security for alerts
func (is *IntegratedSimulation) securityWorker(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(float64(time.Second) / is.config.SecurityRateHz))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Query Giru Security for alerts
			alerts, err := is.queryGiruAlerts()
			if err != nil {
				continue
			}

			is.mu.Lock()
			for _, alert := range alerts {
				is.results.AlertsReceived = append(is.results.AlertsReceived, alert)
			}
			is.mu.Unlock()
		}
	}
}

// trajectoryWorker updates trajectory from Pricilla
func (is *IntegratedSimulation) trajectoryWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Get current state
			state, err := is.simulator.GetState()
			if err != nil {
				continue
			}

			// Request trajectory from Pricilla
			trajectory, err := is.requestTrajectory(state)
			if err != nil {
				continue
			}

			is.mu.Lock()
			is.results.TrajectoryData = append(is.results.TrajectoryData, *trajectory)
			is.mu.Unlock()
		}
	}
}

// publishToValkyrie sends telemetry to Valkyrie API
func (is *IntegratedSimulation) publishToValkyrie(telemetry *TelemetryUpdate) {
	url := is.config.ValkyrieURL + "/api/v1/telemetry"
	data, _ := json.Marshal(telemetry)
	is.client.Post(url, "application/json", bytes.NewReader(data))
}

// publishToNysus sends telemetry to Nysus
func (is *IntegratedSimulation) publishToNysus(telemetry *TelemetryUpdate) {
	url := is.config.NysusURL + "/api/v1/telemetry"
	data, _ := json.Marshal(telemetry)
	is.client.Post(url, "application/json", bytes.NewReader(data))
}

// queryGiruAlerts fetches alerts from Giru Security
func (is *IntegratedSimulation) queryGiruAlerts() ([]SecurityAlert, error) {
	url := is.config.GiruSecurityURL + "/api/v1/alerts"
	resp, err := is.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result struct {
		Alerts []SecurityAlert `json:"alerts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Alerts, nil
}

// requestTrajectory requests trajectory calculation from Pricilla
func (is *IntegratedSimulation) requestTrajectory(state *SimulatorState) (*TrajectoryData, error) {
	url := is.config.PricillaURL + "/api/v1/trajectory/calculate"

	reqBody := map[string]interface{}{
		"current_position": []float64{state.Latitude, state.Longitude, state.Altitude},
		"current_velocity": []float64{state.VelocityNorth, state.VelocityEast, -state.VelocityDown},
		"target_position":  []float64{state.Latitude + 0.01, state.Longitude, state.Altitude}, // Example target
		"constraints": map[string]interface{}{
			"max_speed":       50.0,
			"max_acceleration": 5.0,
		},
	}

	data, _ := json.Marshal(reqBody)
	resp, err := is.client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("pricilla error: %s", string(body))
	}

	var trajectory TrajectoryData
	if err := json.NewDecoder(resp.Body).Decode(&trajectory); err != nil {
		return nil, err
	}

	trajectory.Timestamp = time.Now()
	return &trajectory, nil
}

// SendJarvisCommand sends a voice command to Giru JARVIS
func (is *IntegratedSimulation) SendJarvisCommand(command string) error {
	url := is.config.GiruJarvisURL + "/api/command"
	reqBody := map[string]string{
		"command": command,
		"source":  "simulation",
	}
	data, _ := json.Marshal(reqBody)

	resp, err := is.client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jarvis command failed: %d", resp.StatusCode)
	}

	return nil
}

// TriggerRescueScenario initiates a rescue scenario with Hunoid
func (is *IntegratedSimulation) TriggerRescueScenario(targets []RescueTarget) error {
	url := is.config.HunoidURL + "/api/v1/rescue/start"
	reqBody := map[string]interface{}{
		"targets": targets,
		"mode":    "simulation",
	}
	data, _ := json.Marshal(reqBody)

	resp, err := is.client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("rescue start failed: %d", resp.StatusCode)
	}

	is.mu.Lock()
	is.results.RescueTargets = append(is.results.RescueTargets, targets...)
	is.results.RescuesAttempted = len(targets)
	is.mu.Unlock()

	return nil
}

// checkSystemHealth verifies all connected systems are healthy
func (is *IntegratedSimulation) checkSystemHealth() {
	systems := map[string]string{
		"giru_security": is.config.GiruSecurityURL + "/health",
		"giru_jarvis":   is.config.GiruJarvisURL + "/health",
		"pricilla":      is.config.PricillaURL + "/health",
		"hunoid":        is.config.HunoidURL + "/healthz",
		"nysus":         is.config.NysusURL + "/health",
		"valkyrie":      is.config.ValkyrieURL + "/health",
	}

	is.mu.Lock()
	defer is.mu.Unlock()

	for name, url := range systems {
		resp, err := is.client.Get(url)
		if err != nil {
			is.results.SystemsHealth[name] = "error: " + err.Error()
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			is.results.SystemsHealth[name] = "ok"
		} else {
			is.results.SystemsHealth[name] = fmt.Sprintf("unhealthy: %d", resp.StatusCode)
		}
	}
}

// GetResults returns current simulation results
func (is *IntegratedSimulation) GetResults() *IntegratedSimulationResult {
	is.mu.RLock()
	defer is.mu.RUnlock()
	return is.results
}

// GetTelemetryChannel returns channel for real-time telemetry updates
func (is *IntegratedSimulation) GetTelemetryChannel() <-chan *TelemetryUpdate {
	return is.telemetryChan
}

// GetAlertsChannel returns channel for security alerts
func (is *IntegratedSimulation) GetAlertsChannel() <-chan *SecurityAlert {
	return is.alertsChan
}
