// Package services provides business logic services for ASGARD.
package services

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/asgard/pandora/internal/platform/satellite"
)

// SatelliteTrackingService provides real-time satellite tracking for the ASGARD system.
// Integrates with N2YO API and TLE propagation for orbital awareness.
type SatelliteTrackingService struct {
	client     *satellite.Client
	cache      *trackingCache
	observer   satellite.Observer
	mu         sync.RWMutex
	fleetTLEs  map[int]*satellite.TLE
	propagators map[int]*satellite.Propagator
}

// SatelliteTrackingConfig holds configuration for the tracking service.
type SatelliteTrackingConfig struct {
	N2YOAPIKey string
	Observer   satellite.Observer
	FleetIDs   []int // NORAD IDs of satellites to track
	UpdateInterval time.Duration
}

// DefaultTrackingConfig returns a default configuration.
func DefaultTrackingConfig() SatelliteTrackingConfig {
	return SatelliteTrackingConfig{
		Observer: satellite.Observer{
			Latitude:  40.7128,  // NYC default
			Longitude: -74.0060,
			Altitude:  10,
		},
		FleetIDs: []int{
			satellite.NoradISS,      // ISS for testing
			satellite.NoradTerra,    // Earth observation
			satellite.NoradAqua,     // Earth observation
			satellite.NoradNOAA19,   // Weather
			satellite.NoradLandsat8, // Imaging
		},
		UpdateInterval: 1 * time.Hour,
	}
}

// NewSatelliteTrackingService creates a new satellite tracking service.
func NewSatelliteTrackingService(cfg SatelliteTrackingConfig) *SatelliteTrackingService {
	clientCfg := satellite.DefaultConfig()
	clientCfg.N2YOAPIKey = cfg.N2YOAPIKey

	return &SatelliteTrackingService{
		client:      satellite.NewClient(clientCfg),
		cache:       newTrackingCache(cfg.UpdateInterval),
		observer:    cfg.Observer,
		fleetTLEs:   make(map[int]*satellite.TLE),
		propagators: make(map[int]*satellite.Propagator),
	}
}

// Initialize loads TLE data for all fleet satellites.
func (s *SatelliteTrackingService) Initialize(ctx context.Context, fleetIDs []int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, noradID := range fleetIDs {
		tle, err := s.client.GetTLE(ctx, noradID)
		if err != nil {
			log.Printf("[SatelliteTracking] Failed to fetch TLE for %d: %v", noradID, err)
			continue
		}

		s.fleetTLEs[noradID] = tle

		propagator, err := satellite.NewPropagator(tle)
		if err != nil {
			log.Printf("[SatelliteTracking] Failed to create propagator for %d: %v", noradID, err)
			continue
		}

		s.propagators[noradID] = propagator
		log.Printf("[SatelliteTracking] Loaded TLE for %s (NORAD %d)", tle.Name, noradID)
	}

	return nil
}

// TrackedSatellite represents a satellite with current position.
type TrackedSatellite struct {
	NoradID     int       `json:"norad_id"`
	Name        string    `json:"name"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	Altitude    float64   `json:"altitude_km"`
	Velocity    float64   `json:"velocity_kmh,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	Source      string    `json:"source"` // "propagated" or "realtime"
	IsEclipsed  bool      `json:"is_eclipsed,omitempty"`
}

// GetPosition returns the current position of a satellite.
func (s *SatelliteTrackingService) GetPosition(ctx context.Context, noradID int) (*TrackedSatellite, error) {
	s.mu.RLock()
	propagator, exists := s.propagators[noradID]
	tle := s.fleetTLEs[noradID]
	s.mu.RUnlock()

	now := time.Now().UTC()

	// If we have a propagator, use it
	if exists && propagator != nil {
		lat, lon, alt := propagator.Propagate(now)
		return &TrackedSatellite{
			NoradID:   noradID,
			Name:      tle.Name,
			Latitude:  lat,
			Longitude: lon,
			Altitude:  alt,
			Timestamp: now,
			Source:    "propagated",
		}, nil
	}

	// Otherwise, try to fetch TLE and create propagator
	tle, err := s.client.GetTLE(ctx, noradID)
	if err != nil {
		return nil, err
	}

	propagator, err = satellite.NewPropagator(tle)
	if err != nil {
		return nil, err
	}

	// Cache for future use
	s.mu.Lock()
	s.fleetTLEs[noradID] = tle
	s.propagators[noradID] = propagator
	s.mu.Unlock()

	lat, lon, alt := propagator.Propagate(now)
	return &TrackedSatellite{
		NoradID:   noradID,
		Name:      tle.Name,
		Latitude:  lat,
		Longitude: lon,
		Altitude:  alt,
		Timestamp: now,
		Source:    "propagated",
	}, nil
}

// GetRealtimePosition fetches real-time position from N2YO API.
func (s *SatelliteTrackingService) GetRealtimePosition(ctx context.Context, noradID int) (*TrackedSatellite, error) {
	positions, err := s.client.GetPosition(ctx, noradID, s.observer, 1)
	if err != nil {
		return nil, err
	}

	if len(positions) == 0 {
		return s.GetPosition(ctx, noradID) // Fallback to propagated
	}

	p := positions[0]
	return &TrackedSatellite{
		NoradID:    noradID,
		Name:       p.Name,
		Latitude:   p.Latitude,
		Longitude:  p.Longitude,
		Altitude:   p.Altitude,
		Timestamp:  p.Timestamp,
		Source:     "realtime",
		IsEclipsed: p.Eclipsed,
	}, nil
}

// GetFleetPositions returns current positions of all tracked satellites.
func (s *SatelliteTrackingService) GetFleetPositions(ctx context.Context) []TrackedSatellite {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().UTC()
	positions := make([]TrackedSatellite, 0, len(s.propagators))

	for noradID, propagator := range s.propagators {
		tle := s.fleetTLEs[noradID]
		lat, lon, alt := propagator.Propagate(now)

		positions = append(positions, TrackedSatellite{
			NoradID:   noradID,
			Name:      tle.Name,
			Latitude:  lat,
			Longitude: lon,
			Altitude:  alt,
			Timestamp: now,
			Source:    "propagated",
		})
	}

	return positions
}

// GroundTrack represents a satellite's ground track over time.
type GroundTrack struct {
	NoradID   int                           `json:"norad_id"`
	Name      string                        `json:"name"`
	StartTime time.Time                     `json:"start_time"`
	EndTime   time.Time                     `json:"end_time"`
	Points    []satellite.PropagatedPosition `json:"points"`
}

// GetGroundTrack computes the ground track for a satellite.
func (s *SatelliteTrackingService) GetGroundTrack(ctx context.Context, noradID int, duration time.Duration) (*GroundTrack, error) {
	s.mu.RLock()
	propagator, exists := s.propagators[noradID]
	tle := s.fleetTLEs[noradID]
	s.mu.RUnlock()

	if !exists {
		// Try to load TLE
		var err error
		tle, err = s.client.GetTLE(ctx, noradID)
		if err != nil {
			return nil, err
		}

		propagator, err = satellite.NewPropagator(tle)
		if err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC()
	positions := propagator.PropagateRange(now, duration, time.Minute)

	return &GroundTrack{
		NoradID:   noradID,
		Name:      tle.Name,
		StartTime: now,
		EndTime:   now.Add(duration),
		Points:    positions,
	}, nil
}

// ContactWindow represents a time window when a satellite is visible.
type ContactWindow struct {
	NoradID        int       `json:"norad_id"`
	Name           string    `json:"name"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	MaxElevation   float64   `json:"max_elevation_deg"`
	DurationSec    int       `json:"duration_seconds"`
}

// GetContactWindows returns upcoming contact windows for a satellite.
func (s *SatelliteTrackingService) GetContactWindows(ctx context.Context, noradID int, days int) ([]ContactWindow, error) {
	passes, err := s.client.GetVisualPasses(ctx, noradID, s.observer, days, 60)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	tle := s.fleetTLEs[noradID]
	s.mu.RUnlock()

	name := ""
	if tle != nil {
		name = tle.Name
	}

	windows := make([]ContactWindow, 0, len(passes))
	for _, p := range passes {
		windows = append(windows, ContactWindow{
			NoradID:      noradID,
			Name:         name,
			StartTime:    p.StartTime,
			EndTime:      p.EndTime,
			MaxElevation: p.MaxElevation,
			DurationSec:  p.Duration,
		})
	}

	return windows, nil
}

// SatellitesAbove returns all satellites currently above the observer.
func (s *SatelliteTrackingService) SatellitesAbove(ctx context.Context, searchRadius int, categoryID int) ([]satellite.SatelliteAbove, error) {
	return s.client.GetSatellitesAbove(ctx, s.observer, searchRadius, categoryID)
}

// SetObserver updates the ground observer location.
func (s *SatelliteTrackingService) SetObserver(lat, lon, alt float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.observer = satellite.Observer{
		Latitude:  lat,
		Longitude: lon,
		Altitude:  alt,
	}
}

// RefreshTLEs updates TLE data for all tracked satellites.
func (s *SatelliteTrackingService) RefreshTLEs(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for noradID := range s.fleetTLEs {
		tle, err := s.client.GetTLE(ctx, noradID)
		if err != nil {
			log.Printf("[SatelliteTracking] Failed to refresh TLE for %d: %v", noradID, err)
			continue
		}

		s.fleetTLEs[noradID] = tle

		propagator, err := satellite.NewPropagator(tle)
		if err != nil {
			log.Printf("[SatelliteTracking] Failed to update propagator for %d: %v", noradID, err)
			continue
		}

		s.propagators[noradID] = propagator
		log.Printf("[SatelliteTracking] Refreshed TLE for %s", tle.Name)
	}

	return nil
}

// trackingCache provides caching for satellite data.
type trackingCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
}

type cacheEntry struct {
	data      interface{}
	expiresAt time.Time
}

func newTrackingCache(ttl time.Duration) *trackingCache {
	return &trackingCache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
	}
}

func (c *trackingCache) get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.data, true
}

func (c *trackingCache) set(key string, data interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(c.ttl),
	}
}
