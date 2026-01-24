package control

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RealHunoidController implements MotionController and HunoidController interfaces for real humanoid robot control.
// Supports multiple communication protocols: ROS2, CAN, EtherCAT, WebSocket, gRPC.
type RealHunoidController struct {
	mu            sync.RWMutex
	config        HunoidConfig
	conn          io.ReadWriteCloser
	httpClient    *http.Client
	currentPose   Pose
	targetPose    Pose
	joints        map[string]*Joint
	isMoving      bool
	isInitialized bool
	stopChan      chan struct{}
	telemetry     HunoidTelemetry
}

// HunoidConfig holds robot configuration
type HunoidConfig struct {
	// Communication protocol: "ros2", "can", "ethercat", "websocket", "http", "grpc"
	Protocol string `json:"protocol"`

	// Connection settings
	Address    string `json:"address"`    // IP address or ROS2 namespace
	Port       int    `json:"port"`
	ROS2Topic  string `json:"ros2Topic"`  // For ROS2 protocol
	CANBusID   int    `json:"canBusId"`   // For CAN protocol

	// Robot configuration
	RobotID      string   `json:"robotId"`
	Model        string   `json:"model"`        // e.g., "hunoid-v1", "atlas", "spot"
	JointNames   []string `json:"jointNames"`
	JointLimits  map[string][2]float64 `json:"jointLimits"` // [min, max] radians

	// Motion settings
	MaxLinearVelocity  float64 `json:"maxLinearVelocity"`  // m/s
	MaxAngularVelocity float64 `json:"maxAngularVelocity"` // rad/s
	MaxAcceleration    float64 `json:"maxAcceleration"`    // m/s^2

	// Safety settings
	CollisionAvoidance bool    `json:"collisionAvoidance"`
	EmergencyStopPin   int     `json:"emergencyStopPin"`
	SafetyZone         float64 `json:"safetyZone"` // meters

	// Update rates
	ControlLoopRate  int `json:"controlLoopRate"`  // Hz
	TelemetryRate    int `json:"telemetryRate"`    // Hz
}

// HunoidTelemetry contains real-time robot telemetry
type HunoidTelemetry struct {
	Position       Pose               `json:"position"`
	JointStates    []Joint            `json:"jointStates"`
	BatteryPercent float64            `json:"batteryPercent"`
	BatteryVoltage float64            `json:"batteryVoltage"`
	Temperature    float64            `json:"temperature"`
	Status         string             `json:"status"`
	Errors         []string           `json:"errors"`
	IMUData        IMUReading         `json:"imuData"`
	FootForces     [2]float64         `json:"footForces"` // Left, Right
	Timestamp      time.Time          `json:"timestamp"`
}

// IMUReading represents IMU sensor data
type IMUReading struct {
	AccelX float64 `json:"accelX"`
	AccelY float64 `json:"accelY"`
	AccelZ float64 `json:"accelZ"`
	GyroX  float64 `json:"gyroX"`
	GyroY  float64 `json:"gyroY"`
	GyroZ  float64 `json:"gyroZ"`
}

// NewHunoidController creates a new humanoid robot controller
func NewHunoidController(config HunoidConfig) *RealHunoidController {
	if config.ControlLoopRate == 0 {
		config.ControlLoopRate = 100 // 100 Hz default
	}
	if config.TelemetryRate == 0 {
		config.TelemetryRate = 10 // 10 Hz default
	}
	if config.MaxLinearVelocity == 0 {
		config.MaxLinearVelocity = 1.0 // 1 m/s default
	}
	if config.MaxAngularVelocity == 0 {
		config.MaxAngularVelocity = 1.0 // 1 rad/s default
	}

	// Initialize default joints if not provided
	if len(config.JointNames) == 0 {
		config.JointNames = defaultHunoidJoints()
	}

	joints := make(map[string]*Joint)
	for _, name := range config.JointNames {
		joints[name] = &Joint{
			ID:       name,
			Position: 0,
		}
	}

	return &RealHunoidController{
		config: config,
		joints: joints,
		currentPose: Pose{
			Position:    Vector3{X: 0, Y: 0, Z: 0},
			Orientation: Quaternion{W: 1, X: 0, Y: 0, Z: 0},
			Timestamp:   time.Now(),
		},
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func defaultHunoidJoints() []string {
	return []string{
		// Head
		"head_pan", "head_tilt",
		// Left arm
		"l_shoulder_pitch", "l_shoulder_roll", "l_shoulder_yaw",
		"l_elbow_pitch", "l_elbow_yaw",
		"l_wrist_pitch", "l_wrist_roll",
		// Right arm
		"r_shoulder_pitch", "r_shoulder_roll", "r_shoulder_yaw",
		"r_elbow_pitch", "r_elbow_yaw",
		"r_wrist_pitch", "r_wrist_roll",
		// Torso
		"torso_yaw",
		// Left leg
		"l_hip_yaw", "l_hip_roll", "l_hip_pitch",
		"l_knee_pitch",
		"l_ankle_pitch", "l_ankle_roll",
		// Right leg
		"r_hip_yaw", "r_hip_roll", "r_hip_pitch",
		"r_knee_pitch",
		"r_ankle_pitch", "r_ankle_roll",
	}
}

// Initialize establishes connection to the robot
func (h *RealHunoidController) Initialize(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	var err error
	switch h.config.Protocol {
	case "http", "websocket":
		err = h.initHTTP(ctx)
	case "ros2":
		err = h.initROS2(ctx)
	case "can":
		err = h.initCAN(ctx)
	case "ethercat":
		err = h.initEtherCAT(ctx)
	default:
		return fmt.Errorf("unsupported protocol: %s", h.config.Protocol)
	}

	if err != nil {
		return err
	}

	// Start telemetry polling
	h.stopChan = make(chan struct{})
	go h.telemetryLoop(ctx)

	// Enable motors
	if err := h.enableMotors(); err != nil {
		return fmt.Errorf("failed to enable motors: %w", err)
	}

	h.isInitialized = true
	return nil
}

func (h *RealHunoidController) initHTTP(ctx context.Context) error {
	url := fmt.Sprintf("http://%s:%d/api/status", h.config.Address, h.config.Port)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to robot: %w", err)
	}
	resp.Body.Close()

	return nil
}

func (h *RealHunoidController) initROS2(ctx context.Context) error {
	// ROS2 initialization via ros2-go or DDS directly
	// Real implementation would use rclgo or similar
	
	// For now, attempt TCP connection to ROS2 bridge
	addr := net.JoinHostPort(h.config.Address, strconv.Itoa(h.config.Port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to ROS2 bridge: %w", err)
	}

	h.conn = conn
	return nil
}

func (h *RealHunoidController) initCAN(ctx context.Context) error {
	// CAN bus initialization via SocketCAN
	addr := fmt.Sprintf("can%d", h.config.CANBusID)
	
	conn, err := net.Dial("unixgram", "/tmp/can_socket_"+addr)
	if err != nil {
		// Fallback to TCP CAN gateway
		tcpAddr := net.JoinHostPort(h.config.Address, strconv.Itoa(h.config.Port))
		conn, err = net.DialTimeout("tcp", tcpAddr, 5*time.Second)
		if err != nil {
			return fmt.Errorf("failed to connect to CAN bus: %w", err)
		}
	}

	h.conn = conn
	return nil
}

func (h *RealHunoidController) initEtherCAT(ctx context.Context) error {
	// EtherCAT initialization
	// Real implementation would use SOEM or IgH EtherCAT Master
	
	addr := net.JoinHostPort(h.config.Address, strconv.Itoa(h.config.Port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to EtherCAT master: %w", err)
	}

	h.conn = conn
	return nil
}

func (h *RealHunoidController) enableMotors() error {
	switch h.config.Protocol {
	case "http":
		return h.httpCommand("POST", "/api/motors/enable", nil)
	case "ros2", "can", "ethercat":
		if h.conn != nil {
			cmd := []byte{0xAA, 0x01, 0x01, 0x00} // Enable motors command
			_, err := h.conn.Write(cmd)
			return err
		}
	}
	return nil
}

func (h *RealHunoidController) httpCommand(method, path string, body interface{}) error {
	url := fmt.Sprintf("http://%s:%d%s", h.config.Address, h.config.Port, path)
	
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("command failed with status %d", resp.StatusCode)
	}

	return nil
}

func (h *RealHunoidController) telemetryLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second / time.Duration(h.config.TelemetryRate))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.updateTelemetry()
		case <-ctx.Done():
			return
		case <-h.stopChan:
			return
		}
	}
}

func (h *RealHunoidController) updateTelemetry() {
	h.mu.Lock()
	defer h.mu.Unlock()

	switch h.config.Protocol {
	case "http":
		h.updateTelemetryHTTP()
	case "ros2", "can", "ethercat":
		h.updateTelemetryDirect()
	}
}

func (h *RealHunoidController) updateTelemetryHTTP() {
	url := fmt.Sprintf("http://%s:%d/api/telemetry", h.config.Address, h.config.Port)
	
	resp, err := h.httpClient.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var telem HunoidTelemetry
	if err := json.NewDecoder(resp.Body).Decode(&telem); err != nil {
		return
	}

	h.telemetry = telem
	h.currentPose = telem.Position
	
	for _, js := range telem.JointStates {
		if joint, exists := h.joints[js.ID]; exists {
			*joint = js
		}
	}
}

func (h *RealHunoidController) updateTelemetryDirect() {
	if h.conn == nil {
		return
	}

	// Send telemetry request
	cmd := []byte{0xAA, 0x02, 0x00, 0x00} // Request telemetry
	if _, err := h.conn.Write(cmd); err != nil {
		return
	}

	// Read telemetry response
	buf := make([]byte, 512)
	n, err := h.conn.Read(buf)
	if err != nil || n < 48 {
		return
	}

	// Parse telemetry frame
	h.telemetry.BatteryPercent = float64(binary.BigEndian.Uint16(buf[0:2])) / 100.0
	h.telemetry.BatteryVoltage = float64(binary.BigEndian.Uint16(buf[2:4])) / 100.0
	h.telemetry.Temperature = float64(int16(binary.BigEndian.Uint16(buf[4:6]))) / 10.0
	
	// Parse position
	h.currentPose.Position.X = math.Float64frombits(binary.BigEndian.Uint64(buf[8:16]))
	h.currentPose.Position.Y = math.Float64frombits(binary.BigEndian.Uint64(buf[16:24]))
	h.currentPose.Position.Z = math.Float64frombits(binary.BigEndian.Uint64(buf[24:32]))
	
	h.telemetry.Position = h.currentPose
	h.telemetry.Timestamp = time.Now()
}

// GetCurrentPose returns the current robot pose
func (h *RealHunoidController) GetCurrentPose() (Pose, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.isInitialized {
		return Pose{}, fmt.Errorf("controller not initialized")
	}

	return h.currentPose, nil
}

// MoveTo commands the robot to move to a target pose
func (h *RealHunoidController) MoveTo(ctx context.Context, target Pose) error {
	h.mu.Lock()
	h.targetPose = target
	h.isMoving = true
	h.mu.Unlock()

	// Send move command
	switch h.config.Protocol {
	case "http":
		return h.moveToHTTP(ctx, target)
	case "ros2", "can", "ethercat":
		return h.moveToDirect(ctx, target)
	}

	return fmt.Errorf("unsupported protocol")
}

func (h *RealHunoidController) moveToHTTP(ctx context.Context, target Pose) error {
	payload := map[string]interface{}{
		"target": map[string]interface{}{
			"position": map[string]float64{
				"x": target.Position.X,
				"y": target.Position.Y,
				"z": target.Position.Z,
			},
			"orientation": map[string]float64{
				"w": target.Orientation.W,
				"x": target.Orientation.X,
				"y": target.Orientation.Y,
				"z": target.Orientation.Z,
			},
		},
		"velocity": h.config.MaxLinearVelocity,
	}

	return h.httpCommand("POST", "/api/motion/moveto", payload)
}

func (h *RealHunoidController) moveToDirect(ctx context.Context, target Pose) error {
	if h.conn == nil {
		return fmt.Errorf("not connected")
	}

	// Build move command
	buf := make([]byte, 64)
	buf[0] = 0xAA // Magic
	buf[1] = 0x03 // Move command
	
	binary.BigEndian.PutUint64(buf[8:16], math.Float64bits(target.Position.X))
	binary.BigEndian.PutUint64(buf[16:24], math.Float64bits(target.Position.Y))
	binary.BigEndian.PutUint64(buf[24:32], math.Float64bits(target.Position.Z))
	binary.BigEndian.PutUint64(buf[32:40], math.Float64bits(target.Orientation.W))
	binary.BigEndian.PutUint64(buf[40:48], math.Float64bits(target.Orientation.X))
	binary.BigEndian.PutUint64(buf[48:56], math.Float64bits(target.Orientation.Y))
	binary.BigEndian.PutUint64(buf[56:64], math.Float64bits(target.Orientation.Z))

	_, err := h.conn.Write(buf)
	return err
}

// Stop halts all robot motion
func (h *RealHunoidController) Stop() error {
	h.mu.Lock()
	h.isMoving = false
	h.mu.Unlock()

	switch h.config.Protocol {
	case "http":
		return h.httpCommand("POST", "/api/motion/stop", nil)
	case "ros2", "can", "ethercat":
		if h.conn != nil {
			cmd := []byte{0xAA, 0x04, 0x00, 0x00} // Stop command
			_, err := h.conn.Write(cmd)
			return err
		}
	}

	return nil
}

// GetJointStates returns current joint states
func (h *RealHunoidController) GetJointStates() ([]Joint, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	joints := make([]Joint, 0, len(h.joints))
	for _, joint := range h.joints {
		joints = append(joints, *joint)
	}

	return joints, nil
}

// SetJointPositions sets target positions for multiple joints
func (h *RealHunoidController) SetJointPositions(positions map[string]float64) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Validate joint limits
	for jointID, position := range positions {
		if limits, ok := h.config.JointLimits[jointID]; ok {
			if position < limits[0] || position > limits[1] {
				return fmt.Errorf("joint %s position %f out of limits [%f, %f]",
					jointID, position, limits[0], limits[1])
			}
		}
	}

	switch h.config.Protocol {
	case "http":
		return h.httpCommand("POST", "/api/joints/positions", positions)
	case "ros2", "can", "ethercat":
		return h.setJointPositionsDirect(positions)
	}

	return nil
}

func (h *RealHunoidController) setJointPositionsDirect(positions map[string]float64) error {
	if h.conn == nil {
		return fmt.Errorf("not connected")
	}

	// Build joint command packet
	buf := make([]byte, 4+len(positions)*12)
	buf[0] = 0xAA
	buf[1] = 0x05 // Joint command
	buf[2] = byte(len(positions))

	offset := 4
	for jointID, position := range positions {
		// Find joint index
		idx := 0
		for i, name := range h.config.JointNames {
			if name == jointID {
				idx = i
				break
			}
		}
		
		buf[offset] = byte(idx)
		offset++
		binary.BigEndian.PutUint64(buf[offset:offset+8], math.Float64bits(position))
		offset += 8
	}

	_, err := h.conn.Write(buf[:offset])
	return err
}

// IsMoving returns whether the robot is currently in motion
func (h *RealHunoidController) IsMoving() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.isMoving
}

// GetBatteryPercent returns current battery level
func (h *RealHunoidController) GetBatteryPercent() float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.telemetry.BatteryPercent
}

// GetTelemetry returns full telemetry data
func (h *RealHunoidController) GetTelemetry() HunoidTelemetry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.telemetry
}

// ExecuteGait executes a predefined gait pattern
func (h *RealHunoidController) ExecuteGait(gaitType string, params map[string]interface{}) error {
	payload := map[string]interface{}{
		"gait":   gaitType,
		"params": params,
	}

	switch h.config.Protocol {
	case "http":
		return h.httpCommand("POST", "/api/motion/gait", payload)
	default:
		return fmt.Errorf("gait execution not supported on %s protocol", h.config.Protocol)
	}
}

// Shutdown disconnects from the robot
func (h *RealHunoidController) Shutdown() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Disable motors
	switch h.config.Protocol {
	case "http":
		h.httpCommand("POST", "/api/motors/disable", nil)
	case "ros2", "can", "ethercat":
		if h.conn != nil {
			cmd := []byte{0xAA, 0x01, 0x00, 0x00} // Disable motors
			h.conn.Write(cmd)
		}
	}

	if h.stopChan != nil {
		close(h.stopChan)
	}

	if h.conn != nil {
		h.conn.Close()
	}

	h.isInitialized = false
	return nil
}
