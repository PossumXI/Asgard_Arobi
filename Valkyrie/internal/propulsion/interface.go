// Package propulsion provides propulsion system abstraction for VALKYRIE
// autonomous flight system. It supports multiple propulsion types including
// electric, combustion, turbine, hybrid, and rocket propulsion.
//
// Copyright 2026 Arobi. All Rights Reserved.
package propulsion

import (
	"context"
	"time"
)

// PropulsionType defines the propulsion system type
type PropulsionType int

const (
	PropulsionElectric PropulsionType = iota
	PropulsionCombustion
	PropulsionTurbine
	PropulsionHybrid
	PropulsionRocket
)

func (t PropulsionType) String() string {
	switch t {
	case PropulsionElectric:
		return "electric"
	case PropulsionCombustion:
		return "combustion"
	case PropulsionTurbine:
		return "turbine"
	case PropulsionHybrid:
		return "hybrid"
	case PropulsionRocket:
		return "rocket"
	default:
		return "unknown"
	}
}

// EnergyState represents the current energy status of the propulsion system
type EnergyState struct {
	// Battery state (for electric/hybrid)
	BatterySOC         float64   `json:"batterySOC"`         // State of charge (0.0-1.0)
	BatteryVoltage     float64   `json:"batteryVoltage"`     // Pack voltage (V)
	BatteryCurrent     float64   `json:"batteryCurrent"`     // Current draw (A)
	BatteryTemperature float64   `json:"batteryTemperature"` // Pack temperature (C)
	BatteryHealth      float64   `json:"batteryHealth"`      // Estimated health (0.0-1.0)
	CellVoltages       []float64 `json:"cellVoltages"`       // Individual cell voltages
	CellTemperatures   []float64 `json:"cellTemperatures"`   // Individual cell temperatures

	// Fuel state (for combustion/hybrid)
	FuelLevel    float64 `json:"fuelLevel"`    // Fuel remaining (0.0-1.0)
	FuelMassKg   float64 `json:"fuelMassKg"`   // Fuel mass in kg
	FuelFlowRate float64 `json:"fuelFlowRate"` // Current consumption (L/hr or kg/hr)

	// Derived values
	RemainingEnergy    float64       `json:"remainingEnergy"`    // Total remaining energy (Wh or J)
	EstimatedEndurance time.Duration `json:"estimatedEndurance"` // Estimated remaining flight time
	SpecificEnergy     float64       `json:"specificEnergy"`     // Energy per unit mass (Wh/kg)

	Timestamp  time.Time `json:"timestamp"`
	Confidence float64   `json:"confidence"` // Estimation confidence (0.0-1.0)
}

// ThermalState represents thermal conditions of propulsion components
type ThermalState struct {
	MotorTemperature   float64   `json:"motorTemperature"`   // Motor winding temperature (C)
	ESCTemperature     float64   `json:"escTemperature"`     // ESC temperature (C)
	BatteryTemperature float64   `json:"batteryTemperature"` // Battery temperature (C)
	EngineTemperature  float64   `json:"engineTemperature"`  // Engine temperature (for ICE)
	AmbientTemperature float64   `json:"ambientTemperature"` // Ambient air temperature (C)
	CoolingEfficiency  float64   `json:"coolingEfficiency"`  // Current cooling effectiveness (0.0-1.0)
	ThermalMargin      float64   `json:"thermalMargin"`      // Margin to thermal limits (0.0-1.0)
	Timestamp          time.Time `json:"timestamp"`
}

// ThrustCapability describes achievable thrust characteristics
type ThrustCapability struct {
	MaxThrust           float64       `json:"maxThrust"`           // Maximum available thrust (N)
	CurrentThrust       float64       `json:"currentThrust"`       // Current thrust output (N)
	ThrustVector        [3]float64    `json:"thrustVector"`        // Thrust direction vector
	ResponseTime        time.Duration `json:"responseTime"`        // Time to reach 90% of commanded thrust
	EfficiencyAtCurrent float64       `json:"efficiencyAtCurrent"` // Current operating efficiency
	SustainableDuration time.Duration `json:"sustainableDuration"` // How long max thrust is sustainable
}

// HealthStatus indicates the health state of a component
type HealthStatus int

const (
	HealthOK HealthStatus = iota
	HealthDegraded
	HealthCritical
	HealthFailed
)

func (h HealthStatus) String() string {
	switch h {
	case HealthOK:
		return "ok"
	case HealthDegraded:
		return "degraded"
	case HealthCritical:
		return "critical"
	case HealthFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// Fault represents a detected fault in the propulsion system
type Fault struct {
	ID          string    `json:"id"`
	Component   string    `json:"component"`
	Severity    float64   `json:"severity"` // 0.0-1.0
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
}

// PropulsionHealth represents overall system health status
type PropulsionHealth struct {
	OverallHealth    HealthStatus `json:"overallHealth"`
	MotorHealth      HealthStatus `json:"motorHealth"`
	BatteryHealth    HealthStatus `json:"batteryHealth"`
	ESCHealth        HealthStatus `json:"escHealth"`
	FuelSystemHealth HealthStatus `json:"fuelSystemHealth"`
	ThermalHealth    HealthStatus `json:"thermalHealth"`
	ActiveFaults     []Fault      `json:"activeFaults"`
	Timestamp        time.Time    `json:"timestamp"`
}

// PropulsionSystem is the main interface all propulsion types must implement
type PropulsionSystem interface {
	// Lifecycle management
	Initialize(ctx context.Context) error
	Start(ctx context.Context) error
	Stop() error

	// State queries
	GetEnergyState() *EnergyState
	GetThermalState() *ThermalState
	GetThrustCapability() *ThrustCapability
	GetHealth() *PropulsionHealth

	// Commands
	SetThrustCommand(thrust float64) error     // 0.0-1.0 normalized
	SetThrustVector(vector [3]float64) error   // For vectored thrust
	EmergencyShutdown() error

	// Predictions
	PredictEndurance(powerProfile []float64) time.Duration
	PredictThermalState(duration time.Duration) *ThermalState

	// Configuration
	GetType() PropulsionType
	GetConfig() interface{}
}

// EnergyObserver interface for EKF integration
type EnergyObserver interface {
	// GetMeasurement returns energy state as measurement vector with covariance
	GetMeasurement() (data []float64, covariance [][]float64)
	// GetMeasurementDimension returns the dimension of the measurement vector
	GetMeasurementDimension() int
}

// ThrustController interface for flight control integration
type ThrustController interface {
	// CommandThrust sets the desired thrust level (0.0-1.0)
	CommandThrust(level float64) error
	// CommandVector sets thrust vectoring for multi-rotor or vectored thrust
	CommandVector(roll, pitch, yaw, thrust float64) error
	// GetActualThrust returns the current actual thrust output
	GetActualThrust() float64
	// GetMaxThrust returns maximum available thrust at current conditions
	GetMaxThrust() float64
}

// PowerManager interface for energy management
type PowerManager interface {
	// GetAvailablePower returns power available for mission (above reserves)
	GetAvailablePower() float64
	// GetReserveLevel returns current reserve tier
	GetReserveLevel() ReserveLevel
	// RequestPower requests allocation of specified power
	RequestPower(watts float64) (granted float64, err error)
	// ReleasePower releases previously allocated power
	ReleasePower(watts float64)
}

// ReserveLevel defines energy reserve protection tiers
type ReserveLevel int

const (
	ReserveLevelMission     ReserveLevel = iota // Normal mission operations
	ReserveLevelContingency                     // Contingency operations
	ReserveLevelEmergency                       // Emergency landing only
	ReserveLevelAbsolute                        // Cannot be used - immediate landing
)

func (r ReserveLevel) String() string {
	switch r {
	case ReserveLevelMission:
		return "mission"
	case ReserveLevelContingency:
		return "contingency"
	case ReserveLevelEmergency:
		return "emergency"
	case ReserveLevelAbsolute:
		return "absolute"
	default:
		return "unknown"
	}
}
