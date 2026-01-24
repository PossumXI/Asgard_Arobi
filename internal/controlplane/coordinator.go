package controlplane

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Coordinator handles cross-domain decisions based on policies.
type Coordinator struct {
	mu sync.RWMutex

	// Reference to control plane
	controlPlane *UnifiedControlPlane

	// Active policies
	policies []CoordinationPolicy

	// Active responses
	activeResponses map[uuid.UUID]*CoordinationResponse

	// Configuration
	config CoordinatorConfig

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics
	metrics *CoordinatorMetrics
}

// CoordinatorConfig holds coordinator configuration.
type CoordinatorConfig struct {
	ResponseTimeout     time.Duration
	MaxConcurrentActions int
	PolicyEvalInterval  time.Duration
}

// DefaultCoordinatorConfig returns sensible defaults.
func DefaultCoordinatorConfig() CoordinatorConfig {
	return CoordinatorConfig{
		ResponseTimeout:      30 * time.Second,
		MaxConcurrentActions: 10,
		PolicyEvalInterval:   5 * time.Second,
	}
}

// CoordinatorMetrics tracks coordinator performance.
type CoordinatorMetrics struct {
	mu                sync.RWMutex
	PoliciesEvaluated int64 `json:"policies_evaluated"`
	PoliciesTriggered int64 `json:"policies_triggered"`
	ActionsExecuted   int64 `json:"actions_executed"`
	ActionsSucceeded  int64 `json:"actions_succeeded"`
	ActionsFailed     int64 `json:"actions_failed"`
	LastEvaluation    time.Time `json:"last_evaluation"`
}

// CoordinationPolicy defines rules for cross-domain coordination.
type CoordinationPolicy struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Priority    int                     `json:"priority"` // Higher = more urgent
	Enabled     bool                    `json:"enabled"`
	TriggerType CrossDomainEventType    `json:"trigger_type,omitempty"`
	Condition   PolicyCondition         `json:"-"`
	Actions     []PolicyAction          `json:"actions"`
	Cooldown    time.Duration           `json:"cooldown_ns"`
	LastTriggered time.Time             `json:"last_triggered"`
}

// PolicyCondition evaluates whether a policy should trigger.
type PolicyCondition func(event CrossDomainEvent, ctx *PolicyContext) bool

// PolicyAction defines an action to take when a policy triggers.
type PolicyAction struct {
	TargetDomain EventDomain            `json:"target_domain"`
	CommandType  string                 `json:"command_type"`
	Parameters   map[string]interface{} `json:"parameters"`
	Async        bool                   `json:"async"`
}

// PolicyContext provides context for policy evaluation.
type PolicyContext struct {
	RecentEvents    []CrossDomainEvent
	SystemStatus    map[string]*SystemStatus
	HealthStatus    *HealthStatus
	ActiveThreats   int
	CongestionLevel float64
}

// CoordinationResponse tracks an ongoing coordination response.
type CoordinationResponse struct {
	ID          uuid.UUID              `json:"id"`
	PolicyID    string                 `json:"policy_id"`
	TriggerEvent CrossDomainEvent      `json:"trigger_event"`
	Actions     []ActionStatus         `json:"actions"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time,omitempty"`
	Status      string                 `json:"status"` // pending, executing, completed, failed
}

// ActionStatus tracks the status of an individual action.
type ActionStatus struct {
	Action    PolicyAction `json:"action"`
	Status    string       `json:"status"` // pending, executing, completed, failed
	StartTime time.Time    `json:"start_time,omitempty"`
	EndTime   time.Time    `json:"end_time,omitempty"`
	Error     string       `json:"error,omitempty"`
}

// NewCoordinator creates a new coordinator.
func NewCoordinator(cp *UnifiedControlPlane) *Coordinator {
	ctx, cancel := context.WithCancel(context.Background())

	c := &Coordinator{
		controlPlane:    cp,
		policies:        make([]CoordinationPolicy, 0),
		activeResponses: make(map[uuid.UUID]*CoordinationResponse),
		config:          DefaultCoordinatorConfig(),
		ctx:             ctx,
		cancel:          cancel,
		metrics: &CoordinatorMetrics{
			LastEvaluation: time.Now(),
		},
	}

	// Register default policies
	c.registerDefaultPolicies()

	return c
}

// Start begins coordinator operations.
func (c *Coordinator) Start() {
	c.wg.Add(1)
	go c.runResponseCleanup()
	log.Println("[Coordinator] Started")
}

// Stop gracefully shuts down the coordinator.
func (c *Coordinator) Stop() {
	c.cancel()
	c.wg.Wait()
	log.Println("[Coordinator] Stopped")
}

// HandleEvent evaluates an event against all policies.
func (c *Coordinator) HandleEvent(event CrossDomainEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Build policy context
	ctx := c.buildPolicyContext()

	// Evaluate each policy
	for i := range c.policies {
		policy := &c.policies[i]

		if !policy.Enabled {
			continue
		}

		// Check cooldown
		if time.Since(policy.LastTriggered) < policy.Cooldown {
			continue
		}

		// Check trigger type match if specified
		if policy.TriggerType != "" && policy.TriggerType != event.Type {
			continue
		}

		c.metrics.mu.Lock()
		c.metrics.PoliciesEvaluated++
		c.metrics.mu.Unlock()

		// Evaluate condition
		if policy.Condition != nil && policy.Condition(event, ctx) {
			c.triggerPolicy(policy, event)
		}
	}

	c.metrics.mu.Lock()
	c.metrics.LastEvaluation = time.Now()
	c.metrics.mu.Unlock()
}

// RegisterPolicy adds a new coordination policy.
func (c *Coordinator) RegisterPolicy(policy CoordinationPolicy) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.policies = append(c.policies, policy)
	log.Printf("[Coordinator] Registered policy: %s", policy.Name)
}

// DisablePolicy disables a policy by ID.
func (c *Coordinator) DisablePolicy(policyID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.policies {
		if c.policies[i].ID == policyID {
			c.policies[i].Enabled = false
			log.Printf("[Coordinator] Disabled policy: %s", policyID)
			return
		}
	}
}

// EnablePolicy enables a policy by ID.
func (c *Coordinator) EnablePolicy(policyID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.policies {
		if c.policies[i].ID == policyID {
			c.policies[i].Enabled = true
			log.Printf("[Coordinator] Enabled policy: %s", policyID)
			return
		}
	}
}

// GetPolicies returns all registered policies.
func (c *Coordinator) GetPolicies() []CoordinationPolicy {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]CoordinationPolicy, len(c.policies))
	copy(result, c.policies)
	return result
}

// GetActiveResponses returns all active coordination responses.
func (c *Coordinator) GetActiveResponses() []*CoordinationResponse {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*CoordinationResponse, 0, len(c.activeResponses))
	for _, r := range c.activeResponses {
		result = append(result, r)
	}
	return result
}

// GetMetrics returns coordinator metrics.
func (c *Coordinator) GetMetrics() *CoordinatorMetrics {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()

	// Copy metrics fields individually to avoid copying the mutex
	return &CoordinatorMetrics{
		PoliciesEvaluated: c.metrics.PoliciesEvaluated,
		PoliciesTriggered: c.metrics.PoliciesTriggered,
		ActionsExecuted:   c.metrics.ActionsExecuted,
		ActionsSucceeded:  c.metrics.ActionsSucceeded,
		ActionsFailed:     c.metrics.ActionsFailed,
		LastEvaluation:    c.metrics.LastEvaluation,
	}
}

// buildPolicyContext creates context for policy evaluation.
func (c *Coordinator) buildPolicyContext() *PolicyContext {
	ctx := &PolicyContext{
		RecentEvents: c.controlPlane.GetRecentEvents(100),
		SystemStatus: c.controlPlane.GetStatus(),
		HealthStatus: c.controlPlane.GetHealth(),
	}

	// Count active threats
	for _, event := range ctx.RecentEvents {
		if event.Type == EventSecurityThreat && event.Severity >= SeverityHigh {
			ctx.ActiveThreats++
		}
	}

	// Calculate congestion level
	congestionEvents := 0
	for _, event := range ctx.RecentEvents {
		if event.Type == EventDTNCongestion {
			congestionEvents++
		}
	}
	if len(ctx.RecentEvents) > 0 {
		ctx.CongestionLevel = float64(congestionEvents) / float64(len(ctx.RecentEvents))
	}

	return ctx
}

// triggerPolicy executes a policy's actions.
func (c *Coordinator) triggerPolicy(policy *CoordinationPolicy, triggerEvent CrossDomainEvent) {
	log.Printf("[Coordinator] Policy triggered: %s (event: %s)", policy.Name, triggerEvent.Type)

	policy.LastTriggered = time.Now()

	c.metrics.mu.Lock()
	c.metrics.PoliciesTriggered++
	c.metrics.mu.Unlock()

	// Create coordination response
	response := &CoordinationResponse{
		ID:           uuid.New(),
		PolicyID:     policy.ID,
		TriggerEvent: triggerEvent,
		Actions:      make([]ActionStatus, len(policy.Actions)),
		StartTime:    time.Now(),
		Status:       "executing",
	}

	for i, action := range policy.Actions {
		response.Actions[i] = ActionStatus{
			Action: action,
			Status: "pending",
		}
	}

	c.activeResponses[response.ID] = response

	// Execute actions
	go c.executeActions(response, policy.Actions)
}

// executeActions runs policy actions.
func (c *Coordinator) executeActions(response *CoordinationResponse, actions []PolicyAction) {
	for i, action := range actions {
		c.mu.Lock()
		response.Actions[i].Status = "executing"
		response.Actions[i].StartTime = time.Now()
		c.mu.Unlock()

		// Build command
		cmd := NewControlCommand(
			"coordinator",
			action.TargetDomain,
			action.CommandType,
			action.Parameters,
			5, // Default priority
		)

		// Add context from trigger event
		if response.TriggerEvent.ID != uuid.Nil {
			if cmd.Parameters == nil {
				cmd.Parameters = make(map[string]interface{})
			}
			cmd.Parameters["trigger_event_id"] = response.TriggerEvent.ID.String()
			cmd.Parameters["trigger_type"] = string(response.TriggerEvent.Type)
		}

		// Execute command
		resp, err := c.controlPlane.IssueCommand(cmd)

		c.mu.Lock()
		response.Actions[i].EndTime = time.Now()

		if err != nil {
			response.Actions[i].Status = "failed"
			response.Actions[i].Error = err.Error()
			c.metrics.mu.Lock()
			c.metrics.ActionsFailed++
			c.metrics.mu.Unlock()
		} else if !resp.Success {
			response.Actions[i].Status = "failed"
			response.Actions[i].Error = resp.Error
			c.metrics.mu.Lock()
			c.metrics.ActionsFailed++
			c.metrics.mu.Unlock()
		} else {
			response.Actions[i].Status = "completed"
			c.metrics.mu.Lock()
			c.metrics.ActionsSucceeded++
			c.metrics.mu.Unlock()
		}

		c.metrics.mu.Lock()
		c.metrics.ActionsExecuted++
		c.metrics.mu.Unlock()
		c.mu.Unlock()
	}

	// Mark response as completed
	c.mu.Lock()
	response.EndTime = time.Now()
	allSuccess := true
	for _, action := range response.Actions {
		if action.Status == "failed" {
			allSuccess = false
			break
		}
	}
	if allSuccess {
		response.Status = "completed"
	} else {
		response.Status = "partial_failure"
	}
	c.mu.Unlock()

	log.Printf("[Coordinator] Response %s completed (status: %s)", response.ID.String()[:8], response.Status)
}

// runResponseCleanup periodically cleans up old responses.
func (c *Coordinator) runResponseCleanup() {
	defer c.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.cleanupOldResponses()
		}
	}
}

// cleanupOldResponses removes completed responses older than 1 hour.
func (c *Coordinator) cleanupOldResponses() {
	c.mu.Lock()
	defer c.mu.Unlock()

	cutoff := time.Now().Add(-1 * time.Hour)

	for id, response := range c.activeResponses {
		if response.Status != "executing" && response.EndTime.Before(cutoff) {
			delete(c.activeResponses, id)
		}
	}
}

// registerDefaultPolicies sets up built-in coordination policies.
func (c *Coordinator) registerDefaultPolicies() {
	// Policy 1: Security threat pauses autonomous operations
	c.policies = append(c.policies, CoordinationPolicy{
		ID:          "security-halt-autonomy",
		Name:        "Security Threat Halt",
		Description: "Pauses all autonomous operations when a critical security threat is detected",
		Priority:    100,
		Enabled:     true,
		TriggerType: EventSecurityThreat,
		Condition: func(event CrossDomainEvent, ctx *PolicyContext) bool {
			return event.Severity == SeverityCritical || event.Severity == SeverityHigh
		},
		Actions: []PolicyAction{
			{
				TargetDomain: DomainAutonomy,
				CommandType:  "halt_all",
				Parameters:   map[string]interface{}{"reason": "security_threat"},
			},
		},
		Cooldown: 5 * time.Minute,
	})

	// Policy 2: DTN congestion adjusts bundle priorities
	c.policies = append(c.policies, CoordinationPolicy{
		ID:          "dtn-congestion-priority",
		Name:        "DTN Congestion Management",
		Description: "Adjusts bundle priorities when DTN nodes are congested",
		Priority:    50,
		Enabled:     true,
		TriggerType: EventDTNCongestion,
		Condition: func(event CrossDomainEvent, ctx *PolicyContext) bool {
			if congestion, ok := event.Payload["queue_utilization"].(float64); ok {
				return congestion > 0.8
			}
			return false
		},
		Actions: []PolicyAction{
			{
				TargetDomain: DomainDTN,
				CommandType:  "adjust_priority",
				Parameters: map[string]interface{}{
					"mode":                "congestion_relief",
					"drop_low_priority":   true,
					"expedite_critical":   true,
				},
			},
		},
		Cooldown: 2 * time.Minute,
	})

	// Policy 3: Ethics escalation notifies all systems
	c.policies = append(c.policies, CoordinationPolicy{
		ID:          "ethics-escalation-notify",
		Name:        "Ethics Escalation Notification",
		Description: "Notifies all systems when an ethics decision requires human review",
		Priority:    90,
		Enabled:     true,
		TriggerType: EventEthicsEscalation,
		Condition: func(event CrossDomainEvent, ctx *PolicyContext) bool {
			return event.RequiresAck
		},
		Actions: []PolicyAction{
			{
				TargetDomain: DomainAutonomy,
				CommandType:  "halt",
				Parameters: map[string]interface{}{
					"reason": "ethics_review_required",
					"wait_for_resolution": true,
				},
			},
			{
				TargetDomain: DomainSecurity,
				CommandType:  "escalate_threat",
				Parameters: map[string]interface{}{
					"type":        "ethics_escalation",
					"notify_human": true,
				},
			},
		},
		Cooldown: 1 * time.Minute,
	})

	// Policy 4: System offline triggers DTN rerouting
	c.policies = append(c.policies, CoordinationPolicy{
		ID:          "system-offline-reroute",
		Name:        "System Offline Rerouting",
		Description: "Reroutes DTN traffic when a system goes offline",
		Priority:    60,
		Enabled:     true,
		TriggerType: EventAutonomyStatus,
		Condition: func(event CrossDomainEvent, ctx *PolicyContext) bool {
			if state, ok := event.Payload["state"].(string); ok {
				return state == string(StateOffline) || state == string(StateEmergency)
			}
			return false
		},
		Actions: []PolicyAction{
			{
				TargetDomain: DomainDTN,
				CommandType:  "pause_node",
				Parameters: map[string]interface{}{
					"reason": "system_offline",
				},
			},
		},
		Cooldown: 30 * time.Second,
	})

	// Policy 5: Threat mitigation resumes operations
	c.policies = append(c.policies, CoordinationPolicy{
		ID:          "threat-mitigated-resume",
		Name:        "Threat Mitigation Resume",
		Description: "Resumes autonomous operations after a threat is mitigated",
		Priority:    80,
		Enabled:     true,
		TriggerType: EventSecurityMitigated,
		Condition: func(event CrossDomainEvent, ctx *PolicyContext) bool {
			return ctx.ActiveThreats == 0
		},
		Actions: []PolicyAction{
			{
				TargetDomain: DomainAutonomy,
				CommandType:  "resume",
				Parameters: map[string]interface{}{
					"reason": "threat_cleared",
				},
			},
			{
				TargetDomain: DomainSecurity,
				CommandType:  "resume_scanning",
				Parameters:   map[string]interface{}{},
			},
		},
		Cooldown: 1 * time.Minute,
	})

	// Policy 6: Multiple threats escalate to emergency mode
	c.policies = append(c.policies, CoordinationPolicy{
		ID:          "multi-threat-emergency",
		Name:        "Multi-Threat Emergency",
		Description: "Triggers emergency mode when multiple active threats detected",
		Priority:    110,
		Enabled:     true,
		TriggerType: EventSecurityThreat,
		Condition: func(event CrossDomainEvent, ctx *PolicyContext) bool {
			return ctx.ActiveThreats >= 3
		},
		Actions: []PolicyAction{
			{
				TargetDomain: DomainAutonomy,
				CommandType:  "emergency_stop",
				Parameters: map[string]interface{}{
					"reason": "multi_threat_emergency",
					"preserve_state": true,
				},
			},
			{
				TargetDomain: DomainDTN,
				CommandType:  "adjust_priority",
				Parameters: map[string]interface{}{
					"mode":        "emergency",
					"critical_only": true,
				},
			},
		},
		Cooldown: 10 * time.Minute,
	})

	log.Printf("[Coordinator] Registered %d default policies", len(c.policies))
}
