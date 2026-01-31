package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/asgard/pandora/internal/orbital/hal"
	"github.com/asgard/pandora/internal/orbital/tracking"
	"github.com/asgard/pandora/internal/orbital/vision"
	"github.com/asgard/pandora/internal/platform/db"
	"github.com/asgard/pandora/internal/platform/dtn"
	"github.com/asgard/pandora/internal/platform/observability"
	"github.com/asgard/pandora/pkg/bundle"
)

func main() {
	// Command-line flags
	satelliteID := flag.String("id", "sat001", "Satellite ID")
	modelPath := flag.String("model", "models/yolov8n.onnx", "Vision model path")
	visionBackend := flag.String("vision-backend", "simple", "Vision backend: simple, tflite")
	alertMinConfidence := flag.Float64("alert-min-confidence", 0.85, "Alert confidence threshold")
	alertEID := flag.String("alert-eid", "dtn://earth/nysus/alerts", "Alert destination EID")
	telemetryEID := flag.String("telemetry-eid", "dtn://earth/nysus/telemetry", "Telemetry destination EID")
	metricsAddr := flag.String("metrics-addr", ":9093", "Metrics server address")
	flag.Parse()

	log.Printf("Starting ASGARD Silenus (Satellite %s)", *satelliteID)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownTracing, err := observability.InitTracing(context.Background(), "silenus")
	if err != nil {
		log.Printf("Tracing disabled: %v", err)
	} else {
		defer func() {
			if err := shutdownTracing(context.Background()); err != nil {
				log.Printf("Tracing shutdown error: %v", err)
			}
		}()
	}

	bypassHardware := getEnvBool("SILENUS_BYPASS_HARDWARE", false)

	// Initialize hardware (or fall back to simulated controllers)
	var camera hal.CameraController
	var powerCtrl hal.PowerController
	var gpsCtrl hal.GPSController

	if bypassHardware {
		log.Println("Hardware bypass enabled; using simulated camera/power/GPS controllers.")
		camera = newMockCamera(*satelliteID)
		powerCtrl = newMockPowerController()
		gpsCtrl = newMockGPSController()
		if err := camera.Initialize(ctx); err != nil {
			log.Fatalf("Failed to initialize mock camera: %v", err)
		}
		defer camera.Shutdown()
	} else {
		cameraConfig, err := loadCameraConfig()
		if err != nil {
			log.Fatalf("Camera configuration error: %v", err)
		}
		camera = hal.NewCamera(cameraConfig)
		if err := camera.Initialize(ctx); err != nil {
			log.Fatalf("Failed to initialize camera: %v", err)
		}
		defer camera.Shutdown()

		powerEndpoint := os.Getenv("POWER_CONTROLLER_URL")
		powerCtrl, err = hal.NewRemotePowerController(powerEndpoint)
		if err != nil {
			log.Fatalf("Failed to initialize power controller: %v", err)
		}

		// Use real orbital position tracking
		orbitalCfg := hal.DefaultOrbitalConfig()
		if n2yoKey := os.Getenv("N2YO_API_KEY"); n2yoKey != "" {
			orbitalCfg.N2YOAPIKey = n2yoKey
		}
		realPos, err := hal.NewRealOrbitalPosition(orbitalCfg)
		if err != nil {
			log.Fatalf("Failed to initialize orbital position provider: %v", err)
		}
		gpsCtrl = realPos
	}

	// Initialize vision processor - use real implementations only
	var visionProc vision.VisionProcessor
	switch *visionBackend {
	case "simple":
		visionProc = vision.NewSimpleVisionProcessor()
	case "tflite":
		visionProc = vision.NewTFLiteVisionProcessor()
	default:
		log.Printf("Unknown vision backend '%s', defaulting to 'simple'", *visionBackend)
		visionProc = vision.NewSimpleVisionProcessor()
	}
	if err := visionProc.Initialize(ctx, *modelPath); err != nil {
		log.Fatalf("Failed to initialize vision processor: %v", err)
	}
	defer visionProc.Shutdown()

	log.Printf("Vision Model: %s v%s", visionProc.GetModelInfo().Name, visionProc.GetModelInfo().Version)

	// Initialize Sat_Net node for forwarding alerts/telemetry
	nodeID := fmt.Sprintf("silenus-%s", *satelliteID)
	nodeEID := fmt.Sprintf("dtn://silenus/%s", *satelliteID)
	storage, cleanup := buildStorage()
	defer cleanup()
	router := dtn.NewEnergyAwareRouter(nodeEID)
	transportConfig := dtn.DefaultTCPTransportConfig()
	transportConfig.ListenAddress = getEnvDefault("SATNET_LISTEN_ADDR", ":4556")
	transport := dtn.NewTCPTransport(nodeID, transportConfig)

	satnetNode := dtn.NewNodeWithTransport(nodeID, nodeEID, storage, router, transport, dtn.DefaultNodeConfig())
	if err := satnetNode.Start(); err != nil {
		log.Fatalf("Failed to start Sat_Net node: %v", err)
	}
	defer satnetNode.Stop()

	gatewayAddr := os.Getenv("SATNET_GATEWAY_ADDR")
	if gatewayAddr == "" {
		if bypassHardware {
			log.Println("SATNET_GATEWAY_ADDR not set; running in offline Sat_Net mode.")
		} else {
			log.Fatalf("SATNET_GATEWAY_ADDR is required for Sat_Net connectivity")
		}
	} else {
		registerSatNetNeighbor(satnetNode, transport, gatewayAddr)
	}

	// Create alert channel
	alertChan := make(chan tracking.Alert, 100)

	// Create tracker with criteria
	criteria := vision.AlertCriteria{
		MinConfidence: *alertMinConfidence,
		AlertClasses:  []string{"fire", "smoke", "aircraft", "ship"},
	}
	frameBuffer := newFrameBuffer(100)
	tracker := tracking.NewTracker(
		criteria,
		alertChan,
		func(ctx context.Context) (string, error) {
			lat, lon, alt, err := gpsCtrl.GetPosition()
			if err != nil {
				return "", err
			}
			// Convert altitude from km to meters for display
			return fmt.Sprintf("lat=%.6f, lon=%.6f, alt=%.0fm", lat, lon, alt*1000), nil
		},
		func(ctx context.Context) ([]byte, error) {
			frames := frameBuffer.Snapshot()
			clip := make([]clipFrame, 0, len(frames))
			for _, sample := range frames {
				clip = append(clip, clipFrame{
					Timestamp: sample.Timestamp.Format(time.RFC3339Nano),
					Data:      base64.StdEncoding.EncodeToString(sample.Data),
				})
			}

			payload := clipPayload{
				Format: "jpeg_sequence",
				Frames: clip,
			}

			return json.Marshal(payload)
		},
	)

	// Start alert processor
	go processAlerts(ctx, alertChan, satnetNode, *alertEID)

	// Start vision processing loop
	go runVisionLoop(ctx, camera, visionProc, tracker, frameBuffer)

	// Start telemetry loop
	go runTelemetryLoop(ctx, *satelliteID, powerCtrl, gpsCtrl, satnetNode, *telemetryEID)

	metricsServer := startMetricsServer(*metricsAddr)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down Silenus...")
	cancel()
	shutdownMetricsServer(metricsServer)
	time.Sleep(2 * time.Second) // Allow goroutines to finish
	log.Println("Silenus stopped")
}

func runVisionLoop(ctx context.Context, camera hal.CameraController, visionProc vision.VisionProcessor, tracker *tracking.Tracker, buffer *frameBuffer) {
	frameChan := make(chan []byte, 200)
	if err := camera.StartStream(ctx, frameChan); err != nil {
		log.Fatalf("Failed to start camera stream: %v", err)
	}
	defer camera.StopStream()

	frameCount := 0

	for {
		select {
		case frame := <-frameChan:
			if frame == nil {
				continue
			}
			frameCount++
			buffer.Add(frame)

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

				tracker.ProcessDetections(ctx, detections)
			}

		case <-ctx.Done():
			return
		}
	}
}

func processAlerts(ctx context.Context, alertChan <-chan tracking.Alert, node *dtn.Node, alertEID string) {
	for {
		select {
		case <-ctx.Done():
			return
		case alert := <-alertChan:
			payload := alertPayload{
				ID:         alert.ID.String(),
				Type:       alert.Type,
				Confidence: alert.Confidence,
				Location:   alert.Location,
				Timestamp:  alert.Timestamp.Format(time.RFC3339Nano),
				VideoClip:  base64.StdEncoding.EncodeToString(alert.VideoClip),
			}

			data, err := json.Marshal(payload)
			if err != nil {
				log.Printf("Failed to serialize alert: %v", err)
				continue
			}

			if err := node.CreateBundle(alertEID, data, bundle.PriorityExpedited); err != nil {
				log.Printf("Failed to send alert bundle: %v", err)
				continue
			}

			log.Printf("Alert forwarded to Sat_Net: %s", alert.ID)
		}
	}
}

func runTelemetryLoop(ctx context.Context, satelliteID string, powerCtrl hal.PowerController, gpsCtrl hal.GPSController, node *dtn.Node, telemetryEID string) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			battery, _ := powerCtrl.GetBatteryPercent()
			voltage, _ := powerCtrl.GetBatteryVoltage()
			solarPower, _ := powerCtrl.GetSolarPanelPower()
			inEclipse, _ := powerCtrl.IsInEclipse()
			lat, lon, alt, _ := gpsCtrl.GetPosition()

			log.Printf("Telemetry: Battery=%.1f%%, Voltage=%.2fV, Solar=%.1fW, Eclipse=%t",
				battery, voltage, solarPower, inEclipse)

			telemetry := telemetryPayload{
				SatelliteID: satelliteID,
				Timestamp:   time.Now().UTC().Format(time.RFC3339Nano),
				Battery:     battery,
				Voltage:     voltage,
				SolarPower:  solarPower,
				Eclipse:     inEclipse,
			Latitude:    lat,
			Longitude:   lon,
			Altitude:    alt * 1000, // Convert km to meters
			}

			data, err := json.Marshal(telemetry)
			if err != nil {
				log.Printf("Failed to serialize telemetry: %v", err)
				continue
			}

			if err := node.CreateBundle(telemetryEID, data, bundle.PriorityNormal); err != nil {
				log.Printf("Failed to send telemetry bundle: %v", err)
			}

		case <-ctx.Done():
			return
		}
	}
}

type mockCamera struct {
	satelliteID string
	stopChan    chan struct{}
}

func newMockCamera(satelliteID string) *mockCamera {
	return &mockCamera{
		satelliteID: satelliteID,
		stopChan:    make(chan struct{}),
	}
}

func (m *mockCamera) Initialize(ctx context.Context) error { return nil }
func (m *mockCamera) Shutdown() error {
	return m.StopStream()
}
func (m *mockCamera) CaptureFrame(ctx context.Context) ([]byte, error) {
	return generateMockFrame(), nil
}
func (m *mockCamera) StartStream(ctx context.Context, frameChan chan<- []byte) error {
	ticker := time.NewTicker(1 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-m.stopChan:
				return
			case <-ticker.C:
				frame := generateMockFrame()
				select {
				case frameChan <- frame:
				default:
				}
			}
		}
	}()
	return nil
}
func (m *mockCamera) StopStream() error {
	select {
	case <-m.stopChan:
	default:
		close(m.stopChan)
	}
	return nil
}
func (m *mockCamera) SetExposure(microseconds int) error { return nil }
func (m *mockCamera) SetGain(gain float64) error        { return nil }
func (m *mockCamera) GetDiagnostics() (hal.CameraDiagnostics, error) {
	return hal.CameraDiagnostics{
		Temperature: 24.5,
		Voltage:     12.1,
		FrameCount:  0,
		ErrorCount:  0,
	}, nil
}

type mockPowerController struct {
	start time.Time
}

func newMockPowerController() *mockPowerController {
	return &mockPowerController{start: time.Now()}
}

func (m *mockPowerController) GetBatteryPercent() (float64, error) {
	elapsed := time.Since(m.start).Minutes()
	percent := 95.0 - elapsed*0.05
	if percent < 20.0 {
		percent = 20.0
	}
	return percent, nil
}
func (m *mockPowerController) GetBatteryVoltage() (float64, error) {
	return 12.2, nil
}
func (m *mockPowerController) GetSolarPanelPower() (float64, error) {
	return 320.0, nil
}
func (m *mockPowerController) IsInEclipse() (bool, error) {
	return false, nil
}
func (m *mockPowerController) SetPowerMode(mode hal.PowerMode) error {
	return nil
}

type mockGPSController struct{}

func newMockGPSController() *mockGPSController { return &mockGPSController{} }
func (m *mockGPSController) GetPosition() (lat, lon, alt float64, err error) {
	return 37.7749, -122.4194, 408000.0, nil
}
func (m *mockGPSController) GetTime() (time.Time, error) {
	return time.Now().UTC(), nil
}
func (m *mockGPSController) GetVelocity() (vx, vy, vz float64, err error) {
	return 0, 0, 0, nil
}

func generateMockFrame() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 640, 480))
	bg := color.RGBA{R: 12, G: 16, B: 22, A: 255}
	fire := color.RGBA{R: 220, G: 80, B: 30, A: 255}
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			img.SetRGBA(x, y, bg)
		}
	}
	for y := 180; y < 300; y++ {
		for x := 260; x < 380; x++ {
			img.SetRGBA(x, y, fire)
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes()
}

func getEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes"
}

func startMetricsServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", observability.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	log.Printf("Metrics server listening on %s", addr)
	return server
}

func shutdownMetricsServer(server *http.Server) {
	if server == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Metrics server shutdown error: %v", err)
	}
}

func loadCameraConfig() (hal.CameraConfig, error) {
	backend := getEnvDefault("CAMERA_BACKEND", "mjpeg")
	address := os.Getenv("CAMERA_ADDRESS")
	devicePath := os.Getenv("CAMERA_DEVICE_PATH")
	port := parseEnvInt("CAMERA_PORT", 80)
	streamPath := getEnvDefault("CAMERA_STREAM_PATH", "/stream")
	frameRate := parseEnvInt("CAMERA_FRAME_RATE", 10)

	if backend == "mjpeg" || backend == "rtsp" || backend == "gige" {
		if address == "" {
			return hal.CameraConfig{}, fmt.Errorf("CAMERA_ADDRESS is required for backend %s", backend)
		}
	}
	if backend == "v4l2" || backend == "usb" {
		if devicePath == "" {
			return hal.CameraConfig{}, fmt.Errorf("CAMERA_DEVICE_PATH is required for backend %s", backend)
		}
	}

	return hal.CameraConfig{
		Backend:     backend,
		Address:     address,
		Port:        port,
		StreamPath:  streamPath,
		FrameRate:   frameRate,
		Username:    os.Getenv("CAMERA_USERNAME"),
		Password:    os.Getenv("CAMERA_PASSWORD"),
		DevicePath:  devicePath,
		Resolution:  getEnvDefault("CAMERA_RESOLUTION", "1920x1080"),
		Codec:       getEnvDefault("CAMERA_CODEC", "mjpeg"),
		SatelliteID: getEnvDefault("SATELLITE_ID", "sat001"),
	}, nil
}

func parseEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func buildStorage() (dtn.BundleStorage, func()) {
	backend := strings.ToLower(strings.TrimSpace(os.Getenv("DTN_STORAGE_BACKEND")))
	if backend == "" {
		backend = "memory"
	}
	switch backend {
	case "postgres":
		cfg, err := db.LoadConfig()
		if err != nil {
			log.Fatalf("Failed to load DB config: %v", err)
		}
		pgDB, err := db.NewPostgresDB(cfg)
		if err != nil {
			log.Fatalf("Failed to connect to Postgres: %v", err)
		}
		storage, err := dtn.NewPostgresBundleStorage(pgDB)
		if err != nil {
			log.Fatalf("Failed to initialize Postgres storage: %v", err)
		}
		log.Printf("Using Postgres-backed DTN storage")
		return storage, func() { _ = pgDB.Close() }
	default:
		log.Printf("Using in-memory DTN storage")
		return dtn.NewInMemoryStorage(5000), func() {}
	}
}

type frameSample struct {
	Timestamp time.Time
	Data      []byte
}

type frameBuffer struct {
	mu       sync.Mutex
	capacity int
	frames   []frameSample
}

func newFrameBuffer(capacity int) *frameBuffer {
	return &frameBuffer{
		capacity: capacity,
		frames:   make([]frameSample, 0, capacity),
	}
}

func (b *frameBuffer) Add(frame []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()

	copied := make([]byte, len(frame))
	copy(copied, frame)

	b.frames = append(b.frames, frameSample{
		Timestamp: time.Now().UTC(),
		Data:      copied,
	})

	if len(b.frames) > b.capacity {
		b.frames = b.frames[len(b.frames)-b.capacity:]
	}
}

func (b *frameBuffer) Snapshot() []frameSample {
	b.mu.Lock()
	defer b.mu.Unlock()

	clone := make([]frameSample, len(b.frames))
	for i, sample := range b.frames {
		dataCopy := make([]byte, len(sample.Data))
		copy(dataCopy, sample.Data)
		clone[i] = frameSample{
			Timestamp: sample.Timestamp,
			Data:      dataCopy,
		}
	}
	return clone
}

type clipFrame struct {
	Timestamp string `json:"timestamp"`
	Data      string `json:"data_base64"`
}

type clipPayload struct {
	Format string      `json:"format"`
	Frames []clipFrame `json:"frames"`
}

type alertPayload struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	Confidence float64 `json:"confidence"`
	Location   string  `json:"location"`
	Timestamp  string  `json:"timestamp"`
	VideoClip  string  `json:"video_clip_base64"`
}

type telemetryPayload struct {
	SatelliteID string  `json:"satellite_id"`
	Timestamp   string  `json:"timestamp"`
	Battery     float64 `json:"battery_percent"`
	Voltage     float64 `json:"battery_voltage"`
	SolarPower  float64 `json:"solar_power"`
	Eclipse     bool    `json:"in_eclipse"`
	Latitude    float64 `json:"lat"`
	Longitude   float64 `json:"lon"`
	Altitude    float64 `json:"alt"`
}

func registerSatNetNeighbor(node *dtn.Node, transport *dtn.TCPTransport, address string) {
	neighbor := &dtn.Neighbor{
		ID:           "satnet_gateway",
		EID:          "dtn://earth/nysus",
		LinkQuality:  0.9,
		IsActive:     true,
		Latency:      120 * time.Millisecond,
		Bandwidth:    3_000_000,
		ContactStart: time.Now().UTC(),
		ContactEnd:   time.Now().UTC().Add(4 * time.Hour),
	}
	node.RegisterNeighbor(neighbor)
	if err := transport.Connect(context.Background(), neighbor.ID, address); err != nil {
		log.Printf("Failed to connect to Sat_Net gateway at %s: %v", address, err)
	}
}
