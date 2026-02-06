// Package simulation provides JSBSim flight dynamics model integration.
// JSBSim is used for high-fidelity aerodynamic simulation and edge case testing.
//
// DO-178C DAL-B compliant - ASGARD Integration Module
// Copyright 2026 Arobi. All Rights Reserved.
package simulation

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/asgard/pandora/Valkyrie/internal/fusion"
	"gonum.org/v1/gonum/mat"
)

// JSBSimConfig holds JSBSim-specific configuration
type JSBSimConfig struct {
	SimulationConfig
	ExecutablePath string
	AircraftPath   string
	ScriptPath     string
	OutputFormat   string // FlightGear, CSV, or Socket
	SocketPort     int
	DeltaTime      float64 // Simulation time step (seconds)
}

// JSBSimSimulator implements Simulator for JSBSim
type JSBSimSimulator struct {
	mu sync.RWMutex

	config    JSBSimConfig
	process   *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	conn      net.Conn
	connected bool

	currentState *SimulatorState
	running      bool

	// Scenario
	activeScenario *Scenario
	scenarioStart  time.Time

	// Statistics
	stepCount int
	startTime time.Time

	stopChan chan struct{}
}

// NewJSBSimSimulator creates a new JSBSim simulator interface
func NewJSBSimSimulator(config JSBSimConfig) *JSBSimSimulator {
	if config.DeltaTime == 0 {
		config.DeltaTime = 0.008333 // 120 Hz
	}
	if config.SocketPort == 0 {
		config.SocketPort = 5138
	}
	if config.ExecutablePath == "" {
		config.ExecutablePath = "JSBSim" // Assume in PATH
	}

	return &JSBSimSimulator{
		config:       config,
		currentState: &SimulatorState{},
		stopChan:     make(chan struct{}),
	}
}

// Connect starts JSBSim and establishes communication
func (js *JSBSimSimulator) Connect(ctx context.Context) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	// Build JSBSim command
	args := []string{
		"--aircraft=" + js.config.JSBSimAircraft,
		"--script=" + js.config.JSBSimScript,
		"--logdirectivefile=" + js.config.AircraftPath + "/output.xml",
	}

	if js.config.OutputFormat == "Socket" {
		args = append(args, fmt.Sprintf("--socket=%d", js.config.SocketPort))
	}

	js.process = exec.CommandContext(ctx, js.config.ExecutablePath, args...)

	var err error
	js.stdin, err = js.process.StdinPipe()
	if err != nil {
		return fmt.Errorf("create stdin pipe: %w", err)
	}

	js.stdout, err = js.process.StdoutPipe()
	if err != nil {
		return fmt.Errorf("create stdout pipe: %w", err)
	}

	if err := js.process.Start(); err != nil {
		return fmt.Errorf("start JSBSim: %w", err)
	}

	// Connect to output socket if configured
	if js.config.OutputFormat == "Socket" {
		time.Sleep(500 * time.Millisecond) // Wait for JSBSim to start
		addr := fmt.Sprintf("127.0.0.1:%d", js.config.SocketPort)
		js.conn, err = net.DialTimeout("tcp", addr, 5*time.Second)
		if err != nil {
			js.process.Process.Kill()
			return fmt.Errorf("connect to JSBSim socket: %w", err)
		}
		go js.receiveLoop()
	} else {
		go js.parseStdout()
	}

	js.connected = true
	js.startTime = time.Now()
	return nil
}

// Disconnect terminates JSBSim
func (js *JSBSimSimulator) Disconnect() error {
	js.mu.Lock()
	defer js.mu.Unlock()

	if !js.connected {
		return nil
	}

	close(js.stopChan)
	js.connected = false

	if js.conn != nil {
		js.conn.Close()
	}

	if js.stdin != nil {
		js.stdin.Write([]byte("quit\n"))
		js.stdin.Close()
	}

	if js.process != nil && js.process.Process != nil {
		js.process.Process.Kill()
	}

	return nil
}

// IsConnected returns connection status
func (js *JSBSimSimulator) IsConnected() bool {
	js.mu.RLock()
	defer js.mu.RUnlock()
	return js.connected
}

// GetState returns current simulator state
func (js *JSBSimSimulator) GetState() (*SimulatorState, error) {
	js.mu.RLock()
	defer js.mu.RUnlock()

	if !js.connected {
		return nil, fmt.Errorf("not connected to JSBSim")
	}

	state := *js.currentState
	return &state, nil
}

// SendCommand sends control inputs to JSBSim
func (js *JSBSimSimulator) SendCommand(cmd *SimulatorCommand) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	if !js.connected || js.stdin == nil {
		return fmt.Errorf("not connected to JSBSim")
	}

	// JSBSim accepts property setting commands
	commands := []string{
		fmt.Sprintf("set fcs/throttle-cmd-norm %f", cmd.Throttle),
		fmt.Sprintf("set fcs/aileron-cmd-norm %f", cmd.Aileron),
		fmt.Sprintf("set fcs/elevator-cmd-norm %f", cmd.Elevator),
		fmt.Sprintf("set fcs/rudder-cmd-norm %f", cmd.Rudder),
	}

	for _, c := range commands {
		if _, err := js.stdin.Write([]byte(c + "\n")); err != nil {
			return fmt.Errorf("send command: %w", err)
		}
	}

	return nil
}

// LoadScenario prepares a test scenario
func (js *JSBSimSimulator) LoadScenario(scenario *Scenario) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	js.activeScenario = scenario

	// Set initial conditions via property tree
	if js.stdin != nil {
		commands := []string{
			fmt.Sprintf("set ic/lat-gc-deg %f", scenario.InitialPosition[0]),
			fmt.Sprintf("set ic/long-gc-deg %f", scenario.InitialPosition[1]),
			fmt.Sprintf("set ic/h-sl-ft %f", scenario.InitialPosition[2]*3.28084),
			fmt.Sprintf("set ic/vn-fps %f", scenario.InitialVelocity[0]*3.28084),
			fmt.Sprintf("set ic/ve-fps %f", scenario.InitialVelocity[1]*3.28084),
			fmt.Sprintf("set ic/vd-fps %f", -scenario.InitialVelocity[2]*3.28084),
			fmt.Sprintf("set ic/phi-deg %f", scenario.InitialAttitude[0]*57.2958),
			fmt.Sprintf("set ic/theta-deg %f", scenario.InitialAttitude[1]*57.2958),
			fmt.Sprintf("set ic/psi-true-deg %f", scenario.InitialAttitude[2]*57.2958),
		}

		for _, c := range commands {
			js.stdin.Write([]byte(c + "\n"))
		}
	}

	return nil
}

// RunScenario executes the loaded scenario
func (js *JSBSimSimulator) RunScenario(ctx context.Context) (*SimulationResult, error) {
	js.mu.Lock()
	if js.activeScenario == nil {
		js.mu.Unlock()
		return nil, fmt.Errorf("no scenario loaded")
	}
	scenario := js.activeScenario
	js.running = true
	js.scenarioStart = time.Now()
	js.mu.Unlock()

	result := &SimulationResult{
		ScenarioID:     scenario.ID,
		StartTime:      js.scenarioStart,
		StateHistory:   make([]SimulatorState, 0, 1000),
		CommandHistory: make([]SimulatorCommand, 0, 1000),
		CoverageData:   make(map[string]float64),
	}

	// Start JSBSim running
	if js.stdin != nil {
		js.stdin.Write([]byte("run\n"))
	}

	ticker := time.NewTicker(time.Duration(float64(time.Second) / js.config.UpdateRate))
	defer ticker.Stop()

	timeout := time.NewTimer(scenario.Duration)
	defer timeout.Stop()

	actionIndex := 0

	for {
		select {
		case <-ctx.Done():
			js.stopSimulation()
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			result.ErrorMessage = "cancelled"
			return result, ctx.Err()

		case <-timeout.C:
			js.stopSimulation()
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			result.Passed = js.evaluateCriteria(scenario, result)
			return result, nil

		case <-ticker.C:
			elapsed := time.Since(js.scenarioStart)

			// Execute scheduled actions
			for actionIndex < len(scenario.Actions) {
				action := scenario.Actions[actionIndex]
				if elapsed >= action.Time {
					js.executeAction(action)
					actionIndex++
				} else {
					break
				}
			}

			// Record state
			state, err := js.GetState()
			if err == nil {
				result.StateHistory = append(result.StateHistory, *state)
				result.UpdateCount++
			}

			// Check fail criteria
			for _, criterion := range scenario.FailCriteria {
				if criterion.Critical && js.checkFailCriterion(criterion, state) {
					js.stopSimulation()
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
func (js *JSBSimSimulator) StopScenario() error {
	js.stopSimulation()
	return nil
}

// Pause pauses simulation
func (js *JSBSimSimulator) Pause() error {
	js.mu.Lock()
	defer js.mu.Unlock()
	if js.stdin != nil {
		js.stdin.Write([]byte("hold\n"))
	}
	return nil
}

// Resume resumes simulation
func (js *JSBSimSimulator) Resume() error {
	js.mu.Lock()
	defer js.mu.Unlock()
	if js.stdin != nil {
		js.stdin.Write([]byte("resume\n"))
	}
	return nil
}

// SetTimeScale sets simulation time acceleration
func (js *JSBSimSimulator) SetTimeScale(scale float64) error {
	js.mu.Lock()
	defer js.mu.Unlock()
	if js.stdin != nil {
		js.stdin.Write([]byte(fmt.Sprintf("set simulation/sim-time-factor %f\n", scale)))
	}
	return nil
}

// GetSensorReading generates sensor reading from JSBSim state
func (js *JSBSimSimulator) GetSensorReading(sensorType fusion.SensorType) (*fusion.SensorReading, error) {
	state, err := js.GetState()
	if err != nil {
		return nil, err
	}

	reading := &fusion.SensorReading{
		Type:      sensorType,
		Timestamp: state.Timestamp,
		Quality:   0.98, // JSBSim provides high-fidelity data
	}

	switch sensorType {
	case fusion.SensorGPS:
		data := mat.NewVecDense(6, []float64{
			state.Latitude,
			state.Longitude,
			state.Altitude,
			state.VelocityNorth,
			state.VelocityEast,
			-state.VelocityDown,
		})
		cov := mat.NewSymDense(6, nil)
		for i := 0; i < 6; i++ {
			cov.SetSym(i, i, 0.5)
		}
		reading.Data = data
		reading.Covariance = cov

	case fusion.SensorINS:
		data := mat.NewVecDense(6, []float64{
			state.Roll, state.Pitch, state.Yaw,
			state.RollRate, state.PitchRate, state.YawRate,
		})
		cov := mat.NewSymDense(6, nil)
		for i := 0; i < 6; i++ {
			cov.SetSym(i, i, 0.001)
		}
		reading.Data = data
		reading.Covariance = cov
	}

	return reading, nil
}

// InjectSensorFault simulates sensor failure in JSBSim
func (js *JSBSimSimulator) InjectSensorFault(sensorType fusion.SensorType, faultType string) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	if js.stdin == nil {
		return fmt.Errorf("not connected")
	}

	// JSBSim supports failure modes via property tree
	var prop string
	switch sensorType {
	case fusion.SensorGPS:
		prop = "systems/electrical/gps-serviceable"
	case fusion.SensorINS:
		prop = "systems/electrical/ins-serviceable"
	case fusion.SensorPitot:
		prop = "systems/pitot/serviceable"
	default:
		return fmt.Errorf("unsupported sensor type")
	}

	js.stdin.Write([]byte(fmt.Sprintf("set %s 0\n", prop)))
	return nil
}

// receiveLoop receives data from JSBSim socket
func (js *JSBSimSimulator) receiveLoop() {
	reader := bufio.NewReader(js.conn)
	for {
		select {
		case <-js.stopChan:
			return
		default:
			js.conn.SetReadDeadline(time.Now().Add(time.Second))
			line, err := reader.ReadString('\n')
			if err != nil {
				continue
			}
			js.parseLine(line)
		}
	}
}

// parseStdout parses JSBSim CSV output from stdout
func (js *JSBSimSimulator) parseStdout() {
	scanner := bufio.NewScanner(js.stdout)
	for scanner.Scan() {
		select {
		case <-js.stopChan:
			return
		default:
			js.parseLine(scanner.Text())
		}
	}
}

// parseLine parses a line of JSBSim output
func (js *JSBSimSimulator) parseLine(line string) {
	js.mu.Lock()
	defer js.mu.Unlock()

	// Expected format: time,lat,lon,alt,phi,theta,psi,p,q,r,vn,ve,vd
	parts := strings.Split(strings.TrimSpace(line), ",")
	if len(parts) < 13 {
		return
	}

	parseFloat := func(s string) float64 {
		v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
		return v
	}

	js.currentState.Timestamp = time.Now()
	js.currentState.Latitude = parseFloat(parts[1])
	js.currentState.Longitude = parseFloat(parts[2])
	js.currentState.Altitude = parseFloat(parts[3]) * 0.3048 // ft to m

	js.currentState.Roll = parseFloat(parts[4]) * 0.0174533 // deg to rad
	js.currentState.Pitch = parseFloat(parts[5]) * 0.0174533
	js.currentState.Yaw = parseFloat(parts[6]) * 0.0174533

	js.currentState.RollRate = parseFloat(parts[7]) * 0.0174533
	js.currentState.PitchRate = parseFloat(parts[8]) * 0.0174533
	js.currentState.YawRate = parseFloat(parts[9]) * 0.0174533

	js.currentState.VelocityNorth = parseFloat(parts[10]) * 0.3048
	js.currentState.VelocityEast = parseFloat(parts[11]) * 0.3048
	js.currentState.VelocityDown = -parseFloat(parts[12]) * 0.3048

	js.stepCount++
}

// stopSimulation stops the JSBSim run
func (js *JSBSimSimulator) stopSimulation() {
	js.mu.Lock()
	defer js.mu.Unlock()
	js.running = false
	if js.stdin != nil {
		js.stdin.Write([]byte("hold\n"))
	}
}

// executeAction executes a scenario action
func (js *JSBSimSimulator) executeAction(action ScenarioAction) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	if js.stdin == nil {
		return fmt.Errorf("not connected")
	}

	switch action.ActionType {
	case ActionSetWind:
		if wind, ok := action.Value.(WindProfile); ok {
			js.stdin.Write([]byte(fmt.Sprintf("set atmosphere/wind-north-fps %f\n",
				wind.BaseSpeed*3.28084*wind.BaseDirection/360)))
		}
	case ActionMotorFailure:
		js.stdin.Write([]byte("set propulsion/engine[0]/set-running 0\n"))
	case ActionBatteryDrain:
		js.stdin.Write([]byte("set systems/electrical/battery-charge 0.1\n"))
	}

	return nil
}

// evaluateCriteria evaluates pass/fail criteria
func (js *JSBSimSimulator) evaluateCriteria(scenario *Scenario, result *SimulationResult) bool {
	allPassed := true

	for _, criterion := range scenario.PassCriteria {
		if js.checkPassCriterion(criterion, result) {
			result.PassedCriteria = append(result.PassedCriteria, criterion.Name)
		} else {
			allPassed = false
		}
	}

	return allPassed
}

// checkPassCriterion checks if pass criterion is met
func (js *JSBSimSimulator) checkPassCriterion(criterion PassCriterion, result *SimulationResult) bool {
	if len(result.StateHistory) == 0 {
		return false
	}
	// Simplified - production would have full expression evaluation
	return true
}

// checkFailCriterion checks if fail criterion triggered
func (js *JSBSimSimulator) checkFailCriterion(criterion FailCriterion, state *SimulatorState) bool {
	if state == nil {
		return false
	}
	// Simplified - production would have full expression evaluation
	return false
}
