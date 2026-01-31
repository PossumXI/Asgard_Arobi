// Package redteam implements automated red team security testing agents.
package redteam

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// AttackType categorizes attack vectors
type AttackType string

const (
	AttackTypeReconnaissance AttackType = "reconnaissance"
	AttackTypeExploitation   AttackType = "exploitation"
	AttackTypePersistence    AttackType = "persistence"
	AttackTypeLateralMove    AttackType = "lateral_movement"
	AttackTypeExfiltration   AttackType = "exfiltration"
	AttackTypePrivEsc        AttackType = "privilege_escalation"
	AttackTypeDenial         AttackType = "denial_of_service"
)

// AttackResult holds the outcome of an attack attempt
type AttackResult struct {
	ID         string
	AttackType AttackType
	Target     string
	Success    bool
	Blocked    bool
	BlockedBy  string
	StartTime  time.Time
	EndTime    time.Time
	Evidence   []string
	Findings   []Finding
	MITRE      []string // MITRE ATT&CK technique IDs
}

// Finding represents a discovered vulnerability
type Finding struct {
	ID          string
	Severity    string
	Title       string
	Description string
	Remediation string
	CVE         string
	CVSS        float64
}

// Campaign represents a coordinated attack campaign
type Campaign struct {
	ID          string
	Name        string
	Description string
	Targets     []string
	Attacks     []AttackType
	Status      string
	StartTime   time.Time
	EndTime     *time.Time
	Results     []AttackResult
}

// Agent is an automated red team agent
type Agent struct {
	mu         sync.RWMutex
	id         string
	name       string
	campaigns  map[string]*Campaign
	results    []AttackResult
	config     AgentConfig
	httpClient *http.Client
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// AgentConfig configures the red team agent
type AgentConfig struct {
	MaxConcurrentAttacks int
	AttackTimeout        time.Duration
	SafeMode             bool // Prevents actual exploitation
	TargetScope          []string
	ExcludedTargets      []string
}

// DefaultAgentConfig returns safe defaults
func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		MaxConcurrentAttacks: 5,
		AttackTimeout:        30 * time.Second,
		SafeMode:             true,
		TargetScope:          []string{"127.0.0.1", "localhost"},
		ExcludedTargets:      []string{},
	}
}

// NewAgent creates a new red team agent
func NewAgent(name string, cfg AgentConfig) *Agent {
	return &Agent{
		id:        uuid.New().String(),
		name:      name,
		campaigns: make(map[string]*Campaign),
		results:   make([]AttackResult, 0),
		config:    cfg,
		httpClient: &http.Client{
			Timeout: cfg.AttackTimeout,
		},
		stopCh: make(chan struct{}),
	}
}

// Start begins the agent
func (a *Agent) Start(ctx context.Context) error {
	log.Printf("[RedTeam] Agent %s (%s) started", a.name, a.id)
	return nil
}

// Stop shuts down the agent
func (a *Agent) Stop() {
	close(a.stopCh)
	a.wg.Wait()
	log.Printf("[RedTeam] Agent %s stopped", a.name)
}

// CreateCampaign sets up a new attack campaign
func (a *Agent) CreateCampaign(name, description string, targets []string, attacks []AttackType) (*Campaign, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Validate targets against scope
	for _, target := range targets {
		if !a.isInScope(target) {
			return nil, fmt.Errorf("target %s is not in scope", target)
		}
	}

	campaign := &Campaign{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Targets:     targets,
		Attacks:     attacks,
		Status:      "created",
		Results:     make([]AttackResult, 0),
	}

	a.campaigns[campaign.ID] = campaign
	log.Printf("[RedTeam] Campaign created: %s (%s)", name, campaign.ID)
	return campaign, nil
}

// ExecuteCampaign runs all attacks in a campaign
func (a *Agent) ExecuteCampaign(ctx context.Context, campaignID string) error {
	a.mu.Lock()
	campaign, exists := a.campaigns[campaignID]
	if !exists {
		a.mu.Unlock()
		return fmt.Errorf("campaign not found: %s", campaignID)
	}
	campaign.Status = "running"
	campaign.StartTime = time.Now()
	a.mu.Unlock()

	log.Printf("[RedTeam] Executing campaign: %s", campaign.Name)

	// Run attacks concurrently with limit
	sem := make(chan struct{}, a.config.MaxConcurrentAttacks)
	var wg sync.WaitGroup

	for _, target := range campaign.Targets {
		for _, attackType := range campaign.Attacks {
			wg.Add(1)
			go func(t string, at AttackType) {
				defer wg.Done()
				select {
				case sem <- struct{}{}:
					defer func() { <-sem }()
				case <-ctx.Done():
					return
				}

				result := a.executeAttack(ctx, t, at)

				a.mu.Lock()
				campaign.Results = append(campaign.Results, result)
				a.results = append(a.results, result)
				a.mu.Unlock()
			}(target, attackType)
		}
	}

	wg.Wait()

	a.mu.Lock()
	campaign.Status = "completed"
	now := time.Now()
	campaign.EndTime = &now
	a.mu.Unlock()

	log.Printf("[RedTeam] Campaign %s completed with %d results", campaign.Name, len(campaign.Results))
	return nil
}

// ExecuteAttack runs a single attack
func (a *Agent) ExecuteAttack(ctx context.Context, target string, attackType AttackType) (*AttackResult, error) {
	if !a.isInScope(target) {
		return nil, fmt.Errorf("target %s is not in scope", target)
	}

	result := a.executeAttack(ctx, target, attackType)

	a.mu.Lock()
	a.results = append(a.results, result)
	a.mu.Unlock()

	return &result, nil
}

func (a *Agent) executeAttack(ctx context.Context, target string, attackType AttackType) AttackResult {
	result := AttackResult{
		ID:         uuid.New().String(),
		AttackType: attackType,
		Target:     target,
		StartTime:  time.Now(),
		Evidence:   make([]string, 0),
		Findings:   make([]Finding, 0),
		MITRE:      make([]string, 0),
	}

	log.Printf("[RedTeam] Executing %s attack on %s", attackType, target)

	// Execute attack based on type
	switch attackType {
	case AttackTypeReconnaissance:
		a.executeRecon(ctx, &result)
	case AttackTypeExploitation:
		a.executeExploit(ctx, &result)
	case AttackTypePersistence:
		a.executePersistence(ctx, &result)
	case AttackTypeLateralMove:
		a.executeLateralMove(ctx, &result)
	case AttackTypeExfiltration:
		a.executeExfiltration(ctx, &result)
	case AttackTypePrivEsc:
		a.executePrivEsc(ctx, &result)
	case AttackTypeDenial:
		a.executeDenial(ctx, &result)
	}

	result.EndTime = time.Now()
	return result
}

func (a *Agent) executeRecon(ctx context.Context, result *AttackResult) {
	result.MITRE = append(result.MITRE, "T1046", "T1018", "T1082")

	commonPorts := []int{22, 80, 443, 3306, 5432, 6379, 8080, 9090}
	openPorts := a.scanOpenPorts(result.Target, commonPorts, 2*time.Second)
	for _, port := range openPorts {
		result.Evidence = append(result.Evidence, fmt.Sprintf("Port %d is open", port))
	}

	if len(openPorts) > 0 {
		result.Success = true
		result.Findings = append(result.Findings, Finding{
			ID:          uuid.New().String(),
			Severity:    "info",
			Title:       "Open Ports Discovered",
			Description: fmt.Sprintf("Found %d open ports on target", len(openPorts)),
			Remediation: "Review firewall rules and close unnecessary ports",
		})
	}

	// HTTP service detection
	resp, err := a.httpClient.Get(fmt.Sprintf("http://%s", result.Target))
	if err == nil {
		resp.Body.Close()
		if serverHeader := resp.Header.Get("Server"); serverHeader != "" {
			result.Evidence = append(result.Evidence, fmt.Sprintf("HTTP service detected: %s", serverHeader))
		} else {
			result.Evidence = append(result.Evidence, "HTTP service detected")
		}
	}
}

func (a *Agent) executeExploit(ctx context.Context, result *AttackResult) {
	result.MITRE = append(result.MITRE, "T1190", "T1210")

	if a.config.SafeMode {
		result.Evidence = append(result.Evidence, "Safe mode enabled: non-invasive checks only")
	}

	// Check for common exposures via HTTP
	vulnChecks := []struct {
		path string
		vuln string
	}{
		{"/.git/config", "Exposed Git Repository"},
		{"/.env", "Exposed Environment File"},
		{"/robots.txt", "Robots.txt Information Disclosure"},
		{"/server-status", "Apache Server Status Exposed"},
		{"/phpinfo.php", "PHP Info Exposure"},
	}

	for _, check := range vulnChecks {
		url := fmt.Sprintf("http://%s%s", result.Target, check.path)
		resp, err := a.httpClient.Get(url)
		if err == nil {
			if resp.StatusCode == 200 {
				result.Success = true
				result.Findings = append(result.Findings, Finding{
					ID:          uuid.New().String(),
					Severity:    "high",
					Title:       check.vuln,
					Description: fmt.Sprintf("Found accessible endpoint: %s", check.path),
					Remediation: "Restrict access to sensitive files",
					CVSS:        7.5,
				})
			}
			resp.Body.Close()
		}
	}
}

func (a *Agent) executePersistence(ctx context.Context, result *AttackResult) {
	result.MITRE = append(result.MITRE, "T1136", "T1078", "T1543")

	ports := []int{22, 3389, 445, 5985, 5986}
	openPorts := a.scanOpenPorts(result.Target, ports, 2*time.Second)
	if len(openPorts) == 0 {
		return
	}

	for _, port := range openPorts {
		result.Evidence = append(result.Evidence, fmt.Sprintf("Remote access port open: %d", port))
	}

	result.Success = true
	result.Findings = append(result.Findings, Finding{
		ID:          uuid.New().String(),
		Severity:    "medium",
		Title:       "Remote Access Services Exposed",
		Description: fmt.Sprintf("Detected %d remote access ports open", len(openPorts)),
		Remediation: "Restrict remote access services to trusted networks and enforce MFA",
	})
}

func (a *Agent) executeLateralMove(ctx context.Context, result *AttackResult) {
	result.MITRE = append(result.MITRE, "T1021", "T1563")

	ports := []int{22, 135, 445, 5985, 5986}
	openPorts := a.scanOpenPorts(result.Target, ports, 2*time.Second)
	if len(openPorts) == 0 {
		return
	}

	for _, port := range openPorts {
		result.Evidence = append(result.Evidence, fmt.Sprintf("Lateral movement protocol port open: %d", port))
	}

	result.Success = true
	result.Findings = append(result.Findings, Finding{
		ID:          uuid.New().String(),
		Severity:    "medium",
		Title:       "Lateral Movement Pathways Available",
		Description: fmt.Sprintf("Detected %d lateral movement ports open", len(openPorts)),
		Remediation: "Limit lateral movement protocols with segmentation and least privilege",
	})
}

func (a *Agent) executeExfiltration(ctx context.Context, result *AttackResult) {
	result.MITRE = append(result.MITRE, "T1048", "T1041")

	ports := []int{53, 80, 443, 8080}
	openPorts := a.scanOpenPorts(result.Target, ports, 2*time.Second)
	if len(openPorts) == 0 {
		return
	}

	for _, port := range openPorts {
		result.Evidence = append(result.Evidence, fmt.Sprintf("Egress-friendly port open: %d", port))
	}

	result.Success = true
	result.Findings = append(result.Findings, Finding{
		ID:          uuid.New().String(),
		Severity:    "low",
		Title:       "Potential Exfiltration Channels",
		Description: "Detected exposed services on common egress ports",
		Remediation: "Monitor outbound traffic and enforce egress filtering",
	})
}

func (a *Agent) executePrivEsc(ctx context.Context, result *AttackResult) {
	result.MITRE = append(result.MITRE, "T1548", "T1068")

	ports := []int{22, 3389, 5900, 3306, 5432}
	openPorts := a.scanOpenPorts(result.Target, ports, 2*time.Second)
	if len(openPorts) == 0 {
		return
	}

	for _, port := range openPorts {
		result.Evidence = append(result.Evidence, fmt.Sprintf("Privileged service port open: %d", port))
	}

	result.Success = true
	result.Findings = append(result.Findings, Finding{
		ID:          uuid.New().String(),
		Severity:    "low",
		Title:       "Privileged Services Exposed",
		Description: "Detected exposed services that often require elevated access",
		Remediation: "Restrict admin services and enforce strong authentication",
	})
}

func (a *Agent) executeDenial(ctx context.Context, result *AttackResult) {
	result.MITRE = append(result.MITRE, "T1498", "T1499")

	url := fmt.Sprintf("http://%s", result.Target)
	limited := 0
	successful := 0
	for i := 0; i < 5; i++ {
		req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
		if err != nil {
			continue
		}
		resp, err := a.httpClient.Do(req)
		if err != nil {
			continue
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			limited++
		} else if resp.StatusCode < 500 {
			successful++
		}
		resp.Body.Close()
	}

	if limited > 0 {
		result.Blocked = true
		result.Evidence = append(result.Evidence, "Rate limiting detected")
		return
	}
	if successful > 0 {
		result.Success = true
		result.Findings = append(result.Findings, Finding{
			ID:          uuid.New().String(),
			Severity:    "low",
			Title:       "Rate Limiting Not Detected",
			Description: "No rate limiting responses observed during safe probe",
			Remediation: "Enable rate limiting and connection throttling at the edge",
		})
	}
}

// GetResults returns all attack results
func (a *Agent) GetResults() []AttackResult {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return append([]AttackResult{}, a.results...)
}

// GetCampaign returns campaign details
func (a *Agent) GetCampaign(campaignID string) (*Campaign, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	campaign, exists := a.campaigns[campaignID]
	if !exists {
		return nil, fmt.Errorf("campaign not found: %s", campaignID)
	}
	return campaign, nil
}

// GetStatistics returns agent statistics
func (a *Agent) GetStatistics() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	successful := 0
	blocked := 0
	for _, r := range a.results {
		if r.Success {
			successful++
		}
		if r.Blocked {
			blocked++
		}
	}

	return map[string]interface{}{
		"agent_id":           a.id,
		"agent_name":         a.name,
		"total_campaigns":    len(a.campaigns),
		"total_attacks":      len(a.results),
		"successful_attacks": successful,
		"blocked_attacks":    blocked,
		"safe_mode":          a.config.SafeMode,
	}
}

// GenerateReport creates a JSON report of findings
func (a *Agent) GenerateReport() ([]byte, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	report := map[string]interface{}{
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"agent":        a.name,
		"statistics":   a.GetStatistics(),
		"campaigns":    a.campaigns,
		"all_findings": a.collectFindings(),
	}

	return json.MarshalIndent(report, "", "  ")
}

func (a *Agent) collectFindings() []Finding {
	findings := make([]Finding, 0)
	for _, r := range a.results {
		findings = append(findings, r.Findings...)
	}
	return findings
}

func (a *Agent) isInScope(target string) bool {
	// Check exclusions
	for _, excluded := range a.config.ExcludedTargets {
		if target == excluded {
			return false
		}
	}

	// Check scope
	for _, allowed := range a.config.TargetScope {
		if target == allowed || allowed == "*" {
			return true
		}
	}

	return false
}

// Utility for generating realistic but safe test data
func (a *Agent) scanOpenPorts(target string, ports []int, timeout time.Duration) []int {
	openPorts := make([]int, 0)
	for _, port := range ports {
		addr := fmt.Sprintf("%s:%d", target, port)
		conn, err := net.DialTimeout("tcp", addr, timeout)
		if err == nil {
			conn.Close()
			openPorts = append(openPorts, port)
		}
	}
	return openPorts
}
