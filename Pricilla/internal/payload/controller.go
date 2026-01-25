package payload

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

// PayloadType defines the type of payload being guided
type PayloadType string

const (
	PayloadHunoid       PayloadType = "hunoid"       // Humanoid robot
	PayloadUAV          PayloadType = "uav"          // Unmanned Aerial Vehicle
	PayloadRocket       PayloadType = "rocket"       // Launch vehicle
	PayloadMissile      PayloadType = "missile"      // Guided missile
	PayloadSpacecraft   PayloadType = "spacecraft"   // Orbital spacecraft
	PayloadGroundRobot  PayloadType = "ground_robot" // Ground-based robot
	PayloadDrone        PayloadType = "drone"        // Multirotor drone
	PayloadSubmarine    PayloadType = "submarine"    // Underwater vehicle
	PayloadInterstellar PayloadType = "interstellar" // Deep space probe
)

// Vector3D represents 3D coordinates
type Vector3D struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// Quaternion represents orientation
type Quaternion struct {
	W float64 `json:"w"`
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// PayloadState represents the current state of a payload
type PayloadState struct {
	ID           string      `json:"id"`
	Type         PayloadType `json:"type"`
	Position     Vector3D    `json:"position"`
	Velocity     Vector3D    `json:"velocity"`
	Acceleration Vector3D    `json:"acceleration"`
	Orientation  Quaternion  `json:"orientation"`
	AngularVel   Vector3D    `json:"angularVelocity"`
	Fuel         float64     `json:"fuel"`         // percentage
	Battery      float64     `json:"battery"`      // percentage
	Health       float64     `json:"health"`       // 0.0-1.0
	Armed        bool        `json:"armed"`
	Status       string      `json:"status"`       // idle, active, mission, error
	Timestamp    time.Time   `json:"timestamp"`
}

// PayloadCapabilities defines what a payload can do
type PayloadCapabilities struct {
	MaxSpeed         float64 `json:"maxSpeed"`         // m/s
	MaxAcceleration  float64 `json:"maxAcceleration"`  // m/s²
	MaxTurnRate      float64 `json:"maxTurnRate"`      // rad/s
	MaxAltitude      float64 `json:"maxAltitude"`      // meters
	MinAltitude      float64 `json:"minAltitude"`      // meters
	MaxRange         float64 `json:"maxRange"`         // meters
	MaxFlightTime    time.Duration `json:"maxFlightTime"`
	CanHover         bool    `json:"canHover"`
	CanVerticalTakeoff bool  `json:"canVerticalTakeoff"`
	HasStealth       bool    `json:"hasStealth"`
	HasWeapons       bool    `json:"hasWeapons"`
	HasSensors       bool    `json:"hasSensors"`
	HasCommunications bool   `json:"hasCommunications"`
}

// Command represents a control command to a payload
type Command struct {
	ID          string      `json:"id"`
	PayloadID   string      `json:"payloadId"`
	Type        CommandType `json:"type"`
	Parameters  map[string]interface{} `json:"parameters"`
	Priority    int         `json:"priority"`
	Timestamp   time.Time   `json:"timestamp"`
	ExpiresAt   time.Time   `json:"expiresAt"`
}

// CommandType defines the type of command
type CommandType string

const (
	CmdNavigateTo    CommandType = "navigate_to"
	CmdHold          CommandType = "hold"
	CmdReturn        CommandType = "return"
	CmdArm           CommandType = "arm"
	CmdDisarm        CommandType = "disarm"
	CmdExecute       CommandType = "execute"
	CmdAbort         CommandType = "abort"
	CmdSetSpeed      CommandType = "set_speed"
	CmdSetAltitude   CommandType = "set_altitude"
	CmdSetHeading    CommandType = "set_heading"
	CmdEngageStealth CommandType = "engage_stealth"
	CmdEmergencyStop CommandType = "emergency_stop"
)

// PayloadController manages a single payload
type PayloadController struct {
	mu sync.RWMutex

	id           string
	payloadID    string
	payloadType  PayloadType
	capabilities PayloadCapabilities
	state        PayloadState
	commands     chan Command
	isRunning    bool
	ctx          context.Context
	cancel       context.CancelFunc

	// Callbacks
	onStateChange   func(state PayloadState)
	onCommandResult func(cmd Command, err error)
}

// NewPayloadController creates a new payload controller
func NewPayloadController(payloadID string, payloadType PayloadType, caps PayloadCapabilities) *PayloadController {
	return &PayloadController{
		id:           uuid.New().String(),
		payloadID:    payloadID,
		payloadType:  payloadType,
		capabilities: caps,
		commands:     make(chan Command, 100),
		state: PayloadState{
			ID:     payloadID,
			Type:   payloadType,
			Status: "idle",
			Fuel:   100.0,
			Battery: 100.0,
			Health: 1.0,
		},
	}
}

// Start begins the payload controller
func (pc *PayloadController) Start(ctx context.Context) error {
	pc.mu.Lock()
	if pc.isRunning {
		pc.mu.Unlock()
		return fmt.Errorf("controller already running")
	}

	pc.ctx, pc.cancel = context.WithCancel(ctx)
	pc.isRunning = true
	pc.state.Status = "active"
	pc.mu.Unlock()

	go pc.commandLoop()

	return nil
}

// Stop stops the payload controller
func (pc *PayloadController) Stop() {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.cancel != nil {
		pc.cancel()
	}
	pc.isRunning = false
	pc.state.Status = "idle"
}

// SendCommand sends a command to the payload
func (pc *PayloadController) SendCommand(cmd Command) error {
	if !pc.isRunning {
		return fmt.Errorf("controller not running")
	}

	cmd.ID = uuid.New().String()
	cmd.PayloadID = pc.payloadID
	cmd.Timestamp = time.Now()

	if cmd.ExpiresAt.IsZero() {
		cmd.ExpiresAt = time.Now().Add(30 * time.Second)
	}

	select {
	case pc.commands <- cmd:
		return nil
	default:
		return fmt.Errorf("command queue full")
	}
}

// GetState returns the current payload state
func (pc *PayloadController) GetState() PayloadState {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.state
}

// GetCapabilities returns the payload capabilities
func (pc *PayloadController) GetCapabilities() PayloadCapabilities {
	return pc.capabilities
}

// commandLoop processes incoming commands
func (pc *PayloadController) commandLoop() {
	for {
		select {
		case <-pc.ctx.Done():
			return

		case cmd := <-pc.commands:
			if time.Now().After(cmd.ExpiresAt) {
				continue // Command expired
			}

			err := pc.executeCommand(cmd)
			if pc.onCommandResult != nil {
				go pc.onCommandResult(cmd, err)
			}
		}
	}
}

// executeCommand executes a single command
func (pc *PayloadController) executeCommand(cmd Command) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	switch cmd.Type {
	case CmdNavigateTo:
		return pc.handleNavigateTo(cmd)
	case CmdHold:
		return pc.handleHold(cmd)
	case CmdReturn:
		return pc.handleReturn(cmd)
	case CmdArm:
		pc.state.Armed = true
		return nil
	case CmdDisarm:
		pc.state.Armed = false
		return nil
	case CmdAbort:
		return pc.handleAbort(cmd)
	case CmdSetSpeed:
		return pc.handleSetSpeed(cmd)
	case CmdSetAltitude:
		return pc.handleSetAltitude(cmd)
	case CmdSetHeading:
		return pc.handleSetHeading(cmd)
	case CmdEngageStealth:
		return pc.handleEngageStealth(cmd)
	case CmdEmergencyStop:
		return pc.handleEmergencyStop(cmd)
	default:
		return fmt.Errorf("unknown command type: %s", cmd.Type)
	}
}

// handleNavigateTo processes navigation commands
func (pc *PayloadController) handleNavigateTo(cmd Command) error {
	x, ok := cmd.Parameters["x"].(float64)
	if !ok {
		return fmt.Errorf("missing x parameter")
	}
	y, ok := cmd.Parameters["y"].(float64)
	if !ok {
		return fmt.Errorf("missing y parameter")
	}
	z, _ := cmd.Parameters["z"].(float64)

	// Calculate direction to target
	target := Vector3D{X: x, Y: y, Z: z}
	direction := Vector3D{
		X: target.X - pc.state.Position.X,
		Y: target.Y - pc.state.Position.Y,
		Z: target.Z - pc.state.Position.Z,
	}

	distance := magnitude(direction)
	if distance < 1.0 {
		return nil // Already at target
	}

	// Calculate velocity towards target
	speed := math.Min(pc.capabilities.MaxSpeed, distance/10.0)
	normalized := normalize(direction)

	pc.state.Velocity = Vector3D{
		X: normalized.X * speed,
		Y: normalized.Y * speed,
		Z: normalized.Z * speed,
	}

	pc.state.Status = "navigating"
	return nil
}

// handleHold stops movement and holds position
func (pc *PayloadController) handleHold(cmd Command) error {
	pc.state.Velocity = Vector3D{X: 0, Y: 0, Z: 0}
	pc.state.Acceleration = Vector3D{X: 0, Y: 0, Z: 0}
	pc.state.Status = "holding"
	return nil
}

// handleReturn initiates return to base
func (pc *PayloadController) handleReturn(cmd Command) error {
	// Navigate to origin
	return pc.handleNavigateTo(Command{
		Parameters: map[string]interface{}{
			"x": 0.0,
			"y": 0.0,
			"z": pc.capabilities.MinAltitude,
		},
	})
}

// handleAbort aborts current mission
func (pc *PayloadController) handleAbort(cmd Command) error {
	pc.state.Velocity = Vector3D{X: 0, Y: 0, Z: 0}
	pc.state.Status = "aborted"
	pc.state.Armed = false
	return nil
}

// handleSetSpeed sets target speed
func (pc *PayloadController) handleSetSpeed(cmd Command) error {
	speed, ok := cmd.Parameters["speed"].(float64)
	if !ok {
		return fmt.Errorf("missing speed parameter")
	}

	speed = math.Min(speed, pc.capabilities.MaxSpeed)

	// Scale current velocity to new speed
	currentSpeed := magnitude(pc.state.Velocity)
	if currentSpeed > 0 {
		scale := speed / currentSpeed
		pc.state.Velocity.X *= scale
		pc.state.Velocity.Y *= scale
		pc.state.Velocity.Z *= scale
	}

	return nil
}

// handleSetAltitude sets target altitude
func (pc *PayloadController) handleSetAltitude(cmd Command) error {
	altitude, ok := cmd.Parameters["altitude"].(float64)
	if !ok {
		return fmt.Errorf("missing altitude parameter")
	}

	altitude = clamp(altitude, pc.capabilities.MinAltitude, pc.capabilities.MaxAltitude)

	// Set vertical velocity to reach target altitude
	altError := altitude - pc.state.Position.Z
	pc.state.Velocity.Z = clamp(altError, -pc.capabilities.MaxSpeed, pc.capabilities.MaxSpeed)

	return nil
}

// handleSetHeading sets target heading
func (pc *PayloadController) handleSetHeading(cmd Command) error {
	heading, ok := cmd.Parameters["heading"].(float64)
	if !ok {
		return fmt.Errorf("missing heading parameter")
	}

	// Convert heading to quaternion
	pc.state.Orientation = headingToQuaternion(heading)
	return nil
}

// handleEngageStealth activates stealth mode
func (pc *PayloadController) handleEngageStealth(cmd Command) error {
	if !pc.capabilities.HasStealth {
		return fmt.Errorf("payload does not support stealth")
	}

	enabled, ok := cmd.Parameters["enabled"].(bool)
	if !ok {
		enabled = true
	}

	if enabled {
		pc.state.Status = "stealth"
		// Reduce speed for stealth
		scale := 0.5
		pc.state.Velocity.X *= scale
		pc.state.Velocity.Y *= scale
		pc.state.Velocity.Z *= scale
	} else {
		pc.state.Status = "active"
	}

	return nil
}

// handleEmergencyStop immediately stops the payload
func (pc *PayloadController) handleEmergencyStop(cmd Command) error {
	pc.state.Velocity = Vector3D{X: 0, Y: 0, Z: 0}
	pc.state.Acceleration = Vector3D{X: 0, Y: 0, Z: 0}
	pc.state.Armed = false
	pc.state.Status = "emergency_stop"
	return nil
}

// ApplyTelemetry updates state from a real telemetry source.
func (pc *PayloadController) ApplyTelemetry(state PayloadState) {
	pc.mu.Lock()
	if state.ID == "" {
		state.ID = pc.payloadID
	}
	state.Timestamp = time.Now().UTC()
	pc.state = state
	pc.mu.Unlock()

	if pc.onStateChange != nil {
		go pc.onStateChange(state)
	}
}

// OnStateChange sets callback for state changes
func (pc *PayloadController) OnStateChange(callback func(state PayloadState)) {
	pc.onStateChange = callback
}

// OnCommandResult sets callback for command results
func (pc *PayloadController) OnCommandResult(callback func(cmd Command, err error)) {
	pc.onCommandResult = callback
}

// MultiPayloadController manages multiple payloads
type MultiPayloadController struct {
	mu sync.RWMutex

	id          string
	controllers map[string]*PayloadController
	isRunning   bool
}

// NewMultiPayloadController creates a multi-payload controller
func NewMultiPayloadController() *MultiPayloadController {
	return &MultiPayloadController{
		id:          uuid.New().String(),
		controllers: make(map[string]*PayloadController),
	}
}

// AddPayload adds a payload to the controller
func (mpc *MultiPayloadController) AddPayload(payloadID string, payloadType PayloadType, caps PayloadCapabilities) *PayloadController {
	mpc.mu.Lock()
	defer mpc.mu.Unlock()

	controller := NewPayloadController(payloadID, payloadType, caps)
	mpc.controllers[payloadID] = controller
	return controller
}

// RemovePayload removes a payload from the controller
func (mpc *MultiPayloadController) RemovePayload(payloadID string) {
	mpc.mu.Lock()
	defer mpc.mu.Unlock()

	if controller, exists := mpc.controllers[payloadID]; exists {
		controller.Stop()
		delete(mpc.controllers, payloadID)
	}
}

// GetPayload returns a specific payload controller
func (mpc *MultiPayloadController) GetPayload(payloadID string) (*PayloadController, bool) {
	mpc.mu.RLock()
	defer mpc.mu.RUnlock()

	controller, exists := mpc.controllers[payloadID]
	return controller, exists
}

// GetAllPayloads returns all payload controllers
func (mpc *MultiPayloadController) GetAllPayloads() map[string]*PayloadController {
	mpc.mu.RLock()
	defer mpc.mu.RUnlock()

	result := make(map[string]*PayloadController)
	for k, v := range mpc.controllers {
		result[k] = v
	}
	return result
}

// Start starts all payload controllers
func (mpc *MultiPayloadController) Start(ctx context.Context) error {
	mpc.mu.Lock()
	defer mpc.mu.Unlock()

	for _, controller := range mpc.controllers {
		if err := controller.Start(ctx); err != nil {
			return err
		}
	}

	mpc.isRunning = true
	return nil
}

// Stop stops all payload controllers
func (mpc *MultiPayloadController) Stop() {
	mpc.mu.Lock()
	defer mpc.mu.Unlock()

	for _, controller := range mpc.controllers {
		controller.Stop()
	}

	mpc.isRunning = false
}

// BroadcastCommand sends a command to all payloads
func (mpc *MultiPayloadController) BroadcastCommand(cmd Command) map[string]error {
	mpc.mu.RLock()
	defer mpc.mu.RUnlock()

	results := make(map[string]error)
	for id, controller := range mpc.controllers {
		results[id] = controller.SendCommand(cmd)
	}
	return results
}

// GetAllStates returns states of all payloads
func (mpc *MultiPayloadController) GetAllStates() map[string]PayloadState {
	mpc.mu.RLock()
	defer mpc.mu.RUnlock()

	states := make(map[string]PayloadState)
	for id, controller := range mpc.controllers {
		states[id] = controller.GetState()
	}
	return states
}

// Helper functions

func magnitude(v Vector3D) float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

func normalize(v Vector3D) Vector3D {
	mag := magnitude(v)
	if mag == 0 {
		return Vector3D{}
	}
	return Vector3D{X: v.X / mag, Y: v.Y / mag, Z: v.Z / mag}
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func headingToQuaternion(heading float64) Quaternion {
	// Convert heading (yaw) to quaternion
	halfAngle := heading / 2.0
	return Quaternion{
		W: math.Cos(halfAngle),
		X: 0,
		Y: 0,
		Z: math.Sin(halfAngle),
	}
}

// Default capabilities for different payload types
func GetDefaultCapabilities(payloadType PayloadType) PayloadCapabilities {
	switch payloadType {
	case PayloadHunoid:
		return PayloadCapabilities{
			MaxSpeed:           3.0,     // m/s walking
			MaxAcceleration:    2.0,     // m/s²
			MaxTurnRate:        1.0,     // rad/s
			MaxAltitude:        0,       // ground only
			MinAltitude:        0,
			MaxRange:           50000,   // 50km
			MaxFlightTime:      0,
			CanHover:           false,
			CanVerticalTakeoff: false,
			HasStealth:         false,
			HasWeapons:         false,
			HasSensors:         true,
			HasCommunications:  true,
		}

	case PayloadUAV:
		return PayloadCapabilities{
			MaxSpeed:           80.0,    // m/s
			MaxAcceleration:    15.0,    // m/s²
			MaxTurnRate:        3.0,     // rad/s
			MaxAltitude:        10000,   // 10km
			MinAltitude:        50,
			MaxRange:           200000,  // 200km
			MaxFlightTime:      8 * time.Hour,
			CanHover:           false,
			CanVerticalTakeoff: false,
			HasStealth:         true,
			HasWeapons:         false,
			HasSensors:         true,
			HasCommunications:  true,
		}

	case PayloadRocket:
		return PayloadCapabilities{
			MaxSpeed:           3000.0,  // m/s
			MaxAcceleration:    50.0,    // m/s²
			MaxTurnRate:        0.5,     // rad/s
			MaxAltitude:        400000,  // 400km (LEO)
			MinAltitude:        0,
			MaxRange:           40000000, // around the world
			MaxFlightTime:      15 * time.Minute,
			CanHover:           true,    // with propulsive landing
			CanVerticalTakeoff: true,
			HasStealth:         false,
			HasWeapons:         false,
			HasSensors:         true,
			HasCommunications:  true,
		}

	case PayloadMissile:
		return PayloadCapabilities{
			MaxSpeed:           1000.0,  // m/s (Mach 3)
			MaxAcceleration:    100.0,   // m/s²
			MaxTurnRate:        5.0,     // rad/s
			MaxAltitude:        30000,   // 30km
			MinAltitude:        10,
			MaxRange:           500000,  // 500km
			MaxFlightTime:      30 * time.Minute,
			CanHover:           false,
			CanVerticalTakeoff: true,
			HasStealth:         true,
			HasWeapons:         true,
			HasSensors:         true,
			HasCommunications:  true,
		}

	case PayloadSpacecraft:
		return PayloadCapabilities{
			MaxSpeed:           10000.0, // m/s
			MaxAcceleration:    5.0,     // m/s²
			MaxTurnRate:        0.1,     // rad/s
			MaxAltitude:        1000000000, // 1M km
			MinAltitude:        200000,  // LEO min
			MaxRange:           1e12,    // solar system scale
			MaxFlightTime:      8760 * time.Hour, // 1 year
			CanHover:           false,
			CanVerticalTakeoff: false,
			HasStealth:         false,
			HasWeapons:         false,
			HasSensors:         true,
			HasCommunications:  true,
		}

	case PayloadDrone:
		return PayloadCapabilities{
			MaxSpeed:           20.0,    // m/s
			MaxAcceleration:    10.0,    // m/s²
			MaxTurnRate:        5.0,     // rad/s
			MaxAltitude:        500,     // 500m
			MinAltitude:        1,
			MaxRange:           10000,   // 10km
			MaxFlightTime:      30 * time.Minute,
			CanHover:           true,
			CanVerticalTakeoff: true,
			HasStealth:         false,
			HasWeapons:         false,
			HasSensors:         true,
			HasCommunications:  true,
		}

	case PayloadGroundRobot:
		return PayloadCapabilities{
			MaxSpeed:           5.0,     // m/s
			MaxAcceleration:    3.0,     // m/s²
			MaxTurnRate:        2.0,     // rad/s
			MaxAltitude:        0,
			MinAltitude:        0,
			MaxRange:           100000,  // 100km
			MaxFlightTime:      0,
			CanHover:           false,
			CanVerticalTakeoff: false,
			HasStealth:         true,
			HasWeapons:         false,
			HasSensors:         true,
			HasCommunications:  true,
		}

	case PayloadSubmarine:
		return PayloadCapabilities{
			MaxSpeed:           15.0,    // m/s (~30 knots)
			MaxAcceleration:    2.0,     // m/s²
			MaxTurnRate:        0.3,     // rad/s
			MaxAltitude:        0,       // surface
			MinAltitude:        -1000,   // 1km depth
			MaxRange:           10000000, // global
			MaxFlightTime:      2160 * time.Hour, // 90 days
			CanHover:           true,
			CanVerticalTakeoff: false,
			HasStealth:         true,
			HasWeapons:         true,
			HasSensors:         true,
			HasCommunications:  true,
		}

	case PayloadInterstellar:
		return PayloadCapabilities{
			MaxSpeed:           50000.0, // m/s (relative to Sun)
			MaxAcceleration:    0.01,    // m/s²
			MaxTurnRate:        0.001,   // rad/s
			MaxAltitude:        1e15,    // light years
			MinAltitude:        0,
			MaxRange:           1e18,    // interstellar
			MaxFlightTime:      876000 * time.Hour, // 100 years
			CanHover:           false,
			CanVerticalTakeoff: false,
			HasStealth:         false,
			HasWeapons:         false,
			HasSensors:         true,
			HasCommunications:  true,
		}

	default:
		return PayloadCapabilities{
			MaxSpeed:           10.0,
			MaxAcceleration:    5.0,
			MaxTurnRate:        1.0,
			MaxAltitude:        1000,
			MinAltitude:        0,
			MaxRange:           10000,
			MaxFlightTime:      1 * time.Hour,
		}
	}
}
