// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"net/http"
	"os"
	"time"

	"github.com/asgard/pandora/internal/repositories"
	"github.com/asgard/pandora/internal/services"
	"github.com/go-chi/chi/v5"
)

// DashboardHandler handles dashboard endpoints.
type DashboardHandler struct {
	dashboardService *services.DashboardService
	trackingService  *services.SatelliteTrackingService
}

// NewDashboardHandler creates a new dashboard handler.
func NewDashboardHandler(dashboardService *services.DashboardService) *DashboardHandler {
	cfg := services.DefaultTrackingConfig()
	cfg.N2YOAPIKey = os.Getenv("N2YO_API_KEY")

	return &DashboardHandler{
		dashboardService: dashboardService,
		trackingService:  services.NewSatelliteTrackingService(cfg),
	}
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

// GetSatelliteTelemetry handles GET /api/telemetry/satellite/{satelliteId}
func (h *DashboardHandler) GetSatelliteTelemetry(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "satelliteId")
	telemetry, err := h.dashboardService.GetSatelliteTelemetry(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	timestamp := time.Now().UTC()
	if telemetry.LastTelemetry != nil {
		timestamp = telemetry.LastTelemetry.UTC()
	}

	var location *repositories.GeoLocation
	source := "unknown"
	if h.trackingService != nil {
		if satellite, satErr := h.dashboardService.GetSatellite(id); satErr == nil && satellite.NoradID.Valid {
			noradID := int(satellite.NoradID.Int32)
			position, posErr := h.trackingService.GetRealtimePosition(r.Context(), noradID)
			if posErr == nil {
				source = "realtime"
			} else {
				position, posErr = h.trackingService.GetPosition(r.Context(), noradID)
				if posErr == nil {
					source = "propagated"
				}
			}
			if posErr == nil && position != nil {
				location = &repositories.GeoLocation{
					Latitude:  position.Latitude,
					Longitude: position.Longitude,
					Altitude:  position.Altitude,
				}
			}
		}
	}

	jsonResponse(w, http.StatusOK, TelemetryResponse{
		EntityID:       id,
		EntityType:     "satellite",
		Timestamp:      timestamp.Format(time.RFC3339),
		BatteryPercent: telemetry.BatteryPercent,
		Status:         telemetry.Status,
		Location:       location,
		Source:         source,
		Metrics:        map[string]float64{},
	})
}

// GetHunoidTelemetry handles GET /api/telemetry/hunoid/{hunoidId}
func (h *DashboardHandler) GetHunoidTelemetry(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "hunoidId")
	telemetry, err := h.dashboardService.GetHunoidTelemetry(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	timestamp := time.Now().UTC()
	if telemetry.LastTelemetry != nil {
		timestamp = telemetry.LastTelemetry.UTC()
	}

	jsonResponse(w, http.StatusOK, TelemetryResponse{
		EntityID:       id,
		EntityType:     "hunoid",
		Timestamp:      timestamp.Format(time.RFC3339),
		BatteryPercent: telemetry.BatteryPercent,
		Status:         telemetry.Status,
		Location:       telemetry.Location,
		Source:         "database",
		Metrics:        map[string]float64{},
	})
}

// TelemetryResponse describes telemetry payloads for Hubs.
type TelemetryResponse struct {
	EntityID       string            `json:"entityId"`
	EntityType     string            `json:"entityType"`
	Timestamp      string            `json:"timestamp"`
	BatteryPercent float64           `json:"batteryPercent"`
	Status         string            `json:"status"`
	Location       *repositories.GeoLocation `json:"location,omitempty"`
	Source         string            `json:"source,omitempty"`
	Metrics        map[string]float64        `json:"metrics"`
}
