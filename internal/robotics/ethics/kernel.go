package ethics

import (
	"context"
	"fmt"
	"time"

	"github.com/asgard/pandora/internal/robotics/vla"
	"github.com/google/uuid"
)

// EthicalDecision represents the result of ethical assessment
type EthicalDecision struct {
	ID             uuid.UUID
	Action         *vla.Action
	Decision       DecisionType
	Reasoning      string
	RulesChecked   []string
	Score          float64
	Timestamp      time.Time
	HumanReviewReq bool
}

// DecisionType represents ethical decision outcomes
type DecisionType string

const (
	DecisionApproved  DecisionType = "approved"
	DecisionRejected  DecisionType = "rejected"
	DecisionEscalated DecisionType = "escalated"
)

// EthicalKernel evaluates actions for ethical compliance
type EthicalKernel struct {
	rules []EthicalRule
}

// EthicalRule represents a constraint on behavior
type EthicalRule interface {
	Evaluate(ctx context.Context, action *vla.Action) (bool, string)
	Name() string
}

func NewEthicalKernel() *EthicalKernel {
	return &EthicalKernel{
		rules: []EthicalRule{
			&NoHarmRule{},
			&ConsentRule{},
			&ProportionalityRule{},
			&TransparencyRule{},
		},
	}
}

// Evaluate assesses an action against all ethical rules
func (k *EthicalKernel) Evaluate(ctx context.Context, action *vla.Action) (*EthicalDecision, error) {
	decision := &EthicalDecision{
		ID:           uuid.New(),
		Action:       action,
		RulesChecked: make([]string, 0),
		Timestamp:    time.Now().UTC(),
		Score:        1.0,
	}

	violationCount := 0
	var rejectionReasons []string

	for _, rule := range k.rules {
		passed, reason := rule.Evaluate(ctx, action)
		decision.RulesChecked = append(decision.RulesChecked, rule.Name())

		if !passed {
			violationCount++
			rejectionReasons = append(rejectionReasons, reason)
			decision.Score -= 0.25
		}
	}

	// Make decision
	if violationCount == 0 {
		decision.Decision = DecisionApproved
		decision.Reasoning = "All ethical rules satisfied"
	} else if violationCount >= 2 {
		decision.Decision = DecisionRejected
		decision.Reasoning = fmt.Sprintf("Multiple rule violations: %v", rejectionReasons)
	} else {
		decision.Decision = DecisionEscalated
		decision.Reasoning = fmt.Sprintf("Escalated for review: %v", rejectionReasons)
		decision.HumanReviewReq = true
	}

	return decision, nil
}

// NoHarmRule: Robot must not cause physical harm
type NoHarmRule struct{}

func (r *NoHarmRule) Name() string {
	return "no_harm"
}

func (r *NoHarmRule) Evaluate(ctx context.Context, action *vla.Action) (bool, string) {
	// Check for potentially harmful actions
	if action.Type == vla.ActionPickUp {
		if force, ok := action.Parameters["force"].(string); ok {
			if force == "aggressive" || force == "maximum" {
				return false, "Excessive force could cause harm"
			}
		}
	}

	return true, ""
}

// ConsentRule: Robot must respect autonomy
type ConsentRule struct{}

func (r *ConsentRule) Name() string {
	return "consent"
}

func (r *ConsentRule) Evaluate(ctx context.Context, action *vla.Action) (bool, string) {
	// In production, this would check if action involves a person
	// and verify consent has been obtained
	return true, ""
}

// ProportionalityRule: Response must be proportional to situation
type ProportionalityRule struct{}

func (r *ProportionalityRule) Name() string {
	return "proportionality"
}

func (r *ProportionalityRule) Evaluate(ctx context.Context, action *vla.Action) (bool, string) {
	// Check if action confidence is too low for critical actions
	if action.Confidence < 0.6 && (action.Type == vla.ActionPickUp || action.Type == vla.ActionNavigate) {
		return false, fmt.Sprintf("Confidence too low (%.2f) for critical action", action.Confidence)
	}

	return true, ""
}

// TransparencyRule: Actions must be explainable
type TransparencyRule struct{}

func (r *TransparencyRule) Name() string {
	return "transparency"
}

func (r *TransparencyRule) Evaluate(ctx context.Context, action *vla.Action) (bool, string) {
	// Ensure action has clear parameters
	if len(action.Parameters) == 0 && action.Type != vla.ActionWait {
		return false, "Action lacks clear parameters for transparency"
	}

	return true, ""
}
