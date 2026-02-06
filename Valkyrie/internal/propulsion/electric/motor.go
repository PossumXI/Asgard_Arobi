// Package electric provides motor modeling for brushless DC motors
// used in electric propulsion systems.
//
// Copyright 2026 Arobi. All Rights Reserved.
package electric

import (
	"math"
	"sync"
	"time"
)

// MotorType defines the motor type
type MotorType int

const (
	MotorBrushlessOutrunner MotorType = iota
	MotorBrushlessInrunner
	MotorBrushed
)

// MotorConfig holds motor configuration parameters
type MotorConfig struct {
	Type              MotorType         `yaml:"type"`
	KVRating          float64           `yaml:"kv_rating"`            // RPM per volt
	MaxPowerWatts     float64           `yaml:"max_power_watts"`      // Maximum power output
	PeakEfficiency    float64           `yaml:"efficiency_peak"`      // Peak efficiency (0.0-1.0)
	NoLoadCurrent     float64           `yaml:"no_load_current"`      // No-load current draw (A)
	WindingResistance float64           `yaml:"winding_resistance"`   // Phase resistance (Ohms)
	ThermalMass       float64           `yaml:"thermal_mass_j_per_k"` // Thermal mass (J/K)
	MaxWindingTempC   float64           `yaml:"max_winding_temp_c"`   // Max winding temperature
	CoolingCoeff      float64           `yaml:"cooling_coefficient"`  // Cooling effectiveness (W/K)
	EfficiencyMap     []EfficiencyPoint `yaml:"efficiency_map"`       // Load vs efficiency
}

// EfficiencyPoint represents a point on the efficiency curve
type EfficiencyPoint struct {
	LoadPercent float64 `yaml:"load_percent"` // 0.0-1.0
	Efficiency  float64 `yaml:"efficiency"`   // 0.0-1.0
}

// DefaultMotorConfig returns a default brushless outrunner configuration
func DefaultMotorConfig() MotorConfig {
	return MotorConfig{
		Type:              MotorBrushlessOutrunner,
		KVRating:          1000,
		MaxPowerWatts:     2000,
		PeakEfficiency:    0.90,
		NoLoadCurrent:     1.5,
		WindingResistance: 0.015,
		ThermalMass:       50.0,
		MaxWindingTempC:   120.0,
		CoolingCoeff:      2.5,
		EfficiencyMap: []EfficiencyPoint{
			{0.1, 0.60}, {0.3, 0.80}, {0.5, 0.88},
			{0.7, 0.90}, {0.9, 0.87}, {1.0, 0.82},
		},
	}
}

// MotorState holds real-time motor state
type MotorState struct {
	RPM             float64
	Torque          float64 // Nm
	Current         float64 // A
	Power           float64 // W
	Efficiency      float64 // 0.0-1.0
	WindingTemp     float64 // C
	ThrottleCommand float64 // 0.0-1.0
	Timestamp       time.Time
}

// MotorModel provides motor modeling and state estimation
type MotorModel struct {
	mu sync.RWMutex

	config MotorConfig
	state  *MotorState

	// Interpolated efficiency curve
	efficiencyCurve []EfficiencyPoint
}

// NewMotorModel creates a new motor model
func NewMotorModel(config MotorConfig) *MotorModel {
	mm := &MotorModel{
		config: config,
		state: &MotorState{
			WindingTemp: 25.0, // Room temperature
			Timestamp:   time.Now(),
		},
		efficiencyCurve: config.EfficiencyMap,
	}

	// Use default efficiency map if not provided
	if len(mm.efficiencyCurve) == 0 {
		mm.efficiencyCurve = DefaultMotorConfig().EfficiencyMap
	}

	return mm
}

// Update updates motor state with new measurements
func (mm *MotorModel) Update(voltage, throttle float64, ambientTemp float64, airspeed float64, dt float64) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.state.ThrottleCommand = throttle

	// Calculate RPM from voltage and KV rating
	// RPM = KV * Voltage * (throttle factor)
	mm.state.RPM = mm.config.KVRating * voltage * throttle

	// Calculate current from power model
	powerDemand := throttle * mm.config.MaxPowerWatts
	efficiency := mm.getEfficiency(throttle)
	mm.state.Efficiency = efficiency

	// Electrical power needed for mechanical output
	electricalPower := powerDemand / efficiency
	mm.state.Current = electricalPower / voltage

	// Add no-load current
	mm.state.Current += mm.config.NoLoadCurrent * (1 - throttle*0.5)

	// Calculate actual mechanical power
	mm.state.Power = powerDemand

	// Calculate torque from power and RPM
	// P = T * omega, omega = RPM * 2*pi/60
	if mm.state.RPM > 0 {
		omega := mm.state.RPM * 2 * math.Pi / 60
		mm.state.Torque = mm.state.Power / omega
	} else {
		mm.state.Torque = 0
	}

	// Thermal model
	// Heat generated from losses
	losses := electricalPower - mm.state.Power
	// Additional I²R losses in windings
	losses += mm.state.Current * mm.state.Current * mm.config.WindingResistance

	// Cooling from airflow
	coolingPower := mm.config.CoolingCoeff * (1 + airspeed*0.1) * (mm.state.WindingTemp - ambientTemp)

	// Temperature change
	tempChange := (losses - coolingPower) * dt / mm.config.ThermalMass
	mm.state.WindingTemp += tempChange

	// Clamp to reasonable values
	mm.state.WindingTemp = math.Max(ambientTemp, mm.state.WindingTemp)

	mm.state.Timestamp = time.Now()
}

// getEfficiency returns efficiency at given throttle/load
func (mm *MotorModel) getEfficiency(load float64) float64 {
	if len(mm.efficiencyCurve) == 0 {
		return mm.config.PeakEfficiency
	}

	// Linear interpolation on efficiency curve
	for i := 0; i < len(mm.efficiencyCurve)-1; i++ {
		if load >= mm.efficiencyCurve[i].LoadPercent && load <= mm.efficiencyCurve[i+1].LoadPercent {
			loadRange := mm.efficiencyCurve[i+1].LoadPercent - mm.efficiencyCurve[i].LoadPercent
			effRange := mm.efficiencyCurve[i+1].Efficiency - mm.efficiencyCurve[i].Efficiency
			return mm.efficiencyCurve[i].Efficiency + (load-mm.efficiencyCurve[i].LoadPercent)/loadRange*effRange
		}
	}

	// Return nearest endpoint
	if load < mm.efficiencyCurve[0].LoadPercent {
		return mm.efficiencyCurve[0].Efficiency
	}
	return mm.efficiencyCurve[len(mm.efficiencyCurve)-1].Efficiency
}

// GetState returns a copy of the current motor state
func (mm *MotorModel) GetState() MotorState {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return *mm.state
}

// GetThrust estimates thrust output (simplified propeller model)
// thrust = Ct * rho * n² * D⁴, simplified to proportional to RPM²
func (mm *MotorModel) GetThrust(propDiameter float64, airDensity float64) float64 {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	// Simplified thrust coefficient
	Ct := 0.1 // Typical propeller thrust coefficient

	// n = RPM / 60 (revolutions per second)
	n := mm.state.RPM / 60

	// Thrust = Ct * rho * n² * D⁴
	thrust := Ct * airDensity * n * n * math.Pow(propDiameter, 4)

	return thrust
}

// IsThermalOK checks if motor is within thermal limits
func (mm *MotorModel) IsThermalOK() bool {
	mm.mu.RLock()
	defer mm.mu.RUnlock()
	return mm.state.WindingTemp < mm.config.MaxWindingTempC
}

// GetThermalMargin returns margin to thermal limit (0.0-1.0)
func (mm *MotorModel) GetThermalMargin() float64 {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	margin := (mm.config.MaxWindingTempC - mm.state.WindingTemp) / (mm.config.MaxWindingTempC - 25.0)
	return math.Max(0, math.Min(1, margin))
}

// GetMaxThrottle returns maximum safe throttle based on thermal state
func (mm *MotorModel) GetMaxThrottle() float64 {
	margin := mm.GetThermalMargin()

	// If margin is low, reduce max throttle
	if margin < 0.2 {
		return 0.5 // Limit to 50% if near thermal limit
	} else if margin < 0.4 {
		return 0.75 // Limit to 75%
	}
	return 1.0
}

// GetConfig returns the motor configuration
func (mm *MotorModel) GetConfig() MotorConfig {
	return mm.config
}
