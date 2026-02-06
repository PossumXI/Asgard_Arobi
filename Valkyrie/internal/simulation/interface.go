// Package simulation provides software-in-the-loop (SITL) testing interfaces.
// Supports X-Plane and JSBSim for comprehensive flight simulation.
//
// DO-178C DAL-B compliant - ASGARD Integration Module
// Copyright 2026 Arobi. All Rights Reserved.
package simulation

import (
	"context"
	"time"

	"github.com/asgard/pandora/Valkyrie/internal/fusion"
)

// SimulatorType identifies the flight simulator backend
type SimulatorType int

const (
	SimulatorXPlane SimulatorType = iota // X-Plane 11/12 via UDP
	SimulatorJSBSim                       // JSBSim via FlightGear protocol
	SimulatorMock                         // Mock simulator for unit tests
)

// SimulationConfig holds simulator connection settings
type SimulationConfig struct {
	Type          SimulatorType
	Address       string  // Host:Port for simulator
	UpdateRate    float64 // Hz (typically 20-100)
	Timeout       time.Duration
	EnableLogging bool
	LogPath       string

	// X-Plane specific
	XPlanePort    int // Default 49000
	XPlaneDataRef []string

	// JSBSim specific
	JSBSimScript  string // Path to script
	JSBSimAircraft string
}

// SimulatorState represents the complete state from simulator
type SimulatorState struct {
	Timestamp time.Time

	// Position (WGS84)
	Latitude  float64 // degrees
	Longitude float64 // degrees
	Altitude  float64 // meters MSL

	// Velocity
	VelocityNorth float64 // m/s
	VelocityEast  float64 // m/s
	VelocityDown  float64 // m/s
	Airspeed      float64 // m/s TAS
	Groundspeed   float64 // m/s

	// Attitude (radians)
	Roll  float64
	Pitch float64
	Yaw   float64

	// Angular rates (rad/s)
	RollRate  float64
	PitchRate float64
	YawRate   float64

	// Acceleration (m/s²)
	AccelX float64
	AccelY float64
	AccelZ float64

	// Environment
	WindNorth     float64 // m/s
	WindEast      float64 // m/s
	Temperature   float64 // Celsius
	Pressure      float64 // hPa
	Density       float64 // kg/m³

	// Engine/Propulsion
	ThrottlePosition float64   // 0-1
	EngineRPM        []float64 // Per engine
	FuelRemaining    float64   // kg

	// Control surfaces
	Aileron  float64 // -1 to 1
	Elevator float64 // -1 to 1
	Rudder   float64 // -1 to 1
	Flaps    float64 // 0 to 1
}

// SimulatorCommand represents control inputs to the simulator
type SimulatorCommand struct {
	Timestamp time.Time

	// Flight controls
	Throttle float64 // 0-1
	Aileron  float64 // -1 to 1
	Elevator float64 // -1 to 1
	Rudder   float64 // -1 to 1
	Flaps    float64 // 0 to 1

	// Mode commands
	GearDown   bool
	BrakesOn   bool
	Autopilot  bool

	// Override flags
	OverrideControls bool
	OverrideEngine   bool
}

// Scenario represents a test scenario for simulation
type Scenario struct {
	ID          string
	Name        string
	Description string
	Category    ScenarioCategory

	// Initial conditions
	InitialPosition  [3]float64 // lat, lon, alt
	InitialVelocity  [3]float64 // N, E, D
	InitialAttitude  [3]float64 // roll, pitch, yaw
	InitialFuel      float64

	// Environment
	WindConditions   WindProfile
	WeatherPreset    string

	// Test parameters
	Duration         time.Duration
	PassCriteria     []PassCriterion
	FailCriteria     []FailCriterion

	// Actions
	Actions          []ScenarioAction
}

// ScenarioCategory groups scenarios by type
type ScenarioCategory string

const (
	CategoryNominal     ScenarioCategory = "nominal"
	CategoryDegraded    ScenarioCategory = "degraded"
	CategoryEmergency   ScenarioCategory = "emergency"
	CategoryFailure     ScenarioCategory = "failure"
	CategoryEthical     ScenarioCategory = "ethical"
	CategoryRescue      ScenarioCategory = "rescue"
	CategoryFormation   ScenarioCategory = "formation"
)

// WindProfile defines wind conditions
type WindProfile struct {
	BaseSpeed     float64 // m/s
	BaseDirection float64 // degrees true
	GustSpeed     float64 // m/s
	GustProbability float64 // 0-1
	Turbulence    TurbulenceLevel
}

// TurbulenceLevel defines turbulence intensity
type TurbulenceLevel int

const (
	TurbulenceNone TurbulenceLevel = iota
	TurbulenceLight
	TurbulenceModerate
	TurbulenceSevere
)

// PassCriterion defines when a test passes
type PassCriterion struct {
	Name      string
	Condition string // e.g., "altitude > 100 && pitch_stable"
	Value     float64
	Tolerance float64
}

// FailCriterion defines when a test fails
type FailCriterion struct {
	Name      string
	Condition string
	Value     float64
	Critical  bool // If true, test stops immediately
}

// ScenarioAction defines an event during simulation
type ScenarioAction struct {
	Time       time.Duration // Time from scenario start
	ActionType ActionType
	Target     string
	Value      interface{}
}

// ActionType defines scenario action types
type ActionType string

const (
	ActionInjectFault   ActionType = "inject_fault"
	ActionSetWind       ActionType = "set_wind"
	ActionSetPosition   ActionType = "set_position"
	ActionTriggerRescue ActionType = "trigger_rescue"
	ActionSensorFailure ActionType = "sensor_failure"
	ActionMotorFailure  ActionType = "motor_failure"
	ActionBatteryDrain  ActionType = "battery_drain"
)

// SimulationResult holds scenario execution results
type SimulationResult struct {
	ScenarioID    string
	StartTime     time.Time
	EndTime       time.Time
	Duration      time.Duration

	// Outcome
	Passed        bool
	PassedCriteria []string
	FailedCriteria []string
	ErrorMessage  string

	// Telemetry
	StateHistory  []SimulatorState
	CommandHistory []SimulatorCommand

	// Performance metrics
	AverageLatency time.Duration
	MaxLatency     time.Duration
	UpdateCount    int

	// DO-178C compliance data
	CoverageData  map[string]float64
	Deviations    []string
}

// Simulator is the main interface for flight simulators
type Simulator interface {
	// Lifecycle
	Connect(ctx context.Context) error
	Disconnect() error
	IsConnected() bool

	// State
	GetState() (*SimulatorState, error)
	SendCommand(cmd *SimulatorCommand) error

	// Scenario execution
	LoadScenario(scenario *Scenario) error
	RunScenario(ctx context.Context) (*SimulationResult, error)
	StopScenario() error

	// Time control
	Pause() error
	Resume() error
	SetTimeScale(scale float64) error

	// Sensor simulation
	GetSensorReading(sensorType fusion.SensorType) (*fusion.SensorReading, error)
	InjectSensorFault(sensorType fusion.SensorType, faultType string) error
}

// MonteCarloConfig configures Monte Carlo simulation runs
type MonteCarloConfig struct {
	NumIterations     int
	RandomSeed        int64
	ParameterRanges   map[string]ParameterRange
	ParallelWorkers   int
	ResultsPath       string
}

// ParameterRange defines randomization range for a parameter
type ParameterRange struct {
	Min          float64
	Max          float64
	Distribution string // "uniform", "normal", "triangular"
	Mean         float64 // For normal distribution
	StdDev       float64 // For normal distribution
}

// MonteCarloResult holds results from Monte Carlo analysis
type MonteCarloResult struct {
	Config         MonteCarloConfig
	TotalRuns      int
	SuccessfulRuns int
	FailedRuns     int
	SuccessRate    float64

	// Statistical analysis
	LatencyMean    time.Duration
	LatencyStdDev  time.Duration
	LatencyP95     time.Duration
	LatencyP99     time.Duration

	// Per-scenario results
	ScenarioResults map[string]*ScenarioStatistics

	// Ethical compliance
	EthicalViolations   int
	FirstLawViolations  int
	BiasDetections      int
}

// ScenarioStatistics holds statistics for a single scenario
type ScenarioStatistics struct {
	ScenarioID   string
	Runs         int
	Successes    int
	Failures     int
	SuccessRate  float64
	MeanDuration time.Duration
	StdDevDuration time.Duration
}
