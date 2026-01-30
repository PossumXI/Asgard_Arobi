// Package actuators provides flight controller interface via MAVLink
package actuators

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// MAVLinkController interfaces with MAVLink flight controllers
type MAVLinkController struct {
	mu sync.RWMutex

	port     string
	baudRate int
	connected bool
	protocol *MAVLinkProtocol

	// Command channels
	attitudeCmd chan AttitudeCommand
	positionCmd chan PositionCommand
	velocityCmd chan VelocityCommand

	// Current state from telemetry
	currentAttitude  [3]float64
	currentPosition  [3]float64
	currentVelocity  [3]float64
	armed            bool
	flightMode       string

	// Configuration
	config MAVLinkConfig

	// Logger
	logger *logrus.Logger

	// Statistics
	cmdsSent      uint64
	telemetryRcvd uint64
	lastHeartbeat time.Time
}

// MAVLinkConfig holds MAVLink configuration
type MAVLinkConfig struct {
	Port          string
	BaudRate      int
	SystemID      uint8
	ComponentID   uint8
	HeartbeatHz   float64
	CommandHz     float64
	SimulationMode bool
}

// AttitudeCommand sets desired attitude
type AttitudeCommand struct {
	Roll      float64 // radians
	Pitch     float64 // radians
	Yaw       float64 // radians
	Throttle  float64 // 0.0 to 1.0
	Timestamp time.Time
}

// PositionCommand sets desired position
type PositionCommand struct {
	X, Y, Z   float64 // meters (NED or ENU depending on config)
	Timestamp time.Time
}

// VelocityCommand sets desired velocity
type VelocityCommand struct {
	Vx, Vy, Vz float64 // m/s
	Timestamp  time.Time
}

// ControlSurfaceCommand sets control surface positions
type ControlSurfaceCommand struct {
	Aileron  float64 // -1.0 to 1.0
	Elevator float64 // -1.0 to 1.0
	Rudder   float64 // -1.0 to 1.0
	Flaps    float64 // 0.0 to 1.0
	Throttle float64 // 0.0 to 1.0
}

// FlightControlCommand combines all control inputs
type FlightControlCommand struct {
	Type      CommandType
	Attitude  *AttitudeCommand
	Position  *PositionCommand
	Velocity  *VelocityCommand
	Surfaces  *ControlSurfaceCommand
	Timestamp time.Time
}

// CommandType defines the type of control command
type CommandType int

const (
	CommandTypeAttitude CommandType = iota
	CommandTypePosition
	CommandTypeVelocity
	CommandTypeSurfaces
	CommandTypeRaw
)

// Telemetry represents data received from the flight controller
type Telemetry struct {
	Timestamp time.Time

	// Attitude
	Roll  float64
	Pitch float64
	Yaw   float64

	// Position (NED)
	PosX float64
	PosY float64
	PosZ float64

	// Velocity
	VelX float64
	VelY float64
	VelZ float64

	// Angular rates
	RollRate  float64
	PitchRate float64
	YawRate   float64

	// System status
	Armed      bool
	FlightMode string
	BatteryV   float64
	BatteryPct float64
	GPSFix     int
	GPSSats    int

	// Air data
	Airspeed    float64
	GroundSpeed float64
	AltitudeMSL float64
	AltitudeAGL float64
	Heading     float64
}

var (
	ErrNotConnected     = fmt.Errorf("not connected to flight controller")
	ErrConnectionFailed = fmt.Errorf("failed to connect to flight controller")
	ErrTimeout          = fmt.Errorf("command timeout")
)

// NewMAVLinkController creates a new controller
func NewMAVLinkController(config MAVLinkConfig) *MAVLinkController {
	if config.HeartbeatHz == 0 {
		config.HeartbeatHz = 1.0
	}
	if config.CommandHz == 0 {
		config.CommandHz = 50.0
	}
	if config.SystemID == 0 {
		config.SystemID = 1
	}
	if config.ComponentID == 0 {
		config.ComponentID = 1
	}

	return &MAVLinkController{
		port:        config.Port,
		baudRate:    config.BaudRate,
		attitudeCmd: make(chan AttitudeCommand, 10),
		positionCmd: make(chan PositionCommand, 10),
		velocityCmd: make(chan VelocityCommand, 10),
		config:      config,
		logger:      logrus.New(),
		flightMode:  "UNKNOWN",
	}
}

// Connect establishes connection to flight controller
func (mc *MAVLinkController) Connect(ctx context.Context) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.connected {
		return nil
	}

	mc.logger.WithFields(logrus.Fields{
		"port":     mc.port,
		"baudRate": mc.baudRate,
	}).Info("Connecting to flight controller...")

	if mc.config.SimulationMode {
		// Simulation mode - no actual hardware
		mc.connected = true
		mc.lastHeartbeat = time.Now()
		mc.logger.Info("Connected in simulation mode")
		return nil
	}

	// Open serial port with MAVLink protocol
	protocol := NewMAVLinkProtocol(mc.config.SystemID, mc.config.ComponentID)
	if err := protocol.OpenSerialPort(mc.port, mc.baudRate); err != nil {
		return fmt.Errorf("failed to open serial port: %w", err)
	}

	mc.mu.Lock()
	mc.protocol = protocol
	mc.connected = true
	mc.lastHeartbeat = time.Now()
	mc.mu.Unlock()

	mc.logger.Info("Connected to flight controller")
	return nil
}

// Disconnect closes the connection
func (mc *MAVLinkController) Disconnect() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.connected {
		return nil
	}

	mc.connected = false
	mc.logger.Info("Disconnected from flight controller")
	return nil
}

// IsConnected returns connection status
func (mc *MAVLinkController) IsConnected() bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.connected
}

// SendAttitudeCommand sends attitude setpoint
func (mc *MAVLinkController) SendAttitudeCommand(cmd AttitudeCommand) error {
	if !mc.IsConnected() {
		return ErrNotConnected
	}

	select {
	case mc.attitudeCmd <- cmd:
		mc.mu.Lock()
		mc.cmdsSent++
		mc.mu.Unlock()
		return nil
	default:
		return fmt.Errorf("attitude command buffer full")
	}
}

// SendPositionCommand sends position setpoint
func (mc *MAVLinkController) SendPositionCommand(cmd PositionCommand) error {
	if !mc.IsConnected() {
		return ErrNotConnected
	}

	select {
	case mc.positionCmd <- cmd:
		mc.mu.Lock()
		mc.cmdsSent++
		mc.mu.Unlock()
		return nil
	default:
		return fmt.Errorf("position command buffer full")
	}
}

// SendVelocityCommand sends velocity setpoint
func (mc *MAVLinkController) SendVelocityCommand(cmd VelocityCommand) error {
	if !mc.IsConnected() {
		return ErrNotConnected
	}

	select {
	case mc.velocityCmd <- cmd:
		mc.mu.Lock()
		mc.cmdsSent++
		mc.mu.Unlock()
		return nil
	default:
		return fmt.Errorf("velocity command buffer full")
	}
}

// Arm arms the flight controller
func (mc *MAVLinkController) Arm(ctx context.Context) error {
	if !mc.IsConnected() {
		return ErrNotConnected
	}

	mc.logger.Info("Arming flight controller...")

	mc.mu.RLock()
	protocol := mc.protocol
	mc.mu.RUnlock()

	if protocol == nil {
		return fmt.Errorf("not connected")
	}

	// Send arm command
	params := [7]float32{1.0, 0, 0, 0, 0, 0, 0} // Arm = 1.0
	if err := protocol.SendCommandLong(1, 1, MAV_CMD_COMPONENT_ARM_DISARM, 0, params); err != nil {
		return fmt.Errorf("failed to send arm command: %w", err)
	}

	mc.mu.Lock()
	mc.armed = true
	mc.mu.Unlock()

	mc.logger.Info("Flight controller armed")
	return nil
}

// Disarm disarms the flight controller
func (mc *MAVLinkController) Disarm(ctx context.Context) error {
	if !mc.IsConnected() {
		return ErrNotConnected
	}

	mc.logger.Info("Disarming flight controller...")

	mc.mu.RLock()
	protocol := mc.protocol
	mc.mu.RUnlock()

	if protocol == nil {
		return fmt.Errorf("not connected")
	}

	// Send disarm command
	params := [7]float32{0.0, 0, 0, 0, 0, 0, 0} // Disarm = 0.0
	if err := protocol.SendCommandLong(1, 1, MAV_CMD_COMPONENT_ARM_DISARM, 0, params); err != nil {
		return fmt.Errorf("failed to send disarm command: %w", err)
	}

	mc.mu.Lock()
	mc.armed = false
	mc.mu.Unlock()

	mc.logger.Info("Flight controller disarmed")
	return nil
}

// SetFlightMode changes the flight mode
func (mc *MAVLinkController) SetFlightMode(mode string) error {
	if !mc.IsConnected() {
		return ErrNotConnected
	}

	mc.logger.WithField("mode", mode).Info("Setting flight mode...")

	mc.mu.RLock()
	protocol := mc.protocol
	mc.mu.RUnlock()

	if protocol == nil {
		return fmt.Errorf("not connected")
	}

	// Map mode string to MAVLink mode
	baseMode := uint8(MAV_MODE_GUIDED_ARMED)
	customMode := uint8(0)
	switch mode {
	case "MANUAL":
		baseMode = MAV_MODE_MANUAL_ARMED
	case "STABILIZE":
		baseMode = MAV_MODE_STABILIZE_ARMED
	case "GUIDED":
		baseMode = MAV_MODE_GUIDED_ARMED
	case "AUTO":
		baseMode = MAV_MODE_AUTO_ARMED
	}

	if err := protocol.SendSetMode(1, baseMode, customMode); err != nil {
		return fmt.Errorf("failed to set mode: %w", err)
	}

	mc.mu.Lock()
	mc.flightMode = mode
	mc.mu.Unlock()

	return nil
}

// GetTelemetry returns current telemetry
func (mc *MAVLinkController) GetTelemetry() *Telemetry {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return &Telemetry{
		Timestamp:  time.Now(),
		Roll:       mc.currentAttitude[0],
		Pitch:      mc.currentAttitude[1],
		Yaw:        mc.currentAttitude[2],
		PosX:       mc.currentPosition[0],
		PosY:       mc.currentPosition[1],
		PosZ:       mc.currentPosition[2],
		VelX:       mc.currentVelocity[0],
		VelY:       mc.currentVelocity[1],
		VelZ:       mc.currentVelocity[2],
		Armed:      mc.armed,
		FlightMode: mc.flightMode,
	}
}

// Run starts the control loop
func (mc *MAVLinkController) Run(ctx context.Context) error {
	if !mc.IsConnected() {
		if err := mc.Connect(ctx); err != nil {
			return err
		}
	}

	// Start heartbeat
	go mc.heartbeatLoop(ctx)

	// Start telemetry reader
	go mc.telemetryLoop(ctx)

	// Command processing loop
	ticker := time.NewTicker(time.Duration(float64(time.Second) / mc.config.CommandHz))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			mc.Disconnect()
			return ctx.Err()

		case <-ticker.C:
			// Process pending commands
			mc.processCommands()

		case cmd := <-mc.attitudeCmd:
			mc.sendMAVLinkAttitude(cmd)

		case cmd := <-mc.positionCmd:
			mc.sendMAVLinkPosition(cmd)

		case cmd := <-mc.velocityCmd:
			mc.sendMAVLinkVelocity(cmd)
		}
	}
}

// heartbeatLoop sends periodic heartbeats
func (mc *MAVLinkController) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(float64(time.Second) / mc.config.HeartbeatHz))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mc.sendHeartbeat()
		}
	}
}

// telemetryLoop reads telemetry from the flight controller
func (mc *MAVLinkController) telemetryLoop(ctx context.Context) {
	ticker := time.NewTicker(20 * time.Millisecond) // 50 Hz
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			mc.readTelemetry()
		}
	}
}

// processCommands handles queued commands
func (mc *MAVLinkController) processCommands() {
	// Process any buffered commands
	select {
	case cmd := <-mc.attitudeCmd:
		mc.sendMAVLinkAttitude(cmd)
	default:
	}

	select {
	case cmd := <-mc.positionCmd:
		mc.sendMAVLinkPosition(cmd)
	default:
	}

	select {
	case cmd := <-mc.velocityCmd:
		mc.sendMAVLinkVelocity(cmd)
	default:
	}
}

// sendHeartbeat sends MAVLink heartbeat
func (mc *MAVLinkController) sendHeartbeat() {
	mc.mu.RLock()
	protocol := mc.protocol
	mc.mu.RUnlock()

	if protocol == nil {
		return
	}

	autopilot := uint8(8) // MAV_AUTOPILOT_INVALID
	baseMode := uint8(0)
	if mc.armed {
		baseMode |= 0x80 // MAV_MODE_FLAG_SAFETY_ARMED
	}
	customMode := uint8(0)
	systemStatus := uint8(3) // MAV_STATE_STANDBY

	if err := protocol.SendHeartbeat(autopilot, baseMode, customMode, systemStatus); err != nil {
		mc.logger.WithError(err).Warn("Failed to send heartbeat")
		return
	}

	mc.mu.Lock()
	mc.lastHeartbeat = time.Now()
	mc.mu.Unlock()
}

// readTelemetry reads incoming telemetry messages
func (mc *MAVLinkController) readTelemetry() {
	mc.mu.RLock()
	protocol := mc.protocol
	mc.mu.RUnlock()

	if protocol == nil {
		if mc.config.SimulationMode {
			mc.mu.Lock()
			mc.telemetryRcvd++
			mc.mu.Unlock()
		}
		return
	}

	// Read message with 20ms timeout
	msg, err := protocol.ReadMessage(20 * time.Millisecond)
	if err != nil {
		// Timeout is normal, just return
		return
	}

	mc.mu.Lock()
	mc.telemetryRcvd++
	mc.mu.Unlock()

	// Parse message based on ID
	switch msg.MessageID {
	case MAVLINK_MSG_ID_ATTITUDE:
		mc.parseAttitude(msg.Payload)
	case MAVLINK_MSG_ID_LOCAL_POSITION_NED:
		mc.parseLocalPositionNED(msg.Payload)
	case MAVLINK_MSG_ID_SYS_STATUS:
		mc.parseSysStatus(msg.Payload)
	case MAVLINK_MSG_ID_HEARTBEAT:
		mc.parseHeartbeat(msg.Payload)
	}
}

// parseAttitude parses ATTITUDE message
func (mc *MAVLinkController) parseAttitude(payload []byte) {
	if len(payload) < 28 {
		return
	}
	// Parse roll, pitch, yaw, rollspeed, pitchspeed, yawspeed (all float32)
	// Implementation would parse binary data
}

// parseLocalPositionNED parses LOCAL_POSITION_NED message
func (mc *MAVLinkController) parseLocalPositionNED(payload []byte) {
	if len(payload) < 28 {
		return
	}
	// Parse x, y, z, vx, vy, vz (all float32)
	// Implementation would parse binary data
}

// parseSysStatus parses SYS_STATUS message
func (mc *MAVLinkController) parseSysStatus(payload []byte) {
	// Parse system status, battery, etc.
}

// parseHeartbeat parses HEARTBEAT message
func (mc *MAVLinkController) parseHeartbeat(payload []byte) {
	if len(payload) < 9 {
		return
	}
	baseMode := payload[1]
	mc.mu.Lock()
	mc.armed = (baseMode & 0x80) != 0
	mc.mu.Unlock()
}

// sendMAVLinkAttitude sends attitude setpoint
func (mc *MAVLinkController) sendMAVLinkAttitude(cmd AttitudeCommand) {
	mc.mu.RLock()
	protocol := mc.protocol
	mc.mu.RUnlock()

	if protocol == nil {
		mc.logger.Debug("Not connected, skipping attitude command")
		return
	}

	// Convert roll, pitch, yaw to quaternion
	q := eulerToQuaternion(cmd.Roll, cmd.Pitch, cmd.Yaw)
	typeMask := uint8(0b00000111) // Ignore body rates
	timeBootMs := uint32(time.Since(time.Time{}).Milliseconds())

	if err := protocol.SendSetAttitudeTarget(1, 1, timeBootMs, typeMask, q, 0, 0, 0, float32(cmd.Throttle)); err != nil {
		mc.logger.WithError(err).Warn("Failed to send attitude command")
		return
	}

	mc.logger.WithFields(logrus.Fields{
		"roll":     cmd.Roll,
		"pitch":    cmd.Pitch,
		"yaw":      cmd.Yaw,
		"throttle": cmd.Throttle,
	}).Debug("Sent attitude command")
}

// eulerToQuaternion converts Euler angles to quaternion
func eulerToQuaternion(roll, pitch, yaw float64) [4]float32 {
	cy := math.Cos(yaw * 0.5)
	sy := math.Sin(yaw * 0.5)
	cp := math.Cos(pitch * 0.5)
	sp := math.Sin(pitch * 0.5)
	cr := math.Cos(roll * 0.5)
	sr := math.Sin(roll * 0.5)

	return [4]float32{
		float32(cr*cp*cy + sr*sp*sy), // w
		float32(sr*cp*cy - cr*sp*sy), // x
		float32(cr*sp*cy + sr*cp*sy), // y
		float32(cr*cp*sy - sr*sp*cy), // z
	}
}

// sendMAVLinkPosition sends position setpoint
func (mc *MAVLinkController) sendMAVLinkPosition(cmd PositionCommand) {
	mc.mu.RLock()
	protocol := mc.protocol
	mc.mu.RUnlock()

	if protocol == nil {
		mc.logger.Debug("Not connected, skipping position command")
		return
	}

	timeBootMs := uint32(time.Since(time.Time{}).Milliseconds())
	coordinateFrame := uint8(MAVLINK_FRAME_LOCAL_NED)
	typeMask := uint16(0b0000110111111000) // Position only, ignore velocity/acceleration/yaw

	if err := protocol.SendSetPositionTargetLocalNED(1, 1, timeBootMs, coordinateFrame, typeMask,
		float32(cmd.X), float32(cmd.Y), float32(cmd.Z),
		0, 0, 0, // velocity
		0, 0, 0, // acceleration
		0, 0); err != nil { // yaw, yaw rate
		mc.logger.WithError(err).Warn("Failed to send position command")
		return
	}

	mc.logger.WithFields(logrus.Fields{
		"x": cmd.X,
		"y": cmd.Y,
		"z": cmd.Z,
	}).Debug("Sent position command")
}

// sendMAVLinkVelocity sends velocity setpoint
func (mc *MAVLinkController) sendMAVLinkVelocity(cmd VelocityCommand) {
	mc.mu.RLock()
	protocol := mc.protocol
	mc.mu.RUnlock()

	if protocol == nil {
		mc.logger.Debug("Not connected, skipping velocity command")
		return
	}

	timeBootMs := uint32(time.Since(time.Time{}).Milliseconds())
	coordinateFrame := uint8(MAVLINK_FRAME_LOCAL_NED)
	typeMask := uint16(0b0000110111111111) // Velocity only, ignore position/acceleration/yaw

	if err := protocol.SendSetPositionTargetLocalNED(1, 1, timeBootMs, coordinateFrame, typeMask,
		0, 0, 0, // position
		float32(cmd.Vx), float32(cmd.Vy), float32(cmd.Vz), // velocity
		0, 0, 0, // acceleration
		0, 0); err != nil { // yaw, yaw rate
		mc.logger.WithError(err).Warn("Failed to send velocity command")
		return
	}

	mc.logger.WithFields(logrus.Fields{
		"vx": cmd.Vx,
		"vy": cmd.Vy,
		"vz": cmd.Vz,
	}).Debug("Sent velocity command")
}

// GetStats returns controller statistics
func (mc *MAVLinkController) GetStats() (sent, received uint64, lastHB time.Time) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.cmdsSent, mc.telemetryRcvd, mc.lastHeartbeat
}

// IsArmed returns arm status
func (mc *MAVLinkController) IsArmed() bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.armed
}

// GetFlightMode returns current flight mode
func (mc *MAVLinkController) GetFlightMode() string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.flightMode
}
