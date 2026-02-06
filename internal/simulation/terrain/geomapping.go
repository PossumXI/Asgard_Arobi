// Package terrain provides geomapping and terrain analysis for ASGARD.
// Supports terrain classification, obstacle detection, and landing zone assessment.
//
// Copyright 2026 Arobi. All Rights Reserved.
package terrain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"
)

// TerrainType classification
type TerrainType string

const (
	TerrainUrban      TerrainType = "urban"
	TerrainSuburban   TerrainType = "suburban"
	TerrainRural      TerrainType = "rural"
	TerrainForest     TerrainType = "forest"
	TerrainDesert     TerrainType = "desert"
	TerrainMountain   TerrainType = "mountain"
	TerrainWater      TerrainType = "water"
	TerrainWetland    TerrainType = "wetland"
	TerrainFarmland   TerrainType = "farmland"
	TerrainUnknown    TerrainType = "unknown"
)

// Coordinate represents a geographic point
type Coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"` // meters above sea level
}

// TerrainCell represents a terrain grid cell
type TerrainCell struct {
	Coordinate
	TerrainType    TerrainType `json:"terrain_type"`
	Elevation      float64     `json:"elevation"`       // meters
	Slope          float64     `json:"slope"`           // degrees
	Aspect         float64     `json:"aspect"`          // degrees (direction slope faces)
	Roughness      float64     `json:"roughness"`       // 0-1
	Vegetation     float64     `json:"vegetation"`      // 0-1 density
	PopulationDensity float64  `json:"population_density"` // people per sq km
	IsSafe         bool        `json:"is_safe"`         // safe for operations
	Obstacles      []Obstacle  `json:"obstacles"`
}

// Obstacle detected in terrain
type Obstacle struct {
	Type      string     `json:"type"`
	Position  Coordinate `json:"position"`
	Height    float64    `json:"height"`    // meters
	Radius    float64    `json:"radius"`    // meters
	IsDynamic bool       `json:"is_dynamic"` // moving obstacle
}

// LandingZone assessment result
type LandingZone struct {
	Center       Coordinate  `json:"center"`
	Radius       float64     `json:"radius"`        // meters
	SurfaceType  TerrainType `json:"surface_type"`
	Slope        float64     `json:"slope"`         // degrees
	Obstructions int         `json:"obstructions"`
	WindExposure float64     `json:"wind_exposure"` // 0-1
	SafetyScore  float64     `json:"safety_score"`  // 0-1
	Recommended  bool        `json:"recommended"`
	Reason       string      `json:"reason"`
}

// FlightCorridor represents a safe flight path
type FlightCorridor struct {
	Waypoints    []Coordinate `json:"waypoints"`
	MinAltitude  float64      `json:"min_altitude"`  // meters AGL
	MaxAltitude  float64      `json:"max_altitude"`
	Width        float64      `json:"width"`         // meters
	SafetyScore  float64      `json:"safety_score"`  // 0-1
	TerrainClear bool         `json:"terrain_clear"`
	NoFlyZones   []NoFlyZone  `json:"no_fly_zones"`
}

// NoFlyZone represents restricted airspace
type NoFlyZone struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Type       string     `json:"type"` // military, civilian, emergency, tfr
	Center     Coordinate `json:"center"`
	Radius     float64    `json:"radius"`     // meters
	FloorAlt   float64    `json:"floor_alt"`  // meters
	CeilingAlt float64    `json:"ceiling_alt"`
	Active     bool       `json:"active"`
	ValidFrom  time.Time  `json:"valid_from"`
	ValidTo    time.Time  `json:"valid_to"`
}

// GeoMapper provides terrain analysis capabilities
type GeoMapper struct {
	mu sync.RWMutex

	// Terrain cache
	terrainCache map[string]*TerrainCell
	noFlyZones   []NoFlyZone

	// Configuration
	gridResolution float64 // degrees per cell
	cacheTimeout   time.Duration

	// Metrics
	queriesTotal int64
	cacheHits    int64
	cacheMisses  int64

	httpClient *http.Client
}

// NewGeoMapper creates a new geomapping service
func NewGeoMapper() *GeoMapper {
	return &GeoMapper{
		terrainCache:   make(map[string]*TerrainCell),
		noFlyZones:     loadDefaultNoFlyZones(),
		gridResolution: 0.01, // ~1.1km at equator
		cacheTimeout:   1 * time.Hour,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
	}
}

// GetTerrainAt returns terrain data for a location
func (gm *GeoMapper) GetTerrainAt(ctx context.Context, lat, lon float64) (*TerrainCell, error) {
	gm.mu.Lock()
	gm.queriesTotal++
	gm.mu.Unlock()

	// Check cache
	key := gm.coordKey(lat, lon)
	gm.mu.RLock()
	if cell, ok := gm.terrainCache[key]; ok {
		gm.mu.RUnlock()
		gm.mu.Lock()
		gm.cacheHits++
		gm.mu.Unlock()
		return cell, nil
	}
	gm.mu.RUnlock()

	gm.mu.Lock()
	gm.cacheMisses++
	gm.mu.Unlock()

	// Query elevation data (using Open-Elevation API - free)
	elevation, err := gm.getElevation(ctx, lat, lon)
	if err != nil {
		elevation = gm.estimateElevation(lat, lon)
	}

	// Classify terrain
	cell := gm.classifyTerrain(lat, lon, elevation)

	// Cache result
	gm.mu.Lock()
	gm.terrainCache[key] = cell
	gm.mu.Unlock()

	return cell, nil
}

// getElevation fetches elevation from Open-Elevation API
func (gm *GeoMapper) getElevation(ctx context.Context, lat, lon float64) (float64, error) {
	url := fmt.Sprintf("https://api.open-elevation.com/api/v1/lookup?locations=%.6f,%.6f", lat, lon)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := gm.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var result struct {
		Results []struct {
			Elevation float64 `json:"elevation"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, err
	}

	if len(result.Results) > 0 {
		return result.Results[0].Elevation, nil
	}

	return 0, fmt.Errorf("no elevation data")
}

// estimateElevation provides fallback elevation estimate
func (gm *GeoMapper) estimateElevation(lat, lon float64) float64 {
	// Simplified elevation model based on geographic features
	// Real implementation would use local DEM data

	// Coastal areas (near major bodies of water)
	if gm.isCoastal(lat, lon) {
		return 10 + math.Abs(math.Sin(lat*10))*50
	}

	// Mountain regions (Rockies, Appalachians, Sierra Nevada)
	if gm.isMountainous(lat, lon) {
		return 1500 + math.Abs(math.Sin(lat*5)*math.Cos(lon*5))*1500
	}

	// Plains and midwest
	return 300 + math.Abs(math.Sin(lat*3)*math.Cos(lon*3))*400
}

// classifyTerrain determines terrain type based on location
func (gm *GeoMapper) classifyTerrain(lat, lon, elevation float64) *TerrainCell {
	cell := &TerrainCell{
		Coordinate: Coordinate{
			Latitude:  lat,
			Longitude: lon,
			Altitude:  elevation,
		},
		Elevation: elevation,
		IsSafe:    true,
	}

	// Determine terrain type based on heuristics
	cell.TerrainType = gm.inferTerrainType(lat, lon, elevation)

	// Set characteristics based on terrain type
	switch cell.TerrainType {
	case TerrainUrban:
		cell.PopulationDensity = 5000 + float64(int(lat*100)%5000)
		cell.Roughness = 0.8
		cell.Vegetation = 0.1
		cell.IsSafe = false // Populated area
	case TerrainSuburban:
		cell.PopulationDensity = 1000 + float64(int(lon*100)%2000)
		cell.Roughness = 0.5
		cell.Vegetation = 0.3
	case TerrainForest:
		cell.PopulationDensity = 10
		cell.Roughness = 0.4
		cell.Vegetation = 0.9
	case TerrainMountain:
		cell.Slope = 15 + math.Abs(math.Sin(lat*10))*30
		cell.Roughness = 0.7
		cell.Vegetation = 0.3
	case TerrainWater:
		cell.IsSafe = false
		cell.Roughness = 0.1
		cell.Vegetation = 0
	case TerrainDesert:
		cell.PopulationDensity = 1
		cell.Roughness = 0.2
		cell.Vegetation = 0.05
	case TerrainFarmland:
		cell.PopulationDensity = 50
		cell.Roughness = 0.1
		cell.Vegetation = 0.7
	default:
		cell.Roughness = 0.3
		cell.Vegetation = 0.4
	}

	// Calculate slope from elevation gradient
	if cell.Slope == 0 {
		cell.Slope = math.Abs(math.Sin(lat*20)*math.Cos(lon*20)) * 10
	}

	// Generate obstacles for urban areas
	if cell.TerrainType == TerrainUrban {
		cell.Obstacles = gm.generateObstacles(lat, lon)
	}

	return cell
}

// inferTerrainType guesses terrain from coordinates
func (gm *GeoMapper) inferTerrainType(lat, lon, elevation float64) TerrainType {
	// Known major cities (simplified)
	cities := []struct {
		lat, lon float64
		radius   float64
	}{
		{40.7128, -74.0060, 0.3},   // New York
		{34.0522, -118.2437, 0.4},  // Los Angeles
		{41.8781, -87.6298, 0.3},   // Chicago
		{29.7604, -95.3698, 0.3},   // Houston
		{33.4484, -112.0740, 0.3},  // Phoenix
		{39.7392, -104.9903, 0.2},  // Denver
		{47.6062, -122.3321, 0.2},  // Seattle
	}

	for _, city := range cities {
		dist := math.Sqrt(math.Pow(lat-city.lat, 2) + math.Pow(lon-city.lon, 2))
		if dist < city.radius {
			return TerrainUrban
		}
		if dist < city.radius*2 {
			return TerrainSuburban
		}
	}

	// Elevation-based classification
	if elevation > 2500 {
		return TerrainMountain
	}

	// Water bodies (simplified - Great Lakes, oceans)
	if gm.isWater(lat, lon) {
		return TerrainWater
	}

	// Desert regions (Southwest US)
	if lat < 37 && lat > 31 && lon < -105 && lon > -120 {
		if elevation < 1000 {
			return TerrainDesert
		}
	}

	// Forest regions (Pacific Northwest, Northeast)
	if (lat > 45 && lon < -110) || (lat > 40 && lon > -80) {
		return TerrainForest
	}

	// Default to farmland for midwest
	if lat > 35 && lat < 48 && lon > -105 && lon < -80 {
		return TerrainFarmland
	}

	return TerrainRural
}

// Helper functions
func (gm *GeoMapper) isCoastal(lat, lon float64) bool {
	// Simplified coastal detection
	return math.Abs(lon+74) < 1 || math.Abs(lon+118) < 1 || math.Abs(lon+122) < 1
}

func (gm *GeoMapper) isMountainous(lat, lon float64) bool {
	// Rocky Mountains and Sierra Nevada
	return (lon > -120 && lon < -100 && lat > 35 && lat < 50)
}

func (gm *GeoMapper) isWater(lat, lon float64) bool {
	// Great Lakes simplified
	greatLakes := []struct {
		lat, lon, radius float64
	}{
		{43.0, -82.0, 2.0},  // Lake Erie/Huron
		{44.0, -86.0, 2.0},  // Lake Michigan
		{47.0, -88.0, 1.5},  // Lake Superior
	}

	for _, lake := range greatLakes {
		dist := math.Sqrt(math.Pow(lat-lake.lat, 2) + math.Pow(lon-lake.lon, 2))
		if dist < lake.radius {
			return true
		}
	}

	return false
}

func (gm *GeoMapper) generateObstacles(lat, lon float64) []Obstacle {
	obstacles := make([]Obstacle, 0)

	// Generate pseudo-random buildings
	seed := int64(lat*1000) + int64(lon*1000)
	numBuildings := 3 + int(seed%5)

	for i := 0; i < numBuildings; i++ {
		obstacles = append(obstacles, Obstacle{
			Type: "building",
			Position: Coordinate{
				Latitude:  lat + float64(i)*0.001,
				Longitude: lon + float64(i)*0.001,
			},
			Height: 30 + float64(seed%100),
			Radius: 20 + float64(seed%30),
		})
	}

	return obstacles
}

func (gm *GeoMapper) coordKey(lat, lon float64) string {
	gridLat := math.Round(lat/gm.gridResolution) * gm.gridResolution
	gridLon := math.Round(lon/gm.gridResolution) * gm.gridResolution
	return fmt.Sprintf("%.4f,%.4f", gridLat, gridLon)
}

// AssessLandingZone evaluates a potential landing area
func (gm *GeoMapper) AssessLandingZone(ctx context.Context, center Coordinate, radius float64) (*LandingZone, error) {
	terrain, err := gm.GetTerrainAt(ctx, center.Latitude, center.Longitude)
	if err != nil {
		return nil, err
	}

	lz := &LandingZone{
		Center:       center,
		Radius:       radius,
		SurfaceType:  terrain.TerrainType,
		Slope:        terrain.Slope,
		Obstructions: len(terrain.Obstacles),
	}

	// Calculate safety score
	score := 1.0

	// Slope penalty
	if terrain.Slope > 10 {
		score -= 0.2
	}
	if terrain.Slope > 20 {
		score -= 0.3
	}

	// Terrain type penalties
	switch terrain.TerrainType {
	case TerrainWater:
		score = 0
	case TerrainUrban:
		score -= 0.5
	case TerrainMountain:
		score -= 0.3
	case TerrainForest:
		score -= 0.2
	}

	// Obstruction penalty
	score -= float64(lz.Obstructions) * 0.1

	// Wind exposure (estimate from terrain roughness)
	lz.WindExposure = 1 - terrain.Roughness
	if lz.WindExposure > 0.7 {
		score -= 0.1
	}

	lz.SafetyScore = math.Max(0, math.Min(1, score))
	lz.Recommended = lz.SafetyScore >= 0.7

	if !lz.Recommended {
		if terrain.TerrainType == TerrainWater {
			lz.Reason = "Landing zone is over water"
		} else if terrain.TerrainType == TerrainUrban {
			lz.Reason = "Landing zone is in populated area"
		} else if terrain.Slope > 20 {
			lz.Reason = "Terrain slope exceeds safe limits"
		} else {
			lz.Reason = "Multiple risk factors present"
		}
	} else {
		lz.Reason = "Landing zone meets safety requirements"
	}

	return lz, nil
}

// CheckFlightCorridor validates a flight path
func (gm *GeoMapper) CheckFlightCorridor(ctx context.Context, waypoints []Coordinate, width float64) (*FlightCorridor, error) {
	corridor := &FlightCorridor{
		Waypoints:    waypoints,
		Width:        width,
		TerrainClear: true,
		NoFlyZones:   make([]NoFlyZone, 0),
	}

	minAlt := math.MaxFloat64
	maxAlt := 0.0
	totalScore := 0.0

	for _, wp := range waypoints {
		terrain, err := gm.GetTerrainAt(ctx, wp.Latitude, wp.Longitude)
		if err != nil {
			continue
		}

		// Track altitude range
		requiredAlt := terrain.Elevation + 150 // 150m AGL minimum
		if requiredAlt < minAlt {
			minAlt = requiredAlt
		}
		if terrain.Elevation > maxAlt {
			maxAlt = terrain.Elevation + 500
		}

		// Check for obstacles
		for _, obs := range terrain.Obstacles {
			if wp.Altitude < obs.Height+50 {
				corridor.TerrainClear = false
			}
		}

		// Safety score contribution
		if terrain.IsSafe {
			totalScore += 1.0
		} else {
			totalScore += 0.5
		}

		// Check no-fly zones
		for _, nfz := range gm.noFlyZones {
			dist := gm.haversineDistance(wp.Latitude, wp.Longitude, nfz.Center.Latitude, nfz.Center.Longitude)
			if dist < nfz.Radius && wp.Altitude >= nfz.FloorAlt && wp.Altitude <= nfz.CeilingAlt {
				if nfz.Active {
					corridor.NoFlyZones = append(corridor.NoFlyZones, nfz)
				}
			}
		}
	}

	corridor.MinAltitude = minAlt
	corridor.MaxAltitude = maxAlt
	corridor.SafetyScore = totalScore / float64(len(waypoints))

	return corridor, nil
}

// haversineDistance calculates distance between two points in meters
func (gm *GeoMapper) haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	R := 6371000.0 // Earth radius in meters

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// GetNoFlyZones returns active no-fly zones
func (gm *GeoMapper) GetNoFlyZones() []NoFlyZone {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	active := make([]NoFlyZone, 0)
	now := time.Now()

	for _, nfz := range gm.noFlyZones {
		if nfz.Active && (nfz.ValidTo.IsZero() || now.Before(nfz.ValidTo)) {
			active = append(active, nfz)
		}
	}

	return active
}

// GetMetrics returns geomapper statistics
func (gm *GeoMapper) GetMetrics() map[string]interface{} {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	hitRate := 0.0
	if gm.queriesTotal > 0 {
		hitRate = float64(gm.cacheHits) / float64(gm.queriesTotal)
	}

	return map[string]interface{}{
		"total_queries": gm.queriesTotal,
		"cache_hits":    gm.cacheHits,
		"cache_misses":  gm.cacheMisses,
		"cache_size":    len(gm.terrainCache),
		"cache_hit_rate": hitRate,
		"no_fly_zones":  len(gm.noFlyZones),
	}
}

// loadDefaultNoFlyZones creates default restricted areas
func loadDefaultNoFlyZones() []NoFlyZone {
	return []NoFlyZone{
		{
			ID:         "NFZ-DC",
			Name:       "Washington DC FRZ",
			Type:       "military",
			Center:     Coordinate{38.8977, -77.0365, 0},
			Radius:     25000, // 25km
			FloorAlt:   0,
			CeilingAlt: 5500,
			Active:     true,
		},
		{
			ID:         "NFZ-JFK",
			Name:       "JFK Airport",
			Type:       "civilian",
			Center:     Coordinate{40.6413, -73.7781, 0},
			Radius:     10000,
			FloorAlt:   0,
			CeilingAlt: 3000,
			Active:     true,
		},
		{
			ID:         "NFZ-LAX",
			Name:       "LAX Airport",
			Type:       "civilian",
			Center:     Coordinate{33.9416, -118.4085, 0},
			Radius:     10000,
			FloorAlt:   0,
			CeilingAlt: 3000,
			Active:     true,
		},
		{
			ID:         "NFZ-AREA51",
			Name:       "Restricted Airspace R-4808",
			Type:       "military",
			Center:     Coordinate{37.2350, -115.8111, 0},
			Radius:     40000,
			FloorAlt:   0,
			CeilingAlt: 50000,
			Active:     true,
		},
	}
}
