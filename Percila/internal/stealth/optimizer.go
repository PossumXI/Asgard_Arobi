package stealth

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Vector3D represents 3D coordinates
type Vector3D struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// Waypoint represents a navigation waypoint
type Waypoint struct {
	ID          string    `json:"id"`
	Position    Vector3D  `json:"position"`
	Velocity    Vector3D  `json:"velocity"`
	Timestamp   time.Time `json:"timestamp"`
	Stealth     bool      `json:"stealth"`
}

// Trajectory represents a flight path
type Trajectory struct {
	ID           string     `json:"id"`
	PayloadType  string     `json:"payloadType"`
	Waypoints    []Waypoint `json:"waypoints"`
	StealthScore float64    `json:"stealthScore"`
}

// RadarSite represents a radar detection system
type RadarSite struct {
	ID           string   `json:"id"`
	Position     Vector3D `json:"position"`
	FrequencyGHz float64  `json:"frequencyGHz"` // S-band: 2-4, X-band: 8-12, Ka-band: 26-40
	RangeKm      float64  `json:"rangeKm"`
	BeamWidth    float64  `json:"beamWidth"`    // degrees
	MinElevation float64  `json:"minElevation"` // degrees
	Active       bool     `json:"active"`
}

// SAMSite represents a Surface-to-Air Missile site
type SAMSite struct {
	ID           string   `json:"id"`
	Position     Vector3D `json:"position"`
	RangeKm      float64  `json:"rangeKm"`
	MaxAltitude  float64  `json:"maxAltitude"` // meters
	Active       bool     `json:"active"`
}

// ThermalModel represents thermal signature parameters
type ThermalModel struct {
	AmbientTemp     float64 `json:"ambientTemp"`     // celsius
	EngineTemp      float64 `json:"engineTemp"`      // celsius
	ExhaustTemp     float64 `json:"exhaustTemp"`     // celsius
	SkinTemp        float64 `json:"skinTemp"`        // celsius
	CoolingRate     float64 `json:"coolingRate"`     // per second
	EmissivityCoeff float64 `json:"emissivityCoeff"` // 0.0-1.0
}

// RCSProfile represents Radar Cross Section characteristics
type RCSProfile struct {
	FrontalRCS  float64 `json:"frontalRcs"`  // m² from front
	SideRCS     float64 `json:"sideRcs"`     // m² from side
	RearRCS     float64 `json:"rearRcs"`     // m² from rear
	TopRCS      float64 `json:"topRcs"`      // m² from above
	BottomRCS   float64 `json:"bottomRcs"`   // m² from below
}

// StealthConfig holds stealth optimization parameters
type StealthConfig struct {
	MaxDetectionProbability float64 `json:"maxDetectionProbability"` // 0.0-1.0
	MinTerrainClearance     float64 `json:"minTerrainClearance"`     // meters
	ThermalReduction        bool    `json:"thermalReduction"`
	RadarEvasion            bool    `json:"radarEvasion"`
	UseDecoys               bool    `json:"useDecoys"`
	NightOpsPreferred       bool    `json:"nightOpsPreferred"`
}

// StealthOptimizer minimizes detection probability
type StealthOptimizer struct {
	mu sync.RWMutex

	id              string
	config          StealthConfig
	radarSites      []RadarSite
	samSites        []SAMSite
	thermalModel    *ThermalModel
	rcsProfile      *RCSProfile
	terrainElevation [][]float64
	terrainOriginX  float64
	terrainOriginY  float64
	terrainCellSize float64
}

// NewStealthOptimizer creates a new stealth optimizer
func NewStealthOptimizer(config StealthConfig) *StealthOptimizer {
	return &StealthOptimizer{
		id:         uuid.New().String(),
		config:     config,
		radarSites: make([]RadarSite, 0),
		samSites:   make([]SAMSite, 0),
		thermalModel: &ThermalModel{
			AmbientTemp:     15.0,
			EngineTemp:      800.0,
			ExhaustTemp:     1200.0,
			SkinTemp:        50.0,
			CoolingRate:     0.05,
			EmissivityCoeff: 0.3,
		},
		rcsProfile: &RCSProfile{
			FrontalRCS: 0.5,
			SideRCS:    2.0,
			RearRCS:    1.0,
			TopRCS:     5.0,
			BottomRCS:  3.0,
		},
	}
}

// SetTerrain provides terrain data for terrain masking calculations
func (s *StealthOptimizer) SetTerrain(elevation [][]float64, originX, originY, cellSize float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.terrainElevation = elevation
	s.terrainOriginX = originX
	s.terrainOriginY = originY
	s.terrainCellSize = cellSize
}

// AddRadarSite adds a radar site to avoid
func (s *StealthOptimizer) AddRadarSite(site RadarSite) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.radarSites = append(s.radarSites, site)
}

// AddSAMSite adds a SAM site to avoid
func (s *StealthOptimizer) AddSAMSite(site SAMSite) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.samSites = append(s.samSites, site)
}

// SetRCSProfile sets the radar cross section profile
func (s *StealthOptimizer) SetRCSProfile(profile RCSProfile) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rcsProfile = &profile
}

// SetThermalModel sets the thermal signature model
func (s *StealthOptimizer) SetThermalModel(model ThermalModel) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.thermalModel = &model
}

// OptimizeTrajectory optimizes a trajectory for minimum detection
func (s *StealthOptimizer) OptimizeTrajectory(ctx context.Context, traj *Trajectory) (*Trajectory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	optimized := &Trajectory{
		ID:          uuid.New().String(),
		PayloadType: traj.PayloadType,
		Waypoints:   make([]Waypoint, 0),
	}

	for i, wp := range traj.Waypoints {
		optWP := s.optimizeWaypoint(wp, i > 0 && i < len(traj.Waypoints)-1)
		optimized.Waypoints = append(optimized.Waypoints, optWP)
	}

	// Calculate overall stealth score
	optimized.StealthScore = s.calculateTrajectoryStealthScore(optimized)

	return optimized, nil
}

// optimizeWaypoint optimizes a single waypoint for stealth
func (s *StealthOptimizer) optimizeWaypoint(wp Waypoint, canModify bool) Waypoint {
	optWP := Waypoint{
		ID:        wp.ID,
		Position:  wp.Position,
		Velocity:  wp.Velocity,
		Timestamp: wp.Timestamp,
		Stealth:   true,
	}

	if !canModify {
		return optWP
	}

	// Calculate radar exposure at current position
	radarExposure := s.calculateRadarExposure(wp.Position)

	// If high exposure, try to reduce altitude for terrain masking
	if radarExposure > s.config.MaxDetectionProbability {
		terrainElev := s.getTerrainElevation(wp.Position.X, wp.Position.Y)
		minAlt := terrainElev + s.config.MinTerrainClearance

		// Reduce altitude to minimum safe
		if wp.Position.Z > minAlt+100 {
			optWP.Position.Z = minAlt + 50 // Just above terrain
		}
	}

	// Reduce speed for thermal signature reduction
	if s.config.ThermalReduction {
		speed := magnitude(wp.Velocity)
		if speed > 200 { // m/s
			scale := 200.0 / speed
			optWP.Velocity = scaleVector(wp.Velocity, scale)
		}
	}

	return optWP
}

// CalculateRadarCrossSection estimates RCS for given position and heading
func (s *StealthOptimizer) CalculateRadarCrossSection(position Vector3D, heading float64, radarPosition Vector3D) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Calculate aspect angle relative to radar
	dx := position.X - radarPosition.X
	dy := position.Y - radarPosition.Y
	aspectAngle := math.Atan2(dy, dx) - heading

	// Normalize to [0, 2π]
	for aspectAngle < 0 {
		aspectAngle += 2 * math.Pi
	}
	for aspectAngle >= 2*math.Pi {
		aspectAngle -= 2 * math.Pi
	}

	// Interpolate RCS based on aspect angle
	// 0 = front, π/2 = side, π = rear
	var rcs float64

	if aspectAngle < math.Pi/4 || aspectAngle > 7*math.Pi/4 {
		// Frontal aspect
		rcs = s.rcsProfile.FrontalRCS
	} else if aspectAngle < 3*math.Pi/4 {
		// Right side
		rcs = s.rcsProfile.SideRCS
	} else if aspectAngle < 5*math.Pi/4 {
		// Rear
		rcs = s.rcsProfile.RearRCS
	} else {
		// Left side
		rcs = s.rcsProfile.SideRCS
	}

	// Altitude affects atmospheric attenuation
	altitudeFactor := math.Exp(-position.Z / 8000.0)

	// Speed affects Doppler signature
	speed := 0.0 // Would need velocity to calculate
	dopplerFactor := 1.0 + (speed / 340.0)

	return rcs * altitudeFactor * dopplerFactor
}

// CalculateThermalSignature estimates IR detectability
func (s *StealthOptimizer) CalculateThermalSignature(position Vector3D, velocity Vector3D, engineThrottle float64) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	speed := magnitude(velocity)

	// Friction heating (aerodynamic)
	frictionTemp := s.thermalModel.AmbientTemp + (speed*speed*0.001)

	// Engine heat contribution (scaled by throttle)
	engineContribution := s.thermalModel.EngineTemp * engineThrottle

	// Exhaust plume (scaled by throttle)
	exhaustContribution := s.thermalModel.ExhaustTemp * engineThrottle * 0.3

	// Altitude cooling (higher = colder)
	coolingFactor := 1.0 + (position.Z / 10000.0)

	// Calculate total temperature
	totalTemp := (frictionTemp + engineContribution + exhaustContribution) / coolingFactor

	// Stefan-Boltzmann: radiance ∝ T⁴
	signature := s.thermalModel.EmissivityCoeff * math.Pow(totalTemp/100.0, 4)

	return signature
}

// CalculateRadarDetectionProbability calculates probability of detection by radar
func (s *StealthOptimizer) CalculateRadarDetectionProbability(position Vector3D, heading float64, radar RadarSite) float64 {
	if !radar.Active {
		return 0.0
	}

	// Calculate distance to radar
	dx := position.X - radar.Position.X
	dy := position.Y - radar.Position.Y
	dz := position.Z - radar.Position.Z
	distance := math.Sqrt(dx*dx + dy*dy + dz*dz)

	// Check if within radar range
	radarRangeM := radar.RangeKm * 1000
	if distance > radarRangeM {
		return 0.0
	}

	// Check elevation angle
	elevationAngle := math.Atan2(dz, math.Sqrt(dx*dx+dy*dy)) * 180 / math.Pi
	if elevationAngle < radar.MinElevation {
		return 0.0 // Below radar horizon
	}

	// Check terrain masking
	if s.isTerrainMasked(position, radar.Position) {
		return 0.0
	}

	// Calculate RCS
	rcs := s.CalculateRadarCrossSection(position, heading, radar.Position)

	// Radar equation (simplified)
	// Detection ∝ RCS / distance⁴
	detectionFactor := rcs / math.Pow(distance/1000, 4)

	// Frequency affects detection (higher freq = more sensitive but shorter range)
	freqFactor := 1.0 + (radar.FrequencyGHz - 10) * 0.05

	probability := detectionFactor * freqFactor

	// Clamp to [0, 1]
	return math.Max(0.0, math.Min(1.0, probability))
}

// calculateRadarExposure calculates combined radar detection probability
func (s *StealthOptimizer) calculateRadarExposure(position Vector3D) float64 {
	maxProb := 0.0

	for _, radar := range s.radarSites {
		prob := s.CalculateRadarDetectionProbability(position, 0, radar)
		if prob > maxProb {
			maxProb = prob
		}
	}

	return maxProb
}

// CalculateSAMThreat calculates SAM engagement probability
func (s *StealthOptimizer) CalculateSAMThreat(position Vector3D, velocity Vector3D) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	maxThreat := 0.0

	for _, sam := range s.samSites {
		if !sam.Active {
			continue
		}

		// Check altitude constraint
		if position.Z > sam.MaxAltitude {
			continue
		}

		// Calculate distance
		dx := position.X - sam.Position.X
		dy := position.Y - sam.Position.Y
		dz := position.Z - sam.Position.Z
		distance := math.Sqrt(dx*dx + dy*dy + dz*dz)

		samRangeM := sam.RangeKm * 1000
		if distance > samRangeM {
			continue
		}

		// Threat increases as we get closer
		threat := 1.0 - (distance / samRangeM)

		// Speed affects evasion probability
		speed := magnitude(velocity)
		evasionFactor := speed / 1000.0 // Higher speed = better evasion
		threat *= (1.0 - math.Min(0.5, evasionFactor))

		if threat > maxThreat {
			maxThreat = threat
		}
	}

	return maxThreat
}

// OptimizeTerrainMasking adjusts trajectory to use terrain features
func (s *StealthOptimizer) OptimizeTerrainMasking(traj *Trajectory) *Trajectory {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.terrainElevation == nil {
		return traj
	}

	optimized := &Trajectory{
		ID:          traj.ID + "_terrain",
		PayloadType: traj.PayloadType,
		Waypoints:   make([]Waypoint, len(traj.Waypoints)),
	}

	copy(optimized.Waypoints, traj.Waypoints)

	for i := range optimized.Waypoints {
		if i == 0 || i == len(optimized.Waypoints)-1 {
			continue // Don't modify start/end
		}

		wp := &optimized.Waypoints[i]

		// Get terrain elevation
		terrainElev := s.getTerrainElevation(wp.Position.X, wp.Position.Y)

		// Nap-of-earth flying
		targetAlt := terrainElev + s.config.MinTerrainClearance
		wp.Position.Z = targetAlt
		wp.Stealth = true
	}

	return optimized
}

// GenerateDecoyPath creates a false trajectory for deception
func (s *StealthOptimizer) GenerateDecoyPath(realTraj *Trajectory, offset Vector3D) *Trajectory {
	s.mu.RLock()
	defer s.mu.RUnlock()

	decoy := &Trajectory{
		ID:          realTraj.ID + "_decoy",
		PayloadType: realTraj.PayloadType,
		Waypoints:   make([]Waypoint, len(realTraj.Waypoints)),
	}

	for i, wp := range realTraj.Waypoints {
		decoy.Waypoints[i] = Waypoint{
			ID: uuid.New().String(),
			Position: Vector3D{
				X: wp.Position.X + offset.X,
				Y: wp.Position.Y + offset.Y,
				Z: wp.Position.Z + offset.Z,
			},
			Velocity:  wp.Velocity,
			Timestamp: wp.Timestamp,
			Stealth:   false, // Decoy is visible
		}
	}

	return decoy
}

// GenerateEvasionManeuver creates an evasion path around a threat
func (s *StealthOptimizer) GenerateEvasionManeuver(currentPos Vector3D, currentVel Vector3D, threatPos Vector3D, threatRadius float64) []Waypoint {
	evasionPath := make([]Waypoint, 0)

	// Calculate direction away from threat
	threatDir := Vector3D{
		X: currentPos.X - threatPos.X,
		Y: currentPos.Y - threatPos.Y,
		Z: currentPos.Z - threatPos.Z,
	}
	threatDir = normalize(threatDir)

	// Generate perpendicular evasion direction
	perpDir := Vector3D{
		X: -threatDir.Y,
		Y: threatDir.X,
		Z: 0,
	}

	// Create evasion waypoints
	evadeDistance := threatRadius * 1.5

	// Initial break
	wp1 := Waypoint{
		ID: uuid.New().String(),
		Position: Vector3D{
			X: currentPos.X + perpDir.X*evadeDistance*0.5,
			Y: currentPos.Y + perpDir.Y*evadeDistance*0.5,
			Z: currentPos.Z - 100, // Drop altitude
		},
		Velocity:  scaleVector(perpDir, magnitude(currentVel)),
		Timestamp: time.Now().Add(5 * time.Second),
		Stealth:   true,
	}
	evasionPath = append(evasionPath, wp1)

	// Continue evasion
	wp2 := Waypoint{
		ID: uuid.New().String(),
		Position: Vector3D{
			X: wp1.Position.X + perpDir.X*evadeDistance,
			Y: wp1.Position.Y + perpDir.Y*evadeDistance,
			Z: wp1.Position.Z,
		},
		Velocity:  scaleVector(perpDir, magnitude(currentVel)),
		Timestamp: time.Now().Add(15 * time.Second),
		Stealth:   true,
	}
	evasionPath = append(evasionPath, wp2)

	return evasionPath
}

// CalculateBestApproachVector finds optimal approach direction
func (s *StealthOptimizer) CalculateBestApproachVector(targetPos Vector3D, threatPositions []Vector3D) Vector3D {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Test multiple approach angles
	bestAngle := 0.0
	bestScore := -1.0

	for angle := 0.0; angle < 2*math.Pi; angle += math.Pi / 18 { // 10 degree increments
		// Calculate approach path from this angle
		approachDir := Vector3D{
			X: math.Cos(angle),
			Y: math.Sin(angle),
			Z: -0.1, // Slight descent
		}

		// Score based on distance from threats
		score := 0.0
		testPoints := 10
		for i := 1; i <= testPoints; i++ {
			dist := float64(i) * 1000 // 1km steps
			testPos := Vector3D{
				X: targetPos.X - approachDir.X*dist,
				Y: targetPos.Y - approachDir.Y*dist,
				Z: targetPos.Z - approachDir.Z*dist,
			}

			minThreatDist := math.MaxFloat64
			for _, threatPos := range threatPositions {
				d := distance(testPos, threatPos)
				if d < minThreatDist {
					minThreatDist = d
				}
			}

			score += minThreatDist
		}

		if score > bestScore {
			bestScore = score
			bestAngle = angle
		}
	}

	return Vector3D{
		X: math.Cos(bestAngle),
		Y: math.Sin(bestAngle),
		Z: -0.1,
	}
}

// isTerrainMasked checks if terrain blocks line of sight
func (s *StealthOptimizer) isTerrainMasked(targetPos, observerPos Vector3D) bool {
	if s.terrainElevation == nil {
		return false
	}

	// Sample points along line of sight
	dx := targetPos.X - observerPos.X
	dy := targetPos.Y - observerPos.Y
	dz := targetPos.Z - observerPos.Z
	dist := math.Sqrt(dx*dx + dy*dy)

	steps := int(dist / 100) // 100m steps
	if steps < 2 {
		return false
	}

	for i := 1; i < steps; i++ {
		t := float64(i) / float64(steps)
		x := observerPos.X + dx*t
		y := observerPos.Y + dy*t
		losZ := observerPos.Z + dz*t

		terrainZ := s.getTerrainElevation(x, y)
		if terrainZ > losZ {
			return true // Terrain blocks LOS
		}
	}

	return false
}

// getTerrainElevation returns terrain elevation at given position
func (s *StealthOptimizer) getTerrainElevation(x, y float64) float64 {
	if s.terrainElevation == nil {
		return 0
	}

	gridX := int((x - s.terrainOriginX) / s.terrainCellSize)
	gridY := int((y - s.terrainOriginY) / s.terrainCellSize)

	if gridX < 0 || gridY < 0 {
		return 0
	}
	if gridY >= len(s.terrainElevation) {
		return 0
	}
	if gridX >= len(s.terrainElevation[gridY]) {
		return 0
	}

	return s.terrainElevation[gridY][gridX]
}

// calculateTrajectoryStealthScore calculates overall stealth score
func (s *StealthOptimizer) calculateTrajectoryStealthScore(traj *Trajectory) float64 {
	if len(traj.Waypoints) == 0 {
		return 0
	}

	totalScore := 0.0
	for _, wp := range traj.Waypoints {
		// Radar exposure (lower is better)
		radarExp := s.calculateRadarExposure(wp.Position)

		// SAM threat (lower is better)
		samThreat := s.CalculateSAMThreat(wp.Position, wp.Velocity)

		// Thermal signature (lower is better)
		thermal := s.CalculateThermalSignature(wp.Position, wp.Velocity, 0.7)
		thermalNorm := thermal / 100.0 // Normalize

		// Combine factors (stealth score: higher is better)
		wpScore := 1.0 - (radarExp*0.4 + samThreat*0.4 + thermalNorm*0.2)
		totalScore += wpScore
	}

	return totalScore / float64(len(traj.Waypoints))
}

// GetStealthReport generates a detailed stealth analysis report
func (s *StealthOptimizer) GetStealthReport(traj *Trajectory) *StealthReport {
	s.mu.RLock()
	defer s.mu.RUnlock()

	report := &StealthReport{
		TrajectoryID:      traj.ID,
		OverallScore:      s.calculateTrajectoryStealthScore(traj),
		WaypointAnalysis:  make([]WaypointStealthAnalysis, len(traj.Waypoints)),
		RecommendedChanges: make([]string, 0),
	}

	for i, wp := range traj.Waypoints {
		analysis := WaypointStealthAnalysis{
			WaypointID:        wp.ID,
			Position:          wp.Position,
			RadarExposure:     s.calculateRadarExposure(wp.Position),
			SAMThreat:         s.CalculateSAMThreat(wp.Position, wp.Velocity),
			ThermalSignature:  s.CalculateThermalSignature(wp.Position, wp.Velocity, 0.7),
			TerrainMasked:     false,
			RecommendedAlt:    wp.Position.Z,
		}

		// Check if terrain masking is possible
		terrainElev := s.getTerrainElevation(wp.Position.X, wp.Position.Y)
		if terrainElev > 0 {
			analysis.RecommendedAlt = terrainElev + s.config.MinTerrainClearance
			analysis.TerrainMasked = wp.Position.Z < terrainElev+200
		}

		report.WaypointAnalysis[i] = analysis
	}

	// Generate recommendations
	if report.OverallScore < 0.5 {
		report.RecommendedChanges = append(report.RecommendedChanges, "Consider terrain masking for waypoints with high radar exposure")
	}

	avgRadar := 0.0
	for _, wa := range report.WaypointAnalysis {
		avgRadar += wa.RadarExposure
	}
	avgRadar /= float64(len(report.WaypointAnalysis))
	if avgRadar > 0.3 {
		report.RecommendedChanges = append(report.RecommendedChanges, "Reduce altitude to minimize radar cross-section")
	}

	return report
}

// StealthReport contains stealth analysis results
type StealthReport struct {
	TrajectoryID       string                     `json:"trajectoryId"`
	OverallScore       float64                    `json:"overallScore"` // 0.0-1.0, higher is stealthier
	WaypointAnalysis   []WaypointStealthAnalysis  `json:"waypointAnalysis"`
	RecommendedChanges []string                   `json:"recommendedChanges"`
}

// WaypointStealthAnalysis contains per-waypoint stealth data
type WaypointStealthAnalysis struct {
	WaypointID       string   `json:"waypointId"`
	Position         Vector3D `json:"position"`
	RadarExposure    float64  `json:"radarExposure"`    // 0.0-1.0
	SAMThreat        float64  `json:"samThreat"`        // 0.0-1.0
	ThermalSignature float64  `json:"thermalSignature"` // arbitrary units
	TerrainMasked    bool     `json:"terrainMasked"`
	RecommendedAlt   float64  `json:"recommendedAlt"`   // meters
}

// Helper functions

func magnitude(v Vector3D) float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

func normalize(v Vector3D) Vector3D {
	mag := magnitude(v)
	if mag == 0 {
		return Vector3D{}
	}
	return Vector3D{X: v.X / mag, Y: v.Y / mag, Z: v.Z / mag}
}

func scaleVector(v Vector3D, s float64) Vector3D {
	return Vector3D{X: v.X * s, Y: v.Y * s, Z: v.Z * s}
}

func distance(a, b Vector3D) float64 {
	dx := b.X - a.X
	dy := b.Y - a.Y
	dz := b.Z - a.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// Magnitude is exported for use by other packages
func Magnitude(v Vector3D) float64 {
	return magnitude(v)
}
