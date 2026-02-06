// Package electric provides electric propulsion system implementation
// including battery modeling, motor control, and ESC integration.
//
// Copyright 2026 Arobi. All Rights Reserved.
package electric

import (
	"math"
	"sync"
	"time"
)

// BatteryChemistry defines battery chemistry types
type BatteryChemistry int

const (
	ChemistryLiPo BatteryChemistry = iota // Lithium Polymer
	ChemistryLiFe                         // Lithium Iron Phosphate
	ChemistryLiIon                        // Lithium Ion
	ChemistryLiHV                         // High Voltage LiPo
)

func (c BatteryChemistry) String() string {
	switch c {
	case ChemistryLiPo:
		return "lipo"
	case ChemistryLiFe:
		return "life"
	case ChemistryLiIon:
		return "lion"
	case ChemistryLiHV:
		return "lihv"
	default:
		return "unknown"
	}
}

// BatteryConfig holds battery configuration parameters
type BatteryConfig struct {
	Chemistry             BatteryChemistry `yaml:"chemistry"`
	CellCount             int              `yaml:"cell_count"`
	ParallelCount         int              `yaml:"parallel_count"`
	NominalCapacityAh     float64          `yaml:"nominal_capacity_ah"`
	NominalVoltagePerCell float64          `yaml:"nominal_voltage_per_cell"`
	MaxVoltagePerCell     float64          `yaml:"max_voltage_per_cell"`
	MinVoltagePerCell     float64          `yaml:"min_voltage_per_cell"`
	CutoffVoltagePerCell  float64          `yaml:"cutoff_voltage_per_cell"`
	MaxDischargeC         float64          `yaml:"max_discharge_c"`
	MaxChargeC            float64          `yaml:"max_charge_c"`
	InternalResistance    float64          `yaml:"internal_resistance_ohms"`
	ThermalMass           float64          `yaml:"thermal_mass_j_per_k"`
	CoolingCoefficient    float64          `yaml:"cooling_coefficient"`
	MaxChargeTempC        float64          `yaml:"max_charge_temp_c"`
	MaxDischargeTempC     float64          `yaml:"max_discharge_temp_c"`
	MinOperatingTempC     float64          `yaml:"min_operating_temp_c"`
}

// DefaultBatteryConfig returns a default 6S LiPo configuration
func DefaultBatteryConfig() BatteryConfig {
	return BatteryConfig{
		Chemistry:             ChemistryLiPo,
		CellCount:             6,
		ParallelCount:         2,
		NominalCapacityAh:     5.0,
		NominalVoltagePerCell: 3.70,
		MaxVoltagePerCell:     4.20,
		MinVoltagePerCell:     3.30,
		CutoffVoltagePerCell:  3.00,
		MaxDischargeC:         25.0,
		MaxChargeC:            2.0,
		InternalResistance:    0.015,
		ThermalMass:           50.0,
		CoolingCoefficient:    2.5,
		MaxChargeTempC:        45.0,
		MaxDischargeTempC:     60.0,
		MinOperatingTempC:     0.0,
	}
}

// BatteryState holds real-time battery state
type BatteryState struct {
	PackVoltage      float64
	PackCurrent      float64
	CellVoltages     []float64
	CellTemperatures []float64
	SOC              float64 // State of charge (0.0-1.0)
	SOH              float64 // State of health (0.0-1.0)
	UsableCapacity   float64 // Current usable capacity (Ah)
	EnergyUsed       float64 // Energy consumed since last reset (Wh)
	CoulombCount     float64 // Coulombs consumed
	Timestamp        time.Time
}

// VoltageSOCPoint represents a point on the discharge curve
type VoltageSOCPoint struct {
	SOC     float64
	Voltage float64
}

// BatteryModel provides battery modeling and state estimation
type BatteryModel struct {
	mu sync.RWMutex

	config BatteryConfig
	state  *BatteryState

	// Discharge curves (voltage vs SOC at different C-rates)
	dischargeCurves map[float64][]VoltageSOCPoint

	// Temperature coefficients
	tempCoeffCapacity    float64 // Capacity reduction per degree C
	tempCoeffResistance  float64 // Resistance increase per degree C

	// Aging model
	cycleCount  int
	calendarAge time.Duration
}

// NewBatteryModel creates a new battery model with the given configuration
func NewBatteryModel(config BatteryConfig) *BatteryModel {
	bm := &BatteryModel{
		config: config,
		state: &BatteryState{
			CellVoltages:     make([]float64, config.CellCount),
			CellTemperatures: make([]float64, config.CellCount),
			SOC:              1.0,
			SOH:              1.0,
			UsableCapacity:   config.NominalCapacityAh * float64(config.ParallelCount),
			Timestamp:        time.Now(),
		},
		dischargeCurves:     make(map[float64][]VoltageSOCPoint),
		tempCoeffCapacity:   0.002,  // 0.2% capacity loss per degree C below 25C
		tempCoeffResistance: 0.005,  // 0.5% resistance increase per degree C
	}

	// Initialize default discharge curves based on chemistry
	bm.initializeDischargeCurves()

	// Initialize cell voltages to nominal
	for i := range bm.state.CellVoltages {
		bm.state.CellVoltages[i] = config.MaxVoltagePerCell
		bm.state.CellTemperatures[i] = 25.0 // Room temperature
	}
	bm.state.PackVoltage = config.MaxVoltagePerCell * float64(config.CellCount)

	return bm
}

// initializeDischargeCurves sets up chemistry-specific discharge curves
func (bm *BatteryModel) initializeDischargeCurves() {
	switch bm.config.Chemistry {
	case ChemistryLiPo:
		// LiPo discharge curve (1C rate)
		bm.dischargeCurves[1.0] = []VoltageSOCPoint{
			{1.0, 4.20}, {0.9, 4.08}, {0.8, 3.96}, {0.7, 3.87},
			{0.6, 3.80}, {0.5, 3.73}, {0.4, 3.70}, {0.3, 3.65},
			{0.2, 3.55}, {0.1, 3.40}, {0.0, 3.00},
		}
		// Higher C-rate curve (voltage sag)
		bm.dischargeCurves[5.0] = []VoltageSOCPoint{
			{1.0, 4.10}, {0.9, 3.98}, {0.8, 3.86}, {0.7, 3.77},
			{0.6, 3.70}, {0.5, 3.63}, {0.4, 3.60}, {0.3, 3.55},
			{0.2, 3.45}, {0.1, 3.30}, {0.0, 2.90},
		}
	case ChemistryLiFe:
		// LiFePO4 discharge curve (1C rate) - flatter plateau
		bm.dischargeCurves[1.0] = []VoltageSOCPoint{
			{1.0, 3.60}, {0.9, 3.35}, {0.8, 3.30}, {0.7, 3.28},
			{0.6, 3.26}, {0.5, 3.25}, {0.4, 3.24}, {0.3, 3.22},
			{0.2, 3.18}, {0.1, 3.10}, {0.0, 2.50},
		}
	case ChemistryLiIon:
		// Li-Ion discharge curve
		bm.dischargeCurves[1.0] = []VoltageSOCPoint{
			{1.0, 4.20}, {0.9, 4.06}, {0.8, 3.95}, {0.7, 3.85},
			{0.6, 3.77}, {0.5, 3.71}, {0.4, 3.67}, {0.3, 3.61},
			{0.2, 3.50}, {0.1, 3.35}, {0.0, 3.00},
		}
	case ChemistryLiHV:
		// High Voltage LiPo (4.35V max)
		bm.dischargeCurves[1.0] = []VoltageSOCPoint{
			{1.0, 4.35}, {0.9, 4.22}, {0.8, 4.08}, {0.7, 3.97},
			{0.6, 3.88}, {0.5, 3.80}, {0.4, 3.75}, {0.3, 3.68},
			{0.2, 3.58}, {0.1, 3.42}, {0.0, 3.00},
		}
	}
}

// Update updates battery state with new measurements
func (bm *BatteryModel) Update(voltage, current float64, cellTemps []float64) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	dt := time.Since(bm.state.Timestamp).Seconds()
	if dt <= 0 {
		dt = 0.01 // Minimum dt
	}

	// Update measurements
	bm.state.PackVoltage = voltage
	bm.state.PackCurrent = current
	if len(cellTemps) == len(bm.state.CellTemperatures) {
		copy(bm.state.CellTemperatures, cellTemps)
	}

	// Calculate cell voltage from pack voltage
	cellVoltage := voltage / float64(bm.config.CellCount)
	for i := range bm.state.CellVoltages {
		bm.state.CellVoltages[i] = cellVoltage
	}

	// Coulomb counting for SOC estimation
	coulombs := current * dt
	bm.state.CoulombCount += coulombs

	capacityAh := bm.config.NominalCapacityAh * float64(bm.config.ParallelCount)
	usableCoulombs := capacityAh * 3600 * bm.state.SOH

	// SOC from coulomb counting
	socCoulomb := bm.state.SOC - (coulombs / usableCoulombs)

	// SOC from voltage (for correction)
	cRate := math.Abs(current) / capacityAh
	socVoltage := bm.voltageToSOC(cellVoltage, cRate)

	// Fuse SOC estimates (weighted by confidence)
	if current == 0 {
		// At rest, voltage is more accurate
		bm.state.SOC = 0.7*socVoltage + 0.3*socCoulomb
	} else {
		// Under load, coulomb counting is more accurate
		bm.state.SOC = 0.3*socVoltage + 0.7*socCoulomb
	}

	// Clamp SOC
	bm.state.SOC = math.Max(0, math.Min(1, bm.state.SOC))

	// Update energy accounting
	power := voltage * current
	bm.state.EnergyUsed += power * dt / 3600 // Convert to Wh

	bm.state.Timestamp = time.Now()
}

// voltageToSOC converts cell voltage to SOC using discharge curves
func (bm *BatteryModel) voltageToSOC(voltage, cRate float64) float64 {
	// Select appropriate curve based on C-rate
	curve, ok := bm.dischargeCurves[1.0]
	if cRate >= 3.0 {
		if hcCurve, ok := bm.dischargeCurves[5.0]; ok {
			curve = hcCurve
		}
	}

	if !ok || len(curve) < 2 {
		// Fallback linear estimation
		maxV := bm.config.MaxVoltagePerCell
		minV := bm.config.MinVoltagePerCell
		return (voltage - minV) / (maxV - minV)
	}

	// Linear interpolation on discharge curve
	for i := 0; i < len(curve)-1; i++ {
		if voltage <= curve[i].Voltage && voltage >= curve[i+1].Voltage {
			vRange := curve[i].Voltage - curve[i+1].Voltage
			socRange := curve[i].SOC - curve[i+1].SOC
			return curve[i+1].SOC + (voltage-curve[i+1].Voltage)/vRange*socRange
		}
	}

	if voltage >= curve[0].Voltage {
		return 1.0
	}
	return 0.0
}

// GetState returns a copy of the current battery state
func (bm *BatteryModel) GetState() BatteryState {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	state := *bm.state
	state.CellVoltages = make([]float64, len(bm.state.CellVoltages))
	copy(state.CellVoltages, bm.state.CellVoltages)
	state.CellTemperatures = make([]float64, len(bm.state.CellTemperatures))
	copy(state.CellTemperatures, bm.state.CellTemperatures)

	return state
}

// GetSOC returns the current state of charge
func (bm *BatteryModel) GetSOC() float64 {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	return bm.state.SOC
}

// GetRemainingEnergy returns remaining energy in Wh
func (bm *BatteryModel) GetRemainingEnergy() float64 {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	capacityAh := bm.config.NominalCapacityAh * float64(bm.config.ParallelCount)
	nominalVoltage := bm.config.NominalVoltagePerCell * float64(bm.config.CellCount)
	totalEnergy := capacityAh * nominalVoltage // Wh

	return totalEnergy * bm.state.SOC * bm.state.SOH
}

// GetMaxDischargePower returns maximum discharge power at current conditions
func (bm *BatteryModel) GetMaxDischargePower() float64 {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	capacityAh := bm.config.NominalCapacityAh * float64(bm.config.ParallelCount)
	maxCurrent := capacityAh * bm.config.MaxDischargeC
	return bm.state.PackVoltage * maxCurrent
}

// PredictEndurance predicts remaining flight time at given power draw
func (bm *BatteryModel) PredictEndurance(powerWatts float64) time.Duration {
	remainingEnergy := bm.GetRemainingEnergy()
	if powerWatts <= 0 {
		return time.Duration(math.MaxInt64)
	}
	hours := remainingEnergy / powerWatts
	return time.Duration(hours * float64(time.Hour))
}

// GetAverageTemperature returns average cell temperature
func (bm *BatteryModel) GetAverageTemperature() float64 {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	sum := 0.0
	for _, t := range bm.state.CellTemperatures {
		sum += t
	}
	return sum / float64(len(bm.state.CellTemperatures))
}

// IsThermalOK checks if battery is within thermal limits
func (bm *BatteryModel) IsThermalOK() bool {
	avgTemp := bm.GetAverageTemperature()
	if bm.state.PackCurrent > 0 {
		// Charging
		return avgTemp >= bm.config.MinOperatingTempC && avgTemp <= bm.config.MaxChargeTempC
	}
	// Discharging
	return avgTemp >= bm.config.MinOperatingTempC && avgTemp <= bm.config.MaxDischargeTempC
}

// GetConfig returns the battery configuration
func (bm *BatteryModel) GetConfig() BatteryConfig {
	return bm.config
}
