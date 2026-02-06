// Package perception provides 360-degree perception and tracking for Hunoid robotics.
// It enables real-time object detection, tracking, and rescue prioritization.
//
// Copyright 2026 Arobi. All Rights Reserved.
package perception

import (
	"context"
	"math"
	"sync"
	"time"
)

// ObjectClass defines the type of detected object
type ObjectClass string

const (
	ClassHuman    ObjectClass = "human"
	ClassVehicle  ObjectClass = "vehicle"
	ClassDebris   ObjectClass = "debris"
	ClassObstacle ObjectClass = "obstacle"
	ClassUnknown  ObjectClass = "unknown"
)

// Vector3 represents a 3D vector
type Vector3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// Magnitude returns the magnitude of the vector
func (v Vector3) Magnitude() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

// Distance returns the distance to another vector
func (v Vector3) Distance(other Vector3) float64 {
	dx := v.X - other.X
	dy := v.Y - other.Y
	dz := v.Z - other.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// Subtract returns v - other
func (v Vector3) Subtract(other Vector3) Vector3 {
	return Vector3{X: v.X - other.X, Y: v.Y - other.Y, Z: v.Z - other.Z}
}

// Add returns v + other
func (v Vector3) Add(other Vector3) Vector3 {
	return Vector3{X: v.X + other.X, Y: v.Y + other.Y, Z: v.Z + other.Z}
}

// Scale returns v * scalar
func (v Vector3) Scale(s float64) Vector3 {
	return Vector3{X: v.X * s, Y: v.Y * s, Z: v.Z * s}
}

// BoundingBox3D represents an axis-aligned bounding box
type BoundingBox3D struct {
	Min Vector3 `json:"min"`
	Max Vector3 `json:"max"`
}

// Center returns the center of the bounding box
func (b BoundingBox3D) Center() Vector3 {
	return Vector3{
		X: (b.Min.X + b.Max.X) / 2,
		Y: (b.Min.Y + b.Max.Y) / 2,
		Z: (b.Min.Z + b.Max.Z) / 2,
	}
}

// PredictedPoint represents a point on a predicted trajectory
type PredictedPoint struct {
	Position   Vector3       `json:"position"`
	Velocity   Vector3       `json:"velocity"`
	TimeOffset time.Duration `json:"timeOffset"`
	Confidence float64       `json:"confidence"`
}

// TrackedObject represents an object being tracked in 360-degree space
type TrackedObject struct {
	ID             string                 `json:"id"`
	ClassType      ObjectClass            `json:"classType"`
	Position       Vector3                `json:"position"`
	Velocity       Vector3                `json:"velocity"`
	Acceleration   Vector3                `json:"acceleration"`
	BoundingBox    BoundingBox3D          `json:"boundingBox"`
	Confidence     float64                `json:"confidence"`
	FirstSeen      time.Time              `json:"firstSeen"`
	LastSeen       time.Time              `json:"lastSeen"`
	TrackAge       int                    `json:"trackAge"`
	KalmanState    *KalmanState9D         `json:"-"` // Internal Kalman state
	PredictedPath  []PredictedPoint       `json:"predictedPath"`
	ThreatLevel    float64                `json:"threatLevel"`
	RescuePriority float64                `json:"rescuePriority"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// KalmanState9D represents the 9-state Kalman filter state
// States: [x, y, z, vx, vy, vz, ax, ay, az]
type KalmanState9D struct {
	X [9]float64    // State vector
	P [9][9]float64 // Covariance matrix
}

// SensorFusionResult holds information about sensor coverage
type SensorFusionResult struct {
	CameraCoverage   float64 `json:"cameraCoverage"` // Percentage of 360 covered
	LidarCoverage    float64 `json:"lidarCoverage"`
	DepthCoverage    float64 `json:"depthCoverage"`
	FusionConfidence float64 `json:"fusionConfidence"`
}

// ScanResult360 represents the result of a 360-degree scan
type ScanResult360 struct {
	Timestamp      time.Time          `json:"timestamp"`
	ProcessingTime time.Duration      `json:"processingTime"`
	Objects        []TrackedObject    `json:"objects"`
	HumanCount     int                `json:"humanCount"`
	ThreatCount    int                `json:"threatCount"`
	OctreeRoot     *OctreeNode        `json:"-"` // Spatial index
	SensorFusion   SensorFusionResult `json:"sensorFusion"`
}

// Scanner360Config holds scanner configuration
type Scanner360Config struct {
	MaxRange          float64       // Maximum detection range (meters)
	UpdateRate        float64       // Target update rate (Hz)
	MaxLatencyMs      int64         // Maximum allowed latency (ms)
	TrackTimeout      time.Duration // Time before track is dropped
	MinConfidence     float64       // Minimum confidence for valid detection
	PredictionHorizon time.Duration // How far to predict trajectories
}

// DefaultScanner360Config returns default configuration
func DefaultScanner360Config() Scanner360Config {
	return Scanner360Config{
		MaxRange:          50.0,
		UpdateRate:        10.0,
		MaxLatencyMs:      100,
		TrackTimeout:      5 * time.Second,
		MinConfidence:     0.5,
		PredictionHorizon: 5 * time.Second,
	}
}

// Scanner360 provides 360-degree perception capabilities
type Scanner360 struct {
	mu sync.RWMutex

	config     Scanner360Config
	tracks     map[string]*TrackedObject
	octree     *Octree
	lastScan   *ScanResult360
	trackIDSeq int64

	// Sensor interfaces (would be implemented by actual hardware)
	cameraFeed chan []byte
	lidarFeed  chan []Vector3
	depthFeed  chan [][]float64

	// Processing metrics
	avgLatency time.Duration
	frameCount int64
}

// NewScanner360 creates a new 360-degree scanner
func NewScanner360(config Scanner360Config) *Scanner360 {
	return &Scanner360{
		config:     config,
		tracks:     make(map[string]*TrackedObject),
		octree:     NewOctree(Vector3{0, 0, 0}, config.MaxRange),
		cameraFeed: make(chan []byte, 10),
		lidarFeed:  make(chan []Vector3, 10),
		depthFeed:  make(chan [][]float64, 10),
	}
}

// Start begins the 360-degree scanning process
func (s *Scanner360) Start(ctx context.Context) error {
	ticker := time.NewTicker(time.Duration(float64(time.Second) / s.config.UpdateRate))
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			s.processScan()
		}
	}
}

// processScan performs one scan cycle
func (s *Scanner360) processScan() {
	startTime := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Update existing tracks with Kalman prediction
	s.predictTracks()

	// Process any pending sensor data
	// In real implementation, this would fuse camera, lidar, and depth data

	// Clean up stale tracks
	s.cleanupTracks()

	// Build scan result
	objects := make([]TrackedObject, 0, len(s.tracks))
	humanCount := 0
	threatCount := 0

	for _, track := range s.tracks {
		objects = append(objects, *track)
		if track.ClassType == ClassHuman {
			humanCount++
		}
		if track.ThreatLevel > 0.5 {
			threatCount++
		}
	}

	processingTime := time.Since(startTime)

	s.lastScan = &ScanResult360{
		Timestamp:      time.Now(),
		ProcessingTime: processingTime,
		Objects:        objects,
		HumanCount:     humanCount,
		ThreatCount:    threatCount,
		OctreeRoot:     s.octree.root,
		SensorFusion: SensorFusionResult{
			CameraCoverage:   1.0, // Simulated full coverage
			LidarCoverage:    1.0,
			DepthCoverage:    1.0,
			FusionConfidence: 0.95,
		},
	}

	// Update latency metrics
	s.frameCount++
	s.avgLatency = (s.avgLatency*time.Duration(s.frameCount-1) + processingTime) / time.Duration(s.frameCount)
}

// predictTracks updates all tracks with Kalman prediction
func (s *Scanner360) predictTracks() {
	dt := 1.0 / s.config.UpdateRate

	for _, track := range s.tracks {
		if track.KalmanState == nil {
			continue
		}

		// State transition: x_new = x + v*dt + 0.5*a*dt^2
		track.KalmanState.X[0] += track.KalmanState.X[3]*dt + 0.5*track.KalmanState.X[6]*dt*dt
		track.KalmanState.X[1] += track.KalmanState.X[4]*dt + 0.5*track.KalmanState.X[7]*dt*dt
		track.KalmanState.X[2] += track.KalmanState.X[5]*dt + 0.5*track.KalmanState.X[8]*dt*dt

		// Velocity update: v_new = v + a*dt
		track.KalmanState.X[3] += track.KalmanState.X[6] * dt
		track.KalmanState.X[4] += track.KalmanState.X[7] * dt
		track.KalmanState.X[5] += track.KalmanState.X[8] * dt

		// Update track state from Kalman
		track.Position = Vector3{
			X: track.KalmanState.X[0],
			Y: track.KalmanState.X[1],
			Z: track.KalmanState.X[2],
		}
		track.Velocity = Vector3{
			X: track.KalmanState.X[3],
			Y: track.KalmanState.X[4],
			Z: track.KalmanState.X[5],
		}
		track.Acceleration = Vector3{
			X: track.KalmanState.X[6],
			Y: track.KalmanState.X[7],
			Z: track.KalmanState.X[8],
		}

		// Generate predicted path
		track.PredictedPath = s.generatePredictedPath(track)
	}
}

// generatePredictedPath creates a trajectory prediction for an object
func (s *Scanner360) generatePredictedPath(track *TrackedObject) []PredictedPoint {
	steps := 10
	dt := float64(s.config.PredictionHorizon) / float64(steps) / float64(time.Second)

	path := make([]PredictedPoint, steps)
	pos := track.Position
	vel := track.Velocity
	acc := track.Acceleration

	for i := 0; i < steps; i++ {
		// Constant acceleration model
		pos = pos.Add(vel.Scale(dt)).Add(acc.Scale(0.5 * dt * dt))
		vel = vel.Add(acc.Scale(dt))

		// Confidence decreases with time
		confidence := track.Confidence * math.Exp(-float64(i)*0.1)

		path[i] = PredictedPoint{
			Position:   pos,
			Velocity:   vel,
			TimeOffset: time.Duration(float64(i+1) * dt * float64(time.Second)),
			Confidence: confidence,
		}
	}

	return path
}

// cleanupTracks removes stale tracks
func (s *Scanner360) cleanupTracks() {
	now := time.Now()
	for id, track := range s.tracks {
		if now.Sub(track.LastSeen) > s.config.TrackTimeout {
			delete(s.tracks, id)
			s.octree.Remove(track)
		}
	}
}

// AddDetection adds a new detection to be tracked
func (s *Scanner360) AddDetection(class ObjectClass, position, velocity Vector3, confidence float64) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.trackIDSeq++
	id := generateTrackID(s.trackIDSeq)

	track := &TrackedObject{
		ID:          id,
		ClassType:   class,
		Position:    position,
		Velocity:    velocity,
		Confidence:  confidence,
		FirstSeen:   time.Now(),
		LastSeen:    time.Now(),
		TrackAge:    1,
		Metadata:    make(map[string]interface{}),
		KalmanState: initKalmanState(position, velocity),
	}

	s.tracks[id] = track
	s.octree.Insert(track)

	return id
}

// GetLatestScan returns the most recent scan result
func (s *Scanner360) GetLatestScan() *ScanResult360 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastScan
}

// GetTrack returns a specific tracked object
func (s *Scanner360) GetTrack(id string) *TrackedObject {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tracks[id]
}

// GetHumansInDanger returns all human objects with high threat levels
func (s *Scanner360) GetHumansInDanger() []*TrackedObject {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*TrackedObject, 0)
	for _, track := range s.tracks {
		if track.ClassType == ClassHuman && track.ThreatLevel > 0.3 {
			result = append(result, track)
		}
	}
	return result
}

// QueryRadius returns all objects within radius of a point
func (s *Scanner360) QueryRadius(center Vector3, radius float64) []*TrackedObject {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.octree.QueryRadius(center, radius)
}

// GetAverageLatency returns the average processing latency
func (s *Scanner360) GetAverageLatency() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.avgLatency
}

// Helper functions

func generateTrackID(seq int64) string {
	return "TRK-" + time.Now().Format("20060102") + "-" + string(rune(seq))
}

func initKalmanState(position, velocity Vector3) *KalmanState9D {
	state := &KalmanState9D{}
	state.X[0] = position.X
	state.X[1] = position.Y
	state.X[2] = position.Z
	state.X[3] = velocity.X
	state.X[4] = velocity.Y
	state.X[5] = velocity.Z
	// Acceleration starts at 0

	// Initialize covariance with uncertainty
	for i := 0; i < 9; i++ {
		state.P[i][i] = 1.0 // Initial uncertainty
	}

	return state
}
