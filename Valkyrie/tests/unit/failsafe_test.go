package unit

import (
	"testing"
	"time"

	"github.com/PossumXI/Asgard/Valkyrie/internal/failsafe"
)

func TestEmergencySystem_Creation(t *testing.T) {
	config := failsafe.FailsafeConfig{
		EnableAutoRTB:       true,
		EnableAutoLand:      true,
		EnableParachute:     false,
		MinSafeAltitudeAGL:  50.0,
		MinSafeFuel:         0.15,
		MinSafeBattery:      0.20,
		MaxTimeWithoutComms: 5 * time.Minute,
		RTBLocation:         [3]float64{0, 0, 500},
		CheckInterval:       100 * time.Millisecond,
	}

	es := failsafe.NewEmergencySystem(config)
	if es == nil {
		t.Fatal("EmergencySystem creation failed")
	}
}

func TestEmergencySystem_InitialHealth(t *testing.T) {
	config := failsafe.FailsafeConfig{
		MinSafeFuel:    0.15,
		MinSafeBattery: 0.20,
	}

	es := failsafe.NewEmergencySystem(config)

	if !es.IsHealthy() {
		t.Error("Initial health should be true")
	}
}

func TestEmergencySystem_GetMode(t *testing.T) {
	config := failsafe.FailsafeConfig{}
	es := failsafe.NewEmergencySystem(config)

	mode := es.GetMode()
	if mode != failsafe.ModePrimary {
		t.Errorf("Initial mode should be Primary, got %v", mode)
	}
}

func TestEmergencySystem_UpdateFuel(t *testing.T) {
	config := failsafe.FailsafeConfig{
		MinSafeFuel: 0.15,
	}

	es := failsafe.NewEmergencySystem(config)

	// Normal fuel
	es.UpdateFuel(0.80)
	if !es.IsHealthy() {
		t.Error("Should be healthy with 80% fuel")
	}

	// Low fuel - triggers emergency
	es.UpdateFuel(0.10)
	// Note: health check happens in Monitor(), not immediately
}

func TestEmergencySystem_UpdateBattery(t *testing.T) {
	config := failsafe.FailsafeConfig{
		MinSafeBattery: 0.20,
	}

	es := failsafe.NewEmergencySystem(config)

	// Normal battery
	es.UpdateBattery(0.90)
	if !es.IsHealthy() {
		t.Error("Should be healthy with 90% battery")
	}
}

func TestEmergencySystem_UpdateHealth(t *testing.T) {
	config := failsafe.FailsafeConfig{}
	es := failsafe.NewEmergencySystem(config)

	// Update various health statuses
	es.UpdateHealth("primary_flight", failsafe.HealthOK)
	es.UpdateHealth("gps", failsafe.HealthOK)
	es.UpdateHealth("ins", failsafe.HealthOK)
	es.UpdateHealth("comm", failsafe.HealthOK)

	if !es.IsHealthy() {
		t.Error("Should be healthy with all systems OK")
	}
}

func TestEmergencySystem_GetActiveEmergencies(t *testing.T) {
	config := failsafe.FailsafeConfig{}
	es := failsafe.NewEmergencySystem(config)

	emergencies := es.GetActiveEmergencies()
	if len(emergencies) != 0 {
		t.Errorf("Expected 0 active emergencies, got %d", len(emergencies))
	}
}

func TestHealthStatus_Values(t *testing.T) {
	statuses := []failsafe.HealthStatus{
		failsafe.HealthOK,
		failsafe.HealthDegraded,
		failsafe.HealthCritical,
		failsafe.HealthFailed,
	}

	for i, status := range statuses {
		if int(status) != i {
			t.Errorf("HealthStatus %d has wrong value %d", i, status)
		}
	}
}

func TestHealthStatus_String(t *testing.T) {
	tests := []struct {
		status   failsafe.HealthStatus
		expected string
	}{
		{failsafe.HealthOK, "OK"},
		{failsafe.HealthDegraded, "Degraded"},
		{failsafe.HealthCritical, "Critical"},
		{failsafe.HealthFailed, "Failed"},
	}

	for _, tc := range tests {
		result := tc.status.String()
		if result != tc.expected {
			t.Errorf("HealthStatus %d: expected %q, got %q", tc.status, tc.expected, result)
		}
	}
}

func TestFlightMode_Values(t *testing.T) {
	modes := []failsafe.FlightMode{
		failsafe.ModePrimary,
		failsafe.ModeBackup,
		failsafe.ModeEmergency,
		failsafe.ModeManual,
	}

	for i, mode := range modes {
		if int(mode) != i {
			t.Errorf("FlightMode %d has wrong value %d", i, mode)
		}
	}
}

func TestFlightMode_String(t *testing.T) {
	tests := []struct {
		mode     failsafe.FlightMode
		expected string
	}{
		{failsafe.ModePrimary, "Primary"},
		{failsafe.ModeBackup, "Backup"},
		{failsafe.ModeEmergency, "Emergency"},
		{failsafe.ModeManual, "Manual"},
	}

	for _, tc := range tests {
		result := tc.mode.String()
		if result != tc.expected {
			t.Errorf("FlightMode %d: expected %q, got %q", tc.mode, tc.expected, result)
		}
	}
}

func TestEmergencyType_Values(t *testing.T) {
	types := []failsafe.EmergencyType{
		failsafe.EmergencyEngineFailure,
		failsafe.EmergencyElectricalFailure,
		failsafe.EmergencyHydraulicFailure,
		failsafe.EmergencyStructuralDamage,
		failsafe.EmergencyWeatherSevere,
		failsafe.EmergencyThreatInbound,
		failsafe.EmergencyFuelCritical,
		failsafe.EmergencySensorFailure,
		failsafe.EmergencyCommunicationLoss,
		failsafe.EmergencyLowBattery,
	}

	for i, et := range types {
		if int(et) != i {
			t.Errorf("EmergencyType %d has wrong value %d", i, et)
		}
	}
}

func TestEmergencyType_String(t *testing.T) {
	tests := []struct {
		et       failsafe.EmergencyType
		expected string
	}{
		{failsafe.EmergencyEngineFailure, "Engine Failure"},
		{failsafe.EmergencyFuelCritical, "Fuel Critical"},
		{failsafe.EmergencyCommunicationLoss, "Communication Loss"},
	}

	for _, tc := range tests {
		result := tc.et.String()
		if result != tc.expected {
			t.Errorf("EmergencyType %d: expected %q, got %q", tc.et, tc.expected, result)
		}
	}
}
