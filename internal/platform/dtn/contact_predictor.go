// Package dtn provides Delay Tolerant Networking implementation.
package dtn

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/asgard/pandora/internal/platform/satellite"
)

// ContactPredictor uses real orbital data to predict communication windows.
type ContactPredictor struct {
	client         *satellite.Client
	propagators    map[int]*satellite.Propagator
	groundStations []GroundStation
	satellites     []SatelliteNode
	mu             sync.RWMutex
}

// GroundStation represents a ground-based communication station.
type GroundStation struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Altitude     float64 `json:"altitude_m"`
	MinElevation float64 `json:"min_elevation_deg"` // Minimum elevation for contact
}

// SatelliteNode represents a satellite in the network.
type SatelliteNode struct {
	NoradID int    `json:"norad_id"`
	Name    string `json:"name"`
	EID     string `json:"eid"` // DTN endpoint ID
}

// PredictedContact represents a future communication window.
type PredictedContact struct {
	SatelliteID   int       `json:"satellite_id"`
	SatelliteName string    `json:"satellite_name"`
	GroundStation string    `json:"ground_station"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	MaxElevation  float64   `json:"max_elevation_deg"`
	DurationSec   int       `json:"duration_seconds"`
	Quality       float64   `json:"link_quality"` // 0-1 based on elevation
}

// ContactPredictorConfig holds configuration.
type ContactPredictorConfig struct {
	N2YOAPIKey     string
	GroundStations []GroundStation
	Satellites     []SatelliteNode
}

// DefaultContactPredictorConfig returns a default configuration.
func DefaultContactPredictorConfig() ContactPredictorConfig {
	return ContactPredictorConfig{
		GroundStations: []GroundStation{
			{
				ID:           "gs_nyc",
				Name:         "New York",
				Latitude:     40.7128,
				Longitude:    -74.0060,
				Altitude:     10,
				MinElevation: 10,
			},
			{
				ID:           "gs_la",
				Name:         "Los Angeles",
				Latitude:     34.0522,
				Longitude:    -118.2437,
				Altitude:     71,
				MinElevation: 10,
			},
			{
				ID:           "gs_london",
				Name:         "London",
				Latitude:     51.5074,
				Longitude:    -0.1278,
				Altitude:     11,
				MinElevation: 10,
			},
		},
		Satellites: []SatelliteNode{
			{NoradID: satellite.NoradISS, Name: "ISS", EID: "dtn://iss/main"},
			{NoradID: satellite.NoradTerra, Name: "Terra", EID: "dtn://terra/main"},
			{NoradID: satellite.NoradAqua, Name: "Aqua", EID: "dtn://aqua/main"},
		},
	}
}

// NewContactPredictor creates a new contact predictor.
func NewContactPredictor(cfg ContactPredictorConfig) *ContactPredictor {
	clientCfg := satellite.DefaultConfig()
	clientCfg.N2YOAPIKey = cfg.N2YOAPIKey

	return &ContactPredictor{
		client:         satellite.NewClient(clientCfg),
		propagators:    make(map[int]*satellite.Propagator),
		groundStations: cfg.GroundStations,
		satellites:     cfg.Satellites,
	}
}

// Initialize loads TLE data for all satellites.
func (cp *ContactPredictor) Initialize(ctx context.Context) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	for _, sat := range cp.satellites {
		tle, err := cp.client.GetTLE(ctx, sat.NoradID)
		if err != nil {
			log.Printf("[ContactPredictor] Failed to load TLE for %s: %v", sat.Name, err)
			continue
		}

		propagator, err := satellite.NewPropagator(tle)
		if err != nil {
			log.Printf("[ContactPredictor] Failed to create propagator for %s: %v", sat.Name, err)
			continue
		}

		cp.propagators[sat.NoradID] = propagator
		log.Printf("[ContactPredictor] Initialized %s (NORAD %d)", sat.Name, sat.NoradID)
	}

	return nil
}

// PredictContacts returns predicted contact windows for all satellite-ground station pairs.
func (cp *ContactPredictor) PredictContacts(ctx context.Context, duration time.Duration, step time.Duration) []PredictedContact {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	var contacts []PredictedContact
	now := time.Now().UTC()
	endTime := now.Add(duration)

	for _, sat := range cp.satellites {
		propagator, ok := cp.propagators[sat.NoradID]
		if !ok {
			continue
		}

		for _, gs := range cp.groundStations {
			gsContacts := cp.predictContactsForPair(propagator, sat, gs, now, endTime, step)
			contacts = append(contacts, gsContacts...)
		}
	}

	return contacts
}

// predictContactsForPair predicts contacts between one satellite and one ground station.
func (cp *ContactPredictor) predictContactsForPair(
	propagator *satellite.Propagator,
	sat SatelliteNode,
	gs GroundStation,
	startTime, endTime time.Time,
	step time.Duration,
) []PredictedContact {
	var contacts []PredictedContact
	var currentContact *PredictedContact
	var maxElevation float64

	for t := startTime; t.Before(endTime); t = t.Add(step) {
		lat, lon, alt := propagator.Propagate(t)
		elevation := cp.calculateElevation(lat, lon, alt, gs)

		if elevation >= gs.MinElevation {
			if currentContact == nil {
				// Start of new contact
				currentContact = &PredictedContact{
					SatelliteID:   sat.NoradID,
					SatelliteName: sat.Name,
					GroundStation: gs.Name,
					StartTime:     t,
				}
				maxElevation = elevation
			}
			if elevation > maxElevation {
				maxElevation = elevation
			}
		} else {
			if currentContact != nil {
				// End of contact
				currentContact.EndTime = t
				currentContact.MaxElevation = maxElevation
				currentContact.DurationSec = int(currentContact.EndTime.Sub(currentContact.StartTime).Seconds())
				currentContact.Quality = cp.calculateLinkQuality(maxElevation)
				contacts = append(contacts, *currentContact)
				currentContact = nil
				maxElevation = 0
			}
		}
	}

	// Handle contact that extends past endTime
	if currentContact != nil {
		currentContact.EndTime = endTime
		currentContact.MaxElevation = maxElevation
		currentContact.DurationSec = int(currentContact.EndTime.Sub(currentContact.StartTime).Seconds())
		currentContact.Quality = cp.calculateLinkQuality(maxElevation)
		contacts = append(contacts, *currentContact)
	}

	return contacts
}

// calculateElevation calculates elevation angle from ground station to satellite.
func (cp *ContactPredictor) calculateElevation(satLat, satLon, satAlt float64, gs GroundStation) float64 {
	// Simplified elevation calculation
	// In production, use proper geodetic calculations

	const earthRadius = 6371.0 // km
	const deg2rad = 0.017453292519943295

	// Great circle distance
	dLat := (satLat - gs.Latitude) * deg2rad
	dLon := (satLon - gs.Longitude) * deg2rad

	a := sin(dLat/2)*sin(dLat/2) +
		cos(gs.Latitude*deg2rad)*cos(satLat*deg2rad)*
			sin(dLon/2)*sin(dLon/2)
	c := 2 * atan2(sqrt(a), sqrt(1-a))

	groundDistance := earthRadius * c // km

	// Satellite height above ground station
	height := satAlt // km (approximate, ignoring Earth curvature for simplicity)

	// Elevation angle
	if groundDistance < 0.001 {
		return 90.0 // Directly overhead
	}

	elevation := atan(height/groundDistance) * 180.0 / 3.14159265358979

	// Account for Earth curvature (simplified)
	curvatureCorrection := groundDistance / (2 * earthRadius) * 180.0 / 3.14159265358979
	elevation -= curvatureCorrection

	if elevation < 0 {
		elevation = -10 // Below horizon
	}

	return elevation
}

// calculateLinkQuality estimates link quality based on elevation.
func (cp *ContactPredictor) calculateLinkQuality(elevation float64) float64 {
	// Higher elevation = better link quality
	// 90° = 1.0, 10° = 0.5, 0° = 0.0
	if elevation <= 0 {
		return 0.0
	}
	if elevation >= 90 {
		return 1.0
	}
	return 0.3 + 0.7*(elevation/90.0)
}

// GetNextContact returns the next contact opportunity for a satellite.
func (cp *ContactPredictor) GetNextContact(ctx context.Context, noradID int) (*PredictedContact, error) {
	contacts := cp.PredictContacts(ctx, 24*time.Hour, time.Minute)

	now := time.Now().UTC()
	for _, c := range contacts {
		if c.SatelliteID == noradID && c.StartTime.After(now) {
			return &c, nil
		}
	}

	return nil, nil
}

// UpdateRouterContactGraph updates the DTN router with predicted contacts.
func (cp *ContactPredictor) UpdateRouterContactGraph(router *EnergyAwareRouter) {
	contacts := cp.PredictContacts(context.Background(), 4*time.Hour, time.Minute)

	for _, contact := range contacts {
		neighbor := &Neighbor{
			ID:           contact.GroundStation,
			EID:          "dtn://earth/" + contact.GroundStation,
			LinkQuality:  contact.Quality,
			IsActive:     time.Now().UTC().After(contact.StartTime) && time.Now().UTC().Before(contact.EndTime),
			Latency:      50 * time.Millisecond, // Typical LEO latency
			Bandwidth:    10_000_000,            // 10 Mbps typical
			ContactStart: contact.StartTime,
			ContactEnd:   contact.EndTime,
		}
		router.UpdateContactGraph(contact.GroundStation, neighbor)
	}
}

// Math helper functions
func sin(x float64) float64 {
	// Normalize to [-π, π]
	for x > 3.14159265358979 {
		x -= 2 * 3.14159265358979
	}
	for x < -3.14159265358979 {
		x += 2 * 3.14159265358979
	}
	// Taylor series for sine
	x2 := x * x
	return x - x*x2/6 + x*x2*x2/120 - x*x2*x2*x2/5040
}

func cos(x float64) float64 {
	return sin(x + 3.14159265358979/2)
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	// Newton's method
	z := x / 2
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

func atan(x float64) float64 {
	// Taylor series for arctan, valid for |x| <= 1
	if x > 1 {
		return 3.14159265358979/2 - atan(1/x)
	}
	if x < -1 {
		return -3.14159265358979/2 - atan(1/x)
	}
	x2 := x * x
	return x - x*x2/3 + x*x2*x2/5 - x*x2*x2*x2/7
}

func atan2(y, x float64) float64 {
	if x > 0 {
		return atan(y / x)
	}
	if x < 0 && y >= 0 {
		return atan(y/x) + 3.14159265358979
	}
	if x < 0 && y < 0 {
		return atan(y/x) - 3.14159265358979
	}
	if x == 0 && y > 0 {
		return 3.14159265358979 / 2
	}
	if x == 0 && y < 0 {
		return -3.14159265358979 / 2
	}
	return 0
}
