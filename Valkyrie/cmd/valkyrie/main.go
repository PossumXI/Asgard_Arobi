// VALKYRIE - Autonomous Flight System
// The Tesla Autopilot for Aircraft
//
// Combines Pricilla's precision guidance with Giru's AI security
// for fully autonomous flight control.

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/PossumXI/Asgard/Valkyrie/internal/actuators"
	"github.com/PossumXI/Asgard/Valkyrie/internal/ai"
	"github.com/PossumXI/Asgard/Valkyrie/internal/failsafe"
	"github.com/PossumXI/Asgard/Valkyrie/internal/fusion"
	"github.com/PossumXI/Asgard/Valkyrie/internal/integration"
	"github.com/PossumXI/Asgard/Valkyrie/internal/livefeed"
	"github.com/PossumXI/Asgard/Valkyrie/internal/security"
)

var (
	// Version info
	version   = "1.0.0"
	buildTime = "unknown"
	gitCommit = "unknown"

	// Configuration flags
	httpPort    = flag.Int("http-port", 8093, "HTTP API port")
	metricsPort = flag.Int("metrics-port", 9093, "Metrics port")
	configFile  = flag.String("config", "configs/config.yaml", "Configuration file path")

	// ASGARD endpoints
	nysusURL   = flag.String("nysus", "http://localhost:8080", "Nysus endpoint")
	silenusURL = flag.String("silenus", "http://localhost:9093", "Silenus endpoint")
	satnetURL  = flag.String("satnet", "http://localhost:8081", "Sat_Net endpoint")
	giruURL    = flag.String("giru", "http://localhost:9090", "Giru endpoint")
	hunoidURL  = flag.String("hunoid", "http://localhost:8090", "Hunoid endpoint")

	// Feature flags
	enableSecurity = flag.Bool("security", true, "Enable security monitoring")
	enableAI       = flag.Bool("ai", true, "Enable AI decision engine")
	enableFailsafe = flag.Bool("failsafe", true, "Enable fail-safe systems")
	enableLiveFeed = flag.Bool("livefeed", true, "Enable live telemetry feed")

	// Mode
	simMode = flag.Bool("sim", false, "Simulation mode (no real hardware)")

	// MAVLink
	mavlinkPort = flag.String("mavlink-port", "COM3", "MAVLink serial port")
	mavlinkBaud = flag.Int("mavlink-baud", 921600, "MAVLink baud rate")
)

// Valkyrie is the main application struct
type Valkyrie struct {
	// Core systems
	fusionEngine    *fusion.ExtendedKalmanFilter
	decisionEngine  *ai.DecisionEngine
	shadowMonitor   *security.ShadowMonitor
	emergencySystem *failsafe.EmergencySystem
	liveFeed        *livefeed.LiveFeedStreamer
	mavlink         *actuators.MAVLinkController
	asgardClients   *integration.ASGARDClients

	// HTTP server
	httpServer *http.Server

	// State
	running bool
	mu      sync.RWMutex

	// Context
	ctx    context.Context
	cancel context.CancelFunc
}

func main() {
	flag.Parse()

	// Banner
	printBanner()

	// Context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Create Valkyrie instance
	valkyrie := &Valkyrie{
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize all systems
	if err := valkyrie.Initialize(); err != nil {
		log.Fatalf("Failed to initialize VALKYRIE: %v", err)
	}

	// Start all systems
	if err := valkyrie.Start(); err != nil {
		log.Fatalf("Failed to start VALKYRIE: %v", err)
	}

	log.Println("✅ VALKYRIE is OPERATIONAL")
	log.Println("   Press Ctrl+C to shutdown")

	// Wait for shutdown signal
	<-sigChan
	log.Println("\n🛑 Shutdown signal received, gracefully stopping...")

	// Shutdown
	if err := valkyrie.Shutdown(); err != nil {
		log.Printf("Shutdown error: %v", err)
	}

	log.Println("✅ VALKYRIE shutdown complete")
}

// Initialize sets up all subsystems
func (v *Valkyrie) Initialize() error {
	log.Println("🚀 Initializing VALKYRIE Autonomous Flight System...")

	// 1. Sensor Fusion Engine
	log.Println("   Initializing sensor fusion engine...")
	fusionConfig := fusion.FusionConfig{
		UpdateRate:       100.0, // 100 Hz
		SensorWeights:    make(map[fusion.SensorType]float64),
		OutlierThreshold: 3.0,
		MinSensors:       2,
		EnableAdaptive:   true,
	}
	fusionConfig.SensorWeights[fusion.SensorGPS] = 1.0
	fusionConfig.SensorWeights[fusion.SensorINS] = 0.9
	fusionConfig.SensorWeights[fusion.SensorRADAR] = 0.7
	fusionConfig.SensorWeights[fusion.SensorLIDAR] = 0.8
	fusionConfig.SensorWeights[fusion.SensorBarometer] = 0.6

	v.fusionEngine = fusion.NewEKF(fusionConfig)
	log.Println("   ✓ Sensor fusion engine initialized")

	// 2. MAVLink Controller
	log.Println("   Initializing MAVLink controller...")
	mavlinkConfig := actuators.MAVLinkConfig{
		Port:           *mavlinkPort,
		BaudRate:       *mavlinkBaud,
		SimulationMode: *simMode,
	}
	v.mavlink = actuators.NewMAVLinkController(mavlinkConfig)
	log.Println("   ✓ MAVLink controller initialized")

	// 2. AI Decision Engine
	if *enableAI {
		log.Println("   Initializing AI decision engine...")
		aiConfig := ai.DecisionConfig{
			SafetyPriority:     0.9,
			EfficiencyPriority: 0.7,
			StealthPriority:    0.5,
			MaxRollAngle:       0.785, // 45 degrees
			MaxPitchAngle:      0.524, // 30 degrees
			MaxYawRate:         0.349, // 20 deg/s
			MinSafeAltitude:    100.0,
			MaxVerticalSpeed:   10.0,
			EnableAutoland:     true,
			EnableThreatAvoid:  true,
			EnableWeatherAvoid: true,
			DecisionRate:       50.0, // 50 Hz
		}
		v.decisionEngine = ai.NewDecisionEngine(aiConfig, v.asgardClients)
		if err := v.decisionEngine.Initialize(v.ctx); err != nil {
			return fmt.Errorf("failed to initialize AI engine: %w", err)
		}
		v.decisionEngine.SetFusionEngine(v.fusionEngine)
		log.Println("   ✓ AI decision engine initialized")
	}

	// 3. Security Monitor
	if *enableSecurity {
		log.Println("   Initializing security monitor...")
		secConfig := security.ShadowConfig{
			MonitorFlightController: true,
			MonitorSensorDrivers:    true,
			MonitorNavigation:       true,
			MonitorCommunication:    true,
			AnomalyThreshold:        0.7,
			ResponseMode:            security.ResponseModeAlert,
			ScanInterval:            100 * time.Millisecond,
		}
		v.shadowMonitor = security.NewShadowMonitor(secConfig, v.asgardClients)
		log.Println("   ✓ Security monitor initialized")
	}

	// 4. Fail-Safe System
	if *enableFailsafe {
		log.Println("   Initializing fail-safe system...")
		failConfig := failsafe.FailsafeConfig{
			EnableAutoRTB:       true,
			EnableAutoLand:      true,
			EnableParachute:     false,
			MinSafeAltitudeAGL:  50.0,
			MinSafeFuel:         0.15,
			MinSafeBattery:      0.20,
			MaxTimeWithoutComms: 5 * time.Minute,
			RTBLocation:         [3]float64{0, 0, 500},
			CheckInterval:       100 * time.Millisecond,
		}
		mavlinkAdapter := failsafe.NewMAVLinkAdapter(v.mavlink)
		v.emergencySystem = failsafe.NewEmergencySystem(failConfig, mavlinkAdapter)
		log.Println("   ✓ Fail-safe system initialized")
	}

	// 5. LiveFeed Streamer
	if *enableLiveFeed {
		log.Println("   Initializing live feed streamer...")
		v.liveFeed = livefeed.NewLiveFeedStreamer()
		log.Println("   ✓ Live feed streamer initialized")
	}

	return nil
}

// Start begins all subsystems
func (v *Valkyrie) Start() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	var wg sync.WaitGroup

	// Start fusion engine
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := v.fusionEngine.Run(v.ctx); err != nil && err != context.Canceled {
			log.Printf("Fusion engine error: %v", err)
		}
	}()

	// Start AI decision engine
	if *enableAI && v.decisionEngine != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := v.decisionEngine.Run(v.ctx); err != nil && err != context.Canceled {
				log.Printf("Decision engine error: %v", err)
			}
		}()
	}

	// Start security monitor
	if *enableSecurity && v.shadowMonitor != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := v.shadowMonitor.Start(v.ctx); err != nil && err != context.Canceled {
				log.Printf("Security monitor error: %v", err)
			}
		}()
	}

	// Start fail-safe monitor
	if *enableFailsafe && v.emergencySystem != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := v.emergencySystem.Monitor(v.ctx); err != nil && err != context.Canceled {
				log.Printf("Emergency system error: %v", err)
			}
		}()
	}

	// Start live feed streamer
	if *enableLiveFeed && v.liveFeed != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := v.liveFeed.Run(v.ctx); err != nil && err != context.Canceled {
				log.Printf("LiveFeed error: %v", err)
			}
		}()
	}

	// Start MAVLink controller
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := v.mavlink.Run(v.ctx); err != nil && err != context.Canceled {
			log.Printf("MAVLink error: %v", err)
		}
	}()

	// Start telemetry broadcasting
	go v.broadcastTelemetry()

	// Start HTTP server
	if err := v.startHTTPServer(); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	v.running = true

	return nil
}

// Shutdown gracefully stops all subsystems
func (v *Valkyrie) Shutdown() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Cancel context
	v.cancel()

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if v.httpServer != nil {
		if err := v.httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP shutdown error: %v", err)
		}
	}

	// Disconnect MAVLink
	if v.mavlink != nil {
		v.mavlink.Disconnect()
	}

	v.running = false

	return nil
}

// startHTTPServer starts the HTTP API server
func (v *Valkyrie) startHTTPServer() error {
	mux := http.NewServeMux()

	// Health and status
	mux.HandleFunc("/health", v.healthHandler)
	mux.HandleFunc("/api/v1/status", v.statusHandler)
	mux.HandleFunc("/api/v1/state", v.stateHandler)
	mux.HandleFunc("/api/v1/version", v.versionHandler)

	// Mission control
	mux.HandleFunc("/api/v1/mission", v.missionHandler)
	mux.HandleFunc("/api/v1/mission/waypoints", v.waypointsHandler)

	// Flight control
	mux.HandleFunc("/api/v1/arm", v.armHandler)
	mux.HandleFunc("/api/v1/disarm", v.disarmHandler)
	mux.HandleFunc("/api/v1/mode", v.modeHandler)

	// Emergency
	mux.HandleFunc("/api/v1/emergency/rtb", v.rtbHandler)
	mux.HandleFunc("/api/v1/emergency/land", v.landHandler)

	// WebSocket for live telemetry
	if v.liveFeed != nil {
		mux.HandleFunc("/ws/telemetry", v.liveFeed.HandleWebSocket)
	}

	v.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", *httpPort),
		Handler: mux,
	}

	go func() {
		log.Printf("🌐 HTTP API listening on :%d", *httpPort)
		if err := v.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	return nil
}

// broadcastTelemetry sends telemetry to live feed
func (v *Valkyrie) broadcastTelemetry() {
	ticker := time.NewTicker(100 * time.Millisecond) // 10 Hz
	defer ticker.Stop()

	for {
		select {
		case <-v.ctx.Done():
			return
		case <-ticker.C:
			if v.liveFeed != nil && v.fusionEngine != nil {
				state := v.fusionEngine.GetState()
				msg := &livefeed.TelemetryMessage{
					Timestamp:        time.Now(),
					Position:         state.Position,
					Velocity:         state.Velocity,
					Attitude:         state.Attitude,
					AngularRate:      state.AngularRate,
					Acceleration:     state.Acceleration,
					Status:           "operational",
					FlightMode:       v.mavlink.GetFlightMode(),
					FusionConfidence: state.Confidence,
					Clearance:        livefeed.ClearanceBasic,
				}
				v.liveFeed.BroadcastTelemetry(msg)
			}
		}
	}
}

// HTTP Handlers

func (v *Valkyrie) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"service": "valkyrie",
		"version": version,
	})
}

func (v *Valkyrie) statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	state := v.fusionEngine.GetState()
	securityHealthy := true
	if v.shadowMonitor != nil {
		securityHealthy = v.shadowMonitor.IsHealthy()
	}
	failsafeHealthy := true
	if v.emergencySystem != nil {
		failsafeHealthy = v.emergencySystem.IsHealthy()
	}

	status := map[string]interface{}{
		"fusion_active":     true,
		"ai_active":         *enableAI,
		"security_active":   *enableSecurity,
		"security_healthy":  securityHealthy,
		"failsafe_active":   *enableFailsafe,
		"failsafe_healthy":  failsafeHealthy,
		"simulation_mode":   *simMode,
		"mavlink_connected": v.mavlink.IsConnected(),
		"armed":             v.mavlink.IsArmed(),
		"flight_mode":       v.mavlink.GetFlightMode(),
		"position":          state.Position,
		"velocity":          state.Velocity,
		"confidence":        state.Confidence,
	}

	json.NewEncoder(w).Encode(status)
}

func (v *Valkyrie) stateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	state := v.fusionEngine.GetState()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"position": map[string]float64{
			"x": state.Position[0],
			"y": state.Position[1],
			"z": state.Position[2],
		},
		"velocity": map[string]float64{
			"x": state.Velocity[0],
			"y": state.Velocity[1],
			"z": state.Velocity[2],
		},
		"attitude": map[string]float64{
			"roll":  state.Attitude[0],
			"pitch": state.Attitude[1],
			"yaw":   state.Attitude[2],
		},
		"angular_rate": map[string]float64{
			"p": state.AngularRate[0],
			"q": state.AngularRate[1],
			"r": state.AngularRate[2],
		},
		"timestamp":  state.Timestamp.Format(time.RFC3339),
		"confidence": state.Confidence,
	})
}

func (v *Valkyrie) versionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"version":    version,
		"build_time": buildTime,
		"git_commit": gitCommit,
	})
}

func (v *Valkyrie) missionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "GET" {
		if v.decisionEngine != nil {
			status := v.decisionEngine.GetMissionStatus()
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": status,
			})
		} else {
			json.NewEncoder(w).Encode(map[string]string{"status": "no_ai"})
		}
		return
	}

	if r.Method == "POST" {
		// TODO: Parse and set new mission
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"message": "mission accepted"})
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (v *Valkyrie) waypointsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

func (v *Valkyrie) armHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if err := v.mavlink.Arm(v.ctx); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "armed"})
}

func (v *Valkyrie) disarmHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if err := v.mavlink.Disarm(v.ctx); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "disarmed"})
}

func (v *Valkyrie) modeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "GET" {
		json.NewEncoder(w).Encode(map[string]string{
			"mode": v.mavlink.GetFlightMode(),
		})
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (v *Valkyrie) rtbHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	log.Println("⚠️ Emergency RTB initiated")
	// TODO: Trigger RTB

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "rtb_initiated"})
}

func (v *Valkyrie) landHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	log.Println("⚠️ Emergency landing initiated")
	// TODO: Trigger emergency landing

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "landing_initiated"})
}

func printBanner() {
	banner := `
╦  ╦╔═╗╦  ╦╔═╦ ╦╦═╗╦╔═╗
╚╗╔╝╠═╣║  ╠╩╗╚╦╝╠╦╝║║╣ 
 ╚╝ ╩ ╩╩═╝╩ ╩ ╩ ╩╚═╩╚═╝
Autonomous Flight System v` + version + `
Powered by PRICILLA + GIRU AI

`
	fmt.Println(banner)
}
