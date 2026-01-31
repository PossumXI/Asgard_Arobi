// Package services provides business logic services for the API.
package services

import (
	"fmt"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/asgard/pandora/internal/repositories"
)

// DashboardService handles dashboard-related business logic.
type DashboardService struct {
	satelliteRepo *repositories.SatelliteRepository
	hunoidRepo    *repositories.HunoidRepository
	missionRepo   *repositories.MissionRepository
	alertRepo     *repositories.AlertRepository
	threatRepo    *repositories.ThreatRepository
}

// NewDashboardService creates a new dashboard service.
func NewDashboardService(
	satelliteRepo *repositories.SatelliteRepository,
	hunoidRepo *repositories.HunoidRepository,
	missionRepo *repositories.MissionRepository,
	alertRepo *repositories.AlertRepository,
	threatRepo *repositories.ThreatRepository,
) *DashboardService {
	return &DashboardService{
		satelliteRepo: satelliteRepo,
		hunoidRepo:    hunoidRepo,
		missionRepo:   missionRepo,
		alertRepo:     alertRepo,
		threatRepo:    threatRepo,
	}
}

// GetStats returns dashboard statistics.
func (s *DashboardService) GetStats() (map[string]interface{}, error) {
	activeSatellites, err := s.satelliteRepo.GetActiveCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get satellite count: %w", err)
	}

	activeHunoids, err := s.hunoidRepo.GetActiveCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get hunoid count: %w", err)
	}

	pendingAlerts, err := s.alertRepo.GetPendingCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get alert count: %w", err)
	}

	activeMissions, err := s.missionRepo.GetActiveCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get mission count: %w", err)
	}

	threatsToday, err := s.threatRepo.GetTodayCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get threat count: %w", err)
	}

	// Calculate system health based on operational status
	systemHealth := s.calculateSystemHealth(activeSatellites, activeHunoids, pendingAlerts, threatsToday)

	return map[string]interface{}{
		"activeSatellites": activeSatellites,
		"activeHunoids":    activeHunoids,
		"pendingAlerts":    pendingAlerts,
		"activeMissions":   activeMissions,
		"threatsToday":     threatsToday,
		"systemHealth":     systemHealth,
	}, nil
}

// calculateSystemHealth computes system health percentage based on operational metrics.
func (s *DashboardService) calculateSystemHealth(satellites, hunoids, alerts, threats int) float64 {
	// Base health starts at 100%
	health := 100.0

	// Deduct points for low satellite count (expect at least 1 operational)
	if satellites == 0 {
		health -= 30.0 // Critical: no satellites operational
	} else if satellites < 5 {
		health -= float64(5-satellites) * 2.0 // Minor deduction for low count
	}

	// Deduct points for low hunoid count (expect at least 1 operational)
	if hunoids == 0 {
		health -= 20.0 // Significant: no hunoids operational
	} else if hunoids < 3 {
		health -= float64(3-hunoids) * 3.0
	}

	// Deduct points for high alert backlog (alerts > 50 is concerning)
	if alerts > 50 {
		health -= 15.0
	} else if alerts > 20 {
		health -= float64(alerts-20) * 0.3
	}

	// Deduct points for threats (each threat reduces health)
	if threats > 10 {
		health -= 20.0 // Critical threat level
	} else if threats > 5 {
		health -= float64(threats-5) * 2.0
	} else if threats > 0 {
		health -= float64(threats) * 1.0
	}

	// Ensure health stays within bounds
	if health < 0 {
		health = 0
	}
	if health > 100 {
		health = 100
	}

	return health
}

// GetAlerts retrieves all alerts.
func (s *DashboardService) GetAlerts() ([]*db.Alert, error) {
	alerts, err := s.alertRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get alerts: %w", err)
	}
	return alerts, nil
}

// GetAlert retrieves an alert by ID.
func (s *DashboardService) GetAlert(id string) (*db.Alert, error) {
	alert, err := s.alertRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}
	return alert, nil
}

// GetMissions retrieves all missions.
func (s *DashboardService) GetMissions() ([]*db.Mission, error) {
	missions, err := s.missionRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get missions: %w", err)
	}
	return missions, nil
}

// GetMission retrieves a mission by ID.
func (s *DashboardService) GetMission(id string) (*db.Mission, error) {
	mission, err := s.missionRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get mission: %w", err)
	}
	return mission, nil
}

// GetSatellites retrieves all satellites.
func (s *DashboardService) GetSatellites() ([]*db.Satellite, error) {
	satellites, err := s.satelliteRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get satellites: %w", err)
	}
	return satellites, nil
}

// GetSatellite retrieves a satellite by ID.
func (s *DashboardService) GetSatellite(id string) (*db.Satellite, error) {
	satellite, err := s.satelliteRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get satellite: %w", err)
	}
	return satellite, nil
}

// GetHunoids retrieves all hunoids.
func (s *DashboardService) GetHunoids() ([]*db.Hunoid, error) {
	hunoids, err := s.hunoidRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to get hunoids: %w", err)
	}
	return hunoids, nil
}

// GetHunoid retrieves a hunoid by ID.
func (s *DashboardService) GetHunoid(id string) (*db.Hunoid, error) {
	hunoid, err := s.hunoidRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get hunoid: %w", err)
	}
	return hunoid, nil
}

// GetHunoidLocation retrieves a hunoid's current location if available.
func (s *DashboardService) GetHunoidLocation(id string) (*repositories.GeoLocation, error) {
	location, err := s.hunoidRepo.GetLocation(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get hunoid location: %w", err)
	}
	return location, nil
}

// TelemetrySnapshot represents telemetry fields for API responses.
type TelemetrySnapshot struct {
	BatteryPercent float64
	Status         string
	LastTelemetry  *time.Time
	Location       *repositories.GeoLocation
}

// GetSatelliteTelemetry returns telemetry from the satellites_api view.
func (s *DashboardService) GetSatelliteTelemetry(id string) (*TelemetrySnapshot, error) {
	telemetry, err := s.satelliteRepo.GetTelemetry(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get satellite telemetry: %w", err)
	}

	var lastTelemetry *time.Time
	if telemetry.LastTelemetry.Valid {
		timestamp := telemetry.LastTelemetry.Time.UTC()
		lastTelemetry = &timestamp
	}

	battery := 0.0
	if telemetry.BatteryPercent.Valid {
		battery = telemetry.BatteryPercent.Float64
	}

	return &TelemetrySnapshot{
		BatteryPercent: battery,
		Status:         telemetry.Status,
		LastTelemetry:  lastTelemetry,
		Location:       nil,
	}, nil
}

// GetHunoidTelemetry returns telemetry from the hunoids_api view.
func (s *DashboardService) GetHunoidTelemetry(id string) (*TelemetrySnapshot, error) {
	telemetry, err := s.hunoidRepo.GetTelemetry(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get hunoid telemetry: %w", err)
	}

	var lastTelemetry *time.Time
	if telemetry.LastTelemetry.Valid {
		timestamp := telemetry.LastTelemetry.Time.UTC()
		lastTelemetry = &timestamp
	}

	battery := 0.0
	if telemetry.BatteryPercent.Valid {
		battery = telemetry.BatteryPercent.Float64
	}

	var location *repositories.GeoLocation
	if telemetry.Latitude.Valid && telemetry.Longitude.Valid {
		location = &repositories.GeoLocation{
			Latitude:  telemetry.Latitude.Float64,
			Longitude: telemetry.Longitude.Float64,
		}
		if telemetry.Altitude.Valid {
			location.Altitude = telemetry.Altitude.Float64
		}
	}

	return &TelemetrySnapshot{
		BatteryPercent: battery,
		Status:         telemetry.Status,
		LastTelemetry:  lastTelemetry,
		Location:       location,
	}, nil
}
