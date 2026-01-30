package unit

import (
	"testing"
	"time"

	"github.com/PossumXI/Asgard/Valkyrie/internal/security"
)

func TestShadowMonitor_Creation(t *testing.T) {
	config := security.ShadowConfig{
		MonitorFlightController: true,
		MonitorSensorDrivers:    true,
		MonitorNavigation:       true,
		MonitorCommunication:    true,
		AnomalyThreshold:        0.7,
		ResponseMode:            security.ResponseModeAlert,
		ScanInterval:            100 * time.Millisecond,
	}

	sm := security.NewShadowMonitor(config)
	if sm == nil {
		t.Fatal("ShadowMonitor creation failed")
	}
}

func TestShadowMonitor_InitialHealth(t *testing.T) {
	config := security.ShadowConfig{
		MonitorFlightController: true,
		MonitorSensorDrivers:    true,
		AnomalyThreshold:        0.7,
		ResponseMode:            security.ResponseModeLog,
	}

	sm := security.NewShadowMonitor(config)

	if !sm.IsHealthy() {
		t.Error("Initial health should be healthy")
	}
}

func TestShadowMonitor_GetProcessStatus(t *testing.T) {
	config := security.ShadowConfig{
		MonitorFlightController: true,
		MonitorSensorDrivers:    true,
		MonitorNavigation:       true,
		MonitorCommunication:    true,
		AnomalyThreshold:        0.7,
		ResponseMode:            security.ResponseModeLog,
	}

	sm := security.NewShadowMonitor(config)
	status := sm.GetProcessStatus()

	// Should have entries for enabled monitors
	if len(status) == 0 {
		t.Error("Expected process status entries")
	}

	// All should be healthy initially
	for name, s := range status {
		if s != security.ProcessStatusHealthy {
			t.Errorf("Process %s should be healthy, got %v", name, s)
		}
	}
}

func TestShadowMonitor_GetStats(t *testing.T) {
	config := security.ShadowConfig{
		MonitorFlightController: true,
		AnomalyThreshold:        0.7,
		ResponseMode:            security.ResponseModeLog,
	}

	sm := security.NewShadowMonitor(config)

	scans, anomalies := sm.GetStats()

	// Initially should have 0 scans and 0 anomalies
	if scans != 0 {
		t.Errorf("Expected 0 scans initially, got %d", scans)
	}

	if anomalies != 0 {
		t.Errorf("Expected 0 anomalies initially, got %d", anomalies)
	}
}

func TestResponseMode_Values(t *testing.T) {
	modes := []security.ResponseMode{
		security.ResponseModeLog,
		security.ResponseModeAlert,
		security.ResponseModeQuarantine,
		security.ResponseModeKill,
	}

	for i, mode := range modes {
		if int(mode) != i {
			t.Errorf("ResponseMode %d has wrong value %d", i, mode)
		}
	}
}

func TestProcessStatus_Values(t *testing.T) {
	statuses := []security.ProcessStatus{
		security.ProcessStatusHealthy,
		security.ProcessStatusSuspicious,
		security.ProcessStatusCompromised,
		security.ProcessStatusQuarantined,
	}

	for i, status := range statuses {
		if int(status) != i {
			t.Errorf("ProcessStatus %d has wrong value %d", i, status)
		}
	}
}

func TestAnomalyType_String(t *testing.T) {
	types := []struct {
		at       security.AnomalyType
		expected string
	}{
		{security.AnomalyProcessInjection, "Process Injection"},
		{security.AnomalyPrivilegeEscalation, "Privilege Escalation"},
		{security.AnomalySuspiciousSyscall, "Suspicious Syscall"},
		{security.AnomalyNetworkExfiltration, "Network Exfiltration"},
		{security.AnomalyFileIntegrity, "File Integrity Violation"},
		{security.AnomalyBehavioralDeviation, "Behavioral Deviation"},
		{security.AnomalyMemoryCorruption, "Memory Corruption"},
	}

	for _, tc := range types {
		result := tc.at.String()
		if result != tc.expected {
			t.Errorf("AnomalyType %d: expected %q, got %q", tc.at, tc.expected, result)
		}
	}
}

func TestAnomaly_Structure(t *testing.T) {
	anomaly := &security.Anomaly{
		ID:          "ANM-001",
		Timestamp:   time.Now(),
		ProcessName: "test_process",
		PID:         1234,
		Type:        security.AnomalyBehavioralDeviation,
		Severity:    0.85,
		Description: "Test anomaly",
		Evidence:    []string{"evidence1", "evidence2"},
		Handled:     false,
	}

	if anomaly.ID != "ANM-001" {
		t.Errorf("Anomaly ID mismatch")
	}

	if anomaly.Severity != 0.85 {
		t.Errorf("Anomaly severity mismatch")
	}

	if len(anomaly.Evidence) != 2 {
		t.Errorf("Expected 2 evidence items, got %d", len(anomaly.Evidence))
	}
}

func TestBehaviorProfile_Structure(t *testing.T) {
	profile := &security.BehaviorProfile{
		FileAccess:     []string{"/dev/ttyS0", "/dev/imu"},
		NetworkAccess:  []string{"localhost:8080"},
		Syscalls:       []string{"read", "write", "ioctl"},
		CPUBaseline:    5.0,
		MemoryBaseline: 100 * 1024 * 1024,
	}

	if len(profile.FileAccess) != 2 {
		t.Errorf("Expected 2 file access patterns, got %d", len(profile.FileAccess))
	}

	if profile.CPUBaseline != 5.0 {
		t.Errorf("CPU baseline mismatch")
	}
}
