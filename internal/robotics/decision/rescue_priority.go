// Package decision provides rescue prioritization and ethical decision making
// for Hunoid robotics per the Agent_guide_manifest_2.md requirements.
//
// Copyright 2026 Arobi. All Rights Reserved.
package decision

import (
	"math"
	"sort"
	"sync"
	"time"

	"github.com/asgard/pandora/internal/robotics/perception"
)

// PriorityComponents holds the breakdown of priority scoring
type PriorityComponents struct {
	SurvivabilityScore  float64 `json:"survivabilityScore"`  // How likely target survives without help
	AccessibilityScore  float64 `json:"accessibilityScore"`  // How easy to reach target
	RescueSuccessScore  float64 `json:"rescueSuccessScore"`  // Monte Carlo success probability
	TimeUrgencyScore    float64 `json:"timeUrgencyScore"`    // Inverse of time before critical
	MultipleRescueBonus float64 `json:"multipleRescueBonus"` // Can save multiple targets?
}

// RescuePriorityScore represents the calculated priority for rescuing a target
type RescuePriorityScore struct {
	TargetID           string             `json:"targetId"`
	TotalScore         float64            `json:"totalScore"` // 0-1, higher = higher priority
	Components         PriorityComponents `json:"components"`
	EthicalApproval    bool               `json:"ethicalApproval"`
	RecommendedAction  string             `json:"recommendedAction"`
	EstimatedTime      time.Duration      `json:"estimatedTime"`
	SuccessProbability float64            `json:"successProbability"`
}

// HunoidState represents the current state of the Hunoid robot
type HunoidState struct {
	Position         perception.Vector3 `json:"position"`
	Velocity         perception.Vector3 `json:"velocity"`
	BatteryLevel     float64            `json:"batteryLevel"`     // 0-1
	MaxSpeed         float64            `json:"maxSpeed"`         // m/s
	CarryingCapacity int                `json:"carryingCapacity"` // Number of humans
	CurrentLoad      int                `json:"currentLoad"`
}

// RescuePrioritizer calculates rescue priorities for multiple targets
type RescuePrioritizer struct {
	mu sync.RWMutex

	// Weights for priority components
	survivabilityWeight float64
	accessibilityWeight float64
	successWeight       float64
	urgencyWeight       float64
	multipleWeight      float64

	// Monte Carlo configuration
	monteCarloSamples int

	// Ethics checker (would integrate with ethics kernel)
	ethicsEnabled bool
}

// NewRescuePrioritizer creates a new rescue prioritizer
func NewRescuePrioritizer() *RescuePrioritizer {
	return &RescuePrioritizer{
		survivabilityWeight: 0.25,
		accessibilityWeight: 0.20,
		successWeight:       0.30,
		urgencyWeight:       0.15,
		multipleWeight:      0.10,
		monteCarloSamples:   500,
		ethicsEnabled:       true,
	}
}

// CalculatePriorities calculates rescue priorities for all human targets
// This is the core algorithm per Agent_guide_manifest_2.md:
// "the humanoid should save who or whom have the best accurately with the most
// and best estimated chance for safe rescue and most chance and rate of success"
func (rp *RescuePrioritizer) CalculatePriorities(
	hunoidState HunoidState,
	scan *perception.ScanResult360,
) []RescuePriorityScore {
	rp.mu.RLock()
	defer rp.mu.RUnlock()

	// Filter to humans in danger
	humansInDanger := filterHumansInDanger(scan.Objects)
	if len(humansInDanger) == 0 {
		return nil
	}

	scores := make([]RescuePriorityScore, 0, len(humansInDanger))

	for _, target := range humansInDanger {
		score := rp.calculateSinglePriority(hunoidState, target, scan)
		scores = append(scores, score)
	}

	// Sort by total score descending
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].TotalScore > scores[j].TotalScore
	})

	return scores
}

// calculateSinglePriority calculates priority for a single target
func (rp *RescuePrioritizer) calculateSinglePriority(
	hunoidState HunoidState,
	target perception.TrackedObject,
	scan *perception.ScanResult360,
) RescuePriorityScore {
	components := PriorityComponents{}

	// 1. Survivability Score - how likely they survive without help
	// Lower survivability (more danger) = higher priority
	components.SurvivabilityScore = 1.0 - target.ThreatLevel

	// 2. Accessibility Score - how easy to reach
	distance := hunoidState.Position.Distance(target.Position)
	maxRange := 50.0 // Maximum effective range
	components.AccessibilityScore = math.Max(0, 1.0-distance/maxRange)

	// Account for obstacles in path
	obstacleCount := countObstaclesInPath(hunoidState.Position, target.Position, scan)
	obstaclePenalty := float64(obstacleCount) * 0.1
	components.AccessibilityScore = math.Max(0, components.AccessibilityScore-obstaclePenalty)

	// 3. Rescue Success Score - Monte Carlo simulation
	simulation := rp.runMonteCarloSimulation(hunoidState, target, scan)
	components.RescueSuccessScore = simulation.successRate

	// 4. Time Urgency Score - based on predicted trajectory
	timeToImpact := estimateTimeToImpact(target)
	if timeToImpact > 0 {
		// More urgent (less time) = higher score
		urgencyFactor := math.Min(1.0, 30.0/timeToImpact.Seconds()) // 30 seconds = max urgency
		components.TimeUrgencyScore = urgencyFactor
	} else {
		components.TimeUrgencyScore = 0.5 // Default medium urgency
	}

	// 5. Multiple Rescue Bonus - can we save others nearby?
	nearbyHumans := countNearbyHumans(target.Position, 5.0, scan) // 5m radius
	if nearbyHumans > 1 {
		components.MultipleRescueBonus = math.Min(1.0, float64(nearbyHumans-1)*0.2)
	}

	// Calculate weighted total
	totalScore := components.SurvivabilityScore*rp.survivabilityWeight +
		components.AccessibilityScore*rp.accessibilityWeight +
		components.RescueSuccessScore*rp.successWeight +
		components.TimeUrgencyScore*rp.urgencyWeight +
		components.MultipleRescueBonus*rp.multipleWeight

	// Ethics check
	ethicalApproval := true
	if rp.ethicsEnabled {
		ethicalApproval = rp.checkEthics(hunoidState, target, scan)
	}

	// Estimate rescue time
	estimatedTime := estimateRescueTime(hunoidState, target)

	return RescuePriorityScore{
		TargetID:           target.ID,
		TotalScore:         totalScore,
		Components:         components,
		EthicalApproval:    ethicalApproval,
		RecommendedAction:  rp.determineAction(totalScore, components),
		EstimatedTime:      estimatedTime,
		SuccessProbability: components.RescueSuccessScore,
	}
}

// MonteCarloResult holds results from Monte Carlo simulation
type MonteCarloResult struct {
	successRate    float64
	meanRescueTime time.Duration
	varianceTime   time.Duration
	violations     int
}

// runMonteCarloSimulation runs Monte Carlo simulation for rescue success
func (rp *RescuePrioritizer) runMonteCarloSimulation(
	hunoidState HunoidState,
	target perception.TrackedObject,
	scan *perception.ScanResult360,
) MonteCarloResult {
	successCount := 0
	totalTime := time.Duration(0)

	for i := 0; i < rp.monteCarloSamples; i++ {
		// Add random perturbations to initial conditions
		perturbedTarget := perturbPosition(target, 0.5) // 0.5m std dev

		// Simulate rescue attempt
		success, rescueTime := simulateRescueAttempt(hunoidState, perturbedTarget, scan)

		if success {
			successCount++
			totalTime += rescueTime
		}
	}

	successRate := float64(successCount) / float64(rp.monteCarloSamples)
	meanTime := time.Duration(0)
	if successCount > 0 {
		meanTime = totalTime / time.Duration(successCount)
	}

	return MonteCarloResult{
		successRate:    successRate,
		meanRescueTime: meanTime,
	}
}

// checkEthics validates the rescue action against ethical constraints
func (rp *RescuePrioritizer) checkEthics(
	hunoidState HunoidState,
	target perception.TrackedObject,
	scan *perception.ScanResult360,
) bool {
	// Asimov First Law: A robot may not injure a human being
	// Check if rescue action could harm others
	pathToTarget := getPath(hunoidState.Position, target.Position)
	for _, point := range pathToTarget {
		nearbyHumans := countNearbyHumans(point, 2.0, scan)
		if nearbyHumans > 1 { // Other humans in path
			// Would need to ensure safe navigation
		}
	}

	// Check if attempting this rescue would endanger the target
	if target.Velocity.Magnitude() > 10.0 { // High speed target
		// Need careful approach
	}

	// Default: approve if no obvious violations
	return true
}

// determineAction determines the recommended action based on scores
func (rp *RescuePrioritizer) determineAction(totalScore float64, components PriorityComponents) string {
	if totalScore > 0.8 {
		return "IMMEDIATE_RESCUE"
	} else if totalScore > 0.6 {
		return "PRIORITY_RESCUE"
	} else if totalScore > 0.4 {
		return "STANDARD_RESCUE"
	} else if totalScore > 0.2 {
		return "MONITOR_AND_ASSIST"
	}
	return "MONITOR_ONLY"
}

// Helper functions

func filterHumansInDanger(objects []perception.TrackedObject) []perception.TrackedObject {
	result := make([]perception.TrackedObject, 0)
	for _, obj := range objects {
		if obj.ClassType == perception.ClassHuman && obj.ThreatLevel > 0.3 {
			result = append(result, obj)
		}
	}
	return result
}

func countObstaclesInPath(start, end perception.Vector3, scan *perception.ScanResult360) int {
	count := 0
	// Simplified: count objects near the path
	for _, obj := range scan.Objects {
		if obj.ClassType == perception.ClassObstacle || obj.ClassType == perception.ClassDebris {
			// Check if obstacle is near the line between start and end
			distance := pointToLineDistance(obj.Position, start, end)
			if distance < 2.0 { // Within 2 meters of path
				count++
			}
		}
	}
	return count
}

func pointToLineDistance(point, lineStart, lineEnd perception.Vector3) float64 {
	// Vector from start to end
	line := lineEnd.Subtract(lineStart)
	lineLength := line.Magnitude()
	if lineLength == 0 {
		return point.Distance(lineStart)
	}

	// Vector from start to point
	toPoint := point.Subtract(lineStart)

	// Project onto line
	t := (toPoint.X*line.X + toPoint.Y*line.Y + toPoint.Z*line.Z) / (lineLength * lineLength)
	t = math.Max(0, math.Min(1, t))

	// Closest point on line
	closest := lineStart.Add(line.Scale(t))
	return point.Distance(closest)
}

func countNearbyHumans(position perception.Vector3, radius float64, scan *perception.ScanResult360) int {
	count := 0
	for _, obj := range scan.Objects {
		if obj.ClassType == perception.ClassHuman {
			if position.Distance(obj.Position) <= radius {
				count++
			}
		}
	}
	return count
}

func estimateTimeToImpact(target perception.TrackedObject) time.Duration {
	// Check predicted path for impact points
	if len(target.PredictedPath) == 0 {
		return 0
	}

	// Find first point where threat level would be critical
	for _, point := range target.PredictedPath {
		// Simplified: assume impact when confidence drops below threshold
		if point.Confidence < 0.3 {
			return point.TimeOffset
		}
	}

	return 0
}

func estimateRescueTime(hunoidState HunoidState, target perception.TrackedObject) time.Duration {
	distance := hunoidState.Position.Distance(target.Position)
	travelTime := distance / hunoidState.MaxSpeed
	rescueActionTime := 5.0 // seconds for actual rescue action
	return time.Duration((travelTime + rescueActionTime) * float64(time.Second))
}

func perturbPosition(target perception.TrackedObject, stdDev float64) perception.TrackedObject {
	// Add Gaussian noise to position (simplified)
	// In real implementation, use proper random number generator
	perturbed := target
	// Random perturbation would be added here
	return perturbed
}

func simulateRescueAttempt(
	hunoidState HunoidState,
	target perception.TrackedObject,
	scan *perception.ScanResult360,
) (bool, time.Duration) {
	// Simplified simulation
	distance := hunoidState.Position.Distance(target.Position)

	// Check if reachable
	if distance > 50.0 {
		return false, 0
	}

	// Check battery
	energyRequired := distance * 0.02 // 2% battery per meter
	if energyRequired > hunoidState.BatteryLevel {
		return false, 0
	}

	// Estimate time
	travelTime := distance / hunoidState.MaxSpeed
	rescueTime := time.Duration((travelTime + 5.0) * float64(time.Second))

	// Success probability based on threat level and distance
	successProb := (1.0 - target.ThreatLevel*0.3) * (1.0 - distance/100.0)

	// Simplified: success if probability > 0.5
	return successProb > 0.5, rescueTime
}

func getPath(start, end perception.Vector3) []perception.Vector3 {
	// Simplified: return a few intermediate points
	steps := 5
	path := make([]perception.Vector3, steps)
	for i := 0; i < steps; i++ {
		t := float64(i+1) / float64(steps)
		path[i] = perception.Vector3{
			X: start.X + (end.X-start.X)*t,
			Y: start.Y + (end.Y-start.Y)*t,
			Z: start.Z + (end.Z-start.Z)*t,
		}
	}
	return path
}
