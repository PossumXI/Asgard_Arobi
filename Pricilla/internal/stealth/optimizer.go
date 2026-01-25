package stealth

import (
	"fmt"
	"math"

	"github.com/asgard/pandora/Pricilla/internal/guidance"
)

// StealthOptimizer minimizes detection probability.
type StealthOptimizer struct {
	radarFrequencies []float64 // GHz
	thermalModel     *ThermalModel
}

type ThermalModel struct {
	AmbientTemp float64
	EngineTemp  float64
	CoolingRate float64
}

func NewStealthOptimizer() *StealthOptimizer {
	return &StealthOptimizer{
		radarFrequencies: []float64{3.0, 10.0, 35.0}, // S, X, Ka bands
		thermalModel: &ThermalModel{
			AmbientTemp: 15.0,
			EngineTemp:  800.0,
			CoolingRate: 0.05,
		},
	}
}

// OptimizeTrajectory applies stealth scoring and returns a new trajectory.
func (s *StealthOptimizer) OptimizeTrajectory(traj *guidance.Trajectory, mode guidance.StealthMode) (*guidance.Trajectory, error) {
	if traj == nil {
		return nil, fmt.Errorf("trajectory is nil")
	}

	optimized := &guidance.Trajectory{
		ID:             traj.ID + "_stealth",
		PayloadType:    traj.PayloadType,
		Waypoints:      make([]guidance.Waypoint, len(traj.Waypoints)),
		TotalDistance:  traj.TotalDistance,
		EstimatedTime:  traj.EstimatedTime,
		FuelRequired:   traj.FuelRequired,
		ThreatExposure: traj.ThreatExposure,
		Confidence:     traj.Confidence,
		CreatedAt:      traj.CreatedAt,
	}

	copy(optimized.Waypoints, traj.Waypoints)

	if len(optimized.Waypoints) == 0 {
		optimized.StealthScore = 1.0
		return optimized, nil
	}

	signatureSum := 0.0
	for _, wp := range optimized.Waypoints {
		signatureSum += s.CalculateRCS(wp, 0) + s.CalculateThermalSignature(wp)
	}

	avgSignature := signatureSum / float64(len(optimized.Waypoints))
	risk := avgSignature / (avgSignature + 1.0)

	modeWeight := map[guidance.StealthMode]float64{
		guidance.StealthModeNone:    0.1,
		guidance.StealthModeLow:     0.3,
		guidance.StealthModeMedium:  0.5,
		guidance.StealthModeHigh:    0.7,
		guidance.StealthModeMaximum: 0.9,
	}[mode]
	if modeWeight == 0 {
		modeWeight = 0.5
	}

	optimized.StealthScore = math.Max(0.0, math.Min(1.0, 1.0-(risk*modeWeight)))
	if optimized.ThreatExposure == 0 {
		optimized.ThreatExposure = math.Min(1.0, risk)
	}

	return optimized, nil
}

// CalculateRCS estimates RCS for given waypoint and heading.
func (s *StealthOptimizer) CalculateRCS(wp guidance.Waypoint, heading float64) float64 {
	return s.CalculateRadarCrossSection(wp, heading)
}

// CalculateRadarCrossSection estimates RCS for given trajectory.
func (s *StealthOptimizer) CalculateRadarCrossSection(wp guidance.Waypoint, heading float64) float64 {
	// Simplified RCS model
	baseRCS := 1.0 // square meters

	// Altitude affects atmospheric attenuation
	altitudeFactor := math.Exp(-wp.Position.Z / 8000.0)

	// Speed affects Doppler signature
	speedSq := guidance.Magnitude(wp.Velocity)
	speed := math.Sqrt(speedSq)
	dopplerFactor := 1.0 + (speed / 340.0) // Mach effect

	// Aspect angle (simplified)
	aspectFactor := 1.0 + math.Abs(math.Sin(heading))

	effectiveRCS := baseRCS * altitudeFactor * dopplerFactor * aspectFactor

	return effectiveRCS
}

// OptimizeTerrainMasking adjusts altitude to use terrain features.
func (s *StealthOptimizer) OptimizeTerrainMasking(traj *guidance.Trajectory, terrainMap [][]float64) *guidance.Trajectory {
	optimized := &guidance.Trajectory{
		ID:          traj.ID + "_stealth",
		PayloadType: traj.PayloadType,
		Waypoints:   make([]guidance.Waypoint, len(traj.Waypoints)),
	}

	copy(optimized.Waypoints, traj.Waypoints)

	for i := range optimized.Waypoints {
		wp := &optimized.Waypoints[i]

		// Get terrain elevation at this position
		terrainElev := getTerrainElevation(wp.Position.X, wp.Position.Y, terrainMap)

		// Fly low over valleys, higher over peaks (nap-of-earth)
		if terrainElev > 0 {
			wp.Position.Z = terrainElev + 100 // 100m clearance
		}
	}

	return optimized
}

// CalculateThermalSignature estimates IR detectability.
func (s *StealthOptimizer) CalculateThermalSignature(wp guidance.Waypoint) float64 {
	speedSq := guidance.Magnitude(wp.Velocity)
	speed := math.Sqrt(speedSq)

	// Friction heating
	frictionTemp := s.thermalModel.AmbientTemp + (speed*speed*0.001)

	// Engine heat (if powered flight)
	engineContribution := s.thermalModel.EngineTemp * 0.3

	// Altitude affects cooling
	coolingFactor := 1.0 + (wp.Position.Z / 10000.0)

	totalTemp := (frictionTemp + engineContribution) / coolingFactor

	// Convert to signature strength (simplified Stefan-Boltzmann)
	signature := math.Pow(totalTemp/100.0, 4)

	return signature
}

// GenerateDecoyPath creates false trajectory.
func (s *StealthOptimizer) GenerateDecoyPath(realTraj *guidance.Trajectory, offset guidance.Vector3D) *guidance.Trajectory {
	decoy := &guidance.Trajectory{
		ID:          realTraj.ID + "_decoy",
		PayloadType: realTraj.PayloadType,
		Waypoints:   make([]guidance.Waypoint, len(realTraj.Waypoints)),
	}

	for i, wp := range realTraj.Waypoints {
		decoy.Waypoints[i] = guidance.Waypoint{
			Position: guidance.Vector3D{
				X: wp.Position.X + offset.X,
				Y: wp.Position.Y + offset.Y,
				Z: wp.Position.Z + offset.Z,
			},
			Velocity:    wp.Velocity,
			Timestamp:   wp.Timestamp,
			Constraints: wp.Constraints,
		}
	}

	return decoy
}

func getTerrainElevation(x, y float64, terrainMap [][]float64) float64 {
	// Simplified terrain lookup
	if len(terrainMap) == 0 {
		return 0
	}

	// Convert coordinates to grid indices
	gridX := int(x/1000) % len(terrainMap)
	gridY := int(y/1000) % len(terrainMap[0])

	if gridX < 0 || gridY < 0 || gridX >= len(terrainMap) || gridY >= len(terrainMap[0]) {
		return 0
	}

	return terrainMap[gridX][gridY]
}
