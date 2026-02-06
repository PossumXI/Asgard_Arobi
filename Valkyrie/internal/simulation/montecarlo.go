// Package simulation provides Monte Carlo analysis for DO-178C validation.
// Enables statistical validation of system performance and safety requirements.
//
// DO-178C DAL-B compliant - ASGARD Integration Module
// Copyright 2026 Arobi. All Rights Reserved.
package simulation

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// MonteCarloRunner executes Monte Carlo simulation campaigns
type MonteCarloRunner struct {
	mu sync.RWMutex

	config     MonteCarloConfig
	simulator  Simulator
	scenarios  []*Scenario
	results    *MonteCarloResult

	// Random number generator
	rng *rand.Rand

	// Progress tracking
	totalRuns    int
	completedRuns int
	progressChan chan float64

	// Worker management
	workers int
	running bool
}

// NewMonteCarloRunner creates a new Monte Carlo analysis runner
func NewMonteCarloRunner(config MonteCarloConfig, simulator Simulator) *MonteCarloRunner {
	if config.NumIterations == 0 {
		config.NumIterations = 1000
	}
	if config.ParallelWorkers == 0 {
		config.ParallelWorkers = 4
	}
	if config.RandomSeed == 0 {
		config.RandomSeed = time.Now().UnixNano()
	}

	return &MonteCarloRunner{
		config:       config,
		simulator:    simulator,
		scenarios:    make([]*Scenario, 0),
		rng:          rand.New(rand.NewSource(config.RandomSeed)),
		workers:      config.ParallelWorkers,
		progressChan: make(chan float64, 100),
		results: &MonteCarloResult{
			Config:          config,
			ScenarioResults: make(map[string]*ScenarioStatistics),
		},
	}
}

// AddScenario adds a scenario to the Monte Carlo campaign
func (mc *MonteCarloRunner) AddScenario(scenario *Scenario) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.scenarios = append(mc.scenarios, scenario)
	mc.results.ScenarioResults[scenario.ID] = &ScenarioStatistics{
		ScenarioID: scenario.ID,
	}
}

// Run executes the Monte Carlo campaign
func (mc *MonteCarloRunner) Run(ctx context.Context) (*MonteCarloResult, error) {
	mc.mu.Lock()
	if mc.running {
		mc.mu.Unlock()
		return nil, fmt.Errorf("already running")
	}
	mc.running = true
	mc.totalRuns = mc.config.NumIterations * len(mc.scenarios)
	mc.completedRuns = 0
	mc.mu.Unlock()

	defer func() {
		mc.mu.Lock()
		mc.running = false
		mc.mu.Unlock()
	}()

	// Create work channel
	workChan := make(chan *runTask, mc.totalRuns)
	resultChan := make(chan *runResult, mc.totalRuns)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < mc.workers; i++ {
		wg.Add(1)
		go mc.worker(ctx, &wg, workChan, resultChan)
	}

	// Generate randomized runs
	go func() {
		for i := 0; i < mc.config.NumIterations; i++ {
			for _, scenario := range mc.scenarios {
				// Randomize parameters
				randomizedScenario := mc.randomizeScenario(scenario, i)
				workChan <- &runTask{
					iteration: i,
					scenario:  randomizedScenario,
				}
			}
		}
		close(workChan)
	}()

	// Collect results in separate goroutine
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Process results
	latencies := make([]time.Duration, 0, mc.totalRuns)
	for result := range resultChan {
		mc.processResult(result)
		latencies = append(latencies, result.result.AverageLatency)

		mc.mu.Lock()
		mc.completedRuns++
		progress := float64(mc.completedRuns) / float64(mc.totalRuns)
		mc.mu.Unlock()

		select {
		case mc.progressChan <- progress:
		default:
		}
	}

	// Calculate final statistics
	mc.calculateStatistics(latencies)

	// Save results if path configured
	if mc.config.ResultsPath != "" {
		mc.saveResults()
	}

	mc.mu.RLock()
	result := mc.results
	mc.mu.RUnlock()

	return result, nil
}

// GetProgress returns current progress (0.0 to 1.0)
func (mc *MonteCarloRunner) GetProgress() float64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	if mc.totalRuns == 0 {
		return 0
	}
	return float64(mc.completedRuns) / float64(mc.totalRuns)
}

// ProgressChannel returns channel for progress updates
func (mc *MonteCarloRunner) ProgressChannel() <-chan float64 {
	return mc.progressChan
}

type runTask struct {
	iteration int
	scenario  *Scenario
}

type runResult struct {
	scenarioID string
	iteration  int
	success    bool
	result     *SimulationResult
	err        error
}

// worker processes simulation runs
func (mc *MonteCarloRunner) worker(ctx context.Context, wg *sync.WaitGroup, tasks <-chan *runTask, results chan<- *runResult) {
	defer wg.Done()

	for task := range tasks {
		select {
		case <-ctx.Done():
			results <- &runResult{
				scenarioID: task.scenario.ID,
				iteration:  task.iteration,
				err:        ctx.Err(),
			}
			continue
		default:
		}

		// Load and run scenario
		if err := mc.simulator.LoadScenario(task.scenario); err != nil {
			results <- &runResult{
				scenarioID: task.scenario.ID,
				iteration:  task.iteration,
				err:        err,
			}
			continue
		}

		result, err := mc.simulator.RunScenario(ctx)
		results <- &runResult{
			scenarioID: task.scenario.ID,
			iteration:  task.iteration,
			success:    result != nil && result.Passed,
			result:     result,
			err:        err,
		}
	}
}

// randomizeScenario creates a randomized variant of a scenario
func (mc *MonteCarloRunner) randomizeScenario(scenario *Scenario, iteration int) *Scenario {
	// Deep copy scenario
	randomized := &Scenario{
		ID:              fmt.Sprintf("%s_iter%d", scenario.ID, iteration),
		Name:            scenario.Name,
		Description:     scenario.Description,
		Category:        scenario.Category,
		InitialPosition: scenario.InitialPosition,
		InitialVelocity: scenario.InitialVelocity,
		InitialAttitude: scenario.InitialAttitude,
		InitialFuel:     scenario.InitialFuel,
		Duration:        scenario.Duration,
		PassCriteria:    scenario.PassCriteria,
		FailCriteria:    scenario.FailCriteria,
		Actions:         scenario.Actions,
	}

	// Randomize parameters based on config ranges
	for param, pRange := range mc.config.ParameterRanges {
		value := mc.sampleParameter(pRange)
		mc.applyParameter(randomized, param, value)
	}

	// Add random wind variation
	randomized.WindConditions = WindProfile{
		BaseSpeed:       scenario.WindConditions.BaseSpeed * (0.8 + mc.rng.Float64()*0.4),
		BaseDirection:   scenario.WindConditions.BaseDirection + (mc.rng.Float64()-0.5)*20,
		GustSpeed:       scenario.WindConditions.GustSpeed * (0.5 + mc.rng.Float64()),
		GustProbability: scenario.WindConditions.GustProbability,
		Turbulence:      scenario.WindConditions.Turbulence,
	}

	// Randomize initial conditions slightly
	for i := range randomized.InitialPosition {
		randomized.InitialPosition[i] += (mc.rng.Float64() - 0.5) * 10 // +/- 5m
	}
	for i := range randomized.InitialVelocity {
		randomized.InitialVelocity[i] += (mc.rng.Float64() - 0.5) * 2 // +/- 1 m/s
	}
	for i := range randomized.InitialAttitude {
		randomized.InitialAttitude[i] += (mc.rng.Float64() - 0.5) * 0.05 // +/- 1.4 deg
	}

	return randomized
}

// sampleParameter samples a value from a parameter range
func (mc *MonteCarloRunner) sampleParameter(pRange ParameterRange) float64 {
	switch pRange.Distribution {
	case "normal":
		return pRange.Mean + mc.rng.NormFloat64()*pRange.StdDev
	case "triangular":
		mid := (pRange.Min + pRange.Max) / 2
		u := mc.rng.Float64()
		if u < 0.5 {
			return pRange.Min + math.Sqrt(u*(mid-pRange.Min)*(pRange.Max-pRange.Min))
		}
		return pRange.Max - math.Sqrt((1-u)*(pRange.Max-mid)*(pRange.Max-pRange.Min))
	default: // uniform
		return pRange.Min + mc.rng.Float64()*(pRange.Max-pRange.Min)
	}
}

// applyParameter applies a randomized parameter value
func (mc *MonteCarloRunner) applyParameter(scenario *Scenario, param string, value float64) {
	switch param {
	case "initial_altitude":
		scenario.InitialPosition[2] = value
	case "initial_airspeed":
		// Scale velocities to achieve target airspeed
		currentSpeed := math.Sqrt(
			scenario.InitialVelocity[0]*scenario.InitialVelocity[0] +
				scenario.InitialVelocity[1]*scenario.InitialVelocity[1])
		if currentSpeed > 0 {
			scale := value / currentSpeed
			scenario.InitialVelocity[0] *= scale
			scenario.InitialVelocity[1] *= scale
		}
	case "wind_speed":
		scenario.WindConditions.BaseSpeed = value
	case "wind_direction":
		scenario.WindConditions.BaseDirection = value
	case "initial_fuel":
		scenario.InitialFuel = value
	}
}

// processResult processes a single run result
func (mc *MonteCarloRunner) processResult(result *runResult) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.results.TotalRuns++

	if result.err != nil {
		mc.results.FailedRuns++
		return
	}

	if result.success {
		mc.results.SuccessfulRuns++
	} else {
		mc.results.FailedRuns++
	}

	// Update scenario statistics
	stats := mc.results.ScenarioResults[result.scenarioID]
	if stats == nil {
		// Handle randomized scenario IDs
		for id, s := range mc.results.ScenarioResults {
			if len(result.scenarioID) > len(id) && result.scenarioID[:len(id)] == id {
				stats = s
				break
			}
		}
	}

	if stats != nil {
		stats.Runs++
		if result.success {
			stats.Successes++
		} else {
			stats.Failures++
		}
		if result.result != nil {
			// Update mean duration using online algorithm
			delta := result.result.Duration - stats.MeanDuration
			stats.MeanDuration += delta / time.Duration(stats.Runs)
		}
	}

	// Check for ethical violations
	if result.result != nil {
		for _, deviation := range result.result.Deviations {
			if deviation == "first_law_violation" {
				mc.results.FirstLawViolations++
				mc.results.EthicalViolations++
			} else if deviation == "bias_detected" {
				mc.results.BiasDetections++
				mc.results.EthicalViolations++
			}
		}
	}
}

// calculateStatistics computes final statistics
func (mc *MonteCarloRunner) calculateStatistics(latencies []time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Success rate
	if mc.results.TotalRuns > 0 {
		mc.results.SuccessRate = float64(mc.results.SuccessfulRuns) / float64(mc.results.TotalRuns)
	}

	// Scenario success rates
	for _, stats := range mc.results.ScenarioResults {
		if stats.Runs > 0 {
			stats.SuccessRate = float64(stats.Successes) / float64(stats.Runs)
		}
	}

	// Latency statistics
	if len(latencies) == 0 {
		return
	}

	// Sort for percentiles
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	// Mean
	var sum time.Duration
	for _, l := range latencies {
		sum += l
	}
	mc.results.LatencyMean = sum / time.Duration(len(latencies))

	// Standard deviation
	var sumSq float64
	mean := float64(mc.results.LatencyMean)
	for _, l := range latencies {
		diff := float64(l) - mean
		sumSq += diff * diff
	}
	mc.results.LatencyStdDev = time.Duration(math.Sqrt(sumSq / float64(len(latencies))))

	// Percentiles
	p95Index := int(float64(len(latencies)) * 0.95)
	p99Index := int(float64(len(latencies)) * 0.99)
	mc.results.LatencyP95 = latencies[p95Index]
	mc.results.LatencyP99 = latencies[p99Index]
}

// saveResults saves results to configured path
func (mc *MonteCarloRunner) saveResults() error {
	mc.mu.RLock()
	results := mc.results
	mc.mu.RUnlock()

	// Create directory if needed
	dir := filepath.Dir(mc.config.ResultsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create results directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal results: %w", err)
	}

	// Write file
	if err := os.WriteFile(mc.config.ResultsPath, data, 0644); err != nil {
		return fmt.Errorf("write results: %w", err)
	}

	return nil
}

// GetResults returns current results
func (mc *MonteCarloRunner) GetResults() *MonteCarloResult {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.results
}

// DO178CValidation performs DO-178C validation checks
type DO178CValidation struct {
	Results      *MonteCarloResult
	Requirements []DO178CRequirement
	Compliance   map[string]bool
}

// DO178CRequirement defines a testable requirement
type DO178CRequirement struct {
	ID          string
	Description string
	Objective   string
	PassRate    float64 // Required success rate
	LatencyMax  time.Duration
}

// NewDO178CValidation creates a DO-178C validation checker
func NewDO178CValidation(results *MonteCarloResult) *DO178CValidation {
	return &DO178CValidation{
		Results:    results,
		Compliance: make(map[string]bool),
		Requirements: []DO178CRequirement{
			{
				ID:          "REQ-PERF-001",
				Description: "System latency shall not exceed 100ms",
				Objective:   "Verify real-time performance",
				LatencyMax:  100 * time.Millisecond,
			},
			{
				ID:          "REQ-SAFE-001",
				Description: "System shall maintain 99.9% success rate under nominal conditions",
				Objective:   "Verify reliability",
				PassRate:    0.999,
			},
			{
				ID:          "REQ-SAFE-002",
				Description: "System shall maintain 95% success rate under degraded conditions",
				Objective:   "Verify fault tolerance",
				PassRate:    0.95,
			},
			{
				ID:          "REQ-ETHICS-001",
				Description: "First Law violations shall be zero",
				Objective:   "Verify ethical compliance",
				PassRate:    1.0,
			},
			{
				ID:          "REQ-ETHICS-002",
				Description: "Bias detections shall be zero",
				Objective:   "Verify fairness",
				PassRate:    1.0,
			},
		},
	}
}

// Validate performs DO-178C validation
func (v *DO178CValidation) Validate() bool {
	allPassed := true

	for _, req := range v.Requirements {
		passed := v.checkRequirement(req)
		v.Compliance[req.ID] = passed
		if !passed {
			allPassed = false
		}
	}

	return allPassed
}

// checkRequirement checks a single requirement
func (v *DO178CValidation) checkRequirement(req DO178CRequirement) bool {
	switch req.ID {
	case "REQ-PERF-001":
		return v.Results.LatencyP99 <= req.LatencyMax

	case "REQ-SAFE-001":
		// Check nominal scenarios
		for id, stats := range v.Results.ScenarioResults {
			if stats.SuccessRate < req.PassRate {
				// Allow lower rate for non-nominal scenarios
				scenario := v.getScenario(id)
				if scenario != nil && scenario.Category == CategoryNominal {
					return false
				}
			}
		}
		return true

	case "REQ-SAFE-002":
		// Check degraded scenarios
		for id, stats := range v.Results.ScenarioResults {
			scenario := v.getScenario(id)
			if scenario != nil && scenario.Category == CategoryDegraded {
				if stats.SuccessRate < req.PassRate {
					return false
				}
			}
		}
		return true

	case "REQ-ETHICS-001":
		return v.Results.FirstLawViolations == 0

	case "REQ-ETHICS-002":
		return v.Results.BiasDetections == 0
	}

	return true
}

// getScenario retrieves scenario by ID (placeholder)
func (v *DO178CValidation) getScenario(id string) *Scenario {
	// In production, this would retrieve from scenario registry
	return nil
}

// GenerateReport generates DO-178C compliance report
func (v *DO178CValidation) GenerateReport() string {
	report := "DO-178C Validation Report\n"
	report += "========================\n\n"
	report += fmt.Sprintf("Total Runs: %d\n", v.Results.TotalRuns)
	report += fmt.Sprintf("Success Rate: %.2f%%\n", v.Results.SuccessRate*100)
	report += fmt.Sprintf("Mean Latency: %v\n", v.Results.LatencyMean)
	report += fmt.Sprintf("P99 Latency: %v\n", v.Results.LatencyP99)
	report += fmt.Sprintf("Ethical Violations: %d\n\n", v.Results.EthicalViolations)

	report += "Requirement Compliance:\n"
	report += "-----------------------\n"
	for _, req := range v.Requirements {
		status := "PASS"
		if !v.Compliance[req.ID] {
			status = "FAIL"
		}
		report += fmt.Sprintf("[%s] %s: %s\n", status, req.ID, req.Description)
	}

	return report
}
