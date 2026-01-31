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

// RealManipulator implements ManipulatorController for real robot arm/gripper control.
// Supports various robot arms: UR, Kinova, Franka, custom grippers.
type RealManipulator struct {
	mu            sync.RWMutex
	config        ManipulatorConfig
	conn          io.ReadWriteCloser
	httpClient    *http.Client
	gripperState  float64
	armPosition   Vector3
	jointStates   []float64
	isInitialized bool
	stopChan      chan struct{}
}

// ManipulatorConfig holds manipulator configuration
type ManipulatorConfig struct {
	// Communication protocol: "ur", "ros2", "modbus", "http", "grpc"
	Protocol string `json:"protocol"`

	// Connection settings
	Address string `json:"address"`
	Port    int    `json:"port"`

	// Arm configuration
	ArmModel    string  `json:"armModel"` // e.g., "ur5e", "kinova-gen3", "franka"
	NumJoints   int     `json:"numJoints"`
	ReachRadius float64 `json:"reachRadius"` // meters
	PayloadMax  float64 `json:"payloadMax"`  // kg

	// Gripper configuration
	GripperModel    string  `json:"gripperModel"`    // e.g., "robotiq-2f85", "wsg50"
	GripperMaxWidth float64 `json:"gripperMaxWidth"` // meters
	GripperForce    float64 `json:"gripperForce"`    // Newtons

	// Motion settings
	MaxJointVelocity float64 `json:"maxJointVelocity"` // rad/s
	MaxLinearSpeed   float64 `json:"maxLinearSpeed"`   // m/s
	Acceleration     float64 `json:"acceleration"`     // m/s^2

	// Safety settings
	ForceLimit  float64 `json:"forceLimit"`  // Newtons
	TorqueLimit float64 `json:"torqueLimit"` // Nm
}

// GripperState represents the current gripper state
type GripperState struct {
	Position float64 `json:"position"` // 0.0 (closed) to 1.0 (open)
	Force    float64 `json:"force"`    // Current force in Newtons
	Gripping bool    `json:"gripping"` // Object detected
}

// NewRealManipulator creates a new manipulator controller
func NewRealManipulator(config ManipulatorConfig) *RealManipulator {
	if config.NumJoints == 0 {
		config.NumJoints = 6
	}
	if config.ReachRadius == 0 {
		config.ReachRadius = 0.85 // Default UR5 reach
	}
	if config.GripperMaxWidth == 0 {
		config.GripperMaxWidth = 0.085 // Robotiq 2F-85
	}
	if config.MaxLinearSpeed == 0 {
		config.MaxLinearSpeed = 0.5 // 0.5 m/s default
	}
	if config.MaxJointVelocity == 0 {
		config.MaxJointVelocity = 1.0 // 1 rad/s default
	}

	return &RealManipulator{
		config:       config,
		gripperState: 1.0, // Start open
		armPosition:  Vector3{X: 0.3, Y: 0, Z: 0.5},
		jointStates:  make([]float64, config.NumJoints),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Initialize establishes connection to the manipulator
func (m *RealManipulator) Initialize(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var err error
	switch m.config.Protocol {
	case "ur":
		err = m.initUR(ctx)
	case "ros2":
		err = m.initROS2(ctx)
	case "modbus":
		err = m.initModbus(ctx)
	case "http":
		err = m.initHTTP(ctx)
	default:
		return fmt.Errorf("unsupported protocol: %s", m.config.Protocol)
	}

	if err != nil {
		return err
	}

	// Start state polling
	m.stopChan = make(chan struct{})
	go m.statePollingLoop(ctx)

	m.isInitialized = true
	return nil
}

func (m *RealManipulator) initUR(ctx context.Context) error {
	// Universal Robots use multiple ports:
	// 30001: Primary interface (10 Hz)
	// 30002: Secondary interface (10 Hz)
	// 30003: Real-time interface (125 Hz)
	// 30004: RTDE (Real-Time Data Exchange)

	addr := net.JoinHostPort(m.config.Address, strconv.Itoa(30003)) // Real-time interface

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to UR robot: %w", err)
	}

	m.conn = conn
	return nil
}

func (m *RealManipulator) initROS2(ctx context.Context) error {
	// Connect to ROS2 bridge
	addr := net.JoinHostPort(m.config.Address, strconv.Itoa(m.config.Port))

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to ROS2 bridge: %w", err)
	}

	m.conn = conn
	return nil
}

func (m *RealManipulator) initModbus(ctx context.Context) error {
	// Modbus TCP for industrial grippers
	port := m.config.Port
	if port == 0 {
		port = 502 // Default Modbus port
	}
	addr := net.JoinHostPort(m.config.Address, strconv.Itoa(port))

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect via Modbus: %w", err)
	}

	m.conn = conn
	return nil
}

func (m *RealManipulator) initHTTP(ctx context.Context) error {
	url := fmt.Sprintf("http://%s:%d/api/status", m.config.Address, m.config.Port)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to manipulator: %w", err)
	}
	resp.Body.Close()

	return nil
}

func (m *RealManipulator) statePollingLoop(ctx context.Context) {
	ticker := time.NewTicker(50 * time.Millisecond) // 20 Hz
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.updateState()
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		}
	}
}

func (m *RealManipulator) updateState() {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch m.config.Protocol {
	case "ur":
		m.updateStateUR()
	case "ros2":
		m.updateStateROS2()
	case "modbus":
		m.updateStateModbus()
	case "http":
		m.updateStateHTTP()
	}
}

func (m *RealManipulator) updateStateUR() {
	if m.conn == nil {
		return
	}

	// Read UR real-time state packet (1108 bytes for UR5e)
	buf := make([]byte, 1108)
	n, err := m.conn.Read(buf)
	if err != nil || n < 200 {
		return
	}

	// Parse packet
	// Offset 252: Actual joint positions (6 * 8 bytes)
	for i := 0; i < m.config.NumJoints && i < 6; i++ {
		offset := 252 + i*8
		m.jointStates[i] = math.Float64frombits(binary.BigEndian.Uint64(buf[offset : offset+8]))
	}

	// Forward kinematics to get TCP position
	m.armPosition = m.calculateForwardKinematics()
}

func (m *RealManipulator) updateStateROS2() {
	if m.conn == nil {
		return
	}

	// Request state
	cmd := []byte{0xAA, 0x01, 0x00, 0x00}
	if _, err := m.conn.Write(cmd); err != nil {
		return
	}

	buf := make([]byte, 256)
	n, err := m.conn.Read(buf)
	if err != nil || n < 48 {
		return
	}

	// Parse response
	for i := 0; i < m.config.NumJoints && i*8+8 <= n; i++ {
		m.jointStates[i] = math.Float64frombits(binary.BigEndian.Uint64(buf[i*8 : i*8+8]))
	}
}

func (m *RealManipulator) updateStateModbus() {
	if m.conn == nil {
		return
	}

	// Modbus read holding registers
	request := []byte{
		0x00, 0x01, // Transaction ID
		0x00, 0x00, // Protocol ID
		0x00, 0x06, // Length
		0x01,       // Unit ID
		0x03,       // Function code (read holding registers)
		0x00, 0x00, // Start address
		0x00, 0x10, // Number of registers
	}

	if _, err := m.conn.Write(request); err != nil {
		return
	}

	buf := make([]byte, 256)
	n, err := m.conn.Read(buf)
	if err != nil || n < 9 {
		return
	}

	// Parse gripper position
	if n >= 11 {
		m.gripperState = float64(binary.BigEndian.Uint16(buf[9:11])) / 255.0
	}
}

func (m *RealManipulator) updateStateHTTP() {
	url := fmt.Sprintf("http://%s:%d/api/state", m.config.Address, m.config.Port)

	resp, err := m.httpClient.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var state struct {
		JointStates  []float64 `json:"jointStates"`
		GripperState float64   `json:"gripperState"`
		Position     Vector3   `json:"position"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
		return
	}

	m.jointStates = state.JointStates
	m.gripperState = state.GripperState
	m.armPosition = state.Position
}

func (m *RealManipulator) calculateForwardKinematics() Vector3 {
	// Simplified forward kinematics for 6-DOF arm
	// Real implementation would use DH parameters and proper FK

	// UR5 DH parameters (simplified)
	d := []float64{0.089, 0, 0, 0.109, 0.094, 0.082}
	a := []float64{0, -0.425, -0.392, 0, 0, 0}

	x, y, z := 0.0, 0.0, 0.0

	for i, q := range m.jointStates {
		if i >= len(d) {
			break
		}
		x += a[i] * math.Cos(q)
		y += a[i] * math.Sin(q)
		z += d[i]
	}

	return Vector3{X: x, Y: y, Z: z}
}

// OpenGripper opens the gripper
func (m *RealManipulator) OpenGripper() error {
	m.mu.Lock()
	protocol := m.config.Protocol
	m.mu.Unlock()

	var err error
	switch protocol {
	case "ur":
		err = m.gripperCommandUR(1.0, 0.5) // Position 1.0 (open), speed 0.5
	case "modbus":
		err = m.gripperCommandModbus(255) // 255 = fully open
	case "http":
		err = m.gripperCommandHTTP(1.0)
	case "ros2":
		err = m.gripperCommandROS2(1.0)
	default:
		return fmt.Errorf("gripper control not supported on %s", protocol)
	}

	if err == nil {
		m.mu.Lock()
		m.gripperState = 1.0
		m.mu.Unlock()
	}

	return err
}

// CloseGripper closes the gripper
func (m *RealManipulator) CloseGripper() error {
	m.mu.Lock()
	protocol := m.config.Protocol
	m.mu.Unlock()

	var err error
	switch protocol {
	case "ur":
		err = m.gripperCommandUR(0.0, 0.5)
	case "modbus":
		err = m.gripperCommandModbus(0)
	case "http":
		err = m.gripperCommandHTTP(0.0)
	case "ros2":
		err = m.gripperCommandROS2(0.0)
	default:
		return fmt.Errorf("gripper control not supported on %s", protocol)
	}

	if err == nil {
		m.mu.Lock()
		m.gripperState = 0.0
		m.mu.Unlock()
	}

	return err
}

func (m *RealManipulator) gripperCommandUR(position, speed float64) error {
	if m.conn == nil {
		return fmt.Errorf("not connected")
	}

	// URScript command for Robotiq gripper
	script := fmt.Sprintf("rq_move_and_wait_for_pos(%d, %d, %d)\n",
		int(position*255), int(speed*255), 100) // position, speed, force

	_, err := m.conn.Write([]byte(script))
	return err
}

func (m *RealManipulator) gripperCommandModbus(position byte) error {
	if m.conn == nil {
		return fmt.Errorf("not connected")
	}

	// Modbus write single register
	request := []byte{
		0x00, 0x02, // Transaction ID
		0x00, 0x00, // Protocol ID
		0x00, 0x06, // Length
		0x01,       // Unit ID
		0x06,       // Function code (write single register)
		0x00, 0x00, // Register address
		0x00, position, // Value
	}

	_, err := m.conn.Write(request)
	return err
}

func (m *RealManipulator) gripperCommandHTTP(position float64) error {
	url := fmt.Sprintf("http://%s:%d/api/gripper/position", m.config.Address, m.config.Port)

	payload := map[string]float64{"position": position}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

func (m *RealManipulator) gripperCommandROS2(position float64) error {
	if m.conn == nil {
		return fmt.Errorf("not connected")
	}

	cmd := make([]byte, 16)
	cmd[0] = 0xAA
	cmd[1] = 0x02 // Gripper command
	binary.BigEndian.PutUint64(cmd[8:16], math.Float64bits(position))

	_, err := m.conn.Write(cmd)
	return err
}

// GetGripperState returns current gripper state (0.0 = closed, 1.0 = open)
func (m *RealManipulator) GetGripperState() (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.isInitialized {
		return 0, fmt.Errorf("controller not initialized")
	}

	return m.gripperState, nil
}

// ReachTo moves the arm end-effector to a target position
func (m *RealManipulator) ReachTo(ctx context.Context, position Vector3) error {
	m.mu.Lock()
	reachRadius := m.config.ReachRadius
	protocol := m.config.Protocol
	m.mu.Unlock()

	// Validate reachability
	distance := math.Sqrt(position.X*position.X + position.Y*position.Y + position.Z*position.Z)
	if distance > reachRadius {
		return fmt.Errorf("position out of reach: %.3fm (max %.3fm)", distance, reachRadius)
	}

	switch protocol {
	case "ur":
		return m.moveToUR(ctx, position)
	case "ros2":
		return m.moveToROS2(ctx, position)
	case "http":
		return m.moveToHTTP(ctx, position)
	default:
		return fmt.Errorf("move not supported on %s", protocol)
	}
}

func (m *RealManipulator) moveToUR(ctx context.Context, position Vector3) error {
	if m.conn == nil {
		return fmt.Errorf("not connected")
	}

	// URScript moveL command
	script := fmt.Sprintf("movel(p[%.4f, %.4f, %.4f, 0, 3.14, 0], a=%.2f, v=%.2f)\n",
		position.X, position.Y, position.Z,
		m.config.Acceleration, m.config.MaxLinearSpeed)

	_, err := m.conn.Write([]byte(script))
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.armPosition = position
	m.mu.Unlock()

	return nil
}

func (m *RealManipulator) moveToROS2(ctx context.Context, position Vector3) error {
	if m.conn == nil {
		return fmt.Errorf("not connected")
	}

	cmd := make([]byte, 32)
	cmd[0] = 0xAA
	cmd[1] = 0x03 // Move command
	binary.BigEndian.PutUint64(cmd[8:16], math.Float64bits(position.X))
	binary.BigEndian.PutUint64(cmd[16:24], math.Float64bits(position.Y))
	binary.BigEndian.PutUint64(cmd[24:32], math.Float64bits(position.Z))

	_, err := m.conn.Write(cmd)
	return err
}

func (m *RealManipulator) moveToHTTP(ctx context.Context, position Vector3) error {
	url := fmt.Sprintf("http://%s:%d/api/arm/moveto", m.config.Address, m.config.Port)

	payload := map[string]interface{}{
		"position": map[string]float64{
			"x": position.X,
			"y": position.Y,
			"z": position.Z,
		},
		"speed": m.config.MaxLinearSpeed,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

// GetJointStates returns current joint positions
func (m *RealManipulator) GetJointStates() []float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]float64, len(m.jointStates))
	copy(result, m.jointStates)
	return result
}

// GetPosition returns current end-effector position
func (m *RealManipulator) GetPosition() Vector3 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.armPosition
}

// Shutdown disconnects from the manipulator
func (m *RealManipulator) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.stopChan != nil {
		close(m.stopChan)
	}

	if m.conn != nil {
		m.conn.Close()
	}

	m.isInitialized = false
	return nil
}
