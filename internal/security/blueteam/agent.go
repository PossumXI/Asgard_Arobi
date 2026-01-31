// Package blueteam implements automated blue team defensive agents.
package blueteam

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// DefenseType categorizes defensive actions
type DefenseType string

const (
	DefenseTypeDetection   DefenseType = "detection"
	DefenseTypeContainment DefenseType = "containment"
	DefenseTypeEradication DefenseType = "eradication"
	DefenseTypeRecovery    DefenseType = "recovery"
	DefenseTypeHardening   DefenseType = "hardening"
)

// ThreatLevel indicates severity of detected threats
type ThreatLevel string

const (
	ThreatLevelCritical ThreatLevel = "critical"
	ThreatLevelHigh     ThreatLevel = "high"
	ThreatLevelMedium   ThreatLevel = "medium"
	ThreatLevelLow      ThreatLevel = "low"
	ThreatLevelInfo     ThreatLevel = "info"
)

// Detection represents a security detection
type Detection struct {
	ID          string
	Type        string
	Source      string
	ThreatLevel ThreatLevel
	Description string
	Evidence    []string
	Timestamp   time.Time
	Responded   bool
	ResponseID  string
}

// Response represents a defensive response action
type Response struct {
	ID          string
	DetectionID string
	Type        DefenseType
	Action      string
	Target      string
	Status      string
	StartTime   time.Time
	EndTime     *time.Time
	Success     bool
	Notes       []string
}

// Rule represents a detection rule
type Rule struct {
	ID          string
	Name        string
	Description string
	Enabled     bool
	Severity    ThreatLevel
	Logic       RuleLogic
	Actions     []string
}

// RuleLogic defines the detection logic
type RuleLogic struct {
	Field     string
	Operator  string
	Value     interface{}
	Threshold int
	Window    time.Duration
}

// Agent is an automated blue team agent
type Agent struct {
	mu         sync.RWMutex
	id         string
	name       string
	detections []Detection
	responses  []Response
	rules      map[string]*Rule
	blocklist  map[string]time.Time
	config     AgentConfig
	alertChan  chan Detection
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// AgentConfig configures the blue team agent
type AgentConfig struct {
	AutoRespond      bool
	ResponseTimeout  time.Duration
	BlockDuration    time.Duration
	MaxBlocklistSize int
	AlertThreshold   int
	EnableHeuristics bool
}

// DefaultAgentConfig returns safe defaults
func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		AutoRespond:      true,
		ResponseTimeout:  30 * time.Second,
		BlockDuration:    24 * time.Hour,
		MaxBlocklistSize: 10000,
		AlertThreshold:   5,
		EnableHeuristics: true,
	}
}

// NewAgent creates a new blue team agent
func NewAgent(name string, cfg AgentConfig) *Agent {
	return &Agent{
		id:         uuid.New().String(),
		name:       name,
		detections: make([]Detection, 0),
		responses:  make([]Response, 0),
		rules:      make(map[string]*Rule),
		blocklist:  make(map[string]time.Time),
		config:     cfg,
		alertChan:  make(chan Detection, 100),
		stopCh:     make(chan struct{}),
	}
}

// Start begins the agent
func (a *Agent) Start(ctx context.Context) error {
	a.loadDefaultRules()

	// Start detection processor
	a.wg.Add(1)
	go a.processDetections(ctx)

	// Start blocklist cleaner
	a.wg.Add(1)
	go a.cleanBlocklist(ctx)

	log.Printf("[BlueTeam] Agent %s (%s) started with %d rules", a.name, a.id, len(a.rules))
	return nil
}

// Stop shuts down the agent
func (a *Agent) Stop() {
	close(a.stopCh)
	a.wg.Wait()
	log.Printf("[BlueTeam] Agent %s stopped", a.name)
}

// AddRule adds a detection rule
func (a *Agent) AddRule(rule *Rule) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.rules[rule.ID] = rule
	log.Printf("[BlueTeam] Rule added: %s", rule.Name)
}

// RemoveRule removes a detection rule
func (a *Agent) RemoveRule(ruleID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.rules[ruleID]; !exists {
		return fmt.Errorf("rule not found: %s", ruleID)
	}
	delete(a.rules, ruleID)
	return nil
}

// ReportDetection submits a new detection
func (a *Agent) ReportDetection(detection Detection) error {
	if detection.ID == "" {
		detection.ID = uuid.New().String()
	}
	detection.Timestamp = time.Now()

	a.mu.Lock()
	a.detections = append(a.detections, detection)
	a.mu.Unlock()

	// Send to processing channel
	select {
	case a.alertChan <- detection:
		log.Printf("[BlueTeam] Detection reported: %s (%s)", detection.Type, detection.ThreatLevel)
	default:
		log.Printf("[BlueTeam] Warning: alert channel full")
	}

	return nil
}

// ProcessEvent evaluates an event against rules
func (a *Agent) ProcessEvent(eventType string, data map[string]interface{}) []Detection {
	a.mu.RLock()
	rules := make([]*Rule, 0, len(a.rules))
	for _, rule := range a.rules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}
	a.mu.RUnlock()

	detections := make([]Detection, 0)

	for _, rule := range rules {
		if a.matchRule(rule, eventType, data) {
			detection := Detection{
				ID:          uuid.New().String(),
				Type:        rule.Name,
				Source:      "rule_engine",
				ThreatLevel: rule.Severity,
				Description: rule.Description,
				Evidence:    []string{fmt.Sprintf("Matched rule: %s", rule.ID)},
				Timestamp:   time.Now(),
			}
			detections = append(detections, detection)
			a.ReportDetection(detection)
		}
	}

	return detections
}

// BlockIP adds an IP to the blocklist
func (a *Agent) BlockIP(ip string, reason string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.blocklist) >= a.config.MaxBlocklistSize {
		return fmt.Errorf("blocklist is full")
	}

	expiry := time.Now().Add(a.config.BlockDuration)
	a.blocklist[ip] = expiry

	log.Printf("[BlueTeam] IP blocked: %s (reason: %s, expires: %s)", ip, reason, expiry.Format(time.RFC3339))

	// Create response record
	response := Response{
		ID:        uuid.New().String(),
		Type:      DefenseTypeContainment,
		Action:    "block_ip",
		Target:    ip,
		Status:    "completed",
		StartTime: time.Now(),
		Success:   true,
		Notes:     []string{reason},
	}
	endTime := time.Now()
	response.EndTime = &endTime
	a.responses = append(a.responses, response)

	return nil
}

// UnblockIP removes an IP from the blocklist
func (a *Agent) UnblockIP(ip string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.blocklist[ip]; !exists {
		return fmt.Errorf("IP not in blocklist: %s", ip)
	}

	delete(a.blocklist, ip)
	log.Printf("[BlueTeam] IP unblocked: %s", ip)
	return nil
}

// IsBlocked checks if an IP is blocked
func (a *Agent) IsBlocked(ip string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	expiry, exists := a.blocklist[ip]
	if !exists {
		return false
	}

	return time.Now().Before(expiry)
}

// Respond executes a defensive response
func (a *Agent) Respond(ctx context.Context, detectionID string, defenseType DefenseType, action string, target string) (*Response, error) {
	a.mu.Lock()

	// Find detection
	var detection *Detection
	for i := range a.detections {
		if a.detections[i].ID == detectionID {
			detection = &a.detections[i]
			break
		}
	}

	if detection == nil {
		a.mu.Unlock()
		return nil, fmt.Errorf("detection not found: %s", detectionID)
	}

	response := Response{
		ID:          uuid.New().String(),
		DetectionID: detectionID,
		Type:        defenseType,
		Action:      action,
		Target:      target,
		Status:      "in_progress",
		StartTime:   time.Now(),
		Notes:       make([]string, 0),
	}

	detection.Responded = true
	detection.ResponseID = response.ID
	a.responses = append(a.responses, response)
	a.mu.Unlock()

	log.Printf("[BlueTeam] Executing response: %s (%s) on %s", action, defenseType, target)

	// Execute response action
	success := a.executeResponse(ctx, &response)

	a.mu.Lock()
	endTime := time.Now()
	response.EndTime = &endTime
	response.Success = success
	response.Status = "completed"
	if !success {
		response.Status = "failed"
	}

	// Update stored response
	for i := range a.responses {
		if a.responses[i].ID == response.ID {
			a.responses[i] = response
			break
		}
	}
	a.mu.Unlock()

	return &response, nil
}

// GetDetections returns all detections
func (a *Agent) GetDetections() []Detection {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return append([]Detection{}, a.detections...)
}

// GetResponses returns all responses
func (a *Agent) GetResponses() []Response {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return append([]Response{}, a.responses...)
}

// GetBlocklist returns the current blocklist
func (a *Agent) GetBlocklist() map[string]time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := make(map[string]time.Time)
	for ip, expiry := range a.blocklist {
		result[ip] = expiry
	}
	return result
}

// GetStatistics returns agent statistics
func (a *Agent) GetStatistics() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	detectionsByLevel := make(map[ThreatLevel]int)
	for _, d := range a.detections {
		detectionsByLevel[d.ThreatLevel]++
	}

	successfulResponses := 0
	for _, r := range a.responses {
		if r.Success {
			successfulResponses++
		}
	}

	return map[string]interface{}{
		"agent_id":             a.id,
		"agent_name":           a.name,
		"total_detections":     len(a.detections),
		"detections_by_level":  detectionsByLevel,
		"total_responses":      len(a.responses),
		"successful_responses": successfulResponses,
		"active_rules":         len(a.rules),
		"blocklist_size":       len(a.blocklist),
		"auto_respond":         a.config.AutoRespond,
	}
}

// GenerateReport creates a JSON report
func (a *Agent) GenerateReport() ([]byte, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	report := map[string]interface{}{
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"agent":        a.name,
		"statistics":   a.GetStatistics(),
		"detections":   a.detections,
		"responses":    a.responses,
		"blocklist":    a.blocklist,
	}

	return json.MarshalIndent(report, "", "  ")
}

func (a *Agent) processDetections(ctx context.Context) {
	defer a.wg.Done()

	for {
		select {
		case <-a.stopCh:
			return
		case <-ctx.Done():
			return
		case detection := <-a.alertChan:
			if a.config.AutoRespond {
				a.autoRespond(ctx, detection)
			}
		}
	}
}

func (a *Agent) autoRespond(ctx context.Context, detection Detection) {
	var action string
	var target string
	var defenseType DefenseType

	switch detection.ThreatLevel {
	case ThreatLevelCritical:
		defenseType = DefenseTypeContainment
		action = "isolate"
		target = detection.Source
	case ThreatLevelHigh:
		defenseType = DefenseTypeContainment
		action = "block"
		target = detection.Source
	case ThreatLevelMedium:
		defenseType = DefenseTypeDetection
		action = "monitor"
		target = detection.Source
	default:
		// Low/Info - just log
		return
	}

	_, err := a.Respond(ctx, detection.ID, defenseType, action, target)
	if err != nil {
		log.Printf("[BlueTeam] Auto-response failed: %v", err)
	}
}

func (a *Agent) executeResponse(ctx context.Context, response *Response) bool {
	switch response.Action {
	case "block_ip", "block":
		return a.BlockIP(response.Target, fmt.Sprintf("Auto-blocked for detection %s", response.DetectionID)) == nil
	case "isolate":
		response.Notes = append(response.Notes, "Isolation requested; awaiting control plane execution")
		return true
	case "monitor":
		response.Notes = append(response.Notes, "Enhanced monitoring enabled")
		return true
	case "terminate":
		response.Notes = append(response.Notes, "Termination requested; awaiting control plane execution")
		return true
	default:
		response.Notes = append(response.Notes, fmt.Sprintf("Unknown action: %s", response.Action))
		return false
	}
}

func (a *Agent) cleanBlocklist(ctx context.Context) {
	defer a.wg.Done()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.mu.Lock()
			now := time.Now()
			for ip, expiry := range a.blocklist {
				if now.After(expiry) {
					delete(a.blocklist, ip)
					log.Printf("[BlueTeam] IP expired from blocklist: %s", ip)
				}
			}
			a.mu.Unlock()
		}
	}
}

func (a *Agent) matchRule(rule *Rule, eventType string, data map[string]interface{}) bool {
	// Get field value from data
	value, exists := data[rule.Logic.Field]
	if !exists {
		return false
	}

	// Match based on operator
	switch rule.Logic.Operator {
	case "equals":
		return value == rule.Logic.Value
	case "contains":
		if strVal, ok := value.(string); ok {
			if searchVal, ok := rule.Logic.Value.(string); ok {
				return strings.Contains(strVal, searchVal)
			}
		}
		return false
	case "greater_than":
		if numVal, ok := value.(float64); ok {
			if threshold, ok := rule.Logic.Value.(float64); ok {
				return numVal > threshold
			}
		}
		return false
	case "less_than":
		if numVal, ok := value.(float64); ok {
			if threshold, ok := rule.Logic.Value.(float64); ok {
				return numVal < threshold
			}
		}
		return false
	default:
		return false
	}
}

func (a *Agent) loadDefaultRules() {
	// Brute force detection
	a.AddRule(&Rule{
		ID:          "rule-brute-force",
		Name:        "Brute Force Attack",
		Description: "Detects multiple failed authentication attempts",
		Enabled:     true,
		Severity:    ThreatLevelHigh,
		Logic: RuleLogic{
			Field:     "failed_logins",
			Operator:  "greater_than",
			Value:     float64(5),
			Threshold: 5,
			Window:    5 * time.Minute,
		},
		Actions: []string{"block_ip"},
	})

	// Port scan detection
	a.AddRule(&Rule{
		ID:          "rule-port-scan",
		Name:        "Port Scan Detected",
		Description: "Detects rapid connection attempts to multiple ports",
		Enabled:     true,
		Severity:    ThreatLevelMedium,
		Logic: RuleLogic{
			Field:     "unique_ports",
			Operator:  "greater_than",
			Value:     float64(10),
			Threshold: 10,
			Window:    1 * time.Minute,
		},
		Actions: []string{"monitor", "alert"},
	})

	// SQL injection detection
	a.AddRule(&Rule{
		ID:          "rule-sqli",
		Name:        "SQL Injection Attempt",
		Description: "Detects SQL injection patterns in requests",
		Enabled:     true,
		Severity:    ThreatLevelCritical,
		Logic: RuleLogic{
			Field:    "request_body",
			Operator: "contains",
			Value:    "UNION SELECT",
		},
		Actions: []string{"block_ip", "alert"},
	})

	// XSS detection
	a.AddRule(&Rule{
		ID:          "rule-xss",
		Name:        "XSS Attempt",
		Description: "Detects cross-site scripting patterns",
		Enabled:     true,
		Severity:    ThreatLevelHigh,
		Logic: RuleLogic{
			Field:    "request_body",
			Operator: "contains",
			Value:    "<script>",
		},
		Actions: []string{"block_ip", "sanitize"},
	})

	// Anomalous traffic detection
	a.AddRule(&Rule{
		ID:          "rule-traffic-anomaly",
		Name:        "Traffic Anomaly",
		Description: "Detects unusual traffic patterns",
		Enabled:     true,
		Severity:    ThreatLevelMedium,
		Logic: RuleLogic{
			Field:     "requests_per_second",
			Operator:  "greater_than",
			Value:     float64(100),
			Threshold: 100,
			Window:    1 * time.Minute,
		},
		Actions: []string{"rate_limit"},
	})
}
