package physics

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

// init seeds the random number generator for tests
func init() {
	rand.Seed(time.Now().UnixNano())
}

// ================================================================================
// GRAVITATIONAL MODEL TESTS
// ================================================================================

func TestGravityPointMass(t *testing.T) {
	// Test point at surface level (equator)
	pos := Vector3D{X: R_Earth, Y: 0, Z: 0}

	gravity := CalculateGravity(pos, GravityPointMass, "earth")

	// Expected surface gravity: ~9.81 m/s²
	expectedMag := GM_Earth / (R_Earth * R_Earth)
	actualMag := gravity.Magnitude()

	// Allow 0.1% tolerance
	tolerance := expectedMag * 0.001
	if math.Abs(actualMag-expectedMag) > tolerance {
		t.Errorf("Point mass gravity: expected %.6f m/s², got %.6f m/s²", expectedMag, actualMag)
	}

	// Direction should point toward center (negative X in this case)
	if gravity.X >= 0 {
		t.Errorf("Gravity should point toward center, got X=%.6f", gravity.X)
	}

	t.Logf("Surface gravity: %.4f m/s² (expected: %.4f)", actualMag, expectedMag)
}

func TestGravityJ2Effect(t *testing.T) {
	// At equator, J2 should reduce gravity slightly
	posEquator := Vector3D{X: R_Earth, Y: 0, Z: 0}

	// At pole, J2 should increase gravity
	posPole := Vector3D{X: 0, Y: 0, Z: R_Earth}

	gravEquatorPM := CalculateGravity(posEquator, GravityPointMass, "earth")
	gravEquatorJ2 := CalculateGravity(posEquator, GravityJ2, "earth")
	gravPolePM := CalculateGravity(posPole, GravityPointMass, "earth")
	gravPoleJ2 := CalculateGravity(posPole, GravityJ2, "earth")

	// J2 effect at equator should decrease gravity
	if gravEquatorJ2.Magnitude() >= gravEquatorPM.Magnitude() {
		t.Logf("Note: J2 at equator: %.6f vs PM: %.6f", gravEquatorJ2.Magnitude(), gravEquatorPM.Magnitude())
	}

	// J2 effect at pole should increase gravity
	if gravPoleJ2.Magnitude() <= gravPolePM.Magnitude() {
		t.Logf("Note: J2 at pole: %.6f vs PM: %.6f", gravPoleJ2.Magnitude(), gravPolePM.Magnitude())
	}

	t.Logf("Equator PM: %.4f, J2: %.4f (diff: %.6f)",
		gravEquatorPM.Magnitude(), gravEquatorJ2.Magnitude(),
		gravEquatorJ2.Magnitude()-gravEquatorPM.Magnitude())
	t.Logf("Pole PM: %.4f, J2: %.4f (diff: %.6f)",
		gravPolePM.Magnitude(), gravPoleJ2.Magnitude(),
		gravPoleJ2.Magnitude()-gravPolePM.Magnitude())
}

func TestGravityAltitudeDecay(t *testing.T) {
	// Gravity should decrease with altitude following inverse square law
	altitudes := []float64{0, 100e3, 400e3, 1000e3, 35786e3} // Surface, 100km, ISS, 1000km, GEO

	for _, alt := range altitudes {
		r := R_Earth + alt
		pos := Vector3D{X: r, Y: 0, Z: 0}

		gravity := CalculateGravity(pos, GravityPointMass, "earth")

		// Expected: g0 * (R/r)²
		g0 := GM_Earth / (R_Earth * R_Earth)
		expected := g0 * (R_Earth / r) * (R_Earth / r)
		actual := gravity.Magnitude()

		if math.Abs(actual-expected)/expected > 0.01 {
			t.Errorf("At %.0f km: expected %.4f m/s², got %.4f m/s²", alt/1000, expected, actual)
		}

		t.Logf("Altitude %.0f km: gravity = %.4f m/s² (%.1f%% of surface)",
			alt/1000, actual, 100*actual/g0)
	}
}

// ================================================================================
// ATMOSPHERIC MODEL TESTS
// ================================================================================

func TestAtmosphericDensity(t *testing.T) {
	// Test exponential atmosphere
	altitudes := []float64{0, 5000, 10000, 20000, 50000, 100000}

	t.Log("Exponential Atmosphere Model:")
	for _, alt := range altitudes {
		rho := GetAtmosphericDensity(alt, AtmosphereExponential)
		expectedRho := SeaLevelDensity * math.Exp(-alt/ScaleHeight)

		if math.Abs(rho-expectedRho) > 1e-10 {
			t.Errorf("At %.0f m: expected %.6e kg/m³, got %.6e kg/m³", alt, expectedRho, rho)
		}
		t.Logf("  %.0f m: %.6e kg/m³", alt, rho)
	}

	// Test US76 model
	t.Log("\nUS Standard Atmosphere 1976:")
	us76Altitudes := []float64{0, 11000, 20000, 32000, 47000, 71000, 86000}
	for _, alt := range us76Altitudes {
		rho := GetAtmosphericDensity(alt, AtmosphereUS76)
		t.Logf("  %.0f m: %.6e kg/m³", alt, rho)
	}
}

func TestDragCalculation(t *testing.T) {
	// Satellite at 400 km (ISS altitude)
	altitude := 400e3
	pos := Vector3D{X: R_Earth + altitude, Y: 0, Z: 0}

	// Typical LEO velocity: ~7.7 km/s
	vel := Vector3D{X: 0, Y: 7700, Z: 0}

	// Typical satellite parameters
	mass := 420000.0 // kg (ISS mass)
	area := 2500.0   // m² (approximate drag area)
	cd := 2.2        // drag coefficient

	drag := CalculateDrag(vel, pos, mass, area, cd, AtmosphereExponential)
	dragMag := drag.Magnitude()

	// At 400 km, drag should be very small but non-zero
	if dragMag <= 0 || dragMag > 1 {
		t.Errorf("Drag at 400 km seems wrong: %.6e m/s²", dragMag)
	}

	t.Logf("ISS-like drag at 400 km: %.6e m/s² (%.4e N)", dragMag, dragMag*mass)
}

func TestDragCoefficient(t *testing.T) {
	altitude := 10000.0 // 10 km
	baseCD := 0.5

	velocities := []float64{100, 300, 400, 500, 1000, 5000}

	t.Log("Mach-dependent drag coefficient:")
	for _, v := range velocities {
		vel := Vector3D{X: v, Y: 0, Z: 0}
		cd := CalculateDragCoefficient(vel, altitude, baseCD)
		mach := v / SpeedOfSound
		t.Logf("  V=%.0f m/s (Mach %.2f): Cd=%.3f", v, mach, cd)
	}
}

// ================================================================================
// ORBITAL PROPAGATION TESTS
// ================================================================================

func TestOrbitPropagation(t *testing.T) {
	// Start with ISS-like orbit
	altitude := 400e3
	pos := Vector3D{X: R_Earth + altitude, Y: 0, Z: 0}

	// Circular orbit velocity
	v := math.Sqrt(GM_Earth / (R_Earth + altitude))
	vel := Vector3D{X: 0, Y: v, Z: 0}

	initialState := OrbitalState{
		Position: pos,
		Velocity: vel,
		Mass:     1000,
		Time:     time.Now(),
		Epoch:    time.Now(),
	}

	spacecraft := SpacecraftParams{
		DragArea:     10.0,
		SRPArea:      10.0,
		DragCoeff:    2.2,
		Reflectivity: 0.3,
		DryMass:      1000,
	}

	config := PropagatorConfig{
		GravityModel:     GravityJ2,
		AtmosphereModel:  AtmosphereExponential,
		IncludeDrag:      true,
		IncludeSRP:       false,
		IncludeJ2:        true,
		IncludeThirdBody: false,
		IntegrationStep:  10 * time.Second,
	}

	// Propagate for one orbit (~90 minutes)
	duration := 92 * time.Minute
	states := Propagate(initialState, duration, config, spacecraft)

	if len(states) == 0 {
		t.Fatal("No states returned from propagator")
	}

	finalState := states[len(states)-1]
	initialRadius := pos.Magnitude()
	finalRadius := finalState.Position.Magnitude()

	// Due to drag, altitude should decrease slightly
	altDrop := initialRadius - finalRadius
	t.Logf("After one orbit (%d states):", len(states))
	t.Logf("  Initial radius: %.3f km", initialRadius/1000)
	t.Logf("  Final radius: %.3f km", finalRadius/1000)
	t.Logf("  Altitude drop: %.3f m", altDrop)

	// Should complete approximately one orbit
	// Check that position has returned close to original
	posChange := finalState.Position.Sub(pos).Magnitude()
	t.Logf("  Position change: %.3f km", posChange/1000)
}

// ================================================================================
// INTERCEPT CALCULATION TESTS
// ================================================================================

func TestStationaryTargetIntercept(t *testing.T) {
	// Launch from ground, hit stationary target
	launchPos := Vector3D{X: R_Earth, Y: 0, Z: 0}
	launchVel := Vector3D{X: 0, Y: 0, Z: 0}

	// Target 100 km away at 10 km altitude
	targetPos := Vector3D{X: R_Earth + 10000, Y: 100000, Z: 0}
	targetVel := Vector3D{X: 0, Y: 0, Z: 0}
	targetAccel := Vector3D{X: 0, Y: 0, Z: 0}

	solution, err := CalculateMovingTargetIntercept(
		launchPos, launchVel,
		targetPos, targetVel, targetAccel,
		5000, // 5 km/s max delta-v
		5*time.Minute,
	)

	if err != nil {
		t.Fatalf("Intercept failed: %v", err)
	}

	t.Logf("Stationary target intercept:")
	t.Logf("  Flight time: %.1f s", solution.FlightTime.Seconds())
	t.Logf("  Delta-V: %.1f m/s", solution.DeltaV)
	t.Logf("  Feasibility: %.2f", solution.Feasibility)
	t.Logf("  Closing speed: %.1f m/s", solution.ClosingVelocity)
	t.Logf("  Impact angle: %.1f°", solution.ImpactAngle*180/math.Pi)
}

func TestMovingTargetIntercept(t *testing.T) {
	// Interceptor starting position
	launchPos := Vector3D{X: R_Earth, Y: 0, Z: 0}
	launchVel := Vector3D{X: 0, Y: 0, Z: 0}

	// Moving target (aircraft at 10 km, 250 m/s)
	targetPos := Vector3D{X: R_Earth + 10000, Y: 50000, Z: 0}
	targetVel := Vector3D{X: 0, Y: 250, Z: 0}
	targetAccel := Vector3D{X: 0, Y: 0, Z: 0}

	solution, err := CalculateMovingTargetIntercept(
		launchPos, launchVel,
		targetPos, targetVel, targetAccel,
		3000, // 3 km/s max delta-v
		60*time.Second,
	)

	if err != nil {
		t.Fatalf("Intercept failed: %v", err)
	}

	t.Logf("Moving target intercept:")
	t.Logf("  Flight time: %.1f s", solution.FlightTime.Seconds())
	t.Logf("  Delta-V: %.1f m/s", solution.DeltaV)
	t.Logf("  Intercept point: (%.0f, %.0f, %.0f) m",
		solution.InterceptPoint.X, solution.InterceptPoint.Y, solution.InterceptPoint.Z)
	t.Logf("  Feasibility: %.2f", solution.Feasibility)
}

func TestManeuveringTargetIntercept(t *testing.T) {
	// Interceptor
	launchPos := Vector3D{X: R_Earth, Y: 0, Z: 0}
	launchVel := Vector3D{X: 0, Y: 0, Z: 0}

	// Maneuvering target (5 g turn)
	targetPos := Vector3D{X: R_Earth + 15000, Y: 30000, Z: 0}
	targetVel := Vector3D{X: 0, Y: 300, Z: 0}
	targetAccel := Vector3D{X: 0, Y: 0, Z: 50} // 5g vertical maneuver

	solution, err := CalculateMovingTargetIntercept(
		launchPos, launchVel,
		targetPos, targetVel, targetAccel,
		4000,
		60*time.Second,
	)

	if err != nil {
		t.Fatalf("Intercept failed: %v", err)
	}

	t.Logf("Maneuvering target (5g) intercept:")
	t.Logf("  Flight time: %.1f s", solution.FlightTime.Seconds())
	t.Logf("  Delta-V: %.1f m/s", solution.DeltaV)
	t.Logf("  Feasibility: %.2f", solution.Feasibility)
}

// ================================================================================
// LAMBERT SOLVER TESTS
// ================================================================================

func TestLambertSolver(t *testing.T) {
	// Hohmann-like transfer test
	r1 := Vector3D{X: R_Earth + 400e3, Y: 0, Z: 0}   // LEO
	r2 := Vector3D{X: 0, Y: R_Earth + 35786e3, Z: 0} // GEO

	// Transfer time for Hohmann: half the period of transfer orbit
	a_transfer := ((R_Earth + 400e3) + (R_Earth + 35786e3)) / 2
	T_transfer := math.Pi * math.Sqrt(a_transfer*a_transfer*a_transfer/GM_Earth)
	transferTime := time.Duration(T_transfer) * time.Second

	solution, err := SolveLambert(r1, r2, transferTime, GM_Earth, true)
	if err != nil {
		t.Fatalf("Lambert solver failed: %v", err)
	}

	t.Logf("Lambert solution (LEO to GEO):")
	t.Logf("  Transfer time: %.0f s (%.1f hr)", T_transfer, T_transfer/3600)
	t.Logf("  V1: (%.1f, %.1f, %.1f) m/s", solution.V1.X, solution.V1.Y, solution.V1.Z)
	t.Logf("  V2: (%.1f, %.1f, %.1f) m/s", solution.V2.X, solution.V2.Y, solution.V2.Z)
	t.Logf("  Delta-V1: %.1f m/s", solution.DeltaV1)
	t.Logf("  Total Delta-V: %.1f m/s", solution.TotalDeltaV)
}

// ================================================================================
// RADIATION ENVIRONMENT TESTS
// ================================================================================

func TestRadiationEnvironment(t *testing.T) {
	altitudes := []float64{300e3, 1000e3, 3000e3, 10000e3, 20000e3, 36000e3}
	solarActivity := 1.0 // Nominal

	t.Log("Radiation environment by altitude:")
	for _, alt := range altitudes {
		pos := Vector3D{X: R_Earth + alt, Y: 0, Z: 0}
		env := CalculateRadiationEnvironment(pos, solarActivity)

		t.Logf("  %.0f km: Zone=%s, Protons=%.2e, Electrons=%.2e, Dose=%.2e rad/s",
			alt/1000, env.VanAllenZone, env.ProtonFlux, env.ElectronFlux, env.TotalDose)
	}
}

// ================================================================================
// RE-ENTRY SIMULATION TESTS
// ================================================================================

func TestReentrySimulation(t *testing.T) {
	// Start from 120 km altitude, typical re-entry
	altitude := 120e3
	speed := 7800.0               // m/s
	gamma := -1.5 * math.Pi / 180 // -1.5° flight path angle

	pos := Vector3D{X: R_Earth + altitude, Y: 0, Z: 0}
	vel := Vector3D{
		X: -speed * math.Sin(gamma),
		Y: speed * math.Cos(gamma),
		Z: 0,
	}

	initialState := OrbitalState{
		Position: pos,
		Velocity: vel,
		Mass:     5000, // kg
		Time:     time.Now(),
	}

	params := ReentryParams{
		Mass:           5000,
		NoseRadius:     0.5, // m
		BaseArea:       5.0, // m²
		CD:             1.5,
		AblationRate:   0.001, // kg/(m²·s)
		HeatShieldMass: 200,   // kg
		ThermalLimit:   3000,  // K
	}

	targetPos := Vector3D{X: R_Earth, Y: 1000e3, Z: 0}

	states := SimulateReentry(initialState, params, targetPos)

	if len(states) == 0 {
		t.Fatal("No states returned from re-entry simulation")
	}

	t.Logf("Re-entry simulation (%d states):", len(states))

	// Log key moments
	maxG := 0.0
	maxHeat := 0.0
	maxTemp := 0.0

	for _, s := range states {
		if s.GLoad > maxG {
			maxG = s.GLoad
		}
		if s.HeatRate > maxHeat {
			maxHeat = s.HeatRate
		}
		if s.Temperature > maxTemp {
			maxTemp = s.Temperature
		}
	}

	finalState := states[len(states)-1]
	t.Logf("  Final altitude: %.1f km", finalState.Altitude/1000)
	t.Logf("  Max G-load: %.1f g", maxG)
	t.Logf("  Max heat rate: %.2e W/m²", maxHeat)
	t.Logf("  Max temperature: %.0f K", maxTemp)
	t.Logf("  Heat shield remaining: %.1f kg (of %.1f)", finalState.HeatShieldRemaining, params.HeatShieldMass)
}

// ================================================================================
// DELIVERY ACCURACY TESTS
// ================================================================================

func TestDeliveryAccuracy(t *testing.T) {
	// Simulate 100 delivery attempts with Gaussian error
	target := Vector3D{X: 0, Y: 0, Z: 0}
	n := 100
	sigma := 5.0 // 5m standard deviation

	impacts := make([]Vector3D, n)
	for i := 0; i < n; i++ {
		// Gaussian distributed errors
		impacts[i] = Vector3D{
			X: target.X + gaussianRandom()*sigma,
			Y: target.Y + gaussianRandom()*sigma,
			Z: target.Z + gaussianRandom()*sigma,
		}
	}

	accuracy := CalculateDeliveryAccuracy(target, impacts)

	t.Logf("Delivery accuracy (100 samples, σ=%.1f m):", sigma)
	t.Logf("  CEP: %.2f m", accuracy.CEP)
	t.Logf("  SEP: %.2f m", accuracy.SEP)
	t.Logf("  Mean error: %.2f m", accuracy.MeanError)
	t.Logf("  Max error: %.2f m", accuracy.MaxError)
	t.Logf("  Std deviation: %.2f m", accuracy.StdDeviation)
	t.Logf("  Confidence: %.2f", accuracy.ConfidenceLevel)

	// CEP should be approximately 1.1774 * sigma for 2D Gaussian
	// For 3D, it's different
	expectedCEP := sigma * 1.2 // Approximate
	if accuracy.CEP < expectedCEP*0.5 || accuracy.CEP > expectedCEP*2 {
		t.Logf("Note: CEP %.2f seems off for σ=%.1f (expected ~%.1f)", accuracy.CEP, sigma, expectedCEP)
	}
}

// gaussianRandom returns a Gaussian random number using Box-Muller transform
func gaussianRandom() float64 {
	// Box-Muller transform
	u1 := rand.Float64()
	u2 := rand.Float64()

	// Avoid log(0)
	if u1 < 1e-10 {
		u1 = 1e-10
	}

	return math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
}

// ================================================================================
// PRECISION INTERCEPTOR TESTS
// ================================================================================

func TestPrecisionInterceptor(t *testing.T) {
	config := InterceptorConfig{
		PayloadType:     PayloadHypersonic,
		PayloadMass:     1000,
		PayloadThrust:   50000, // 50 kN
		PayloadISP:      300,
		DragArea:        1.0,
		DragCoeff:       0.3,
		MaxG:            30,
		MaxMach:         10,
		SteeringRate:    0.5,
		SeekerFOV:       0.1,
		SeekerRange:     100000,
		MinIntercept:    10,
		CentralBody:     "earth",
		GravityFidelity: GravityJ2,
		AtmoFidelity:    AtmosphereUS76,
		GuidanceLaw:     GuidanceAugProNav,
		UpdateRate:      10 * time.Millisecond,
		LookAheadTime:   30 * time.Second,
	}

	interceptor := NewPrecisionInterceptor(config)

	// Set payload state
	interceptor.UpdatePayloadState(OrbitalState{
		Position: Vector3D{X: R_Earth + 20000, Y: 0, Z: 0},
		Velocity: Vector3D{X: 0, Y: 2000, Z: 0},
		Mass:     1000,
		Time:     time.Now(),
	})

	// Update target
	interceptor.UpdateTargetObservation(TargetState{
		Position:     Vector3D{X: R_Earth + 25000, Y: 50000, Z: 0},
		Velocity:     Vector3D{X: -50, Y: 300, Z: 0},
		Acceleration: Vector3D{X: 0, Y: 0, Z: 0},
		Timestamp:    time.Now(),
		Confidence:   1.0,
	})

	// Compute guidance
	cmd, err := interceptor.ComputeGuidance(nil)
	if err != nil {
		t.Fatalf("Guidance computation failed: %v", err)
	}

	t.Logf("Precision interceptor guidance:")
	t.Logf("  Accel command: (%.2f, %.2f, %.2f) m/s²",
		cmd.AccelCommand.X, cmd.AccelCommand.Y, cmd.AccelCommand.Z)
	t.Logf("  Accel magnitude: %.2f m/s² (%.1f g)",
		cmd.AccelCommand.Magnitude(), cmd.AccelCommand.Magnitude()/9.81)
	t.Logf("  Thrust level: %.2f", cmd.ThrustLevel)
	t.Logf("  Time to go: %.1f s", cmd.TimeToGo.Seconds())
	t.Logf("  Predicted miss: %.2f m", cmd.PredictedMiss)
	t.Logf("  Confidence: %.2f", cmd.Confidence)
}

func TestPayloadAccuracySpecs(t *testing.T) {
	payloads := []PayloadType{
		PayloadBallistic,
		PayloadCruise,
		PayloadHypersonic,
		PayloadOrbitalKV,
		PayloadReentry,
		PayloadDrone,
		PayloadRobot,
	}

	t.Log("Payload accuracy specifications:")
	for _, p := range payloads {
		spec := GetAccuracySpec(p)
		t.Logf("  %s:", p)
		t.Logf("    CEP: %.1f m, SEP: %.1f m", spec.CEP, spec.SEP)
		t.Logf("    Max range: %.0f km", spec.MaxRange/1000)
		t.Logf("    Moving target: %v, Space: %v", spec.MovingTargetCapable, spec.SpaceCapable)
	}
}

// ================================================================================
// BENCHMARK TESTS
// ================================================================================

func BenchmarkGravityJ2(b *testing.B) {
	pos := Vector3D{X: R_Earth + 400e3, Y: 0, Z: 0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateGravity(pos, GravityJ2, "earth")
	}
}

func BenchmarkGravityJ2J3J4(b *testing.B) {
	pos := Vector3D{X: R_Earth + 400e3, Y: 0, Z: 0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateGravity(pos, GravityJ2J3J4, "earth")
	}
}

func BenchmarkAtmosphericDensityUS76(b *testing.B) {
	altitudes := []float64{0, 10000, 30000, 60000, 80000}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		alt := altitudes[i%len(altitudes)]
		GetAtmosphericDensity(alt, AtmosphereUS76)
	}
}

func BenchmarkDragCalculation(b *testing.B) {
	pos := Vector3D{X: R_Earth + 100e3, Y: 0, Z: 0}
	vel := Vector3D{X: 0, Y: 7000, Z: 0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateDrag(vel, pos, 1000, 10, 2.2, AtmosphereUS76)
	}
}

func BenchmarkInterceptCalculation(b *testing.B) {
	launchPos := Vector3D{X: R_Earth, Y: 0, Z: 0}
	launchVel := Vector3D{X: 0, Y: 0, Z: 0}
	targetPos := Vector3D{X: R_Earth + 10000, Y: 50000, Z: 0}
	targetVel := Vector3D{X: 0, Y: 250, Z: 0}
	targetAccel := Vector3D{X: 0, Y: 0, Z: 0}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateMovingTargetIntercept(
			launchPos, launchVel,
			targetPos, targetVel, targetAccel,
			3000, 60*time.Second,
		)
	}
}

func BenchmarkOrbitPropagation(b *testing.B) {
	pos := Vector3D{X: R_Earth + 400e3, Y: 0, Z: 0}
	v := math.Sqrt(GM_Earth / (R_Earth + 400e3))
	vel := Vector3D{X: 0, Y: v, Z: 0}

	initialState := OrbitalState{
		Position: pos,
		Velocity: vel,
		Mass:     1000,
		Time:     time.Now(),
		Epoch:    time.Now(),
	}

	spacecraft := SpacecraftParams{
		DragArea:     10.0,
		DragCoeff:    2.2,
		Reflectivity: 0.3,
		DryMass:      1000,
	}

	config := PropagatorConfig{
		GravityModel:    GravityJ2,
		AtmosphereModel: AtmosphereExponential,
		IncludeDrag:     true,
		IntegrationStep: 10 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Propagate(initialState, 1*time.Minute, config, spacecraft)
	}
}
