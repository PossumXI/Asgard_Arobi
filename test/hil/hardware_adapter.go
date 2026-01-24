package hil

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/asgard/pandora/internal/orbital/hal"
	"github.com/asgard/pandora/internal/robotics/control"
	"github.com/asgard/pandora/internal/robotics/ethics"
	"github.com/asgard/pandora/internal/robotics/vla"
)

// HardwareAdapter is the interface for switching between mock and real hardware
type HardwareAdapter interface {
	Initialize(ctx context.Context) error
	Shutdown() error
	GetMode() HardwareMode
	IsAvailable() bool
}

// =============================================================================
// Silenus Adapter - Wraps Silenus HAL interfaces (camera, power, GPS)
// =============================================================================

// SilenusAdapter wraps Silenus hardware components
type SilenusAdapter struct {
	mu       sync.RWMutex
	config   *HardwareConfig
	mode     HardwareMode
	
	// Hardware interfaces
	camera   hal.CameraController
	power    hal.PowerController
	gps      hal.GPSController
	
	// State
	initialized bool
}

// NewSilenusAdapter creates a new Silenus hardware adapter
func NewSilenusAdapter(config *HardwareConfig) *SilenusAdapter {
	return &SilenusAdapter{
		config: config,
		mode:   config.Mode,
	}
}

// Initialize sets up Silenus hardware based on configuration
func (a *SilenusAdapter) Initialize(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.initialized {
		return fmt.Errorf("adapter already initialized")
	}

	// Determine hardware mode
	if a.config.Mode == HardwareModeAuto {
		a.mode = a.detectHardwareMode()
		if a.mode != HardwareModeReal {
			return fmt.Errorf("Silenus hardware unavailable in auto mode")
		}
	}

	// Initialize camera
	camera, err := a.initializeCamera(ctx)
	if err != nil {
		return fmt.Errorf("camera initialization failed: %w", err)
	}
	a.camera = camera

	// Initialize power controller
	power, err := a.initializePower(ctx)
	if err != nil {
		// Cleanup camera
		if a.camera != nil {
			_ = a.camera.Shutdown()
		}
		return fmt.Errorf("power initialization failed: %w", err)
	}
	a.power = power

	// Initialize GPS
	gps, err := a.initializeGPS(ctx)
	if err != nil {
		// Cleanup
		if a.camera != nil {
			_ = a.camera.Shutdown()
		}
		return fmt.Errorf("gps initialization failed: %w", err)
	}
	a.gps = gps

	a.initialized = true
	return nil
}

// detectHardwareMode checks if real hardware is available
func (a *SilenusAdapter) detectHardwareMode() HardwareMode {
	if a.config.CameraDevice != "" || os.Getenv("HIL_CAMERA_ADDRESS") != "" {
		return HardwareModeReal
	}
	if a.config.PowerMonitorAddr != "" || os.Getenv("HIL_POWER_ENDPOINT") != "" {
		return HardwareModeReal
	}
	if os.Getenv("N2YO_API_KEY") != "" {
		return HardwareModeReal
	}
	return HardwareModeAuto
}

func (a *SilenusAdapter) initializeCamera(ctx context.Context) (hal.CameraController, error) {
	cameraConfig, err := loadHILCameraConfig(a.config)
	if err != nil {
		return nil, err
	}

	camera := hal.NewCamera(cameraConfig)
	if err := camera.Initialize(ctx); err != nil {
		return nil, err
	}
	return camera, nil
}

func (a *SilenusAdapter) initializePower(ctx context.Context) (hal.PowerController, error) {
	endpoint := strings.TrimSpace(a.config.PowerMonitorAddr)
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("HIL_POWER_ENDPOINT"))
	}
	if endpoint == "" {
		return nil, fmt.Errorf("power monitor endpoint is required")
	}

	return hal.NewRemotePowerController(endpoint)
}

func (a *SilenusAdapter) initializeGPS(ctx context.Context) (hal.GPSController, error) {
	cfg := hal.DefaultOrbitalConfig()
	if n2yoKey := os.Getenv("N2YO_API_KEY"); n2yoKey != "" {
		cfg.N2YOAPIKey = n2yoKey
	}
	if noradID := os.Getenv("HIL_NORAD_ID"); noradID != "" {
		if parsed, err := strconv.Atoi(noradID); err == nil {
			cfg.NoradID = parsed
		}
	}

	return hal.NewRealOrbitalPosition(cfg)
}

// Shutdown cleans up Silenus hardware
func (a *SilenusAdapter) Shutdown() error {
	return a.ShutdownWithContext(context.Background())
}

// ShutdownWithContext cleans up with a context
func (a *SilenusAdapter) ShutdownWithContext(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.initialized {
		return nil
	}

	var errs []error

	if a.camera != nil {
		if err := a.camera.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("camera shutdown: %w", err))
		}
		a.camera = nil
	}

	// Power and GPS don't have shutdown methods in the interface
	a.power = nil
	a.gps = nil

	a.initialized = false

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}
	return nil
}

// GetMode returns the current hardware mode
func (a *SilenusAdapter) GetMode() HardwareMode {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.mode
}

// IsAvailable returns whether Silenus hardware is available
func (a *SilenusAdapter) IsAvailable() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.initialized
}

// Camera returns the camera controller
func (a *SilenusAdapter) Camera() hal.CameraController {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.camera
}

// Power returns the power controller
func (a *SilenusAdapter) Power() hal.PowerController {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.power
}

// GPS returns the GPS controller
func (a *SilenusAdapter) GPS() hal.GPSController {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.gps
}

// =============================================================================
// Hunoid Adapter - Wraps Hunoid control interfaces (motion, manipulator)
// =============================================================================

// HunoidAdapter wraps Hunoid hardware components
type HunoidAdapter struct {
	mu       sync.RWMutex
	config   *HardwareConfig
	mode     HardwareMode
	
	// Hardware interfaces
	motion       control.MotionController
	manipulator  control.ManipulatorController
	ethics       *ethics.EthicalKernel
	vla          vla.VLAModel
	
	// State
	initialized bool
}

// NewHunoidAdapter creates a new Hunoid hardware adapter
func NewHunoidAdapter(config *HardwareConfig) *HunoidAdapter {
	return &HunoidAdapter{
		config: config,
		mode:   config.Mode,
	}
}

// Initialize sets up Hunoid hardware based on configuration
func (a *HunoidAdapter) Initialize(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.initialized {
		return fmt.Errorf("adapter already initialized")
	}

	// Determine hardware mode
	if a.config.Mode == HardwareModeAuto {
		a.mode = a.detectHardwareMode()
		if a.mode != HardwareModeReal {
			return fmt.Errorf("Hunoid hardware unavailable in auto mode")
		}
	}

	// Initialize motion controller
	motion, err := a.initializeMotion(ctx)
	if err != nil {
		return fmt.Errorf("motion initialization failed: %w", err)
	}
	a.motion = motion

	// Initialize manipulator
	manipulator, err := a.initializeManipulator(ctx)
	if err != nil {
		// Cleanup motion
		if a.motion != nil {
			_ = a.motion.Stop()
		}
		return fmt.Errorf("manipulator initialization failed: %w", err)
	}
	a.manipulator = manipulator

	// Initialize ethics kernel (always mock for safety)
	a.ethics = ethics.NewEthicalKernel()

	// Initialize VLA
	vla, err := a.initializeVLA(ctx)
	if err != nil {
		return fmt.Errorf("VLA initialization failed: %w", err)
	}
	a.vla = vla

	a.initialized = true
	return nil
}

// detectHardwareMode checks if real Hunoid hardware is available
func (a *HunoidAdapter) detectHardwareMode() HardwareMode {
	if a.config.HunoidControlAddr != "" || os.Getenv("HIL_HUNOID_ENDPOINT") != "" {
		return HardwareModeReal
	}
	return HardwareModeAuto
}

func (a *HunoidAdapter) initializeMotion(ctx context.Context) (control.MotionController, error) {
	endpoint := strings.TrimSpace(a.config.HunoidControlAddr)
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("HIL_HUNOID_ENDPOINT"))
	}
	if endpoint == "" {
		return nil, fmt.Errorf("Hunoid control endpoint is required")
	}

	hunoid, err := control.NewRemoteHunoid(a.config.HunoidID, endpoint)
	if err != nil {
		return nil, err
	}
	if err := hunoid.Initialize(ctx); err != nil {
		return nil, err
	}
	return hunoid, nil
}

func (a *HunoidAdapter) initializeManipulator(ctx context.Context) (control.ManipulatorController, error) {
	endpoint := strings.TrimSpace(a.config.ManipulatorAddr)
	if endpoint == "" {
		endpoint = strings.TrimSpace(a.config.HunoidControlAddr)
	}
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("HIL_HUNOID_ENDPOINT"))
	}
	if endpoint == "" {
		return nil, fmt.Errorf("Manipulator endpoint is required")
	}

	return control.NewRemoteManipulator(a.config.HunoidID, endpoint)
}

func (a *HunoidAdapter) initializeVLA(ctx context.Context) (vla.VLAModel, error) {
	endpoint := strings.TrimSpace(os.Getenv("HIL_VLA_ENDPOINT"))
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("VLA_ENDPOINT"))
	}
	if endpoint == "" {
		return nil, fmt.Errorf("VLA endpoint is required")
	}

	modelPath := os.Getenv("HIL_VLA_MODEL")
	if modelPath == "" {
		modelPath = "models/openvla.onnx"
	}

	client, err := vla.NewHTTPVLA(endpoint)
	if err != nil {
		return nil, err
	}
	if err := client.Initialize(ctx, modelPath); err != nil {
		return nil, err
	}
	return client, nil
}

// Shutdown cleans up Hunoid hardware
func (a *HunoidAdapter) Shutdown() error {
	return a.ShutdownWithContext(context.Background())
}

// ShutdownWithContext cleans up with a context
func (a *HunoidAdapter) ShutdownWithContext(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.initialized {
		return nil
	}

	var errs []error

	// Stop motion controller
	if a.motion != nil {
		if err := a.motion.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("motion stop: %w", err))
		}
		a.motion = nil
	}

	// Manipulator doesn't have explicit shutdown
	a.manipulator = nil
	a.ethics = nil

	// Shutdown VLA
	if a.vla != nil {
		if err := a.vla.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("VLA shutdown: %w", err))
		}
		a.vla = nil
	}

	a.initialized = false

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}
	return nil
}

// GetMode returns the current hardware mode
func (a *HunoidAdapter) GetMode() HardwareMode {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.mode
}

// IsAvailable returns whether Hunoid hardware is available
func (a *HunoidAdapter) IsAvailable() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.initialized
}

// Motion returns the motion controller
func (a *HunoidAdapter) Motion() control.MotionController {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.motion
}

// Manipulator returns the manipulator controller
func (a *HunoidAdapter) Manipulator() control.ManipulatorController {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.manipulator
}

// Ethics returns the ethics kernel
func (a *HunoidAdapter) Ethics() *ethics.EthicalKernel {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.ethics
}

// VLA returns the VLA model
func (a *HunoidAdapter) VLA() vla.VLAModel {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.vla
}

func loadHILCameraConfig(config *HardwareConfig) (hal.CameraConfig, error) {
	backend := strings.TrimSpace(os.Getenv("HIL_CAMERA_BACKEND"))
	if backend == "" {
		backend = "mjpeg"
	}

	address := strings.TrimSpace(os.Getenv("HIL_CAMERA_ADDRESS"))
	port := parseEnvInt("HIL_CAMERA_PORT", 80)
	streamPath := getEnvDefault("HIL_CAMERA_STREAM_PATH", "/stream")
	frameRate := parseEnvInt("HIL_CAMERA_FRAME_RATE", 10)

	if address == "" {
		return hal.CameraConfig{}, fmt.Errorf("HIL_CAMERA_ADDRESS is required for backend %s", backend)
	}

	return hal.CameraConfig{
		Backend:    backend,
		Address:    address,
		Port:       port,
		StreamPath: streamPath,
		FrameRate:  frameRate,
		Username:   os.Getenv("HIL_CAMERA_USERNAME"),
		Password:   os.Getenv("HIL_CAMERA_PASSWORD"),
		Resolution: getEnvDefault("HIL_CAMERA_RESOLUTION", "1920x1080"),
		Codec:      getEnvDefault("HIL_CAMERA_CODEC", "mjpeg"),
	}, nil
}

func parseEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
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
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}


