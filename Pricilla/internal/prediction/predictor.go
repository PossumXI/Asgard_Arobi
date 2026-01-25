package prediction

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

// PredictionType defines what is being predicted
type PredictionType string

const (
	PredictTrajectory    PredictionType = "trajectory"     // Future path prediction
	PredictIntercept     PredictionType = "intercept"      // Intercept point calculation
	PredictThreat        PredictionType = "threat"         // Threat movement prediction
	PredictWeather       PredictionType = "weather"        // Weather impact prediction
	PredictContact       PredictionType = "contact"        // Satellite contact windows
	PredictFuel          PredictionType = "fuel"           // Fuel consumption prediction
	PredictArrival       PredictionType = "arrival"        // ETA prediction
)

// Vector3D represents 3D coordinates
type Vector3D struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// State represents an object's kinematic state
type State struct {
	Position     Vector3D  `json:"position"`
	Velocity     Vector3D  `json:"velocity"`
	Acceleration Vector3D  `json:"acceleration"`
	Heading      float64   `json:"heading"`
	Timestamp    time.Time `json:"timestamp"`
}

// Prediction represents a predicted future state
type Prediction struct {
	ID          string         `json:"id"`
	Type        PredictionType `json:"type"`
	TargetID    string         `json:"targetId"`
	PredictedAt time.Time      `json:"predictedAt"`
	Horizon     time.Duration  `json:"horizon"`
	States      []State        `json:"states"`
	Confidence  float64        `json:"confidence"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// InterceptSolution represents an intercept calculation
type InterceptSolution struct {
	ID               string    `json:"id"`
	TargetID         string    `json:"targetId"`
	InterceptPoint   Vector3D  `json:"interceptPoint"`
	InterceptTime    time.Time `json:"interceptTime"`
	TimeToIntercept  time.Duration `json:"timeToIntercept"`
	RequiredVelocity Vector3D  `json:"requiredVelocity"`
	ClosingSpeed     float64   `json:"closingSpeed"`
	Feasibility      float64   `json:"feasibility"` // 0.0-1.0
	DeltaV           float64   `json:"deltaV"`      // Required velocity change
}

// ContactWindow represents a communication opportunity
type ContactWindow struct {
	ID             string    `json:"id"`
	SatelliteID    string    `json:"satelliteId"`
	GroundStation  string    `json:"groundStation"`
	StartTime      time.Time `json:"startTime"`
	EndTime        time.Time `json:"endTime"`
	Duration       time.Duration `json:"duration"`
	MaxElevation   float64   `json:"maxElevation"`    // degrees
	AvgLinkQuality float64   `json:"avgLinkQuality"`  // 0.0-1.0
}

// WeatherPrediction represents weather impact on operations
type WeatherPrediction struct {
	ID           string    `json:"id"`
	Location     Vector3D  `json:"location"`
	ValidFrom    time.Time `json:"validFrom"`
	ValidUntil   time.Time `json:"validUntil"`
	WindSpeed    float64   `json:"windSpeed"`     // m/s
	WindHeading  float64   `json:"windHeading"`   // radians
	Visibility   float64   `json:"visibility"`    // meters
	Precipitation float64  `json:"precipitation"` // mm/hr
	CloudCeiling float64   `json:"cloudCeiling"`  // meters
	OperationalImpact float64 `json:"operationalImpact"` // 0.0-1.0
}

// FuelPrediction represents fuel consumption forecast
type FuelPrediction struct {
	ID              string    `json:"id"`
	PayloadID       string    `json:"payloadId"`
	CurrentFuel     float64   `json:"currentFuel"`     // percentage
	FuelAtArrival   float64   `json:"fuelAtArrival"`   // percentage
	ConsumptionRate float64   `json:"consumptionRate"` // %/minute
	RangeRemaining  float64   `json:"rangeRemaining"`  // meters
	SafetyMargin    float64   `json:"safetyMargin"`    // percentage
	CanReachTarget  bool      `json:"canReachTarget"`
	OptimalSpeed    float64   `json:"optimalSpeed"`    // m/s for fuel efficiency
}

// PredictorConfig holds predictor configuration
type PredictorConfig struct {
	DefaultHorizon    time.Duration `json:"defaultHorizon"`
	UpdateInterval    time.Duration `json:"updateInterval"`
	MinConfidence     float64       `json:"minConfidence"`
	MaxPredictions    int           `json:"maxPredictions"`
	EnableKalman      bool          `json:"enableKalman"`
	EnableML          bool          `json:"enableML"`          // Machine learning predictions
	HistorySize       int           `json:"historySize"`       // States to keep for ML
}

// Predictor provides AI-powered prediction capabilities
type Predictor struct {
	mu sync.RWMutex

	id          string
	config      PredictorConfig
	stateHistory map[string][]State        // History per target
	predictions  map[string]*Prediction
	contactCache []ContactWindow
	weatherCache map[string]*WeatherPrediction

	// Kalman filter state per target
	kalmanFilters map[string]*KalmanState
}

// KalmanState holds Kalman filter state for a target
type KalmanState struct {
	X       []float64   // State vector [x, y, z, vx, vy, vz, ax, ay, az]
	P       [][]float64 // Covariance matrix
	LastUpdate time.Time
}

// NewPredictor creates a new Predictor instance
func NewPredictor(id string, config PredictorConfig) *Predictor {
	if config.DefaultHorizon == 0 {
		config.DefaultHorizon = 5 * time.Minute
	}
	if config.HistorySize == 0 {
		config.HistorySize = 100
	}

	return &Predictor{
		id:            id,
		config:        config,
		stateHistory:  make(map[string][]State),
		predictions:   make(map[string]*Prediction),
		contactCache:  make([]ContactWindow, 0),
		weatherCache:  make(map[string]*WeatherPrediction),
		kalmanFilters: make(map[string]*KalmanState),
	}
}

// UpdateState updates the observed state for a target
func (p *Predictor) UpdateState(targetID string, state State) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Initialize history if needed
	if _, exists := p.stateHistory[targetID]; !exists {
		p.stateHistory[targetID] = make([]State, 0, p.config.HistorySize)
	}

	// Add to history
	p.stateHistory[targetID] = append(p.stateHistory[targetID], state)

	// Trim to max size
	if len(p.stateHistory[targetID]) > p.config.HistorySize {
		p.stateHistory[targetID] = p.stateHistory[targetID][1:]
	}

	// Update Kalman filter if enabled
	if p.config.EnableKalman {
		p.updateKalman(targetID, state)
	}
}

// PredictTrajectory predicts future trajectory for a target
func (p *Predictor) PredictTrajectory(ctx context.Context, targetID string, horizon time.Duration) (*Prediction, error) {
	p.mu.RLock()
	history, exists := p.stateHistory[targetID]
	p.mu.RUnlock()

	if !exists || len(history) == 0 {
		return nil, &PredictionError{Message: "no state history for target"}
	}

	if horizon == 0 {
		horizon = p.config.DefaultHorizon
	}

	// Get latest state
	latestState := history[len(history)-1]

	// Generate prediction using physics-based model
	prediction := &Prediction{
		ID:          uuid.New().String(),
		Type:        PredictTrajectory,
		TargetID:    targetID,
		PredictedAt: time.Now(),
		Horizon:     horizon,
		States:      make([]State, 0),
		Metadata:    make(map[string]interface{}),
	}

	// Use Kalman filter if available
	if p.config.EnableKalman {
		p.mu.RLock()
		kalman, hasKalman := p.kalmanFilters[targetID]
		p.mu.RUnlock()

		if hasKalman {
			prediction.States = p.propagateKalman(kalman, horizon)
			prediction.Confidence = p.calculateKalmanConfidence(kalman)
			prediction.Metadata["method"] = "kalman"
			return prediction, nil
		}
	}

	// Fall back to simple physics propagation
	prediction.States = p.propagatePhysics(latestState, horizon)
	prediction.Confidence = p.calculatePhysicsConfidence(history)
	prediction.Metadata["method"] = "physics"

	return prediction, nil
}

// CalculateIntercept calculates intercept solution
func (p *Predictor) CalculateIntercept(ctx context.Context, pursuerState State, targetID string, maxSpeed float64) (*InterceptSolution, error) {
	p.mu.RLock()
	history, exists := p.stateHistory[targetID]
	p.mu.RUnlock()

	if !exists || len(history) == 0 {
		return nil, &PredictionError{Message: "no state history for target"}
	}

	targetState := history[len(history)-1]

	// Calculate intercept using proportional navigation
	solution := p.calculateProportionalNav(pursuerState, targetState, maxSpeed)

	return solution, nil
}

// PredictContactWindows predicts communication windows
func (p *Predictor) PredictContactWindows(ctx context.Context, satelliteID string, groundStation string, horizon time.Duration) ([]ContactWindow, error) {
	// Simplified contact prediction - in reality would use orbital mechanics
	windows := make([]ContactWindow, 0)

	now := time.Now()
	orbitPeriod := 90 * time.Minute // LEO typical

	for t := now; t.Before(now.Add(horizon)); t = t.Add(orbitPeriod) {
		// Simulate contact window (10-15 minutes per pass)
		duration := time.Duration(10+5*math.Sin(float64(t.Unix()))) * time.Minute

		window := ContactWindow{
			ID:             uuid.New().String(),
			SatelliteID:    satelliteID,
			GroundStation:  groundStation,
			StartTime:      t.Add(15 * time.Minute), // Pass starts after 15 min
			EndTime:        t.Add(15*time.Minute + duration),
			Duration:       duration,
			MaxElevation:   30 + 50*math.Abs(math.Sin(float64(t.Unix()))), // degrees
			AvgLinkQuality: 0.7 + 0.2*math.Abs(math.Sin(float64(t.Unix()))),
		}

		windows = append(windows, window)
	}

	return windows, nil
}

// PredictFuelConsumption predicts fuel usage
func (p *Predictor) PredictFuelConsumption(ctx context.Context, payloadID string, trajectory []State, currentFuel float64) (*FuelPrediction, error) {
	if len(trajectory) < 2 {
		return nil, &PredictionError{Message: "trajectory too short for fuel prediction"}
	}

	// Calculate total distance and velocity changes
	totalDistance := 0.0
	totalDeltaV := 0.0

	for i := 1; i < len(trajectory); i++ {
		prev := trajectory[i-1]
		curr := trajectory[i]

		// Distance
		dx := curr.Position.X - prev.Position.X
		dy := curr.Position.Y - prev.Position.Y
		dz := curr.Position.Z - prev.Position.Z
		totalDistance += math.Sqrt(dx*dx + dy*dy + dz*dz)

		// Velocity change
		dvx := curr.Velocity.X - prev.Velocity.X
		dvy := curr.Velocity.Y - prev.Velocity.Y
		dvz := curr.Velocity.Z - prev.Velocity.Z
		totalDeltaV += math.Sqrt(dvx*dvx + dvy*dvy + dvz*dvz)
	}

	// Fuel consumption model (simplified)
	// Base consumption: 0.001% per meter
	// Maneuver consumption: 0.1% per m/s delta-v
	baseConsumption := totalDistance * 0.00001
	maneuverConsumption := totalDeltaV * 0.1

	totalConsumption := baseConsumption + maneuverConsumption
	fuelAtArrival := currentFuel - totalConsumption

	// Calculate time span
	timeSpan := trajectory[len(trajectory)-1].Timestamp.Sub(trajectory[0].Timestamp)
	consumptionRate := totalConsumption / timeSpan.Minutes()

	// Calculate range remaining with current fuel
	fuelEfficiency := totalDistance / totalConsumption
	rangeRemaining := currentFuel * fuelEfficiency

	// Calculate optimal speed for fuel efficiency
	avgSpeed := totalDistance / timeSpan.Seconds()
	optimalSpeed := avgSpeed * 0.8 // 80% of current speed is more efficient

	prediction := &FuelPrediction{
		ID:              uuid.New().String(),
		PayloadID:       payloadID,
		CurrentFuel:     currentFuel,
		FuelAtArrival:   fuelAtArrival,
		ConsumptionRate: consumptionRate,
		RangeRemaining:  rangeRemaining,
		SafetyMargin:    fuelAtArrival - 10.0, // 10% reserve
		CanReachTarget:  fuelAtArrival > 10.0,
		OptimalSpeed:    optimalSpeed,
	}

	return prediction, nil
}

// updateKalman updates Kalman filter with new observation
func (p *Predictor) updateKalman(targetID string, state State) {
	kalman, exists := p.kalmanFilters[targetID]
	if !exists {
		// Initialize Kalman filter
		kalman = &KalmanState{
			X: []float64{
				state.Position.X, state.Position.Y, state.Position.Z,
				state.Velocity.X, state.Velocity.Y, state.Velocity.Z,
				state.Acceleration.X, state.Acceleration.Y, state.Acceleration.Z,
			},
			P:          makeIdentityMatrix(9, 1.0),
			LastUpdate: state.Timestamp,
		}
		p.kalmanFilters[targetID] = kalman
		return
	}

	dt := state.Timestamp.Sub(kalman.LastUpdate).Seconds()
	if dt <= 0 {
		return
	}

	// Predict step
	F := makeStateTransitionMatrix(dt)
	Q := makeProcessNoiseMatrix(dt, 0.1)

	xPred := multiplyMatrixVector(F, kalman.X)
	PPred := addMatrices(multiplyMatrices(multiplyMatrices(F, kalman.P), transpose(F)), Q)

	// Update step
	z := []float64{
		state.Position.X, state.Position.Y, state.Position.Z,
		state.Velocity.X, state.Velocity.Y, state.Velocity.Z,
		state.Acceleration.X, state.Acceleration.Y, state.Acceleration.Z,
	}

	R := makeIdentityMatrix(9, 0.1) // Measurement noise
	// Note: Observation matrix H is identity (direct state observation), so H*x = x

	y := subtractVectors(z, xPred)
	S := addMatrices(PPred, R)
	K := multiplyMatrices(PPred, inverse(S))

	kalman.X = addVectors(xPred, multiplyMatrixVector(K, y))
	IminusKH := subtractMatrices(makeIdentityMatrix(9, 1.0), K)
	kalman.P = multiplyMatrices(IminusKH, PPred)
	kalman.LastUpdate = state.Timestamp
}

// propagateKalman propagates Kalman state forward
func (p *Predictor) propagateKalman(kalman *KalmanState, horizon time.Duration) []State {
	states := make([]State, 0)
	step := time.Second
	current := kalman.X

	for t := time.Duration(0); t <= horizon; t += step {
		state := State{
			Position: Vector3D{
				X: current[0],
				Y: current[1],
				Z: current[2],
			},
			Velocity: Vector3D{
				X: current[3],
				Y: current[4],
				Z: current[5],
			},
			Acceleration: Vector3D{
				X: current[6],
				Y: current[7],
				Z: current[8],
			},
			Timestamp: kalman.LastUpdate.Add(t),
		}
		states = append(states, state)

		// Propagate forward
		F := makeStateTransitionMatrix(step.Seconds())
		current = multiplyMatrixVector(F, current)
	}

	return states
}

// propagatePhysics uses simple physics to predict trajectory
func (p *Predictor) propagatePhysics(initial State, horizon time.Duration) []State {
	states := make([]State, 0)
	step := time.Second

	pos := initial.Position
	vel := initial.Velocity
	acc := initial.Acceleration

	for t := time.Duration(0); t <= horizon; t += step {
		dt := step.Seconds()

		// Update position
		pos.X += vel.X*dt + 0.5*acc.X*dt*dt
		pos.Y += vel.Y*dt + 0.5*acc.Y*dt*dt
		pos.Z += vel.Z*dt + 0.5*acc.Z*dt*dt

		// Update velocity
		vel.X += acc.X * dt
		vel.Y += acc.Y * dt
		vel.Z += acc.Z * dt

		state := State{
			Position:     pos,
			Velocity:     vel,
			Acceleration: acc,
			Heading:      math.Atan2(vel.Y, vel.X),
			Timestamp:    initial.Timestamp.Add(t),
		}
		states = append(states, state)
	}

	return states
}

// calculateProportionalNav calculates intercept using proportional navigation
func (p *Predictor) calculateProportionalNav(pursuer State, target State, maxSpeed float64) *InterceptSolution {
	// Relative position
	rx := target.Position.X - pursuer.Position.X
	ry := target.Position.Y - pursuer.Position.Y
	rz := target.Position.Z - pursuer.Position.Z

	// Range
	r := math.Sqrt(rx*rx + ry*ry + rz*rz)

	// Relative velocity
	vx := target.Velocity.X - pursuer.Velocity.X
	vy := target.Velocity.Y - pursuer.Velocity.Y
	vz := target.Velocity.Z - pursuer.Velocity.Z

	// Closing velocity
	closingSpeed := -(rx*vx + ry*vy + rz*vz) / r

	// Time to intercept
	if closingSpeed <= 0 {
		closingSpeed = maxSpeed // Target moving away, assume we can catch up
	}
	tgo := r / closingSpeed

	// Predicted intercept point
	interceptPoint := Vector3D{
		X: target.Position.X + target.Velocity.X*tgo,
		Y: target.Position.Y + target.Velocity.Y*tgo,
		Z: target.Position.Z + target.Velocity.Z*tgo,
	}

	// Required velocity to reach intercept
	reqVx := (interceptPoint.X - pursuer.Position.X) / tgo
	reqVy := (interceptPoint.Y - pursuer.Position.Y) / tgo
	reqVz := (interceptPoint.Z - pursuer.Position.Z) / tgo

	reqSpeed := math.Sqrt(reqVx*reqVx + reqVy*reqVy + reqVz*reqVz)

	// Calculate delta-V needed
	dvx := reqVx - pursuer.Velocity.X
	dvy := reqVy - pursuer.Velocity.Y
	dvz := reqVz - pursuer.Velocity.Z
	deltaV := math.Sqrt(dvx*dvx + dvy*dvy + dvz*dvz)

	// Feasibility (1.0 if achievable with max speed)
	feasibility := 1.0
	if reqSpeed > maxSpeed {
		feasibility = maxSpeed / reqSpeed
	}

	return &InterceptSolution{
		ID:               uuid.New().String(),
		TargetID:         "",
		InterceptPoint:   interceptPoint,
		InterceptTime:    pursuer.Timestamp.Add(time.Duration(tgo) * time.Second),
		TimeToIntercept:  time.Duration(tgo) * time.Second,
		RequiredVelocity: Vector3D{X: reqVx, Y: reqVy, Z: reqVz},
		ClosingSpeed:     closingSpeed,
		Feasibility:      feasibility,
		DeltaV:           deltaV,
	}
}

// calculateKalmanConfidence calculates prediction confidence from Kalman state
func (p *Predictor) calculateKalmanConfidence(kalman *KalmanState) float64 {
	// Confidence based on trace of covariance matrix
	trace := 0.0
	for i := 0; i < len(kalman.P); i++ {
		trace += kalman.P[i][i]
	}

	// Lower trace = higher confidence
	confidence := 1.0 / (1.0 + trace/100.0)
	return math.Max(0.1, math.Min(0.99, confidence))
}

// calculatePhysicsConfidence calculates confidence from state history
func (p *Predictor) calculatePhysicsConfidence(history []State) float64 {
	if len(history) < 3 {
		return 0.5
	}

	// Measure consistency of acceleration
	accVar := 0.0
	for i := 2; i < len(history); i++ {
		prev := history[i-1]
		curr := history[i]

		dax := curr.Acceleration.X - prev.Acceleration.X
		day := curr.Acceleration.Y - prev.Acceleration.Y
		daz := curr.Acceleration.Z - prev.Acceleration.Z

		accVar += dax*dax + day*day + daz*daz
	}

	accVar /= float64(len(history) - 2)

	// Lower variance = higher confidence
	confidence := 1.0 / (1.0 + accVar)
	return math.Max(0.3, math.Min(0.95, confidence))
}

// Matrix helper functions (simplified implementations)

func makeIdentityMatrix(size int, scale float64) [][]float64 {
	m := make([][]float64, size)
	for i := range m {
		m[i] = make([]float64, size)
		m[i][i] = scale
	}
	return m
}

func makeStateTransitionMatrix(dt float64) [][]float64 {
	// State transition for position-velocity-acceleration model
	F := makeIdentityMatrix(9, 1.0)
	// Position depends on velocity
	F[0][3] = dt
	F[1][4] = dt
	F[2][5] = dt
	// Position depends on acceleration
	F[0][6] = 0.5 * dt * dt
	F[1][7] = 0.5 * dt * dt
	F[2][8] = 0.5 * dt * dt
	// Velocity depends on acceleration
	F[3][6] = dt
	F[4][7] = dt
	F[5][8] = dt
	return F
}

func makeProcessNoiseMatrix(dt float64, q float64) [][]float64 {
	Q := makeIdentityMatrix(9, q*dt)
	return Q
}

func multiplyMatrixVector(m [][]float64, v []float64) []float64 {
	result := make([]float64, len(m))
	for i := range m {
		for j := range v {
			result[i] += m[i][j] * v[j]
		}
	}
	return result
}

func multiplyMatrices(a, b [][]float64) [][]float64 {
	n := len(a)
	result := make([][]float64, n)
	for i := range result {
		result[i] = make([]float64, n)
		for j := range result[i] {
			for k := range a[i] {
				result[i][j] += a[i][k] * b[k][j]
			}
		}
	}
	return result
}

func addMatrices(a, b [][]float64) [][]float64 {
	result := make([][]float64, len(a))
	for i := range result {
		result[i] = make([]float64, len(a[i]))
		for j := range result[i] {
			result[i][j] = a[i][j] + b[i][j]
		}
	}
	return result
}

func subtractMatrices(a, b [][]float64) [][]float64 {
	result := make([][]float64, len(a))
	for i := range result {
		result[i] = make([]float64, len(a[i]))
		for j := range result[i] {
			result[i][j] = a[i][j] - b[i][j]
		}
	}
	return result
}

func transpose(m [][]float64) [][]float64 {
	n := len(m)
	result := make([][]float64, n)
	for i := range result {
		result[i] = make([]float64, n)
		for j := range result[i] {
			result[i][j] = m[j][i]
		}
	}
	return result
}

func inverse(m [][]float64) [][]float64 {
	// Simplified: for small matrices, use diagonal approximation
	n := len(m)
	result := make([][]float64, n)
	for i := range result {
		result[i] = make([]float64, n)
		if m[i][i] != 0 {
			result[i][i] = 1.0 / m[i][i]
		}
	}
	return result
}

func addVectors(a, b []float64) []float64 {
	result := make([]float64, len(a))
	for i := range result {
		result[i] = a[i] + b[i]
	}
	return result
}

func subtractVectors(a, b []float64) []float64 {
	result := make([]float64, len(a))
	for i := range result {
		result[i] = a[i] - b[i]
	}
	return result
}

// PredictionError represents a prediction error
type PredictionError struct {
	Message string
}

func (e *PredictionError) Error() string {
	return e.Message
}
