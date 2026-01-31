package access

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// AccessHTTPHandler handles HTTP requests for access control
type AccessHTTPHandler struct {
	controller *AccessController
}

// NewAccessHTTPHandler creates a new HTTP handler
func NewAccessHTTPHandler(controller *AccessController) *AccessHTTPHandler {
	return &AccessHTTPHandler{
		controller: controller,
	}
}

// ServeHTTP handles HTTP requests
func (h *AccessHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Session-ID")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	path := r.URL.Path

	switch {
	// Authentication
	case path == "/api/v1/auth/login" && r.Method == "POST":
		h.handleLogin(w, r)
	case path == "/api/v1/auth/logout" && r.Method == "POST":
		h.handleLogout(w, r)
	case path == "/api/v1/auth/validate" && r.Method == "GET":
		h.handleValidateSession(w, r)
	case path == "/api/v1/auth/refresh" && r.Method == "POST":
		h.handleRefreshSession(w, r)

	// Users
	case path == "/api/v1/users" && r.Method == "GET":
		h.handleListUsers(w, r)
	case path == "/api/v1/users" && r.Method == "POST":
		h.handleCreateUser(w, r)
	case strings.HasPrefix(path, "/api/v1/users/") && r.Method == "GET":
		userID := path[14:]
		h.handleGetUser(w, r, userID)

	// Terminals
	case path == "/api/v1/terminals" && r.Method == "GET":
		h.handleListTerminals(w, r)
	case path == "/api/v1/terminals" && r.Method == "POST":
		h.handleRegisterTerminal(w, r)
	case strings.HasPrefix(path, "/api/v1/terminals/") && r.Method == "GET":
		terminalID := path[18:]
		h.handleGetTerminal(w, r, terminalID)
	case strings.HasPrefix(path, "/api/v1/terminals/") && r.Method == "PUT" && strings.HasSuffix(path, "/heartbeat"):
		parts := strings.Split(path, "/")
		if len(parts) >= 5 {
			terminalID := parts[4]
			h.handleTerminalHeartbeat(w, r, terminalID)
		}

	// Access Control
	case path == "/api/v1/access/check" && r.Method == "POST":
		h.handleCheckAccess(w, r)
	case path == "/api/v1/access/audit" && r.Method == "GET":
		h.handleGetAuditLogs(w, r)

	// Clearance Info
	case path == "/api/v1/clearance/levels" && r.Method == "GET":
		h.handleGetClearanceLevels(w, r)

	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	MFACode  string `json:"mfaCode,omitempty"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	SessionID     string         `json:"sessionId"`
	Token         string         `json:"token"`
	UserID        string         `json:"userId"`
	Username      string         `json:"username"`
	Clearance     ClearanceLevel `json:"clearance"`
	ClearanceName string         `json:"clearanceName"`
	ExpiresAt     time.Time      `json:"expiresAt"`
}

// handleLogin handles user login
func (h *AccessHTTPHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Authenticate user credentials
	user, err := h.controller.Authenticate(req.Username, req.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Create session
	ipAddress := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ipAddress = strings.Split(forwarded, ",")[0]
	}

	session, err := h.controller.CreateSession(user.ID, ipAddress, r.UserAgent())
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	response := LoginResponse{
		SessionID:     session.ID,
		Token:         session.Token,
		UserID:        user.ID,
		Username:      user.Username,
		Clearance:     user.Clearance,
		ClearanceName: ClearanceLevelName(user.Clearance),
		ExpiresAt:     session.ExpiresAt,
	}

	json.NewEncoder(w).Encode(response)
}

// handleLogout handles user logout
func (h *AccessHTTPHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	sessionID := r.Header.Get("X-Session-ID")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	if err := h.controller.RevokeSession(sessionID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleValidateSession validates a session
func (h *AccessHTTPHandler) handleValidateSession(w http.ResponseWriter, r *http.Request) {
	// Try session ID from header first
	sessionID := r.Header.Get("X-Session-ID")

	// Try Authorization header (Bearer token)
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		session, err := h.controller.ValidateSession(token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		json.NewEncoder(w).Encode(session)
		return
	}

	if sessionID == "" {
		http.Error(w, "Session ID or Bearer token required", http.StatusBadRequest)
		return
	}

	session, err := h.controller.ValidateSessionByID(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	json.NewEncoder(w).Encode(session)
}

// handleRefreshSession refreshes a session
func (h *AccessHTTPHandler) handleRefreshSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.Header.Get("X-Session-ID")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	if err := h.controller.RefreshSession(sessionID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	session, _ := h.controller.ValidateSessionByID(sessionID)
	json.NewEncoder(w).Encode(session)
}

// handleListUsers lists all users
func (h *AccessHTTPHandler) handleListUsers(w http.ResponseWriter, r *http.Request) {
	h.controller.mu.RLock()
	users := make([]*User, 0, len(h.controller.users))
	for _, user := range h.controller.users {
		users = append(users, user)
	}
	h.controller.mu.RUnlock()

	json.NewEncoder(w).Encode(users)
}

// handleCreateUser creates a new user
func (h *AccessHTTPHandler) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		User     User   `json:"user"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Password == "" {
		http.Error(w, "Password is required", http.StatusBadRequest)
		return
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to process password", http.StatusInternalServerError)
		return
	}
	req.User.PasswordHash = string(hashed)

	if err := h.controller.CreateUser(&req.User); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req.User)
}

// handleGetUser gets a specific user
func (h *AccessHTTPHandler) handleGetUser(w http.ResponseWriter, r *http.Request, userID string) {
	user, err := h.controller.GetUser(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(user)
}

// handleListTerminals lists all terminals
func (h *AccessHTTPHandler) handleListTerminals(w http.ResponseWriter, r *http.Request) {
	// Get clearance from header
	clearanceStr := r.Header.Get("X-Clearance")
	clearance := ClearancePublic
	if clearanceStr != "" {
		clearance = parseClearance(clearanceStr)
	}

	terminals := h.controller.GetTerminalsForClearance(clearance)
	json.NewEncoder(w).Encode(terminals)
}

// handleRegisterTerminal registers a new terminal
func (h *AccessHTTPHandler) handleRegisterTerminal(w http.ResponseWriter, r *http.Request) {
	var terminal Terminal
	if err := json.NewDecoder(r.Body).Decode(&terminal); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.controller.RegisterTerminal(&terminal); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(terminal)
}

// handleGetTerminal gets a specific terminal
func (h *AccessHTTPHandler) handleGetTerminal(w http.ResponseWriter, r *http.Request, terminalID string) {
	terminal, err := h.controller.GetTerminal(terminalID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(terminal)
}

// handleTerminalHeartbeat handles terminal heartbeat
func (h *AccessHTTPHandler) handleTerminalHeartbeat(w http.ResponseWriter, r *http.Request, terminalID string) {
	if err := h.controller.TerminalHeartbeat(terminalID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AccessCheckRequest represents an access check request
type AccessCheckRequest struct {
	SessionID    string     `json:"sessionId"`
	Resource     string     `json:"resource"`
	ResourceType string     `json:"resourceType"`
	AccessType   AccessType `json:"accessType"`
}

// handleCheckAccess checks access to a resource
func (h *AccessHTTPHandler) handleCheckAccess(w http.ResponseWriter, r *http.Request) {
	var req AccessCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Get session to determine clearance
	session, err := h.controller.ValidateSessionByID(req.SessionID)
	if err != nil {
		json.NewEncoder(w).Encode(&AccessDecision{
			Allowed:     false,
			Reason:      err.Error(),
			RequestedAt: time.Now(),
			DecidedAt:   time.Now(),
		})
		return
	}

	// Get required clearance for resource (would normally look this up)
	requiredClearance := ClearancePublic // Default

	accessReq := AccessRequest{
		UserID:       session.UserID,
		SessionID:    req.SessionID,
		Resource:     req.Resource,
		ResourceType: req.ResourceType,
		AccessType:   req.AccessType,
		Clearance:    requiredClearance,
		IPAddress:    r.RemoteAddr,
		Timestamp:    time.Now(),
	}

	decision := h.controller.CheckAccess(accessReq)
	json.NewEncoder(w).Encode(decision)
}

// handleGetAuditLogs gets recent audit logs
func (h *AccessHTTPHandler) handleGetAuditLogs(w http.ResponseWriter, r *http.Request) {
	limit := 100 // Default limit

	logs := h.controller.GetAuditLogs(limit)
	json.NewEncoder(w).Encode(logs)
}

// ClearanceLevelInfo represents clearance level information
type ClearanceLevelInfo struct {
	Level       ClearanceLevel `json:"level"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Color       string         `json:"color"`
}

// handleGetClearanceLevels returns all clearance levels
func (h *AccessHTTPHandler) handleGetClearanceLevels(w http.ResponseWriter, r *http.Request) {
	levels := []ClearanceLevelInfo{
		{Level: ClearancePublic, Name: "PUBLIC", Description: "Public access - basic information only", Color: "#22c55e"},
		{Level: ClearanceCivilian, Name: "CIVILIAN", Description: "Civilian operations - humanitarian missions", Color: "#3b82f6"},
		{Level: ClearanceMilitary, Name: "MILITARY", Description: "Military operations - tactical missions", Color: "#f59e0b"},
		{Level: ClearanceGov, Name: "GOVERNMENT", Description: "Government classified - restricted missions", Color: "#8b5cf6"},
		{Level: ClearanceSecret, Name: "SECRET", Description: "Secret operations - classified missions", Color: "#ef4444"},
		{Level: ClearanceUltra, Name: "ULTRA", Description: "Ultra clearance - highest classification", Color: "#ec4899"},
	}

	json.NewEncoder(w).Encode(levels)
}

func parseClearance(s string) ClearanceLevel {
	switch strings.ToUpper(s) {
	case "PUBLIC", "0":
		return ClearancePublic
	case "CIVILIAN", "1":
		return ClearanceCivilian
	case "MILITARY", "2":
		return ClearanceMilitary
	case "GOVERNMENT", "GOV", "3":
		return ClearanceGov
	case "SECRET", "4":
		return ClearanceSecret
	case "ULTRA", "5":
		return ClearanceUltra
	default:
		return ClearancePublic
	}
}

// InitializeDefaultData initializes default users and terminals
