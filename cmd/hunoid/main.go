package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asgard/pandora/internal/robotics/control"
	"github.com/asgard/pandora/internal/robotics/ethics"
	"github.com/asgard/pandora/internal/robotics/vla"
)

func main() {
	// Command-line flags
	hunoidID := flag.String("id", "hunoid001", "Hunoid ID")
	serialNum := flag.String("serial", "HND-2026-001", "Serial number")
	flag.Parse()

	log.Printf("Starting ASGARD Hunoid: %s (%s)", *hunoidID, *serialNum)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize robot controller
	robot := control.NewMockHunoid(*hunoidID)
	if err := robot.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize robot: %v", err)
	}
	log.Println("Robot controller initialized")

	// Initialize manipulator
	manipulator := control.NewMockManipulator()
	log.Println("Manipulator initialized")

	// Initialize VLA model
	vlaModel := vla.NewMockVLA()
	if err := vlaModel.Initialize(ctx, "models/openvla.onnx"); err != nil {
		log.Fatalf("Failed to initialize VLA: %v", err)
	}
	defer vlaModel.Shutdown()

	modelInfo := vlaModel.GetModelInfo()
	log.Printf("VLA Model: %s v%s", modelInfo.Name, modelInfo.Version)

	// Initialize ethical kernel
	ethicsKernel := ethics.NewEthicalKernel()
	log.Println("Ethical kernel initialized")

	// Start telemetry reporting
	go reportTelemetry(ctx, robot)

	// Start command processing loop
	go processCommands(ctx, robot, manipulator, vlaModel, ethicsKernel)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down Hunoid...")
	cancel()
	time.Sleep(2 * time.Second)
	log.Println("Hunoid stopped")
}

func reportTelemetry(ctx context.Context, robot *control.MockHunoid) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pose, _ := robot.GetCurrentPose()
			battery := robot.GetBatteryPercent()
			isMoving := robot.IsMoving()

			log.Printf("Telemetry: Position=(%.2f, %.2f, %.2f), Battery=%.1f%%, Moving=%t",
				pose.Position.X, pose.Position.Y, pose.Position.Z, battery, isMoving)

			// TODO: Send to MongoDB via NATS

		case <-ctx.Done():
			return
		}
	}
}

func processCommands(ctx context.Context, robot *control.MockHunoid, manip *control.MockManipulator, vlaModel vla.VLAModel, ethicsKernel *ethics.EthicalKernel) {
	// Simulate receiving commands
	testCommands := []string{
		"Navigate to the supply depot",
		"Pick up the medical kit",
		"Move to the injured person",
		"Put down the medical kit gently",
		"Inspect the area for hazards",
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	commandIdx := 0

	for {
		select {
		case <-ticker.C:
			if commandIdx >= len(testCommands) {
				commandIdx = 0
			}

			command := testCommands[commandIdx]
			commandIdx++

			log.Printf("Received command: '%s'", command)

			// Use VLA to infer action
			action, err := vlaModel.InferAction(ctx, []byte{}, command)
			if err != nil {
				log.Printf("VLA inference failed: %v", err)
				continue
			}

			log.Printf("VLA inferred action: %s (confidence: %.2f)", action.Type, action.Confidence)

			// Ethical evaluation
			decision, err := ethicsKernel.Evaluate(ctx, action)
			if err != nil {
				log.Printf("Ethical evaluation failed: %v", err)
				continue
			}

			log.Printf("Ethical decision: %s - %s (score: %.2f)", decision.Decision, decision.Reasoning, decision.Score)

			if decision.Decision != ethics.DecisionApproved {
				log.Printf("Action blocked by ethical kernel")
				continue
			}

			// Execute action
			if err := executeAction(ctx, robot, manip, action); err != nil {
				log.Printf("Action execution failed: %v", err)
				continue
			}

			log.Printf("Action completed successfully")

		case <-ctx.Done():
			return
		}
	}
}

func executeAction(ctx context.Context, robot *control.MockHunoid, manip *control.MockManipulator, action *vla.Action) error {
	switch action.Type {
	case vla.ActionNavigate:
		x, _ := action.Parameters["x"].(float64)
		y, _ := action.Parameters["y"].(float64)
		z, _ := action.Parameters["z"].(float64)

		targetPose := control.Pose{
			Position:    control.Vector3{X: x, Y: y, Z: z},
			Orientation: control.Quaternion{W: 1, X: 0, Y: 0, Z: 0},
		}

		return robot.MoveTo(ctx, targetPose)

	case vla.ActionPickUp:
		return manip.CloseGripper()

	case vla.ActionPutDown:
		return manip.OpenGripper()

	case vla.ActionOpen:
		return manip.OpenGripper()

	case vla.ActionClose:
		return manip.CloseGripper()

	case vla.ActionInspect:
		if duration, ok := action.Parameters["duration_seconds"].(int); ok {
			time.Sleep(time.Duration(duration) * time.Second)
		}
		return nil

	case vla.ActionWait:
		time.Sleep(2 * time.Second)
		return nil

	default:
		return nil
	}
}
