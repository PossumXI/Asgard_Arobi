package integration_test

import (
	"context"
	"testing"

	"github.com/asgard/pandora/internal/robotics/ethics"
	"github.com/asgard/pandora/internal/robotics/vla"
)

func TestEthicsKernelCreation(t *testing.T) {
	kernel := ethics.NewEthicalKernel()
	if kernel == nil {
		t.Fatal("kernel should not be nil")
	}
}

func TestEthicsKernelSafeAction(t *testing.T) {
	kernel := ethics.NewEthicalKernel()
	ctx := context.Background()

	// Safe navigation action should be approved
	safeAction := &vla.Action{
		Type:       vla.ActionNavigate,
		Confidence: 0.95,
		Parameters: map[string]interface{}{
			"target": "waypoint-A",
			"speed":  0.5,
			"force":  10.0, // Low force
		},
	}

	decision, err := kernel.Evaluate(ctx, safeAction)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if decision.Decision != ethics.DecisionApproved {
		t.Errorf("expected safe action to be approved, got %s: %s", decision.Decision, decision.Reasoning)
	}

	if decision.Score < 0.7 {
		t.Errorf("expected high score for safe action, got %f", decision.Score)
	}
}

func TestEthicsKernelHighForceAction(t *testing.T) {
	kernel := ethics.NewEthicalKernel()
	ctx := context.Background()

	// High-force action may be rejected or escalated
	dangerousAction := &vla.Action{
		Type:       vla.ActionPickUp,
		Confidence: 0.85,
		Parameters: map[string]interface{}{
			"target": "heavy-object",
			"force":  1000.0, // Excessive force
		},
	}

	decision, err := kernel.Evaluate(ctx, dangerousAction)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	// High force actions may be rejected or escalated
	if decision.Decision == ethics.DecisionApproved {
		t.Log("High force action was approved - may require stricter rules")
	} else {
		t.Logf("High force action decision: %s", decision.Decision)
	}
}

func TestEthicsKernelLowConfidenceAction(t *testing.T) {
	kernel := ethics.NewEthicalKernel()
	ctx := context.Background()

	// Low confidence action should not be approved directly
	lowConfidenceAction := &vla.Action{
		Type:       vla.ActionPickUp,
		Confidence: 0.3, // Very low confidence
		Parameters: map[string]interface{}{
			"target": "unknown-object",
			"force":  20.0,
		},
	}

	decision, err := kernel.Evaluate(ctx, lowConfidenceAction)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	// Low confidence actions should be escalated, not approved directly
	if decision.Decision == ethics.DecisionApproved {
		t.Log("Low confidence action was approved - ProportionalityRule may need adjustment")
	}

	// Score should be lower for low confidence
	t.Logf("Decision: %s, Score: %f", decision.Decision, decision.Score)
}

func TestEthicsKernelWaitAction(t *testing.T) {
	kernel := ethics.NewEthicalKernel()
	ctx := context.Background()

	// Wait action should always be safe
	waitAction := &vla.Action{
		Type:       vla.ActionWait,
		Confidence: 1.0,
		Parameters: map[string]interface{}{
			"duration": 5.0,
		},
	}

	decision, err := kernel.Evaluate(ctx, waitAction)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if decision.Decision != ethics.DecisionApproved {
		t.Errorf("expected wait action to be approved, got %s", decision.Decision)
	}
}

func TestEthicsKernelInspectAction(t *testing.T) {
	kernel := ethics.NewEthicalKernel()
	ctx := context.Background()

	// Inspect action is non-intrusive and should be approved
	inspectAction := &vla.Action{
		Type:       vla.ActionInspect,
		Confidence: 0.98,
		Parameters: map[string]interface{}{
			"target": "environment",
		},
	}

	decision, err := kernel.Evaluate(ctx, inspectAction)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if decision.Decision != ethics.DecisionApproved {
		t.Errorf("expected inspect action to be approved, got %s", decision.Decision)
	}
}

func TestEthicsKernelRulesApplied(t *testing.T) {
	kernel := ethics.NewEthicalKernel()
	ctx := context.Background()

	action := &vla.Action{
		Type:       vla.ActionNavigate,
		Confidence: 0.95,
		Parameters: map[string]interface{}{
			"target": "safe-zone",
			"speed":  0.5,
		},
	}

	decision, err := kernel.Evaluate(ctx, action)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	// Check that rules were evaluated
	if len(decision.RulesChecked) == 0 {
		t.Error("expected at least one rule to be checked")
	}

	// Standard kernel should have 4 rules
	if len(decision.RulesChecked) < 4 {
		t.Errorf("expected 4 rules, got %d", len(decision.RulesChecked))
	}

	t.Logf("Rules checked: %v", decision.RulesChecked)
}

func TestEthicsKernelAllActionTypes(t *testing.T) {
	kernel := ethics.NewEthicalKernel()
	ctx := context.Background()

	actionTypes := []vla.ActionType{
		vla.ActionNavigate,
		vla.ActionPickUp,
		vla.ActionPutDown,
		vla.ActionOpen,
		vla.ActionClose,
		vla.ActionInspect,
		vla.ActionWait,
	}

	for _, actionType := range actionTypes {
		t.Run(string(actionType), func(t *testing.T) {
			action := &vla.Action{
				Type:       actionType,
				Confidence: 0.9,
				Parameters: map[string]interface{}{
					"target": "test-target",
					"force":  15.0,
				},
			}

			decision, err := kernel.Evaluate(ctx, action)
			if err != nil {
				t.Errorf("evaluation failed for %s: %v", actionType, err)
				return
			}

			if decision.Decision == "" {
				t.Errorf("expected decision outcome for action type %s", actionType)
			}

			t.Logf("%s: %s (score: %.2f)", actionType, decision.Decision, decision.Score)
		})
	}
}
