// Package physics provides high-fidelity physics models for precision payload delivery.
// Includes gravitational, atmospheric, radiation, and orbital mechanics calculations.
package physics

import (
	"errors"
	"math"
	"sort"
	"time"
)

// ================================================================================
// PHYSICAL CONSTANTS
// ================================================================================

const (
	// Gravitational constants (GM, m³/s²)
	GM_Earth   = 3.986004418e14 // Earth standard gravitational parameter
	GM_Moon    = 4.9028695e12   // Moon
	GM_Sun     = 1.32712440018e20
	GM_Mars    = 4.282837e13
	GM_Jupiter = 1.26686534e17

	// Planetary radii (meters)
	R_Earth = 6.371e6  // Mean Earth radius
	R_Moon  = 1.7374e6 // Moon radius
	R_Mars  = 3.3895e6 // Mars radius

	// Earth-specific constants
	EarthJ2           = 1.08263e-3  // J2 oblateness coefficient
	EarthJ3           = -2.54e-6   // J3 coefficient
	EarthJ4           = -1.62e-6   // J4 coefficient
	EarthEquatorialR  = 6.378137e6 // Equatorial radius
	EarthPolarR       = 6.356752e6 // Polar radius
	EarthRotationRate = 7.2921159e-5 // rad/s

	// Atmospheric constants
	SeaLevelDensity    = 1.225        // kg/m³
	SeaLevelPressure   = 101325.0     // Pa
	ScaleHeight        = 8500.0       // meters (Earth atmosphere)
	KarmanLine         = 100000.0     // meters - edge of space
	SpeedOfSound       = 343.0        // m/s at sea level

	// Physical constants
	StefanBoltzmann = 5.67e-8     // W/(m²·K⁴)
	SolarConstant   = 1361.0      // W/m² at 1 AU
	SpeedOfLight    = 299792458.0 // m/s

	// Radiation constants
	VanAllenInnerBelt = 1000e3  // ~1000 km altitude start
	VanAllenOuterBelt = 13000e3 // ~13000 km altitude start
	VanAllenPeakFlux  = 1e8     // particles/cm²/s at peak
)

// ================================================================================
// VECTOR MATHEMATICS
// ================================================================================

// Vector3D represents a 3D vector
type Vector3D struct {
	X, Y, Z float64
}

// Add returns the sum of two vectors
func (v Vector3D) Add(other Vector3D) Vector3D {
	return Vector3D{v.X + other.X, v.Y + other.Y, v.Z + other.Z}
}

// Sub returns the difference of two vectors
func (v Vector3D) Sub(other Vector3D) Vector3D {
	return Vector3D{v.X - other.X, v.Y - other.Y, v.Z - other.Z}
}

// Scale returns the vector scaled by a scalar
func (v Vector3D) Scale(s float64) Vector3D {
	return Vector3D{v.X * s, v.Y * s, v.Z * s}
}

// Dot returns the dot product
func (v Vector3D) Dot(other Vector3D) float64 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

// Cross returns the cross product
func (v Vector3D) Cross(other Vector3D) Vector3D {
	return Vector3D{
		v.Y*other.Z - v.Z*other.Y,
		v.Z*other.X - v.X*other.Z,
		v.X*other.Y - v.Y*other.X,
	}
}

// Magnitude returns the length of the vector
func (v Vector3D) Magnitude() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

// Normalize returns a unit vector
func (v Vector3D) Normalize() Vector3D {
	mag := v.Magnitude()
	if mag == 0 {
		return Vector3D{}
	}
	return v.Scale(1 / mag)
}

// ================================================================================
// STATE REPRESENTATION
// ================================================================================

// OrbitalState represents the complete state of a body
type OrbitalState struct {
	Position     Vector3D  // ECI position (meters)
	Velocity     Vector3D  // ECI velocity (m/s)
	Mass         float64   // kg
	Time         time.Time
	Epoch        time.Time
	CoordFrame   string    // "ECI", "ECEF", "LCI", etc.
}

// KeplerianElements represents classical orbital elements
type KeplerianElements struct {
	SemiMajorAxis     float64 // a (meters)
	Eccentricity      float64 // e (dimensionless)
	Inclination       float64 // i (radians)
	RAAN              float64 // Ω (radians) - Right Ascension of Ascending Node
	ArgOfPeriapsis    float64 // ω (radians)
	TrueAnomaly       float64 // ν (radians)
	MeanAnomaly       float64 // M (radians)
	Epoch             time.Time
}

// ================================================================================
// GRAVITATIONAL ACCELERATION
// ================================================================================

// GravityModel defines the gravity model fidelity
type GravityModel int

const (
	GravityPointMass GravityModel = iota // Simple point mass
	GravityJ2                            // J2 oblateness
	GravityJ2J3J4                        // Full zonal harmonics
)

// CalculateGravity computes gravitational acceleration at a position
func CalculateGravity(position Vector3D, model GravityModel, centralBody string) Vector3D {
	var gm float64
	var r_eq float64
	var j2, j3, j4 float64

	switch centralBody {
	case "earth":
		gm = GM_Earth
		r_eq = EarthEquatorialR
		j2 = EarthJ2
		j3 = EarthJ3
		j4 = EarthJ4
	case "moon":
		gm = GM_Moon
		r_eq = R_Moon
		j2 = 0 // Simplified
	case "mars":
		gm = GM_Mars
		r_eq = R_Mars
		j2 = 1.96e-3 // Mars J2
	default:
		gm = GM_Earth
		r_eq = EarthEquatorialR
		j2 = EarthJ2
	}

	r := position.Magnitude()
	if r == 0 {
		return Vector3D{}
	}

	// Point mass acceleration
	r3 := r * r * r
	aPointMass := position.Scale(-gm / r3)

	if model == GravityPointMass {
		return aPointMass
	}

	// J2 perturbation (oblateness)
	z := position.Z
	r2 := r * r
	r5 := r2 * r3
	r7 := r5 * r2

	re2 := r_eq * r_eq
	z2 := z * z

	// J2 acceleration components
	factor_j2 := 1.5 * j2 * gm * re2 / r5
	j2_factor := 5.0 * z2 / r2
	
	ax_j2 := factor_j2 * position.X * (j2_factor - 1.0)
	ay_j2 := factor_j2 * position.Y * (j2_factor - 1.0)
	az_j2 := factor_j2 * position.Z * (j2_factor - 3.0)

	if model == GravityJ2 {
		return Vector3D{
			aPointMass.X + ax_j2,
			aPointMass.Y + ay_j2,
			aPointMass.Z + az_j2,
		}
	}

	// J3, J4 perturbations for high fidelity
	re3 := re2 * r_eq
	re4 := re3 * r_eq
	z3 := z2 * z
	z4 := z2 * z2

	// J3 terms
	factor_j3 := 2.5 * j3 * gm * re3 / r7
	j3_xy_factor := 7.0 * z3 / r2 - 3.0 * z
	ax_j3 := factor_j3 * position.X * j3_xy_factor
	ay_j3 := factor_j3 * position.Y * j3_xy_factor
	az_j3 := factor_j3 * (6.0*z2 - 7.0*z4/r2 - 0.6*r2)

	// J4 terms
	factor_j4 := 5.0 / 8.0 * j4 * gm * re4 / r7
	j4_factor := 3.0 - 42.0*z2/r2 + 63.0*z4/(r2*r2)
	ax_j4 := factor_j4 * position.X * j4_factor
	ay_j4 := factor_j4 * position.Y * j4_factor
	az_j4 := factor_j4 * position.Z * (15.0 - 70.0*z2/r2 + 63.0*z4/(r2*r2))

	return Vector3D{
		aPointMass.X + ax_j2 + ax_j3 + ax_j4,
		aPointMass.Y + ay_j2 + ay_j3 + ay_j4,
		aPointMass.Z + az_j2 + az_j3 + az_j4,
	}
}

// ================================================================================
// ATMOSPHERIC DRAG
// ================================================================================

// AtmosphereModel defines the atmosphere model
type AtmosphereModel int

const (
	AtmosphereExponential AtmosphereModel = iota // Simple exponential
	AtmosphereUS76                               // US Standard Atmosphere 1976
)

// AtmosphericProperties contains atmospheric state
type AtmosphericProperties struct {
	Density     float64 // kg/m³
	Pressure    float64 // Pa
	Temperature float64 // K
	SpeedOfSound float64 // m/s
	DynamicViscosity float64 // Pa·s
	MolarMass   float64 // kg/mol
}

// GetAtmosphericDensity returns atmospheric density at altitude
func GetAtmosphericDensity(altitude float64, model AtmosphereModel) float64 {
	if altitude < 0 {
		altitude = 0
	}
	if altitude > 1000e3 { // Above 1000 km, essentially vacuum
		return 1e-15
	}

	switch model {
	case AtmosphereUS76:
		return us76Density(altitude)
	default:
		// Exponential atmosphere
		return SeaLevelDensity * math.Exp(-altitude/ScaleHeight)
	}
}

// us76Density implements US Standard Atmosphere 1976
func us76Density(h float64) float64 {
	// Geopotential altitude layers (km)
	// Using piecewise model with lapse rates
	
	hKm := h / 1000.0 // Convert to km
	
	// Define atmospheric layers
	type layer struct {
		hBase   float64 // Base altitude (km)
		TBase   float64 // Base temperature (K)
		lapse   float64 // Lapse rate (K/km)
		rhoBase float64 // Base density (kg/m³)
	}

	layers := []layer{
		{0, 288.15, -6.5, 1.225},
		{11, 216.65, 0, 0.36391},
		{20, 216.65, 1.0, 0.08803},
		{32, 228.65, 2.8, 0.01322},
		{47, 270.65, 0, 0.00143},
		{51, 270.65, -2.8, 0.00086},
		{71, 214.65, -2.0, 0.000064},
		{86, 186.95, 0, 0.0000037},
	}

	// Find applicable layer
	var l layer
	for i := len(layers) - 1; i >= 0; i-- {
		if hKm >= layers[i].hBase {
			l = layers[i]
			break
		}
	}

	dh := hKm - l.hBase
	
	if l.lapse == 0 {
		// Isothermal layer
		return l.rhoBase * math.Exp(-34.1632*dh/l.TBase)
	}

	// Gradient layer
	T := l.TBase + l.lapse*dh
	return l.rhoBase * math.Pow(T/l.TBase, -1.0-34.1632/l.lapse)
}

// CalculateDrag computes drag acceleration
func CalculateDrag(velocity, position Vector3D, mass, area, cd float64, atmoModel AtmosphereModel) Vector3D {
	altitude := position.Magnitude() - R_Earth
	rho := GetAtmosphericDensity(altitude, atmoModel)

	// Relative velocity (accounting for Earth rotation for ECEF)
	vRel := velocity // Simplified - would subtract Earth rotation
	vMag := vRel.Magnitude()

	if vMag == 0 || mass == 0 {
		return Vector3D{}
	}

	// Drag force magnitude: F = 0.5 * ρ * v² * Cd * A
	dragMag := 0.5 * rho * vMag * vMag * cd * area

	// Drag acceleration (opposite to velocity)
	vUnit := vRel.Normalize()
	return vUnit.Scale(-dragMag / mass)
}

// CalculateDragCoefficient estimates Cd based on Mach number and shape
func CalculateDragCoefficient(velocity Vector3D, altitude float64, baseCD float64) float64 {
	// Speed of sound varies with altitude/temperature
	vMag := velocity.Magnitude()
	a := SpeedOfSound * math.Sqrt(math.Max(0.5, 1-altitude/(100*ScaleHeight)))
	mach := vMag / a

	// Transonic drag rise (wave drag)
	if mach < 0.8 {
		return baseCD // Subsonic
	} else if mach < 1.2 {
		// Transonic - significant drag rise
		return baseCD * (1.0 + 0.5*(mach-0.8)/0.4)
	} else if mach < 5.0 {
		// Supersonic - drag coefficient decreases
		return baseCD * (1.5 - 0.1*(mach-1.2))
	}
	// Hypersonic
	return baseCD * (1.1 + 0.05*math.Log(mach))
}

// ================================================================================
// SOLAR RADIATION PRESSURE
// ================================================================================

// CalculateSolarRadiationPressure computes SRP acceleration
func CalculateSolarRadiationPressure(position, sunPosition Vector3D, mass, area, reflectivity float64) Vector3D {
	if mass == 0 {
		return Vector3D{}
	}

	// Sun-spacecraft vector
	rSun := sunPosition.Sub(position)
	rSunMag := rSun.Magnitude()

	// AU in meters
	AU := 1.496e11

	// Check if in Earth's shadow (simplified umbra check)
	if isInShadow(position, sunPosition) {
		return Vector3D{}
	}

	// Solar radiation pressure: P = S/c where S = solar constant
	// Force = P * A * (1 + ρ) where ρ = reflectivity
	pressure := SolarConstant / SpeedOfLight
	srp_force := pressure * area * (1.0 + reflectivity)

	// Scale by distance from sun (inverse square)
	scaleFactor := (AU * AU) / (rSunMag * rSunMag)
	srp_force *= scaleFactor

	// Acceleration in sun-spacecraft direction
	sunDir := rSun.Normalize()
	return sunDir.Scale(-srp_force / mass)
}

// isInShadow checks if position is in Earth's shadow
func isInShadow(position, sunPosition Vector3D) bool {
	// Vector from spacecraft to sun
	toSun := sunPosition.Sub(position)
	toSunMag := toSun.Magnitude()
	toSunDir := toSun.Normalize()

	// Check if Earth is between spacecraft and sun
	posMag := position.Magnitude()
	
	// Project position onto sun line
	proj := position.Dot(toSunDir)
	
	if proj > 0 {
		return false // Sun is in front
	}

	// Distance from sun line
	perpDist := math.Sqrt(posMag*posMag - proj*proj)
	
	// Simple cylindrical shadow model
	return perpDist < R_Earth && math.Abs(proj) < toSunMag
}

// ================================================================================
// RADIATION ENVIRONMENT
// ================================================================================

// RadiationEnvironment contains radiation flux data
type RadiationEnvironment struct {
	TotalDose        float64 // rad/s
	ProtonFlux       float64 // particles/cm²/s
	ElectronFlux     float64 // particles/cm²/s
	SolarParticleFlux float64 // particles/cm²/s
	IsInVanAllen     bool
	VanAllenZone     string // "inner", "outer", "slot", "none"
}

// CalculateRadiationEnvironment estimates radiation at position
func CalculateRadiationEnvironment(position Vector3D, solarActivity float64) RadiationEnvironment {
	altitude := position.Magnitude() - R_Earth
	
	env := RadiationEnvironment{}

	// Determine Van Allen belt zone
	if altitude < 500e3 {
		env.VanAllenZone = "none"
		env.IsInVanAllen = false
		env.ProtonFlux = 100 * solarActivity // Minimal
		env.ElectronFlux = 1000 * solarActivity
	} else if altitude >= VanAllenInnerBelt && altitude < 6000e3 {
		// Inner belt - high proton flux
		env.VanAllenZone = "inner"
		env.IsInVanAllen = true
		// Peak at ~3000 km
		peakFactor := math.Exp(-math.Pow((altitude-3000e3)/(1500e3), 2))
		env.ProtonFlux = VanAllenPeakFlux * peakFactor * solarActivity
		env.ElectronFlux = VanAllenPeakFlux * 0.1 * peakFactor
	} else if altitude >= 6000e3 && altitude < 10000e3 {
		// Slot region - lower flux
		env.VanAllenZone = "slot"
		env.IsInVanAllen = false
		env.ProtonFlux = VanAllenPeakFlux * 0.01 * solarActivity
		env.ElectronFlux = VanAllenPeakFlux * 0.05 * solarActivity
	} else if altitude >= VanAllenOuterBelt && altitude < 60000e3 {
		// Outer belt - high electron flux
		env.VanAllenZone = "outer"
		env.IsInVanAllen = true
		// Peak at ~20000 km
		peakFactor := math.Exp(-math.Pow((altitude-20000e3)/(10000e3), 2))
		env.ElectronFlux = VanAllenPeakFlux * peakFactor * solarActivity
		env.ProtonFlux = VanAllenPeakFlux * 0.01 * peakFactor
	} else {
		env.VanAllenZone = "none"
		env.IsInVanAllen = false
		env.SolarParticleFlux = 1e4 * solarActivity // Interplanetary
	}

	// Calculate total dose (simplified model)
	env.TotalDose = (env.ProtonFlux*1e-6 + env.ElectronFlux*1e-8 + env.SolarParticleFlux*1e-7)

	return env
}

// ================================================================================
// ORBITAL PROPAGATION
// ================================================================================

// PropagatorConfig configures the orbital propagator
type PropagatorConfig struct {
	GravityModel    GravityModel
	AtmosphereModel AtmosphereModel
	IncludeDrag     bool
	IncludeSRP      bool
	IncludeJ2       bool
	IncludeThirdBody bool // Moon/Sun perturbations
	IntegrationStep time.Duration
}

// DefaultPropagatorConfig returns a high-fidelity default configuration
func DefaultPropagatorConfig() PropagatorConfig {
	return PropagatorConfig{
		GravityModel:    GravityJ2J3J4,
		AtmosphereModel: AtmosphereUS76,
		IncludeDrag:     true,
		IncludeSRP:      true,
		IncludeJ2:       true,
		IncludeThirdBody: true,
		IntegrationStep:  10 * time.Second,
	}
}

// Propagate advances the orbital state using RK4 integration
func Propagate(state OrbitalState, duration time.Duration, config PropagatorConfig, spacecraft SpacecraftParams) []OrbitalState {
	dt := config.IntegrationStep.Seconds()
	steps := int(duration.Seconds() / dt)
	
	states := make([]OrbitalState, 0, steps+1)
	states = append(states, state)

	current := state
	
	for i := 0; i < steps; i++ {
		// RK4 integration
		k1_v, k1_a := computeAcceleration(current, config, spacecraft)
		
		mid1 := OrbitalState{
			Position: current.Position.Add(k1_v.Scale(dt / 2)),
			Velocity: current.Velocity.Add(k1_a.Scale(dt / 2)),
			Mass:     current.Mass,
		}
		k2_v, k2_a := computeAcceleration(mid1, config, spacecraft)
		
		mid2 := OrbitalState{
			Position: current.Position.Add(k2_v.Scale(dt / 2)),
			Velocity: current.Velocity.Add(k2_a.Scale(dt / 2)),
			Mass:     current.Mass,
		}
		k3_v, k3_a := computeAcceleration(mid2, config, spacecraft)
		
		end := OrbitalState{
			Position: current.Position.Add(k3_v.Scale(dt)),
			Velocity: current.Velocity.Add(k3_a.Scale(dt)),
			Mass:     current.Mass,
		}
		k4_v, k4_a := computeAcceleration(end, config, spacecraft)

		// Combine RK4 terms
		dv := k1_v.Add(k2_v.Scale(2)).Add(k3_v.Scale(2)).Add(k4_v).Scale(dt / 6)
		da := k1_a.Add(k2_a.Scale(2)).Add(k3_a.Scale(2)).Add(k4_a).Scale(dt / 6)

		current = OrbitalState{
			Position: current.Position.Add(dv),
			Velocity: current.Velocity.Add(da),
			Mass:     current.Mass,
			Time:     current.Time.Add(config.IntegrationStep),
			Epoch:    state.Epoch,
		}

		states = append(states, current)
	}

	return states
}

// SpacecraftParams contains spacecraft physical properties
type SpacecraftParams struct {
	DragArea      float64 // m²
	SRPArea       float64 // m²
	DragCoeff     float64 // dimensionless
	Reflectivity  float64 // 0-1
	DryMass       float64 // kg
}

// computeAcceleration computes total acceleration at a state
func computeAcceleration(state OrbitalState, config PropagatorConfig, sc SpacecraftParams) (Vector3D, Vector3D) {
	// Velocity derivative is just velocity
	vDot := state.Velocity

	// Acceleration is sum of all forces
	var aDot Vector3D

	// Gravity
	centralBody := "earth"
	aDot = aDot.Add(CalculateGravity(state.Position, config.GravityModel, centralBody))

	// Atmospheric drag
	if config.IncludeDrag {
		drag := CalculateDrag(state.Velocity, state.Position, state.Mass, sc.DragArea, sc.DragCoeff, config.AtmosphereModel)
		aDot = aDot.Add(drag)
	}

	// Solar radiation pressure
	if config.IncludeSRP {
		// Approximate sun position (would use ephemeris in production)
		sunPos := Vector3D{X: 1.496e11, Y: 0, Z: 0}
		srp := CalculateSolarRadiationPressure(state.Position, sunPos, state.Mass, sc.SRPArea, sc.Reflectivity)
		aDot = aDot.Add(srp)
	}

	// Third body perturbations (Moon, Sun)
	if config.IncludeThirdBody {
		// Simplified third body - would use ephemeris
		moonPos := Vector3D{X: 384400e3, Y: 0, Z: 0} // Approximate
		moonAccel := thirdBodyAccel(state.Position, moonPos, GM_Moon)
		aDot = aDot.Add(moonAccel)
	}

	return vDot, aDot
}

// thirdBodyAccel computes gravitational acceleration from third body
func thirdBodyAccel(position, bodyPosition Vector3D, gm float64) Vector3D {
	// Vector from body to spacecraft
	r := position.Sub(bodyPosition)
	rMag := r.Magnitude()
	rMag3 := rMag * rMag * rMag

	// Vector from origin to body
	rBody := bodyPosition
	rBodyMag := rBody.Magnitude()
	rBodyMag3 := rBodyMag * rBodyMag * rBodyMag

	if rMag == 0 || rBodyMag == 0 {
		return Vector3D{}
	}

	// Third body acceleration
	return r.Scale(-gm/rMag3).Sub(rBody.Scale(gm / rBodyMag3))
}

// ================================================================================
// INTERCEPT & TARGETING
// ================================================================================

// InterceptSolution contains the solution for intercepting a moving target
type InterceptSolution struct {
	InterceptPoint   Vector3D
	InterceptTime    time.Duration
	LaunchVelocity   Vector3D
	ImpactVelocity   Vector3D
	FlightTime       time.Duration
	DeltaV           float64
	Feasibility      float64
	ClosingVelocity  float64
	ImpactAngle      float64 // radians from vertical
}

// CalculateMovingTargetIntercept computes intercept for a maneuvering target
func CalculateMovingTargetIntercept(
	launchPos, launchVel Vector3D,
	targetPos, targetVel, targetAccel Vector3D,
	maxDeltaV float64,
	maxFlightTime time.Duration,
) (*InterceptSolution, error) {
	
	// Iterative solution using predicted impact point (PIP)
	// Start with current target position as first guess
	
	bestSolution := &InterceptSolution{Feasibility: 0}
	dt := 1.0 // seconds
	
	for tFlight := 10.0; tFlight <= maxFlightTime.Seconds(); tFlight += dt {
		// Predict target position at time tFlight
		// Using constant acceleration model: r = r0 + v0*t + 0.5*a*t²
		targetFuture := Vector3D{
			X: targetPos.X + targetVel.X*tFlight + 0.5*targetAccel.X*tFlight*tFlight,
			Y: targetPos.Y + targetVel.Y*tFlight + 0.5*targetAccel.Y*tFlight*tFlight,
			Z: targetPos.Z + targetVel.Z*tFlight + 0.5*targetAccel.Z*tFlight*tFlight,
		}

		// Calculate required velocity to reach target in tFlight
		// Simplified: straight line (would use Lambert solver for orbital)
		toTarget := targetFuture.Sub(launchPos)
		distance := toTarget.Magnitude()
		
		// Check if distance is reasonable for the flight time
		minSpeed := distance / tFlight
		if minSpeed > 50000 { // 50 km/s is unreasonable for most payloads
			continue
		}

		// Account for gravity (simplified)
		altitude := launchPos.Magnitude() - R_Earth
		gravityCorrection := Vector3D{Z: 0.5 * 9.81 * tFlight * tFlight}
		if altitude > KarmanLine {
			gravityCorrection = Vector3D{} // In space, use orbital mechanics
		}
		
		toTargetCorrected := toTarget.Add(gravityCorrection)
		requiredVel := toTargetCorrected.Scale(1 / tFlight)

		// Delta-V required
		deltaV := requiredVel.Sub(launchVel).Magnitude()

		if deltaV > maxDeltaV {
			continue // Not feasible
		}

		// Calculate closing velocity
		targetVelAtIntercept := Vector3D{
			X: targetVel.X + targetAccel.X*tFlight,
			Y: targetVel.Y + targetAccel.Y*tFlight,
			Z: targetVel.Z + targetAccel.Z*tFlight,
		}
		closingVel := requiredVel.Sub(targetVelAtIntercept).Magnitude()

		// Impact angle
		impactDir := requiredVel.Normalize()
		verticalDir := targetFuture.Normalize()
		impactAngle := math.Acos(math.Abs(impactDir.Dot(verticalDir)))

		// Score this solution
		feasibility := 1.0 - (deltaV / maxDeltaV)
		feasibility *= math.Min(1.0, 50.0/tFlight) // Prefer shorter flight times
		feasibility *= math.Min(1.0, closingVel/500.0) // Prefer higher closing speeds

		if feasibility > bestSolution.Feasibility {
			bestSolution = &InterceptSolution{
				InterceptPoint:  targetFuture,
				InterceptTime:   time.Duration(tFlight) * time.Second,
				LaunchVelocity:  requiredVel,
				ImpactVelocity:  requiredVel,
				FlightTime:      time.Duration(tFlight) * time.Second,
				DeltaV:          deltaV,
				Feasibility:     feasibility,
				ClosingVelocity: closingVel,
				ImpactAngle:     impactAngle,
			}
		}
	}

	if bestSolution.Feasibility == 0 {
		return nil, errors.New("no feasible intercept found within constraints")
	}

	return bestSolution, nil
}

// ================================================================================
// LAMBERT SOLVER (Orbital Transfer)
// ================================================================================

// LambertSolution contains a Lambert problem solution
type LambertSolution struct {
	V1          Vector3D // Initial velocity
	V2          Vector3D // Final velocity
	DeltaV1     float64  // Delta-V at departure
	DeltaV2     float64  // Delta-V at arrival
	TotalDeltaV float64
	TransferTime time.Duration
	ShortWay    bool
}

// SolveLambert solves Lambert's problem for orbital transfer
// Given two positions and transfer time, find the connecting orbit
func SolveLambert(r1, r2 Vector3D, transferTime time.Duration, gm float64, shortWay bool) (*LambertSolution, error) {
	tof := transferTime.Seconds()
	if tof <= 0 {
		return nil, errors.New("transfer time must be positive")
	}

	r1Mag := r1.Magnitude()
	r2Mag := r2.Magnitude()
	
	// Cross product determines transfer direction
	cross := r1.Cross(r2)
	
	// Angle between position vectors
	cosTA := r1.Dot(r2) / (r1Mag * r2Mag)
	cosTA = math.Max(-1, math.Min(1, cosTA)) // Clamp
	
	var sinTA float64
	if shortWay {
		if cross.Z >= 0 {
			sinTA = math.Sqrt(1 - cosTA*cosTA)
		} else {
			sinTA = -math.Sqrt(1 - cosTA*cosTA)
		}
	} else {
		if cross.Z < 0 {
			sinTA = math.Sqrt(1 - cosTA*cosTA)
		} else {
			sinTA = -math.Sqrt(1 - cosTA*cosTA)
		}
	}

	// Geometric parameter
	A := sinTA * math.Sqrt(r1Mag*r2Mag/(1-cosTA))

	// Universal variable iteration
	z := 0.0 // Initial guess
	
	for iter := 0; iter < 100; iter++ {
		C, S := stumpffCS(z)
		
		y := r1Mag + r2Mag + A*(z*S-1)/math.Sqrt(C)
		if y < 0 {
			z += 0.1
			continue
		}
		
		sqrtY := math.Sqrt(y)
		x := sqrtY / math.Sqrt(C)
		
		// Time of flight equation
		tofCalc := (x*x*x*S + A*sqrtY) / math.Sqrt(gm)
		
		// Derivative
		if math.Abs(z) > 1e-6 {
			// Non-zero z
		} else {
			// z ≈ 0
		}
		
		// Newton iteration
		if math.Abs(tofCalc-tof) < 1e-6 {
			break
		}
		
		// Simple iteration adjustment
		if tofCalc > tof {
			z += 0.5
		} else {
			z -= 0.5
		}
	}

	// Compute velocities
	C, S := stumpffCS(z)
	y := r1Mag + r2Mag + A*(z*S-1)/math.Sqrt(C)
	
	f := 1 - y/r1Mag
	g := A * math.Sqrt(y/gm)
	gDot := 1 - y/r2Mag

	v1 := r2.Sub(r1.Scale(f)).Scale(1 / g)
	v2 := r2.Scale(gDot).Sub(r1).Scale(1 / g)

	return &LambertSolution{
		V1:           v1,
		V2:           v2,
		DeltaV1:      v1.Magnitude(),
		DeltaV2:      v2.Magnitude(),
		TotalDeltaV:  v1.Magnitude() + v2.Magnitude(),
		TransferTime: transferTime,
		ShortWay:     shortWay,
	}, nil
}

// stumpffCS computes Stumpff functions C(z) and S(z)
func stumpffCS(z float64) (float64, float64) {
	if z > 1e-6 {
		sqrtZ := math.Sqrt(z)
		C := (1 - math.Cos(sqrtZ)) / z
		S := (sqrtZ - math.Sin(sqrtZ)) / math.Pow(z, 1.5)
		return C, S
	} else if z < -1e-6 {
		sqrtNegZ := math.Sqrt(-z)
		C := (1 - math.Cosh(sqrtNegZ)) / z
		S := (math.Sinh(sqrtNegZ) - sqrtNegZ) / math.Pow(-z, 1.5)
		return C, S
	}
	// z ≈ 0
	return 0.5, 1.0 / 6.0
}

// ================================================================================
// RE-ENTRY BALLISTICS
// ================================================================================

// ReentryParams contains re-entry vehicle parameters
type ReentryParams struct {
	Mass            float64 // kg
	NoseRadius      float64 // m
	BaseArea        float64 // m²
	CD              float64 // drag coefficient
	AblationRate    float64 // kg/(m²·s) at reference flux
	HeatShieldMass  float64 // kg
	ThermalLimit    float64 // K (max temperature)
}

// ReentryState contains re-entry trajectory state
type ReentryState struct {
	Position     Vector3D
	Velocity     Vector3D
	Mass         float64
	HeatRate     float64 // W/m²
	Temperature  float64 // K surface temperature
	HeatShieldRemaining float64 // kg
	Altitude     float64
	Mach         float64
	GLoad        float64 // g's
}

// SimulateReentry simulates atmospheric re-entry
func SimulateReentry(initialState OrbitalState, params ReentryParams, targetPos Vector3D) []ReentryState {
	states := make([]ReentryState, 0)
	
	dt := 0.1 // seconds
	state := ReentryState{
		Position:     initialState.Position,
		Velocity:     initialState.Velocity,
		Mass:         params.Mass,
		HeatShieldRemaining: params.HeatShieldMass,
	}

	for i := 0; i < 10000; i++ { // Max 1000 seconds
		altitude := state.Position.Magnitude() - R_Earth
		if altitude < 0 {
			break // Impact
		}

		// Atmospheric density
		rho := GetAtmosphericDensity(altitude, AtmosphereUS76)
		
		// Velocity magnitude
		vMag := state.Velocity.Magnitude()
		
		// Mach number
		T := 288.15 - 0.0065*math.Min(altitude, 11000) // Temperature approximation
		a := math.Sqrt(1.4 * 287 * T) // Speed of sound
		state.Mach = vMag / a

		// Drag force
		dynamicPressure := 0.5 * rho * vMag * vMag
		dragForce := dynamicPressure * params.CD * params.BaseArea
		dragAccel := dragForce / state.Mass

		// Gravity
		r := state.Position.Magnitude()
		gravAccel := GM_Earth / (r * r)

		// Total deceleration in g's
		state.GLoad = dragAccel / 9.81

		// Heat rate (Sutton-Graves correlation)
		// q = k * sqrt(rho/rn) * v³
		k := 1.83e-4 // W/(m²·(kg/m³)^0.5·(m/s)³)
		heatRate := k * math.Sqrt(rho/params.NoseRadius) * math.Pow(vMag, 3)
		state.HeatRate = heatRate

		// Surface temperature (radiation equilibrium)
		// q = ε σ T⁴
		epsilon := 0.9 // Emissivity
		state.Temperature = math.Pow(heatRate/(epsilon*StefanBoltzmann), 0.25)

		// Ablation mass loss
		if state.Temperature > 2000 && state.HeatShieldRemaining > 0 {
			ablation := params.AblationRate * heatRate / 1e6 * dt * params.BaseArea
			state.HeatShieldRemaining -= ablation
			state.Mass -= ablation
		}

		state.Altitude = altitude

		states = append(states, state)

		// Update position and velocity
		vUnit := state.Velocity.Normalize()
		posUnit := state.Position.Normalize()

		// Drag (opposite to velocity)
		aDrag := vUnit.Scale(-dragAccel)
		// Gravity (toward center)
		aGrav := posUnit.Scale(-gravAccel)

		totalAccel := aDrag.Add(aGrav)

		state.Velocity = state.Velocity.Add(totalAccel.Scale(dt))
		state.Position = state.Position.Add(state.Velocity.Scale(dt))

		// Check thermal limit
		if state.Temperature > params.ThermalLimit || state.HeatShieldRemaining <= 0 {
			// Vehicle destroyed
			break
		}
	}

	return states
}

// ================================================================================
// PRECISION DELIVERY METRICS
// ================================================================================

// DeliveryAccuracy contains precision metrics
type DeliveryAccuracy struct {
	CEP             float64 // Circular Error Probable (meters) - 50% within this radius
	SEP             float64 // Spherical Error Probable (meters)
	MaxError        float64 // Maximum error (meters)
	MeanError       float64 // Mean error (meters)
	StdDeviation    float64 // Standard deviation (meters)
	Bias            Vector3D // Systematic bias
	ConfidenceLevel float64 // 0-1
}

// CalculateDeliveryAccuracy computes accuracy metrics from a set of delivery points
func CalculateDeliveryAccuracy(targetPos Vector3D, impactPoints []Vector3D) DeliveryAccuracy {
	n := len(impactPoints)
	if n == 0 {
		return DeliveryAccuracy{}
	}

	// Calculate errors
	errors := make([]float64, n)
	var sumError float64
	var sumBias Vector3D
	maxError := 0.0

	for i, impact := range impactPoints {
		diff := impact.Sub(targetPos)
		err := diff.Magnitude()
		errors[i] = err
		sumError += err
		sumBias = sumBias.Add(diff)
		if err > maxError {
			maxError = err
		}
	}

	meanError := sumError / float64(n)
	bias := sumBias.Scale(1 / float64(n))

	// Standard deviation
	var sumSqDev float64
	for _, err := range errors {
		sumSqDev += (err - meanError) * (err - meanError)
	}
	stdDev := math.Sqrt(sumSqDev / float64(n))

	// CEP - sort errors and find 50th percentile
	sortedErrors := make([]float64, n)
	copy(sortedErrors, errors)
	sort.Float64s(sortedErrors)
	
	cepIndex := n / 2
	cep := sortedErrors[cepIndex]

	// SEP - 3D equivalent
	sep := cep * 1.2 // Approximation

	return DeliveryAccuracy{
		CEP:             cep,
		SEP:             sep,
		MaxError:        maxError,
		MeanError:       meanError,
		StdDeviation:    stdDev,
		Bias:            bias,
		ConfidenceLevel: math.Max(0, 1.0-stdDev/100.0),
	}
}

// Import fmt for error formatting - this is handled at the package level
// errors use the standard fmt.Errorf from the "fmt" package
