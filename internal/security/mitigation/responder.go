package mitigation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/smtp"
	"sync"
	"time"

	"github.com/asgard/pandora/internal/platform/observability"
	"github.com/asgard/pandora/internal/security/threat"
)

// MitigationAction represents a mitigation response
type MitigationAction struct {
	ThreatID   string
	ActionType string
	Target     string
	Parameters map[string]interface{}
	ExecutedAt time.Time
	Success    bool
	Error      string
}

// FirewallBackend defines the interface for firewall operations
type FirewallBackend interface {
	// BlockIP blocks an IP address for a specified duration
	BlockIP(ctx context.Context, ip string, duration time.Duration, reason string) error
	// UnblockIP removes a block on an IP address
	UnblockIP(ctx context.Context, ip string) error
	// IsBlocked checks if an IP is currently blocked
	IsBlocked(ip string) bool
	// ListBlocked returns all currently blocked IPs
	ListBlocked() []BlockedIP
}

// BlockedIP represents a blocked IP entry
type BlockedIP struct {
	IP        string
	BlockedAt time.Time
	ExpiresAt time.Time
	Reason    string
}

// AlertBackend defines the interface for alerting operations
type AlertBackend interface {
	// SendAlert sends an alert notification
	SendAlert(ctx context.Context, alert Alert) error
}

// Alert represents an alert notification
type Alert struct {
	ID          string
	Severity    string
	Title       string
	Description string
	Source      string
	Target      string
	Timestamp   time.Time
	Metadata    map[string]interface{}
}

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	mu       sync.RWMutex
	limits   map[string]*rateLimitEntry
	maxReqs  int
	window   time.Duration
	blockDur time.Duration
}

type rateLimitEntry struct {
	requests  []time.Time
	blocked   bool
	blockedAt time.Time
}

// ResponderConfig holds configuration for the Responder
type ResponderConfig struct {
	// Firewall backend (optional)
	FirewallBackend FirewallBackend
	// Alert backends (optional, can have multiple)
	AlertBackends []AlertBackend
	// Rate limiting config
	RateLimitRequests int           // Max requests per window
	RateLimitWindow   time.Duration // Window duration
	RateLimitBlockDur time.Duration // Block duration when limit exceeded
	// Webhook URLs for alerting
	WebhookURLs []string
	// Email alerting config
	EmailConfig *EmailConfig
}

// EmailConfig holds email alerting configuration
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	Username     string
	Password     string
	From         string
	To           []string
	UseTLS       bool
}

// Responder handles threat mitigation
type Responder struct {
	actionChan      chan<- MitigationAction
	firewall        FirewallBackend
	alertBackends   []AlertBackend
	rateLimiter     *RateLimiter
	webhookURLs     []string
	emailConfig     *EmailConfig
	httpClient      *http.Client
	mu              sync.RWMutex
	blockedIPs      map[string]*BlockedIP
	mitigationStats MitigationStats
}

// MitigationStats tracks mitigation statistics
type MitigationStats struct {
	TotalActions      int64
	SuccessfulActions int64
	FailedActions     int64
	IPsBlocked        int64
	IPsUnblocked      int64
	AlertsSent        int64
	RateLimited       int64
}

// NewResponder creates a new mitigation responder
func NewResponder(actionChan chan<- MitigationAction) *Responder {
	return NewResponderWithConfig(actionChan, ResponderConfig{
		RateLimitRequests: 100,
		RateLimitWindow:   time.Minute,
		RateLimitBlockDur: time.Hour,
	})
}

// NewResponderWithConfig creates a new responder with custom configuration
func NewResponderWithConfig(actionChan chan<- MitigationAction, config ResponderConfig) *Responder {
	r := &Responder{
		actionChan:    actionChan,
		firewall:      config.FirewallBackend,
		alertBackends: config.AlertBackends,
		webhookURLs:   config.WebhookURLs,
		emailConfig:   config.EmailConfig,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		blockedIPs: make(map[string]*BlockedIP),
	}

	// Initialize rate limiter
	maxReqs := config.RateLimitRequests
	if maxReqs == 0 {
		maxReqs = 100
	}
	window := config.RateLimitWindow
	if window == 0 {
		window = time.Minute
	}
	blockDur := config.RateLimitBlockDur
	if blockDur == 0 {
		blockDur = time.Hour
	}

	r.rateLimiter = &RateLimiter{
		limits:   make(map[string]*rateLimitEntry),
		maxReqs:  maxReqs,
		window:   window,
		blockDur: blockDur,
	}

	// Start background cleanup goroutine
	go r.cleanupExpiredBlocks()

	return r
}

// SetFirewallBackend sets the firewall backend
func (r *Responder) SetFirewallBackend(fb FirewallBackend) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.firewall = fb
}

// AddAlertBackend adds an alert backend
func (r *Responder) AddAlertBackend(ab AlertBackend) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.alertBackends = append(r.alertBackends, ab)
}

// AddWebhookURL adds a webhook URL for alerting
func (r *Responder) AddWebhookURL(url string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.webhookURLs = append(r.webhookURLs, url)
}

// SetEmailConfig sets the email alerting configuration
func (r *Responder) SetEmailConfig(config *EmailConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.emailConfig = config
}

// GetStats returns mitigation statistics
func (r *Responder) GetStats() MitigationStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.mitigationStats
}

// MitigateThreat attempts to mitigate a threat
func (r *Responder) MitigateThreat(ctx context.Context, t threat.Threat) error {
	log.Printf("Mitigating threat: %s (type: %s, severity: %s)", t.ID, t.Type, t.Severity)

	r.mu.Lock()
	r.mitigationStats.TotalActions++
	r.mu.Unlock()

	var action *MitigationAction
	var mitigationErr error

	// Determine mitigation based on threat severity
	switch t.Severity {
	case "critical", "high":
		// For high/critical threats, block immediately and send alerts
		action, mitigationErr = r.handleCriticalThreat(ctx, t)

	case "medium":
		// For medium threats, rate limit and monitor
		action, mitigationErr = r.handleMediumThreat(ctx, t)

	default:
		// For low threats, log and apply soft rate limiting
		action, mitigationErr = r.handleLowThreat(ctx, t)
	}

	if action != nil {
		if mitigationErr != nil {
			action.Success = false
			action.Error = mitigationErr.Error()
		}

		// Update stats
		r.mu.Lock()
		if action.Success {
			r.mitigationStats.SuccessfulActions++
		} else {
			r.mitigationStats.FailedActions++
		}
		r.mu.Unlock()

		select {
		case r.actionChan <- *action:
			log.Printf("Mitigation action executed: %s for threat %s (success: %t)", 
				action.ActionType, action.ThreatID, action.Success)
			observability.RecordMitigation(action.ActionType, action.Success)
		default:
			log.Printf("Action channel full, mitigation may be delayed")
			observability.RecordMitigation(action.ActionType, false)
		}
	}

	return mitigationErr
}

// handleCriticalThreat handles critical/high severity threats
func (r *Responder) handleCriticalThreat(ctx context.Context, t threat.Threat) (*MitigationAction, error) {
	action := &MitigationAction{
		ThreatID:   t.ID.String(),
		ActionType: "block_ip",
		Target:     t.SourceIP,
		Parameters: map[string]interface{}{
			"duration_hours": 24,
			"reason":         t.Description,
			"threat_type":    t.Type,
		},
		ExecutedAt: time.Now(),
		Success:    true,
	}

	// Block the IP
	blockDuration := 24 * time.Hour
	if err := r.BlockIP(ctx, t.SourceIP, blockDuration, t.Description); err != nil {
		log.Printf("Failed to block IP %s: %v", t.SourceIP, err)
		action.Success = false
		action.Error = err.Error()
	}

	// Send alerts for critical threats
	alert := Alert{
		ID:          t.ID.String(),
		Severity:    string(t.Severity),
		Title:       fmt.Sprintf("Critical Threat Detected: %s", t.Type),
		Description: t.Description,
		Source:      t.SourceIP,
		Timestamp:   time.Now(),
		Metadata: map[string]interface{}{
			"threat_type": t.Type,
			"action":      "block_ip",
			"duration":    "24h",
		},
	}

	if err := r.SendAlerts(ctx, alert); err != nil {
		log.Printf("Failed to send alerts for threat %s: %v", t.ID, err)
	}

	return action, nil
}

// handleMediumThreat handles medium severity threats
func (r *Responder) handleMediumThreat(ctx context.Context, t threat.Threat) (*MitigationAction, error) {
	action := &MitigationAction{
		ThreatID:   t.ID.String(),
		ActionType: "rate_limit",
		Target:     t.SourceIP,
		Parameters: map[string]interface{}{
			"duration_minutes": 15,
			"max_requests":     50,
		},
		ExecutedAt: time.Now(),
		Success:    true,
	}

	// Apply rate limiting
	limited, err := r.CheckRateLimit(t.SourceIP)
	if err != nil {
		action.Success = false
		action.Error = err.Error()
		return action, err
	}

	if limited {
		action.ActionType = "block_ip"
		action.Parameters["reason"] = "Rate limit exceeded"
		// Block for shorter duration
		if err := r.BlockIP(ctx, t.SourceIP, time.Hour, "Rate limit exceeded"); err != nil {
			log.Printf("Failed to block rate-limited IP %s: %v", t.SourceIP, err)
		}
	}

	// Send webhook notification for medium threats
	if len(r.webhookURLs) > 0 {
		alert := Alert{
			ID:          t.ID.String(),
			Severity:    string(t.Severity),
			Title:       fmt.Sprintf("Medium Threat Detected: %s", t.Type),
			Description: t.Description,
			Source:      t.SourceIP,
			Timestamp:   time.Now(),
		}
		go r.sendWebhookAlerts(ctx, alert)
	}

	return action, nil
}

// handleLowThreat handles low severity threats
func (r *Responder) handleLowThreat(ctx context.Context, t threat.Threat) (*MitigationAction, error) {
	action := &MitigationAction{
		ThreatID:   t.ID.String(),
		ActionType: "log",
		Target:     t.SourceIP,
		Parameters: map[string]interface{}{
			"threat_type": t.Type,
			"monitored":   true,
		},
		ExecutedAt: time.Now(),
		Success:    true,
	}

	// Just track the request for rate limiting purposes
	_, _ = r.CheckRateLimit(t.SourceIP)

	return action, nil
}

// BlockIP blocks an IP address
func (r *Responder) BlockIP(ctx context.Context, ip string, duration time.Duration, reason string) error {
	// Validate IP address
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}

	r.mu.Lock()
	r.blockedIPs[ip] = &BlockedIP{
		IP:        ip,
		BlockedAt: time.Now(),
		ExpiresAt: time.Now().Add(duration),
		Reason:    reason,
	}
	r.mitigationStats.IPsBlocked++
	r.mu.Unlock()

	log.Printf("Blocked IP %s for %v (reason: %s)", ip, duration, reason)

	// If we have a firewall backend, use it
	if r.firewall != nil {
		if err := r.firewall.BlockIP(ctx, ip, duration, reason); err != nil {
			return fmt.Errorf("firewall block failed: %w", err)
		}
	}

	return nil
}

// UnblockIP removes a block on an IP address
func (r *Responder) UnblockIP(ctx context.Context, ip string) error {
	r.mu.Lock()
	delete(r.blockedIPs, ip)
	r.mitigationStats.IPsUnblocked++
	r.mu.Unlock()

	log.Printf("Unblocked IP %s", ip)

	// If we have a firewall backend, use it
	if r.firewall != nil {
		if err := r.firewall.UnblockIP(ctx, ip); err != nil {
			return fmt.Errorf("firewall unblock failed: %w", err)
		}
	}

	return nil
}

// IsBlocked checks if an IP is currently blocked
func (r *Responder) IsBlocked(ip string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	blocked, exists := r.blockedIPs[ip]
	if !exists {
		return false
	}

	// Check if block has expired
	if time.Now().After(blocked.ExpiresAt) {
		return false
	}

	return true
}

// ListBlockedIPs returns all currently blocked IPs
func (r *Responder) ListBlockedIPs() []BlockedIP {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]BlockedIP, 0, len(r.blockedIPs))
	now := time.Now()
	for _, blocked := range r.blockedIPs {
		if now.Before(blocked.ExpiresAt) {
			result = append(result, *blocked)
		}
	}
	return result
}

// CheckRateLimit checks and updates rate limit for an IP
func (r *Responder) CheckRateLimit(ip string) (bool, error) {
	r.rateLimiter.mu.Lock()
	defer r.rateLimiter.mu.Unlock()

	entry, exists := r.rateLimiter.limits[ip]
	now := time.Now()

	if !exists {
		entry = &rateLimitEntry{
			requests: make([]time.Time, 0),
		}
		r.rateLimiter.limits[ip] = entry
	}

	// Check if currently blocked by rate limiter
	if entry.blocked {
		if now.Sub(entry.blockedAt) < r.rateLimiter.blockDur {
			return true, nil // Still blocked
		}
		// Unblock
		entry.blocked = false
		entry.requests = make([]time.Time, 0)
	}

	// Clean old requests outside the window
	cutoff := now.Add(-r.rateLimiter.window)
	newRequests := make([]time.Time, 0)
	for _, reqTime := range entry.requests {
		if reqTime.After(cutoff) {
			newRequests = append(newRequests, reqTime)
		}
	}
	entry.requests = newRequests

	// Add current request
	entry.requests = append(entry.requests, now)

	// Check if over limit
	if len(entry.requests) > r.rateLimiter.maxReqs {
		entry.blocked = true
		entry.blockedAt = now
		r.mu.Lock()
		r.mitigationStats.RateLimited++
		r.mu.Unlock()
		log.Printf("Rate limit exceeded for IP %s (%d requests in %v)", 
			ip, len(entry.requests), r.rateLimiter.window)
		return true, nil
	}

	return false, nil
}

// SendAlerts sends alerts through all configured backends
func (r *Responder) SendAlerts(ctx context.Context, alert Alert) error {
	var lastErr error

	// Send through alert backends
	r.mu.RLock()
	backends := r.alertBackends
	webhooks := r.webhookURLs
	emailCfg := r.emailConfig
	r.mu.RUnlock()

	for _, backend := range backends {
		if err := backend.SendAlert(ctx, alert); err != nil {
			log.Printf("Alert backend error: %v", err)
			lastErr = err
		}
	}

	// Send webhook alerts
	if len(webhooks) > 0 {
		r.sendWebhookAlerts(ctx, alert)
	}

	// Send email alerts for critical
	if emailCfg != nil && (alert.Severity == "critical" || alert.Severity == "high") {
		if err := r.sendEmailAlert(ctx, alert); err != nil {
			log.Printf("Email alert error: %v", err)
			lastErr = err
		}
	}

	r.mu.Lock()
	r.mitigationStats.AlertsSent++
	r.mu.Unlock()

	return lastErr
}

// sendWebhookAlerts sends alerts to configured webhook URLs
func (r *Responder) sendWebhookAlerts(ctx context.Context, alert Alert) {
	r.mu.RLock()
	urls := r.webhookURLs
	r.mu.RUnlock()

	payload, err := json.Marshal(map[string]interface{}{
		"id":          alert.ID,
		"severity":    alert.Severity,
		"title":       alert.Title,
		"description": alert.Description,
		"source":      alert.Source,
		"target":      alert.Target,
		"timestamp":   alert.Timestamp.Format(time.RFC3339),
		"metadata":    alert.Metadata,
	})
	if err != nil {
		log.Printf("Failed to marshal webhook payload: %v", err)
		return
	}

	for _, url := range urls {
		go func(webhookURL string) {
			req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(payload))
			if err != nil {
				log.Printf("Failed to create webhook request: %v", err)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Alert-Severity", alert.Severity)

			resp, err := r.httpClient.Do(req)
			if err != nil {
				log.Printf("Webhook request failed to %s: %v", webhookURL, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 400 {
				log.Printf("Webhook returned error status %d from %s", resp.StatusCode, webhookURL)
			}
		}(url)
	}
}

// sendEmailAlert sends an email alert
func (r *Responder) sendEmailAlert(ctx context.Context, alert Alert) error {
	r.mu.RLock()
	cfg := r.emailConfig
	r.mu.RUnlock()

	if cfg == nil {
		return fmt.Errorf("email not configured")
	}

	subject := fmt.Sprintf("[%s] Security Alert: %s", alert.Severity, alert.Title)
	body := fmt.Sprintf(`Security Alert

Severity: %s
Title: %s
Source: %s
Time: %s

Description:
%s

This is an automated alert from the Asgard Security System.
`, alert.Severity, alert.Title, alert.Source, alert.Timestamp.Format(time.RFC1123), alert.Description)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		cfg.From, cfg.To[0], subject, body)

	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)
	
	var auth smtp.Auth
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.SMTPHost)
	}

	if err := smtp.SendMail(addr, auth, cfg.From, cfg.To, []byte(msg)); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Email alert sent for threat %s", alert.ID)
	return nil
}

// cleanupExpiredBlocks periodically removes expired IP blocks
func (r *Responder) cleanupExpiredBlocks() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		r.mu.Lock()
		now := time.Now()
		for ip, blocked := range r.blockedIPs {
			if now.After(blocked.ExpiresAt) {
				delete(r.blockedIPs, ip)
				log.Printf("Expired block removed for IP %s", ip)
			}
		}
		r.mu.Unlock()

		// Also cleanup rate limiter entries
		r.rateLimiter.mu.Lock()
		for ip, entry := range r.rateLimiter.limits {
			if len(entry.requests) == 0 && !entry.blocked {
				delete(r.rateLimiter.limits, ip)
			}
		}
		r.rateLimiter.mu.Unlock()
	}
}
