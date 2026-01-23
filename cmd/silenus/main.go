package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asgard/pandora/internal/orbital/hal"
	"github.com/asgard/pandora/internal/orbital/vision"
	"github.com/asgard/pandora/internal/orbital/tracking"
)

func main() {
	// Command-line flags
	satelliteID := flag.String("id", "sat001", "Satellite ID")
	modelPath := flag.String("model", "models/yolov8n.onnx", "Vision model path")
	flag.Parse()

	log.Printf("Starting ASGARD Silenus (Satellite %s)", *satelliteID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize hardware
	camera := hal.NewMockCamera()
	if err := camera.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize camera: %v", err)
	}
	defer camera.Shutdown()

	powerCtrl := hal.NewMockPowerController()

	// Initialize vision processor
	visionProc := vision.NewMockVisionProcessor()
	if err := visionProc.Initialize(ctx, *modelPath); err != nil {
		log.Fatalf("Failed to initialize vision processor: %v", err)
	}
	defer visionProc.Shutdown()

	log.Printf("Vision Model: %s v%s", visionProc.GetModelInfo().Name, visionProc.GetModelInfo().Version)

	// Create alert channel
	alertChan := make(chan tracking.Alert, 100)

	// Create tracker with criteria
	criteria := vision.AlertCriteria{
		MinConfidence: 0.85,
		AlertClasses:  []string{"fire", "smoke", "aircraft", "ship"},
	}
	tracker := tracking.NewTracker(criteria, alertChan)

	// Start alert processor
	go processAlerts(alertChan)

	// Start vision processing loop
	go runVisionLoop(ctx, camera, visionProc, tracker)

	// Start telemetry loop
	go runTelemetryLoop(ctx, *satelliteID, powerCtrl)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down Silenus...")
	cancel()
	time.Sleep(2 * time.Second) // Allow goroutines to finish
	log.Println("Silenus stopped")
}

func runVisionLoop(ctx context.Context, camera hal.CameraController, visionProc vision.VisionProcessor, tracker *tracking.Tracker) {
	ticker := time.NewTicker(1 * time.Second) // Process 1 frame per second
	defer ticker.Stop()

	frameCount := 0

	for {
		select {
		case <-ticker.C:
			frame, err := camera.CaptureFrame(ctx)
			if err != nil {
				log.Printf("Failed to capture frame: %v", err)
				continue
			}

			frameCount++

			// Run AI detection
			detections, err := visionProc.Detect(ctx, frame)
			if err != nil {
				log.Printf("Detection failed: %v", err)
				continue
			}

			if len(detections) > 0 {
				log.Printf("Frame %d: %d detections", frameCount, len(detections))
				for _, det := range detections {
					log.Printf("  - %s (%.2f confidence)", det.Class, det.Confidence)
				}

				// Process detections for alerts
				tracker.ProcessDetections(ctx, detections)
			}

		case <-ctx.Done():
			return
		}
	}
}

func processAlerts(alertChan <-chan tracking.Alert) {
	for alert := range alertChan {
		log.Printf("=== ALERT ===")
		log.Printf("ID: %s", alert.ID)
		log.Printf("Type: %s", alert.Type)
		log.Printf("Confidence: %.2f", alert.Confidence)
		log.Printf("Time: %s", alert.Timestamp.Format(time.RFC3339))
		log.Printf("============")

		// TODO: Send to Sat_Net for forwarding to Nysus
	}
}

func runTelemetryLoop(ctx context.Context, satelliteID string, powerCtrl hal.PowerController) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			battery, _ := powerCtrl.GetBatteryPercent()
			voltage, _ := powerCtrl.GetBatteryVoltage()
			solarPower, _ := powerCtrl.GetSolarPanelPower()
			inEclipse, _ := powerCtrl.IsInEclipse()

			log.Printf("Telemetry: Battery=%.1f%%, Voltage=%.2fV, Solar=%.1fW, Eclipse=%t",
				battery, voltage, solarPower, inEclipse)

			// TODO: Send telemetry to MongoDB via Sat_Net

		case <-ctx.Done():
			return
		}
	}
}
