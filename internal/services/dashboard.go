// Package services provides business logic services for the API.
package services

import (
	"fmt"

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

	return map[string]interface{}{
		"activeSatellites": activeSatellites,
		"activeHunoids":    activeHunoids,
		"pendingAlerts":     pendingAlerts,
		"activeMissions":    activeMissions,
		"threatsToday":      threatsToday,
		"systemHealth":      95, // Placeholder
	}, nil
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
