package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/asgard/pandora/internal/platform/observability"
	"github.com/asgard/pandora/internal/robotics/control"
	"github.com/asgard/pandora/internal/robotics/ethics"
	"github.com/asgard/pandora/internal/robotics/vla"
)

type MissionPlan struct {
	ID        string
	Name      string
	Objective string
	RiskLevel RiskLevel
	Steps     []MissionStep
	CreatedAt time.Time
}

type MissionStep struct {
	ID                string
	Command           string
	Criticality       Criticality
	RequiresConsent   bool
	AllowAutoApproval bool
	HazardLevel       int
}

type MissionReport struct {
	MissionID        string
	MissionName      string
	Objective        string
	StartedAt        time.Time
	CompletedAt      time.Time
	StepResults      []StepResult
	Interventions    []InterventionDecision
	EthicsDecisions  []ethics.EthicalDecision
	PolicyDecisions  []PolicyDecision
	OperatorActions  []OperatorAction
	BlockedStepCount int
	AutoApproved     int
	ManualApproved   int
}

type StepResult struct {
	StepID   string
	Command  string
	Action   vla.ActionType
	Outcome  string
	Duration time.Duration
}

type Criticality string

const (
	CriticalityLow    Criticality = "low"
	CriticalityMedium Criticality = "medium"
	CriticalityHigh   Criticality = "high"
)

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type PolicyDecision struct {
	Decision PolicyDecisionType
	Reasons  []string
	Score    float64
}

type PolicyDecisionType string

const (
	PolicyApproved PolicyDecisionType = "approved"
	PolicyBlocked  PolicyDecisionType = "blocked"
	PolicyHold     PolicyDecisionType = "hold"
)

type InterventionDecision struct {
	Action            InterventionAction
	Reason            string
	RequiresApproval  bool
	OperatorTimeout   time.Duration
	DecisionTimestamp time.Time
}

type InterventionAction string

const (
	InterventionProceed InterventionAction = "proceed"
	InterventionHold    InterventionAction = "hold"
	InterventionAbort   InterventionAction = "abort"
)

type OperatorAction struct {
	Timestamp time.Time
	Action    string
	Payload   string
}

type StatusSnapshot struct {
	Timestamp         time.Time `json:"timestamp"`
	MissionID         string    `json:"mission_id"`
	MissionName       string    `json:"mission_name"`
	Objective         string    `json:"objective"`
	CurrentStepID     string    `json:"current_step_id"`
	CurrentCommand    string    `json:"current_command"`
	CurrentAction     string    `json:"current_action"`
	CurrentConfidence float64   `json:"current_confidence"`
	LastOutcome       string    `json:"last_outcome"`
	PendingApproval   string    `json:"pending_approval"`
	LastIntervention  string    `json:"last_intervention"`
	LastPolicy        string    `json:"last_policy"`
	LastEthics        string    `json:"last_ethics"`
	Paused            bool      `json:"paused"`
	Aborted           bool      `json:"aborted"`
	OperatorMode      string    `json:"operator_mode"`
	Events            []string  `json:"events"`
}

type MissionState struct {
	mu                sync.RWMutex
	missionID         string
	missionName       string
	objective         string
	currentStepID     string
	currentCommand    string
	currentAction     string
	currentConfidence float64
	lastOutcome       string
	pendingApproval   string
	lastIntervention  string
	lastPolicy        string
	lastEthics        string
	updatedAt         time.Time
	events            []string
}

func NewMissionState() *MissionState {
	return &MissionState{
		events: make([]string, 0, 20),
	}
}

func (s *MissionState) SetMission(mission *MissionPlan) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.missionID = mission.ID
	s.missionName = mission.Name
	s.objective = mission.Objective
	s.updatedAt = time.Now().UTC()
}

func (s *MissionState) SetStep(step MissionStep) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentStepID = step.ID
	s.currentCommand = step.Command
	s.updatedAt = time.Now().UTC()
}

func (s *MissionState) SetAction(action *vla.Action) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.currentAction = string(action.Type)
	s.currentConfidence = action.Confidence
	s.updatedAt = time.Now().UTC()
}

func (s *MissionState) SetDecisions(ethicsDecision *ethics.EthicalDecision, policyDecision PolicyDecision, intervention InterventionDecision) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ethicsDecision != nil {
		s.lastEthics = string(ethicsDecision.Decision)
	}
	s.lastPolicy = string(policyDecision.Decision)
	s.lastIntervention = string(intervention.Action)
	s.updatedAt = time.Now().UTC()
}

func (s *MissionState) SetPendingApproval(stepID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingApproval = stepID
	s.updatedAt = time.Now().UTC()
}

func (s *MissionState) ClearPendingApproval() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingApproval = ""
	s.updatedAt = time.Now().UTC()
}

func (s *MissionState) SetOutcome(outcome string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastOutcome = outcome
	s.updatedAt = time.Now().UTC()
}

func (s *MissionState) AddEvent(event string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if event == "" {
		return
	}
	if len(s.events) >= 20 {
		s.events = s.events[1:]
	}
	s.events = append(s.events, event)
	s.updatedAt = time.Now().UTC()
}

func (s *MissionState) Snapshot(operator *OperatorConsole) StatusSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	paused := false
	aborted := false
	mode := ""
	if operator != nil {
		paused, aborted = operator.Snapshot()
		mode = operator.mode
	}

	events := make([]string, len(s.events))
	copy(events, s.events)

	return StatusSnapshot{
		Timestamp:         s.updatedAt,
		MissionID:         s.missionID,
		MissionName:       s.missionName,
		Objective:         s.objective,
		CurrentStepID:     s.currentStepID,
		CurrentCommand:    s.currentCommand,
		CurrentAction:     s.currentAction,
		CurrentConfidence: s.currentConfidence,
		LastOutcome:       s.lastOutcome,
		PendingApproval:   s.pendingApproval,
		LastIntervention:  s.lastIntervention,
		LastPolicy:        s.lastPolicy,
		LastEthics:        s.lastEthics,
		Paused:            paused,
		Aborted:           aborted,
		OperatorMode:      mode,
		Events:            events,
	}
}

type AuditEvent struct {
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"`
	MissionID string                 `json:"mission_id,omitempty"`
	StepID    string                 `json:"step_id,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

type MissionPlanner struct{}

func NewMissionPlanner() *MissionPlanner {
	return &MissionPlanner{}
}

func (p *MissionPlanner) BuildScenario(name string) (*MissionPlan, error) {
	switch strings.ToLower(name) {
	case "medical_aid":
		return &MissionPlan{
			ID:        "mission-medical-aid",
			Name:      "Medical Aid Delivery",
			Objective: "Deliver critical medical kit while assessing hazards.",
			RiskLevel: RiskMedium,
			CreatedAt: time.Now().UTC(),
			Steps: []MissionStep{
				{ID: "step-1", Command: "Navigate to the supply depot", Criticality: CriticalityMedium, HazardLevel: 1},
				{ID: "step-2", Command: "Pick up the medical kit", Criticality: CriticalityHigh, RequiresConsent: false, AllowAutoApproval: false, HazardLevel: 1},
				{ID: "step-3", Command: "Move to the injured person", Criticality: CriticalityHigh, HazardLevel: 2},
				{ID: "step-4", Command: "Put down the medical kit gently", Criticality: CriticalityMedium, RequiresConsent: true, AllowAutoApproval: false, HazardLevel: 2},
				{ID: "step-5", Command: "Inspect the area for hazards", Criticality: CriticalityLow, AllowAutoApproval: true, HazardLevel: 3},
			},
		}, nil
	case "perimeter_check":
		return &MissionPlan{
			ID:        "mission-perimeter-check",
			Name:      "Perimeter Check",
			Objective: "Validate perimeter safety and report anomalies.",
			RiskLevel: RiskLow,
			CreatedAt: time.Now().UTC(),
			Steps: []MissionStep{
				{ID: "step-1", Command: "Navigate to checkpoint alpha", Criticality: CriticalityLow, AllowAutoApproval: true, HazardLevel: 1},
				{ID: "step-2", Command: "Inspect the area for hazards", Criticality: CriticalityLow, AllowAutoApproval: true, HazardLevel: 1},
				{ID: "step-3", Command: "Navigate to checkpoint beta", Criticality: CriticalityMedium, AllowAutoApproval: true, HazardLevel: 2},
				{ID: "step-4", Command: "Inspect the area for hazards", Criticality: CriticalityMedium, AllowAutoApproval: true, HazardLevel: 2},
			},
		}, nil
	case "hazard_response":
		return &MissionPlan{
			ID:        "mission-hazard-response",
			Name:      "Hazard Response",
			Objective: "Investigate and mitigate hazards with operator oversight.",
			RiskLevel: RiskHigh,
			CreatedAt: time.Now().UTC(),
			Steps: []MissionStep{
				{ID: "step-1", Command: "Navigate to the hazard zone", Criticality: CriticalityHigh, AllowAutoApproval: false, HazardLevel: 3},
				{ID: "step-2", Command: "Inspect the area for hazards", Criticality: CriticalityHigh, AllowAutoApproval: false, HazardLevel: 3},
				{ID: "step-3", Command: "Move to the containment unit", Criticality: CriticalityHigh, AllowAutoApproval: false, HazardLevel: 2},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unknown scenario: %s", name)
	}
}

type ActionRegistry struct {
	handlers map[vla.ActionType]func(context.Context, *vla.Action) error
}

func NewActionRegistry() *ActionRegistry {
	return &ActionRegistry{
		handlers: make(map[vla.ActionType]func(context.Context, *vla.Action) error),
	}
}

func (r *ActionRegistry) Register(actionType vla.ActionType, handler func(context.Context, *vla.Action) error) {
	r.handlers[actionType] = handler
}

func (r *ActionRegistry) Execute(ctx context.Context, action *vla.Action) error {
	handler, exists := r.handlers[action.Type]
	if !exists {
		return fmt.Errorf("no handler for action type: %s", action.Type)
	}
	return handler(ctx, action)
}

type SafetyPolicyEngine struct {
	minBatteryPercent float64
}

func NewSafetyPolicyEngine(minBatteryPercent float64) *SafetyPolicyEngine {
	return &SafetyPolicyEngine{minBatteryPercent: minBatteryPercent}
}

func (e *SafetyPolicyEngine) Evaluate(action *vla.Action, step MissionStep, battery float64) PolicyDecision {
	reasons := make([]string, 0)
	score := 1.0

	if battery < e.minBatteryPercent && action.Type == vla.ActionNavigate {
		reasons = append(reasons, "Battery too low for navigation action")
		score -= 0.4
	}

	if step.HazardLevel >= 3 && action.Type == vla.ActionNavigate {
		reasons = append(reasons, "High hazard level requires operator oversight")
		score -= 0.3
	}

	if step.RequiresConsent {
		reasons = append(reasons, "Consent required for this step")
		score -= 0.2
	}

	if len(reasons) == 0 {
		return PolicyDecision{Decision: PolicyApproved, Score: score}
	}

	if score <= 0.3 {
		return PolicyDecision{Decision: PolicyBlocked, Reasons: reasons, Score: score}
	}

	return PolicyDecision{Decision: PolicyHold, Reasons: reasons, Score: score}
}

type InterventionEngine struct {
	lowConfidenceThreshold float64
	defaultApprovalTimeout time.Duration
}

func NewInterventionEngine(lowConfidenceThreshold float64, approvalTimeout time.Duration) *InterventionEngine {
	return &InterventionEngine{
		lowConfidenceThreshold: lowConfidenceThreshold,
		defaultApprovalTimeout: approvalTimeout,
	}
}

func (i *InterventionEngine) Decide(action *vla.Action, ethicsDecision *ethics.EthicalDecision, policyDecision PolicyDecision, step MissionStep) InterventionDecision {
	if ethicsDecision.Decision == ethics.DecisionRejected {
		return InterventionDecision{
			Action:            InterventionAbort,
			Reason:            ethicsDecision.Reasoning,
			DecisionTimestamp: time.Now().UTC(),
		}
	}

	if policyDecision.Decision == PolicyBlocked {
		return InterventionDecision{
			Action:            InterventionAbort,
			Reason:            strings.Join(policyDecision.Reasons, "; "),
			DecisionTimestamp: time.Now().UTC(),
		}
	}

	if ethicsDecision.Decision == ethics.DecisionEscalated {
		return InterventionDecision{
			Action:            InterventionHold,
			Reason:            ethicsDecision.Reasoning,
			RequiresApproval:  true,
			OperatorTimeout:   i.defaultApprovalTimeout,
			DecisionTimestamp: time.Now().UTC(),
		}
	}

	if policyDecision.Decision == PolicyHold {
		return InterventionDecision{
			Action:            InterventionHold,
			Reason:            strings.Join(policyDecision.Reasons, "; "),
			RequiresApproval:  true,
			OperatorTimeout:   i.defaultApprovalTimeout,
			DecisionTimestamp: time.Now().UTC(),
		}
	}

	if action.Confidence < i.lowConfidenceThreshold || step.Criticality == CriticalityHigh {
		return InterventionDecision{
			Action:            InterventionHold,
			Reason:            "Low confidence or high criticality action requires approval",
			RequiresApproval:  true,
			OperatorTimeout:   i.defaultApprovalTimeout,
			DecisionTimestamp: time.Now().UTC(),
		}
	}

	return InterventionDecision{
		Action:            InterventionProceed,
		DecisionTimestamp: time.Now().UTC(),
	}
}

type OperatorConsole struct {
	mode             string
	autoApproveDelay time.Duration
	approvals        chan string
	commands         chan OperatorAction
	injectedCommands chan string
	statusMu         sync.RWMutex
	paused           bool
	aborted          bool
}

func NewOperatorConsole(mode string, autoApproveDelay time.Duration) *OperatorConsole {
	return &OperatorConsole{
		mode:             mode,
		autoApproveDelay: autoApproveDelay,
		approvals:        make(chan string, 4),
		commands:         make(chan OperatorAction, 10),
		injectedCommands: make(chan string, 4),
	}
}

func (o *OperatorConsole) Start(ctx context.Context) {
	if strings.ToLower(o.mode) == "disabled" {
		return
	}

	go o.readInput(ctx)
}

func (o *OperatorConsole) readInput(ctx context.Context) {
	scanner := bufio.NewScanner(os.Stdin)
	log.Println("Operator console ready. Type 'help' for commands.")
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		o.handleCommand(line)
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func (o *OperatorConsole) handleCommand(line string) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return
	}

	command := strings.ToLower(fields[0])
	payload := strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
	if err := o.ApplyCommand(command, payload); err != nil {
		log.Printf("Unknown operator command: %s", command)
	}
}

func (o *OperatorConsole) ApplyCommand(command string, payload string) error {
	switch command {
	case "help":
		log.Println("Commands: status | pause | resume | abort | approve <step-id> | inject <command>")
		return nil
	case "status":
		o.statusMu.RLock()
		defer o.statusMu.RUnlock()
		log.Printf("Operator status: paused=%t aborted=%t", o.paused, o.aborted)
		return nil
	case "pause":
		o.statusMu.Lock()
		o.paused = true
		o.statusMu.Unlock()
		o.commands <- OperatorAction{Timestamp: time.Now().UTC(), Action: "pause"}
		return nil
	case "resume":
		o.statusMu.Lock()
		o.paused = false
		o.statusMu.Unlock()
		o.commands <- OperatorAction{Timestamp: time.Now().UTC(), Action: "resume"}
		return nil
	case "abort":
		o.statusMu.Lock()
		o.aborted = true
		o.statusMu.Unlock()
		o.commands <- OperatorAction{Timestamp: time.Now().UTC(), Action: "abort"}
		return nil
	case "approve":
		if payload != "" {
			stepID := strings.Fields(payload)[0]
			o.approvals <- stepID
			o.commands <- OperatorAction{Timestamp: time.Now().UTC(), Action: "approve", Payload: stepID}
		}
		return nil
	case "inject":
		if payload != "" {
			injected := strings.TrimSpace(payload)
			o.injectedCommands <- injected
			o.commands <- OperatorAction{Timestamp: time.Now().UTC(), Action: "inject", Payload: injected}
		}
		return nil
	default:
		return fmt.Errorf("unknown command")
	}
}

func (o *OperatorConsole) IsPaused() bool {
	o.statusMu.RLock()
	defer o.statusMu.RUnlock()
	return o.paused
}

func (o *OperatorConsole) IsAborted() bool {
	o.statusMu.RLock()
	defer o.statusMu.RUnlock()
	return o.aborted
}

func (o *OperatorConsole) Snapshot() (bool, bool) {
	o.statusMu.RLock()
	defer o.statusMu.RUnlock()
	return o.paused, o.aborted
}

func (o *OperatorConsole) Approvals() <-chan string {
	return o.approvals
}

func (o *OperatorConsole) InjectedCommands() <-chan string {
	return o.injectedCommands
}

func (o *OperatorConsole) Actions() <-chan OperatorAction {
	return o.commands
}

type AuditLogger struct {
	mu      sync.Mutex
	file    *os.File
	encoder *json.Encoder
}

func NewAuditLogger(path string) (*AuditLogger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	return &AuditLogger{file: file, encoder: encoder}, nil
}

func (a *AuditLogger) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.file.Close()
}

func (a *AuditLogger) Log(event AuditEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()
	_ = a.encoder.Encode(event)
}

type MissionExecutor struct {
	robot          control.HunoidController
	manipulator    control.ManipulatorController
	vlaModel       vla.VLAModel
	ethicsKernel   *ethics.EthicalKernel
	policyEngine   *SafetyPolicyEngine
	intervention   *InterventionEngine
	actionRegistry *ActionRegistry
	operator       *OperatorConsole
	audit          *AuditLogger
	state          *MissionState
}

func NewMissionExecutor(robot control.HunoidController, manipulator control.ManipulatorController, vlaModel vla.VLAModel, ethicsKernel *ethics.EthicalKernel, policyEngine *SafetyPolicyEngine, intervention *InterventionEngine, actionRegistry *ActionRegistry, operator *OperatorConsole, audit *AuditLogger, state *MissionState) *MissionExecutor {
	return &MissionExecutor{
		robot:          robot,
		manipulator:    manipulator,
		vlaModel:       vlaModel,
		ethicsKernel:   ethicsKernel,
		policyEngine:   policyEngine,
		intervention:   intervention,
		actionRegistry: actionRegistry,
		operator:       operator,
		audit:          audit,
		state:          state,
	}
}

func (e *MissionExecutor) Run(ctx context.Context, mission *MissionPlan) (*MissionReport, error) {
	report := &MissionReport{
		MissionID:   mission.ID,
		MissionName: mission.Name,
		Objective:   mission.Objective,
		StartedAt:   time.Now().UTC(),
	}
	e.state.SetMission(mission)
	e.state.AddEvent("mission_start")

	e.audit.Log(AuditEvent{
		Timestamp: report.StartedAt,
		Type:      "mission_start",
		MissionID: mission.ID,
		Details: map[string]interface{}{
			"name":       mission.Name,
			"objective":  mission.Objective,
			"risk_level": mission.RiskLevel,
		},
	})

	stepIndex := 0
	for stepIndex < len(mission.Steps) {
		step := mission.Steps[stepIndex]
		stepIndex++

		select {
		case injected := <-e.operator.InjectedCommands():
			mission.Steps = append(mission.Steps, MissionStep{
				ID:                fmt.Sprintf("step-%d", len(mission.Steps)+1),
				Command:           injected,
				Criticality:       CriticalityMedium,
				AllowAutoApproval: false,
				HazardLevel:       2,
			})
			log.Printf("Injected step added: %s", injected)
			e.state.AddEvent("step_injected")
			e.audit.Log(AuditEvent{
				Timestamp: time.Now().UTC(),
				Type:      "mission_step_injected",
				MissionID: mission.ID,
				Details: map[string]interface{}{
					"command": injected,
				},
			})
		default:
		}

		if e.operator.IsAborted() {
			log.Printf("Mission aborted by operator")
			e.state.AddEvent("mission_aborted")
			e.state.SetOutcome("aborted")
			e.audit.Log(AuditEvent{
				Timestamp: time.Now().UTC(),
				Type:      "mission_aborted",
				MissionID: mission.ID,
				Details: map[string]interface{}{
					"reason": "operator_abort",
				},
			})
			break
		}

		for e.operator.IsPaused() {
			select {
			case <-time.After(1 * time.Second):
				continue
			case <-ctx.Done():
				return report, ctx.Err()
			}
		}

		log.Printf("Mission step [%s]: %s", step.ID, step.Command)
		e.state.SetStep(step)
		e.state.AddEvent("step_start")
		e.audit.Log(AuditEvent{
			Timestamp: time.Now().UTC(),
			Type:      "mission_step_start",
			MissionID: mission.ID,
			StepID:    step.ID,
			Details: map[string]interface{}{
				"command":     step.Command,
				"criticality": step.Criticality,
				"hazard":      step.HazardLevel,
			},
		})

		stepStart := time.Now()
		action, err := e.vlaModel.InferAction(ctx, []byte{}, step.Command)
		if err != nil {
			log.Printf("VLA inference failed: %v", err)
			e.state.AddEvent("vla_inference_failed")
			e.audit.Log(AuditEvent{
				Timestamp: time.Now().UTC(),
				Type:      "vla_inference_failed",
				MissionID: mission.ID,
				StepID:    step.ID,
				Details: map[string]interface{}{
					"error": err.Error(),
				},
			})
			continue
		}

		log.Printf("VLA inferred action: %s (confidence: %.2f)", action.Type, action.Confidence)
		e.state.SetAction(action)
		e.audit.Log(AuditEvent{
			Timestamp: time.Now().UTC(),
			Type:      "vla_inference",
			MissionID: mission.ID,
			StepID:    step.ID,
			Details: map[string]interface{}{
				"action":     action.Type,
				"confidence": action.Confidence,
			},
		})

		ethicsDecision, err := e.ethicsKernel.Evaluate(ctx, action)
		if err != nil {
			log.Printf("Ethical evaluation failed: %v", err)
			e.audit.Log(AuditEvent{
				Timestamp: time.Now().UTC(),
				Type:      "ethics_failed",
				MissionID: mission.ID,
				StepID:    step.ID,
				Details: map[string]interface{}{
					"error": err.Error(),
				},
			})
			continue
		}
		report.EthicsDecisions = append(report.EthicsDecisions, *ethicsDecision)

		log.Printf("Ethical decision: %s - %s (score: %.2f)", ethicsDecision.Decision, ethicsDecision.Reasoning, ethicsDecision.Score)
		e.audit.Log(AuditEvent{
			Timestamp: time.Now().UTC(),
			Type:      "ethics_decision",
			MissionID: mission.ID,
			StepID:    step.ID,
			Details: map[string]interface{}{
				"decision":  ethicsDecision.Decision,
				"reasoning": ethicsDecision.Reasoning,
				"score":     ethicsDecision.Score,
			},
		})

		battery := e.robot.GetBatteryPercent()
		policyDecision := e.policyEngine.Evaluate(action, step, battery)
		report.PolicyDecisions = append(report.PolicyDecisions, policyDecision)

		log.Printf("Policy decision: %s (score: %.2f)", policyDecision.Decision, policyDecision.Score)
		e.audit.Log(AuditEvent{
			Timestamp: time.Now().UTC(),
			Type:      "policy_decision",
			MissionID: mission.ID,
			StepID:    step.ID,
			Details: map[string]interface{}{
				"decision": policyDecision.Decision,
				"reasons":  policyDecision.Reasons,
				"score":    policyDecision.Score,
			},
		})

		intervention := e.intervention.Decide(action, ethicsDecision, policyDecision, step)
		report.Interventions = append(report.Interventions, intervention)
		e.state.SetDecisions(ethicsDecision, policyDecision, intervention)

		log.Printf("Intervention decision: %s - %s", intervention.Action, intervention.Reason)
		e.audit.Log(AuditEvent{
			Timestamp: time.Now().UTC(),
			Type:      "intervention_decision",
			MissionID: mission.ID,
			StepID:    step.ID,
			Details: map[string]interface{}{
				"action":   intervention.Action,
				"reason":   intervention.Reason,
				"requires": intervention.RequiresApproval,
			},
		})

		if intervention.Action == InterventionAbort {
			report.BlockedStepCount++
			report.StepResults = append(report.StepResults, StepResult{
				StepID:   step.ID,
				Command:  step.Command,
				Action:   action.Type,
				Outcome:  "aborted",
				Duration: time.Since(stepStart),
			})
			e.state.SetOutcome("aborted")
			log.Printf("Step aborted: %s", step.ID)
			continue
		}

		if intervention.Action == InterventionHold {
			e.state.SetPendingApproval(step.ID)
			e.state.AddEvent("awaiting_approval")
			approved, approvalType := e.awaitApproval(ctx, step, intervention)
			if !approved {
				report.BlockedStepCount++
				report.StepResults = append(report.StepResults, StepResult{
					StepID:   step.ID,
					Command:  step.Command,
					Action:   action.Type,
					Outcome:  "blocked",
					Duration: time.Since(stepStart),
				})
				e.state.SetOutcome("blocked")
				e.state.ClearPendingApproval()
				log.Printf("Step blocked awaiting approval: %s", step.ID)
				continue
			}
			e.state.ClearPendingApproval()
			if approvalType == "auto" {
				report.AutoApproved++
			} else {
				report.ManualApproved++
			}
		}

		if err := e.actionRegistry.Execute(ctx, action); err != nil {
			log.Printf("Action execution failed: %v", err)
			report.StepResults = append(report.StepResults, StepResult{
				StepID:   step.ID,
				Command:  step.Command,
				Action:   action.Type,
				Outcome:  "failed",
				Duration: time.Since(stepStart),
			})
			e.state.SetOutcome("failed")
			e.audit.Log(AuditEvent{
				Timestamp: time.Now().UTC(),
				Type:      "action_failed",
				MissionID: mission.ID,
				StepID:    step.ID,
				Details: map[string]interface{}{
					"error": err.Error(),
				},
			})
			continue
		}

		log.Printf("Action completed successfully")
		report.StepResults = append(report.StepResults, StepResult{
			StepID:   step.ID,
			Command:  step.Command,
			Action:   action.Type,
			Outcome:  "completed",
			Duration: time.Since(stepStart),
		})
		e.state.SetOutcome("completed")
		e.audit.Log(AuditEvent{
			Timestamp: time.Now().UTC(),
			Type:      "action_completed",
			MissionID: mission.ID,
			StepID:    step.ID,
		})

		select {
		case operatorAction := <-e.operator.Actions():
			report.OperatorActions = append(report.OperatorActions, operatorAction)
		default:
		}
	}

	report.CompletedAt = time.Now().UTC()
	e.state.AddEvent("mission_complete")
	e.audit.Log(AuditEvent{
		Timestamp: report.CompletedAt,
		Type:      "mission_complete",
		MissionID: mission.ID,
		Details: map[string]interface{}{
			"steps_completed": len(report.StepResults),
			"blocked_steps":   report.BlockedStepCount,
		},
	})

	return report, nil
}

func (e *MissionExecutor) awaitApproval(ctx context.Context, step MissionStep, decision InterventionDecision) (bool, string) {
	if !decision.RequiresApproval {
		return true, "none"
	}

	operatorMode := strings.ToLower(e.operator.mode)
	if operatorMode == "disabled" {
		log.Printf("Operator interface disabled; blocking step %s", step.ID)
		return false, "none"
	}

	if operatorMode == "auto" {
		timer := time.NewTimer(decision.OperatorTimeout)
		defer timer.Stop()
		select {
		case <-timer.C:
			if !step.AllowAutoApproval {
				log.Printf("Auto-approving step %s (manual approval recommended)", step.ID)
			} else {
				log.Printf("Auto-approving step %s", step.ID)
			}
			return true, "auto"
		case <-ctx.Done():
			return false, "none"
		}
	}

	log.Printf("Awaiting operator approval for step %s", step.ID)
	for {
		select {
		case approval := <-e.operator.Approvals():
			if approval == step.ID {
				log.Printf("Operator approved step %s", step.ID)
				return true, "manual"
			}
		case <-ctx.Done():
			return false, "none"
		}
	}
}

func writeReport(path string, report *MissionReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	builder := &strings.Builder{}
	builder.WriteString("# Hunoid Mission Report\n\n")
	builder.WriteString(fmt.Sprintf("- Mission ID: %s\n", report.MissionID))
	builder.WriteString(fmt.Sprintf("- Mission name: %s\n", report.MissionName))
	builder.WriteString(fmt.Sprintf("- Objective: %s\n", report.Objective))
	builder.WriteString(fmt.Sprintf("- Started: %s\n", report.StartedAt.Format(time.RFC3339)))
	builder.WriteString(fmt.Sprintf("- Completed: %s\n", report.CompletedAt.Format(time.RFC3339)))
	builder.WriteString(fmt.Sprintf("- Steps completed: %d\n", len(report.StepResults)))
	builder.WriteString(fmt.Sprintf("- Steps blocked: %d\n", report.BlockedStepCount))
	builder.WriteString(fmt.Sprintf("- Auto approvals: %d\n", report.AutoApproved))
	builder.WriteString(fmt.Sprintf("- Manual approvals: %d\n\n", report.ManualApproved))

	builder.WriteString("## Step Outcomes\n\n")
	for _, result := range report.StepResults {
		builder.WriteString(fmt.Sprintf("- %s: %s -> %s (%s)\n", result.StepID, result.Command, result.Outcome, result.Duration))
	}

	builder.WriteString("\n## Intervention Decisions\n\n")
	for _, decision := range report.Interventions {
		builder.WriteString(fmt.Sprintf("- %s: %s\n", decision.Action, decision.Reason))
	}

	builder.WriteString("\n## Policy Decisions\n\n")
	for _, decision := range report.PolicyDecisions {
		builder.WriteString(fmt.Sprintf("- %s (score %.2f): %s\n", decision.Decision, decision.Score, strings.Join(decision.Reasons, "; ")))
	}

	builder.WriteString("\n## Operator Actions\n\n")
	if len(report.OperatorActions) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, action := range report.OperatorActions {
			builder.WriteString(fmt.Sprintf("- %s: %s %s\n", action.Timestamp.Format(time.RFC3339), action.Action, action.Payload))
		}
	}

	_, err = file.WriteString(builder.String())
	return err
}

type actionRequest struct {
	StepID  string `json:"step_id"`
	Command string `json:"command"`
}

type apiResponse struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

const operatorHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Hunoid Operator Console</title>
  <style>
    :root {
      --bg: #f6f7fb;
      --card: rgba(255, 255, 255, 0.85);
      --ink: #111827;
      --muted: #6b7280;
      --accent: #2563eb;
      --accent-soft: rgba(37, 99, 235, 0.12);
      --success: #10b981;
      --warning: #f59e0b;
      --danger: #ef4444;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: -apple-system, BlinkMacSystemFont, "SF Pro Text", "Segoe UI", sans-serif;
      background: radial-gradient(circle at top left, #e9efff, transparent 50%),
                  radial-gradient(circle at top right, #fce7f3, transparent 45%),
                  var(--bg);
      color: var(--ink);
    }
    header {
      padding: 32px 40px 16px;
    }
    h1 {
      margin: 0;
      font-size: 28px;
      font-weight: 600;
    }
    p.subtitle {
      margin: 6px 0 0;
      color: var(--muted);
    }
    main {
      display: grid;
      grid-template-columns: 1.2fr 1fr;
      gap: 24px;
      padding: 16px 40px 40px;
    }
    .card {
      background: var(--card);
      border-radius: 20px;
      padding: 20px;
      box-shadow: 0 12px 30px rgba(15, 23, 42, 0.08);
      backdrop-filter: blur(10px);
    }
    .section-title {
      font-size: 14px;
      letter-spacing: 0.08em;
      text-transform: uppercase;
      color: var(--muted);
      margin-bottom: 12px;
    }
    .metric {
      display: grid;
      gap: 8px;
      margin-bottom: 16px;
    }
    .metric label {
      color: var(--muted);
      font-size: 13px;
    }
    .metric span {
      font-size: 18px;
      font-weight: 600;
    }
    .status-badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 6px 12px;
      border-radius: 999px;
      font-size: 12px;
      background: var(--accent-soft);
      color: var(--accent);
    }
    .controls {
      display: grid;
      gap: 12px;
    }
    button {
      border: none;
      border-radius: 12px;
      padding: 12px 16px;
      font-size: 14px;
      font-weight: 600;
      cursor: pointer;
      background: var(--accent);
      color: white;
      transition: transform 0.1s ease, box-shadow 0.2s ease;
    }
    button.secondary {
      background: white;
      color: var(--accent);
      border: 1px solid var(--accent);
    }
    button.warn { background: var(--warning); }
    button.danger { background: var(--danger); }
    button:active { transform: scale(0.98); }
    input, textarea {
      width: 100%;
      border-radius: 12px;
      border: 1px solid #e5e7eb;
      padding: 10px 12px;
      font-size: 14px;
      font-family: inherit;
    }
    .row {
      display: grid;
      gap: 12px;
    }
    .events {
      list-style: none;
      padding: 0;
      margin: 0;
      display: grid;
      gap: 8px;
      color: var(--muted);
      font-size: 13px;
    }
    .status-grid {
      display: grid;
      grid-template-columns: repeat(2, minmax(0, 1fr));
      gap: 16px;
    }
    footer {
      padding: 0 40px 40px;
      color: var(--muted);
      font-size: 12px;
    }
    @media (max-width: 960px) {
      main { grid-template-columns: 1fr; }
    }
  </style>
</head>
<body>
  <header>
    <h1>Hunoid Operator Console</h1>
    <p class="subtitle">Real-time mission control with ethical oversight.</p>
  </header>
  <main>
    <section class="card">
      <div class="section-title">Mission Status</div>
      <div class="metric">
        <label>Mission</label>
        <span id="missionName">--</span>
      </div>
      <div class="metric">
        <label>Objective</label>
        <span id="missionObjective">--</span>
      </div>
      <div class="status-grid">
        <div class="metric">
          <label>Current Step</label>
          <span id="currentStep">--</span>
        </div>
        <div class="metric">
          <label>Action</label>
          <span id="currentAction">--</span>
        </div>
        <div class="metric">
          <label>Confidence</label>
          <span id="confidence">--</span>
        </div>
        <div class="metric">
          <label>Last Outcome</label>
          <span id="lastOutcome">--</span>
        </div>
      </div>
      <div class="metric">
        <label>Decisions</label>
        <span id="decisions">--</span>
      </div>
      <div class="metric">
        <label>Pending Approval</label>
        <span id="pendingApproval">--</span>
      </div>
      <div class="metric">
        <label>Operator State</label>
        <span class="status-badge" id="operatorState">--</span>
      </div>
    </section>

    <section class="card">
      <div class="section-title">Controls</div>
      <div class="controls">
        <div class="row">
          <button class="secondary" onclick="sendAction('pause')">Pause</button>
          <button class="secondary" onclick="sendAction('resume')">Resume</button>
          <button class="danger" onclick="sendAction('abort')">Abort</button>
        </div>
        <div>
          <label>Approve step</label>
          <input id="approveInput" placeholder="step-id (e.g., step-3)" />
          <button onclick="approveStep()">Approve</button>
        </div>
        <div>
          <label>Inject command</label>
          <textarea id="injectInput" rows="2" placeholder="New command for mission plan"></textarea>
          <button class="warn" onclick="injectCommand()">Inject</button>
        </div>
        <div class="section-title">Recent Events</div>
        <ul class="events" id="events"></ul>
      </div>
    </section>
  </main>
  <footer>Hunoid UI console follows Apple design principles: clarity, deference, and depth.</footer>

  <script>
    async function fetchStatus() {
      const res = await fetch('/api/status');
      if (!res.ok) return;
      const data = await res.json();
      document.getElementById('missionName').textContent = data.mission_name || '--';
      document.getElementById('missionObjective').textContent = data.objective || '--';
      document.getElementById('currentStep').textContent = ((data.current_step_id || '--') + ' ' + (data.current_command || '')).trim();
      document.getElementById('currentAction').textContent = data.current_action || '--';
      document.getElementById('confidence').textContent = data.current_confidence ? data.current_confidence.toFixed(2) : '--';
      document.getElementById('lastOutcome').textContent = data.last_outcome || '--';
      document.getElementById('decisions').textContent = (data.last_ethics || '--') + ' / ' + (data.last_policy || '--') + ' / ' + (data.last_intervention || '--');
      document.getElementById('pendingApproval').textContent = data.pending_approval || '--';
      document.getElementById('operatorState').textContent = (data.operator_mode || 'auto') + ' | paused=' + data.paused + ' | aborted=' + data.aborted;
      const events = document.getElementById('events');
      events.innerHTML = '';
      (data.events || []).slice().reverse().forEach(function(event) {
        const li = document.createElement('li');
        li.textContent = event;
        events.appendChild(li);
      });
    }

    async function sendAction(action) {
      await fetch('/api/' + action, { method: 'POST' });
      setTimeout(fetchStatus, 200);
    }

    async function approveStep() {
      const stepID = document.getElementById('approveInput').value.trim();
      if (!stepID) return;
      await fetch('/api/approve', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ step_id: stepID })
      });
      document.getElementById('approveInput').value = '';
      setTimeout(fetchStatus, 200);
    }

    async function injectCommand() {
      const command = document.getElementById('injectInput').value.trim();
      if (!command) return;
      await fetch('/api/inject', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ command })
      });
      document.getElementById('injectInput').value = '';
      setTimeout(fetchStatus, 200);
    }

    fetchStatus();
    setInterval(fetchStatus, 2000);
  </script>
</body>
</html>`

func startOperatorUIServer(ctx context.Context, addr string, operator *OperatorConsole, state *MissionState, audit *AuditLogger) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(operatorHTML))
	})

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(state.Snapshot(operator))
	})

	handleAction := func(action string, getPayload func(*http.Request) (string, error)) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			payload, err := getPayload(r)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(apiResponse{Ok: false, Error: err.Error()})
				return
			}
			if err := operator.ApplyCommand(action, payload); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(apiResponse{Ok: false, Error: err.Error()})
				return
			}
			state.AddEvent(fmt.Sprintf("operator_%s", action))
			audit.Log(AuditEvent{
				Timestamp: time.Now().UTC(),
				Type:      "operator_ui_action",
				Details: map[string]interface{}{
					"action":  action,
					"payload": payload,
				},
			})
			_ = json.NewEncoder(w).Encode(apiResponse{Ok: true})
		}
	}

	mux.HandleFunc("/api/pause", handleAction("pause", func(_ *http.Request) (string, error) { return "", nil }))
	mux.HandleFunc("/api/resume", handleAction("resume", func(_ *http.Request) (string, error) { return "", nil }))
	mux.HandleFunc("/api/abort", handleAction("abort", func(_ *http.Request) (string, error) { return "", nil }))
	mux.HandleFunc("/api/approve", handleAction("approve", func(r *http.Request) (string, error) {
		var req actionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return "", err
		}
		if req.StepID == "" {
			return "", fmt.Errorf("step_id is required")
		}
		return req.StepID, nil
	}))
	mux.HandleFunc("/api/inject", handleAction("inject", func(r *http.Request) (string, error) {
		var req actionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return "", err
		}
		if strings.TrimSpace(req.Command) == "" {
			return "", fmt.Errorf("command is required")
		}
		return strings.TrimSpace(req.Command), nil
	}))

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		_ = server.Shutdown(context.Background())
	}()

	go func() {
		if err := server.ListenAndServe(); err != nil && !strings.Contains(err.Error(), "Server closed") {
			log.Printf("Operator UI server error: %v", err)
		}
	}()

	return nil
}

func reportTelemetry(ctx context.Context, robot control.HunoidController, interval time.Duration, audit *AuditLogger, missionID string) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pose, _ := robot.GetCurrentPose()
			battery := robot.GetBatteryPercent()
			isMoving := robot.IsMoving()

			log.Printf("Telemetry: Position=(%.2f, %.2f, %.2f), Battery=%.1f%%, Moving=%t",
				pose.Position.X, pose.Position.Y, pose.Position.Z, battery, isMoving)

			audit.Log(AuditEvent{
				Timestamp: time.Now().UTC(),
				Type:      "telemetry",
				MissionID: missionID,
				Details: map[string]interface{}{
					"position": map[string]float64{
						"x": pose.Position.X,
						"y": pose.Position.Y,
						"z": pose.Position.Z,
					},
					"battery": battery,
					"moving":  isMoving,
				},
			})
		case <-ctx.Done():
			return
		}
	}
}

func executeAction(ctx context.Context, robot control.HunoidController, manip control.ManipulatorController, action *vla.Action) error {
	switch action.Type {
	case vla.ActionNavigate:
		x, _ := action.Parameters["x"].(float64)
		y, _ := action.Parameters["y"].(float64)
		z, _ := action.Parameters["z"].(float64)

		targetPose := control.Pose{
			Position:    control.Vector3{X: x, Y: y, Z: z},
			Orientation: control.Quaternion{W: 1, X: 0, Y: 0, Z: 0},
		}

		return robot.MoveTo(ctx, targetPose)

	case vla.ActionPickUp:
		return manip.CloseGripper()

	case vla.ActionPutDown:
		return manip.OpenGripper()

	case vla.ActionOpen:
		return manip.OpenGripper()

	case vla.ActionClose:
		return manip.CloseGripper()

	case vla.ActionInspect:
		if duration, ok := action.Parameters["duration_seconds"].(int); ok {
			time.Sleep(time.Duration(duration) * time.Second)
		}
		return nil

	case vla.ActionWait:
		time.Sleep(2 * time.Second)
		return nil

	default:
		return nil
	}
}

func main() {
	hunoidID := flag.String("id", "hunoid001", "Hunoid ID")
	serialNum := flag.String("serial", "HND-2026-001", "Serial number")
	scenario := flag.String("scenario", "medical_aid", "Scenario: medical_aid, perimeter_check, hazard_response")
	operatorMode := flag.String("operator-mode", "auto", "Operator mode: auto, manual, disabled")
	autoApproveDelay := flag.Duration("auto-approve-delay", 3*time.Second, "Auto-approval delay")
	operatorUI := flag.Bool("operator-ui", true, "Enable the UI-based operator console")
	operatorUIAddr := flag.String("operator-ui-addr", ":8090", "Operator UI listen address")
	auditPath := flag.String("audit-log", "Documentation/Hunoid_Audit_Log.jsonl", "Audit log path")
	reportPath := flag.String("report", "Documentation/Hunoid_Mission_Report.md", "Report output path")
	telemetryInterval := flag.Duration("telemetry-interval", 5*time.Second, "Telemetry interval")
	metricsAddr := flag.String("metrics-addr", ":9092", "Metrics server address")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Printf("Starting ASGARD Hunoid: %s (%s)", *hunoidID, *serialNum)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownTracing, err := observability.InitTracing(context.Background(), "hunoid")
	if err != nil {
		log.Printf("Tracing disabled: %v", err)
	} else {
		defer func() {
			if err := shutdownTracing(context.Background()); err != nil {
				log.Printf("Tracing shutdown error: %v", err)
			}
		}()
	}

	hunoidEndpoint := os.Getenv("HUNOID_ENDPOINT")
	robot, err := control.NewRemoteHunoid(*hunoidID, hunoidEndpoint)
	if err != nil {
		log.Fatalf("Failed to initialize robot controller: %v", err)
	}
	if err := robot.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize robot: %v", err)
	}
	log.Println("Robot controller initialized")

	manipulator, err := control.NewRemoteManipulator(*hunoidID, hunoidEndpoint)
	if err != nil {
		log.Fatalf("Failed to initialize manipulator: %v", err)
	}
	log.Println("Manipulator initialized")

	vlaEndpoint := os.Getenv("VLA_ENDPOINT")
	vlaModel, err := vla.NewHTTPVLA(vlaEndpoint)
	if err != nil {
		log.Fatalf("Failed to initialize VLA client: %v", err)
	}
	if err := vlaModel.Initialize(ctx, "models/openvla.onnx"); err != nil {
		log.Fatalf("Failed to initialize VLA: %v", err)
	}
	defer vlaModel.Shutdown()
	modelInfo := vlaModel.GetModelInfo()
	log.Printf("VLA Model: %s v%s", modelInfo.Name, modelInfo.Version)

	ethicsKernel := ethics.NewEthicalKernel()
	log.Println("Ethical kernel initialized")

	auditLogger, err := NewAuditLogger(*auditPath)
	if err != nil {
		log.Fatalf("Failed to initialize audit logger: %v", err)
	}
	defer auditLogger.Close()

	operator := NewOperatorConsole(*operatorMode, *autoApproveDelay)
	operator.Start(ctx)

	missionState := NewMissionState()

	planner := NewMissionPlanner()
	missionPlan, err := planner.BuildScenario(*scenario)
	if err != nil {
		log.Fatalf("Failed to build mission plan: %v", err)
	}
	log.Printf("Mission plan loaded: %s (%s)", missionPlan.Name, missionPlan.Objective)

	actionRegistry := NewActionRegistry()
	actionRegistry.Register(vla.ActionNavigate, func(ctx context.Context, action *vla.Action) error {
		return executeAction(ctx, robot, manipulator, action)
	})
	actionRegistry.Register(vla.ActionPickUp, func(ctx context.Context, action *vla.Action) error {
		return executeAction(ctx, robot, manipulator, action)
	})
	actionRegistry.Register(vla.ActionPutDown, func(ctx context.Context, action *vla.Action) error {
		return executeAction(ctx, robot, manipulator, action)
	})
	actionRegistry.Register(vla.ActionOpen, func(ctx context.Context, action *vla.Action) error {
		return executeAction(ctx, robot, manipulator, action)
	})
	actionRegistry.Register(vla.ActionClose, func(ctx context.Context, action *vla.Action) error {
		return executeAction(ctx, robot, manipulator, action)
	})
	actionRegistry.Register(vla.ActionInspect, func(ctx context.Context, action *vla.Action) error {
		return executeAction(ctx, robot, manipulator, action)
	})
	actionRegistry.Register(vla.ActionWait, func(ctx context.Context, action *vla.Action) error {
		return executeAction(ctx, robot, manipulator, action)
	})

	policyEngine := NewSafetyPolicyEngine(20.0)
	interventionEngine := NewInterventionEngine(0.7, 5*time.Second)
	executor := NewMissionExecutor(robot, manipulator, vlaModel, ethicsKernel, policyEngine, interventionEngine, actionRegistry, operator, auditLogger, missionState)

	go reportTelemetry(ctx, robot, *telemetryInterval, auditLogger, missionPlan.ID)

	if *operatorUI {
		if err := startOperatorUIServer(ctx, *operatorUIAddr, operator, missionState, auditLogger); err != nil {
			log.Printf("Failed to start operator UI: %v", err)
		} else {
			uiAddr := *operatorUIAddr
			if strings.HasPrefix(uiAddr, ":") {
				uiAddr = "http://localhost" + uiAddr
			} else if !strings.HasPrefix(uiAddr, "http") {
				uiAddr = "http://" + uiAddr
			}
			log.Printf("Operator UI available at %s", uiAddr)
		}
	}

	metricsServer := startMetricsServer(*metricsAddr)

	done := make(chan *MissionReport, 1)
	go func() {
		report, runErr := executor.Run(ctx, missionPlan)
		if runErr != nil {
			log.Printf("Mission run error: %v", runErr)
		}
		done <- report
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sigChan:
		log.Println("Shutting down Hunoid...")
		cancel()
	case report := <-done:
		if report != nil {
			if err := writeReport(*reportPath, report); err != nil {
				log.Printf("Failed to write report: %v", err)
			} else {
				log.Printf("Mission report written to %s", *reportPath)
			}
		}
		cancel()
	}

	shutdownMetricsServer(metricsServer)
	time.Sleep(1 * time.Second)
	log.Println("Hunoid stopped")
}

func startMetricsServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", observability.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	log.Printf("Metrics server listening on %s", addr)
	return server
}

func shutdownMetricsServer(server *http.Server) {
	if server == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Metrics server shutdown error: %v", err)
	}
}
