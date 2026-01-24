// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/asgard/pandora/internal/repositories"
	"github.com/asgard/pandora/internal/services"
	"github.com/go-chi/chi/v5"
)

// AuditHandler handles audit and ethics endpoints.
type AuditHandler struct {
	auditService *services.AuditService
}

// NewAuditHandler creates a new audit handler.
func NewAuditHandler(auditService *services.AuditService) *AuditHandler {
	return &AuditHandler{auditService: auditService}
}

// GetAuditLogs handles GET /api/audit/logs
// Query params: component, action, user_id, since, until, limit, offset
func (h *AuditHandler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	filters := repositories.AuditLogFilters{
		Component: r.URL.Query().Get("component"),
		Action:    r.URL.Query().Get("action"),
		UserID:    r.URL.Query().Get("user_id"),
	}

	// Parse time filters
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		since, err := time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "Invalid 'since' date format. Use RFC3339", "INVALID_DATE")
			return
		}
		filters.Since = since
	}

	if untilStr := r.URL.Query().Get("until"); untilStr != "" {
		until, err := time.Parse(time.RFC3339, untilStr)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "Invalid 'until' date format. Use RFC3339", "INVALID_DATE")
			return
		}
		filters.Until = until
	}

	// Parse pagination
	limit, offset := parsePaginationParams(r)
	filters.Limit = limit
	filters.Offset = offset

	logs, err := h.auditService.GetAuditLogs(ctx, filters)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error(), "QUERY_ERROR")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"logs":   logs,
		"count":  len(logs),
		"limit":  limit,
		"offset": offset,
	})
}

// GetAuditLog handles GET /api/audit/logs/:id
func (h *AuditHandler) GetAuditLog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "Invalid log ID", "INVALID_ID")
		return
	}

	log, err := h.auditService.GetAuditLogByID(ctx, id)
	if err != nil {
		jsonError(w, http.StatusNotFound, "Audit log not found", "NOT_FOUND")
		return
	}

	jsonResponse(w, http.StatusOK, log)
}

// GetAuditLogsByComponent handles GET /api/audit/logs/component/:component
func (h *AuditHandler) GetAuditLogsByComponent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	component := chi.URLParam(r, "component")

	since := time.Now().AddDate(0, 0, -7) // Default to last 7 days
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		parsedSince, err := time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "Invalid 'since' date format. Use RFC3339", "INVALID_DATE")
			return
		}
		since = parsedSince
	}

	logs, err := h.auditService.GetAuditLogsByComponent(ctx, component, since)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error(), "QUERY_ERROR")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"logs":      logs,
		"count":     len(logs),
		"component": component,
		"since":     since,
	})
}

// GetAuditLogsByUser handles GET /api/audit/logs/user/:userId
func (h *AuditHandler) GetAuditLogsByUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := chi.URLParam(r, "userId")

	limit, _ := parsePaginationParams(r)

	logs, err := h.auditService.GetAuditLogsByUser(ctx, userID, limit)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error(), "QUERY_ERROR")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"logs":    logs,
		"count":   len(logs),
		"user_id": userID,
	})
}

// GetAuditStats handles GET /api/audit/stats
func (h *AuditHandler) GetAuditStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	since := time.Now().AddDate(0, 0, -30) // Default to last 30 days
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		parsedSince, err := time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "Invalid 'since' date format. Use RFC3339", "INVALID_DATE")
			return
		}
		since = parsedSince
	}

	stats, err := h.auditService.GetAuditStats(ctx, since)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error(), "QUERY_ERROR")
		return
	}

	jsonResponse(w, http.StatusOK, stats)
}

// GetEthicalDecisions handles GET /api/ethics/decisions
// Query params: hunoid_id, mission_id, decision_type, limit, offset
func (h *AuditHandler) GetEthicalDecisions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit, _ := parsePaginationParams(r)

	hunoidID := r.URL.Query().Get("hunoid_id")
	missionID := r.URL.Query().Get("mission_id")
	decisionType := r.URL.Query().Get("decision_type")

	var decisions interface{}
	var err error

	switch {
	case hunoidID != "":
		decisions, err = h.auditService.GetEthicalDecisionsByHunoid(ctx, hunoidID, limit)
	case missionID != "":
		decisions, err = h.auditService.GetEthicalDecisionsByMission(ctx, missionID)
	case decisionType != "":
		decisions, err = h.auditService.GetEthicalDecisionsByType(ctx, decisionType, limit)
	default:
		decisions, err = h.auditService.GetEthicalDecisions(ctx, limit)
	}

	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error(), "QUERY_ERROR")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"decisions": decisions,
		"limit":     limit,
	})
}

// GetEthicalDecision handles GET /api/ethics/decisions/:id
func (h *AuditHandler) GetEthicalDecision(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	decision, err := h.auditService.GetEthicalDecisionByID(ctx, id)
	if err != nil {
		jsonError(w, http.StatusNotFound, "Ethical decision not found", "NOT_FOUND")
		return
	}

	jsonResponse(w, http.StatusOK, decision)
}

// GetEthicalDecisionsByHunoid handles GET /api/ethics/decisions/hunoid/:hunoidId
func (h *AuditHandler) GetEthicalDecisionsByHunoid(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hunoidID := chi.URLParam(r, "hunoidId")

	limit, _ := parsePaginationParams(r)

	decisions, err := h.auditService.GetEthicalDecisionsByHunoid(ctx, hunoidID, limit)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error(), "QUERY_ERROR")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"decisions": decisions,
		"count":     len(decisions),
		"hunoid_id": hunoidID,
	})
}

// GetEthicalDecisionsByMission handles GET /api/ethics/decisions/mission/:missionId
func (h *AuditHandler) GetEthicalDecisionsByMission(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	missionID := chi.URLParam(r, "missionId")

	decisions, err := h.auditService.GetEthicalDecisionsByMission(ctx, missionID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error(), "QUERY_ERROR")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"decisions":  decisions,
		"count":      len(decisions),
		"mission_id": missionID,
	})
}

// GetEthicsStats handles GET /api/ethics/stats
func (h *AuditHandler) GetEthicsStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := h.auditService.GetEthicsStats(ctx)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error(), "QUERY_ERROR")
		return
	}

	jsonResponse(w, http.StatusOK, stats)
}
