// Package fusion provides multi-sensor fusion using Extended Kalman Filter
package fusion

import (
	"context"
	"sync"
	"time"

	"gonum.org/v1/gonum/mat"
)

// SensorType represents different sensor inputs
type SensorType int

const (
	SensorGPS SensorType = iota
	SensorINS
	SensorRADAR
	SensorLIDAR
	SensorVisual
	SensorIR
	SensorWiFiCSI
	SensorBarometer
	SensorPitot
)

// SensorReading represents a sensor measurement
type SensorReading struct {
	Type       SensorType
	Timestamp  time.Time
	Data       *mat.VecDense
	Covariance *mat.SymDense
	Quality    float64 // 0.0 to 1.0
}

// FusionState represents the fused estimate
type FusionState struct {
	Position    [3]float64 // X, Y, Z in meters
	Velocity    [3]float64 // Vx, Vy, Vz in m/s
	Acceleration [3]float64 // Ax, Ay, Az in m/s²
	Attitude    [3]float64 // Roll, Pitch, Yaw in radians
	AngularRate [3]float64 // P, Q, R in rad/s
	Timestamp   time.Time
	Covariance  *mat.SymDense
	Confidence  float64
}

// ExtendedKalmanFilter implements multi-sensor fusion
type ExtendedKalmanFilter struct {
	mu           sync.RWMutex
	state        *mat.VecDense  // State vector (15x1)
	covariance   *mat.SymDense  // Covariance matrix (15x15)
	processNoise *mat.SymDense  // Q matrix
	dt           float64        // Time step

	// Sensor buffers
	sensorReadings chan SensorReading
	fusedState     *FusionState

	// Configuration
	config FusionConfig

	// Statistics
	updateCount uint64
	lastUpdate  time.Time
}

// FusionConfig holds fusion parameters
type FusionConfig struct {
	UpdateRate       float64                 // Hz
	SensorWeights    map[SensorType]float64
	OutlierThreshold float64
	MinSensors       int
	EnableAdaptive   bool
}

// NewEKF creates a new Extended Kalman Filter
func NewEKF(config FusionConfig) *ExtendedKalmanFilter {
	ekf := &ExtendedKalmanFilter{
		state:          mat.NewVecDense(15, nil),
		covariance:     mat.NewSymDense(15, nil),
		processNoise:   mat.NewSymDense(15, nil),
		dt:             1.0 / config.UpdateRate,
		sensorReadings: make(chan SensorReading, 100),
		config:         config,
	}

	// Initialize state to zero
	ekf.Reset()

	return ekf
}

// Reset initializes the filter
func (ekf *ExtendedKalmanFilter) Reset() {
	ekf.mu.Lock()
	defer ekf.mu.Unlock()

	// Zero state
	for i := 0; i < 15; i++ {
		ekf.state.SetVec(i, 0)
	}

	// Initialize covariance with high uncertainty
	for i := 0; i < 15; i++ {
		ekf.covariance.SetSym(i, i, 1000.0)
	}

	// Process noise (tuned values)
	processNoise := []float64{
		0.01, 0.01, 0.01, // Position noise
		0.1, 0.1, 0.1, // Velocity noise
		1.0, 1.0, 1.0, // Acceleration noise
		0.001, 0.001, 0.001, // Attitude noise
		0.01, 0.01, 0.01, // Angular rate noise
	}

	for i := 0; i < 15; i++ {
		ekf.processNoise.SetSym(i, i, processNoise[i])
	}

	ekf.fusedState = &FusionState{
		Timestamp: time.Now(),
	}
}

// Predict performs the prediction step
func (ekf *ExtendedKalmanFilter) Predict(ctx context.Context) error {
	ekf.mu.Lock()
	defer ekf.mu.Unlock()

	// State transition matrix F
	F := ekf.buildStateTransition()

	// Predict state: x̂ₖ₊₁ = F * x̂ₖ
	var predicted mat.VecDense
	predicted.MulVec(F, ekf.state)
	ekf.state.CopyVec(&predicted)

	// Predict covariance: Pₖ₊₁ = F * Pₖ * Fᵀ + Q
	var temp mat.Dense
	temp.Mul(F, ekf.covariance)

	var FT mat.Dense
	FT.CloneFrom(F.T())

	var predictedCov mat.Dense
	predictedCov.Mul(&temp, &FT)

	// Add process noise
	for i := 0; i < 15; i++ {
		for j := 0; j < 15; j++ {
			val := predictedCov.At(i, j)
			if i == j {
				val += ekf.processNoise.At(i, i)
			}
			predictedCov.Set(i, j, val)
		}
	}

	// Convert back to SymDense
	covData := make([]float64, 15*15)
	for i := 0; i < 15; i++ {
		for j := 0; j <= i; j++ {
			covData[i*15+j] = predictedCov.At(i, j)
			covData[j*15+i] = predictedCov.At(i, j)
		}
	}
	ekf.covariance = mat.NewSymDense(15, covData)

	return nil
}

// Update performs the update step with a sensor reading
func (ekf *ExtendedKalmanFilter) Update(reading SensorReading) error {
	ekf.mu.Lock()
	defer ekf.mu.Unlock()

	// Build measurement matrix H for this sensor
	H := ekf.buildMeasurementMatrix(reading.Type)
	rows, _ := H.Dims()

	// Innovation: y = z - H * x̂
	var expected mat.VecDense
	expected.MulVec(H, ekf.state)

	innovation := mat.NewVecDense(rows, nil)
	for i := 0; i < rows; i++ {
		innovation.SetVec(i, reading.Data.AtVec(i)-expected.AtVec(i))
	}

	// Innovation covariance: S = H * P * Hᵀ + R
	var temp mat.Dense
	temp.Mul(H, ekf.covariance)

	var HT mat.Dense
	HT.CloneFrom(H.T())

	var S mat.Dense
	S.Mul(&temp, &HT)

	// Add measurement noise R
	r, c := S.Dims()
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			val := S.At(i, j)
			if i == j && reading.Covariance != nil {
				val += reading.Covariance.At(i, i)
			}
			S.Set(i, j, val)
		}
	}

	// Kalman gain: K = P * Hᵀ * S⁻¹
	var Sinv mat.Dense
	err := Sinv.Inverse(&S)
	if err != nil {
		return err
	}

	var K mat.Dense
	var temp2 mat.Dense
	temp2.Mul(ekf.covariance, &HT)
	K.Mul(&temp2, &Sinv)

	// Update state: x̂ₖ = x̂ₖ + K * y
	var correction mat.VecDense
	correction.MulVec(&K, innovation)
	ekf.state.AddVec(ekf.state, &correction)

	// Update covariance: Pₖ = (I - K * H) * Pₖ
	var KH mat.Dense
	KH.Mul(&K, H)

	I := mat.NewDense(15, 15, nil)
	for i := 0; i < 15; i++ {
		I.Set(i, i, 1.0)
	}

	var IminusKH mat.Dense
	IminusKH.Sub(I, &KH)

	var updatedCov mat.Dense
	updatedCov.Mul(&IminusKH, ekf.covariance)

	// Convert to SymDense
	covData := make([]float64, 15*15)
	for i := 0; i < 15; i++ {
		for j := 0; j <= i; j++ {
			covData[i*15+j] = updatedCov.At(i, j)
			covData[j*15+i] = updatedCov.At(i, j)
		}
	}
	ekf.covariance = mat.NewSymDense(15, covData)

	ekf.updateCount++
	ekf.lastUpdate = time.Now()

	return nil
}

// buildStateTransition builds the state transition matrix F
func (ekf *ExtendedKalmanFilter) buildStateTransition() *mat.Dense {
	// 15x15 matrix for [pos, vel, acc, att, ang_rate]
	F := mat.NewDense(15, 15, nil)

	// Identity for all states
	for i := 0; i < 15; i++ {
		F.Set(i, i, 1.0)
	}

	dt := ekf.dt

	// Position integrates velocity
	F.Set(0, 3, dt) // X = X + Vx*dt
	F.Set(1, 4, dt) // Y = Y + Vy*dt
	F.Set(2, 5, dt) // Z = Z + Vz*dt

	// Velocity integrates acceleration
	F.Set(3, 6, dt) // Vx = Vx + Ax*dt
	F.Set(4, 7, dt) // Vy = Vy + Ay*dt
	F.Set(5, 8, dt) // Vz = Vz + Az*dt

	// Attitude integrates angular rate
	F.Set(9, 12, dt)  // Roll
	F.Set(10, 13, dt) // Pitch
	F.Set(11, 14, dt) // Yaw

	return F
}

// buildMeasurementMatrix builds H for a given sensor type
func (ekf *ExtendedKalmanFilter) buildMeasurementMatrix(sensorType SensorType) *mat.Dense {
	switch sensorType {
	case SensorGPS:
		// GPS measures position (3 measurements)
		H := mat.NewDense(3, 15, nil)
		H.Set(0, 0, 1.0) // X
		H.Set(1, 1, 1.0) // Y
		H.Set(2, 2, 1.0) // Z
		return H

	case SensorINS:
		// INS measures acceleration and angular rates (6 measurements)
		H := mat.NewDense(6, 15, nil)
		H.Set(0, 6, 1.0)  // Ax
		H.Set(1, 7, 1.0)  // Ay
		H.Set(2, 8, 1.0)  // Az
		H.Set(3, 12, 1.0) // P
		H.Set(4, 13, 1.0) // Q
		H.Set(5, 14, 1.0) // R
		return H

	case SensorRADAR, SensorLIDAR:
		// RADAR/LIDAR measure position (3 measurements)
		H := mat.NewDense(3, 15, nil)
		H.Set(0, 0, 1.0) // X
		H.Set(1, 1, 1.0) // Y
		H.Set(2, 2, 1.0) // Z
		return H

	case SensorBarometer:
		// Barometer measures altitude (1 measurement)
		H := mat.NewDense(1, 15, nil)
		H.Set(0, 2, 1.0) // Z
		return H

	case SensorPitot:
		// Pitot measures airspeed (1 measurement - approximate as Vx)
		H := mat.NewDense(1, 15, nil)
		H.Set(0, 3, 1.0) // Vx
		return H

	default:
		// Default: measure position
		H := mat.NewDense(3, 15, nil)
		H.Set(0, 0, 1.0)
		H.Set(1, 1, 1.0)
		H.Set(2, 2, 1.0)
		return H
	}
}

// GetState returns the current fused state
func (ekf *ExtendedKalmanFilter) GetState() *FusionState {
	ekf.mu.RLock()
	defer ekf.mu.RUnlock()

	state := &FusionState{
		Position: [3]float64{
			ekf.state.AtVec(0),
			ekf.state.AtVec(1),
			ekf.state.AtVec(2),
		},
		Velocity: [3]float64{
			ekf.state.AtVec(3),
			ekf.state.AtVec(4),
			ekf.state.AtVec(5),
		},
		Acceleration: [3]float64{
			ekf.state.AtVec(6),
			ekf.state.AtVec(7),
			ekf.state.AtVec(8),
		},
		Attitude: [3]float64{
			ekf.state.AtVec(9),
			ekf.state.AtVec(10),
			ekf.state.AtVec(11),
		},
		AngularRate: [3]float64{
			ekf.state.AtVec(12),
			ekf.state.AtVec(13),
			ekf.state.AtVec(14),
		},
		Timestamp:  ekf.lastUpdate,
		Covariance: ekf.covariance,
	}

	// Calculate confidence from covariance trace
	trace := 0.0
	for i := 0; i < 15; i++ {
		trace += ekf.covariance.At(i, i)
	}
	state.Confidence = 1.0 / (1.0 + trace/15.0)

	return state
}

// Run starts the fusion loop
func (ekf *ExtendedKalmanFilter) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Duration(float64(time.Second) / ekf.config.UpdateRate))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			// Predict step
			if err := ekf.Predict(ctx); err != nil {
				return err
			}

		case reading := <-ekf.sensorReadings:
			// Update step with new measurement
			if err := ekf.Update(reading); err != nil {
				// Log error but continue
				continue
			}
		}
	}
}

// AddReading adds a new sensor reading to the fusion queue
func (ekf *ExtendedKalmanFilter) AddReading(reading SensorReading) {
	select {
	case ekf.sensorReadings <- reading:
	default:
		// Buffer full, drop oldest
		select {
		case <-ekf.sensorReadings:
		default:
		}
		ekf.sensorReadings <- reading
	}
}

// GetUpdateCount returns the number of updates performed
func (ekf *ExtendedKalmanFilter) GetUpdateCount() uint64 {
	ekf.mu.RLock()
	defer ekf.mu.RUnlock()
	return ekf.updateCount
}
