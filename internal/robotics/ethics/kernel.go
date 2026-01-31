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

// ConsentRule: Robot must respect autonomy and obtain proper consent
type ConsentRule struct {
	// consentRegistry stores consent status for interactions
	consentRegistry map[string]ConsentRecord
}

// ConsentRecord tracks consent status for a person/entity
type ConsentRecord struct {
	EntityID    string
	ConsentType string // "explicit", "implicit", "emergency_override"
	GrantedAt   time.Time
	ExpiresAt   time.Time
	Scope       []string // Actions covered by this consent
}

func (r *ConsentRule) Name() string {
	return "consent"
}

func (r *ConsentRule) Evaluate(ctx context.Context, action *vla.Action) (bool, string) {
	// Check if action involves human interaction
	targetPerson, involvesHuman := r.checkHumanInvolvement(action)
	if !involvesHuman {
		// No human involved, consent not required
		return true, ""
	}

	// Check for emergency override scenarios
	if r.isEmergencyScenario(action) {
		return true, "Emergency override: consent waived for safety"
	}

	// Verify consent exists for this interaction
	if !r.hasValidConsent(targetPerson, action) {
		return false, fmt.Sprintf("No valid consent from person: %s for action type: %s", targetPerson, action.Type)
	}

	// Check if action is within consented scope
	if !r.isWithinConsentScope(targetPerson, action) {
		return false, fmt.Sprintf("Action type %s not within consented scope for person: %s", action.Type, targetPerson)
	}

	return true, ""
}

// checkHumanInvolvement determines if the action involves human interaction
func (r *ConsentRule) checkHumanInvolvement(action *vla.Action) (string, bool) {
	// Check for person_id in action parameters
	if personID, ok := action.Parameters["person_id"].(string); ok && personID != "" {
		return personID, true
	}

	// Check for target indicating a person
	if target, ok := action.Parameters["target"].(string); ok {
		if r.targetIndicatesPerson(target) {
			return target, true
		}
	}

	// Check for interaction type
	if interactionType, ok := action.Parameters["interaction_type"].(string); ok {
		humanInteractionTypes := []string{"assist", "guide", "handoff", "communicate", "escort", "medical"}
		for _, hitType := range humanInteractionTypes {
			if interactionType == hitType {
				// Try to extract person identifier
				if personID, ok := action.Parameters["subject"].(string); ok {
					return personID, true
				}
				return "unknown_person", true
			}
		}
	}

	// Check specific action types that typically involve humans
	humanInvolvingActions := map[vla.ActionType]bool{
		vla.ActionPickUp: true, // Could involve taking something from/to a person
	}

	if humanInvolvingActions[action.Type] {
		// Check if target is a person or involves personal items
		if target, ok := action.Parameters["target"].(string); ok {
			if r.targetIndicatesPersonalInteraction(target) {
				return "unidentified_person", true
			}
		}
	}

	return "", false
}

// targetIndicatesPerson checks if target string suggests a person
func (r *ConsentRule) targetIndicatesPerson(target string) bool {
	personIndicators := []string{"person", "human", "patient", "user", "operator", "civilian", "subject"}
	targetLower := target
	for _, indicator := range personIndicators {
		if contains(targetLower, indicator) {
			return true
		}
	}
	return false
}

// targetIndicatesPersonalInteraction checks for personal item interaction
func (r *ConsentRule) targetIndicatesPersonalInteraction(target string) bool {
	personalIndicators := []string{"hand", "arm", "belonging", "personal", "from_person", "to_person"}
	for _, indicator := range personalIndicators {
		if contains(target, indicator) {
			return true
		}
	}
	return false
}

// isEmergencyScenario checks if the situation allows emergency consent override
func (r *ConsentRule) isEmergencyScenario(action *vla.Action) bool {
	// Check for emergency flag
	if emergency, ok := action.Parameters["emergency"].(bool); ok && emergency {
		return true
	}

	// Check for emergency context
	if context, ok := action.Parameters["context"].(string); ok {
		emergencyContexts := []string{"life_threatening", "medical_emergency", "rescue", "evacuation", "imminent_danger"}
		for _, ec := range emergencyContexts {
			if contains(context, ec) {
				return true
			}
		}
	}

	// Check priority level (1-2 indicates emergency)
	if priority, ok := action.Parameters["priority"].(int); ok && priority <= 2 {
		return true
	}

	return false
}

// hasValidConsent checks if valid consent exists for the person
func (r *ConsentRule) hasValidConsent(personID string, action *vla.Action) bool {
	// Check explicit consent in action parameters
	if consent, ok := action.Parameters["consent_granted"].(bool); ok && consent {
		return true
	}

	// Check consent token/reference
	if consentToken, ok := action.Parameters["consent_token"].(string); ok && consentToken != "" {
		// In production, validate token against consent registry
		return true
	}

	// Check for pre-authorized consent status
	if preAuth, ok := action.Parameters["pre_authorized"].(bool); ok && preAuth {
		return true
	}

	// Check consent registry if available
	if r.consentRegistry != nil {
		if record, exists := r.consentRegistry[personID]; exists {
			// Check if consent is still valid (not expired)
			if record.ExpiresAt.IsZero() || time.Now().Before(record.ExpiresAt) {
				return true
			}
		}
	}

	// For certain safe actions, implicit consent may apply
	if r.implicitConsentApplies(action) {
		return true
	}

	return false
}

// implicitConsentApplies checks if implicit consent can be assumed
func (r *ConsentRule) implicitConsentApplies(action *vla.Action) bool {
	// Non-contact observation actions may have implicit consent
	if action.Type == vla.ActionInspect || action.Type == vla.ActionWait {
		// Check if it's non-invasive
		if invasive, ok := action.Parameters["invasive"].(bool); ok && !invasive {
			return true
		}
		// Default inspect/wait are typically non-invasive
		if _, hasInvasive := action.Parameters["invasive"]; !hasInvasive {
			return true
		}
	}

	// Low-confidence interactions require explicit consent
	if action.Confidence < 0.8 {
		return false
	}

	return false
}

// isWithinConsentScope verifies the action is within consented boundaries
func (r *ConsentRule) isWithinConsentScope(personID string, action *vla.Action) bool {
	// Check if action scope is specified in parameters
	if scope, ok := action.Parameters["consent_scope"].([]string); ok {
		actionTypeStr := string(action.Type)
		for _, s := range scope {
			if s == actionTypeStr || s == "all" {
				return true
			}
		}
		return false
	}

	// Check consent registry for scope
	if r.consentRegistry != nil {
		if record, exists := r.consentRegistry[personID]; exists {
			actionTypeStr := string(action.Type)
			for _, s := range record.Scope {
				if s == actionTypeStr || s == "all" {
					return true
				}
			}
			// If registry exists but scope doesn't match, deny
			if len(record.Scope) > 0 {
				return false
			}
		}
	}

	// Default: assume within scope if consent is granted but no scope specified
	return true
}

// RegisterConsent adds a consent record to the registry
func (r *ConsentRule) RegisterConsent(entityID, consentType string, scope []string, duration time.Duration) {
	if r.consentRegistry == nil {
		r.consentRegistry = make(map[string]ConsentRecord)
	}

	var expiresAt time.Time
	if duration > 0 {
		expiresAt = time.Now().Add(duration)
	}

	r.consentRegistry[entityID] = ConsentRecord{
		EntityID:    entityID,
		ConsentType: consentType,
		GrantedAt:   time.Now(),
		ExpiresAt:   expiresAt,
		Scope:       scope,
	}
}

// RevokeConsent removes consent for an entity
func (r *ConsentRule) RevokeConsent(entityID string) {
	if r.consentRegistry != nil {
		delete(r.consentRegistry, entityID)
	}
}

// contains is a simple substring check helper
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
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
