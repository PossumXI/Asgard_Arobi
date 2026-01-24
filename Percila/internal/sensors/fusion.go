// Package sensors provides advanced sensor fusion capabilities for PERCILA payload guidance.
package sensors

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SensorType defines the type of sensor
type SensorType string

const (
	SensorGPS    SensorType = "gps"    // Global Positioning System
	SensorINS    SensorType = "ins"    // Inertial Navigation System
	SensorRADAR  SensorType = "radar"  // Radio Detection and Ranging
	SensorLIDAR  SensorType = "lidar"  // Light Detection and Ranging
	SensorVISUAL SensorType = "visual" // Visual/Camera-based tracking
	SensorIR     SensorType = "ir"     // Infrared sensor
)

// SensorStatus represents the operational status of a sensor
type SensorStatus string

const (
	StatusHealthy   SensorStatus = "healthy"
	StatusDegraded  SensorStatus = "degraded"
	StatusFailed    SensorStatus = "failed"
	StatusCalibrate SensorStatus = "calibrating"
	StatusOffline   SensorStatus = "offline"
)

// Vector3D represents position/velocity in 3D space
type Vector3D struct {
	X float64 `json:"x"` // meters or m/s (East)
	Y float64 `json:"y"` // meters or m/s (North)
	Z float64 `json:"z"` // meters or m/s (Altitude/Up)
}

// Matrix3x3 represents a 3x3 covariance matrix
type Matrix3x3 [3][3]float64

// Matrix6x6 represents a 6x6 state covariance matrix
type Matrix6x6 [6][6]float64

// SensorReading represents a single sensor measurement
type SensorReading struct {
	SensorID   string     `json:"sensorId"`
	SensorType SensorType `json:"sensorType"`
	Position   Vector3D   `json:"position"`
	Velocity   Vector3D   `json:"velocity"`
	Covariance Matrix3x3  `json:"covariance"`
	Timestamp  time.Time  `json:"timestamp"`
	Confidence float64    `json:"confidence"` // 0.0-1.0
	IsValid    bool       `json:"isValid"`
}

// SensorHealth represents the health metrics of a sensor
type SensorHealth struct {
	SensorID       string        `json:"sensorId"`
	SensorType     SensorType    `json:"sensorType"`
	Status         SensorStatus  `json:"status"`
	LastReading    time.Time     `json:"lastReading"`
	ReadingRate    float64       `json:"readingRate"` // Hz
	ErrorRate      float64       `json:"errorRate"`   // 0.0-1.0
	NoiseLevel     float64       `json:"noiseLevel"`
	DriftRate      float64       `json:"driftRate"`   // drift per second
	Temperature    float64       `json:"temperature"` // Celsius
	Uptime         time.Duration `json:"uptime"`
	ReadingsTotal  int64         `json:"readingsTotal"`
	ReadingsValid  int64         `json:"readingsValid"`
	AnomalyCount   int64         `json:"anomalyCount"`
	LastCalibrated time.Time     `json:"lastCalibrated"`
}

// CalibrationData stores sensor calibration parameters
type CalibrationData struct {
	SensorID     string    `json:"sensorId"`
	SensorType   SensorType `json:"sensorType"`
	BiasX        float64   `json:"biasX"`
	BiasY        float64   `json:"biasY"`
	BiasZ        float64   `json:"biasZ"`
	ScaleX       float64   `json:"scaleX"`
	ScaleY       float64   `json:"scaleY"`
	ScaleZ       float64   `json:"scaleZ"`
	Misalignment Matrix3x3 `json:"misalignment"`
	Timestamp    time.Time `json:"timestamp"`
	ValidUntil   time.Time `json:"validUntil"`
}

// FusedState represents the combined state estimate from all sensors
type FusedState struct {
	Position       Vector3D  `json:"position"`
	Velocity       Vector3D  `json:"velocity"`
	Acceleration   Vector3D  `json:"acceleration"`
	Covariance     Matrix6x6 `json:"covariance"`
	Timestamp      time.Time `json:"timestamp"`
	Confidence     float64   `json:"confidence"`
	ActiveSensors  int       `json:"activeSensors"`
	PrimarySensor  SensorType `json:"primarySensor"`
	FusionQuality  float64   `json:"fusionQuality"` // 0.0-1.0
	IsConverged    bool      `json:"isConverged"`
}

// AnomalyReport describes a detected sensor anomaly
type AnomalyReport struct {
	ID          string     `json:"id"`
	SensorID    string     `json:"sensorId"`
	SensorType  SensorType `json:"sensorType"`
	AnomalyType string     `json:"anomalyType"` // spike, drift, dropout, noise, inconsistent
	Severity    float64    `json:"severity"`    // 0.0-1.0
	Description string     `json:"description"`
	Reading     SensorReading `json:"reading"`
	Expected    Vector3D   `json:"expected"`
	Actual      Vector3D   `json:"actual"`
	Timestamp   time.Time  `json:"timestamp"`
}

// EKFConfig holds Extended Kalman Filter configuration
type EKFConfig struct {
	ProcessNoisePos     float64 `json:"processNoisePos"`
	ProcessNoiseVel     float64 `json:"processNoiseVel"`
	InitialCovariance   float64 `json:"initialCovariance"`
	MahalanobisThreshold float64 `json:"mahalanobisThreshold"` // For outlier rejection
	ConvergenceThreshold float64 `json:"convergenceThreshold"`
	MaxIterations       int     `json:"maxIterations"`
}

// FusionConfig holds sensor fusion configuration
type FusionConfig struct {
	EKF                  EKFConfig          `json:"ekf"`
	SensorPriorities     map[SensorType]int `json:"sensorPriorities"`
	SensorWeights        map[SensorType]float64 `json:"sensorWeights"`
	MinSensorsRequired   int               `json:"minSensorsRequired"`
	SensorTimeout        time.Duration     `json:"sensorTimeout"`
	AnomalyThreshold     float64           `json:"anomalyThreshold"`
	CalibrationInterval  time.Duration     `json:"calibrationInterval"`
	EnableFailover       bool              `json:"enableFailover"`
	FailoverPriority     []SensorType      `json:"failoverPriority"`
	UpdateRate           time.Duration     `json:"updateRate"`
}

// DefaultFusionConfig returns default fusion configuration
func DefaultFusionConfig() FusionConfig {
	return FusionConfig{
		EKF: EKFConfig{
			ProcessNoisePos:      0.1,
			ProcessNoiseVel:      0.05,
			InitialCovariance:    100.0,
			MahalanobisThreshold: 5.991, // 95% confidence for 2 DOF
			ConvergenceThreshold: 0.001,
			MaxIterations:        10,
		},
		SensorPriorities: map[SensorType]int{
			SensorGPS:    1,
			SensorINS:    2,
			SensorRADAR:  3,
			SensorLIDAR:  4,
			SensorVISUAL: 5,
			SensorIR:     6,
		},
		SensorWeights: map[SensorType]float64{
			SensorGPS:    1.0,
			SensorINS:    0.9,
			SensorRADAR:  0.8,
			SensorLIDAR:  0.85,
			SensorVISUAL: 0.7,
			SensorIR:     0.6,
		},
		MinSensorsRequired:  2,
		SensorTimeout:       2 * time.Second,
		AnomalyThreshold:    3.0, // 3 sigma
		CalibrationInterval: 1 * time.Hour,
		EnableFailover:      true,
		FailoverPriority: []SensorType{
			SensorINS, SensorRADAR, SensorLIDAR, SensorGPS, SensorVISUAL, SensorIR,
		},
		UpdateRate: 50 * time.Millisecond, // 20Hz
	}
}

// SensorFusion provides advanced multi-sensor fusion for payload guidance
type SensorFusion struct {
	mu sync.RWMutex

	id       string
	config   FusionConfig
	isRunning bool
	ctx      context.Context
	cancel   context.CancelFunc

	// EKF state
	state          FusedState
	stateHistory   []FusedState
	historyMaxSize int

	// Sensor management
	sensorHealth    map[string]*SensorHealth
	sensorReadings  map[string]SensorReading
	calibrations    map[string]*CalibrationData
	lastReadingTime map[string]time.Time

	// Anomaly detection
	anomalies      []AnomalyReport
	anomalyMaxSize int

	// Callbacks
	onStateUpdate    func(state FusedState)
	onSensorFailure  func(sensorID string, health SensorHealth)
	onAnomalyDetect  func(anomaly AnomalyReport)
	onFailoverEvent  func(from, to SensorType)
}

// NewSensorFusion creates a new sensor fusion instance
func NewSensorFusion(config FusionConfig) *SensorFusion {
	sf := &SensorFusion{
		id:              uuid.New().String(),
		config:          config,
		sensorHealth:    make(map[string]*SensorHealth),
		sensorReadings:  make(map[string]SensorReading),
		calibrations:    make(map[string]*CalibrationData),
		lastReadingTime: make(map[string]time.Time),
		anomalies:       make([]AnomalyReport, 0),
		stateHistory:    make([]FusedState, 0),
		historyMaxSize:  1000,
		anomalyMaxSize:  500,
	}

	// Initialize state with high uncertainty
	sf.state = FusedState{
		Covariance: initializeCovariance(config.EKF.InitialCovariance),
		Timestamp:  time.Now(),
		Confidence: 0.0,
	}

	return sf
}

// Start begins the sensor fusion processing
func (sf *SensorFusion) Start(ctx context.Context) error {
	sf.mu.Lock()
	if sf.isRunning {
		sf.mu.Unlock()
		return fmt.Errorf("sensor fusion already running")
	}

	sf.ctx, sf.cancel = context.WithCancel(ctx)
	sf.isRunning = true
	sf.mu.Unlock()

	go sf.fusionLoop()
	go sf.healthMonitorLoop()

	return nil
}

// Stop stops the sensor fusion processing
func (sf *SensorFusion) Stop() {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if sf.cancel != nil {
		sf.cancel()
	}
	sf.isRunning = false
}

// RegisterSensor registers a new sensor with the fusion system
func (sf *SensorFusion) RegisterSensor(sensorID string, sensorType SensorType, calibration *CalibrationData) error {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if _, exists := sf.sensorHealth[sensorID]; exists {
		return fmt.Errorf("sensor %s already registered", sensorID)
	}

	sf.sensorHealth[sensorID] = &SensorHealth{
		SensorID:       sensorID,
		SensorType:     sensorType,
		Status:         StatusOffline,
		LastCalibrated: time.Now(),
	}

	if calibration != nil {
		sf.calibrations[sensorID] = calibration
	} else {
		sf.calibrations[sensorID] = defaultCalibration(sensorID, sensorType)
	}

	return nil
}

// UnregisterSensor removes a sensor from the fusion system
func (sf *SensorFusion) UnregisterSensor(sensorID string) {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	delete(sf.sensorHealth, sensorID)
	delete(sf.sensorReadings, sensorID)
	delete(sf.calibrations, sensorID)
	delete(sf.lastReadingTime, sensorID)
}

// ProcessReading processes a new sensor reading
func (sf *SensorFusion) ProcessReading(reading SensorReading) error {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	// Check if sensor is registered
	health, exists := sf.sensorHealth[reading.SensorID]
	if !exists {
		return fmt.Errorf("sensor %s not registered", reading.SensorID)
	}

	// Apply calibration
	calibration, hasCalib := sf.calibrations[reading.SensorID]
	if hasCalib {
		reading = sf.applyCalibration(reading, calibration)
	}

	// Update sensor health
	health.ReadingsTotal++
	health.LastReading = reading.Timestamp
	if reading.IsValid {
		health.ReadingsValid++
		health.Status = StatusHealthy
	}

	// Calculate reading rate
	if lastTime, hasLast := sf.lastReadingTime[reading.SensorID]; hasLast {
		elapsed := reading.Timestamp.Sub(lastTime).Seconds()
		if elapsed > 0 {
			health.ReadingRate = 1.0 / elapsed
		}
	}
	sf.lastReadingTime[reading.SensorID] = reading.Timestamp

	// Anomaly detection
	if anomaly := sf.detectAnomaly(reading); anomaly != nil {
		sf.anomalies = append(sf.anomalies, *anomaly)
		if len(sf.anomalies) > sf.anomalyMaxSize {
			sf.anomalies = sf.anomalies[1:]
		}
		health.AnomalyCount++

		if sf.onAnomalyDetect != nil {
			go sf.onAnomalyDetect(*anomaly)
		}

		// Skip readings with severe anomalies
		if anomaly.Severity > 0.8 {
			reading.IsValid = false
		}
	}

	// Store reading
	sf.sensorReadings[reading.SensorID] = reading

	return nil
}

// GetFusedState returns the current fused state estimate
func (sf *SensorFusion) GetFusedState() FusedState {
	sf.mu.RLock()
	defer sf.mu.RUnlock()
	return sf.state
}

// GetSensorHealth returns health information for a specific sensor
func (sf *SensorFusion) GetSensorHealth(sensorID string) (*SensorHealth, bool) {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	health, exists := sf.sensorHealth[sensorID]
	if !exists {
		return nil, false
	}

	// Return a copy
	healthCopy := *health
	return &healthCopy, true
}

// GetAllSensorHealth returns health information for all sensors
func (sf *SensorFusion) GetAllSensorHealth() map[string]SensorHealth {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	result := make(map[string]SensorHealth)
	for id, health := range sf.sensorHealth {
		result[id] = *health
	}
	return result
}

// GetActiveSensors returns the count and list of active sensors
func (sf *SensorFusion) GetActiveSensors() (int, []string) {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	var active []string
	now := time.Now()

	for id, health := range sf.sensorHealth {
		if health.Status == StatusHealthy || health.Status == StatusDegraded {
			if now.Sub(health.LastReading) < sf.config.SensorTimeout {
				active = append(active, id)
			}
		}
	}

	return len(active), active
}

// UpdateCalibration updates calibration data for a sensor
func (sf *SensorFusion) UpdateCalibration(sensorID string, calibration CalibrationData) error {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if _, exists := sf.sensorHealth[sensorID]; !exists {
		return fmt.Errorf("sensor %s not registered", sensorID)
	}

	sf.calibrations[sensorID] = &calibration
	sf.sensorHealth[sensorID].LastCalibrated = calibration.Timestamp

	return nil
}

// GetAnomalies returns recent anomaly reports
func (sf *SensorFusion) GetAnomalies(limit int) []AnomalyReport {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	if limit <= 0 || limit > len(sf.anomalies) {
		limit = len(sf.anomalies)
	}

	// Return most recent anomalies
	start := len(sf.anomalies) - limit
	result := make([]AnomalyReport, limit)
	copy(result, sf.anomalies[start:])

	return result
}

// fusionLoop is the main sensor fusion processing loop
func (sf *SensorFusion) fusionLoop() {
	ticker := time.NewTicker(sf.config.UpdateRate)
	defer ticker.Stop()

	for {
		select {
		case <-sf.ctx.Done():
			return

		case <-ticker.C:
			sf.performFusion()
		}
	}
}

// performFusion executes one fusion cycle using EKF
func (sf *SensorFusion) performFusion() {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	now := time.Now()

	// Collect valid readings
	validReadings := sf.collectValidReadings(now)

	if len(validReadings) < sf.config.MinSensorsRequired {
		// Not enough sensors - attempt failover
		if sf.config.EnableFailover {
			sf.handleFailover()
		}
		sf.state.Confidence *= 0.95 // Decay confidence
		return
	}

	// Calculate time delta
	dt := now.Sub(sf.state.Timestamp).Seconds()
	if dt <= 0 || dt > 1.0 {
		dt = sf.config.UpdateRate.Seconds()
	}

	// EKF Prediction Step
	sf.ekfPredict(dt)

	// EKF Update Step for each sensor
	for _, reading := range validReadings {
		sf.ekfUpdate(reading)
	}

	// Calculate fusion quality and confidence
	sf.updateFusionQuality(validReadings)

	// Update timestamp
	sf.state.Timestamp = now
	sf.state.ActiveSensors = len(validReadings)

	// Determine primary sensor
	sf.state.PrimarySensor = sf.determinePrimarySensor(validReadings)

	// Store history
	sf.stateHistory = append(sf.stateHistory, sf.state)
	if len(sf.stateHistory) > sf.historyMaxSize {
		sf.stateHistory = sf.stateHistory[1:]
	}

	// Callback
	if sf.onStateUpdate != nil {
		stateCopy := sf.state
		go sf.onStateUpdate(stateCopy)
	}
}

// collectValidReadings gathers valid sensor readings
func (sf *SensorFusion) collectValidReadings(now time.Time) []SensorReading {
	var readings []SensorReading

	for _, reading := range sf.sensorReadings {
		// Check freshness
		if now.Sub(reading.Timestamp) > sf.config.SensorTimeout {
			continue
		}

		// Check validity
		if !reading.IsValid {
			continue
		}

		// Check sensor health
		health, exists := sf.sensorHealth[reading.SensorID]
		if !exists || health.Status == StatusFailed || health.Status == StatusOffline {
			continue
		}

		readings = append(readings, reading)
	}

	return readings
}

// ekfPredict performs the EKF prediction step
func (sf *SensorFusion) ekfPredict(dt float64) {
	// State transition: simple kinematic model
	// position = position + velocity * dt + 0.5 * acceleration * dt^2
	// velocity = velocity + acceleration * dt

	sf.state.Position.X += sf.state.Velocity.X*dt + 0.5*sf.state.Acceleration.X*dt*dt
	sf.state.Position.Y += sf.state.Velocity.Y*dt + 0.5*sf.state.Acceleration.Y*dt*dt
	sf.state.Position.Z += sf.state.Velocity.Z*dt + 0.5*sf.state.Acceleration.Z*dt*dt

	sf.state.Velocity.X += sf.state.Acceleration.X * dt
	sf.state.Velocity.Y += sf.state.Acceleration.Y * dt
	sf.state.Velocity.Z += sf.state.Acceleration.Z * dt

	// Update covariance with process noise
	// P = F * P * F' + Q
	// Using simplified diagonal update
	processNoisePos := sf.config.EKF.ProcessNoisePos * dt * dt
	processNoiseVel := sf.config.EKF.ProcessNoiseVel * dt

	for i := 0; i < 3; i++ {
		sf.state.Covariance[i][i] += processNoisePos
		sf.state.Covariance[i+3][i+3] += processNoiseVel
	}
}

// ekfUpdate performs the EKF update step for a single sensor reading
func (sf *SensorFusion) ekfUpdate(reading SensorReading) {
	// Get sensor weight
	weight := sf.config.SensorWeights[reading.SensorType]
	if weight == 0 {
		weight = 0.5
	}

	// Calculate innovation (measurement residual)
	innovationPos := Vector3D{
		X: reading.Position.X - sf.state.Position.X,
		Y: reading.Position.Y - sf.state.Position.Y,
		Z: reading.Position.Z - sf.state.Position.Z,
	}

	innovationVel := Vector3D{
		X: reading.Velocity.X - sf.state.Velocity.X,
		Y: reading.Velocity.Y - sf.state.Velocity.Y,
		Z: reading.Velocity.Z - sf.state.Velocity.Z,
	}

	// Calculate Mahalanobis distance for outlier rejection
	mahalDist := sf.calculateMahalanobis(innovationPos, reading.Covariance)
	if mahalDist > sf.config.EKF.MahalanobisThreshold {
		// Reject outlier
		return
	}

	// Calculate Kalman gain (simplified diagonal form)
	// K = P * H' * (H * P * H' + R)^-1
	// For direct measurement H = I, so K = P * (P + R)^-1

	kalmanGainPos := make([]float64, 3)
	kalmanGainVel := make([]float64, 3)

	for i := 0; i < 3; i++ {
		pPos := sf.state.Covariance[i][i]
		rPos := reading.Covariance[i][i] / weight // Scale measurement noise by weight

		if pPos+rPos > 0 {
			kalmanGainPos[i] = pPos / (pPos + rPos)
		}

		pVel := sf.state.Covariance[i+3][i+3]
		// Use position covariance scaled for velocity measurement noise
		rVel := rPos * 0.1

		if pVel+rVel > 0 {
			kalmanGainVel[i] = pVel / (pVel + rVel)
		}
	}

	// Update state estimate
	sf.state.Position.X += kalmanGainPos[0] * innovationPos.X
	sf.state.Position.Y += kalmanGainPos[1] * innovationPos.Y
	sf.state.Position.Z += kalmanGainPos[2] * innovationPos.Z

	sf.state.Velocity.X += kalmanGainVel[0] * innovationVel.X
	sf.state.Velocity.Y += kalmanGainVel[1] * innovationVel.Y
	sf.state.Velocity.Z += kalmanGainVel[2] * innovationVel.Z

	// Update covariance: P = (I - K*H) * P
	for i := 0; i < 3; i++ {
		sf.state.Covariance[i][i] *= (1 - kalmanGainPos[i])
		sf.state.Covariance[i+3][i+3] *= (1 - kalmanGainVel[i])
	}
}

// calculateMahalanobis computes the Mahalanobis distance
func (sf *SensorFusion) calculateMahalanobis(innovation Vector3D, cov Matrix3x3) float64 {
	// Simplified: assume diagonal covariance
	// d² = Σ (innovation[i]² / covariance[i][i])

	var dist float64
	if cov[0][0] > 0 {
		dist += (innovation.X * innovation.X) / cov[0][0]
	}
	if cov[1][1] > 0 {
		dist += (innovation.Y * innovation.Y) / cov[1][1]
	}
	if cov[2][2] > 0 {
		dist += (innovation.Z * innovation.Z) / cov[2][2]
	}

	return math.Sqrt(dist)
}

// updateFusionQuality calculates the overall fusion quality
func (sf *SensorFusion) updateFusionQuality(readings []SensorReading) {
	if len(readings) == 0 {
		sf.state.FusionQuality = 0
		sf.state.Confidence = 0
		sf.state.IsConverged = false
		return
	}

	// Quality based on number of sensors
	sensorContrib := math.Min(float64(len(readings))/float64(len(sf.config.SensorWeights)), 1.0)

	// Quality based on covariance (lower is better)
	traceCov := sf.state.Covariance[0][0] + sf.state.Covariance[1][1] +
		sf.state.Covariance[2][2]
	covQuality := math.Exp(-traceCov / 1000.0) // Exponential decay

	// Quality based on sensor agreement
	agreement := sf.calculateSensorAgreement(readings)

	// Combined quality
	sf.state.FusionQuality = 0.3*sensorContrib + 0.4*covQuality + 0.3*agreement

	// Confidence is fusion quality weighted by reading confidence
	avgConfidence := 0.0
	for _, r := range readings {
		avgConfidence += r.Confidence
	}
	avgConfidence /= float64(len(readings))

	sf.state.Confidence = sf.state.FusionQuality * avgConfidence

	// Check convergence
	sf.state.IsConverged = traceCov < sf.config.EKF.ConvergenceThreshold*1000
}

// calculateSensorAgreement measures how well sensors agree
func (sf *SensorFusion) calculateSensorAgreement(readings []SensorReading) float64 {
	if len(readings) < 2 {
		return 1.0
	}

	// Calculate mean position
	mean := Vector3D{}
	for _, r := range readings {
		mean.X += r.Position.X
		mean.Y += r.Position.Y
		mean.Z += r.Position.Z
	}
	n := float64(len(readings))
	mean.X /= n
	mean.Y /= n
	mean.Z /= n

	// Calculate variance from mean
	variance := 0.0
	for _, r := range readings {
		dx := r.Position.X - mean.X
		dy := r.Position.Y - mean.Y
		dz := r.Position.Z - mean.Z
		variance += dx*dx + dy*dy + dz*dz
	}
	variance /= n

	// Convert to agreement score (lower variance = higher agreement)
	// Using exponential decay with reasonable scale
	return math.Exp(-variance / 100.0)
}

// determinePrimarySensor selects the primary sensor based on priority and health
func (sf *SensorFusion) determinePrimarySensor(readings []SensorReading) SensorType {
	if len(readings) == 0 {
		return SensorINS // Default fallback
	}

	var bestSensor SensorType
	bestPriority := 999

	for _, r := range readings {
		priority := sf.config.SensorPriorities[r.SensorType]
		if priority < bestPriority {
			bestPriority = priority
			bestSensor = r.SensorType
		}
	}

	return bestSensor
}

// detectAnomaly checks a reading for anomalies
func (sf *SensorFusion) detectAnomaly(reading SensorReading) *AnomalyReport {
	// Skip if no prior state to compare
	if sf.state.Confidence < 0.1 {
		return nil
	}

	// Calculate deviation from predicted state
	deviationPos := Vector3D{
		X: reading.Position.X - sf.state.Position.X,
		Y: reading.Position.Y - sf.state.Position.Y,
		Z: reading.Position.Z - sf.state.Position.Z,
	}

	// Calculate magnitude of deviation
	devMag := math.Sqrt(deviationPos.X*deviationPos.X +
		deviationPos.Y*deviationPos.Y +
		deviationPos.Z*deviationPos.Z)

	// Expected deviation based on covariance
	expectedDev := math.Sqrt(sf.state.Covariance[0][0] +
		sf.state.Covariance[1][1] +
		sf.state.Covariance[2][2])

	// Check for spike anomaly
	if expectedDev > 0 && devMag > expectedDev*sf.config.AnomalyThreshold {
		severity := math.Min((devMag/expectedDev-sf.config.AnomalyThreshold)/sf.config.AnomalyThreshold, 1.0)

		return &AnomalyReport{
			ID:          uuid.New().String(),
			SensorID:    reading.SensorID,
			SensorType:  reading.SensorType,
			AnomalyType: "spike",
			Severity:    severity,
			Description: fmt.Sprintf("Position deviation %.2fm exceeds %.2f sigma threshold",
				devMag, sf.config.AnomalyThreshold),
			Reading:   reading,
			Expected:  sf.state.Position,
			Actual:    reading.Position,
			Timestamp: time.Now(),
		}
	}

	// Check for noise anomaly (high covariance in reading)
	readingVar := reading.Covariance[0][0] + reading.Covariance[1][1] + reading.Covariance[2][2]
	if readingVar > expectedDev*10 {
		return &AnomalyReport{
			ID:          uuid.New().String(),
			SensorID:    reading.SensorID,
			SensorType:  reading.SensorType,
			AnomalyType: "noise",
			Severity:    0.5,
			Description: fmt.Sprintf("Sensor noise level %.2f exceeds normal range", readingVar),
			Reading:     reading,
			Expected:    sf.state.Position,
			Actual:      reading.Position,
			Timestamp:   time.Now(),
		}
	}

	return nil
}

// handleFailover manages sensor failover when primary sensors fail
func (sf *SensorFusion) handleFailover() {
	now := time.Now()
	var availableSensors []SensorType

	// Find available sensors by failover priority
	for _, sensorType := range sf.config.FailoverPriority {
		for _, health := range sf.sensorHealth {
			if health.SensorType == sensorType {
				if health.Status == StatusHealthy || health.Status == StatusDegraded {
					if now.Sub(health.LastReading) < sf.config.SensorTimeout {
						availableSensors = append(availableSensors, sensorType)
						break
					}
				}
			}
		}
	}

	if len(availableSensors) > 0 && sf.state.PrimarySensor != availableSensors[0] {
		oldPrimary := sf.state.PrimarySensor
		sf.state.PrimarySensor = availableSensors[0]

		if sf.onFailoverEvent != nil {
			go sf.onFailoverEvent(oldPrimary, availableSensors[0])
		}
	}
}

// healthMonitorLoop monitors sensor health continuously
func (sf *SensorFusion) healthMonitorLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sf.ctx.Done():
			return

		case <-ticker.C:
			sf.checkSensorHealth()
		}
	}
}

// checkSensorHealth evaluates health of all sensors
func (sf *SensorFusion) checkSensorHealth() {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	now := time.Now()

	for sensorID, health := range sf.sensorHealth {
		previousStatus := health.Status

		// Check for timeout
		if now.Sub(health.LastReading) > sf.config.SensorTimeout {
			if health.Status != StatusOffline && health.Status != StatusFailed {
				health.Status = StatusOffline
			}
			continue
		}

		// Calculate error rate
		if health.ReadingsTotal > 0 {
			health.ErrorRate = 1.0 - float64(health.ReadingsValid)/float64(health.ReadingsTotal)
		}

		// Determine status based on metrics
		if health.ErrorRate > 0.5 {
			health.Status = StatusFailed
		} else if health.ErrorRate > 0.2 || health.NoiseLevel > 5.0 {
			health.Status = StatusDegraded
		} else if health.ReadingRate < 1.0 { // Less than 1 Hz
			health.Status = StatusDegraded
		} else {
			health.Status = StatusHealthy
		}

		// Check if calibration is needed
		if now.Sub(health.LastCalibrated) > sf.config.CalibrationInterval {
			health.Status = StatusCalibrate
		}

		// Calculate uptime
		if health.Status == StatusHealthy || health.Status == StatusDegraded {
			health.Uptime += time.Second
		}

		// Trigger callback on status change
		if health.Status != previousStatus && health.Status == StatusFailed {
			if sf.onSensorFailure != nil {
				healthCopy := *health
				go sf.onSensorFailure(sensorID, healthCopy)
			}
		}
	}
}

// applyCalibration applies calibration corrections to a reading
func (sf *SensorFusion) applyCalibration(reading SensorReading, cal *CalibrationData) SensorReading {
	// Apply bias correction
	reading.Position.X = (reading.Position.X - cal.BiasX) * cal.ScaleX
	reading.Position.Y = (reading.Position.Y - cal.BiasY) * cal.ScaleY
	reading.Position.Z = (reading.Position.Z - cal.BiasZ) * cal.ScaleZ

	reading.Velocity.X = (reading.Velocity.X - cal.BiasX) * cal.ScaleX
	reading.Velocity.Y = (reading.Velocity.Y - cal.BiasY) * cal.ScaleY
	reading.Velocity.Z = (reading.Velocity.Z - cal.BiasZ) * cal.ScaleZ

	// Apply misalignment correction (rotation matrix)
	reading.Position = applyRotation(reading.Position, cal.Misalignment)
	reading.Velocity = applyRotation(reading.Velocity, cal.Misalignment)

	return reading
}

// Callback setters

// OnStateUpdate sets callback for state updates
func (sf *SensorFusion) OnStateUpdate(callback func(state FusedState)) {
	sf.onStateUpdate = callback
}

// OnSensorFailure sets callback for sensor failure events
func (sf *SensorFusion) OnSensorFailure(callback func(sensorID string, health SensorHealth)) {
	sf.onSensorFailure = callback
}

// OnAnomalyDetect sets callback for anomaly detection events
func (sf *SensorFusion) OnAnomalyDetect(callback func(anomaly AnomalyReport)) {
	sf.onAnomalyDetect = callback
}

// OnFailoverEvent sets callback for failover events
func (sf *SensorFusion) OnFailoverEvent(callback func(from, to SensorType)) {
	sf.onFailoverEvent = callback
}

// Helper functions

// initializeCovariance creates an initial covariance matrix
func initializeCovariance(variance float64) Matrix6x6 {
	var cov Matrix6x6
	for i := 0; i < 6; i++ {
		cov[i][i] = variance
	}
	return cov
}

// defaultCalibration creates default calibration data
func defaultCalibration(sensorID string, sensorType SensorType) *CalibrationData {
	return &CalibrationData{
		SensorID:   sensorID,
		SensorType: sensorType,
		BiasX:      0,
		BiasY:      0,
		BiasZ:      0,
		ScaleX:     1.0,
		ScaleY:     1.0,
		ScaleZ:     1.0,
		Misalignment: Matrix3x3{
			{1, 0, 0},
			{0, 1, 0},
			{0, 0, 1},
		},
		Timestamp:  time.Now(),
		ValidUntil: time.Now().Add(24 * time.Hour),
	}
}

// applyRotation applies a rotation matrix to a vector
func applyRotation(v Vector3D, rot Matrix3x3) Vector3D {
	return Vector3D{
		X: rot[0][0]*v.X + rot[0][1]*v.Y + rot[0][2]*v.Z,
		Y: rot[1][0]*v.X + rot[1][1]*v.Y + rot[1][2]*v.Z,
		Z: rot[2][0]*v.X + rot[2][1]*v.Y + rot[2][2]*v.Z,
	}
}

// GetStateHistory returns historical fused states
func (sf *SensorFusion) GetStateHistory(limit int) []FusedState {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	if limit <= 0 || limit > len(sf.stateHistory) {
		limit = len(sf.stateHistory)
	}

	start := len(sf.stateHistory) - limit
	result := make([]FusedState, limit)
	copy(result, sf.stateHistory[start:])

	return result
}

// ResetState resets the fusion state to initial values
func (sf *SensorFusion) ResetState() {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	sf.state = FusedState{
		Covariance: initializeCovariance(sf.config.EKF.InitialCovariance),
		Timestamp:  time.Now(),
		Confidence: 0.0,
	}
	sf.stateHistory = make([]FusedState, 0)
	sf.anomalies = make([]AnomalyReport, 0)
}

// SetAcceleration sets the current acceleration estimate (from IMU)
func (sf *SensorFusion) SetAcceleration(accel Vector3D) {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	sf.state.Acceleration = accel
}

// IsRunning returns whether the fusion system is active
func (sf *SensorFusion) IsRunning() bool {
	sf.mu.RLock()
	defer sf.mu.RUnlock()
	return sf.isRunning
}

// GetConfig returns the current fusion configuration
func (sf *SensorFusion) GetConfig() FusionConfig {
	sf.mu.RLock()
	defer sf.mu.RUnlock()
	return sf.config
}

// UpdateConfig updates the fusion configuration
func (sf *SensorFusion) UpdateConfig(config FusionConfig) {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	sf.config = config
}
