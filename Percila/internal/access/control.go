package access

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ClearanceLevel defines access tiers
type ClearanceLevel int

const (
	ClearancePublic    ClearanceLevel = 0  // Public access - basic info only
	ClearanceCivilian  ClearanceLevel = 1  // Civilian - humanitarian missions
	ClearanceMilitary  ClearanceLevel = 2  // Military - tactical missions
	ClearanceGov       ClearanceLevel = 3  // Government - classified missions
	ClearanceSecret    ClearanceLevel = 4  // Secret - top secret missions
	ClearanceUltra     ClearanceLevel = 5  // Ultra - highest classification
)

// AccessType defines what type of access is requested
type AccessType string

const (
	AccessView      AccessType = "view"      // View only access
	AccessCommand   AccessType = "command"   // Can issue commands
	AccessControl   AccessType = "control"   // Full control
	AccessAdmin     AccessType = "admin"     // Administrative access
)

// MissionClassification defines mission security level
type MissionClassification string

const (
	ClassPublic       MissionClassification = "PUBLIC"       // Open humanitarian
	ClassCivilian     MissionClassification = "CIVILIAN"     // Civilian operations
	ClassRestricted   MissionClassification = "RESTRICTED"   // Limited access
	ClassConfidential MissionClassification = "CONFIDENTIAL" // Military
	ClassSecret       MissionClassification = "SECRET"       // Classified
	ClassTopSecret    MissionClassification = "TOP_SECRET"   // Top secret
	ClassUltra        MissionClassification = "ULTRA"        // Above top secret
)

// User represents a system user
type User struct {
	ID             string         `json:"id"`
	Username       string         `json:"username"`
	Email          string         `json:"email"`
	PasswordHash   string         `json:"-"`
	Clearance      ClearanceLevel `json:"clearance"`
	AccessTypes    []AccessType   `json:"accessTypes"`
	Organization   string         `json:"organization"`
	Role           string         `json:"role"`
	Active         bool           `json:"active"`
	MFAEnabled     bool           `json:"mfaEnabled"`
	MFASecret      string         `json:"-"`
	LastLogin      time.Time      `json:"lastLogin"`
	CreatedAt      time.Time      `json:"createdAt"`
	ExpiresAt      *time.Time     `json:"expiresAt,omitempty"`
	AllowedIPs     []string       `json:"allowedIPs,omitempty"`
}

// Session represents an authenticated session
type Session struct {
	ID           string         `json:"id"`
	UserID       string         `json:"userId"`
	Token        string         `json:"-"`
	TokenHash    string         `json:"tokenHash"`
	Clearance    ClearanceLevel `json:"clearance"`
	AccessTypes  []AccessType   `json:"accessTypes"`
	IPAddress    string         `json:"ipAddress"`
	UserAgent    string         `json:"userAgent"`
	CreatedAt    time.Time      `json:"createdAt"`
	ExpiresAt    time.Time      `json:"expiresAt"`
	LastActivity time.Time      `json:"lastActivity"`
	Revoked      bool           `json:"revoked"`
}

// AccessRequest represents a request for access
type AccessRequest struct {
	UserID       string         `json:"userId"`
	SessionID    string         `json:"sessionId"`
	Resource     string         `json:"resource"`
	ResourceType string         `json:"resourceType"` // mission, feed, terminal, command
	AccessType   AccessType     `json:"accessType"`
	Clearance    ClearanceLevel `json:"clearance"`
	IPAddress    string         `json:"ipAddress"`
	Timestamp    time.Time      `json:"timestamp"`
}

// AccessDecision represents the result of an access check
type AccessDecision struct {
	Allowed      bool           `json:"allowed"`
	Reason       string         `json:"reason"`
	RequestedAt  time.Time      `json:"requestedAt"`
	DecidedAt    time.Time      `json:"decidedAt"`
	Clearance    ClearanceLevel `json:"clearance"`
	Requirements []string       `json:"requirements,omitempty"`
}

// AuditLog represents an access audit entry
type AuditLog struct {
	ID           string         `json:"id"`
	UserID       string         `json:"userId"`
	SessionID    string         `json:"sessionId"`
	Action       string         `json:"action"`
	Resource     string         `json:"resource"`
	ResourceType string         `json:"resourceType"`
	Allowed      bool           `json:"allowed"`
	Reason       string         `json:"reason"`
	IPAddress    string         `json:"ipAddress"`
	Timestamp    time.Time      `json:"timestamp"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Terminal represents an access terminal
type Terminal struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Type           TerminalType   `json:"type"`
	Location       string         `json:"location"`
	Clearance      ClearanceLevel `json:"clearance"`
	Status         string         `json:"status"` // online, offline, maintenance
	ActiveUsers    int            `json:"activeUsers"`
	LastHeartbeat  time.Time      `json:"lastHeartbeat"`
	IPAddress      string         `json:"ipAddress"`
	Capabilities   []string       `json:"capabilities"`
}

// TerminalType defines the type of access terminal
type TerminalType string

const (
	TerminalPublic   TerminalType = "public"   // Public kiosk
	TerminalCivilian TerminalType = "civilian" // Civilian workstation
	TerminalTactical TerminalType = "tactical" // Military command post
	TerminalCommand  TerminalType = "command"  // Command center
	TerminalSCIF     TerminalType = "scif"     // Sensitive Compartmented Information Facility
)

// AccessController manages access control
type AccessController struct {
	mu sync.RWMutex

	users     map[string]*User
	sessions  map[string]*Session
	terminals map[string]*Terminal
	auditLogs []AuditLog

	sessionTTL time.Duration
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewAccessController creates a new access controller
func NewAccessController() *AccessController {
	return &AccessController{
		users:      make(map[string]*User),
		sessions:   make(map[string]*Session),
		terminals:  make(map[string]*Terminal),
		auditLogs:  make([]AuditLog, 0),
		sessionTTL: 4 * time.Hour,
	}
}

// Start begins the access controller
func (ac *AccessController) Start(ctx context.Context) error {
	ac.ctx, ac.cancel = context.WithCancel(ctx)
	go ac.sessionCleanupLoop()
	return nil
}

// Stop halts the access controller
func (ac *AccessController) Stop() {
	if ac.cancel != nil {
		ac.cancel()
	}
}

// CreateUser creates a new user
func (ac *AccessController) CreateUser(user *User) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	if user.PasswordHash == "" {
		return fmt.Errorf("password hash is required")
	}
	user.Active = true
	user.CreatedAt = time.Now()

	ac.users[user.ID] = user
	return nil
}

// Authenticate verifies credentials and returns the user.
func (ac *AccessController) Authenticate(username, password string) (*User, error) {
	user, err := ac.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}
	if user.PasswordHash == "" {
		return nil, fmt.Errorf("password not configured")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}
	return user, nil
}

// GetUser retrieves a user by ID
func (ac *AccessController) GetUser(userID string) (*User, error) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	user, exists := ac.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found: %s", userID)
	}
	return user, nil
}

// GetUserByUsername retrieves a user by username
func (ac *AccessController) GetUserByUsername(username string) (*User, error) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	for _, user := range ac.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, fmt.Errorf("user not found: %s", username)
}

// CreateSession creates a new authenticated session
func (ac *AccessController) CreateSession(userID string, ipAddress string, userAgent string) (*Session, error) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	user, exists := ac.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	if !user.Active {
		return nil, fmt.Errorf("user account is inactive")
	}

	// Check IP whitelist if configured
	if len(user.AllowedIPs) > 0 {
		allowed := false
		for _, ip := range user.AllowedIPs {
			if ip == ipAddress || ip == "*" {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("IP address not authorized: %s", ipAddress)
		}
	}

	// Generate session token
	token := generateSecureToken()
	tokenHash := hashToken(token)

	session := &Session{
		ID:           uuid.New().String(),
		UserID:       userID,
		Token:        token,
		TokenHash:    tokenHash,
		Clearance:    user.Clearance,
		AccessTypes:  user.AccessTypes,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(ac.sessionTTL),
		LastActivity: time.Now(),
		Revoked:      false,
	}

	ac.sessions[session.ID] = session
	user.LastLogin = time.Now()

	return session, nil
}

// ValidateSession validates a session token
func (ac *AccessController) ValidateSession(token string) (*Session, error) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	tokenHash := hashToken(token)

	for _, session := range ac.sessions {
		if session.TokenHash == tokenHash {
			if session.Revoked {
				return nil, fmt.Errorf("session has been revoked")
			}
			if time.Now().After(session.ExpiresAt) {
				return nil, fmt.Errorf("session has expired")
			}
			return session, nil
		}
	}

	return nil, fmt.Errorf("invalid session token")
}

// ValidateSessionByID validates a session by ID
func (ac *AccessController) ValidateSessionByID(sessionID string) (*Session, error) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	session, exists := ac.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	if session.Revoked {
		return nil, fmt.Errorf("session has been revoked")
	}
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session has expired")
	}

	return session, nil
}

// RefreshSession extends session expiry
func (ac *AccessController) RefreshSession(sessionID string) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	session, exists := ac.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.LastActivity = time.Now()
	session.ExpiresAt = time.Now().Add(ac.sessionTTL)
	return nil
}

// RevokeSession revokes a session
func (ac *AccessController) RevokeSession(sessionID string) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	session, exists := ac.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	session.Revoked = true
	return nil
}

// CheckAccess verifies access to a resource
func (ac *AccessController) CheckAccess(request AccessRequest) *AccessDecision {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	decision := &AccessDecision{
		Allowed:     false,
		RequestedAt: request.Timestamp,
		DecidedAt:   time.Now(),
		Clearance:   request.Clearance,
	}

	// Validate session
	session, exists := ac.sessions[request.SessionID]
	if !exists {
		decision.Reason = "invalid session"
		ac.logAudit(request, decision)
		return decision
	}

	if session.Revoked {
		decision.Reason = "session revoked"
		ac.logAudit(request, decision)
		return decision
	}

	if time.Now().After(session.ExpiresAt) {
		decision.Reason = "session expired"
		ac.logAudit(request, decision)
		return decision
	}

	// Check clearance level
	if session.Clearance < request.Clearance {
		decision.Reason = fmt.Sprintf("insufficient clearance: requires %s, have %s",
			ClearanceLevelName(request.Clearance),
			ClearanceLevelName(session.Clearance))
		decision.Requirements = []string{
			fmt.Sprintf("Clearance level %s required", ClearanceLevelName(request.Clearance)),
		}
		ac.logAudit(request, decision)
		return decision
	}

	// Check access type
	hasAccessType := false
	for _, at := range session.AccessTypes {
		if at == request.AccessType || at == AccessAdmin {
			hasAccessType = true
			break
		}
	}

	if !hasAccessType {
		decision.Reason = fmt.Sprintf("access type not permitted: %s", request.AccessType)
		decision.Requirements = []string{
			fmt.Sprintf("Access type '%s' required", request.AccessType),
		}
		ac.logAudit(request, decision)
		return decision
	}

	// Access granted
	decision.Allowed = true
	decision.Reason = "access granted"
	ac.logAudit(request, decision)
	return decision
}

// RegisterTerminal registers an access terminal
func (ac *AccessController) RegisterTerminal(terminal *Terminal) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	if terminal.ID == "" {
		terminal.ID = uuid.New().String()
	}
	terminal.Status = "online"
	terminal.LastHeartbeat = time.Now()

	ac.terminals[terminal.ID] = terminal
	return nil
}

// GetTerminal retrieves a terminal by ID
func (ac *AccessController) GetTerminal(terminalID string) (*Terminal, error) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	terminal, exists := ac.terminals[terminalID]
	if !exists {
		return nil, fmt.Errorf("terminal not found: %s", terminalID)
	}
	return terminal, nil
}

// GetTerminalsByType returns terminals of a specific type
func (ac *AccessController) GetTerminalsByType(terminalType TerminalType) []*Terminal {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	result := make([]*Terminal, 0)
	for _, terminal := range ac.terminals {
		if terminal.Type == terminalType && terminal.Status == "online" {
			result = append(result, terminal)
		}
	}
	return result
}

// GetTerminalsForClearance returns terminals accessible at given clearance
func (ac *AccessController) GetTerminalsForClearance(clearance ClearanceLevel) []*Terminal {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	result := make([]*Terminal, 0)
	for _, terminal := range ac.terminals {
		if terminal.Clearance <= clearance && terminal.Status == "online" {
			result = append(result, terminal)
		}
	}
	return result
}

// TerminalHeartbeat updates terminal heartbeat
func (ac *AccessController) TerminalHeartbeat(terminalID string) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	terminal, exists := ac.terminals[terminalID]
	if !exists {
		return fmt.Errorf("terminal not found: %s", terminalID)
	}

	terminal.LastHeartbeat = time.Now()
	terminal.Status = "online"
	return nil
}

// GetAuditLogs returns recent audit logs
func (ac *AccessController) GetAuditLogs(limit int) []AuditLog {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	if limit <= 0 || limit > len(ac.auditLogs) {
		limit = len(ac.auditLogs)
	}

	start := len(ac.auditLogs) - limit
	if start < 0 {
		start = 0
	}

	return ac.auditLogs[start:]
}

// logAudit logs an access attempt
func (ac *AccessController) logAudit(request AccessRequest, decision *AccessDecision) {
	log := AuditLog{
		ID:           uuid.New().String(),
		UserID:       request.UserID,
		SessionID:    request.SessionID,
		Action:       string(request.AccessType),
		Resource:     request.Resource,
		ResourceType: request.ResourceType,
		Allowed:      decision.Allowed,
		Reason:       decision.Reason,
		IPAddress:    request.IPAddress,
		Timestamp:    time.Now(),
	}

	ac.auditLogs = append(ac.auditLogs, log)

	// Keep only last 10000 logs
	if len(ac.auditLogs) > 10000 {
		ac.auditLogs = ac.auditLogs[1000:]
	}
}

// sessionCleanupLoop removes expired sessions
func (ac *AccessController) sessionCleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ac.ctx.Done():
			return
		case <-ticker.C:
			ac.cleanupExpiredSessions()
			ac.checkTerminalHealth()
		}
	}
}

// cleanupExpiredSessions removes expired sessions
func (ac *AccessController) cleanupExpiredSessions() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	for id, session := range ac.sessions {
		if time.Now().After(session.ExpiresAt) || session.Revoked {
			delete(ac.sessions, id)
		}
	}
}

// checkTerminalHealth marks offline terminals
func (ac *AccessController) checkTerminalHealth() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	offlineThreshold := 2 * time.Minute

	for _, terminal := range ac.terminals {
		if time.Since(terminal.LastHeartbeat) > offlineThreshold {
			terminal.Status = "offline"
		}
	}
}

// Helper functions

func generateSecureToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// ClearanceLevelName returns human-readable name for clearance
func ClearanceLevelName(level ClearanceLevel) string {
	switch level {
	case ClearancePublic:
		return "PUBLIC"
	case ClearanceCivilian:
		return "CIVILIAN"
	case ClearanceMilitary:
		return "MILITARY"
	case ClearanceGov:
		return "GOVERNMENT"
	case ClearanceSecret:
		return "SECRET"
	case ClearanceUltra:
		return "ULTRA"
	default:
		return "UNKNOWN"
	}
}

// ClassificationToClearance converts mission classification to required clearance
func ClassificationToClearance(class MissionClassification) ClearanceLevel {
	switch class {
	case ClassPublic:
		return ClearancePublic
	case ClassCivilian:
		return ClearanceCivilian
	case ClassRestricted:
		return ClearanceMilitary
	case ClassConfidential:
		return ClearanceMilitary
	case ClassSecret:
		return ClearanceSecret
	case ClassTopSecret:
		return ClearanceSecret
	case ClassUltra:
		return ClearanceUltra
	default:
		return ClearancePublic
	}
}

// MissionAccessRules defines access rules for a mission
type MissionAccessRules struct {
	MissionID        string                `json:"missionId"`
	Classification   MissionClassification `json:"classification"`
	RequiredClearance ClearanceLevel       `json:"requiredClearance"`
	AllowedViewers   []string              `json:"allowedViewers,omitempty"`  // User IDs
	AllowedCommanders []string             `json:"allowedCommanders,omitempty"` // User IDs
	Compartments     []string              `json:"compartments,omitempty"` // Special access programs
	CreatedAt        time.Time             `json:"createdAt"`
}

