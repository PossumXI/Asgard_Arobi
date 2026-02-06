// Package vault implements an AI-powered monitoring agent for vault security.
//
// The VaultAgent uses behavioral analysis and anomaly detection to identify
// suspicious access patterns and potential security breaches.
//
// Copyright 2026 Arobi. All Rights Reserved.
package vault

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

// AccessEvent represents a vault access event
type AccessEvent struct {
	Type          string
	EntryID       string
	Actor         string
	SecurityLevel SecurityLevel
	Timestamp     time.Time
	IP            string
	UserAgent     string
	FIDO2Used     bool
	Success       bool
}

// AnomalyType categorizes detected anomalies
type AnomalyType string

const (
	AnomalyTypeRapidAccess      AnomalyType = "rapid_access"
	AnomalyTypeUnusualTime      AnomalyType = "unusual_time"
	AnomalyTypeBruteForce       AnomalyType = "brute_force"
	AnomalyTypeElevationAttempt AnomalyType = "elevation_attempt"
	AnomalyTypeDataExfiltration AnomalyType = "data_exfiltration"
	AnomalyTypeNewLocation      AnomalyType = "new_location"
	AnomalyTypeFailedFIDO2      AnomalyType = "failed_fido2"
)

// SecurityAnomaly represents a detected security anomaly
type SecurityAnomaly struct {
	ID          string
	Type        AnomalyType
	Severity    string // critical, high, medium, low
	Actor       string
	Description string
	Evidence    []AccessEvent
	DetectedAt  time.Time
	Resolved    bool
	Response    string
}

// VaultAgentConfig holds agent configuration
type VaultAgentConfig struct {
	// Rate limiting thresholds
	MaxAccessPerMinute   int
	MaxFailedAttempts    int
	SuspiciousTimeWindow time.Duration

	// Analysis settings
	EnableBehaviorAnalysis bool
	EnableAnomalyDetection bool
	LearningPeriod         time.Duration

	// Alert thresholds
	AlertOnFailedFIDO2        bool
	AlertOnHighSecurityAccess bool
	AlertOnUnusualTime        bool
}

// DefaultVaultAgentConfig returns default configuration
func DefaultVaultAgentConfig() VaultAgentConfig {
	return VaultAgentConfig{
		MaxAccessPerMinute:        30,
		MaxFailedAttempts:         5,
		SuspiciousTimeWindow:      time.Minute,
		EnableBehaviorAnalysis:    true,
		EnableAnomalyDetection:    true,
		LearningPeriod:            7 * 24 * time.Hour, // 1 week
		AlertOnFailedFIDO2:        true,
		AlertOnHighSecurityAccess: true,
		AlertOnUnusualTime:        true,
	}
}

// UserBehaviorProfile represents learned behavior for a user
type UserBehaviorProfile struct {
	UserID            string
	FirstSeen         time.Time
	LastSeen          time.Time
	TotalAccesses     int
	TypicalHours      []int          // hours of day (0-23) when user is active
	TypicalDays       []int          // days of week (0-6) when user is active
	AccessedSecrets   map[string]int // secretID -> access count
	CommonIPs         map[string]int
	AverageAccessRate float64 // accesses per hour
	FailedAttempts    int
}

// VaultAgent monitors vault access and detects anomalies
type VaultAgent struct {
	mu          sync.RWMutex
	vault       *Vault
	config      VaultAgentConfig
	events      []AccessEvent
	anomalies   []SecurityAnomaly
	profiles    map[string]*UserBehaviorProfile
	eventChan   chan AccessEvent
	anomalyChan chan SecurityAnomaly
	stopCh      chan struct{}
	wg          sync.WaitGroup
	running     bool
}

// NewVaultAgent creates a new vault monitoring agent
func NewVaultAgent(vault *Vault) *VaultAgent {
	return &VaultAgent{
		vault:       vault,
		config:      DefaultVaultAgentConfig(),
		events:      make([]AccessEvent, 0),
		anomalies:   make([]SecurityAnomaly, 0),
		profiles:    make(map[string]*UserBehaviorProfile),
		eventChan:   make(chan AccessEvent, 1000),
		anomalyChan: make(chan SecurityAnomaly, 100),
		stopCh:      make(chan struct{}),
	}
}

// NewVaultAgentWithConfig creates an agent with custom configuration
func NewVaultAgentWithConfig(vault *Vault, cfg VaultAgentConfig) *VaultAgent {
	agent := NewVaultAgent(vault)
	agent.config = cfg
	return agent
}

// Start begins the monitoring agent
func (a *VaultAgent) Start(ctx context.Context) {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return
	}
	a.running = true
	a.mu.Unlock()

	log.Printf("[VaultAgent] Starting security monitoring agent")

	// Start event processor
	a.wg.Add(1)
	go a.processEvents(ctx)

	// Start periodic analysis
	a.wg.Add(1)
	go a.periodicAnalysis(ctx)

	// Start anomaly handler
	a.wg.Add(1)
	go a.handleAnomalies(ctx)

	log.Printf("[VaultAgent] Agent started with behavioral analysis=%v, anomaly detection=%v",
		a.config.EnableBehaviorAnalysis, a.config.EnableAnomalyDetection)
}

// Stop shuts down the monitoring agent
func (a *VaultAgent) Stop() {
	a.mu.Lock()
	if !a.running {
		a.mu.Unlock()
		return
	}
	a.running = false
	a.mu.Unlock()

	close(a.stopCh)
	a.wg.Wait()
	log.Printf("[VaultAgent] Agent stopped")
}

// NotifyAccess reports an access event to the agent
func (a *VaultAgent) NotifyAccess(event AccessEvent) {
	select {
	case a.eventChan <- event:
	default:
		log.Printf("[VaultAgent] Warning: event channel full, dropping event")
	}
}

// processEvents handles incoming access events
func (a *VaultAgent) processEvents(ctx context.Context) {
	defer a.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case event := <-a.eventChan:
			a.analyzeEvent(event)
		}
	}
}

// analyzeEvent analyzes a single access event
func (a *VaultAgent) analyzeEvent(event AccessEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Store event
	a.events = append(a.events, event)
	if len(a.events) > 10000 {
		a.events = a.events[1000:] // Keep last 9000
	}

	// Update user profile
	profile := a.getOrCreateProfile(event.Actor)
	profile.LastSeen = event.Timestamp
	profile.TotalAccesses++

	hour := event.Timestamp.Hour()
	if !contains(profile.TypicalHours, hour) {
		profile.TypicalHours = append(profile.TypicalHours, hour)
	}

	day := int(event.Timestamp.Weekday())
	if !contains(profile.TypicalDays, day) {
		profile.TypicalDays = append(profile.TypicalDays, day)
	}

	if event.EntryID != "" {
		profile.AccessedSecrets[event.EntryID]++
	}

	if event.IP != "" {
		profile.CommonIPs[event.IP]++
	}

	if !event.Success {
		profile.FailedAttempts++
	}

	// Run anomaly detection
	if a.config.EnableAnomalyDetection {
		a.detectAnomalies(event, profile)
	}
}

// detectAnomalies checks for suspicious patterns
func (a *VaultAgent) detectAnomalies(event AccessEvent, profile *UserBehaviorProfile) {
	// Check for rapid access (rate limiting)
	recentAccesses := a.countRecentAccesses(event.Actor, time.Minute)
	if recentAccesses > a.config.MaxAccessPerMinute {
		a.raiseAnomaly(SecurityAnomaly{
			Type:        AnomalyTypeRapidAccess,
			Severity:    "high",
			Actor:       event.Actor,
			Description: fmt.Sprintf("Rapid access detected: %d accesses in 1 minute (limit: %d)", recentAccesses, a.config.MaxAccessPerMinute),
			Evidence:    a.getRecentEvents(event.Actor, 10),
			DetectedAt:  time.Now(),
		})
	}

	// Check for unusual time access
	if a.config.AlertOnUnusualTime && len(profile.TypicalHours) > 5 {
		hour := event.Timestamp.Hour()
		if !contains(profile.TypicalHours, hour) {
			a.raiseAnomaly(SecurityAnomaly{
				Type:        AnomalyTypeUnusualTime,
				Severity:    "medium",
				Actor:       event.Actor,
				Description: fmt.Sprintf("Access at unusual time: %02d:00 (typical hours: %v)", hour, profile.TypicalHours),
				Evidence:    []AccessEvent{event},
				DetectedAt:  time.Now(),
			})
		}
	}

	// Check for brute force (failed attempts)
	recentFailures := a.countRecentFailures(event.Actor, 5*time.Minute)
	if recentFailures >= a.config.MaxFailedAttempts {
		a.raiseAnomaly(SecurityAnomaly{
			Type:        AnomalyTypeBruteForce,
			Severity:    "critical",
			Actor:       event.Actor,
			Description: fmt.Sprintf("Possible brute force: %d failed attempts in 5 minutes", recentFailures),
			Evidence:    a.getRecentFailedEvents(event.Actor, 10),
			DetectedAt:  time.Now(),
		})
	}

	// Check for failed FIDO2
	if a.config.AlertOnFailedFIDO2 && !event.Success && event.FIDO2Used {
		a.raiseAnomaly(SecurityAnomaly{
			Type:        AnomalyTypeFailedFIDO2,
			Severity:    "high",
			Actor:       event.Actor,
			Description: "Failed FIDO2 authentication attempt",
			Evidence:    []AccessEvent{event},
			DetectedAt:  time.Now(),
		})
	}

	// Check for high security access patterns
	if a.config.AlertOnHighSecurityAccess && (event.SecurityLevel == SecurityLevelGovernment || event.SecurityLevel == SecurityLevelMilitary) {
		// Log but don't necessarily raise anomaly
		log.Printf("[VaultAgent] High security access: %s accessed %s-level secret", event.Actor, event.SecurityLevel)
	}

	// Check for potential data exfiltration (high volume retrieval)
	recentRetrieves := a.countRecentRetrieves(event.Actor, 10*time.Minute)
	if recentRetrieves > 50 {
		a.raiseAnomaly(SecurityAnomaly{
			Type:        AnomalyTypeDataExfiltration,
			Severity:    "critical",
			Actor:       event.Actor,
			Description: fmt.Sprintf("Potential data exfiltration: %d secrets retrieved in 10 minutes", recentRetrieves),
			Evidence:    a.getRecentEvents(event.Actor, 20),
			DetectedAt:  time.Now(),
		})
	}

	// Check for new IP
	if event.IP != "" && len(profile.CommonIPs) > 3 {
		if _, exists := profile.CommonIPs[event.IP]; !exists {
			a.raiseAnomaly(SecurityAnomaly{
				Type:        AnomalyTypeNewLocation,
				Severity:    "low",
				Actor:       event.Actor,
				Description: fmt.Sprintf("Access from new IP: %s", event.IP),
				Evidence:    []AccessEvent{event},
				DetectedAt:  time.Now(),
			})
		}
	}
}

// raiseAnomaly creates and stores an anomaly
func (a *VaultAgent) raiseAnomaly(anomaly SecurityAnomaly) {
	anomaly.ID = fmt.Sprintf("anomaly-%d", time.Now().UnixNano())
	a.anomalies = append(a.anomalies, anomaly)

	// Send to anomaly channel for handling
	select {
	case a.anomalyChan <- anomaly:
	default:
		log.Printf("[VaultAgent] Warning: anomaly channel full")
	}

	log.Printf("[VaultAgent] ANOMALY DETECTED: %s - %s (%s)", anomaly.Type, anomaly.Description, anomaly.Severity)
}

// handleAnomalies processes detected anomalies
func (a *VaultAgent) handleAnomalies(ctx context.Context) {
	defer a.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case anomaly := <-a.anomalyChan:
			a.respondToAnomaly(anomaly)
		}
	}
}

// respondToAnomaly takes action on detected anomalies
func (a *VaultAgent) respondToAnomaly(anomaly SecurityAnomaly) {
	switch anomaly.Severity {
	case "critical":
		// For critical anomalies, consider blocking the actor
		log.Printf("[VaultAgent] CRITICAL: %s - Consider blocking actor: %s", anomaly.Type, anomaly.Actor)
		// In production: trigger immediate alert, possibly auto-lock vault

	case "high":
		// High severity: alert security team
		log.Printf("[VaultAgent] HIGH: %s - Alert security team about: %s", anomaly.Type, anomaly.Actor)
		// In production: send alert to security operations

	case "medium":
		// Medium severity: log and monitor
		log.Printf("[VaultAgent] MEDIUM: %s - Monitoring actor: %s", anomaly.Type, anomaly.Actor)

	case "low":
		// Low severity: just log
		log.Printf("[VaultAgent] LOW: %s - Noted for actor: %s", anomaly.Type, anomaly.Actor)
	}

	// Log to vault audit
	if a.vault != nil && a.vault.auditLog != nil {
		a.vault.auditLog.LogEvent(AuditEvent{
			Timestamp: time.Now(),
			Action:    AuditActionSuspiciousAccess,
			Actor:     anomaly.Actor,
			Success:   false,
			Details:   anomaly.Description,
		})
	}
}

// periodicAnalysis runs scheduled analysis
func (a *VaultAgent) periodicAnalysis(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.runPeriodicAnalysis()
		}
	}
}

// runPeriodicAnalysis performs scheduled security analysis
func (a *VaultAgent) runPeriodicAnalysis() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Calculate statistics
	now := time.Now()
	hourAgo := now.Add(-time.Hour)

	accessesLastHour := 0
	failuresLastHour := 0
	uniqueActors := make(map[string]bool)

	for _, event := range a.events {
		if event.Timestamp.After(hourAgo) {
			accessesLastHour++
			uniqueActors[event.Actor] = true
			if !event.Success {
				failuresLastHour++
			}
		}
	}

	log.Printf("[VaultAgent] Periodic analysis: %d accesses, %d failures, %d unique actors in last hour",
		accessesLastHour, failuresLastHour, len(uniqueActors))

	// Clean old events
	cutoff := now.Add(-24 * time.Hour)
	newEvents := make([]AccessEvent, 0)
	for _, event := range a.events {
		if event.Timestamp.After(cutoff) {
			newEvents = append(newEvents, event)
		}
	}
	a.events = newEvents

	// Clean old anomalies
	anomalyCutoff := now.Add(-7 * 24 * time.Hour)
	newAnomalies := make([]SecurityAnomaly, 0)
	for _, anomaly := range a.anomalies {
		if anomaly.DetectedAt.After(anomalyCutoff) {
			newAnomalies = append(newAnomalies, anomaly)
		}
	}
	a.anomalies = newAnomalies
}

// Helper methods

func (a *VaultAgent) getOrCreateProfile(actor string) *UserBehaviorProfile {
	profile, exists := a.profiles[actor]
	if !exists {
		profile = &UserBehaviorProfile{
			UserID:          actor,
			FirstSeen:       time.Now(),
			LastSeen:        time.Now(),
			TypicalHours:    make([]int, 0),
			TypicalDays:     make([]int, 0),
			AccessedSecrets: make(map[string]int),
			CommonIPs:       make(map[string]int),
		}
		a.profiles[actor] = profile
	}
	return profile
}

func (a *VaultAgent) countRecentAccesses(actor string, window time.Duration) int {
	cutoff := time.Now().Add(-window)
	count := 0
	for _, event := range a.events {
		if event.Actor == actor && event.Timestamp.After(cutoff) {
			count++
		}
	}
	return count
}

func (a *VaultAgent) countRecentFailures(actor string, window time.Duration) int {
	cutoff := time.Now().Add(-window)
	count := 0
	for _, event := range a.events {
		if event.Actor == actor && !event.Success && event.Timestamp.After(cutoff) {
			count++
		}
	}
	return count
}

func (a *VaultAgent) countRecentRetrieves(actor string, window time.Duration) int {
	cutoff := time.Now().Add(-window)
	count := 0
	for _, event := range a.events {
		if event.Actor == actor && event.Type == "retrieve" && event.Timestamp.After(cutoff) {
			count++
		}
	}
	return count
}

func (a *VaultAgent) getRecentEvents(actor string, limit int) []AccessEvent {
	result := make([]AccessEvent, 0)
	for i := len(a.events) - 1; i >= 0 && len(result) < limit; i-- {
		if a.events[i].Actor == actor {
			result = append(result, a.events[i])
		}
	}
	return result
}

func (a *VaultAgent) getRecentFailedEvents(actor string, limit int) []AccessEvent {
	result := make([]AccessEvent, 0)
	for i := len(a.events) - 1; i >= 0 && len(result) < limit; i-- {
		if a.events[i].Actor == actor && !a.events[i].Success {
			result = append(result, a.events[i])
		}
	}
	return result
}

// GetAnomalies returns all detected anomalies
func (a *VaultAgent) GetAnomalies() []SecurityAnomaly {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return append([]SecurityAnomaly{}, a.anomalies...)
}

// GetStatistics returns agent statistics
func (a *VaultAgent) GetStatistics() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	anomaliesBySeverity := make(map[string]int)
	for _, anomaly := range a.anomalies {
		anomaliesBySeverity[anomaly.Severity]++
	}

	return map[string]interface{}{
		"events_tracked":        len(a.events),
		"anomalies_detected":    len(a.anomalies),
		"user_profiles":         len(a.profiles),
		"anomalies_by_severity": anomaliesBySeverity,
		"running":               a.running,
	}
}

// GetUserProfile returns the behavior profile for a user
func (a *VaultAgent) GetUserProfile(userID string) *UserBehaviorProfile {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if profile, exists := a.profiles[userID]; exists {
		// Return copy
		copy := *profile
		return &copy
	}
	return nil
}

// CalculateRiskScore calculates a risk score for a user
func (a *VaultAgent) CalculateRiskScore(userID string) float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()

	profile := a.profiles[userID]
	if profile == nil {
		return 0.5 // Unknown user = medium risk
	}

	score := 0.0
	factors := 0.0

	// Factor 1: Account age (newer = higher risk)
	accountAge := time.Since(profile.FirstSeen)
	if accountAge < 24*time.Hour {
		score += 0.8
	} else if accountAge < 7*24*time.Hour {
		score += 0.5
	} else {
		score += 0.2
	}
	factors++

	// Factor 2: Failed attempt ratio
	if profile.TotalAccesses > 0 {
		failRatio := float64(profile.FailedAttempts) / float64(profile.TotalAccesses)
		score += math.Min(failRatio*2, 1.0)
		factors++
	}

	// Factor 3: Recent anomalies
	recentAnomalies := 0
	for _, anomaly := range a.anomalies {
		if anomaly.Actor == userID && time.Since(anomaly.DetectedAt) < 24*time.Hour {
			recentAnomalies++
		}
	}
	score += math.Min(float64(recentAnomalies)*0.2, 1.0)
	factors++

	// Factor 4: Access diversity
	if len(profile.CommonIPs) > 5 {
		score += 0.3 // Many IPs = slightly suspicious
		factors++
	}

	if factors == 0 {
		return 0.5
	}

	return score / factors
}

func contains(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
