// Package hil provides a Hardware-in-the-Loop test framework for Silenus and Hunoid
package hil

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// HardwareMode defines whether to use real hardware or auto-detect.
type HardwareMode string

const (
	// HardwareModeReal uses actual hardware
	HardwareModeReal HardwareMode = "real"
	// HardwareModeAuto auto-detects and errors if unavailable
	HardwareModeAuto HardwareMode = "auto"
)

// HardwareConfig configures hardware for HIL tests
type HardwareConfig struct {
	// Mode determines hardware selection strategy
	Mode HardwareMode

	// Silenus configuration
	SilenusEnabled   bool
	CameraDevice     string // e.g., "/dev/video0" or "COM3"
	GPSDevice        string // e.g., "/dev/ttyUSB0"
	PowerMonitorAddr string // e.g., "192.168.1.10:502"

	// Hunoid configuration
	HunoidEnabled     bool
	HunoidID          string
	HunoidControlAddr string // e.g., "192.168.1.20:50051"
	ManipulatorAddr   string // e.g., "192.168.1.21:50051"

	// Timeouts
	InitTimeout      time.Duration
	OperationTimeout time.Duration
	ShutdownTimeout  time.Duration

	// Test behavior
	SkipSlowTests    bool
	VerboseLogging   bool
	RecordMetrics    bool
	MetricsOutputDir string
}

// DefaultConfig returns a default HIL configuration using real hardware.
func DefaultConfig() *HardwareConfig {
	return &HardwareConfig{
		Mode:             HardwareModeAuto,
		SilenusEnabled:   true,
		HunoidEnabled:    true,
		HunoidID:         "test-hunoid-001",
		InitTimeout:      30 * time.Second,
		OperationTimeout: 10 * time.Second,
		ShutdownTimeout:  5 * time.Second,
		SkipSlowTests:    false,
		VerboseLogging:   false,
		RecordMetrics:    false,
		MetricsOutputDir: "./test_metrics",
	}
}

// TestResult represents the outcome of a single test
type TestResult struct {
	Name      string
	Passed    bool
	Duration  time.Duration
	Error     error
	Metrics   map[string]float64
	Timestamp time.Time
}

// HILTestSuite manages Hardware-in-the-Loop test execution
type HILTestSuite struct {
	mu             sync.RWMutex
	config         *HardwareConfig
	silenusAdapter *SilenusAdapter
	hunoidAdapter  *HunoidAdapter
	initialized    bool
	results        []*TestResult
	startTime      time.Time
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewHILTestSuite creates a new HIL test suite
func NewHILTestSuite(config *HardwareConfig) *HILTestSuite {
	if config == nil {
		config = DefaultConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &HILTestSuite{
		config:  config,
		results: make([]*TestResult, 0),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// SetupHardware initializes hardware based on configuration
func (s *HILTestSuite) SetupHardware() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		return fmt.Errorf("hardware already initialized")
	}

	ctx, cancel := context.WithTimeout(s.ctx, s.config.InitTimeout)
	defer cancel()

	s.startTime = time.Now()

	// Initialize Silenus adapter
	if s.config.SilenusEnabled {
		s.silenusAdapter = NewSilenusAdapter(s.config)
		if err := s.silenusAdapter.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize Silenus adapter: %w", err)
		}
		s.log("Silenus adapter initialized (mode: %s)", s.silenusAdapter.GetMode())
	}

	// Initialize Hunoid adapter
	if s.config.HunoidEnabled {
		s.hunoidAdapter = NewHunoidAdapter(s.config)
		if err := s.hunoidAdapter.Initialize(ctx); err != nil {
			// Cleanup Silenus if Hunoid fails
			if s.silenusAdapter != nil {
				_ = s.silenusAdapter.Shutdown()
			}
			return fmt.Errorf("failed to initialize Hunoid adapter: %w", err)
		}
		s.log("Hunoid adapter initialized (mode: %s)", s.hunoidAdapter.GetMode())
	}

	s.initialized = true
	return nil
}

// TeardownHardware cleans up all hardware resources
func (s *HILTestSuite) TeardownHardware() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	var errs []error

	// Shutdown Hunoid adapter
	if s.hunoidAdapter != nil {
		if err := s.hunoidAdapter.ShutdownWithContext(ctx); err != nil {
			errs = append(errs, fmt.Errorf("hunoid shutdown: %w", err))
		}
		s.hunoidAdapter = nil
	}

	// Shutdown Silenus adapter
	if s.silenusAdapter != nil {
		if err := s.silenusAdapter.ShutdownWithContext(ctx); err != nil {
			errs = append(errs, fmt.Errorf("silenus shutdown: %w", err))
		}
		s.silenusAdapter = nil
	}

	s.initialized = false

	if len(errs) > 0 {
		return fmt.Errorf("teardown errors: %v", errs)
	}

	return nil
}

// Silenus returns the Silenus hardware adapter
func (s *HILTestSuite) Silenus() *SilenusAdapter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.silenusAdapter
}

// Hunoid returns the Hunoid hardware adapter
func (s *HILTestSuite) Hunoid() *HunoidAdapter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.hunoidAdapter
}

// Context returns the suite's context
func (s *HILTestSuite) Context() context.Context {
	return s.ctx
}

// Config returns the suite's configuration
func (s *HILTestSuite) Config() *HardwareConfig {
	return s.config
}

// IsInitialized returns whether hardware is initialized
func (s *HILTestSuite) IsInitialized() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.initialized
}

// RunTest executes a single test with hardware setup/teardown
func (s *HILTestSuite) RunTest(t *testing.T, name string, testFunc func(t *testing.T, suite *HILTestSuite)) {
	t.Run(name, func(t *testing.T) {
		start := time.Now()
		result := &TestResult{
			Name:      name,
			Metrics:   make(map[string]float64),
			Timestamp: start,
		}

		// Run the test
		defer func() {
			if r := recover(); r != nil {
				result.Passed = false
				result.Error = fmt.Errorf("panic: %v", r)
				t.Errorf("Test panicked: %v", r)
			}
			result.Duration = time.Since(start)
			s.recordResult(result)
		}()

		// Check if suite is initialized
		if !s.IsInitialized() {
			t.Fatal("HIL test suite not initialized - call SetupHardware() first")
		}

		testFunc(t, s)
		result.Passed = !t.Failed()
	})
}

// RunTestWithContext runs a test with a custom context
func (s *HILTestSuite) RunTestWithContext(t *testing.T, name string, timeout time.Duration, testFunc func(ctx context.Context, t *testing.T, suite *HILTestSuite)) {
	t.Run(name, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(s.ctx, timeout)
		defer cancel()

		start := time.Now()
		result := &TestResult{
			Name:      name,
			Metrics:   make(map[string]float64),
			Timestamp: start,
		}

		defer func() {
			if r := recover(); r != nil {
				result.Passed = false
				result.Error = fmt.Errorf("panic: %v", r)
				t.Errorf("Test panicked: %v", r)
			}
			result.Duration = time.Since(start)
			s.recordResult(result)
		}()

		if !s.IsInitialized() {
			t.Fatal("HIL test suite not initialized")
		}

		testFunc(ctx, t, s)
		result.Passed = !t.Failed()
	})
}

// SkipIfSlow skips the test if slow tests are disabled
func (s *HILTestSuite) SkipIfSlow(t *testing.T) {
	if s.config.SkipSlowTests {
		t.Skip("Slow tests disabled")
	}
}

// SkipIfNoHardware skips if hardware is unavailable.
func (s *HILTestSuite) SkipIfNoHardware(t *testing.T, component string) {
	switch component {
	case "silenus":
		if s.silenusAdapter != nil && !s.silenusAdapter.IsAvailable() {
			t.Skip("Skipping test - Silenus hardware unavailable")
		}
	case "hunoid":
		if s.hunoidAdapter != nil && !s.hunoidAdapter.IsAvailable() {
			t.Skip("Skipping test - Hunoid hardware unavailable")
		}
	}
}

// RecordMetric records a metric for the current test
func (s *HILTestSuite) RecordMetric(name string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.results) > 0 {
		s.results[len(s.results)-1].Metrics[name] = value
	}
}

// GetResults returns all test results
func (s *HILTestSuite) GetResults() []*TestResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make([]*TestResult, len(s.results))
	copy(results, s.results)
	return results
}

// PrintSummary prints a summary of test results
func (s *HILTestSuite) PrintSummary() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	passed := 0
	failed := 0
	totalDuration := time.Duration(0)

	for _, r := range s.results {
		totalDuration += r.Duration
		if r.Passed {
			passed++
		} else {
			failed++
		}
	}

	fmt.Printf("\n=== HIL Test Summary ===\n")
	fmt.Printf("Total:    %d\n", len(s.results))
	fmt.Printf("Passed:   %d\n", passed)
	fmt.Printf("Failed:   %d\n", failed)
	fmt.Printf("Duration: %v\n", totalDuration)
	fmt.Printf("========================\n")
}

func (s *HILTestSuite) recordResult(result *TestResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results = append(s.results, result)
}

func (s *HILTestSuite) log(format string, args ...interface{}) {
	if s.config.VerboseLogging {
		fmt.Printf("[HIL] "+format+"\n", args...)
	}
}

// Close shuts down the test suite
func (s *HILTestSuite) Close() error {
	s.cancel()
	return s.TeardownHardware()
}

// SetupAndRun is a convenience function for running tests with setup/teardown
func SetupAndRun(t *testing.T, config *HardwareConfig, testFunc func(t *testing.T, suite *HILTestSuite)) {
	suite := NewHILTestSuite(config)

	if err := suite.SetupHardware(); err != nil {
		t.Fatalf("Failed to setup hardware: %v", err)
	}

	defer func() {
		if err := suite.Close(); err != nil {
			t.Errorf("Failed to teardown hardware: %v", err)
		}
	}()

	testFunc(t, suite)
}
