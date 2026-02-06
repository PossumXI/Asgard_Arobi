// Package orbital provides orbital mechanics calculations for ASGARD.
// Supports mission planning at various altitudes including LEO, MEO, and GEO.
//
// Copyright 2026 Arobi. All Rights Reserved.
package orbital

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

// Physical constants
const (
	EarthRadius      = 6371000.0      // meters
	EarthMass        = 5.972e24       // kg
	GravitationalG   = 6.674e-11      // N⋅m²/kg²
	EarthMu          = 3.986004418e14 // m³/s² (standard gravitational parameter)
	SiderealDay      = 86164.0905     // seconds
	J2Coefficient    = 1.08263e-3     // Earth's J2 oblateness coefficient
	AtmosphereHeight = 100000.0       // Karman line in meters
)

// OrbitType classification
type OrbitType string

const (
	OrbitLEO           OrbitType = "LEO"           // Low Earth Orbit (160-2000 km)
	OrbitMEO           OrbitType = "MEO"           // Medium Earth Orbit (2000-35786 km)
	OrbitGEO           OrbitType = "GEO"           // Geostationary (35786 km)
	OrbitHEO           OrbitType = "HEO"           // Highly Elliptical
	OrbitSSO           OrbitType = "SSO"           // Sun-Synchronous
	OrbitSubOrbital    OrbitType = "SUB_ORBITAL"   // Below orbital velocity
	OrbitAtmospheric   OrbitType = "ATMOSPHERIC"   // Within atmosphere
)

// OrbitalElements defines a Keplerian orbit
type OrbitalElements struct {
	SemiMajorAxis       float64   `json:"semi_major_axis"`        // a, meters
	Eccentricity        float64   `json:"eccentricity"`           // e, 0-1
	Inclination         float64   `json:"inclination"`            // i, radians
	RAAN                float64   `json:"raan"`                   // Ω, Right Ascension of Ascending Node, radians
	ArgumentOfPeriapsis float64   `json:"argument_of_periapsis"`  // ω, radians
	TrueAnomaly         float64   `json:"true_anomaly"`           // ν, radians
	Epoch               time.Time `json:"epoch"`
}

// StateVector represents position and velocity in inertial frame
type StateVector struct {
	Position  Vector3   `json:"position"`   // meters, ECI frame
	Velocity  Vector3   `json:"velocity"`   // m/s, ECI frame
	Timestamp time.Time `json:"timestamp"`
}

// Vector3 is a 3D vector
type Vector3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// GroundTrack point for visualization
type GroundTrack struct {
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Altitude  float64   `json:"altitude"`
	Timestamp time.Time `json:"timestamp"`
}

// OrbitalPredictor calculates orbital trajectories
type OrbitalPredictor struct {
	mu sync.RWMutex

	elements    OrbitalElements
	stateVector StateVector
	groundTrack []GroundTrack

	// Propagation settings
	stepSize    time.Duration
	maxDuration time.Duration

	// Callbacks
	onStateUpdate func(StateVector)
	onGroundTrack func(GroundTrack)

	running bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewOrbitalPredictor creates a new orbital predictor
func NewOrbitalPredictor() *OrbitalPredictor {
	return &OrbitalPredictor{
		groundTrack: make([]GroundTrack, 0),
		stepSize:    1 * time.Second,
		maxDuration: 24 * time.Hour,
		stopCh:      make(chan struct{}),
	}
}

// SetOrbit sets initial orbital elements
func (op *OrbitalPredictor) SetOrbit(elements OrbitalElements) {
	op.mu.Lock()
	defer op.mu.Unlock()
	op.elements = elements
	op.stateVector = op.elementsToState(elements)
}

// GetOrbitType returns the type of current orbit
func (op *OrbitalPredictor) GetOrbitType() OrbitType {
	op.mu.RLock()
	defer op.mu.RUnlock()

	altitude := op.elements.SemiMajorAxis - EarthRadius

	if altitude < 0 {
		return OrbitAtmospheric
	}
	if altitude < AtmosphereHeight {
		return OrbitSubOrbital
	}
	if altitude < 2000000 {
		return OrbitLEO
	}
	if altitude < 35786000 {
		return OrbitMEO
	}
	if math.Abs(altitude-35786000) < 1000 && op.elements.Eccentricity < 0.01 {
		return OrbitGEO
	}
	if op.elements.Eccentricity > 0.5 {
		return OrbitHEO
	}
	return OrbitMEO
}

// GetOrbitalPeriod returns orbital period in seconds
func (op *OrbitalPredictor) GetOrbitalPeriod() float64 {
	op.mu.RLock()
	defer op.mu.RUnlock()
	return 2 * math.Pi * math.Sqrt(math.Pow(op.elements.SemiMajorAxis, 3)/EarthMu)
}

// GetOrbitalVelocity returns current orbital velocity
func (op *OrbitalPredictor) GetOrbitalVelocity() float64 {
	op.mu.RLock()
	defer op.mu.RUnlock()
	return op.stateVector.Velocity.Magnitude()
}

// GetAltitude returns current altitude above Earth surface
func (op *OrbitalPredictor) GetAltitude() float64 {
	op.mu.RLock()
	defer op.mu.RUnlock()
	return op.stateVector.Position.Magnitude() - EarthRadius
}

// Start begins orbital propagation
func (op *OrbitalPredictor) Start(ctx context.Context) error {
	op.mu.Lock()
	if op.running {
		op.mu.Unlock()
		return fmt.Errorf("already running")
	}
	op.running = true
	op.mu.Unlock()

	op.wg.Add(1)
	go op.propagationLoop(ctx)

	log.Println("[Orbital] Predictor started")
	return nil
}

// Stop halts orbital propagation
func (op *OrbitalPredictor) Stop() {
	op.mu.Lock()
	if !op.running {
		op.mu.Unlock()
		return
	}
	op.running = false
	op.mu.Unlock()

	close(op.stopCh)
	op.wg.Wait()
	log.Println("[Orbital] Predictor stopped")
}

// propagationLoop continuously updates orbital state
func (op *OrbitalPredictor) propagationLoop(ctx context.Context) {
	defer op.wg.Done()

	ticker := time.NewTicker(op.stepSize)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-op.stopCh:
			return
		case <-ticker.C:
			op.propagateStep()
		}
	}
}

// propagateStep advances orbital state by one time step
func (op *OrbitalPredictor) propagateStep() {
	op.mu.Lock()
	defer op.mu.Unlock()

	dt := op.stepSize.Seconds()

	// Simple two-body propagation with J2 perturbation
	state := op.stateVector
	r := state.Position.Magnitude()

	// Gravitational acceleration (two-body)
	acc := Vector3{
		X: -EarthMu * state.Position.X / math.Pow(r, 3),
		Y: -EarthMu * state.Position.Y / math.Pow(r, 3),
		Z: -EarthMu * state.Position.Z / math.Pow(r, 3),
	}

	// J2 perturbation
	z2 := state.Position.Z * state.Position.Z
	r2 := r * r
	factor := 1.5 * J2Coefficient * EarthMu * EarthRadius * EarthRadius / math.Pow(r, 5)

	acc.X += factor * state.Position.X * (5*z2/r2 - 1)
	acc.Y += factor * state.Position.Y * (5*z2/r2 - 1)
	acc.Z += factor * state.Position.Z * (5*z2/r2 - 3)

	// Update velocity (Euler integration for demo)
	state.Velocity.X += acc.X * dt
	state.Velocity.Y += acc.Y * dt
	state.Velocity.Z += acc.Z * dt

	// Update position
	state.Position.X += state.Velocity.X * dt
	state.Position.Y += state.Velocity.Y * dt
	state.Position.Z += state.Velocity.Z * dt

	state.Timestamp = time.Now()
	op.stateVector = state

	// Update ground track
	track := op.calculateGroundTrack(state)
	op.groundTrack = append(op.groundTrack, track)

	// Keep only last 1000 points
	if len(op.groundTrack) > 1000 {
		op.groundTrack = op.groundTrack[len(op.groundTrack)-1000:]
	}

	// Callbacks
	if op.onStateUpdate != nil {
		go op.onStateUpdate(state)
	}
	if op.onGroundTrack != nil {
		go op.onGroundTrack(track)
	}
}

// calculateGroundTrack converts ECI position to ground coordinates
func (op *OrbitalPredictor) calculateGroundTrack(state StateVector) GroundTrack {
	r := state.Position.Magnitude()

	// Calculate latitude (declination)
	lat := math.Asin(state.Position.Z / r)

	// Calculate longitude (right ascension minus Earth rotation)
	lon := math.Atan2(state.Position.Y, state.Position.X)

	// Account for Earth's rotation (simplified)
	gmst := op.getGMST(state.Timestamp)
	lon = lon - gmst

	// Normalize longitude to -π to π
	for lon > math.Pi {
		lon -= 2 * math.Pi
	}
	for lon < -math.Pi {
		lon += 2 * math.Pi
	}

	return GroundTrack{
		Latitude:  lat * 180 / math.Pi,
		Longitude: lon * 180 / math.Pi,
		Altitude:  r - EarthRadius,
		Timestamp: state.Timestamp,
	}
}

// getGMST returns Greenwich Mean Sidereal Time in radians
func (op *OrbitalPredictor) getGMST(t time.Time) float64 {
	// J2000.0 epoch
	j2000 := time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)
	daysSinceJ2000 := t.Sub(j2000).Hours() / 24.0

	// Simplified GMST calculation
	gmst := 4.894961212823756 + 6.300388098984957*daysSinceJ2000
	return math.Mod(gmst, 2*math.Pi)
}

// elementsToState converts Keplerian elements to state vector
func (op *OrbitalPredictor) elementsToState(el OrbitalElements) StateVector {
	a := el.SemiMajorAxis
	e := el.Eccentricity
	i := el.Inclination
	omega := el.RAAN
	w := el.ArgumentOfPeriapsis
	nu := el.TrueAnomaly

	// Distance from focus
	r := a * (1 - e*e) / (1 + e*math.Cos(nu))

	// Position in orbital plane
	xOrb := r * math.Cos(nu)
	yOrb := r * math.Sin(nu)

	// Velocity in orbital plane
	p := a * (1 - e*e)
	h := math.Sqrt(EarthMu * p)
	vxOrb := -EarthMu / h * math.Sin(nu)
	vyOrb := EarthMu / h * (e + math.Cos(nu))

	// Rotation matrices
	cosO := math.Cos(omega)
	sinO := math.Sin(omega)
	cosI := math.Cos(i)
	sinI := math.Sin(i)
	cosW := math.Cos(w)
	sinW := math.Sin(w)

	// Transform to ECI frame
	pos := Vector3{
		X: (cosO*cosW-sinO*sinW*cosI)*xOrb + (-cosO*sinW-sinO*cosW*cosI)*yOrb,
		Y: (sinO*cosW+cosO*sinW*cosI)*xOrb + (-sinO*sinW+cosO*cosW*cosI)*yOrb,
		Z: sinI*sinW*xOrb + sinI*cosW*yOrb,
	}

	vel := Vector3{
		X: (cosO*cosW-sinO*sinW*cosI)*vxOrb + (-cosO*sinW-sinO*cosW*cosI)*vyOrb,
		Y: (sinO*cosW+cosO*sinW*cosI)*vxOrb + (-sinO*sinW+cosO*cosW*cosI)*vyOrb,
		Z: sinI*sinW*vxOrb + sinI*cosW*vyOrb,
	}

	return StateVector{
		Position:  pos,
		Velocity:  vel,
		Timestamp: el.Epoch,
	}
}

// GetState returns current state vector
func (op *OrbitalPredictor) GetState() StateVector {
	op.mu.RLock()
	defer op.mu.RUnlock()
	return op.stateVector
}

// GetGroundTrack returns ground track history
func (op *OrbitalPredictor) GetGroundTrack() []GroundTrack {
	op.mu.RLock()
	defer op.mu.RUnlock()
	return append([]GroundTrack{}, op.groundTrack...)
}

// PredictPosition predicts position at future time
func (op *OrbitalPredictor) PredictPosition(futureTime time.Time) StateVector {
	op.mu.RLock()
	current := op.stateVector
	op.mu.RUnlock()

	dt := futureTime.Sub(current.Timestamp).Seconds()

	// Simplified prediction (Kepler's equation for circular/near-circular orbits)
	r := current.Position.Magnitude()
	n := math.Sqrt(EarthMu / math.Pow(r, 3)) // Mean motion

	// Rotate position by mean anomaly change
	dM := n * dt
	cos_dM := math.Cos(dM)
	sin_dM := math.Sin(dM)

	// Simple rotation (assumes circular orbit)
	return StateVector{
		Position: Vector3{
			X: current.Position.X*cos_dM - current.Position.Y*sin_dM,
			Y: current.Position.X*sin_dM + current.Position.Y*cos_dM,
			Z: current.Position.Z,
		},
		Velocity: Vector3{
			X: current.Velocity.X*cos_dM - current.Velocity.Y*sin_dM,
			Y: current.Velocity.X*sin_dM + current.Velocity.Y*cos_dM,
			Z: current.Velocity.Z,
		},
		Timestamp: futureTime,
	}
}

// CalculateTransfer computes Hohmann transfer parameters
func (op *OrbitalPredictor) CalculateTransfer(targetAltitude float64) TransferParameters {
	op.mu.RLock()
	currentAlt := op.GetAltitude()
	op.mu.RUnlock()

	r1 := EarthRadius + currentAlt
	r2 := EarthRadius + targetAltitude

	// Hohmann transfer
	a_transfer := (r1 + r2) / 2

	// Velocities
	v1_circular := math.Sqrt(EarthMu / r1)
	v2_circular := math.Sqrt(EarthMu / r2)
	v1_transfer := math.Sqrt(EarthMu * (2/r1 - 1/a_transfer))
	v2_transfer := math.Sqrt(EarthMu * (2/r2 - 1/a_transfer))

	// Delta-V requirements
	dv1 := v1_transfer - v1_circular
	dv2 := v2_circular - v2_transfer

	// Transfer time (half orbital period of transfer ellipse)
	transferTime := math.Pi * math.Sqrt(math.Pow(a_transfer, 3)/EarthMu)

	return TransferParameters{
		InitialAltitude: currentAlt,
		FinalAltitude:   targetAltitude,
		DeltaV1:         dv1,
		DeltaV2:         dv2,
		TotalDeltaV:     math.Abs(dv1) + math.Abs(dv2),
		TransferTime:    time.Duration(transferTime) * time.Second,
		TransferType:    "hohmann",
	}
}

// TransferParameters for orbital maneuvers
type TransferParameters struct {
	InitialAltitude float64       `json:"initial_altitude"`
	FinalAltitude   float64       `json:"final_altitude"`
	DeltaV1         float64       `json:"delta_v_1"`       // First burn, m/s
	DeltaV2         float64       `json:"delta_v_2"`       // Second burn, m/s
	TotalDeltaV     float64       `json:"total_delta_v"`   // Total, m/s
	TransferTime    time.Duration `json:"transfer_time"`
	TransferType    string        `json:"transfer_type"`
}

// OnStateUpdate sets callback for state updates
func (op *OrbitalPredictor) OnStateUpdate(cb func(StateVector)) {
	op.mu.Lock()
	defer op.mu.Unlock()
	op.onStateUpdate = cb
}

// OnGroundTrack sets callback for ground track updates
func (op *OrbitalPredictor) OnGroundTrack(cb func(GroundTrack)) {
	op.mu.Lock()
	defer op.mu.Unlock()
	op.onGroundTrack = cb
}

// GetStatistics returns orbital statistics
func (op *OrbitalPredictor) GetStatistics() map[string]interface{} {
	op.mu.RLock()
	defer op.mu.RUnlock()

	period := op.GetOrbitalPeriod()

	return map[string]interface{}{
		"orbit_type":           op.GetOrbitType(),
		"altitude_km":          op.GetAltitude() / 1000,
		"velocity_km_s":        op.GetOrbitalVelocity() / 1000,
		"period_minutes":       period / 60,
		"semi_major_axis_km":   op.elements.SemiMajorAxis / 1000,
		"eccentricity":         op.elements.Eccentricity,
		"inclination_degrees":  op.elements.Inclination * 180 / math.Pi,
		"ground_track_points":  len(op.groundTrack),
	}
}

// Vector3 helper methods
func (v Vector3) Magnitude() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

func (v Vector3) Normalize() Vector3 {
	mag := v.Magnitude()
	if mag == 0 {
		return Vector3{}
	}
	return Vector3{X: v.X / mag, Y: v.Y / mag, Z: v.Z / mag}
}

func (v Vector3) Add(other Vector3) Vector3 {
	return Vector3{X: v.X + other.X, Y: v.Y + other.Y, Z: v.Z + other.Z}
}

func (v Vector3) Sub(other Vector3) Vector3 {
	return Vector3{X: v.X - other.X, Y: v.Y - other.Y, Z: v.Z - other.Z}
}

func (v Vector3) Scale(s float64) Vector3 {
	return Vector3{X: v.X * s, Y: v.Y * s, Z: v.Z * s}
}

func (v Vector3) Dot(other Vector3) float64 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

func (v Vector3) Cross(other Vector3) Vector3 {
	return Vector3{
		X: v.Y*other.Z - v.Z*other.Y,
		Y: v.Z*other.X - v.X*other.Z,
		Z: v.X*other.Y - v.Y*other.X,
	}
}
