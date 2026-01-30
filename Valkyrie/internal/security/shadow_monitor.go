// Package security provides security monitoring using shadow stack techniques
package security

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ShadowMonitor watches for zero-day threats in flight systems
type ShadowMonitor struct {
	mu sync.RWMutex

	// Monitored processes
	processes map[string]*ProcessMonitor

	// Anomaly detection
	anomalies chan *Anomaly

	// Configuration
	config ShadowConfig

	// Logger
	logger *logrus.Logger

	// Statistics
	scansCompleted uint64
	anomaliesFound uint64
}

// ShadowConfig holds shadow stack parameters
type ShadowConfig struct {
	MonitorFlightController bool
	MonitorSensorDrivers    bool
	MonitorNavigation       bool
	MonitorCommunication    bool

	AnomalyThreshold float64
	ResponseMode     ResponseMode

	ScanInterval time.Duration
}

// ResponseMode defines how to respond to threats
type ResponseMode int

const (
	ResponseModeLog ResponseMode = iota
	ResponseModeAlert
	ResponseModeQuarantine
	ResponseModeKill
)

// ProcessMonitor tracks a single process
type ProcessMonitor struct {
	PID              int
	Name             string
	ExpectedBehavior *BehaviorProfile
	CurrentBehavior  *BehaviorProfile
	AnomalyScore     float64
	LastCheck        time.Time
	Status           ProcessStatus
}

// ProcessStatus represents process health
type ProcessStatus int

const (
	ProcessStatusHealthy ProcessStatus = iota
	ProcessStatusSuspicious
	ProcessStatusCompromised
	ProcessStatusQuarantined
)

// BehaviorProfile defines expected process behavior
type BehaviorProfile struct {
	FileAccess     []string
	NetworkAccess  []string
	Syscalls       []string
	MemoryPattern  []byte
	CPUBaseline    float64
	MemoryBaseline int64
}

// Anomaly represents a detected anomaly
type Anomaly struct {
	ID          string
	Timestamp   time.Time
	ProcessName string
	PID         int
	Type        AnomalyType
	Severity    float64
	Description string
	Evidence    []string
	Handled     bool
}

// AnomalyType categorizes anomalies
type AnomalyType int

const (
	AnomalyProcessInjection AnomalyType = iota
	AnomalyPrivilegeEscalation
	AnomalySuspiciousSyscall
	AnomalyNetworkExfiltration
	AnomalyFileIntegrity
	AnomalyBehavioralDeviation
	AnomalyMemoryCorruption
	AnomalyCodeInjection
	AnomalyUnauthorizedAccess
)

// String returns the string representation of AnomalyType
func (at AnomalyType) String() string {
	types := []string{
		"Process Injection",
		"Privilege Escalation",
		"Suspicious Syscall",
		"Network Exfiltration",
		"File Integrity Violation",
		"Behavioral Deviation",
		"Memory Corruption",
		"Code Injection",
		"Unauthorized Access",
	}
	if int(at) < len(types) {
		return types[at]
	}
	return "Unknown"
}

// NewShadowMonitor creates a new shadow monitor
func NewShadowMonitor(config ShadowConfig) *ShadowMonitor {
	if config.ScanInterval == 0 {
		config.ScanInterval = 100 * time.Millisecond
	}

	sm := &ShadowMonitor{
		processes: make(map[string]*ProcessMonitor),
		anomalies: make(chan *Anomaly, 100),
		config:    config,
		logger:    logrus.New(),
	}

	// Register critical processes
	sm.registerCriticalProcesses()

	return sm
}

// registerCriticalProcesses sets up monitoring for critical flight systems
func (sm *ShadowMonitor) registerCriticalProcesses() {
	if sm.config.MonitorFlightController {
		sm.processes["flight_controller"] = &ProcessMonitor{
			Name: "flight_controller",
			ExpectedBehavior: &BehaviorProfile{
				FileAccess:    []string{"/dev/ttyS*", "/dev/spi*"},
				NetworkAccess: []string{"localhost:8080"},
				Syscalls:      []string{"read", "write", "ioctl"},
			},
			Status: ProcessStatusHealthy,
		}
	}

	if sm.config.MonitorSensorDrivers {
		sm.processes["sensor_fusion"] = &ProcessMonitor{
			Name: "sensor_fusion",
			ExpectedBehavior: &BehaviorProfile{
				FileAccess:    []string{"/dev/imu*", "/dev/gps*"},
				NetworkAccess: []string{},
				Syscalls:      []string{"read", "write", "mmap"},
			},
			Status: ProcessStatusHealthy,
		}
	}

	if sm.config.MonitorNavigation {
		sm.processes["navigation"] = &ProcessMonitor{
			Name: "navigation",
			ExpectedBehavior: &BehaviorProfile{
				FileAccess:    []string{"/data/maps/*"},
				NetworkAccess: []string{"*:443"},
				Syscalls:      []string{"read", "write", "socket"},
			},
			Status: ProcessStatusHealthy,
		}
	}

	if sm.config.MonitorCommunication {
		sm.processes["communication"] = &ProcessMonitor{
			Name: "communication",
			ExpectedBehavior: &BehaviorProfile{
				FileAccess:    []string{"/dev/radio*"},
				NetworkAccess: []string{"*:*"},
				Syscalls:      []string{"read", "write", "sendto", "recvfrom"},
			},
			Status: ProcessStatusHealthy,
		}
	}
}

// Start begins monitoring
func (sm *ShadowMonitor) Start(ctx context.Context) error {
	sm.logger.Info("Shadow Monitor starting...")

	// Start anomaly handler
	go sm.handleAnomalies(ctx)

	// Monitor loop
	ticker := time.NewTicker(sm.config.ScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			sm.logger.Info("Shadow Monitor stopping...")
			return ctx.Err()

		case <-ticker.C:
			sm.scanProcesses()
		}
	}
}

// scanProcesses checks all monitored processes
func (sm *ShadowMonitor) scanProcesses() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for name, proc := range sm.processes {
		// Get current behavior from shadow stack
		behavior := sm.getCurrentBehavior(proc.PID)
		proc.CurrentBehavior = behavior

		// Compare with expected
		anomalyScore := sm.compareBehaviors(proc.ExpectedBehavior, behavior)
		proc.AnomalyScore = anomalyScore
		proc.LastCheck = time.Now()

		if anomalyScore > sm.config.AnomalyThreshold {
			proc.Status = ProcessStatusSuspicious

			anomaly := &Anomaly{
				ID:          fmt.Sprintf("ANM-%d", time.Now().UnixNano()),
				Timestamp:   time.Now(),
				ProcessName: name,
				PID:         proc.PID,
				Type:        AnomalyBehavioralDeviation,
				Severity:    anomalyScore,
				Description: fmt.Sprintf("Behavioral deviation detected in %s (score: %.2f)", name, anomalyScore),
			}

			select {
			case sm.anomalies <- anomaly:
				sm.anomaliesFound++
			default:
				// Buffer full, log and continue
				sm.logger.Warn("Anomaly buffer full, dropping anomaly")
			}
		} else {
			proc.Status = ProcessStatusHealthy
		}
	}

	sm.scansCompleted++
}

// handleAnomalies processes detected anomalies
func (sm *ShadowMonitor) handleAnomalies(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case anomaly := <-sm.anomalies:
			sm.handleAnomaly(anomaly)
		}
	}
}

// handleAnomaly responds to a detected anomaly
func (sm *ShadowMonitor) handleAnomaly(anomaly *Anomaly) {
	sm.logger.WithFields(logrus.Fields{
		"process":  anomaly.ProcessName,
		"pid":      anomaly.PID,
		"type":     anomaly.Type.String(),
		"severity": anomaly.Severity,
	}).Warn("Anomaly detected")

	switch sm.config.ResponseMode {
	case ResponseModeLog:
		// Just log (already done above)
		anomaly.Handled = true

	case ResponseModeAlert:
		// Send alert to operator
		sm.sendAlert(anomaly)
		anomaly.Handled = true

	case ResponseModeQuarantine:
		// Isolate the process
		sm.quarantineProcess(anomaly.PID)
		anomaly.Handled = true

	case ResponseModeKill:
		// Terminate the process
		sm.killProcess(anomaly.PID)
		anomaly.Handled = true
	}
}

// getCurrentBehavior gets current process behavior
func (sm *ShadowMonitor) getCurrentBehavior(pid int) *BehaviorProfile {
	// In a real implementation, this would:
	// 1. Use eBPF or ptrace to monitor syscalls
	// 2. Check /proc/pid/fd for file descriptors
	// 3. Monitor network connections
	// 4. Track memory patterns

	// For now, return simulated behavior
	return &BehaviorProfile{
		FileAccess:     []string{"/dev/ttyS0"},
		NetworkAccess:  []string{"localhost:8080"},
		Syscalls:       []string{"read", "write"},
		CPUBaseline:    5.0,
		MemoryBaseline: 50 * 1024 * 1024,
	}
}

// compareBehaviors compares expected vs actual behavior
func (sm *ShadowMonitor) compareBehaviors(expected, actual *BehaviorProfile) float64 {
	if expected == nil || actual == nil {
		return 0.0
	}

	score := 0.0
	maxScore := 0.0

	// Check file access deviations
	for _, file := range actual.FileAccess {
		maxScore += 1.0
		found := false
		for _, expectedFile := range expected.FileAccess {
			if matchPattern(expectedFile, file) {
				found = true
				break
			}
		}
		if !found {
			score += 1.0
		}
	}

	// Check network access deviations
	for _, net := range actual.NetworkAccess {
		maxScore += 1.0
		found := false
		for _, expectedNet := range expected.NetworkAccess {
			if matchPattern(expectedNet, net) {
				found = true
				break
			}
		}
		if !found {
			score += 1.0
		}
	}

	// Check syscall deviations
	for _, syscall := range actual.Syscalls {
		maxScore += 0.5
		found := false
		for _, expectedSyscall := range expected.Syscalls {
			if syscall == expectedSyscall {
				found = true
				break
			}
		}
		if !found {
			score += 0.5
		}
	}

	if maxScore == 0 {
		return 0.0
	}

	return score / maxScore
}

// matchPattern checks if a pattern matches a string (simple wildcard support)
func matchPattern(pattern, str string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == str {
		return true
	}
	// Simple wildcard at end
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(str) >= len(prefix) && str[:len(prefix)] == prefix
	}
	// Simple wildcard at start
	if len(pattern) > 0 && pattern[0] == '*' {
		suffix := pattern[1:]
		return len(str) >= len(suffix) && str[len(str)-len(suffix):] == suffix
	}
	return false
}

// sendAlert sends an alert notification
func (sm *ShadowMonitor) sendAlert(anomaly *Anomaly) {
	sm.logger.WithFields(logrus.Fields{
		"anomaly_id": anomaly.ID,
		"process":    anomaly.ProcessName,
		"severity":   anomaly.Severity,
	}).Error("SECURITY ALERT: Anomaly requires attention")

	// TODO: Send to alerting system (Nysus, email, etc.)
}

// quarantineProcess isolates a process
func (sm *ShadowMonitor) quarantineProcess(pid int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, proc := range sm.processes {
		if proc.PID == pid {
			proc.Status = ProcessStatusQuarantined
			sm.logger.WithField("pid", pid).Warn("Process quarantined")
			// TODO: Implement actual isolation (cgroups, namespaces, etc.)
			break
		}
	}
}

// killProcess terminates a process
func (sm *ShadowMonitor) killProcess(pid int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.logger.WithField("pid", pid).Error("Terminating compromised process")
	// TODO: Implement actual process termination
	// syscall.Kill(pid, syscall.SIGKILL)
}

// GetStats returns monitoring statistics
func (sm *ShadowMonitor) GetStats() (scans, anomalies uint64) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.scansCompleted, sm.anomaliesFound
}

// GetProcessStatus returns status of all monitored processes
func (sm *ShadowMonitor) GetProcessStatus() map[string]ProcessStatus {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	status := make(map[string]ProcessStatus)
	for name, proc := range sm.processes {
		status[name] = proc.Status
	}
	return status
}

// IsHealthy returns true if all processes are healthy
func (sm *ShadowMonitor) IsHealthy() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, proc := range sm.processes {
		if proc.Status != ProcessStatusHealthy {
			return false
		}
	}
	return true
}
