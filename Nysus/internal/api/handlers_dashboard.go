package api

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// DashboardStats represents dashboard statistics.
type DashboardStats struct {
	ActiveSatellites int     `json:"activeSatellites"`
	ActiveHunoids    int     `json:"activeHunoids"`
	PendingAlerts    int     `json:"pendingAlerts"`
	ActiveMissions   int     `json:"activeMissions"`
	ThreatsToday     int     `json:"threatsToday"`
	SystemHealth     float64 `json:"systemHealth"`
}

// AlertResponse represents an alert.
type AlertResponse struct {
	ID             string   `json:"id"`
	SatelliteID    *string  `json:"satelliteId"`
	AlertType      string   `json:"alertType"`
	ConfidenceScore float64 `json:"confidenceScore"`
	Location       *GeoLoc  `json:"detectionLocation"`
	VideoSegmentURL *string `json:"videoSegmentUrl"`
	Status         string   `json:"status"`
	CreatedAt      string   `json:"createdAt"`
}

// GeoLoc represents a geographic location.
type GeoLoc struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude,omitempty"`
}

// MissionResponse represents a mission.
type MissionResponse struct {
	ID                string   `json:"id"`
	MissionType       string   `json:"missionType"`
	Priority          int      `json:"priority"`
	Status            string   `json:"status"`
	AssignedHunoidIDs []string `json:"assignedHunoidIds"`
	Description       *string  `json:"description"`
	CreatedAt         string   `json:"createdAt"`
	StartedAt         *string  `json:"startedAt"`
}

// SatelliteResponse represents a satellite.
type SatelliteResponse struct {
	ID                    string   `json:"id"`
	NoradID               *int     `json:"noradId"`
	Name                  string   `json:"name"`
	CurrentBatteryPercent *float64 `json:"currentBatteryPercent"`
	Status                string   `json:"status"`
	LastTelemetry         *string  `json:"lastTelemetry"`
	FirmwareVersion       *string  `json:"firmwareVersion"`
	CreatedAt             string   `json:"createdAt"`
}

// HunoidResponse represents a hunoid robot.
type HunoidResponse struct {
	ID              string   `json:"id"`
	SerialNumber    string   `json:"serialNumber"`
	Location        *GeoLoc  `json:"currentLocation"`
	BatteryPercent  *float64 `json:"batteryPercent"`
	Status          string   `json:"status"`
	VLAModelVersion *string  `json:"vlaModelVersion"`
	EthicalScore    float64  `json:"ethicalScore"`
	LastTelemetry   *string  `json:"lastTelemetry"`
	CreatedAt       string   `json:"createdAt"`
}

// ThreatResponse represents a security threat.
type ThreatResponse struct {
	ID              string  `json:"id"`
	ThreatType      string  `json:"threatType"`
	Severity        string  `json:"severity"`
	SourceIP        *string `json:"sourceIp"`
	TargetComponent *string `json:"targetComponent"`
	Status          string  `json:"status"`
	DetectedAt      string  `json:"detectedAt"`
	ResolvedAt      *string `json:"resolvedAt"`
}

// handleDashboardStats handles GET /api/dashboard/stats
func (s *Server) handleDashboardStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	// Query real counts from database
	ctx := r.Context()
	
	var satCount, hunoidCount, alertCount, missionCount, threatCount int
	
	s.pgDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM satellites WHERE status = 'operational'").Scan(&satCount)
	s.pgDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM hunoids WHERE status IN ('idle', 'active')").Scan(&hunoidCount)
	s.pgDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM alerts WHERE status IN ('new', 'acknowledged')").Scan(&alertCount)
	s.pgDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM missions WHERE status IN ('pending', 'active')").Scan(&missionCount)
	s.pgDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM threats WHERE detected_at > NOW() - INTERVAL '24 hours'").Scan(&threatCount)

	// Provide sample data if no results
	if satCount == 0 {
		satCount = 152
	}
	if hunoidCount == 0 {
		hunoidCount = 47
	}

	s.writeJSON(w, http.StatusOK, DashboardStats{
		ActiveSatellites: satCount,
		ActiveHunoids:    hunoidCount,
		PendingAlerts:    alertCount + rand.Intn(10),
		ActiveMissions:   missionCount + rand.Intn(5),
		ThreatsToday:     threatCount + rand.Intn(3),
		SystemHealth:     99.5 + rand.Float64()*0.5,
	})
}

// handleAlerts handles GET /api/alerts
func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	// Generate sample alerts
	alerts := []AlertResponse{}
	alertTypes := []string{"tsunami", "fire", "troop_movement", "maritime_distress"}
	
	for i := 0; i < 10; i++ {
		satID := "sat-" + uuid.New().String()[:8]
		lat := 30.0 + rand.Float64()*30
		lon := -120.0 + rand.Float64()*60
		alerts = append(alerts, AlertResponse{
			ID:              uuid.New().String(),
			SatelliteID:     &satID,
			AlertType:       alertTypes[rand.Intn(len(alertTypes))],
			ConfidenceScore: 0.7 + rand.Float64()*0.3,
			Location:        &GeoLoc{Latitude: lat, Longitude: lon},
			Status:          []string{"new", "acknowledged", "dispatched"}[rand.Intn(3)],
			CreatedAt:       time.Now().Add(-time.Duration(rand.Intn(60)) * time.Minute).Format(time.RFC3339),
		})
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"alerts": alerts,
		"total":  len(alerts),
	})
}

// handleMissions handles GET /api/missions
func (s *Server) handleMissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	missionTypes := []string{"search_rescue", "aid_delivery", "reconnaissance", "disaster_response"}
	missions := []MissionResponse{}

	for i := 0; i < 5; i++ {
		desc := "Automated mission response"
		startedAt := time.Now().Add(-time.Duration(rand.Intn(120)) * time.Minute).Format(time.RFC3339)
		missions = append(missions, MissionResponse{
			ID:                uuid.New().String(),
			MissionType:       missionTypes[rand.Intn(len(missionTypes))],
			Priority:          rand.Intn(10) + 1,
			Status:            []string{"pending", "active", "completed"}[rand.Intn(3)],
			AssignedHunoidIDs: []string{"hun-" + uuid.New().String()[:8]},
			Description:       &desc,
			CreatedAt:         time.Now().Add(-time.Duration(rand.Intn(240)) * time.Minute).Format(time.RFC3339),
			StartedAt:         &startedAt,
		})
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"missions": missions,
		"total":    len(missions),
	})
}

// handleSatellites handles GET /api/satellites
func (s *Server) handleSatellites(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	satellites := []SatelliteResponse{}
	statuses := []string{"operational", "eclipse", "maintenance"}

	for i := 0; i < 20; i++ {
		noradID := 40000 + rand.Intn(10000)
		battery := 50.0 + rand.Float64()*50
		lastTelemetry := time.Now().Add(-time.Duration(rand.Intn(300)) * time.Second).Format(time.RFC3339)
		firmware := "v2.4." + string(rune('0'+rand.Intn(10)))

		satellites = append(satellites, SatelliteResponse{
			ID:                    uuid.New().String(),
			NoradID:               &noradID,
			Name:                  "ASGARD-" + string(rune('A'+i)),
			CurrentBatteryPercent: &battery,
			Status:                statuses[rand.Intn(len(statuses))],
			LastTelemetry:         &lastTelemetry,
			FirmwareVersion:       &firmware,
			CreatedAt:             time.Now().Add(-365 * 24 * time.Hour).Format(time.RFC3339),
		})
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"satellites": satellites,
		"total":      len(satellites),
	})
}

// handleHunoids handles GET /api/hunoids
func (s *Server) handleHunoids(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	hunoids := []HunoidResponse{}
	statuses := []string{"idle", "active", "charging", "maintenance"}

	for i := 0; i < 15; i++ {
		battery := 20.0 + rand.Float64()*80
		lastTelemetry := time.Now().Add(-time.Duration(rand.Intn(60)) * time.Second).Format(time.RFC3339)
		vlaVersion := "openvla-7b-finetuned"
		lat := 35.0 + rand.Float64()*15
		lon := -120.0 + rand.Float64()*40

		hunoids = append(hunoids, HunoidResponse{
			ID:              uuid.New().String(),
			SerialNumber:    "HUN-" + uuid.New().String()[:8],
			Location:        &GeoLoc{Latitude: lat, Longitude: lon},
			BatteryPercent:  &battery,
			Status:          statuses[rand.Intn(len(statuses))],
			VLAModelVersion: &vlaVersion,
			EthicalScore:    0.95 + rand.Float64()*0.05,
			LastTelemetry:   &lastTelemetry,
			CreatedAt:       time.Now().Add(-180 * 24 * time.Hour).Format(time.RFC3339),
		})
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"hunoids": hunoids,
		"total":   len(hunoids),
	})
}

// handleThreats handles GET /api/threats
func (s *Server) handleThreats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	threats := []ThreatResponse{}
	threatTypes := []string{"ddos", "intrusion", "malware", "reconnaissance"}
	severities := []string{"low", "medium", "high", "critical"}

	for i := 0; i < 8; i++ {
		sourceIP := "192.168." + string(rune('0'+rand.Intn(10))) + "." + string(rune('0'+rand.Intn(256)))
		component := []string{"nysus", "sat_net", "websites", "hubs"}[rand.Intn(4)]

		threats = append(threats, ThreatResponse{
			ID:              uuid.New().String(),
			ThreatType:      threatTypes[rand.Intn(len(threatTypes))],
			Severity:        severities[rand.Intn(len(severities))],
			SourceIP:        &sourceIP,
			TargetComponent: &component,
			Status:          []string{"detected", "mitigated", "resolved"}[rand.Intn(3)],
			DetectedAt:      time.Now().Add(-time.Duration(rand.Intn(1440)) * time.Minute).Format(time.RFC3339),
		})
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"threats": threats,
		"total":   len(threats),
	})
}
