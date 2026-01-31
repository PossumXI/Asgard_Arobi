// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/asgard/pandora/internal/controlplane"
	"github.com/go-chi/chi/v5"
)

// ControlPlaneHandler handles control plane API endpoints.
type ControlPlaneHandler struct {
	controlPlane *controlplane.UnifiedControlPlane
}

// NewControlPlaneHandler creates a new control plane handler.
func NewControlPlaneHandler(cp *controlplane.UnifiedControlPlane) *ControlPlaneHandler {
	return &ControlPlaneHandler{
		controlPlane: cp,
	}
}

// ControlPlaneStatusResponse represents the overall system status.
type ControlPlaneStatusResponse struct {
	Health      *controlplane.HealthStatus            `json:"health"`
	Systems     map[string]*controlplane.SystemStatus `json:"systems"`
	Metrics     *controlplane.ControlPlaneMetrics     `json:"metrics"`
	Coordinator *CoordinatorStatusResponse            `json:"coordinator"`
	Timestamp   time.Time                             `json:"timestamp"`
}

// CoordinatorStatusResponse contains coordinator status information.
type CoordinatorStatusResponse struct {
	PoliciesCount   int                              `json:"policies_count"`
	EnabledPolicies int                              `json:"enabled_policies"`
	ActiveResponses int                              `json:"active_responses"`
	Metrics         *controlplane.CoordinatorMetrics `json:"metrics"`
}

// EventsResponse represents paginated events.
type EventsResponse struct {
	Events     []controlplane.CrossDomainEvent `json:"events"`
	TotalCount int                             `json:"total_count"`
	Limit      int                             `json:"limit"`
	Offset     int                             `json:"offset"`
}

// CommandRequest represents an incoming control command.
type CommandRequest struct {
	TargetDomain string                 `json:"target_domain"`
	TargetSystem string                 `json:"target_system,omitempty"`
	CommandType  string                 `json:"command_type"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	Priority     int                    `json:"priority,omitempty"`
}

// CommandResponse represents the result of a command execution.
type CommandResponse struct {
	CommandID  string                 `json:"command_id"`
	Success    bool                   `json:"success"`
	Error      string                 `json:"error,omitempty"`
	Result     map[string]interface{} `json:"result,omitempty"`
	ExecutedAt time.Time              `json:"executed_at"`
	DurationMs int64                  `json:"duration_ms"`
}

// PolicyResponse represents a coordination policy.
type PolicyResponse struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Priority      int                    `json:"priority"`
	Enabled       bool                   `json:"enabled"`
	TriggerType   string                 `json:"trigger_type,omitempty"`
	Actions       []PolicyActionResponse `json:"actions"`
	CooldownMs    int64                  `json:"cooldown_ms"`
	LastTriggered *time.Time             `json:"last_triggered,omitempty"`
}

// PolicyActionResponse represents a policy action.
type PolicyActionResponse struct {
	TargetDomain string                 `json:"target_domain"`
	CommandType  string                 `json:"command_type"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	Async        bool                   `json:"async"`
}

// GetStatus handles GET /api/controlplane/status
func (h *ControlPlaneHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	if h.controlPlane == nil {
		jsonError(w, http.StatusServiceUnavailable, "Control plane not initialized", "CONTROL_PLANE_UNAVAILABLE")
		return
	}

	health := h.controlPlane.GetHealth()
	systems := h.controlPlane.GetStatus()
	metrics := h.controlPlane.GetMetrics()

	// Get coordinator status
	coordinator := h.controlPlane.GetCoordinator()
	var coordStatus *CoordinatorStatusResponse
	if coordinator != nil {
		policies := coordinator.GetPolicies()
		enabledCount := 0
		for _, p := range policies {
			if p.Enabled {
				enabledCount++
			}
		}

		coordStatus = &CoordinatorStatusResponse{
			PoliciesCount:   len(policies),
			EnabledPolicies: enabledCount,
			ActiveResponses: len(coordinator.GetActiveResponses()),
			Metrics:         coordinator.GetMetrics(),
		}
	}

	response := ControlPlaneStatusResponse{
		Health:      health,
		Systems:     systems,
		Metrics:     metrics,
		Coordinator: coordStatus,
		Timestamp:   time.Now().UTC(),
	}

	jsonResponse(w, http.StatusOK, response)
}

// GetHealth handles GET /api/controlplane/health
func (h *ControlPlaneHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	if h.controlPlane == nil {
		jsonError(w, http.StatusServiceUnavailable, "Control plane not initialized", "CONTROL_PLANE_UNAVAILABLE")
		return
	}

	health := h.controlPlane.GetHealth()

	status := http.StatusOK
	if health.Status == "critical" {
		status = http.StatusServiceUnavailable
	} else if health.Status == "degraded" {
		status = http.StatusPartialContent
	}

	jsonResponse(w, status, health)
}

// GetEvents handles GET /api/controlplane/events
func (h *ControlPlaneHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	if h.controlPlane == nil {
		jsonError(w, http.StatusServiceUnavailable, "Control plane not initialized", "CONTROL_PLANE_UNAVAILABLE")
		return
	}

	// Parse query parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	// Get events
	events := h.controlPlane.GetRecentEvents(limit + offset)

	// Apply offset
	if offset >= len(events) {
		events = []controlplane.CrossDomainEvent{}
	} else {
		endIdx := offset + limit
		if endIdx > len(events) {
			endIdx = len(events)
		}
		events = events[offset:endIdx]
	}

	// Filter by domain if specified
	domainFilter := r.URL.Query().Get("domain")
	if domainFilter != "" {
		filtered := make([]controlplane.CrossDomainEvent, 0)
		for _, e := range events {
			if string(e.Domain) == domainFilter {
				filtered = append(filtered, e)
			}
		}
		events = filtered
	}

	// Filter by severity if specified
	severityFilter := r.URL.Query().Get("severity")
	if severityFilter != "" {
		filtered := make([]controlplane.CrossDomainEvent, 0)
		for _, e := range events {
			if string(e.Severity) == severityFilter {
				filtered = append(filtered, e)
			}
		}
		events = filtered
	}

	// Filter by type if specified
	typeFilter := r.URL.Query().Get("type")
	if typeFilter != "" {
		filtered := make([]controlplane.CrossDomainEvent, 0)
		for _, e := range events {
			if string(e.Type) == typeFilter {
				filtered = append(filtered, e)
			}
		}
		events = filtered
	}

	response := EventsResponse{
		Events:     events,
		TotalCount: len(events),
		Limit:      limit,
		Offset:     offset,
	}

	jsonResponse(w, http.StatusOK, response)
}

// GetEvent handles GET /api/controlplane/events/{id}
func (h *ControlPlaneHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	if h.controlPlane == nil {
		jsonError(w, http.StatusServiceUnavailable, "Control plane not initialized", "CONTROL_PLANE_UNAVAILABLE")
		return
	}

	eventID := chi.URLParam(r, "id")
	if eventID == "" {
		jsonError(w, http.StatusBadRequest, "Missing event ID", "INVALID_REQUEST")
		return
	}

	// Search for the event
	events := h.controlPlane.GetRecentEvents(10000)
	for _, event := range events {
		if event.ID.String() == eventID {
			jsonResponse(w, http.StatusOK, event)
			return
		}
	}

	jsonError(w, http.StatusNotFound, "Event not found", "NOT_FOUND")
}

// PostCommand handles POST /api/controlplane/command
func (h *ControlPlaneHandler) PostCommand(w http.ResponseWriter, r *http.Request) {
	if h.controlPlane == nil {
		jsonError(w, http.StatusServiceUnavailable, "Control plane not initialized", "CONTROL_PLANE_UNAVAILABLE")
		return
	}

	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request body", "INVALID_JSON")
		return
	}

	// Validate required fields
	if req.TargetDomain == "" {
		jsonError(w, http.StatusBadRequest, "target_domain is required", "INVALID_REQUEST")
		return
	}
	if req.CommandType == "" {
		jsonError(w, http.StatusBadRequest, "command_type is required", "INVALID_REQUEST")
		return
	}

	// Convert domain string to EventDomain
	domain := controlplane.EventDomain(req.TargetDomain)

	// Set default priority
	if req.Priority <= 0 {
		req.Priority = 5
	}

	// Create command
	cmd := controlplane.NewControlCommand(
		getUserIDFromContext(r),
		domain,
		req.CommandType,
		req.Parameters,
		req.Priority,
	)
	cmd.TargetSystem = req.TargetSystem

	// Execute command
	result, err := h.controlPlane.IssueCommand(cmd)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error(), "COMMAND_FAILED")
		return
	}

	response := CommandResponse{
		CommandID:  cmd.ID.String(),
		Success:    result.Success,
		Error:      result.Error,
		Result:     result.Result,
		ExecutedAt: result.ExecutedAt,
		DurationMs: result.Duration.Milliseconds(),
	}

	jsonResponse(w, http.StatusOK, response)
}

// GetPolicies handles GET /api/controlplane/policies
func (h *ControlPlaneHandler) GetPolicies(w http.ResponseWriter, r *http.Request) {
	if h.controlPlane == nil {
		jsonError(w, http.StatusServiceUnavailable, "Control plane not initialized", "CONTROL_PLANE_UNAVAILABLE")
		return
	}

	coordinator := h.controlPlane.GetCoordinator()
	if coordinator == nil {
		jsonError(w, http.StatusServiceUnavailable, "Coordinator not available", "COORDINATOR_UNAVAILABLE")
		return
	}

	policies := coordinator.GetPolicies()
	response := make([]PolicyResponse, len(policies))

	for i, p := range policies {
		actions := make([]PolicyActionResponse, len(p.Actions))
		for j, a := range p.Actions {
			actions[j] = PolicyActionResponse{
				TargetDomain: string(a.TargetDomain),
				CommandType:  a.CommandType,
				Parameters:   a.Parameters,
				Async:        a.Async,
			}
		}

		var lastTriggered *time.Time
		if !p.LastTriggered.IsZero() {
			lastTriggered = &p.LastTriggered
		}

		response[i] = PolicyResponse{
			ID:            p.ID,
			Name:          p.Name,
			Description:   p.Description,
			Priority:      p.Priority,
			Enabled:       p.Enabled,
			TriggerType:   string(p.TriggerType),
			Actions:       actions,
			CooldownMs:    p.Cooldown.Milliseconds(),
			LastTriggered: lastTriggered,
		}
	}

	jsonResponse(w, http.StatusOK, response)
}

// PatchPolicy handles PATCH /api/controlplane/policies/{id}
func (h *ControlPlaneHandler) PatchPolicy(w http.ResponseWriter, r *http.Request) {
	if h.controlPlane == nil {
		jsonError(w, http.StatusServiceUnavailable, "Control plane not initialized", "CONTROL_PLANE_UNAVAILABLE")
		return
	}

	coordinator := h.controlPlane.GetCoordinator()
	if coordinator == nil {
		jsonError(w, http.StatusServiceUnavailable, "Coordinator not available", "COORDINATOR_UNAVAILABLE")
		return
	}

	policyID := chi.URLParam(r, "id")
	if policyID == "" {
		jsonError(w, http.StatusBadRequest, "Missing policy ID", "INVALID_REQUEST")
		return
	}

	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid request body", "INVALID_JSON")
		return
	}

	if req.Enabled != nil {
		if *req.Enabled {
			coordinator.EnablePolicy(policyID)
		} else {
			coordinator.DisablePolicy(policyID)
		}
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"id":      policyID,
		"updated": true,
	})
}

// GetResponses handles GET /api/controlplane/responses
func (h *ControlPlaneHandler) GetResponses(w http.ResponseWriter, r *http.Request) {
	if h.controlPlane == nil {
		jsonError(w, http.StatusServiceUnavailable, "Control plane not initialized", "CONTROL_PLANE_UNAVAILABLE")
		return
	}

	coordinator := h.controlPlane.GetCoordinator()
	if coordinator == nil {
		jsonError(w, http.StatusServiceUnavailable, "Coordinator not available", "COORDINATOR_UNAVAILABLE")
		return
	}

	responses := coordinator.GetActiveResponses()
	jsonResponse(w, http.StatusOK, responses)
}

// GetSystems handles GET /api/controlplane/systems
func (h *ControlPlaneHandler) GetSystems(w http.ResponseWriter, r *http.Request) {
	if h.controlPlane == nil {
		jsonError(w, http.StatusServiceUnavailable, "Control plane not initialized", "CONTROL_PLANE_UNAVAILABLE")
		return
	}

	systems := h.controlPlane.GetStatus()
	jsonResponse(w, http.StatusOK, systems)
}

// GetSystem handles GET /api/controlplane/systems/{id}
func (h *ControlPlaneHandler) GetSystem(w http.ResponseWriter, r *http.Request) {
	if h.controlPlane == nil {
		jsonError(w, http.StatusServiceUnavailable, "Control plane not initialized", "CONTROL_PLANE_UNAVAILABLE")
		return
	}

	systemID := chi.URLParam(r, "id")
	if systemID == "" {
		jsonError(w, http.StatusBadRequest, "Missing system ID", "INVALID_REQUEST")
		return
	}

	systems := h.controlPlane.GetStatus()
	if system, exists := systems[systemID]; exists {
		jsonResponse(w, http.StatusOK, system)
		return
	}

	jsonError(w, http.StatusNotFound, "System not found", "NOT_FOUND")
}

// GetMetrics handles GET /api/controlplane/metrics
func (h *ControlPlaneHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	if h.controlPlane == nil {
		jsonError(w, http.StatusServiceUnavailable, "Control plane not initialized", "CONTROL_PLANE_UNAVAILABLE")
		return
	}

	metrics := h.controlPlane.GetMetrics()

	coordinator := h.controlPlane.GetCoordinator()
	var coordMetrics *controlplane.CoordinatorMetrics
	if coordinator != nil {
		coordMetrics = coordinator.GetMetrics()
	}

	response := map[string]interface{}{
		"controlplane": metrics,
		"coordinator":  coordMetrics,
		"timestamp":    time.Now().UTC(),
	}

	jsonResponse(w, http.StatusOK, response)
}
