// Package energy provides energy management for propulsion systems
// including reserve protection, optimization, and prediction.
//
// Copyright 2026 Arobi. All Rights Reserved.
package energy

import (
	"sync"
	"time"

	"github.com/PossumXI/Asgard/Valkyrie/internal/propulsion"
)

// ReserveConfig defines reserve thresholds for energy protection
type ReserveConfig struct {
	// Battery SOC thresholds
	MissionBatterySOC     float64 `yaml:"mission_battery_soc"`     // Below this, mission objectives suspended
	ContingencyBatterySOC float64 `yaml:"contingency_battery_soc"` // Below this, RTB required
	EmergencyBatterySOC   float64 `yaml:"emergency_battery_soc"`   // Below this, nearest landing
	AbsoluteBatterySOC    float64 `yaml:"absolute_battery_soc"`    // Below this, immediate autoland

	// Fuel level thresholds (for hybrid/combustion)
	MissionFuelLevel     float64 `yaml:"mission_fuel_level"`
	ContingencyFuelLevel float64 `yaml:"contingency_fuel_level"`
	EmergencyFuelLevel   float64 `yaml:"emergency_fuel_level"`
	AbsoluteFuelLevel    float64 `yaml:"absolute_fuel_level"`

	// Time-based reserves
	MissionReserveMinutes     float64 `yaml:"mission_reserve_minutes"` // Minutes of flight at cruise
	ContingencyReserveMinutes float64 `yaml:"contingency_reserve_minutes"`
	EmergencyReserveMinutes   float64 `yaml:"emergency_reserve_minutes"`
}

// DefaultReserveConfig returns default reserve configuration
func DefaultReserveConfig() ReserveConfig {
	return ReserveConfig{
		MissionBatterySOC:         0.40,
		ContingencyBatterySOC:     0.30,
		EmergencyBatterySOC:       0.20,
		AbsoluteBatterySOC:        0.10,
		MissionFuelLevel:          0.40,
		ContingencyFuelLevel:      0.30,
		EmergencyFuelLevel:        0.20,
		AbsoluteFuelLevel:         0.10,
		MissionReserveMinutes:     10.0,
		ContingencyReserveMinutes: 5.0,
		EmergencyReserveMinutes:   2.0,
	}
}

// ReserveAction defines actions triggered by reserve levels
type ActionType int

const (
	ActionWarnOperator ActionType = iota
	ActionAbortMission
	ActionReturnToBase
	ActionReducePower
	ActionFindNearestLanding
	ActionImmediateLanding
)

func (a ActionType) String() string {
	switch a {
	case ActionWarnOperator:
		return "warn_operator"
	case ActionAbortMission:
		return "abort_mission"
	case ActionReturnToBase:
		return "return_to_base"
	case ActionReducePower:
		return "reduce_power"
	case ActionFindNearestLanding:
		return "find_nearest_landing"
	case ActionImmediateLanding:
		return "immediate_landing"
	default:
		return "unknown"
	}
}

// ReserveAction represents an action triggered by reserve level
type ReserveAction struct {
	Priority  int        `json:"priority"`
	Action    ActionType `json:"action"`
	Mandatory bool       `json:"mandatory"`
}

// ReserveManager manages tiered energy reserves
type ReserveManager struct {
	mu sync.RWMutex

	config       ReserveConfig
	currentLevel propulsion.ReserveLevel

	// Energy source accessors
	getBatterySOC func() float64
	getFuelLevel  func() float64

	// Callbacks for level changes
	onLevelChange func(old, new propulsion.ReserveLevel)

	// Predictions
	cruisePowerWatts float64
	distanceToHome   float64

	lastCheck time.Time
}

// NewReserveManager creates a new reserve manager
func NewReserveManager(config ReserveConfig) *ReserveManager {
	return &ReserveManager{
		config:       config,
		currentLevel: propulsion.ReserveLevelMission,
		lastCheck:    time.Now(),
	}
}

// SetBatterySOCSource sets the function to get battery SOC
func (rm *ReserveManager) SetBatterySOCSource(fn func() float64) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.getBatterySOC = fn
}

// SetFuelLevelSource sets the function to get fuel level
func (rm *ReserveManager) SetFuelLevelSource(fn func() float64) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.getFuelLevel = fn
}

// SetLevelChangeCallback sets the callback for level changes
func (rm *ReserveManager) SetLevelChangeCallback(fn func(old, new propulsion.ReserveLevel)) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.onLevelChange = fn
}

// Check evaluates current energy against reserves
func (rm *ReserveManager) Check() propulsion.ReserveLevel {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	oldLevel := rm.currentLevel

	// Check battery SOC if source is available
	if rm.getBatterySOC != nil {
		soc := rm.getBatterySOC()
		rm.currentLevel = rm.socToLevel(soc, rm.config.AbsoluteBatterySOC,
			rm.config.EmergencyBatterySOC, rm.config.ContingencyBatterySOC,
			rm.config.MissionBatterySOC)
	}

	// Check fuel level if source is available (use worst case)
	if rm.getFuelLevel != nil {
		fuel := rm.getFuelLevel()
		fuelLevel := rm.socToLevel(fuel, rm.config.AbsoluteFuelLevel,
			rm.config.EmergencyFuelLevel, rm.config.ContingencyFuelLevel,
			rm.config.MissionFuelLevel)

		// Use worst case between battery and fuel
		if fuelLevel > rm.currentLevel {
			rm.currentLevel = fuelLevel
		}
	}

	// Trigger callback if level changed
	if rm.currentLevel != oldLevel && rm.onLevelChange != nil {
		rm.onLevelChange(oldLevel, rm.currentLevel)
	}

	rm.lastCheck = time.Now()
	return rm.currentLevel
}

// socToLevel converts SOC/fuel level to reserve level
func (rm *ReserveManager) socToLevel(value, absolute, emergency, contingency, mission float64) propulsion.ReserveLevel {
	switch {
	case value <= absolute:
		return propulsion.ReserveLevelAbsolute
	case value <= emergency:
		return propulsion.ReserveLevelEmergency
	case value <= contingency:
		return propulsion.ReserveLevelContingency
	default:
		return propulsion.ReserveLevelMission
	}
}

// GetCurrentLevel returns the current reserve level
func (rm *ReserveManager) GetCurrentLevel() propulsion.ReserveLevel {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.currentLevel
}

// GetAvailableEnergy returns energy available for mission use
func (rm *ReserveManager) GetAvailableEnergy(totalEnergy float64) float64 {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if rm.getBatterySOC == nil {
		return 0
	}

	soc := rm.getBatterySOC()

	// Calculate usable SOC above contingency reserve
	usableSOC := soc - rm.config.ContingencyBatterySOC
	if usableSOC < 0 {
		return 0
	}

	return usableSOC * totalEnergy / soc
}

// GetReserveActions returns required actions for current reserve level
func (rm *ReserveManager) GetReserveActions() []ReserveAction {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	switch rm.currentLevel {
	case propulsion.ReserveLevelAbsolute:
		return []ReserveAction{
			{Priority: 1, Action: ActionImmediateLanding, Mandatory: true},
		}
	case propulsion.ReserveLevelEmergency:
		return []ReserveAction{
			{Priority: 1, Action: ActionFindNearestLanding, Mandatory: true},
			{Priority: 2, Action: ActionReducePower, Mandatory: true},
		}
	case propulsion.ReserveLevelContingency:
		return []ReserveAction{
			{Priority: 1, Action: ActionReturnToBase, Mandatory: true},
			{Priority: 2, Action: ActionAbortMission, Mandatory: true},
		}
	case propulsion.ReserveLevelMission:
		if rm.getBatterySOC != nil {
			soc := rm.getBatterySOC()
			if soc <= rm.config.MissionBatterySOC+0.05 {
				return []ReserveAction{
					{Priority: 3, Action: ActionWarnOperator, Mandatory: false},
				}
			}
		}
		return nil
	}
	return nil
}

// IsOperationAllowed checks if a power-consuming operation is allowed
func (rm *ReserveManager) IsOperationAllowed(requiredEnergy float64, totalEnergy float64) bool {
	availableEnergy := rm.GetAvailableEnergy(totalEnergy)
	return requiredEnergy <= availableEnergy
}

// GetConfig returns the reserve configuration
func (rm *ReserveManager) GetConfig() ReserveConfig {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.config
}
