package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asgard/pandora/Percila/internal/guidance"
	"github.com/asgard/pandora/Percila/internal/integration"
	"github.com/asgard/pandora/Percila/internal/stealth"
	"github.com/asgard/pandora/internal/nysus/events"
	"github.com/asgard/pandora/internal/platform/dtn"
)

func main() {
	systemID := flag.String("id", "percila001", "Percila system ID")
	flag.Parse()

	log.Printf("Starting ASGARD Percila - Advanced Guidance System %s", *systemID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Stealth Optimizer
	stealthOpt := stealth.NewStealthOptimizer()
	log.Println("Stealth Optimizer initialized")

	// Initialize AI Guidance Engine
	guidanceEngine := guidance.NewAIGuidanceEngine(stealthOpt)
	log.Println("AI Guidance Engine initialized")

	// Initialize DTN Node for Sat_Net integration
	storage := dtn.NewInMemoryStorage(10000)
	router := dtn.NewContactGraphRouter("dtn://asgard/percila")
	config := dtn.DefaultNodeConfig()
	dtnNode := dtn.NewNode(*systemID, "dtn://asgard/percila", storage, router, config)
	if err := dtnNode.Start(); err != nil {
		log.Fatalf("Failed to start DTN node: %v", err)
	}
	defer dtnNode.Stop()

	log.Println("DTN Node connected to Sat_Net")

	// Initialize Event Bus for Nysus integration
	eventBus := events.NewEventBus()
	eventBus.Start()
	defer eventBus.Stop()

	log.Println("Event Bus connected to Nysus")

	// Create System Coordinator
	coordinator := integration.NewSystemCoordinator(guidanceEngine, dtnNode, eventBus)
	log.Println("System Coordinator ready")

	// Run test mission
	go runTestMission(ctx, coordinator, guidanceEngine, stealthOpt)

	// Start mission command listener
	go listenForMissionCommands(ctx, coordinator)

	// Wait for shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down Percila...")
	cancel()
	time.Sleep(2 * time.Second)
	log.Println("Percila stopped")
}

func runTestMission(ctx context.Context, coord *integration.SystemCoordinator, engine *guidance.AIGuidanceEngine, stealthOpt *stealth.StealthOptimizer) {
	time.Sleep(5 * time.Second) // Wait for system initialization

	log.Println("=== PERCILA TEST MISSION ===")

	// Test 1: Plan trajectory for Hunoid
	log.Println("\nTest 1: Hunoid Navigation")
	req := guidance.TrajectoryRequest{
		PayloadType:    guidance.PayloadHunoid,
		StartPosition:  guidance.Vector3D{X: 0, Y: 0, Z: 0},
		TargetPosition: guidance.Vector3D{X: 10000, Y: 5000, Z: 0},
		Priority:       guidance.PriorityHigh,
		Constraints: guidance.MissionConstraints{
			StealthRequired: false,
		},
	}

	traj, err := engine.PlanTrajectory(ctx, req)
	if err != nil {
		log.Printf("Trajectory planning failed: %v", err)
		return
	}

	log.Printf("Trajectory planned: %d waypoints, distance: %.2fm, time: %s",
		len(traj.Waypoints), traj.TotalDistance, traj.EstimatedTime)

	// Test 2: Stealth-optimized rocket trajectory
	log.Println("\nTest 2: Stealth Rocket Mission")
	rocketReq := guidance.TrajectoryRequest{
		PayloadType:    guidance.PayloadRocket,
		StartPosition:  guidance.Vector3D{X: 0, Y: 0, Z: 100},
		TargetPosition: guidance.Vector3D{X: 50000, Y: 30000, Z: 5000},
		Priority:       guidance.PriorityCritical,
		Constraints: guidance.MissionConstraints{
			StealthRequired:  true,
			MaxDetectionRisk: 0.2,
		},
		StealthMode: guidance.StealthModeHigh,
	}

	rocketTraj, err := engine.PlanTrajectory(ctx, rocketReq)
	if err != nil {
		log.Printf("Rocket trajectory failed: %v", err)
		return
	}

	// Optimize for stealth
	stealthTraj, err := engine.OptimizeForStealth(rocketTraj)
	if err != nil {
		log.Printf("Stealth optimization failed: %v", err)
		return
	}

	log.Printf("Stealth trajectory: score %.2f, threat exposure: %.2f",
		stealthTraj.StealthScore, stealthTraj.ThreatExposure)

	// Calculate RCS for waypoints
	for i, wp := range stealthTraj.Waypoints {
		rcs := stealthOpt.CalculateRCS(wp, 0)
		thermalSig := stealthOpt.CalculateThermalSignature(wp)
		log.Printf("  WP%d: Alt=%.0fm, RCS=%.2fm², Thermal=%.2f", i, wp.Position.Z, rcs, thermalSig)
	}

	// Test 3: Start integrated mission
	log.Println("\nTest 3: Integrated Mission with System Coordination")
	err = coord.StartGuidedMission(ctx, "hunoid001", guidance.PayloadHunoid, guidance.Vector3D{X: 15000, Y: 8000, Z: 0})
	if err != nil {
		log.Printf("Failed to start mission: %v", err)
		return
	}

	log.Println("=== TEST MISSION COMPLETE ===")
}

func listenForMissionCommands(ctx context.Context, coord *integration.SystemCoordinator) {
	// In production, this would listen to NATS or gRPC for mission requests
	// For now, it's a placeholder

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("Percila: Listening for mission commands...")
		case <-ctx.Done():
			return
		}
	}
}
