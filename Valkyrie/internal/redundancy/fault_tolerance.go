// Package redundancy provides fault tolerance and redundancy management
package redundancy

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// RedundantSystem manages multiple instances of a critical system
type RedundantSystem struct {
	mu sync.RWMutex

	primary   System
	backup    System
	emergency System

	currentMode SystemMode

	healthCheck func(System) bool

	logger *logrus.Logger

	// Statistics
	failovers    int
	lastFailover time.Time
}

// System interface for redundant components
type System interface {
	IsHealthy() bool
	Start(context.Context) error
	Stop() error
	Process(interface{}) (interface{}, error)
	Name() string
}

// SystemMode represents which system is active
type SystemMode int

const (
	ModePrimary SystemMode = iota
	ModeBackup
	ModeEmergency
	ModeDegraded
)

// String returns the string representation of SystemMode
func (sm SystemMode) String() string {
	modes := []string{"Primary", "Backup", "Emergency", "Degraded"}
	if int(sm) < len(modes) {
		return modes[sm]
	}
	return "Unknown"
}

// VotingResult represents the result of sensor voting
type VotingResult struct {
	Value      interface{}
	Confidence float64
	Agreement  int
	Total      int
	Outliers   []int
}

// SensorVoter implements voting-based fault tolerance for sensors
type SensorVoter struct {
	mu sync.RWMutex

	sensors   []SensorInput
	threshold float64

	logger *logrus.Logger
}

// SensorInput represents a sensor value
type SensorInput struct {
	Value     float64
	Quality   float64
	Timestamp time.Time
	SensorID  int
}

// NewRedundantSystem creates a triple-redundant system
func NewRedundantSystem(primary, backup, emergency System) *RedundantSystem {
	return &RedundantSystem{
		primary:     primary,
		backup:      backup,
		emergency:   emergency,
		currentMode: ModePrimary,
		healthCheck: func(s System) bool {
			if s == nil {
				return false
			}
			return s.IsHealthy()
		},
		logger: logrus.New(),
	}
}

// Process executes on the active system with automatic failover
func (rs *RedundantSystem) Process(input interface{}) (interface{}, error) {
	rs.mu.RLock()
	mode := rs.currentMode
	rs.mu.RUnlock()

	var system System
	switch mode {
	case ModePrimary:
		system = rs.primary
	case ModeBackup:
		system = rs.backup
	case ModeEmergency:
		system = rs.emergency
	case ModeDegraded:
		// Try to find any working system
		if rs.healthCheck(rs.primary) {
			system = rs.primary
		} else if rs.healthCheck(rs.backup) {
			system = rs.backup
		} else if rs.healthCheck(rs.emergency) {
			system = rs.emergency
		} else {
			rs.logger.Error("All systems failed!")
			return nil, ErrAllSystemsFailed
		}
	}

	if system == nil {
		rs.failover()
		return rs.Process(input)
	}

	result, err := system.Process(input)
	if err != nil {
		rs.logger.WithError(err).WithField("system", system.Name()).Warn("System processing failed")
		rs.failover()
		return rs.Process(input)
	}

	return result, nil
}

// Monitor continuously checks system health
func (rs *RedundantSystem) Monitor(ctx context.Context) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			rs.checkHealth()
		}
	}
}

// checkHealth verifies all systems
func (rs *RedundantSystem) checkHealth() {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	primaryHealthy := rs.healthCheck(rs.primary)
	backupHealthy := rs.healthCheck(rs.backup)
	emergencyHealthy := rs.healthCheck(rs.emergency)

	// Count healthy systems
	healthyCount := 0
	if primaryHealthy {
		healthyCount++
	}
	if backupHealthy {
		healthyCount++
	}
	if emergencyHealthy {
		healthyCount++
	}

	// Log status changes
	if healthyCount == 0 {
		rs.logger.Error("CRITICAL: All redundant systems failed!")
		rs.currentMode = ModeDegraded
		return
	}

	// Try to restore to primary if healthy
	if primaryHealthy && rs.currentMode != ModePrimary {
		rs.logger.Info("Primary system healthy, restoring...")
		rs.currentMode = ModePrimary
		return
	}

	// Handle primary failure
	if !primaryHealthy && rs.currentMode == ModePrimary {
		rs.doFailover(backupHealthy, emergencyHealthy)
	}

	// Handle backup failure when in backup mode
	if !backupHealthy && rs.currentMode == ModeBackup {
		if emergencyHealthy {
			rs.logger.Warn("Backup failed, switching to emergency system")
			rs.currentMode = ModeEmergency
			rs.failovers++
			rs.lastFailover = time.Now()
		} else if primaryHealthy {
			rs.logger.Info("Backup failed but primary recovered, switching back")
			rs.currentMode = ModePrimary
		}
	}
}

// doFailover performs the actual failover
func (rs *RedundantSystem) doFailover(backupHealthy, emergencyHealthy bool) {
	if backupHealthy {
		rs.logger.Warn("Primary failed, switching to backup system")
		rs.currentMode = ModeBackup
	} else if emergencyHealthy {
		rs.logger.Error("Primary and backup failed, switching to emergency system")
		rs.currentMode = ModeEmergency
	} else {
		rs.logger.Error("All systems unhealthy!")
		rs.currentMode = ModeDegraded
	}
	rs.failovers++
	rs.lastFailover = time.Now()
}

// failover switches to backup system
func (rs *RedundantSystem) failover() {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	switch rs.currentMode {
	case ModePrimary:
		if rs.healthCheck(rs.backup) {
			rs.logger.Warn("Failing over from primary to backup")
			rs.currentMode = ModeBackup
		} else if rs.healthCheck(rs.emergency) {
			rs.logger.Error("Failing over from primary to emergency")
			rs.currentMode = ModeEmergency
		}
	case ModeBackup:
		if rs.healthCheck(rs.emergency) {
			rs.logger.Error("Failing over from backup to emergency")
			rs.currentMode = ModeEmergency
		} else if rs.healthCheck(rs.primary) {
			rs.logger.Info("Primary recovered, switching back")
			rs.currentMode = ModePrimary
		}
	case ModeEmergency:
		// Try to recover
		if rs.healthCheck(rs.primary) {
			rs.logger.Info("Primary recovered from emergency")
			rs.currentMode = ModePrimary
		} else if rs.healthCheck(rs.backup) {
			rs.logger.Info("Backup recovered from emergency")
			rs.currentMode = ModeBackup
		}
	}

	rs.failovers++
	rs.lastFailover = time.Now()
}

// GetCurrentMode returns the current operating mode
func (rs *RedundantSystem) GetCurrentMode() SystemMode {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.currentMode
}

// GetStats returns redundancy statistics
func (rs *RedundantSystem) GetStats() (mode SystemMode, failovers int, lastFailover time.Time) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	return rs.currentMode, rs.failovers, rs.lastFailover
}

// NewSensorVoter creates a new sensor voter
func NewSensorVoter(threshold float64) *SensorVoter {
	if threshold == 0 {
		threshold = 0.1 // 10% default threshold
	}
	return &SensorVoter{
		sensors:   make([]SensorInput, 0),
		threshold: threshold,
		logger:    logrus.New(),
	}
}

// Vote performs voting on sensor inputs and returns consensus value
func (sv *SensorVoter) Vote(inputs []SensorInput) *VotingResult {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	if len(inputs) == 0 {
		return &VotingResult{
			Confidence: 0,
			Agreement:  0,
			Total:      0,
		}
	}

	if len(inputs) == 1 {
		return &VotingResult{
			Value:      inputs[0].Value,
			Confidence: inputs[0].Quality,
			Agreement:  1,
			Total:      1,
		}
	}

	// Calculate weighted median
	totalWeight := 0.0
	weightedSum := 0.0

	for _, input := range inputs {
		weight := input.Quality
		totalWeight += weight
		weightedSum += input.Value * weight
	}

	median := weightedSum / totalWeight

	// Find outliers and calculate agreement
	outliers := make([]int, 0)
	agreement := 0

	for _, input := range inputs {
		deviation := abs(input.Value - median)
		relativeDeviation := deviation / abs(median)

		if relativeDeviation > sv.threshold {
			outliers = append(outliers, input.SensorID)
			sv.logger.WithFields(logrus.Fields{
				"sensor":    input.SensorID,
				"value":     input.Value,
				"median":    median,
				"deviation": relativeDeviation,
			}).Warn("Sensor outlier detected")
		} else {
			agreement++
		}
	}

	// Calculate confidence based on agreement
	confidence := float64(agreement) / float64(len(inputs))

	// If less than majority agrees, flag as low confidence
	if agreement < len(inputs)/2+1 {
		sv.logger.Warn("Low agreement among sensors")
		confidence *= 0.5
	}

	return &VotingResult{
		Value:      median,
		Confidence: confidence,
		Agreement:  agreement,
		Total:      len(inputs),
		Outliers:   outliers,
	}
}

// abs returns absolute value of float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Error definitions
var (
	ErrAllSystemsFailed = &RedundancyError{Message: "all redundant systems have failed"}
)

// RedundancyError represents a redundancy-related error
type RedundancyError struct {
	Message string
}

func (e *RedundancyError) Error() string {
	return e.Message
}

// TripleModularRedundancy implements TMR voting
type TripleModularRedundancy struct {
	modules [3]func(interface{}) (interface{}, error)
	voter   func([]interface{}) interface{}
}

// NewTMR creates a new TMR system
func NewTMR(
	module1, module2, module3 func(interface{}) (interface{}, error),
	voter func([]interface{}) interface{},
) *TripleModularRedundancy {
	return &TripleModularRedundancy{
		modules: [3]func(interface{}) (interface{}, error){module1, module2, module3},
		voter:   voter,
	}
}

// Process runs all three modules and votes on the result
func (tmr *TripleModularRedundancy) Process(input interface{}) (interface{}, error) {
	results := make([]interface{}, 0, 3)
	errors := make([]error, 0)

	for _, module := range tmr.modules {
		result, err := module(input)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		results = append(results, result)
	}

	// Need at least 2 results for voting
	if len(results) < 2 {
		if len(errors) > 0 {
			return nil, errors[0]
		}
		return nil, ErrAllSystemsFailed
	}

	// Vote on results
	return tmr.voter(results), nil
}
