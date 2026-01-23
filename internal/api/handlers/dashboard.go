// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"net/http"

	"github.com/asgard/pandora/internal/services"
	"github.com/go-chi/chi/v5"
)

// DashboardHandler handles dashboard endpoints.
type DashboardHandler struct {
	dashboardService *services.DashboardService
}

// NewDashboardHandler creates a new dashboard handler.
func NewDashboardHandler(dashboardService *services.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboardService: dashboardService}
}

// GetStats handles GET /api/dashboard/stats
func (h *DashboardHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.dashboardService.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, stats)
}

// GetAlerts handles GET /api/alerts
func (h *DashboardHandler) GetAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := h.dashboardService.GetAlerts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, alerts)
}

// GetAlert handles GET /api/alerts/{id}
func (h *DashboardHandler) GetAlert(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	alert, err := h.dashboardService.GetAlert(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	jsonResponse(w, http.StatusOK, alert)
}

// GetMissions handles GET /api/missions
func (h *DashboardHandler) GetMissions(w http.ResponseWriter, r *http.Request) {
	missions, err := h.dashboardService.GetMissions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, missions)
}

// GetMission handles GET /api/missions/{id}
func (h *DashboardHandler) GetMission(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	mission, err := h.dashboardService.GetMission(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	jsonResponse(w, http.StatusOK, mission)
}

// GetSatellites handles GET /api/satellites
func (h *DashboardHandler) GetSatellites(w http.ResponseWriter, r *http.Request) {
	satellites, err := h.dashboardService.GetSatellites()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, satellites)
}

// GetSatellite handles GET /api/satellites/{id}
func (h *DashboardHandler) GetSatellite(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	satellite, err := h.dashboardService.GetSatellite(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	jsonResponse(w, http.StatusOK, satellite)
}

// GetHunoids handles GET /api/hunoids
func (h *DashboardHandler) GetHunoids(w http.ResponseWriter, r *http.Request) {
	hunoids, err := h.dashboardService.GetHunoids()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, hunoids)
}

// GetHunoid handles GET /api/hunoids/{id}
func (h *DashboardHandler) GetHunoid(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	hunoid, err := h.dashboardService.GetHunoid(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	jsonResponse(w, http.StatusOK, hunoid)
}
