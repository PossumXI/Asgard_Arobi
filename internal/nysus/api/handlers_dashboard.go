package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"
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

	ctx := r.Context()
	
	var satCount, hunoidCount, alertCount, missionCount, threatCount int
	var systemHealth float64 = 100.0
	
	if s.pgDB != nil {
		// Query satellite count
		if err := s.pgDB.QueryRowContext(ctx, 
			"SELECT COUNT(*) FROM satellites WHERE status = 'operational'").Scan(&satCount); err != nil {
			s.writeError(w, http.StatusInternalServerError, "Database error", "DB_ERROR")
			return
		}
		
		// Query hunoid count
		if err := s.pgDB.QueryRowContext(ctx, 
			"SELECT COUNT(*) FROM hunoids WHERE status IN ('idle', 'active')").Scan(&hunoidCount); err != nil {
			s.writeError(w, http.StatusInternalServerError, "Database error", "DB_ERROR")
			return
		}
		
		// Query pending alerts
		if err := s.pgDB.QueryRowContext(ctx, 
			"SELECT COUNT(*) FROM alerts WHERE status IN ('new', 'acknowledged')").Scan(&alertCount); err != nil {
			s.writeError(w, http.StatusInternalServerError, "Database error", "DB_ERROR")
			return
		}
		
		// Query active missions
		if err := s.pgDB.QueryRowContext(ctx, 
			"SELECT COUNT(*) FROM missions WHERE status IN ('pending', 'active')").Scan(&missionCount); err != nil {
			s.writeError(w, http.StatusInternalServerError, "Database error", "DB_ERROR")
			return
		}
		
		// Query threats today
		if err := s.pgDB.QueryRowContext(ctx, 
			"SELECT COUNT(*) FROM threats WHERE detected_at > NOW() - INTERVAL '24 hours'").Scan(&threatCount); err != nil {
			s.writeError(w, http.StatusInternalServerError, "Database error", "DB_ERROR")
			return
		}
		
		// Calculate system health based on component status
		var degradedCount int
		s.pgDB.QueryRowContext(ctx, 
			`SELECT COUNT(*) FROM (
				SELECT 1 FROM satellites WHERE status != 'operational'
				UNION ALL
				SELECT 1 FROM hunoids WHERE status = 'error'
			) AS degraded`).Scan(&degradedCount)
		
		totalComponents := satCount + hunoidCount
		if totalComponents > 0 {
			systemHealth = 100.0 * float64(totalComponents-degradedCount) / float64(totalComponents)
		}
	} else {
		s.writeError(w, http.StatusServiceUnavailable, "Database not available", "DB_UNAVAILABLE")
		return
	}

	s.writeJSON(w, http.StatusOK, DashboardStats{
		ActiveSatellites: satCount,
		ActiveHunoids:    hunoidCount,
		PendingAlerts:    alertCount,
		ActiveMissions:   missionCount,
		ThreatsToday:     threatCount,
		SystemHealth:     systemHealth,
	})
}

// handleAlerts handles GET /api/alerts
func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	ctx := r.Context()
	
	if s.pgDB == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Database not available", "DB_UNAVAILABLE")
		return
	}

	// Parse query parameters
	status := r.URL.Query().Get("status")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Build query
	query := `
		SELECT a.id, a.satellite_id, a.alert_type, a.confidence_score, 
			   a.latitude, a.longitude, a.altitude, a.video_segment_url,
			   a.status, a.created_at
		FROM alerts a
	`
	args := []interface{}{}
	
	if status != "" {
		query += " WHERE a.status = $1"
		args = append(args, status)
	}
	
	query += " ORDER BY a.created_at DESC LIMIT $" + strconv.Itoa(len(args)+1)
	args = append(args, limit)

	rows, err := s.pgDB.QueryContext(ctx, query, args...)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Database query failed", "DB_ERROR")
		return
	}
	defer rows.Close()

	alerts := []AlertResponse{}
	for rows.Next() {
		var alert AlertResponse
		var lat, lon, alt sql.NullFloat64
		var satID, videoURL sql.NullString
		var createdAt time.Time

		err := rows.Scan(
			&alert.ID, &satID, &alert.AlertType, &alert.ConfidenceScore,
			&lat, &lon, &alt, &videoURL,
			&alert.Status, &createdAt,
		)
		if err != nil {
			continue
		}

		if satID.Valid {
			alert.SatelliteID = &satID.String
		}
		if lat.Valid && lon.Valid {
			alert.Location = &GeoLoc{
				Latitude:  lat.Float64,
				Longitude: lon.Float64,
				Altitude:  alt.Float64,
			}
		}
		if videoURL.Valid {
			alert.VideoSegmentURL = &videoURL.String
		}
		alert.CreatedAt = createdAt.Format(time.RFC3339)

		alerts = append(alerts, alert)
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM alerts"
	if status != "" {
		countQuery += " WHERE status = $1"
		s.pgDB.QueryRowContext(ctx, countQuery, status).Scan(&total)
	} else {
		s.pgDB.QueryRowContext(ctx, countQuery).Scan(&total)
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"alerts": alerts,
		"total":  total,
	})
}

// handleMissions handles GET /api/missions
func (s *Server) handleMissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	ctx := r.Context()
	
	if s.pgDB == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Database not available", "DB_UNAVAILABLE")
		return
	}

	// Parse query parameters
	status := r.URL.Query().Get("status")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Build query
	query := `
		SELECT m.id, m.mission_type, m.priority, m.status, m.description,
			   m.created_at, m.started_at,
			   COALESCE(array_agg(mh.hunoid_id) FILTER (WHERE mh.hunoid_id IS NOT NULL), '{}')
		FROM missions m
		LEFT JOIN mission_hunoids mh ON m.id = mh.mission_id
	`
	args := []interface{}{}
	
	if status != "" {
		query += " WHERE m.status = $1"
		args = append(args, status)
	}
	
	query += " GROUP BY m.id ORDER BY m.created_at DESC LIMIT $" + strconv.Itoa(len(args)+1)
	args = append(args, limit)

	rows, err := s.pgDB.QueryContext(ctx, query, args...)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Database query failed", "DB_ERROR")
		return
	}
	defer rows.Close()

	missions := []MissionResponse{}
	for rows.Next() {
		var mission MissionResponse
		var desc sql.NullString
		var createdAt time.Time
		var startedAt sql.NullTime
		var hunoidIDs []string

		err := rows.Scan(
			&mission.ID, &mission.MissionType, &mission.Priority, &mission.Status,
			&desc, &createdAt, &startedAt, &hunoidIDs,
		)
		if err != nil {
			continue
		}

		if desc.Valid {
			mission.Description = &desc.String
		}
		mission.CreatedAt = createdAt.Format(time.RFC3339)
		if startedAt.Valid {
			s := startedAt.Time.Format(time.RFC3339)
			mission.StartedAt = &s
		}
		mission.AssignedHunoidIDs = hunoidIDs

		missions = append(missions, mission)
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM missions"
	if status != "" {
		countQuery += " WHERE status = $1"
		s.pgDB.QueryRowContext(ctx, countQuery, status).Scan(&total)
	} else {
		s.pgDB.QueryRowContext(ctx, countQuery).Scan(&total)
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"missions": missions,
		"total":    total,
	})
}

// handleSatellites handles GET /api/satellites
func (s *Server) handleSatellites(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	ctx := r.Context()
	
	if s.pgDB == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Database not available", "DB_UNAVAILABLE")
		return
	}

	// Parse query parameters
	status := r.URL.Query().Get("status")
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}

	// Build query
	query := `
		SELECT s.id, s.norad_id, s.name, s.current_battery_percent, s.status,
			   s.last_telemetry, s.firmware_version, s.created_at
		FROM satellites s
	`
	args := []interface{}{}
	
	if status != "" {
		query += " WHERE s.status = $1"
		args = append(args, status)
	}
	
	query += " ORDER BY s.name ASC LIMIT $" + strconv.Itoa(len(args)+1)
	args = append(args, limit)

	rows, err := s.pgDB.QueryContext(ctx, query, args...)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Database query failed", "DB_ERROR")
		return
	}
	defer rows.Close()

	satellites := []SatelliteResponse{}
	for rows.Next() {
		var sat SatelliteResponse
		var noradID sql.NullInt64
		var battery sql.NullFloat64
		var lastTelemetry sql.NullTime
		var firmware sql.NullString
		var createdAt time.Time

		err := rows.Scan(
			&sat.ID, &noradID, &sat.Name, &battery, &sat.Status,
			&lastTelemetry, &firmware, &createdAt,
		)
		if err != nil {
			continue
		}

		if noradID.Valid {
			n := int(noradID.Int64)
			sat.NoradID = &n
		}
		if battery.Valid {
			sat.CurrentBatteryPercent = &battery.Float64
		}
		if lastTelemetry.Valid {
			t := lastTelemetry.Time.Format(time.RFC3339)
			sat.LastTelemetry = &t
		}
		if firmware.Valid {
			sat.FirmwareVersion = &firmware.String
		}
		sat.CreatedAt = createdAt.Format(time.RFC3339)

		satellites = append(satellites, sat)
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM satellites"
	if status != "" {
		countQuery += " WHERE status = $1"
		s.pgDB.QueryRowContext(ctx, countQuery, status).Scan(&total)
	} else {
		s.pgDB.QueryRowContext(ctx, countQuery).Scan(&total)
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"satellites": satellites,
		"total":      total,
	})
}

// handleHunoids handles GET /api/hunoids
func (s *Server) handleHunoids(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	ctx := r.Context()
	
	if s.pgDB == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Database not available", "DB_UNAVAILABLE")
		return
	}

	// Parse query parameters
	status := r.URL.Query().Get("status")
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}

	// Build query
	query := `
		SELECT h.id, h.serial_number, h.latitude, h.longitude, h.battery_percent,
			   h.status, h.vla_model_version, h.ethical_score, h.last_telemetry, h.created_at
		FROM hunoids h
	`
	args := []interface{}{}
	
	if status != "" {
		query += " WHERE h.status = $1"
		args = append(args, status)
	}
	
	query += " ORDER BY h.serial_number ASC LIMIT $" + strconv.Itoa(len(args)+1)
	args = append(args, limit)

	rows, err := s.pgDB.QueryContext(ctx, query, args...)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Database query failed", "DB_ERROR")
		return
	}
	defer rows.Close()

	hunoids := []HunoidResponse{}
	for rows.Next() {
		var hun HunoidResponse
		var lat, lon, battery sql.NullFloat64
		var vlaVersion sql.NullString
		var lastTelemetry sql.NullTime
		var createdAt time.Time

		err := rows.Scan(
			&hun.ID, &hun.SerialNumber, &lat, &lon, &battery,
			&hun.Status, &vlaVersion, &hun.EthicalScore, &lastTelemetry, &createdAt,
		)
		if err != nil {
			continue
		}

		if lat.Valid && lon.Valid {
			hun.Location = &GeoLoc{Latitude: lat.Float64, Longitude: lon.Float64}
		}
		if battery.Valid {
			hun.BatteryPercent = &battery.Float64
		}
		if vlaVersion.Valid {
			hun.VLAModelVersion = &vlaVersion.String
		}
		if lastTelemetry.Valid {
			t := lastTelemetry.Time.Format(time.RFC3339)
			hun.LastTelemetry = &t
		}
		hun.CreatedAt = createdAt.Format(time.RFC3339)

		hunoids = append(hunoids, hun)
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM hunoids"
	if status != "" {
		countQuery += " WHERE status = $1"
		s.pgDB.QueryRowContext(ctx, countQuery, status).Scan(&total)
	} else {
		s.pgDB.QueryRowContext(ctx, countQuery).Scan(&total)
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"hunoids": hunoids,
		"total":   total,
	})
}

// handleThreats handles GET /api/threats
func (s *Server) handleThreats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	ctx := r.Context()
	
	if s.pgDB == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Database not available", "DB_UNAVAILABLE")
		return
	}

	// Parse query parameters
	status := r.URL.Query().Get("status")
	severity := r.URL.Query().Get("severity")
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}

	// Build query
	query := `
		SELECT t.id, t.threat_type, t.severity, t.source_ip, t.target_component,
			   t.status, t.detected_at, t.resolved_at
		FROM threats t
		WHERE 1=1
	`
	args := []interface{}{}
	argNum := 1
	
	if status != "" {
		query += " AND t.status = $" + strconv.Itoa(argNum)
		args = append(args, status)
		argNum++
	}
	if severity != "" {
		query += " AND t.severity = $" + strconv.Itoa(argNum)
		args = append(args, severity)
		argNum++
	}
	
	query += " ORDER BY t.detected_at DESC LIMIT $" + strconv.Itoa(argNum)
	args = append(args, limit)

	rows, err := s.pgDB.QueryContext(ctx, query, args...)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Database query failed", "DB_ERROR")
		return
	}
	defer rows.Close()

	threats := []ThreatResponse{}
	for rows.Next() {
		var threat ThreatResponse
		var sourceIP, component sql.NullString
		var detectedAt time.Time
		var resolvedAt sql.NullTime

		err := rows.Scan(
			&threat.ID, &threat.ThreatType, &threat.Severity, &sourceIP, &component,
			&threat.Status, &detectedAt, &resolvedAt,
		)
		if err != nil {
			continue
		}

		if sourceIP.Valid {
			threat.SourceIP = &sourceIP.String
		}
		if component.Valid {
			threat.TargetComponent = &component.String
		}
		threat.DetectedAt = detectedAt.Format(time.RFC3339)
		if resolvedAt.Valid {
			t := resolvedAt.Time.Format(time.RFC3339)
			threat.ResolvedAt = &t
		}

		threats = append(threats, threat)
	}

	// Get total count
	var total int
	s.pgDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM threats").Scan(&total)

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"threats": threats,
		"total":   total,
	})
}
