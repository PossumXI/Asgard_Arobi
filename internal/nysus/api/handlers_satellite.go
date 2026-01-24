package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/asgard/pandora/internal/platform/satellite"
	"github.com/asgard/pandora/internal/services"
)

// SatelliteHandlers provides HTTP handlers for satellite tracking.
type SatelliteHandlers struct {
	trackingService *services.SatelliteTrackingService
}

// NewSatelliteHandlers creates satellite tracking handlers.
func NewSatelliteHandlers(apiKey string) *SatelliteHandlers {
	cfg := services.DefaultTrackingConfig()
	cfg.N2YOAPIKey = apiKey
	
	service := services.NewSatelliteTrackingService(cfg)
	
	return &SatelliteHandlers{
		trackingService: service,
	}
}

// RegisterRoutes registers satellite tracking API routes.
func (h *SatelliteHandlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/satellites/position", h.handleGetPosition)
	mux.HandleFunc("/api/satellites/realtime", h.handleGetRealtimePosition)
	mux.HandleFunc("/api/satellites/fleet", h.handleGetFleetPositions)
	mux.HandleFunc("/api/satellites/groundtrack", h.handleGetGroundTrack)
	mux.HandleFunc("/api/satellites/contacts", h.handleGetContactWindows)
	mux.HandleFunc("/api/satellites/above", h.handleGetSatellitesAbove)
	mux.HandleFunc("/api/satellites/tle", h.handleGetTLE)
}

// handleGetPosition returns propagated position for a satellite.
func (h *SatelliteHandlers) handleGetPosition(w http.ResponseWriter, r *http.Request) {
	noradID, err := strconv.Atoi(r.URL.Query().Get("norad_id"))
	if err != nil {
		noradID = satellite.NoradISS // Default to ISS
	}

	pos, err := h.trackingService.GetPosition(r.Context(), noradID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, pos)
}

// handleGetRealtimePosition returns real-time position from N2YO API.
func (h *SatelliteHandlers) handleGetRealtimePosition(w http.ResponseWriter, r *http.Request) {
	noradID, err := strconv.Atoi(r.URL.Query().Get("norad_id"))
	if err != nil {
		noradID = satellite.NoradISS
	}

	pos, err := h.trackingService.GetRealtimePosition(r.Context(), noradID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, pos)
}

// handleGetFleetPositions returns positions of all tracked satellites.
func (h *SatelliteHandlers) handleGetFleetPositions(w http.ResponseWriter, r *http.Request) {
	positions := h.trackingService.GetFleetPositions(r.Context())

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"count":      len(positions),
		"satellites": positions,
		"timestamp":  time.Now().UTC(),
	})
}

// handleGetGroundTrack returns ground track for a satellite.
func (h *SatelliteHandlers) handleGetGroundTrack(w http.ResponseWriter, r *http.Request) {
	noradID, _ := strconv.Atoi(r.URL.Query().Get("norad_id"))
	if noradID == 0 {
		noradID = satellite.NoradISS
	}

	durationMins, _ := strconv.Atoi(r.URL.Query().Get("duration"))
	if durationMins == 0 {
		durationMins = 90 // One orbit
	}

	track, err := h.trackingService.GetGroundTrack(r.Context(), noradID, time.Duration(durationMins)*time.Minute)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, track)
}

// handleGetContactWindows returns upcoming contact windows.
func (h *SatelliteHandlers) handleGetContactWindows(w http.ResponseWriter, r *http.Request) {
	noradID, _ := strconv.Atoi(r.URL.Query().Get("norad_id"))
	if noradID == 0 {
		noradID = satellite.NoradISS
	}

	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days == 0 {
		days = 5
	}

	windows, err := h.trackingService.GetContactWindows(r.Context(), noradID, days)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"norad_id": noradID,
		"days":     days,
		"windows":  windows,
	})
}

// handleGetSatellitesAbove returns satellites currently above observer.
func (h *SatelliteHandlers) handleGetSatellitesAbove(w http.ResponseWriter, r *http.Request) {
	radius, _ := strconv.Atoi(r.URL.Query().Get("radius"))
	if radius == 0 {
		radius = 70 // degrees from zenith
	}

	categoryID, _ := strconv.Atoi(r.URL.Query().Get("category"))
	// 0 = all, 52 = Starlink, 1 = brightest, etc.

	satellites, err := h.trackingService.SatellitesAbove(r.Context(), radius, categoryID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]interface{}{
		"count":      len(satellites),
		"radius":     radius,
		"category":   categoryID,
		"satellites": satellites,
		"timestamp":  time.Now().UTC(),
	})
}

// handleGetTLE returns TLE data for a satellite.
func (h *SatelliteHandlers) handleGetTLE(w http.ResponseWriter, r *http.Request) {
	noradID, _ := strconv.Atoi(r.URL.Query().Get("norad_id"))
	if noradID == 0 {
		noradID = satellite.NoradISS
	}

	cfg := satellite.DefaultConfig()
	client := satellite.NewClient(cfg)

	tle, err := client.GetTLE(r.Context(), noradID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, tle)
}

func (h *SatelliteHandlers) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *SatelliteHandlers) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}
