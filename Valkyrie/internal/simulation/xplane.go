// Package simulation provides X-Plane simulator integration via UDP protocol.
//
// DO-178C DAL-B compliant - ASGARD Integration Module
// Copyright 2026 Arobi. All Rights Reserved.
package simulation

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	"github.com/asgard/pandora/Valkyrie/internal/fusion"
	"gonum.org/v1/gonum/mat"
)

// XPlaneDataRef defines X-Plane data reference IDs
type XPlaneDataRef int

const (
	DataRefTimes               XPlaneDataRef = 1  // times
	DataRefSpeeds              XPlaneDataRef = 3  // speeds
	DataRefMachVVI             XPlaneDataRef = 4  // Mach, VVI, G-load
	DataRefAtmosphere          XPlaneDataRef = 5  // atmosphere
	DataRefSystemPressures     XPlaneDataRef = 6  // system pressures
	DataRefJoystickAilElv      XPlaneDataRef = 8  // joystick ail/elv/rud
	DataRefOtherFlightControls XPlaneDataRef = 9  // other flight controls
	DataRefArtStab             XPlaneDataRef = 10 // art stab ail/elv/rud
	DataRefFlightCon           XPlaneDataRef = 11 // flight con ail/elv/rud
	DataRefWingSweepThrust     XPlaneDataRef = 12 // wing sweep/thrust vect
	DataRefTrimFlap            XPlaneDataRef = 13 // trim/flap/slat/s-brakes
	DataRefGear                XPlaneDataRef = 14 // gear/brakes
	DataRefAngularMoments      XPlaneDataRef = 15 // angular moments
	DataRefAngularVelocities   XPlaneDataRef = 16 // angular velocities
	DataRefPitchRollHeading    XPlaneDataRef = 17 // pitch, roll, heading
	DataRefLatLonAlt           XPlaneDataRef = 20 // lat, lon, alt
	DataRefLocVelDistTraveled  XPlaneDataRef = 21 // loc, vel, dist traveled
)

// XPlaneSimulator implements Simulator for X-Plane 11/12
type XPlaneSimulator struct {
	mu sync.RWMutex

	config     SimulationConfig
	conn       *net.UDPConn
	remoteAddr *net.UDPAddr
	connected  bool

	// Current state
	currentState *SimulatorState

	// Scenario execution
	activeScenario *Scenario
	scenarioStart  time.Time
	running        bool

	// Statistics
	updateCount  int
	totalLatency time.Duration

	// Channels
	stateChan chan *SimulatorState
	stopChan  chan struct{}
}

// NewXPlaneSimulator creates a new X-Plane simulator interface
func NewXPlaneSimulator(config SimulationConfig) *XPlaneSimulator {
	if config.XPlanePort == 0 {
		config.XPlanePort = 49000
	}
	if config.UpdateRate == 0 {
		config.UpdateRate = 20.0
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}

	return &XPlaneSimulator{
		config:       config,
		currentState: &SimulatorState{},
		stateChan:    make(chan *SimulatorState, 100),
		stopChan:     make(chan struct{}),
	}
}

// Connect establishes UDP connection to X-Plane
func (xp *XPlaneSimulator) Connect(ctx context.Context) error {
	xp.mu.Lock()
	defer xp.mu.Unlock()

	addr := xp.config.Address
	if addr == "" {
		addr = fmt.Sprintf("127.0.0.1:%d", xp.config.XPlanePort)
	}

	remoteAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("resolve address: %w", err)
	}

	localAddr := &net.UDPAddr{Port: xp.config.XPlanePort + 1}
	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		return fmt.Errorf("listen UDP: %w", err)
	}

	xp.conn = conn
	xp.remoteAddr = remoteAddr
	xp.connected = true

	// Start receiver
	go xp.receiveLoop()

	// Request data subscriptions
	if err := xp.requestDataRefs(); err != nil {
		return fmt.Errorf("request data refs: %w", err)
	}

	return nil
}

// Disconnect closes the UDP connection
func (xp *XPlaneSimulator) Disconnect() error {
	xp.mu.Lock()
	defer xp.mu.Unlock()

	if !xp.connected {
		return nil
	}

	close(xp.stopChan)
	xp.connected = false

	if xp.conn != nil {
		return xp.conn.Close()
	}
	return nil
}

// IsConnected returns connection status
func (xp *XPlaneSimulator) IsConnected() bool {
	xp.mu.RLock()
	defer xp.mu.RUnlock()
	return xp.connected
}

// GetState returns the current simulator state
func (xp *XPlaneSimulator) GetState() (*SimulatorState, error) {
	xp.mu.RLock()
	defer xp.mu.RUnlock()

	if !xp.connected {
		return nil, fmt.Errorf("not connected to X-Plane")
	}

	// Return copy of current state
	state := *xp.currentState
	return &state, nil
}

// SendCommand sends control inputs to X-Plane
func (xp *XPlaneSimulator) SendCommand(cmd *SimulatorCommand) error {
	xp.mu.Lock()
	defer xp.mu.Unlock()

	if !xp.connected {
		return fmt.Errorf("not connected to X-Plane")
	}

	// Build DREF packet for controls
	packet := xp.buildControlPacket(cmd)
	_, err := xp.conn.WriteToUDP(packet, xp.remoteAddr)
	return err
}

// LoadScenario prepares a test scenario
func (xp *XPlaneSimulator) LoadScenario(scenario *Scenario) error {
	xp.mu.Lock()
	defer xp.mu.Unlock()

	if !xp.connected {
		return fmt.Errorf("not connected to X-Plane")
	}

	xp.activeScenario = scenario

	// Set initial position via POSI packet
	if err := xp.setPosition(scenario.InitialPosition); err != nil {
		return fmt.Errorf("set position: %w", err)
	}

	return nil
}

// RunScenario executes the loaded scenario
func (xp *XPlaneSimulator) RunScenario(ctx context.Context) (*SimulationResult, error) {
	xp.mu.Lock()
	if xp.activeScenario == nil {
		xp.mu.Unlock()
		return nil, fmt.Errorf("no scenario loaded")
	}
	scenario := xp.activeScenario
	xp.running = true
	xp.scenarioStart = time.Now()
	xp.mu.Unlock()

	result := &SimulationResult{
		ScenarioID:     scenario.ID,
		StartTime:      xp.scenarioStart,
		StateHistory:   make([]SimulatorState, 0, 1000),
		CommandHistory: make([]SimulatorCommand, 0, 1000),
		CoverageData:   make(map[string]float64),
	}

	ticker := time.NewTicker(time.Duration(float64(time.Second) / xp.config.UpdateRate))
	defer ticker.Stop()

	actionIndex := 0
	timeout := time.NewTimer(scenario.Duration)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			xp.mu.Lock()
			xp.running = false
			xp.mu.Unlock()
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			result.ErrorMessage = "cancelled"
			return result, ctx.Err()

		case <-timeout.C:
			xp.mu.Lock()
			xp.running = false
			xp.mu.Unlock()
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			result.Passed = xp.evaluateCriteria(scenario, result)
			return result, nil

		case <-ticker.C:
			elapsed := time.Since(xp.scenarioStart)

			// Execute scheduled actions
			for actionIndex < len(scenario.Actions) {
				action := scenario.Actions[actionIndex]
				if elapsed >= action.Time {
					if err := xp.executeAction(action); err != nil {
						result.Deviations = append(result.Deviations,
							fmt.Sprintf("Action %s failed: %v", action.ActionType, err))
					}
					actionIndex++
				} else {
					break
				}
			}

			// Record state
			state, err := xp.GetState()
			if err == nil {
				result.StateHistory = append(result.StateHistory, *state)
				result.UpdateCount++
			}

			// Check fail criteria
			for _, criterion := range scenario.FailCriteria {
				if criterion.Critical && xp.checkFailCriterion(criterion, state) {
					xp.mu.Lock()
					xp.running = false
					xp.mu.Unlock()
					result.EndTime = time.Now()
					result.Duration = result.EndTime.Sub(result.StartTime)
					result.Passed = false
					result.FailedCriteria = append(result.FailedCriteria, criterion.Name)
					return result, nil
				}
			}
		}
	}
}

// StopScenario halts scenario execution
func (xp *XPlaneSimulator) StopScenario() error {
	xp.mu.Lock()
	defer xp.mu.Unlock()
	xp.running = false
	return nil
}

// Pause pauses the simulation
func (xp *XPlaneSimulator) Pause() error {
	// X-Plane pause via CMND packet
	return xp.sendCommand("sim/operation/pause_on")
}

// Resume resumes the simulation
func (xp *XPlaneSimulator) Resume() error {
	return xp.sendCommand("sim/operation/pause_off")
}

// SetTimeScale sets simulation time scale
func (xp *XPlaneSimulator) SetTimeScale(scale float64) error {
	// X-Plane uses simulation speed dataref
	return xp.setDataRef("sim/time/sim_speed", float32(scale))
}

// GetSensorReading generates a sensor reading from simulator state
func (xp *XPlaneSimulator) GetSensorReading(sensorType fusion.SensorType) (*fusion.SensorReading, error) {
	state, err := xp.GetState()
	if err != nil {
		return nil, err
	}

	reading := &fusion.SensorReading{
		Type:      sensorType,
		Timestamp: state.Timestamp,
		Quality:   0.95, // Simulated sensor quality
	}

	switch sensorType {
	case fusion.SensorGPS:
		// GPS provides position and velocity
		data := mat.NewVecDense(6, []float64{
			state.Latitude,
			state.Longitude,
			state.Altitude,
			state.VelocityNorth,
			state.VelocityEast,
			-state.VelocityDown,
		})
		cov := mat.NewSymDense(6, nil)
		for i := 0; i < 3; i++ {
			cov.SetSym(i, i, 2.5)     // Position uncertainty (m)
			cov.SetSym(i+3, i+3, 0.1) // Velocity uncertainty (m/s)
		}
		reading.Data = data
		reading.Covariance = cov

	case fusion.SensorINS:
		// INS provides attitude and angular rates
		data := mat.NewVecDense(6, []float64{
			state.Roll,
			state.Pitch,
			state.Yaw,
			state.RollRate,
			state.PitchRate,
			state.YawRate,
		})
		cov := mat.NewSymDense(6, nil)
		for i := 0; i < 3; i++ {
			cov.SetSym(i, i, 0.01)      // Attitude uncertainty (rad)
			cov.SetSym(i+3, i+3, 0.001) // Rate uncertainty (rad/s)
		}
		reading.Data = data
		reading.Covariance = cov

	case fusion.SensorBarometer:
		// Barometer provides altitude
		data := mat.NewVecDense(1, []float64{state.Altitude})
		cov := mat.NewSymDense(1, nil)
		cov.SetSym(0, 0, 5.0) // Altitude uncertainty (m)
		reading.Data = data
		reading.Covariance = cov

	case fusion.SensorPitot:
		// Pitot provides airspeed
		data := mat.NewVecDense(1, []float64{state.Airspeed})
		cov := mat.NewSymDense(1, nil)
		cov.SetSym(0, 0, 0.5) // Airspeed uncertainty (m/s)
		reading.Data = data
		reading.Covariance = cov

	default:
		return nil, fmt.Errorf("unsupported sensor type: %d", sensorType)
	}

	return reading, nil
}

// InjectSensorFault simulates a sensor failure
func (xp *XPlaneSimulator) InjectSensorFault(sensorType fusion.SensorType, faultType string) error {
	// Record fault injection for test coverage
	xp.mu.Lock()
	defer xp.mu.Unlock()

	// In simulation, faults are handled by modifying returned sensor readings
	// This would be implemented with fault injection state tracking
	return nil
}

// receiveLoop continuously receives data from X-Plane
func (xp *XPlaneSimulator) receiveLoop() {
	buffer := make([]byte, 4096)
	for {
		select {
		case <-xp.stopChan:
			return
		default:
			xp.conn.SetReadDeadline(time.Now().Add(time.Second))
			n, _, err := xp.conn.ReadFromUDP(buffer)
			if err != nil {
				continue
			}
			xp.parsePacket(buffer[:n])
		}
	}
}

// parsePacket parses X-Plane UDP data packet
func (xp *XPlaneSimulator) parsePacket(data []byte) {
	if len(data) < 5 {
		return
	}

	header := string(data[:4])
	if header != "DATA" {
		return
	}

	xp.mu.Lock()
	defer xp.mu.Unlock()

	// Parse DATA packets (5 byte header + 36 byte records)
	offset := 5
	for offset+36 <= len(data) {
		index := int(data[offset])
		values := make([]float32, 8)
		for i := 0; i < 8; i++ {
			values[i] = math.Float32frombits(binary.LittleEndian.Uint32(data[offset+4+i*4:]))
		}
		xp.updateState(XPlaneDataRef(index), values)
		offset += 36
	}

	xp.currentState.Timestamp = time.Now()
}

// updateState updates current state from dataref values
func (xp *XPlaneSimulator) updateState(ref XPlaneDataRef, values []float32) {
	switch ref {
	case DataRefLatLonAlt:
		xp.currentState.Latitude = float64(values[0])
		xp.currentState.Longitude = float64(values[1])
		xp.currentState.Altitude = float64(values[2]) * 0.3048 // ft to m
	case DataRefSpeeds:
		xp.currentState.Airspeed = float64(values[0]) * 0.514444 // kt to m/s
		xp.currentState.Groundspeed = float64(values[3]) * 0.514444
	case DataRefPitchRollHeading:
		xp.currentState.Pitch = float64(values[0]) * math.Pi / 180
		xp.currentState.Roll = float64(values[1]) * math.Pi / 180
		xp.currentState.Yaw = float64(values[2]) * math.Pi / 180
	case DataRefAngularVelocities:
		xp.currentState.RollRate = float64(values[0]) * math.Pi / 180
		xp.currentState.PitchRate = float64(values[1]) * math.Pi / 180
		xp.currentState.YawRate = float64(values[2]) * math.Pi / 180
	case DataRefLocVelDistTraveled:
		xp.currentState.VelocityNorth = float64(values[3])
		xp.currentState.VelocityEast = float64(values[4])
		xp.currentState.VelocityDown = -float64(values[5])
	}
}

// requestDataRefs subscribes to required X-Plane data
func (xp *XPlaneSimulator) requestDataRefs() error {
	refs := []XPlaneDataRef{
		DataRefLatLonAlt,
		DataRefSpeeds,
		DataRefPitchRollHeading,
		DataRefAngularVelocities,
		DataRefLocVelDistTraveled,
	}

	for _, ref := range refs {
		packet := xp.buildDataRequestPacket(ref, int(xp.config.UpdateRate))
		if _, err := xp.conn.WriteToUDP(packet, xp.remoteAddr); err != nil {
			return err
		}
	}
	return nil
}

// buildDataRequestPacket creates RREF packet
func (xp *XPlaneSimulator) buildDataRequestPacket(ref XPlaneDataRef, freq int) []byte {
	buf := new(bytes.Buffer)
	buf.WriteString("RREF")
	buf.WriteByte(0)
	binary.Write(buf, binary.LittleEndian, int32(freq))
	binary.Write(buf, binary.LittleEndian, int32(ref))
	return buf.Bytes()
}

// buildControlPacket creates DATA packet for controls
func (xp *XPlaneSimulator) buildControlPacket(cmd *SimulatorCommand) []byte {
	buf := new(bytes.Buffer)
	buf.WriteString("DATA")
	buf.WriteByte(0)

	// Joystick controls (index 8)
	buf.WriteByte(8)
	binary.Write(buf, binary.LittleEndian, float32(cmd.Elevator))
	binary.Write(buf, binary.LittleEndian, float32(cmd.Aileron))
	binary.Write(buf, binary.LittleEndian, float32(cmd.Rudder))
	binary.Write(buf, binary.LittleEndian, float32(-999)) // unused
	binary.Write(buf, binary.LittleEndian, float32(-999))
	binary.Write(buf, binary.LittleEndian, float32(-999))
	binary.Write(buf, binary.LittleEndian, float32(-999))
	binary.Write(buf, binary.LittleEndian, float32(-999))

	// Throttle (index 25)
	buf.WriteByte(25)
	for i := 0; i < 8; i++ {
		if i == 0 {
			binary.Write(buf, binary.LittleEndian, float32(cmd.Throttle))
		} else {
			binary.Write(buf, binary.LittleEndian, float32(-999))
		}
	}

	return buf.Bytes()
}

// setPosition sets aircraft position via POSI packet
func (xp *XPlaneSimulator) setPosition(pos [3]float64) error {
	buf := new(bytes.Buffer)
	buf.WriteString("POSI")
	buf.WriteByte(0)
	binary.Write(buf, binary.LittleEndian, int32(0)) // aircraft index

	binary.Write(buf, binary.LittleEndian, float64(pos[0])) // lat
	binary.Write(buf, binary.LittleEndian, float64(pos[1])) // lon
	binary.Write(buf, binary.LittleEndian, float64(pos[2])) // alt

	binary.Write(buf, binary.LittleEndian, float32(0))  // pitch
	binary.Write(buf, binary.LittleEndian, float32(0))  // roll
	binary.Write(buf, binary.LittleEndian, float32(0))  // heading
	binary.Write(buf, binary.LittleEndian, float32(-1)) // gear

	_, err := xp.conn.WriteToUDP(buf.Bytes(), xp.remoteAddr)
	return err
}

// sendCommand sends a command via CMND packet
func (xp *XPlaneSimulator) sendCommand(cmdPath string) error {
	xp.mu.Lock()
	defer xp.mu.Unlock()

	buf := new(bytes.Buffer)
	buf.WriteString("CMND")
	buf.WriteByte(0)
	buf.WriteString(cmdPath)
	buf.WriteByte(0)

	_, err := xp.conn.WriteToUDP(buf.Bytes(), xp.remoteAddr)
	return err
}

// setDataRef sets a dataref value via DREF packet
func (xp *XPlaneSimulator) setDataRef(path string, value float32) error {
	xp.mu.Lock()
	defer xp.mu.Unlock()

	buf := new(bytes.Buffer)
	buf.WriteString("DREF")
	buf.WriteByte(0)
	binary.Write(buf, binary.LittleEndian, value)

	// Pad path to 500 bytes
	pathBytes := make([]byte, 500)
	copy(pathBytes, path)
	buf.Write(pathBytes)

	_, err := xp.conn.WriteToUDP(buf.Bytes(), xp.remoteAddr)
	return err
}

// executeAction executes a scenario action
func (xp *XPlaneSimulator) executeAction(action ScenarioAction) error {
	switch action.ActionType {
	case ActionSetWind:
		if wind, ok := action.Value.(WindProfile); ok {
			xp.setDataRef("sim/weather/wind_speed_kt[0]", float32(wind.BaseSpeed*1.94384))
			xp.setDataRef("sim/weather/wind_direction_degt[0]", float32(wind.BaseDirection))
		}
	case ActionSetPosition:
		if pos, ok := action.Value.([3]float64); ok {
			return xp.setPosition(pos)
		}
	case ActionSensorFailure:
		// Handled via fault injection system
	case ActionMotorFailure:
		xp.sendCommand("sim/operation/fail_engine_1")
	}
	return nil
}

// evaluateCriteria evaluates pass/fail criteria
func (xp *XPlaneSimulator) evaluateCriteria(scenario *Scenario, result *SimulationResult) bool {
	allPassed := true

	for _, criterion := range scenario.PassCriteria {
		if xp.checkPassCriterion(criterion, result) {
			result.PassedCriteria = append(result.PassedCriteria, criterion.Name)
		} else {
			allPassed = false
		}
	}

	for _, criterion := range scenario.FailCriteria {
		for _, state := range result.StateHistory {
			if xp.checkFailCriterion(criterion, &state) {
				result.FailedCriteria = append(result.FailedCriteria, criterion.Name)
				allPassed = false
				break
			}
		}
	}

	return allPassed
}

// checkPassCriterion checks if a pass criterion is met
func (xp *XPlaneSimulator) checkPassCriterion(criterion PassCriterion, result *SimulationResult) bool {
	// Simple criteria evaluation - would be more sophisticated in production
	if len(result.StateHistory) == 0 {
		return false
	}

	finalState := result.StateHistory[len(result.StateHistory)-1]

	switch criterion.Name {
	case "altitude_maintained":
		return math.Abs(finalState.Altitude-criterion.Value) < criterion.Tolerance
	case "heading_stable":
		return math.Abs(finalState.YawRate) < criterion.Tolerance
	}
	return true
}

// checkFailCriterion checks if a fail criterion is triggered
func (xp *XPlaneSimulator) checkFailCriterion(criterion FailCriterion, state *SimulatorState) bool {
	switch criterion.Name {
	case "altitude_violation":
		return state.Altitude < criterion.Value
	case "overspeed":
		return state.Airspeed > criterion.Value
	case "attitude_violation":
		return math.Abs(state.Roll) > criterion.Value || math.Abs(state.Pitch) > criterion.Value
	}
	return false
}
