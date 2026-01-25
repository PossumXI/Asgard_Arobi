// Package shadow implements the shadow stack for zero-day detection.
// It runs a parallel execution environment to detect anomalous behavior.
package shadow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Execution represents a tracked execution in the shadow stack
type Execution struct {
	ID            string
	ProcessName   string
	CommandLine   string
	ParentID      string
	StartTime     time.Time
	EndTime       *time.Time
	ExitCode      *int
	MemoryUsage   uint64
	CPUTime       float64
	NetworkIO     uint64
	FileAccess    []FileAccessEvent
	NetworkAccess []NetworkAccessEvent
	Syscalls      []SyscallEvent
	Anomalies     []Anomaly
	Status        ExecutionStatus
}

// ExecutionStatus represents the state of an execution
type ExecutionStatus string

const (
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusSuspended ExecutionStatus = "suspended"
	ExecutionStatusKilled    ExecutionStatus = "killed"
)

// FileAccessEvent tracks file system access
type FileAccessEvent struct {
	Timestamp time.Time
	Path      string
	Operation string // read, write, delete, create
	Allowed   bool
	Hash      string
}

// NetworkAccessEvent tracks network activity
type NetworkAccessEvent struct {
	Timestamp   time.Time
	RemoteAddr  string
	RemotePort  int
	Protocol    string
	Direction   string // inbound, outbound
	BytesSent   uint64
	BytesRecv   uint64
	Blocked     bool
}

// SyscallEvent tracks system calls
type SyscallEvent struct {
	Timestamp time.Time
	Name      string
	Args      []interface{}
	RetVal    int
	Suspicious bool
}

// Anomaly represents detected anomalous behavior
type Anomaly struct {
	ID          string
	Type        AnomalyType
	Severity    string
	Description string
	Evidence    interface{}
	DetectedAt  time.Time
	ExecutionID string
}

// AnomalyType categorizes anomalies
type AnomalyType string

const (
	AnomalyTypeProcessInjection   AnomalyType = "process_injection"
	AnomalyTypePrivilegeEscalation AnomalyType = "privilege_escalation"
	AnomalyTypeSuspiciousSyscall  AnomalyType = "suspicious_syscall"
	AnomalyTypeNetworkExfiltration AnomalyType = "network_exfiltration"
	AnomalyTypeFileIntegrity      AnomalyType = "file_integrity"
	AnomalyTypeBehaviorDeviation  AnomalyType = "behavior_deviation"
	AnomalyTypeMemoryCorruption   AnomalyType = "memory_corruption"
)

// BehaviorProfile represents expected behavior for a process
type BehaviorProfile struct {
	ProcessName      string
	AllowedSyscalls  map[string]bool
	AllowedPaths     []string
	AllowedNetworks  []string
	MaxMemoryMB      uint64
	MaxCPUPercent    float64
	MaxNetworkMBps   float64
	ExpectedDuration time.Duration
}

// ShadowStack manages parallel execution monitoring
type ShadowStack struct {
	mu              sync.RWMutex
	executions      map[string]*Execution
	profiles        map[string]*BehaviorProfile
	anomalies       []Anomaly
	anomalyChan     chan Anomaly
	config          Config
	stopCh          chan struct{}
	wg              sync.WaitGroup
}

// Config holds shadow stack configuration
type Config struct {
	MaxExecutions      int
	AnomalyThreshold   float64
	ProfileStrictness  string // strict, moderate, permissive
	EnableHeuristics   bool
	MonitorInterval    time.Duration
	RetentionDuration  time.Duration
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		MaxExecutions:     1000,
		AnomalyThreshold:  0.7,
		ProfileStrictness: "moderate",
		EnableHeuristics:  true,
		MonitorInterval:   100 * time.Millisecond,
		RetentionDuration: 24 * time.Hour,
	}
}

// NewShadowStack creates a new shadow stack
func NewShadowStack(cfg Config) *ShadowStack {
	return &ShadowStack{
		executions:  make(map[string]*Execution),
		profiles:    make(map[string]*BehaviorProfile),
		anomalies:   make([]Anomaly, 0),
		anomalyChan: make(chan Anomaly, 100),
		config:      cfg,
		stopCh:      make(chan struct{}),
	}
}

// Start begins the shadow stack monitoring
func (s *ShadowStack) Start(ctx context.Context) error {
	s.loadDefaultProfiles()

	// Start monitoring loop
	s.wg.Add(1)
	go s.monitorLoop(ctx)

	// Start anomaly processor
	s.wg.Add(1)
	go s.processAnomalies(ctx)

	// Start cleanup routine
	s.wg.Add(1)
	go s.cleanupLoop(ctx)

	log.Printf("[Shadow] Shadow stack started with %d profiles", len(s.profiles))
	return nil
}

// Stop shuts down the shadow stack
func (s *ShadowStack) Stop() {
	close(s.stopCh)
	s.wg.Wait()
	log.Println("[Shadow] Shadow stack stopped")
}

// TrackExecution starts monitoring an execution
func (s *ShadowStack) TrackExecution(processName, commandLine, parentID string) (*Execution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.executions) >= s.config.MaxExecutions {
		return nil, fmt.Errorf("maximum executions limit reached")
	}

	exec := &Execution{
		ID:            uuid.New().String(),
		ProcessName:   processName,
		CommandLine:   commandLine,
		ParentID:      parentID,
		StartTime:     time.Now(),
		FileAccess:    make([]FileAccessEvent, 0),
		NetworkAccess: make([]NetworkAccessEvent, 0),
		Syscalls:      make([]SyscallEvent, 0),
		Anomalies:     make([]Anomaly, 0),
		Status:        ExecutionStatusRunning,
	}

	s.executions[exec.ID] = exec
	log.Printf("[Shadow] Tracking execution: %s (%s)", exec.ID, processName)
	return exec, nil
}

// RecordFileAccess logs a file access event
func (s *ShadowStack) RecordFileAccess(execID, path, operation string, allowed bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	exec, exists := s.executions[execID]
	if !exists {
		return fmt.Errorf("execution not found: %s", execID)
	}

	event := FileAccessEvent{
		Timestamp: time.Now(),
		Path:      path,
		Operation: operation,
		Allowed:   allowed,
		Hash:      hashString(path + operation),
	}
	exec.FileAccess = append(exec.FileAccess, event)

	// Check against profile
	if profile := s.profiles[exec.ProcessName]; profile != nil {
		if !s.isPathAllowed(path, profile.AllowedPaths) {
			s.raiseAnomaly(exec.ID, AnomalyTypeFileIntegrity, "medium",
				fmt.Sprintf("Unauthorized file access: %s", path), event)
		}
	}

	return nil
}

// RecordNetworkAccess logs a network access event
func (s *ShadowStack) RecordNetworkAccess(execID, remoteAddr string, remotePort int, protocol, direction string, bytesSent, bytesRecv uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	exec, exists := s.executions[execID]
	if !exists {
		return fmt.Errorf("execution not found: %s", execID)
	}

	event := NetworkAccessEvent{
		Timestamp:  time.Now(),
		RemoteAddr: remoteAddr,
		RemotePort: remotePort,
		Protocol:   protocol,
		Direction:  direction,
		BytesSent:  bytesSent,
		BytesRecv:  bytesRecv,
		Blocked:    false,
	}
	exec.NetworkAccess = append(exec.NetworkAccess, event)
	exec.NetworkIO += bytesSent + bytesRecv

	// Check for exfiltration patterns
	if profile := s.profiles[exec.ProcessName]; profile != nil {
		if !s.isNetworkAllowed(remoteAddr, profile.AllowedNetworks) {
			event.Blocked = true
			s.raiseAnomaly(exec.ID, AnomalyTypeNetworkExfiltration, "high",
				fmt.Sprintf("Unauthorized network access to %s:%d", remoteAddr, remotePort), event)
		}
	}

	return nil
}

// RecordSyscall logs a system call
func (s *ShadowStack) RecordSyscall(execID, name string, args []interface{}, retVal int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	exec, exists := s.executions[execID]
	if !exists {
		return fmt.Errorf("execution not found: %s", execID)
	}

	suspicious := s.isSyscallSuspicious(name, args)

	event := SyscallEvent{
		Timestamp:  time.Now(),
		Name:       name,
		Args:       args,
		RetVal:     retVal,
		Suspicious: suspicious,
	}
	exec.Syscalls = append(exec.Syscalls, event)

	// Check against profile
	if profile := s.profiles[exec.ProcessName]; profile != nil {
		if !profile.AllowedSyscalls[name] && s.config.ProfileStrictness == "strict" {
			s.raiseAnomaly(exec.ID, AnomalyTypeSuspiciousSyscall, "medium",
				fmt.Sprintf("Disallowed syscall: %s", name), event)
		}
	}

	if suspicious {
		s.raiseAnomaly(exec.ID, AnomalyTypeSuspiciousSyscall, "high",
			fmt.Sprintf("Suspicious syscall detected: %s", name), event)
	}

	return nil
}

// CompleteExecution marks an execution as finished
func (s *ShadowStack) CompleteExecution(execID string, exitCode int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	exec, exists := s.executions[execID]
	if !exists {
		return fmt.Errorf("execution not found: %s", execID)
	}

	now := time.Now()
	exec.EndTime = &now
	exec.ExitCode = &exitCode
	exec.Status = ExecutionStatusCompleted

	// Final behavior analysis
	s.analyzeBehavior(exec)

	log.Printf("[Shadow] Execution completed: %s (exit: %d, anomalies: %d)",
		execID, exitCode, len(exec.Anomalies))
	return nil
}

// GetExecution returns execution details
func (s *ShadowStack) GetExecution(execID string) (*Execution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	exec, exists := s.executions[execID]
	if !exists {
		return nil, fmt.Errorf("execution not found: %s", execID)
	}
	return exec, nil
}

// GetAnomalies returns all detected anomalies
func (s *ShadowStack) GetAnomalies() []Anomaly {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Anomaly{}, s.anomalies...)
}

// GetStatistics returns shadow stack statistics
func (s *ShadowStack) GetStatistics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"active_executions": len(s.executions),
		"total_anomalies":   len(s.anomalies),
		"profiles_loaded":   len(s.profiles),
		"config":            s.config,
	}
}

// AnomalyChan returns channel for anomaly notifications
func (s *ShadowStack) AnomalyChan() <-chan Anomaly {
	return s.anomalyChan
}

func (s *ShadowStack) raiseAnomaly(execID string, anomalyType AnomalyType, severity, description string, evidence interface{}) {
	anomaly := Anomaly{
		ID:          uuid.New().String(),
		Type:        anomalyType,
		Severity:    severity,
		Description: description,
		Evidence:    evidence,
		DetectedAt:  time.Now(),
		ExecutionID: execID,
	}

	if exec, exists := s.executions[execID]; exists {
		exec.Anomalies = append(exec.Anomalies, anomaly)
	}
	s.anomalies = append(s.anomalies, anomaly)

	// Non-blocking send to anomaly channel
	select {
	case s.anomalyChan <- anomaly:
	default:
		log.Printf("[Shadow] Warning: anomaly channel full")
	}

	log.Printf("[Shadow] ANOMALY: %s - %s (severity: %s)", anomalyType, description, severity)
}

func (s *ShadowStack) monitorLoop(ctx context.Context) {
	defer s.wg.Done()
	ticker := time.NewTicker(s.config.MonitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkExecutions()
		}
	}
}

func (s *ShadowStack) checkExecutions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, exec := range s.executions {
		if exec.Status != ExecutionStatusRunning {
			continue
		}

		// Check for long-running processes
		if profile := s.profiles[exec.ProcessName]; profile != nil {
			if profile.ExpectedDuration > 0 && time.Since(exec.StartTime) > profile.ExpectedDuration*2 {
				s.raiseAnomaly(exec.ID, AnomalyTypeBehaviorDeviation, "medium",
					"Process running longer than expected", nil)
			}

			// Check memory usage
			if profile.MaxMemoryMB > 0 && exec.MemoryUsage > profile.MaxMemoryMB*1024*1024 {
				s.raiseAnomaly(exec.ID, AnomalyTypeMemoryCorruption, "high",
					"Process exceeding memory limits", nil)
			}
		}
	}
}

func (s *ShadowStack) processAnomalies(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		case anomaly := <-s.anomalyChan:
			// Process anomaly (could integrate with Giru threat system)
			data, _ := json.Marshal(anomaly)
			log.Printf("[Shadow] Processing anomaly: %s", string(data))
		}
	}
}

func (s *ShadowStack) cleanupLoop(ctx context.Context) {
	defer s.wg.Done()
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cleanup()
		}
	}
}

func (s *ShadowStack) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-s.config.RetentionDuration)

	// Clean old completed executions
	for id, exec := range s.executions {
		if exec.Status == ExecutionStatusCompleted && exec.EndTime != nil && exec.EndTime.Before(cutoff) {
			delete(s.executions, id)
		}
	}

	// Clean old anomalies
	newAnomalies := make([]Anomaly, 0)
	for _, a := range s.anomalies {
		if a.DetectedAt.After(cutoff) {
			newAnomalies = append(newAnomalies, a)
		}
	}
	s.anomalies = newAnomalies
}

func (s *ShadowStack) analyzeBehavior(exec *Execution) {
	// Check for process injection patterns
	if len(exec.FileAccess) > 100 && len(exec.NetworkAccess) > 50 {
		s.raiseAnomaly(exec.ID, AnomalyTypeBehaviorDeviation, "medium",
			"High I/O activity detected", nil)
	}

	// Check for privilege escalation patterns
	for _, sc := range exec.Syscalls {
		if sc.Name == "setuid" || sc.Name == "setgid" || sc.Name == "ptrace" {
			s.raiseAnomaly(exec.ID, AnomalyTypePrivilegeEscalation, "critical",
				fmt.Sprintf("Privilege escalation attempt via %s", sc.Name), sc)
		}
	}
}

func (s *ShadowStack) isPathAllowed(path string, allowedPaths []string) bool {
	for _, allowed := range allowedPaths {
		if len(path) >= len(allowed) && path[:len(allowed)] == allowed {
			return true
		}
	}
	return len(allowedPaths) == 0 // If no restrictions, allow all
}

func (s *ShadowStack) isNetworkAllowed(addr string, allowedNetworks []string) bool {
	for _, allowed := range allowedNetworks {
		if addr == allowed || allowed == "*" {
			return true
		}
	}
	return len(allowedNetworks) == 0
}

func (s *ShadowStack) isSyscallSuspicious(name string, args []interface{}) bool {
	suspiciousCalls := map[string]bool{
		"ptrace":      true,
		"mprotect":    true,
		"process_vm_writev": true,
		"process_vm_readv": true,
		"memfd_create": true,
	}
	return suspiciousCalls[name]
}

func (s *ShadowStack) loadDefaultProfiles() {
	// System service profile
	s.profiles["systemd"] = &BehaviorProfile{
		ProcessName:     "systemd",
		AllowedSyscalls: map[string]bool{"read": true, "write": true, "open": true, "close": true},
		AllowedPaths:    []string{"/etc", "/var", "/run"},
		AllowedNetworks: []string{"127.0.0.1", "::1"},
		MaxMemoryMB:     256,
		MaxCPUPercent:   10,
	}

	// Web server profile
	s.profiles["nginx"] = &BehaviorProfile{
		ProcessName:     "nginx",
		AllowedSyscalls: map[string]bool{"read": true, "write": true, "accept": true, "epoll_wait": true},
		AllowedPaths:    []string{"/var/www", "/etc/nginx", "/var/log/nginx"},
		AllowedNetworks: []string{"*"},
		MaxMemoryMB:     512,
		MaxCPUPercent:   80,
	}

	// Database profile
	s.profiles["postgres"] = &BehaviorProfile{
		ProcessName:     "postgres",
		AllowedSyscalls: map[string]bool{"read": true, "write": true, "fsync": true, "fdatasync": true},
		AllowedPaths:    []string{"/var/lib/postgresql", "/etc/postgresql"},
		AllowedNetworks: []string{"127.0.0.1", "::1"},
		MaxMemoryMB:     4096,
		MaxCPUPercent:   90,
	}
}

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8])
}
