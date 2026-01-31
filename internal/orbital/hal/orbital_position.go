// Package hal provides hardware abstraction for orbital systems.
package hal

import (
	"context"
	"sync"
	"time"

	"github.com/asgard/pandora/internal/platform/satellite"
)

// OrbitalPositionProvider defines the interface for satellite position data.
type OrbitalPositionProvider interface {
	// GetPosition returns current latitude, longitude (degrees), and altitude (km).
	GetPosition() (lat, lon, alt float64, err error)
	// GetTime returns the current UTC time from the position system.
	GetTime() (time.Time, error)
	// GetVelocity returns velocity components in km/s.
	GetVelocity() (vx, vy, vz float64, err error)
}

// RealOrbitalPosition provides real orbital position from N2YO/TLE data.
// This replaces MockGPSController for production use.
type RealOrbitalPosition struct {
	noradID    int
	client     *satellite.Client
	propagator *satellite.Propagator
	tle        *satellite.TLE
	mu         sync.RWMutex
	lastUpdate time.Time
	updateTTL  time.Duration
}

// RealOrbitalConfig holds configuration for real orbital position tracking.
type RealOrbitalConfig struct {
	NoradID    int
	N2YOAPIKey string
	UpdateTTL  time.Duration // How often to refresh TLE data
}

// DefaultOrbitalConfig returns default configuration for ISS testing.
func DefaultOrbitalConfig() RealOrbitalConfig {
	return RealOrbitalConfig{
		NoradID:   satellite.NoradISS,
		UpdateTTL: 1 * time.Hour,
	}
}

// NewRealOrbitalPosition creates a real orbital position provider.
func NewRealOrbitalPosition(cfg RealOrbitalConfig) (*RealOrbitalPosition, error) {
	clientCfg := satellite.DefaultConfig()
	clientCfg.N2YOAPIKey = cfg.N2YOAPIKey

	client := satellite.NewClient(clientCfg)

	// Fetch initial TLE
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tle, err := client.GetTLE(ctx, cfg.NoradID)
	if err != nil {
		return nil, err
	}

	propagator, err := satellite.NewPropagator(tle)
	if err != nil {
		return nil, err
	}

	return &RealOrbitalPosition{
		noradID:    cfg.NoradID,
		client:     client,
		propagator: propagator,
		tle:        tle,
		lastUpdate: time.Now(),
		updateTTL:  cfg.UpdateTTL,
	}, nil
}

// GetPosition returns the current orbital position.
func (r *RealOrbitalPosition) GetPosition() (lat, lon, alt float64, err error) {
	r.mu.RLock()
	propagator := r.propagator
	r.mu.RUnlock()

	// Check if TLE needs refresh
	if time.Since(r.lastUpdate) > r.updateTTL {
		go r.refreshTLE()
	}

	lat, lon, alt = propagator.Propagate(time.Now().UTC())
	return lat, lon, alt, nil
}

// GetTime returns current UTC time.
func (r *RealOrbitalPosition) GetTime() (time.Time, error) {
	return time.Now().UTC(), nil
}

// GetVelocity returns approximate orbital velocity.
// For LEO satellites, orbital velocity is approximately 7.8 km/s.
func (r *RealOrbitalPosition) GetVelocity() (vx, vy, vz float64, err error) {
	// Compute velocity from position changes
	now := time.Now().UTC()

	r.mu.RLock()
	propagator := r.propagator
	r.mu.RUnlock()

	lat1, lon1, _ := propagator.Propagate(now)
	lat2, lon2, _ := propagator.Propagate(now.Add(time.Second))

	// Approximate velocity in degrees/sec, convert to km/s
	// 1 degree latitude ≈ 111 km
	// 1 degree longitude ≈ 111 km * cos(lat)
	dLat := (lat2 - lat1) * 111.0
	dLon := (lon2 - lon1) * 111.0 * cosine(lat1)

	// Ground velocity (simplified)
	vx = dLon // km/s east-west
	vy = dLat // km/s north-south
	vz = 0.0  // Altitude change is small

	return vx, vy, vz, nil
}

// refreshTLE updates the TLE data in the background.
func (r *RealOrbitalPosition) refreshTLE() {
	r.mu.Lock()
	// Double-check after acquiring lock
	if time.Since(r.lastUpdate) < r.updateTTL {
		r.mu.Unlock()
		return
	}
	r.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tle, err := r.client.GetTLE(ctx, r.noradID)
	if err != nil {
		return // Keep using old TLE
	}

	propagator, err := satellite.NewPropagator(tle)
	if err != nil {
		return
	}

	r.mu.Lock()
	r.tle = tle
	r.propagator = propagator
	r.lastUpdate = time.Now()
	r.mu.Unlock()
}

// GetSatelliteName returns the satellite name from TLE.
func (r *RealOrbitalPosition) GetSatelliteName() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.tle != nil {
		return r.tle.Name
	}
	return "Unknown"
}

// GetNoradID returns the NORAD catalog ID.
func (r *RealOrbitalPosition) GetNoradID() int {
	return r.noradID
}

// cosine returns cosine of angle in degrees.
func cosine(degrees float64) float64 {
	const deg2rad = 0.017453292519943295
	return cos(degrees * deg2rad)
}

// cos is a simple cosine implementation.
func cos(radians float64) float64 {
	// Taylor series approximation for cosine
	// cos(x) = 1 - x^2/2! + x^4/4! - x^6/6! + ...
	x := radians
	// Normalize to [-π, π]
	for x > 3.14159265358979 {
		x -= 2 * 3.14159265358979
	}
	for x < -3.14159265358979 {
		x += 2 * 3.14159265358979
	}

	x2 := x * x
	return 1 - x2/2 + x2*x2/24 - x2*x2*x2/720 + x2*x2*x2*x2/40320
}
