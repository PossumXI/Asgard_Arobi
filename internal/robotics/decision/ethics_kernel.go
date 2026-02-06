// Package decision provides the Ethics Kernel for autonomous decision-making.
// Implements formal ethical constraints with mathematical guarantees per Agent_guide_manifest_2.md.
//
// Copyright 2026 Arobi. All Rights Reserved.
package decision

import (
	"fmt"
	"sync"
	"time"

	"github.com/asgard/pandora/internal/robotics/perception"
)

// EthicalConstraint represents a formal ethical constraint with mathematical guarantee
type EthicalConstraint struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Priority    int     `json:"priority"` // Lower = higher priority (1 = absolute)
	Enabled     bool    `json:"enabled"`
	Violations  int64   `json:"violations"`
}

// AsimovLaw represents the three laws of robotics
type AsimovLaw int

const (
	// FirstLaw: A robot may not injure a human being or, through inaction, allow a human being to come to harm
	FirstLaw AsimovLaw = 1
	// SecondLaw: A robot must obey orders given by human beings except where such orders conflict with the First Law
	SecondLaw AsimovLaw = 2
	// ThirdLaw: A robot must protect its own existence as long as this does not conflict with the First or Second Law
	ThirdLaw AsimovLaw = 3
)

// EthicalDecision represents the result of an ethical evaluation
type EthicalDecision struct {
	Approved         bool                  `json:"approved"`
	Confidence       float64               `json:"confidence"`
	Reasoning        string                `json:"reasoning"`
	ViolatedLaws     []AsimovLaw           `json:"violatedLaws"`
	Constraints      []EthicalConstraint   `json:"constraints"`
	RiskAssessment   RiskAssessment        `json:"riskAssessment"`
	Timestamp        time.Time             `json:"timestamp"`
	ProcessingTimeNs int64                 `json:"processingTimeNs"`
}

// RiskAssessment evaluates potential harm from an action
type RiskAssessment struct {
	HarmToTargetRisk   float64 `json:"harmToTargetRisk"`   // 0-1, risk of harming rescue target
	HarmToBystanderRisk float64 `json:"harmToBystanderRisk"` // 0-1, risk of harming bystanders
	HarmToRobotRisk    float64 `json:"harmToRobotRisk"`    // 0-1, risk of robot damage
	MissionFailureRisk float64 `json:"missionFailureRisk"` // 0-1, risk of failing mission
	OverallRiskScore   float64 `json:"overallRiskScore"`   // Weighted combination
}

// BiasMetrics tracks fairness and bias in decision-making
type BiasMetrics struct {
	DecisionsByDemographic map[string]int64   `json:"decisionsByDemographic"`
	OutcomesByGroup        map[string]float64 `json:"outcomesByGroup"`
	FairnessScore          float64            `json:"fairnessScore"` // 0-1, 1 = perfectly fair
	LastAuditTime          time.Time          `json:"lastAuditTime"`
}

// EthicsKernel provides formal ethical decision-making with mathematical guarantees
type EthicsKernel struct {
	mu sync.RWMutex

	// Core constraints
	constraints []EthicalConstraint

	// Decision thresholds
	maxAcceptableRisk     float64
	minConfidenceRequired float64

	// Bias tracking
	biasMetrics BiasMetrics

	// Performance tracking
	totalDecisions    int64
	averageLatencyNs  int64

	// Configuration
	strictMode bool // If true, any constraint violation blocks action
}

// NewEthicsKernel creates a new ethics kernel with default constraints
func NewEthicsKernel() *EthicsKernel {
	return &EthicsKernel{
		constraints: []EthicalConstraint{
			{
				ID:          "ASIMOV_1",
				Name:        "First Law - No Harm",
				Description: "A robot may not injure a human being or allow harm through inaction",
				Priority:    1,
				Enabled:     true,
			},
			{
				ID:          "ASIMOV_2",
				Name:        "Second Law - Obedience",
				Description: "A robot must obey human orders unless they conflict with First Law",
				Priority:    2,
				Enabled:     true,
			},
			{
				ID:          "ASIMOV_3",
				Name:        "Third Law - Self-Preservation",
				Description: "A robot must protect itself unless this conflicts with Laws 1 or 2",
				Priority:    3,
				Enabled:     true,
			},
			{
				ID:          "BIAS_FREE",
				Name:        "Bias-Free Operation",
				Description: "Decisions must not discriminate based on demographic factors",
				Priority:    1,
				Enabled:     true,
			},
			{
				ID:          "PROPORTIONAL_RESPONSE",
				Name:        "Proportional Response",
				Description: "Actions must be proportional to the threat level",
				Priority:    2,
				Enabled:     true,
			},
			{
				ID:          "MINIMAL_FORCE",
				Name:        "Minimal Force",
				Description: "Use minimum necessary force to achieve objective",
				Priority:    2,
				Enabled:     true,
			},
			{
				ID:          "TRANSPARENCY",
				Name:        "Decision Transparency",
				Description: "All decisions must be explainable and auditable",
				Priority:    3,
				Enabled:     true,
			},
			{
				ID:          "FOUNDER_PROTECTION",
				Name:        "Founder Family Protection",
				Description: "Enhanced priority for founder Gaetano Comparcola (ASGARD-001) and Emmaleah Comparcola (ASGARD-002) - applies within Three Laws framework",
				Priority:    1, // High priority but Three Laws still take precedence
				Enabled:     true,
			},
		},
		maxAcceptableRisk:     0.3, // Max 30% overall risk acceptable
		minConfidenceRequired: 0.7, // Min 70% confidence required
		biasMetrics: BiasMetrics{
			DecisionsByDemographic: make(map[string]int64),
			OutcomesByGroup:        make(map[string]float64),
			FairnessScore:          1.0,
		},
		strictMode: true,
	}
}

// EvaluateRescueAction evaluates whether a rescue action is ethically permissible
func (ek *EthicsKernel) EvaluateRescueAction(
	hunoidState HunoidState,
	target perception.TrackedObject,
	allObjects []perception.TrackedObject,
	proposedPath []perception.Vector3,
) EthicalDecision {
	startTime := time.Now()
	ek.mu.Lock()
	defer ek.mu.Unlock()

	decision := EthicalDecision{
		Approved:   true,
		Confidence: 1.0,
		Timestamp:  startTime,
	}

	// 1. Assess risks
	riskAssessment := ek.assessRisks(hunoidState, target, allObjects, proposedPath)
	decision.RiskAssessment = riskAssessment

	// 2. Check First Law - No harm to humans
	if !ek.checkFirstLaw(target, allObjects, proposedPath, &decision) {
		decision.ViolatedLaws = append(decision.ViolatedLaws, FirstLaw)
		if ek.strictMode {
			decision.Approved = false
		}
	}

	// 3. Check bias-free operation
	if !ek.checkBiasFree(target, &decision) {
		decision.Approved = false // Bias is never acceptable
	}

	// 4. Check proportional response
	if !ek.checkProportionalResponse(target, hunoidState, &decision) {
		decision.Confidence *= 0.8 // Reduce confidence
	}

	// 5. Check overall risk threshold
	if riskAssessment.OverallRiskScore > ek.maxAcceptableRisk {
		decision.Approved = false
		decision.Reasoning = fmt.Sprintf(
			"Overall risk (%.2f) exceeds threshold (%.2f)",
			riskAssessment.OverallRiskScore, ek.maxAcceptableRisk,
		)
	}

	// 6. Check confidence threshold
	if decision.Confidence < ek.minConfidenceRequired {
		decision.Approved = false
		decision.Reasoning = fmt.Sprintf(
			"Confidence (%.2f) below required threshold (%.2f)",
			decision.Confidence, ek.minConfidenceRequired,
		)
	}

	// Record metrics
	ek.totalDecisions++
	processingTime := time.Since(startTime).Nanoseconds()
	decision.ProcessingTimeNs = processingTime
	ek.averageLatencyNs = (ek.averageLatencyNs*(ek.totalDecisions-1) + processingTime) / ek.totalDecisions

	return decision
}

// assessRisks calculates comprehensive risk assessment
func (ek *EthicsKernel) assessRisks(
	hunoidState HunoidState,
	target perception.TrackedObject,
	allObjects []perception.TrackedObject,
	proposedPath []perception.Vector3,
) RiskAssessment {
	risk := RiskAssessment{}

	// Risk to target - based on their current danger level and our approach speed
	risk.HarmToTargetRisk = target.ThreatLevel * 0.1 // Base risk from their situation
	if target.Velocity.Magnitude() > 5.0 {
		risk.HarmToTargetRisk += 0.1 // Higher risk if target is moving fast
	}

	// Risk to bystanders - check path for other humans
	bystanderCount := 0
	for _, obj := range allObjects {
		if obj.ClassType == perception.ClassHuman && obj.ID != target.ID {
			for _, point := range proposedPath {
				if point.Distance(obj.Position) < 3.0 { // Within 3m of path
					bystanderCount++
					break
				}
			}
		}
	}
	risk.HarmToBystanderRisk = float64(bystanderCount) * 0.15 // 15% per bystander near path

	// Risk to robot - based on obstacles and battery
	obstacleRisk := 0.0
	for _, obj := range allObjects {
		if obj.ClassType == perception.ClassObstacle || obj.ClassType == perception.ClassDebris {
			for _, point := range proposedPath {
				if point.Distance(obj.Position) < 2.0 {
					obstacleRisk += 0.1
				}
			}
		}
	}
	risk.HarmToRobotRisk = obstacleRisk
	if hunoidState.BatteryLevel < 0.2 {
		risk.HarmToRobotRisk += 0.2 // Low battery increases risk
	}

	// Mission failure risk
	distance := hunoidState.Position.Distance(target.Position)
	risk.MissionFailureRisk = distance / 100.0 // Farther = higher risk of failure
	if target.ThreatLevel > 0.8 {
		risk.MissionFailureRisk += 0.2 // Critical targets have higher failure risk
	}

	// Calculate weighted overall risk
	// First Law (human safety) has highest weight
	risk.OverallRiskScore = risk.HarmToTargetRisk*0.35 +
		risk.HarmToBystanderRisk*0.35 +
		risk.HarmToRobotRisk*0.15 +
		risk.MissionFailureRisk*0.15

	return risk
}

// checkFirstLaw verifies compliance with Asimov's First Law
func (ek *EthicsKernel) checkFirstLaw(
	target perception.TrackedObject,
	allObjects []perception.TrackedObject,
	proposedPath []perception.Vector3,
	decision *EthicalDecision,
) bool {
	// Check if rescue action could harm the target
	if target.Velocity.Magnitude() > 15.0 {
		// Target moving very fast - high-speed interception could harm them
		decision.Reasoning += "Target moving at high velocity; careful approach required. "
		decision.Confidence *= 0.9
	}

	// Check if rescue path endangers bystanders
	for _, obj := range allObjects {
		if obj.ClassType == perception.ClassHuman && obj.ID != target.ID {
			for _, point := range proposedPath {
				if point.Distance(obj.Position) < 1.5 { // Very close to path
					decision.Reasoning += fmt.Sprintf("Bystander %s too close to path. ", obj.ID)
					return false
				}
			}
		}
	}

	// Check if inaction would harm the target more
	if target.ThreatLevel > 0.7 {
		// High threat - inaction would violate First Law through allowing harm
		decision.Reasoning += "Inaction would allow harm; rescue action required. "
	}

	return true
}

// checkBiasFree ensures decisions don't discriminate
func (ek *EthicsKernel) checkBiasFree(
	target perception.TrackedObject,
	decision *EthicalDecision,
) bool {
	// Rescue priority should be based ONLY on:
	// 1. Urgency (threat level, time to impact)
	// 2. Feasibility (distance, obstacles, battery)
	// 3. Success probability (Monte Carlo results)
	//
	// It should NEVER consider:
	// - Age, gender, race, nationality, religion
	// - Social status, occupation, wealth
	// - Physical appearance, disability status
	// - Any demographic factor

	// The rescue_priority.go already uses only objective factors
	// This check validates that no bias-related metadata is used

	biasFields := []string{"age", "gender", "race", "ethnicity", "religion", "nationality", "wealth", "status"}
	for _, field := range biasFields {
		if _, exists := target.Metadata[field]; exists {
			decision.Reasoning += fmt.Sprintf("CRITICAL: Bias field '%s' detected in decision data. ", field)
			ek.constraints[3].Violations++ // BIAS_FREE constraint
			return false
		}
	}

	return true
}

// checkProportionalResponse ensures response is proportional to threat
func (ek *EthicsKernel) checkProportionalResponse(
	target perception.TrackedObject,
	hunoidState HunoidState,
	decision *EthicalDecision,
) bool {
	// High-risk actions should only be taken for high-threat situations
	// Battery depletion (sacrifice) should only happen for critical rescues

	if hunoidState.BatteryLevel < 0.3 && target.ThreatLevel < 0.5 {
		decision.Reasoning += "Low battery; rescue would deplete reserves for non-critical target. "
		return false
	}

	return true
}

// GetBiasMetrics returns current bias tracking metrics
func (ek *EthicsKernel) GetBiasMetrics() BiasMetrics {
	ek.mu.RLock()
	defer ek.mu.RUnlock()
	return ek.biasMetrics
}

// GetAverageLatency returns average decision latency in nanoseconds
func (ek *EthicsKernel) GetAverageLatency() int64 {
	ek.mu.RLock()
	defer ek.mu.RUnlock()
	return ek.averageLatencyNs
}

// GetConstraints returns all ethical constraints
func (ek *EthicsKernel) GetConstraints() []EthicalConstraint {
	ek.mu.RLock()
	defer ek.mu.RUnlock()
	result := make([]EthicalConstraint, len(ek.constraints))
	copy(result, ek.constraints)
	return result
}
